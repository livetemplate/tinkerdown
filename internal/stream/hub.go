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

	// DefaultIdleStop is how long an input keeps running with no page
	// subscribed. Short tab closes keep history; an abandoned input (a page
	// closed for good, or a cmd edited under --watch) stops and frees its rows.
	DefaultIdleStop = 5 * time.Minute
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

	// IdleStop is how long an input runs after its last subscriber leaves.
	IdleStop time.Duration
}

// NewHub returns an empty hub. Inputs start lazily on their first Subscribe.
func NewHub() *Hub {
	ctx, cancel := context.WithCancel(context.Background())
	return &Hub{buffers: map[string]*Buffer{}, ctx: ctx, cancel: cancel, now: time.Now, IdleStop: DefaultIdleStop}
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
	// IdleStop, when set, replaces the hub's IdleStop for this input.
	IdleStop time.Duration `json:"idle_stop"`
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
	b.idleStop = spec.IdleStop
	b.run = func(ctx context.Context) {
		if spec.Cmd != "" {
			runCmd(ctx, spec, b)
		} else {
			tailFile(ctx, spec, b)
		}
	}
	h.buffers[key] = b
	return b, nil
}

// KeySpec makes a merge keyed: one row per distinct value of Paths, updated
// in place. Collect names a field whose text is gathered across a key's rows.
type KeySpec struct {
	Paths   []string
	Collect string
	// Reset starts a key's collected text over when a row matches, such as a
	// test's next run.
	Reset Match
	// Drop removes a key's row when a row matches, such as a removed
	// container. The matching row is not stored.
	Drop Match
}

// Merge returns the buffer that interleaves inputs in seq order, creating it
// on first use and seeding it with the inputs' current rows. Subscribing to it
// starts every input. Inputs must be Input buffers, not other merges. A keyed
// merge (key.Paths set) keeps one row per key instead of every row.
func (h *Hub) Merge(name string, inputs []*Buffer, size int, key KeySpec) (*Buffer, error) {
	if size <= 0 {
		size = DefaultBufferSize
	}
	parts := []string{name, fmt.Sprint(size), strings.Join(key.Paths, ","), key.Collect, key.Reset.String(), key.Drop.String()}
	for _, in := range inputs {
		if in.merge {
			return nil, fmt.Errorf("merge %q: input %q is itself a merge", name, in.name)
		}
		parts = append(parts, in.key)
	}
	bufKey := "merge:" + hashKey(strings.Join(parts, "\x00"))

	h.mu.Lock()
	defer h.mu.Unlock()
	if b, ok := h.buffers[bufKey]; ok {
		return b, nil
	}
	b := newBuffer(h, bufKey, name, size)
	b.merge = true
	b.inputs = inputs
	if len(key.Paths) > 0 {
		b.keyed = &keyedRows{spec: key, index: map[string]int{}}
	}
	var seed []Row
	for _, in := range inputs {
		in.merges = append(in.merges, b)
		seed = append(seed, in.snapshotLocked()...)
	}
	slices.SortFunc(seed, func(x, y Row) int { return cmp.Compare(x["seq"].(int64), y["seq"].(int64)) })
	if b.keyed == nil {
		seed = seed[max(0, len(seed)-size):]
	}
	for _, r := range seed {
		b.appendLocked(r)
	}
	h.buffers[bufKey] = b
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
	merges []*Buffer  // merges this input feeds
	inputs []*Buffer  // for a merge: the inputs it holds while subscribed
	keyed  *keyedRows // for a keyed merge: one row per key, in place of ring

	// snap caches the oldest-first view until the next append, so every
	// connection refreshing after one publish shares a single copy.
	snap atomic.Pointer[[]Row]

	// Lifecycle, guarded by hub.mu. refs counts subscribers plus the merges
	// holding this input; the input runs while refs > 0 and for IdleStop after.
	run      func(ctx context.Context) // nil for a merge
	refs     int
	stopRun  context.CancelFunc // non-nil while the input runs
	idle     *time.Timer
	idleSeen int           // refs epoch, so a stale idle timer doesn't stop a re-used input
	idleStop time.Duration // overrides hub.IdleStop when set
}

func newBuffer(h *Hub, key, name string, size int) *Buffer {
	return &Buffer{hub: h, key: key, name: name, size: size, subs: map[*Subscription]chan struct{}{}}
}

