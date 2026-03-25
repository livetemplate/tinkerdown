package tinkerdown

import (
	"os"
	"strings"
	"testing"
)

func TestDetectTableSections(t *testing.T) {
	content := []byte(`## Expenses
| Description | Category | Amount |
|-------------|----------|--------|

## Notes
Some text here.
`)

	sections := detectTableSections(content)
	if len(sections) != 1 {
		t.Fatalf("expected 1 section, got %d", len(sections))
	}

	sec := sections[0]
	if sec.anchor != "expenses" {
		t.Errorf("expected anchor 'expenses', got %q", sec.anchor)
	}
	if sec.level != 2 {
		t.Errorf("expected level 2, got %d", sec.level)
	}
	if len(sec.columns) != 3 {
		t.Fatalf("expected 3 columns, got %d", len(sec.columns))
	}
	if sec.columns[0] != "Description" || sec.columns[1] != "Category" || sec.columns[2] != "Amount" {
		t.Errorf("unexpected columns: %v", sec.columns)
	}
}

func TestDetectTableSectionsWithData(t *testing.T) {
	content := []byte(`## Users
| Name   | Email            |
|--------|------------------|
| Alice  | alice@example.com|
| Bob    | bob@example.com  |

## Other Section
`)

	sections := detectTableSections(content)
	if len(sections) != 1 {
		t.Fatalf("expected 1 section, got %d", len(sections))
	}

	sec := sections[0]
	if sec.anchor != "users" {
		t.Errorf("expected anchor 'users', got %q", sec.anchor)
	}
	if len(sec.columns) != 2 {
		t.Fatalf("expected 2 columns, got %d", len(sec.columns))
	}
}

func TestDetectTableSectionsMultiple(t *testing.T) {
	content := []byte(`## Tasks
| Title | Done |
|-------|------|

## Contacts
| Name  | Email |
|-------|-------|

## Notes
Regular text, no table.
`)

	sections := detectTableSections(content)
	if len(sections) != 2 {
		t.Fatalf("expected 2 sections, got %d", len(sections))
	}

	if sections[0].anchor != "tasks" {
		t.Errorf("expected first section anchor 'tasks', got %q", sections[0].anchor)
	}
	if sections[1].anchor != "contacts" {
		t.Errorf("expected second section anchor 'contacts', got %q", sections[1].anchor)
	}
}

func TestDetectTableSectionsNoTable(t *testing.T) {
	content := []byte(`## About
This section has no table.

## More
Regular content only.
`)

	sections := detectTableSections(content)
	if len(sections) != 0 {
		t.Fatalf("expected 0 sections, got %d", len(sections))
	}
}

func TestDetectTableSectionsBlankLineBetween(t *testing.T) {
	content := []byte(`## Items

| Name | Price |
|------|-------|
`)

	sections := detectTableSections(content)
	if len(sections) != 1 {
		t.Fatalf("expected 1 section (blank line between heading and table ok), got %d", len(sections))
	}
}

func TestDetectTableSectionsNonTableContent(t *testing.T) {
	// Heading followed by prose then table — should NOT detect
	// because there's non-blank, non-table content between heading and table
	content := []byte(`## Items
Here are the items we have:
| Name | Price |
|------|-------|
`)

	sections := detectTableSections(content)
	// The prose line is between heading and table, so the table is not
	// directly under the heading. It should NOT be detected.
	if len(sections) != 0 {
		t.Fatalf("expected 0 sections (prose between heading and table), got %d", len(sections))
	}
}

func TestDetectTableSectionsExplicitAnchor(t *testing.T) {
	content := []byte(`## My Custom Section {#inventory}
| Item | Count |
|------|-------|
`)

	sections := detectTableSections(content)
	if len(sections) != 1 {
		t.Fatalf("expected 1 section, got %d", len(sections))
	}
	if sections[0].anchor != "inventory" {
		t.Errorf("expected anchor 'inventory', got %q", sections[0].anchor)
	}
	if sections[0].heading != "My Custom Section" {
		t.Errorf("expected heading 'My Custom Section', got %q", sections[0].heading)
	}
}

func TestDetectTableSectionsNoSeparator(t *testing.T) {
	// Table without separator row is not a valid markdown table
	content := []byte(`## Items
| Name | Price |
| foo  | 10    |
`)

	sections := detectTableSections(content)
	if len(sections) != 0 {
		t.Fatalf("expected 0 sections (no separator row), got %d", len(sections))
	}
}

