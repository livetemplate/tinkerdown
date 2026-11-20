//go:build !ci

package tinkerdown_test

import (
	"context"
	"fmt"
	"log"
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
	serverLog := captureServerLogs(t)
	log.SetFlags(log.LstdFlags | log.Lmicroseconds)
	defer log.SetFlags(log.LstdFlags)
	config.SetAllowExec(true)
	defer config.SetAllowExec(false)

	marker := fmt.Sprintf("tdstream%d", os.Getpid())
	container := marker + "-box"
	if out, err := exec.Command("docker", "run", "-d", "--rm", "--name", container, "alpine:3.21",
		"sh", "-c", `i=0; while true; do echo "box line $i"; i=$((i+1)); sleep 0.3; done`).CombinedOutput(); err != nil {
		t.Fatalf("start container: %v\n%s", err, out)
	}
	defer exec.Command("docker", "rm", "-f", container).Run()

	dir := t.TempDir()
	appLog := filepath.Join(dir, "app.log")
	os.WriteFile(appLog, []byte("app started\n"), 0o644)
	genMarker := "genmark" + fmt.Sprint(os.Getpid())
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
		t.Logf("console:\n%s", strings.Join(logs.console, "\n"))
		n := len(logs.wsRecv)
		last := logs.wsRecv[max(0, n-5):]
		logs.mu.Unlock()
		t.Logf("ws frames received: %d, last: %s", n, strings.Join(last, "\n"))
		logs.mu.Lock()
		t.Logf("ws frames sent: %s", strings.Join(logs.wsSent, "\n"))
		logs.mu.Unlock()
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
		chromedp.Run(ctx, chromedp.Evaluate(fmt.Sprintf(`document.querySelectorAll(%q).length`, sel), &n))
		return n
	}
	waitCount := func(what, sel string, ok func(int) bool) int {
		t.Helper()
		deadline := time.Now().Add(15 * time.Second)
		for time.Now().Before(deadline) {
			if n := count(sel); ok(n) {
				return n
			}
			time.Sleep(100 * time.Millisecond)
		}
		dump()
		t.Fatalf("timed out waiting for %s (%s = %d)", what, sel, count(sel))
		return 0
	}

	if err := chromedp.Run(ctx, network.Enable(), chromedp.Navigate(url), chromedp.WaitVisible(`[data-testid="log"]`, chromedp.ByQuery)); err != nil {
		dump()
		t.Fatal(err)
	}

	t.Run("rows arrive with no user action, from every input", func(t *testing.T) {
		waitCount("gen stdout rows", "li.src-gen.stream-stdout", func(n int) bool { return n >= 3 })
		waitCount("gen stderr rows", "li.src-gen.stream-stderr", func(n int) bool { return n >= 3 })
		waitCount("container rows via docker logs -f", "li.src-box", func(n int) bool { return n >= 2 })
		waitCount("tailed file backlog", "li.src-app", func(n int) bool { return n >= 1 })
	})

	// Docker Chrome's network path can stall frame delivery for several
	// seconds early on (a raw client on the host sees no gap), so let delivery
	// settle before the timing-sensitive checks.
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
		log.Printf("e2e: appending to app.log")
		fmt.Fprintln(f, "app wrote this")
		f.Close()
		start := time.Now()
		waitCount("appended file line", "li.src-app", func(n int) bool { return n >= 2 })
		if d := time.Since(start); d > time.Second {
			logs.mu.Lock()
			for i, f := range logs.wsRecv {
				if strings.Contains(f, "app wrote this") {
					t.Logf("frame with the line arrived %v after the write", logs.recvAt[i].Sub(start))
				}
			}
			logs.mu.Unlock()
			all := serverLog.String()
			i := strings.Index(all, "e2e: appending to app.log")
			seg := all[i:]
			if j := strings.Index(seg, "app wrote this"); j > 0 {
				seg = seg[:j+200]
			}
			var short []string
			for _, ln := range strings.Split(seg, "\n") {
				short = append(short, tail(ln, 160))
			}
			t.Logf("server log from the write to the line:\n%s", strings.Join(short, "\n"))
			t.Fatalf("file line took %v to appear", d)
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
		waitCount("rows after resume", "li.src-gen", func(n int) bool { return n > before })
	})

	var tabs string
	run(chromedp.OuterHTML(".tinkerdown-tabs-bar", &tabs, chromedp.ByQuery))
	t.Logf("tab bar: %s", tabs)

	t.Run("search filter narrows rows, tab filter combines with it", func(t *testing.T) {
		if err := run(chromedp.SendKeys(`[data-testid="search"]`, "spill", chromedp.ByQuery)); err != nil {
			t.Fatal(err)
		}
		waitCount("only matching rows", "li:not(.stream-stderr)", func(n int) bool { return n == 0 })
		if err := run(chromedp.Click(`button[role="tab"][data-filter="source = box"]`, chromedp.ByQuery)); err != nil {
			t.Fatal(err)
		}
		// box rows never contain "spill", and gen rows are on another tab
		waitCount("box tab with spill search", "li", func(n int) bool { return n == 0 })
	})

	if t.Failed() {
		dump()
	}
}

func tail(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return "…" + s[len(s)-n:]
}
