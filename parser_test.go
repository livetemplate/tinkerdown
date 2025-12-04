package livepage

import (
	"strings"
	"testing"
)

func TestParseFrontmatter(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		wantFM   Frontmatter
		wantBody string
	}{
		{
			name: "complete frontmatter",
			content: `---
title: "Test Tutorial"
type: guide
persist: server
steps: 5
---

# Hello World`,
			wantFM: Frontmatter{
				Title:   "Test Tutorial",
				Type:    "guide",
				Persist: PersistServer,
				Steps:   5,
			},
			wantBody: "# Hello World",
		},
		{
			name: "no frontmatter",
			content: `# Hello World

Some content`,
			wantFM: Frontmatter{
				Type:    "tutorial",
				Persist: PersistLocalStorage,
			},
			wantBody: "# Hello World\n\nSome content",
		},
		{
			name: "minimal frontmatter",
			content: `---
title: "Simple"
---

Content`,
			wantFM: Frontmatter{
				Title:   "Simple",
				Type:    "tutorial",
				Persist: PersistLocalStorage,
			},
			wantBody: "Content",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fm, remaining, err := extractFrontmatter([]byte(tt.content))
			if err != nil {
				t.Fatalf("extractFrontmatter() error = %v", err)
			}

			if fm.Title != tt.wantFM.Title {
				t.Errorf("Title = %q, want %q", fm.Title, tt.wantFM.Title)
			}
			if fm.Type != tt.wantFM.Type {
				t.Errorf("Type = %q, want %q", fm.Type, tt.wantFM.Type)
			}
			if fm.Persist != tt.wantFM.Persist {
				t.Errorf("Persist = %q, want %q", fm.Persist, tt.wantFM.Persist)
			}
			if fm.Steps != tt.wantFM.Steps {
				t.Errorf("Steps = %d, want %d", fm.Steps, tt.wantFM.Steps)
			}

			body := strings.TrimSpace(string(remaining))
			want := strings.TrimSpace(tt.wantBody)
			if body != want {
				t.Errorf("remaining body = %q, want %q", body, want)
			}
		})
	}
}

func TestParseCodeBlocks(t *testing.T) {
	tests := []struct {
		name       string
		content    string
		wantBlocks int
		checkBlock func(*testing.T, *CodeBlock)
	}{
		{
			name: "server readonly block",
			content: `# Tutorial

` + "```go server readonly id=\"counter-state\"" + `
type CounterState struct {
    Counter int
}
` + "```" + `

Text after`,
			wantBlocks: 1,
			checkBlock: func(t *testing.T, b *CodeBlock) {
				if b.Type != "server" {
					t.Errorf("Type = %q, want \"server\"", b.Type)
				}
				if b.Language != "go" {
					t.Errorf("Language = %q, want \"go\"", b.Language)
				}
				if !containsString(b.Flags, "readonly") {
					t.Errorf("Flags = %v, want to contain \"readonly\"", b.Flags)
				}
				if b.Metadata["id"] != "counter-state" {
					t.Errorf("Metadata[id] = %q, want \"counter-state\"", b.Metadata["id"])
				}
				if !strings.Contains(b.Content, "CounterState") {
					t.Errorf("Content missing \"CounterState\"")
				}
			},
		},
		{
			name: "wasm editable block",
			content: `# Tutorial

` + "```go wasm editable" + `
package main
func main() {}
` + "```",
			wantBlocks: 1,
			checkBlock: func(t *testing.T, b *CodeBlock) {
				if b.Type != "wasm" {
					t.Errorf("Type = %q, want \"wasm\"", b.Type)
				}
				if !containsString(b.Flags, "editable") {
					t.Errorf("Flags = %v, want to contain \"editable\"", b.Flags)
				}
				if !strings.Contains(b.Content, "package main") {
					t.Errorf("Content missing \"package main\"")
				}
			},
		},
		{
			name: "lvt interactive block",
			content: `# Tutorial

` + "```lvt interactive state=\"counter\"" + `
<button lvt-click="increment">{{.Counter}}</button>
` + "```",
			wantBlocks: 1,
			checkBlock: func(t *testing.T, b *CodeBlock) {
				if b.Type != "lvt" {
					t.Errorf("Type = %q, want \"lvt\"", b.Type)
				}
				if b.Language != "lvt" {
					t.Errorf("Language = %q, want \"lvt\"", b.Language)
				}
				if b.Metadata["state"] != "counter" {
					t.Errorf("Metadata[state] = %q, want \"counter\"", b.Metadata["state"])
				}
				if !strings.Contains(b.Content, "lvt-click") {
					t.Errorf("Content missing \"lvt-click\"")
				}
			},
		},
		{
			name: "regular code block ignored",
			content: `# Tutorial

` + "```go" + `
func Hello() {}
` + "```",
			wantBlocks: 0,
		},
		{
			name: "multiple blocks",
			content: `# Tutorial

` + "```go server readonly id=\"state\"" + `
type State struct {}
` + "```" + `

Some text

` + "```lvt interactive state=\"state\"" + `
<div>Interactive</div>
` + "```" + `

More text

` + "```go wasm editable" + `
package main
` + "```",
			wantBlocks: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, blocks, _, err := ParseMarkdown([]byte(tt.content))
			if err != nil {
				t.Fatalf("ParseMarkdown() error = %v", err)
			}

			if len(blocks) != tt.wantBlocks {
				t.Fatalf("got %d blocks, want %d", len(blocks), tt.wantBlocks)
			}

			if tt.checkBlock != nil && len(blocks) > 0 {
				tt.checkBlock(t, blocks[0])
			}
		})
	}
}

