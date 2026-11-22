//go:build !ci

package tinkerdown_test

import (
	"context"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/chromedp"
	"github.com/livetemplate/tinkerdown/internal/config"
	"github.com/livetemplate/tinkerdown/internal/server"
)

// presentationPages are pages of each shape a presentation can start on. The
// first slide must show something on every one of them.
var presentationPages = map[string]string{
	// A tabs heading is the page's only ##, as in onefile's apps.
	"tabs.md": "---\ntitle: Tabs\n---\n\n## [All] | [Open] status = open\n\nTab body.\n",
	// No ## at all.
	"plain.md": "---\ntitle: Plain\n---\n\nJust a paragraph, with no sections.\n",
	// Ordinary ## sections, one slide each.
	"sections.md": "---\ntitle: Sections\n---\n\n## First\n\nFirst body.\n\n## Second\n\nSecond body.\n",
}

// TestPresentationShowsContent_E2E enters presentation mode on each page and
// checks that the first slide shows visible text, not a blank screen.
func TestPresentationShowsContent_E2E(t *testing.T) {
	serverLog := captureServerLog(t)
	dir := t.TempDir()
	for name, body := range presentationPages {
		os.WriteFile(filepath.Join(dir, name), []byte(body), 0o644)
	}
	cfg, err := config.LoadFromDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	srv := server.NewWithConfig(dir, cfg)
	if err := srv.Discover(); err != nil {
		t.Fatal(err)
	}
	ts := httptest.NewServer(server.WithCompression(srv))
	t.Cleanup(func() { ts.Close(); srv.StopStreams() })
	base := ConvertURLForDockerChrome(ts.URL)

	chrome, cleanup := SetupDockerChrome(t, 90*time.Second)
	defer cleanup()
	ctx := chrome.Context
	logs := &streamBrowserLogs{}
	logs.listen(ctx)
	if err := chromedp.Run(ctx, network.Enable()); err != nil {
		t.Fatal(err)
	}
	dump := func() {
		var html string
		chromedp.Run(ctx, chromedp.OuterHTML("body", &html, chromedp.ByQuery))
		t.Logf("console:\n%s", strings.Join(logs.console, "\n"))
		frames := logs.frames()
		t.Logf("ws received: %d frames, last: %s", len(frames), strings.Join(frames[max(0, len(frames)-3):], "\n"))
		t.Logf("rendered html: %s", tail(html, 4000))
		t.Logf("server log tail: %s", tail(serverLog.String(), 4000))
	}

	// visibleText is the text of the page's content that is rendered on
	// screen, ignoring the toolbar and the navigation.
	const visibleText = `(() => {
		const w = document.querySelector(".content-wrapper");
		if (!w) return "";
		return [...w.querySelectorAll("*")].filter(el => el.checkVisibility() && el.children.length === 0)
			.map(el => el.textContent.trim()).filter(Boolean).join(" ");
	})()`

	for page, want := range map[string]string{"tabs": "Tab body", "plain": "Just a paragraph", "sections": "First body"} {
		t.Run(page, func(t *testing.T) {
			actx, cancel := context.WithTimeout(ctx, 20*time.Second)
			defer cancel()
			var before, after string
			if err := chromedp.Run(actx,
				chromedp.Navigate(base+"/"+page),
				waitForDOM(`!!document.querySelector("#presentation-toggle") && `+visibleText+`.includes(`+jsString(want)+`)`, 15*time.Second),
				chromedp.Evaluate(visibleText, &before),
				chromedp.Click("#presentation-toggle", chromedp.ByQuery),
				waitForDOM(`document.body.classList.contains("presentation-mode")`, 5*time.Second),
				chromedp.Evaluate(visibleText, &after),
			); err != nil {
				dump()
				t.Fatal(err)
			}
			if !strings.Contains(after, want) {
				dump()
				t.Fatalf("presentation mode shows %q, want it to include %q (before: %q)", after, want, before)
			}
		})
	}
}

// jsString quotes s as a JavaScript string literal.
func jsString(s string) string {
	return `"` + strings.NewReplacer(`\`, `\\`, `"`, `\"`).Replace(s) + `"`
}
