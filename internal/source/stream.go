package source

import (
	"context"
	"fmt"
	"path/filepath"
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
type StreamSource struct {
	name  string
	buf   *stream.Buffer
	flush time.Duration
}

// Name returns the source identifier.
func (s *StreamSource) Name() string { return s.name }

// Fetch returns the buffered rows, oldest first. The rows are shared and must
// not be modified.
func (s *StreamSource) Fetch(ctx context.Context) ([]map[string]interface{}, error) {
	return s.buf.Snapshot(), nil
}

// Close is a no-op: the input belongs to the hub and outlives one page.
func (s *StreamSource) Close() error { return nil }

// Subscribe starts the input if needed and wakes the caller on new rows.
func (s *StreamSource) Subscribe() *stream.Subscription { return s.buf.Subscribe() }

// FlushInterval is the minimum time between pushes to one page.
func (s *StreamSource) FlushInterval() time.Duration { return s.flush }

// NewStreamSource builds a stream or merge source on hub. lookup resolves the
// stream sources a merge lists in inputs.
func NewStreamSource(hub *stream.Hub, name string, cfg config.SourceConfig, siteDir string, lookup func(string) (config.SourceConfig, bool)) (*StreamSource, error) {
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

	var buf *stream.Buffer
	if cfg.Type == "stream" {
		buf, err = streamInput(hub, name, cfg, dir)
	} else {
		inputs := make([]*stream.Buffer, 0, len(cfg.Inputs))
		for _, in := range cfg.Inputs {
			inCfg, _ := lookup(in) // ValidateStreamSource checked it exists and is a stream
			b, err := streamInput(hub, in, inCfg, dir)
			if err != nil {
				return nil, fmt.Errorf("merge source %q: %w", name, err)
			}
			inputs = append(inputs, b)
		}
		buf, err = hub.Merge(name, inputs, cfg.Buffer)
	}
	if err != nil {
		return nil, err
	}
	return &StreamSource{name: name, buf: buf, flush: flush}, nil
}

// streamInput resolves a stream source's command or file against dir (the
// absolute site directory) and returns its hub buffer. A command runs from dir,
// like exec, so ./script.sh means the script next to tinkerdown.yaml. A file
// may be absolute: like the json and csv sources, a stream reads any file the
// server's user can.
func streamInput(hub *stream.Hub, name string, cfg config.SourceConfig, dir string) (*stream.Buffer, error) {
	spec := cfg.StreamSpec(name)
	if spec.Cmd != "" {
		if err := requireExec(name, "cmd", "stream sources that run a command"); err != nil {
			return nil, err
		}
		spec.Dir = dir
		spec.Env = expandEnv(spec.Env)
	} else {
		spec.File = resolveSitePath(dir, spec.File)
	}
	return hub.Input(spec)
}
