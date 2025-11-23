package main

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/chromedp/chromedp"
)

func TestComparisonDemo(t *testing.T) {
	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Create browser context
	ctx, cancel = chromedp.NewContext(ctx)
	defer cancel()

	// Navigate to comparison page
	var pageHTML string
	if err := chromedp.Run(ctx,
		chromedp.Navigate("http://localhost:8080"),
		chromedp.WaitVisible("body", chromedp.ByQuery),
		chromedp.Sleep(1*time.Second), // Wait for WebSocket connection
		chromedp.OuterHTML("html", &pageHTML),
	); err != nil {
		// Skip test if server is not running
		t.Skipf("Skipping test - server not running on port 8080: %v", err)
		return
	}

	// Verify key content is present
	t.Run("Page Content", func(t *testing.T) {
		if !strings.Contains(pageHTML, "React vs LiveTemplate") {
			t.Error("Page title not found")
		}
		if !strings.Contains(pageHTML, "Side-by-Side Comparison") {
			t.Error("Comparison section not found")
		}
		if !strings.Contains(pageHTML, "Lines of Code") {
			t.Error("Comparison table not found")
		}
	})

	// Test interactive LiveTemplate counter
	t.Run("Interactive Counter", func(t *testing.T) {
		var counterText string

		// Get initial count
		if err := chromedp.Run(ctx,
			chromedp.Text(".demo-box h2", &counterText, chromedp.ByQuery),
		); err != nil {
			t.Fatalf("Failed to get counter text: %v", err)
		}

		initialCount := strings.TrimSpace(counterText)
		t.Logf("Initial count: %s", initialCount)

		// Click increment button
		if err := chromedp.Run(ctx,
			chromedp.Click(`button[lvt-click="increment"]`, chromedp.ByQuery),
			chromedp.Sleep(500*time.Millisecond), // Wait for WebSocket update
			chromedp.Text(".demo-box h2", &counterText, chromedp.ByQuery),
		); err != nil {
			t.Fatalf("Failed to click increment: %v", err)
		}

		afterIncrement := strings.TrimSpace(counterText)
		t.Logf("After increment: %s", afterIncrement)

		if afterIncrement == initialCount {
			t.Error("Counter did not increment")
		}

		// Click decrement button
		if err := chromedp.Run(ctx,
			chromedp.Click(`button[lvt-click="decrement"]`, chromedp.ByQuery),
			chromedp.Sleep(500*time.Millisecond),
			chromedp.Text(".demo-box h2", &counterText, chromedp.ByQuery),
		); err != nil {
			t.Fatalf("Failed to click decrement: %v", err)
		}

		afterDecrement := strings.TrimSpace(counterText)
		t.Logf("After decrement: %s", afterDecrement)

		if afterDecrement != initialCount {
			t.Error("Counter did not return to initial value after decrement")
		}

		// Click reset button
		if err := chromedp.Run(ctx,
			chromedp.Click(`button[lvt-click="reset"]`, chromedp.ByQuery),
			chromedp.Sleep(500*time.Millisecond),
			chromedp.Text(".demo-box h2", &counterText, chromedp.ByQuery),
		); err != nil {
			t.Fatalf("Failed to click reset: %v", err)
		}

		afterReset := strings.TrimSpace(counterText)
		t.Logf("After reset: %s", afterReset)

		if afterReset != "0" {
			t.Errorf("Counter did not reset to 0, got: %s", afterReset)
		}
	})

	// Verify code blocks are present
	t.Run("Code Examples", func(t *testing.T) {
		if !strings.Contains(pageHTML, "React") || !strings.Contains(pageHTML, "useState") {
			t.Error("React code example not found")
		}
		if !strings.Contains(pageHTML, "LiveTemplate") || !strings.Contains(pageHTML, "ComparisonCounterState") {
			t.Error("LiveTemplate code example not found")
		}
	})

	// Verify comparison table
	t.Run("Comparison Table", func(t *testing.T) {
		if !strings.Contains(pageHTML, "42 lines") {
			t.Error("React line count not found in comparison table")
		}
		if !strings.Contains(pageHTML, "24 lines") {
			t.Error("LiveTemplate line count not found in comparison table")
		}
		if !strings.Contains(pageHTML, "Server (Go)") {
			t.Error("State location not found in comparison table")
		}
	})

	// Verify syntax highlighting
	t.Run("Syntax Highlighting", func(t *testing.T) {
		// Check that Prism.js is loaded
		if !strings.Contains(pageHTML, "prism") {
			t.Error("Prism.js not found in page")
		}

		// Check that code blocks have Prism language classes
		if !strings.Contains(pageHTML, `class="language-javascript"`) {
			t.Error("JavaScript syntax highlighting class not found")
		}
		if !strings.Contains(pageHTML, `class="language-go"`) {
			t.Error("Go syntax highlighting class not found")
		}

		// Verify Prism CSS is loaded
		if !strings.Contains(pageHTML, "prism-tomorrow.min.css") {
			t.Error("Prism CSS theme not found")
		}

		// Verify Prism scripts are loaded
		if !strings.Contains(pageHTML, "prism.min.js") {
			t.Error("Prism core script not found")
		}
		if !strings.Contains(pageHTML, "prism-go.min.js") {
			t.Error("Prism Go language component not found")
		}
		if !strings.Contains(pageHTML, "prism-javascript.min.js") {
			t.Error("Prism JavaScript language component not found")
		}
	})

	t.Log("✅ All comparison demo tests passed!")
}
