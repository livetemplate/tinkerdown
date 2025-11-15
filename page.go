package livepage

import (
	"fmt"
	"os"
	"path/filepath"
)

// ParseFile parses a markdown file and creates a Page.
func ParseFile(path string) (*Page, error) {
	// Read file
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	// Parse markdown
	fm, codeBlocks, staticHTML, err := ParseMarkdown(content)
	if err != nil {
		return nil, fmt.Errorf("failed to parse markdown: %w", err)
	}

	// Create page
	page := New(filepath.Base(path))
	page.Title = fm.Title
	page.Type = fm.Type
	page.StaticHTML = staticHTML
	page.Config = PageConfig{
		Persist:   fm.Persist,
		MultiStep: fm.Steps > 0,
		StepCount: fm.Steps,
	}

	// Build blocks
	if err := page.buildBlocks(codeBlocks); err != nil {
		return nil, fmt.Errorf("failed to build blocks: %w", err)
	}

	return page, nil
}

// buildBlocks converts parsed code blocks into typed block structures.
func (p *Page) buildBlocks(codeBlocks []*CodeBlock) error {
	// Track the most recent server block ID for auto-linking
	var lastServerBlockID string

	// First pass: build all blocks
	for i, cb := range codeBlocks {
		switch cb.Type {
		case "server":
			block := &ServerBlock{
				ID:       getBlockID(cb, i),
				Language: cb.Language,
				Content:  cb.Content,
				Metadata: cb.Metadata,
			}
			p.ServerBlocks[block.ID] = block
			lastServerBlockID = block.ID // Track for auto-linking

		case "wasm":
			block := &WasmBlock{
				ID:            getBlockID(cb, i),
				Language:      cb.Language,
				DefaultCode:   cb.Content,
				ShowRunButton: true, // Default to showing run button
				Metadata:      cb.Metadata,
			}
			p.WasmBlocks[block.ID] = block

		case "lvt":
			// Auto-link to nearest previous server block if no explicit state ref
			stateRef := cb.Metadata["state"]
			if stateRef == "" {
				stateRef = lastServerBlockID
			}

			block := &InteractiveBlock{
				ID:       getBlockID(cb, i),
				StateRef: stateRef,
				Content:  cb.Content,
				Metadata: cb.Metadata,
			}
			p.InteractiveBlocks[block.ID] = block

		default:
			return fmt.Errorf("unknown block type: %s", cb.Type)
		}
	}

	// Second pass: validate references
	for id, block := range p.InteractiveBlocks {
		if block.StateRef == "" {
			return fmt.Errorf("interactive block %s has no state reference (no server block found before it)", id)
		}

		if _, ok := p.ServerBlocks[block.StateRef]; !ok {
			return fmt.Errorf("interactive block %s references unknown state %s", id, block.StateRef)
		}
	}

	return nil
}

// getBlockID extracts or generates a block ID.
func getBlockID(cb *CodeBlock, index int) string {
	if id, ok := cb.Metadata["id"]; ok {
		return id
	}

	// Generate ID from type and index
	return fmt.Sprintf("%s-%d", cb.Type, index)
}
