package config

import (
	"fmt"
	"slices"
	"time"

	"github.com/livetemplate/tinkerdown/internal/stream"
)

// ValidateStreamSources checks the stream and merge sources of the config:
// a stream passes stream.InputSpec.Normalize and has a valid flush; a merge
// lists at least one input, and every input is a stream source. A ["*"] merge
// is checked against the site and flag sources it would take.
func (c *Config) ValidateStreamSources() error {
	lookup := func(n string) (SourceConfig, bool) {
		if s, ok := c.FlagSources[n]; ok {
			return s, true
		}
		s, ok := c.Sources[n]
		return s, ok
	}
	var names []string
	for n := range c.Sources {
		names = append(names, n)
	}
	for n := range c.FlagSources {
		names = append(names, n)
	}
	for name, src := range c.Sources {
		if err := ValidateStreamSource(name, ExpandInputs(src, names, lookup), lookup); err != nil {
			return err
		}
	}
	return nil
}

// ValidationError is a mistake in a source's configuration. Its message names
// the source and the setting, never a command, path or environment value, so
// it is safe to show on the page.
type ValidationError struct{ msg string }

func (e *ValidationError) Error() string { return e.msg }

func invalid(format string, args ...any) error {
	return &ValidationError{fmt.Sprintf(format, args...)}
}

// ValidateStreamSource checks one stream or merge source; other types pass.
// lookup resolves the names a merge lists in inputs. A merge with inputs ["*"]
// must be expanded with ExpandInputs first; it may then hold no inputs.
func ValidateStreamSource(name string, src SourceConfig, lookup func(string) (SourceConfig, bool)) error {
	switch src.Type {
	case "stream":
		if _, err := src.StreamSpec(name).Normalize(); err != nil {
			return invalid("%v", err)
		}
	case "merge":
		if err := validateAccepts(name, src); err != nil {
			return err
		}
		if src.WildcardInputs() {
			return invalid("merge source %q: inputs [\"*\"] was not expanded", name)
		}
		if len(src.Inputs) == 0 && !src.expanded {
			return invalid("merge source %q: inputs lists no sources", name)
		}
		for _, in := range src.Inputs {
			inCfg, ok := lookup(in)
			if !ok {
				return invalid("merge source %q: input %q is not a defined source", name, in)
			}
			if inCfg.Type != "stream" {
				return invalid("merge source %q: input %q is a %s source, not a stream", name, in, inCfg.Type)
			}
		}
	default:
		return nil
	}
	if _, err := src.FlushInterval(); err != nil {
		return invalid("%s source %q: %v", src.Type, name, err)
	}
	return nil
}

// WildcardInputs reports whether a merge takes every stream source it can see.
func (s SourceConfig) WildcardInputs() bool {
	return s.Type == "merge" && len(s.Inputs) == 1 && s.Inputs[0] == "*"
}

// Kind is "cmd" or "file" for a stream source, and "" otherwise.
func (s SourceConfig) Kind() string {
	switch {
	case s.Type != "stream":
		return ""
	case s.Cmd != "":
		return "cmd"
	default:
		return "file"
	}
}

// AcceptsKind reports whether a merge's accepts block allows a stream source
// of the given kind. A merge without accepts, or without kinds, takes both.
func (s SourceConfig) AcceptsKind(kind string) bool {
	if s.Accepts == nil || len(s.Accepts.Kinds) == 0 {
		return true
	}
	return slices.Contains(s.Accepts.Kinds, kind)
}

// ExpandInputs replaces a merge's ["*"] with every name lookup resolves to a
// stream source of a kind the merge accepts, sorted. names lists the sources
// to consider. Any other source comes back unchanged.
func ExpandInputs(src SourceConfig, names []string, lookup func(string) (SourceConfig, bool)) SourceConfig {
	if !src.WildcardInputs() {
		return src
	}
	var inputs []string
	for _, n := range names {
		if in, ok := lookup(n); ok && in.Type == "stream" && src.AcceptsKind(in.Kind()) {
			inputs = append(inputs, n)
		}
	}
	slices.Sort(inputs)
	src.Inputs = slices.Compact(inputs)
	src.expanded = true
	return src
}

func validateAccepts(name string, src SourceConfig) error {
	a := src.Accepts
	if a == nil {
		return nil
	}
	if !src.WildcardInputs() && !src.expanded {
		return invalid("merge source %q: accepts needs inputs [\"*\"]", name)
	}
	for _, k := range a.Kinds {
		if k != "cmd" && k != "file" {
			return invalid("merge source %q: accepts kind %q is not cmd or file", name, k)
		}
	}
	if a.Min < 0 {
		return invalid("merge source %q: accepts min is negative", name)
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

// CheckFlagSources checks c.FlagSources before serving. A flag may not replace
// an approved source. Each flag must be used: by name, as a source a page or
// the site defines, or by a ["*"] merge that accepts its kind. A flag nothing
// uses is most likely a typo. pages holds each page's own sources.
func (c *Config) CheckFlagSources(pages []map[string]SourceConfig) error {
	sets := append([]map[string]SourceConfig{c.Sources}, pages...)
	names := make([]string, 0, len(c.FlagSources))
	for n := range c.FlagSources {
		names = append(names, n)
	}
	slices.Sort(names)
	for _, name := range names {
		if c.ApprovedSource(name) {
			return fmt.Errorf("--source %s: %q is an approved source in tinkerdown.yaml and can't be replaced", name, name)
		}
		if !flagSourceUsed(name, c.FlagSources[name].Kind(), sets) {
			return fmt.Errorf("--source %s: no page takes it. A page takes a --source through a merge with inputs [\"*\"] that accepts %s sources, or a source of the same name", name, c.FlagSources[name].Kind())
		}
	}
	return nil
}

func flagSourceUsed(name, kind string, sets []map[string]SourceConfig) bool {
	for _, set := range sets {
		if _, ok := set[name]; ok {
			return true
		}
		for _, src := range set {
			if src.WildcardInputs() && src.AcceptsKind(kind) {
				return true
			}
		}
	}
	return false
}
