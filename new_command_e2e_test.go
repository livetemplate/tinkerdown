package tinkerdown_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// getProjectRoot returns the project root directory based on this test file's location
func getProjectRoot() string {
	_, filename, _, _ := runtime.Caller(0)
	return filepath.Dir(filename)
}

// buildTinkerdown builds the tinkerdown binary for testing
func buildTinkerdown(t *testing.T, outputDir string) string {
	t.Helper()
	binPath := filepath.Join(outputDir, "tinkerdown")
	buildCmd := exec.Command("go", "build", "-o", binPath, "./cmd/tinkerdown")
	buildCmd.Dir = getProjectRoot()
	buildCmd.Env = append(os.Environ(), "GOWORK=off")
	output, err := buildCmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to build tinkerdown: %v\nOutput: %s", err, output)
	}
	return binPath
}

// TestNewCommandBasicTemplate tests the default basic template works
func TestNewCommandBasicTemplate(t *testing.T) {
	tmpDir := t.TempDir()
	binPath := buildTinkerdown(t, tmpDir)

	projectName := "test-basic"
	projectPath := filepath.Join(tmpDir, projectName)

	// Run tinkerdown new (default basic template)
	cmd := exec.Command(binPath, "new", projectPath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to run tinkerdown new: %v\nOutput: %s", err, output)
	}

	// Verify expected files exist
	expectedFiles := []string{"index.md", "get-pods.sh", "README.md"}
	for _, f := range expectedFiles {
		path := filepath.Join(projectPath, f)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Errorf("Expected file %s does not exist", f)
		}
	}

	// Verify get-pods.sh is executable
	info, err := os.Stat(filepath.Join(projectPath, "get-pods.sh"))
	if err != nil {
		t.Errorf("Failed to stat get-pods.sh: %v", err)
	} else if info.Mode()&0111 == 0 {
		t.Error("get-pods.sh is not executable")
	}

	// Verify title substitution in index.md
	content, err := os.ReadFile(filepath.Join(projectPath, "index.md"))
	if err != nil {
		t.Fatalf("Failed to read index.md: %v", err)
	}
	if !strings.Contains(string(content), "Test Basic") {
		t.Error("Title 'Test Basic' not found in index.md")
	}

	// Verify title substitution in README.md
	readme, err := os.ReadFile(filepath.Join(projectPath, "README.md"))
	if err != nil {
		t.Fatalf("Failed to read README.md: %v", err)
	}
	if !strings.Contains(string(readme), "Test Basic") {
		t.Error("Title 'Test Basic' not found in README.md")
	}
	// Verify ProjectName substitution
	if !strings.Contains(string(readme), projectName) {
		t.Error("ProjectName not found in README.md")
	}

	// Verify output message
	if !strings.Contains(string(output), "Created new app") {
		t.Error("Success message not found in output")
	}
}

// TestNewCommandTodoTemplate tests the todo template with --template=todo
func TestNewCommandTodoTemplate(t *testing.T) {
	tmpDir := t.TempDir()
	binPath := buildTinkerdown(t, tmpDir)

	projectName := "my-todo-app"
	projectPath := filepath.Join(tmpDir, projectName)

	// Run tinkerdown new with todo template
	cmd := exec.Command(binPath, "new", projectPath, "--template=todo")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to run tinkerdown new --template=todo: %v\nOutput: %s", err, output)
	}

	// Verify expected files exist
	expectedFiles := []string{"index.md", "README.md"}
	for _, f := range expectedFiles {
		path := filepath.Join(projectPath, f)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Errorf("Expected file %s does not exist", f)
		}
	}

	// Verify index.md contains SQLite source configuration
	content, err := os.ReadFile(filepath.Join(projectPath, "index.md"))
	if err != nil {
		t.Fatalf("Failed to read index.md: %v", err)
	}
	contentStr := string(content)

	if !strings.Contains(contentStr, "type: sqlite") {
		t.Error("SQLite source type not found in index.md")
	}
	if !strings.Contains(contentStr, "tasks.db") {
		t.Error("tasks.db database not found in index.md")
	}
	if !strings.Contains(contentStr, "My Todo App") {
		t.Error("Title 'My Todo App' not found in index.md")
	}

	// Verify output mentions template
	if !strings.Contains(string(output), "template: todo") {
		t.Error("Template name 'todo' not found in output")
	}
}

