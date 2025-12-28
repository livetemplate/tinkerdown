package tinkerdown

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
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

	// Parse markdown with partial support
	baseDir := filepath.Dir(absPath)
	fm, codeBlocks, staticHTML, err := ParseMarkdownWithPartials(content, baseDir)
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
	page.Sidebar = fm.Sidebar // Page-level sidebar override
	page.Config = PageConfig{
		Persist:   fm.Persist,
		MultiStep: fm.Steps > 0,
		StepCount: fm.Steps,
	}

	// Apply frontmatter config options (sources, styling, blocks, features)
	page.Config.MergeFromFrontmatter(fm)

	// Build blocks (pass source file for error context)
	if err := page.buildBlocks(codeBlocks, absPath); err != nil {
		return nil, err // Already a ParseError from buildBlocks
	}

	return page, nil
}

// MergeFromFrontmatter applies frontmatter config options to PageConfig.
// Frontmatter values take precedence over any existing values.
func (pc *PageConfig) MergeFromFrontmatter(fm *Frontmatter) {
	// Sources - copy from frontmatter if present
	if fm.Sources != nil {
		if pc.Sources == nil {
			pc.Sources = make(map[string]SourceConfig)
		}
		for name, src := range fm.Sources {
			pc.Sources[name] = src
		}
	}

	// Styling - frontmatter takes precedence for non-zero values
	if fm.Styling != nil {
		if fm.Styling.Theme != "" {
			pc.Styling.Theme = fm.Styling.Theme
		}
		if fm.Styling.PrimaryColor != "" {
			pc.Styling.PrimaryColor = fm.Styling.PrimaryColor
		}
		if fm.Styling.Font != "" {
			pc.Styling.Font = fm.Styling.Font
		}
	}

	// Blocks - frontmatter takes precedence
	if fm.Blocks != nil {
		pc.Blocks = *fm.Blocks
	}

	// Features - frontmatter takes precedence
	if fm.Features != nil {
		pc.Features = *fm.Features
	}
}

