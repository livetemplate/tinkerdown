//go:build !ci

// Package tinkerdown E2E test helpers.
//
// NOTE: This file is intentionally duplicated in e2e_helpers_external_test.go
// because Go does not allow sharing code between internal tests (package tinkerdown)
// and external tests (package tinkerdown_test) without exporting it. Since these
// helpers are test-only infrastructure, we duplicate rather than export.
// The only differences are: package declaration and container name prefix.

package tinkerdown

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os/exec"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/chromedp/chromedp"
)

const (
	// Use stable tag for reliable CI builds - specific version 131.0.6778.264
	// was removed from Docker Hub
	dockerImage           = "chromedp/headless-shell:stable"
	chromeContainerPrefix = "chrome-e2e-tinkerdown-"
)

// DockerChromeContext provides a Docker Chrome context for E2E tests.
type DockerChromeContext struct {
	Context    context.Context
	Cancel     context.CancelFunc
	ChromePort int
	t          *testing.T
}

// SetupDockerChrome starts a Docker Chrome container and returns a chromedp context.
// Call cleanup() when done to stop the container and cancel the context.
func SetupDockerChrome(t *testing.T, timeout time.Duration) (*DockerChromeContext, func()) {
	t.Helper()

	chromePort, err := getFreePort()
	if err != nil {
		t.Fatalf("Failed to allocate Chrome port: %v", err)
	}

	if err := startDockerChrome(t, chromePort); err != nil {
		t.Fatalf("Failed to start Docker Chrome: %v", err)
	}

	chromeURL := fmt.Sprintf("http://localhost:%d", chromePort)
	allocCtx, allocCancel := chromedp.NewRemoteAllocator(context.Background(), chromeURL)

	ctx, ctxCancel := chromedp.NewContext(allocCtx, chromedp.WithLogf(t.Logf))

	// Apply timeout
	ctx, timeoutCancel := context.WithTimeout(ctx, timeout)

	dcc := &DockerChromeContext{
		Context:    ctx,
		Cancel:     timeoutCancel,
		ChromePort: chromePort,
		t:          t,
	}

	cleanup := func() {
		timeoutCancel()
		ctxCancel()
		allocCancel()
		stopDockerChrome(t, chromePort)
	}

	return dcc, cleanup
}

// GetChromeTestURL returns the URL for Chrome (in Docker) to access the test server.
// On Linux (--network host), Chrome accesses localhost directly.
// On macOS, Chrome needs host.docker.internal to reach the host.
func GetChromeTestURL(port int) string {
	if runtime.GOOS == "linux" {
		return fmt.Sprintf("http://localhost:%d", port)
	}
	return fmt.Sprintf("http://host.docker.internal:%d", port)
}

// getFreePort asks the kernel for a free open port that is ready to use.
func getFreePort() (port int, err error) {
	var a *net.TCPAddr
	if a, err = net.ResolveTCPAddr("tcp", "localhost:0"); err == nil {
		var l *net.TCPListener
		if l, err = net.ListenTCP("tcp", a); err == nil {
			defer l.Close()
			return l.Addr().(*net.TCPAddr).Port, nil
		}
	}
	return
}

