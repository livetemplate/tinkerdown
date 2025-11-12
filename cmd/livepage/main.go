// Command livepage is the CLI tool for creating and serving interactive documentation.
package main

import (
	"fmt"
	"os"
)

const version = "0.1.0-dev"

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	command := os.Args[1]

	switch command {
	case "serve":
		fmt.Println("livepage serve: Not yet implemented")
		fmt.Println("See PROGRESS.md for implementation status")
		os.Exit(1)
	case "new":
		fmt.Println("livepage new: Not yet implemented")
		fmt.Println("See PROGRESS.md for implementation status")
		os.Exit(1)
	case "version":
		fmt.Printf("livepage version %s\n", version)
	case "help", "-h", "--help":
		printUsage()
	default:
		fmt.Printf("Unknown command: %s\n\n", command)
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println("livepage - Interactive documentation made easy")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  livepage serve [directory]   Start development server")
	fmt.Println("  livepage new <name>          Create new tutorial")
	fmt.Println("  livepage version             Show version")
	fmt.Println("  livepage help                Show this help")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  livepage serve               # Serve current directory")
	fmt.Println("  livepage serve ./tutorials   # Serve tutorials directory")
	fmt.Println("  livepage new my-tutorial     # Create new tutorial")
	fmt.Println()
	fmt.Println("Documentation: https://github.com/livetemplate/livepage")
}
