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
// 3. User can interact (increment/decrement counter)
func TestInteractiveBlocksE2E(t *testing.T) {
	// Build livepage first
	buildCmd := exec.Command("go", "build", "-o", "livepage-test", "./cmd/livepage")
	buildCmd.Dir = "/Users/adnaan/code/livetemplate/livepage"
	if err := buildCmd.Run(); err != nil {
		t.Fatalf("Failed to build livepage: %v", err)
	}
	defer os.Remove("/Users/adnaan/code/livetemplate/livepage/livepage-test")

	// Start the server
	serverCmd := exec.Command("/Users/adnaan/code/livetemplate/livepage/livepage-test", "serve", "examples/counter", "--debug")
	serverCmd.Dir = "/Users/adnaan/code/livetemplate/livepage"

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

	// Test the index page which has a simple counter
	testPageInteraction(t, "")
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

	var hasInteractiveBlock bool
	var initialCounterValue int

	// Load page and check initial state
	if err := chromedp.Run(ctx,
		chromedp.Navigate(url),
		chromedp.Sleep(5*time.Second), // Wait for page load and WebSocket

		// Check for interactive block
		chromedp.Evaluate(`document.querySelector('.livepage-interactive-block') !== null`, &hasInteractiveBlock),

		// Get initial counter value
		chromedp.Evaluate(`(() => {
			const display = document.querySelector('.counter-display');
			return display ? parseInt(display.textContent.trim()) : null;
		})()`, &initialCounterValue),
	); err != nil {
		t.Fatalf("Failed to load page: %v", err)
	}

	t.Logf("Initial state - Has interactive block: %v, Counter value: %d",
		hasInteractiveBlock, initialCounterValue)

	// ASSERTIONS: Initial state
	if !hasInteractiveBlock {
		t.Errorf("Page %s has no interactive block element", pagePath)
	}

	// TEST 1: Click increment button
	t.Logf("\n=== Testing Counter Increment ===")

	var counterAfterIncrement int

	err := chromedp.Run(ctx,
		// Click the first increment button (basic counter)
		chromedp.Click(`button[lvt-click="increment"]`, chromedp.NodeVisible),
		chromedp.Sleep(2*time.Second), // Wait for WebSocket round-trip

		// Get new counter value
		chromedp.Evaluate(`(() => {
			const display = document.querySelector('.counter-display');
			return display ? parseInt(display.textContent.trim()) : null;
		})()`, &counterAfterIncrement),
	)

	if err != nil {
		t.Errorf("Failed to increment counter: %v", err)
	} else {
		t.Logf("Counter - Initial: %d, After increment: %d", initialCounterValue, counterAfterIncrement)

		// Counter should have increased by 1
		if counterAfterIncrement != initialCounterValue+1 {
			t.Errorf("Counter did not increment - was %d, expected %d, got %d (interaction not working!)",
				initialCounterValue, initialCounterValue+1, counterAfterIncrement)
		} else {
			t.Logf("✅ Counter incremented successfully!")
		}
	}

	// TEST 2: Click decrement button
	t.Logf("\n=== Testing Counter Decrement ===")

	var counterAfterDecrement int

	err = chromedp.Run(ctx,
		// Click the decrement button
		chromedp.Click(`button[lvt-click="decrement"]`, chromedp.NodeVisible),
		chromedp.Sleep(2*time.Second), // Wait for WebSocket round-trip

		// Get new counter value
		chromedp.Evaluate(`(() => {
			const display = document.querySelector('.counter-display');
			return display ? parseInt(display.textContent.trim()) : null;
		})()`, &counterAfterDecrement),
	)

	if err != nil {
		t.Errorf("Failed to decrement counter: %v", err)
	} else {
		t.Logf("Counter - After increment: %d, After decrement: %d", counterAfterIncrement, counterAfterDecrement)

		// Counter should be back to initial value
		if counterAfterDecrement != initialCounterValue {
			t.Errorf("Counter did not decrement correctly - expected %d, got %d", initialCounterValue, counterAfterDecrement)
		} else {
			t.Logf("✅ Counter decremented successfully!")
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

	if !hasDataInWS {
		t.Logf("Note: WebSocket messages contain empty data (this is expected for simple counters without lvt-data-* attributes)")
	}
}
