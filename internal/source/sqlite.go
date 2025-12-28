package source

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"sync"
	"time"

	_ "modernc.org/sqlite" // Pure Go SQLite driver
)

// SQLiteSource provides read/write access to SQLite tables.
// It implements WritableSource for Add, Update, Delete operations.
type SQLiteSource struct {
	name     string
	db       *sql.DB
	table    string
	dbPath   string
	readonly bool
	siteDir  string

	// Schema tracking
	columns   []string
	mu        sync.RWMutex
	hasSchema bool
}

// NewSQLiteSource creates a new SQLite source
func NewSQLiteSource(name, dbPath, table, siteDir string, readonly bool) (*SQLiteSource, error) {
	if table == "" {
		return nil, fmt.Errorf("sqlite source %q: table name is required", name)
	}

	// Validate table name (prevent SQL injection)
	if !isValidIdentifier(table) {
		return nil, fmt.Errorf("sqlite source %q: invalid table name %q", name, table)
	}

	// Default database path
	if dbPath == "" {
		dbPath = "./tinkerdown.db"
	}

	// Resolve relative path
	if !strings.HasPrefix(dbPath, "/") {
		dbPath = siteDir + "/" + dbPath
	}

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("sqlite source %q: failed to open database: %w", name, err)
	}

	// Test connection
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("sqlite source %q: failed to connect: %w", name, err)
	}

	s := &SQLiteSource{
		name:     name,
		db:       db,
		table:    table,
		dbPath:   dbPath,
		readonly: readonly,
		siteDir:  siteDir,
	}

	// Try to discover existing schema
	s.discoverSchema()

	return s, nil
}

// Name returns the source identifier
func (s *SQLiteSource) Name() string {
	return s.name
}

// Fetch retrieves all records from the table
func (s *SQLiteSource) Fetch(ctx context.Context) ([]map[string]interface{}, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if !s.hasSchema {
		// Table doesn't exist yet, return empty
		return []map[string]interface{}{}, nil
	}

	query := fmt.Sprintf("SELECT * FROM %s ORDER BY created_at DESC", s.table)
	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("sqlite source %q: fetch failed: %w", s.name, err)
	}
	defer rows.Close()

	columns, err := rows.Columns()
	if err != nil {
		return nil, err
	}

	var results []map[string]interface{}
	for rows.Next() {
		values := make([]interface{}, len(columns))
		valuePtrs := make([]interface{}, len(columns))
		for i := range values {
			valuePtrs[i] = &values[i]
		}

		if err := rows.Scan(valuePtrs...); err != nil {
			return nil, err
		}

		row := make(map[string]interface{})
		for i, col := range columns {
			row[col] = values[i]
		}
		results = append(results, row)
	}

	return results, rows.Err()
}

// Close releases the database connection
func (s *SQLiteSource) Close() error {
	if s.db != nil {
		return s.db.Close()
	}
	return nil
}

// IsReadonly returns whether the source is read-only
func (s *SQLiteSource) IsReadonly() bool {
	return s.readonly
}

// WriteItem performs write operations (add, update, delete)
func (s *SQLiteSource) WriteItem(ctx context.Context, action string, data map[string]interface{}) error {
	if s.readonly {
		return fmt.Errorf("sqlite source %q is read-only", s.name)
	}

	switch action {
	case "add":
		return s.addItem(ctx, data)
	case "update":
		return s.updateItem(ctx, data)
	case "delete":
		return s.deleteItem(ctx, data)
	default:
		return fmt.Errorf("sqlite source %q: unknown action %q", s.name, action)
	}
}

// addItem inserts a new record
func (s *SQLiteSource) addItem(ctx context.Context, data map[string]interface{}) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Ensure table exists with schema from data
	if err := s.ensureTable(data); err != nil {
		return err
	}

	// Filter out special fields
	fields := filterDataFields(data)
	if len(fields) == 0 {
		return fmt.Errorf("no fields to insert")
	}

	// Build INSERT statement
	columns := make([]string, 0, len(fields))
	placeholders := make([]string, 0, len(fields))
	values := make([]interface{}, 0, len(fields))

	for col, val := range fields {
		if !isValidIdentifier(col) {
			continue
		}
		columns = append(columns, col)
		placeholders = append(placeholders, "?")
		values = append(values, val)
	}

	query := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)",
		s.table,
		strings.Join(columns, ", "),
		strings.Join(placeholders, ", "))

	_, err := s.db.ExecContext(ctx, query, values...)
	return err
}

