package commands

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// setupTestProject creates a test project directory with a SQLite source config
func setupTestProject(t *testing.T) (string, func()) {
	t.Helper()
	tmpDir := t.TempDir()

	// Create tinkerdown.yaml with a SQLite source
	configContent := `title: Test Project
sources:
  tasks:
    type: sqlite
    db: ./test.db
    table: tasks
    readonly: false
  readonly_tasks:
    type: sqlite
    db: ./test.db
    table: readonly_tasks
    readonly: true
`
	configPath := filepath.Join(tmpDir, "tinkerdown.yaml")
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to write config: %v", err)
	}

	cleanup := func() {
		// Remove test database if created
		os.Remove(filepath.Join(tmpDir, "test.db"))
	}

	return tmpDir, cleanup
}

func TestCLICommandUsage(t *testing.T) {
	// Test that CLI command returns usage error when not enough args
	err := CLICommand([]string{})
	if err == nil {
		t.Fatal("Expected error for missing arguments")
	}
	if !strings.Contains(err.Error(), "usage:") {
		t.Errorf("Expected usage message, got: %v", err)
	}
}

func TestCLICommandInvalidPath(t *testing.T) {
	err := CLICommand([]string{"/nonexistent/path", "list", "tasks"})
	if err == nil {
		t.Fatal("Expected error for invalid path")
	}
	if !strings.Contains(err.Error(), "does not exist") {
		t.Errorf("Expected 'does not exist' error, got: %v", err)
	}
}

func TestCLICommandNoConfig(t *testing.T) {
	tmpDir := t.TempDir()

	err := CLICommand([]string{tmpDir, "list", "tasks"})
	if err == nil {
		t.Fatal("Expected error for missing config")
	}
	if !strings.Contains(err.Error(), "no sources defined") {
		t.Errorf("Expected 'no sources defined' error, got: %v", err)
	}
}

func TestCLICommandSourceNotFound(t *testing.T) {
	tmpDir, cleanup := setupTestProject(t)
	defer cleanup()

	err := CLICommand([]string{tmpDir, "list", "nonexistent"})
	if err == nil {
		t.Fatal("Expected error for nonexistent source")
	}
	if !strings.Contains(err.Error(), "source \"nonexistent\" not found") {
		t.Errorf("Expected 'source not found' error, got: %v", err)
	}
}

func TestCLICommandInvalidAction(t *testing.T) {
	tmpDir, cleanup := setupTestProject(t)
	defer cleanup()

	err := CLICommand([]string{tmpDir, "invalid", "tasks"})
	if err == nil {
		t.Fatal("Expected error for invalid action")
	}
	if !strings.Contains(err.Error(), "unknown action") {
		t.Errorf("Expected 'unknown action' error, got: %v", err)
	}
}

func TestCLIListEmpty(t *testing.T) {
	tmpDir, cleanup := setupTestProject(t)
	defer cleanup()

	// Capture stdout
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := CLICommand([]string{tmpDir, "list", "tasks"})

	w.Close()
	os.Stdout = old

	if err != nil {
		t.Fatalf("CLICommand failed: %v", err)
	}

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	if !strings.Contains(output, "No items found") {
		t.Errorf("Expected 'No items found' message, got: %s", output)
	}
}

func TestCLIAddAndList(t *testing.T) {
	tmpDir, cleanup := setupTestProject(t)
	defer cleanup()

	// Add an item
	err := CLICommand([]string{tmpDir, "add", "tasks", "--text=Test Task", "--priority=1"})
	if err != nil {
		t.Fatalf("Add failed: %v", err)
	}

	// List items with JSON format
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err = CLICommand([]string{tmpDir, "list", "tasks", "--format=json"})

	w.Close()
	os.Stdout = old

	if err != nil {
		t.Fatalf("List failed: %v", err)
	}

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	// Parse JSON output
	var items []map[string]interface{}
	if err := json.Unmarshal([]byte(output), &items); err != nil {
		t.Fatalf("Failed to parse JSON output: %v\nOutput: %s", err, output)
	}

	if len(items) != 1 {
		t.Fatalf("Expected 1 item, got %d", len(items))
	}

	if items[0]["text"] != "Test Task" {
		t.Errorf("Expected text 'Test Task', got %v", items[0]["text"])
	}
}

