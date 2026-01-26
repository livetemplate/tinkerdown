//go:build !ci

package tinkerdown

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/chromedp/chromedp"
)

const (
	dockerImage           = "chromedp/headless-shell:latest"
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
// Chrome container uses host.docker.internal to reach the host on all platforms.
func GetChromeTestURL(port int) string {
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
	cleanupContainerByName(t, containerName)

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

	// Start the container
	t.Log("Starting Chrome headless Docker container...")
	portMapping := fmt.Sprintf("%d:9222", debugPort)

	cmd := exec.Command("docker", "run", "-d",
		"--rm",
		"--memory", "512m",
		"--cpus", "0.5",
		"-p", portMapping,
		"--name", containerName,
		"--add-host", "host.docker.internal:host-gateway",
		dockerImage,
	)

	if _, err := cmd.Output(); err != nil {
		return fmt.Errorf("failed to start Chrome Docker container: %w", err)
	}

	// Wait for Chrome to be ready
	t.Log("Waiting for Chrome to be ready...")
	chromeURL := fmt.Sprintf("http://localhost:%d/json/version", debugPort)
	ready := false
	var lastErr error
	for i := 0; i < 120; i++ { // 60 seconds
		resp, err := http.Get(chromeURL)
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
func cleanupContainerByName(t *testing.T, name string) {
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

// ConvertURLForDockerChrome converts an httptest URL (like http://127.0.0.1:12345)
// to a URL accessible from Docker Chrome (http://host.docker.internal:12345).
func ConvertURLForDockerChrome(httptestURL string) string {
	// httptest URLs are like "http://127.0.0.1:12345" or "http://[::1]:12345"
	// We need to replace the host with host.docker.internal
	url := strings.Replace(httptestURL, "127.0.0.1", "host.docker.internal", 1)
	url = strings.Replace(url, "[::1]", "host.docker.internal", 1)
	return url
}
