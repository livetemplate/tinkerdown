package livepage_test

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"net/http/httptest"

	"github.com/chromedp/cdproto/runtime"
	"github.com/chromedp/chromedp"
	"github.com/livetemplate/livepage/internal/config"
	"github.com/livetemplate/livepage/internal/server"
)

// TestLvtSourceJSON tests the lvt-source functionality with JSON files
func TestLvtSourceJSON(t *testing.T) {
	// Load config from test example
	cfg, err := config.LoadFromDir("examples/lvt-source-file-test")
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Verify sources are configured
	if cfg.Sources == nil {
		t.Fatal("No sources configured in livepage.yaml")
	}
	userSource, ok := cfg.Sources["users"]
	if !ok {
		t.Fatal("users source not found in config")
	}
	if userSource.Type != "json" {
		t.Fatalf("Expected json source type, got: %s", userSource.Type)
	}
	t.Logf("Source config: type=%s, file=%s", userSource.Type, userSource.File)

	// Create test server
	srv := server.NewWithConfig("examples/lvt-source-file-test", cfg)
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

	// Store console logs for debugging
	var consoleLogs []string
	chromedp.ListenTarget(ctx, func(ev interface{}) {
		if ev, ok := ev.(*runtime.EventConsoleAPICalled); ok {
			for _, arg := range ev.Args {
				consoleLogs = append(consoleLogs, fmt.Sprintf("[Console] %s", arg.Value))
			}
		}
	})

	t.Logf("Test server URL: %s", ts.URL)

	// Test 1: Navigate and wait for WebSocket to render content
	var hasUserTable bool
	err = chromedp.Run(ctx,
		chromedp.Navigate(ts.URL+"/"),
		chromedp.Sleep(5*time.Second),
		chromedp.Evaluate(`document.querySelector('[lvt-source="users"] table') !== null`, &hasUserTable),
	)
	if err != nil {
		t.Fatalf("Failed to navigate: %v", err)
	}

	if !hasUserTable {
		var htmlContent string
		chromedp.Run(ctx, chromedp.OuterHTML("html", &htmlContent))
		t.Logf("HTML (first 3000 chars): %s", htmlContent[:min(3000, len(htmlContent))])
		t.Logf("Console logs: %v", consoleLogs)
		t.Fatal("User table was not rendered - JSON source failed")
	}
	t.Log("User table rendered from JSON file")

	// Test 2: Verify user data is rendered (using semantic selectors)
	var userRowCount int
	var firstUserName string
	err = chromedp.Run(ctx,
		chromedp.Evaluate(`document.querySelectorAll('[lvt-source="users"] tbody tr').length`, &userRowCount),
		chromedp.Evaluate(`
			(() => {
				const row = document.querySelector('[lvt-source="users"] tbody tr');
				const nameCell = row ? row.querySelector('td:nth-child(2)') : null;
				return nameCell ? nameCell.textContent : '';
			})()
		`, &firstUserName),
	)
	if err != nil {
		t.Fatalf("Failed to check user data: %v", err)
	}

	if userRowCount != 3 {
		t.Fatalf("Expected 3 users from JSON, got %d", userRowCount)
	}
	t.Log("Correct number of users rendered from JSON")

	if !strings.Contains(firstUserName, "Alice") {
		t.Fatalf("Expected first user name to be 'Alice', got: %s", firstUserName)
	}
	t.Log("First user name is Alice")

	t.Log("JSON file source test passed!")
}

// TestLvtSourceCSV tests the lvt-source functionality with CSV files
func TestLvtSourceCSV(t *testing.T) {
	// Load config from test example
	cfg, err := config.LoadFromDir("examples/lvt-source-file-test")
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Verify products source is configured
	productSource, ok := cfg.Sources["products"]
	if !ok {
		t.Fatal("products source not found in config")
	}
	if productSource.Type != "csv" {
		t.Fatalf("Expected csv source type, got: %s", productSource.Type)
	}
	t.Logf("Source config: type=%s, file=%s", productSource.Type, productSource.File)

	// Create test server
	srv := server.NewWithConfig("examples/lvt-source-file-test", cfg)
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

	// Store console logs for debugging
	var consoleLogs []string
	chromedp.ListenTarget(ctx, func(ev interface{}) {
		if ev, ok := ev.(*runtime.EventConsoleAPICalled); ok {
			for _, arg := range ev.Args {
				consoleLogs = append(consoleLogs, fmt.Sprintf("[Console] %s", arg.Value))
			}
		}
	})

	t.Logf("Test server URL: %s", ts.URL)

	// Test 1: Navigate and wait for WebSocket to render content
	var hasProductTable bool
	err = chromedp.Run(ctx,
		chromedp.Navigate(ts.URL+"/"),
		chromedp.Sleep(5*time.Second),
		chromedp.Evaluate(`document.querySelector('[lvt-source="products"] table') !== null`, &hasProductTable),
	)
	if err != nil {
		t.Fatalf("Failed to navigate: %v", err)
	}

	if !hasProductTable {
		var htmlContent string
		chromedp.Run(ctx, chromedp.OuterHTML("html", &htmlContent))
		t.Logf("HTML (first 3000 chars): %s", htmlContent[:min(3000, len(htmlContent))])
		t.Logf("Console logs: %v", consoleLogs)
		t.Fatal("Product table was not rendered - CSV source failed")
	}
	t.Log("Product table rendered from CSV file")

	// Test 2: Verify product data is rendered (using semantic selectors)
	var productRowCount int
	var firstProductName string
	var firstProductPrice string
	err = chromedp.Run(ctx,
		chromedp.Evaluate(`document.querySelectorAll('[lvt-source="products"] tbody tr').length`, &productRowCount),
		chromedp.Evaluate(`
			(() => {
				const row = document.querySelector('[lvt-source="products"] tbody tr');
				const nameCell = row ? row.querySelector('td:nth-child(2)') : null;
				return nameCell ? nameCell.textContent : '';
			})()
		`, &firstProductName),
		chromedp.Evaluate(`
			(() => {
				const row = document.querySelector('[lvt-source="products"] tbody tr');
				const priceCell = row ? row.querySelector('td:nth-child(3)') : null;
				return priceCell ? priceCell.textContent : '';
			})()
		`, &firstProductPrice),
	)
	if err != nil {
		t.Fatalf("Failed to check product data: %v", err)
	}

	if productRowCount != 3 {
		t.Fatalf("Expected 3 products from CSV, got %d", productRowCount)
	}
	t.Log("Correct number of products rendered from CSV")

	if !strings.Contains(firstProductName, "Widget") {
		t.Fatalf("Expected first product name to be 'Widget', got: %s", firstProductName)
	}
	t.Log("First product name is Widget")

	if !strings.Contains(firstProductPrice, "19.99") {
		t.Fatalf("Expected first product price to contain '19.99', got: %s", firstProductPrice)
	}
	t.Log("First product price is $19.99")

	t.Log("CSV file source test passed!")
}
