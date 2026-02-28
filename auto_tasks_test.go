package tinkerdown

import (
	"strings"
	"testing"
)

func TestDetectTaskListSections(t *testing.T) {
	content := []byte(`## Todos
- [ ] Buy groceries
- [x] Call mom

## Notes
Some regular text here.
`)

	sections := detectTaskListSections(content)
	if len(sections) != 1 {
		t.Fatalf("expected 1 section, got %d", len(sections))
	}

	sec := sections[0]
	if sec.anchor != "todos" {
		t.Errorf("expected anchor 'todos', got %q", sec.anchor)
	}
	if sec.level != 2 {
		t.Errorf("expected level 2, got %d", sec.level)
	}
	if !sec.hasTaskItems(content) {
		t.Error("section should contain task items")
	}
}

func TestDetectNoTasks(t *testing.T) {
	content := []byte(`## About
This is a regular section with no tasks.

## More
Just regular content.
`)

	sections := detectTaskListSections(content)
	if len(sections) != 0 {
		t.Fatalf("expected 0 sections, got %d", len(sections))
	}
}

func TestDetectMixedContent(t *testing.T) {
	content := []byte(`## Tasks
Here are some tasks:
- [ ] Buy groceries
- [x] Call mom
And some more text.
`)

	sections := detectTaskListSections(content)
	if len(sections) != 0 {
		t.Fatalf("expected 0 sections (mixed content), got %d", len(sections))
	}
}

func TestDetectMultipleSections(t *testing.T) {
	content := []byte(`## Morning
- [ ] Make coffee
- [x] Exercise

## Evening
- [ ] Cook dinner
- [ ] Read book
`)

	sections := detectTaskListSections(content)
	if len(sections) != 2 {
		t.Fatalf("expected 2 sections, got %d", len(sections))
	}

	if sections[0].anchor != "morning" {
		t.Errorf("expected first anchor 'morning', got %q", sections[0].anchor)
	}
	if sections[1].anchor != "evening" {
		t.Errorf("expected second anchor 'evening', got %q", sections[1].anchor)
	}
}

func TestDetectNoHeading(t *testing.T) {
	content := []byte(`- [ ] Buy groceries
- [x] Call mom
`)

	sections := detectTaskListSections(content)
	if len(sections) != 0 {
		t.Fatalf("expected 0 sections (no heading), got %d", len(sections))
	}
}

func TestDetectWithExplicitAnchor(t *testing.T) {
	content := []byte(`## My Todo List {#my-tasks}
- [ ] Item one
- [x] Item two
`)

	sections := detectTaskListSections(content)
	if len(sections) != 1 {
		t.Fatalf("expected 1 section, got %d", len(sections))
	}

	if sections[0].anchor != "my-tasks" {
		t.Errorf("expected anchor 'my-tasks', got %q", sections[0].anchor)
	}
}

func TestDetectDifferentHeadingLevels(t *testing.T) {
	content := []byte(`# Top Level
- [ ] Task A

### Deep Level
- [ ] Task B
- [x] Task C
`)

	sections := detectTaskListSections(content)
	if len(sections) != 2 {
		t.Fatalf("expected 2 sections, got %d", len(sections))
	}

	if sections[0].level != 1 {
		t.Errorf("expected first section level 1, got %d", sections[0].level)
	}
	if sections[1].level != 3 {
		t.Errorf("expected second section level 3, got %d", sections[1].level)
	}
}

func TestPreprocessAutoTasks(t *testing.T) {
	content := []byte(`# My Day

## Todos
- [ ] Buy groceries
- [x] Call mom

## Notes
Just some regular text.
`)

	processed, sources, warnings := preprocessAutoTasks(content, "/test/page.md")

	// Should have one source
	if len(sources) != 1 {
		t.Fatalf("expected 1 source, got %d", len(sources))
	}
	if len(warnings) != 0 {
		t.Errorf("expected no warnings, got %v", warnings)
	}

	src, ok := sources["_auto_todos"]
	if !ok {
		t.Fatal("expected source '_auto_todos'")
	}

	if src.Type != "markdown" {
		t.Errorf("expected type 'markdown', got %q", src.Type)
	}
	if src.File != "/test/page.md" {
		t.Errorf("expected file '/test/page.md', got %q", src.File)
	}
	if src.Anchor != "#todos" {
		t.Errorf("expected anchor '#todos', got %q", src.Anchor)
	}
	if src.Readonly == nil || *src.Readonly != false {
		t.Error("expected readonly to be false")
	}

	// Processed content should contain lvt code block
	processedStr := string(processed)
	if !strings.Contains(processedStr, "```lvt") {
		t.Error("processed content should contain ```lvt block")
	}
	if !strings.Contains(processedStr, `lvt-source="_auto_todos"`) {
		t.Error("processed content should contain lvt-source attribute")
	}

	// Original task items should be removed from processed content
	if strings.Contains(processedStr, "- [ ] Buy groceries") {
		t.Error("processed content should not contain original task items")
	}

	// Notes section should be preserved
	if !strings.Contains(processedStr, "## Notes") {
		t.Error("non-task sections should be preserved")
	}
	if !strings.Contains(processedStr, "Just some regular text.") {
		t.Error("non-task content should be preserved")
	}
}

