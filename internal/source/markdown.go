package source

import (
	"bufio"
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"hash/fnv"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"
)

// ConflictError is returned when a write operation detects concurrent modification
type ConflictError struct {
	OriginalPath string
	ConflictPath string
	Message      string
}

func (e *ConflictError) Error() string {
	return e.Message
}

// MarkdownSource reads data from markdown sections (task lists, bullet lists, tables)
type MarkdownSource struct {
	name        string
	filePath    string // empty means current file (handled by caller)
	anchor      string // e.g., "#todos"
	readonly    bool
	siteDir     string
	currentFile string // the markdown file being served (for same-file anchors)

	// Concurrency control
	mu       sync.RWMutex
	lastMtime time.Time // mtime of file when last read
}

// NewMarkdownSource creates a new markdown source
func NewMarkdownSource(name, file, anchor, siteDir, currentFile string, readonly bool) (*MarkdownSource, error) {
	if file == "" {
		return nil, fmt.Errorf("markdown source %q: 'file' field is required - data must be in a separate file from the UI", name)
	}

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

	// Get file info for mtime tracking
	info, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("markdown source %q: failed to stat file: %w", s.name, err)
	}

	content, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("markdown source %q: failed to read file: %w", s.name, err)
	}

	// Store mtime for conflict detection on writes
	s.mu.Lock()
	s.lastMtime = info.ModTime()
	s.mu.Unlock()

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
	anchorName := strings.TrimPrefix(s.anchor, "#")

	// Try to find the section header - explicit {#anchor} first, then text-based
	matches := s.findSectionHeader(content, anchorName)
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

