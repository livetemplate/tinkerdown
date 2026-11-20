package source

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/livetemplate/tinkerdown/internal/config"
	"github.com/livetemplate/tinkerdown/internal/stream"
)

// StreamSource is a Source backed by a stream.Buffer shared with every other
// page showing the same input. Fetch returns the buffer's current rows; the
// server subscribes to learn when there are more.
type StreamSource struct {
	name string
	buf  *stream.Buffer
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

// NewStreamSource builds a stream or merge source on hub. lookup resolves the
// stream sources a merge lists in inputs, with the same precedence the page
// uses for its own sources.
func NewStreamSource(hub *stream.Hub, name string, cfg config.SourceConfig, siteDir string, lookup func(string) (config.SourceConfig, bool)) (*StreamSource, error) {
	if hub == nil {
		return nil, fmt.Errorf("%s source %q: needs a running server", cfg.Type, name)
	}
	if err := config.ValidateStreamSource(name, cfg, lookup); err != nil {
		return nil, err
	}
	if cfg.Type == "stream" {
		buf, err := streamInput(hub, name, cfg, siteDir)
		if err != nil {
			return nil, err
		}
		return &StreamSource{name: name, buf: buf}, nil
	}

	inputs := make([]*stream.Buffer, 0, len(cfg.Inputs))
	for _, in := range cfg.Inputs {
		inCfg, _ := lookup(in) // ValidateStreamSource checked it exists and is a stream
		buf, err := streamInput(hub, in, inCfg, siteDir)
		if err != nil {
			return nil, fmt.Errorf("merge source %q: %w", name, err)
		}
		inputs = append(inputs, buf)
	}
	buf, err := hub.Merge(name, inputs, cfg.Buffer)
	if err != nil {
		return nil, err
	}
	return &StreamSource{name: name, buf: buf}, nil
}

func streamInput(hub *stream.Hub, name string, cfg config.SourceConfig, siteDir string) (*stream.Buffer, error) {
	if cfg.Cmd != "" && !config.IsExecAllowed() {
		return nil, &ValidationError{
			Source: name,
			Field:  "cmd",
			Reason: "stream sources that run a command are disabled by default for security. Use --allow-exec flag to enable.",
		}
	}
	dir, err := filepath.Abs(siteDir)
	if err != nil {
		return nil, err
	}
	spec := stream.InputSpec{
		Name:    name,
		Cmd:     cfg.Cmd,
		Format:  cfg.Format,
		Buffer:  cfg.Buffer,
		Backlog: cfg.Backlog,
		Restart: cfg.Restart,
	}
	if cfg.Cmd != "" {
		// Like exec: run from the site directory, so ./script.sh means the
		// script next to tinkerdown.yaml, with env values expanded.
		spec.Dir = dir
		spec.Env = make(map[string]string, len(cfg.Env))
		for k, v := range cfg.Env {
			spec.Env[k] = os.ExpandEnv(v)
		}
	} else if filepath.IsAbs(cfg.File) {
		spec.File = cfg.File
	} else {
		spec.File = filepath.Join(dir, cfg.File)
	}
	return hub.Input(spec)
}
