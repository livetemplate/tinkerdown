//go:build !ci

package tinkerdown_test

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
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

// TestRemotePageCSP_E2E serves a page the way serve serves one from a URL,
// with flag sources. Its template tries to send every row to another origin
// through an <img>; the strict policy must block that while the page works.
func TestRemotePageCSP_E2E(t *testing.T) {
	serverLog := captureServerLog(t)
	// The default policy allows https: images, so the other origin is https.
	// Chrome rejects its test certificate, but even trying connects, so the
	// test counts connections, not requests.
	var leaked atomic.Int64
	other := httptest.NewUnstartedServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	other.Config.ConnState = func(_ net.Conn, s http.ConnState) {
		if s == http.StateNew {
			leaked.Add(1)
		}
	}
	other.StartTLS()
	defer other.Close()
	otherURL := ConvertURLForDockerChrome(other.URL)

	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "api.log"), []byte("secret one\nsecret two\n"), 0o644)
	page := "---\ntitle: Remote\nsources:\n  all:\n    type: merge\n    inputs: [\"*\"]\n---\n\n## [All] | [*] source\n\n```lvt\n" +
		`<ol lvt-source="all" data-testid="log">{{range .Data}}<li data-key="{{.seq}}" class="src-{{.source}}">{{.msg}}<img src="` + otherURL + `/?{{.msg}}" alt=""></li>{{end}}</ol>` + "\n```\n"
	os.WriteFile(filepath.Join(dir, "page.md"), []byte(page), 0o600)

	cfg := config.DefaultConfig()
	cfg.FlagSources = map[string]config.SourceConfig{"api": {Type: "stream", File: filepath.Join(dir, "api.log")}}
	srv := server.NewWithConfig(dir, cfg)
	srv.ServeSinglePage("page.md", true)
	if err := srv.Discover(); err != nil {
		t.Fatal(err)
	}
	ts := httptest.NewServer(server.WithCompression(srv))
	defer ts.Close()
	defer srv.StopStreams()

	chrome, cleanup := SetupDockerChrome(t, 90*time.Second)
	defer cleanup()
	ctx := chrome.Context
	logs := &streamBrowserLogs{}
	logs.listen(ctx)
	dump := func() {
		t.Logf("console:\n%s", strings.Join(logs.console, "\n"))
		t.Logf("server log tail: %s", tail(serverLog.String(), 3000))
	}
	if err := chromedp.Run(ctx, network.Enable(), chromedp.Navigate(ConvertURLForDockerChrome(ts.URL)),
		chromedp.WaitVisible(`[data-testid="log"]`, chromedp.ByQuery)); err != nil {
		dump()
		t.Fatal(err)
	}
	if err := chromedp.Run(ctx, waitForDOM(`document.querySelectorAll("li.src-api").length === 2 && document.querySelectorAll('button[role="tab"]').length === 2`, 15*time.Second)); err != nil {
		dump()
		t.Fatalf("the page didn't work under the strict policy: %v", err)
	}
	time.Sleep(time.Second) // give any image request time to arrive
	if n := leaked.Load(); n != 0 {
		dump()
		t.Fatalf("the other origin got %d connections; the policy must block them", n)
	}
}

