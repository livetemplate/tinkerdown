package source

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestNewMarkdownSource(t *testing.T) {
	tests := []struct {
		name        string
		sourceName  string
		file        string
		anchor      string
		siteDir     string
		currentFile string
		readonly    bool
		wantErr     bool
		wantAnchor  string
	}{
		{
			name:        "valid source with # anchor",
			sourceName:  "todos",
			file:        "data/tasks.md",
			anchor:      "#tasks",
			siteDir:     "/site",
			currentFile: "/site/index.md",
			readonly:    true,
			wantErr:     false,
			wantAnchor:  "#tasks",
		},
		{
			name:        "valid source without # in anchor",
			sourceName:  "todos",
			file:        "data/tasks.md",
			anchor:      "tasks",
			siteDir:     "/site",
			currentFile: "/site/index.md",
			readonly:    true,
			wantErr:     false,
			wantAnchor:  "#tasks",
		},
		{
			name:        "missing file",
			sourceName:  "todos",
			file:        "",
			anchor:      "#tasks",
			siteDir:     "/site",
			currentFile: "/site/index.md",
			readonly:    true,
			wantErr:     true,
		},
		{
			name:        "missing anchor",
			sourceName:  "todos",
			file:        "data/tasks.md",
			anchor:      "",
			siteDir:     "/site",
			currentFile: "/site/index.md",
			readonly:    true,
			wantErr:     true,
		},
		{
			name:        "external file",
			sourceName:  "data",
			file:        "data/items.md",
			anchor:      "#items",
			siteDir:     "/site",
			currentFile: "/site/index.md",
			readonly:    false,
			wantErr:     false,
			wantAnchor:  "#items",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			src, err := NewMarkdownSource(tt.sourceName, tt.file, tt.anchor, tt.siteDir, tt.currentFile, tt.readonly)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewMarkdownSource() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil {
				return
			}
			if src.GetAnchor() != tt.wantAnchor {
				t.Errorf("anchor = %q, want %q", src.GetAnchor(), tt.wantAnchor)
			}
			if src.Name() != tt.sourceName {
				t.Errorf("Name() = %q, want %q", src.Name(), tt.sourceName)
			}
			if src.IsReadonly() != tt.readonly {
				t.Errorf("IsReadonly() = %v, want %v", src.IsReadonly(), tt.readonly)
			}
		})
	}
}

func TestMarkdownSourceParseTaskList(t *testing.T) {
	content := `# Tasks {#tasks}

- [ ] Buy groceries <!-- id:task1 -->
- [x] Clean house <!-- id:task2 -->
- [ ] Walk the dog <!-- id:task3 -->
- [X] Send email

## Other Section
`
	src := &MarkdownSource{anchor: "#tasks"}
	results, err := src.parseSection(content)
	if err != nil {
		t.Fatalf("parseSection() error = %v", err)
	}

	if len(results) != 4 {
		t.Fatalf("expected 4 tasks, got %d", len(results))
	}

	// Check first task
	if results[0]["text"] != "Buy groceries" {
		t.Errorf("task 0 text = %q, want %q", results[0]["text"], "Buy groceries")
	}
	if results[0]["done"] != false {
		t.Errorf("task 0 done = %v, want false", results[0]["done"])
	}
	if results[0]["id"] != "task1" {
		t.Errorf("task 0 id = %q, want %q", results[0]["id"], "task1")
	}

	// Check completed task
	if results[1]["done"] != true {
		t.Errorf("task 1 done = %v, want true", results[1]["done"])
	}

	// Check uppercase X
	if results[3]["done"] != true {
		t.Errorf("task 3 done = %v, want true (uppercase X)", results[3]["done"])
	}

	// Task without ID should have auto-generated ID
	if results[3]["id"] == "" {
		t.Error("task 3 should have auto-generated ID")
	}
}

func TestMarkdownSourceParseBulletList(t *testing.T) {
	content := `# Bookmarks {#bookmarks}

- Example Site <!-- id:bm1 -->
- Another Site <!-- id:bm2 -->
- Third Item
`
	src := &MarkdownSource{anchor: "#bookmarks"}
	results, err := src.parseSection(content)
	if err != nil {
		t.Fatalf("parseSection() error = %v", err)
	}

	if len(results) != 3 {
		t.Fatalf("expected 3 items, got %d", len(results))
	}

	if results[0]["text"] != "Example Site" {
		t.Errorf("item 0 text = %q, want %q", results[0]["text"], "Example Site")
	}
	if results[0]["id"] != "bm1" {
		t.Errorf("item 0 id = %q, want %q", results[0]["id"], "bm1")
	}

	// Item without ID should have auto-generated ID
	if results[2]["id"] == "" {
		t.Error("item 2 should have auto-generated ID")
	}
}

func TestMarkdownSourceParseTable(t *testing.T) {
	content := `# Products {#products}

| Name | Price | Stock |
|------|-------|-------|
| Widget | $19.99 | 100 | <!-- id:p1 -->
| Gadget | $29.99 | 50 | <!-- id:p2 -->
| Thing | $9.99 | 200 |

Some other content
`
	src := &MarkdownSource{anchor: "#products"}
	results, err := src.parseSection(content)
	if err != nil {
		t.Fatalf("parseSection() error = %v", err)
	}

	if len(results) != 3 {
		t.Fatalf("expected 3 rows, got %d", len(results))
	}

	// Check first row
	if results[0]["Name"] != "Widget" {
		t.Errorf("row 0 Name = %q, want %q", results[0]["Name"], "Widget")
	}
	if results[0]["Price"] != "$19.99" {
		t.Errorf("row 0 Price = %q, want %q", results[0]["Price"], "$19.99")
	}
	if results[0]["Stock"] != "100" {
		t.Errorf("row 0 Stock = %q, want %q", results[0]["Stock"], "100")
	}
	if results[0]["id"] != "p1" {
		t.Errorf("row 0 id = %q, want %q", results[0]["id"], "p1")
	}

	// Row without ID should have auto-generated ID
	if results[2]["id"] == "" {
		t.Error("row 2 should have auto-generated ID")
	}
}

