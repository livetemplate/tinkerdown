//go:build !ci

package tinkerdown_test

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/chromedp/chromedp"
)

// TestBuildCommandSingleFile tests building a single markdown file
func TestBuildCommandSingleFile(t *testing.T) {
	tmpDir := t.TempDir()
	binPath := buildTinkerdown(t, tmpDir)

	// Create a simple test markdown file
	mdContent := `---
title: Test App
---

# Hello World

This is a simple test app.
`
	mdPath := filepath.Join(tmpDir, "app.md")
	if err := os.WriteFile(mdPath, []byte(mdContent), 0644); err != nil {
		t.Fatalf("Failed to write test markdown: %v", err)
	}

	// Build the app
	outputPath := filepath.Join(tmpDir, "test-app")
	cmd := exec.Command(binPath, "build", mdPath, "-o", outputPath)
	cmd.Env = append(os.Environ(), "GOWORK=off")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to build app: %v\nOutput: %s", err, output)
	}

	// Verify output contains success message
	if !strings.Contains(string(output), "Build successful") {
		t.Errorf("Success message not found in output: %s", output)
	}

	// Verify binary was created
	if _, err := os.Stat(outputPath); os.IsNotExist(err) {
		t.Fatal("Built binary does not exist")
	}

	// Verify binary is executable
	info, err := os.Stat(outputPath)
	if err != nil {
		t.Fatalf("Failed to stat binary: %v", err)
	}
	if info.Mode()&0111 == 0 {
		t.Error("Built binary is not executable")
	}
}

// TestBuildCommandDirectory tests building from a directory
func TestBuildCommandDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	binPath := buildTinkerdown(t, tmpDir)

	// Create a directory with markdown files
	appDir := filepath.Join(tmpDir, "my-app")
	if err := os.MkdirAll(appDir, 0755); err != nil {
		t.Fatalf("Failed to create app dir: %v", err)
	}

	// Create index.md
	indexContent := `---
title: Multi-Page App
---

# Home

Welcome to the app!
`
	if err := os.WriteFile(filepath.Join(appDir, "index.md"), []byte(indexContent), 0644); err != nil {
		t.Fatalf("Failed to write index.md: %v", err)
	}

	// Create about.md
	aboutContent := `---
title: About
---

# About

About this app.
`
	if err := os.WriteFile(filepath.Join(appDir, "about.md"), []byte(aboutContent), 0644); err != nil {
		t.Fatalf("Failed to write about.md: %v", err)
	}

	// Build the app
	outputPath := filepath.Join(tmpDir, "my-app-binary")
	cmd := exec.Command(binPath, "build", appDir, "-o", outputPath)
	cmd.Env = append(os.Environ(), "GOWORK=off")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to build app: %v\nOutput: %s", err, output)
	}

	// Verify binary was created
	if _, err := os.Stat(outputPath); os.IsNotExist(err) {
		t.Fatal("Built binary does not exist")
	}
}

// TestBuildCommandDefaultOutput tests default output name generation
func TestBuildCommandDefaultOutput(t *testing.T) {
	tmpDir := t.TempDir()
	binPath := buildTinkerdown(t, tmpDir)

	// Create a test markdown file
	mdPath := filepath.Join(tmpDir, "myapp.md")
	if err := os.WriteFile(mdPath, []byte("# Test"), 0644); err != nil {
		t.Fatalf("Failed to write test markdown: %v", err)
	}

	// Build without specifying output
	cmd := exec.Command(binPath, "build", mdPath)
	cmd.Dir = tmpDir
	cmd.Env = append(os.Environ(), "GOWORK=off")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to build app: %v\nOutput: %s", err, output)
	}

	// Verify default output name is used (myapp without .md)
	expectedOutput := filepath.Join(tmpDir, "myapp")
	if _, err := os.Stat(expectedOutput); os.IsNotExist(err) {
		t.Errorf("Expected default output 'myapp' not found")
	}
}

// TestBuildCommandMissingInput tests error on missing input
func TestBuildCommandMissingInput(t *testing.T) {
	tmpDir := t.TempDir()
	binPath := buildTinkerdown(t, tmpDir)

	// Run build without input
	cmd := exec.Command(binPath, "build")
	output, err := cmd.CombinedOutput()

	// Should fail
	if err == nil {
		t.Fatal("Expected error for missing input, but command succeeded")
	}

	// Verify error message
	if !strings.Contains(string(output), "input path required") {
		t.Errorf("Error message doesn't mention input required: %s", output)
	}
}