// startDockerChrome starts the chromedp headless-shell Docker container.
func startDockerChrome(t *testing.T, debugPort int) error {
	t.Helper()

	// Check if Docker is available
	if _, err := exec.Command("docker", "version").CombinedOutput(); err != nil {
		t.Skip("Docker not available, skipping E2E test")
	}

	containerName := fmt.Sprintf("%s%d", chromeContainerPrefix, debugPort)
	cleanupContainerByName(containerName)

	// Check if image exists, if not try to pull it (with timeout)
	checkCmd := exec.Command("docker", "image", "inspect", dockerImage)
	if _, err := checkCmd.CombinedOutput(); err != nil {
		// Image doesn't exist, try to pull with timeout
		t.Log("Pulling chromedp/headless-shell Docker image...")

		pullCtx, pullCancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer pullCancel()

		pullCmd := exec.CommandContext(pullCtx, "docker", "pull", dockerImage)
		if output, err := pullCmd.CombinedOutput(); err != nil {
			if pullCtx.Err() == context.DeadlineExceeded {
				t.Fatal("Docker pull timed out after 60 seconds")
			}
			t.Fatalf("Failed to pull Docker image: %v\nOutput: %s", err, output)
		}
		t.Log("Docker image pulled successfully")
	}

	// Start the container:
	// - Linux: use --network host so Chrome listens directly on the host's debugPort
	// - macOS: --network host doesn't expose ports (Docker runs in a VM),
	//   so use port mapping to the container's default port 9222 instead
	t.Log("Starting Chrome headless Docker container...")

	args := []string{"run", "-d", "--rm", "--memory", "512m", "--cpus", "0.5", "--name", containerName}
	if runtime.GOOS == "linux" {
		args = append(args, "--network", "host")
		args = append(args, dockerImage, fmt.Sprintf("--remote-debugging-port=%d", debugPort))
	} else {
		// Map our dynamic port to the container's default 9222 (handled by entrypoint socat)
		args = append(args, "-p", fmt.Sprintf("%d:9222", debugPort))
		args = append(args, dockerImage)
	}
	cmd := exec.Command("docker", args...)

	if _, err := cmd.Output(); err != nil {
		return fmt.Errorf("failed to start Chrome Docker container: %w", err)
	}

	// Wait for Chrome to be ready
	t.Log("Waiting for Chrome to be ready...")
	chromeURL := fmt.Sprintf("http://localhost:%d/json/version", debugPort)
	ready := false
	var lastErr error
	// Use HTTP client with timeout to avoid hanging indefinitely
	httpClient := &http.Client{Timeout: 2 * time.Second}
	for i := 0; i < 120; i++ { // 60 seconds
		resp, err := httpClient.Get(chromeURL)
		if err == nil {
			resp.Body.Close()
			ready = true
			t.Logf("Chrome ready after %d attempts (%.1fs)", i+1, float64(i+1)*0.5)
			break
		}
		lastErr = err
		time.Sleep(500 * time.Millisecond)
	}

	if !ready {
		t.Logf("Chrome failed to start within 60 seconds. Last error: %v", lastErr)

		// Try to get container logs for debugging
		logsCmd := exec.Command("docker", "logs", "--tail", "50", containerName)
		if output, err := logsCmd.CombinedOutput(); err == nil && len(output) > 0 {
			t.Logf("Chrome container logs:\n%s", string(output))
		}

		// Clean up the container
		_, _ = exec.Command("docker", "rm", "-f", containerName).CombinedOutput()
		return fmt.Errorf("Chrome failed to start within 60 seconds: %w", lastErr)
	}

	t.Log("Chrome headless Docker container ready")
	return nil
}

// stopDockerChrome stops and removes the Chrome Docker container.
func stopDockerChrome(t *testing.T, debugPort int) {
	t.Helper()
	t.Log("Stopping Chrome Docker container...")

	containerName := fmt.Sprintf("%s%d", chromeContainerPrefix, debugPort)

	rmCmd := exec.Command("docker", "rm", "-f", containerName)
	if output, err := rmCmd.CombinedOutput(); err != nil {
		errMsg := string(output)
		if !strings.Contains(errMsg, "No such container") && !strings.Contains(err.Error(), "No such container") {
			t.Logf("Warning: Failed to remove Docker container: %v (output: %s)", err, errMsg)
		}
	}
}

// cleanupContainerByName removes any existing container with the given name.
// Errors are ignored since the container may not exist.
func cleanupContainerByName(name string) {
	rmCmd := exec.Command("docker", "rm", "-f", name)
	rmCmd.CombinedOutput() // Ignore errors - container may not exist
}

// WaitForServer polls an HTTP server until it responds or timeout is reached.
func WaitForServer(t *testing.T, serverURL string, timeout time.Duration) {
	t.Helper()

	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		resp, err := http.Get(serverURL)
		if err == nil {
			resp.Body.Close()
			t.Logf("Server ready at %s", serverURL)
			return
		}
		time.Sleep(100 * time.Millisecond)
	}
	t.Fatalf("Server at %s failed to become ready within %v", serverURL, timeout)
}

// ConvertURLForDockerChrome converts an httptest URL for Docker Chrome access.
// On Linux (--network host), Chrome shares the host network so localhost works.
// On macOS, Chrome is in an isolated container and needs host.docker.internal.
func ConvertURLForDockerChrome(httptestURL string) string {
	// httptest URLs are like "http://127.0.0.1:12345" or "http://[::1]:12345"
	if runtime.GOOS == "linux" {
		url := strings.Replace(httptestURL, "127.0.0.1", "localhost", 1)
		url = strings.Replace(url, "[::1]", "localhost", 1)
		return url
	}
	// macOS: container is isolated, use host.docker.internal to reach host
	url := strings.Replace(httptestURL, "127.0.0.1", "host.docker.internal", 1)
	url = strings.Replace(url, "[::1]", "host.docker.internal", 1)
	url = strings.Replace(url, "localhost", "host.docker.internal", 1)
	return url
}
