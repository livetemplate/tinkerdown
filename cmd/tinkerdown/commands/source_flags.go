package commands

import (
	"bufio"
	"fmt"
	"io"
	"maps"
	"os"
	"regexp"
	"slices"
	"strings"

	"github.com/livetemplate/tinkerdown/internal/config"
	"github.com/livetemplate/tinkerdown/internal/runtime"
	"github.com/livetemplate/tinkerdown/internal/security"
	"github.com/livetemplate/tinkerdown/internal/server"
	"golang.org/x/term"
)

// sourceName is what a --source name may hold. Names end up in filter
// expressions (source = NAME), CSS classes and tab labels, so they stay plain.
var sourceName = regexp.MustCompile(`^[A-Za-z0-9_-]{1,32}$`)

// parseSourceFlag parses a --source value: NAME=COMMAND for a stream source
// that runs COMMAND, or NAME=file:PATH for one that follows PATH. It splits on
// the first "=" only, so a command may contain "=".
func parseSourceFlag(v string) (string, config.SourceConfig, error) {
	name, value, ok := strings.Cut(v, "=")
	if !ok || value == "" {
		return "", config.SourceConfig{}, fmt.Errorf("--source %q: want NAME=COMMAND or NAME=file:PATH", v)
	}
	if !sourceName.MatchString(name) {
		return "", config.SourceConfig{}, fmt.Errorf("--source %q: name %q must be 1-32 letters, digits, _ or -", v, name)
	}
	if path, isFile := strings.CutPrefix(value, "file:"); isFile {
		if path == "" {
			return "", config.SourceConfig{}, fmt.Errorf("--source %q: file: needs a path", v)
		}
		return name, config.SourceConfig{Type: "stream", File: path}, nil
	}
	return name, config.SourceConfig{Type: "stream", Cmd: value}, nil
}

// sourceFlagValue turns a source back into the value parseSourceFlag reads.
func sourceFlagValue(name string, src config.SourceConfig) string {
	if src.File != "" {
		return name + "=file:" + src.File
	}
	return name + "=" + src.Cmd
}

// shellQuote quotes s for a POSIX shell, so a printed command line runs the
// same values when pasted back.
func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

// addFlagSource parses a --source value (or a prompt answer) and adds it to
// flags: one place for the rules both paths share.
func addFlagSource(flags map[string]config.SourceConfig, value string, allowExec bool) (string, error) {
	name, src, err := parseSourceFlag(value)
	switch {
	case err != nil:
		return "", err
	case flags[name].Type != "":
		return "", fmt.Errorf("--source %s is given twice", name)
	case src.Cmd != "" && !allowExec:
		return "", fmt.Errorf("--source %s runs a command; add --allow-exec to allow it", name)
	}
	if err := config.ValidateStreamSource(name, src, nil); err != nil {
		return "", fmt.Errorf("--source %s: %w", name, err)
	}
	flags[name] = src
	return name, nil
}

// addFlagAction parses an --action NAME=COMMAND value and adds it to flags.
// It splits on the first "=" only, so a command may contain "=". A builtin
// name (pause, delete, ...) is refused, since a builtin would run instead.
func addFlagAction(flags map[string]*config.Action, v string, allowExec bool) error {
	name, cmd, ok := strings.Cut(v, "=")
	switch {
	case !ok || strings.TrimSpace(cmd) == "":
		return fmt.Errorf("--action %q: want NAME=COMMAND", v)
	case !sourceName.MatchString(name):
		return fmt.Errorf("--action %q: name %q must be 1-32 letters, digits, _ or -", v, name)
	case runtime.IsBuiltinAction(name):
		return fmt.Errorf("--action %s: %q is a builtin action; pick another name", name, name)
	case flags[name] != nil:
		return fmt.Errorf("--action %s is given twice", name)
	case !allowExec:
		return fmt.Errorf("--action %s runs a command; add --allow-exec to allow it", name)
	}
	flags[name] = &config.Action{Kind: "exec", Cmd: cmd, FromFlag: true}
	return nil
}

// checkFlagActions refuses a flag that replaces an approved action.
func checkFlagActions(cfg *config.Config) error {
	for _, name := range slices.Sorted(maps.Keys(cfg.FlagActions)) {
		if cfg.ApprovedAction(name) {
			return fmt.Errorf("--action %s: %q is an approved action in tinkerdown.yaml and can't be replaced", name, name)
		}
	}
	return nil
}

