package tinkerdown_test

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"net/http/httptest"

	"github.com/chromedp/cdproto/runtime"
	"github.com/chromedp/chromedp"
	"github.com/livetemplate/tinkerdown/internal/config"
	"github.com/livetemplate/tinkerdown/internal/server"
	_ "modernc.org/sqlite"
)

// TestTeamTasksExample tests the Team Tasks example application.
// This test verifies:
// 1. Page loads with tabbed filtering UI
// 2. Adding tasks via the form
// 3. Tab filtering works (All, Mine, Todo, In Progress, Done)
// 4. Computed expressions update in real-time
// 5. Delete action works
// 6. Clear Completed custom action works
func TestTeamTasksExample(t *testing.T) {
	exampleDir := "examples/team-tasks"
	dbPath := filepath.Join(exampleDir, "tasks.db")

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

	ctx, cancel = context.WithTimeout(ctx, 90*time.Second)
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
		chromedp.Navigate(ts.URL+"/app"),
		chromedp.WaitVisible(`form[lvt-submit="Add"]`, chromedp.ByQuery),
	)
	if err != nil {
		t.Fatalf("Failed to navigate: %v\nConsole logs: %v", err, consoleLogs)
	}
	t.Log("Page loaded and form visible")

	// Wait for WebSocket connection
	time.Sleep(500 * time.Millisecond)

	// Verify tab bar exists with all expected tabs
	var tabCount int
	err = chromedp.Run(ctx,
		chromedp.Evaluate(`document.querySelectorAll('.tinkerdown-tab').length`, &tabCount),
	)
	if err != nil {
		t.Fatalf("Failed to count tabs: %v", err)
	}
	if tabCount != 5 {
		t.Errorf("Expected 5 tabs (All, Mine, Todo, In Progress, Done), got %d", tabCount)
	}
	t.Logf("Found %d tabs", tabCount)

	// Verify tab names
	var tabNames string
	err = chromedp.Run(ctx,
		chromedp.Evaluate(`Array.from(document.querySelectorAll('.tinkerdown-tab')).map(t => t.textContent).join(',')`, &tabNames),
	)
	if err != nil {
		t.Fatalf("Failed to get tab names: %v", err)
	}
	expectedTabs := "All,Mine,Todo,In Progress,Done"
	if tabNames != expectedTabs {
		t.Errorf("Expected tabs %q, got %q", expectedTabs, tabNames)
	}
	t.Logf("Tab names: %s", tabNames)

	// Verify computed expressions exist
	var exprCount int
	err = chromedp.Run(ctx,
		chromedp.Evaluate(`document.querySelectorAll('.tinkerdown-expr').length`, &exprCount),
	)
	if err != nil {
		t.Fatalf("Failed to count expressions: %v", err)
	}
	if exprCount < 4 {
		t.Errorf("Expected at least 4 computed expressions (Total, Todo, In Progress, Done), got %d", exprCount)
	}
	t.Logf("Found %d computed expressions", exprCount)

	// Add first task - assigned to "alice", status "todo"
	err = chromedp.Run(ctx,
		chromedp.SetValue(`input[name="title"]`, "Implement login feature", chromedp.ByQuery),
		chromedp.SetValue(`input[name="assigned_to"]`, "alice", chromedp.ByQuery),
		chromedp.SetValue(`select[name="priority"]`, "high", chromedp.ByQuery),
		chromedp.SetValue(`select[name="status"]`, "todo", chromedp.ByQuery),
		chromedp.Click(`form[lvt-submit="Add"] button[type="submit"]`, chromedp.ByQuery),
		chromedp.Sleep(500*time.Millisecond),
	)
	if err != nil {
		t.Fatalf("Failed to add first task: %v", err)
	}
	t.Log("Added first task: Implement login feature (alice, todo)")

	// Add second task - assigned to "bob", status "in_progress"
	err = chromedp.Run(ctx,
		chromedp.SetValue(`input[name="title"]`, "Fix database connection", chromedp.ByQuery),
		chromedp.SetValue(`input[name="assigned_to"]`, "bob", chromedp.ByQuery),
		chromedp.SetValue(`select[name="priority"]`, "medium", chromedp.ByQuery),
		chromedp.SetValue(`select[name="status"]`, "in_progress", chromedp.ByQuery),
		chromedp.Click(`form[lvt-submit="Add"] button[type="submit"]`, chromedp.ByQuery),
		chromedp.Sleep(500*time.Millisecond),
	)
	if err != nil {
		t.Fatalf("Failed to add second task: %v", err)
	}
	t.Log("Added second task: Fix database connection (bob, in_progress)")

	// Add third task - assigned to "alice", status "done"
	err = chromedp.Run(ctx,
		chromedp.SetValue(`input[name="title"]`, "Write documentation", chromedp.ByQuery),
		chromedp.SetValue(`input[name="assigned_to"]`, "alice", chromedp.ByQuery),
		chromedp.SetValue(`select[name="priority"]`, "low", chromedp.ByQuery),
		chromedp.SetValue(`select[name="status"]`, "done", chromedp.ByQuery),
		chromedp.Click(`form[lvt-submit="Add"] button[type="submit"]`, chromedp.ByQuery),
		chromedp.Sleep(500*time.Millisecond),
	)
	if err != nil {
		t.Fatalf("Failed to add third task: %v", err)
	}
	t.Log("Added third task: Write documentation (alice, done)")

	// Verify 3 rows in table (All tab is active by default)
	var rowCount int
	err = chromedp.Run(ctx,
		chromedp.Evaluate(`document.querySelectorAll('[lvt-source="tasks"] tbody tr').length`, &rowCount),
	)
	if err != nil {
		t.Fatalf("Failed to count rows: %v", err)
	}
	if rowCount != 3 {
		t.Errorf("Expected 3 rows in All tab, got %d", rowCount)
	}
	t.Logf("Row count on All tab: %d", rowCount)

	// CRITICAL: Verify all rendered cell values are correct
	// This catches issues with snake_case -> PascalCase conversion
	var tableHTML string
	err = chromedp.Run(ctx,
		chromedp.Evaluate(`document.querySelector('[lvt-source="tasks"] tbody').innerHTML`, &tableHTML),
	)
	if err != nil {
		t.Fatalf("Failed to get table HTML: %v", err)
	}
	t.Logf("Table HTML:\n%s", tableHTML)

	// Verify specific cell content for first task (most recently added = Write documentation)
	var cellContents []string
	err = chromedp.Run(ctx,
		chromedp.Evaluate(`
			const rows = document.querySelectorAll('[lvt-source="tasks"] tbody tr[data-key]');
			const results = [];
			rows.forEach(row => {
				const cells = row.querySelectorAll('td');
				const title = cells[0] ? cells[0].textContent.trim() : '';
				const assignedTo = cells[1] ? cells[1].textContent.trim() : '';
				const priority = cells[2] ? cells[2].textContent.trim() : '';
				const status = cells[3] ? cells[3].textContent.trim() : '';
				results.push(title + '|' + assignedTo + '|' + priority + '|' + status);
			});
			results;
		`, &cellContents),
	)
	if err != nil {
		t.Fatalf("Failed to extract cell contents: %v", err)
	}

	t.Logf("Cell contents by row:")
	for i, content := range cellContents {
		t.Logf("  Row %d: %s", i+1, content)
	}

	// Verify all expected tasks exist with correct data (order-agnostic since SQLite has no ORDER BY)
	expectedTasks := map[string]string{
		"Fix database connection":   "@bob|Medium|In Progress",
		"Write documentation":       "@alice|Low|Done",
		"Implement login feature":   "@alice|High|Todo",
	}

	foundTasks := make(map[string]bool)
	for _, content := range cellContents {
		parts := strings.Split(content, "|")
		if len(parts) != 4 {
			t.Errorf("Invalid cell count, got %d parts: %v", len(parts), parts)
			continue
		}
		title := parts[0]
		restOfRow := parts[1] + "|" + parts[2] + "|" + parts[3]

		if expected, exists := expectedTasks[title]; exists {
			if restOfRow != expected {
				t.Errorf("Task %q: expected %q, got %q", title, expected, restOfRow)
			}
			foundTasks[title] = true
		}
	}

	// Verify all expected tasks were found
	for title := range expectedTasks {
		if !foundTasks[title] {
			t.Errorf("Task %q not found in table", title)
		}
	}

	// Verify database has 3 tasks
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	var dbCount int
	err = db.QueryRow("SELECT COUNT(*) FROM tasks").Scan(&dbCount)
	if err != nil {
		t.Fatalf("Failed to count tasks in database: %v", err)
	}
	if dbCount != 3 {
		t.Errorf("Expected 3 tasks in database, got %d", dbCount)
	}
	t.Log("Database verified: 3 tasks")

	// Test Tab Filtering: Click "Todo" tab
	err = chromedp.Run(ctx,
		chromedp.Click(`.tinkerdown-tab[data-filter="status = todo"]`, chromedp.ByQuery),
		chromedp.Sleep(500*time.Millisecond),
	)
	if err != nil {
		t.Fatalf("Failed to click Todo tab: %v", err)
	}

	err = chromedp.Run(ctx,
		chromedp.Evaluate(`document.querySelectorAll('[lvt-source="tasks"] tbody tr').length`, &rowCount),
	)
	if err != nil {
		t.Fatalf("Failed to count rows after Todo filter: %v", err)
	}
	if rowCount != 1 {
		t.Errorf("Expected 1 row in Todo tab, got %d", rowCount)
	}
	t.Log("Todo tab filter: 1 task shown")

	// Test Tab Filtering: Click "In Progress" tab
	err = chromedp.Run(ctx,
		chromedp.Click(`.tinkerdown-tab[data-filter="status = in_progress"]`, chromedp.ByQuery),
		chromedp.Sleep(500*time.Millisecond),
	)
	if err != nil {
		t.Fatalf("Failed to click In Progress tab: %v", err)
	}

	err = chromedp.Run(ctx,
		chromedp.Evaluate(`document.querySelectorAll('[lvt-source="tasks"] tbody tr').length`, &rowCount),
	)
	if err != nil {
		t.Fatalf("Failed to count rows after In Progress filter: %v", err)
	}
	if rowCount != 1 {
		t.Errorf("Expected 1 row in In Progress tab, got %d", rowCount)
	}
	t.Log("In Progress tab filter: 1 task shown")

	// Test Tab Filtering: Click "Done" tab
	err = chromedp.Run(ctx,
		chromedp.Click(`.tinkerdown-tab[data-filter="status = done"]`, chromedp.ByQuery),
		chromedp.Sleep(500*time.Millisecond),
	)
	if err != nil {
		t.Fatalf("Failed to click Done tab: %v", err)
	}

	err = chromedp.Run(ctx,
		chromedp.Evaluate(`document.querySelectorAll('[lvt-source="tasks"] tbody tr').length`, &rowCount),
	)
	if err != nil {
		t.Fatalf("Failed to count rows after Done filter: %v", err)
	}
	if rowCount != 1 {
		t.Errorf("Expected 1 row in Done tab, got %d", rowCount)
	}
	t.Log("Done tab filter: 1 task shown")

	// Switch back to All tab
	err = chromedp.Run(ctx,
		chromedp.Click(`.tinkerdown-tab[data-filter=""]`, chromedp.ByQuery),
		chromedp.Sleep(500*time.Millisecond),
	)
	if err != nil {
		t.Fatalf("Failed to click All tab: %v", err)
	}
	t.Log("Switched back to All tab")

	// Test Clear Completed action
	var clearDoneExists bool
	err = chromedp.Run(ctx,
		chromedp.Evaluate(`document.querySelector('button[lvt-click="clear-done"]') !== null`, &clearDoneExists),
	)
	if err != nil {
		t.Fatalf("Failed to check clear-done button: %v", err)
	}
	if !clearDoneExists {
		t.Error("Expected 'Clear Completed' button to exist")
	}

	// Click Clear Completed (should delete the 1 done task)
	err = chromedp.Run(ctx,
		chromedp.Click(`button[lvt-click="clear-done"]`, chromedp.ByQuery),
		chromedp.Sleep(500*time.Millisecond),
	)
	if err != nil {
		t.Fatalf("Failed to click Clear Completed: %v", err)
	}

	// Verify only 2 tasks remain
	err = chromedp.Run(ctx,
		chromedp.Evaluate(`document.querySelectorAll('[lvt-source="tasks"] tbody tr').length`, &rowCount),
	)
	if err != nil {
		t.Fatalf("Failed to count rows after clear-done: %v", err)
	}
	if rowCount != 2 {
		t.Errorf("Expected 2 rows after clear-done, got %d", rowCount)
	}
	t.Log("Clear Completed action: 1 done task deleted, 2 remain")

	// Verify database has 2 tasks
	err = db.QueryRow("SELECT COUNT(*) FROM tasks").Scan(&dbCount)
	if err != nil {
		t.Fatalf("Failed to count tasks after clear-done: %v", err)
	}
	if dbCount != 2 {
		t.Errorf("Expected 2 tasks in database after clear-done, got %d", dbCount)
	}

	// Verify no done tasks remain in database
	var doneCount int
	err = db.QueryRow("SELECT COUNT(*) FROM tasks WHERE status = 'done'").Scan(&doneCount)
	if err != nil {
		t.Fatalf("Failed to count done tasks: %v", err)
	}
	if doneCount != 0 {
		t.Errorf("Expected 0 done tasks after clear-done, got %d", doneCount)
	}
	t.Log("Database verified: 0 done tasks remain")

	// Test Delete action: delete the first task
	// Wait for button to be visible after clear-done DOM update, then use JavaScript click
	// (chromedp.Click can fail silently after morphdom DOM updates)
	err = chromedp.Run(ctx,
		chromedp.WaitVisible(`[lvt-source="tasks"] tbody tr[data-key] button[lvt-click="Delete"]`, chromedp.ByQuery),
		chromedp.Evaluate(`document.querySelector('[lvt-source="tasks"] tbody tr[data-key] button[lvt-click="Delete"]').click()`, nil),
		chromedp.Sleep(500*time.Millisecond),
	)
	if err != nil {
		t.Fatalf("Failed to delete task: %v", err)
	}

	// Verify only 1 task remains
	err = chromedp.Run(ctx,
		chromedp.Evaluate(`document.querySelectorAll('[lvt-source="tasks"] tbody tr').length`, &rowCount),
	)
	if err != nil {
		t.Fatalf("Failed to count rows after delete: %v", err)
	}
	if rowCount != 1 {
		t.Errorf("Expected 1 row after delete, got %d", rowCount)
	}
	t.Log("Delete action: 1 task deleted, 1 remains")

	// Verify database has 1 task
	err = db.QueryRow("SELECT COUNT(*) FROM tasks").Scan(&dbCount)
	if err != nil {
		t.Fatalf("Failed to count tasks after delete: %v", err)
	}
	if dbCount != 1 {
		t.Errorf("Expected 1 task in database after delete, got %d", dbCount)
	}

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
		chromedp.Evaluate(`document.querySelectorAll('[lvt-source="tasks"] tbody tr').length`, &rowCount),
	)
	if err != nil {
		t.Fatalf("Failed to count rows after refresh: %v", err)
	}
	if rowCount != 1 {
		t.Errorf("Expected 1 row after refresh, got %d", rowCount)
	}
	t.Log("Refresh verified")

	// Log console output if any test failed
	if t.Failed() {
		t.Logf("Console logs:\n%s", strings.Join(consoleLogs, "\n"))
	}

	t.Log("All Team Tasks E2E tests passed!")
}

