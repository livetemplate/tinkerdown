package livepage

import (
	"os"
	"strings"
	"testing"
)

func TestParseErrorFormatting(t *testing.T) {
	// Create a temporary test file
	content := "---\ntitle: \"Test\"\n---\n\n# Test\n\n```go server\ntype TestState struct {\n    Value int\n}\n```\n\n```lvt state=\"wrong-state\"\n<div>{{.Value}}</div>\n```\n"
	tmpfile, err := os.CreateTemp("", "test-*.md")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpfile.Name())

	if _, err := tmpfile.Write([]byte(content)); err != nil {
		t.Fatal(err)
	}
	tmpfile.Close()

	// Try to parse - should fail with nice error
	_, err = ParseFile(tmpfile.Name())
	if err == nil {
		t.Fatal("Expected error, got nil")
	}

	errMsg := err.Error()
	t.Logf("Error message:\n%s", errMsg)

	// Check that error message contains expected elements
	if !strings.Contains(errMsg, "❌ Error in") {
		t.Errorf("Error should start with ❌ Error in")
	}

	if !strings.Contains(errMsg, "Line") {
		t.Errorf("Error should mention line number")
	}

	if !strings.Contains(errMsg, "wrong-state") {
		t.Errorf("Error should mention the invalid reference")
	}

	if !strings.Contains(errMsg, "💡 Tip:") {
		t.Errorf("Error should include helpful tip")
	}
}

func TestParseErrorWithContext(t *testing.T) {
	err := NewParseError("/path/to/file.md", 42, "Something went wrong").
		WithHint("Try doing X instead").
		WithRelated("See line 10 for related issue")

	errMsg := err.Error()
	t.Logf("Error message:\n%s", errMsg)

	if !strings.Contains(errMsg, "❌ Error in /path/to/file.md") {
		t.Errorf("Error should mention file path")
	}

	if !strings.Contains(errMsg, "Line 42") {
		t.Errorf("Error should mention line 42")
	}

	if !strings.Contains(errMsg, "Something went wrong") {
		t.Errorf("Error should include message")
	}

	if !strings.Contains(errMsg, "💡 Tip: Try doing X instead") {
		t.Errorf("Error should include hint")
	}

	if !strings.Contains(errMsg, "🔗 See line 10 for related issue") {
		t.Errorf("Error should include related info")
	}
}
