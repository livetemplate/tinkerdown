//go:build !ci

package tinkerdown_test

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
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
	page := "---\ntitle: Tests\nsources:\n  tests:\n    type: merge\n    inputs: [\"*\"]\n    key: [fields.Package, fields.Test]\n    collect: fields.Output\n    drop: fields.Action=skip|gone\n---\n\n## [All] | [Failed] fields.Action = fail\n\n```lvt\n" +
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
	if !sawOp(logs.frames()[before:], "u") {
		dump()
		t.Fatal(`no ["u", ...] op: the keyed row was not updated in place`)
	}

	if err := chromedp.Run(ctx, chromedp.Click(`button[role="tab"][data-filter="fields.Action = fail"]`, chromedp.ByQuery)); err != nil {
		dump()
		t.Fatal(err)
	}
	wait("only the failing test", `(() => { const r = document.querySelectorAll("ol li"); return r.length === 1 && r[0].classList.contains("a-fail"); })()`)

	// A drop row removes its key's row: the page gets a remove op, not a new row.
	if err := chromedp.Run(ctx, chromedp.Click(`button[role="tab"][data-filter=""]`, chromedp.ByQuery)); err != nil {
		dump()
		t.Fatal(err)
	}
	wait("both tests again", `document.querySelectorAll("ol li").length === 2`)
	before = len(logs.frames())
	appendEvents(`{"Action":"gone","Package":"api","Test":"TestA"}`)
	wait("TestA dropped", `(() => { const r = [...document.querySelectorAll("ol li b")].map(b => b.textContent); return r.length === 1 && r[0] === "TestB"; })()`)
	if !sawOp(logs.frames()[before:], "r") {
		dump()
		t.Fatal(`no ["r", ...] op: the dropped row was not removed in place`)
	}
}

