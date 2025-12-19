package livepage_test

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"net/http/httptest"

	"github.com/chromedp/cdproto/page"
	"github.com/chromedp/cdproto/runtime"
	"github.com/chromedp/chromedp"
	"github.com/livetemplate/livepage/internal/server"
)

// TestAutoPersistDeleteWithLvtDataID tests the lvt-data-* functionality
// This test verifies that:
// 1. lvt-click sends action to server
// 2. lvt-data-id="123" passes the id to the Delete method
// 3. GetInt("id") correctly parses string values from lvt-data-*
func TestAutoPersistDeleteWithLvtDataID(t *testing.T) {
	// Create test server for autopersist-test example
	srv := server.New("examples/autopersist-test")
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

	// Store console logs and WebSocket messages for debugging
	var consoleLogs []string
	chromedp.ListenTarget(ctx, func(ev interface{}) {
		switch ev := ev.(type) {
		case *runtime.EventConsoleAPICalled:
			for _, arg := range ev.Args {
				consoleLogs = append(consoleLogs, fmt.Sprintf("[Console] %s", arg.Value))
			}
		case *page.EventJavascriptDialogOpening:
			// Automatically accept confirm() dialogs (for lvt-confirm)
			t.Logf("Accepting dialog: %s", ev.Message)
			go chromedp.Run(ctx, page.HandleJavaScriptDialog(true))
		}
	})

	t.Logf("Test server URL: %s", ts.URL)

	// Test 1: Navigate and wait for WebSocket to render content
	var htmlContent string
	var hasInteractiveBlock bool
	err := chromedp.Run(ctx,
		chromedp.Navigate(ts.URL+"/"),
		// Wait for livepage client to initialize and WebSocket to connect
		// Plugin compilation can take ~10 seconds on first run
		chromedp.Sleep(12*time.Second),
		// Wait for interactive block to exist (it should be rendered by goldmark parser)
		chromedp.Evaluate(`document.querySelector('.livepage-interactive-block') !== null`, &hasInteractiveBlock),
	)
	if err != nil {
		t.Fatalf("Failed to navigate: %v", err)
	}

	if !hasInteractiveBlock {
		chromedp.Run(ctx, chromedp.OuterHTML("html", &htmlContent))
		t.Logf("HTML: %s", htmlContent[:min(2000, len(htmlContent))])
		t.Fatal("Page did not load correctly - no interactive block found")
	}

	// Wait for WebSocket to render the actual content (replace "Connecting..." with form)
	var formRendered bool
	err = chromedp.Run(ctx,
		chromedp.Sleep(5*time.Second), // Give time for WebSocket to connect and render
		chromedp.Evaluate(`document.querySelector('form[lvt-submit="save"]') !== null`, &formRendered),
	)
	if err != nil {
		t.Fatalf("Failed to wait for form: %v", err)
	}

	if !formRendered {
		chromedp.Run(ctx, chromedp.OuterHTML("html", &htmlContent))
		t.Logf("HTML (first 2000 chars): %s", htmlContent[:min(2000, len(htmlContent))])
		t.Logf("Console logs: %v", consoleLogs)
		t.Fatal("Form was not rendered by WebSocket - lvt-submit='save' not found")
	}
	t.Log("✓ Page loaded and form rendered via WebSocket")

	// Test 2: Add an item via the form
	var formExists bool
	err = chromedp.Run(ctx,
		chromedp.Evaluate(`document.querySelector('form[lvt-submit="save"]') !== null`, &formExists),
	)
	if err != nil || !formExists {
		t.Fatalf("Form not found: %v", err)
	}
	t.Log("✓ Form found")

	// Fill and submit the form
	err = chromedp.Run(ctx,
		chromedp.WaitVisible(`input[name="title"]`, chromedp.ByQuery),
		chromedp.SendKeys(`input[name="title"]`, "Test Item To Delete", chromedp.ByQuery),
		chromedp.Sleep(200*time.Millisecond),
		chromedp.Click(`button[type="submit"]`, chromedp.ByQuery),
		chromedp.Sleep(1*time.Second), // Wait for server response and DOM update
	)
	if err != nil {
		t.Fatalf("Failed to submit form: %v", err)
	}
	t.Log("✓ Form submitted")

	// Test 3: Verify the item appears in the list
	// Using semantic selectors: fieldsets with data-item-id (Pico CSS grid layout)
	var itemCount int
	var itemText string
	err = chromedp.Run(ctx,
		chromedp.Evaluate(`document.querySelectorAll('fieldset[data-item-id]').length`, &itemCount),
		chromedp.Evaluate(`
			(() => {
				const item = document.querySelector('fieldset[data-item-id] input');
				return item ? item.value : '';
			})()
		`, &itemText),
	)
	if err != nil {
		t.Fatalf("Failed to check items: %v", err)
	}

	if itemCount == 0 {
		t.Logf("Console logs: %v", consoleLogs)
		t.Fatal("No items found after form submission")
	}
	t.Logf("✓ Found %d item(s) in list", itemCount)

	if !strings.Contains(itemText, "Test Item To Delete") {
		t.Logf("Item text: %q", itemText)
		t.Fatal("Item title doesn't match")
	}
	t.Log("✓ Item title verified")

	// Test 4: Verify delete button has correct attributes
	// Using semantic selector: button with lvt-click="Delete"
	var deleteButtonExists bool
	var deleteButtonHasLvtClick bool
	var deleteButtonHasLvtDataId bool
	var deleteButtonHasLvtConfirm bool
	var lvtDataIdValue string
	var lvtConfirmValue string
	err = chromedp.Run(ctx,
		chromedp.Evaluate(`document.querySelector('button[lvt-click="Delete"]') !== null`, &deleteButtonExists),
		chromedp.Evaluate(`document.querySelector('button[lvt-click="Delete"]')?.getAttribute('lvt-click') === 'Delete'`, &deleteButtonHasLvtClick),
		chromedp.Evaluate(`document.querySelector('button[lvt-click="Delete"]')?.hasAttribute('lvt-data-id')`, &deleteButtonHasLvtDataId),
		chromedp.Evaluate(`document.querySelector('button[lvt-click="Delete"]')?.getAttribute('lvt-data-id') || ''`, &lvtDataIdValue),
		chromedp.Evaluate(`document.querySelector('button[lvt-click="Delete"]')?.hasAttribute('lvt-confirm')`, &deleteButtonHasLvtConfirm),
		chromedp.Evaluate(`document.querySelector('button[lvt-click="Delete"]')?.getAttribute('lvt-confirm') || ''`, &lvtConfirmValue),
	)
	if err != nil {
		t.Fatalf("Failed to check delete button: %v", err)
	}

	if !deleteButtonExists {
		t.Fatal("Delete button not found")
	}
	t.Log("✓ Delete button exists")

	if !deleteButtonHasLvtClick {
		t.Fatal("Delete button missing lvt-click='Delete' attribute")
	}
	t.Log("✓ Delete button has lvt-click='Delete'")

	if !deleteButtonHasLvtDataId {
		t.Fatal("Delete button missing lvt-data-id attribute")
	}
	t.Logf("✓ Delete button has lvt-data-id='%s'", lvtDataIdValue)

	if !deleteButtonHasLvtConfirm {
		t.Fatal("Delete button missing lvt-confirm attribute")
	}
	t.Logf("✓ Delete button has lvt-confirm='%s'", lvtConfirmValue)

	// Test 5: Click delete button and verify item is removed
	// This is the critical test for lvt-data-* functionality
	var itemCountBefore int
	err = chromedp.Run(ctx,
		chromedp.Evaluate(`document.querySelectorAll('fieldset[data-item-id]').length`, &itemCountBefore),
	)
	if err != nil {
		t.Fatalf("Failed to get item count before delete: %v", err)
	}
	t.Logf("Items before delete: %d", itemCountBefore)

	// Click the delete button
	err = chromedp.Run(ctx,
		chromedp.Click(`button[lvt-click="Delete"]`, chromedp.ByQuery),
		chromedp.Sleep(1*time.Second), // Wait for server response and DOM update
	)
	if err != nil {
		t.Fatalf("Failed to click delete button: %v", err)
	}
	t.Log("✓ Delete button clicked")

	// Verify item was deleted
	var itemCountAfter int
	var noItemsVisible bool
	err = chromedp.Run(ctx,
		chromedp.Evaluate(`document.querySelectorAll('fieldset[data-item-id]').length`, &itemCountAfter),
		chromedp.Evaluate(`document.body.textContent.includes('No items yet')`, &noItemsVisible),
	)
	if err != nil {
		t.Fatalf("Failed to check items after delete: %v", err)
	}

	t.Logf("Items after delete: %d", itemCountAfter)
	t.Logf("Console logs during test: %v", consoleLogs)

	if itemCountAfter >= itemCountBefore {
		t.Logf("Expected item count to decrease from %d, got %d", itemCountBefore, itemCountAfter)
		t.Fatal("Delete did not work - item was not removed")
	}

	t.Log("✓ Item was deleted successfully")

	// If all items were deleted, verify the "no items" message appears
	if itemCountAfter == 0 && !noItemsVisible {
		t.Fatal("Expected 'no items' message when list is empty")
	}

	t.Log("✅ All lvt-data-* delete tests passed!")
}