// --- matchTablesToSources tests ---

func TestMatchExact(t *testing.T) {
	sections := []tableSection{
		{heading: "Expenses", anchor: "expenses", columns: []string{"Description", "Amount"}},
	}
	sources := map[string]SourceConfig{
		"expenses": {Type: "sqlite", DB: "./test.db", Table: "expenses"},
	}

	matches, warnings := matchTablesToSources(sections, sources)
	if len(warnings) != 0 {
		t.Errorf("unexpected warnings: %v", warnings)
	}
	if len(matches) != 1 {
		t.Fatalf("expected 1 match, got %d", len(matches))
	}
	if matches[0].sourceName != "expenses" {
		t.Errorf("expected source 'expenses', got %q", matches[0].sourceName)
	}
}

func TestMatchContainment(t *testing.T) {
	sections := []tableSection{
		{heading: "My Monthly Expenses", anchor: "my-monthly-expenses", columns: []string{"Description"}},
	}
	sources := map[string]SourceConfig{
		"expenses": {Type: "sqlite"},
	}

	matches, warnings := matchTablesToSources(sections, sources)
	if len(warnings) != 0 {
		t.Errorf("unexpected warnings: %v", warnings)
	}
	if len(matches) != 1 {
		t.Fatalf("expected 1 match, got %d", len(matches))
	}
	if matches[0].sourceName != "expenses" {
		t.Errorf("expected source 'expenses', got %q", matches[0].sourceName)
	}
}

func TestMatchNoMatch(t *testing.T) {
	sections := []tableSection{
		{heading: "Something Else", anchor: "something-else", columns: []string{"Name"}},
	}
	sources := map[string]SourceConfig{
		"expenses": {Type: "sqlite"},
	}

	matches, warnings := matchTablesToSources(sections, sources)
	if len(warnings) != 0 {
		t.Errorf("unexpected warnings: %v", warnings)
	}
	if len(matches) != 0 {
		t.Fatalf("expected 0 matches, got %d", len(matches))
	}
}

func TestMatchAmbiguous(t *testing.T) {
	sections := []tableSection{
		{heading: "Recent Tasks And Contacts", anchor: "recent-tasks-and-contacts", columns: []string{"Name"}},
	}
	sources := map[string]SourceConfig{
		"tasks":    {Type: "sqlite"},
		"contacts": {Type: "sqlite"},
	}

	matches, warnings := matchTablesToSources(sections, sources)
	if len(matches) != 0 {
		t.Fatalf("expected 0 matches (ambiguous), got %d", len(matches))
	}
	if len(warnings) != 1 {
		t.Fatalf("expected 1 warning, got %d", len(warnings))
	}
	if !strings.Contains(warnings[0], "multiple sources") {
		t.Errorf("expected 'multiple sources' warning, got %q", warnings[0])
	}
}

func TestMatchAutoBind(t *testing.T) {
	autoBindFalse := false
	sections := []tableSection{
		{heading: "Status", anchor: "status", columns: []string{"Name"}},
	}
	sources := map[string]SourceConfig{
		"status": {Type: "rest", AutoBind: &autoBindFalse},
	}

	matches, warnings := matchTablesToSources(sections, sources)
	if len(warnings) != 0 {
		t.Errorf("unexpected warnings: %v", warnings)
	}
	if len(matches) != 0 {
		t.Fatalf("expected 0 matches (auto_bind: false), got %d", len(matches))
	}
}

func TestMatchExactTakesPriority(t *testing.T) {
	// "tasks" heading should exact-match "tasks" source, not containment-match "my-tasks"
	sections := []tableSection{
		{heading: "Tasks", anchor: "tasks", columns: []string{"Title"}},
	}
	sources := map[string]SourceConfig{
		"tasks":    {Type: "sqlite"},
		"my-tasks": {Type: "sqlite"},
	}

	matches, warnings := matchTablesToSources(sections, sources)
	if len(warnings) != 0 {
		t.Errorf("unexpected warnings: %v", warnings)
	}
	if len(matches) != 1 {
		t.Fatalf("expected 1 match, got %d", len(matches))
	}
	if matches[0].sourceName != "tasks" {
		t.Errorf("expected exact match 'tasks', got %q", matches[0].sourceName)
	}
}

