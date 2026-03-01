// Package slug provides heading-to-anchor conversion (GitHub-style slugification).
package slug

import (
	"regexp"
	"strings"
)

var nonAlphanumPattern = regexp.MustCompile(`[^a-z0-9-]`)

// Heading converts heading text to a URL-safe anchor slug.
// Example: "My Task List" → "my-task-list"
func Heading(text string) string {
	text = strings.ToLower(text)
	text = strings.ReplaceAll(text, " ", "-")
	return nonAlphanumPattern.ReplaceAllString(text, "")
}
