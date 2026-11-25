package commands

import (
	"fmt"
	"io"
	"maps"
	"regexp"
	"slices"
	"strings"

	"golang.org/x/net/html"

	"github.com/livetemplate/tinkerdown"
	"github.com/livetemplate/tinkerdown/internal/config"
	"github.com/livetemplate/tinkerdown/internal/runtime"
	"github.com/livetemplate/tinkerdown/internal/security"
	"github.com/livetemplate/tinkerdown/internal/server"
)

// describeTakes prints what each page takes from the command line, for
// `serve <page> --help`. It reads only what the page already declares: its
// ["*"] merges take any sources, a merge's other inputs that nothing defines
// are sources the operator names, and its buttons name actions. Page text
// (names come from the page, hints too) is cleaned before it is printed.
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
		// Help runs before any --source is read, so a name that resolves is
		// one the page or tinkerdown.yaml defines.
		defined := func(name string) bool {
			_, ok := server.ResolveSource(cfg, page, name)
			return ok
		}
		onButtons := map[string]bool{}
		for _, name := range slices.Sorted(maps.Keys(page.Config.Sources)) {
			src, ok := server.ResolveSource(cfg, page, name)
			if !ok || src.Type != "merge" {
				continue
			}
			if src.On != "" {
				onButtons[src.On] = true
			}
			if src.Wildcard {
				flag, what := wildcardTakes(name, src.Accepts)
				add(flag, what)
				continue
			}
			for _, in := range src.Inputs {
				if defined(in) {
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
		for _, b := range pageButtons(page.InteractiveBlocks) {
			if !onButtons[b] && !runtime.IsBuiltinAction(b) {
				add("--action "+b+"=COMMAND", "run by the "+b+" button")
			}
		}
		label := shown
		if label == "" {
			label = route.FilePath
		}
		if len(lines) == 0 {
			fmt.Fprintf(w, "\n%s takes no sources or actions.\n", security.CleanPageText(label))
			continue
		}
		fmt.Fprintf(w, "\n%s takes:\n%s\n", security.CleanPageText(label), strings.Join(lines, "\n"))
	}
}

// wildcardTakes says how to pass a ["*"] merge's sources, and what it takes:
// how many, and the page's hint.
func wildcardTakes(name string, a *config.SourceAccepts) (flag, what string) {
	need, hint, kinds := 1, "", []string(nil)
	if a != nil {
		need, hint, kinds = max(a.Min, 1), a.Hint, a.Kinds
	}
	if len(kinds) == 0 {
		kinds = []string{"cmd", "file", "http"}
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
	what = fmt.Sprintf("%d or more, shown in %s", need, name)
	if hint != "" {
		what += fmt.Sprintf(". The page says: %q", hint)
	}
	return strings.Join(forms, ", "), what
}

// templateAction is a {{ }} action, replaced before tokenizing so a
// template reads as HTML.
var templateAction = regexp.MustCompile(`\{\{.*?\}\}`)

// pageButtons lists the names of the buttons in a page's lvt blocks, sorted
// and without repeats. A name built by a template action is skipped, since
// only a fixed name can be passed as --action.
func pageButtons(blocks map[string]*tinkerdown.InteractiveBlock) []string {
	var names []string
	for _, b := range blocks {
		z := html.NewTokenizer(strings.NewReader(templateAction.ReplaceAllString(b.Content, "{{}}")))
		for tt := z.Next(); tt != html.ErrorToken; tt = z.Next() {
			tok := z.Token()
			if tok.Data != "button" || (tt != html.StartTagToken && tt != html.SelfClosingTagToken) {
				continue
			}
			for _, a := range tok.Attr {
				if a.Key == "name" && a.Val != "" && !strings.Contains(a.Val, "{{") {
					names = append(names, a.Val)
				}
			}
		}
	}
	slices.Sort(names)
	return slices.Compact(names)
}
