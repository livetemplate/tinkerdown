package commands

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNewCommandBasicTemplate(t *testing.T) {
	// Create temp directory for test
	tmpDir := t.TempDir()
	projectDir := filepath.Join(tmpDir, "test-basic")

	// Change to temp directory
	defer chdir(t, tmpDir)()

	// Run new command with basic template (default)
	err := NewCommand([]string{"test-basic"}, "")
	if err != nil {
		t.Fatalf("NewCommand failed: %v", err)
	}

	// Verify files were created
	assertFileExists(t, projectDir, "index.md")
	assertFileExists(t, projectDir, "README.md")

	// Verify title substitution
	content := readFile(t, filepath.Join(projectDir, "index.md"))
	if !strings.Contains(content, "title: \"Test Basic\"") {
		t.Errorf("Expected title to be substituted, got: %s", content[:100])
	}
}

func TestNewCommandTodoTemplate(t *testing.T) {
	tmpDir := t.TempDir()
	projectDir := filepath.Join(tmpDir, "my-todo-app")

	defer chdir(t, tmpDir)()

	err := NewCommand([]string{"my-todo-app"}, "todo")
	if err != nil {
		t.Fatalf("NewCommand failed: %v", err)
	}

	assertFileExists(t, projectDir, "index.md")
	assertFileExists(t, projectDir, "README.md")

	// Verify SQLite source configuration
	content := readFile(t, filepath.Join(projectDir, "index.md"))
	if !strings.Contains(content, "type: sqlite") {
		t.Error("Expected sqlite source type in todo template")
	}
	if !strings.Contains(content, "# My Todo App") {
		t.Error("Expected title to be substituted")
	}
}

func TestNewCommandDashboardTemplate(t *testing.T) {
	tmpDir := t.TempDir()
	projectDir := filepath.Join(tmpDir, "my-dashboard")

	defer chdir(t, tmpDir)()

	err := NewCommand([]string{"my-dashboard"}, "dashboard")
	if err != nil {
		t.Fatalf("NewCommand failed: %v", err)
	}

	assertFileExists(t, projectDir, "index.md")
	assertFileExists(t, projectDir, "README.md")
	assertFileExists(t, projectDir, "system-info.sh")

	// Verify multiple sources configured (REST + exec)
	content := readFile(t, filepath.Join(projectDir, "index.md"))
	if !strings.Contains(content, "type: rest") {
		t.Error("Expected rest source type in dashboard template")
	}
	if !strings.Contains(content, "type: exec") {
		t.Error("Expected exec source type in dashboard template")
	}
}

func TestNewCommandFormTemplate(t *testing.T) {
	tmpDir := t.TempDir()
	projectDir := filepath.Join(tmpDir, "contact-form")

	defer chdir(t, tmpDir)()

	err := NewCommand([]string{"contact-form"}, "form")
	if err != nil {
		t.Fatalf("NewCommand failed: %v", err)
	}

	assertFileExists(t, projectDir, "index.md")
	assertFileExists(t, projectDir, "README.md")

	content := readFile(t, filepath.Join(projectDir, "index.md"))
	if !strings.Contains(content, "type: sqlite") {
		t.Error("Expected sqlite source type in form template")
	}
	if !strings.Contains(content, "lvt-submit") {
		t.Error("Expected form submit handler in form template")
	}
}

func TestNewCommandAPIExplorerTemplate(t *testing.T) {
	tmpDir := t.TempDir()
	projectDir := filepath.Join(tmpDir, "api-test")

	defer chdir(t, tmpDir)()

	err := NewCommand([]string{"api-test"}, "api-explorer")
	if err != nil {
		t.Fatalf("NewCommand failed: %v", err)
	}

	assertFileExists(t, projectDir, "index.md")
	assertFileExists(t, projectDir, "README.md")

	content := readFile(t, filepath.Join(projectDir, "index.md"))
	if !strings.Contains(content, "type: rest") {
		t.Error("Expected rest source type in api-explorer template")
	}
	if !strings.Contains(content, "api.github.com") {
		t.Error("Expected GitHub API URL in api-explorer template")
	}
}

func TestNewCommandTutorialTemplate(t *testing.T) {
	tmpDir := t.TempDir()
	projectDir := filepath.Join(tmpDir, "my-tutorial")

	defer chdir(t, tmpDir)()

	err := NewCommand([]string{"my-tutorial"}, "tutorial")
	if err != nil {
		t.Fatalf("NewCommand failed: %v", err)
	}

	assertFileExists(t, projectDir, "index.md")
	assertFileExists(t, projectDir, "README.md")
}