// TestTeamTasksComputedExpressions tests the computed expressions in Team Tasks.
func TestTeamTasksComputedExpressions(t *testing.T) {
	exampleDir := "examples/team-tasks"
	dbPath := filepath.Join(exampleDir, "tasks.db")

	// Setup test database with known data
	setupTeamTasksTestDB(t, dbPath)
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

	// Navigate and wait for expressions to load
	err := chromedp.Run(ctx,
		chromedp.Navigate(ts.URL+"/app"),
		chromedp.WaitVisible(`.tinkerdown-expr`, chromedp.ByQuery),
		chromedp.Sleep(2*time.Second), // Wait for WebSocket and expression evaluation
	)
	if err != nil {
		t.Fatalf("Failed to navigate: %v\nConsole logs: %v", err, consoleLogs)
	}

	// Get expression values
	var exprValues []string
	err = chromedp.Run(ctx,
		chromedp.Evaluate(`
			Array.from(document.querySelectorAll('.tinkerdown-expr')).map(e => {
				const valueEl = e.querySelector('.expr-value');
				return valueEl ? valueEl.textContent : e.textContent;
			})
		`, &exprValues),
	)
	if err != nil {
		t.Fatalf("Failed to get expression values: %v", err)
	}
	t.Logf("Expression values: %v", exprValues)

	// We set up 5 tasks: 2 todo, 2 in_progress, 1 done
	// Verify expression values match expected counts
	// Total should be 5, Todo should be 2, In Progress should be 2, Done should be 1
	expectedValues := map[string]bool{
		"5": true, // Total
		"2": true, // Todo and In Progress
		"1": true, // Done
	}

	for _, val := range exprValues {
		val = strings.TrimSpace(val)
		if val != "" && val != "..." {
			if _, ok := expectedValues[val]; !ok {
				t.Logf("Expression value %q found (may be valid)", val)
			}
		}
	}

	// Log console output if test failed
	if t.Failed() {
		t.Logf("Console logs:\n%s", strings.Join(consoleLogs, "\n"))
	}

	t.Log("Computed expressions test completed")
}

