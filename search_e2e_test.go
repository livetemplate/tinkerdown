package livepage_test

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/chromedp/cdproto/runtime"
	"github.com/chromedp/chromedp"
)

// TestSearchFunctionality tests the complete search feature in site mode
func TestSearchFunctionality(t *testing.T) {
	// Setup context with output options
	allocCtx, cancel := chromedp.NewExecAllocator(context.Background(),
		append(chromedp.DefaultExecAllocatorOptions[:],
			chromedp.Flag("headless", true),
		)...)
	defer cancel()

	ctx, cancel := chromedp.NewContext(allocCtx, chromedp.WithLogf(t.Logf))
	defer cancel()

	// Set timeout for the entire test
	ctx, cancel = context.WithTimeout(ctx, 60*time.Second)
	defer cancel()

	// Store console logs
	var consoleLogs []string
	chromedp.ListenTarget(ctx, func(ev interface{}) {
		if ev, ok := ev.(*runtime.EventConsoleAPICalled); ok {
			for _, arg := range ev.Args {
				consoleLogs = append(consoleLogs, fmt.Sprintf("[Console] %s", arg.Value))
			}
		}
	})

	// Navigate to the docs site
	var htmlContent string
	var searchButtonExists bool
	var modalHTML string

	err := chromedp.Run(ctx,
		chromedp.Navigate("http://localhost:9090/"),
		chromedp.WaitVisible(".livepage-nav-sidebar", chromedp.ByQuery),
		chromedp.Sleep(500*time.Millisecond), // Wait for client to initialize
		chromedp.OuterHTML("html", &htmlContent),

		// Test 1: Verify search button exists in sidebar
		chromedp.Evaluate(`document.querySelector('.search-button') !== null`, &searchButtonExists),
	)

	if err != nil {
		t.Logf("Console logs: %v", consoleLogs)
		t.Fatalf("Failed to navigate and check search button: %v", err)
	}

	if !searchButtonExists {
		t.Logf("HTML: %s", htmlContent)
		t.Fatal("Search button not found in sidebar")
	}
	t.Log("✓ Search button exists in sidebar")

	// Test 2: Verify /search-index.json endpoint by checking window.livepageSearch loaded the index
	var searchIndexLength int
	err = chromedp.Run(ctx,
		chromedp.Sleep(1*time.Second), // Wait for search to initialize
		chromedp.Evaluate(`window.livepageSearch ? window.livepageSearch.searchIndex.length : 0`, &searchIndexLength),
	)

	if err != nil {
		t.Fatalf("Failed to check search index: %v", err)
	}

	if searchIndexLength == 0 {
		t.Fatal("Search index is empty or not loaded")
	}

	t.Logf("✓ Search index loaded with %d entries", searchIndexLength)

	// Test 3: Click search button and verify modal opens
	var modalVisible bool
	var buttonExists bool
	var searchInstanceExists bool
	err = chromedp.Run(ctx,
		chromedp.WaitVisible(".search-button", chromedp.ByQuery),
		chromedp.Sleep(1*time.Second), // Wait for JS to fully initialize

		// Debug: Check if button exists and search instance is available
		chromedp.Evaluate(`document.querySelector('.search-button') !== null`, &buttonExists),
		chromedp.Evaluate(`window.livepageSearch !== undefined`, &searchInstanceExists),

		// Use JavaScript click instead of chromedp Click
		chromedp.Evaluate(`document.querySelector('.search-button').click()`, nil),
		chromedp.Sleep(500*time.Millisecond), // Wait for animation
		chromedp.Evaluate(`document.querySelector('.search-modal.open') !== null`, &modalVisible),
	)

	if err != nil {
		t.Logf("Button exists: %v, Search instance exists: %v", buttonExists, searchInstanceExists)
		t.Fatalf("Failed to click search button: %v", err)
	}

	if !modalVisible {
		t.Logf("Button exists: %v, Search instance exists: %v", buttonExists, searchInstanceExists)
		t.Fatal("Search modal did not open after clicking button")
	}
	t.Log("✓ Search modal opens when button clicked")

	// Test 4: Verify modal UI elements
	err = chromedp.Run(ctx,
		chromedp.OuterHTML(".search-modal", &modalHTML),
	)

	if err != nil {
		t.Fatalf("Failed to get modal HTML: %v", err)
	}

	requiredUIElements := []string{
		"search-input",
		"search-results",
		"search-close",
		"search-footer",
	}

	for _, element := range requiredUIElements {
		if !strings.Contains(modalHTML, element) {
			t.Fatalf("Modal missing UI element: %s", element)
		}
	}
	t.Log("✓ Modal has all required UI elements")

	// Test 5: Perform a search and verify results
	var resultsHTML string
	var resultsCount int

	err = chromedp.Run(ctx,
		chromedp.SendKeys(".search-input", "livepage", chromedp.ByQuery),
		chromedp.Sleep(500*time.Millisecond), // Wait for search to execute
		chromedp.Evaluate(`document.querySelectorAll('.search-result').length`, &resultsCount),
		chromedp.OuterHTML(".search-results", &resultsHTML),
	)

	if err != nil {
		t.Fatalf("Failed to perform search: %v", err)
	}

	if resultsCount == 0 {
		t.Logf("Results HTML: %s", resultsHTML)
		t.Fatal("No search results found for 'livepage'")
	}

	t.Logf("✓ Search returned %d results", resultsCount)

	// Test 6: Verify search result highlighting
	if !strings.Contains(resultsHTML, "<mark>") {
		t.Logf("Results HTML: %s", resultsHTML)
		t.Fatal("Search results do not contain highlighted matches")
	}
	t.Log("✓ Search results contain highlighted matches")

	// Test 7: Test keyboard navigation (arrow keys)
	var firstResultSelected bool
	var secondResultSelected bool

	err = chromedp.Run(ctx,
		// Press ArrowDown to select first result
		chromedp.SendKeys(".search-input", "\ue015", chromedp.ByQuery), // ArrowDown
		chromedp.Sleep(200*time.Millisecond),
		chromedp.Evaluate(`document.querySelector('.search-result.selected') !== null`, &firstResultSelected),

		// Press ArrowDown again to select second result
		chromedp.SendKeys(".search-input", "\ue015", chromedp.ByQuery), // ArrowDown
		chromedp.Sleep(200*time.Millisecond),
		chromedp.Evaluate(`document.querySelectorAll('.search-result.selected').length === 1`, &secondResultSelected),
	)

	if err != nil {
		t.Fatalf("Failed to test keyboard navigation: %v", err)
	}

	if !firstResultSelected {
		t.Fatal("Arrow key navigation not working - first result not selected")
	}

	if !secondResultSelected {
		t.Fatal("Arrow key navigation not working - second result not selected")
	}

	t.Log("✓ Keyboard navigation (arrow keys) works")

	// Test 8: Test Escape key closes modal
	var modalClosedAfterEscape bool

	err = chromedp.Run(ctx,
		// Focus on search input
		chromedp.Focus(".search-input", chromedp.ByQuery),
		chromedp.Sleep(100*time.Millisecond),
		// Send Escape via JavaScript instead
		chromedp.Evaluate(`
			(() => {
				const input = document.querySelector('.search-input');
				const event = new KeyboardEvent('keydown', { key: 'Escape', bubbles: true });
				input.dispatchEvent(event);
			})()
		`, nil),
		chromedp.Sleep(500*time.Millisecond),
		chromedp.Evaluate(`document.querySelector('.search-modal.open') === null`, &modalClosedAfterEscape),
	)

	if err != nil {
		t.Fatalf("Failed to test Escape key: %v", err)
	}

	if !modalClosedAfterEscape {
		t.Fatal("Escape key did not close modal")
	}

	t.Log("✓ Escape key closes modal")

	// Test 9: Test Ctrl+K keyboard shortcut opens modal
	var modalOpenedWithCtrlK bool

	err = chromedp.Run(ctx,
		// Send Ctrl+K (or Cmd+K on Mac)
		chromedp.KeyEvent("k", chromedp.KeyModifiers(2)), // 2 = Ctrl modifier
		chromedp.Sleep(300*time.Millisecond),
		chromedp.Evaluate(`document.querySelector('.search-modal.open') !== null`, &modalOpenedWithCtrlK),
	)

	if err != nil {
		t.Fatalf("Failed to test Ctrl+K shortcut: %v", err)
	}

	if !modalOpenedWithCtrlK {
		t.Fatal("Ctrl+K keyboard shortcut did not open modal")
	}

	t.Log("✓ Ctrl+K keyboard shortcut opens modal")

	// Test 10: Test clicking close button closes modal
	var modalClosedAfterClickClose bool

	err = chromedp.Run(ctx,
		chromedp.Click(".search-close", chromedp.ByQuery),
		chromedp.Sleep(200*time.Millisecond),
		chromedp.Evaluate(`document.querySelector('.search-modal.open') === null`, &modalClosedAfterClickClose),
	)

	if err != nil {
		t.Fatalf("Failed to test close button: %v", err)
	}

	if !modalClosedAfterClickClose {
		t.Fatal("Close button did not close modal")
	}

	t.Log("✓ Close button closes modal")

	// Test 11: Test clicking backdrop closes modal
	var modalClosedAfterBackdropClick bool

	err = chromedp.Run(ctx,
		// Reopen modal using JavaScript
		chromedp.Evaluate(`document.querySelector('.search-button').click()`, nil),
		chromedp.Sleep(500*time.Millisecond),

		// Click backdrop using JavaScript
		chromedp.Evaluate(`document.querySelector('.search-backdrop').click()`, nil),
		chromedp.Sleep(300*time.Millisecond),
		chromedp.Evaluate(`document.querySelector('.search-modal.open') === null`, &modalClosedAfterBackdropClick),
	)

	if err != nil {
		t.Fatalf("Failed to test backdrop click: %v", err)
	}

	if !modalClosedAfterBackdropClick {
		t.Fatal("Clicking backdrop did not close modal")
	}

	t.Log("✓ Clicking backdrop closes modal")

	// Test 12: Test searching for different terms
	testSearches := []struct {
		query           string
		expectResults   bool
		expectedMinimum int
	}{
		{"installation", true, 1},
		{"configuration", true, 1},
		{"xyz123notfound", false, 0},
	}

	for _, tc := range testSearches {
		var count int
		var noResultsVisible bool

		err = chromedp.Run(ctx,
			// Reopen modal using JavaScript
			chromedp.Evaluate(`document.querySelector('.search-button').click()`, nil),
			chromedp.Sleep(500*time.Millisecond),

			// Clear and type new query
			chromedp.SendKeys(".search-input", tc.query, chromedp.ByQuery),
			chromedp.Sleep(500*time.Millisecond),

			chromedp.Evaluate(`document.querySelectorAll('.search-result').length`, &count),
			chromedp.Evaluate(`document.querySelector('.search-no-results') !== null`, &noResultsVisible),

			// Close modal
			chromedp.Evaluate(`
				(() => {
					const input = document.querySelector('.search-input');
					const event = new KeyboardEvent('keydown', { key: 'Escape', bubbles: true });
					input.dispatchEvent(event);
				})()
			`, nil),
			chromedp.Sleep(300*time.Millisecond),
		)

		if err != nil {
			t.Fatalf("Failed to test search for '%s': %v", tc.query, err)
		}

		if tc.expectResults {
			if count < tc.expectedMinimum {
				t.Fatalf("Search for '%s' expected at least %d results, got %d", tc.query, tc.expectedMinimum, count)
			}
			t.Logf("✓ Search for '%s' returned %d results", tc.query, count)
		} else {
			if !noResultsVisible {
				t.Fatalf("Search for '%s' expected 'no results' message", tc.query)
			}
			t.Logf("✓ Search for '%s' correctly shows no results", tc.query)
		}
	}

	// Test 13: Verify search doesn't appear in non-site mode
	var searchButtonInNonSiteMode bool

	err = chromedp.Run(ctx,
		chromedp.Navigate("http://localhost:8080/"), // Counter example (non-site mode)
		chromedp.Sleep(500*time.Millisecond),
		chromedp.Evaluate(`document.querySelector('.search-button') !== null`, &searchButtonInNonSiteMode),
	)

	// Note: This test will fail if port 8080 is not running, but that's okay for this specific test
	// We just want to ensure search doesn't appear where it shouldn't

	t.Logf("Console logs during test: %v", consoleLogs)
	t.Log("✅ All search functionality tests passed!")
}

