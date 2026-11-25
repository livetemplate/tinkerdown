package config

import (
	"cmp"
	"fmt"
	"maps"
	"regexp"
	"slices"
	"strconv"
	"strings"
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
			return invalid("stream source %q: key, collect, collect_reset, drop, sort, params, on and line_max apply to merge sources", name)
		}
		// A click names a request or query by its entry, NAME/1, which must
		// pass the value rules.
		if k := src.Kind(); k == "http" || k == "sql" {
			if err := CheckTDValue("entry", stream.EntryID(name, 0)); err != nil {
				return invalid("stream source %q: an %s source's name can't start with - or hold a control character, since its rows' entries start with it", name, k)
			}
		}
	case "merge":
		if err := validateAccepts(name, src); err != nil {
			return err
		}
		if err := validateKey(name, src); err != nil {
			return err
		}
		if err := validateParams(name, src, lookup); err != nil {
			return err
		}
		if _, err := ParseLineMax(src.LineMax); err != nil {
			return invalid("merge source %q: %v", name, err)
		}
		if src.On != "" && (len(src.Params) == 0 || !onName.MatchString(src.On)) {
			return invalid("merge source %q: on %q needs params, and must be 1-32 lowercase letters, digits or _, starting with a letter", name, src.On)
		}
		if BuiltinActions[src.On] {
			return invalid("merge source %q: on %q is a built-in action, so a button named %s runs that and never starts this merge; pick another name", name, src.On, src.On)
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
			if src.On != "" && inCfg.Cmd == "" {
				return invalid("merge source %q: input %q must run a command, since a click starts it", name, in)
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

// ParamsMerge reports whether a merge takes values from the page URL's query.
func (s SourceConfig) ParamsMerge() bool {
	return s.Type == "merge" && len(s.Params) > 0
}

// StreamKinds lists a stream source's kinds: what it reads.
var StreamKinds = []string{"cmd", "file", "http", "sql"}

// Kind is "cmd", "file", "http" or "sql" for a stream source, and "" otherwise.
func (s SourceConfig) Kind() string {
	switch {
	case s.Type != "stream":
		return ""
	case s.Cmd != "":
		return "cmd"
	case s.HTTP != "":
		return "http"
	case s.SQL != "":
		return "sql"
	default:
		return "file"
	}
}

// StreamPath is a file, http or sql stream's path, and "" for any other source.
func (s SourceConfig) StreamPath() string {
	return cmp.Or(s.File, s.HTTP, s.SQL)
}

// WithPath returns a stream source of kind, file, http or sql, that reads
// path; ok is false for any other kind.
func WithPath(kind, path string) (src SourceConfig, ok bool) {
	src = SourceConfig{Type: "stream"}
	switch kind {
	case "file":
		src.File = path
	case "http":
		src.HTTP = path
	case "sql":
		src.SQL = path
	default:
		return SourceConfig{}, false
	}
	return src, true
}

// ParseLineMax reads a line_max value: bytes, or a number with a KiB or MiB
// suffix. "" is 0, meaning the default.
func ParseLineMax(v string) (int, error) {
	if v == "" {
		return 0, nil
	}
	m := lineMaxValue.FindStringSubmatch(v)
	if m == nil {
		return 0, fmt.Errorf("line_max %q is not a size such as 65536, 64KiB or 1MiB", v)
	}
	n, _ := strconv.Atoi(m[1])
	switch m[2] {
	case "KiB":
		n *= 1 << 10
	case "MiB":
		n *= 1 << 20
	}
	if n < 1 || n > stream.MaxLineMax {
		return 0, fmt.Errorf("line_max %q is not between 1 byte and 16MiB", v)
	}
	return n, nil
}

var lineMaxValue = regexp.MustCompile(`^([0-9]{1,9})(KiB|MiB)?$`)

// onName is what On may hold: a button name a page can't confuse with
// another action.
var onName = regexp.MustCompile(`^[a-z][a-z0-9_]{0,31}$`)

// BuiltinActions are the lowercase action names the runtime handles for any
// source, independent of the manifest: source affordances (refresh a view,
// add or delete a row), not custom, governed actions. The policy lint doesn't
// treat them as unapproved, since whether the source is approved and writable
// governs them. A button with one of these names runs the built-in, so it
// never starts a merge's on:. Kept in sync with runtime's HandleAction switch.
var BuiltinActions = map[string]bool{
	"refresh": true, "run": true, "filter": true, "edit": true, "canceledit": true,
	"add": true, "toggle": true, "delete": true, "update": true,
	"pause": true, "resume": true, "clear": true,
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
	case src.Sort != "" && (len(src.Key) == 0 || !validSort(src.Sort)):
		return invalid("merge source %q: sort %q must be a field (NAME or fields.NAME, with - for descending), with key set", name, src.Sort)
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

// validSort reports whether s is a field path, with an optional leading -.
func validSort(s string) bool {
	o, ok := stream.ParseOrder(s)
	return ok && fieldPath.MatchString(o.Path)
}

// mergeOnly reports whether src sets an option that only a merge uses.
func mergeOnly(src SourceConfig) bool {
	return len(src.Key) > 0 || src.Collect != "" || src.CollectReset != "" || src.Drop != "" || src.Sort != "" || len(src.Params) > 0 || src.On != "" || src.LineMax != ""
}

// validateParams checks a merge's params: each name is a TD_ name, and each
// in names a merge without params of its own, and a field path in it.
func validateParams(name string, src SourceConfig, lookup func(string) (SourceConfig, bool)) error {
	if len(src.Params) > 0 && (src.WildcardInputs() || src.Wildcard) {
		return invalid("merge source %q: params needs inputs listed by name, not [\"*\"]", name)
	}
	for _, p := range slices.Sorted(maps.Keys(src.Params)) {
		if !TDName.MatchString(p) {
			return invalid("merge source %q: param %q must be 1-32 letters, digits or _", name, p)
		}
		gate, path := src.Params[p].Gate()
		if !fieldPath.MatchString(path) {
			return invalid("merge source %q: param %s needs in: MERGE.PATH, where PATH is NAME or fields.NAME", name, p)
		}
		var g SourceConfig
		ok := false
		if lookup != nil {
			g, ok = lookup(gate)
		}
		if !ok || g.Type != "merge" || len(g.Params) > 0 || gate == name {
			return invalid("merge source %q: param %s: in names %q, which must be another merge on the page without params", name, p, gate)
		}
	}
	return nil
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
		if !slices.Contains(StreamKinds, k) {
			return invalid("merge source %q: accepts kind %q is not one of %s", name, k, strings.Join(StreamKinds, ", "))
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
		HTTP:    s.HTTP,
		SQL:     s.SQL,
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

// KeySpec maps a merge's key, collect, collect_reset, drop and sort onto the
// hub's.
func (s SourceConfig) KeySpec() stream.KeySpec {
	k := stream.KeySpec{Paths: s.Key, Collect: s.Collect}
	k.Reset, _ = stream.ParseMatch(s.CollectReset)
	k.Drop, _ = stream.ParseMatch(s.Drop)
	k.Sort, _ = stream.ParseOrder(s.Sort)
	return k
}
