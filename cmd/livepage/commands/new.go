package commands

import (
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"
)

//go:embed templates/*
var templatesFS embed.FS

// NewCommand implements the new command.
func NewCommand(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("project name required\n\nUsage: livepage new <project-name>")
	}

	projectName := args[0]

	// Validate project name
	if projectName == "" {
		return fmt.Errorf("project name cannot be empty")
	}
	if strings.Contains(projectName, " ") {
		return fmt.Errorf("project name cannot contain spaces")
	}

	// Check if directory already exists
	if _, err := os.Stat(projectName); !os.IsNotExist(err) {
		return fmt.Errorf("directory '%s' already exists", projectName)
	}

	// Create project directory
	if err := os.MkdirAll(projectName, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Template data
	data := map[string]string{
		"Title":       toTitle(projectName),
		"ProjectName": projectName,
	}

	// Get list of template files
	templateFiles := []string{
		"templates/basic/index.md",
		"templates/basic/README.md",
	}

	// Process each template file
	for _, templatePath := range templateFiles {
		// Read template content
		content, err := templatesFS.ReadFile(templatePath)
		if err != nil {
			// Clean up on error
			os.RemoveAll(projectName)
			return fmt.Errorf("failed to read template %s: %w", templatePath, err)
		}

		// Parse and execute template
		tmpl, err := template.New("file").Parse(string(content))
		if err != nil {
			os.RemoveAll(projectName)
			return fmt.Errorf("failed to parse template %s: %w", templatePath, err)
		}

		// Get output file name (strip "templates/basic/" prefix)
		outputFile := filepath.Join(projectName, filepath.Base(templatePath))

		// Create output file
		f, err := os.Create(outputFile)
		if err != nil {
			os.RemoveAll(projectName)
			return fmt.Errorf("failed to create file %s: %w", outputFile, err)
		}

		// Execute template
		if err := tmpl.Execute(f, data); err != nil {
			f.Close()
			os.RemoveAll(projectName)
			return fmt.Errorf("failed to write template %s: %w", templatePath, err)
		}
		f.Close()
	}

	// Success message
	fmt.Printf("✨ Created new tutorial: %s\n\n", projectName)
	fmt.Printf("📁 Project structure:\n")
	fmt.Printf("   %s/\n", projectName)
	fmt.Printf("   ├── index.md\n")
	fmt.Printf("   └── README.md\n\n")
	fmt.Printf("🚀 Next steps:\n")
	fmt.Printf("   cd %s\n", projectName)
	fmt.Printf("   livepage serve\n\n")
	fmt.Printf("📚 Your tutorial will be available at http://localhost:8080\n")

	return nil
}

// toTitle converts a project name to a title case string
// Example: "my-tutorial" -> "My Tutorial"
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