// TestNewCommandDashboardTemplate tests the dashboard template
func TestNewCommandDashboardTemplate(t *testing.T) {
	tmpDir := t.TempDir()
	binPath := buildTinkerdown(t, tmpDir)

	projectName := "my-dashboard"
	projectPath := filepath.Join(tmpDir, projectName)

	// Run tinkerdown new with dashboard template
	cmd := exec.Command(binPath, "new", projectPath, "--template=dashboard")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to run tinkerdown new --template=dashboard: %v\nOutput: %s", err, output)
	}

	// Verify expected files exist
	expectedFiles := []string{"index.md", "README.md", "system-info.sh"}
	for _, f := range expectedFiles {
		path := filepath.Join(projectPath, f)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Errorf("Expected file %s does not exist", f)
		}
	}

	// Verify system-info.sh is executable
	info, err := os.Stat(filepath.Join(projectPath, "system-info.sh"))
	if err != nil {
		t.Errorf("Failed to stat system-info.sh: %v", err)
	} else if info.Mode()&0111 == 0 {
		t.Error("system-info.sh is not executable")
	}

	// Verify index.md contains REST and exec sources
	content, err := os.ReadFile(filepath.Join(projectPath, "index.md"))
	if err != nil {
		t.Fatalf("Failed to read index.md: %v", err)
	}
	contentStr := string(content)

	if !strings.Contains(contentStr, "type: rest") {
		t.Error("REST source type not found in index.md")
	}
	if !strings.Contains(contentStr, "type: exec") {
		t.Error("Exec source type not found in index.md")
	}
	if !strings.Contains(contentStr, "My Dashboard") {
		t.Error("Title 'My Dashboard' not found in index.md")
	}
}

// TestNewCommandFormTemplate tests the form template
func TestNewCommandFormTemplate(t *testing.T) {
	tmpDir := t.TempDir()
	binPath := buildTinkerdown(t, tmpDir)

	projectName := "contact-form"
	projectPath := filepath.Join(tmpDir, projectName)

	// Run tinkerdown new with form template
	cmd := exec.Command(binPath, "new", projectPath, "--template=form")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to run tinkerdown new --template=form: %v\nOutput: %s", err, output)
	}

	// Verify expected files exist
	expectedFiles := []string{"index.md", "README.md"}
	for _, f := range expectedFiles {
		path := filepath.Join(projectPath, f)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Errorf("Expected file %s does not exist", f)
		}
	}

	// Verify index.md contains form submission structure
	content, err := os.ReadFile(filepath.Join(projectPath, "index.md"))
	if err != nil {
		t.Fatalf("Failed to read index.md: %v", err)
	}
	contentStr := string(content)

	if !strings.Contains(contentStr, "submissions.db") {
		t.Error("submissions.db database not found in index.md")
	}
	if !strings.Contains(contentStr, "lvt-submit") {
		t.Error("lvt-submit attribute not found in index.md")
	}
	if !strings.Contains(contentStr, "Contact Form") {
		t.Error("Title 'Contact Form' not found in index.md")
	}
}

// TestNewCommandApiExplorerTemplate tests the api-explorer template
func TestNewCommandApiExplorerTemplate(t *testing.T) {
	tmpDir := t.TempDir()
	binPath := buildTinkerdown(t, tmpDir)

	projectName := "repo-search"
	projectPath := filepath.Join(tmpDir, projectName)

	// Run tinkerdown new with api-explorer template
	cmd := exec.Command(binPath, "new", projectPath, "--template=api-explorer")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to run tinkerdown new --template=api-explorer: %v\nOutput: %s", err, output)
	}

	// Verify expected files exist
	expectedFiles := []string{"index.md", "README.md"}
	for _, f := range expectedFiles {
		path := filepath.Join(projectPath, f)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Errorf("Expected file %s does not exist", f)
		}
	}

	// Verify index.md contains REST source with GitHub API
	content, err := os.ReadFile(filepath.Join(projectPath, "index.md"))
	if err != nil {
		t.Fatalf("Failed to read index.md: %v", err)
	}
	contentStr := string(content)

	if !strings.Contains(contentStr, "type: rest") {
		t.Error("REST source type not found in index.md")
	}
	if !strings.Contains(contentStr, "api.github.com") {
		t.Error("GitHub API URL not found in index.md")
	}
	if !strings.Contains(contentStr, "persist: localstorage") {
		t.Error("LocalStorage persistence not found in index.md")
	}
	if !strings.Contains(contentStr, "Repo Search") {
		t.Error("Title 'Repo Search' not found in index.md")
	}
}

