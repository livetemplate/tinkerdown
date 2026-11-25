//go:build !ci

package tinkerdown_test

import (
	"context"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/cdproto/runtime"
	"github.com/chromedp/chromedp"
	"github.com/livetemplate/tinkerdown/internal/server"
)

// TestTutorialToolbarInSidebar: a page with several H2s and sidebar: true gets
// a client-built tutorial sidebar, and the page toolbar ends up in that sidebar's footer.
func TestTutorialToolbarInSidebar(t *testing.T) {
	serverLog := captureServerLog(t)

	dir := t.TempDir()
	page := "---\nsidebar: true\n---\n\n# Tutorial\n\nIntro.\n\n## First step\n\nOne.\n\n## Second step\n\nTwo.\n\n## Third step\n\nThree.\n"
	if err := os.WriteFile(filepath.Join(dir, "index.md"), []byte(page), 0o644); err != nil {
		t.Fatal(err)
	}
	srv := server.New(dir)
	if err := srv.Discover(); err != nil {
		t.Fatalf("discover: %v", err)
	}
	ts := httptest.NewServer(server.WithCompression(srv))
	defer ts.Close()

	chromeCtx, cleanup := SetupDockerChrome(t, 60*time.Second)
	defer cleanup()
	ctx := chromeCtx.Context

	var mu sync.Mutex
	var consoleLogs, wsFrames []string
	chromedp.ListenTarget(ctx, func(ev any) {
		mu.Lock()
		defer mu.Unlock()
		switch e := ev.(type) {
		case *runtime.EventConsoleAPICalled:
			for _, arg := range e.Args {
				consoleLogs = append(consoleLogs, string(arg.Value))
			}
		case *runtime.EventExceptionThrown:
			consoleLogs = append(consoleLogs, "EXCEPTION: "+e.ExceptionDetails.Error())
		case *network.EventWebSocketFrameReceived:
			if e.Response != nil {
				wsFrames = append(wsFrames, e.Response.PayloadData)
			}
		}
	})
	fail := func(format string, args ...any) {
		t.Helper()
		var html string
		_ = chromedp.Run(ctx, chromedp.OuterHTML("body", &html, chromedp.ByQuery))
		mu.Lock()
		t.Logf("console logs:\n%s", strings.Join(consoleLogs, "\n"))
		t.Logf("ws frames received: %d", len(wsFrames))
		mu.Unlock()
		t.Logf("server log:\n%s", serverLog.String())
		t.Logf("rendered html: %s", excerpt(html, `tinkerdown-nav-sidebar`, 3000))
		t.Fatalf(format, args...)
	}

	if err := chromedp.Run(ctx,
		network.Enable(),
		chromedp.Navigate(ConvertURLForDockerChrome(ts.URL)+"/"),
	); err != nil {
		fail("load page: %v", err)
	}
	waitCtx, cancelWait := context.WithTimeout(ctx, 15*time.Second)
	err := chromedp.Run(waitCtx, chromedp.WaitVisible(".tinkerdown-nav-sidebar .nav-step", chromedp.ByQuery))
	cancelWait()
	if err != nil {
		fail("tutorial sidebar never appeared: %v", err)
	}

	var steps, toolbars int
	var inFooter, footerLast bool
	if err := chromedp.Run(ctx,
		chromedp.Poll(`document.querySelector('.tinkerdown-nav-sidebar .nav-sidebar-footer .page-toolbar') !== null`, &inFooter, chromedp.WithPollingTimeout(5*time.Second)),
		chromedp.Evaluate(`document.querySelectorAll('.tinkerdown-nav-sidebar .nav-step').length`, &steps),
		chromedp.Evaluate(`document.querySelectorAll('.page-toolbar').length`, &toolbars),
		chromedp.Evaluate(`(() => { const s = document.querySelector('.tinkerdown-nav-sidebar'); return s.lastElementChild === s.querySelector('.nav-sidebar-footer'); })()`, &footerLast),
	); err != nil {
		fail("inspect sidebar: %v", err)
	}
	if steps != 3 {
		fail("sidebar has %d steps, want 3", steps)
	}
	if !inFooter || toolbars != 1 || !footerLast {
		fail("toolbar not moved into the sidebar footer once (inFooter=%v toolbars=%d footerLast=%v)", inFooter, toolbars, footerLast)
	}
}
