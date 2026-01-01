// Command tinkerdown is the CLI tool for creating and serving interactive documentation.
package main

import (
	"fmt"
	"os"

	"github.com/livetemplate/tinkerdown/cmd/tinkerdown/commands"
)

const version = "0.1.0-dev"

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	command := os.Args[1]
	args := os.Args[2:]

	var err error
	switch command {
	case "serve":
		err = commands.ServeCommand(args)
	case "validate":
		err = commands.ValidateCommand(args)
	case "fix":
		err = commands.FixCommand(args)
	case "new":
		err = commands.NewCommand(args)
	case "blocks":
		err = commands.BlocksCommand(args)
	case "version":
		fmt.Printf("tinkerdown version %s\n", version)
	case "help", "-h", "--help":
		printUsage()
	default:
		fmt.Printf("Unknown command: %s\n\n", command)
		printUsage()
		os.Exit(1)
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println("tinkerdown - Interactive documentation made easy")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  tinkerdown serve [directory]     Start development server")
	fmt.Println("  tinkerdown validate [directory]  Validate markdown files")
	fmt.Println("  tinkerdown fix [directory]       Auto-fix common issues")
	fmt.Println("  tinkerdown blocks [directory]    Inspect code blocks")
	fmt.Println("  tinkerdown new <name> [--template=TYPE]  Create new project")
	fmt.Println("  tinkerdown version               Show version")
	fmt.Println("  tinkerdown help                  Show this help")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  tinkerdown serve                 # Serve current directory")
	fmt.Println("  tinkerdown serve ./tutorials     # Serve tutorials directory")
	fmt.Println("  tinkerdown serve --watch         # Serve with live reload")
	fmt.Println("  tinkerdown validate              # Validate current directory")
	fmt.Println("  tinkerdown validate examples/    # Validate specific directory")
	fmt.Println("  tinkerdown fix                   # Auto-fix issues in current directory")
	fmt.Println("  tinkerdown fix --dry-run         # Preview fixes without applying")
	fmt.Println("  tinkerdown blocks examples/      # Inspect blocks in examples/")
	fmt.Println("  tinkerdown blocks . --verbose    # Show detailed block info")
	fmt.Println("  tinkerdown new my-app                  # Create new app (basic template)")
	fmt.Println("  tinkerdown new my-app --template=todo  # Create with todo template")
	fmt.Println("  tinkerdown new --list             # List available templates")
	fmt.Println()
	fmt.Println("Documentation: https://github.com/livetemplate/tinkerdown")
}
