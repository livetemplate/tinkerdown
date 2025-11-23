package livepage_test

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/cdproto/runtime"
	"github.com/chromedp/chromedp"
)

// TestCounterTutorial tests the interactive counter tutorial end-to-end
func TestCounterTutorial(t *testing.T) {
	// Build livepage first
	buildCmd := exec.Command("go", "build", "-o", "livepage-test", "./cmd/livepage")
	if err := buildCmd.Run(); err != nil {
		t.Fatalf("Failed to build livepage: %v", err)
	}
	defer os.Remove("livepage-test")

	// Start the server
	serverCmd := exec.Command("./livepage-test", "serve", "examples/counter")

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
	time.Sleep(2 * time.Second)

	// Setup chromedp
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("headless", true),
		chromedp.Flag("disable-gpu", true),
		chromedp.Flag("no-sandbox", true),
	)

	allocCtx, cancel := chromedp.NewExecAllocator(context.Background(), opts...)
	defer cancel()

	ctx, cancel := chromedp.NewContext(allocCtx, chromedp.WithLogf(log.Printf))
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
		switch ev := ev.(type) {
		case *network.EventWebSocketFrameReceived:
			t.Logf("[WebSocket <-] %s", ev.Response.PayloadData)
		case *network.EventWebSocketFrameSent:
			t.Logf("[WebSocket ->] %s", ev.Response.PayloadData)
		case *network.EventWebSocketCreated:
			t.Logf("[WebSocket] Connection created: %s", ev.URL)
		case *network.EventWebSocketHandshakeResponseReceived:
			t.Logf("[WebSocket] Handshake completed")
		}
	})

	// Enable network events
	if err := chromedp.Run(ctx, network.Enable()); err != nil {
		t.Fatalf("Failed to enable network: %v", err)
	}

	// Run the test
	var htmlContent string
	var interactiveBlockExists bool
	var blockContent string

	if err := chromedp.Run(ctx,
		chromedp.Navigate("http://localhost:8080/"),
		chromedp.Sleep(3*time.Second), // Wait for page load and WebSocket connection

		// Capture rendered HTML
		chromedp.OuterHTML("body", &htmlContent),

		// Check if interactive block exists
		chromedp.Evaluate(`document.querySelector('[data-livepage-block][data-block-type="lvt"]') !== null`, &interactiveBlockExists),

		// Check block content
		chromedp.Text(`[data-livepage-block][data-block-type="lvt"]`, &blockContent),
	); err != nil {
		t.Fatalf("Failed to navigate: %v", err)
	}

	t.Logf("Interactive block exists: %v", interactiveBlockExists)
	t.Logf("Block content: %s", blockContent)

	// Dump HTML for debugging
	htmlFile := "/tmp/livepage-test.html"
	if err := os.WriteFile(htmlFile, []byte(htmlContent), 0644); err != nil {
		t.Logf("Warning: Failed to write HTML: %v", err)
	} else {
		t.Logf("Rendered HTML saved to: %s", htmlFile)
	}

	// Verify interactive block is present
	if !interactiveBlockExists {
		t.Fatal("Interactive block not found in page")
	}

	// Wait a bit more for WebSocket updates
	time.Sleep(2 * time.Second)

	// Check if buttons are rendered
	var hasButtons bool
	var counterText string

	if err := chromedp.Run(ctx,
		// Re-read content after WebSocket updates
		chromedp.Text(`[data-livepage-block][data-block-type="lvt"]`, &counterText),

		// Check for increment button
		chromedp.Evaluate(`document.querySelector('[data-livepage-block][data-block-type="lvt"] button[lvt-click="increment"]') !== null`, &hasButtons),
	); err != nil {
		t.Fatalf("Failed to check buttons: %v", err)
	}

	t.Logf("Has buttons: %v", hasButtons)
	t.Logf("Counter text: %s", counterText)

	if !hasButtons {
		t.Fatal("Interactive buttons not rendered - WebSocket connection may have failed. Check logs above.")
	}

	// Test button interaction - Click increment
	if err := chromedp.Run(ctx,
		chromedp.Click(`[data-livepage-block][data-block-type="lvt"] button[lvt-click="increment"]`, chromedp.ByQuery),
		chromedp.Sleep(1*time.Second), // Wait for WebSocket update
		chromedp.Text(`[data-livepage-block][data-block-type="lvt"]`, &counterText),
	); err != nil {
		t.Fatalf("Failed to click increment: %v", err)
	}

	t.Logf("Counter after increment: %s", counterText)

	// Verify counter incremented - check that counter shows "1" (not "0")
	// The counter display just shows the number, not "Count: X"
	if !strings.Contains(counterText, "1") || strings.HasPrefix(strings.TrimSpace(counterText), "0") {
		t.Fatalf("Counter did not increment. Expected counter to show '1', got: %s", counterText)
	}

	t.Log("✓ Counter tutorial working correctly!")
}
