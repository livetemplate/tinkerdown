package commands

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/livetemplate/tinkerdown/internal/config"
	"github.com/livetemplate/tinkerdown/internal/server"
)

// ServeCommand implements the serve command.
func ServeCommand(args []string) error {
	// Parse arguments
	dir := "."
	var configPath string
	var port string
	var host string
	var watch *bool
	var operator string
	var allowExec bool
	var headless bool

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
		} else if arg == "--operator" || arg == "-o" {
			if i+1 < len(args) {
				operator = args[i+1]
				i++
			}
		} else if arg == "--allow-exec" {
			allowExec = true
		} else if arg == "--headless" {
			headless = true
		} else if !strings.HasPrefix(arg, "-") {
			// Positional argument (directory)
			dir = arg
		}
	}

	// Set operator identity (defaults to $USER if not specified)
	config.SetOperator(operator)

	// Set exec permission (disabled by default for security)
	config.SetAllowExec(allowExec)

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
	if headless {
		cfg.Features.Headless = true
	}

	if cfg.Features.Headless {
		fmt.Printf("📚 Tinkerdown Headless Server\n\n")
		fmt.Printf("Directory: %s\n", absDir)
		fmt.Printf("Mode: 🤖 Headless (API, webhooks, schedules only)\n")
	} else {
		fmt.Printf("📚 Livemdtools Development Server\n\n")
		fmt.Printf("Serving: %s\n", absDir)
		if cfg.IsSiteMode() {
			fmt.Printf("Mode: 📖 Multi-page documentation site\n")
		} else {
			fmt.Printf("Mode: 📝 Single tutorial\n")
		}
	}

	// Create server
	srv := server.NewWithConfig(absDir, cfg)

	// Discover pages (needed for schedules even in headless mode)
	if err := srv.Discover(); err != nil {
		return fmt.Errorf("failed to discover pages: %w", err)
	}

	// Print discovered pages (only in non-headless mode)
	if !cfg.Features.Headless {
		fmt.Printf("\nPages discovered:\n")
		for _, route := range srv.Routes() {
			fmt.Printf("  %-30s %s\n", route.Pattern, route.FilePath)
		}
	}

	// Enable watch mode if requested (and not in headless mode)
	if cfg.Features.HotReload && !cfg.Features.Headless {
		if err := srv.EnableWatch(true); err != nil {
			return fmt.Errorf("failed to enable watch mode: %w", err)
		}
		defer srv.StopWatch()
		fmt.Printf("\n👀 Watch mode enabled - files will auto-reload on changes\n")
	} else if cfg.Features.HotReload && cfg.Features.Headless {
		fmt.Printf("⚠️  Watch mode disabled (not supported in headless mode)\n")
	}

	// Start schedule runner (always, for both headless and normal mode)
	// NOTE: Discover() must be called before StartSchedules() to register page schedules
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := srv.StartSchedules(ctx); err != nil {
		cancel() // Cancel context before returning to clean up
		return fmt.Errorf("failed to start schedule runner: %w", err)
	}
	// NOTE: StopSchedules() is called in the signal handler to ensure proper shutdown sequencing

	// Start server
	addr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
	fmt.Printf("\n🌐 Server running at http://%s\n", addr)
	if op := config.GetOperator(); op != "" {
		fmt.Printf("👤 Operator: %s\n", op)
	}
	if config.IsExecAllowed() {
		fmt.Printf("⚠️  Exec sources enabled (--allow-exec)\n")
	}
	if cfg.Features.Headless {
		fmt.Printf("🏥 Health endpoint at /health\n")
	}
	if cfg.IsAPIEnabled() {
		fmt.Printf("🔌 REST API enabled at /api/sources/{name}\n")
		if cfg.API.IsAuthEnabled() {
			fmt.Printf("🔐 API authentication enabled (header: %s)\n", cfg.API.Auth.GetHeaderName())
		}
	}
	if len(cfg.Webhooks) > 0 {
		fmt.Printf("🪝 Webhooks enabled at /webhook/{name}\n")
	}
	if scheduleCount := srv.GetScheduledJobCount(); scheduleCount > 0 {
		fmt.Printf("⏰ %d scheduled job(s) running\n", scheduleCount)
	}
	if cfg.Features.HotReload && !cfg.Features.Headless {
		fmt.Printf("📝 Edit .md files and see changes instantly\n")
	}
	if !cfg.Features.Headless {
		fmt.Printf("⚡ Gzip compression enabled\n")
	}
	fmt.Printf("Press Ctrl+C to stop\n\n")

	// Set up graceful shutdown
	httpServer := &http.Server{
		Addr:    addr,
		Handler: server.WithCompression(srv),
	}

	// Handle shutdown signals
	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
		<-sigChan

		fmt.Printf("\n🛑 Shutting down gracefully...\n")

		// Create a timeout context for shutdown (10 seconds)
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer shutdownCancel()

		// Stop HTTP server first with timeout
		if err := httpServer.Shutdown(shutdownCtx); err != nil {
			fmt.Printf("Warning: HTTP server shutdown error: %v\n", err)
		}

		// Cancel schedule runner context
		cancel()

		// Stop schedule runner and log any errors
		if err := srv.StopSchedules(); err != nil {
			fmt.Printf("Warning: Failed to stop schedules: %v\n", err)
		}
	}()

	if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("server error: %w", err)
	}

	return nil
}

func init() {
	log.SetFlags(0) // Remove timestamp from logs
}
