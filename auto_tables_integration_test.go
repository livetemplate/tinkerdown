package tinkerdown_test

import (
	"database/sql"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/livetemplate/tinkerdown/internal/server"
	_ "modernc.org/sqlite"
)

// TestAutoTablesServerSide_SQLiteRenders tests the full server pipeline without
// a browser: create DB → write markdown → start server → HTTP GET → verify
// the page renders without errors and contains the auto-generated block.
//
// The lvt block content (lvt-source, lvt-click etc.) is delivered over WebSocket,
// not in the initial HTML. So we verify:
//   - The page returns 200 (no source fetch errors during discovery)
//   - The auto-generated block placeholder is present (data-tinkerdown-block)
//   - The original static markdown table was replaced (no |---| separator)
func TestAutoTablesServerSide_SQLiteRenders(t *testing.T) {
	tempDir := t.TempDir()

	// Create SQLite database with the schema the source expects (including created_at)
	dbPath := filepath.Join(tempDir, "test.db")
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("Failed to open SQLite: %v", err)
	}

	_, err = db.Exec(`CREATE TABLE items (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL,
		price TEXT NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	)`)
	if err != nil {
		db.Close()
		t.Fatalf("Failed to create table: %v", err)
	}

	_, err = db.Exec(`INSERT INTO items (name, price) VALUES ('Widget', '9.99'), ('Gadget', '19.99')`)
	if err != nil {
		db.Close()
		t.Fatalf("Failed to insert data: %v", err)
	}
	db.Close()

	content := `---
title: Integration Test
sources:
  items:
    type: sqlite
    db: ./test.db
    table: items
    readonly: false
---

# Integration Test

## Items
| Name | Price |
|------|-------|
`
	indexPath := filepath.Join(tempDir, "index.md")
	if err := os.WriteFile(indexPath, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write index.md: %v", err)
	}

	srv := server.New(tempDir)
	if err := srv.Discover(); err != nil {
		t.Fatalf("server.Discover() failed: %v", err)
	}

	ts := httptest.NewServer(srv)
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/")
	if err != nil {
		t.Fatalf("HTTP GET failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		t.Fatalf("Expected 200, got %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("Failed to read body: %v", err)
	}
	html := string(body)

	// The auto-table should have generated an lvt block (rendered as a tinkerdown-block)
	if !strings.Contains(html, "data-tinkerdown-block") {
		t.Error("Expected data-tinkerdown-block in rendered HTML (auto-table should generate lvt block)")
		t.Logf("HTML (first 2000 chars): %s", html[:min(2000, len(html))])
	}

	// The block type should be lvt
	if !strings.Contains(html, `data-block-type="lvt"`) {
		t.Error("Expected data-block-type=\"lvt\" for auto-generated block")
	}

	// The original markdown table separator should NOT be in the HTML
	// (it was replaced by the lvt code block)
	if strings.Contains(html, "|------|") {
		t.Error("Original markdown table separator should have been replaced by lvt block")
	}

	// The heading should still be present
	if !strings.Contains(html, "Items") {
		t.Error("Heading 'Items' should be preserved in rendered HTML")
	}
}

// TestAutoTablesServerSide_NoMatch verifies tables that don't match any source
// render as normal static tables.
func TestAutoTablesServerSide_NoMatch(t *testing.T) {
	tempDir := t.TempDir()

	jsonPath := filepath.Join(tempDir, "data.json")
	if err := os.WriteFile(jsonPath, []byte(`[{"name":"x"}]`), 0644); err != nil {
		t.Fatalf("Failed to write data.json: %v", err)
	}

	content := `---
title: No Match
sources:
  products:
    type: json
    file: ./data.json
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

	srv := server.New(tempDir)
	if err := srv.Discover(); err != nil {
		t.Fatalf("server.Discover() failed: %v", err)
	}

	ts := httptest.NewServer(srv)
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/")
	if err != nil {
		t.Fatalf("HTTP GET failed: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("Failed to read body: %v", err)
	}
	html := string(body)

	// Static table data should render normally
	if !strings.Contains(html, "Widget") {
		t.Error("Static table data 'Widget' should be in rendered HTML")
	}

	// Should have normal <table> with static content, not an lvt block
	if !strings.Contains(html, "<table>") {
		t.Error("Expected a static <table> element for unmatched section")
	}
}

// TestAutoTablesServerSide_MissingColumn tests that a SQLite table without
// the expected created_at column produces a clear error rather than silently breaking.
// This is the exact bug we hit: the SQLite source does ORDER BY created_at DESC.
func TestAutoTablesServerSide_MissingColumn(t *testing.T) {
	tempDir := t.TempDir()

	// Deliberately create table WITHOUT created_at to reproduce the bug
	dbPath := filepath.Join(tempDir, "test.db")
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("Failed to open SQLite: %v", err)
	}

	_, err = db.Exec(`CREATE TABLE items (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL
	)`)
	if err != nil {
		db.Close()
		t.Fatalf("Failed to create table: %v", err)
	}
	_, err = db.Exec(`INSERT INTO items (name) VALUES ('Test')`)
	if err != nil {
		db.Close()
		t.Fatalf("Failed to insert: %v", err)
	}
	db.Close()

	content := `---
title: Missing Column Test
sources:
  items:
    type: sqlite
    db: ./test.db
    table: items
    readonly: false
---

# Test

## Items
| Name |
|------|
`
	indexPath := filepath.Join(tempDir, "index.md")
	if err := os.WriteFile(indexPath, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write index.md: %v", err)
	}

	srv := server.New(tempDir)
	if err := srv.Discover(); err != nil {
		t.Fatalf("server.Discover() failed: %v", err)
	}

	ts := httptest.NewServer(srv)
	defer ts.Close()

	// The page should still return 200 (errors are shown in the UI, not as HTTP errors)
	resp, err := http.Get(ts.URL + "/")
	if err != nil {
		t.Fatalf("HTTP GET failed: %v", err)
	}
	defer resp.Body.Close()

	// After the fix, the page should render successfully even without created_at.
	// The SQLite source now conditionally uses ORDER BY created_at DESC only when
	// the column exists.
	if resp.StatusCode != 200 {
		t.Fatalf("Expected 200, got %d", resp.StatusCode)
	}
}
