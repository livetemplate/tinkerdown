package tinkerdown

import (
	"bytes"
	"fmt"
	"html"
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/livetemplate/tinkerdown/internal/slug"
)

// tableSection represents a detected markdown table under a heading.
type tableSection struct {
	heading     string   // raw heading text
	anchor      string   // slugified anchor
	level       int      // heading level (number of # chars)
	columns     []string // column names extracted from table header row
	headingLine int      // line number of the heading (0-indexed)
	startLine   int      // first line of table (header row, 0-indexed)
	endLine     int      // line past the last table line (exclusive, 0-indexed)
}

// Pre-compiled patterns for table detection
var (
	// Matches a markdown table row (header or data): | Col1 | Col2 | Col3 |
	tableRowPattern = regexp.MustCompile(`^\s*\|(.+)\|\s*$`)
	// Matches the separator row: |---|---|---|
	tableSepPattern = regexp.MustCompile(`^\s*\|[\s:]*-+[\s:|-]*\|\s*$`)
)

// detectTableSections scans markdown content (after frontmatter) and finds
// headings followed by markdown tables. Returns the detected sections.
//
// A valid table section is:
//   - A heading (## Foo)
//   - Optionally separated by blank lines
//   - Followed by a markdown table (header row + separator row + optional data rows)
//
// Sections where the table is preceded by non-blank, non-table content are skipped.
func detectTableSections(content []byte) []tableSection {
	// Quick check: if no pipe character, no tables possible
	if bytes.IndexByte(content, '|') < 0 {
		return nil
	}

	lines := strings.Split(string(content), "\n")
	var sections []tableSection

	for i := 0; i < len(lines); i++ {
		// Look for headings
		matches := headingPattern.FindStringSubmatch(lines[i])
		if matches == nil {
			continue
		}

		level := len(matches[1])
		headingText := strings.TrimSpace(matches[2])

		// Strip explicit anchor syntax {#anchor} if present
		anchor := ""
		if anchorMatch := explicitAnchorPattern.FindStringSubmatch(headingText); anchorMatch != nil {
			anchor = anchorMatch[1]
			headingText = strings.TrimSpace(explicitAnchorPattern.ReplaceAllString(headingText, ""))
		} else {
			anchor = slug.Heading(headingText)
		}

		if anchor == "" {
			continue
		}

		headingLine := i

		// Skip blank lines after heading
		j := i + 1
		for j < len(lines) && strings.TrimSpace(lines[j]) == "" {
			j++
		}

		if j >= len(lines) {
			continue
		}

		// Check for table header row
		headerMatch := tableRowPattern.FindStringSubmatch(lines[j])
		if headerMatch == nil {
			continue
		}

		// Parse column names from header
		columns := parseTableColumns(headerMatch[1])
		if len(columns) == 0 {
			continue
		}

		tableStart := j
		j++

		// Must have separator row
		if j >= len(lines) || !tableSepPattern.MatchString(lines[j]) {
			continue
		}
		j++

		// Consume data rows
		for j < len(lines) && tableRowPattern.MatchString(lines[j]) {
			j++
		}

		sections = append(sections, tableSection{
			heading:     headingText,
			anchor:      anchor,
			level:       level,
			columns:     columns,
			headingLine: headingLine,
			startLine:   tableStart,
			endLine:     j,
		})
	}

	return sections
}

// parseTableColumns extracts column names from a table header row content.
// Input: " Col1 | Col2 | Col3 " (the part between outer pipes)
// Output: ["Col1", "Col2", "Col3"]
// Empty column names are included (they may be action columns).
func parseTableColumns(headerContent string) []string {
	parts := strings.Split(headerContent, "|")
	var columns []string
	for _, p := range parts {
		columns = append(columns, strings.TrimSpace(p))
	}
	return columns
}

// matchResult represents a successful source match for a table section.
type matchResult struct {
	section    tableSection
	sourceName string
	writable   bool
}

