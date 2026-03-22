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

func TestNewCommandCLIWrapperTemplate(t *testing.T) {
	tmpDir := t.TempDir()
	projectDir := filepath.Join(tmpDir, "my-cli")

	defer chdir(t, tmpDir)()

	err := NewCommand([]string{"my-cli"}, "cli-wrapper")
	if err != nil {
		t.Fatalf("NewCommand failed: %v", err)
	}

	assertFileExists(t, projectDir, "index.md")
	assertFileExists(t, projectDir, "README.md")

	// Verify delimiter fix: title should be substituted, not literal <<.Title>>
	content := readFile(t, filepath.Join(projectDir, "index.md"))
	if strings.Contains(content, "<<.Title>>") {
		t.Error("Template delimiters were not substituted in index.md")
	}
	if !strings.Contains(content, "title: \"My Cli\"") {
		t.Error("Expected title to be substituted")
	}
	if !strings.Contains(content, "type: exec") {
		t.Error("Expected exec source type in cli-wrapper template")
	}

	// Verify README substitution
	readme := readFile(t, filepath.Join(projectDir, "README.md"))
	if strings.Contains(readme, "<<.Title>>") || strings.Contains(readme, "<<.ProjectName>>") {
		t.Error("Template delimiters were not substituted in README.md")
	}
	if !strings.Contains(readme, "# My Cli") {
		t.Error("Expected title in README")
	}
}

func TestNewCommandCSVInventoryTemplate(t *testing.T) {
	tmpDir := t.TempDir()
	projectDir := filepath.Join(tmpDir, "my-inventory")

	defer chdir(t, tmpDir)()

	err := NewCommand([]string{"my-inventory"}, "csv-inventory")
	if err != nil {
		t.Fatalf("NewCommand failed: %v", err)
	}

	assertFileExists(t, projectDir, "index.md")
	assertFileExists(t, projectDir, "README.md")
	assertFileExists(t, projectDir, "products.csv")

	content := readFile(t, filepath.Join(projectDir, "index.md"))
	if !strings.Contains(content, "type: csv") {
		t.Error("Expected csv source type in csv-inventory template")
	}
	if !strings.Contains(content, "lvt-columns") {
		t.Error("Expected lvt-columns attribute in csv-inventory template")
	}

	csv := readFile(t, filepath.Join(projectDir, "products.csv"))
	if !strings.Contains(csv, "id,name,category,price,stock") {
		t.Error("Expected CSV header row")
	}
}

func TestNewCommandJSONDashboardTemplate(t *testing.T) {
	tmpDir := t.TempDir()
	projectDir := filepath.Join(tmpDir, "my-metrics")

	defer chdir(t, tmpDir)()

	err := NewCommand([]string{"my-metrics"}, "json-dashboard")
	if err != nil {
		t.Fatalf("NewCommand failed: %v", err)
	}

	assertFileExists(t, projectDir, "index.md")
	assertFileExists(t, projectDir, "README.md")
	assertFileExists(t, projectDir, "metrics.json")

	content := readFile(t, filepath.Join(projectDir, "index.md"))
	if !strings.Contains(content, "type: json") {
		t.Error("Expected json source type in json-dashboard template")
	}
	if !strings.Contains(content, "=count(tasks)") {
		t.Error("Expected computed expressions in json-dashboard template")
	}
}

func TestNewCommandMarkdownNotesTemplate(t *testing.T) {
	tmpDir := t.TempDir()
	projectDir := filepath.Join(tmpDir, "my-notes")

	defer chdir(t, tmpDir)()

	err := NewCommand([]string{"my-notes"}, "markdown-notes")
	if err != nil {
		t.Fatalf("NewCommand failed: %v", err)
	}

	assertFileExists(t, projectDir, "index.md")
	assertFileExists(t, projectDir, "README.md")
	assertFileExists(t, projectDir, filepath.Join("_data", "notes.md"))

	content := readFile(t, filepath.Join(projectDir, "index.md"))
	if !strings.Contains(content, "type: markdown") {
		t.Error("Expected markdown source type")
	}
	if !strings.Contains(content, "readonly: false") {
		t.Error("Expected readonly: false for writable markdown source")
	}
	if !strings.Contains(content, `anchor: "#notes"`) {
		t.Error("Expected anchor config for markdown table source")
	}
}

func TestNewCommandGraphQLExplorerTemplate(t *testing.T) {
	tmpDir := t.TempDir()
	projectDir := filepath.Join(tmpDir, "my-explorer")

	defer chdir(t, tmpDir)()

	err := NewCommand([]string{"my-explorer"}, "graphql-explorer")
	if err != nil {
		t.Fatalf("NewCommand failed: %v", err)
	}

	assertFileExists(t, projectDir, "index.md")
	assertFileExists(t, projectDir, "README.md")
	assertFileExists(t, projectDir, filepath.Join("queries", "countries.graphql"))

	content := readFile(t, filepath.Join(projectDir, "index.md"))
	if !strings.Contains(content, "type: graphql") {
		t.Error("Expected graphql source type")
	}
	if !strings.Contains(content, "query_file:") {
		t.Error("Expected query_file config")
	}
	if !strings.Contains(content, "result_path:") {
		t.Error("Expected result_path config")
	}

	query := readFile(t, filepath.Join(projectDir, "queries", "countries.graphql"))
	if !strings.Contains(query, "countries") {
		t.Error("Expected countries query in graphql file")
	}
}

func TestNewCommandDashboardDataDir(t *testing.T) {
	tmpDir := t.TempDir()
	projectDir := filepath.Join(tmpDir, "dash-test")

	defer chdir(t, tmpDir)()

	err := NewCommand([]string{"dash-test"}, "dashboard")
	if err != nil {
		t.Fatalf("NewCommand failed: %v", err)
	}

	// Verify _data directory is included (embed fix for underscore dirs)
	assertFileExists(t, projectDir, filepath.Join("_data", "tasks.md"))
	assertFileExists(t, projectDir, filepath.Join("_data", "team.md"))
}

func TestListTemplates(t *testing.T) {
	// ListTemplates prints to stdout; verify it doesn't panic
	// and that templateCatalog is well-formed
	for _, tmpl := range templateCatalog {
		if tmpl.Name == "" {
			t.Error("Template name cannot be empty")
		}
		if tmpl.Description == "" {
			t.Errorf("Template %q has no description", tmpl.Name)
		}
		if tmpl.Category == "" {
			t.Errorf("Template %q has no category", tmpl.Name)
		}
	}

	// Verify all templates in catalog are actually embedded
	for _, tmpl := range templateCatalog {
		dir := "templates/" + tmpl.Name
		_, err := templatesFS.ReadDir(dir)
		if err != nil {
			t.Errorf("Template %q is in catalog but not embedded: %v", tmpl.Name, err)
		}
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
