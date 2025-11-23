package livepage

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ParseFile parses a markdown file and creates a Page.
func ParseFile(path string) (*Page, error) {
	// Read file
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	// Get absolute path for better error messages
	absPath, err := filepath.Abs(path)
	if err != nil {
		absPath = path
	}

	// Parse markdown
	fm, codeBlocks, staticHTML, err := ParseMarkdown(content)
	if err != nil {
		// Wrap with file context
		return nil, NewParseError(absPath, 1, fmt.Sprintf("Failed to parse markdown: %v", err))
	}

	// Create page
	page := New(filepath.Base(path))
	page.Title = fm.Title
	page.Type = fm.Type
	page.StaticHTML = staticHTML
	page.SourceFile = absPath // Track source file
	page.Config = PageConfig{
		Persist:   fm.Persist,
		MultiStep: fm.Steps > 0,
		StepCount: fm.Steps,
	}

	// Build blocks (pass source file for error context)
	if err := page.buildBlocks(codeBlocks, absPath); err != nil {
		return nil, err // Already a ParseError from buildBlocks
	}

	return page, nil
}

// buildBlocks converts parsed code blocks into typed block structures.
func (p *Page) buildBlocks(codeBlocks []*CodeBlock, sourceFile string) error {
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
			return NewParseError(sourceFile, cb.Line, fmt.Sprintf("Unknown block type: %s", cb.Type)).
				WithHint(fmt.Sprintf("Valid block types are: server, wasm, lvt"))
		}
	}

	// Second pass: validate references
	for id, block := range p.InteractiveBlocks {
		if block.StateRef == "" {
			// Find the lvt block's line number
			var blockLine int
			for _, cb := range codeBlocks {
				if getBlockID(cb, 0) == id || cb.Metadata["id"] == id {
					blockLine = cb.Line
					break
				}
			}
			return NewParseError(sourceFile, blockLine, "Interactive block has no state reference").
				WithHint("Add a server block with state definition before this interactive block, or specify state=\"block-id\"")
		}

		if _, ok := p.ServerBlocks[block.StateRef]; !ok {
			// Find the lvt block's line number
			var blockLine int
			for _, cb := range codeBlocks {
				if getBlockID(cb, 0) == id || cb.Metadata["id"] == id {
					blockLine = cb.Line
					break
				}
			}

			// Find if there's a similar block name (did you mean?)
			var hint string
			for serverID := range p.ServerBlocks {
				if strings.Contains(serverID, block.StateRef) || strings.Contains(block.StateRef, serverID) {
					hint = fmt.Sprintf("Did you mean state=\"%s\"?", serverID)
					break
				}
			}
			if hint == "" {
				hint = fmt.Sprintf("Available server blocks: %v", getMapKeys(p.ServerBlocks))
			}

			return NewParseError(sourceFile, blockLine, fmt.Sprintf("Interactive block references unknown state '%s'", block.StateRef)).
				WithHint(hint)
		}
	}

	return nil
}

// Helper to get map keys as a slice
func getMapKeys(m map[string]*ServerBlock) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

// getBlockID extracts or generates a block ID.
func getBlockID(cb *CodeBlock, index int) string {
	if id, ok := cb.Metadata["id"]; ok {
		return id
	}

	// Generate ID from type and index
	return fmt.Sprintf("%s-%d", cb.Type, index)
}