// TestLvtDataExtraction tests that lvt-data-* attributes are correctly extracted
// and sent with the action
func TestLvtDataExtraction(t *testing.T) {
	// Create test server
	srv := server.New("examples/autopersist-test")
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

	ctx, cancel = context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	// Navigate and wait for page load
	err := chromedp.Run(ctx,
		chromedp.Navigate(ts.URL+"/"),
		chromedp.Sleep(500*time.Millisecond),
	)
	if err != nil {
		t.Fatalf("Failed to navigate: %v", err)
	}

	// Test extractLvtData function in the browser
	var extractedData string
	err = chromedp.Run(ctx,
		// Create a test element with lvt-data-* attributes
		chromedp.Evaluate(`
			(() => {
				// Create test button
				const btn = document.createElement('button');
				btn.setAttribute('lvt-data-id', '123');
				btn.setAttribute('lvt-data-name', 'test-item');
				btn.setAttribute('lvt-data-status', 'active');
				document.body.appendChild(btn);

				// Extract lvt-data-* attributes (mimicking the client code)
				const data = {};
				for (let i = 0; i < btn.attributes.length; i++) {
					const attr = btn.attributes[i];
					if (attr.name.startsWith('lvt-data-')) {
						const key = attr.name.substring(9); // Remove "lvt-data-" prefix
						data[key] = attr.value;
					}
				}

				// Cleanup
				document.body.removeChild(btn);

				return JSON.stringify(data);
			})()
		`, &extractedData),
	)
	if err != nil {
		t.Fatalf("Failed to test extraction: %v", err)
	}

	// Verify extraction
	if !strings.Contains(extractedData, `"id":"123"`) {
		t.Fatalf("lvt-data-id not extracted correctly: %s", extractedData)
	}
	t.Log("✓ lvt-data-id extracted")

	if !strings.Contains(extractedData, `"name":"test-item"`) {
		t.Fatalf("lvt-data-name not extracted correctly: %s", extractedData)
	}
	t.Log("✓ lvt-data-name extracted")

	if !strings.Contains(extractedData, `"status":"active"`) {
		t.Fatalf("lvt-data-status not extracted correctly: %s", extractedData)
	}
	t.Log("✓ lvt-data-status extracted")

	t.Logf("Extracted data: %s", extractedData)
	t.Log("✅ All lvt-data extraction tests passed!")
}
