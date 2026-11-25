package commands

import (
	"bufio"
	"cmp"
	"fmt"
	"io"
	"maps"
	"os"
	"path"
	"path/filepath"
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

// parseSourceFlag parses a --source value: COMMAND for a stream source that
// runs COMMAND, file:PATH for one that follows PATH, or http:PATH for one that
// lists the requests in a .http file. Each may start with NAME=, which names
// the source; without it the name is "" and addSource picks a label. NAME= is
// the part before the first "=" when that part is a plain name, so a command
// such as `grep a=b` is unnamed, and one that starts with an environment
// assignment is written `env FOO=1 cmd`. PATH is kept as given; see
// absSourcePaths for a page from a URL.
func parseSourceFlag(v string) (string, config.SourceConfig, error) {
	if strings.HasPrefix(v, "=") {
		return "", config.SourceConfig{}, fmt.Errorf("--source %q: a name goes before the \"=\", as in api=COMMAND; or leave out NAME= and give the command alone", v)
	}
	name, value := "", v
	if n, rest, ok := strings.Cut(v, "="); ok && sourceName.MatchString(n) {
		name, value = n, rest
	}
	if strings.TrimSpace(value) == "" {
		return "", config.SourceConfig{}, fmt.Errorf("--source %q: want COMMAND, file:PATH or http:PATH, optionally after NAME=", v)
	}
	if path, ok := strings.CutPrefix(value, "file:"); ok {
		return name, config.SourceConfig{Type: "stream", File: path}, pathErr(v, "file", path)
	}
	if path, ok := strings.CutPrefix(value, "http:"); ok {
		return name, config.SourceConfig{Type: "stream", HTTP: path}, pathErr(v, "http", path)
	}
	return name, config.SourceConfig{Type: "stream", Cmd: value}, nil
}

// pathErr refuses a file: or http: flag with no path.
func pathErr(flag, kind, path string) error {
	if path == "" {
		return fmt.Errorf("--source %q: %s: needs a path", flag, kind)
	}
	return nil
}

// absSourcePaths makes each relative file and http path in srcs absolute
// from base. A page from a URL lives in a temporary directory, so serve
// resolves its flags from the directory it runs in and its --config file's
// sources from that file's directory; a local page keeps paths relative to
// its own directory.
func absSourcePaths(srcs map[string]config.SourceConfig, base string) {
	for name, src := range srcs {
		for _, p := range []*string{&src.File, &src.HTTP} {
			if *p != "" && !filepath.IsAbs(*p) {
				*p = filepath.Join(base, *p)
			}
		}
		srcs[name] = src
	}
}

// sourceLabel is the label of a source with no name: from its command, or
// from its file's name without the extension.
func sourceLabel(src config.SourceConfig) string {
	switch label := wordLabel(cmp.Or(src.File, src.HTTP)); {
	case src.File == "" && src.HTTP == "":
		return defaultLabel(src.Cmd)
	case sourceName.MatchString(label):
		return label
	case src.HTTP != "":
		return "http"
	}
	return "file"
}

// defaultLabel picks the tab label for an unnamed command: its last plain
// word, which usually names what the command follows. Flags are skipped, and
// a path counts as its base name without the extension, so
// `docker logs -f api` is api, `tail -f /var/log/app.log` is app and
// `watchexec -e go -- go test -json ./...` is test. When no word fits
// sourceName, the label is the first word with other characters dropped, or
// "cmd".
func defaultLabel(cmd string) string {
	words := strings.Fields(cmd)
	for i := len(words) - 1; i >= 0; i-- {
		if strings.HasPrefix(words[i], "-") {
			continue
		}
		if w := wordLabel(words[i]); sourceName.MatchString(w) {
			return w
		}
	}
	if len(words) > 0 {
		kept := notNameChars.ReplaceAllString(wordLabel(words[0]), "")
		if kept = kept[:min(len(kept), 32)]; kept != "" {
			return kept
		}
	}
	return "cmd"
}

// notNameChars matches what sourceName doesn't allow.
var notNameChars = regexp.MustCompile(`[^A-Za-z0-9_-]`)

// wordLabel is a command word as a label: unquoted, and a path reduced to its
// base name without the extension.
func wordLabel(w string) string {
	w = path.Base(strings.Trim(w, `'"`))
	return strings.TrimSuffix(w, path.Ext(w))
}

// uniqueLabel returns label, or label-2, label-3, ... when taken has it.
// The suffix keeps the result within sourceName's 32 characters.
func uniqueLabel(label string, taken func(string) bool) string {
	if !taken(label) {
		return label
	}
	for n := 2; ; n++ {
		suffix := fmt.Sprintf("-%d", n)
		l := label[:min(len(label), 32-len(suffix))] + suffix
		if !taken(l) {
			return l
		}
	}
}

// pageSourceNames lists the names the pages and the site use: every source
// they define and every input a merge lists. An unnamed source's label skips
// these, so it never replaces a page's source or fills a role (send, logs,
// ...) the operator didn't name. Only a NAME= does that. It reads what the
// pages declare, not the flag sources.
func pageSourceNames(cfg *config.Config, srv *server.Server) map[string]bool {
	names := map[string]bool{}
	add := func(name, typ string, inputs []string) {
		names[name] = true
		if typ == "merge" {
			for _, in := range inputs {
				if in != "*" {
					names[in] = true
				}
			}
		}
	}
	for name, src := range cfg.Sources {
		add(name, src.Type, src.Inputs)
	}
	for _, route := range srv.Routes() {
		if route.Page != nil {
			for name, src := range route.Page.Config.Sources {
				add(name, src.Type, src.Inputs)
			}
		}
	}
	return names
}

// sourceFlagValue turns a source back into the value parseSourceFlag reads.
func sourceFlagValue(name string, src config.SourceConfig) string {
	switch {
	case src.File != "":
		return name + "=file:" + src.File
	case src.HTTP != "":
		return name + "=http:" + src.HTTP
	}
	return name + "=" + src.Cmd
}

// shellQuote quotes s for a POSIX shell, so a printed command line runs the
// same values when pasted back.
func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

// addFlagSource parses a --source value (or a prompt answer) and adds it to
// flags. pageNames are the names an unnamed source's label skips.
func addFlagSource(flags map[string]config.SourceConfig, pageNames map[string]bool, value string, allowExec bool) (string, error) {
	name, src, err := parseSourceFlag(value)
	if err != nil {
		return "", err
	}
	return addSource(flags, pageNames, name, src, allowExec)
}

// flagSources parses serve's --source values and the command after "--".
// Named sources go in first, so a name the operator gave is never taken by
// an unnamed source's label; neither is one of pageNames.
func flagSources(values []string, dashCmd string, pageNames map[string]bool, allowExec bool) (map[string]config.SourceConfig, error) {
	flags := map[string]config.SourceConfig{}
	var unnamed []config.SourceConfig
	for _, v := range values {
		name, src, err := parseSourceFlag(v)
		switch {
		case err != nil:
			return nil, err
		case name == "":
			unnamed = append(unnamed, src)
		default:
			if _, err := addSource(flags, pageNames, name, src, allowExec); err != nil {
				return nil, err
			}
		}
	}
	if dashCmd != "" {
		unnamed = append(unnamed, config.SourceConfig{Type: "stream", Cmd: dashCmd})
	}
	for _, src := range unnamed {
		if _, err := addSource(flags, pageNames, "", src, allowExec); err != nil {
			return nil, err
		}
	}
	return flags, nil
}

// addSource adds a parsed source to flags: one place for the rules every
// path shares. A source with no name gets defaultLabel's label, made unique
// with -2, -3, ... against flags and pageNames, so two commands with the
// same label both show and neither replaces a page's source.
func addSource(flags map[string]config.SourceConfig, pageNames map[string]bool, name string, src config.SourceConfig, allowExec bool) (string, error) {
	if name == "" {
		name = uniqueLabel(sourceLabel(src), func(l string) bool { return flags[l].Type != "" || pageNames[l] })
	}
	switch {
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
// page uses. A page uses a flag when it has the flag's name (pageSourceNames),
// or when one of the page's ["*"] merges takes it, as server.ResolveSource
// expands it for the page's blocks.
func checkFlagSources(cfg *config.Config, srv *server.Server) error {
	used := pageSourceNames(cfg, srv)
	for _, route := range srv.Routes() {
		if route.Page == nil {
			continue
		}
		for _, name := range server.VisibleSourceNames(cfg, route.Page) {
			if src, ok := server.ResolveSource(cfg, route.Page, name); ok && src.Type == "merge" {
				for _, in := range src.Inputs {
					used[in] = true
				}
			}
		}
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
			need, hint := src.Accepts.Need()
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
func promptSources(in io.Reader, out io.Writer, flags map[string]config.SourceConfig, pageNames map[string]bool, allowExec bool) []string {
	fmt.Fprintln(out, "Add one as a command, file:PATH or http:PATH, optionally after NAME=. Leave empty to finish.")
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
		name, err := addFlagSource(flags, pageNames, line, allowExec)
		if err != nil {
			fmt.Fprintln(out, err)
			continue
		}
		given = append(given, sourceFlagValue(name, flags[name]))
	}
}

// reuseLine is the serve command line that repeats this run without the
// prompt: the original arguments plus a --source for each answer. The new
// flags go before a "--", since everything after it is one command.
func reuseLine(args, given []string) string {
	before, after := args, []string(nil)
	if i := slices.Index(args, "--"); i >= 0 {
		before, after = args[:i], args[i:]
	}
	parts := append([]string{"tinkerdown", "serve"}, quoteArgs(before)...)
	for _, g := range given {
		parts = append(parts, "--source", shellQuote(g))
	}
	return strings.Join(append(parts, quoteArgs(after)...), " ")
}

// quoteArgs quotes each argument that needs it for a POSIX shell.
func quoteArgs(args []string) []string {
	quoted := make([]string, len(args))
	for i, a := range args {
		quoted[i] = shellQuoteIfNeeded(a)
	}
	return quoted
}

// commandAfterDashes joins the arguments after "--" into one shell command,
// each quoted when it needs it, so the command runs as typed. Only the first
// "--" ends serve's flags; later ones are the command's own.
func commandAfterDashes(args []string) string {
	return strings.Join(quoteArgs(args), " ")
}

func shellQuoteIfNeeded(s string) string {
	if s != "" && strings.Trim(s, "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789-_./:=") == "" {
		return s
	}
	return shellQuote(s)
}
