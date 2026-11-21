//go:build !ci

package tinkerdown_test

import (
	"context"
	"fmt"
	"net/http/httptest"
	neturl "net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/cdproto/runtime"
	"github.com/chromedp/chromedp"
	"github.com/gorilla/websocket"
	"github.com/livetemplate/tinkerdown/internal/config"
	"github.com/livetemplate/tinkerdown/internal/server"
)

// streamPage is one lvt block over the merged "all" source, with the toolbar
// the log viewer uses: a level select, a search box, Pause/Resume and Clear.
const streamPage = "---\ntitle: Stream\n---\n\n# Stream\n\n" +
	"## [All] | [gen] source = gen | [app] source = app | [box] source = box\n\n" +
	"```lvt\n" + `<main lvt-source="all">
  <form name="Filter">
    <select name="f.level" data-testid="level" lvt-on:change="Filter">
      <option value="">any</option><option>error</option><option>warn</option><option>info</option>
    </select>
    <input type="search" name="f.line" data-testid="search" lvt-on:input="Filter" lvt-mod:debounce="100">
  </form>
  {{if .Paused}}<button name="Resume" data-testid="resume">Resume</button>{{else}}<button name="Pause" data-testid="pause">Pause</button>{{end}}
  <button name="Clear" data-testid="clear">Clear</button>
  {{if .Error}}<p data-testid="error">{{.Error}}</p>{{end}}
  <ol data-testid="log" lvt-fx:scroll="bottom-sticky">
    {{range .Data}}<li data-key="{{.seq}}" class="src-{{.source}} stream-{{.stream}} lvl-{{.level}}">{{.msg}}</li>{{end}}
  </ol>
</main>` + "\n```\n"

type streamBrowserLogs struct {
	mu      sync.Mutex
	console []string
	wsRecv  []string
	wsSent  []string
	recvAt  []time.Time
}

func (l *streamBrowserLogs) listen(ctx context.Context) {
	chromedp.ListenTarget(ctx, func(ev interface{}) {
		l.mu.Lock()
		defer l.mu.Unlock()
		switch e := ev.(type) {
		case *runtime.EventConsoleAPICalled:
			for _, arg := range e.Args {
				l.console = append(l.console, fmt.Sprintf("[%s] %s", e.Type, arg.Value))
			}
		case *runtime.EventExceptionThrown:
			l.console = append(l.console, "[exception] "+e.ExceptionDetails.Text)
		case *network.EventWebSocketFrameReceived:
			l.wsRecv = append(l.wsRecv, e.Response.PayloadData)
			l.recvAt = append(l.recvAt, time.Now())
		case *network.EventWebSocketFrameSent:
			l.wsSent = append(l.wsSent, e.Response.PayloadData)
		}
	})
}

func (l *streamBrowserLogs) frames() []string {
	l.mu.Lock()
	defer l.mu.Unlock()
	return append([]string(nil), l.wsRecv...)
}

