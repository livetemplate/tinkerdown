package source

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"maps"
	"net/url"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/livetemplate/tinkerdown/internal/config"
	"github.com/livetemplate/tinkerdown/internal/stream"
)

// Followable is a source that grows on its own. The server subscribes to it
// and pushes new rows to the page at most every FlushInterval.
type Followable interface {
	Source
	Subscribe() *stream.Subscription
	FlushInterval() time.Duration
}

// QuerySetter is a source whose rows depend on the page URL's query.
// SetQuery reports whether the values it reads changed, so the page must
// refresh. Params returns the values it accepted, by param name.
type QuerySetter interface {
	SetQuery(url.Values) bool
	Params() map[string]string
}

// Clicker is a source whose inputs a click starts: a merge with on. On names
// the button; Click starts the run for the button's values.
type Clicker interface {
	On() string
	Click(values map[string]interface{}) error
}

// Runner is a source that reports whether its inputs are running.
type Runner interface {
	Running() bool
}

var (
	_ QuerySetter = (*StreamSource)(nil)
	_ Clicker     = (*StreamSource)(nil)
	_ Runner      = (*StreamSource)(nil)
)

// MaxClickValue caps one value a click passes to a command from a row's
// private values, counted as the whole TD_NAME=value environment string with
// its terminating NUL: Linux refuses a longer one (MAX_ARG_STRLEN).
const MaxClickValue = 128 * 1024

// ClickGap is the least time between two runs a page's clicks start. A click
// sooner than that runs when the gap ends, and a later click in the gap
// replaces it, so clicks sent in a loop restart a command, and repeat what it
// does, at most once per gap, and the last click is never lost.
const ClickGap = 250 * time.Millisecond

// StreamSource is a Source backed by a stream.Buffer shared with every other
// page showing the same input. Fetch returns the buffer's current rows.
//
// A merge with params is gated: it watches its gate merges and, with every
// param in the page's query, shows rows only while each value is in its gate.
// Its own buffer is built the first time every gate opens, so a value never
// listed creates nothing in the hub, and it holds that buffer (running its
// inputs) only while every gate is open. SetQuery changes the values.
type StreamSource struct {
	name  string
	hub   *stream.Hub
	buf   *stream.Buffer
	flush time.Duration

	mu       sync.Mutex
	gates    []gate                                              // one per param with in, sorted by name
	typed    []string                                            // the params a click's typed values fill, sorted
	build    func(env map[string]string) (*stream.Buffer, error) // a gated source's buffer, on first open
	env      map[string]string                                   // TD_ values; nil while a param is missing or refused
	params   map[string]string                                   // env by param name, without TD_; nil with env
	valueErr error                                               // why the query's values, or the last click's, were refused
	on       string                                              // the button that starts the inputs; "" when the URL does
	sub      *stream.Subscription                                // the page's subscription, once Subscribe ran
	attached bool                                                // buf is attached to sub
	clicked  time.Time                                           // when a click last started a run
	later    *time.Timer                                         // runs laterValues when the gap ends
	laterVal map[string]interface{}                              // the latest click inside the gap
	closed   bool                                                // the page left; no deferred click runs
}

// gate is one param and the merge that must list its value at path.
type gate struct {
	param string
	buf   *stream.Buffer
	path  string
}

// MergeIdle is how long a params merge's inputs, and an every merge's,
// run after the last page showing them leaves: each value opened starts its
// own commands, and a list keeps no history worth holding.
const MergeIdle = 30 * time.Second

// Page is what a gated merge needs from the page showing it: the query its
// params come from, and how the page resolves a gate merge and that merge's
// inputs. A gate built the way the gate's own block is built shares its
// buffer, so the gate's commands run once.
type Page struct {
	Query       url.Values
	Resolve     func(name string) (config.SourceConfig, bool)
	InputLookup func(merge string) func(string) (config.SourceConfig, bool)
}

// Name returns the source identifier.
func (s *StreamSource) Name() string { return s.name }

