// Command livepage is the CLI tool for creating and serving interactive documentation.
package main

import (
	"fmt"
	"os"

	"github.com/livetemplate/livepage/cmd/livepage/commands"
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
	case "new":
		err = commands.NewCommand(args)
	case "version":
		fmt.Printf("livepage version %s\n", version)
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
	fmt.Println("livepage - Interactive documentation made easy")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  livepage serve [directory]     Start development server")
	fmt.Println("  livepage validate [directory]  Validate markdown files")
	fmt.Println("  livepage new <name>            Create new tutorial")
	fmt.Println("  livepage version               Show version")
	fmt.Println("  livepage help                  Show this help")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  livepage serve                 # Serve current directory")
	fmt.Println("  livepage serve ./tutorials     # Serve tutorials directory")
	fmt.Println("  livepage serve --watch         # Serve with live reload")
	fmt.Println("  livepage validate              # Validate current directory")
	fmt.Println("  livepage validate examples/    # Validate specific directory")
	fmt.Println("  livepage new my-tutorial       # Create new tutorial")
	fmt.Println()
	fmt.Println("Documentation: https://github.com/livetemplate/livepage")
}