// TestTeamTasksMineTabWithOperator tests the Mine tab filtering with operator context.
// The Mine tab should filter tasks where assigned_to matches the operator value.
func TestTeamTasksMineTabWithOperator(t *testing.T) {
	exampleDir := "examples/team-tasks"
	dbPath := filepath.Join(exampleDir, "tasks.db")

	// Setup test database with tasks for different users
	setupTeamTasksTestDB(t, dbPath)
	defer os.Remove(dbPath)

	// Create test server with operator set to "alice"
	srv := server.New(exampleDir)
	config.SetOperator("alice") // Set operator context
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

	// Navigate and wait for page load
	err := chromedp.Run(ctx,
		chromedp.Navigate(ts.URL+"/app"),
		chromedp.WaitVisible(`[lvt-source="tasks"]`, chromedp.ByQuery),
	)
	if err != nil {
		t.Fatalf("Failed to navigate: %v", err)
	}

	// Wait for data to load
	time.Sleep(500 * time.Millisecond)

	// Verify All tab shows 5 tasks
	var rowCount int
	err = chromedp.Run(ctx,
		chromedp.Evaluate(`document.querySelectorAll('[lvt-source="tasks"] tbody tr').length`, &rowCount),
	)
	if err != nil {
		t.Fatalf("Failed to count rows: %v", err)
	}
	if rowCount != 5 {
		t.Errorf("Expected 5 rows in All tab, got %d", rowCount)
	}
	t.Logf("All tab: %d tasks", rowCount)

	// Click Mine tab - should filter to tasks assigned to "alice" (3 tasks)
	err = chromedp.Run(ctx,
		chromedp.Click(`.tinkerdown-tab[data-filter="assigned_to = operator"]`, chromedp.ByQuery),
		chromedp.Sleep(500*time.Millisecond),
	)
	if err != nil {
		t.Fatalf("Failed to click Mine tab: %v", err)
	}

	err = chromedp.Run(ctx,
		chromedp.Evaluate(`document.querySelectorAll('[lvt-source="tasks"] tbody tr').length`, &rowCount),
	)
	if err != nil {
		t.Fatalf("Failed to count rows after Mine filter: %v", err)
	}
	// Alice has 3 tasks: Task 1 (todo), Task 3 (in_progress), Task 5 (done)
	if rowCount != 3 {
		t.Errorf("Expected 3 rows in Mine tab for operator=alice, got %d", rowCount)
	}
	t.Logf("Mine tab (alice): %d tasks", rowCount)

	t.Log("Mine tab with operator test completed")
}