// updateItem updates an existing record by ID
func (s *SQLiteSource) updateItem(ctx context.Context, data map[string]interface{}) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	id, ok := getID(data)
	if !ok {
		return fmt.Errorf("update requires 'id' field")
	}

	fields := filterDataFields(data)
	delete(fields, "id")

	if len(fields) == 0 {
		return fmt.Errorf("no fields to update")
	}

	// Build UPDATE statement
	setClauses := make([]string, 0, len(fields))
	values := make([]interface{}, 0, len(fields)+1)

	for col, val := range fields {
		if !isValidIdentifier(col) {
			continue
		}
		setClauses = append(setClauses, fmt.Sprintf("%s = ?", col))
		values = append(values, val)
	}
	values = append(values, id)

	query := fmt.Sprintf("UPDATE %s SET %s WHERE id = ?",
		s.table,
		strings.Join(setClauses, ", "))

	_, err := s.db.ExecContext(ctx, query, values...)
	return err
}

// deleteItem removes a record by ID
func (s *SQLiteSource) deleteItem(ctx context.Context, data map[string]interface{}) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	id, ok := getID(data)
	if !ok {
		return fmt.Errorf("delete requires 'id' field")
	}

	query := fmt.Sprintf("DELETE FROM %s WHERE id = ?", s.table)
	_, err := s.db.ExecContext(ctx, query, id)
	return err
}

// ensureTable creates the table if it doesn't exist, based on data fields
func (s *SQLiteSource) ensureTable(data map[string]interface{}) error {
	if s.hasSchema {
		return nil
	}

	// Build CREATE TABLE from data fields
	var columnDefs []string
	columnDefs = append(columnDefs, "id INTEGER PRIMARY KEY AUTOINCREMENT")

	for col, val := range data {
		if col == "id" || !isValidIdentifier(col) {
			continue
		}
		sqlType := inferSQLType(val)
		columnDefs = append(columnDefs, fmt.Sprintf("%s %s", col, sqlType))
		s.columns = append(s.columns, col)
	}

	columnDefs = append(columnDefs, "created_at DATETIME DEFAULT CURRENT_TIMESTAMP")

	query := fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s (%s)",
		s.table,
		strings.Join(columnDefs, ", "))

	if _, err := s.db.Exec(query); err != nil {
		return fmt.Errorf("failed to create table: %w", err)
	}

	s.hasSchema = true
	return nil
}

// discoverSchema checks if the table exists and reads its columns
func (s *SQLiteSource) discoverSchema() {
	query := fmt.Sprintf("SELECT name FROM sqlite_master WHERE type='table' AND name=?")
	var name string
	err := s.db.QueryRow(query, s.table).Scan(&name)
	if err != nil {
		// Table doesn't exist
		return
	}

	// Get column names
	colQuery := fmt.Sprintf("PRAGMA table_info(%s)", s.table)
	rows, err := s.db.Query(colQuery)
	if err != nil {
		return
	}
	defer rows.Close()

	for rows.Next() {
		var cid int
		var name, typeName string
		var notNull, pk int
		var dfltValue interface{}
		if err := rows.Scan(&cid, &name, &typeName, &notNull, &dfltValue, &pk); err != nil {
			continue
		}
		if name != "id" && name != "created_at" {
			s.columns = append(s.columns, name)
		}
	}

	s.hasSchema = true
}

// Helper functions

func isValidIdentifier(name string) bool {
	if name == "" || len(name) > 64 {
		return false
	}
	for i, c := range name {
		if i == 0 {
			if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || c == '_') {
				return false
			}
		} else {
			if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '_') {
				return false
			}
		}
	}
	return true
}

func inferSQLType(val interface{}) string {
	switch val.(type) {
	case int, int32, int64, float64, float32:
		return "INTEGER"
	case bool:
		return "INTEGER" // SQLite uses INTEGER for boolean
	case time.Time:
		return "DATETIME"
	default:
		return "TEXT"
	}
}

func filterDataFields(data map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{})
	for k, v := range data {
		// Skip internal/special fields
		if strings.HasPrefix(k, "_") {
			continue
		}
		result[k] = v
	}
	return result
}

func getID(data map[string]interface{}) (interface{}, bool) {
	if id, ok := data["id"]; ok {
		return id, true
	}
	if id, ok := data["Id"]; ok {
		return id, true
	}
	if id, ok := data["ID"]; ok {
		return id, true
	}
	return nil, false
}