// retainLocked counts one more holder and starts the input if it is stopped.
// A merge passes the hold on to its inputs.
func (b *Buffer) retainLocked() {
	b.refs++
	b.idleSeen++
	if b.idle != nil {
		b.idle.Stop()
		b.idle = nil
	}
	for _, in := range b.inputs {
		in.retainLocked()
	}
	if b.run != nil && b.stopRun == nil && b.hub.ctx.Err() == nil {
		ctx, cancel := context.WithCancel(b.hub.ctx)
		b.stopRun = cancel
		run := b.run
		b.hub.wg.Go(func() {
			// A bug in one input must not take down the server: report it on
			// the input and stop. The next subscriber after IdleStop restarts it.
			defer func() {
				if r := recover(); r != nil {
					b.system("error", fmt.Sprintf("input stopped after an internal error: %v", r))
				}
			}()
			run(ctx)
		})
	}
}

// releaseLocked drops one holder. After IdleStop with no holder, the input
// stops and its rows are dropped; a merge just drops its rows.
func (b *Buffer) releaseLocked() {
	b.refs--
	for _, in := range b.inputs {
		in.releaseLocked()
	}
	if b.refs > 0 {
		return
	}
	epoch := b.idleSeen
	wait := b.hub.IdleStop
	if b.idleStop > 0 {
		wait = b.idleStop
	}
	b.idle = time.AfterFunc(wait, func() {
		b.hub.mu.Lock()
		defer b.hub.mu.Unlock()
		if b.refs > 0 || b.idleSeen != epoch {
			return
		}
		b.idle = nil
		if b.stopRun != nil {
			b.stopRun()
			b.stopRun = nil
		}
		b.ring, b.head = nil, 0
		if b.keyed != nil {
			b.keyed.rows, b.keyed.index = nil, map[string]int{}
		}
		b.snap.Store(nil)
	})
}

func (b *Buffer) appendLocked(r Row) {
	b.snap.Store(nil)
	if b.keyed != nil {
		b.keyed.add(r, b.size)
		return
	}
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
	var out []Row
	if b.keyed != nil {
		out = slices.Clone(b.keyed.rows)
	} else {
		out = make([]Row, 0, len(b.ring))
		out = append(out, b.ring[b.head:]...)
		out = append(out, b.ring[:b.head]...)
	}
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
	C   <-chan struct{}
	c   chan struct{}
	hub *Hub
	// bufs are the buffers that wake C; guarded by hub.mu. Each is held
	// (and so running) while attached.
	bufs   []*Buffer
	closed bool
}

// Subscribe starts the input if it isn't running and returns a subscription
// whose channel holds at most one pending wake-up.
func (b *Buffer) Subscribe() *Subscription {
	s := b.hub.NewSubscription()
	s.Attach(b)
	return s
}

// NewSubscription returns a subscription attached to no buffer. Attach adds
// buffers whose rows should wake it.
func (h *Hub) NewSubscription() *Subscription {
	c := make(chan struct{}, 1)
	return &Subscription{C: c, c: c, hub: h}
}

// Attach makes b's new rows wake s, and holds b, starting its input if it is
// stopped. Attaching a buffer twice, or after Close, does nothing.
func (s *Subscription) Attach(b *Buffer) {
	s.hub.mu.Lock()
	defer s.hub.mu.Unlock()
	if s.closed || slices.Contains(s.bufs, b) {
		return
	}
	s.bufs = append(s.bufs, b)
	b.subs[s] = s.c
	b.retainLocked()
}

// Detach stops b waking s and releases it; the input stops after its idle
// time if nothing else holds it. Detaching a buffer not attached does nothing.
func (s *Subscription) Detach(b *Buffer) {
	s.hub.mu.Lock()
	defer s.hub.mu.Unlock()
	s.detachLocked(b)
}

func (s *Subscription) detachLocked(b *Buffer) {
	i := slices.Index(s.bufs, b)
	if i < 0 {
		return
	}
	s.bufs = slices.Delete(s.bufs, i, i+1)
	delete(b.subs, s)
	b.releaseLocked()
}

// Close stops wake-ups. The inputs keep running for other subscribers, and
// for their idle time after the last one leaves. Close is idempotent.
func (s *Subscription) Close() {
	s.hub.mu.Lock()
	defer s.hub.mu.Unlock()
	for len(s.bufs) > 0 {
		s.detachLocked(s.bufs[0])
	}
	s.closed = true
}