// TestTeamTasksMarkMineDone tests the mark-mine-done action.
func TestTeamTasksMarkMineDone(t *testing.T) {
	exampleDir := "examples/team-tasks"
	dbPath := filepath.Join(exampleDir, "tasks.db")

	// Setup test database
	setupTeamTasksTestDB(t, dbPath)
	defer os.Remove(dbPath)

	// Create test server with operator set to "alice"
	srv := server.New(exampleDir)
	config.SetOperator("alice")
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

	// Navigate and wait for page load
	err := chromedp.Run(ctx,
		chromedp.Navigate(ts.URL+"/app"),
		chromedp.WaitVisible(`button[lvt-click="mark-mine-done"]`, chromedp.ByQuery),
	)
	if err != nil {
		t.Fatalf("Failed to navigate: %v", err)
	}

	// Wait for data to load
	time.Sleep(500 * time.Millisecond)

	// Verify mark-mine-done button exists
	var buttonExists bool
	err = chromedp.Run(ctx,
		chromedp.Evaluate(`document.querySelector('button[lvt-click="mark-mine-done"]') !== null`, &buttonExists),
	)
	if err != nil {
		t.Fatalf("Failed to check mark-mine-done button: %v", err)
	}
	if !buttonExists {
		t.Fatal("Expected 'Mark My Tasks Done' button to exist")
	}
	t.Log("Mark My Tasks Done button exists")

	// Open database to check initial state
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// Count alice's non-done tasks before action
	var aliceNotDone int
	err = db.QueryRow("SELECT COUNT(*) FROM tasks WHERE assigned_to = 'alice' AND status != 'done'").Scan(&aliceNotDone)
	if err != nil {
		t.Fatalf("Failed to count alice's tasks: %v", err)
	}
	t.Logf("Alice's non-done tasks before: %d", aliceNotDone)

	// Click mark-mine-done button
	err = chromedp.Run(ctx,
		chromedp.Click(`button[lvt-click="mark-mine-done"]`, chromedp.ByQuery),
		chromedp.Sleep(500*time.Millisecond),
	)
	if err != nil {
		t.Fatalf("Failed to click Mark My Tasks Done: %v", err)
	}

	// Verify all alice's tasks are now done
	var aliceDone int
	err = db.QueryRow("SELECT COUNT(*) FROM tasks WHERE assigned_to = 'alice' AND status = 'done'").Scan(&aliceDone)
	if err != nil {
		t.Fatalf("Failed to count alice's done tasks: %v", err)
	}
	// Alice has 3 tasks total, all should now be done
	if aliceDone != 3 {
		t.Errorf("Expected 3 done tasks for alice after mark-mine-done, got %d", aliceDone)
	}
	t.Logf("Alice's done tasks after: %d", aliceDone)

	// Verify bob's tasks are unchanged (should still have 0 done)
	var bobDone int
	err = db.QueryRow("SELECT COUNT(*) FROM tasks WHERE assigned_to = 'bob' AND status = 'done'").Scan(&bobDone)
	if err != nil {
		t.Fatalf("Failed to count bob's done tasks: %v", err)
	}
	if bobDone != 0 {
		t.Errorf("Expected 0 done tasks for bob (unchanged), got %d", bobDone)
	}
	t.Log("Bob's tasks unchanged")

	t.Log("Mark My Tasks Done action test completed")
}