// Fetch returns the buffered rows, oldest first. The rows are shared and must
// not be modified. A gated source returns no rows while a gate is closed, and
// attaches or detaches its buffer to match. That works because a gate's new
// rows wake the page's subscription, and every wake-up refreshes, so Fetch
// runs each time a gate may have opened or closed.
func (s *StreamSource) Fetch(ctx context.Context) ([]map[string]interface{}, error) {
	if !s.gated() { // buf never changes without params
		if s.buf == nil {
			return nil, nil
		}
		if msg := s.buf.Err(); msg != "" {
			// The page keeps its last rows and shows why they aren't current.
			return nil, &ValidationError{Source: s.name, Field: "input", Reason: msg}
		}
		return s.buf.Snapshot(), nil
	}
	s.mu.Lock()
	if s.valueErr != nil {
		s.mu.Unlock()
		return nil, s.valueErr
	}
	open := s.openLocked()
	err := s.syncLocked(open)
	buf := s.buf
	s.mu.Unlock()
	if err != nil || !open {
		return nil, err
	}
	if msg := buf.Err(); msg != "" {
		return nil, &ValidationError{Source: s.name, Field: "input", Reason: msg}
	}
	return buf.Snapshot(), nil
}

// SetQuery takes a gated source's values from the page's query, and reports
// whether they changed. New values let go of the earlier values' buffer,
// which stops once no page shows it. A refused value is returned by the next
// Fetch. A source without params ignores the query.
func (s *StreamSource) SetQuery(q url.Values) bool {
	// A merge started by a click ignores the URL: a reload, Back or a shared
	// link must not show, or start, a run nobody clicked here.
	if !s.gated() || s.on != "" {
		return false
	}
	var env map[string]string // stays nil while a value is missing or refused
	var err error
	for _, g := range s.gates {
		v := q.Get(g.param)
		if v == "" {
			env = nil
			break
		}
		if e := config.CheckTDValue(g.param, v); e != nil {
			env, err = nil, &ValidationError{Source: s.name, Field: "params", Reason: e.Error()}
			break
		}
		if env == nil {
			env = map[string]string{}
		}
		env["TD_"+g.param] = v
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if maps.Equal(env, s.env) && errText(err) == errText(s.valueErr) {
		return false
	}
	if s.attached {
		s.sub.Detach(s.buf)
	}
	s.buf, s.attached, s.env, s.valueErr = nil, false, env, err
	s.params = nil
	if env != nil {
		s.params = make(map[string]string, len(env))
		for k, v := range env {
			s.params[strings.TrimPrefix(k, "TD_")] = v
		}
	}
	return true
}

// Params returns the query's values for a gated source's params, by name,
// once every one is present and passes the value rules; nil otherwise.
// The map is shared and must not be modified.
func (s *StreamSource) Params() map[string]string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.params
}