func TestCLIListTableFormat(t *testing.T) {
	tmpDir, cleanup := setupTestProject(t)
	defer cleanup()

	// Add a task
	err := CLICommand([]string{tmpDir, "add", "tasks", "--text=Task One"})
	if err != nil {
		t.Fatalf("Add failed: %v", err)
	}

	// List with table format (default)
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err = CLICommand([]string{tmpDir, "list", "tasks", "--format=table"})

	w.Close()
	os.Stdout = old

	if err != nil {
		t.Fatalf("List failed: %v", err)
	}

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	// Table output should have headers and separators
	if !strings.Contains(output, "id") {
		t.Errorf("Expected 'id' column header, got: %s", output)
	}
	if !strings.Contains(output, "text") {
		t.Errorf("Expected 'text' column header, got: %s", output)
	}
	if !strings.Contains(output, "---") {
		t.Errorf("Expected table separator, got: %s", output)
	}
	if !strings.Contains(output, "1 item(s)") {
		t.Errorf("Expected item count, got: %s", output)
	}
}

func TestCLIListCSVFormat(t *testing.T) {
	tmpDir, cleanup := setupTestProject(t)
	defer cleanup()

	// Add a task
	err := CLICommand([]string{tmpDir, "add", "tasks", "--text=Task CSV"})
	if err != nil {
		t.Fatalf("Add failed: %v", err)
	}

	// List with CSV format
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err = CLICommand([]string{tmpDir, "list", "tasks", "--format=csv"})

	w.Close()
	os.Stdout = old

	if err != nil {
		t.Fatalf("List failed: %v", err)
	}

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	lines := strings.Split(strings.TrimSpace(output), "\n")
	if len(lines) < 2 {
		t.Fatalf("Expected at least 2 lines (header + data), got %d", len(lines))
	}

	// First line should be headers
	headers := lines[0]
	if !strings.Contains(headers, "id") {
		t.Errorf("Expected 'id' in CSV headers, got: %s", headers)
	}
	if !strings.Contains(headers, "text") {
		t.Errorf("Expected 'text' in CSV headers, got: %s", headers)
	}
}

func TestCLIListFilter(t *testing.T) {
	tmpDir, cleanup := setupTestProject(t)
	defer cleanup()

	// Add multiple tasks
	CLICommand([]string{tmpDir, "add", "tasks", "--text=Task A", "--done=false"})
	CLICommand([]string{tmpDir, "add", "tasks", "--text=Task B", "--done=true"})

	// List with filter
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := CLICommand([]string{tmpDir, "list", "tasks", "--format=json", "--filter=done=true"})

	w.Close()
	os.Stdout = old

	if err != nil {
		t.Fatalf("List failed: %v", err)
	}

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	var items []map[string]interface{}
	if err := json.Unmarshal([]byte(output), &items); err != nil {
		t.Fatalf("Failed to parse JSON: %v", err)
	}

	// Should only have the done=true item
	if len(items) != 1 {
		t.Errorf("Expected 1 filtered item, got %d", len(items))
	}
}

func TestCLIUpdate(t *testing.T) {
	tmpDir, cleanup := setupTestProject(t)
	defer cleanup()

	// Add a task
	err := CLICommand([]string{tmpDir, "add", "tasks", "--text=Original Text"})
	if err != nil {
		t.Fatalf("Add failed: %v", err)
	}

	// Get the item ID
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	CLICommand([]string{tmpDir, "list", "tasks", "--format=json"})

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	buf.ReadFrom(r)

	var items []map[string]interface{}
	json.Unmarshal(buf.Bytes(), &items)

	if len(items) == 0 {
		t.Fatal("No items found")
	}

	// Get ID (should be integer or string)
	itemID := items[0]["id"]
	var idStr string
	switch v := itemID.(type) {
	case float64:
		idStr = "1" // First auto-generated ID
	case string:
		idStr = v
	}

	// Update the task
	err = CLICommand([]string{tmpDir, "update", "tasks", "--id=" + idStr, "--text=Updated Text"})
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}

	// Verify update
	old = os.Stdout
	r, w, _ = os.Pipe()
	os.Stdout = w

	CLICommand([]string{tmpDir, "list", "tasks", "--format=json"})

	w.Close()
	os.Stdout = old

	buf.Reset()
	buf.ReadFrom(r)

	json.Unmarshal(buf.Bytes(), &items)

	if items[0]["text"] != "Updated Text" {
		t.Errorf("Expected 'Updated Text', got %v", items[0]["text"])
	}
}