// TestFlagActions_E2E clicks row buttons on a page that declares no actions:
// each runs an --action command with the row's value as $TD_name, a value
// that looks like shell code arrives as text, and a refused value runs nothing
// and shows the block's error.
func TestFlagActions_E2E(t *testing.T) {
	serverLog := captureServerLog(t)
	orig := config.IsExecAllowed()
	config.SetAllowExec(true)
	defer config.SetAllowExec(orig)

	dir := t.TempDir()
	feed := filepath.Join(dir, "feed.jsonl")
	os.WriteFile(feed, []byte(`{"name":"web"}`+"\n"+`{"name":"a b $(touch pwned)"}`+"\n"+`{"name":"-rf"}`+"\n"), 0o644)
	page := "---\ntitle: Rows\nsources:\n  rows:\n    type: merge\n    inputs: [\"*\"]\n    key: [fields.name]\n---\n\n```lvt\n" +
		`<div lvt-source="rows">{{if .Error}}<p data-testid="error">{{.Error}}</p>{{end}}<ol data-testid="rows">{{range .Data}}<li data-key="{{.key}}"><button name="mark" data-name="{{.fields.name}}">{{.fields.name}}</button> <button name="slow" data-name="{{.fields.name}}">slow</button></li>{{end}}</ol></div>` + "\n```\n"
	os.WriteFile(filepath.Join(dir, "index.md"), []byte(page), 0o644)

	cfg := config.DefaultConfig()
	cfg.FlagSources = map[string]config.SourceConfig{"feed": {Type: "stream", File: feed, Format: "json"}}
	cfg.FlagActions = map[string]*config.Action{
		"mark": {Kind: "exec", Cmd: `printf '%s\n' "$TD_name" >> clicked.txt`, FromFlag: true},
		"slow": {Kind: "exec", Cmd: `sleep 2; printf 'slow %s\n' "$TD_name" >> slow.txt`, FromFlag: true},
	}
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
		var html string
		chromedp.Run(ctx, chromedp.OuterHTML("body", &html, chromedp.ByQuery))
		t.Logf("console:\n%s", strings.Join(logs.console, "\n"))
		t.Logf("ws received: %d frames, last: %s", len(frames), strings.Join(frames[max(0, len(frames)-4):], "\n"))
		logs.mu.Lock()
		t.Logf("ws sent:\n%s", strings.Join(logs.wsSent, "\n"))
		logs.mu.Unlock()
		t.Logf("html: %s", tail(html, 3000))
		t.Logf("server log tail: %s", tail(serverLog.String(), 3000))
	}
	clicked := func() string {
		b, _ := os.ReadFile(filepath.Join(dir, "clicked.txt"))
		return string(b)
	}
	if err := chromedp.Run(ctx, network.Enable(), chromedp.Navigate(ConvertURLForDockerChrome(ts.URL)),
		chromedp.WaitVisible(`[data-testid="rows"]`, chromedp.ByQuery),
		waitForDOM(`document.querySelectorAll('[data-testid="rows"] button[name="mark"]').length === 3`, 15*time.Second)); err != nil {
		dump()
		t.Fatal(err)
	}
	click := func(i int, want string) {
		t.Helper()
		if err := chromedp.Run(ctx, chromedp.Click(fmt.Sprintf(`[data-testid="rows"] li:nth-child(%d) button`, i), chromedp.ByQuery)); err != nil {
			dump()
			t.Fatal(err)
		}
		deadline := time.Now().Add(10 * time.Second)
		for clicked() != want && time.Now().Before(deadline) {
			time.Sleep(50 * time.Millisecond)
		}
		if got := clicked(); got != want {
			dump()
			t.Fatalf("clicked.txt = %q, want %q", got, want)
		}
	}
	click(1, "web\n")
	click(2, "web\na b $(touch pwned)\n")
	if _, err := os.Stat(filepath.Join(dir, "pwned")); err == nil {
		t.Fatal("a value ran as shell code")
	}

	if err := chromedp.Run(ctx, chromedp.Click(`[data-testid="rows"] li:nth-child(3) button`, chromedp.ByQuery),
		waitForDOM(`(document.querySelector('[data-testid="error"]')?.textContent || "").includes("starts with -")`, 10*time.Second)); err != nil {
		dump()
		t.Fatalf("a refused value showed no error: %v", err)
	}
	if got := clicked(); got != "web\na b $(touch pwned)\n" {
		t.Fatalf("a refused value ran: clicked.txt = %q", got)
	}

	// A button stays aria-busy until the server answers its click, and a
	// second click on a busy button sends nothing.
	const slow = `[data-testid="rows"] li:nth-child(1) button[name="slow"]`
	if err := chromedp.Run(ctx, chromedp.Click(slow, chromedp.ByQuery),
		waitForDOM(`document.querySelector('`+slow+`').getAttribute("aria-busy") === "true"`, 2*time.Second),
		chromedp.Evaluate(`document.querySelector('`+slow+`').click()`, nil)); err != nil {
		dump()
		t.Fatalf("a running action's button isn't busy: %v", err)
	}
	if err := chromedp.Run(ctx, waitForDOM(`document.querySelector('`+slow+`').getAttribute("aria-busy") === null`, 10*time.Second)); err != nil {
		dump()
		t.Fatalf("the button stayed busy after its action finished: %v", err)
	}
	if b, _ := os.ReadFile(filepath.Join(dir, "slow.txt")); string(b) != "slow web\n" {
		t.Fatalf("slow.txt = %q; a click on a busy button must not send again", b)
	}
	// Each click the server ran (two marks, the refused value, one slow) got
	// one update carrying its ref back.
	refs := map[string]int{}
	for _, f := range logs.frames() {
		if m := regexp.MustCompile(`"ref":"([^"]+)"`).FindStringSubmatch(f); m != nil {
			refs[m[1]]++
		}
	}
	if len(refs) != 4 {
		dump()
		t.Fatalf("updates carried refs %v; want one for each of the 4 clicks the server ran", refs)
	}
	for ref, n := range refs {
		if n != 1 {
			t.Fatalf("ref %s came back %d times", ref, n)
		}
	}
}

