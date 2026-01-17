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

// TestActionButtons tests custom action buttons declared in frontmatter.
// This test verifies that:
// 1. Custom SQL actions execute correctly when button is clicked
// 2. Data is refreshed after action execution
// 3. The "clear-done" action deletes all completed tasks
// 4. The "mark-all-done" action marks all tasks as complete
func TestActionButtons(t *testing.T) {
	exampleDir := "examples/action-buttons"

	// Setup: create test database with initial data
	dbPath := filepath.Join(exampleDir, "tasks.db")
	setupActionButtonsTestDB(t, dbPath)
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

	// Verify initial data is displayed (4 tasks: 2 done, 2 not done)
	// Use specific selector to only count rows in the tasks table, not documentation tables
	var taskCount int
	err = chromedp.Run(ctx,
		chromedp.Evaluate(`document.querySelectorAll('[lvt-source="tasks"] tbody tr').length`, &taskCount),
	)
	if err != nil {
		t.Fatalf("Failed to count tasks: %v", err)
	}
	if taskCount != 4 {
		t.Errorf("Expected 4 tasks initially, got %d", taskCount)
	}
	t.Logf("Initial task count: %d", taskCount)

	// Verify custom action buttons exist
	var clearDoneExists, markAllDoneExists bool
	err = chromedp.Run(ctx,
		chromedp.Evaluate(`document.querySelector('button[lvt-click="clear-done"]') !== null`, &clearDoneExists),
		chromedp.Evaluate(`document.querySelector('button[lvt-click="mark-all-done"]') !== null`, &markAllDoneExists),
	)
	if err != nil {
		t.Fatalf("Failed to check buttons: %v", err)
	}
	if !clearDoneExists {
		t.Error("Expected 'Clear Completed' button to exist")
	}
	if !markAllDoneExists {
		t.Error("Expected 'Mark All Done' button to exist")
	}
	t.Log("Custom action buttons exist")

	// Test 1: Click "Clear Completed" button
	// This should delete the 2 completed tasks
	err = chromedp.Run(ctx,
		chromedp.Click(`button[lvt-click="clear-done"]`, chromedp.ByQuery),
		chromedp.Sleep(500*time.Millisecond),
	)
	if err != nil {
		t.Fatalf("Failed to click 'Clear Completed': %v\nConsole logs: %v", err, consoleLogs)
	}

	// Verify 2 tasks remain (the ones that were not done)
	err = chromedp.Run(ctx,
		chromedp.Evaluate(`document.querySelectorAll('[lvt-source="tasks"] tbody tr').length`, &taskCount),
	)
	if err != nil {
		t.Fatalf("Failed to count tasks after clear-done: %v", err)
	}
	if taskCount != 2 {
		t.Errorf("Expected 2 tasks after clear-done, got %d", taskCount)
	}
	t.Log("clear-done action passed: deleted 2 completed tasks")

	// Verify in database
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("Failed to open database for verification: %v", err)
	}
	defer db.Close()

	var dbTaskCount int
	err = db.QueryRow("SELECT COUNT(*) FROM tasks").Scan(&dbTaskCount)
	if err != nil {
		t.Fatalf("Failed to count tasks in database: %v", err)
	}
	if dbTaskCount != 2 {
		t.Errorf("Expected 2 tasks in database, got %d", dbTaskCount)
	}

	// Test 2: Click "Mark All Done" button
	// This should mark all remaining tasks as done
	err = chromedp.Run(ctx,
		chromedp.Click(`button[lvt-click="mark-all-done"]`, chromedp.ByQuery),
		chromedp.Sleep(500*time.Millisecond),
	)
	if err != nil {
		t.Fatalf("Failed to click 'Mark All Done': %v\nConsole logs: %v", err, consoleLogs)
	}

	// Verify all tasks are now done (check database)
	var doneCount int
	err = db.QueryRow("SELECT COUNT(*) FROM tasks WHERE done = 1").Scan(&doneCount)
	if err != nil {
		t.Fatalf("Failed to count done tasks: %v", err)
	}
	if doneCount != 2 {
		t.Errorf("Expected 2 done tasks after mark-all-done, got %d", doneCount)
	}
	t.Log("mark-all-done action passed: marked all tasks as done")

	// Test 3: Clear completed again (should now delete all remaining tasks)
	err = chromedp.Run(ctx,
		chromedp.Click(`button[lvt-click="clear-done"]`, chromedp.ByQuery),
		chromedp.Sleep(500*time.Millisecond),
	)
	if err != nil {
		t.Fatalf("Failed to click 'Clear Completed' second time: %v", err)
	}

	// Verify no tasks remain
	err = chromedp.Run(ctx,
		chromedp.Evaluate(`document.querySelectorAll('[lvt-source="tasks"] tbody tr').length`, &taskCount),
	)
	if err != nil {
		t.Fatalf("Failed to count tasks after final clear-done: %v", err)
	}
	if taskCount != 0 {
		t.Errorf("Expected 0 tasks after final clear-done, got %d", taskCount)
	}
	t.Log("Final clear-done passed: all tasks deleted")

	t.Log("All Action Buttons E2E tests passed!")
}

// setupActionButtonsTestDB creates a test database with initial data
func setupActionButtonsTestDB(t *testing.T, dbPath string) {
	t.Helper()

	// Remove existing database
	os.Remove(dbPath)

	// Create new database
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	defer db.Close()

	// Create tasks table
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

	// Insert test data: 2 done, 2 not done
	_, err = db.Exec(`
		INSERT INTO tasks (text, done, priority) VALUES
		('Buy groceries', 1, 'high'),
		('Call mom', 1, 'medium'),
		('Write code', 0, 'high'),
		('Review PR', 0, 'low')
	`)
	if err != nil {
		t.Fatalf("Failed to insert test data: %v", err)
	}

	t.Logf("Created test database at %s with 4 tasks (2 done, 2 not done)", dbPath)
}