// TestNewCommandWasmSourceTemplate tests the wasm-source template with nested directories
func TestNewCommandWasmSourceTemplate(t *testing.T) {
	tmpDir := t.TempDir()
	binPath := buildTinkerdown(t, tmpDir)

	projectName := "my-wasm-source"
	projectPath := filepath.Join(tmpDir, projectName)

	// Run tinkerdown new with wasm-source template
	cmd := exec.Command(binPath, "new", projectPath, "--template=wasm-source")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to run tinkerdown new --template=wasm-source: %v\nOutput: %s", err, output)
	}

	// Verify expected files exist including nested directory structure
	expectedFiles := []string{
		"README.md",
		"source.go",
		"Makefile",
		"test-app/index.md",
		"test-app/tinkerdown.yaml",
	}
	for _, f := range expectedFiles {
		path := filepath.Join(projectPath, f)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Errorf("Expected file %s does not exist", f)
		}
	}

	// Verify nested test-app directory was created
	testAppDir := filepath.Join(projectPath, "test-app")
	info, err := os.Stat(testAppDir)
	if err != nil {
		t.Errorf("test-app directory does not exist: %v", err)
	} else if !info.IsDir() {
		t.Error("test-app is not a directory")
	}

	// Verify README.md title substitution
	readme, err := os.ReadFile(filepath.Join(projectPath, "README.md"))
	if err != nil {
		t.Fatalf("Failed to read README.md: %v", err)
	}
	if !strings.Contains(string(readme), "My Wasm Source") {
		t.Error("Title 'My Wasm Source' not found in README.md")
	}
}

// TestNewCommandTutorialTemplate tests the tutorial template
func TestNewCommandTutorialTemplate(t *testing.T) {
	tmpDir := t.TempDir()
	binPath := buildTinkerdown(t, tmpDir)

	projectName := "learn-livetemplate"
	projectPath := filepath.Join(tmpDir, projectName)

	// Run tinkerdown new with tutorial template
	cmd := exec.Command(binPath, "new", projectPath, "--template=tutorial")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to run tinkerdown new --template=tutorial: %v\nOutput: %s", err, output)
	}

	// Verify expected files exist
	expectedFiles := []string{"index.md", "README.md"}
	for _, f := range expectedFiles {
		path := filepath.Join(projectPath, f)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Errorf("Expected file %s does not exist", f)
		}
	}

	// Verify index.md contains tutorial structure
	content, err := os.ReadFile(filepath.Join(projectPath, "index.md"))
	if err != nil {
		t.Fatalf("Failed to read index.md: %v", err)
	}
	contentStr := string(content)

	if !strings.Contains(contentStr, "type: tutorial") {
		t.Error("Tutorial type not found in index.md")
	}
	if !strings.Contains(contentStr, "steps: 3") {
		t.Error("Tutorial steps config not found in index.md")
	}
	if !strings.Contains(contentStr, "Learn Livetemplate") {
		t.Error("Title 'Learn Livetemplate' not found in index.md")
	}
	if !strings.Contains(contentStr, "persist: localstorage") {
		t.Error("LocalStorage persistence not found in index.md")
	}
}

// TestNewCommandInvalidTemplate tests error on invalid template name
func TestNewCommandInvalidTemplate(t *testing.T) {
	tmpDir := t.TempDir()
	binPath := buildTinkerdown(t, tmpDir)

	projectName := "test-invalid"
	projectPath := filepath.Join(tmpDir, projectName)

	// Run tinkerdown new with invalid template
	cmd := exec.Command(binPath, "new", projectPath, "--template=nonexistent")
	output, err := cmd.CombinedOutput()

	// Should fail with error
	if err == nil {
		t.Fatal("Expected error for invalid template, but command succeeded")
	}

	// Verify error message mentions unknown template
	if !strings.Contains(string(output), "unknown template") {
		t.Errorf("Error message doesn't mention 'unknown template': %s", output)
	}

	// Verify error message lists available templates
	if !strings.Contains(string(output), "basic") {
		t.Errorf("Error message doesn't list available templates: %s", output)
	}

	// Verify project directory was NOT created
	if _, err := os.Stat(projectPath); !os.IsNotExist(err) {
		t.Error("Project directory should not exist after failed template")
	}
}

// TestNewCommandShortFlag tests that -t flag works same as --template
func TestNewCommandShortFlag(t *testing.T) {
	tmpDir := t.TempDir()
	binPath := buildTinkerdown(t, tmpDir)

	projectName := "short-flag-test"
	projectPath := filepath.Join(tmpDir, projectName)

	// Run tinkerdown new with -t flag (equals format)
	cmd := exec.Command(binPath, "new", projectPath, "-t=todo")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to run tinkerdown new -t=todo: %v\nOutput: %s", err, output)
	}

	// Verify todo template was used by checking for SQLite config
	content, err := os.ReadFile(filepath.Join(projectPath, "index.md"))
	if err != nil {
		t.Fatalf("Failed to read index.md: %v", err)
	}
	if !strings.Contains(string(content), "type: sqlite") {
		t.Error("SQLite source not found - wrong template may have been used")
	}

	// Verify output mentions template
	if !strings.Contains(string(output), "template: todo") {
		t.Error("Template name 'todo' not found in output")
	}
}

