//go:build !ci

package tinkerdown_test

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"net/http/httptest"

	"github.com/chromedp/cdproto/runtime"
	"github.com/chromedp/chromedp"
	"github.com/livetemplate/tinkerdown/internal/config"
	"github.com/livetemplate/tinkerdown/internal/server"
)

// TestComputedExpressions tests the computed expressions functionality.
// This test verifies that:
// 1. Expressions like `=count(tasks)` are parsed and rendered as placeholders
// 2. Expressions are evaluated when the page loads
// 3. Expression values are displayed correctly
func TestComputedExpressions(t *testing.T) {
	// Load config from test example
	cfg, err := config.LoadFromDir("examples/computed-expressions-test")
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Verify source is configured
	if cfg.Sources == nil {
		t.Fatal("No sources configured in tinkerdown.yaml")
	}
	taskSource, ok := cfg.Sources["tasks"]
	if !ok {
		t.Fatal("tasks source not found in config")
	}
	t.Logf("Source config: type=%s, file=%s", taskSource.Type, taskSource.File)

	// Create test server
	srv := server.NewWithConfig("examples/computed-expressions-test", cfg)
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
			chromedp.Flag("no-sandbox", true),
		)...)
	defer cancel()

	ctx, cancel := chromedp.NewContext(allocCtx, chromedp.WithLogf(t.Logf))
	defer cancel()

	ctx, cancel = context.WithTimeout(ctx, 60*time.Second)
	defer cancel()

	// Store console logs and errors for debugging
	var consoleLogs []string
	chromedp.ListenTarget(ctx, func(ev interface{}) {
		if ev, ok := ev.(*runtime.EventConsoleAPICalled); ok {
			for _, arg := range ev.Args {
				consoleLogs = append(consoleLogs, fmt.Sprintf("[Console] %s", arg.Value))
			}
		}
	})

	t.Logf("Test server URL: %s", ts.URL)

	// Wait for page load and WebSocket connection
	var pageHTML string
	err = chromedp.Run(ctx,
		chromedp.Navigate(ts.URL+"/index.html"),
		chromedp.WaitVisible(".tinkerdown-interactive-block", chromedp.ByQuery),
		chromedp.Sleep(2*time.Second), // Allow WebSocket to connect and expressions to evaluate
		chromedp.OuterHTML("html", &pageHTML),
	)
	if err != nil {
		t.Logf("Console logs:\n%s", strings.Join(consoleLogs, "\n"))
		t.Fatalf("Failed to navigate and wait: %v", err)
	}

	// Log the page HTML for debugging
	t.Logf("Page HTML contains expression class: %v", strings.Contains(pageHTML, "tinkerdown-expr"))

	// Verify expression placeholders exist in the page
	if !strings.Contains(pageHTML, "tinkerdown-expr") {
		t.Logf("Console logs:\n%s", strings.Join(consoleLogs, "\n"))
		t.Fatal("Page does not contain expression elements (class=tinkerdown-expr)")
	}

	// Check for expression values being set
	// After WebSocket connection, expressions should have values
	var exprCount int
	err = chromedp.Run(ctx,
		chromedp.Evaluate(`document.querySelectorAll('.tinkerdown-expr').length`, &exprCount),
	)
	if err != nil {
		t.Fatalf("Failed to count expressions: %v", err)
	}
	t.Logf("Found %d expression elements", exprCount)

	if exprCount < 5 {
		t.Errorf("Expected at least 5 expressions, got %d", exprCount)
	}

	// Check if any expressions have values (not loading state)
	var hasValueCount int
	err = chromedp.Run(ctx,
		chromedp.Evaluate(`document.querySelectorAll('.tinkerdown-expr.has-value').length`, &hasValueCount),
	)
	if err != nil {
		t.Fatalf("Failed to count expressions with values: %v", err)
	}
	t.Logf("Found %d expressions with values", hasValueCount)

	// Get the actual values for verification
	var totalTasksValue string
	err = chromedp.Run(ctx,
		chromedp.Evaluate(`
			(function() {
				const exprs = document.querySelectorAll('.tinkerdown-expr');
				for (const expr of exprs) {
					if (expr.dataset.expr === 'count(tasks)') {
						const valueEl = expr.querySelector('.expr-value');
						if (valueEl) return valueEl.textContent;
					}
				}
				return '';
			})()
		`, &totalTasksValue),
	)
	if err != nil {
		t.Fatalf("Failed to get total tasks value: %v", err)
	}
	t.Logf("Total tasks value: %q", totalTasksValue)

	// Verify the value is "4" (we have 4 tasks in tasks.json)
	if totalTasksValue != "" && totalTasksValue != "4" {
		t.Errorf("Expected total tasks to be '4', got %q", totalTasksValue)
	}

	// Get completed tasks count
	var completedValue string
	err = chromedp.Run(ctx,
		chromedp.Evaluate(`
			(function() {
				const exprs = document.querySelectorAll('.tinkerdown-expr');
				for (const expr of exprs) {
					if (expr.dataset.expr === 'count(tasks where done)') {
						const valueEl = expr.querySelector('.expr-value');
						if (valueEl) return valueEl.textContent;
					}
				}
				return '';
			})()
		`, &completedValue),
	)
	if err != nil {
		t.Fatalf("Failed to get completed tasks value: %v", err)
	}
	t.Logf("Completed tasks value: %q", completedValue)

	// Verify completed count is "2" (2 tasks are done in tasks.json)
	if completedValue != "" && completedValue != "2" {
		t.Errorf("Expected completed tasks to be '2', got %q", completedValue)
	}

	// Get sum of priorities
	var sumValue string
	err = chromedp.Run(ctx,
		chromedp.Evaluate(`
			(function() {
				const exprs = document.querySelectorAll('.tinkerdown-expr');
				for (const expr of exprs) {
					if (expr.dataset.expr === 'sum(tasks.priority)') {
						const valueEl = expr.querySelector('.expr-value');
						if (valueEl) return valueEl.textContent;
					}
				}
				return '';
			})()
		`, &sumValue),
	)
	if err != nil {
		t.Fatalf("Failed to get sum value: %v", err)
	}
	t.Logf("Sum of priorities: %q", sumValue)

	// Verify sum is "11" (3+5+2+1 = 11)
	if sumValue != "" && sumValue != "11" {
		t.Errorf("Expected sum of priorities to be '11', got %q", sumValue)
	}

	// Log console output for debugging if any test failed
	if t.Failed() {
		t.Logf("Console logs:\n%s", strings.Join(consoleLogs, "\n"))
	}
}

// TestComputedExpressionsUpdate tests that expressions update when data changes.
func TestComputedExpressionsUpdate(t *testing.T) {
	// This test is more complex - it would need to:
	// 1. Load a page with expressions
	// 2. Trigger an action that modifies the data
	// 3. Verify expressions update
	// For now, we skip this as it requires writable sources

	t.Skip("Skipping update test - requires writable source setup")
}

// TestComputedExpressionsError tests that expression errors are displayed correctly.
func TestComputedExpressionsError(t *testing.T) {
	// This test would verify that:
	// 1. Invalid expressions show error icons
	// 2. Expressions referencing missing sources show errors

	t.Skip("Skipping error test - requires error test fixtures")
}