func TestCLIUpdateRequiresID(t *testing.T) {
	tmpDir, cleanup := setupTestProject(t)
	defer cleanup()

	err := CLICommand([]string{tmpDir, "update", "tasks", "--text=New Text"})
	if err == nil {
		t.Fatal("Expected error when ID is missing")
	}
	if !strings.Contains(err.Error(), "--id is required") {
		t.Errorf("Expected '--id is required' error, got: %v", err)
	}
}

func TestCLIUpdateRequiresFields(t *testing.T) {
	tmpDir, cleanup := setupTestProject(t)
	defer cleanup()

	err := CLICommand([]string{tmpDir, "update", "tasks", "--id=1"})
	if err == nil {
		t.Fatal("Expected error when no fields provided")
	}
	if !strings.Contains(err.Error(), "no fields to update") {
		t.Errorf("Expected 'no fields to update' error, got: %v", err)
	}
}

func TestCLIDeleteWithConfirmation(t *testing.T) {
	tmpDir, cleanup := setupTestProject(t)
	defer cleanup()

	// Add a task
	CLICommand([]string{tmpDir, "add", "tasks", "--text=To Delete"})

	// Delete with -y flag (skip confirmation)
	err := CLICommand([]string{tmpDir, "delete", "tasks", "--id=1", "-y"})
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	// Verify deletion
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	CLICommand([]string{tmpDir, "list", "tasks", "--format=json"})

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	buf.ReadFrom(r)

	var items []map[string]interface{}
	json.Unmarshal(buf.Bytes(), &items)

	if len(items) != 0 {
		t.Errorf("Expected 0 items after delete, got %d", len(items))
	}
}

func TestCLIDeleteRequiresID(t *testing.T) {
	tmpDir, cleanup := setupTestProject(t)
	defer cleanup()

	err := CLICommand([]string{tmpDir, "delete", "tasks", "-y"})
	if err == nil {
		t.Fatal("Expected error when ID is missing")
	}
	if !strings.Contains(err.Error(), "--id is required") {
		t.Errorf("Expected '--id is required' error, got: %v", err)
	}
}

func TestCLIAddRequiresFields(t *testing.T) {
	tmpDir, cleanup := setupTestProject(t)
	defer cleanup()

	err := CLICommand([]string{tmpDir, "add", "tasks"})
	if err == nil {
		t.Fatal("Expected error when no fields provided")
	}
	if !strings.Contains(err.Error(), "no fields provided") {
		t.Errorf("Expected 'no fields provided' error, got: %v", err)
	}
}

func TestCLIReadonlySource(t *testing.T) {
	tmpDir, cleanup := setupTestProject(t)
	defer cleanup()

	err := CLICommand([]string{tmpDir, "add", "readonly_tasks", "--text=Test"})
	if err == nil {
		t.Fatal("Expected error for readonly source")
	}
	if !strings.Contains(err.Error(), "read-only") {
		t.Errorf("Expected 'read-only' error, got: %v", err)
	}
}

func TestCLIWithMarkdownFilePath(t *testing.T) {
	tmpDir, cleanup := setupTestProject(t)
	defer cleanup()

	// Create an index.md file
	indexContent := `# Test
Some content
`
	indexPath := filepath.Join(tmpDir, "index.md")
	if err := os.WriteFile(indexPath, []byte(indexContent), 0644); err != nil {
		t.Fatalf("Failed to write index.md: %v", err)
	}

	// Test that CLI works with .md file path (should use directory)
	err := CLICommand([]string{indexPath, "add", "tasks", "--text=From MD Path"})
	if err != nil {
		t.Fatalf("CLI with .md path failed: %v", err)
	}
}

func TestParseValue(t *testing.T) {
	tests := []struct {
		input    string
		expected interface{}
	}{
		{"true", true},
		{"false", false},
		{"123", int64(123)},
		{"-456", int64(-456)},
		{"3.14", 3.14},
		{"hello", "hello"},
		{"", ""},
	}

	for _, tc := range tests {
		result := parseValue(tc.input)
		if result != tc.expected {
			t.Errorf("parseValue(%q) = %v (%T), want %v (%T)", tc.input, result, result, tc.expected, tc.expected)
		}
	}
}