func TestPreprocessWithFrontmatter(t *testing.T) {
	content := []byte(`---
title: "Test Page"
sources:
  other:
    type: markdown
    file: data.md
    anchor: "#data"
---

# My Day

## Todos
- [ ] Buy groceries
- [x] Call mom
`)

	processed, sources, _ := preprocessAutoTasks(content, "/test/page.md")

	// Should have auto source
	if len(sources) != 1 {
		t.Fatalf("expected 1 auto source, got %d", len(sources))
	}

	if _, ok := sources["_auto_todos"]; !ok {
		t.Fatal("expected auto source '_auto_todos'")
	}

	// Frontmatter should be preserved
	processedStr := string(processed)
	if !strings.Contains(processedStr, `title: "Test Page"`) {
		t.Error("frontmatter should be preserved")
	}

	// Should contain lvt block
	if !strings.Contains(processedStr, "```lvt") {
		t.Error("processed content should contain ```lvt block")
	}
}

func TestPreprocessNoTasks(t *testing.T) {
	content := []byte(`# About
This page has no tasks.

## More
Just text.
`)

	processed, sources, _ := preprocessAutoTasks(content, "/test/page.md")

	if sources != nil {
		t.Errorf("expected nil sources, got %d", len(sources))
	}

	// Content should be unchanged
	if string(processed) != string(content) {
		t.Error("content should be unchanged when no tasks detected")
	}
}

func TestPreprocessMultipleSections(t *testing.T) {
	content := []byte(`# My Day

## Morning
- [ ] Make coffee
- [x] Exercise

## Evening
- [ ] Cook dinner
- [ ] Read book
`)

	_, sources, warnings := preprocessAutoTasks(content, "/test/page.md")

	// Also guards the no-collision regression path (distinct anchors both kept)
	if len(warnings) != 0 {
		t.Errorf("expected no warnings for distinct anchors, got %v", warnings)
	}
	if len(sources) != 2 {
		t.Fatalf("expected 2 sources, got %d", len(sources))
	}

	if _, ok := sources["_auto_morning"]; !ok {
		t.Fatal("expected source '_auto_morning'")
	}
	if _, ok := sources["_auto_evening"]; !ok {
		t.Fatal("expected source '_auto_evening'")
	}
}

