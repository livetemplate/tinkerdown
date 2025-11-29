package livepage_test

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/cdproto/runtime"
	"github.com/chromedp/chromedp"
)

// TestInteractiveBlocksE2E verifies that interactive blocks work end-to-end:
// 1. Page loads without "Connecting..." message
// 2. Interactive blocks are rendered
// 3. User can interact (toggle checkbox, add todo)
func TestInteractiveBlocksE2E(t *testing.T) {
	// Build livepage first
	buildCmd := exec.Command("go", "build", "-o", "livepage-test", "./cmd/livepage")
	buildCmd.Dir = "/Users/adnaan/code/livetemplate/livepage"
	if err := buildCmd.Run(); err != nil {
		t.Fatalf("Failed to build livepage: %v", err)
	}
	defer os.Remove("/Users/adnaan/code/livetemplate/livepage/livepage-test")

	// Start the server
	serverCmd := exec.Command("/Users/adnaan/code/livetemplate/livepage/livepage-test", "serve", "examples/todos-workshop", "--debug")
	serverCmd.Dir = "/Users/adnaan/code/livetemplate/livepage/.worktrees/todos-workshop"

	// Capture server output
	serverOut := &strings.Builder{}
	serverCmd.Stdout = serverOut
	serverCmd.Stderr = serverOut

	if err := serverCmd.Start(); err != nil {
		t.Fatalf("Failed to start server: %v", err)
	}
	defer func() {
		serverCmd.Process.Kill()
		t.Logf("\n=== Server Logs ===\n%s", serverOut.String())
	}()

	// Wait for server to be ready
	time.Sleep(3 * time.Second)

	// Test each page
	pages := []struct {
		path string
		name string
	}{
		{"validation", "Validation Page"},
		{"persistence", "Persistence Page"},
	}

	for _, page := range pages {
		t.Run(page.name, func(t *testing.T) {
			testPageInteraction(t, page.path)
		})
	}
}