// TestParamsMerge_E2E opens a params merge from the page URL. A value not yet
// in the gate merge runs nothing until its row appears; each value gets its
// own buffer; a row link opens another value with a full page load; and the
// gate's ["*"] doesn't take the params merge's input.
func TestParamsMerge_E2E(t *testing.T) {
	serverLog := captureServerLog(t)
	orig := config.IsExecAllowed()
	config.SetAllowExec(true)
	defer config.SetAllowExec(orig)

	marker := fmt.Sprintf("pmark%d", os.Getpid())
	running := func() int {
		out, _ := exec.Command("pgrep", "-f", marker).Output()
		return len(strings.Fields(string(out)))
	}
	dir := t.TempDir()
	list := filepath.Join(dir, "list.jsonl")
	os.WriteFile(list, []byte(`{"name":"a"}`+"\n"+`{"name":"b"}`+"\n"), 0o644)
	page := "---\ntitle: Things\nsources:\n  things:\n    type: merge\n    inputs: [\"*\"]\n    key: [fields.name]\n  logview:\n    type: merge\n    inputs: [logs]\n    params:\n      name: {in: things.fields.name}\n---\n\n```lvt\n" +
		`<div lvt-source="things"><input data-testid="keep"><ol data-testid="list">{{range .Data}}<li data-key="{{.key}}"><a href="?name={{.fields.name}}">{{.fields.name}}</a> <i>{{.source}}</i></li>{{end}}</ol></div>` +
		"\n```\n\n```lvt\n" +
		`<div lvt-source="logview">{{with .Params.name}}<h3 data-testid="logname">{{.}}</h3>{{end}}{{if .Error}}<p data-testid="logerr">{{.Error}}</p>{{end}}<ol data-testid="logs">{{range .Data}}<li data-key="{{.seq}}">{{.msg}}</li>{{end}}</ol></div>` + "\n```\n"
	os.WriteFile(filepath.Join(dir, "index.md"), []byte(page), 0o644)

	cfg := config.DefaultConfig()
	cfg.FlagSources = map[string]config.SourceConfig{
		"list": {Type: "stream", File: list, Format: "json"},
		"logs": {Type: "stream", Cmd: ": " + marker + `; echo "line from $TD_name"; while true; do sleep 1; done`, Format: "lines"},
	}
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
		var html string
		chromedp.Run(ctx, chromedp.OuterHTML("body", &html, chromedp.ByQuery))
		t.Logf("console:\n%s", strings.Join(logs.console, "\n"))
		t.Logf("ws received: %d frames, last: %s", len(frames), strings.Join(frames[max(0, len(frames)-4):], "\n"))
		t.Logf("html: %s", tail(html, 3000))
		t.Logf("server log tail: %s", tail(serverLog.String(), 4000))
	}
	wait := func(what, js string) {
		t.Helper()
		if err := chromedp.Run(ctx, waitForDOM(js, 15*time.Second)); err != nil {
			dump()
			t.Fatalf("waiting for %s: %v", what, err)
		}
	}
	logText := `[...document.querySelectorAll('[data-testid="logs"] li')].map(li => li.textContent).join("|")`

	// ?name=c isn't listed yet: the page shows the list and no logs, and no
	// logs command runs.
	if err := chromedp.Run(ctx, network.Enable(), chromedp.Navigate(ConvertURLForDockerChrome(ts.URL)+"/?name=c")); err != nil {
		dump()
		t.Fatal(err)
	}
	wait("the list", `document.querySelectorAll('[data-testid="list"] li').length === 2`)
	var sources string
	chromedp.Run(ctx, chromedp.Evaluate(`[...document.querySelectorAll('[data-testid="list"] i')].map(i => i.textContent).join(",")`, &sources))
	if sources != "list,list" {
		dump()
		t.Fatalf("list rows came from %q; the [\"*\"] merge must not take the params merge's input", sources)
	}
	time.Sleep(time.Second)
	if n := running(); n != 0 {
		dump()
		t.Fatalf("%d logs commands run for a value not in the list", n)
	}

	// The row appears: the logs start.
	f, _ := os.OpenFile(list, os.O_APPEND|os.O_WRONLY, 0)
	fmt.Fprintln(f, `{"name":"c"}`)
	f.Close()
	wait("c's logs once c is listed", logText+` === "line from c"`)
	logName := `(document.querySelector('[data-testid="logname"]')?.textContent || "")`
	wait("the block naming c", logName+` === "c"`)

	// A row link opens another value, with its own buffer, without a page
	// load: the window marker and the text typed in the list block survive.
	if err := chromedp.Run(ctx,
		chromedp.Evaluate(`window.__sameDoc = true`, nil),
		chromedp.SendKeys(`[data-testid="keep"]`, "typed", chromedp.ByQuery),
		chromedp.Click(`[data-testid="list"] a[href="?name=a"]`, chromedp.ByQuery)); err != nil {
		dump()
		t.Fatal(err)
	}
	wait("a's logs, not c's", logText+` === "line from a"`)
	wait("the block naming a", logName+` === "a"`)
	var href string
	var sameDoc bool
	var typed string
	chromedp.Run(ctx,
		chromedp.Evaluate(`location.search`, &href),
		chromedp.Evaluate(`window.__sameDoc === true`, &sameDoc),
		chromedp.Value(`[data-testid="keep"]`, &typed, chromedp.ByQuery))
	if href != "?name=a" || !sameDoc || typed != "typed" {
		dump()
		t.Fatalf("after the link: location.search %q, same document %v, typed text %q; want ?name=a, true, typed", href, sameDoc, typed)
	}
	// Back restores c's logs, still without a page load.
	if err := chromedp.Run(ctx, chromedp.Evaluate(`history.back()`, nil)); err != nil {
		dump()
		t.Fatal(err)
	}
	wait("c's logs after Back", logText+` === "line from c" && location.search === "?name=c" && window.__sameDoc === true`)
	if err := chromedp.Run(ctx, chromedp.Navigate(ConvertURLForDockerChrome(ts.URL)+"/?name=b")); err != nil {
		dump()
		t.Fatal(err)
	}
	wait("b's logs", logText+` === "line from b"`)
	if n := running(); n < 2 {
		dump()
		t.Fatalf("%d logs commands run; each value opened should have its own", n)
	}

	// A refused value shows why, and runs nothing new.
	before := running()
	if err := chromedp.Run(ctx, chromedp.Navigate(ConvertURLForDockerChrome(ts.URL)+"/?name=-rf")); err != nil {
		dump()
		t.Fatal(err)
	}
	wait("the refusal", `document.body.textContent.includes("starts with -")`)
	wait("no name for a refused value", logName+` === ""`)
	if n := running(); n > before {
		t.Fatalf("a refused value started a command (%d -> %d)", before, n)
	}
}

