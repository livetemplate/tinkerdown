//go:build !ci

package tinkerdown_test

import (
	"context"
	"fmt"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/chromedp/chromedp"
	"github.com/livetemplate/tinkerdown/internal/config"
	"github.com/livetemplate/tinkerdown/internal/server"
)

// sqlSourcePage lists the queries of two sql: sources with a Run button, and
// the requests of two .http files with a Send button, each beside the pane a
// click fills. The first row of each source has the same place, 1, so a
// click runs the right one only when entries name their source.
const sqlSourcePage = "---\ntitle: SQL source\n---\n\n# SQL source\n\n" +
	"```lvt\n" + `<main lvt-source="queries">
  {{if .Error}}<p data-testid="q-error">{{.Error}}</p>{{end}}
  <ol data-testid="queries">{{range .Data}}<li data-key="{{.key}}" class="kind-{{.fields.kind}}" data-testid="q-{{.fields.entry}}"><b>{{.fields.name}}</b> <i>{{.fields.kind}}</i> <span>{{.fields.file}}:{{.fields.line}}</span> <code>{{.line}}</code>
    <button name="execute" data-entry="{{.fields.entry}}" data-testid="run-{{.fields.entry}}">Run</button></li>{{end}}</ol>
</main>` + "\n```\n\n```lvt\n" + `<aside lvt-source="result">
  <p data-testid="result-state">{{if .Running}}running{{else}}idle{{end}}</p>
  {{if .Error}}<p data-testid="result-error">{{.Error}}</p>{{end}}
  <pre data-testid="result">{{range .Data}}<span data-key="{{.seq}}">{{.line}}
</span>{{end}}</pre>
</aside>` + "\n```\n\n```lvt\n" + `<main lvt-source="requests">
  <ol>{{range .Data}}<li data-key="{{.key}}"><button name="send" data-entry="{{.fields.entry}}" data-testid="send-{{.fields.entry}}">Send</button></li>{{end}}</ol>
</main>` + "\n```\n\n```lvt\n" + `<aside lvt-source="response">
  <p data-testid="response-state">{{if .Running}}running{{else}}idle{{end}}</p>
  <pre data-testid="response">{{range .Data}}<span data-key="{{.seq}}">{{.line}}
</span>{{end}}</pre>
</aside>` + "\n```\n"

