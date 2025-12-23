package source

import (
	"bufio"
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// MarkdownSource reads data from markdown sections (task lists, bullet lists, tables)
type MarkdownSource struct {
	name       string
	filePath   string // empty means current file (handled by caller)
	anchor     string // e.g., "#todos"
	readonly   bool
	siteDir    string
	currentFile string // the markdown file being served (for same-file anchors)
}

// NewMarkdownSource creates a new markdown source
func NewMarkdownSource(name, file, anchor, siteDir, currentFile string, readonly bool) (*MarkdownSource, error) {
	if anchor == "" {
		return nil, fmt.Errorf("markdown source %q: anchor is required", name)
	}

	// Normalize anchor to include #
	if !strings.HasPrefix(anchor, "#") {
		anchor = "#" + anchor
	}

	return &MarkdownSource{
		name:        name,
		filePath:    file,
		anchor:      anchor,
		readonly:    readonly,
		siteDir:     siteDir,
		currentFile: currentFile,
	}, nil
}

// Name returns the source identifier
func (s *MarkdownSource) Name() string {
	return s.name
}

// Fetch reads and parses the markdown section
func (s *MarkdownSource) Fetch(ctx context.Context) ([]map[string]interface{}, error) {
	path := s.resolvePath()

	content, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("markdown source %q: failed to read file: %w", s.name, err)
	}

	return s.parseSection(string(content))
}

// Close is a no-op for file sources
func (s *MarkdownSource) Close() error {
	return nil
}

// IsReadonly returns whether the source is read-only
func (s *MarkdownSource) IsReadonly() bool {
	return s.readonly
}

// resolvePath determines which file to read
func (s *MarkdownSource) resolvePath() string {
	if s.filePath == "" {
		// Same file - use currentFile
		return s.currentFile
	}
	if filepath.IsAbs(s.filePath) {
		return s.filePath
	}
	return filepath.Join(s.siteDir, s.filePath)
}

// parseSection finds and parses the data section by anchor
func (s *MarkdownSource) parseSection(content string) ([]map[string]interface{}, error) {
	// Find the section by anchor: ## Title {#anchor}
	anchorName := strings.TrimPrefix(s.anchor, "#")

	// Pattern: ## Title {#anchor} or # Title {#anchor}
	headerPattern := regexp.MustCompile(`(?m)^(#{1,6})\s+(.+?)\s*\{#` + regexp.QuoteMeta(anchorName) + `\}\s*$`)

	matches := headerPattern.FindStringSubmatchIndex(content)
	if matches == nil {
		return []map[string]interface{}{}, nil // No section found, return empty
	}

	// Find where section content starts (after the header line)
	sectionStart := matches[1] // End of the match

	// Find where section ends (next header of same or higher level, or EOF)
	headerLevel := len(content[matches[2]:matches[3]]) // Number of # characters

	sectionEnd := len(content)
	nextHeaderPattern := regexp.MustCompile(`(?m)^#{1,` + fmt.Sprintf("%d", headerLevel) + `}\s+`)
	if loc := nextHeaderPattern.FindStringIndex(content[sectionStart:]); loc != nil {
		sectionEnd = sectionStart + loc[0]
	}

	sectionContent := content[sectionStart:sectionEnd]

	// Detect format and parse
	return s.detectAndParse(sectionContent)
}

// detectAndParse auto-detects the format and parses accordingly
func (s *MarkdownSource) detectAndParse(content string) ([]map[string]interface{}, error) {
	lines := strings.Split(content, "\n")

	// Check for task list: - [ ] or - [x]
	taskListPattern := regexp.MustCompile(`^\s*-\s+\[([ xX])\]\s+`)
	// Check for bullet list: - item
	bulletListPattern := regexp.MustCompile(`^\s*-\s+[^\[]`)
	// Check for table: | col | col |
	tablePattern := regexp.MustCompile(`^\s*\|.+\|`)

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		if taskListPattern.MatchString(line) {
			return s.parseTaskList(lines)
		}
		if tablePattern.MatchString(line) {
			return s.parseTable(lines)
		}
		if bulletListPattern.MatchString(line) {
			return s.parseBulletList(lines)
		}
	}

	return []map[string]interface{}{}, nil
}

