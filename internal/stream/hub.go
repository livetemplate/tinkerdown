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
	"regexp"
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
	MaxLineMax        = 16 * 1024 * 1024 // the largest line_max

	// DefaultIdleStop is how long an input keeps running with no page
	// subscribed. Short tab closes keep history; an abandoned input (a page
	// closed for good, or a cmd edited under --watch) stops and frees its rows.
	DefaultIdleStop = 5 * time.Minute
)

// Hub owns every Buffer of one server.
type Hub struct {
	mu      sync.RWMutex // guards seq, buffers, private, and every buffer's ring and subscribers
	seq     int64
	buffers map[string]*Buffer
	// private holds values published with a row but kept out of it, by the
	// row's seq. Rows become template state, so a value a page must never see
	// (a filled-in token) lives here; see Private.
	private map[int64]Private

	ctx    context.Context // cancelled by Close
	cancel context.CancelFunc
	wg     sync.WaitGroup
	now    func() time.Time // replaced in tests

	// IdleStop is how long an input runs after its last subscriber leaves.
	IdleStop time.Duration

	// WatchDir, when set before the first Subscribe, makes list inputs (see
	// InputSpec.Every) run again when a file under it changes, in place of
	// their timer.
	WatchDir string
	watchMu  sync.Mutex
	watch    *dirWatch // the one watch over WatchDir, while a list input runs
	watchErr error     // why the watch couldn't start; list inputs keep their timer
}

// NewHub returns an empty hub. Inputs start lazily on their first Subscribe.
func NewHub() *Hub {
	ctx, cancel := context.WithCancel(context.Background())
	return &Hub{buffers: map[string]*Buffer{}, private: map[int64]Private{}, ctx: ctx, cancel: cancel, now: time.Now, IdleStop: DefaultIdleStop}
}

// Private is what an input published alongside a row without putting it in
// the row: Values, or Err saying why it has none.
type Private struct {
	Values map[string]string
	Err    string
}

// Private returns what was published privately with r, by its seq. A keyed
// merge's copy of a row keeps the seq, so it finds the same values.
func (h *Hub) Private(r Row) (Private, bool) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	p, ok := h.private[seqOf(r)]
	return p, ok
}

// InputSpec describes one stream input. Exactly one of Cmd, File, HTTP or
// SQL is set.
type InputSpec struct {
	Name string            `json:"name"` // source name, copied into each row's "source"
	Cmd  string            `json:"cmd"`  // run with /bin/sh -c
	Dir  string            `json:"dir"`  // working directory for Cmd
	Env  map[string]string `json:"env"`  // extra environment for Cmd
	File string            `json:"file"` // absolute path to tail
	HTTP string            `json:"http"` // absolute path of a .http file, re-read whole on change
	SQL  string            `json:"sql"`  // absolute path of a .sql file or a folder of them, re-read on change
	// Manual makes a Cmd start only when Restart is called, never when a page
	// attaches, and run once. A command that changes something (sends a
	// request) must not run again because a page reloaded.
	Manual bool `json:"manual"`
	// LineMax caps a line's length in bytes; 0 means MaxLineBytes.
	LineMax int    `json:"line_max"`
	Format  string `json:"format"`
	Buffer  int    `json:"buffer"`  // rows kept
	Backlog int    `json:"backlog"` // lines read from the end of File on start
	Restart string `json:"restart"` // on-exit or never
	// IdleStop, when set, replaces the hub's IdleStop for this input.
	IdleStop time.Duration `json:"idle_stop"`
	// Every makes a Cmd a list: each run's stdout replaces the rows the last
	// run printed, and the next run starts Every after it exits, or with the
	// hub's WatchDir when a file there changes. RerunLists starts it sooner.
	Every time.Duration `json:"every"`
	// Pattern, a regex with named groups, gives a line that matches it the
	// groups as its fields.
	Pattern string         `json:"pattern"`
	re      *regexp.Regexp // Pattern, compiled by Normalize
}

// Formats lists the accepted values of InputSpec.Format.
var Formats = []string{"auto", "json", "logfmt", "lines"}

