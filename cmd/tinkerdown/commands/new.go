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

//go:embed templates/*
var templatesFS embed.FS

var validTemplates = []string{
	"basic",
	"tutorial",
	"todo",
	"dashboard",
	"form",
	"api-explorer",
	"wasm-source",
}

// NewCommand implements the new command.
func NewCommand(args []string, templateName string) error {
	if len(args) < 1 {
		return fmt.Errorf("project name required\n\nUsage: tinkerdown new <project-name> [--template=<name>]\n\nAvailable templates: %s", strings.Join(validTemplates, ", "))
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
		return fmt.Errorf("unknown template '%s'\n\nAvailable templates: %s", templateName, strings.Join(validTemplates, ", "))
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

func isValidTemplate(name string) bool {
	for _, t := range validTemplates {
		if t == name {
			return true
		}
	}
	return false
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
