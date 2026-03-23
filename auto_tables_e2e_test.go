//go:build !ci

package tinkerdown_test

import (
	"database/sql"
	"fmt"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/chromedp/cdproto/runtime"
	"github.com/chromedp/chromedp"
	"github.com/livetemplate/tinkerdown/internal/server"
	_ "modernc.org/sqlite"
)

// autoTablesConsoleLogs provides thread-safe console log capture for auto-tables tests
type autoTablesConsoleLogs struct {
	mu   sync.RWMutex
	logs []string
}

func (l *autoTablesConsoleLogs) append(msg string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.logs = append(l.logs, msg)
}

func (l *autoTablesConsoleLogs) get() []string {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return append([]string{}, l.logs...)
}

// autoTablesTestContext holds test infrastructure
type autoTablesTestContext struct {
	Server      *httptest.Server
	ChromeCtx   *DockerChromeContext
	URL         string
	ConsoleLogs *autoTablesConsoleLogs
	srv         *server.Server
}

func setupAutoTablesTest(t *testing.T, exampleDir string) (*autoTablesTestContext, func()) {
	t.Helper()

	srv := server.New(exampleDir)
	if err := srv.Discover(); err != nil {
		t.Fatalf("Failed to discover pages: %v", err)
	}

	handler := server.WithCompression(srv)
	ts := httptest.NewServer(handler)

	chromeCtx, chromeCleanup := SetupDockerChrome(t, 90*time.Second)

	url := ConvertURLForDockerChrome(ts.URL)

	consoleLogs := &autoTablesConsoleLogs{}
	chromedp.ListenTarget(chromeCtx.Context, func(ev any) {
		if ev, ok := ev.(*runtime.EventConsoleAPICalled); ok {
			for _, arg := range ev.Args {
				consoleLogs.append(fmt.Sprintf("[Console] %s", arg.Value))
			}
		}
	})

	testCtx := &autoTablesTestContext{
		Server:      ts,
		ChromeCtx:   chromeCtx,
		URL:         url,
		ConsoleLogs: consoleLogs,
		srv:         srv,
	}

	cleanup := func() {
		chromeCleanup()
		ts.Close()
	}

	return testCtx, cleanup
}

// createAutoTablesSQLiteExample creates a temp dir with a markdown file using a writable SQLite source
func createAutoTablesSQLiteExample(t *testing.T) (string, func()) {
	t.Helper()

	tempDir, err := os.MkdirTemp("", "auto-tables-sqlite-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	// Create SQLite database with test data
	dbPath := filepath.Join(tempDir, "test.db")
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		os.RemoveAll(tempDir)
		t.Fatalf("Failed to open SQLite: %v", err)
	}
	defer db.Close()

	_, err = db.Exec(`CREATE TABLE items (id INTEGER PRIMARY KEY AUTOINCREMENT, name TEXT NOT NULL, price TEXT NOT NULL, created_at DATETIME DEFAULT CURRENT_TIMESTAMP)`)
	if err != nil {
		os.RemoveAll(tempDir)
		t.Fatalf("Failed to create table: %v", err)
	}

	_, err = db.Exec(`INSERT INTO items (name, price) VALUES ('Widget', '9.99'), ('Gadget', '19.99')`)
	if err != nil {
		os.RemoveAll(tempDir)
		t.Fatalf("Failed to insert test data: %v", err)
	}

	// Markdown file: heading "Items" matches source "items"
	content := `---
title: Auto Table Test
sources:
  items:
    type: sqlite
    db: ./test.db
    table: items
    readonly: false
---

# Auto Table Test

## Items
| Name | Price |
|------|-------|
`

	indexPath := filepath.Join(tempDir, "index.md")
	if err := os.WriteFile(indexPath, []byte(content), 0644); err != nil {
		os.RemoveAll(tempDir)
		t.Fatalf("Failed to write index.md: %v", err)
	}

	cleanup := func() {
		os.RemoveAll(tempDir)
	}

	return tempDir, cleanup
}