func testPageInteraction(t *testing.T, pagePath string) {
	// Setup chromedp
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("headless", true),
		chromedp.Flag("disable-gpu", false),
		chromedp.Flag("no-sandbox", true),
	)

	allocCtx, cancel := chromedp.NewExecAllocator(context.Background(), opts...)
	defer cancel()

	ctx, cancel := chromedp.NewContext(allocCtx)
	defer cancel()

	// Set timeout
	ctx, cancel = context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	// Capture browser console logs
	consoleLogs := []string{}
	chromedp.ListenTarget(ctx, func(ev interface{}) {
		switch ev := ev.(type) {
		case *runtime.EventConsoleAPICalled:
			args := make([]string, len(ev.Args))
			for i, arg := range ev.Args {
				args[i] = fmt.Sprintf("%v", arg.Value)
			}
			msg := fmt.Sprintf("[Console %s] %s", ev.Type, strings.Join(args, " "))
			consoleLogs = append(consoleLogs, msg)
			t.Logf("%s", msg)
		case *runtime.EventExceptionThrown:
			msg := fmt.Sprintf("[JS Error] %s", ev.ExceptionDetails.Text)
			consoleLogs = append(consoleLogs, msg)
			t.Errorf("%s", msg) // Fail on JS errors
		}
	})

	// Capture WebSocket messages
	wsMessages := []string{}
	chromedp.ListenTarget(ctx, func(ev interface{}) {
		switch ev := ev.(type) {
		case *network.EventWebSocketFrameReceived:
			msg := fmt.Sprintf("[WS <-] %s", ev.Response.PayloadData)
			wsMessages = append(wsMessages, msg)
			t.Logf("%s", msg)
		case *network.EventWebSocketFrameSent:
			msg := fmt.Sprintf("[WS ->] %s", ev.Response.PayloadData)
			wsMessages = append(wsMessages, msg)
			t.Logf("%s", msg)
		case *network.EventWebSocketCreated:
			t.Logf("[WS] Created: %s", ev.URL)
		case *network.EventWebSocketHandshakeResponseReceived:
			t.Logf("[WS] Handshake completed")
		}
	})

	// Enable network events
	if err := chromedp.Run(ctx, network.Enable()); err != nil {
		t.Fatalf("Failed to enable network: %v", err)
	}

	url := fmt.Sprintf("http://localhost:8080/%s", pagePath)
	t.Logf("Testing page: %s", url)

	var hasConnectingMessage bool
	var hasInteractiveBlock bool
	var initialTodoCount int
	var checkboxExists bool

	// Load page and check initial state
	if err := chromedp.Run(ctx,
		chromedp.Navigate(url),
		chromedp.Sleep(5*time.Second), // Wait for page load and WebSocket

		// Check for "Connecting..." message (should NOT exist)
		chromedp.Evaluate(`document.body.textContent.includes('Connecting...')`, &hasConnectingMessage),

		// Check for interactive block
		chromedp.Evaluate(`document.querySelector('.livepage-interactive-block') !== null`, &hasInteractiveBlock),

		// Count initial todos
		chromedp.Evaluate(`document.querySelectorAll('.todo-item').length`, &initialTodoCount),

		// Check if checkbox exists
		chromedp.Evaluate(`document.querySelector('input[type="checkbox"]') !== null`, &checkboxExists),
	); err != nil {
		t.Fatalf("Failed to load page: %v", err)
	}

	t.Logf("Initial state - Has interactive block: %v, Has 'Connecting...': %v, Todo count: %d, Has checkbox: %v",
		hasInteractiveBlock, hasConnectingMessage, initialTodoCount, checkboxExists)

	// ASSERTIONS: Initial state
	if hasConnectingMessage {
		t.Errorf("Page %s still shows 'Connecting...' - WebSocket not working", pagePath)
	}

	if !hasInteractiveBlock {
		t.Errorf("Page %s has no interactive block element", pagePath)
	}

	if initialTodoCount == 0 {
		t.Errorf("Page %s has no initial todos - state not rendering", pagePath)
	}

	// TEST 1: Click a checkbox to toggle a todo
	if checkboxExists {
		t.Logf("\n=== Testing Checkbox Toggle ===")

		var wasChecked bool
		var isCheckedAfter bool

		err := chromedp.Run(ctx,
			// Get initial checked state
			chromedp.Evaluate(`document.querySelector('input[type="checkbox"]').checked`, &wasChecked),

			// Click the checkbox
			chromedp.Click(`input[type="checkbox"]`, chromedp.NodeVisible),
			chromedp.Sleep(2*time.Second), // Wait for WebSocket round-trip

			// Get new checked state
			chromedp.Evaluate(`document.querySelector('input[type="checkbox"]').checked`, &isCheckedAfter),
		)

		if err != nil {
			t.Errorf("Failed to toggle checkbox: %v", err)
		} else {
			t.Logf("Checkbox - Was checked: %v, Is checked after: %v", wasChecked, isCheckedAfter)

			// Checkbox state should have changed
			if wasChecked == isCheckedAfter {
				t.Errorf("Checkbox did not toggle - was %v, still %v (interaction not working!)", wasChecked, isCheckedAfter)
			} else {
				t.Logf("✅ Checkbox toggled successfully!")
			}
		}
	}

	// TEST 2: Try to add a new todo via form submission
	t.Logf("\n=== Testing Form Submission ===")

	var finalTodoCount int
	testTodoText := fmt.Sprintf("E2E Test Todo %d", time.Now().Unix())

	err := chromedp.Run(ctx,
		// Fill in the form
		chromedp.SendKeys(`input[name="text"]`, testTodoText),
		chromedp.Sleep(500*time.Millisecond),

		// Submit the form
		chromedp.Click(`button[type="submit"]`, chromedp.NodeVisible),
		chromedp.Sleep(3*time.Second), // Wait for WebSocket round-trip and re-render

		// Count todos again
		chromedp.Evaluate(`document.querySelectorAll('.todo-item').length`, &finalTodoCount),
	)

	if err != nil {
		t.Errorf("Failed to submit form: %v", err)
	} else {
		t.Logf("Todo count - Initial: %d, After adding: %d", initialTodoCount, finalTodoCount)

		// Should have one more todo
		if finalTodoCount != initialTodoCount+1 {
			t.Errorf("Form submission did not add todo - expected %d todos, got %d", initialTodoCount+1, finalTodoCount)
		} else {
			t.Logf("✅ Form submission worked! Todo added successfully")
		}
	}

	// Log all captured logs at the end
	t.Logf("\n=== All Console Logs ===")
	for _, log := range consoleLogs {
		t.Logf("%s", log)
	}

	t.Logf("\n=== All WebSocket Messages ===")
	for _, msg := range wsMessages {
		t.Logf("%s", msg)
	}

	// Check for data in WebSocket messages
	hasDataInWS := false
	for _, msg := range wsMessages {
		if strings.Contains(msg, `"data":{`) && !strings.Contains(msg, `"data":{}`) {
			hasDataInWS = true
			break
		}
	}

	if !hasDataInWS && checkboxExists {
		t.Errorf("WebSocket messages contain empty data - lvt-data-* extraction not working!")
	}
}
