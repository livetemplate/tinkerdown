package source

import (
	"database/sql"
	"path/filepath"
	"testing"

	_ "modernc.org/sqlite"
)

func TestQuerySQLiteSchema(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatal(err)
	}
	_, err = db.Exec(`CREATE TABLE items (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL,
		price REAL NOT NULL,
		done BOOLEAN DEFAULT 0,
		due DATE,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	)`)
	if err != nil {
		t.Fatal(err)
	}
	db.Close()

	schema := QuerySQLiteSchema(dbPath, "items", dir)
	if schema == nil {
		t.Fatal("expected non-nil schema")
	}

	// Should exclude id and created_at
	for _, col := range schema {
		if col.Name == "id" || col.Name == "created_at" {
			t.Errorf("internal column %q should be excluded", col.Name)
		}
	}

	// Check types
	typeMap := make(map[string]string)
	for _, col := range schema {
		typeMap[col.Name] = col.Type
	}

	if typeMap["name"] != "text" {
		t.Errorf("expected name type 'text', got %q", typeMap["name"])
	}
	if typeMap["price"] != "real" {
		t.Errorf("expected price type 'real', got %q", typeMap["price"])
	}
	if typeMap["done"] != "boolean" {
		t.Errorf("expected done type 'boolean', got %q", typeMap["done"])
	}
	if typeMap["due"] != "date" {
		t.Errorf("expected due type 'date', got %q", typeMap["due"])
	}
}

func TestQuerySQLiteSchema_NonexistentTable(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatal(err)
	}
	db.Close()

	// Table doesn't exist — should return nil gracefully
	schema := QuerySQLiteSchema(dbPath, "nonexistent", dir)
	if schema != nil {
		t.Errorf("expected nil for nonexistent table, got %v", schema)
	}
}

func TestQuerySQLiteSchema_NonexistentDB(t *testing.T) {
	// DB doesn't exist — should return nil gracefully
	schema := QuerySQLiteSchema("/nonexistent/path.db", "items", "/nonexistent")
	if schema != nil {
		t.Errorf("expected nil for nonexistent DB, got %v", schema)
	}
}

func TestQuerySQLiteSchema_Required(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatal(err)
	}
	_, err = db.Exec(`CREATE TABLE items (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL,
		notes TEXT
	)`)
	if err != nil {
		t.Fatal(err)
	}
	db.Close()

	schema := QuerySQLiteSchema(dbPath, "items", dir)

	reqMap := make(map[string]bool)
	for _, col := range schema {
		reqMap[col.Name] = col.Required
	}

	if !reqMap["name"] {
		t.Error("expected name to be required (NOT NULL)")
	}
	if reqMap["notes"] {
		t.Error("expected notes to be optional (nullable)")
	}
}

func TestInputTypeForColumn(t *testing.T) {
	tests := []struct {
		colType string
		expect  string
	}{
		{"text", "text"},
		{"integer", "number"},
		{"real", "number"},
		{"boolean", "checkbox"},
		{"date", "date"},
		{"datetime", "datetime-local"},
		{"unknown", "text"},
		{"", "text"},
	}

	for _, tt := range tests {
		got := InputTypeForColumn(tt.colType)
		if got != tt.expect {
			t.Errorf("InputTypeForColumn(%q) = %q, want %q", tt.colType, got, tt.expect)
		}
	}
}