func errText(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

// syncLocked builds the buffer on the first open, and attaches it to the
// page's subscription while open, detaching it once closed. s.mu is held.
func (s *StreamSource) syncLocked(open bool) error {
	if open && s.buf == nil {
		buf, err := s.build(s.env)
		if err != nil {
			return err
		}
		s.buf = buf
	}
	if s.sub != nil && s.buf != nil && open != s.attached {
		if open {
			s.sub.Attach(s.buf)
		} else {
			s.sub.Detach(s.buf)
		}
		s.attached = open
	}
	return nil
}

// openLocked reports whether the query has every value and every gate's
// merge has a row holding its value. s.mu is held.
func (s *StreamSource) openLocked() bool {
	if s.env == nil {
		return false
	}
	for _, g := range s.gates {
		if _, err := g.find(s.env["TD_"+g.param]); err != nil {
			return false
		}
	}
	return true
}

// gated reports whether the merge has params, so its buffer depends on
// their values.
func (s *StreamSource) gated() bool { return len(s.gates) > 0 || len(s.typed) > 0 }

// On returns the name of the button whose click starts the inputs, or "".
func (s *StreamSource) On() string { return s.on }

// Click starts the inputs for a button's values. Each param must pass the
// value rules and be in its gate merge, as a URL value must. The gate row it
// names adds its private values (stream.Hub.Private) as TD_ variables, so a
// command gets what the row stands for without the browser sending it. The
// run is shared by everyone who clicks the same values, and a click while it
// runs starts it again. A click within ClickGap of the last run waits for
// the gap to end. A refused click is reported by the next Fetch.
func (s *StreamSource) Click(values map[string]interface{}) error {
	if s.on == "" {
		return fmt.Errorf("source %q is not started by a click", s.name)
	}
	env, params, err := s.clickEnv(values)
	s.mu.Lock()
	if s.closed { // the page left, maybe while a deferred click was firing
		s.mu.Unlock()
		return nil
	}
	if err != nil {
		s.valueErr = &ValidationError{Source: s.name, Field: "click", Reason: err.Error()}
		s.mu.Unlock()
		return s.valueErr
	}
	now := time.Now()
	if wait := ClickGap - now.Sub(s.clicked); wait > 0 {
		s.laterVal = values
		if s.later == nil {
			s.later = time.AfterFunc(wait, s.clickLater)
		}
		s.mu.Unlock()
		return nil
	}
	s.clicked = now
	if s.buf == nil || !maps.Equal(env, s.env) { // the same values keep their run's buffer
		if s.attached {
			s.sub.Detach(s.buf)
		}
		s.buf, s.attached, s.env, s.params = nil, false, env, params
	}
	s.valueErr = nil
	err = s.syncLocked(true)
	buf := s.buf
	s.mu.Unlock()
	if err != nil {
		return err
	}
	s.hub.Restart(buf)
	return nil
}

// clickLater runs the click that came inside the gap. Its run wakes the
// page, as any run does, and Click does nothing once the page has closed.
func (s *StreamSource) clickLater() {
	s.mu.Lock()
	values := s.laterVal
	s.later, s.laterVal = nil, nil
	s.mu.Unlock()
	s.Click(values) // a refusal is in valueErr, which the next Fetch reports
}

// clickEnv checks a click's values and builds the run's TD_ variables: each
// param's value, plus the private values of the gate row it names.
func (s *StreamSource) clickEnv(values map[string]interface{}) (env, params map[string]string, err error) {
	env, params = map[string]string{}, map[string]string{}
	for _, g := range s.gates {
		v, err := config.TDValue(values, g.param)
		if err != nil {
			return nil, nil, err
		}
		row, err := g.find(v)
		if err != nil {
			return nil, nil, err
		}
		if p, ok := s.hub.Private(row); ok {
			if p.Err != "" {
				return nil, nil, errors.New(p.Err)
			}
			for _, k := range slices.Sorted(maps.Keys(p.Values)) {
				pv := p.Values[k]
				switch {
				case !config.TDName.MatchString(k):
					return nil, nil, fmt.Errorf("value name %q is not 1-32 letters, digits or _", k)
				case len("TD_"+k+"=")+len(pv)+1 > MaxClickValue:
					return nil, nil, fmt.Errorf("value %s is over %d KiB", k, MaxClickValue/1024)
				case strings.IndexByte(pv, 0) >= 0:
					return nil, nil, fmt.Errorf("value %s holds a NUL byte", k)
				}
				env["TD_"+k] = pv
			}
		}
		env["TD_"+g.param], params[g.param] = v, v
	}
	for _, p := range s.typed {
		v, err := config.TDValue(values, p)
		if err != nil {
			return nil, nil, err
		}
		env["TD_"+p], params[p] = v, v
	}
	return env, params, nil
}

// Running reports whether the source's inputs are running.
func (s *StreamSource) Running() bool {
	s.mu.Lock()
	buf := s.buf
	s.mu.Unlock()
	return buf != nil && buf.Running()
}

// Close drops a click waiting for its gap. The input belongs to the hub and
// outlives one page, so Close leaves it running.
func (s *StreamSource) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.closed = true
	if s.later != nil {
		s.later.Stop()
		s.later, s.laterVal = nil, nil
	}
	return nil
}