// matchTablesToSources matches detected table sections to declared sources.
//
// Matching logic (smart matching with explicit override):
//  1. Exact: slug(heading) == sourceName → match
//  2. Containment: heading contains source name as a word → match
//  3. No match → skip
//  4. Override: AutoBind == false on a source opts it out
//
// If multiple sources match a heading via containment, skip with warning.
// Exact matches always take priority over containment matches.
func matchTablesToSources(sections []tableSection, sources map[string]SourceConfig) ([]matchResult, []string) {
	var results []matchResult
	var warnings []string

	// Build a set of source names eligible for matching (excluding auto_bind: false)
	eligible := make(map[string]SourceConfig)
	for name, src := range sources {
		if src.AutoBind != nil && !*src.AutoBind {
			continue
		}
		eligible[name] = src
	}

	for _, sec := range sections {
		headingSlug := sec.anchor

		// Phase 1: exact match
		if src, ok := eligible[headingSlug]; ok {
			results = append(results, matchResult{
				section:    sec,
				sourceName: headingSlug,
				writable:   isWritable(src),
			})
			continue
		}

		// Phase 2: word-boundary containment
		// Check if any source name appears as a word in the heading
		headingLower := strings.ToLower(sec.heading)
		var containmentMatches []string
		for name := range eligible {
			if containsWord(headingLower, name) {
				containmentMatches = append(containmentMatches, name)
			}
		}

		if len(containmentMatches) == 1 {
			name := containmentMatches[0]
			results = append(results, matchResult{
				section:    sec,
				sourceName: name,
				writable:   isWritable(eligible[name]),
			})
		} else if len(containmentMatches) > 1 {
			warnings = append(warnings, fmt.Sprintf(
				"heading %q matches multiple sources (%s) — use explicit lvt-source binding (Tier 2)",
				sec.heading, strings.Join(containmentMatches, ", ")))
		}
		// len == 0: no match, skip silently
	}

	return results, warnings
}

// containsWord checks if text contains word as a whole word (at word boundaries).
// Both text and word should be lowercase.
// Iterates over all occurrences so "myexpenses expenses" correctly matches "expenses".
func containsWord(text, word string) bool {
	if word == "" {
		return false
	}

	start := 0
	for {
		idx := strings.Index(text[start:], word)
		if idx < 0 {
			return false
		}
		idx += start // absolute index

		// Check left boundary
		if idx > 0 && isAlphanumeric(text[idx-1]) {
			start = idx + len(word)
			if start >= len(text) {
				return false
			}
			continue
		}

		// Check right boundary
		end := idx + len(word)
		if end < len(text) && isAlphanumeric(text[end]) {
			start = idx + len(word)
			if start >= len(text) {
				return false
			}
			continue
		}

		return true
	}
}

