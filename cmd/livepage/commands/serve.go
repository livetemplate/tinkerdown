package commands

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/livetemplate/livepage/internal/server"
)

// ServeCommand implements the serve command.
func ServeCommand(args []string) error {
	// Parse arguments
	dir := "."
	port := "8080"
	host := "localhost"

	if len(args) > 0 {
		dir = args[0]
	}

	// Check if directory exists
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return fmt.Errorf("directory does not exist: %s", dir)
	}

	// Get absolute path
	absDir, err := filepath.Abs(dir)
	if err != nil {
		return fmt.Errorf("failed to get absolute path: %w", err)
	}

	fmt.Printf("📚 Livepage Development Server\n\n")
	fmt.Printf("Serving: %s\n", absDir)

	// Create server
	srv := server.New(absDir)

	// Discover pages
	if err := srv.Discover(); err != nil {
		return fmt.Errorf("failed to discover pages: %w", err)
	}

	// Print discovered pages
	fmt.Printf("\nPages discovered:\n")
	for _, route := range srv.Routes() {
		fmt.Printf("  %-30s %s\n", route.Pattern, route.FilePath)
	}

	// Start server
	addr := fmt.Sprintf("%s:%s", host, port)
	fmt.Printf("\n🌐 Server running at http://%s\n", addr)
	fmt.Printf("Press Ctrl+C to stop\n\n")

	if err := http.ListenAndServe(addr, srv); err != nil {
		return fmt.Errorf("server error: %w", err)
	}

	return nil
}

func init() {
	log.SetFlags(0) // Remove timestamp from logs
}