func TestMatchWritableDetection(t *testing.T) {
	readonlyFalse := false
	sections := []tableSection{
		{heading: "Tasks", anchor: "tasks", columns: []string{"Title"}},
		{heading: "Users", anchor: "users", columns: []string{"Name"}},
	}
	sources := map[string]SourceConfig{
		"tasks": {Type: "sqlite", Readonly: &readonlyFalse}, // writable
		"users": {Type: "rest"},                              // read-only (default)
	}

	matches, _ := matchTablesToSources(sections, sources)
	if len(matches) != 2 {
		t.Fatalf("expected 2 matches, got %d", len(matches))
	}

	// Find by source name
	for _, m := range matches {
		switch m.sourceName {
		case "tasks":
			if !m.writable {
				t.Error("tasks should be writable")
			}
		case "users":
			if m.writable {
				t.Error("users should be read-only")
			}
		}
	}
}

// --- containsWord tests ---

func TestContainsWord(t *testing.T) {
	tests := []struct {
		text   string
		word   string
		expect bool
	}{
		{"my expenses", "expenses", true},
		{"expenses tracker", "expenses", true},
		{"expenses", "expenses", true},
		{"my-expenses", "expenses", true},
		{"myexpenses", "expenses", false},           // no word boundary
		{"expensesmy", "expenses", false},           // no word boundary
		{"myexpenses expenses", "expenses", true},   // second occurrence is valid
		{"the expense report", "expenses", false}, // different word
		{"team tasks board", "tasks", true},
		{"", "tasks", false},
		{"tasks", "", false}, // empty source name never matches
	}

	for _, tt := range tests {
		got := containsWord(tt.text, tt.word)
		if got != tt.expect {
			t.Errorf("containsWord(%q, %q) = %v, want %v", tt.text, tt.word, got, tt.expect)
		}
	}
}

// --- toFieldName tests ---

func TestToFieldName(t *testing.T) {
	tests := []struct {
		input  string
		expect string
	}{
		{"name", "Name"},
		{"full name", "FullName"},
		{"first_name", "FirstName"},
		{"Description", "Description"},
		{"Amount", "Amount"},
		{"some-field", "SomeField"},
		{"  spaced  ", "Spaced"},
		{"Name}}{{.Password", "NamePassword"}, // template injection stripped
		{"", ""},
	}

	for _, tt := range tests {
		got := toFieldName(tt.input)
		if got != tt.expect {
			t.Errorf("toFieldName(%q) = %q, want %q", tt.input, got, tt.expect)
		}
	}
}

// --- toInputName tests ---

func TestToInputName(t *testing.T) {
	tests := []struct {
		input  string
		expect string
	}{
		{"Name", "name"},
		{"Full Name", "full_name"},
		{"first-name", "first_name"},
		{"Amount ($)", "amount"},
		{"user.email", "user_email"},
		{"  spaced  ", "spaced"},
		{"a/b/c", "a_b_c"},
		{"---leading---", "leading"},
		{"col with  spaces", "col_with_spaces"},
		{"", ""},
	}

	for _, tt := range tests {
		got := toInputName(tt.input)
		if got != tt.expect {
			t.Errorf("toInputName(%q) = %q, want %q", tt.input, got, tt.expect)
		}
	}
}

// --- LVT block generation tests ---

func TestGenerateReadonlyBlock(t *testing.T) {
	block := generateAutoTableLvtBlock("users", []string{"Name", "Email"}, false, nil)

	// Should have lvt-source
	if !strings.Contains(block, `lvt-source="users"`) {
		t.Error("expected lvt-source=\"users\"")
	}
	// Should have Refresh button
	if !strings.Contains(block, `lvt-click="Refresh"`) {
		t.Error("expected Refresh button")
	}
	// Should NOT have Add form
	if strings.Contains(block, `lvt-submit="Add"`) {
		t.Error("read-only block should not have Add form")
	}
	// Should NOT have Delete button
	if strings.Contains(block, `lvt-click="Delete"`) {
		t.Error("read-only block should not have Delete button")
	}
	// Should have column headers
	if !strings.Contains(block, "<th>Name</th>") || !strings.Contains(block, "<th>Email</th>") {
		t.Error("expected column headers")
	}
	// Should use template field accessors
	if !strings.Contains(block, "{{.Name}}") || !strings.Contains(block, "{{.Email}}") {
		t.Error("expected template field accessors")
	}
}