// ParseString parses markdown content from a string and creates a Page.
// This is useful for the playground where content comes from user input.
func ParseString(content string) (*Page, error) {
	// Parse markdown (no partials support for string input)
	fm, codeBlocks, staticHTML, err := ParseMarkdownWithPartials([]byte(content), "")
	if err != nil {
		return nil, NewParseError("playground", 1, fmt.Sprintf("Failed to parse markdown: %v", err))
	}

	// Create page
	page := New("playground")
	page.Title = fm.Title
	if page.Title == "" {
		page.Title = "Playground"
	}
	page.Type = fm.Type
	page.StaticHTML = staticHTML
	page.SourceFile = "playground"
	page.Sidebar = fm.Sidebar // Page-level sidebar override
	page.Config = PageConfig{
		Persist:   fm.Persist,
		MultiStep: fm.Steps > 0,
		StepCount: fm.Steps,
	}

	// Apply frontmatter config options (sources, styling, blocks, features)
	page.Config.MergeFromFrontmatter(fm)

	// Build blocks
	if err := page.buildBlocks(codeBlocks, "playground"); err != nil {
		return nil, err
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
			// IMPORTANT: Extract lvt-source metadata from ORIGINAL content BEFORE template processing
			// because autoGenerateTableTemplate() strips the lvt-source attribute
			stateRef := cb.Metadata["state"]
			sourceName := getLvtSource(cb.Content)
			elementType := getLvtSourceElementType(cb.Content)
			columns := getTableColumns(cb.Content)
			actions := getTableActions(cb.Content)

			// Apply smart template generation for tables/selects with lvt-source
			processedContent := autoGenerateTableTemplate(cb.Content)
			processedContent = autoGenerateSelectTemplate(processedContent)

			if stateRef == "" && sourceName != "" {
				// Create auto-generated server block for lvt-source
				blockID := getBlockID(cb, i)
				autoID := "auto-persist-" + blockID

				// Build metadata for the generated server block
				metadata := map[string]string{
					"lvt-source":  sourceName,
					"lvt-element": elementType,
				}
				if elementType == "table" {
					// Pass column and action info for datatable generation
					if columns != "" {
						metadata["lvt-columns"] = columns
					}
					if actions != "" {
						metadata["lvt-actions"] = actions
					}
				}

				// Create a marker ServerBlock that will be compiled
				block := &ServerBlock{
					ID:       autoID,
					Language: "go",
					Content:  processedContent, // Store processed LVT content
					Metadata: metadata,
				}
				p.ServerBlocks[autoID] = block
				stateRef = autoID
			} else if stateRef == "" {
				stateRef = lastServerBlockID
			}

			block := &InteractiveBlock{
				ID:       getBlockID(cb, i),
				StateRef: stateRef,
				Content:  processedContent, // Use processed content with auto-generated templates
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

// NOTE: lvt-persist has been removed. Use lvt-source with type: sqlite instead.

// getLvtSource extracts the lvt-source attribute value from LVT content
// Returns empty string if not found
func getLvtSource(content string) string {
	// Look for lvt-source="name" on any element
	sourceRegex := regexp.MustCompile(`lvt-source="([^"]+)"`)
	match := sourceRegex.FindStringSubmatch(content)
	if match != nil && len(match) > 1 {
		return match[1]
	}
	return ""
}

// getLvtSourceElementType detects what kind of element has the lvt-source attribute
// Returns "table", "select", or "div" (default)
func getLvtSourceElementType(content string) string {
	// Check if lvt-source is on a table element
	tableRegex := regexp.MustCompile(`(?i)<table[^>]*lvt-source=`)
	if tableRegex.MatchString(content) {
		return "table"
	}
	// Check if lvt-source is on a select element
	selectRegex := regexp.MustCompile(`(?i)<select[^>]*lvt-source=`)
	if selectRegex.MatchString(content) {
		return "select"
	}
	return "div"
}

// getTableColumns extracts lvt-columns from a table element
// Returns a comma-separated list like "name:Name,email:Email"
func getTableColumns(content string) string {
	columnsRegex := regexp.MustCompile(`lvt-columns="([^"]+)"`)
	match := columnsRegex.FindStringSubmatch(content)
	if match != nil && len(match) > 1 {
		return match[1]
	}
	return ""
}

// getTableActions extracts lvt-actions from a table element
// Returns a comma-separated list like "edit:Edit,delete:Delete"
func getTableActions(content string) string {
	actionsRegex := regexp.MustCompile(`lvt-actions="([^"]+)"`)
	match := actionsRegex.FindStringSubmatch(content)
	if match != nil && len(match) > 1 {
		return match[1]
	}
	return ""
}

// autoGenerateTableTemplate transforms <table lvt-source="..."> into generated HTML.
//
// Two modes:
//   - Simple (default): Generates inline <thead>/<tbody> with {{range .Data}}
//   - Rich (lvt-datatable): Uses datatable component with sorting/pagination
//
// Attributes:
//   - lvt-source="name" - Required, specifies the data source
//   - lvt-columns="field:Label,field2:Label2" - Column definitions (optional, auto-discovers if omitted)
//   - lvt-actions="action:Label,action2:Label2" - Action buttons column
//   - lvt-empty="No items" - Message when data is empty
//   - lvt-datatable - Opt-in to rich datatable component mode
func autoGenerateTableTemplate(content string) string {
	// Check if this is a table with lvt-source and empty/minimal content
	tableRegex := regexp.MustCompile(`(?s)<table([^>]*lvt-source="[^"]+[^>]*)>(.*?)</table>`)
	match := tableRegex.FindStringSubmatch(content)
	if match == nil {
		return content
	}

	attrs := match[1]
	innerContent := strings.TrimSpace(match[2])

	// If table has substantial inner content (like {{range}} or {{template}}), don't override
	if strings.Contains(innerContent, "{{range") || strings.Contains(innerContent, "<tbody>") || strings.Contains(innerContent, "{{template") {
		return content
	}

	// Check if lvt-datatable is present (opt-in to rich mode)
	useDatatable := strings.Contains(attrs, "lvt-datatable")

	// Extract lvt-columns="field:Label,field2:Label2"
	columns := getTableColumns(content)

	// Extract lvt-actions="action:Label,action2:Label2"
	actions := getTableActions(content)

	// Extract lvt-empty="message"
	emptyMessage := getTableEmpty(content)

	// Clean attributes (remove lvt-* attributes we've processed)
	cleanedAttrs := attrs
	cleanedAttrs = regexp.MustCompile(`\s*lvt-source="[^"]*"`).ReplaceAllString(cleanedAttrs, "")
	cleanedAttrs = regexp.MustCompile(`\s*lvt-columns="[^"]*"`).ReplaceAllString(cleanedAttrs, "")
	cleanedAttrs = regexp.MustCompile(`\s*lvt-actions="[^"]*"`).ReplaceAllString(cleanedAttrs, "")
	cleanedAttrs = regexp.MustCompile(`\s*lvt-empty="[^"]*"`).ReplaceAllString(cleanedAttrs, "")
	cleanedAttrs = regexp.MustCompile(`\s*lvt-datatable`).ReplaceAllString(cleanedAttrs, "")
	cleanedAttrs = strings.TrimSpace(cleanedAttrs)

	var generated strings.Builder

	if useDatatable {
		// Rich mode: use datatable component
		if cleanedAttrs != "" {
			generated.WriteString(fmt.Sprintf("<div%s>\n", cleanedAttrs))
			generated.WriteString("{{template \"lvt:datatable:default:v1\" .Table}}\n")
			generated.WriteString("</div>")
		} else {
			generated.WriteString("{{template \"lvt:datatable:default:v1\" .Table}}")
		}
	} else {
		// Simple mode: generate inline HTML
		if cleanedAttrs != "" {
			generated.WriteString(fmt.Sprintf("<table %s>\n", cleanedAttrs))
		} else {
			generated.WriteString("<table>\n")
		}
		generateSimpleTable(&generated, columns, actions, emptyMessage)
		generated.WriteString("</table>")
	}

	return tableRegex.ReplaceAllLiteralString(content, generated.String())
}

// generateSimpleTable generates simple inline table HTML with thead/tbody
func generateSimpleTable(w *strings.Builder, columns, actions, emptyMessage string) {
	// Parse columns: "field:Label,field2:Label2" or "field,field2"
	var cols []struct {
		field string
		label string
	}

	if columns != "" {
		for _, pair := range strings.Split(columns, ",") {
			parts := strings.SplitN(strings.TrimSpace(pair), ":", 2)
			field := parts[0]
			label := field
			if len(parts) > 1 {
				label = parts[1]
			} else {
				// Auto-titlecase the field name for label
				label = titleCase(field)
			}
			cols = append(cols, struct {
				field string
				label string
			}{field, label})
		}
	}

	// Parse actions: "action:Label,action2:Label2"
	var acts []struct {
		action string
		label  string
	}
	if actions != "" {
		for _, pair := range strings.Split(actions, ",") {
			parts := strings.SplitN(strings.TrimSpace(pair), ":", 2)
			action := parts[0]
			label := action
			if len(parts) > 1 {
				label = parts[1]
			} else {
				label = titleCase(action)
			}
			acts = append(acts, struct {
				action string
				label  string
			}{action, label})
		}
	}

	// If no columns specified, auto-discover from first row
	if len(cols) == 0 {
		w.WriteString("{{if .Data}}\n")
		w.WriteString("  <thead>\n    <tr>\n")
		w.WriteString("      {{range $key, $_ := index .Data 0}}\n")
		w.WriteString("      <th>{{$key}}</th>\n")
		w.WriteString("      {{end}}\n")
		if len(acts) > 0 {
			w.WriteString("      <th>Actions</th>\n")
		}
		w.WriteString("    </tr>\n  </thead>\n")
		w.WriteString("  <tbody>\n")
		w.WriteString("    {{range .Data}}\n    <tr>\n")
		w.WriteString("      {{range $key, $value := .}}\n")
		w.WriteString("      <td>{{$value}}</td>\n")
		w.WriteString("      {{end}}\n")
		if len(acts) > 0 {
			w.WriteString("      <td>\n")
			for _, act := range acts {
				w.WriteString(fmt.Sprintf("        <button lvt-click=\"%s\" lvt-data-id=\"{{.Id}}\">%s</button>\n", act.action, act.label))
			}
			w.WriteString("      </td>\n")
		}
		w.WriteString("    </tr>\n    {{end}}\n")
		w.WriteString("  </tbody>\n")
		w.WriteString("{{else}}\n")
		if emptyMessage != "" {
			w.WriteString(fmt.Sprintf("  <tbody><tr><td>%s</td></tr></tbody>\n", emptyMessage))
		} else {
			w.WriteString("  <tbody><tr><td>No data</td></tr></tbody>\n")
		}
		w.WriteString("{{end}}\n")
		return
	}

	// Generate with explicit columns
	w.WriteString("  <thead>\n    <tr>\n")
	for _, col := range cols {
		w.WriteString(fmt.Sprintf("      <th>%s</th>\n", col.label))
	}
	if len(acts) > 0 {
		w.WriteString("      <th>Actions</th>\n")
	}
	w.WriteString("    </tr>\n  </thead>\n")

	// Empty state handling
	if emptyMessage != "" {
		w.WriteString("  {{if not .Data}}\n")
		colSpan := len(cols)
		if len(acts) > 0 {
			colSpan++
		}
		w.WriteString(fmt.Sprintf("  <tbody><tr><td colspan=\"%d\">%s</td></tr></tbody>\n", colSpan, emptyMessage))
		w.WriteString("  {{else}}\n")
	}

	w.WriteString("  <tbody>\n")
	w.WriteString("    {{range .Data}}\n    <tr>\n")
	for _, col := range cols {
		// Use titlecase field name for Go template access
		w.WriteString(fmt.Sprintf("      <td>{{.%s}}</td>\n", titleCase(col.field)))
	}
	if len(acts) > 0 {
		w.WriteString("      <td>\n")
		for _, act := range acts {
			w.WriteString(fmt.Sprintf("        <button lvt-click=\"%s\" lvt-data-id=\"{{.Id}}\">%s</button>\n", act.action, act.label))
		}
		w.WriteString("      </td>\n")
	}
	w.WriteString("    </tr>\n    {{end}}\n")
	w.WriteString("  </tbody>\n")

	if emptyMessage != "" {
		w.WriteString("  {{end}}\n")
	}
}

// getTableEmpty extracts lvt-empty attribute value
func getTableEmpty(content string) string {
	emptyRegex := regexp.MustCompile(`lvt-empty="([^"]+)"`)
	match := emptyRegex.FindStringSubmatch(content)
	if match != nil && len(match) > 1 {
		return match[1]
	}
	return ""
}

// titleCase converts a string to title case (first letter uppercase only)
// This is a simple transformation: "name" -> "Name", "created_at" -> "Created_at"
// Note: With dual-key hydration, templates can use either case, but generated
// templates use titlecase for consistency with Go naming conventions.
func titleCase(s string) string {
	if len(s) == 0 {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

// autoGenerateSelectTemplate transforms <select lvt-source="..."> into a full template
// if the select is empty. Supports lvt-value and lvt-label attributes.
func autoGenerateSelectTemplate(content string) string {
	// Check if this is a select with lvt-source and empty/minimal content
	selectRegex := regexp.MustCompile(`(?s)<select([^>]*lvt-source="[^"]+[^>]*)>(.*?)</select>`)
	match := selectRegex.FindStringSubmatch(content)
	if match == nil {
		return content
	}

	attrs := match[1]
	innerContent := strings.TrimSpace(match[2])

	// If select has substantial inner content (like {{range}} or <option>), don't override
	if strings.Contains(innerContent, "{{range") || strings.Contains(innerContent, "<option") {
		return content
	}

	// Parse lvt-value="fieldName" - defaults to "Id"
	// Use titleCase since processMapValues titlecases all nested keys for Go template access
	valueField := "Id"
	valueMatch := regexp.MustCompile(`lvt-value="([^"]+)"`).FindStringSubmatch(attrs)
	if valueMatch != nil {
		valueField = titleCase(valueMatch[1])
	}

	// Parse lvt-label="fieldName" - defaults to "Name"
	// Use titleCase since processMapValues titlecases all nested keys for Go template access
	labelField := "Name"
	labelMatch := regexp.MustCompile(`lvt-label="([^"]+)"`).FindStringSubmatch(attrs)
	if labelMatch != nil {
		labelField = titleCase(labelMatch[1])
	}

	// Build cleaned attributes (remove lvt-value and lvt-label)
	cleanedAttrs := attrs
	cleanedAttrs = regexp.MustCompile(`\s*lvt-value="[^"]*"`).ReplaceAllString(cleanedAttrs, "")
	cleanedAttrs = regexp.MustCompile(`\s*lvt-label="[^"]*"`).ReplaceAllString(cleanedAttrs, "")

	// Generate the template
	var generated strings.Builder
	generated.WriteString("<select")
	generated.WriteString(cleanedAttrs)
	generated.WriteString(">\n")
	generated.WriteString("  {{range .Data}}\n")
	generated.WriteString(fmt.Sprintf("  <option value=\"{{.%s}}\">{{.%s}}</option>\n", valueField, labelField))
	generated.WriteString("  {{end}}\n")
	generated.WriteString("</select>")

	// Use ReplaceAllLiteralString to avoid special chars being interpreted as backreferences
	return selectRegex.ReplaceAllLiteralString(content, generated.String())
}