// TestNewCommandSpaceSeparatedFlag tests -t todo (space separated) format
func TestNewCommandSpaceSeparatedFlag(t *testing.T) {
	tmpDir := t.TempDir()
	binPath := buildTinkerdown(t, tmpDir)

	projectName := "space-flag-test"
	projectPath := filepath.Join(tmpDir, projectName)

	// Run tinkerdown new with -t flag (space separated)
	cmd := exec.Command(binPath, "new", projectPath, "-t", "dashboard")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to run tinkerdown new -t dashboard: %v\nOutput: %s", err, output)
	}

	// Verify dashboard template was used by checking for system-info.sh
	if _, err := os.Stat(filepath.Join(projectPath, "system-info.sh")); os.IsNotExist(err) {
		t.Error("system-info.sh not found - wrong template may have been used")
	}

	// Verify output mentions dashboard template
	if !strings.Contains(string(output), "template: dashboard") {
		t.Error("Template name 'dashboard' not found in output")
	}
}

// TestNewCommandSpaceSeparatedTemplateLongFlag tests --template todo (space separated) format
func TestNewCommandSpaceSeparatedTemplateLongFlag(t *testing.T) {
	tmpDir := t.TempDir()
	binPath := buildTinkerdown(t, tmpDir)

	projectName := "long-space-test"
	projectPath := filepath.Join(tmpDir, projectName)

	// Run tinkerdown new with --template flag (space separated)
	cmd := exec.Command(binPath, "new", projectPath, "--template", "form")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to run tinkerdown new --template form: %v\nOutput: %s", err, output)
	}

	// Verify form template was used
	content, err := os.ReadFile(filepath.Join(projectPath, "index.md"))
	if err != nil {
		t.Fatalf("Failed to read index.md: %v", err)
	}
	if !strings.Contains(string(content), "submissions.db") {
		t.Error("submissions.db not found - wrong template may have been used")
	}
}

// TestNewCommandDirectoryAlreadyExists tests error when directory exists
func TestNewCommandDirectoryAlreadyExists(t *testing.T) {
	tmpDir := t.TempDir()
	binPath := buildTinkerdown(t, tmpDir)

	projectName := "existing-dir"
	projectPath := filepath.Join(tmpDir, projectName)

	// Create the directory first
	if err := os.MkdirAll(projectPath, 0755); err != nil {
		t.Fatalf("Failed to create directory: %v", err)
	}

	// Run tinkerdown new
	cmd := exec.Command(binPath, "new", projectPath)
	output, err := cmd.CombinedOutput()

	// Should fail with error
	if err == nil {
		t.Fatal("Expected error for existing directory, but command succeeded")
	}

	// Verify error message mentions existing directory
	if !strings.Contains(string(output), "already exists") {
		t.Errorf("Error message doesn't mention 'already exists': %s", output)
	}
}

// TestNewCommandEmptyProjectName tests error when no project name is provided
func TestNewCommandEmptyProjectName(t *testing.T) {
	tmpDir := t.TempDir()
	binPath := buildTinkerdown(t, tmpDir)

	// Run tinkerdown new without project name
	cmd := exec.Command(binPath, "new")
	output, err := cmd.CombinedOutput()

	// Should fail with error
	if err == nil {
		t.Fatal("Expected error for missing project name, but command succeeded")
	}

	// Verify error message mentions project name required
	if !strings.Contains(string(output), "project name required") {
		t.Errorf("Error message doesn't mention 'project name required': %s", output)
	}
}

// TestNewCommandProjectNameWithSpaces tests error when project name has spaces
func TestNewCommandProjectNameWithSpaces(t *testing.T) {
	tmpDir := t.TempDir()
	binPath := buildTinkerdown(t, tmpDir)

	// Run tinkerdown new with spaces in name
	cmd := exec.Command(binPath, "new", "my app")
	cmd.Dir = tmpDir
	output, err := cmd.CombinedOutput()

	// Should fail with error
	if err == nil {
		t.Fatal("Expected error for project name with spaces, but command succeeded")
	}

	// Verify error message mentions spaces
	if !strings.Contains(string(output), "cannot contain spaces") {
		t.Errorf("Error message doesn't mention 'cannot contain spaces': %s", output)
	}
}