// createAutoTablesRESTExample creates a temp dir with a read-only REST source
func createAutoTablesRESTExample(t *testing.T) (string, func()) {
	t.Helper()

	tempDir, err := os.MkdirTemp("", "auto-tables-rest-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	content := `---
title: Auto Table REST Test
sources:
  users:
    type: rest
    from: https://jsonplaceholder.typicode.com/users
---

# User Directory

## Users
| Name | Email | Phone |
|------|-------|-------|
`

	indexPath := filepath.Join(tempDir, "index.md")
	if err := os.WriteFile(indexPath, []byte(content), 0644); err != nil {
		os.RemoveAll(tempDir)
		t.Fatalf("Failed to write index.md: %v", err)
	}

	cleanup := func() {
		os.RemoveAll(tempDir)
	}

	return tempDir, cleanup
}

// TestAutoTables_SQLiteRenders verifies that a markdown table under a heading
// matching a writable SQLite source renders as an interactive table with data.
func TestAutoTables_SQLiteRenders(t *testing.T) {
	tempDir, tempCleanup := createAutoTablesSQLiteExample(t)
	defer tempCleanup()

	testCtx, cleanup := setupAutoTablesTest(t, tempDir)
	defer cleanup()

	ctx := testCtx.ChromeCtx.Context
	t.Logf("Test server URL: %s", testCtx.Server.URL)

	// Navigate and wait for content to render
	var hasAutoSource bool
	err := chromedp.Run(ctx,
		chromedp.Navigate(testCtx.URL+"/"),
		chromedp.Sleep(15*time.Second),
		chromedp.Evaluate(`document.querySelector('[lvt-source="items"]') !== null`, &hasAutoSource),
	)
	if err != nil {
		t.Fatalf("Failed to navigate: %v", err)
	}

	if !hasAutoSource {
		var htmlContent string
		chromedp.Run(ctx, chromedp.OuterHTML("html", &htmlContent))
		t.Logf("HTML (first 3000 chars): %s", htmlContent[:min(3000, len(htmlContent))])
		t.Logf("Console logs: %v", testCtx.ConsoleLogs.get())
		t.Fatal("Auto-generated table was not rendered")
	}
	t.Log("Auto-generated table rendered successfully")

	// Verify table has rows with pre-loaded data
	var rowCount int
	err = chromedp.Run(ctx,
		chromedp.Evaluate(`document.querySelectorAll('[lvt-source="items"] tbody tr').length`, &rowCount),
	)
	if err != nil {
		t.Fatalf("Failed to count rows: %v", err)
	}

	if rowCount < 2 {
		t.Fatalf("Expected at least 2 rows (pre-loaded data), got %d", rowCount)
	}
	t.Logf("Found %d rows in auto-generated table", rowCount)

	// Verify add form exists (writable source)
	var hasAddForm bool
	err = chromedp.Run(ctx,
		chromedp.Evaluate(`document.querySelector('[lvt-source="items"] form[lvt-submit="Add"]') !== null`, &hasAddForm),
	)
	if err != nil {
		t.Fatalf("Failed to check for add form: %v", err)
	}

	if !hasAddForm {
		t.Fatal("Expected add form for writable source")
	}
	t.Log("Add form present for writable source")

	// Verify delete buttons exist
	var hasDeleteBtn bool
	err = chromedp.Run(ctx,
		chromedp.Evaluate(`document.querySelector('[lvt-source="items"] [lvt-click="Delete"]') !== null`, &hasDeleteBtn),
	)
	if err != nil {
		t.Fatalf("Failed to check for delete button: %v", err)
	}

	if !hasDeleteBtn {
		t.Fatal("Expected delete buttons for writable source")
	}
	t.Log("Delete buttons present for writable source")
}

// TestAutoTables_SQLiteAdd verifies adding a new item via the auto-generated form.
func TestAutoTables_SQLiteAdd(t *testing.T) {
	tempDir, tempCleanup := createAutoTablesSQLiteExample(t)
	defer tempCleanup()

	testCtx, cleanup := setupAutoTablesTest(t, tempDir)
	defer cleanup()

	ctx := testCtx.ChromeCtx.Context

	// Navigate and wait for content
	err := chromedp.Run(ctx,
		chromedp.Navigate(testCtx.URL+"/"),
		chromedp.Sleep(15*time.Second),
		chromedp.WaitVisible(`[lvt-source="items"]`),
	)
	if err != nil {
		t.Fatalf("Failed to navigate: %v", err)
	}

	// Get initial row count
	var initialRows int
	err = chromedp.Run(ctx,
		chromedp.Evaluate(`document.querySelectorAll('[lvt-source="items"] tbody tr').length`, &initialRows),
	)
	if err != nil {
		t.Fatalf("Failed to count initial rows: %v", err)
	}
	t.Logf("Initial row count: %d", initialRows)

	// Fill in the add form and submit
	err = chromedp.Run(ctx,
		chromedp.SetValue(`[lvt-source="items"] input[name="name"]`, "New Item", chromedp.ByQuery),
		chromedp.SetValue(`[lvt-source="items"] input[name="price"]`, "29.99", chromedp.ByQuery),
		chromedp.Click(`[lvt-source="items"] form[lvt-submit="Add"] button[type="submit"]`, chromedp.ByQuery),
		chromedp.Sleep(3*time.Second),
	)
	if err != nil {
		t.Fatalf("Failed to fill and submit form: %v", err)
	}

	// Verify row count increased
	var newRows int
	err = chromedp.Run(ctx,
		chromedp.Evaluate(`document.querySelectorAll('[lvt-source="items"] tbody tr').length`, &newRows),
	)
	if err != nil {
		t.Fatalf("Failed to count new rows: %v", err)
	}

	if newRows != initialRows+1 {
		var htmlContent string
		chromedp.Run(ctx, chromedp.OuterHTML(`[lvt-source="items"]`, &htmlContent))
		t.Logf("Table HTML: %s", htmlContent[:min(2000, len(htmlContent))])
		t.Logf("Console logs: %v", testCtx.ConsoleLogs.get())
		t.Fatalf("Expected %d rows after add, got %d", initialRows+1, newRows)
	}
	t.Logf("Row count increased from %d to %d after adding item", initialRows, newRows)
}

// TestAutoTables_RESTReadonly verifies a read-only REST source renders
// without CRUD controls.
func TestAutoTables_RESTReadonly(t *testing.T) {
	tempDir, tempCleanup := createAutoTablesRESTExample(t)
	defer tempCleanup()

	testCtx, cleanup := setupAutoTablesTest(t, tempDir)
	defer cleanup()

	ctx := testCtx.ChromeCtx.Context

	// Navigate and wait for content
	var hasAutoSource bool
	err := chromedp.Run(ctx,
		chromedp.Navigate(testCtx.URL+"/"),
		chromedp.Sleep(15*time.Second),
		chromedp.Evaluate(`document.querySelector('[lvt-source="users"]') !== null`, &hasAutoSource),
	)
	if err != nil {
		t.Fatalf("Failed to navigate: %v", err)
	}

	if !hasAutoSource {
		var htmlContent string
		chromedp.Run(ctx, chromedp.OuterHTML("html", &htmlContent))
		t.Logf("HTML (first 3000 chars): %s", htmlContent[:min(3000, len(htmlContent))])
		t.Logf("Console logs: %v", testCtx.ConsoleLogs.get())
		t.Fatal("Auto-generated table was not rendered")
	}

	// Verify refresh button exists
	var hasRefreshBtn bool
	err = chromedp.Run(ctx,
		chromedp.Evaluate(`document.querySelector('[lvt-source="users"] [lvt-click="Refresh"]') !== null`, &hasRefreshBtn),
	)
	if err != nil {
		t.Fatalf("Failed to check for refresh button: %v", err)
	}

	if !hasRefreshBtn {
		t.Fatal("Expected Refresh button for read-only source")
	}

	// Verify NO add form (read-only)
	var hasAddForm bool
	err = chromedp.Run(ctx,
		chromedp.Evaluate(`document.querySelector('[lvt-source="users"] form[lvt-submit="Add"]') !== null`, &hasAddForm),
	)
	if err != nil {
		t.Fatalf("Failed to check for add form: %v", err)
	}

	if hasAddForm {
		t.Fatal("Read-only source should NOT have an add form")
	}

	// Verify NO delete button (read-only)
	var hasDeleteBtn bool
	err = chromedp.Run(ctx,
		chromedp.Evaluate(`document.querySelector('[lvt-source="users"] [lvt-click="Delete"]') !== null`, &hasDeleteBtn),
	)
	if err != nil {
		t.Fatalf("Failed to check for delete button: %v", err)
	}

	if hasDeleteBtn {
		t.Fatal("Read-only source should NOT have delete buttons")
	}

	// Verify table has data (REST API should populate)
	var rowCount int
	err = chromedp.Run(ctx,
		chromedp.Evaluate(`document.querySelectorAll('[lvt-source="users"] tbody tr').length`, &rowCount),
	)
	if err != nil {
		t.Fatalf("Failed to count rows: %v", err)
	}

	if rowCount == 0 {
		t.Logf("Console logs: %v", testCtx.ConsoleLogs.get())
		t.Fatal("Expected rows from REST API, got 0")
	}
	t.Logf("REST API populated %d rows", rowCount)
}

// TestAutoTables_NoMatch verifies tables under headings that don't match any source
// remain as static markdown.
func TestAutoTables_NoMatch(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "auto-tables-nomatch-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Source is "products" but heading is "Orders" — no match
	content := `---
title: No Match Test
sources:
  products:
    type: sqlite
    db: ./test.db
    table: products
---

# No Match

## Orders
| Item | Quantity |
|------|----------|
| Widget | 5 |
`

	indexPath := filepath.Join(tempDir, "index.md")
	if err := os.WriteFile(indexPath, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write index.md: %v", err)
	}

	testCtx, cleanup := setupAutoTablesTest(t, tempDir)
	defer cleanup()

	ctx := testCtx.ChromeCtx.Context

	// Navigate
	err = chromedp.Run(ctx,
		chromedp.Navigate(testCtx.URL+"/"),
		chromedp.Sleep(10*time.Second),
	)
	if err != nil {
		t.Fatalf("Failed to navigate: %v", err)
	}

	// Should NOT have any lvt-source elements
	var hasLvtSource bool
	err = chromedp.Run(ctx,
		chromedp.Evaluate(`document.querySelector('[lvt-source]') !== null`, &hasLvtSource),
	)
	if err != nil {
		t.Fatalf("Failed to check for lvt-source: %v", err)
	}

	if hasLvtSource {
		t.Fatal("No-match table should NOT have lvt-source binding")
	}

	// The original table should be rendered as static HTML
	var hasTable bool
	err = chromedp.Run(ctx,
		chromedp.Evaluate(`document.querySelector('table') !== null`, &hasTable),
	)
	if err != nil {
		t.Fatalf("Failed to check for static table: %v", err)
	}

	if !hasTable {
		t.Fatal("Original static table should still be rendered")
	}
	t.Log("No-match table correctly rendered as static markdown")
}
