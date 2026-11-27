//go:build !ci

package tinkerdown_test

import (
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

// TestMergeEvery_E2E serves a page whose merges rerun a command that prints
// a list. A row the command stops printing leaves the page as a remove op, a
// changed row updates in place, an unchanged run sends nothing, a failed run
// keeps the rows and shows the error, and a click's command reruns a list
// whose timer is an hour away.
func TestMergeEvery_E2E(t *testing.T) {
	serverLog := captureServerLog(t)
	orig := config.IsExecAllowed()
	config.SetAllowExec(true)
	defer config.SetAllowExec(orig)

	dir := t.TempDir()
	write := func(name, s string) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(dir, name), []byte(s), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("list.txt", "M a.txt\n?? b.txt\n")
	write("later.txt", "")
	pattern := `'^(?P<state>\S+) (?P<name>.+)$'`
	page := "---\ntitle: Lists\nsources:\n" +
		"  files:\n    type: merge\n    inputs: [status]\n    every: 200ms\n    key: [fields.name]\n    pattern: " + pattern + "\n" +
		"  later:\n    type: merge\n    inputs: [queue]\n    every: 1h\n    key: [fields.name]\n    pattern: " + pattern + "\n" +
		"---\n\n```lvt\n" +
		`<div lvt-source="files">{{if .Error}}<p data-testid="error">{{.Error}}</p>{{end}}<ol data-testid="files">{{range .Data}}<li data-key="{{.key}}" data-state="{{.fields.state}}">{{.fields.name}}</li>{{end}}</ol></div>` +
		"\n```\n\n```lvt\n" +
		`<div lvt-source="later"><button name="enqueue">Enqueue</button><ol data-testid="later">{{range .Data}}<li data-key="{{.key}}">{{.fields.name}}</li>{{end}}</ol></div>` +
		"\n```\n"
	write("index.md", page)

	cfg := config.DefaultConfig()
	cfg.FlagSources = map[string]config.SourceConfig{
		"status": {Type: "stream", Cmd: "cat list.txt", Format: "lines"},
		"queue":  {Type: "stream", Cmd: "cat later.txt", Format: "lines"},
	}
	cfg.FlagActions = map[string]*config.Action{
		"enqueue": {Kind: "exec", Cmd: `printf 'A new.txt\n' >> later.txt`, FromFlag: true},
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
	dump := func() { logs.dump(t, ctx, serverLog) }
	wait := func(what, js string) {
		t.Helper()
		if err := chromedp.Run(ctx, waitForDOM(js, 15*time.Second)); err != nil {
			dump()
			t.Fatalf("waiting for %s: %v", what, err)
		}
	}
	files := `[...document.querySelectorAll('[data-testid="files"] li')].map(li => li.dataset.state + " " + li.textContent).join(",")`

	if err := chromedp.Run(ctx, network.Enable(), chromedp.Navigate(ConvertURLForDockerChrome(ts.URL)),
		chromedp.WaitVisible(`[data-testid="files"]`, chromedp.ByQuery)); err != nil {
		dump()
		t.Fatal(err)
	}
	wait("the first run's rows", files+` === "M a.txt,?? b.txt"`)

	// A row the command stops printing leaves as a remove, not a new list.
	before := len(logs.frames())
	write("list.txt", "M a.txt\n")
	wait("b.txt to leave", files+` === "M a.txt"`)
	sent := strings.Join(logs.frames()[before:], "\n")
	if !sawOp(logs.frames()[before:], "r") || strings.Contains(sent, "a.txt") {
		dump()
		t.Fatalf("want a remove op that doesn't resend a.txt; frames:\n%s", sent)
	}

	// A changed row updates in place.
	before = len(logs.frames())
	write("list.txt", "A a.txt\n")
	wait("a.txt staged", files+` === "A a.txt"`)
	if !sawOp(logs.frames()[before:], "u") {
		dump()
		t.Fatal(`no ["u", ...] op: the changed row was not updated in place`)
	}

	// Runs that print the same lines send nothing.
	before = len(logs.frames())
	time.Sleep(time.Second) // about five runs
	if n := len(logs.frames()) - before; n != 0 {
		dump()
		t.Fatalf("%d frames while the list didn't change, want 0", n)
	}

	// A failed run keeps the rows and shows why; the next good run clears it.
	os.Remove(filepath.Join(dir, "list.txt"))
	wait("the error", `(document.querySelector('[data-testid="error"]')?.textContent || "").includes("No such file")`)
	wait("the rows kept", files+` === "A a.txt"`)
	write("list.txt", "A a.txt\n")
	wait("the error to clear", `!document.querySelector('[data-testid="error"]')`)

	// A click's command reruns a list whose next timed run is an hour away.
	if err := chromedp.Run(ctx, chromedp.Click(`button[name="enqueue"]`, chromedp.ByQuery)); err != nil {
		dump()
		t.Fatal(err)
	}
	wait("the click's row", `[...document.querySelectorAll('[data-testid="later"] li')].map(li => li.textContent).join(",") === "new.txt"`)
	logs.checkConsole(t)
}