func TestNewCommandWASMSourceTemplate(t *testing.T) {
	tmpDir := t.TempDir()
	projectDir := filepath.Join(tmpDir, "my-source")

	defer chdir(t, tmpDir)()

	err := NewCommand([]string{"my-source"}, "wasm-source")
	if err != nil {
		t.Fatalf("NewCommand failed: %v", err)
	}

	assertFileExists(t, projectDir, "source.go")
	assertFileExists(t, projectDir, "Makefile")
	assertFileExists(t, projectDir, "README.md")

	// Verify Go source has correct build tag
	content := readFile(t, filepath.Join(projectDir, "source.go"))
	if !strings.Contains(content, "//go:build tinygo.wasm") {
		t.Error("Expected tinygo build tag in source.go")
	}

	// Verify Makefile builds wasm
	makefile := readFile(t, filepath.Join(projectDir, "Makefile"))
	if !strings.Contains(makefile, "source.wasm") {
		t.Error("Expected source.wasm in Makefile")
	}
}

func TestNewCommandListTemplates(t *testing.T) {
	// The --list flag is handled by cobra, not NewCommand directly
	// This test verifies that NewCommand with empty args returns an error
	err := NewCommand([]string{}, "")
	if err == nil {
		t.Fatal("Expected error when no project name provided")
	}
}

func TestNewCommandInvalidTemplate(t *testing.T) {
	tmpDir := t.TempDir()

	defer chdir(t, tmpDir)()

	err := NewCommand([]string{"test"}, "invalid")
	if err == nil {
		t.Fatal("Expected error for invalid template")
	}
	if !strings.Contains(err.Error(), "unknown template") {
		t.Errorf("Expected 'unknown template' error, got: %v", err)
	}
}

func TestNewCommandDirectoryExists(t *testing.T) {
	tmpDir := t.TempDir()
	projectDir := filepath.Join(tmpDir, "existing")

	// Create the directory first
	os.MkdirAll(projectDir, 0755)

	defer chdir(t, tmpDir)()

	err := NewCommand([]string{"existing"}, "")
	if err == nil {
		t.Fatal("Expected error when directory exists")
	}
	if !strings.Contains(err.Error(), "already exists") {
		t.Errorf("Expected 'already exists' error, got: %v", err)
	}
}

func TestNewCommandNoProjectName(t *testing.T) {
	err := NewCommand([]string{}, "")
	if err == nil {
		t.Fatal("Expected error when no project name")
	}
	if !strings.Contains(err.Error(), "project name required") {
		t.Errorf("Expected 'project name required' error, got: %v", err)
	}
}

func TestNewCommandProjectNameWithSpaces(t *testing.T) {
	tmpDir := t.TempDir()

	defer chdir(t, tmpDir)()

	err := NewCommand([]string{"my project"}, "")
	if err == nil {
		t.Fatal("Expected error for project name with spaces")
	}
	if !strings.Contains(err.Error(), "cannot contain spaces") {
		t.Errorf("Expected 'cannot contain spaces' error, got: %v", err)
	}
}

func TestNewCommandTitleConversion(t *testing.T) {
	tmpDir := t.TempDir()
	projectDir := filepath.Join(tmpDir, "my-awesome-app")

	defer chdir(t, tmpDir)()

	err := NewCommand([]string{"my-awesome-app"}, "")
	if err != nil {
		t.Fatalf("NewCommand failed: %v", err)
	}

	content := readFile(t, filepath.Join(projectDir, "index.md"))
	if !strings.Contains(content, "title: \"My Awesome App\"") {
		t.Errorf("Expected hyphenated name to be converted to title case")
	}
}

// Helper functions

// chdir changes to tmpDir and returns a cleanup function to restore the original directory
func chdir(t *testing.T, tmpDir string) func() {
	t.Helper()
	oldDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("Failed to change to temp directory: %v", err)
	}
	return func() {
		os.Chdir(oldDir)
	}
}

func assertFileExists(t *testing.T, dir, filename string) {
	t.Helper()
	path := filepath.Join(dir, filename)
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Errorf("Expected file %s to exist", path)
	}
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("Failed to read file %s: %v", path, err)
	}
	return string(content)
}