func TestGenerateWritableBlock(t *testing.T) {
	block := generateAutoTableLvtBlock("expenses", []string{"Description", "Category", "Amount"}, true, nil)

	// Should have lvt-source
	if !strings.Contains(block, `lvt-source="expenses"`) {
		t.Error("expected lvt-source=\"expenses\"")
	}
	// Should have Add form
	if !strings.Contains(block, `lvt-submit="Add"`) {
		t.Error("expected Add form")
	}
	// Should have Delete button
	if !strings.Contains(block, `lvt-click="Delete"`) {
		t.Error("expected Delete button")
	}
	// Should have confirmation on delete
	if !strings.Contains(block, `lvt-confirm=`) {
		t.Error("expected lvt-confirm on delete")
	}
	// Should have input for each column
	if !strings.Contains(block, `name="description"`) {
		t.Error("expected input name=\"description\"")
	}
	if !strings.Contains(block, `name="category"`) {
		t.Error("expected input name=\"category\"")
	}
	if !strings.Contains(block, `name="amount"`) {
		t.Error("expected input name=\"amount\"")
	}
	// Should have column headers
	if !strings.Contains(block, "<th>Description</th>") {
		t.Error("expected Description column header")
	}
	// Should have error display
	if !strings.Contains(block, "{{if .Error}}") {
		t.Error("expected error display")
	}
	// Should have Edit button
	if !strings.Contains(block, `lvt-click="Edit"`) {
		t.Error("expected Edit button")
	}
	// Should have inline edit conditional
	if !strings.Contains(block, `$.EditingId`) {
		t.Error("expected EditingID conditional for inline editing")
	}
	// Should have Update form (external, linked via HTML form attribute)
	if !strings.Contains(block, `lvt-submit="Update"`) {
		t.Error("expected Update form for inline edit")
	}
	if !strings.Contains(block, `form="auto-table-edit-expenses"`) {
		t.Error("expected inputs linked to external form via HTML form attribute with source-specific ID")
	}
	// Should have CancelEdit button
	if !strings.Contains(block, `lvt-click="CancelEdit"`) {
		t.Error("expected CancelEdit button")
	}
}

func TestGenerateWritableBlockWithSchema(t *testing.T) {
	// Provide schema types — Amount is REAL, Done is BOOLEAN
	columnTypes := map[string]string{
		"description": "text",
		"amount":      "real",
		"done":        "boolean",
		"due":         "date",
	}

	block := generateAutoTableLvtBlock("tasks", []string{"Description", "Amount", "Done", "Due"}, true, columnTypes)

	// Description should be text input
	if !strings.Contains(block, `type="text" name="description"`) {
		t.Error("expected type=\"text\" for Description column")
	}
	// Amount should be number input
	if !strings.Contains(block, `type="number" name="amount"`) {
		t.Error("expected type=\"number\" for Amount (real) column")
	}
	// Done should be checkbox input
	if !strings.Contains(block, `type="checkbox" name="done"`) {
		t.Error("expected type=\"checkbox\" for Done (boolean) column")
	}
	// Due should be date input
	if !strings.Contains(block, `type="date" name="due"`) {
		t.Error("expected type=\"date\" for Due (date) column")
	}
}

func TestGenerateWritableBlockWithoutSchema(t *testing.T) {
	// No schema — all inputs should be text
	block := generateAutoTableLvtBlock("tasks", []string{"Name", "Amount"}, true, nil)

	if !strings.Contains(block, `type="text" name="name"`) {
		t.Error("expected type=\"text\" for Name (no schema)")
	}
	if !strings.Contains(block, `type="text" name="amount"`) {
		t.Error("expected type=\"text\" for Amount (no schema)")
	}
}

// --- Full preprocessing tests ---

