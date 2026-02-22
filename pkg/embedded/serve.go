// Package embedded provides functionality for running tinkerdown apps from embedded filesystems.
// This is primarily used by standalone binaries built with `tinkerdown build`.
package embedded

import (
	"context"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/livetemplate/tinkerdown/internal/config"
	"github.com/livetemplate/tinkerdown/internal/server"
)

// Serve starts a server that serves content from an embedded filesystem.
// This is the primary entry point for standalone binaries built with `tinkerdown build`.
//
// Parameters:
//   - contentFS: An embed.FS containing the markdown files and optional config
//   - rootPath: The path prefix within the embed.FS (e.g., "content")
//   - addr: The address to listen on (e.g., "localhost:8080")
//
// Example usage:
//
//	//go:embed content/*
//	var contentFS embed.FS
//
//	func main() {
//	    embedded.Serve(contentFS, "content", "localhost:8080")
//	}
func Serve(contentFS fs.FS, rootPath string, addr string) error {
	return ServeWithOptions(Options{
		ContentFS: contentFS,
		RootPath:  rootPath,
		Addr:      addr,
	})
}

// Options provides configuration for the embedded server.
type Options struct {
	// ContentFS is the embedded filesystem containing markdown files
	ContentFS fs.FS

	// RootPath is the path prefix within the ContentFS (e.g., "content")
	RootPath string

	// Addr is the address to listen on (e.g., "localhost:8080")
	Addr string

	// Config overrides the embedded config (optional)
	Config *config.Config

	// OnReady is called when the server is ready to accept connections (optional)
	OnReady func()

	// Quiet suppresses startup messages when true
	Quiet bool
}

// ServeWithOptions starts a server with more configuration options.
func ServeWithOptions(opts Options) error {
	// Extract embedded content to a temporary directory
	tmpDir, err := os.MkdirTemp("", "tinkerdown-embedded-*")
	if err != nil {
		return fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	// Extract content from embed.FS to temp directory
	if err := extractFS(opts.ContentFS, opts.RootPath, tmpDir); err != nil {
		return fmt.Errorf("failed to extract embedded content: %w", err)
	}

	// Load or use provided configuration
	var cfg *config.Config
	if opts.Config != nil {
		cfg = opts.Config
	} else {
		cfg, err = config.LoadFromDir(tmpDir)
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}
	}

	// Create server
	srv := server.NewWithConfig(tmpDir, cfg)

	// Discover pages
	if err := srv.Discover(); err != nil {
		return fmt.Errorf("failed to discover pages: %w", err)
	}

	// Print discovered pages
	if !opts.Quiet {
		fmt.Printf("\nPages discovered:\n")
		for _, route := range srv.Routes() {
			fmt.Printf("  %-30s %s\n", route.Pattern, route.FilePath)
		}
		fmt.Println()
	}

	// Start schedule runner
	ctx, cancel := context.WithCancel(context.Background())
	if err := srv.StartSchedules(ctx); err != nil {
		cancel()
		return fmt.Errorf("failed to start schedule runner: %w", err)
	}
	defer cancel()

	// Set up HTTP handler with compression
	var handler http.Handler = srv
	if !cfg.Features.Headless {
		handler = server.WithCompression(srv)
	}

	// Set up HTTP server with graceful shutdown
	httpServer := &http.Server{
		Addr:    opts.Addr,
		Handler: handler,
	}

	// Handle shutdown signals
	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
		<-sigChan

		if !opts.Quiet {
			fmt.Printf("\n🛑 Shutting down gracefully...\n")
		}

		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer shutdownCancel()

		if err := httpServer.Shutdown(shutdownCtx); err != nil {
			log.Printf("Warning: HTTP server shutdown error: %v\n", err)
		}

		cancel()

		if err := srv.StopSchedules(); err != nil {
			log.Printf("Warning: Failed to stop schedules: %v\n", err)
		}

		// Stop rate limiter cleanup goroutine
		srv.StopRateLimiter()
	}()

	// Call OnReady callback if provided
	if opts.OnReady != nil {
		opts.OnReady()
	}

	if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("server error: %w", err)
	}

	return nil
}

// extractFS extracts files from an fs.FS to a directory on disk.
func extractFS(contentFS fs.FS, rootPath string, destDir string) error {
	// Get the sub-filesystem at rootPath
	var srcFS fs.FS
	var err error

	if rootPath != "" && rootPath != "." {
		srcFS, err = fs.Sub(contentFS, rootPath)
		if err != nil {
			return fmt.Errorf("failed to get sub-filesystem at %q: %w", rootPath, err)
		}
	} else {
		srcFS = contentFS
	}

	// Clean destDir for path traversal validation
	cleanDestDir := filepath.Clean(destDir)

	return fs.WalkDir(srcFS, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		destPath := filepath.Join(destDir, path)

		// Path traversal validation: ensure destPath stays within destDir
		cleanDestPath := filepath.Clean(destPath)
		if !strings.HasPrefix(cleanDestPath, cleanDestDir+string(os.PathSeparator)) && cleanDestPath != cleanDestDir {
			return fmt.Errorf("path traversal detected: %q resolves outside destination directory", path)
		}

		if d.IsDir() {
			return os.MkdirAll(destPath, 0755)
		}

		// Read file from embed.FS
		content, err := fs.ReadFile(srcFS, path)
		if err != nil {
			return fmt.Errorf("failed to read embedded file %q: %w", path, err)
		}

		// Ensure parent directory exists
		if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
			return err
		}

		return os.WriteFile(destPath, content, 0644)
	})
}
