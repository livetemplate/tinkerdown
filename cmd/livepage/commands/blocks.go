package commands

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/livetemplate/livepage"
)

// BlocksCommand implements the blocks command.
func BlocksCommand(args []string) error {
	// Parse arguments
	dir := "."
	verbose := false

	for i, arg := range args {
		if arg == "--verbose" || arg == "-v" {
			verbose = true
		} else if i == 0 {
			dir = arg
		}
	}

	// Check if directory exists
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return fmt.Errorf("directory does not exist: %s", dir)
	}

	// Get absolute path
	absDir, err := filepath.Abs(dir)
	if err != nil {
		return fmt.Errorf("failed to get absolute path: %w", err)
	}

	fmt.Printf("🔍 Inspecting blocks in: %s\n\n", absDir)

	// Track statistics
	var totalBlocks int
	var serverCount int
	var wasmCount int
	var lvtCount int
	var fileBlocks []fileBlockInfo

	// Discover and inspect all markdown files
	err = filepath.WalkDir(absDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Skip directories
		if d.IsDir() {
			// Skip directories starting with _ or .
			name := d.Name()
			if strings.HasPrefix(name, "_") || strings.HasPrefix(name, ".") {
				return filepath.SkipDir
			}
			return nil
		}

		// Only process .md files
		if filepath.Ext(path) != ".md" {
			return nil
		}

		// Get relative path for display
		relPath, err := filepath.Rel(absDir, path)
		if err != nil {
			relPath = path
		}

		// Parse the file to get blocks
		page, err := livepage.ParseFile(path)
		if err != nil {
			// Collect error but don't fail completely
			fmt.Printf("⚠️  %s: Failed to parse: %v\n\n", relPath, err)
			return nil
		}

		// Parse raw markdown to get code blocks with line numbers
		content, _ := os.ReadFile(path)
		_, codeBlocks, _, err := livepage.ParseMarkdown(content)
		if err != nil {
			return nil
		}

		if len(codeBlocks) > 0 {
			fileBlocks = append(fileBlocks, fileBlockInfo{
				file:       relPath,
				page:       page,
				codeBlocks: codeBlocks,
			})

			// Count blocks
			totalBlocks += len(codeBlocks)
			serverCount += len(page.ServerBlocks)
			wasmCount += len(page.WasmBlocks)
			lvtCount += len(page.InteractiveBlocks)
		}

		return nil
	})

	if err != nil {
		return fmt.Errorf("failed to walk directory: %w", err)
	}

	// Display results
	if len(fileBlocks) == 0 {
		fmt.Println("No code blocks found.")
		return nil
	}

	// Print blocks for each file
	for _, fb := range fileBlocks {
		if verbose {
			printVerboseBlocks(fb)
		} else {
			printBasicBlocks(fb)
		}
	}

	// Print summary
	separator := strings.Repeat("─", 60) + "\n"
	fmt.Print(separator)
	fmt.Println("Summary:")
	fmt.Printf("  Total blocks: %d\n", totalBlocks)
	fmt.Printf("  Server blocks: %d\n", serverCount)
	fmt.Printf("  WASM blocks: %d\n", wasmCount)
	fmt.Printf("  Interactive blocks: %d\n", lvtCount)
	fmt.Println()

	return nil
}

type fileBlockInfo struct {
	file       string
	page       *livepage.Page
	codeBlocks []*livepage.CodeBlock
}

func printBasicBlocks(fb fileBlockInfo) {
	fmt.Printf("%s:\n", fb.file)

	for i, cb := range fb.codeBlocks {
		blockID := getBlockIDFromCodeBlock(cb, i)
		flags := ""
		if len(cb.Flags) > 0 {
			flags = ", " + strings.Join(cb.Flags, ", ")
		}

		fmt.Printf("  Line %d: %s (%s%s)\n", cb.Line, blockID, cb.Type, flags)

		// Additional info based on type
		switch cb.Type {
		case "server":
			// Try to extract state name from content
			if stateName := extractStateName(cb.Content); stateName != "" {
				fmt.Printf("           State: %s\n", stateName)
			}
		case "lvt":
			stateRef := cb.Metadata["state"]
			if stateRef == "" {
				stateRef = "(auto-linked to nearest server)"
			}
			fmt.Printf("           References: %s\n", stateRef)
			lines := strings.Count(cb.Content, "\n") + 1
			fmt.Printf("           Template: %d lines\n", lines)
		case "wasm":
			lines := strings.Count(cb.Content, "\n") + 1
			fmt.Printf("           Code: %d lines\n", lines)
		}
		fmt.Println()
	}
}

func printVerboseBlocks(fb fileBlockInfo) {
	fmt.Printf("%s:\n\n", fb.file)

	for i, cb := range fb.codeBlocks {
		blockID := getBlockIDFromCodeBlock(cb, i)

		fmt.Printf("Block: %s\n", blockID)
		fmt.Printf("  Type: %s\n", cb.Type)
		fmt.Printf("  Language: %s\n", cb.Language)
		if len(cb.Flags) > 0 {
			fmt.Printf("  Flags: %s\n", strings.Join(cb.Flags, ", "))
		}
		fmt.Printf("  Location: %s:%d\n", fb.file, cb.Line)

		// Type-specific details
		switch cb.Type {
		case "server":
			if id := cb.Metadata["id"]; id != "" {
				fmt.Printf("  ID: %s\n", id)
			}
			if stateName := extractStateName(cb.Content); stateName != "" {
				fmt.Printf("  State: %s\n", stateName)
			}
			// Show first few lines of content
			lines := strings.Split(cb.Content, "\n")
			preview := strings.Join(lines[:min(3, len(lines))], "\n")
			if len(lines) > 3 {
				preview += "..."
			}
			fmt.Printf("  Content: %s\n", preview)

		case "lvt":
			stateRef := cb.Metadata["state"]
			if stateRef == "" {
				stateRef = "(auto-linked)"
			}
			fmt.Printf("  State Ref: %s\n", stateRef)
			lines := strings.Count(cb.Content, "\n") + 1
			fmt.Printf("  Template Lines: %d\n", lines)

		case "wasm":
			if id := cb.Metadata["id"]; id != "" {
				fmt.Printf("  ID: %s\n", id)
			}
			lines := strings.Count(cb.Content, "\n") + 1
			fmt.Printf("  Code Lines: %d\n", lines)
		}

		fmt.Println()
	}
}

// getBlockIDFromCodeBlock returns the block ID (explicit or auto-generated)
func getBlockIDFromCodeBlock(cb *livepage.CodeBlock, index int) string {
	if id := cb.Metadata["id"]; id != "" {
		return id
	}
	return fmt.Sprintf("%s-%d", cb.Type, index)
}

// extractStateName tries to extract the state struct name from Go code
func extractStateName(content string) string {
	lines := strings.Split(content, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "type ") && strings.Contains(line, "struct") {
			// Extract: "type FooState struct {" -> "FooState"
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				return parts[1]
			}
		}
	}
	return ""
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
