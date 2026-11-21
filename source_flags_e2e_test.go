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

// flagsPage takes its sources from serve --source: a ["*"] merge, a tab per
// source, and an empty state. It declares no command or file of its own.
const flagsPage = "---\ntitle: Logs\nsources:\n  all:\n    type: merge\n    inputs: [\"*\"]\n    accepts:\n      min: 1\n      hint: Logs to follow\n  broken:\n    type: merge\n    inputs: [nope]\n---\n\n" +
	"## [All] | [*] source\n\n```lvt\n" + `<main lvt-source="all">
  {{if not .Inputs}}<p data-testid="empty">No log sources. {{.Accepts.Hint}}</p>{{end}}
  <ol data-testid="log">{{range .Data}}<li data-key="{{.seq}}" class="src-{{.source}}">{{.msg}}</li>{{end}}</ol>
</main>` + "\n```\n\n## Broken\n\n```lvt\n" + `<p lvt-source="broken">{{len .Data}}</p>` + "\n```\n"

// TestSourceFlags_E2E serves flagsPage with sources from flags, as
// `serve --source api=file:api.log --source app=file:app.log` would, and
// checks the tabs, the filtering, the empty state and a failed block.
func TestSourceFlags_E2E(t *testing.T) {
	serverLog := captureServerLog(t)
	dir := t.TempDir()
	for name, lines := range map[string]string{"api.log": "api one\napi two\n", "app.log": "app one\n"} {
		os.WriteFile(filepath.Join(dir, name), []byte(lines), 0o644)
	}
	os.WriteFile(filepath.Join(dir, "index.md"), []byte(flagsPage), 0o644)

	serve := func(flags map[string]config.SourceConfig) string {
		cfg, err := config.LoadFromDir(dir)
		if err != nil {
			t.Fatal(err)
		}
		cfg.FlagSources = flags
		srv := server.NewWithConfig(dir, cfg)
		if err := srv.Discover(); err != nil {
			t.Fatal(err)
		}
		ts := httptest.NewServer(server.WithCompression(srv))
		t.Cleanup(func() { ts.Close(); srv.StopStreams() })
		return ConvertURLForDockerChrome(ts.URL)
	}
	withFlags := serve(map[string]config.SourceConfig{
		"api": {Type: "stream", File: "api.log"},
		"app": {Type: "stream", File: "app.log"},
	})
	noFlags := serve(nil)

	chrome, cleanup := SetupDockerChrome(t, 90*time.Second)
	defer cleanup()
	ctx := chrome.Context
	logs := &streamBrowserLogs{}
	logs.listen(ctx)
	dump := func() {
		var html string
		chromedp.Run(ctx, chromedp.OuterHTML("body", &html, chromedp.ByQuery))
		t.Logf("console:\n%s", strings.Join(logs.console, "\n"))
		frames := logs.frames()
		t.Logf("ws received: %d frames, last: %s", len(frames), strings.Join(frames[max(0, len(frames)-3):], "\n"))
		t.Logf("rendered html: %s", tail(html, 4000))
		t.Logf("server log tail: %s", tail(serverLog.String(), 6000))
	}
	run := func(actions ...chromedp.Action) {
		t.Helper()
		actx, cancel := context.WithTimeout(ctx, 15*time.Second)
		defer cancel()
		if err := chromedp.Run(actx, actions...); err != nil {
			dump()
			t.Fatal(err)
		}
	}
	wait := func(what, js string) {
		t.Helper()
		if err := chromedp.Run(ctx, waitForDOM(js, 15*time.Second)); err != nil {
			dump()
			t.Fatalf("waiting for %s: %v", what, err)
		}
	}

	// The first Run creates the tab on the long-lived context.
	if err := chromedp.Run(ctx, network.Enable(), chromedp.Navigate(withFlags), chromedp.WaitVisible(`[data-testid="log"]`, chromedp.ByQuery)); err != nil {
		dump()
		t.Fatal(err)
	}

	t.Run("one tab per flag source", func(t *testing.T) {
		var tabs []string
		run(chromedp.Evaluate(`[...document.querySelectorAll('button[role="tab"]')].map(b => b.textContent + "|" + b.dataset.filter)`, &tabs))
		if want := "All||api|source = api|app|source = app"; strings.Join(tabs, "|") != want {
			t.Fatalf("tabs = %q, want %q", strings.Join(tabs, "|"), want)
		}
		wait("rows from both flag sources", `document.querySelectorAll("li.src-api").length === 2 && document.querySelectorAll("li.src-app").length === 1`)
	})

	t.Run("a source tab filters to it", func(t *testing.T) {
		run(chromedp.Click(`button[role="tab"][data-filter="source = app"]`, chromedp.ByQuery))
		wait("only app rows", `(() => { const r = [...document.querySelectorAll("ol li")]; return r.length === 1 && r[0].classList.contains("src-app"); })()`)
	})

	t.Run("an invalid source shows its error, not Connecting", func(t *testing.T) {
		wait("the block error", `[...document.querySelectorAll(".tinkerdown-block-error")].some(e => e.textContent.includes("is not a defined source"))`)
		var text string
		run(chromedp.Evaluate(`[...document.querySelectorAll(".tinkerdown-block-error")].map(e => e.textContent).join("|")`, &text))
		if !strings.Contains(text, `input "nope"`) {
			t.Fatalf("block error reads %q; quotes should display as quotes", text)
		}
	})

	t.Run("no sources shows the empty state", func(t *testing.T) {
		run(chromedp.Navigate(noFlags), chromedp.WaitVisible(`[data-testid="log"]`, chromedp.ByQuery))
		wait("the empty state", `(document.querySelector('[data-testid="empty"]')?.textContent || "").includes("Logs to follow")`)
		var n int
		run(chromedp.Evaluate(`document.querySelectorAll('button[role="tab"]').length`, &n))
		if n != 1 {
			t.Fatalf("%d tabs with no sources, want just All", n)
		}
	})

	if t.Failed() {
		dump()
	}
}
