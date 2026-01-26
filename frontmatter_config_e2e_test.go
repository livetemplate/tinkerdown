//go:build !ci

package tinkerdown_test

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"net/http/httptest"

	"github.com/chromedp/cdproto/runtime"
	"github.com/chromedp/chromedp"
	"github.com/livetemplate/tinkerdown/internal/server"
)

// TestFrontmatterSources tests that sources defined in markdown frontmatter work correctly
// without requiring a tinkerdown.yaml file.
func TestFrontmatterSources(t *testing.T) {
	// Create test server - no config file needed, sources are in frontmatter
	srv := server.New("examples/lvt-source-file-test")
	if err := srv.Discover(); err != nil {
		t.Fatalf("Failed to discover pages: %v", err)
	}

	handler := server.WithCompression(srv)
	ts := httptest.NewServer(handler)
	defer ts.Close()

	// Setup Docker Chrome for reliable CI execution
	chromeCtx, cleanup := SetupDockerChrome(t, 60*time.Second)
	defer cleanup()

	ctx := chromeCtx.Context

	// Store console logs for debugging
	var consoleLogs []string
	chromedp.ListenTarget(ctx, func(ev interface{}) {
		if ev, ok := ev.(*runtime.EventConsoleAPICalled); ok {
			for _, arg := range ev.Args {
				consoleLogs = append(consoleLogs, fmt.Sprintf("[Console] %s", arg.Value))
			}
		}
	})

	// Convert URL for Docker Chrome access
	url := ConvertURLForDockerChrome(ts.URL)
	t.Logf("Test server URL: %s (Docker: %s)", ts.URL, url)

	// Test 1: Navigate and wait for WebSocket to render content
	var hasUserTable bool
	err := chromedp.Run(ctx,
		chromedp.Navigate(url+"/"),
		chromedp.Sleep(5*time.Second),
		chromedp.Evaluate(`document.querySelector('[lvt-source="users"] table') !== null`, &hasUserTable),
	)
	if err != nil {
		t.Fatalf("Failed to navigate: %v", err)
	}

	if !hasUserTable {
		var htmlContent string
		chromedp.Run(ctx, chromedp.OuterHTML("html", &htmlContent))
		t.Logf("HTML (first 3000 chars): %s", htmlContent[:min(3000, len(htmlContent))])
		t.Logf("Console logs: %v", consoleLogs)
		t.Fatal("User table was not rendered - frontmatter source failed")
	}
	t.Log("User table rendered from frontmatter-defined JSON source")

	// Test 2: Verify user data is rendered correctly
	var userRowCount int
	var firstUserName string
	err = chromedp.Run(ctx,
		chromedp.Evaluate(`document.querySelectorAll('[lvt-source="users"] tbody tr').length`, &userRowCount),
		chromedp.Evaluate(`
			(() => {
				const row = document.querySelector('[lvt-source="users"] tbody tr');
				const nameCell = row ? row.querySelector('td:nth-child(2)') : null;
				return nameCell ? nameCell.textContent : '';
			})()
		`, &firstUserName),
	)
	if err != nil {
		t.Fatalf("Failed to check user data: %v", err)
	}

	if userRowCount != 3 {
		t.Fatalf("Expected 3 users from frontmatter JSON source, got %d", userRowCount)
	}
	t.Log("Correct number of users rendered from frontmatter source")

	if !strings.Contains(firstUserName, "Alice") {
		t.Fatalf("Expected first user name to be 'Alice', got: %s", firstUserName)
	}
	t.Log("First user name is Alice - frontmatter source config working!")

	t.Log("Frontmatter source configuration test passed!")
}
