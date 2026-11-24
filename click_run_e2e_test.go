//go:build !ci

package tinkerdown_test

import (
	"context"
	"fmt"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/chromedp"
	"github.com/livetemplate/tinkerdown/internal/config"
	"github.com/livetemplate/tinkerdown/internal/server"
)

// clickRunPage lists the requests of a .http file, each with a Send button,
// and shows the click-started response merge beside them.
const clickRunPage = "---\ntitle: Click run\n---\n\n# Click run\n\n" +
	"```lvt\n" + `<main lvt-source="requests">
  {{if .Error}}<p data-testid="req-error">{{.Error}}</p>{{end}}
  <ol data-testid="reqs">
    {{range .Data}}<li data-key="{{.key}}">{{.fields.name}} <span data-testid="url-{{.fields.entry}}">{{.fields.url}}</span> <code>{{.fields.headers}}</code>
      <button name="send" data-entry="{{.fields.entry}}" data-testid="send-{{.fields.entry}}">Send</button></li>{{end}}
  </ol>
</main>` + "\n```\n\n```lvt\n" + `<aside lvt-source="response">
  <p data-testid="state">{{if .Running}}running{{else}}idle{{end}}</p>
  {{if .Error}}<p data-testid="resp-error">{{.Error}}</p>{{end}}
  <ol data-testid="resp">{{range .Data}}<li data-key="{{.seq}}" class="s-{{.stream}}">{{.line}}</li>{{end}}</ol>
</aside>` + "\n```\n"

const clickRunHTTP = `@host = https://api.test

### Create user
POST {{host}}/users
Authorization: Bearer {{$processEnv TD_E2E_TOKEN}}
Content-Type: application/json

{
  "name": "Ada"
}

### Slow
GET {{host}}/slow
`