func TestMarkdownSourceSectionBoundary(t *testing.T) {
	content := `# Header 1 {#header1}

Task list in section 1:
- [ ] Task A <!-- id:a -->
- [ ] Task B <!-- id:b -->

# Header 2 {#header2}

Different content:
- Item X
- Item Y
`
	// Parse first section
	src1 := &MarkdownSource{anchor: "#header1"}
	results1, err := src1.parseSection(content)
	if err != nil {
		t.Fatalf("parseSection(header1) error = %v", err)
	}

	if len(results1) != 2 {
		t.Fatalf("header1: expected 2 tasks, got %d", len(results1))
	}
	if results1[0]["id"] != "a" {
		t.Errorf("header1 task 0 id = %q, want %q", results1[0]["id"], "a")
	}

	// Parse second section
	src2 := &MarkdownSource{anchor: "#header2"}
	results2, err := src2.parseSection(content)
	if err != nil {
		t.Fatalf("parseSection(header2) error = %v", err)
	}

	if len(results2) != 2 {
		t.Fatalf("header2: expected 2 items, got %d", len(results2))
	}
}

func TestMarkdownSourceNestedHeaders(t *testing.T) {
	content := `# Top Level {#top}

- Top item <!-- id:top1 -->

## Nested {#nested}

- Nested item <!-- id:nest1 -->

### Deep Nested {#deep}

- Deep item <!-- id:deep1 -->

## Another Nested {#another}

- Another item <!-- id:another1 -->
`
	// Test that nested section only includes content until next same-level header
	src := &MarkdownSource{anchor: "#nested"}
	results, err := src.parseSection(content)
	if err != nil {
		t.Fatalf("parseSection() error = %v", err)
	}

	// Should include nested and deep items, but NOT another (same level)
	if len(results) != 2 {
		t.Fatalf("expected 2 items (nested + deep), got %d", len(results))
	}
}

func TestMarkdownSourceNoSection(t *testing.T) {
	content := `# Some Content

No matching anchor here.
`
	src := &MarkdownSource{anchor: "#missing"}
	results, err := src.parseSection(content)
	if err != nil {
		t.Fatalf("parseSection() error = %v", err)
	}

	if len(results) != 0 {
		t.Errorf("expected empty results for missing section, got %d", len(results))
	}
}

func TestMarkdownSourceEmptySection(t *testing.T) {
	content := `# Empty Section {#empty}

# Next Section {#next}

- Some item
`
	src := &MarkdownSource{anchor: "#empty"}
	results, err := src.parseSection(content)
	if err != nil {
		t.Fatalf("parseSection() error = %v", err)
	}

	if len(results) != 0 {
		t.Errorf("expected empty results for empty section, got %d", len(results))
	}
}

func TestScanMarkdownForIDs(t *testing.T) {
	content := `# Tasks {#tasks}

- [ ] Task 1 <!-- id:abc123 -->
- [x] Task 2 <!-- id:def456 -->
- [ ] Task 3 <!-- id:ghi789 -->

| Name | Value |
|------|-------|
| A | 1 | <!-- id:row1 -->
`
	ids := ScanMarkdownForIDs(content)

	if len(ids) != 4 {
		t.Fatalf("expected 4 IDs, got %d: %v", len(ids), ids)
	}

	expected := []string{"abc123", "def456", "ghi789", "row1"}
	for i, id := range expected {
		if ids[i] != id {
			t.Errorf("id[%d] = %q, want %q", i, ids[i], id)
		}
	}
}

func TestEnsureUniqueIDs(t *testing.T) {
	content := `- [ ] Task 1 <!-- id:dup -->
- [ ] Task 2 <!-- id:dup -->
- [ ] Task 3 <!-- id:unique -->
`
	result, modified := EnsureUniqueIDs(content)
	if !modified {
		t.Error("expected content to be modified due to duplicate ID")
	}

	ids := ScanMarkdownForIDs(result)
	seen := make(map[string]bool)
	for _, id := range ids {
		if seen[id] {
			t.Errorf("found duplicate ID after EnsureUniqueIDs: %s", id)
		}
		seen[id] = true
	}
}

func TestEnsureUniqueIDsNoChanges(t *testing.T) {
	content := `- [ ] Task 1 <!-- id:a -->
- [ ] Task 2 <!-- id:b -->
- [ ] Task 3 <!-- id:c -->
`
	_, modified := EnsureUniqueIDs(content)
	if modified {
		t.Error("expected no modification when all IDs are unique")
	}
}

func TestAddIDsToItems(t *testing.T) {
	tests := []struct {
		name         string
		content      string
		wantModified bool
		wantIDCount  int
	}{
		{
			name: "task list without IDs",
			content: `- [ ] Task 1
- [x] Task 2
`,
			wantModified: true,
			wantIDCount:  2,
		},
		{
			name: "task list with existing IDs",
			content: `- [ ] Task 1 <!-- id:existing -->
- [x] Task 2 <!-- id:another -->
`,
			wantModified: false,
			wantIDCount:  2,
		},
		{
			name: "mixed with and without IDs",
			content: `- [ ] Task 1 <!-- id:existing -->
- [x] Task 2
`,
			wantModified: true,
			wantIDCount:  2,
		},
		{
			name: "bullet list without IDs",
			content: `- Item 1
- Item 2
`,
			wantModified: true,
			wantIDCount:  2,
		},
		{
			name: "table without IDs",
			content: `| Name | Value |
|------|-------|
| A | 1 |
| B | 2 |
`,
			wantModified: true,
			wantIDCount:  2, // Only data rows, not header
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, modified := AddIDsToItems(tt.content)
			if modified != tt.wantModified {
				t.Errorf("modified = %v, want %v", modified, tt.wantModified)
			}

			ids := ScanMarkdownForIDs(result)
			if len(ids) != tt.wantIDCount {
				t.Errorf("ID count = %d, want %d", len(ids), tt.wantIDCount)
			}
		})
	}
}