func TestPreprocessAutoTablesExactMatch(t *testing.T) {
	readonlyFalse := false
	content := []byte(`---
title: Test
sources:
  expenses:
    type: sqlite
    db: ./test.db
    table: expenses
    readonly: false
---

# Expense Tracker

## Expenses
| Description | Amount |
|-------------|--------|
`)

	sources := map[string]SourceConfig{
		"expenses": {Type: "sqlite", DB: "./test.db", Table: "expenses", Readonly: &readonlyFalse},
	}

	result, warnings := preprocessAutoTables(content, sources, "")
	if len(warnings) != 0 {
		t.Errorf("unexpected warnings: %v", warnings)
	}

	resultStr := string(result)

	// Should contain lvt code block
	if !strings.Contains(resultStr, "```lvt") {
		t.Error("expected lvt code block")
	}
	// Should contain lvt-source
	if !strings.Contains(resultStr, `lvt-source="expenses"`) {
		t.Error("expected lvt-source=\"expenses\"")
	}
	// Should still have the heading
	if !strings.Contains(resultStr, "## Expenses") {
		t.Error("heading should be preserved")
	}
	// Should have Add form (writable)
	if !strings.Contains(resultStr, `lvt-submit="Add"`) {
		t.Error("expected Add form for writable source")
	}
	// Original table should be removed
	if strings.Contains(resultStr, "|-------------|--------|") {
		t.Error("original table separator should be removed")
	}
}

func TestPreprocessAutoTablesReadonly(t *testing.T) {
	content := []byte(`---
title: Test
---

## Users
| Name | Email |
|------|-------|
`)

	sources := map[string]SourceConfig{
		"users": {Type: "rest", From: "https://example.com/users"},
	}

	result, _ := preprocessAutoTables(content, sources, "")
	resultStr := string(result)

	// Should contain Refresh button
	if !strings.Contains(resultStr, `lvt-click="Refresh"`) {
		t.Error("expected Refresh button for read-only source")
	}
	// Should NOT contain Add form
	if strings.Contains(resultStr, `lvt-submit="Add"`) {
		t.Error("should not have Add form for read-only source")
	}
}

func TestPreprocessAutoTablesNoMatch(t *testing.T) {
	content := []byte(`## Reviews
| Title | Rating |
|-------|--------|
`)

	sources := map[string]SourceConfig{
		"expenses": {Type: "sqlite"},
	}

	result, _ := preprocessAutoTables(content, sources, "")
	resultStr := string(result)

	// Should NOT have lvt code block
	if strings.Contains(resultStr, "```lvt") {
		t.Error("should not generate lvt block for unmatched table")
	}
	// Original table should remain
	if !strings.Contains(resultStr, "|-------|--------|") {
		t.Error("original table should remain for unmatched section")
	}
}

func TestPreprocessAutoTablesNoSources(t *testing.T) {
	content := []byte(`## Items
| Name | Price |
|------|-------|
`)

	result, warnings := preprocessAutoTables(content, nil, "")
	if len(warnings) != 0 {
		t.Errorf("unexpected warnings: %v", warnings)
	}
	if string(result) != string(content) {
		t.Error("content should be unchanged when no sources defined")
	}
}

func TestPreprocessAutoTablesMultipleSources(t *testing.T) {
	readonlyFalse := false
	content := []byte(`## Tasks
| Title | Done |
|-------|------|

## Contacts
| Name | Email |
|------|-------|
`)

	sources := map[string]SourceConfig{
		"tasks":    {Type: "sqlite", Readonly: &readonlyFalse},
		"contacts": {Type: "sqlite", Readonly: &readonlyFalse},
	}

	result, warnings := preprocessAutoTables(content, sources, "")
	if len(warnings) != 0 {
		t.Errorf("unexpected warnings: %v", warnings)
	}

	resultStr := string(result)

	// Should have both sources bound
	if !strings.Contains(resultStr, `lvt-source="tasks"`) {
		t.Error("expected tasks source binding")
	}
	if !strings.Contains(resultStr, `lvt-source="contacts"`) {
		t.Error("expected contacts source binding")
	}

	// Count lvt code blocks
	blockCount := strings.Count(resultStr, "```lvt")
	if blockCount != 2 {
		t.Errorf("expected 2 lvt blocks, got %d", blockCount)
	}
}

func TestPreprocessAutoTablesContainmentMatch(t *testing.T) {
	content := []byte(`## My Monthly Expenses
| Description | Amount |
|-------------|--------|
`)

	sources := map[string]SourceConfig{
		"expenses": {Type: "sqlite"},
	}

	result, _ := preprocessAutoTables(content, sources, "")
	resultStr := string(result)

	if !strings.Contains(resultStr, `lvt-source="expenses"`) {
		t.Error("expected containment match: 'My Monthly Expenses' should match 'expenses'")
	}
}

