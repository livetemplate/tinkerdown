package commands

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/livetemplate/livepage"
)

// FixCommand implements the fix command to auto-fix common issues.
func FixCommand(args []string) error {
	// Parse arguments
	dir := "."
	dryRun := false

	for i, arg := range args {
		if arg == "--dry-run" || arg == "-n" {
			dryRun = true
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

	if dryRun {
		fmt.Printf("🔍 Checking livepage files in: %s (dry-run mode)\n\n", absDir)
	} else {
		fmt.Printf("🔧 Fixing livepage files in: %s\n\n", absDir)
	}

	var totalFiles int
	var fixedFiles int
	var totalFixes int
	var fileResults []fileFixResult

	err = filepath.WalkDir(absDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Skip directories
		if d.IsDir() {
			name := d.Name()
			// Skip hidden directories (starting with . or _)
			if strings.HasPrefix(name, "_") || strings.HasPrefix(name, ".") {
				return filepath.SkipDir
			}
			// Skip common non-documentation directories
			skipDirs := []string{"node_modules", "vendor", "dist", "build", "target", ".git"}
			for _, skip := range skipDirs {
				if name == skip {
					return filepath.SkipDir
				}
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

		totalFiles++

		// Try to fix the file
		fixes, err := fixFile(path, dryRun)
		if err != nil {
			fileResults = append(fileResults, fileFixResult{
				file:  relPath,
				fixes: nil,
				error: err.Error(),
			})
			return nil
		}

		if len(fixes) > 0 {
			fixedFiles++
			totalFixes += len(fixes)
			fileResults = append(fileResults, fileFixResult{
				file:  relPath,
				fixes: fixes,
				error: "",
			})
		}

		return nil
	})

	if err != nil {
		return fmt.Errorf("failed to walk directory: %w", err)
	}

	// Print results
	for _, fr := range fileResults {
		if fr.error != "" {
			fmt.Printf("✗ %s: %s\n", fr.file, fr.error)
		} else if len(fr.fixes) > 0 {
			status := "✓"
			if dryRun {
				status = "○"
			}
			fmt.Printf("%s %s: %d fix(es)\n", status, fr.file, len(fr.fixes))
			for _, fix := range fr.fixes {
				fmt.Printf("  - %s\n", fix)
			}
		}
	}

	// Print summary
	separator := "\n" + strings.Repeat("─", 60) + "\n"
	fmt.Print(separator)
	fmt.Println("Summary:")
	fmt.Printf("  Total files:  %d\n", totalFiles)
	fmt.Printf("  Files fixed:  %d\n", fixedFiles)
	fmt.Printf("  Total fixes:  %d\n", totalFixes)
	fmt.Printf("\n")

	if dryRun && totalFixes > 0 {
		fmt.Printf("💡 Run without --dry-run to apply fixes\n")
	} else if totalFixes > 0 {
		fmt.Printf("✓ Fixed %d issue(s) in %d file(s)\n", totalFixes, fixedFiles)
	} else {
		fmt.Printf("✓ No issues found\n")
	}

	return nil
}

type fileFixResult struct {
	file  string
	fixes []string
	error string
}

// fixFile attempts to fix common issues in a markdown file
func fixFile(path string, dryRun bool) ([]string, error) {
	// Read file
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	original := string(content)
	fixed := original
	var fixes []string

	// Fix 1: Normalize line endings (CRLF -> LF)
	if strings.Contains(fixed, "\r\n") {
		fixed = strings.ReplaceAll(fixed, "\r\n", "\n")
		fixes = append(fixes, "Normalized line endings (CRLF → LF)")
	}

	// Fix 2: Remove trailing whitespace on lines
	lines := strings.Split(fixed, "\n")
	hasTrailingWhitespace := false
	for i, line := range lines {
		trimmed := strings.TrimRight(line, " \t")
		if trimmed != line {
			lines[i] = trimmed
			hasTrailingWhitespace = true
		}
	}
	if hasTrailingWhitespace {
		fixed = strings.Join(lines, "\n")
		fixes = append(fixes, "Removed trailing whitespace")
	}

	// Fix 3: Ensure file ends with single newline
	if !strings.HasSuffix(fixed, "\n") {
		fixed += "\n"
		fixes = append(fixes, "Added newline at end of file")
	} else if strings.HasSuffix(fixed, "\n\n\n") {
		// Remove excessive newlines at end
		fixed = strings.TrimRight(fixed, "\n") + "\n"
		fixes = append(fixes, "Removed excessive newlines at end of file")
	}

	// Fix 4: Normalize multiple blank lines to maximum of 2
	for strings.Contains(fixed, "\n\n\n\n") {
		fixed = strings.ReplaceAll(fixed, "\n\n\n\n", "\n\n\n")
	}
	if fixed != original && !contains(fixes, "Removed excessive newlines") {
		fixes = append(fixes, "Normalized multiple blank lines")
	}

	// Validate that the fixed file parses correctly
	if len(fixes) > 0 && !dryRun {
		// Write to temp file and validate
		tmpFile := path + ".tmp"
		if err := os.WriteFile(tmpFile, []byte(fixed), 0644); err != nil {
			return nil, fmt.Errorf("failed to write temp file: %w", err)
		}

		// Try to parse the fixed content
		_, err := livepage.ParseFile(tmpFile)
		os.Remove(tmpFile) // Clean up temp file

		if err != nil {
			return nil, fmt.Errorf("fixes would break file: %w", err)
		}

		// Write the fixed content
		if err := os.WriteFile(path, []byte(fixed), 0644); err != nil {
			return nil, fmt.Errorf("failed to write file: %w", err)
		}
	}

	return fixes, nil
}

// Helper function to check if a slice contains a string
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if strings.Contains(s, item) {
			return true
		}
	}
	return false
}
