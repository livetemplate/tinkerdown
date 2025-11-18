package commands

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/livetemplate/livepage/internal/config"
	"github.com/livetemplate/livepage/internal/server"
)

// ServeCommand implements the serve command.
func ServeCommand(args []string) error {
	// Parse arguments
	dir := "."
	var configPath string
	var port string
	var host string
	var watch *bool

	// Parse flags
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if arg == "--watch" || arg == "-w" {
			watchVal := true
			watch = &watchVal
		} else if arg == "--port" || arg == "-p" {
			if i+1 < len(args) {
				port = args[i+1]
				i++
			}
		} else if arg == "--host" {
			if i+1 < len(args) {
				host = args[i+1]
				i++
			}
		} else if arg == "--config" || arg == "-c" {
			if i+1 < len(args) {
				configPath = args[i+1]
				i++
			}
		} else if !strings.HasPrefix(arg, "-") {
			// Positional argument (directory)
			dir = arg
		}
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

	// Load configuration
	var cfg *config.Config
	if configPath != "" {
		cfg, err = config.Load(configPath)
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}
		fmt.Printf("📝 Using config: %s\n", configPath)
	} else {
		// Try to load from directory
		cfg, err = config.LoadFromDir(absDir)
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}
	}

	// CLI flags override config
	if port != "" {
		portInt, err := strconv.Atoi(port)
		if err != nil {
			return fmt.Errorf("invalid port: %s", port)
		}
		cfg.Server.Port = portInt
	}
	if host != "" {
		cfg.Server.Host = host
	}
	if watch != nil {
		cfg.Features.HotReload = *watch
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

	// Enable watch mode if requested
	if cfg.Features.HotReload {
		if err := srv.EnableWatch(true); err != nil {
			return fmt.Errorf("failed to enable watch mode: %w", err)
		}
		defer srv.StopWatch()
		fmt.Printf("\n👀 Watch mode enabled - files will auto-reload on changes\n")
	}

	// Start server
	addr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
	fmt.Printf("\n🌐 Server running at http://%s\n", addr)
	if cfg.Features.HotReload {
		fmt.Printf("📝 Edit .md files and see changes instantly\n")
	}
	fmt.Printf("⚡ Gzip compression enabled\n")
	fmt.Printf("Press Ctrl+C to stop\n\n")

	// Wrap server with compression middleware
	handler := server.WithCompression(srv)

	if err := http.ListenAndServe(addr, handler); err != nil {
		return fmt.Errorf("server error: %w", err)
	}

	return nil
}

func init() {
	log.SetFlags(0) // Remove timestamp from logs
}