func TestStreamSource_E2E(t *testing.T) {
	serverLog := captureServerLog(t)
	config.SetAllowExec(true)
	defer config.SetAllowExec(false)

	id := os.Getpid()
	container := fmt.Sprintf("tdstream%d-box", id)
	if out, err := exec.Command("docker", "run", "-d", "--rm", "--name", container, "alpine:3.21",
		"sh", "-c", `i=0; while true; do echo "box line $i"; i=$((i+1)); sleep 0.3; done`).CombinedOutput(); err != nil {
		t.Fatalf("start container: %v\n%s", err, out)
	}
	defer exec.Command("docker", "rm", "-f", container).Run()

	dir := t.TempDir()
	appLog := filepath.Join(dir, "app.log")
	os.WriteFile(appLog, []byte("app started\n"), 0o644)
	genMarker := fmt.Sprintf("genmark%d", id)
	gen := `: ` + genMarker + `; i=0; while true; i=$((i+1)); do echo "{\"level\":\"info\",\"msg\":\"tick $i\"}"; echo "warn spill $i" 1>&2; sleep 0.2; done`
	yaml := fmt.Sprintf(`sources:
  gen: {type: stream, cmd: %q}
  app: {type: stream, file: app.log}
  box: {type: stream, cmd: "docker logs -f --timestamps %s"}
  all: {type: merge, inputs: [gen, app, box], flush: 50ms}
`, gen, container)
	os.WriteFile(filepath.Join(dir, "tinkerdown.yaml"), []byte(yaml), 0o644)
	os.WriteFile(filepath.Join(dir, "index.md"), []byte(streamPage), 0o644)

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
		chromedp.Run(ctx, chromedp.OuterHTML("main", &html, chromedp.ByQuery))
		logs.mu.Lock()
		defer logs.mu.Unlock()
		t.Logf("console:\n%s", strings.Join(logs.console, "\n"))
		t.Logf("ws sent:\n%s", strings.Join(logs.wsSent, "\n"))
		t.Logf("ws received: %d frames, last: %s", len(logs.wsRecv), strings.Join(logs.wsRecv[max(0, len(logs.wsRecv)-5):], "\n"))
		t.Logf("rendered html: %s", html)
		t.Logf("server log tail: %s", tail(serverLog.String(), 3000))
	}
	// run bounds each browser action, so a missing element fails fast with a
	// dump instead of hanging until the Chrome deadline.
	run := func(actions ...chromedp.Action) error {
		actx, cancel := context.WithTimeout(ctx, 10*time.Second)
		defer cancel()
		return chromedp.Run(actx, actions...)
	}
	count := func(sel string) int {
		var n int
		run(chromedp.Evaluate(fmt.Sprintf(`document.querySelectorAll(%q).length`, sel), &n))
		return n
	}
	// waitCount waits until the number of elements matching sel satisfies
	// cond, a JS comparison such as ">= 3", checked in the page.
	waitCount := func(what, sel, cond string) {
		t.Helper()
		js := fmt.Sprintf(`document.querySelectorAll(%q).length %s`, sel, cond)
		if err := chromedp.Run(ctx, waitForDOM(js, 15*time.Second)); err != nil {
			dump()
			t.Fatalf("waiting for %s: %v", what, err)
		}
	}

	if err := chromedp.Run(ctx, network.Enable(), chromedp.Navigate(url), chromedp.WaitVisible(`[data-testid="log"]`, chromedp.ByQuery)); err != nil {
		dump()
		t.Fatal(err)
	}

	t.Run("rows arrive with no user action, from every input", func(t *testing.T) {
		waitCount("gen stdout rows", "li.src-gen.stream-stdout", ">= 3")
		waitCount("gen stderr rows", "li.src-gen.stream-stderr", ">= 3")
		waitCount("container rows via docker logs -f", "li.src-box", ">= 2")
		waitCount("tailed file backlog", "li.src-app", ">= 1")
	})

	// Docker Chrome receives frames late for several seconds once, early in
	// the page's life: the browser-side frame timestamps show the gap, while a
	// raw client on the host sees none. The container's 0.5-CPU limit is the
	// likely cause. Let delivery settle before the timing-sensitive checks.
	waitSteady := func() {
		t.Helper()
		deadline := time.Now().Add(30 * time.Second)
		for time.Now().Before(deadline) {
			logs.mu.Lock()
			n := len(logs.recvAt)
			steady := n > 10 && time.Since(logs.recvAt[n-1]) < 300*time.Millisecond
			for i := n - 1; steady && i > 0 && time.Since(logs.recvAt[i]) < 2*time.Second; i-- {
				steady = logs.recvAt[i].Sub(logs.recvAt[i-1]) < 500*time.Millisecond
			}
			logs.mu.Unlock()
			if steady {
				return
			}
			time.Sleep(200 * time.Millisecond)
		}
		dump()
		t.Fatal("frame delivery never settled")
	}

	t.Run("pushes are range append ops, not full re-renders", func(t *testing.T) {
		appends := 0
		for _, f := range logs.frames() {
			if strings.Contains(f, `["a",`) {
				appends++
			}
		}
		if appends < 3 {
			dump()
			t.Fatalf("frames with an append op = %d, want >= 3", appends)
		}
	})

	t.Run("appending to the tailed file shows up within a second", func(t *testing.T) {
		waitSteady()
		f, _ := os.OpenFile(appLog, os.O_APPEND|os.O_WRONLY, 0)
		fmt.Fprintln(f, "app wrote this")
		f.Close()
		start := time.Now()
		waitCount("appended file line", "li.src-app", ">= 2")
		if d := time.Since(start); d > time.Second {
			dump()
			t.Fatalf("file line took %v to appear", d)
		}
	})

	t.Run("rows are coloured by level with no page CSS", func(t *testing.T) {
		f, _ := os.OpenFile(appLog, os.O_APPEND|os.O_WRONLY, 0)
		fmt.Fprintln(f, `{"level":"error","msg":"styled error"}`)
		f.Close()
		waitCount("an error row", "li.lvl-error", ">= 1")
		waitCount("a warn row", "li.lvl-warn", ">= 1")
		// colour resolves a CSS colour the way the browser computes it inside
		// the block, so the test compares against the theme's own values.
		var got map[string]string
		if err := run(chromedp.Evaluate(`(() => {
			const block = document.querySelector(".tinkerdown-interactive-block");
			const colour = c => { const p = document.createElement("span"); p.style.color = c; block.appendChild(p);
				const v = getComputedStyle(p).color; p.remove(); return v; };
			const of = sel => getComputedStyle(document.querySelector(sel)).color;
			return { error: of("li.lvl-error"), wantError: colour("var(--pico-del-color)"),
				warn: of("li.lvl-warn"), wantWarn: colour("var(--tinkerdown-warn-color)"),
				info: of("li.lvl-info") };
		})()`, &got)); err != nil {
			dump()
			t.Fatal(err)
		}
		if got["error"] != got["wantError"] || got["warn"] != got["wantWarn"] || got["info"] == got["error"] || got["info"] == got["warn"] {
			dump()
			t.Fatalf("row colours: %v", got)
		}
	})

	t.Run("two pages share one process", func(t *testing.T) {
		// A second connection to the same page, as another browser tab makes.
		wsURL := "ws" + strings.TrimPrefix(ts.URL, "http") + "/ws?page=" + neturl.QueryEscape("/")
		conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
		if err != nil {
			t.Fatal(err)
		}
		defer conn.Close()
		conn.SetReadDeadline(time.Now().Add(5 * time.Second))
		if _, first, err := conn.ReadMessage(); err != nil || !strings.Contains(string(first), "tick") {
			t.Fatalf("second page's first frame: err %v, frame %.200s", err, first)
		}
		out, _ := exec.Command("pgrep", "-c", "-f", genMarker).Output()
		if got := strings.TrimSpace(string(out)); got != "1" {
			t.Fatalf("processes running the gen command = %s, want 1", got)
		}
	})

	t.Run("pause stops new rows and resume catches up", func(t *testing.T) {
		if err := run(chromedp.Click(`[data-testid="pause"]`, chromedp.ByQuery),
			chromedp.WaitVisible(`[data-testid="resume"]`, chromedp.ByQuery)); err != nil {
			dump()
			t.Fatal(err)
		}
		before := count("li.src-gen")
		time.Sleep(time.Second)
		if after := count("li.src-gen"); after != before {
			dump()
			t.Fatalf("rows changed while paused: %d -> %d", before, after)
		}
		if err := run(chromedp.Click(`[data-testid="resume"]`, chromedp.ByQuery)); err != nil {
			t.Fatal(err)
		}
		waitCount("rows after resume", "li.src-gen", "> "+fmt.Sprint(before))
	})

	t.Run("search filter narrows rows, tab filter combines with it", func(t *testing.T) {
		if err := run(chromedp.SendKeys(`[data-testid="search"]`, "spill", chromedp.ByQuery)); err != nil {
			t.Fatal(err)
		}
		waitCount("only matching rows", "li:not(.stream-stderr)", "== 0")
		if err := run(chromedp.Click(`button[role="tab"][data-filter="source = box"]`, chromedp.ByQuery)); err != nil {
			t.Fatal(err)
		}
		// box rows never contain "spill", and gen rows are on another tab
		waitCount("box tab with spill search", "li", "== 0")
	})

}

func tail(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return "…" + s[len(s)-n:]
}