// Subscribe starts the input if needed and wakes the caller on new rows. A
// gated source wakes on its gates' rows too, and starts its own input only
// while its gates are open, even before the query has its values.
func (s *StreamSource) Subscribe() *stream.Subscription {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.gated() {
		if s.buf == nil {
			return nil
		}
		return s.buf.Subscribe()
	}
	s.sub = s.hub.NewSubscription()
	for _, g := range s.gates {
		s.sub.Attach(g.buf)
	}
	_ = s.syncLocked(s.openLocked()) // a build error reaches the page through Fetch
	return s.sub
}

// FlushInterval is the minimum time between pushes to one page.
func (s *StreamSource) FlushInterval() time.Duration { return s.flush }

// NewStreamSource builds a stream or merge source on hub. lookup resolves the
// stream sources a merge lists in inputs.
//
// page carries the query and resolver a merge with params needs; it may be
// nil for any other source.
func NewStreamSource(hub *stream.Hub, name string, cfg config.SourceConfig, siteDir string, lookup func(string) (config.SourceConfig, bool), page *Page) (*StreamSource, error) {
	if hub == nil {
		return nil, fmt.Errorf("%s source %q: needs a running server", cfg.Type, name)
	}
	if err := config.ValidateStreamSource(name, cfg, lookup); err != nil {
		return nil, err
	}
	flush, _ := cfg.FlushInterval() // ValidateStreamSource checked it
	dir, err := filepath.Abs(siteDir)
	if err != nil {
		return nil, err
	}

	lineMax, _ := config.ParseLineMax(cfg.LineMax) // ValidateStreamSource checked it
	src := &StreamSource{name: name, hub: hub, flush: flush, on: cfg.On}
	if cfg.Type == "stream" {
		if src.buf, err = streamInput(hub, name, cfg, dir, nil, inputOpts{}); err != nil {
			return nil, err
		}
		return src, nil
	}
	every, _ := cfg.EveryInterval() // ValidateStreamSource checked it
	idle := time.Duration(0)
	if every > 0 || len(cfg.Params) > 0 {
		idle = MergeIdle
	}
	if len(cfg.Params) > 0 {
		if page == nil || page.Resolve == nil || page.InputLookup == nil {
			return nil, fmt.Errorf("merge source %q: params need a page", name)
		}
		for _, p := range slices.Sorted(maps.Keys(cfg.Params)) {
			if cfg.Params[p].Typed {
				src.typed = append(src.typed, p)
				continue
			}
			gateName, _ := cfg.Params[p].Gate()
			gateCfg, _ := page.Resolve(gateName)
			// The gate this merge was validated against is the one lookup
			// resolves; the page's own resolution must be that same merge,
			// or a page could swap in a gate listing any value.
			if want, _ := lookup(gateName); !sameMerge(want, gateCfg) {
				return nil, &ValidationError{Source: name, Field: "params", Reason: fmt.Sprintf("param %s: gate %q is not the merge this source was checked against", p, gateName)}
			}
			g, err := newGate(hub, page, siteDir, p, cfg.Params[p])
			if err != nil {
				return nil, fmt.Errorf("merge source %q: param %s: %w", name, p, err)
			}
			src.gates = append(src.gates, g)
		}
	}
	build := func(env map[string]string) (*stream.Buffer, error) {
		inputs := make([]*stream.Buffer, 0, len(cfg.Inputs))
		for _, in := range cfg.Inputs {
			inCfg, _ := lookup(in) // ValidateStreamSource checked it exists and is a stream
			b, err := streamInput(hub, in, inCfg, dir, env, inputOpts{idle: idle, lineMax: lineMax, manual: cfg.On != "", every: every, pattern: cfg.Pattern})
			if err != nil {
				return nil, fmt.Errorf("merge source %q: %w", name, err)
			}
			inputs = append(inputs, b)
		}
		return hub.Merge(name, inputs, cfg.Buffer, cfg.KeySpec())
	}
	if src.gated() {
		// Check the inputs now, so a misconfigured input fails at page load
		// rather than when the first value opens.
		for _, in := range cfg.Inputs {
			if inCfg, _ := lookup(in); inCfg.Cmd != "" {
				if err := requireExec(in, "cmd", "stream sources that run a command"); err != nil {
					return nil, err
				}
			}
		}
		src.build = build
		src.SetQuery(page.Query) // a click merge ignores it
		return src, nil
	}
	if src.buf, err = build(nil); err != nil {
		return nil, err
	}
	return src, nil
}

