//go:build !ci

package tinkerdown

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/chromedp/chromedp"
)

// TestMermaidDiagramsRendering verifies that Mermaid.js diagrams render correctly
func TestMermaidDiagramsRendering(t *testing.T) {
	// Use a dynamic port to avoid conflicts
	port, err := getFreePort()
	if err != nil {
		t.Fatalf("Failed to get free port: %v", err)
	}

	// Start the server
	serverCmd := exec.Command("./tinkerdown", "serve", "examples/mermaid-diagrams-test", "--port", fmt.Sprintf("%d", port))
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

	// Wait for server to be ready with proper polling
	WaitForServer(t, fmt.Sprintf("http://localhost:%d", port), 30*time.Second)

	// Use Docker Chrome for reliable CI execution
	chromeCtx, cleanup := SetupDockerChrome(t, 60*time.Second)
	defer cleanup()

	ctx := chromeCtx.Context

	var html string
	var mermaidDivCount int
	var svgCount int

	// Use host.docker.internal for Docker Chrome to access host server
	url := GetChromeTestURL(port)

	err = chromedp.Run(ctx,
		chromedp.Navigate(url),
		// Wait for page to be fully loaded and Mermaid to render
		chromedp.WaitVisible(`svg`, chromedp.ByQuery),
		chromedp.Sleep(2*time.Second), // Extra time for Mermaid to render all diagrams

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

	t.Logf("Mermaid diagrams rendered successfully: %d divs, %d SVGs", mermaidDivCount, svgCount)
}