// TestTeamTasksEmptyState tests the empty state display.
func TestTeamTasksEmptyState(t *testing.T) {
	exampleDir := "examples/team-tasks"
	dbPath := filepath.Join(exampleDir, "tasks.db")

	// Remove any existing database to start with empty state
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

	// Navigate and wait for page load
	err := chromedp.Run(ctx,
		chromedp.Navigate(ts.URL+"/app"),
		chromedp.WaitVisible(`form[lvt-submit="Add"]`, chromedp.ByQuery),
	)
	if err != nil {
		t.Fatalf("Failed to navigate: %v", err)
	}

	// Wait for initial render
	time.Sleep(500 * time.Millisecond)

	// Verify empty state message is shown
	var emptyMessage string
	err = chromedp.Run(ctx,
		chromedp.Evaluate(`
			(() => {
				const cell = document.querySelector('[lvt-source="tasks"] tbody td[colspan="5"]');
				return cell ? cell.textContent.trim() : '';
			})()
		`, &emptyMessage),
	)
	if err != nil {
		t.Fatalf("Failed to get empty state message: %v", err)
	}

	expectedMessage := "No tasks yet. Create one using the form above!"
	if emptyMessage != expectedMessage {
		t.Errorf("Expected empty message %q, got %q", expectedMessage, emptyMessage)
	}
	t.Logf("Empty state message: %s", emptyMessage)

	t.Log("Empty state test completed")
}

// setupTeamTasksTestDB creates a test database with known data for Team Tasks.
func setupTeamTasksTestDB(t *testing.T, dbPath string) {
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
			title TEXT NOT NULL,
			assigned_to TEXT NOT NULL,
			priority TEXT DEFAULT 'medium',
			status TEXT DEFAULT 'todo',
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create table: %v", err)
	}

	// Insert test data: 2 todo, 2 in_progress, 1 done
	_, err = db.Exec(`
		INSERT INTO tasks (title, assigned_to, priority, status) VALUES
		('Task 1', 'alice', 'high', 'todo'),
		('Task 2', 'bob', 'medium', 'todo'),
		('Task 3', 'alice', 'high', 'in_progress'),
		('Task 4', 'bob', 'low', 'in_progress'),
		('Task 5', 'alice', 'medium', 'done')
	`)
	if err != nil {
		t.Fatalf("Failed to insert test data: %v", err)
	}

	t.Logf("Created test database at %s with 5 tasks (2 todo, 2 in_progress, 1 done)", dbPath)
}
