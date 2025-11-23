package livepage

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

// ParseError represents a detailed parsing error with context.
type ParseError struct {
	File    string // Source file path
	Line    int    // Line number (1-indexed)
	Column  int    // Column number (1-indexed, optional)
	Message string // Error message
	Code    string // Offending code snippet
	Hint    string // Helpful suggestion
	Related string // Related information (e.g., "State 'foo' defined at line 20")
}

// Error implements the error interface.
func (e *ParseError) Error() string {
	return e.Format()
}

// Format returns a nicely formatted error message with context.
func (e *ParseError) Format() string {
	var b strings.Builder

	// Header
	b.WriteString(fmt.Sprintf("❌ Error in %s\n\n", e.File))

	// Main error message with line number
	b.WriteString(fmt.Sprintf("Line %d: %s\n", e.Line, e.Message))

	// Code context (show surrounding lines)
	if e.Code != "" || e.File != "" {
		context := e.getCodeContext()
		if context != "" {
			b.WriteString(context)
		}
	}

	// Helpful hint
	if e.Hint != "" {
		b.WriteString(fmt.Sprintf("\n💡 Tip: %s\n", e.Hint))
	}

	// Related information
	if e.Related != "" {
		b.WriteString(fmt.Sprintf("\n🔗 %s\n", e.Related))
	}

	return b.String()
}

// getCodeContext reads the source file and extracts context around the error line.
func (e *ParseError) getCodeContext() string {
	if e.File == "" {
		return ""
	}

	file, err := os.Open(e.File)
	if err != nil {
		return ""
	}
	defer file.Close()

	var lines []string
	scanner := bufio.NewScanner(file)
	lineNum := 1

	for scanner.Scan() {
		lines = append(lines, scanner.Text())
		lineNum++
	}

	if e.Line < 1 || e.Line > len(lines) {
		return ""
	}

	var b strings.Builder
	b.WriteString("\n")

	// Show 2 lines before, the error line, and 2 lines after
	start := max(1, e.Line-2)
	end := min(len(lines), e.Line+2)

	for i := start; i <= end; i++ {
		prefix := fmt.Sprintf("  %2d | ", i)
		line := ""
		if i <= len(lines) {
			line = lines[i-1]
		}

		b.WriteString(prefix + line + "\n")

		// Add error pointer on the error line
		if i == e.Line && e.Column > 0 {
			// Calculate spaces needed (accounting for line number prefix)
			spaces := strings.Repeat(" ", len(prefix)+e.Column-1)
			b.WriteString(spaces + "^\n")
		}
	}

	return b.String()
}

// NewParseError creates a new ParseError.
func NewParseError(file string, line int, message string) *ParseError {
	return &ParseError{
		File:    file,
		Line:    line,
		Message: message,
	}
}

// WithColumn adds column information to the error.
func (e *ParseError) WithColumn(col int) *ParseError {
	e.Column = col
	return e
}

// WithHint adds a helpful hint to the error.
func (e *ParseError) WithHint(hint string) *ParseError {
	e.Hint = hint
	return e
}

// WithRelated adds related information to the error.
func (e *ParseError) WithRelated(related string) *ParseError {
	e.Related = related
	return e
}

// Helper functions
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
