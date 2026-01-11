//go:build !ci

package tinkerdown_test

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"net/http/httptest"

	"github.com/chromedp/cdproto/runtime"
	"github.com/chromedp/chromedp"
	"github.com/livetemplate/tinkerdown/internal/server"
	_ "modernc.org/sqlite"
)

// TestExpenseTrackerExample tests the expense tracker example
func TestExpenseTrackerExample(t *testing.T) {
	exampleDir := "examples/expense-tracker"
	dbPath := filepath.Join(exampleDir, "expenses.db")

	// Remove any existing database
	os.Remove(dbPath)
	defer os.Remove(dbPath)

	// Create test server
	srv := server.New(exampleDir)
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

	// Navigate and wait for form to be visible
	err := chromedp.Run(ctx,
		chromedp.Navigate(ts.URL),
		chromedp.WaitVisible(`form[lvt-submit="Add"]`, chromedp.ByQuery),
	)
	if err != nil {
		t.Fatalf("Failed to navigate: %v\nConsole logs: %v", err, consoleLogs)
	}
	t.Log("Page loaded and form visible")

	// Wait for WebSocket connection
	time.Sleep(500 * time.Millisecond)

	// Add an expense
	today := time.Now().Format("2006-01-02")
	err = chromedp.Run(ctx,
		chromedp.SetValue(`input[name="date"]`, today, chromedp.ByQuery),
		chromedp.SetValue(`input[name="amount"]`, "42.50", chromedp.ByQuery),
		chromedp.SetValue(`select[name="category"]`, "Food", chromedp.ByQuery),
		chromedp.SetValue(`input[name="description"]`, "Test lunch", chromedp.ByQuery),
		chromedp.Click(`form[lvt-submit="Add"] button[type="submit"]`, chromedp.ByQuery),
		chromedp.Sleep(500*time.Millisecond),
	)
	if err != nil {
		t.Fatalf("Failed to add expense: %v", err)
	}
	t.Log("Added expense")

	// Verify expense appears in table
	var rowCount int
	err = chromedp.Run(ctx,
		chromedp.Evaluate(`document.querySelectorAll('tbody tr').length`, &rowCount),
	)
	if err != nil {
		t.Fatalf("Failed to count rows: %v", err)
	}
	if rowCount != 1 {
		t.Errorf("Expected 1 row after add, got %d", rowCount)
	}

	// Verify expense was stored in database
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM expenses").Scan(&count)
	if err != nil {
		t.Fatalf("Failed to query expenses: %v", err)
	}
	if count != 1 {
		t.Errorf("Expected 1 expense in database, got %d", count)
	}

	// Verify expense data
	var amount float64
	var category, description string
	err = db.QueryRow("SELECT amount, category, description FROM expenses LIMIT 1").Scan(&amount, &category, &description)
	if err != nil {
		t.Fatalf("Failed to read expense: %v", err)
	}
	if amount != 42.50 {
		t.Errorf("Expected amount 42.50, got %f", amount)
	}
	if category != "Food" {
		t.Errorf("Expected category 'Food', got %q", category)
	}
	if description != "Test lunch" {
		t.Errorf("Expected description 'Test lunch', got %q", description)
	}
	t.Log("Expense data verified in database")

	// Add another expense
	err = chromedp.Run(ctx,
		chromedp.SetValue(`input[name="date"]`, today, chromedp.ByQuery),
		chromedp.SetValue(`input[name="amount"]`, "15.00", chromedp.ByQuery),
		chromedp.SetValue(`select[name="category"]`, "Transport", chromedp.ByQuery),
		chromedp.SetValue(`input[name="description"]`, "Bus fare", chromedp.ByQuery),
		chromedp.Click(`form[lvt-submit="Add"] button[type="submit"]`, chromedp.ByQuery),
		chromedp.Sleep(500*time.Millisecond),
	)
	if err != nil {
		t.Fatalf("Failed to add second expense: %v", err)
	}

	// Verify 2 rows now
	err = chromedp.Run(ctx,
		chromedp.Evaluate(`document.querySelectorAll('tbody tr').length`, &rowCount),
	)
	if err != nil {
		t.Fatalf("Failed to count rows after second add: %v", err)
	}
	if rowCount != 2 {
		t.Errorf("Expected 2 rows, got %d", rowCount)
	}
	t.Log("Second expense added")

	// Delete one expense (first row is newest)
	err = chromedp.Run(ctx,
		chromedp.Click(`tbody tr:first-child button[lvt-click="Delete"]`, chromedp.ByQuery),
		chromedp.Sleep(500*time.Millisecond),
	)
	if err != nil {
		t.Fatalf("Failed to delete expense: %v", err)
	}

	// Verify 1 row remains
	err = chromedp.Run(ctx,
		chromedp.Evaluate(`document.querySelectorAll('tbody tr').length`, &rowCount),
	)
	if err != nil {
		t.Fatalf("Failed to count rows after delete: %v", err)
	}
	if rowCount != 1 {
		t.Errorf("Expected 1 row after delete, got %d", rowCount)
	}
	t.Log("Expense deleted")

	// Test Refresh button
	err = chromedp.Run(ctx,
		chromedp.Click(`button[lvt-click="Refresh"]`, chromedp.ByQuery),
		chromedp.Sleep(500*time.Millisecond),
	)
	if err != nil {
		t.Fatalf("Failed to refresh: %v", err)
	}

	// Verify still 1 row after refresh
	err = chromedp.Run(ctx,
		chromedp.Evaluate(`document.querySelectorAll('tbody tr').length`, &rowCount),
	)
	if err != nil {
		t.Fatalf("Failed to count rows after refresh: %v", err)
	}
	if rowCount != 1 {
		t.Errorf("Expected 1 row after refresh, got %d", rowCount)
	}
	t.Log("Refresh verified")

	t.Log("All expense tracker tests passed!")
}