func TestMarkdownSourceFetch(t *testing.T) {
	// Create temp file with markdown content
	tmpDir := t.TempDir()
	mdContent := `# Tasks {#tasks}

- [ ] Test task 1 <!-- id:t1 -->
- [x] Test task 2 <!-- id:t2 -->
`
	mdPath := filepath.Join(tmpDir, "test.md")
	if err := os.WriteFile(mdPath, []byte(mdContent), 0644); err != nil {
		t.Fatalf("Failed to write temp file: %v", err)
	}

	src, err := NewMarkdownSource("tasks", "test.md", "#tasks", tmpDir, filepath.Join(tmpDir, "index.md"), true)
	if err != nil {
		t.Fatalf("NewMarkdownSource() error = %v", err)
	}

	results, err := src.Fetch(context.Background())
	if err != nil {
		t.Fatalf("Fetch() error = %v", err)
	}

	if len(results) != 2 {
		t.Fatalf("expected 2 tasks, got %d", len(results))
	}

	if results[0]["text"] != "Test task 1" {
		t.Errorf("task 0 text = %q, want %q", results[0]["text"], "Test task 1")
	}
	if results[0]["id"] != "t1" {
		t.Errorf("task 0 id = %q, want %q", results[0]["id"], "t1")
	}
}

func TestMarkdownSourceExternalFile(t *testing.T) {
	// Create temp directory with main file and data file
	tmpDir := t.TempDir()

	// Create data directory
	dataDir := filepath.Join(tmpDir, "data")
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		t.Fatalf("Failed to create data dir: %v", err)
	}

	// Create external data file
	dataContent := `# External Data {#items}

- External item 1 <!-- id:ext1 -->
- External item 2 <!-- id:ext2 -->
`
	dataPath := filepath.Join(dataDir, "items.md")
	if err := os.WriteFile(dataPath, []byte(dataContent), 0644); err != nil {
		t.Fatalf("Failed to write data file: %v", err)
	}

	// Create source pointing to external file
	src, err := NewMarkdownSource("items", "data/items.md", "#items", tmpDir, filepath.Join(tmpDir, "index.md"), true)
	if err != nil {
		t.Fatalf("NewMarkdownSource() error = %v", err)
	}

	results, err := src.Fetch(context.Background())
	if err != nil {
		t.Fatalf("Fetch() error = %v", err)
	}

	if len(results) != 2 {
		t.Fatalf("expected 2 items from external file, got %d", len(results))
	}

	if results[0]["text"] != "External item 1" {
		t.Errorf("item 0 text = %q, want %q", results[0]["text"], "External item 1")
	}
}