// isAlphanumeric returns true if b is a letter or digit.
func isAlphanumeric(b byte) bool {
	return (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z') || (b >= '0' && b <= '9')
}

// isWritable returns true if the source config indicates the source supports writes.
// Sources are read-only by default (Readonly==nil means read-only).
func isWritable(src SourceConfig) bool {
	if src.Readonly == nil {
		return false
	}
	return !*src.Readonly
}

// preprocessAutoTables detects markdown tables under headings, matches them to
// declared sources by heading name, and replaces them with lvt code blocks.
//
// This is the Tier 1 equivalent of preprocessAutoTasks (Tier 0):
//   - Tier 0: task lists → interactive checkboxes + add form
//   - Tier 1: markdown tables + yaml sources → interactive data tables + CRUD
//
// The function does NOT modify the original file — it operates on in-memory content.
func preprocessAutoTables(content []byte, sources map[string]SourceConfig) ([]byte, []string) {
	if len(sources) == 0 {
		return content, nil
	}

	// Skip frontmatter during detection
	bodyOffset := 0
	if loc := frontmatterPattern.FindIndex(content); loc != nil {
		bodyOffset = loc[1]
	}

	body := content[bodyOffset:]
	sections := detectTableSections(body)
	if len(sections) == 0 {
		return content, nil
	}

	matches, warnings := matchTablesToSources(sections, sources)
	if len(matches) == 0 {
		return content, warnings
	}

	lines := strings.Split(string(content), "\n")

	// Calculate frontmatter line offset
	fmLineCount := 0
	if bodyOffset > 0 {
		fmLineCount = strings.Count(string(content[:bodyOffset]), "\n")
	}

	// Process matches in reverse order to preserve line numbers
	for i := len(matches) - 1; i >= 0; i-- {
		m := matches[i]

		// Filter out empty column names (used for action columns in static tables)
		var displayColumns []string
		for _, col := range m.section.columns {
			if col != "" {
				displayColumns = append(displayColumns, col)
			}
		}

		lvtBlock := generateAutoTableLvtBlock(m.sourceName, displayColumns, m.writable)

		// Build replacement: heading + lvt code block
		headingIdx := m.section.headingLine + fmLineCount
		endIdx := m.section.endLine + fmLineCount

		var replacement []string
		replacement = append(replacement, lines[headingIdx]) // keep heading
		replacement = append(replacement, "")
		replacement = append(replacement, "```lvt")
		replacement = append(replacement, lvtBlock)
		replacement = append(replacement, "```")

		newLines := make([]string, 0, len(lines)-(endIdx-headingIdx)+len(replacement))
		newLines = append(newLines, lines[:headingIdx]...)
		newLines = append(newLines, replacement...)
		newLines = append(newLines, lines[endIdx:]...)
		lines = newLines
	}

	return []byte(strings.Join(lines, "\n")), warnings
}

// generateAutoTableLvtBlock generates the lvt template HTML for an auto-table section.
//
// For writable sources: table with edit/delete per row + add form.
// For read-only sources: table with refresh button.
func generateAutoTableLvtBlock(sourceName string, columns []string, writable bool) string {
	if writable {
		return generateWritableTableBlock(sourceName, columns)
	}
	return generateReadonlyTableBlock(sourceName, columns)
}

// generateReadonlyTableBlock generates an auto-rendered read-only table.
func generateReadonlyTableBlock(sourceName string, columns []string) string {
	var b strings.Builder

	b.WriteString(fmt.Sprintf(`<div lvt-source="%s">`, sourceName))
	b.WriteString("\n")

	// Refresh button
	b.WriteString(`<div style="margin-bottom: 8px;">`)
	b.WriteString("\n")
	b.WriteString(`  <button lvt-click="Refresh" style="padding: 6px 12px; background: #6c757d; color: white; border: none; border-radius: 4px; cursor: pointer;">Refresh</button>`)
	b.WriteString("\n")
	b.WriteString(`</div>`)
	b.WriteString("\n")

	// Error display
	b.WriteString(`{{if .Error}}<div style="color: #dc3545; padding: 8px; margin-bottom: 8px; border: 1px solid #dc3545; border-radius: 4px;">{{.Error}}</div>{{end}}`)
	b.WriteString("\n")

	// Table
	b.WriteString(`<table>`)
	b.WriteString("\n")

	// Header
	b.WriteString(`  <thead><tr>`)
	for _, col := range columns {
		b.WriteString(fmt.Sprintf(`<th>%s</th>`, html.EscapeString(col)))
	}
	b.WriteString(`</tr></thead>`)
	b.WriteString("\n")

	// Body
	b.WriteString(`  <tbody>`)
	b.WriteString("\n")
	b.WriteString(`  {{range .Data}}`)
	b.WriteString("\n")
	b.WriteString(`    <tr>`)
	for _, col := range columns {
		fieldName := toFieldName(col)
		b.WriteString(fmt.Sprintf(`<td>{{.%s}}</td>`, fieldName))
	}
	b.WriteString(`</tr>`)
	b.WriteString("\n")
	b.WriteString(`  {{end}}`)
	b.WriteString("\n")
	b.WriteString(`  </tbody>`)
	b.WriteString("\n")
	b.WriteString(`</table>`)
	b.WriteString("\n")

	b.WriteString(`</div>`)

	return b.String()
}

// generateWritableTableBlock generates an auto-rendered writable table with full CRUD.
// Includes inline editing: each row has Edit/Delete buttons. Clicking Edit replaces
// the row's cells with input fields and shows Save/Cancel buttons.
func generateWritableTableBlock(sourceName string, columns []string) string {
	var b strings.Builder

	b.WriteString(fmt.Sprintf(`<div lvt-source="%s">`, sourceName))
	b.WriteString("\n")

	// Error display
	b.WriteString(`{{if .Error}}<div style="color: #dc3545; padding: 8px; margin-bottom: 8px; border: 1px solid #dc3545; border-radius: 4px;">{{.Error}}</div>{{end}}`)
	b.WriteString("\n")

	// Table
	b.WriteString(`<table>`)
	b.WriteString("\n")

	// Header
	b.WriteString(`  <thead><tr>`)
	for _, col := range columns {
		b.WriteString(fmt.Sprintf(`<th>%s</th>`, html.EscapeString(col)))
	}
	b.WriteString(`<th></th>`) // actions column
	b.WriteString(`</tr></thead>`)
	b.WriteString("\n")

	// Body with inline edit support
	b.WriteString(`  <tbody>`)
	b.WriteString("\n")
	b.WriteString(`  {{range .Data}}`)
	b.WriteString("\n")

	// Edit mode: show inputs when this row's ID matches EditingID
	b.WriteString(`  {{if eq (printf "%v" .Id) $.EditingID}}`)
	b.WriteString("\n")
	b.WriteString(`    <tr>`)
	b.WriteString("\n")
	b.WriteString(`      <form lvt-submit="Update">`)
	b.WriteString("\n")
	b.WriteString(`        <input type="hidden" name="id" value="{{.Id}}">`)
	b.WriteString("\n")
	for _, col := range columns {
		fieldName := toFieldName(col)
		inputName := toInputName(col)
		b.WriteString(fmt.Sprintf(`        <td><input type="text" name="%s" value="{{.%s}}"`, inputName, fieldName))
		b.WriteString("\n")
		b.WriteString(`          style="width: 100%; padding: 4px 6px; border: 1px solid #007bff; border-radius: 4px; box-sizing: border-box;"></td>`)
		b.WriteString("\n")
	}
	// Save + Cancel buttons
	b.WriteString(`        <td style="white-space: nowrap;">`)
	b.WriteString("\n")
	b.WriteString(`          <button type="submit"`)
	b.WriteString("\n")
	b.WriteString(`            style="padding: 4px 8px; background: #28a745; color: white; border: none; border-radius: 4px; cursor: pointer; font-size: 0.85em; margin-right: 4px;">`)
	b.WriteString("\n")
	b.WriteString(`            Save</button>`)
	b.WriteString("\n")
	b.WriteString(`          <button type="button" lvt-click="CancelEdit"`)
	b.WriteString("\n")
	b.WriteString(`            style="padding: 4px 8px; background: #6c757d; color: white; border: none; border-radius: 4px; cursor: pointer; font-size: 0.85em;">`)
	b.WriteString("\n")
	b.WriteString(`            Cancel</button>`)
	b.WriteString("\n")
	b.WriteString(`        </td>`)
	b.WriteString("\n")
	b.WriteString(`      </form>`)
	b.WriteString("\n")
	b.WriteString(`    </tr>`)
	b.WriteString("\n")

	// Display mode: show text with Edit/Delete buttons
	b.WriteString(`  {{else}}`)
	b.WriteString("\n")
	b.WriteString(`    <tr>`)
	b.WriteString("\n")
	for _, col := range columns {
		fieldName := toFieldName(col)
		b.WriteString(fmt.Sprintf(`      <td>{{.%s}}</td>`, fieldName))
		b.WriteString("\n")
	}
	// Edit + Delete buttons
	b.WriteString(`      <td style="white-space: nowrap;">`)
	b.WriteString("\n")
	b.WriteString(`        <button lvt-click="Edit" lvt-data-id="{{.Id}}"`)
	b.WriteString("\n")
	b.WriteString(`          style="padding: 4px 8px; background: #007bff; color: white; border: none; border-radius: 4px; cursor: pointer; font-size: 0.85em; margin-right: 4px;">`)
	b.WriteString("\n")
	b.WriteString(`          Edit</button>`)
	b.WriteString("\n")
	b.WriteString(`        <button lvt-click="Delete" lvt-data-id="{{.Id}}" lvt-confirm="Delete this item?"`)
	b.WriteString("\n")
	b.WriteString(`          style="padding: 4px 8px; background: #dc3545; color: white; border: none; border-radius: 4px; cursor: pointer; font-size: 0.85em;">`)
	b.WriteString("\n")
	b.WriteString(`          Delete</button>`)
	b.WriteString("\n")
	b.WriteString(`      </td>`)
	b.WriteString("\n")
	b.WriteString(`    </tr>`)
	b.WriteString("\n")
	b.WriteString(`  {{end}}`)
	b.WriteString("\n")

	b.WriteString(`  {{end}}`)
	b.WriteString("\n")
	b.WriteString(`  </tbody>`)
	b.WriteString("\n")
	b.WriteString(`</table>`)
	b.WriteString("\n")

	// Add form
	b.WriteString("\n")
	b.WriteString(`<form lvt-submit="Add" lvt-reset-on:success style="display: flex; gap: 8px; align-items: flex-end; margin-top: 12px; flex-wrap: wrap;">`)
	b.WriteString("\n")
	for _, col := range columns {
		inputName := toInputName(col)
		b.WriteString(`  <div style="display: flex; flex-direction: column; gap: 2px;">`)
		b.WriteString("\n")
		b.WriteString(fmt.Sprintf(`    <label style="font-size: 0.8em; color: #666;">%s</label>`, html.EscapeString(col)))
		b.WriteString("\n")
		b.WriteString(fmt.Sprintf(`    <input type="text" name="%s" placeholder="%s..." required`, inputName, html.EscapeString(col)))
		b.WriteString("\n")
		b.WriteString(`      style="padding: 6px 8px; border: 1px solid #ccc; border-radius: 4px;">`)
		b.WriteString("\n")
		b.WriteString(`  </div>`)
		b.WriteString("\n")
	}
	b.WriteString(`  <button type="submit"`)
	b.WriteString("\n")
	b.WriteString(`    style="padding: 8px 16px; background: #007bff; color: white; border: none; border-radius: 4px; cursor: pointer;">`)
	b.WriteString("\n")
	b.WriteString(`    Add</button>`)
	b.WriteString("\n")
	b.WriteString(`</form>`)
	b.WriteString("\n")

	b.WriteString(`</div>`)

	return b.String()
}

// toFieldName converts a column header name to a Go template field accessor.
// "Full Name" → "FullName", "email" → "Email", "first_name" → "FirstName"
func toFieldName(column string) string {
	column = strings.TrimSpace(column)
	if column == "" {
		return column
	}

	// Split on spaces, underscores, and hyphens
	parts := strings.FieldsFunc(column, func(r rune) bool {
		return r == ' ' || r == '_' || r == '-'
	})

	var result strings.Builder
	for _, part := range parts {
		if part == "" {
			continue
		}
		// Capitalize first rune of each part (PascalCase), safe for multi-byte.
		// Only emit letters and digits to prevent template injection.
		firstRune, size := utf8.DecodeRuneInString(part)
		result.WriteString(strings.ToUpper(string(firstRune)))
		for _, r := range part[size:] {
			if r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' {
				result.WriteRune(r)
			}
		}
	}

	return result.String()
}

// toInputName converts a column header to a safe HTML input name attribute.
// Normalizes to lowercase, replaces non-alphanumeric chars with underscores,
// collapses repeats, and trims leading/trailing underscores.
func toInputName(column string) string {
	column = strings.TrimSpace(column)
	column = strings.ToLower(column)

	var result strings.Builder
	prevUnderscore := false
	for _, r := range column {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			result.WriteRune(r)
			prevUnderscore = false
		} else if !prevUnderscore {
			result.WriteRune('_')
			prevUnderscore = true
		}
	}

	return strings.Trim(result.String(), "_")
}
