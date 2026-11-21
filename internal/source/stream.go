package source

import (
	"context"
	"fmt"
	"maps"
	"net/url"
	"path/filepath"
	"reflect"
	"slices"
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

// StreamSource is a Source backed by a stream.Buffer shared with every other
// page showing the same input. Fetch returns the buffer's current rows.
//
// A merge with params is gated: with every param in the page's query, it
// watches its gate merges and shows rows only while each value is in its
// gate. Its own buffer is built the first time every gate opens, so a value
// never listed creates nothing in the hub, and it holds that buffer (running
// its inputs) only while every gate is open.
type StreamSource struct {
	name  string
	hub   *stream.Hub
	buf   *stream.Buffer
	flush time.Duration
	gates []gate

	mu       sync.Mutex
	build    func() (*stream.Buffer, error) // a gated source's buffer, on first open
	sub      *stream.Subscription           // the page's subscription, once Subscribe ran
	attached bool                           // buf is attached to sub
}

// gate is one params value and the merge that must list it.
type gate struct {
	buf   *stream.Buffer
	match stream.Match
}

// MergeParamIdle is how long a params merge's inputs run after the last page
// showing them leaves: each value opened starts its own commands.
const MergeParamIdle = 30 * time.Second

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
	if s.gates == nil {
		if s.buf == nil {
			return nil, nil
		}
		return s.buf.Snapshot(), nil
	}
	open := s.open()
	s.mu.Lock()
	err := s.syncLocked(open)
	buf := s.buf
	s.mu.Unlock()
	if err != nil || !open {
		return nil, err
	}
	return buf.Snapshot(), nil
}

// syncLocked builds the buffer on the first open, and attaches it to the
// page's subscription while open, detaching it once closed. s.mu is held.
func (s *StreamSource) syncLocked(open bool) error {
	if open && s.buf == nil {
		buf, err := s.build()
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

// open reports whether every gate's merge has a row holding its value.
func (s *StreamSource) open() bool {
	for _, g := range s.gates {
		if !slices.ContainsFunc(g.buf.Snapshot(), g.match.Matches) {
			return false
		}
	}
	return true
}

// Close is a no-op: the input belongs to the hub and outlives one page.
func (s *StreamSource) Close() error { return nil }

// Subscribe starts the input if needed and wakes the caller on new rows. A
// gated source wakes on its gates' rows too, and starts its own input only
// while its gates are open; it returns nil when the query lacks a param.
func (s *StreamSource) Subscribe() *stream.Subscription {
	if s.gates == nil {
		if s.buf == nil {
			return nil
		}
		return s.buf.Subscribe()
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sub = s.hub.NewSubscription()
	for _, g := range s.gates {
		s.sub.Attach(g.buf)
	}
	_ = s.syncLocked(s.open()) // a build error reaches the page through Fetch
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

	src := &StreamSource{name: name, hub: hub, flush: flush}
	if cfg.Type == "stream" {
		if src.buf, err = streamInput(hub, name, cfg, dir, nil, 0); err != nil {
			return nil, err
		}
		return src, nil
	}
	var env map[string]string
	idle := time.Duration(0)
	if len(cfg.Params) > 0 {
		if page == nil || page.Resolve == nil || page.InputLookup == nil {
			return nil, fmt.Errorf("merge source %q: params need a page", name)
		}
		env, idle = map[string]string{}, MergeParamIdle
		for _, p := range slices.Sorted(maps.Keys(cfg.Params)) {
			spec := cfg.Params[p]
			v := page.Query.Get(p)
			if v == "" {
				return src, nil // no value yet: no rows, nothing runs
			}
			if err := config.CheckTDValue(p, v); err != nil {
				return nil, &ValidationError{Source: name, Field: "params", Reason: err.Error()}
			}
			env["TD_"+p] = v
			gateName, path := spec.Gate()
			gateCfg, _ := page.Resolve(gateName)
			// The gate this merge was validated against is the one lookup
			// resolves; the page's own resolution must be that same merge,
			// or a page could swap in a gate listing any value.
			if want, _ := lookup(gateName); !sameMerge(want, gateCfg) {
				return nil, &ValidationError{Source: name, Field: "params", Reason: fmt.Sprintf("param %s: gate %q is not the merge this source was checked against", p, gateName)}
			}
			g, err := NewStreamSource(hub, gateName, gateCfg, siteDir, page.InputLookup(gateName), nil)
			if err != nil {
				return nil, fmt.Errorf("merge source %q: param %s: %w", name, p, err)
			}
			src.gates = append(src.gates, gate{buf: g.buf, match: stream.Match{Path: path, Values: []string{v}}})
		}
	}
	build := func() (*stream.Buffer, error) {
		inputs := make([]*stream.Buffer, 0, len(cfg.Inputs))
		for _, in := range cfg.Inputs {
			inCfg, _ := lookup(in) // ValidateStreamSource checked it exists and is a stream
			b, err := streamInput(hub, in, inCfg, dir, env, idle)
			if err != nil {
				return nil, fmt.Errorf("merge source %q: %w", name, err)
			}
			inputs = append(inputs, b)
		}
		return hub.Merge(name, inputs, cfg.Buffer, cfg.KeySpec())
	}
	if src.gates != nil {
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
		return src, nil
	}
	if src.buf, err = build(); err != nil {
		return nil, err
	}
	return src, nil
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
// the input a buffer of its own. idle, when set, replaces the hub's idle stop.
func streamInput(hub *stream.Hub, name string, cfg config.SourceConfig, dir string, params map[string]string, idle time.Duration) (*stream.Buffer, error) {
	spec := cfg.StreamSpec(name)
	if spec.Cmd != "" {
		if err := requireExec(name, "cmd", "stream sources that run a command"); err != nil {
			return nil, err
		}
		spec.Dir = dir
		spec.Env = expandEnv(spec.Env)
		maps.Copy(spec.Env, params)
	} else {
		spec.File = resolveSitePath(dir, spec.File)
	}
	spec.IdleStop = idle
	return hub.Input(spec)
}