// checkFlagSources refuses a flag that replaces an approved source, and one no
// page uses. A page uses a flag when the flag replaces a source the page or
// the site defines, or when one of the page's merges resolves to include it.
// Both are decided by server.ResolveSource, as the page's blocks will see them.
func checkFlagSources(cfg *config.Config, srv *server.Server) error {
	used := map[string]bool{}
	for _, route := range srv.Routes() {
		if route.Page == nil {
			continue
		}
		for name := range route.Page.Config.Sources {
			used[name] = true
		}
		for _, name := range server.VisibleSourceNames(cfg, route.Page) {
			if src, ok := server.ResolveSource(cfg, route.Page, name); ok && src.Type == "merge" {
				for _, in := range src.Inputs {
					used[in] = true
				}
			}
		}
	}
	for name := range cfg.Sources {
		used[name] = true
	}
	for _, name := range slices.Sorted(maps.Keys(cfg.FlagSources)) {
		if cfg.ApprovedSource(name) {
			return fmt.Errorf("--source %s: %q is an approved source in tinkerdown.yaml and can't be replaced", name, name)
		}
		if !used[name] {
			return fmt.Errorf("--source %s: no page takes it. A page takes a --source through a merge with inputs [\"*\"] that accepts %s sources, or a source of the same name", name, cfg.FlagSources[name].Kind())
		}
	}
	return nil
}

// sourceNeed is a ["*"] merge on a page that has fewer sources than it needs.
type sourceNeed struct {
	page, hint string
	min        int
}

// sourceNeeds lists every ["*"] merge, on any page, that resolves to fewer
// sources than its accepts.min (at least 1).
func sourceNeeds(cfg *config.Config, srv *server.Server) []sourceNeed {
	var needs []sourceNeed
	for _, route := range srv.Routes() {
		if route.Page == nil {
			continue
		}
		names := slices.Compact(slices.Sorted(slices.Values(server.VisibleSourceNames(cfg, route.Page))))
		for _, name := range names {
			src, ok := server.ResolveSource(cfg, route.Page, name)
			if !ok || !src.Wildcard {
				continue
			}
			need, hint := 1, ""
			if src.Accepts != nil {
				need, hint = max(src.Accepts.Min, 1), src.Accepts.Hint
			}
			if len(src.Inputs) < need {
				needs = append(needs, sourceNeed{page: route.FilePath, hint: hint, min: need})
			}
		}
	}
	return needs
}

// stdinIsTerminal reports whether a person can answer a prompt. Agents, CI,
// pipes and /dev/null get no prompt, so serve never waits on them.
// It is a variable so tests can run serve as if no terminal were attached,
// whatever stdin the test binary got.
var stdinIsTerminal = func() bool {
	return term.IsTerminal(int(os.Stdin.Fd()))
}

// describeNeeds tells the operator which pages need sources and how to pass
// them. Page text (the hint) is untrusted, so it is cleaned and quoted.
func describeNeeds(w io.Writer, needs []sourceNeed) {
	for _, n := range needs {
		fmt.Fprintf(w, "%s needs at least %d source", n.page, n.min)
		if n.min != 1 {
			fmt.Fprint(w, "s")
		}
		if n.hint != "" {
			fmt.Fprintf(w, ". The page says: %q", security.CleanPageText(n.hint))
		}
		fmt.Fprintln(w, ".")
	}
}

// promptSources asks for sources until an empty line, and adds them to
// flags. It returns the --source values the operator gave.
func promptSources(in io.Reader, out io.Writer, flags map[string]config.SourceConfig, allowExec bool) []string {
	fmt.Fprintln(out, "Add one as name=command or name=file:path. Leave empty to finish.")
	scanner := bufio.NewScanner(in)
	var given []string
	for {
		fmt.Fprint(out, "source> ")
		if !scanner.Scan() {
			fmt.Fprintln(out)
			return given
		}
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			return given
		}
		name, err := addFlagSource(flags, line, allowExec)
		if err != nil {
			fmt.Fprintln(out, err)
			continue
		}
		given = append(given, sourceFlagValue(name, flags[name]))
	}
}

// reuseLine is the serve command line that repeats this run without the
// prompt: the original arguments plus a --source for each answer.
func reuseLine(args, given []string) string {
	parts := []string{"tinkerdown", "serve"}
	for _, a := range args {
		parts = append(parts, shellQuoteIfNeeded(a))
	}
	for _, g := range given {
		parts = append(parts, "--source", shellQuote(g))
	}
	return strings.Join(parts, " ")
}

func shellQuoteIfNeeded(s string) string {
	if s != "" && strings.Trim(s, "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789-_./:=") == "" {
		return s
	}
	return shellQuote(s)
}