// findSectionHeader finds a section header by anchor name.
// Tries explicit {#anchor} syntax first, falls back to matching heading text (slugified).
// Headings with explicit anchors are excluded from text-based matching.
func (s *MarkdownSource) findSectionHeader(content, anchorName string) []int {
	// Pattern 1: Explicit {#anchor} syntax - takes precedence
	explicitPattern := regexp.MustCompile(`(?m)^(#{1,6})\s+(.+?)\s*\{#` + regexp.QuoteMeta(anchorName) + `\}\s*$`)
	if matches := explicitPattern.FindStringSubmatchIndex(content); matches != nil {
		return matches
	}

	// Pattern 2: Match heading text (slugified) - fallback
	// Only match headings WITHOUT explicit anchors
	headingPattern := regexp.MustCompile(`(?m)^(#{1,6})\s+(.+?)\s*$`)
	explicitAnchorPattern := regexp.MustCompile(`\{#[^}]+\}\s*$`)
	allMatches := headingPattern.FindAllStringSubmatchIndex(content, -1)
	for _, match := range allMatches {
		headingText := content[match[4]:match[5]]
		// Skip headings that have explicit anchors (they should only match their explicit anchor)
		if explicitAnchorPattern.MatchString(headingText) {
			continue
		}
		if slugify(headingText) == anchorName {
			return match
		}
	}

	return nil
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

		// Generate content-based ID if missing (deterministic from text)
		if id == "" {
			id = generateContentID(text)
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

		// Generate content-based ID if missing (deterministic from text)
		if id == "" {
			id = generateContentID(text)
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

		// Generate content-based ID if missing (deterministic from row content)
		if id == "" {
			// Join all cells to create a unique hash for this row
			id = generateContentID(strings.Join(cells, "|"))
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

// generateContentID creates a deterministic ID from item content using FNV hash.
// Same text always produces the same ID, making IDs stable across file syncs.
func generateContentID(text string) string {
	h := fnv.New32a()
	h.Write([]byte(text))
	return fmt.Sprintf("%08x", h.Sum32())
}

// slugify converts heading text to an anchor-compatible slug (GitHub-style).
// "My Task List" -> "my-task-list"
func slugify(text string) string {
	text = strings.ToLower(text)
	text = strings.ReplaceAll(text, " ", "-")
	// Remove non-alphanumeric chars except hyphens
	re := regexp.MustCompile(`[^a-z0-9-]`)
	return re.ReplaceAllString(text, "")
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
// Supported actions: add, toggle, delete, update
// Returns ConflictError if the file was modified externally since last read
func (s *MarkdownSource) WriteItem(ctx context.Context, action string, data map[string]interface{}) error {
	if s.readonly {
		return fmt.Errorf("markdown source %q is read-only", s.name)
	}

	path := s.resolvePath()

	// Check current mtime before reading
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("failed to stat file: %w", err)
	}
	currentMtime := info.ModTime()

	// Check for conflict (file modified since last read)
	s.mu.RLock()
	lastMtime := s.lastMtime
	s.mu.RUnlock()

	if !lastMtime.IsZero() && !currentMtime.Equal(lastMtime) {
		// File was modified externally - create conflict copy
		conflictPath, err := s.createConflictCopy(path)
		if err != nil {
			return fmt.Errorf("failed to create conflict copy: %w", err)
		}
		return &ConflictError{
			OriginalPath: path,
			ConflictPath: conflictPath,
			Message:      fmt.Sprintf("file was modified externally; your changes saved to %s", conflictPath),
		}
	}

	// Read current content
	contentBytes, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}
	content := string(contentBytes)

	// Find section boundaries
	sectionStart, sectionEnd, headerLevel, err := s.findSectionBoundaries(content)
	if err != nil {
		return err
	}

	sectionContent := content[sectionStart:sectionEnd]

	// Detect format
	format := s.detectFormat(sectionContent)

	// Perform action
	var newSectionContent string
	switch action {
	case "add":
		newSectionContent, err = s.addItem(sectionContent, format, data)
	case "toggle":
		newSectionContent, err = s.toggleItem(sectionContent, format, data)
	case "delete":
		newSectionContent, err = s.deleteItem(sectionContent, format, data)
	case "update":
		newSectionContent, err = s.updateItem(sectionContent, format, data)
	default:
		return fmt.Errorf("unknown action: %s", action)
	}

	if err != nil {
		return err
	}

	// Reconstruct file content
	newContent := content[:sectionStart] + newSectionContent + content[sectionEnd:]

	// Write back to file
	if err := os.WriteFile(path, []byte(newContent), 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	// Update stored mtime after successful write
	if newInfo, err := os.Stat(path); err == nil {
		s.mu.Lock()
		s.lastMtime = newInfo.ModTime()
		s.mu.Unlock()
	}

	_ = headerLevel // Used in section boundary detection
	return nil
}

// createConflictCopy creates a conflict copy of the current pending changes
// Returns the path to the conflict file
func (s *MarkdownSource) createConflictCopy(originalPath string) (string, error) {
	// Read the current file content (what the user was trying to modify)
	content, err := os.ReadFile(originalPath)
	if err != nil {
		return "", fmt.Errorf("failed to read original file: %w", err)
	}

	// Generate conflict filename: file.conflict-{timestamp}.md
	ext := filepath.Ext(originalPath)
	base := strings.TrimSuffix(originalPath, ext)
	timestamp := time.Now().Format("20060102-150405")
	conflictPath := fmt.Sprintf("%s.conflict-%s%s", base, timestamp, ext)

	// Write conflict copy
	if err := os.WriteFile(conflictPath, content, 0644); err != nil {
		return "", fmt.Errorf("failed to write conflict file: %w", err)
	}

	return conflictPath, nil
}

// findSectionBoundaries finds where the section starts and ends
func (s *MarkdownSource) findSectionBoundaries(content string) (start, end, headerLevel int, err error) {
	anchorName := strings.TrimPrefix(s.anchor, "#")

	// Use the same header finding logic as parseSection
	matches := s.findSectionHeader(content, anchorName)
	if matches == nil {
		return 0, 0, 0, fmt.Errorf("section %q not found", s.anchor)
	}

	start = matches[1] // End of header line
	headerLevel = len(content[matches[2]:matches[3]])

	end = len(content)
	nextHeaderPattern := regexp.MustCompile(`(?m)^#{1,` + fmt.Sprintf("%d", headerLevel) + `}\s+`)
	if loc := nextHeaderPattern.FindStringIndex(content[start:]); loc != nil {
		end = start + loc[0]
	}

	return start, end, headerLevel, nil
}

// detectFormat returns the format type: "task", "bullet", or "table"
func (s *MarkdownSource) detectFormat(content string) string {
	lines := strings.Split(content, "\n")

	taskListPattern := regexp.MustCompile(`^\s*-\s+\[([ xX])\]\s+`)
	bulletListPattern := regexp.MustCompile(`^\s*-\s+[^\[]`)
	tablePattern := regexp.MustCompile(`^\s*\|.+\|`)

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if taskListPattern.MatchString(line) {
			return "task"
		}
		if tablePattern.MatchString(line) {
			return "table"
		}
		if bulletListPattern.MatchString(line) {
			return "bullet"
		}
	}
	return "unknown"
}

// addItem adds a new item to the section
func (s *MarkdownSource) addItem(sectionContent, format string, data map[string]interface{}) (string, error) {
	id := generateID()

	var newLine string
	switch format {
	case "task":
		text, _ := data["text"].(string)
		done, _ := data["done"].(bool)
		checkbox := "[ ]"
		if done {
			checkbox = "[x]"
		}
		newLine = fmt.Sprintf("- %s %s <!-- id:%s -->", checkbox, text, id)

	case "bullet":
		text, _ := data["text"].(string)
		newLine = fmt.Sprintf("- %s <!-- id:%s -->", text, id)

	case "table":
		// For tables, we need to find the headers first
		headers := s.extractTableHeaders(sectionContent)
		if len(headers) == 0 {
			return "", fmt.Errorf("cannot add to table: no headers found")
		}
		var cells []string
		for _, h := range headers {
			val := ""
			if v, ok := data[h]; ok {
				val = fmt.Sprintf("%v", v)
			}
			cells = append(cells, val)
		}
		newLine = "| " + strings.Join(cells, " | ") + " | <!-- id:" + id + " -->"

	default:
		return "", fmt.Errorf("cannot add item: unknown format")
	}

	// Append to the end of section (before trailing newlines)
	trimmed := strings.TrimRight(sectionContent, "\n")
	return trimmed + "\n" + newLine + "\n", nil
}

// toggleItem toggles the done state of a task list item
func (s *MarkdownSource) toggleItem(sectionContent, format string, data map[string]interface{}) (string, error) {
	if format != "task" {
		return "", fmt.Errorf("toggle action only supported for task lists")
	}

	id, _ := data["id"].(string)
	if id == "" {
		return "", fmt.Errorf("toggle action requires 'id' field")
	}

	// Find the line with this ID and toggle it
	lines := strings.Split(sectionContent, "\n")
	taskPattern := regexp.MustCompile(`^\s*-\s+\[([ xX])\]\s+(.+?)(?:\s*<!--\s*id:(\w+)\s*-->)?$`)

	found := false
	for i, line := range lines {
		// First try explicit ID comment
		if strings.Contains(line, "<!-- id:"+id+" -->") {
			lines[i] = s.toggleCheckbox(line)
			found = true
			break
		}

		// Fall back to content-based ID matching
		matches := taskPattern.FindStringSubmatch(line)
		if matches != nil {
			explicitID := matches[3]
			if explicitID == "" {
				// No explicit ID - check if content-based ID matches
				text := strings.TrimSpace(matches[2])
				if generateContentID(text) == id {
					lines[i] = s.toggleCheckbox(line)
					found = true
					break
				}
			}
		}
	}

	if !found {
		return "", fmt.Errorf("item with id %q not found", id)
	}

	return strings.Join(lines, "\n"), nil
}

// toggleCheckbox toggles the checkbox state in a task line
func (s *MarkdownSource) toggleCheckbox(line string) string {
	if strings.Contains(line, "[ ]") {
		return strings.Replace(line, "[ ]", "[x]", 1)
	} else if strings.Contains(line, "[x]") {
		return strings.Replace(line, "[x]", "[ ]", 1)
	} else if strings.Contains(line, "[X]") {
		return strings.Replace(line, "[X]", "[ ]", 1)
	}
	return line
}

// deleteItem removes an item from the section
func (s *MarkdownSource) deleteItem(sectionContent, format string, data map[string]interface{}) (string, error) {
	id, _ := data["id"].(string)
	if id == "" {
		return "", fmt.Errorf("delete action requires 'id' field")
	}

	lines := strings.Split(sectionContent, "\n")
	var newLines []string
	found := false

	for _, line := range lines {
		// First try explicit ID comment
		if strings.Contains(line, "<!-- id:"+id+" -->") {
			found = true
			continue // Skip this line (delete it)
		}

		// Check for content-based ID match
		if !found {
			itemID := s.extractItemID(line, format)
			if itemID == id {
				found = true
				continue // Skip this line (delete it)
			}
		}

		newLines = append(newLines, line)
	}

	if !found {
		return "", fmt.Errorf("item with id %q not found", id)
	}

	return strings.Join(newLines, "\n"), nil
}

// extractItemID extracts the ID from a line, using content-based ID if no explicit ID
func (s *MarkdownSource) extractItemID(line, format string) string {
	switch format {
	case "task":
		taskPattern := regexp.MustCompile(`^\s*-\s+\[([ xX])\]\s+(.+?)(?:\s*<!--\s*id:(\w+)\s*-->)?$`)
		if matches := taskPattern.FindStringSubmatch(line); matches != nil {
			if matches[3] != "" {
				return matches[3] // Explicit ID
			}
			return generateContentID(strings.TrimSpace(matches[2])) // Content-based ID
		}
	case "bullet":
		bulletPattern := regexp.MustCompile(`^\s*-\s+(.+?)(?:\s*<!--\s*id:(\w+)\s*-->)?$`)
		if matches := bulletPattern.FindStringSubmatch(line); matches != nil {
			text := strings.TrimSpace(matches[1])
			// Skip task list items
			if strings.HasPrefix(text, "[ ]") || strings.HasPrefix(text, "[x]") || strings.HasPrefix(text, "[X]") {
				return ""
			}
			if matches[2] != "" {
				return matches[2] // Explicit ID
			}
			return generateContentID(text) // Content-based ID
		}
	case "table":
		tableRowPattern := regexp.MustCompile(`^\s*\|(.+)\|(?:\s*<!--\s*id:(\w+)\s*-->)?`)
		if matches := tableRowPattern.FindStringSubmatch(line); matches != nil {
			if matches[2] != "" {
				return matches[2] // Explicit ID
			}
			cells := s.parseTableCells(matches[1])
			return generateContentID(strings.Join(cells, "|")) // Content-based ID
		}
	}
	return ""
}

// updateItem updates fields of an existing item
func (s *MarkdownSource) updateItem(sectionContent, format string, data map[string]interface{}) (string, error) {
	id, _ := data["id"].(string)
	if id == "" {
		return "", fmt.Errorf("update action requires 'id' field")
	}

	lines := strings.Split(sectionContent, "\n")
	found := false

	for i, line := range lines {
		// Check for explicit ID or content-based ID match
		hasExplicitID := strings.Contains(line, "<!-- id:"+id+" -->")
		matchesByContent := !hasExplicitID && s.extractItemID(line, format) == id

		if !hasExplicitID && !matchesByContent {
			continue
		}
		found = true

		switch format {
		case "task":
			// Update text and/or done state
			if text, ok := data["text"].(string); ok {
				if hasExplicitID {
					// Replace text between checkbox and ID comment
					taskPattern := regexp.MustCompile(`^(\s*-\s+\[[ xX]\]\s+)(.+?)(\s*<!--\s*id:\w+\s*-->)`)
					lines[i] = taskPattern.ReplaceAllString(line, "${1}"+text+"${3}")
				} else {
					// No ID comment - just replace the text
					taskPattern := regexp.MustCompile(`^(\s*-\s+\[[ xX]\]\s+)(.+)$`)
					lines[i] = taskPattern.ReplaceAllString(line, "${1}"+text)
				}
			}
			if done, ok := data["done"].(bool); ok {
				if done {
					lines[i] = strings.Replace(lines[i], "[ ]", "[x]", 1)
				} else {
					lines[i] = strings.Replace(lines[i], "[x]", "[ ]", 1)
					lines[i] = strings.Replace(lines[i], "[X]", "[ ]", 1)
				}
			}

		case "bullet":
			// Update text
			if text, ok := data["text"].(string); ok {
				if hasExplicitID {
					bulletPattern := regexp.MustCompile(`^(\s*-\s+)(.+?)(\s*<!--\s*id:\w+\s*-->)`)
					lines[i] = bulletPattern.ReplaceAllString(line, "${1}"+text+"${3}")
				} else {
					bulletPattern := regexp.MustCompile(`^(\s*-\s+)(.+)$`)
					lines[i] = bulletPattern.ReplaceAllString(line, "${1}"+text)
				}
			}

		case "table":
			// Update table cells by header name
			headers := s.extractTableHeaders(sectionContent)
			cells := s.parseTableCells(line)

			// Rebuild cells with updates
			for j, h := range headers {
				if val, ok := data[h]; ok && j < len(cells) {
					cells[j] = fmt.Sprintf("%v", val)
				}
			}

			// Reconstruct line (with or without ID comment)
			if hasExplicitID {
				lines[i] = "| " + strings.Join(cells, " | ") + " | <!-- id:" + id + " -->"
			} else {
				lines[i] = "| " + strings.Join(cells, " | ") + " |"
			}
		}
		break
	}

	if !found {
		return "", fmt.Errorf("item with id %q not found", id)
	}

	return strings.Join(lines, "\n"), nil
}

// extractTableHeaders extracts column headers from a table section
func (s *MarkdownSource) extractTableHeaders(content string) []string {
	lines := strings.Split(content, "\n")
	tableRowPattern := regexp.MustCompile(`^\s*\|(.+)\|`)

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		// Skip separator rows
		if regexp.MustCompile(`^\s*\|[\s\-:|]+\|`).MatchString(line) {
			continue
		}
		if matches := tableRowPattern.FindStringSubmatch(line); matches != nil {
			return s.parseTableCells(matches[1])
		}
	}
	return nil
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