func TestParseMarkdownComplete(t *testing.T) {
	content := `---
title: "Counter Tutorial"
type: tutorial
persist: localstorage
---

# Build a Counter

Learn how to build a counter.

## Step 1: State

` + "```go server readonly id=\"counter-state\"" + `
type CounterState struct {
    Counter int ` + "`json:\"counter\"`" + `
}

// Increment handles the "increment" action
func (s *CounterState) Increment(_ *ActionContext) error {
    s.Counter++
    return nil
}
` + "```" + `

This defines the state.

## Step 2: UI

` + "```lvt interactive state=\"counter-state\"" + `
<div>
    <h2>Count: {{.Counter}}</h2>
    <button lvt-click="increment">+1</button>
</div>
` + "```" + `

Click to increment!

## Try It

` + "```go wasm editable" + `
package main

import "fmt"

func main() {
    fmt.Println("Hello")
}
` + "```" + `

Modify and run!
`

	fm, blocks, html, err := ParseMarkdown([]byte(content))
	if err != nil {
		t.Fatalf("ParseMarkdown() error = %v", err)
	}

	// Check frontmatter
	if fm.Title != "Counter Tutorial" {
		t.Errorf("Title = %q, want \"Counter Tutorial\"", fm.Title)
	}
	if fm.Type != "tutorial" {
		t.Errorf("Type = %q, want \"tutorial\"", fm.Type)
	}
	if fm.Persist != PersistLocalStorage {
		t.Errorf("Persist = %q, want \"localstorage\"", fm.Persist)
	}

	// Check blocks
	if len(blocks) != 3 {
		t.Fatalf("got %d blocks, want 3", len(blocks))
	}

	// Block 1: server
	if blocks[0].Type != "server" {
		t.Errorf("Block 0 Type = %q, want \"server\"", blocks[0].Type)
	}
	if blocks[0].Metadata["id"] != "counter-state" {
		t.Errorf("Block 0 id = %q, want \"counter-state\"", blocks[0].Metadata["id"])
	}

	// Block 2: lvt
	if blocks[1].Type != "lvt" {
		t.Errorf("Block 1 Type = %q, want \"lvt\"", blocks[1].Type)
	}
	if blocks[1].Metadata["state"] != "counter-state" {
		t.Errorf("Block 1 state = %q, want \"counter-state\"", blocks[1].Metadata["state"])
	}

	// Block 3: wasm
	if blocks[2].Type != "wasm" {
		t.Errorf("Block 2 Type = %q, want \"wasm\"", blocks[2].Type)
	}

	// Check HTML (should contain prose)
	if !strings.Contains(html, "Build a Counter") {
		t.Error("HTML missing title")
	}
	if !strings.Contains(html, "Learn how to build") {
		t.Error("HTML missing intro text")
	}
	// Note: readonly code blocks are now intentionally included in HTML
	// for educational/documentation purposes, so we don't check that they're excluded
}

func containsString(slice []string, str string) bool {
	for _, s := range slice {
		if s == str {
			return true
		}
	}
	return false
}
