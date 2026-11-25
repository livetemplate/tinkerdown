package commands

import (
	"cmp"
	"fmt"
	"io"
	"slices"
	"strings"

	"github.com/livetemplate/tinkerdown/internal/config"
	"github.com/livetemplate/tinkerdown/internal/security"
	"github.com/livetemplate/tinkerdown/internal/server"
)

// describeTakes prints what each page takes from the command line, for
// `serve <page> --help`. It reads only what the page already declares: its
// ["*"] merges take any sources, a merge's other inputs that nothing defines
// are sources the operator names, and the actions its markup invokes are
// --action flags. It walks the same sources checkFlagSources and sourceNeeds
// do. Page text (names and hints) is cleaned before it is printed.
func describeTakes(w io.Writer, cfg *config.Config, srv *server.Server, shown string) {
	for _, route := range srv.Routes() {
		page := route.Page
		if page == nil {
			continue
		}
		var lines []string
		add := func(flag, what string) {
			lines = append(lines, fmt.Sprintf("  %s\n      %s", security.CleanPageText(flag), security.CleanPageText(what)))
		}
		onButtons := map[string]bool{}
		for _, name := range slices.Compact(slices.Sorted(slices.Values(server.VisibleSourceNames(cfg, page)))) {
			src, ok := server.ResolveSource(cfg, page, name)
			if !ok || src.Type != "merge" {
				continue
			}
			if src.On != "" {
				onButtons[src.On] = true
			}
			if src.Wildcard {
				add(wildcardForms(src.Accepts), wildcardTakes(name, src.Accepts))
				continue
			}
			for _, in := range src.Inputs {
				// Help reads no --source, so an input that resolves is one
				// the page or tinkerdown.yaml defines.
				if _, ok := server.ResolveSource(cfg, page, in); ok {
					continue
				}
				what := "shown in " + name
				switch {
				case src.On != "":
					what = fmt.Sprintf("run by the %s button, shown in %s", src.On, name)
				case len(src.Params) > 0:
					what = fmt.Sprintf("run for the row you open, shown in %s", name)
				}
				add("--source "+in+"=…", what)
			}
		}
		// Refs lists the actions the markup invokes, sorted and without
		// builtins. A name built by a template action can't be passed as
		// --action, and an on: button starts a merge, not an action.
		for _, a := range page.Refs().Actions {
			if !onButtons[a] && !strings.Contains(a, "{{") {
				add("--action "+a+"=COMMAND", "run by the "+a+" button")
			}
		}
		label := security.CleanPageText(cmp.Or(shown, route.FilePath))
		if len(lines) == 0 {
			fmt.Fprintf(w, "\n%s takes no sources or actions.\n", label)
			continue
		}
		fmt.Fprintf(w, "\n%s takes:\n%s\n", label, strings.Join(lines, "\n"))
	}
}

// wildcardForms is how to pass a ["*"] merge's sources, by the kinds it
// accepts (all three by default).
func wildcardForms(a *config.SourceAccepts) string {
	kinds := []string{"cmd", "file", "http"}
	if a != nil && len(a.Kinds) > 0 {
		kinds = a.Kinds
	}
	var forms []string
	for _, k := range kinds {
		switch k {
		case "cmd":
			forms = append(forms, "-- COMMAND")
		case "file", "http":
			forms = append(forms, "--source "+k+":PATH")
		}
	}
	return strings.Join(forms, ", ")
}

// wildcardTakes says what a ["*"] merge takes: how many, and the page's hint.
func wildcardTakes(name string, a *config.SourceAccepts) string {
	need, hint := a.Need()
	what := fmt.Sprintf("%d or more, shown in %s", need, name)
	if hint != "" {
		what += fmt.Sprintf(". The page says: %q", hint)
	}
	return what
}
