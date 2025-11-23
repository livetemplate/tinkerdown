package e2e

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/chromedp/cdproto/cdp"
	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/cdproto/runtime"
	"github.com/chromedp/chromedp"
)

// TestCounterTutorial tests the interactive counter tutorial end-to-end
func TestCounterTutorial(t *testing.T) {
	// Start the server
	serverCmd := exec.Command("../../livepage", "serve", "../../examples/counter")
	serverCmd.Dir = "."

	// Capture server output
	serverOut := &strings.Builder{}
	serverCmd.Stdout = serverOut
	serverCmd.Stderr = serverOut

	if err := serverCmd.Start(); err != nil {
		t.Fatalf("Failed to start server: %v", err)
	}
	defer serverCmd.Process.Kill()

	// Wait for server to be ready
	time.Sleep(2 * time.Second)

	// Setup chromedp
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("headless", true),
		chromedp.Flag("disable-gpu", true),
		chromedp.Flag("no-sandbox", true),
	)

	allocCtx, cancel := chromedp.NewExecAllocator(context.Background(), opts...)
	defer cancel()

	ctx, cancel := chromedp.NewContext(allocCtx)
	defer cancel()

	// Capture browser console logs
	chromedp.ListenTarget(ctx, func(ev interface{}) {
		switch ev := ev.(type) {
		case *runtime.EventConsoleAPICalled:
			args := make([]string, len(ev.Args))
			for i, arg := range ev.Args {
				args[i] = fmt.Sprintf("%v", arg.Value)
			}
			t.Logf("[Browser Console] %s: %s", ev.Type, strings.Join(args, " "))
		case *runtime.EventExceptionThrown:
			t.Logf("[Browser Error] %s", ev.ExceptionDetails.Text)
		}
	})

	// Capture WebSocket messages
	chromedp.ListenTarget(ctx, func(ev interface{}) {
		if ev, ok := ev.(*network.EventWebSocketFrameReceived); ok {
			t.Logf("[WebSocket Received] %s", ev.Response.PayloadData)
		}
		if ev, ok := ev.(*network.EventWebSocketFrameSent); ok {
			t.Logf("[WebSocket Sent] %s", ev.Response.PayloadData)
		}
	})

	// Enable network events
	if err := chromedp.Run(ctx, network.Enable()); err != nil {
		t.Fatalf("Failed to enable network: %v", err)
	}

	// Run the test
	var htmlContent string
	var interactiveBlockExists bool
	var connectingText string

	if err := chromedp.Run(ctx,
		chromedp.Navigate("http://localhost:8080/"),
		chromedp.Sleep(2*time.Second), // Wait for page load and WebSocket connection

		// Capture rendered HTML
		chromedp.OuterHTML("body", &htmlContent),

		// Check if interactive block exists
		chromedp.Evaluate(`document.querySelector('[data-livepage-block][data-block-type="lvt"]') !== null`, &interactiveBlockExists),

		// Check if we see "Connecting..." text
		chromedp.Text('[data-livepage-block][data-block-type="lvt"]', &connectingText),
	); err != nil {
		t.Fatalf("Failed to navigate: %v", err)
	}

	t.Logf("Interactive block exists: %v", interactiveBlockExists)
	t.Logf("Interactive block text: %s", connectingText)

	// Dump HTML for debugging
	htmlFile := "/tmp/livepage-test.html"
	if err := os.WriteFile(htmlFile, []byte(htmlContent), 0644); err != nil {
		t.Logf("Warning: Failed to write HTML: %v", err)
	} else {
		t.Logf("Rendered HTML saved to: %s", htmlFile)
	}

	// Dump server logs
	t.Logf("\n=== Server Logs ===\n%s", serverOut.String())

	// Verify interactive block is present
	if !interactiveBlockExists {
		t.Fatal("Interactive block not found in page")
	}

	// Wait for WebSocket connection and initial state
	time.Sleep(1 * time.Second)

	// Check if buttons are rendered (should replace "Connecting...")
	var hasIncrementButton bool
	var counterValue string

	if err := chromedp.Run(ctx,
		// Check for increment button
		chromedp.Evaluate(`document.querySelector('[data-livepage-block][data-block-type="lvt"] button') !== null`, &hasIncrementButton),

		// Get counter value
		chromedp.Text('[data-livepage-block][data-block-type="lvt"]', &counterValue),
	); err != nil {
		t.Fatalf("Failed to check buttons: %v", err)
	}

	t.Logf("Has buttons: %v", hasIncrementButton)
	t.Logf("Counter content: %s", counterValue)

	if !hasIncrementButton {
		t.Fatal("Interactive buttons not rendered - WebSocket connection may have failed")
	}

	// Click increment button
	var incrementButton []*cdp.Node
	if err := chromedp.Run(ctx,
		chromedp.Nodes(`[data-livepage-block][data-block-type="lvt"] button[lvt-click="increment"]`, &incrementButton, chromedp.ByQuery),
	); err != nil {
		t.Fatalf("Failed to find increment button: %v", err)
	}

	if len(incrementButton) == 0 {
		t.Fatal("Increment button not found")
	}

	// Click the button and wait for update
	if err := chromedp.Run(ctx,
		chromedp.Click(`[data-livepage-block][data-block-type="lvt"] button[lvt-click="increment"]`, chromedp.ByQuery),
		chromedp.Sleep(500*time.Millisecond),
		chromedp.Text('[data-livepage-block][data-block-type="lvt"]', &counterValue),
	); err != nil {
		t.Fatalf("Failed to click increment: %v", err)
	}

	t.Logf("Counter after increment: %s", counterValue)

	// Verify counter incremented
	if !strings.Contains(counterValue, "1") {
		t.Fatalf("Counter did not increment. Expected to contain '1', got: %s", counterValue)
	}

	t.Log("✓ Counter tutorial working!")
}
