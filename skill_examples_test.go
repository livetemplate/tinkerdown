package livepage_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// TestSkillExamplesValidation runs the validation script on skill examples.
// This is a golden file test - it ensures all skill examples remain valid
// as the LivePage API evolves.
func TestSkillExamplesValidation(t *testing.T) {
	// Find the validation script
	scriptPath := "skills/livepage/scripts/validate.sh"
	if _, err := os.Stat(scriptPath); os.IsNotExist(err) {
		t.Skipf("Validation script not found at %s", scriptPath)
	}

	// Run the validation script
	cmd := exec.Command("bash", scriptPath)
	output, err := cmd.CombinedOutput()

	if err != nil {
		t.Fatalf("Validation script failed:\n%s", string(output))
	}

	// Check output contains expected pass count
	outputStr := string(output)
	if !strings.Contains(outputStr, "All examples valid!") {
		t.Errorf("Expected 'All examples valid!' in output, got:\n%s", outputStr)
	}

	t.Logf("Validation output:\n%s", outputStr)
}

// TestSkillExamplesExist verifies all expected example files exist
func TestSkillExamplesExist(t *testing.T) {
	expectedExamples := []string{
		"01-todo-app.md",
		"02-dashboard.md",
		"03-contact-form.md",
		"04-blog.md",
		"05-inventory.md",
		"06-survey.md",
		"07-booking.md",
		"08-expense-tracker.md",
		"09-faq.md",
		"10-status-page.md",
	}

	examplesDir := "skills/livepage/examples"

	for _, example := range expectedExamples {
		path := filepath.Join(examplesDir, example)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Errorf("Missing example file: %s", path)
		}
	}
}

// TestSkillFilesExist verifies all skill documentation files exist
func TestSkillFilesExist(t *testing.T) {
	requiredFiles := []string{
		"skills/livepage/SKILL.md",
		"skills/livepage/reference.md",
		"skills/livepage/scripts/validate.sh",
	}

	for _, file := range requiredFiles {
		if _, err := os.Stat(file); os.IsNotExist(err) {
			t.Errorf("Missing skill file: %s", file)
		}
	}
}

// TestSkillExampleStructure verifies each example has required elements
func TestSkillExampleStructure(t *testing.T) {
	examplesDir := "skills/livepage/examples"

	files, err := os.ReadDir(examplesDir)
	if err != nil {
		t.Fatalf("Failed to read examples directory: %v", err)
	}

	for _, file := range files {
		if !strings.HasSuffix(file.Name(), ".md") {
			continue
		}

		path := filepath.Join(examplesDir, file.Name())
		content, err := os.ReadFile(path)
		if err != nil {
			t.Errorf("Failed to read %s: %v", file.Name(), err)
			continue
		}

		contentStr := string(content)

		// Check for frontmatter
		if !strings.HasPrefix(contentStr, "---") {
			t.Errorf("%s: Missing frontmatter (should start with ---)", file.Name())
		}

		// Check for title in frontmatter
		if !strings.Contains(contentStr, "title:") {
			t.Errorf("%s: Missing title in frontmatter", file.Name())
		}

		// Check for lvt code block
		if !strings.Contains(contentStr, "```lvt") {
			t.Errorf("%s: Missing lvt code block", file.Name())
		}

		// Check for at least one lvt-* attribute
		if !strings.Contains(contentStr, "lvt-") {
			t.Errorf("%s: No lvt-* attributes found", file.Name())
		}

		// Check for "How It Works" section (documentation)
		if !strings.Contains(contentStr, "## How It Works") {
			t.Errorf("%s: Missing 'How It Works' section", file.Name())
		}

		// Check for "Prompt to Generate This" section
		if !strings.Contains(contentStr, "## Prompt to Generate This") {
			t.Errorf("%s: Missing 'Prompt to Generate This' section", file.Name())
		}
	}
}

// TestLLMSTxtExists verifies the LLMS.txt file exists and has required sections
func TestLLMSTxtExists(t *testing.T) {
	llmsPath := "docs/llms.txt"

	content, err := os.ReadFile(llmsPath)
	if os.IsNotExist(err) {
		t.Fatalf("LLMS.txt not found at %s", llmsPath)
	}
	if err != nil {
		t.Fatalf("Failed to read LLMS.txt: %v", err)
	}

	contentStr := string(content)

	requiredSections := []string{
		"# LivePage",
		"## Quick Start",
		"## Key Attributes",
		"lvt-submit",
		"lvt-persist",
		"lvt-click",
		"lvt-source",
	}

	for _, section := range requiredSections {
		if !strings.Contains(contentStr, section) {
			t.Errorf("LLMS.txt missing required section/content: %s", section)
		}
	}
}