func TestSQLSource_E2E(t *testing.T) {
	serverLog := captureServerLog(t)
	config.SetAllowExec(true)
	defer config.SetAllowExec(false)

	dir := t.TempDir()
	runs, sends := filepath.Join(dir, "runs"), filepath.Join(dir, "sends")
	write := func(name, text string) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(dir, name), []byte(text), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	// The commands stand in for sqlite3 and curl: they record each run and
	// print what they got.
	// Each run ends by printing its number, so a test waits for its own run.
	run := `echo "$TD_entry" >> ` + runs + `; printf 'SQL<%s>\n' "$TD_sql"; echo "done $(($(wc -l < ` + runs + `)))"`
	send := `echo x >> ` + sends + `; printf 'URL<%s>\n' "$TD_url"; echo "done $(($(wc -l < ` + sends + `)))"`
	write("tinkerdown.yaml", fmt.Sprintf(`sources:
  users: {type: stream, sql: users.sql}
  reports: {type: stream, sql: reports}
  queries: {type: merge, inputs: [users, reports], key: [fields.entry]}
  run: {type: stream, cmd: %q, format: lines}
  result: {type: merge, inputs: [run], on: execute, flush: 50ms, params: {entry: {in: queries.fields.entry}}}
  a: {type: stream, http: a.http}
  b: {type: stream, http: b.http}
  requests: {type: merge, inputs: [a, b], key: [fields.entry]}
  send: {type: stream, cmd: %q, format: lines}
  response: {type: merge, inputs: [send], on: send, flush: 50ms, params: {entry: {in: requests.fields.entry}}}
`, run, send))
	write("index.md", sqlSourcePage)
	write("users.sql", "-- Everyone\nSELECT * FROM users LIMIT 100;\n")
	os.Mkdir(filepath.Join(dir, "reports"), 0o755)
	write("reports/a.sql", "-- name: daily :many\nSELECT day, count(*) FROM orders GROUP BY day;\n\n-- name: close-stale :exec\nUPDATE orders SET status = 'closed';\n")
	write("a.http", "### one\nGET https://a.test/one\n")
	write("b.http", "### one\nGET https://b.test/one\n")
	count := func(file string) int {
		data, _ := os.ReadFile(file)
		return strings.Count(string(data), "\n")
	}
	runCount := func() int { return count(runs) }

	cfg, err := config.LoadFromDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	srv := server.NewWithConfig(dir, cfg)
	if err := srv.Discover(); err != nil {
		t.Fatal(err)
	}
	ts := httptest.NewServer(server.WithCompression(srv))
	defer ts.Close()
	defer srv.StopStreams()

	chrome, cleanup := SetupDockerChrome(t, 180*time.Second)
	defer cleanup()
	ctx := chrome.Context
	logs := &streamBrowserLogs{}
	logs.listen(ctx)

	dump := func() { logs.dump(t, ctx, serverLog) }
	do := func(actions ...chromedp.Action) {
		t.Helper()
		actx, cancel := context.WithTimeout(ctx, 10*time.Second)
		defer cancel()
		if err := chromedp.Run(actx, actions...); err != nil {
			dump()
			t.Fatal(err)
		}
	}
	waitJS := func(what, js string) {
		t.Helper()
		if err := chromedp.Run(ctx, waitForDOM(js, 15*time.Second)); err != nil {
			dump()
			t.Fatalf("waiting for %s: %v", what, err)
		}
	}
	text := func(testid string) string {
		var s string
		do(chromedp.Evaluate(fmt.Sprintf(`(document.querySelector('[data-testid=%q]')||{}).innerText||""`, testid), &s))
		return s
	}
	// click runs a button and waits for its pane to show that run's end.
	click := func(button, pane string) string {
		t.Helper()
		file := map[string]string{"result": runs, "response": sends}[pane]
		done := fmt.Sprintf("done %d", count(file)+1)
		do(chromedp.Click(fmt.Sprintf(`[data-testid=%q]`, button), chromedp.ByQuery))
		waitJS(pane+" output", fmt.Sprintf(`document.querySelector('[data-testid=%q]').innerText.includes(%q)`, pane, done))
		waitJS(pane+" idle", fmt.Sprintf(`document.querySelector('[data-testid="%s-state"]').innerText === "idle"`, pane))
		return text(pane)
	}

	if err := chromedp.Run(ctx, chromedp.Navigate(ConvertURLForDockerChrome(ts.URL)),
		chromedp.WaitVisible(`[data-testid="run-reports/2"]`, chromedp.ByQuery)); err != nil {
		dump()
		t.Fatal(err)
	}
	if !browserSettles(ctx) {
		dump()
		t.Fatal("the browser never answered after the first page load")
	}

	t.Run("each query is a row with its fields", func(t *testing.T) {
		for id, want := range map[string]string{
			"q-users/1":   "users select users.sql:1 SELECT * FROM users LIMIT 100;",
			"q-reports/1": "daily select a.sql:1 SELECT day, count(*) FROM orders GROUP BY day;",
			"q-reports/2": "close-stale update a.sql:4 UPDATE orders SET status = 'closed';",
		} {
			if got := strings.Join(strings.Fields(text(id)), " "); got != want+" Run" {
				dump()
				t.Fatalf("%s = %q, want %q", id, got, want+" Run")
			}
		}
		var class string
		do(chromedp.AttributeValue(`[data-testid="q-reports/2"]`, "class", &class, nil, chromedp.ByQuery))
		if class != "kind-update" {
			t.Fatalf("class = %q, want kind-update", class)
		}
	})

	t.Run("Run hands each source's query to the command", func(t *testing.T) {
		if got := click("run-reports/1", "result"); !strings.Contains(got, "SQL<SELECT day, count(*) FROM orders GROUP BY day;>") || strings.Contains(got, "users") {
			dump()
			t.Fatalf("Run on reports/1 printed %q", got)
		}
		got := click("run-users/1", "result")
		if !strings.Contains(got, "SQL<-- Everyone\nSELECT * FROM users LIMIT 100;>") || strings.Contains(got, "orders") {
			dump()
			t.Fatalf("Run on users/1 printed %q", got)
		}
		if n := runCount(); n != 2 {
			t.Fatalf("runs = %d, want 2", n)
		}
	})

	t.Run("Send sends the clicked file's request", func(t *testing.T) {
		if got := click("send-b/1", "response"); !strings.Contains(got, "URL<https://b.test/one>") {
			dump()
			t.Fatalf("Send on b/1 printed %q", got)
		}
		if got := click("send-a/1", "response"); !strings.Contains(got, "URL<https://a.test/one>") || strings.Contains(got, "b.test") {
			dump()
			t.Fatalf("Send on a/1 printed %q", got)
		}
	})

	t.Run("editing the files updates the rows", func(t *testing.T) {
		write("reports/b.sql", "SELECT 2;\n")
		waitJS("the new file's query", `(document.querySelector('[data-testid="q-reports/3"]')||{}).innerText?.includes("b.sql:1")`)
		write("users.sql", "-- name: active\nSELECT * FROM users WHERE active;\n")
		waitJS("the edited query", `document.querySelector('[data-testid="q-users/1"]').innerText.includes("active")`)
		os.Remove(filepath.Join(dir, "reports", "b.sql"))
		waitJS("the removed file's query to leave", `!document.querySelector('[data-testid="q-reports/3"]')`)
		if got := click("run-users/1", "result"); !strings.Contains(got, "SQL<SELECT * FROM users WHERE active;>") {
			dump()
			t.Fatalf("Run after the edit printed %q", got)
		}
	})

	t.Run("a hand-made click on a query not listed runs nothing", func(t *testing.T) {
		before := runCount()
		do(chromedp.Evaluate(`document.querySelector('[data-testid="run-reports/1"]').dataset.entry = "reports/3"`, nil),
			chromedp.Click(`[data-testid="run-reports/1"]`, chromedp.ByQuery))
		waitJS("the refusal", `!!document.querySelector('[data-testid="result-error"]')`)
		time.Sleep(500 * time.Millisecond)
		if n := runCount(); n != before {
			dump()
			t.Fatalf("runs = %d after a refused click, want %d", n, before)
		}
		if !strings.Contains(serverLog.String(), "value entry is not listed") {
			t.Fatalf("server log lacks the refusal:\n%s", tail(serverLog.String(), 2000))
		}
	})

	logs.checkConsole(t)
}
