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

	processed, sources := preprocessAutoTasks(content, "/test/page.md")

	// Should have one source
	if len(sources) != 1 {
		t.Fatalf("expected 1 source, got %d", len(sources))
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

	processed, sources := preprocessAutoTasks(content, "/test/page.md")

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

	processed, sources := preprocessAutoTasks(content, "/test/page.md")

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

	_, sources := preprocessAutoTasks(content, "/test/page.md")

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