func TestClickRun_E2E(t *testing.T) {
	serverLog := captureServerLog(t)
	config.SetAllowExec(true)
	defer config.SetAllowExec(false)
	const token = "s3cret-e2e-token"
	t.Setenv("TD_E2E_TOKEN", token)

	dir := t.TempDir()
	runs := filepath.Join(dir, "runs")
	marker := fmt.Sprintf("clickmark%d", os.Getpid())
	// The command stands in for curl: it records each run, prints what it
	// got, and takes a second on the slow request so a click can land mid-run.
	send := `: ` + marker + `; echo "$TD_url" >> ` + runs + `; echo "$TD_method $TD_url"; printf '%s\n' "$TD_body";` +
		` case "$TD_url" in */slow) sleep 2;; esac; echo "auth=${TD_headers#*Bearer }" | head -c 12; echo; echo done`
	yaml := fmt.Sprintf(`sources:
  api: {type: stream, http: api.http}
  requests: {type: merge, inputs: [api], key: [fields.entry]}
  send: {type: stream, cmd: %q, format: lines}
  response: {type: merge, inputs: [send], on: send, flush: 50ms, params: {entry: {in: requests.fields.entry}}}
`, send)
	os.WriteFile(filepath.Join(dir, "tinkerdown.yaml"), []byte(yaml), 0o644)
	os.WriteFile(filepath.Join(dir, "index.md"), []byte(clickRunPage), 0o644)
	os.WriteFile(filepath.Join(dir, "api.http"), []byte(clickRunHTTP), 0o644)
	runCount := func() int {
		data, _ := os.ReadFile(runs)
		return strings.Count(string(data), "\n")
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
	defer ts.Close()
	defer srv.StopStreams()

	chrome, cleanup := SetupDockerChrome(t, 180*time.Second)
	defer cleanup()
	ctx := chrome.Context
	logs := &streamBrowserLogs{}
	logs.listen(ctx)
	url := ConvertURLForDockerChrome(ts.URL)

	dump := func() {
		var html string
		chromedp.Run(ctx, chromedp.OuterHTML("body", &html, chromedp.ByQuery))
		logs.mu.Lock()
		defer logs.mu.Unlock()
		t.Logf("console:\n%s", strings.Join(logs.console, "\n"))
		t.Logf("ws sent:\n%s", strings.Join(logs.wsSent, "\n"))
		t.Logf("ws received: %d frames, last: %s", len(logs.wsRecv), strings.Join(logs.wsRecv[max(0, len(logs.wsRecv)-5):], "\n"))
		t.Logf("rendered html: %.6000s", html)
		t.Logf("server log tail: %s", tail(serverLog.String(), 4000))
	}
	run := func(actions ...chromedp.Action) error {
		actx, cancel := context.WithTimeout(ctx, 10*time.Second)
		defer cancel()
		return chromedp.Run(actx, actions...)
	}
	waitJS := func(what, js string) {
		t.Helper()
		if err := chromedp.Run(ctx, waitForDOM(js, 15*time.Second)); err != nil {
			dump()
			t.Fatalf("waiting for %s: %v", what, err)
		}
	}
	respText := func() string {
		var s string
		run(chromedp.Evaluate(`document.querySelector('[data-testid="resp"]').innerText`, &s))
		return s
	}

	// The URL holds a valid entry, but opening the page must send nothing.
	if err := chromedp.Run(ctx, network.Enable(), chromedp.Navigate(url+"/?entry=1"),
		chromedp.WaitVisible(`[data-testid="send-2"]`, chromedp.ByQuery)); err != nil {
		dump()
		t.Fatal(err)
	}

	// A fresh Chrome profile freezes the page for a few seconds about a
	// second after its first load: its timers stop and DevTools gets no
	// answers, 2 s without container limits and up to 10 s with them. Wait
	// that out once, so the 10 s per-action deadlines below measure the app.
	// Four answers in a row, 2 s in all, so a probe before the freeze
	// starts doesn't count.
	answers := 0
	for deadline := time.Now().Add(60 * time.Second); answers < 4 && time.Now().Before(deadline); {
		time.Sleep(500 * time.Millisecond)
		pctx, cancel := context.WithTimeout(ctx, time.Second)
		if chromedp.Run(pctx, chromedp.Evaluate(`1`, nil)) == nil {
			answers++
		} else {
			answers = 0
		}
		cancel()
	}
	if answers < 4 {
		dump()
		t.Fatal("the browser never answered after the first page load")
	}

	t.Run("the URL and a reload start nothing", func(t *testing.T) {
		time.Sleep(time.Second)
		if err := run(chromedp.Reload(), chromedp.WaitVisible(`[data-testid="send-2"]`, chromedp.ByQuery)); err != nil {
			dump()
			t.Fatal(err)
		}
		time.Sleep(time.Second)
		if n := runCount(); n != 0 {
			dump()
			t.Fatalf("runs = %d before any click, want 0", n)
		}
	})

	t.Run("a click runs the request once, with its filled-in values", func(t *testing.T) {
		if err := run(chromedp.Click(`[data-testid="send-1"]`, chromedp.ByQuery)); err != nil {
			t.Fatal(err)
		}
		waitJS("the response", `document.querySelector('[data-testid="resp"]').innerText.includes("done")`)
		waitJS("idle", `document.querySelector('[data-testid="state"]').innerText === "idle"`)
		got := respText()
		for _, want := range []string{"POST https://api.test/users", `"name": "Ada"`, "auth=s3cret-"} {
			if !strings.Contains(got, want) {
				dump()
				t.Fatalf("response %q lacks %q", got, want)
			}
		}
		if n := runCount(); n != 1 {
			t.Fatalf("runs = %d, want 1", n)
		}
		// The list shows the header as written, never the token's value.
		var html string
		run(chromedp.OuterHTML(`[data-testid="reqs"]`, &html, chromedp.ByQuery))
		if !strings.Contains(html, "{{$processEnv TD_E2E_TOKEN}}") {
			dump()
			t.Fatalf("the request list doesn't show the header as written: %s", html)
		}
	})

	t.Run("the token never reaches the browser", func(t *testing.T) {
		// The command echoes the token's first 7 bytes on purpose
		// (auth=s3cret-), so look for the whole token: only the private
		// values hold it.
		for _, f := range logs.frames() {
			if strings.Contains(f, token) {
				dump()
				t.Fatalf("a WebSocket frame holds the token: %.300s", f)
			}
		}
		var html string
		run(chromedp.OuterHTML("html", &html, chromedp.ByQuery))
		if strings.Contains(html, token) {
			t.Fatal("the page's HTML holds the token")
		}
	})

	t.Run("a click mid-run starts it again and kills the first", func(t *testing.T) {
		before := runCount()
		if err := run(chromedp.Click(`[data-testid="send-2"]`, chromedp.ByQuery)); err != nil {
			t.Fatal(err)
		}
		waitJS("running", `document.querySelector('[data-testid="state"]').innerText === "running"`)
		waitJS("the first run's line", `document.querySelector('[data-testid="resp"]').innerText.includes("GET https://api.test/slow")`)
		if err := run(chromedp.Click(`[data-testid="send-2"]`, chromedp.ByQuery)); err != nil {
			t.Fatal(err)
		}
		waitJS("the second run to finish", `document.querySelector('[data-testid="resp"]').innerText.includes("done")`)
		waitJS("idle", `document.querySelector('[data-testid="state"]').innerText === "idle"`)
		if got := respText(); strings.Count(got, "GET https://api.test/slow") != 1 || strings.Contains(got, "POST") {
			dump()
			t.Fatalf("response shows more than the last run: %q", got)
		}
		if n := runCount(); n != before+2 {
			t.Fatalf("runs = %d, want %d", n, before+2)
		}
		if exec.Command("pgrep", "-f", marker).Run() == nil {
			t.Fatal("a run's process is still running")
		}
	})

	t.Run("a hand-made click on an unlisted value runs nothing", func(t *testing.T) {
		before := runCount()
		if err := run(chromedp.Evaluate(`document.querySelector('[data-testid="send-1"]').dataset.entry = "99"`, nil),
			chromedp.Click(`[data-testid="send-1"]`, chromedp.ByQuery)); err != nil {
			t.Fatal(err)
		}
		waitJS("the refusal", `!!document.querySelector('[data-testid="resp-error"]')`)
		time.Sleep(500 * time.Millisecond)
		if n := runCount(); n != before {
			dump()
			t.Fatalf("runs = %d after a refused click, want %d", n, before)
		}
		if !strings.Contains(serverLog.String(), "value entry is not listed") {
			t.Fatalf("server log lacks the refusal:\n%s", tail(serverLog.String(), 2000))
		}
	})

	t.Run("editing the file updates the list", func(t *testing.T) {
		os.WriteFile(filepath.Join(dir, "api.http"), []byte(clickRunHTTP+"\n### Third\nDELETE {{host}}/users/{{id}}\n"), 0o644)
		waitJS("a third request", `!!document.querySelector('[data-testid="send-3"]')`)
		// A URL that starts with {{ and ends with }} shows as written; the
		// client once rendered any such value as empty.
		var u string
		run(chromedp.Evaluate(`document.querySelector('[data-testid="url-3"]').textContent`, &u))
		if u != "{{host}}/users/{{id}}" {
			dump()
			t.Fatalf("the third request's URL shows %q, want {{host}}/users/{{id}}", u)
		}
		os.WriteFile(filepath.Join(dir, "api.http"), []byte(clickRunHTTP+"\n### Broken\nGET https://x\nnot a header\n"), 0o644)
		waitJS("the parse error", `(document.querySelector('[data-testid="req-error"]')||{}).innerText?.includes("line 17:")`)
		if err := run(chromedp.WaitVisible(`[data-testid="send-3"]`, chromedp.ByQuery)); err != nil {
			dump()
			t.Fatal("a parse error dropped the last good rows")
		}
		os.WriteFile(filepath.Join(dir, "api.http"), []byte("### Only\nGET https://api.test/only\n"), 0o644)
		waitJS("one request, no error", `!document.querySelector('[data-testid="send-2"]') && !document.querySelector('[data-testid="req-error"]')`)
	})

	logs.mu.Lock()
	defer logs.mu.Unlock()
	for _, c := range logs.console {
		if strings.Contains(c, "[error]") || strings.Contains(c, "[exception]") {
			t.Errorf("browser console: %s", c)
		}
	}
}