// TestNewCommandHyphenatedName tests title conversion from hyphenated name
func TestNewCommandHyphenatedName(t *testing.T) {
	tmpDir := t.TempDir()
	binPath := buildTinkerdown(t, tmpDir)

	projectName := "my-awesome-app"
	projectPath := filepath.Join(tmpDir, projectName)

	// Run tinkerdown new
	cmd := exec.Command(binPath, "new", projectPath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to run tinkerdown new: %v\nOutput: %s", err, output)
	}

	// Verify title was properly converted
	content, err := os.ReadFile(filepath.Join(projectPath, "index.md"))
	if err != nil {
		t.Fatalf("Failed to read index.md: %v", err)
	}

	// "my-awesome-app" should become "My Awesome App"
	if !strings.Contains(string(content), "My Awesome App") {
		t.Errorf("Title not properly converted. Expected 'My Awesome App' in content:\n%s", string(content))
	}
}

// TestNewCommandUnderscoreName tests title conversion from underscored name
func TestNewCommandUnderscoreName(t *testing.T) {
	tmpDir := t.TempDir()
	binPath := buildTinkerdown(t, tmpDir)

	projectName := "my_underscore_app"
	projectPath := filepath.Join(tmpDir, projectName)

	// Run tinkerdown new
	cmd := exec.Command(binPath, "new", projectPath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to run tinkerdown new: %v\nOutput: %s", err, output)
	}

	// Verify title was properly converted
	content, err := os.ReadFile(filepath.Join(projectPath, "index.md"))
	if err != nil {
		t.Fatalf("Failed to read index.md: %v", err)
	}

	// "my_underscore_app" should become "My Underscore App"
	if !strings.Contains(string(content), "My Underscore App") {
		t.Errorf("Title not properly converted. Expected 'My Underscore App' in content:\n%s", string(content))
	}
}

// TestNewCommandAllTemplatesListedInError tests that error shows all available templates
func TestNewCommandAllTemplatesListedInError(t *testing.T) {
	tmpDir := t.TempDir()
	binPath := buildTinkerdown(t, tmpDir)

	projectName := "test-listing"
	projectPath := filepath.Join(tmpDir, projectName)

	// Run tinkerdown new with invalid template to see template list
	cmd := exec.Command(binPath, "new", projectPath, "--template=invalid")
	output, err := cmd.CombinedOutput()

	if err == nil {
		t.Fatal("Expected error for invalid template")
	}

	outputStr := string(output)
	expectedTemplates := []string{"basic", "tutorial", "todo", "dashboard", "form", "api-explorer", "wasm-source"}
	for _, tmpl := range expectedTemplates {
		if !strings.Contains(outputStr, tmpl) {
			t.Errorf("Template '%s' not listed in error message: %s", tmpl, outputStr)
		}
	}
}

// TestNewCommandEmptyTemplateValue tests that empty --template= value uses default
func TestNewCommandEmptyTemplateValue(t *testing.T) {
	tmpDir := t.TempDir()
	binPath := buildTinkerdown(t, tmpDir)

	projectName := "test-empty-template"
	projectPath := filepath.Join(tmpDir, projectName)

	// Run tinkerdown new with empty template value --template=
	cmd := exec.Command(binPath, "new", projectPath, "--template=")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to run tinkerdown new with empty --template=: %v\nOutput: %s", err, output)
	}

	// Should use default (basic) template - check index.md exists
	indexPath := filepath.Join(projectPath, "index.md")
	if _, err := os.Stat(indexPath); os.IsNotExist(err) {
		t.Error("index.md should exist when using default template")
	}
}

// TestNewCommandMissingTemplateValueAtEnd tests -t at end of args with no value
func TestNewCommandMissingTemplateValueAtEnd(t *testing.T) {
	tmpDir := t.TempDir()
	binPath := buildTinkerdown(t, tmpDir)

	projectName := "test-missing-template"
	projectPath := filepath.Join(tmpDir, projectName)

	// Run tinkerdown new with -t at the end (no value after it)
	cmd := exec.Command(binPath, "new", projectPath, "-t")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to run tinkerdown new with trailing -t: %v\nOutput: %s", err, output)
	}

	// Should use default (basic) template - check index.md exists
	indexPath := filepath.Join(projectPath, "index.md")
	if _, err := os.Stat(indexPath); os.IsNotExist(err) {
		t.Error("index.md should exist when using default template")
	}
}
