// Command tinkerdown is the CLI tool for creating and serving interactive documentation.
package main

import (
	"fmt"
	"os"
	"strings"

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
		// Parse --template flag from args
		templateName := ""
		filteredArgs := make([]string, 0, len(args))
		skipNext := false
		for i, arg := range args {
			if skipNext {
				skipNext = false
				continue
			}
			if strings.HasPrefix(arg, "--template=") {
				templateName = strings.TrimPrefix(arg, "--template=")
			} else if strings.HasPrefix(arg, "-t=") {
				templateName = strings.TrimPrefix(arg, "-t=")
			} else if arg == "--template" || arg == "-t" {
				// Handle space-separated: -t todo or --template todo
				if i+1 < len(args) {
					templateName = args[i+1]
					skipNext = true
				}
			} else {
				filteredArgs = append(filteredArgs, arg)
			}
		}
		err = commands.NewCommand(filteredArgs, templateName)
	case "blocks":
		err = commands.BlocksCommand(args)
	case "cli":
		err = commands.CLICommand(args)
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
	fmt.Println("  tinkerdown new <name>            Create new app from template")
	fmt.Println("  tinkerdown cli <path> <action> <source>  CLI mode for CRUD operations")
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
	fmt.Println("  tinkerdown new my-app            # Create new app (basic template)")
	fmt.Println("  tinkerdown new my-app --template=todo  # Use todo template")
	fmt.Println("  tinkerdown cli app.md list tasks # List items from source")
	fmt.Println("  tinkerdown cli . add tasks --text=\"New task\"  # Add item")
	fmt.Println()
	fmt.Println("Documentation: https://github.com/livetemplate/tinkerdown")
}
