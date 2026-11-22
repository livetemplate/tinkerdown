package config

import (
	"fmt"
	"maps"
	"regexp"
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
	names := slices.Concat(slices.Collect(maps.Keys(c.Sources)), slices.Collect(maps.Keys(c.FlagSources)))
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
		if mergeOnly(src) {
			return invalid("stream source %q: key, collect, collect_reset and drop apply to merge sources", name)
		}
	case "merge":
		if err := validateAccepts(name, src); err != nil {
			return err
		}
		if err := validateKey(name, src); err != nil {
			return err
		}
		if len(src.Inputs) == 0 && !src.Wildcard {
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
	src.Wildcard = true
	return src
}

// fieldPath is a row field (NAME) or a parsed JSON or logfmt field (fields.NAME).
var fieldPath = regexp.MustCompile(`^(fields\.)?[A-Za-z_][A-Za-z0-9_]*$`)

func validateKey(name string, src SourceConfig) error {
	for _, p := range src.Key {
		if !fieldPath.MatchString(p) {
			return invalid("merge source %q: key %q is not a field (NAME or fields.NAME)", name, p)
		}
	}
	switch {
	case src.CollectReset != "" && (src.Collect == "" || !validMatch(src.CollectReset)):
		return invalid("merge source %q: collect_reset %q must be PATH=VALUE, with collect set", name, src.CollectReset)
	case src.Drop != "" && (len(src.Key) == 0 || !validMatch(src.Drop)):
		return invalid("merge source %q: drop %q must be PATH=VALUE, with key set", name, src.Drop)
	case src.Collect != "" && len(src.Key) == 0:
		return invalid("merge source %q: collect needs key", name)
	case src.Collect != "" && !fieldPath.MatchString(src.Collect):
		return invalid("merge source %q: collect %q is not a field (NAME or fields.NAME)", name, src.Collect)
	}
	return nil
}

// validMatch reports whether s is PATH=VALUE with PATH a field path.
func validMatch(s string) bool {
	m, ok := stream.ParseMatch(s)
	return ok && fieldPath.MatchString(m.Path)
}

// mergeOnly reports whether src sets an option that only a merge uses.
func mergeOnly(src SourceConfig) bool {
	return len(src.Key) > 0 || src.Collect != "" || src.CollectReset != "" || src.Drop != ""
}

func validateAccepts(name string, src SourceConfig) error {
	a := src.Accepts
	if a == nil {
		return nil
	}
	if !src.WildcardInputs() && !src.Wildcard {
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

// KeySpec maps a merge's key, collect, collect_reset and drop onto the hub's.
func (s SourceConfig) KeySpec() stream.KeySpec {
	k := stream.KeySpec{Paths: s.Key, Collect: s.Collect}
	k.Reset, _ = stream.ParseMatch(s.CollectReset)
	k.Drop, _ = stream.ParseMatch(s.Drop)
	return k
}