// Listed checks a click value against in (MERGE.PATH): the value must pass
// the value rules and equal PATH in a row of MERGE, as the page resolves
// MERGE, so it is the buffer the page's own block shows.
func Listed(hub *stream.Hub, page *Page, siteDir, name string, in config.SourceParam, value string) error {
	if hub == nil || page == nil || page.Resolve == nil || page.InputLookup == nil {
		return fmt.Errorf("value %s: checking it needs a page", name)
	}
	if err := config.CheckIn(in, "", page.Resolve); err != nil {
		return fmt.Errorf("value %s: %v", name, err)
	}
	g, err := newGate(hub, page, siteDir, name, in)
	if err != nil {
		return fmt.Errorf("value %s: %w", name, err)
	}
	_, err = g.find(value)
	return err
}

// newGate resolves in's merge as the page does, so the gate shares the
// buffer the page's own block shows.
func newGate(hub *stream.Hub, page *Page, siteDir, param string, in config.SourceParam) (gate, error) {
	gateName, path := in.Gate()
	gateCfg, _ := page.Resolve(gateName)
	g, err := NewStreamSource(hub, gateName, gateCfg, siteDir, page.InputLookup(gateName), nil)
	if err != nil {
		return gate{}, err
	}
	return gate{param: param, buf: g.buf, path: path}, nil
}

// find returns the gate merge's row whose field at path is value.
func (g gate) find(value string) (stream.Row, error) {
	m := stream.Match{Path: g.path, Values: []string{value}}
	rows := g.buf.Snapshot()
	if i := slices.IndexFunc(rows, m.Matches); i >= 0 {
		return rows[i], nil
	}
	return nil, fmt.Errorf("value %s is not listed", g.param)
}

// sameMerge reports whether got, a merge as the page resolves it, is the
// merge want describes. A ["*"] merge resolves to the inputs found, so its
// inputs are left out of the comparison.
func sameMerge(want, got config.SourceConfig) bool {
	if want.WildcardInputs() {
		got.Inputs, got.Wildcard = want.Inputs, want.Wildcard
	}
	return reflect.DeepEqual(want, got)
}

// streamInput resolves a stream source's command or file against dir (the
// absolute site directory) and returns its hub buffer. A command runs from dir,
// like exec, so ./script.sh means the script next to tinkerdown.yaml. A file
// may be absolute: like the json and csv sources, a stream reads any file the
// server's user can.
//
// params, from a params merge, are TD_ variables added to a command's
// environment after $VAR expansion, so a value is never expanded; they give
// the input a buffer of its own. opts are what the merge sets on its inputs.
func streamInput(hub *stream.Hub, name string, cfg config.SourceConfig, dir string, params map[string]string, opts inputOpts) (*stream.Buffer, error) {
	spec := cfg.StreamSpec(name)
	spec.LineMax, spec.Manual, spec.Every, spec.Pattern = opts.lineMax, opts.manual, opts.every, opts.pattern
	switch {
	case spec.HTTP != "":
		spec.HTTP = resolveSitePath(dir, spec.HTTP)
	case spec.SQL != "":
		spec.SQL = resolveSitePath(dir, spec.SQL)
	case spec.Cmd != "":
		if err := requireExec(name, "cmd", "stream sources that run a command"); err != nil {
			return nil, err
		}
		spec.Dir = cmp.Or(cfg.Dir, dir)
		spec.Env = expandEnv(spec.Env)
		maps.Copy(spec.Env, params)
	default:
		spec.File = resolveSitePath(dir, spec.File)
	}
	spec.IdleStop = opts.idle
	return hub.Input(spec)
}

// inputOpts are what a merge sets on the inputs it starts.
type inputOpts struct {
	idle    time.Duration // replaces the hub's idle stop when set
	lineMax int           // replaces the 16 KiB line cap when set
	manual  bool          // a click starts the input (on)
	every   time.Duration // reruns the input's command, replacing its rows (every)
	pattern string        // named groups that become a line's fields (pattern)
}