func TestSlugifyHeading(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"Todos", "todos"},
		{"My Task List", "my-task-list"},
		{"Morning Tasks!", "morning-tasks"},
		{"Hello World (v2)", "hello-world-v2"},
		{"café", "caf"},
	}

	for _, tt := range tests {
		result := slugifyHeading(tt.input)
		if result != tt.expected {
			t.Errorf("slugifyHeading(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}

func TestGenerateAutoTaskLvtBlock(t *testing.T) {
	block := generateAutoTaskLvtBlock("_auto_todos")

	// Should contain lvt-source
	if !strings.Contains(block, `lvt-source="_auto_todos"`) {
		t.Error("block should contain lvt-source attribute")
	}

	// Should contain toggle handler
	if !strings.Contains(block, `lvt-click="Toggle"`) {
		t.Error("block should contain Toggle action")
	}

	// Should contain add form
	if !strings.Contains(block, `lvt-submit="Add"`) {
		t.Error("block should contain Add form")
	}

	// Should contain checkbox template
	if !strings.Contains(block, `{{if .Done}}checked{{end}}`) {
		t.Error("block should contain checkbox template")
	}
}

func TestPreprocessDuplicateAnchorSameCasing(t *testing.T) {
	content := []byte(`# My Day

## Tasks
- [ ] Buy groceries

## Tasks
- [ ] Walk the dog
`)

	processed, sources, warnings := preprocessAutoTasks(content, "/test/page.md")

	// Only the first "Tasks" section should produce a source
	if len(sources) != 1 {
		t.Fatalf("expected 1 source (first occurrence only), got %d", len(sources))
	}

	src, ok := sources["_auto_tasks"]
	if !ok {
		t.Fatal("expected source '_auto_tasks' for first occurrence")
	}
	if src.Anchor != "#tasks" {
		t.Errorf("expected anchor '#tasks', got %q", src.Anchor)
	}

	// Should have exactly one warning about the collision
	if len(warnings) != 1 {
		t.Fatalf("expected 1 warning, got %d", len(warnings))
	}
	if !strings.Contains(warnings[0], "collides with") {
		t.Errorf("warning should mention collision, got %q", warnings[0])
	}
	// The duplicate "## Tasks" is on line 6 of the input (1-indexed)
	if !strings.Contains(warnings[0], "line 6") {
		t.Errorf("warning should report correct line number for duplicate, got %q", warnings[0])
	}

	// Processed content: first section replaced with lvt block, second stays as plain markdown
	processedStr := string(processed)
	if count := strings.Count(processedStr, `lvt-source="_auto_tasks"`); count != 1 {
		t.Errorf("expected exactly 1 lvt-source block, got %d", count)
	}
	// The skipped section's task items should remain as plain markdown
	if !strings.Contains(processedStr, "- [ ] Walk the dog") {
		t.Error("skipped duplicate section should retain its task items as plain markdown")
	}
	// The first section's task items should be replaced (not present as raw markdown)
	if strings.Contains(processedStr, "- [ ] Buy groceries") {
		t.Error("first section's task items should be replaced by lvt block")
	}
}

func TestPreprocessDuplicateAnchorDifferentCasing(t *testing.T) {
	content := []byte(`# My Day

## Tasks
- [ ] Buy groceries

## TASKS
- [ ] Walk the dog
`)

	processed, sources, warnings := preprocessAutoTasks(content, "/test/page.md")

	// "Tasks" and "TASKS" both slugify to "tasks" — only first kept
	if len(sources) != 1 {
		t.Fatalf("expected 1 source (same slug despite different casing), got %d", len(sources))
	}

	if _, ok := sources["_auto_tasks"]; !ok {
		t.Fatal("expected source '_auto_tasks'")
	}

	if len(warnings) != 1 {
		t.Fatalf("expected 1 warning, got %d", len(warnings))
	}

	// The skipped "TASKS" section should retain its task items as plain markdown
	processedStr := string(processed)
	if !strings.Contains(processedStr, "- [ ] Walk the dog") {
		t.Error("skipped duplicate section should retain its task items as plain markdown")
	}
}

func TestPreprocessDuplicateExplicitAnchor(t *testing.T) {
	content := []byte(`# My Day

## Morning Tasks {#my-tasks}
- [ ] Buy groceries

## Evening Tasks {#my-tasks}
- [ ] Walk the dog
`)

	processed, sources, warnings := preprocessAutoTasks(content, "/test/page.md")

	// Both use explicit anchor {#my-tasks} — only first kept
	if len(sources) != 1 {
		t.Fatalf("expected 1 source (explicit anchor collision), got %d", len(sources))
	}

	if _, ok := sources["_auto_my-tasks"]; !ok {
		t.Fatal("expected source '_auto_my-tasks'")
	}

	if len(warnings) != 1 {
		t.Fatalf("expected 1 warning, got %d", len(warnings))
	}

	// The skipped section should retain its task items as plain markdown
	processedStr := string(processed)
	if !strings.Contains(processedStr, "- [ ] Walk the dog") {
		t.Error("skipped duplicate section should retain its task items as plain markdown")
	}
	if count := strings.Count(processedStr, `lvt-source="_auto_my-tasks"`); count != 1 {
		t.Errorf("expected exactly 1 lvt-source block, got %d", count)
	}
}

func TestPreprocessTripleDuplicateAnchor(t *testing.T) {
	content := []byte(`## Tasks
- [ ] A

## Tasks
- [ ] B

## Tasks
- [ ] C
`)

	processed, sources, warnings := preprocessAutoTasks(content, "/test/page.md")

	if len(sources) != 1 {
		t.Fatalf("expected 1 source (first of three), got %d", len(sources))
	}
	if len(warnings) != 2 {
		t.Fatalf("expected 2 warnings (one per extra duplicate), got %d", len(warnings))
	}

	// The second and third sections' task items should remain as plain markdown
	processedStr := string(processed)
	if !strings.Contains(processedStr, "- [ ] B") {
		t.Error("second duplicate section should retain its task items as plain markdown")
	}
	if !strings.Contains(processedStr, "- [ ] C") {
		t.Error("third duplicate section should retain its task items as plain markdown")
	}
	// First section replaced
	if strings.Contains(processedStr, "- [ ] A") {
		t.Error("first section's task items should be replaced by lvt block")
	}
}

// Helper to verify task items exist in a section
func (s taskListSection) hasTaskItems(content []byte) bool {
	lines := strings.Split(string(content), "\n")
	for i := s.startLine; i < s.endLine && i < len(lines); i++ {
		if taskItemPattern.MatchString(lines[i]) {
			return true
		}
	}
	return false
}
