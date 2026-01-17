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

// TestLvtSourceSQLite tests the lvt-source functionality with SQLite
// This test verifies that:
// 1. lvt-source="tasks" fetches data from the configured SQLite table
// 2. The data is rendered in the template
// 3. Add action inserts new records
// 4. Toggle action updates records
// 5. Delete action removes records
// 6. The Refresh action re-fetches data
func TestLvtSourceSQLite(t *testing.T) {
	exampleDir := "examples/lvt-source-sqlite-test"

	// Setup: create test database with initial data
	dbPath := filepath.Join(exampleDir, "tasks.db")
	setupSQLiteTestDB(t, dbPath)
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
			chromedp.Flag("no-sandbox", true),
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

	// Navigate and wait for WebSocket connection
	var wsConnected bool
	err := chromedp.Run(ctx,
		chromedp.Navigate(ts.URL),
		chromedp.WaitVisible(`[lvt-source="tasks"]`, chromedp.ByQuery),
		chromedp.Evaluate(`window.tinkerdownWS !== undefined && window.tinkerdownWS.readyState === 1`, &wsConnected),
	)
	if err != nil {
		t.Fatalf("Failed to navigate: %v\nConsole logs: %v", err, consoleLogs)
	}
	t.Log("Page loaded and lvt-source element visible")

	// Wait for data to load
	time.Sleep(500 * time.Millisecond)

	// Verify initial data is displayed
	var taskCount int
	err = chromedp.Run(ctx,
		chromedp.Evaluate(`document.querySelectorAll('tbody tr').length`, &taskCount),
	)
	if err != nil {
		t.Fatalf("Failed to count tasks: %v", err)
	}
	if taskCount != 2 {
		t.Errorf("Expected 2 tasks initially, got %d", taskCount)
	}
	t.Logf("Initial task count: %d", taskCount)

	// Test: Verify task text is displayed
	var firstTaskText string
	err = chromedp.Run(ctx,
		chromedp.Evaluate(`document.querySelector('tbody tr td:nth-child(2)').textContent.trim()`, &firstTaskText),
	)
	if err != nil {
		t.Fatalf("Failed to get task text: %v", err)
	}
	if firstTaskText != "Buy groceries" {
		t.Errorf("Expected 'Buy groceries', got '%s'", firstTaskText)
	}
	t.Logf("First task text: %s", firstTaskText)

	// Test: Add a new task
	err = chromedp.Run(ctx,
		chromedp.SetValue(`input[name="text"]`, "New test task", chromedp.ByQuery),
		chromedp.Click(`form[lvt-submit="Add"] button[type="submit"]`, chromedp.ByQuery),
		chromedp.Sleep(500*time.Millisecond),
	)
	if err != nil {
		t.Fatalf("Failed to add task: %v", err)
	}

	// Verify task was added
	err = chromedp.Run(ctx,
		chromedp.Evaluate(`document.querySelectorAll('tbody tr').length`, &taskCount),
	)
	if err != nil {
		t.Fatalf("Failed to count tasks after add: %v", err)
	}
	if taskCount != 3 {
		t.Errorf("Expected 3 tasks after add, got %d", taskCount)
	}
	t.Log("Add task test passed")

	// Test: Toggle a task (check the first checkbox)
	// After adding a task, with ORDER BY created_at DESC, the newest task is first
	// Get the ID of the first row's checkbox before clicking
	var toggledID string
	err = chromedp.Run(ctx,
		chromedp.AttributeValue(`tbody tr:first-child input[type="checkbox"]`, "lvt-data-id", &toggledID, nil, chromedp.ByQuery),
	)
	if err != nil {
		t.Fatalf("Failed to get checkbox ID: %v", err)
	}
	t.Logf("Toggling task with ID: %s", toggledID)

	err = chromedp.Run(ctx,
		chromedp.Click(`tbody tr:first-child input[type="checkbox"]`, chromedp.ByQuery),
		chromedp.Sleep(500*time.Millisecond),
	)
	if err != nil {
		t.Fatalf("Failed to toggle task: %v", err)
	}

	// Verify task was toggled (check database directly)
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("Failed to open database for verification: %v", err)
	}
	defer db.Close()

	var done int
	err = db.QueryRow("SELECT done FROM tasks WHERE id = ?", toggledID).Scan(&done)
	if err != nil {
		t.Fatalf("Failed to query toggled task: %v", err)
	}
	if done != 1 {
		t.Errorf("Expected task to be done (1), got %d", done)
	}
	t.Log("Toggle task test passed")

	// Test: Delete a task
	err = chromedp.Run(ctx,
		chromedp.Click(`tbody tr:last-child button[lvt-click="Delete"]`, chromedp.ByQuery),
		chromedp.Sleep(500*time.Millisecond),
	)
	if err != nil {
		t.Fatalf("Failed to delete task: %v", err)
	}

	// Verify task was deleted
	err = chromedp.Run(ctx,
		chromedp.Evaluate(`document.querySelectorAll('tbody tr').length`, &taskCount),
	)
	if err != nil {
		t.Fatalf("Failed to count tasks after delete: %v", err)
	}
	if taskCount != 2 {
		t.Errorf("Expected 2 tasks after delete, got %d", taskCount)
	}
	t.Log("Delete task test passed")

	// Test: Refresh button
	err = chromedp.Run(ctx,
		chromedp.Click(`button[lvt-click="Refresh"]`, chromedp.ByQuery),
		chromedp.Sleep(500*time.Millisecond),
	)
	if err != nil {
		t.Fatalf("Failed to click refresh: %v", err)
	}

	// Verify data still present after refresh
	err = chromedp.Run(ctx,
		chromedp.Evaluate(`document.querySelectorAll('tbody tr').length`, &taskCount),
	)
	if err != nil {
		t.Fatalf("Failed to count tasks after refresh: %v", err)
	}
	if taskCount != 2 {
		t.Errorf("Expected 2 tasks after refresh, got %d", taskCount)
	}
	t.Log("Refresh test passed")

	t.Log("All SQLite E2E tests passed!")
}

// setupSQLiteTestDB creates a test database with initial data
func setupSQLiteTestDB(t *testing.T, dbPath string) {
	t.Helper()

	// Remove existing database
	os.Remove(dbPath)

	// Create new database
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	defer db.Close()

	// Create tasks table (must include created_at for ORDER BY)
	_, err = db.Exec(`
		CREATE TABLE tasks (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			text TEXT NOT NULL,
			done INTEGER DEFAULT 0,
			priority TEXT DEFAULT 'medium',
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create table: %v", err)
	}

	// Insert test data
	_, err = db.Exec(`
		INSERT INTO tasks (text, done, priority) VALUES
		('Buy groceries', 0, 'high'),
		('Call mom', 0, 'medium')
	`)
	if err != nil {
		t.Fatalf("Failed to insert test data: %v", err)
	}

	t.Logf("Created test database at %s with 2 tasks", dbPath)
}