// sawOp reports whether any WebSocket frame carries a range op such as "p".
func sawOp(frames []string, op string) bool {
	for _, f := range frames {
		if strings.Contains(f, `["`+op+`",`) {
			return true
		}
	}
	return false
}

// A keyed merge with sort shows rows in the field's order, and a new row that
// sorts first reaches the page as a prepend, not a whole-list replace.
func TestKeyedSort_E2E(t *testing.T) {
	serverLog := captureServerLog(t)
	dir := t.TempDir()
	feed := filepath.Join(dir, "runs.jsonl")
	os.WriteFile(feed, []byte(`{"id":"a","t":"2026-11-22T10:00:00Z"}`+"\n"+`{"id":"b","t":"2026-11-22T11:00:00Z"}`+"\n"), 0o644)
	page := "---\ntitle: Runs\nsources:\n  runs:\n    type: merge\n    inputs: [\"*\"]\n    key: [fields.id]\n    sort: -fields.t\n---\n\n```lvt\n" +
		`<ol lvt-source="runs" data-testid="runs">{{range .Data}}<li data-key="{{.key}}">{{.fields.id}}</li>{{end}}</ol>` + "\n```\n"
	os.WriteFile(filepath.Join(dir, "index.md"), []byte(page), 0o644)

	cfg := config.DefaultConfig()
	cfg.FlagSources = map[string]config.SourceConfig{"gh": {Type: "stream", File: feed, Format: "json"}}
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
	order := `[...document.querySelectorAll('[data-testid="runs"] li')].map(li => li.textContent).join(",")`
	wait := func(want string) {
		t.Helper()
		if err := chromedp.Run(ctx, waitForDOM(order+` === "`+want+`"`, 15*time.Second)); err != nil {
			dump()
			t.Fatalf("waiting for rows %s: %v", want, err)
		}
	}
	appendRun := func(line string) {
		f, _ := os.OpenFile(feed, os.O_APPEND|os.O_WRONLY, 0)
		fmt.Fprintln(f, line)
		f.Close()
	}
	if err := chromedp.Run(ctx, network.Enable(), chromedp.Navigate(ConvertURLForDockerChrome(ts.URL)),
		chromedp.WaitVisible(`[data-testid="runs"]`, chromedp.ByQuery)); err != nil {
		dump()
		t.Fatal(err)
	}
	wait("b,a")

	before := len(logs.frames())
	appendRun(`{"id":"c","t":"2026-11-22T12:00:00Z"}`)
	wait("c,b,a")
	if !sawOp(logs.frames()[before:], "p") {
		dump()
		t.Fatal(`a newest row didn't arrive as a ["p", ...] prepend`)
	}

	before = len(logs.frames())
	appendRun(`{"id":"m","t":"2026-11-22T10:30:00Z"}`)
	wait("c,b,m,a")
	if !sawOp(logs.frames()[before:], "i") {
		dump()
		t.Fatal(`a row between others didn't arrive as an ["i", ...] insert`)
	}

	// A row whose sort value changes moves; nothing else is re-rendered.
	appendRun(`{"id":"a","t":"2026-11-22T13:00:00Z"}`)
	wait("a,c,b,m")
}
