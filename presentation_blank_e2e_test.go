//go:build !ci

package tinkerdown_test

import (
	"context"
	"encoding/json"
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
	// A tabs heading, then ## sections: the tabs content goes on the first
	// slide, with the first section.
	"mixed.md": "---\ntitle: Mixed\n---\n\n## [All] | [Open] status = open\n\nTab body.\n\n## Later\n\nLater body.\n\n## Last\n\nLast body.\n",
	// Ordinary ## sections, one slide each.
	"sections.md": "---\ntitle: Sections\n---\n\n## First\n\nFirst body.\n\n## Second\n\nSecond body.\n",
}

// TestPresentationShowsContent_E2E enters presentation mode on each page and
// checks that the first slide shows visible text, not a blank screen.
func TestPresentationShowsContent_E2E(t *testing.T) {
	serverLog := captureServerLog(t)
	dir := t.TempDir()
	for name, body := range presentationPages {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
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

	// Each page's first slide must show want, and not hide, which is on a
	// later slide.
	for page, c := range map[string]struct{ want, hidden string }{
		"tabs":     {want: "Tab body"},
		"plain":    {want: "Just a paragraph"},
		"mixed":    {want: "Tab body", hidden: "Last body"},
		"sections": {want: "First body", hidden: "Second body"},
	} {
		t.Run(page, func(t *testing.T) {
			want, _ := json.Marshal(c.want)
			actx, cancel := context.WithTimeout(ctx, 20*time.Second)
			defer cancel()
			var before, after string
			if err := chromedp.Run(actx,
				chromedp.Navigate(base+"/"+page),
				waitForDOM(`!!document.querySelector("#presentation-toggle") && `+visibleText+`.includes(`+string(want)+`)`, 15*time.Second),
				chromedp.Evaluate(visibleText, &before),
				chromedp.Click("#presentation-toggle", chromedp.ByQuery),
				waitForDOM(`document.body.classList.contains("presentation-mode")`, 5*time.Second),
				chromedp.Evaluate(visibleText, &after),
			); err != nil {
				dump()
				t.Fatal(err)
			}
			if !strings.Contains(after, c.want) {
				dump()
				t.Fatalf("presentation mode shows %q, want it to include %q (before: %q)", after, c.want, before)
			}
			if c.hidden != "" && strings.Contains(after, c.hidden) {
				dump()
				t.Fatalf("the first slide shows %q, which belongs on a later slide: %q", c.hidden, after)
			}
		})
	}
}
