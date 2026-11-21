package commands

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"regexp"
	"slices"
	"strings"

	"github.com/livetemplate/tinkerdown"
	"github.com/livetemplate/tinkerdown/internal/config"
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
		names := make([]string, 0, len(route.Page.Config.Sources)+len(cfg.Sources))
		for n := range route.Page.Config.Sources {
			names = append(names, n)
		}
		for n := range cfg.Sources {
			names = append(names, n)
		}
		slices.Sort(names)
		for _, name := range slices.Compact(names) {
			if !wasWildcard(cfg, route.Page, name) {
				continue
			}
			src, _ := server.ResolveSource(cfg, route.Page, name)
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

// wasWildcard reports whether a page's merge source was declared with ["*"].
func wasWildcard(cfg *config.Config, page *tinkerdown.Page, name string) bool {
	if src, ok := page.Config.Sources[name]; ok {
		return server.PageSourceConfig(src).WildcardInputs()
	}
	return cfg.Sources[name].WildcardInputs()
}

// stdinIsTerminal reports whether a person can answer a prompt. Agents, CI,
// pipes and /dev/null get no prompt, so serve never waits on them.
func stdinIsTerminal() bool {
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
		name, src, err := parseSourceFlag(line)
		switch {
		case err != nil:
			fmt.Fprintln(out, err)
			continue
		case src.Cmd != "" && !allowExec:
			fmt.Fprintf(out, "%s runs a command; restart with --allow-exec to allow it\n", name)
			continue
		}
		if _, dup := flags[name]; dup {
			fmt.Fprintf(out, "%s is already set\n", name)
			continue
		}
		flags[name] = src
		given = append(given, sourceFlagValue(name, src))
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
