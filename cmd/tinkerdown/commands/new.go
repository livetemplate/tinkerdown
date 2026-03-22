package commands

import (
	"embed"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"text/template"
)

//go:embed all:templates
var templatesFS embed.FS

type templateInfo struct {
	Name        string
	Description string
	Category    string
}

var templateCatalog = []templateInfo{
	// Getting Started
	{"basic", "Kubernetes pods dashboard (exec source)", "Getting Started"},
	{"tutorial", "Interactive LiveTemplate tutorial (server state)", "Getting Started"},
	// Data Sources
	{"todo", "SQLite task manager with CRUD operations", "Data Sources"},
	{"csv-inventory", "Product inventory from CSV file", "Data Sources"},
	{"json-dashboard", "Metrics dashboard from JSON with computed expressions", "Data Sources"},
	{"markdown-notes", "Notes manager with markdown table storage", "Data Sources"},
	{"graphql-explorer", "Countries browser via GraphQL API", "Data Sources"},
	// Patterns
	{"dashboard", "Multi-source dashboard (REST API + exec)", "Patterns"},
	{"form", "Contact form with SQLite persistence", "Patterns"},
	{"api-explorer", "GitHub repository search (REST API)", "Patterns"},
	{"cli-wrapper", "Web UI wrapper for CLI tools (exec)", "Patterns"},
	// Advanced
	{"wasm-source", "Custom TinyGo WASM data source", "Advanced"},
}

// NewCommand implements the new command.
func NewCommand(args []string, templateName string) error {
	if len(args) < 1 {
		return fmt.Errorf("project name required\n\nUsage: tinkerdown new <project-name> [--template=<name>]\n\nAvailable templates: %s\n\nRun 'tinkerdown new --list' to see all templates with descriptions.", strings.Join(templateNames(), ", "))
	}

	projectName := args[0]

	// Validate project name
	if projectName == "" {
		return fmt.Errorf("project name cannot be empty")
	}
	if strings.Contains(projectName, " ") {
		return fmt.Errorf("project name cannot contain spaces")
	}

	// Default template
	if templateName == "" {
		templateName = "basic"
	}

	// Validate template name
	if !isValidTemplate(templateName) {
		return fmt.Errorf("unknown template '%s'\n\nAvailable templates: %s\n\nRun 'tinkerdown new --list' to see all templates with descriptions.", templateName, strings.Join(templateNames(), ", "))
	}

	// Check if directory already exists
	if _, err := os.Stat(projectName); !os.IsNotExist(err) {
		return fmt.Errorf("directory '%s' already exists", projectName)
	}

	// Create project directory
	if err := os.MkdirAll(projectName, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Template data - use base name for title, not full path
	baseName := filepath.Base(projectName)
	data := map[string]string{
		"Title":       toTitle(baseName),
		"ProjectName": baseName,
	}

	// Process template files
	templateDir := "templates/" + templateName
	err := fs.WalkDir(templatesFS, templateDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Calculate relative path from template root
		relPath, err := filepath.Rel(templateDir, path)
		if err != nil {
			return err
		}

		// Skip the root directory itself
		if relPath == "." {
			return nil
		}

		// Target path
		targetPath := filepath.Join(projectName, relPath)

		if d.IsDir() {
			// Create directory
			return os.MkdirAll(targetPath, 0755)
		}

		// Read file content
		content, err := templatesFS.ReadFile(path)
		if err != nil {
			return fmt.Errorf("failed to read %s: %w", path, err)
		}

		// Determine if file should be processed as template
		ext := filepath.Ext(path)
		if ext == ".md" || ext == ".yaml" || ext == ".sh" {
			// Process as Go template with custom delimiters to avoid conflicts
			// with Tinkerdown runtime templates that use {{ }}
			tmpl, err := template.New(filepath.Base(path)).Delims("<<", ">>").Parse(string(content))
			if err != nil {
				return fmt.Errorf("failed to parse template %s: %w", path, err)
			}

			f, err := os.Create(targetPath)
			if err != nil {
				return fmt.Errorf("failed to create %s: %w", targetPath, err)
			}
			defer f.Close()

			if err := tmpl.Execute(f, data); err != nil {
				return fmt.Errorf("failed to execute template %s: %w", path, err)
			}
		} else {
			// Copy file as-is
			if err := os.WriteFile(targetPath, content, 0644); err != nil {
				return fmt.Errorf("failed to write %s: %w", targetPath, err)
			}
		}

		// Make .sh files executable
		if ext == ".sh" {
			if err := os.Chmod(targetPath, 0755); err != nil {
				return fmt.Errorf("failed to chmod %s: %w", targetPath, err)
			}
		}

		return nil
	})

	if err != nil {
		// Clean up on error
		os.RemoveAll(projectName)
		return fmt.Errorf("failed to create project: %w", err)
	}

	// Success message
	fmt.Printf("✨ Created new app: %s (template: %s)\n\n", projectName, templateName)
	printProjectStructure(projectName)
	fmt.Printf("\n🚀 Next steps:\n")
	fmt.Printf("   cd %s\n", projectName)
	fmt.Printf("   tinkerdown serve\n\n")
	fmt.Printf("📚 Your app will be available at http://localhost:8080\n")

	return nil
}

// ListTemplates prints all available templates grouped by category.
func ListTemplates() {
	fmt.Println("Available templates:")

	// Collect categories in order of first appearance.
	var categories []string
	seen := map[string]bool{}
	for _, t := range templateCatalog {
		if !seen[t.Category] {
			seen[t.Category] = true
			categories = append(categories, t.Category)
		}
	}

	for _, cat := range categories {
		fmt.Printf("\n  %s:\n", cat)
		for _, t := range templateCatalog {
			if t.Category == cat {
				fmt.Printf("    %-20s %s\n", t.Name, t.Description)
			}
		}
	}

	fmt.Println()
	fmt.Println("Usage: tinkerdown new <project-name> --template=<name>")
	fmt.Println("Default template: basic")
}

func isValidTemplate(name string) bool {
	for _, t := range templateCatalog {
		if t.Name == name {
			return true
		}
	}
	return false
}

func templateNames() []string {
	names := make([]string, len(templateCatalog))
	for i, t := range templateCatalog {
		names[i] = t.Name
	}
	return names
}

func printProjectStructure(projectName string) {
	fmt.Printf("📁 Project structure:\n")
	fmt.Printf("   %s/\n", projectName)

	// Walk the created directory and print structure
	filepath.WalkDir(projectName, func(path string, d fs.DirEntry, err error) error {
		if err != nil || path == projectName {
			return nil
		}
		relPath, _ := filepath.Rel(projectName, path)
		depth := strings.Count(relPath, string(os.PathSeparator))
		indent := strings.Repeat("   ", depth)
		prefix := "├── "
		fmt.Printf("   %s%s%s\n", indent, prefix, d.Name())
		return nil
	})
}

// toTitle converts a project name to a title case string
// Example: "my-app" -> "My App"
func toTitle(name string) string {
	// Replace hyphens and underscores with spaces
	name = strings.ReplaceAll(name, "-", " ")
	name = strings.ReplaceAll(name, "_", " ")

	// Title case each word
	words := strings.Fields(name)
	for i, word := range words {
		if len(word) > 0 {
			words[i] = strings.ToUpper(word[:1]) + strings.ToLower(word[1:])
		}
	}

	return strings.Join(words, " ")
}