func TestParseFlags(t *testing.T) {
	args := []string{
		"--format=json",
		"--filter=done=true",
		"--id=abc123",
		"--text=Hello World",
		"--count=42",
		"--active=true",
		"-y",
	}

	opts := parseFlags(args)

	if opts.format != "json" {
		t.Errorf("Expected format 'json', got %s", opts.format)
	}
	if opts.filter != "done=true" {
		t.Errorf("Expected filter 'done=true', got %s", opts.filter)
	}
	if opts.id != "abc123" {
		t.Errorf("Expected id 'abc123', got %s", opts.id)
	}
	if !opts.yes {
		t.Error("Expected yes to be true")
	}
	if opts.fields["text"] != "Hello World" {
		t.Errorf("Expected text 'Hello World', got %v", opts.fields["text"])
	}
	if opts.fields["count"] != int64(42) {
		t.Errorf("Expected count 42, got %v", opts.fields["count"])
	}
	if opts.fields["active"] != true {
		t.Errorf("Expected active true, got %v", opts.fields["active"])
	}
}

func TestApplyFilter(t *testing.T) {
	data := []map[string]interface{}{
		{"id": 1, "name": "Alice", "active": true},
		{"id": 2, "name": "Bob", "active": false},
		{"id": 3, "name": "Charlie", "active": true},
	}

	// Test equality filter
	result := applyFilter(data, "active=true")
	if len(result) != 2 {
		t.Errorf("Expected 2 items with active=true, got %d", len(result))
	}

	// Test inequality filter
	result = applyFilter(data, "active!=true")
	if len(result) != 1 {
		t.Errorf("Expected 1 item with active!=true, got %d", len(result))
	}

	// Test string filter
	result = applyFilter(data, "name=Bob")
	if len(result) != 1 {
		t.Errorf("Expected 1 item with name=Bob, got %d", len(result))
	}

	// Test empty filter (returns all)
	result = applyFilter(data, "")
	if len(result) != 3 {
		t.Errorf("Expected 3 items with empty filter, got %d", len(result))
	}
}

func TestOutputTable(t *testing.T) {
	data := []map[string]interface{}{
		{"id": 1, "text": "Task 1"},
		{"id": 2, "text": "Task 2"},
	}

	// Capture output
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := outputTable(data)

	w.Close()
	os.Stdout = old

	if err != nil {
		t.Fatalf("outputTable failed: %v", err)
	}

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	// Check table structure
	if !strings.Contains(output, "id") {
		t.Error("Expected 'id' column")
	}
	if !strings.Contains(output, "text") {
		t.Error("Expected 'text' column")
	}
	if !strings.Contains(output, "Task 1") {
		t.Error("Expected 'Task 1' in output")
	}
	if !strings.Contains(output, "2 item(s)") {
		t.Error("Expected '2 item(s)' count")
	}
}

func TestOutputJSON(t *testing.T) {
	data := []map[string]interface{}{
		{"id": 1, "text": "Task 1"},
	}

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := outputJSON(data)

	w.Close()
	os.Stdout = old

	if err != nil {
		t.Fatalf("outputJSON failed: %v", err)
	}

	var buf bytes.Buffer
	buf.ReadFrom(r)

	var result []map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("Failed to parse JSON output: %v", err)
	}

	if len(result) != 1 {
		t.Errorf("Expected 1 item, got %d", len(result))
	}
}

func TestOutputCSV(t *testing.T) {
	data := []map[string]interface{}{
		{"id": 1, "text": "Task 1"},
		{"id": 2, "text": "Task 2"},
	}

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := outputCSV(data)

	w.Close()
	os.Stdout = old

	if err != nil {
		t.Fatalf("outputCSV failed: %v", err)
	}

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	lines := strings.Split(strings.TrimSpace(output), "\n")
	if len(lines) != 3 { // header + 2 data rows
		t.Errorf("Expected 3 lines, got %d", len(lines))
	}
}

// cliListContext tests the cliList function directly with context
func TestCLIListWithContext(t *testing.T) {
	ctx := context.Background()

	// Create a mock source for testing
	mockData := []map[string]interface{}{
		{"id": 1, "text": "Test item"},
	}

	src := &mockSource{data: mockData}
	opts := cliOptions{format: "json"}

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := cliList(ctx, src, opts)

	w.Close()
	os.Stdout = old

	if err != nil {
		t.Fatalf("cliList failed: %v", err)
	}

	var buf bytes.Buffer
	buf.ReadFrom(r)

	var result []map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("Failed to parse output: %v", err)
	}

	if len(result) != 1 {
		t.Errorf("Expected 1 item, got %d", len(result))
	}
}

// mockSource implements source.Source for testing
type mockSource struct {
	data     []map[string]interface{}
	readonly bool
}

func (m *mockSource) Name() string { return "mock" }

func (m *mockSource) Fetch(ctx context.Context) ([]map[string]interface{}, error) {
	return m.data, nil
}

func (m *mockSource) Close() error { return nil }