func TestMarkdownSourceWriteReadonly(t *testing.T) {
	src, err := NewMarkdownSource("tasks", "data/tasks.md", "#tasks", "/site", "/site/index.md", true)
	if err != nil {
		t.Fatalf("NewMarkdownSource() error = %v", err)
	}

	err = src.WriteItem(context.Background(), "add", map[string]interface{}{"text": "New task"})
	if err == nil {
		t.Error("expected error when writing to readonly source")
	}
	if err.Error() != `markdown source "tasks" is read-only` {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestMarkdownSourceParser(t *testing.T) {
	content := `# Tasks {#tasks}

- [ ] Parser task 1 <!-- id:p1 -->
- [x] Parser task 2 <!-- id:p2 -->
`
	parser := &MarkdownSourceParser{}
	results, err := parser.ParseContent(content, "#tasks")
	if err != nil {
		t.Fatalf("ParseContent() error = %v", err)
	}

	if len(results) != 2 {
		t.Fatalf("expected 2 tasks, got %d", len(results))
	}

	if results[0]["id"] != "p1" {
		t.Errorf("task 0 id = %q, want %q", results[0]["id"], "p1")
	}
}

func TestGenerateID(t *testing.T) {
	// Generate multiple IDs and ensure they're unique and correct format
	ids := make(map[string]bool)
	for i := 0; i < 100; i++ {
		id := generateID()
		if len(id) != 8 {
			t.Errorf("expected 8-character ID, got %d characters: %s", len(id), id)
		}
		if ids[id] {
			t.Errorf("duplicate ID generated: %s", id)
		}
		ids[id] = true
	}
}

func TestGenerateContentID(t *testing.T) {
	// Test that same content always produces same ID
	id1 := generateContentID("Buy groceries")
	id2 := generateContentID("Buy groceries")
	if id1 != id2 {
		t.Errorf("same content should produce same ID: got %s and %s", id1, id2)
	}

	// Test that different content produces different IDs
	id3 := generateContentID("Walk the dog")
	if id1 == id3 {
		t.Errorf("different content should produce different IDs: both got %s", id1)
	}

	// Test ID format (8 hex characters)
	if len(id1) != 8 {
		t.Errorf("expected 8-character ID, got %d characters: %s", len(id1), id1)
	}
}

func TestContentBasedIDsInParsing(t *testing.T) {
	// Test that items without explicit IDs get deterministic content-based IDs
	tmpDir := t.TempDir()
	mdContent := `# Tasks {#tasks}

- [ ] Buy groceries
- [x] Clean the house
- [ ] Buy groceries
`
	mdPath := filepath.Join(tmpDir, "test.md")
	if err := os.WriteFile(mdPath, []byte(mdContent), 0644); err != nil {
		t.Fatalf("Failed to write temp file: %v", err)
	}

	src, err := NewMarkdownSource("tasks", "test.md", "#tasks", tmpDir, filepath.Join(tmpDir, "index.md"), true)
	if err != nil {
		t.Fatalf("NewMarkdownSource() error = %v", err)
	}

	results, err := src.Fetch(context.Background())
	if err != nil {
		t.Fatalf("Fetch() error = %v", err)
	}

	if len(results) != 3 {
		t.Fatalf("expected 3 tasks, got %d", len(results))
	}

	// First and third have same text, should have same ID (content-based)
	if results[0]["id"] != results[2]["id"] {
		t.Errorf("items with same text should have same ID: got %s and %s", results[0]["id"], results[2]["id"])
	}

	// Second has different text, should have different ID
	if results[0]["id"] == results[1]["id"] {
		t.Errorf("items with different text should have different IDs: both got %s", results[0]["id"])
	}

	// Fetch again - IDs should be identical (deterministic)
	results2, err := src.Fetch(context.Background())
	if err != nil {
		t.Fatalf("Second Fetch() error = %v", err)
	}

	for i := range results {
		if results[i]["id"] != results2[i]["id"] {
			t.Errorf("IDs should be consistent across fetches: item %d got %s then %s", i, results[i]["id"], results2[i]["id"])
		}
	}
}

func TestExplicitIDsTakePrecedence(t *testing.T) {
	// Test that explicit <!-- id:xxx --> comments take precedence over content-based IDs
	tmpDir := t.TempDir()
	mdContent := `# Tasks {#tasks}

- [ ] Buy groceries <!-- id:my_custom_id -->
- [ ] Walk the dog
`
	mdPath := filepath.Join(tmpDir, "test.md")
	if err := os.WriteFile(mdPath, []byte(mdContent), 0644); err != nil {
		t.Fatalf("Failed to write temp file: %v", err)
	}

	src, err := NewMarkdownSource("tasks", "test.md", "#tasks", tmpDir, filepath.Join(tmpDir, "index.md"), true)
	if err != nil {
		t.Fatalf("NewMarkdownSource() error = %v", err)
	}

	results, err := src.Fetch(context.Background())
	if err != nil {
		t.Fatalf("Fetch() error = %v", err)
	}

	// First item should have the explicit ID
	if results[0]["id"] != "my_custom_id" {
		t.Errorf("explicit ID should be used: got %s, want my_custom_id", results[0]["id"])
	}

	// Second item should have a content-based ID (not the explicit one)
	if results[1]["id"] == "my_custom_id" {
		t.Error("second item should not have the same ID as the explicit one")
	}
}

func TestParseTableCells(t *testing.T) {
	src := &MarkdownSource{}

	tests := []struct {
		row  string
		want []string
	}{
		{" A | B | C ", []string{"A", "B", "C"}},
		{" Single ", []string{"Single"}},
		{"  Spaced  |  Values  ", []string{"Spaced", "Values"}},
		{"", nil},
	}

	for _, tt := range tests {
		t.Run(tt.row, func(t *testing.T) {
			got := src.parseTableCells(tt.row)
			if len(got) != len(tt.want) {
				t.Fatalf("parseTableCells(%q) = %v, want %v", tt.row, got, tt.want)
			}
			for i, v := range tt.want {
				if got[i] != v {
					t.Errorf("parseTableCells(%q)[%d] = %q, want %q", tt.row, i, got[i], v)
				}
			}
		})
	}
}

func TestTaskListMixedCheckboxStyles(t *testing.T) {
	content := `# Tasks {#tasks}

- [ ] Unchecked lowercase
- [x] Checked lowercase x
- [X] Checked uppercase X
- [ ] Another unchecked
`
	src := &MarkdownSource{anchor: "#tasks"}
	results, err := src.parseSection(content)
	if err != nil {
		t.Fatalf("parseSection() error = %v", err)
	}

	if len(results) != 4 {
		t.Fatalf("expected 4 tasks, got %d", len(results))
	}

	expectedDone := []bool{false, true, true, false}
	for i, expected := range expectedDone {
		if results[i]["done"] != expected {
			t.Errorf("task %d done = %v, want %v", i, results[i]["done"], expected)
		}
	}
}

func TestBulletListSkipsTaskItems(t *testing.T) {
	// Ensure bullet list parser doesn't accidentally capture task list items
	lines := []string{
		"- [ ] This is a task",
		"- [x] This is completed",
		"- Regular bullet item",
	}

	src := &MarkdownSource{}
	results, err := src.parseBulletList(lines)
	if err != nil {
		t.Fatalf("parseBulletList() error = %v", err)
	}

	// Should only capture the regular bullet item
	if len(results) != 1 {
		t.Fatalf("expected 1 bullet item (skipping tasks), got %d", len(results))
	}
	if results[0]["text"] != "Regular bullet item" {
		t.Errorf("text = %q, want %q", results[0]["text"], "Regular bullet item")
	}
}

// ============ Phase 2: Write Operation Tests ============

func TestWriteItemAddTask(t *testing.T) {
	tmpDir := t.TempDir()
	mdContent := `# Tasks {#tasks}

- [ ] Existing task 1 <!-- id:t1 -->
- [x] Existing task 2 <!-- id:t2 -->
`
	mdPath := filepath.Join(tmpDir, "test.md")
	if err := os.WriteFile(mdPath, []byte(mdContent), 0644); err != nil {
		t.Fatalf("Failed to write temp file: %v", err)
	}

	src, err := NewMarkdownSource("tasks", "test.md", "#tasks", tmpDir, filepath.Join(tmpDir, "index.md"), false)
	if err != nil {
		t.Fatalf("NewMarkdownSource() error = %v", err)
	}

	// Add a new task
	err = src.WriteItem(context.Background(), "add", map[string]interface{}{
		"text": "New task 3",
		"done": false,
	})
	if err != nil {
		t.Fatalf("WriteItem(add) error = %v", err)
	}

	// Verify the new task was added
	results, err := src.Fetch(context.Background())
	if err != nil {
		t.Fatalf("Fetch() error = %v", err)
	}

	if len(results) != 3 {
		t.Fatalf("expected 3 tasks after add, got %d", len(results))
	}

	lastTask := results[2]
	if lastTask["text"] != "New task 3" {
		t.Errorf("new task text = %q, want %q", lastTask["text"], "New task 3")
	}
	if lastTask["done"] != false {
		t.Errorf("new task done = %v, want false", lastTask["done"])
	}
}

func TestWriteItemToggleTask(t *testing.T) {
	tmpDir := t.TempDir()
	mdContent := `# Tasks {#tasks}

- [ ] Task to toggle <!-- id:toggle1 -->
- [x] Already done <!-- id:toggle2 -->
`
	mdPath := filepath.Join(tmpDir, "test.md")
	if err := os.WriteFile(mdPath, []byte(mdContent), 0644); err != nil {
		t.Fatalf("Failed to write temp file: %v", err)
	}

	src, err := NewMarkdownSource("tasks", "test.md", "#tasks", tmpDir, filepath.Join(tmpDir, "index.md"), false)
	if err != nil {
		t.Fatalf("NewMarkdownSource() error = %v", err)
	}

	// Toggle the first task (unchecked -> checked)
	err = src.WriteItem(context.Background(), "toggle", map[string]interface{}{
		"id": "toggle1",
	})
	if err != nil {
		t.Fatalf("WriteItem(toggle) error = %v", err)
	}

	results, err := src.Fetch(context.Background())
	if err != nil {
		t.Fatalf("Fetch() error = %v", err)
	}

	if results[0]["done"] != true {
		t.Errorf("task 0 done after toggle = %v, want true", results[0]["done"])
	}

	// Toggle again (checked -> unchecked)
	err = src.WriteItem(context.Background(), "toggle", map[string]interface{}{
		"id": "toggle1",
	})
	if err != nil {
		t.Fatalf("WriteItem(toggle again) error = %v", err)
	}

	results, err = src.Fetch(context.Background())
	if err != nil {
		t.Fatalf("Fetch() error = %v", err)
	}

	if results[0]["done"] != false {
		t.Errorf("task 0 done after second toggle = %v, want false", results[0]["done"])
	}
}

func TestWriteItemDeleteTask(t *testing.T) {
	tmpDir := t.TempDir()
	mdContent := `# Tasks {#tasks}

- [ ] Task 1 <!-- id:del1 -->
- [ ] Task 2 (to delete) <!-- id:del2 -->
- [ ] Task 3 <!-- id:del3 -->
`
	mdPath := filepath.Join(tmpDir, "test.md")
	if err := os.WriteFile(mdPath, []byte(mdContent), 0644); err != nil {
		t.Fatalf("Failed to write temp file: %v", err)
	}

	src, err := NewMarkdownSource("tasks", "test.md", "#tasks", tmpDir, filepath.Join(tmpDir, "index.md"), false)
	if err != nil {
		t.Fatalf("NewMarkdownSource() error = %v", err)
	}

	// Delete the middle task
	err = src.WriteItem(context.Background(), "delete", map[string]interface{}{
		"id": "del2",
	})
	if err != nil {
		t.Fatalf("WriteItem(delete) error = %v", err)
	}

	results, err := src.Fetch(context.Background())
	if err != nil {
		t.Fatalf("Fetch() error = %v", err)
	}

	if len(results) != 2 {
		t.Fatalf("expected 2 tasks after delete, got %d", len(results))
	}

	// Verify correct tasks remain
	if results[0]["id"] != "del1" {
		t.Errorf("task 0 id = %q, want %q", results[0]["id"], "del1")
	}
	if results[1]["id"] != "del3" {
		t.Errorf("task 1 id = %q, want %q", results[1]["id"], "del3")
	}
}

func TestWriteItemUpdateTask(t *testing.T) {
	tmpDir := t.TempDir()
	mdContent := `# Tasks {#tasks}

- [ ] Original text <!-- id:upd1 -->
`
	mdPath := filepath.Join(tmpDir, "test.md")
	if err := os.WriteFile(mdPath, []byte(mdContent), 0644); err != nil {
		t.Fatalf("Failed to write temp file: %v", err)
	}

	src, err := NewMarkdownSource("tasks", "test.md", "#tasks", tmpDir, filepath.Join(tmpDir, "index.md"), false)
	if err != nil {
		t.Fatalf("NewMarkdownSource() error = %v", err)
	}

	// Update the task text and done state
	err = src.WriteItem(context.Background(), "update", map[string]interface{}{
		"id":   "upd1",
		"text": "Updated text",
		"done": true,
	})
	if err != nil {
		t.Fatalf("WriteItem(update) error = %v", err)
	}

	results, err := src.Fetch(context.Background())
	if err != nil {
		t.Fatalf("Fetch() error = %v", err)
	}

	if results[0]["text"] != "Updated text" {
		t.Errorf("task text = %q, want %q", results[0]["text"], "Updated text")
	}
	if results[0]["done"] != true {
		t.Errorf("task done = %v, want true", results[0]["done"])
	}
}

func TestWriteItemAddBullet(t *testing.T) {
	tmpDir := t.TempDir()
	mdContent := `# Items {#items}

- Item 1 <!-- id:b1 -->
`
	mdPath := filepath.Join(tmpDir, "test.md")
	if err := os.WriteFile(mdPath, []byte(mdContent), 0644); err != nil {
		t.Fatalf("Failed to write temp file: %v", err)
	}

	src, err := NewMarkdownSource("items", "test.md", "#items", tmpDir, filepath.Join(tmpDir, "index.md"), false)
	if err != nil {
		t.Fatalf("NewMarkdownSource() error = %v", err)
	}

	err = src.WriteItem(context.Background(), "add", map[string]interface{}{
		"text": "Item 2",
	})
	if err != nil {
		t.Fatalf("WriteItem(add) error = %v", err)
	}

	results, err := src.Fetch(context.Background())
	if err != nil {
		t.Fatalf("Fetch() error = %v", err)
	}

	if len(results) != 2 {
		t.Fatalf("expected 2 items, got %d", len(results))
	}
	if results[1]["text"] != "Item 2" {
		t.Errorf("new item text = %q, want %q", results[1]["text"], "Item 2")
	}
}

func TestWriteItemAddTableRow(t *testing.T) {
	tmpDir := t.TempDir()
	mdContent := `# Products {#products}

| Name | Price |
|------|-------|
| Widget | $10 | <!-- id:p1 -->
`
	mdPath := filepath.Join(tmpDir, "test.md")
	if err := os.WriteFile(mdPath, []byte(mdContent), 0644); err != nil {
		t.Fatalf("Failed to write temp file: %v", err)
	}

	src, err := NewMarkdownSource("products", "test.md", "#products", tmpDir, filepath.Join(tmpDir, "index.md"), false)
	if err != nil {
		t.Fatalf("NewMarkdownSource() error = %v", err)
	}

	err = src.WriteItem(context.Background(), "add", map[string]interface{}{
		"Name":  "Gadget",
		"Price": "$20",
	})
	if err != nil {
		t.Fatalf("WriteItem(add) error = %v", err)
	}

	results, err := src.Fetch(context.Background())
	if err != nil {
		t.Fatalf("Fetch() error = %v", err)
	}

	if len(results) != 2 {
		t.Fatalf("expected 2 products, got %d", len(results))
	}
	if results[1]["Name"] != "Gadget" {
		t.Errorf("new product Name = %q, want %q", results[1]["Name"], "Gadget")
	}
	if results[1]["Price"] != "$20" {
		t.Errorf("new product Price = %q, want %q", results[1]["Price"], "$20")
	}
}

func TestWriteItemDeleteTableRow(t *testing.T) {
	tmpDir := t.TempDir()
	mdContent := `# Products {#products}

| Name | Price |
|------|-------|
| Widget | $10 | <!-- id:p1 -->
| Gadget | $20 | <!-- id:p2 -->
`
	mdPath := filepath.Join(tmpDir, "test.md")
	if err := os.WriteFile(mdPath, []byte(mdContent), 0644); err != nil {
		t.Fatalf("Failed to write temp file: %v", err)
	}

	src, err := NewMarkdownSource("products", "test.md", "#products", tmpDir, filepath.Join(tmpDir, "index.md"), false)
	if err != nil {
		t.Fatalf("NewMarkdownSource() error = %v", err)
	}

	err = src.WriteItem(context.Background(), "delete", map[string]interface{}{
		"id": "p1",
	})
	if err != nil {
		t.Fatalf("WriteItem(delete) error = %v", err)
	}

	results, err := src.Fetch(context.Background())
	if err != nil {
		t.Fatalf("Fetch() error = %v", err)
	}

	if len(results) != 1 {
		t.Fatalf("expected 1 product after delete, got %d", len(results))
	}
	if results[0]["Name"] != "Gadget" {
		t.Errorf("remaining product Name = %q, want %q", results[0]["Name"], "Gadget")
	}
}

func TestWriteItemNotFoundError(t *testing.T) {
	tmpDir := t.TempDir()
	mdContent := `# Tasks {#tasks}

- [ ] Task 1 <!-- id:t1 -->
`
	mdPath := filepath.Join(tmpDir, "test.md")
	if err := os.WriteFile(mdPath, []byte(mdContent), 0644); err != nil {
		t.Fatalf("Failed to write temp file: %v", err)
	}

	src, err := NewMarkdownSource("tasks", "test.md", "#tasks", tmpDir, filepath.Join(tmpDir, "index.md"), false)
	if err != nil {
		t.Fatalf("NewMarkdownSource() error = %v", err)
	}

	// Try to toggle non-existent item
	err = src.WriteItem(context.Background(), "toggle", map[string]interface{}{
		"id": "nonexistent",
	})
	if err == nil {
		t.Error("expected error for non-existent item")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("error should contain 'not found', got: %v", err)
	}
}

func TestWriteItemUnknownAction(t *testing.T) {
	tmpDir := t.TempDir()
	mdContent := `# Tasks {#tasks}

- [ ] Task 1 <!-- id:t1 -->
`
	mdPath := filepath.Join(tmpDir, "test.md")
	if err := os.WriteFile(mdPath, []byte(mdContent), 0644); err != nil {
		t.Fatalf("Failed to write temp file: %v", err)
	}

	src, err := NewMarkdownSource("tasks", "test.md", "#tasks", tmpDir, filepath.Join(tmpDir, "index.md"), false)
	if err != nil {
		t.Fatalf("NewMarkdownSource() error = %v", err)
	}

	err = src.WriteItem(context.Background(), "invalid_action", map[string]interface{}{})
	if err == nil {
		t.Error("expected error for unknown action")
	}
	if !strings.Contains(err.Error(), "unknown action") {
		t.Errorf("error should contain 'unknown action', got: %v", err)
	}
}

func TestWriteItemToggleOnlyForTasks(t *testing.T) {
	tmpDir := t.TempDir()
	mdContent := `# Items {#items}

- Bullet item <!-- id:b1 -->
`
	mdPath := filepath.Join(tmpDir, "test.md")
	if err := os.WriteFile(mdPath, []byte(mdContent), 0644); err != nil {
		t.Fatalf("Failed to write temp file: %v", err)
	}

	src, err := NewMarkdownSource("items", "test.md", "#items", tmpDir, filepath.Join(tmpDir, "index.md"), false)
	if err != nil {
		t.Fatalf("NewMarkdownSource() error = %v", err)
	}

	err = src.WriteItem(context.Background(), "toggle", map[string]interface{}{
		"id": "b1",
	})
	if err == nil {
		t.Error("expected error when toggling non-task list item")
	}
	if !strings.Contains(err.Error(), "only supported for task lists") {
		t.Errorf("error should mention task lists, got: %v", err)
	}
}

// ============ Phase 4: Concurrency & Conflict Handling Tests ============

func TestWriteItemMtimeTracking(t *testing.T) {
	tmpDir := t.TempDir()
	mdContent := `# Tasks {#tasks}

- [ ] Task 1 <!-- id:t1 -->
`
	mdPath := filepath.Join(tmpDir, "test.md")
	if err := os.WriteFile(mdPath, []byte(mdContent), 0644); err != nil {
		t.Fatalf("Failed to write temp file: %v", err)
	}

	src, err := NewMarkdownSource("tasks", "test.md", "#tasks", tmpDir, filepath.Join(tmpDir, "index.md"), false)
	if err != nil {
		t.Fatalf("NewMarkdownSource() error = %v", err)
	}

	// First Fetch should record mtime
	_, err = src.Fetch(context.Background())
	if err != nil {
		t.Fatalf("Fetch() error = %v", err)
	}

	// Verify mtime was recorded
	src.mu.RLock()
	hasLastMtime := !src.lastMtime.IsZero()
	src.mu.RUnlock()

	if !hasLastMtime {
		t.Error("lastMtime should be set after Fetch()")
	}

	// Write should succeed without conflict
	err = src.WriteItem(context.Background(), "add", map[string]interface{}{
		"text": "Task 2",
	})
	if err != nil {
		t.Fatalf("WriteItem(add) error = %v", err)
	}

	// Verify mtime was updated after write
	src.mu.RLock()
	mtimeAfterWrite := src.lastMtime
	src.mu.RUnlock()

	if mtimeAfterWrite.IsZero() {
		t.Error("lastMtime should be updated after WriteItem()")
	}
}

func TestWriteItemConflictDetection(t *testing.T) {
	tmpDir := t.TempDir()
	mdContent := `# Tasks {#tasks}

- [ ] Task 1 <!-- id:t1 -->
`
	mdPath := filepath.Join(tmpDir, "test.md")
	if err := os.WriteFile(mdPath, []byte(mdContent), 0644); err != nil {
		t.Fatalf("Failed to write temp file: %v", err)
	}

	src, err := NewMarkdownSource("tasks", "test.md", "#tasks", tmpDir, filepath.Join(tmpDir, "index.md"), false)
	if err != nil {
		t.Fatalf("NewMarkdownSource() error = %v", err)
	}

	// Fetch to record initial mtime
	_, err = src.Fetch(context.Background())
	if err != nil {
		t.Fatalf("Fetch() error = %v", err)
	}

	// Simulate external modification by writing to the file directly
	// Wait a bit to ensure different mtime (some filesystems have 1-second resolution)
	time.Sleep(100 * time.Millisecond)
	externalContent := `# Tasks {#tasks}

- [ ] Task 1 <!-- id:t1 -->
- [ ] External task <!-- id:ext1 -->
`
	if err := os.WriteFile(mdPath, []byte(externalContent), 0644); err != nil {
		t.Fatalf("Failed to write external modification: %v", err)
	}

	// Try to write - should detect conflict
	err = src.WriteItem(context.Background(), "add", map[string]interface{}{
		"text": "My new task",
	})

	if err == nil {
		t.Fatal("expected ConflictError, got nil")
	}

	// Verify it's a ConflictError
	conflictErr, ok := err.(*ConflictError)
	if !ok {
		t.Fatalf("expected *ConflictError, got %T: %v", err, err)
	}

	// Verify conflict file was created
	if conflictErr.ConflictPath == "" {
		t.Error("ConflictPath should not be empty")
	}

	if _, err := os.Stat(conflictErr.ConflictPath); os.IsNotExist(err) {
		t.Errorf("conflict file should exist: %s", conflictErr.ConflictPath)
	}

	// Clean up conflict file
	os.Remove(conflictErr.ConflictPath)
}

func TestWriteItemNoConflictWithoutPriorFetch(t *testing.T) {
	tmpDir := t.TempDir()
	mdContent := `# Tasks {#tasks}

- [ ] Task 1 <!-- id:t1 -->
`
	mdPath := filepath.Join(tmpDir, "test.md")
	if err := os.WriteFile(mdPath, []byte(mdContent), 0644); err != nil {
		t.Fatalf("Failed to write temp file: %v", err)
	}

	src, err := NewMarkdownSource("tasks", "test.md", "#tasks", tmpDir, filepath.Join(tmpDir, "index.md"), false)
	if err != nil {
		t.Fatalf("NewMarkdownSource() error = %v", err)
	}

	// Write without prior Fetch - should succeed (no mtime to compare)
	err = src.WriteItem(context.Background(), "add", map[string]interface{}{
		"text": "New task",
	})
	if err != nil {
		t.Fatalf("WriteItem(add) should succeed without prior Fetch, got error: %v", err)
	}

	// Verify task was added
	results, err := src.Fetch(context.Background())
	if err != nil {
		t.Fatalf("Fetch() error = %v", err)
	}

	if len(results) != 2 {
		t.Errorf("expected 2 tasks, got %d", len(results))
	}
}

func TestConflictCopyContainsOriginalContent(t *testing.T) {
	tmpDir := t.TempDir()
	mdContent := `# Tasks {#tasks}

- [ ] Original task <!-- id:t1 -->
`
	mdPath := filepath.Join(tmpDir, "test.md")
	if err := os.WriteFile(mdPath, []byte(mdContent), 0644); err != nil {
		t.Fatalf("Failed to write temp file: %v", err)
	}

	src, err := NewMarkdownSource("tasks", "test.md", "#tasks", tmpDir, filepath.Join(tmpDir, "index.md"), false)
	if err != nil {
		t.Fatalf("NewMarkdownSource() error = %v", err)
	}

	// Fetch to record initial mtime
	_, err = src.Fetch(context.Background())
	if err != nil {
		t.Fatalf("Fetch() error = %v", err)
	}

	// Simulate external modification
	time.Sleep(100 * time.Millisecond)
	externalContent := `# Tasks {#tasks}

- [ ] Original task <!-- id:t1 -->
- [ ] External modification <!-- id:ext1 -->
`
	if err := os.WriteFile(mdPath, []byte(externalContent), 0644); err != nil {
		t.Fatalf("Failed to write external modification: %v", err)
	}

	// Try to write - should create conflict copy
	err = src.WriteItem(context.Background(), "add", map[string]interface{}{
		"text": "My task",
	})

	conflictErr, ok := err.(*ConflictError)
	if !ok {
		t.Fatalf("expected *ConflictError, got %T: %v", err, err)
	}

	// Read conflict file and verify it contains the external modification
	conflictContent, err := os.ReadFile(conflictErr.ConflictPath)
	if err != nil {
		t.Fatalf("Failed to read conflict file: %v", err)
	}

	if !strings.Contains(string(conflictContent), "External modification") {
		t.Error("conflict file should contain the external modification")
	}

	// Clean up
	os.Remove(conflictErr.ConflictPath)
}

func TestSlugify(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"Tasks", "tasks"},
		{"My Task List", "my-task-list"},
		{"Active Items", "active-items"},
		{"Team Members", "team-members"},
		{"Special! Characters?", "special-characters"},
		{"Under_scores", "underscores"}, // underscores are removed (not alphanumeric)
		{"  Spaces  ", "--spaces--"},     // spaces become hyphens
		{"123 Numbers", "123-numbers"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := slugify(tt.input)
			if got != tt.want {
				t.Errorf("slugify(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestTextBasedAnchorMatching(t *testing.T) {
	tmpDir := t.TempDir()

	// Test 1: Simple heading text matches anchor
	t.Run("heading text matches anchor", func(t *testing.T) {
		mdContent := `# Tasks

- [ ] Task one
- [x] Task two
`
		mdPath := filepath.Join(tmpDir, "tasks.md")
		if err := os.WriteFile(mdPath, []byte(mdContent), 0644); err != nil {
			t.Fatalf("Failed to write temp file: %v", err)
		}

		src, err := NewMarkdownSource("tasks", "tasks.md", "#tasks", tmpDir, filepath.Join(tmpDir, "index.md"), true)
		if err != nil {
			t.Fatalf("NewMarkdownSource() error = %v", err)
		}

		results, err := src.Fetch(context.Background())
		if err != nil {
			t.Fatalf("Fetch() error = %v", err)
		}

		if len(results) != 2 {
			t.Errorf("expected 2 tasks, got %d", len(results))
		}
	})

	// Test 2: Multi-word heading slugified
	t.Run("multi-word heading slugified", func(t *testing.T) {
		mdContent := `# My Task List

- [ ] Item one
- [ ] Item two
`
		mdPath := filepath.Join(tmpDir, "my-tasks.md")
		if err := os.WriteFile(mdPath, []byte(mdContent), 0644); err != nil {
			t.Fatalf("Failed to write temp file: %v", err)
		}

		src, err := NewMarkdownSource("tasks", "my-tasks.md", "#my-task-list", tmpDir, filepath.Join(tmpDir, "index.md"), true)
		if err != nil {
			t.Fatalf("NewMarkdownSource() error = %v", err)
		}

		results, err := src.Fetch(context.Background())
		if err != nil {
			t.Fatalf("Fetch() error = %v", err)
		}

		if len(results) != 2 {
			t.Errorf("expected 2 items, got %d", len(results))
		}
	})

	// Test 3: Explicit anchor takes precedence
	t.Run("explicit anchor takes precedence", func(t *testing.T) {
		mdContent := `# Tasks {#todos}

- [ ] Explicit anchor task
`
		mdPath := filepath.Join(tmpDir, "explicit.md")
		if err := os.WriteFile(mdPath, []byte(mdContent), 0644); err != nil {
			t.Fatalf("Failed to write temp file: %v", err)
		}

		// Should match #todos, not #tasks
		src, err := NewMarkdownSource("tasks", "explicit.md", "#todos", tmpDir, filepath.Join(tmpDir, "index.md"), true)
		if err != nil {
			t.Fatalf("NewMarkdownSource() error = %v", err)
		}

		results, err := src.Fetch(context.Background())
		if err != nil {
			t.Fatalf("Fetch() error = %v", err)
		}

		if len(results) != 1 {
			t.Errorf("expected 1 task, got %d", len(results))
		}

		// Should NOT match #tasks (because explicit anchor overrides)
		src2, err := NewMarkdownSource("tasks2", "explicit.md", "#tasks", tmpDir, filepath.Join(tmpDir, "index.md"), true)
		if err != nil {
			t.Fatalf("NewMarkdownSource() error = %v", err)
		}

		results2, err := src2.Fetch(context.Background())
		if err != nil {
			t.Fatalf("Fetch() error = %v", err)
		}

		if len(results2) != 0 {
			t.Errorf("expected 0 tasks for #tasks anchor (explicit #todos should take precedence), got %d", len(results2))
		}
	})

	// Test 4: Special characters in heading
	t.Run("special characters in heading", func(t *testing.T) {
		mdContent := `# Active! Tasks?

- [ ] Special char task
`
		mdPath := filepath.Join(tmpDir, "special.md")
		if err := os.WriteFile(mdPath, []byte(mdContent), 0644); err != nil {
			t.Fatalf("Failed to write temp file: %v", err)
		}

		src, err := NewMarkdownSource("tasks", "special.md", "#active-tasks", tmpDir, filepath.Join(tmpDir, "index.md"), true)
		if err != nil {
			t.Fatalf("NewMarkdownSource() error = %v", err)
		}

		results, err := src.Fetch(context.Background())
		if err != nil {
			t.Fatalf("Fetch() error = %v", err)
		}

		if len(results) != 1 {
			t.Errorf("expected 1 task, got %d", len(results))
		}
	})
}