// TestBuildCommandNonexistentInput tests error on nonexistent input
func TestBuildCommandNonexistentInput(t *testing.T) {
	tmpDir := t.TempDir()
	binPath := buildTinkerdown(t, tmpDir)

	// Run build with nonexistent file
	cmd := exec.Command(binPath, "build", "/nonexistent/path.md")
	output, err := cmd.CombinedOutput()

	// Should fail
	if err == nil {
		t.Fatal("Expected error for nonexistent input, but command succeeded")
	}

	// Verify error message
	if !strings.Contains(string(output), "does not exist") {
		t.Errorf("Error message doesn't mention file doesn't exist: %s", output)
	}
}

// TestBuildCommandOutputFlag tests various output flag formats
func TestBuildCommandOutputFlag(t *testing.T) {
	tmpDir := t.TempDir()
	binPath := buildTinkerdown(t, tmpDir)

	// Create a test markdown file
	mdPath := filepath.Join(tmpDir, "test.md")
	if err := os.WriteFile(mdPath, []byte("# Test"), 0644); err != nil {
		t.Fatalf("Failed to write test markdown: %v", err)
	}

	tests := []struct {
		name   string
		flags  []string
		output string
	}{
		{"long-equals", []string{"--output=custom1"}, "custom1"},
		{"short-equals", []string{"-o=custom2"}, "custom2"},
		{"long-space", []string{"--output", "custom3"}, "custom3"},
		{"short-space", []string{"-o", "custom4"}, "custom4"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			outputPath := filepath.Join(tmpDir, tc.output)
			args := append([]string{"build", mdPath}, tc.flags...)
			cmd := exec.Command(binPath, args...)
			cmd.Dir = tmpDir
			cmd.Env = append(os.Environ(), "GOWORK=off")
			cmdOutput, err := cmd.CombinedOutput()
			if err != nil {
				t.Fatalf("Failed to build: %v\nOutput: %s", err, cmdOutput)
			}

			if _, err := os.Stat(outputPath); os.IsNotExist(err) {
				t.Errorf("Expected output '%s' not found", tc.output)
			}
		})
	}
}

// TestBuildCommandRunsServer tests that the built binary serves content
func TestBuildCommandRunsServer(t *testing.T) {
	tmpDir := t.TempDir()
	binPath := buildTinkerdown(t, tmpDir)

	// Create a test markdown file
	mdContent := `---
title: Server Test
---

# Hello from Built Binary

This page was served from a standalone binary.
`
	mdPath := filepath.Join(tmpDir, "servertest.md")
	if err := os.WriteFile(mdPath, []byte(mdContent), 0644); err != nil {
		t.Fatalf("Failed to write test markdown: %v", err)
	}

	// Build the app
	outputPath := filepath.Join(tmpDir, "server-test-binary")
	cmd := exec.Command(binPath, "build", mdPath, "-o", outputPath)
	cmd.Env = append(os.Environ(), "GOWORK=off")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to build app: %v\nOutput: %s", err, output)
	}

	// Find an available port
	port := findAvailablePort(t)

	// Start the built binary with output capture
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	serverCmd := exec.CommandContext(ctx, outputPath, fmt.Sprintf("--port=%d", port))
	serverCmd.Dir = tmpDir
	serverOutput := &strings.Builder{}
	serverCmd.Stdout = serverOutput
	serverCmd.Stderr = serverOutput
	if err := serverCmd.Start(); err != nil {
		t.Fatalf("Failed to start server: %v", err)
	}
	defer func() {
		serverCmd.Process.Kill()
		// Wait for process to exit to avoid race condition on serverOutput
		serverCmd.Wait()
		t.Logf("Server output:\n%s", serverOutput.String())
	}()

	// Wait for server to be ready
	addr := fmt.Sprintf("http://localhost:%d", port)
	waitForServer(t, addr, 15*time.Second)

	// First verify we can reach the page with http
	resp, err := http.Get(addr + "/app")
	if err != nil {
		t.Fatalf("Failed to fetch /app: %v", err)
	}
	resp.Body.Close()
	t.Logf("HTTP GET /app status: %d", resp.StatusCode)

	// Test with chromedp with its own timeout
	chromeCtx, chromeCancel := chromedp.NewContext(ctx)
	defer chromeCancel()

	chromeCtx, chromeTimeout := context.WithTimeout(chromeCtx, 30*time.Second)
	defer chromeTimeout()

	var pageTitle string
	err = chromedp.Run(chromeCtx,
		chromedp.Navigate(addr+"/app"), // Single files are copied as app.md -> /app route
		chromedp.WaitVisible("h1", chromedp.ByQuery),
		chromedp.Text("h1", &pageTitle, chromedp.ByQuery),
	)
	if err != nil {
		t.Fatalf("Chrome test failed: %v\nServer output:\n%s", err, serverOutput.String())
	}

	if !strings.Contains(pageTitle, "Hello from Built Binary") {
		t.Errorf("Expected page title 'Hello from Built Binary', got: %s", pageTitle)
	}
}