// parseTaskList parses - [ ] item <!-- id:xxx --> format
func (s *MarkdownSource) parseTaskList(lines []string) ([]map[string]interface{}, error) {
	var results []map[string]interface{}

	taskPattern := regexp.MustCompile(`^\s*-\s+\[([ xX])\]\s+(.+?)(?:\s*<!--\s*id:(\w+)\s*-->)?$`)

	for _, line := range lines {
		matches := taskPattern.FindStringSubmatch(line)
		if matches == nil {
			continue
		}

		done := matches[1] == "x" || matches[1] == "X"
		text := strings.TrimSpace(matches[2])
		id := matches[3]

		// Generate ID if missing
		if id == "" {
			id = generateID()
		}

		results = append(results, map[string]interface{}{
			"id":   id,
			"text": text,
			"done": done,
		})
	}

	return results, nil
}

// parseBulletList parses - item <!-- id:xxx --> format
func (s *MarkdownSource) parseBulletList(lines []string) ([]map[string]interface{}, error) {
	var results []map[string]interface{}

	bulletPattern := regexp.MustCompile(`^\s*-\s+(.+?)(?:\s*<!--\s*id:(\w+)\s*-->)?$`)

	for _, line := range lines {
		matches := bulletPattern.FindStringSubmatch(line)
		if matches == nil {
			continue
		}

		text := strings.TrimSpace(matches[1])
		id := matches[2]

		// Skip task list items that might slip through
		if strings.HasPrefix(text, "[ ]") || strings.HasPrefix(text, "[x]") || strings.HasPrefix(text, "[X]") {
			continue
		}

		// Generate ID if missing
		if id == "" {
			id = generateID()
		}

		results = append(results, map[string]interface{}{
			"id":   id,
			"text": text,
		})
	}

	return results, nil
}

// parseTable parses | col1 | col2 | <!-- id:xxx --> format
func (s *MarkdownSource) parseTable(lines []string) ([]map[string]interface{}, error) {
	var results []map[string]interface{}
	var headers []string
	headerParsed := false
	separatorSeen := false

	tableRowPattern := regexp.MustCompile(`^\s*\|(.+)\|(?:\s*<!--\s*id:(\w+)\s*-->)?`)
	separatorPattern := regexp.MustCompile(`^\s*\|[\s\-:|]+\|`)

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Check for separator row (|---|---|)
		if separatorPattern.MatchString(line) {
			separatorSeen = true
			continue
		}

		matches := tableRowPattern.FindStringSubmatch(line)
		if matches == nil {
			continue
		}

		cells := s.parseTableCells(matches[1])
		id := matches[2]

		// First row is headers
		if !headerParsed {
			headers = cells
			headerParsed = true
			continue
		}

		// Skip if we haven't seen the separator yet (still in header area)
		if !separatorSeen {
			continue
		}

		// Generate ID if missing
		if id == "" {
			id = generateID()
		}

		row := map[string]interface{}{
			"id": id,
		}
		for i, cell := range cells {
			if i < len(headers) {
				row[headers[i]] = cell
			}
		}

		results = append(results, row)
	}

	return results, nil
}

// parseTableCells splits a table row into cells
func (s *MarkdownSource) parseTableCells(row string) []string {
	parts := strings.Split(row, "|")
	var cells []string
	for _, part := range parts {
		cell := strings.TrimSpace(part)
		if cell != "" {
			cells = append(cells, cell)
		}
	}
	return cells
}

// generateID creates a random 8-character ID
func generateID() string {
	bytes := make([]byte, 4)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}

// GetFilePath returns the resolved file path for file watching
func (s *MarkdownSource) GetFilePath() string {
	return s.resolvePath()
}

// GetAnchor returns the section anchor
func (s *MarkdownSource) GetAnchor() string {
	return s.anchor
}

// WriteItem adds, updates, or deletes an item in the markdown source
// This will be implemented in Phase 2
func (s *MarkdownSource) WriteItem(ctx context.Context, action string, data map[string]interface{}) error {
	if s.readonly {
		return fmt.Errorf("markdown source %q is read-only", s.name)
	}
	// TODO: Implement in Phase 2
	return fmt.Errorf("write operations not yet implemented")
}

