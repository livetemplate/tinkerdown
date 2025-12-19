package livepage_test

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"net/http/httptest"

	"github.com/chromedp/cdproto/runtime"
	"github.com/chromedp/chromedp"
	"github.com/livetemplate/livepage/internal/config"
	"github.com/livetemplate/livepage/internal/server"
)

// TestLvtSourceExec tests the lvt-source functionality with exec type
// This test verifies that:
// 1. lvt-source="users" fetches data from the configured exec command
// 2. The data is rendered in the template
// 3. The Refresh action re-fetches data
func TestLvtSourceExec(t *testing.T) {
	// Load config from test example
	cfg, err := config.LoadFromDir("examples/lvt-source-test")
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Verify source is configured
	if cfg.Sources == nil {
		t.Fatal("No sources configured in livepage.yaml")
	}
	userSource, ok := cfg.Sources["users"]
	if !ok {
		t.Fatal("users source not found in config")
	}
	if userSource.Type != "exec" {
		t.Fatalf("Expected exec source type, got: %s", userSource.Type)
	}
	t.Logf("Source config: type=%s, cmd=%s", userSource.Type, userSource.Cmd)

	// Create test server
	srv := server.NewWithConfig("examples/lvt-source-test", cfg)
	if err := srv.Discover(); err != nil {
		t.Fatalf("Failed to discover pages: %v", err)
	}

	handler := server.WithCompression(srv)
	ts := httptest.NewServer(handler)
	defer ts.Close()

	// Setup chromedp
	allocCtx, cancel := chromedp.NewExecAllocator(context.Background(),
		append(chromedp.DefaultExecAllocatorOptions[:],
			chromedp.Flag("headless", true),
		)...)
	defer cancel()

	ctx, cancel := chromedp.NewContext(allocCtx, chromedp.WithLogf(t.Logf))
	defer cancel()

	ctx, cancel = context.WithTimeout(ctx, 60*time.Second)
	defer cancel()

	// Store console logs for debugging
	var consoleLogs []string
	chromedp.ListenTarget(ctx, func(ev interface{}) {
		if ev, ok := ev.(*runtime.EventConsoleAPICalled); ok {
			for _, arg := range ev.Args {
				consoleLogs = append(consoleLogs, fmt.Sprintf("[Console] %s", arg.Value))
			}
		}
	})

	t.Logf("Test server URL: %s", ts.URL)

	// Test 1: Navigate and wait for WebSocket to render content
	var hasInteractiveBlock bool
	err = chromedp.Run(ctx,
		chromedp.Navigate(ts.URL+"/"),
		chromedp.Sleep(3*time.Second),
		chromedp.Evaluate(`document.querySelector('.livepage-interactive-block') !== null`, &hasInteractiveBlock),
	)
	if err != nil {
		t.Fatalf("Failed to navigate: %v", err)
	}

	if !hasInteractiveBlock {
		var htmlContent string
		chromedp.Run(ctx, chromedp.OuterHTML("html", &htmlContent))
		t.Logf("HTML: %s", htmlContent[:min(2000, len(htmlContent))])
		t.Fatal("Page did not load correctly - no interactive block found")
	}

	// Wait for WebSocket to render the table
	var tableRendered bool
	err = chromedp.Run(ctx,
		chromedp.Sleep(2*time.Second),
		chromedp.Evaluate(`document.querySelector('[lvt-source] table') !== null`, &tableRendered),
	)
	if err != nil {
		t.Fatalf("Failed to wait for table: %v", err)
	}

	if !tableRendered {
		var htmlContent string
		chromedp.Run(ctx, chromedp.OuterHTML("html", &htmlContent))
		t.Logf("HTML (first 2000 chars): %s", htmlContent[:min(2000, len(htmlContent))])
		t.Logf("Console logs: %v", consoleLogs)
		t.Fatal("Table was not rendered by WebSocket - table not found in lvt-source container")
	}
	t.Log("Page loaded and table rendered via WebSocket")

	// Test 2: Verify user data is rendered (using data attributes instead of classes)
	var rowCount int
	var firstUserName string
	var firstUserEmail string
	err = chromedp.Run(ctx,
		chromedp.Evaluate(`document.querySelectorAll('tr[data-user-id]').length`, &rowCount),
		chromedp.Evaluate(`
			(() => {
				const row = document.querySelector('tr[data-user-id]');
				const nameCell = row ? row.querySelector('td:nth-child(2)') : null;
				return nameCell ? nameCell.textContent : '';
			})()
		`, &firstUserName),
		chromedp.Evaluate(`
			(() => {
				const row = document.querySelector('tr[data-user-id]');
				const emailCell = row ? row.querySelector('td:nth-child(3)') : null;
				return emailCell ? emailCell.textContent : '';
			})()
		`, &firstUserEmail),
	)
	if err != nil {
		t.Fatalf("Failed to check user data: %v", err)
	}

	if rowCount == 0 {
		t.Logf("Console logs: %v", consoleLogs)
		t.Fatal("No user rows found - data was not fetched from source")
	}
	t.Logf("Found %d user rows", rowCount)

	if rowCount != 3 {
		t.Fatalf("Expected 3 users from exec script, got %d", rowCount)
	}
	t.Log("Correct number of users rendered")

	// Verify first user data
	if !strings.Contains(firstUserName, "Alice") {
		t.Fatalf("Expected first user name to be 'Alice', got: %s", firstUserName)
	}
	t.Log("First user name is Alice")

	if !strings.Contains(firstUserEmail, "alice@example.com") {
		t.Fatalf("Expected first user email to be 'alice@example.com', got: %s", firstUserEmail)
	}
	t.Log("First user email is alice@example.com")

	// Test 3: Verify Refresh button exists
	var refreshButtonExists bool
	err = chromedp.Run(ctx,
		chromedp.Evaluate(`document.querySelector('button[lvt-click="Refresh"]') !== null`, &refreshButtonExists),
	)
	if err != nil {
		t.Fatalf("Failed to check refresh button: %v", err)
	}

	if !refreshButtonExists {
		t.Fatal("Refresh button not found")
	}
	t.Log("Refresh button exists")

	// Test 4: Click refresh and verify data is still present (re-fetched)
	err = chromedp.Run(ctx,
		chromedp.Click(`button[lvt-click="Refresh"]`, chromedp.ByQuery),
		chromedp.Sleep(2*time.Second),
	)
	if err != nil {
		t.Fatalf("Failed to click refresh: %v", err)
	}
	t.Log("Clicked refresh button")

	// Verify data is still present after refresh
	var rowCountAfterRefresh int
	err = chromedp.Run(ctx,
		chromedp.Evaluate(`document.querySelectorAll('tr[data-user-id]').length`, &rowCountAfterRefresh),
	)
	if err != nil {
		t.Fatalf("Failed to check rows after refresh: %v", err)
	}

	if rowCountAfterRefresh != 3 {
		t.Logf("Console logs: %v", consoleLogs)
		t.Fatalf("Expected 3 rows after refresh, got %d", rowCountAfterRefresh)
	}
	t.Log("Data persisted after refresh")

	t.Log("All lvt-source exec tests passed!")
}