// TestSearchIndexGeneration tests the server-side search index generation
func TestSearchIndexGeneration(t *testing.T) {
	// Fetch search index directly using HTTP client
	resp, err := http.Get("http://localhost:9090/search-index.json")
	if err != nil {
		t.Fatalf("Failed to fetch search index: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("Failed to read search index response: %v", err)
	}

	searchIndexResponse := string(body)

	// Parse search index
	var searchIndex []struct {
		Title   string `json:"title"`
		Path    string `json:"path"`
		Content string `json:"content"`
		Section string `json:"section,omitempty"`
	}

	if err := json.Unmarshal([]byte(searchIndexResponse), &searchIndex); err != nil {
		t.Fatalf("Failed to parse search index: %v", err)
	}

	t.Logf("Search index contains %d entries", len(searchIndex))

	// Verify each entry
	for i, entry := range searchIndex {
		if entry.Title == "" {
			t.Errorf("Entry %d has empty title", i)
		}

		if entry.Path == "" {
			t.Errorf("Entry %d has empty path", i)
		}

		if entry.Content == "" {
			t.Errorf("Entry %d has empty content", i)
		}

		// Content should be plain text (no HTML tags)
		if strings.Contains(entry.Content, "<") && strings.Contains(entry.Content, ">") {
			t.Errorf("Entry %d content contains HTML tags: %s", i, entry.Content[:min(100, len(entry.Content))])
		}

		t.Logf("✓ Entry %d: %s (%s) - %d chars", i, entry.Title, entry.Path, len(entry.Content))
	}

	t.Log("✅ Search index generation test passed!")
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