// Normalize fills defaults and rejects invalid specs. Input calls it, and
// config validation can call it to report errors before serving.
func (s InputSpec) Normalize() (InputSpec, error) {
	if set := boolInt(s.Cmd != "") + boolInt(s.File != "") + boolInt(s.HTTP != "") + boolInt(s.SQL != ""); set != 1 {
		return s, fmt.Errorf("stream %q: set exactly one of cmd, file, http or sql", s.Name)
	}
	if s.Manual {
		if s.Cmd == "" {
			return s, fmt.Errorf("stream %q: only a cmd can be started by a click", s.Name)
		}
		s.Restart = "never"
	}
	if s.Every != 0 && (s.Cmd == "" || s.Manual || s.Every < 0) {
		return s, fmt.Errorf("stream %q: only a cmd that a page starts can run every %s", s.Name, s.Every)
	}
	if s.Every != 0 && s.Restart == "never" {
		// The source's author said to run it once; a merge, which may come
		// from a page at a URL, can't make it run again and again.
		return s, fmt.Errorf("stream %q: has restart: never, so a merge can't run it every %s", s.Name, s.Every)
	}
	if s.Pattern != "" {
		re, err := CompilePattern(s.Pattern)
		if err != nil {
			return s, fmt.Errorf("stream %q: %v", s.Name, err)
		}
		s.re = re
	}
	switch {
	case s.LineMax == 0:
		s.LineMax = MaxLineBytes
	case s.LineMax < 0 || s.LineMax > MaxLineMax:
		return s, fmt.Errorf("stream %q: line_max %d is not between 1 and %d", s.Name, s.LineMax, MaxLineMax)
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
	b.manual = spec.Manual
	if spec.Every > 0 {
		b.kick = make(chan struct{}, 1)
	}
	b.run = func(ctx context.Context) {
		switch {
		case spec.Every > 0:
			runList(ctx, spec, b)
		case spec.Cmd != "":
			runCmd(ctx, spec, b)
		case spec.HTTP != "":
			readHTTPFile(ctx, spec, b)
		case spec.SQL != "":
			readSQLFiles(ctx, spec, b)
		default:
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
	// Sort orders the rows by a field instead of first-seen order.
	Sort Order
}

// Merge returns the buffer that interleaves inputs in seq order, creating it
// on first use and seeding it with the inputs' current rows. Subscribing to it
// starts every input. Inputs must be Input buffers, not other merges. A keyed
// merge (key.Paths set) keeps one row per key instead of every row.
func (h *Hub) Merge(name string, inputs []*Buffer, size int, key KeySpec) (*Buffer, error) {
	if size <= 0 {
		size = DefaultBufferSize
	}
	parts := []string{name, fmt.Sprint(size), strings.Join(key.Paths, ","), key.Collect, key.Reset.String(), key.Drop.String(), key.Sort.String()}
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
	for _, in := range inputs {
		in.merges = append(in.merges, b)
	}
	b.seedLocked()
	h.buffers[bufKey] = b
	return b, nil
}

// seedLocked fills a merge with its inputs' current rows, in seq order.
func (b *Buffer) seedLocked() {
	var seed []Row
	for _, in := range b.inputs {
		seed = append(seed, in.snapshotLocked()...)
	}
	slices.SortFunc(seed, func(x, y Row) int { return cmp.Compare(x["seq"].(int64), y["seq"].(int64)) })
	if b.keyed == nil {
		seed = seed[max(0, len(seed)-b.size):]
	}
	for _, r := range seed {
		b.appendLocked(r)
	}
}

// publish stamps rows with seq, source and stream, appends them to b and to
// every merge fed by b, and wakes their subscribers. ctx is the run that
// produced them: once Restart or an idle stop has ended that run, its late
// rows are dropped. Checking under h.mu makes that race-free, since both
// cancel under it too.
func (h *Hub) publish(ctx context.Context, b *Buffer, stream string, rows ...Row) {
	if len(rows) == 0 {
		return
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	if ctx.Err() != nil {
		return
	}
	h.appendRowsLocked(b, stream, rows, nil)
}

// replace swaps every row b has published, in b and in the merges it feeds,
// for rows, in one step and one wake-up, so a page gets a single update.
// privs, when set, holds each row's private values, in order.
func (h *Hub) replace(ctx context.Context, b *Buffer, stream string, rows []Row, privs []Private) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if ctx.Err() != nil {
		return
	}
	b.clearLocked()
	h.appendRowsLocked(b, stream, rows, privs)
}

// setErr sets the error b reports in place of its rows, or clears it with "".
func (h *Hub) setErr(ctx context.Context, b *Buffer, msg string) {
	h.mu.RLock()
	same := b.err == msg
	h.mu.RUnlock()
	if same { // a reader re-reports its error on every poll
		return
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	if ctx.Err() != nil || b.err == msg {
		return
	}
	b.err = msg
	b.wakeAllLocked()
}

func (h *Hub) appendRowsLocked(b *Buffer, stream string, rows []Row, privs []Private) {
	for i, r := range rows {
		h.seq++
		r["seq"], r["source"], r["stream"] = h.seq, b.name, stream
		if i < len(privs) {
			h.private[h.seq] = privs[i]
			b.privSeqs = append(b.privSeqs, h.seq)
		}
		b.appendLocked(r)
		for _, m := range b.merges {
			m.appendLocked(r)
		}
	}
	b.wakeAllLocked()
}

// Restart ends the runs of b's manual inputs (b's own, for an input), drops
// the rows they published, and starts each again. It is the only way a manual
// input starts, so a click runs its command exactly once.
func (h *Hub) Restart(b *Buffer) {
	h.mu.Lock()
	defer h.mu.Unlock()
	inputs := b.inputs
	if !b.merge {
		inputs = []*Buffer{b}
	}
	for _, in := range inputs {
		if !in.manual || h.ctx.Err() != nil {
			continue
		}
		if in.stopRun != nil {
			in.stopRun()
			in.stopRun = nil
		}
		in.clearLocked()
		in.startLocked()
	}
}

// RerunLists starts the next run of every list input running now, without
// waiting for its timer or a file change, since a click's command may have
// changed what they print.
func (h *Hub) RerunLists() {
	h.mu.RLock()
	defer h.mu.RUnlock()
	h.rerunListsLocked()
}

func (h *Hub) rerunListsLocked() {
	for _, b := range h.buffers {
		if b.kick != nil && b.stopRun != nil {
			select {
			case b.kick <- struct{}{}:
			default: // a run is already due
			}
		}
	}
}

// settle shows a finished run of a list input on the page, and returns the
// lines the input's rows now hold, for the next run to compare against.
// shown is what the last call returned: nil before the first run.
//
// A failed run keeps the rows, so a brief failure such as a lock held by
// another command doesn't empty the page, and reports its exit status and
// stderr. A run that exits 0 clears the error whatever it wrote to stderr,
// where tools print hints and progress. A run that printed the same lines
// as the rows hold sends nothing.
func (b *Buffer) settle(ctx context.Context, run listRun, shown []string) []string {
	if run.err != nil {
		msg := run.err.Error()
		if s := strings.TrimSpace(run.stderr); s != "" {
			msg += ": " + s
		}
		b.hub.setErr(ctx, b, msg)
		return shown
	}
	b.hub.setErr(ctx, b, "")
	if shown != nil && slices.Equal(run.lines, shown) {
		return shown
	}
	b.hub.replace(ctx, b, "stdout", run.rows(), nil)
	return run.lines
}

// Running reports whether b's input, or any of a merge's inputs, is running
// its command or reading its file.
func (b *Buffer) Running() bool {
	b.hub.mu.RLock()
	defer b.hub.mu.RUnlock()
	if !b.merge {
		return b.running > 0
	}
	return slices.ContainsFunc(b.inputs, func(in *Buffer) bool { return in.running > 0 })
}

// Err returns the error b, or the first of a merge's inputs, reports in place
// of its rows, such as a file that doesn't parse; "" when there is none.
func (b *Buffer) Err() string {
	b.hub.mu.RLock()
	defer b.hub.mu.RUnlock()
	if b.err != "" || !b.merge {
		return b.err
	}
	for _, in := range b.inputs {
		if in.err != "" {
			return in.name + ": " + in.err
		}
	}
	return ""
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
	err    string     // reported in place of rows; see Err
	// privSeqs are the seqs of rows b published with private values, so
	// they go from hub.private along with the rows.
	privSeqs []int64

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
	dropped  bool          // a merge's rows went at an idle stop; reseed on the next holder
	manual   bool          // starts only by Restart; see InputSpec.Manual
	kick     chan struct{} // for a list input: a pending early run; see RerunLists
	running  int           // runs in progress: 1, or briefly 2 while a restarted run winds down
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
	if b.dropped {
		// The rows went at an idle stop; an input that kept running (a
		// longer IdleStop) still has its own, so start from those.
		b.dropped = false
		b.seedLocked()
	}
	if b.run != nil && b.stopRun == nil && !b.manual {
		b.startLocked()
	}
}

// startLocked starts b's run. b.running counts it until it returns, and its
// subscribers wake on both edges, so a page's .Running follows the run.
func (b *Buffer) startLocked() {
	if b.run == nil || b.hub.ctx.Err() != nil {
		return
	}
	ctx, cancel := context.WithCancel(b.hub.ctx)
	b.stopRun = cancel
	b.running++
	b.wakeAllLocked()
	run := b.run
	b.hub.wg.Go(func() {
		defer func() {
			b.hub.mu.Lock()
			b.running--
			b.wakeAllLocked()
			if b.manual {
				// A click's command ended: what lists print may have changed.
				b.hub.rerunListsLocked()
			}
			b.hub.mu.Unlock()
		}()
		// A bug in one input must not take down the server: report it on
		// the input and stop. The next subscriber after IdleStop restarts it.
		defer func() {
			if r := recover(); r != nil {
				b.system(ctx, "error", fmt.Sprintf("input stopped after an internal error: %v", r))
			}
		}()
		run(ctx)
	})
}

// clearLocked drops every row b published, from b and from the merges it
// feeds, with their private values.
func (b *Buffer) clearLocked() {
	b.ring, b.head = nil, 0
	b.snap.Store(nil)
	for _, seq := range b.privSeqs {
		delete(b.hub.private, seq)
	}
	b.privSeqs = nil
	for _, m := range b.merges {
		m.removeSourceLocked(b.name)
	}
}

// removeSourceLocked drops a merge's rows from the input named name.
func (b *Buffer) removeSourceLocked(name string) {
	fromName := func(r Row) bool { return r["source"] == name }
	b.snap.Store(nil)
	if b.keyed != nil {
		b.keyed.rows = slices.DeleteFunc(b.keyed.rows, fromName)
		clear(b.keyed.index)
		b.keyed.reindex(0)
		return
	}
	rows := b.snapshotOrderLocked()
	b.ring, b.head = slices.DeleteFunc(rows, fromName), 0
}

// wakeAllLocked wakes b's subscribers and those of the merges it feeds.
func (b *Buffer) wakeAllLocked() {
	b.wakeLocked()
	for _, m := range b.merges {
		m.wakeLocked()
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
	b.idle = time.AfterFunc(b.idleWait(), func() {
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
		b.clearLocked()
		b.err = ""
		if b.keyed != nil {
			b.keyed.rows, b.keyed.index = nil, map[string]int{}
		}
		b.dropped = b.merge
		if b.clickOnly() {
			// Each click's values make a buffer of their own, which can hold
			// a request body; once idle, nothing reuses it.
			delete(b.hub.buffers, b.key)
		}
	})
}

// clickOnly reports whether b runs only on a click: a manual input, or a
// merge of nothing else.
func (b *Buffer) clickOnly() bool {
	if !b.merge {
		return b.manual
	}
	return len(b.inputs) > 0 && !slices.ContainsFunc(b.inputs, func(in *Buffer) bool { return !in.manual })
}

// idleWait is how long b keeps running and its rows after its last holder
// leaves: its own IdleStop, or the hub's. A merge waits no longer than its
// shortest input, so it drops its rows when they do: an input restarted
// after a stop prints its output again, which would otherwise show twice.
func (b *Buffer) idleWait() time.Duration {
	wait := b.hub.IdleStop
	if b.idleStop > 0 {
		wait = b.idleStop
	}
	for _, in := range b.inputs {
		wait = min(wait, in.idleWait())
	}
	return wait
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
		out = b.snapshotOrderLocked()
	}
	b.snap.Store(&out)
	return out
}

// snapshotOrderLocked copies the ring oldest first.
func (b *Buffer) snapshotOrderLocked() []Row {
	out := make([]Row, 0, len(b.ring))
	out = append(out, b.ring[b.head:]...)
	return append(out, b.ring[:b.head]...)
}

// Snapshot returns the buffered rows, oldest first. The slice and its rows are
// shared with other callers and must not be modified.
func (b *Buffer) Snapshot() []Row {
	b.hub.mu.RLock()
	defer b.hub.mu.RUnlock()
	return b.snapshotLocked()
}

// system publishes a row that reports on the input itself, such as a restart.
func (b *Buffer) system(ctx context.Context, level, msg string) {
	b.hub.publish(ctx, b, "system", makeRow(b.hub.now(), level, msg, msg, false, nil))
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
