package config

import (
	"fmt"
	"time"

	"github.com/livetemplate/tinkerdown/internal/stream"
)

// ValidateStreamSources checks the stream and merge sources of the config:
// a stream passes stream.InputSpec.Normalize and has a valid flush; a merge lists at least one input, and every input is a stream source.
func (c *Config) ValidateStreamSources() error {
	for name, src := range c.Sources {
		if err := ValidateStreamSource(name, src, func(n string) (SourceConfig, bool) {
			s, ok := c.Sources[n]
			return s, ok
		}); err != nil {
			return err
		}
	}
	return nil
}

// ValidateStreamSource checks one stream or merge source; other types pass.
// lookup resolves the names a merge lists in inputs.
func ValidateStreamSource(name string, src SourceConfig, lookup func(string) (SourceConfig, bool)) error {
	switch src.Type {
	case "stream":
		if _, err := src.StreamSpec(name).Normalize(); err != nil {
			return err
		}
	case "merge":
		if len(src.Inputs) == 0 {
			return fmt.Errorf("merge source %q: inputs lists no sources", name)
		}
		for _, in := range src.Inputs {
			inCfg, ok := lookup(in)
			if !ok {
				return fmt.Errorf("merge source %q: input %q is not a defined source", name, in)
			}
			if inCfg.Type != "stream" {
				return fmt.Errorf("merge source %q: input %q is a %s source, not a stream", name, in, inCfg.Type)
			}
		}
	default:
		return nil
	}
	if _, err := src.FlushInterval(); err != nil {
		return fmt.Errorf("%s source %q: %w", src.Type, name, err)
	}
	return nil
}

// StreamSpec maps a stream source's config onto the hub's input spec. Paths and
// environment are left as configured; the source package resolves them.
func (s SourceConfig) StreamSpec(name string) stream.InputSpec {
	return stream.InputSpec{
		Name:    name,
		Cmd:     s.Cmd,
		File:    s.File,
		Env:     s.Env,
		Format:  s.Format,
		Buffer:  s.Buffer,
		Backlog: s.Backlog,
		Restart: s.Restart,
	}
}

// DefaultStreamFlush is the minimum time between pushes of a stream or merge
// source to one page.
const DefaultStreamFlush = 100 * time.Millisecond

// FlushInterval returns the parsed flush value, or DefaultStreamFlush when unset.
func (s SourceConfig) FlushInterval() (time.Duration, error) {
	if s.Flush == "" {
		return DefaultStreamFlush, nil
	}
	d, err := time.ParseDuration(s.Flush)
	if err != nil || d <= 0 {
		return 0, fmt.Errorf("flush %q is not a positive duration", s.Flush)
	}
	return d, nil
}
