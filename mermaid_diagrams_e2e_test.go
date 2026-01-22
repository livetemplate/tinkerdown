//go:build !ci

package tinkerdown

import (
	"context"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/chromedp/chromedp"
)

// TestMermaidDiagramsRendering verifies that Mermaid.js diagrams render correctly
func TestMermaidDiagramsRendering(t *testing.T) {
	// Start the server
	serverCmd := exec.Command("./tinkerdown", "serve", "examples/mermaid-diagrams-test", "--port", "8090")
	serverCmd.Stdout = os.Stdout
	serverCmd.Stderr = os.Stderr

	if err := serverCmd.Start(); err != nil {
		t.Fatalf("Failed to start server: %v", err)
	}
	defer func() {
		if serverCmd.Process != nil {
			serverCmd.Process.Kill()
		}
	}()

	// Wait for server to start
	time.Sleep(3 * time.Second)

	// Create chrome context
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("headless", true),
		chromedp.Flag("disable-gpu", true),
		chromedp.Flag("no-sandbox", true),
	)

	allocCtx, cancel := chromedp.NewExecAllocator(context.Background(), opts...)
	defer cancel()

	ctx, cancel := chromedp.NewContext(allocCtx)
	defer cancel()

	// Set timeout
	ctx, cancel = context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	var html string
	var mermaidDivCount int
	var svgCount int

	err := chromedp.Run(ctx,
		chromedp.Navigate("http://localhost:8090/"),
		chromedp.Sleep(2*time.Second), // Wait for page to load
		chromedp.Sleep(2*time.Second), // Wait for Mermaid to render

		// Get the full HTML to inspect
		chromedp.OuterHTML("html", &html),

		// Count how many mermaid div elements exist
		chromedp.Evaluate(`document.querySelectorAll('div.mermaid').length`, &mermaidDivCount),

		// Count how many SVG elements exist (rendered diagrams)
		chromedp.Evaluate(`document.querySelectorAll('svg').length`, &svgCount),
	)

	if err != nil {
		t.Fatalf("Failed to run chromedp: %v", err)
	}

	// Log output for debugging
	t.Logf("Mermaid div count: %d", mermaidDivCount)
	t.Logf("SVG count: %d", svgCount)

	// Save HTML for inspection
	if err := os.WriteFile("/tmp/mermaid-test.html", []byte(html), 0644); err != nil {
		t.Logf("Warning: Could not save HTML: %v", err)
	}

	// Verify Mermaid.js script is loaded (embedded at /assets/mermaid.js)
	if !strings.Contains(html, "/assets/mermaid.js") {
		t.Error("Mermaid.js script not found in HTML")
	}

	// Verify Mermaid initialization code exists
	if !strings.Contains(html, "mermaid.initialize") {
		t.Error("Mermaid initialization code not found in HTML")
	}

	// Verify SVG elements were created (diagrams rendered)
	// Note: Mermaid.js transforms div.mermaid elements, so we check SVGs which are the rendered output
	if svgCount < 3 {
		t.Errorf("Expected at least 3 SVG diagrams to render, got %d", svgCount)
	}

	t.Logf("✓ Mermaid diagrams rendered successfully: %d divs, %d SVGs", mermaidDivCount, svgCount)
}