// MarkdownSourceParser provides utilities for parsing markdown data sections
// Used by the code generator to inline parsing logic
type MarkdownSourceParser struct{}

// ParseContent parses markdown content and returns structured data
func (p *MarkdownSourceParser) ParseContent(content, anchor string) ([]map[string]interface{}, error) {
	src := &MarkdownSource{anchor: anchor}
	return src.parseSection(content)
}

// ScanMarkdownForIDs scans a markdown file and returns all item IDs found
func ScanMarkdownForIDs(content string) []string {
	var ids []string
	idPattern := regexp.MustCompile(`<!--\s*id:(\w+)\s*-->`)
	matches := idPattern.FindAllStringSubmatch(content, -1)
	for _, match := range matches {
		ids = append(ids, match[1])
	}
	return ids
}

// EnsureUniqueIDs checks for duplicate IDs and generates new ones if needed
func EnsureUniqueIDs(content string) (string, bool) {
	seen := make(map[string]bool)
	modified := false

	idPattern := regexp.MustCompile(`<!--\s*id:(\w+)\s*-->`)

	result := idPattern.ReplaceAllStringFunc(content, func(match string) string {
		submatches := idPattern.FindStringSubmatch(match)
		if len(submatches) < 2 {
			return match
		}
		id := submatches[1]
		if seen[id] {
			// Duplicate found, generate new ID
			newID := generateID()
			modified = true
			return "<!-- id:" + newID + " -->"
		}
		seen[id] = true
		return match
	})

	return result, modified
}

// AddIDsToItems adds ID comments to items that don't have them
func AddIDsToItems(content string) (string, bool) {
	modified := false
	var result strings.Builder

	scanner := bufio.NewScanner(strings.NewReader(content))

	// Patterns for items that should have IDs
	taskPattern := regexp.MustCompile(`^(\s*-\s+\[[ xX]\]\s+.+?)(\s*)$`)
	bulletPattern := regexp.MustCompile(`^(\s*-\s+[^\[].+?)(\s*)$`)
	tableRowPattern := regexp.MustCompile(`^(\s*\|.+\|)(\s*)$`)
	separatorPattern := regexp.MustCompile(`^\s*\|[\s\-:|]+\|`)
	hasIDPattern := regexp.MustCompile(`<!--\s*id:\w+\s*-->`)

	isFirstTableRow := true

	for scanner.Scan() {
		line := scanner.Text()

		// Skip if already has ID
		if hasIDPattern.MatchString(line) {
			result.WriteString(line + "\n")
			continue
		}

		// Check for task list
		if matches := taskPattern.FindStringSubmatch(line); matches != nil {
			id := generateID()
			result.WriteString(matches[1] + " <!-- id:" + id + " -->" + matches[2] + "\n")
			modified = true
			continue
		}

		// Check for bullet list (but not task list)
		if matches := bulletPattern.FindStringSubmatch(line); matches != nil {
			// Make sure it's not a task list item
			if !strings.Contains(matches[1], "[ ]") && !strings.Contains(matches[1], "[x]") && !strings.Contains(matches[1], "[X]") {
				id := generateID()
				result.WriteString(matches[1] + " <!-- id:" + id + " -->" + matches[2] + "\n")
				modified = true
				continue
			}
		}

		// Check for table row (skip header and separator)
		if matches := tableRowPattern.FindStringSubmatch(line); matches != nil {
			if separatorPattern.MatchString(line) {
				// Separator row
				result.WriteString(line + "\n")
				isFirstTableRow = false
				continue
			}
			if isFirstTableRow {
				// Header row
				result.WriteString(line + "\n")
				continue
			}
			// Data row
			id := generateID()
			result.WriteString(matches[1] + " <!-- id:" + id + " -->" + matches[2] + "\n")
			modified = true
			continue
		}

		result.WriteString(line + "\n")
	}

	return strings.TrimSuffix(result.String(), "\n"), modified
}
