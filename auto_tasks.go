package tinkerdown

import (
	"bytes"
	"fmt"
	"regexp"
	"strings"
)

// taskListSection represents a detected section of task list items under a heading.
type taskListSection struct {
	heading   string // raw heading text
	anchor    string // slugified anchor (e.g., "todos")
	level     int    // heading level (number of # chars)
	startLine int    // first line of task items (0-indexed, after heading)
	endLine   int    // last line of task items (exclusive)
}

// Pre-compiled patterns for task list detection
var (
	taskItemPattern    = regexp.MustCompile(`^\s*-\s+\[([ xX])\]\s+`)
	headingPattern     = regexp.MustCompile(`^(#{1,6})\s+(.+?)\s*$`)
	frontmatterPattern = regexp.MustCompile(`(?s)\A---\n(.+?)\n---\n`)
)

// detectTaskListSections scans markdown content (after frontmatter) and finds
// sections where ALL non-blank lines under a heading are task list items.
// Sections with mixed prose + tasks are skipped. Task lists without a heading are skipped.
func detectTaskListSections(content []byte) []taskListSection {
	// Quick check: if no task pattern exists, return early
	if !bytes.Contains(content, []byte("- [ ]")) && !bytes.Contains(content, []byte("- [x]")) && !bytes.Contains(content, []byte("- [X]")) {
		return nil
	}

	lines := strings.Split(string(content), "\n")
	var sections []taskListSection

	var currentHeading string
	var currentAnchor string
	var currentLevel int
	var sectionStart int
	var hasTaskItems bool
	var hasMixedContent bool

	flushSection := func(endLine int) {
		if currentHeading != "" && hasTaskItems && !hasMixedContent && sectionStart < endLine {
			sections = append(sections, taskListSection{
				heading:   currentHeading,
				anchor:    currentAnchor,
				level:     currentLevel,
				startLine: sectionStart,
				endLine:   endLine,
			})
		}
		hasTaskItems = false
		hasMixedContent = false
	}

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Check for heading
		if matches := headingPattern.FindStringSubmatch(line); matches != nil {
			// Flush previous section
			flushSection(i)

			level := len(matches[1])
			headingText := strings.TrimSpace(matches[2])

			// Check for explicit anchor syntax {#anchor}
			explicitAnchorRe := regexp.MustCompile(`\s*\{#([^}]+)\}\s*$`)
			if anchorMatch := explicitAnchorRe.FindStringSubmatch(headingText); anchorMatch != nil {
				currentAnchor = anchorMatch[1]
				headingText = explicitAnchorRe.ReplaceAllString(headingText, "")
			} else {
				currentAnchor = slugifyHeading(headingText)
			}

			currentHeading = headingText
			currentLevel = level
			sectionStart = i + 1
			continue
		}

		// Skip blank lines
		if trimmed == "" {
			continue
		}

		// No heading yet — any content before first heading is ignored
		if currentHeading == "" {
			continue
		}

		// Check if this is a task item
		if taskItemPattern.MatchString(line) {
			hasTaskItems = true
		} else {
			// Non-task, non-blank line under a heading = mixed content
			hasMixedContent = true
		}
	}

	// Flush last section
	flushSection(len(lines))

	return sections
}

// preprocessAutoTasks transforms markdown content by replacing detected task list
// sections with lvt code blocks, and returns source configs to inject.
// The original file on disk is never modified.
func preprocessAutoTasks(content []byte, absPath string) ([]byte, map[string]SourceConfig) {
	// Skip frontmatter during detection
	bodyOffset := 0
	if loc := frontmatterPattern.FindIndex(content); loc != nil {
		bodyOffset = loc[1]
	}

	body := content[bodyOffset:]
	sections := detectTaskListSections(body)
	if len(sections) == 0 {
		return content, nil
	}

	sources := make(map[string]SourceConfig)
	lines := strings.Split(string(content), "\n")

	// Adjust line indices to account for frontmatter
	fmLineCount := 0
	if bodyOffset > 0 {
		fmLineCount = strings.Count(string(content[:bodyOffset]), "\n")
	}

	// Process sections in reverse order to preserve line numbers
	for i := len(sections) - 1; i >= 0; i-- {
		sec := sections[i]
		sourceName := "_auto_" + sec.anchor

		// Skip if this anchor already has a manually configured source
		// (checked later during injection in ParseFile)

		// Build the lvt code block replacement
		lvtBlock := generateAutoTaskLvtBlock(sourceName)

		// Build replacement lines: heading + lvt code block
		var replacement []string
		// Keep the heading line
		headingLine := lines[sec.startLine-1+fmLineCount]
		replacement = append(replacement, headingLine)
		replacement = append(replacement, "")
		replacement = append(replacement, "```lvt")
		replacement = append(replacement, lvtBlock)
		replacement = append(replacement, "```")

		// Replace lines: heading line through end of task items
		startIdx := sec.startLine - 1 + fmLineCount // heading line
		endIdx := sec.endLine + fmLineCount          // exclusive

		newLines := make([]string, 0, len(lines)-(endIdx-startIdx)+len(replacement))
		newLines = append(newLines, lines[:startIdx]...)
		newLines = append(newLines, replacement...)
		newLines = append(newLines, lines[endIdx:]...)
		lines = newLines

		// Build source config pointing to the original file
		readonlyFalse := false
		sources[sourceName] = SourceConfig{
			Type:     "markdown",
			File:     absPath,
			Anchor:   "#" + sec.anchor,
			Readonly: &readonlyFalse,
		}
	}

	return []byte(strings.Join(lines, "\n")), sources
}

// generateAutoTaskLvtBlock generates the lvt template HTML for an auto-task section.
func generateAutoTaskLvtBlock(sourceName string) string {
	return fmt.Sprintf(`<div lvt-source="%s">
{{range .Data}}
<label style="display: block; padding: 4px 0; cursor: pointer;">
  <input type="checkbox" {{if .Done}}checked{{end}} lvt-click="Toggle" lvt-data-id="{{.Id}}">
  <span {{if .Done}}style="text-decoration: line-through; opacity: 0.6"{{end}}>{{.Text}}</span>
</label>
{{end}}
<form lvt-submit="Add" lvt-reset-on:success style="display: flex; gap: 8px; align-items: center; margin-top: 8px;">
  <input type="text" name="text" placeholder="Add new task..." required
         style="flex: 1; min-width: 0; padding: 6px 8px; border: 1px solid #ccc; border-radius: 4px;">
  <button type="submit"
          style="flex-shrink: 0; width: auto; padding: 8px 16px; background: #007bff; color: white; border: none; border-radius: 4px; cursor: pointer;">
    Add
  </button>
</form>
</div>`, sourceName)
}

// slugifyHeading converts heading text to a URL-safe anchor slug (GitHub-style).
// This must match the slugify() function in internal/source/markdown.go for anchor matching.
func slugifyHeading(text string) string {
	text = strings.ToLower(text)
	text = strings.ReplaceAll(text, " ", "-")
	re := regexp.MustCompile(`[^a-z0-9-]`)
	return re.ReplaceAllString(text, "")
}
