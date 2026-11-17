package commands

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeTree(t *testing.T, files map[string]string) string {
	t.Helper()
	dir := t.TempDir()
	for name, body := range files {
		path := filepath.Join(dir, name)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

const navSiteYAML = `type: site
title: Docs
navigation:
  - title: Guides
    path: guides
    pages:
      - title: Scaling
        path: guides/scaling.md
`

// A site page with no nav entry renders nowhere: serve 404s its URL. Every
// other validate check passes for it, so this one must fail the run.
func TestValidateFlagsPageMissingFromNavigation(t *testing.T) {
	dir := writeTree(t, map[string]string{
		"tinkerdown.yaml":             navSiteYAML,
		"guides/scaling.md":           "# Scaling\n",
		"guides/custom-attributes.md": "# Custom attributes\n",
		"_partials/snippet.md":        "included, never served\n",
		"README.md":                   "# Repo readme\n",
	})
	out, err := captureStdout(t, func() error { return ValidateCommand([]string{dir}) })
	if err == nil {
		t.Fatalf("validate passed with an unrouted page:\n%s", out)
	}
	if !strings.Contains(out, "guides/custom-attributes.md") || !strings.Contains(out, "no navigation entry") {
		t.Errorf("output does not name the unrouted page:\n%s", out)
	}
	if strings.Contains(out, "guides/scaling.md:") || strings.Contains(out, "_partials") || strings.Contains(out, "README.md:") {
		t.Errorf("output flags a routed, underscore or repo-meta page:\n%s", out)
	}
}

func TestValidateRoutingCheckOnlyForNavSites(t *testing.T) {
	for name, files := range map[string]map[string]string{
		"site with nav, all listed": {"tinkerdown.yaml": navSiteYAML},
		"site without nav":          {"tinkerdown.yaml": "type: site\ntitle: Docs\n", "welcome.md": "# Welcome\n"},
		"tutorial":                  {"tinkerdown.yaml": "title: Docs\n", "welcome.md": "# Welcome\n"},
		// serve routes site.home even when the nav does not list it.
		"home outside the nav": {
			"tinkerdown.yaml": navSiteYAML + "site:\n  home: welcome.md\n",
			"welcome.md":      "# Welcome\n",
		},
	} {
		t.Run(name, func(t *testing.T) {
			files["guides/scaling.md"] = "# Scaling\n"
			dir := writeTree(t, files)
			out, err := captureStdout(t, func() error { return ValidateCommand([]string{dir}) })
			if err != nil {
				t.Errorf("validate failed: %v\n%s", err, out)
			}
		})
	}
}
