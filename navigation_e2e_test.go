package livepage_test

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/chromedp/chromedp"
	"github.com/livetemplate/livepage/internal/config"
	"github.com/livetemplate/livepage/internal/server"
)

func TestMultiPageNavigation(t *testing.T) {
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

	// Start HTTP server
	port := 9191
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

	// Create Chrome context
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("headless", true),
		chromedp.Flag("disable-gpu", true),
		chromedp.Flag("no-sandbox", true),
	)

	allocCtx, allocCancel := chromedp.NewExecAllocator(context.Background(), opts...)
	defer allocCancel()

	baseURL := fmt.Sprintf("http://%s", addr)

	// Helper function to create a new context for each test
	newTestContext := func() (context.Context, context.CancelFunc) {
		ctx, cancel := chromedp.NewContext(allocCtx)
		ctx, cancel2 := context.WithTimeout(ctx, 10*time.Second)
		return ctx, func() {
			cancel2()
			cancel()
		}
	}

	// Test 1: Home page loads
	t.Run("HomePage", func(t *testing.T) {
		ctx, cancel := newTestContext()
		defer cancel()

		var title string
		err := chromedp.Run(ctx,
			chromedp.Navigate(baseURL+"/"),
			chromedp.Sleep(1*time.Second),
			chromedp.Title(&title),
		)
		if err != nil {
			t.Fatalf("Failed to load home page: %v", err)
		}

		if title == "" {
			t.Error("Page title is empty")
		}
		t.Logf("Home page title: %s", title)
	})

	// Test 2: Navigation sidebar exists
	t.Run("SidebarExists", func(t *testing.T) {
		ctx, cancel := newTestContext()
		defer cancel()

		var sidebarExists bool
		err := chromedp.Run(ctx,
			chromedp.Navigate(baseURL+"/"),
			chromedp.Sleep(1*time.Second),
			chromedp.Evaluate(`document.querySelector('.livepage-nav-sidebar') !== null`, &sidebarExists),
		)
		if err != nil {
			t.Fatalf("Failed to check sidebar: %v", err)
		}

		if !sidebarExists {
			t.Error("Navigation sidebar does not exist")
		}
	})

	// Test 3: Navigation sections are present
	t.Run("NavigationSections", func(t *testing.T) {
		ctx, cancel := newTestContext()
		defer cancel()

		var sectionCount int
		err := chromedp.Run(ctx,
			chromedp.Navigate(baseURL+"/"),
			chromedp.Sleep(1*time.Second),
			chromedp.Evaluate(`document.querySelectorAll('.nav-section').length`, &sectionCount),
		)
		if err != nil {
			t.Fatalf("Failed to count nav sections: %v", err)
		}

		if sectionCount == 0 {
			t.Error("No navigation sections found")
		}
		t.Logf("Found %d navigation sections", sectionCount)
	})

	// Test 5: Active page is highlighted in sidebar
	t.Run("ActivePageHighlight", func(t *testing.T) {
		ctx, cancel := newTestContext()
		defer cancel()

		var hasActiveClass bool
		err := chromedp.Run(ctx,
			chromedp.Navigate(baseURL+"/getting-started/intro"),
			chromedp.Sleep(1*time.Second),
			chromedp.Evaluate(`document.querySelector('.nav-pages li a.active') !== null`, &hasActiveClass),
		)
		if err != nil {
			t.Fatalf("Failed to check active page: %v", err)
		}

		if !hasActiveClass {
			t.Error("Active page is not highlighted in navigation")
		}
	})

	// Test 6: Breadcrumbs exist on non-home pages
	t.Run("Breadcrumbs", func(t *testing.T) {
		ctx, cancel := newTestContext()
		defer cancel()

		var breadcrumbsExist bool
		var breadcrumbCount int

		err := chromedp.Run(ctx,
			chromedp.Navigate(baseURL+"/getting-started/intro"),
			chromedp.Sleep(1*time.Second),
			chromedp.Evaluate(`document.querySelector('.breadcrumbs') !== null`, &breadcrumbsExist),
			chromedp.Evaluate(`document.querySelectorAll('.breadcrumbs li').length`, &breadcrumbCount),
		)
		if err != nil {
			t.Fatalf("Failed to check breadcrumbs: %v", err)
		}

		if !breadcrumbsExist {
			t.Error("Breadcrumbs do not exist on page")
		}

		if breadcrumbCount == 0 {
			t.Error("No breadcrumb items found")
		}
		t.Logf("Found %d breadcrumb items", breadcrumbCount)
	})

	// Test 7: Prev/Next navigation exists
	t.Run("PrevNextNavigation", func(t *testing.T) {
		ctx, cancel := newTestContext()
		defer cancel()

		var pageNavExists bool
		err := chromedp.Run(ctx,
			chromedp.Navigate(baseURL+"/getting-started/intro"),
			chromedp.Sleep(1*time.Second),
			chromedp.Evaluate(`document.querySelector('.page-nav') !== null`, &pageNavExists),
		)
		if err != nil {
			t.Fatalf("Failed to check prev/next nav: %v", err)
		}

		if !pageNavExists {
			t.Error("Prev/Next navigation does not exist")
		}
	})

	// Test 8: Next button has correct href
	t.Run("NextButtonHref", func(t *testing.T) {
		ctx, cancel := newTestContext()
		defer cancel()

		var nextButtonExists bool
		var href string

		err := chromedp.Run(ctx,
			chromedp.Navigate(baseURL+"/getting-started/intro"),
			chromedp.Sleep(500*time.Millisecond),
			chromedp.Evaluate(`document.querySelector('.page-nav-next') !== null`, &nextButtonExists),
		)
		if err != nil {
			t.Fatalf("Failed to check next button: %v", err)
		}

		if !nextButtonExists {
			t.Log("No next button on this page (might be last page)")
			return
		}

		err = chromedp.Run(ctx,
			chromedp.AttributeValue(`.page-nav-next`, "href", &href, nil),
		)
		if err != nil {
			t.Fatalf("Failed to get next button href: %v", err)
		}

		if href == "" {
			t.Error("Next button href is empty")
		}
		t.Logf("Next button href: %s", href)

		// Verify the target page loads
		var pageTitle string
		err = chromedp.Run(ctx,
			chromedp.Navigate(baseURL+href),
			chromedp.Sleep(500*time.Millisecond),
			chromedp.Title(&pageTitle),
		)
		if err != nil {
			t.Fatalf("Failed to navigate to next page %s: %v", href, err)
		}

		if pageTitle == "" {
			t.Error("Next page has empty title")
		}
		t.Logf("Successfully navigated to next page (title: %s)", pageTitle)
	})

	// Test 10: All configured pages are accessible
	t.Run("AllPagesAccessible", func(t *testing.T) {
		ctx, cancel := newTestContext()
		defer cancel()

		pages := []string{
			"/",
			"/getting-started/intro",
			"/getting-started/installation",
			"/guides/creating-pages",
			"/guides/configuration",
		}

		for _, page := range pages {
			var pageTitle string
			err := chromedp.Run(ctx,
				chromedp.Navigate(baseURL+page),
				chromedp.Sleep(500*time.Millisecond),
				chromedp.Title(&pageTitle),
			)
			if err != nil {
				t.Errorf("Failed to load page %s: %v", page, err)
				continue
			}

			if pageTitle == "" {
				t.Errorf("Page %s has empty title", page)
			}
			t.Logf("✓ Page %s loaded successfully (title: %s)", page, pageTitle)
		}
	})

	// Test 11: Site title appears in sidebar
	t.Run("SiteTitleInSidebar", func(t *testing.T) {
		ctx, cancel := newTestContext()
		defer cancel()

		var hasSiteTitle bool
		err := chromedp.Run(ctx,
			chromedp.Navigate(baseURL+"/"),
			chromedp.Sleep(1*time.Second),
			chromedp.Evaluate(`document.querySelector('.nav-header h2') !== null`, &hasSiteTitle),
		)
		if err != nil {
			t.Fatalf("Failed to check site title: %v", err)
		}

		if !hasSiteTitle {
			t.Error("Site title not found in sidebar header")
		}
	})

	// Test 12: Theme toggle works (should not break navigation)
	t.Run("ThemeToggleCompatibility", func(t *testing.T) {
		ctx, cancel := newTestContext()
		defer cancel()

		var sidebarVisibleAfterThemeChange bool

		err := chromedp.Run(ctx,
			chromedp.Navigate(baseURL+"/"),
			chromedp.Sleep(1*time.Second),
			// Toggle theme
			chromedp.Click(`#theme-dark`, chromedp.NodeVisible),
			chromedp.Sleep(500*time.Millisecond),
			// Check sidebar still visible
			chromedp.Evaluate(`
				const sidebar = document.querySelector('.livepage-nav-sidebar');
				sidebar !== null && window.getComputedStyle(sidebar).display !== 'none'
			`, &sidebarVisibleAfterThemeChange),
		)
		if err != nil {
			t.Fatalf("Failed theme toggle test: %v", err)
		}

		if !sidebarVisibleAfterThemeChange {
			t.Error("Sidebar not visible after theme change")
		}
	})
}
