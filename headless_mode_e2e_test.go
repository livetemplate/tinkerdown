//go:build !ci

package tinkerdown_test

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/livetemplate/tinkerdown/internal/config"
	"github.com/livetemplate/tinkerdown/internal/server"
)

// TestHeadlessMode_HealthEndpoint tests the /health endpoint in headless mode.
func TestHeadlessMode_HealthEndpoint(t *testing.T) {
	// Create a temp directory with a simple markdown file
	tmpDir := t.TempDir()
	mdContent := `---
title: "Test Page"
---

# Test

Some content.
`
	if err := os.WriteFile(filepath.Join(tmpDir, "index.md"), []byte(mdContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Create config with headless mode enabled
	cfg := config.DefaultConfig()
	cfg.Features.Headless = true

	// Create server
	srv := server.NewWithConfig(tmpDir, cfg)
	if err := srv.Discover(); err != nil {
		t.Fatal(err)
	}

	// Start schedules
	ctx := context.Background()
	if err := srv.StartSchedules(ctx); err != nil {
		t.Fatal(err)
	}
	defer srv.StopSchedules()

	// Create test server
	ts := httptest.NewServer(srv)
	defer ts.Close()

	// Test /health endpoint
	resp, err := http.Get(ts.URL + "/health")
	if err != nil {
		t.Fatalf("Failed to get /health: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)
	var health map[string]interface{}
	if err := json.Unmarshal(body, &health); err != nil {
		t.Fatalf("Failed to parse health response: %v", err)
	}

	if health["status"] != "healthy" {
		t.Errorf("Expected status 'healthy', got %v", health["status"])
	}
	if health["headless"] != true {
		t.Errorf("Expected headless=true, got %v", health["headless"])
	}

	t.Logf("Health response: %s", string(body))
}

// TestHeadlessMode_WebUIBlocked tests that web UI routes return 404 in headless mode.
func TestHeadlessMode_WebUIBlocked(t *testing.T) {
	// Create a temp directory with a simple markdown file
	tmpDir := t.TempDir()
	mdContent := `---
title: "Test Page"
---

# Test Page

Some content.
`
	if err := os.WriteFile(filepath.Join(tmpDir, "index.md"), []byte(mdContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Create config with headless mode enabled
	cfg := config.DefaultConfig()
	cfg.Features.Headless = true

	// Create server
	srv := server.NewWithConfig(tmpDir, cfg)
	if err := srv.Discover(); err != nil {
		t.Fatal(err)
	}

	// Create test server
	ts := httptest.NewServer(srv)
	defer ts.Close()

	// Test various web UI routes that should be blocked
	testCases := []struct {
		path         string
		expectedCode int
	}{
		{"/", http.StatusNotFound},           // Root page
		{"/index", http.StatusNotFound},      // Index page
		{"/ws", http.StatusNotFound},         // WebSocket
		{"/assets/main.js", http.StatusNotFound},
		{"/playground", http.StatusNotFound},
		{"/playground/render", http.StatusNotFound},
		{"/search-index.json", http.StatusNotFound},
	}

	for _, tc := range testCases {
		t.Run(tc.path, func(t *testing.T) {
			resp, err := http.Get(ts.URL + tc.path)
			if err != nil {
				t.Fatalf("Failed to get %s: %v", tc.path, err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != tc.expectedCode {
				body, _ := io.ReadAll(resp.Body)
				t.Errorf("Expected status %d for %s, got %d: %s", tc.expectedCode, tc.path, resp.StatusCode, string(body))
			}
		})
	}
}

// TestHeadlessMode_APIWorks tests that API endpoints work in headless mode.
func TestHeadlessMode_APIWorks(t *testing.T) {
	// Create a temp directory with config that enables API
	tmpDir := t.TempDir()

	// Create a source config file
	cfgContent := `
title: "Test Site"
api:
  enabled: true
sources:
  test-data:
    type: json
    file: "data.json"
`
	if err := os.WriteFile(filepath.Join(tmpDir, "tinkerdown.yaml"), []byte(cfgContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Create a simple JSON data file
	jsonData := `[{"id": 1, "name": "Test"}]`
	if err := os.WriteFile(filepath.Join(tmpDir, "data.json"), []byte(jsonData), 0644); err != nil {
		t.Fatal(err)
	}

	// Create a minimal markdown file
	mdContent := `---
title: "Test"
---
# Test
`
	if err := os.WriteFile(filepath.Join(tmpDir, "index.md"), []byte(mdContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Load config from directory
	cfg, err := config.LoadFromDir(tmpDir)
	if err != nil {
		t.Fatal(err)
	}
	cfg.Features.Headless = true

	// Create server
	srv := server.NewWithConfig(tmpDir, cfg)
	if err := srv.Discover(); err != nil {
		t.Fatal(err)
	}

	// Create test server
	ts := httptest.NewServer(srv)
	defer ts.Close()

	// Test API endpoint
	resp, err := http.Get(ts.URL + "/api/sources/test-data")
	if err != nil {
		t.Fatalf("Failed to get API: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Errorf("Expected status 200 for API, got %d: %s", resp.StatusCode, string(body))
	}
}

// TestHeadlessMode_WebhooksWork tests that webhook endpoints work in headless mode.
func TestHeadlessMode_WebhooksWork(t *testing.T) {
	// Create a temp directory
	tmpDir := t.TempDir()

	// Create a minimal markdown file
	mdContent := `---
title: "Test"
---
# Test
`
	if err := os.WriteFile(filepath.Join(tmpDir, "index.md"), []byte(mdContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Create config with webhooks
	cfg := config.DefaultConfig()
	cfg.Features.Headless = true
	cfg.Actions = map[string]*config.Action{
		"test-action": {
			Kind: "exec",
			Cmd:  "echo test",
		},
	}
	cfg.Webhooks = map[string]*config.Webhook{
		"test-hook": {
			Action: "test-action",
		},
	}

	// Create server
	srv := server.NewWithConfig(tmpDir, cfg)
	if err := srv.Discover(); err != nil {
		t.Fatal(err)
	}

	// Create test server
	ts := httptest.NewServer(srv)
	defer ts.Close()

	// Test webhook endpoint
	resp, err := http.Post(ts.URL+"/webhook/test-hook", "application/json", strings.NewReader("{}"))
	if err != nil {
		t.Fatalf("Failed to post webhook: %v", err)
	}
	defer resp.Body.Close()

	// Note: exec actions may fail without --allow-exec, but endpoint should still respond
	// We're just testing that the endpoint is accessible
	body, _ := io.ReadAll(resp.Body)
	t.Logf("Webhook response: status=%d body=%s", resp.StatusCode, string(body))

	// Webhook endpoint should be accessible (either 200 success or 500 error from disabled exec)
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusInternalServerError {
		t.Errorf("Expected status 200 or 500 for webhook, got %d", resp.StatusCode)
	}
}

// TestHeadlessMode_ScheduleRunnerIntegration tests schedule runner in headless mode.
func TestHeadlessMode_ScheduleRunnerIntegration(t *testing.T) {
	// Create a temp directory with a markdown file containing schedules
	tmpDir := t.TempDir()
	mdContent := `---
title: "Schedule Test"
---

# Schedule Test

Notify @daily:9am Daily reminder

Run action:backup @weekly:sun:2am
`
	if err := os.WriteFile(filepath.Join(tmpDir, "index.md"), []byte(mdContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Create config with headless mode and action
	cfg := config.DefaultConfig()
	cfg.Features.Headless = true
	cfg.Actions = map[string]*config.Action{
		"backup": {
			Kind: "exec",
			Cmd:  "echo backup",
		},
	}

	// Create server
	srv := server.NewWithConfig(tmpDir, cfg)
	if err := srv.Discover(); err != nil {
		t.Fatal(err)
	}

	// Start schedules
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := srv.StartSchedules(ctx); err != nil {
		t.Fatal(err)
	}
	defer srv.StopSchedules()

	// Check schedule count
	scheduleCount := srv.GetScheduledJobCount()
	t.Logf("Found %d scheduled jobs", scheduleCount)

	// Should have at least the 2 schedules from the markdown
	if scheduleCount < 2 {
		t.Errorf("Expected at least 2 scheduled jobs, got %d", scheduleCount)
	}

	// Test health endpoint reports schedules
	ts := httptest.NewServer(srv)
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/health")
	if err != nil {
		t.Fatalf("Failed to get /health: %v", err)
	}
	defer resp.Body.Close()

	var health map[string]interface{}
	body, _ := io.ReadAll(resp.Body)
	if err := json.Unmarshal(body, &health); err != nil {
		t.Fatalf("Failed to parse health response: %v", err)
	}

	schedules, ok := health["schedules"].(float64)
	if !ok {
		t.Errorf("Expected schedules in health response, got: %s", string(body))
	} else if int(schedules) < 2 {
		t.Errorf("Expected at least 2 schedules in health, got %v", schedules)
	}

	t.Logf("Health reports %d schedules", int(schedules))
}

// TestHeadlessMode_NonHeadlessStillWorks tests that normal mode is unaffected.
func TestHeadlessMode_NonHeadlessStillWorks(t *testing.T) {
	// Create a temp directory with a simple markdown file
	tmpDir := t.TempDir()
	mdContent := `---
title: "Test Page"
---

# Test Page

Some content.
`
	if err := os.WriteFile(filepath.Join(tmpDir, "index.md"), []byte(mdContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Create config with headless mode DISABLED (normal mode)
	cfg := config.DefaultConfig()
	cfg.Features.Headless = false

	// Create server
	srv := server.NewWithConfig(tmpDir, cfg)
	if err := srv.Discover(); err != nil {
		t.Fatal(err)
	}

	// Create test server
	ts := httptest.NewServer(srv)
	defer ts.Close()

	// Test that web UI routes work in normal mode
	resp, err := http.Get(ts.URL + "/index")
	if err != nil {
		t.Fatalf("Failed to get /index: %v", err)
	}
	defer resp.Body.Close()

	// In normal mode, the index page should be accessible
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Errorf("Expected status 200 for /index in normal mode, got %d: %s", resp.StatusCode, string(body))
	}

	// Health endpoint should also work in normal mode
	resp2, err := http.Get(ts.URL + "/health")
	if err != nil {
		t.Fatalf("Failed to get /health: %v", err)
	}
	defer resp2.Body.Close()

	if resp2.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200 for /health, got %d", resp2.StatusCode)
	}

	var health map[string]interface{}
	body, _ := io.ReadAll(resp2.Body)
	json.Unmarshal(body, &health)

	if health["headless"] != false {
		t.Errorf("Expected headless=false in normal mode, got %v", health["headless"])
	}
}

// TestHeadlessMode_GracefulShutdown tests graceful shutdown of schedule runner.
func TestHeadlessMode_GracefulShutdown(t *testing.T) {
	// Create a temp directory with a markdown file containing schedules
	tmpDir := t.TempDir()
	mdContent := `---
title: "Shutdown Test"
---

# Test

Notify @daily:9am reminder
`
	if err := os.WriteFile(filepath.Join(tmpDir, "index.md"), []byte(mdContent), 0644); err != nil {
		t.Fatal(err)
	}

	cfg := config.DefaultConfig()
	cfg.Features.Headless = true

	srv := server.NewWithConfig(tmpDir, cfg)
	if err := srv.Discover(); err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	if err := srv.StartSchedules(ctx); err != nil {
		t.Fatal(err)
	}

	// Verify schedules are running
	if srv.GetScheduledJobCount() == 0 {
		t.Error("Expected at least 1 scheduled job")
	}

	// Cancel context and stop schedules
	cancel()
	time.Sleep(100 * time.Millisecond) // Give some time for cancellation

	if err := srv.StopSchedules(); err != nil {
		t.Errorf("StopSchedules failed: %v", err)
	}

	t.Log("Graceful shutdown completed successfully")
}

// TestHeadlessMode_ConfigFromYAML tests headless mode via config file.
func TestHeadlessMode_ConfigFromYAML(t *testing.T) {
	tmpDir := t.TempDir()

	// Create config file with headless enabled
	cfgContent := `
title: "Headless Test"
features:
  headless: true
api:
  enabled: true
`
	if err := os.WriteFile(filepath.Join(tmpDir, "tinkerdown.yaml"), []byte(cfgContent), 0644); err != nil {
		t.Fatal(err)
	}

	mdContent := `---
title: "Test"
---
# Test
`
	if err := os.WriteFile(filepath.Join(tmpDir, "index.md"), []byte(mdContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Load config from directory
	cfg, err := config.LoadFromDir(tmpDir)
	if err != nil {
		t.Fatal(err)
	}

	// Verify headless is enabled from config
	if !cfg.Features.Headless {
		t.Error("Expected Features.Headless to be true from config file")
	}

	// Create server and test
	srv := server.NewWithConfig(tmpDir, cfg)
	if err := srv.Discover(); err != nil {
		t.Fatal(err)
	}

	ts := httptest.NewServer(srv)
	defer ts.Close()

	// Web UI should be blocked
	resp, err := http.Get(ts.URL + "/index")
	if err != nil {
		t.Fatalf("Failed to get /index: %v", err)
	}
	resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("Expected 404 for /index in headless mode, got %d", resp.StatusCode)
	}

	// Health should work
	resp2, err := http.Get(ts.URL + "/health")
	if err != nil {
		t.Fatalf("Failed to get /health: %v", err)
	}
	resp2.Body.Close()

	if resp2.StatusCode != http.StatusOK {
		t.Errorf("Expected 200 for /health, got %d", resp2.StatusCode)
	}
}

// TestHeadlessMode_ErrorMessageIsDescriptive tests that 404 error message explains headless mode.
func TestHeadlessMode_ErrorMessageIsDescriptive(t *testing.T) {
	tmpDir := t.TempDir()
	mdContent := `---
title: "Test"
---
# Test
`
	if err := os.WriteFile(filepath.Join(tmpDir, "index.md"), []byte(mdContent), 0644); err != nil {
		t.Fatal(err)
	}

	cfg := config.DefaultConfig()
	cfg.Features.Headless = true

	srv := server.NewWithConfig(tmpDir, cfg)
	if err := srv.Discover(); err != nil {
		t.Fatal(err)
	}

	ts := httptest.NewServer(srv)
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/some-page")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	bodyStr := string(body)

	// Error message should mention headless mode and available endpoints
	if !strings.Contains(bodyStr, "headless") {
		t.Errorf("Error message should mention 'headless', got: %s", bodyStr)
	}
	if !strings.Contains(bodyStr, "/health") {
		t.Errorf("Error message should mention /health endpoint, got: %s", bodyStr)
	}
}

// Marker variable to ensure tests are running
var _ = fmt.Sprintf