// TestKeyedMerge_E2E follows go-test-like JSON lines through a keyed merge:
// one row per test, updated in place as its events arrive, with its output
// collected. The update must reach the page as a range ["u", ...] op.
func TestKeyedMerge_E2E(t *testing.T) {
	serverLog := captureServerLog(t)
	dir := t.TempDir()
	events := filepath.Join(dir, "events.jsonl")
	os.WriteFile(events, []byte(`{"Action":"run","Package":"api","Test":"TestA"}`+"\n"), 0o644)
	page := "---\ntitle: Tests\nsources:\n  tests:\n    type: merge\n    inputs: [\"*\"]\n    key: [fields.Package, fields.Test]\n    collect: fields.Output\n---\n\n## [All] | [Failed] fields.Action = fail\n\n```lvt\n" +
		`<ol lvt-source="tests" data-testid="tests">{{range .Data}}<li data-key="{{.key}}" class="a-{{.fields.Action}}"><b>{{.fields.Test}}</b> <i>{{.fields.Action}}</i><pre>{{.collected}}</pre></li>{{end}}</ol>` + "\n```\n"
	os.WriteFile(filepath.Join(dir, "index.md"), []byte(page), 0o644)

	cfg := config.DefaultConfig()
	cfg.FlagSources = map[string]config.SourceConfig{"go": {Type: "stream", File: events, Format: "json"}}
	srv := server.NewWithConfig(dir, cfg)
	if err := srv.Discover(); err != nil {
		t.Fatal(err)
	}
	ts := httptest.NewServer(server.WithCompression(srv))
	defer ts.Close()
	defer srv.StopStreams()

	chrome, cleanup := SetupDockerChrome(t, 90*time.Second)
	defer cleanup()
	ctx := chrome.Context
	logs := &streamBrowserLogs{}
	logs.listen(ctx)
	dump := func() {
		frames := logs.frames()
		t.Logf("console:\n%s", strings.Join(logs.console, "\n"))
		t.Logf("ws received: %d frames, last: %s", len(frames), strings.Join(frames[max(0, len(frames)-4):], "\n"))
		t.Logf("server log tail: %s", tail(serverLog.String(), 3000))
	}
	wait := func(what, js string) {
		t.Helper()
		if err := chromedp.Run(ctx, waitForDOM(js, 15*time.Second)); err != nil {
			dump()
			t.Fatalf("waiting for %s: %v", what, err)
		}
	}
	appendEvents := func(lines ...string) {
		f, _ := os.OpenFile(events, os.O_APPEND|os.O_WRONLY, 0)
		for _, l := range lines {
			fmt.Fprintln(f, l)
		}
		f.Close()
	}
	if err := chromedp.Run(ctx, network.Enable(), chromedp.Navigate(ConvertURLForDockerChrome(ts.URL)),
		chromedp.WaitVisible(`[data-testid="tests"]`, chromedp.ByQuery)); err != nil {
		dump()
		t.Fatal(err)
	}
	wait("TestA running", `document.querySelectorAll("li.a-run").length === 1`)

	before := len(logs.frames())
	appendEvents(
		`{"Action":"output","Package":"api","Test":"TestA","Output":"a_test.go:9: boom\n"}`,
		`{"Action":"fail","Package":"api","Test":"TestA"}`,
		`{"Action":"run","Package":"api","Test":"TestB"}`,
		`{"Action":"pass","Package":"api","Test":"TestB"}`,
	)
	wait("one row per test, updated in place",
		`document.querySelectorAll("ol li").length === 2 && !!document.querySelector("li.a-fail") && document.querySelector("li.a-fail pre").textContent.includes("boom") && !!document.querySelector("li.a-pass")`)
	var order string
	chromedp.Run(ctx, chromedp.Evaluate(`[...document.querySelectorAll("ol li b")].map(b => b.textContent).join(",")`, &order))
	if order != "TestA,TestB" {
		t.Fatalf("rows = %s, want TestA first (first seen), then TestB", order)
	}
	updated := false
	for _, f := range logs.frames()[before:] {
		if strings.Contains(f, `["u",`) {
			updated = true
		}
	}
	if !updated {
		dump()
		t.Fatal(`no ["u", ...] op: the keyed row was not updated in place`)
	}

	if err := chromedp.Run(ctx, chromedp.Click(`button[role="tab"][data-filter="fields.Action = fail"]`, chromedp.ByQuery)); err != nil {
		dump()
		t.Fatal(err)
	}
	wait("only the failing test", `(() => { const r = document.querySelectorAll("ol li"); return r.length === 1 && r[0].classList.contains("a-fail"); })()`)
}