// TestBuildCommandCrossCompilation tests cross-compilation flag parsing
func TestBuildCommandCrossCompilation(t *testing.T) {
	tmpDir := t.TempDir()
	binPath := buildTinkerdown(t, tmpDir)

	// Create a test markdown file
	mdPath := filepath.Join(tmpDir, "cross.md")
	if err := os.WriteFile(mdPath, []byte("# Cross Test"), 0644); err != nil {
		t.Fatalf("Failed to write test markdown: %v", err)
	}

	// Test with invalid target format
	cmd := exec.Command(binPath, "build", mdPath, "--target=invalid")
	cmd.Env = append(os.Environ(), "GOWORK=off")
	output, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatal("Expected error for invalid target format")
	}
	if !strings.Contains(string(output), "invalid target format") {
		t.Errorf("Error should mention invalid target format: %s", output)
	}

	// Test with unsupported GOOS
	cmd = exec.Command(binPath, "build", mdPath, "--target=plan9/amd64")
	cmd.Env = append(os.Environ(), "GOWORK=off")
	output, err = cmd.CombinedOutput()
	if err == nil {
		t.Fatal("Expected error for unsupported GOOS")
	}
	if !strings.Contains(string(output), "unsupported GOOS") {
		t.Errorf("Error should mention unsupported GOOS: %s", output)
	}
}

// TestBuildCommandWithConfig tests that config files are included
func TestBuildCommandWithConfig(t *testing.T) {
	tmpDir := t.TempDir()
	binPath := buildTinkerdown(t, tmpDir)

	// Create a directory with markdown and config
	appDir := filepath.Join(tmpDir, "config-app")
	if err := os.MkdirAll(appDir, 0755); err != nil {
		t.Fatalf("Failed to create app dir: %v", err)
	}

	// Create index.md
	indexContent := `---
title: Config Test
---

# Config Test
`
	if err := os.WriteFile(filepath.Join(appDir, "index.md"), []byte(indexContent), 0644); err != nil {
		t.Fatalf("Failed to write index.md: %v", err)
	}

	// Create tinkerdown.yaml config
	configContent := `server:
  port: 9000
features:
  hot_reload: false
`
	if err := os.WriteFile(filepath.Join(appDir, "tinkerdown.yaml"), []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to write tinkerdown.yaml: %v", err)
	}

	// Build the app
	outputPath := filepath.Join(tmpDir, "config-app-binary")
	cmd := exec.Command(binPath, "build", appDir, "-o", outputPath)
	cmd.Env = append(os.Environ(), "GOWORK=off")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to build app: %v\nOutput: %s", err, output)
	}

	// Verify binary was created
	if _, err := os.Stat(outputPath); os.IsNotExist(err) {
		t.Fatal("Built binary does not exist")
	}
}

// findAvailablePort finds an available TCP port
func findAvailablePort(t *testing.T) int {
	t.Helper()
	listener, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		t.Fatalf("Failed to find available port: %v", err)
	}
	port := listener.Addr().(*net.TCPAddr).Port
	listener.Close()
	return port
}

// waitForServer waits for a server to be ready
func waitForServer(t *testing.T, addr string, timeout time.Duration) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		resp, err := http.Get(addr + "/app")
		if err == nil {
			resp.Body.Close()
			// Any response (including 404) means server is up
			return
		}
		time.Sleep(100 * time.Millisecond)
	}
	t.Fatalf("Server at %s not ready after %v", addr, timeout)
}
