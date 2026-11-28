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

// TestActionValues_E2E clicks a row's button whose action checks its value
// against the rows the page lists: a listed value runs the --action command,
// and a value changed in the page's markup runs nothing and shows the error.
// It then commits through a form: the typed message reaches an on merge as
// $TD_message (A1), and a message the value rules refuse runs nothing.
func TestActionValues_E2E(t *testing.T) {
	serverLog := captureServerLog(t)
	orig := config.IsExecAllowed()
	config.SetAllowExec(true)
	defer config.SetAllowExec(orig)

	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "files.jsonl"), []byte(`{"path":"a b.txt"}`+"\n"+`{"path":"c.txt"}`+"\n"), 0o644)
	page := "---\ntitle: Values\nsources:\n" +
		"  files:\n    type: merge\n    inputs: [list]\n    key: [fields.path]\n" +
		"  committed:\n    type: merge\n    inputs: [commit]\n    on: commit\n    params:\n      message: {typed: true}\n" +
		"actions:\n  stage:\n    values:\n      path: {in: files.fields.path}\n" +
		"---\n\n```lvt\n" +
		`<div lvt-source="files">{{if .Error}}<p data-testid="error">{{.Error}}</p>{{end}}<ol data-testid="files">{{range .Data}}<li data-key="{{.key}}"><button name="stage" data-path="{{.fields.path}}">{{.fields.path}}</button></li>{{end}}</ol></div>` +
		"\n```\n\n```lvt\n" +
		`<div lvt-source="committed"><form><input name="message" maxlength="256"><button name="commit">Commit</button></form>{{if .Error}}<p data-testid="commit-error">{{.Error}}</p>{{end}}<ol data-testid="committed">{{range .Data}}<li data-key="{{.seq}}">{{.msg}}</li>{{end}}</ol></div>` +
		"\n```\n"
	os.WriteFile(filepath.Join(dir, "index.md"), []byte(page), 0o644)

	cfg := config.DefaultConfig()
	cfg.FlagSources = map[string]config.SourceConfig{
		"list":   {Type: "stream", File: filepath.Join(dir, "files.jsonl"), Format: "json"},
		"commit": {Type: "stream", Cmd: `printf 'committed %s\n' "$TD_message"`, Format: "lines"},
	}
	cfg.FlagActions = map[string]*config.Action{
		"stage": {Kind: "exec", Cmd: `printf '%s\n' "$TD_path" >> staged.txt`, FromFlag: true},
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
	staged := func() string {
		b, _ := os.ReadFile(filepath.Join(dir, "staged.txt"))
		return string(b)
	}

	if err := chromedp.Run(ctx, network.Enable(), chromedp.Navigate(ConvertURLForDockerChrome(ts.URL)),
		chromedp.WaitVisible(`[data-testid="files"]`, chromedp.ByQuery)); err != nil {
		dump()
		t.Fatal(err)
	}
	wait("both rows", `document.querySelectorAll('[data-testid="files"] li').length === 2`)

	// A listed value runs the command.
	if err := chromedp.Run(ctx, chromedp.Click(`button[data-path="a b.txt"]`, chromedp.ByQuery)); err != nil {
		dump()
		t.Fatal(err)
	}
	deadline := time.Now().Add(10 * time.Second)
	for staged() != "a b.txt\n" && time.Now().Before(deadline) {
		time.Sleep(50 * time.Millisecond)
	}
	if got := staged(); got != "a b.txt\n" {
		dump()
		t.Fatalf("staged = %q, want the clicked row's path", got)
	}

	// A value changed in the markup, as a hand-made message would send it,
	// runs nothing. The block's next push clears its error, so the refusal
	// is read from the server log (tinkerdown#359).
	const refused = "Error handling action: action failed: value path is not listed"
	for i, bad := range []string{"../outside", ":/"} {
		// A click that lands while the last reply is applied is dropped, so
		// click again until the server has refused it.
		deadline := time.Now().Add(10 * time.Second)
		for strings.Count(serverLog.String(), refused) < i+1 && time.Now().Before(deadline) {
			if err := chromedp.Run(ctx,
				chromedp.SetAttributeValue(`button[name="stage"]`, "data-path", bad, chromedp.ByQuery),
				chromedp.Click(`button[name="stage"]`, chromedp.ByQuery)); err != nil {
				dump()
				t.Fatal(err)
			}
			time.Sleep(500 * time.Millisecond)
		}
		if n := strings.Count(serverLog.String(), refused); n < i+1 {
			dump()
			t.Fatalf("no refusal of %q in the server log (%d so far)", bad, n)
		}
	}
	time.Sleep(300 * time.Millisecond)
	if got := staged(); got != "a b.txt\n" {
		dump()
		t.Fatalf("staged = %q after values the page never listed", got)
	}

	// A typed message reaches the on merge through the form (A1).
	if err := chromedp.Run(ctx,
		chromedp.SendKeys(`input[name="message"]`, `fix "the" typo`, chromedp.ByQuery),
		chromedp.Click(`button[name="commit"]`, chromedp.ByQuery)); err != nil {
		dump()
		t.Fatal(err)
	}
	wait("the commit's output", `[...document.querySelectorAll('[data-testid="committed"] li')].map(li => li.textContent).join(",") === 'committed fix "the" typo'`)

	// A message the value rules refuse runs nothing and says why.
	if err := chromedp.Run(ctx,
		chromedp.SetValue(`input[name="message"]`, "--amend", chromedp.ByQuery),
		chromedp.Click(`button[name="commit"]`, chromedp.ByQuery)); err != nil {
		dump()
		t.Fatal(err)
	}
	wait("the refused message", `(document.querySelector('[data-testid="commit-error"]')?.textContent || "").includes("starts with -")`)
	var rows string
	chromedp.Run(ctx, chromedp.Evaluate(`[...document.querySelectorAll('[data-testid="committed"] li')].map(li => li.textContent).join(",")`, &rows))
	if strings.Contains(rows, "amend") {
		dump()
		t.Fatalf("a refused message ran: %s", rows)
	}
	logs.checkConsole(t)
}
