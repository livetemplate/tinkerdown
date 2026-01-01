package commands

import (
	"embed"
	"flag"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"text/template"
)

//go:embed all:templates
var templatesFS embed.FS

// validTemplates lists all available template types
var validTemplates = []string{
	"basic",
	"todo",
	"dashboard",
	"form",
	"api-explorer",
	"cli-wrapper",
	"wasm-source",
}

// templateDescriptions provides help text for each template
var templateDescriptions = map[string]string{
	"basic":        "Minimal starter with interactive counter example",
	"todo":         "Task list with SQLite CRUD operations",
	"dashboard":    "Multi-source data display with tables and stats",
	"form":         "Contact form with validation and persistence",
	"api-explorer": "REST API explorer with live data refresh",
	"cli-wrapper":  "Wrap CLI tools with an interactive web form",
	"wasm-source":  "Scaffold for building custom WASM data sources",
}

// NewCommand implements the new command.
func NewCommand(args []string) error {
	flagSet := flag.NewFlagSet("new", flag.ContinueOnError)
	templateName := flagSet.String("template", "basic", "Template type: basic, todo, dashboard, form, api-explorer, cli-wrapper, wasm-source")
	showList := flagSet.Bool("list", false, "List available templates")

	// Custom usage
	flagSet.Usage = func() {
		fmt.Println("Usage: tinkerdown new [options] <project-name>")
		fmt.Println()
		fmt.Println("Create a new tinkerdown project from a template.")
		fmt.Println()
		fmt.Println("Options:")
		flagSet.PrintDefaults()
		fmt.Println()
		fmt.Println("Templates:")
		for _, t := range validTemplates {
			fmt.Printf("  %-14s %s\n", t, templateDescriptions[t])
		}
		fmt.Println()
		fmt.Println("Examples:")
		fmt.Println("  tinkerdown new my-app                    # Create with basic template")
		fmt.Println("  tinkerdown new my-app --template=todo    # Create with todo template")
		fmt.Println("  tinkerdown new --list                    # List available templates")
	}

	if err := flagSet.Parse(args); err != nil {
		return err
	}

	// Handle --list flag
	if *showList {
		fmt.Println("Available templates:")
		fmt.Println()
		for _, t := range validTemplates {
			fmt.Printf("  %-14s %s\n", t, templateDescriptions[t])
		}
		return nil
	}

	// Get project name from remaining args
	remainingArgs := flagSet.Args()
	if len(remainingArgs) < 1 {
		return fmt.Errorf("project name required\n\nUsage: tinkerdown new [options] <project-name>\n\nRun 'tinkerdown new --help' for more information")
	}

	projectName := remainingArgs[0]

	// Validate template name
	if !isValidTemplate(*templateName) {
		return fmt.Errorf("unknown template: %s\n\nAvailable templates: %s", *templateName, strings.Join(validTemplates, ", "))
	}

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

	// Create the project
	if err := createProject(projectName, *templateName); err != nil {
		return err
	}

	return nil
}

// isValidTemplate checks if a template name is valid
func isValidTemplate(name string) bool {
	for _, t := range validTemplates {
		if t == name {
			return true
		}
	}
	return false
}

// createProject creates a new project from a template
func createProject(projectName, templateName string) error {
	// Template data available in all templates
	data := map[string]string{
		"Title":        toTitle(projectName),
		"ProjectName":  projectName,
		"TemplateName": templateName,
	}

	// Get the template directory path
	templateDir := fmt.Sprintf("templates/%s", templateName)

	// Walk the template directory and copy all files
	var files []string
	err := fs.WalkDir(templatesFS, templateDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() {
			files = append(files, path)
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("failed to read template directory: %w", err)
	}

	if len(files) == 0 {
		return fmt.Errorf("template '%s' has no files", templateName)
	}

	// Create project directory
	if err := os.MkdirAll(projectName, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Process each template file
	for _, templatePath := range files {
		if err := processTemplateFile(projectName, templateName, templatePath, data); err != nil {
			// Clean up on error
			os.RemoveAll(projectName)
			return err
		}
	}

	// Print success message
	printSuccessMessage(projectName, templateName)
	return nil
}

// processTemplateFile reads a template file and writes it to the project directory
func processTemplateFile(projectName, templateName, templatePath string, data map[string]string) error {
	// Read template content
	content, err := templatesFS.ReadFile(templatePath)
	if err != nil {
		return fmt.Errorf("failed to read template %s: %w", templatePath, err)
	}

	// Calculate the relative path within the template
	templatePrefix := fmt.Sprintf("templates/%s/", templateName)
	relativePath := strings.TrimPrefix(templatePath, templatePrefix)

	// Determine output path
	outputPath := filepath.Join(projectName, relativePath)

	// Create parent directories if needed
	outputDir := filepath.Dir(outputPath)
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", outputDir, err)
	}

	// Check if this is a binary file (like .wasm) - don't template-process these
	if isBinaryFile(templatePath) {
		return os.WriteFile(outputPath, content, 0644)
	}

	// Use custom delimiters to avoid conflicts with tinkerdown template syntax
	// We use [[.Var]] instead of {{.Var}} for scaffolding variables
	tmpl, err := template.New(filepath.Base(templatePath)).Delims("[[", "]]").Parse(string(content))
	if err != nil {
		return fmt.Errorf("failed to parse template %s: %w", templatePath, err)
	}

	// Create output file
	f, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create file %s: %w", outputPath, err)
	}
	defer f.Close()

	// Execute template
	if err := tmpl.Execute(f, data); err != nil {
		return fmt.Errorf("failed to write template %s: %w", templatePath, err)
	}

	return nil
}

// isBinaryFile checks if a file should be copied as-is without template processing.
// This list is scoped to file types that may appear in our bundled templates,
// not a comprehensive list of all binary formats.
func isBinaryFile(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	binaryExts := []string{".wasm", ".png", ".jpg", ".jpeg", ".gif", ".ico", ".db", ".sqlite"}
	for _, e := range binaryExts {
		if ext == e {
			return true
		}
	}
	return false
}

// printSuccessMessage displays the project creation success message
func printSuccessMessage(projectName, templateName string) {
	fmt.Printf("✨ Created new %s project: %s\n\n", templateName, projectName)
	fmt.Printf("📁 Project created from '%s' template\n\n", templateName)
	fmt.Printf("🚀 Next steps:\n")
	fmt.Printf("   cd %s\n", projectName)
	fmt.Printf("   tinkerdown serve\n\n")
	fmt.Printf("📚 Your app will be available at http://localhost:8080\n")
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
