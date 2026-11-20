// Package stream holds long-running inputs (a command's output or a tailed
// file) and fans their rows out to every connection that shows them.
//
// A Hub owns one Buffer per input. Every row gets a hub-wide sequence number,
// and a merge buffer receives rows from its inputs under the same lock that
// assigns the number, so merged rows are always in seq order. Rows are
// immutable once published, so snapshots share them without copying.
package stream

import (
	"cmp"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// Row is one published log row. It must not be modified after publish.
type Row = map[string]interface{}

// Defaults for InputSpec fields left at their zero value.
const (
	DefaultBufferSize = 2000
	DefaultBacklog    = 200
	MaxLineBytes      = 16 * 1024
)

// Hub owns every Buffer of one server.
type Hub struct {
	mu      sync.RWMutex // guards seq, buffers, and every buffer's ring and subscribers
	seq     int64
	buffers map[string]*Buffer

	ctx    context.Context // cancelled by Close
	cancel context.CancelFunc
	wg     sync.WaitGroup
	now    func() time.Time // replaced in tests
}

// NewHub returns an empty hub. Inputs start lazily on their first Subscribe.
func NewHub() *Hub {
	ctx, cancel := context.WithCancel(context.Background())
	return &Hub{buffers: map[string]*Buffer{}, ctx: ctx, cancel: cancel, now: time.Now}
}

// InputSpec describes one stream input. Exactly one of Cmd or File is set.
type InputSpec struct {
	Name    string            `json:"name"` // source name, copied into each row's "source"
	Cmd     string            `json:"cmd"`  // run with /bin/sh -c
	Dir     string            `json:"dir"`  // working directory for Cmd
	Env     map[string]string `json:"env"`  // extra environment for Cmd
	File    string            `json:"file"` // absolute path to tail
	Format  string            `json:"format"`
	Buffer  int               `json:"buffer"`  // rows kept
	Backlog int               `json:"backlog"` // lines read from the end of File on start
	Restart string            `json:"restart"` // on-exit or never
}

// Formats lists the accepted values of InputSpec.Format.
var Formats = []string{"auto", "json", "logfmt", "lines"}

// Normalize fills defaults and rejects invalid specs. Input calls it, and
// config validation can call it to report errors before serving.
func (s InputSpec) Normalize() (InputSpec, error) {
	if (s.Cmd == "") == (s.File == "") {
		return s, fmt.Errorf("stream %q: set exactly one of cmd or file", s.Name)
	}
	if s.Format == "" {
		s.Format = "auto"
	}
	if !slices.Contains(Formats, s.Format) {
		return s, fmt.Errorf("stream %q: format %q is not one of %s", s.Name, s.Format, strings.Join(Formats, ", "))
	}
	switch s.Restart {
	case "":
		s.Restart = "on-exit"
	case "on-exit", "never":
	default:
		return s, fmt.Errorf("stream %q: restart %q is not on-exit or never", s.Name, s.Restart)
	}
	if s.Buffer <= 0 {
		s.Buffer = DefaultBufferSize
	}
	if s.Backlog <= 0 {
		s.Backlog = DefaultBacklog
	}
	return s, nil
}

// Input returns the buffer for spec, creating it on first use. Two calls with
// equal specs share one buffer; any difference gets a buffer of its own.
func (h *Hub) Input(spec InputSpec) (*Buffer, error) {
	spec, err := spec.Normalize()
	if err != nil {
		return nil, err
	}
	enc, _ := json.Marshal(spec) // map keys are sorted, so equal specs encode equally
	key := "input:" + hashKey(string(enc))

	h.mu.Lock()
	defer h.mu.Unlock()
	if b, ok := h.buffers[key]; ok {
		return b, nil
	}
	b := newBuffer(h, key, spec.Name, spec.Buffer)
	b.start = func() {
		h.wg.Go(func() {
			if spec.Cmd != "" {
				runCmd(h.ctx, spec, b)
			} else {
				tailFile(h.ctx, spec, b)
			}
		})
	}
	h.buffers[key] = b
	return b, nil
}

// Merge returns the buffer that interleaves inputs in seq order, creating it
// on first use and seeding it with the inputs' current rows. Subscribing to it
// starts every input. Inputs must be Input buffers, not other merges.
func (h *Hub) Merge(name string, inputs []*Buffer, size int) (*Buffer, error) {
	if size <= 0 {
		size = DefaultBufferSize
	}
	parts := []string{name, fmt.Sprint(size)}
	for _, in := range inputs {
		if in.merge {
			return nil, fmt.Errorf("merge %q: input %q is itself a merge", name, in.name)
		}
		parts = append(parts, in.key)
	}
	key := "merge:" + hashKey(strings.Join(parts, "\x00"))

	h.mu.Lock()
	defer h.mu.Unlock()
	if b, ok := h.buffers[key]; ok {
		return b, nil
	}
	b := newBuffer(h, key, name, size)
	b.merge = true
	var seed []Row
	for _, in := range inputs {
		in.merges = append(in.merges, b)
		seed = append(seed, in.snapshotLocked()...)
	}
	slices.SortFunc(seed, func(x, y Row) int { return cmp.Compare(x["seq"].(int64), y["seq"].(int64)) })
	for _, r := range seed[max(0, len(seed)-size):] {
		b.appendLocked(r)
	}
	b.start = func() {
		for _, in := range inputs {
			in.startOnce()
		}
	}
	h.buffers[key] = b
	return b, nil
}

// publish stamps rows with seq, source and stream, appends them to b and to
// every merge fed by b, and wakes their subscribers.
func (h *Hub) publish(b *Buffer, stream string, rows ...Row) {
	if len(rows) == 0 {
		return
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.ctx.Err() != nil {
		return
	}
	for _, r := range rows {
		h.seq++
		r["seq"], r["source"], r["stream"] = h.seq, b.name, stream
		b.appendLocked(r)
		for _, m := range b.merges {
			m.appendLocked(r)
		}
	}
	b.wakeLocked()
	for _, m := range b.merges {
		m.wakeLocked()
	}
}

// Close stops every input, kills their process groups and waits for them.
func (h *Hub) Close() error {
	h.mu.Lock()
	h.cancel()
	h.mu.Unlock()
	h.wg.Wait()
	return nil
}

func hashKey(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:8])
}

