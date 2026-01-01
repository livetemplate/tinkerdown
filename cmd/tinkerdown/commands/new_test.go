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
	err := NewCommand([]string{"test-basic"})
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

	err := NewCommand([]string{"--template=todo", "my-todo-app"})
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

	err := NewCommand([]string{"--template=dashboard", "my-dashboard"})
	if err != nil {
		t.Fatalf("NewCommand failed: %v", err)
	}

	assertFileExists(t, projectDir, "index.md")
	assertFileExists(t, projectDir, "README.md")
	assertFileExists(t, projectDir, "_data/tasks.md")
	assertFileExists(t, projectDir, "_data/team.md")

	// Verify multiple sources configured
	content := readFile(t, filepath.Join(projectDir, "index.md"))
	if !strings.Contains(content, "type: markdown") {
		t.Error("Expected markdown source type in dashboard template")
	}
}

func TestNewCommandFormTemplate(t *testing.T) {
	tmpDir := t.TempDir()
	projectDir := filepath.Join(tmpDir, "contact-form")

	defer chdir(t, tmpDir)()

	err := NewCommand([]string{"--template=form", "contact-form"})
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

	err := NewCommand([]string{"--template=api-explorer", "api-test"})
	if err != nil {
		t.Fatalf("NewCommand failed: %v", err)
	}

	assertFileExists(t, projectDir, "index.md")
	assertFileExists(t, projectDir, "README.md")

	content := readFile(t, filepath.Join(projectDir, "index.md"))
	if !strings.Contains(content, "type: rest") {
		t.Error("Expected rest source type in api-explorer template")
	}
	if !strings.Contains(content, "jsonplaceholder.typicode.com") {
		t.Error("Expected example API URL in api-explorer template")
	}
}

func TestNewCommandCLIWrapperTemplate(t *testing.T) {
	tmpDir := t.TempDir()
	projectDir := filepath.Join(tmpDir, "cli-tool")

	defer chdir(t, tmpDir)()

	err := NewCommand([]string{"--template=cli-wrapper", "cli-tool"})
	if err != nil {
		t.Fatalf("NewCommand failed: %v", err)
	}

	assertFileExists(t, projectDir, "index.md")
	assertFileExists(t, projectDir, "README.md")

	content := readFile(t, filepath.Join(projectDir, "index.md"))
	if !strings.Contains(content, "type: exec") {
		t.Error("Expected exec source type in cli-wrapper template")
	}
	if !strings.Contains(content, "manual: true") {
		t.Error("Expected manual execution mode in cli-wrapper template")
	}
}

func TestNewCommandWASMSourceTemplate(t *testing.T) {
	tmpDir := t.TempDir()
	projectDir := filepath.Join(tmpDir, "my-source")

	defer chdir(t, tmpDir)()

	err := NewCommand([]string{"--template=wasm-source", "my-source"})
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
	if !strings.Contains(content, "my-source.wasm") {
		t.Error("Expected project name in build comment")
	}

	// Verify Makefile has correct output name
	makefile := readFile(t, filepath.Join(projectDir, "Makefile"))
	if !strings.Contains(makefile, "OUTPUT = my-source.wasm") {
		t.Error("Expected project name in Makefile OUTPUT")
	}
}

func TestNewCommandListTemplates(t *testing.T) {
	// Just verify it doesn't error
	err := NewCommand([]string{"--list"})
	if err != nil {
		t.Fatalf("NewCommand --list failed: %v", err)
	}
}

func TestNewCommandInvalidTemplate(t *testing.T) {
	tmpDir := t.TempDir()

	defer chdir(t, tmpDir)()

	err := NewCommand([]string{"--template=invalid", "test"})
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

	err := NewCommand([]string{"existing"})
	if err == nil {
		t.Fatal("Expected error when directory exists")
	}
	if !strings.Contains(err.Error(), "already exists") {
		t.Errorf("Expected 'already exists' error, got: %v", err)
	}
}

func TestNewCommandNoProjectName(t *testing.T) {
	err := NewCommand([]string{})
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

	err := NewCommand([]string{"my project"})
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

	err := NewCommand([]string{"my-awesome-app"})
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
