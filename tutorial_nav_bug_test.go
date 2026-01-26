package tinkerdown_test

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/chromedp/chromedp"
	"github.com/livetemplate/tinkerdown/internal/config"
	"github.com/livetemplate/tinkerdown/internal/server"
)

// Note: context import is still needed for http server shutdown

// TestTutorialNavigationDoesNotOverrideSiteNavigation verifies that the
// TutorialNavigation client-side JavaScript does not create tutorial navigation
// when site navigation already exists (i.e., in site mode).
//
// Bug: The TutorialNavigation class was automatically creating tutorial-style
// navigation (numbered sidebar with "CONTENTS" header and step counter) whenever
// it found H2 headings, without checking if server-rendered site navigation exists.
//
// Fix: Added check for '.tinkerdown-nav-sidebar' element before creating tutorial nav.
func TestTutorialNavigationDoesNotOverrideSiteNavigation(t *testing.T) {
	// Setup: Start server with docs-site example
	docsDir := "examples/docs-site"
	if _, err := os.Stat(docsDir); os.IsNotExist(err) {
		t.Skipf("Skipping test: %s directory not found", docsDir)
	}

	// Load config
	cfg, err := config.LoadFromDir(docsDir)
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Verify site mode
	if !cfg.IsSiteMode() {
		t.Fatal("Expected site mode, got tutorial mode")
	}

	// Create server
	srv := server.NewWithConfig(docsDir, cfg)
	if err := srv.Discover(); err != nil {
		t.Fatalf("Failed to discover pages: %v", err)
	}

	// Start HTTP server on a different port to avoid conflicts
	port := 9292
	addr := fmt.Sprintf("localhost:%d", port)
	httpServer := &http.Server{
		Addr:    addr,
		Handler: srv,
	}

	go func() {
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			t.Errorf("Server error: %v", err)
		}
	}()

	// Wait for server to start
	time.Sleep(500 * time.Millisecond)

	// Ensure server cleanup
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		httpServer.Shutdown(ctx)
	}()

	// Setup Docker Chrome
	chromeCtx, chromeCleanup := SetupDockerChrome(t, 15*time.Second)
	defer chromeCleanup()

	ctx := chromeCtx.Context

	// Convert URL for Docker Chrome access
	baseURL := ConvertURLForDockerChrome(fmt.Sprintf("http://%s", addr))

	// Test: Verify site navigation exists and tutorial navigation does NOT exist
	var siteNavExists bool
	var tutorialNavElements int
	var hasContentsHeader bool
	var hasStepCounter bool

	err = chromedp.Run(ctx,
		// Navigate to home page
		chromedp.Navigate(baseURL+"/"),
		chromedp.Sleep(2*time.Second), // Give time for JavaScript to run

		// Check for site navigation sidebar
		chromedp.Evaluate(`document.querySelector('.tinkerdown-nav-sidebar') !== null`, &siteNavExists),

		// Check for tutorial navigation elements
		chromedp.Evaluate(`document.querySelectorAll('.nav-sidebar-steps').length`, &tutorialNavElements),
		chromedp.Evaluate(`document.querySelector('.nav-sidebar-header h3')?.textContent === 'Contents'`, &hasContentsHeader),
		chromedp.Evaluate(`document.querySelector('.nav-progress')?.textContent?.includes('Step') || false`, &hasStepCounter),
	)

	if err != nil {
		t.Fatalf("Failed to run test: %v", err)
	}

	// Assertions
	if !siteNavExists {
		t.Error("Site navigation sidebar does not exist")
	}

	if tutorialNavElements > 0 {
		t.Errorf("Tutorial navigation elements found (%d), should be 0 in site mode", tutorialNavElements)
	}

	if hasContentsHeader {
		t.Error("Found 'Contents' header from tutorial navigation, should not exist in site mode")
	}

	if hasStepCounter {
		t.Error("Found step counter from tutorial navigation, should not exist in site mode")
	}

	// Verify site navigation structure (should have sections and pages)
	var hasSections bool
	var hasPages bool
	err = chromedp.Run(ctx,
		chromedp.Evaluate(`document.querySelectorAll('.nav-section').length > 0`, &hasSections),
		chromedp.Evaluate(`document.querySelectorAll('.nav-pages li').length > 0`, &hasPages),
	)

	if err != nil {
		t.Fatalf("Failed to check site navigation structure: %v", err)
	}

	if !hasSections {
		t.Error("Site navigation should have sections")
	}

	if !hasPages {
		t.Error("Site navigation should have pages")
	}

	t.Log("✓ Site navigation exists without tutorial navigation interference")
}