// Buffer is a ring of the newest rows of one input or merge.
type Buffer struct {
	hub    *Hub
	key    string
	name   string // stamped as "source" on rows published to this buffer
	merge  bool
	ring   []Row
	head   int // index of the oldest row once the ring is full
	size   int
	subs   map[*Subscription]chan struct{}
	merges []*Buffer

	// snap caches the oldest-first view until the next append, so every
	// connection refreshing after one publish shares a single copy.
	snap atomic.Pointer[[]Row]

	start   func()
	started sync.Once
}

func newBuffer(h *Hub, key, name string, size int) *Buffer {
	return &Buffer{hub: h, key: key, name: name, size: size, subs: map[*Subscription]chan struct{}{}}
}

func (b *Buffer) startOnce() { b.started.Do(b.start) }

func (b *Buffer) appendLocked(r Row) {
	b.snap.Store(nil)
	if len(b.ring) < b.size {
		b.ring = append(b.ring, r)
		return
	}
	b.ring[b.head] = r
	b.head = (b.head + 1) % b.size
}

func (b *Buffer) wakeLocked() {
	for _, c := range b.subs {
		select {
		case c <- struct{}{}:
		default: // a wake-up is already pending; the next snapshot covers these rows
		}
	}
}

func (b *Buffer) snapshotLocked() []Row {
	if p := b.snap.Load(); p != nil {
		return *p
	}
	out := make([]Row, 0, len(b.ring))
	out = append(out, b.ring[b.head:]...)
	out = append(out, b.ring[:b.head]...)
	b.snap.Store(&out)
	return out
}

// Snapshot returns the buffered rows, oldest first. The slice and its rows are
// shared with other callers and must not be modified.
func (b *Buffer) Snapshot() []Row {
	b.hub.mu.RLock()
	defer b.hub.mu.RUnlock()
	return b.snapshotLocked()
}

// system publishes a row that reports on the input itself, such as a restart.
func (b *Buffer) system(level, msg string) {
	b.hub.publish(b, "system", makeRow(b.hub.now(), level, msg, msg, false, nil))
}

// Subscription delivers a wake-up on C whenever its buffer gains rows.
type Subscription struct {
	C <-chan struct{}
	b *Buffer
}

// Subscribe starts the input on first use and returns a subscription whose
// channel holds at most one pending wake-up.
func (b *Buffer) Subscribe() *Subscription {
	c := make(chan struct{}, 1)
	s := &Subscription{C: c, b: b}
	b.hub.mu.Lock()
	b.subs[s] = c
	b.hub.mu.Unlock()
	b.startOnce()
	return s
}

// Close stops wake-ups. The input keeps running for other subscribers.
func (s *Subscription) Close() {
	s.b.hub.mu.Lock()
	delete(s.b.subs, s)
	s.b.hub.mu.Unlock()
}
