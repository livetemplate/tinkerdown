//go:build !ci

package tinkerdown_test

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/chromedp"
)

// TestPlainCommandFlags_E2E runs the real serve command with sources that
// have no names: one after "--", one --source command and one --source
// file:. It checks in the browser that each gets a tab labelled from its
// command or file, and that a repeated label becomes api-2.
func TestPlainCommandFlags_E2E(t *testing.T) {
	bin := buildTinkerdown(t, t.TempDir())
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "index.md"), []byte(flagsPage), 0o644)
	os.WriteFile(filepath.Join(dir, "app.log"), []byte("app one\n"), 0o644)
	// Prints two lines, then stays up, so it isn't restarted and repeated.
	os.WriteFile(filepath.Join(dir, "api.sh"), []byte("printf 'api one\\napi two\\n'\nexec tail -f /dev/null\n"), 0o755)

	// Chrome takes longest to start and needs nothing else, so it goes first.
	chrome, cleanup := SetupDockerChrome(t, 90*time.Second)
	defer cleanup()

	port, err := getFreePort()
	if err != nil {
		t.Fatal(err)
	}
	serverLog := &syncBuffer{}
	cmd := exec.Command(bin, "serve", dir, "--allow-exec", "--port", fmt.Sprint(port),
		"--source", "sh ./api.sh", "--source", "file:app.log", "--", "sh", "./api.sh")
	cmd.Dir = dir
	cmd.Stdout, cmd.Stderr = serverLog, serverLog
	// Its own process group, so cleanup stops the server and the commands it
	// started, not only the first process.
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
		cmd.Wait()
		if t.Failed() {
			t.Logf("server log:\n%s", serverLog.String())
		}
	})
	url := fmt.Sprintf("http://localhost:%d", port)
	WaitForServer(t, url, 20*time.Second)

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
	if err := chromedp.Run(ctx, network.Enable(), chromedp.Navigate(ConvertURLForDockerChrome(url)),
		chromedp.WaitVisible(`[data-testid="log"]`, chromedp.ByQuery)); err != nil {
		dump()
		t.Fatal(err)
	}

	var tabs []string
	actx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	if err := chromedp.Run(actx, chromedp.Evaluate(`[...document.querySelectorAll('button[role="tab"]')].map(b => b.textContent)`, &tabs)); err != nil {
		dump()
		t.Fatal(err)
	}
	if want := "All|api|api-2|app"; strings.Join(tabs, "|") != want {
		dump()
		t.Fatalf("tabs = %q, want %q", strings.Join(tabs, "|"), want)
	}
	if err := chromedp.Run(ctx, waitForDOM(`document.querySelectorAll("li.src-api").length === 2 && document.querySelectorAll("li.src-api-2").length === 2 && document.querySelectorAll("li.src-app").length === 1`, 15*time.Second)); err != nil {
		dump()
		t.Fatalf("rows from each unnamed source: %v", err)
	}
	for _, line := range logs.console {
		if strings.Contains(strings.ToLower(line), "error") {
			dump()
			t.Fatalf("browser console error: %s", line)
		}
	}
}
