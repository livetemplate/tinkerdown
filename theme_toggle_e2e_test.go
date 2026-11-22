//go:build !ci

package tinkerdown_test

import (
	"log"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/cdproto/runtime"
	"github.com/chromedp/chromedp"
	"github.com/livetemplate/tinkerdown/internal/config"
	"github.com/livetemplate/tinkerdown/internal/server"
)

// TestThemeDropdown drives the single-button theme menu: the trigger opens a
// Light/Dark/System menu, a choice applies and persists across reloads,
// Escape and outside clicks close it, and inside the sidebar footer the menu
// opens upward. SERIAL — it redirects the process-global logger.
func TestThemeDropdown(t *testing.T) {
	var serverLog syncBuffer
	prevOut := log.Writer()
	log.SetOutput(&serverLog)
	t.Cleanup(func() { log.SetOutput(prevOut) })

	cfg, err := config.LoadFromDir("examples/docs-site")
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	srv := server.NewWithConfig("examples/docs-site", cfg)
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

	var html string
	dump := func() {
		mu.Lock()
		defer mu.Unlock()
		t.Logf("console logs:\n%s", strings.Join(consoleLogs, "\n"))
		t.Logf("ws frames received: %d", len(wsFrames))
		t.Logf("server log:\n%s", serverLog.String())
		t.Logf("rendered html (theme toggle region): %s", excerpt(html, `class="theme-toggle`, 2500))
	}
	fail := func(format string, args ...any) {
		t.Helper()
		dump()
		t.Fatalf(format, args...)
	}

	url := ConvertURLForDockerChrome(ts.URL)
	if err := chromedp.Run(ctx,
		network.Enable(),
		chromedp.Navigate(url+"/"),
		chromedp.WaitVisible(".nav-sidebar-footer .theme-toggle", chromedp.ByQuery),
		chromedp.Evaluate(`localStorage.removeItem('tinkerdown-theme')`, nil),
		chromedp.Reload(),
		chromedp.WaitVisible(".nav-sidebar-footer #theme-menu-toggle", chromedp.ByQuery),
		chromedp.OuterHTML("html", &html),
	); err != nil {
		fail("load page: %v", err)
	}

	// One trigger replaces the three buttons: only one icon is visible on it.
	var visibleTriggerIcons int
	var menuHidden bool
	if err := chromedp.Run(ctx,
		chromedp.Evaluate(`[...document.querySelectorAll('#theme-menu-toggle svg')].filter(s => getComputedStyle(s).display !== 'none').length`, &visibleTriggerIcons),
		chromedp.Evaluate(`getComputedStyle(document.getElementById('theme-menu')).display === 'none'`, &menuHidden),
	); err != nil {
		fail("inspect trigger: %v", err)
	}
	if visibleTriggerIcons != 1 {
		fail("trigger shows %d icons, want 1", visibleTriggerIcons)
	}
	if !menuHidden {
		fail("menu visible before the trigger is clicked")
	}

	// Open: menu shows, aria-expanded flips, and in the footer it sits above the trigger.
	var expanded string
	var opensUpward bool
	if err := chromedp.Run(ctx,
		chromedp.Click("#theme-menu-toggle", chromedp.ByQuery),
		chromedp.WaitVisible("#theme-menu", chromedp.ByQuery),
		chromedp.AttributeValue("#theme-menu-toggle", "aria-expanded", &expanded, nil),
		chromedp.Evaluate(`document.getElementById('theme-menu').getBoundingClientRect().bottom <= document.getElementById('theme-menu-toggle').getBoundingClientRect().top`, &opensUpward),
		chromedp.OuterHTML("html", &html),
	); err != nil {
		fail("open menu: %v", err)
	}
	if expanded != "true" {
		fail("aria-expanded = %q after open, want true", expanded)
	}
	if !opensUpward {
		fail("menu in sidebar footer should open upward")
	}

	// Pick Dark: theme applies, choice persists, menu closes, trigger shows the moon.
	var dataTheme, stored, choice, darkChecked string
	if err := chromedp.Run(ctx,
		chromedp.Click("#theme-dark", chromedp.ByQuery),
		chromedp.WaitNotVisible("#theme-menu", chromedp.ByQuery),
		chromedp.AttributeValue("html", "data-theme", &dataTheme, nil),
		chromedp.Evaluate(`localStorage.getItem('tinkerdown-theme')`, &stored),
		chromedp.AttributeValue(".theme-toggle", "data-choice", &choice, nil),
		chromedp.AttributeValue("#theme-dark", "aria-checked", &darkChecked, nil),
	); err != nil {
		fail("select dark: %v", err)
	}
	if dataTheme != "dark" || stored != "dark" || choice != "dark" || darkChecked != "true" {
		fail("after Dark: data-theme=%q stored=%q choice=%q aria-checked=%q", dataTheme, stored, choice, darkChecked)
	}

	// Reload: the choice survives.
	if err := chromedp.Run(ctx,
		chromedp.Reload(),
		chromedp.WaitVisible(".nav-sidebar-footer #theme-menu-toggle", chromedp.ByQuery),
		chromedp.AttributeValue("html", "data-theme", &dataTheme, nil),
		chromedp.AttributeValue(".theme-toggle", "data-choice", &choice, nil),
	); err != nil {
		fail("reload: %v", err)
	}
	if dataTheme != "dark" || choice != "dark" {
		fail("after reload: data-theme=%q choice=%q, want dark", dataTheme, choice)
	}

	// Escape closes the menu and returns focus to the trigger.
	var focusOnTrigger bool
	if err := chromedp.Run(ctx,
		chromedp.Click("#theme-menu-toggle", chromedp.ByQuery),
		chromedp.WaitVisible("#theme-menu", chromedp.ByQuery),
		chromedp.KeyEvent("\u001b"),
		chromedp.WaitNotVisible("#theme-menu", chromedp.ByQuery),
		chromedp.Evaluate(`document.activeElement && document.activeElement.id === 'theme-menu-toggle'`, &focusOnTrigger),
	); err != nil {
		fail("escape: %v", err)
	}
	if !focusOnTrigger {
		fail("focus should return to the trigger after Escape")
	}

	// An outside click closes the menu without changing the theme.
	if err := chromedp.Run(ctx,
		chromedp.Click("#theme-menu-toggle", chromedp.ByQuery),
		chromedp.WaitVisible("#theme-menu", chromedp.ByQuery),
		chromedp.Click(".content-wrapper", chromedp.ByQuery),
		chromedp.WaitNotVisible("#theme-menu", chromedp.ByQuery),
		chromedp.AttributeValue("html", "data-theme", &dataTheme, nil),
	); err != nil {
		fail("outside click: %v", err)
	}
	if dataTheme != "dark" {
		fail("outside click changed theme to %q", dataTheme)
	}

	// Ctrl+Shift+D cycles dark -> auto and the trigger follows.
	if err := chromedp.Run(ctx,
		chromedp.Evaluate(`document.dispatchEvent(new KeyboardEvent('keydown', {key: 'D', ctrlKey: true, shiftKey: true}))`, nil),
		chromedp.AttributeValue(".theme-toggle", "data-choice", &choice, nil),
		chromedp.AttributeValue("#theme-auto", "aria-checked", &darkChecked, nil),
	); err != nil {
		fail("shortcut: %v", err)
	}
	if choice != "auto" || darkChecked != "true" {
		fail("after Ctrl+Shift+D: choice=%q auto aria-checked=%q, want auto/true", choice, darkChecked)
	}

	mu.Lock()
	for _, l := range consoleLogs {
		if strings.HasPrefix(l, "EXCEPTION:") {
			mu.Unlock()
			fail("page threw: %s", l)
		}
	}
	mu.Unlock()
}

// excerpt returns up to n bytes of s starting at the first occurrence of marker.
func excerpt(s, marker string, n int) string {
	i := strings.Index(s, marker)
	if i < 0 {
		return "(marker not found)"
	}
	if i+n > len(s) {
		return s[i:]
	}
	return s[i : i+n]
}