func TestPreprocessAutoTablesPreservesFrontmatter(t *testing.T) {
	content := []byte(`---
title: Test App
sources:
  items:
    type: sqlite
---

# My App

## Items
| Name | Price |
|------|-------|
`)

	sources := map[string]SourceConfig{
		"items": {Type: "sqlite"},
	}

	result, _ := preprocessAutoTables(content, sources, "")
	resultStr := string(result)

	// Frontmatter should be preserved
	if !strings.HasPrefix(resultStr, "---\ntitle: Test App") {
		t.Error("frontmatter should be preserved")
	}
}

// Integration test: verify ParseFile processes auto-tables correctly
func TestParseFileAutoTables(t *testing.T) {
	// Create a temp markdown file with a writable sqlite source
	tmpDir := t.TempDir()
	mdPath := tmpDir + "/test.md"
	err := os.WriteFile(mdPath, []byte(`---
title: Test
sources:
  items:
    type: sqlite
    db: ./test.db
    table: items
    readonly: false
---

# Test App

## Items
| Name | Price |
|------|-------|
`), 0644)
	if err != nil {
		t.Fatal(err)
	}

	page, err := ParseFile(mdPath)
	if err != nil {
		t.Fatalf("ParseFile failed: %v", err)
	}

	// The auto-table should have generated an interactive block
	if len(page.InteractiveBlocks) == 0 {
		t.Fatal("expected at least one interactive block from auto-table")
	}

	// Check that the page has the source configured
	if _, ok := page.Config.Sources["items"]; !ok {
		t.Error("expected 'items' source in page config")
	}

	// The static HTML should NOT contain the original markdown table
	if strings.Contains(page.StaticHTML, "|------|-------|") {
		t.Error("original markdown table should have been replaced by lvt block")
	}
}

func TestParseFileAutoTablesReadonly(t *testing.T) {
	tmpDir := t.TempDir()
	mdPath := tmpDir + "/test.md"
	err := os.WriteFile(mdPath, []byte(`---
title: Test
sources:
  users:
    type: rest
    from: https://example.com/users
---

# Test App

## Users
| Name | Email |
|------|-------|
`), 0644)
	if err != nil {
		t.Fatal(err)
	}

	page, err := ParseFile(mdPath)
	if err != nil {
		t.Fatalf("ParseFile failed: %v", err)
	}

	// Should have an interactive block (read-only table)
	if len(page.InteractiveBlocks) == 0 {
		t.Fatal("expected at least one interactive block from auto-table")
	}

	// Find the block and check it doesn't have Add form
	for _, block := range page.InteractiveBlocks {
		if strings.Contains(block.Content, `lvt-submit="Add"`) {
			t.Error("read-only source should not have Add form")
		}
		if strings.Contains(block.Content, `lvt-click="Delete"`) {
			t.Error("read-only source should not have Delete button")
		}
		if !strings.Contains(block.Content, `lvt-click="Refresh"`) {
			t.Error("read-only source should have Refresh button")
		}
	}
}

func TestParseFileAutoTablesNoMatch(t *testing.T) {
	tmpDir := t.TempDir()
	mdPath := tmpDir + "/test.md"
	err := os.WriteFile(mdPath, []byte(`---
title: Test
sources:
  products:
    type: sqlite
    db: ./test.db
    table: products
---

# Test App

## Orders
| Item | Quantity |
|------|----------|
`), 0644)
	if err != nil {
		t.Fatal(err)
	}

	page, err := ParseFile(mdPath)
	if err != nil {
		t.Fatalf("ParseFile failed: %v", err)
	}

	// No match between "Orders" heading and "products" source
	// So no interactive blocks should be generated from auto-tables
	// (page may have other blocks from other processing)
	for _, block := range page.InteractiveBlocks {
		if strings.Contains(block.Content, `lvt-source="products"`) {
			t.Error("should not auto-bind 'products' source to 'Orders' heading")
		}
	}

	// The original table should remain as static HTML
	if !strings.Contains(page.StaticHTML, "Orders") {
		t.Error("original heading should remain in static HTML")
	}
}
