//go:build !ci

package tinkerdown_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/chromedp/cdproto/runtime"
	"github.com/chromedp/chromedp"
	"github.com/livetemplate/tinkerdown/internal/config"
	"github.com/livetemplate/tinkerdown/internal/server"
	"gopkg.in/yaml.v3"
)

// TestLvtSourceRest tests the lvt-source functionality with REST API
// This test verifies that:
// 1. lvt-source="users" fetches data from the configured REST API
// 2. The data is rendered in the template
// 3. The Refresh action re-fetches data
func TestLvtSourceRest(t *testing.T) {
	// Create mock API server
	mockUsers := []map[string]interface{}{
		{"id": 1, "name": "Alice", "email": "alice@example.com"},
		{"id": 2, "name": "Bob", "email": "bob@example.com"},
		{"id": 3, "name": "Charlie", "email": "charlie@example.com"},
	}

	mockAPI := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Logf("Mock API received request: %s %s", r.Method, r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(mockUsers)
	}))
	defer mockAPI.Close()
	t.Logf("Mock API URL: %s", mockAPI.URL)

	// Create temporary directory with test config
	tmpDir := t.TempDir()

	// Write tinkerdown.yaml with mock API URL
	configContent := fmt.Sprintf(`title: "REST API Test"
sources:
  users:
    type: rest
    from: %s/users
`, mockAPI.URL)

	if err := os.WriteFile(tmpDir+"/tinkerdown.yaml", []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to write config: %v", err)
	}

	// Write index.md
	indexContent := `---
title: "REST API Test"
---

# Users

` + "```lvt\n" + `<main lvt-source="users">
    <h2>Users</h2>
    {{if .Error}}
    <p><mark>Error: {{.Error}}</mark></p>
    {{else}}
    <table>
        <thead>
            <tr>
                <th>ID</th>
                <th>Name</th>
                <th>Email</th>
            </tr>
        </thead>
        <tbody>
            {{range .Data}}
            <tr data-user-id="{{.Id}}">
                <td>{{.Id}}</td>
                <td>{{.Name}}</td>
                <td>{{.Email}}</td>
            </tr>
            {{end}}
        </tbody>
    </table>
    {{end}}
    <button lvt-click="Refresh">Refresh</button>
</main>
` + "```\n"

	if err := os.WriteFile(tmpDir+"/index.md", []byte(indexContent), 0644); err != nil {
		t.Fatalf("Failed to write index.md: %v", err)
	}

	// Load config
	cfgContent, err := os.ReadFile(tmpDir + "/tinkerdown.yaml")
	if err != nil {
		t.Fatalf("Failed to read config: %v", err)
	}
	var cfg config.Config
	if err := yaml.Unmarshal(cfgContent, &cfg); err != nil {
		t.Fatalf("Failed to parse config: %v", err)
	}

	// Verify source is configured
	if cfg.Sources == nil {
		t.Fatal("No sources configured")
	}
	userSource, ok := cfg.Sources["users"]
	if !ok {
		t.Fatal("users source not found")
	}
	if userSource.Type != "rest" {
		t.Fatalf("Expected rest source type, got: %s", userSource.Type)
	}
	t.Logf("Source config: type=%s, url=%s", userSource.Type, userSource.From)

	// Create test server
	srv := server.NewWithConfig(tmpDir, &cfg)
	if err := srv.Discover(); err != nil {
		t.Fatalf("Failed to discover pages: %v", err)
	}

	handler := server.WithCompression(srv)
	ts := httptest.NewServer(handler)
	defer ts.Close()

	// Setup Docker Chrome for reliable CI execution
	chromeCtx, cleanup := SetupDockerChrome(t, 60*time.Second)
	defer cleanup()

	ctx := chromeCtx.Context

	// Store console logs for debugging
	var consoleLogs []string
	chromedp.ListenTarget(ctx, func(ev interface{}) {
		if ev, ok := ev.(*runtime.EventConsoleAPICalled); ok {
			for _, arg := range ev.Args {
				consoleLogs = append(consoleLogs, fmt.Sprintf("[Console] %s", arg.Value))
			}
		}
	})

	// Convert URL for Docker Chrome access
	url := ConvertURLForDockerChrome(ts.URL)
	t.Logf("Test server URL: %s (Docker: %s)", ts.URL, url)

	// Test 1: Navigate and wait for WebSocket to render content
	var hasInteractiveBlock bool
	err = chromedp.Run(ctx,
		chromedp.Navigate(url+"/"),
		chromedp.Sleep(3*time.Second),
		chromedp.Evaluate(`document.querySelector('.tinkerdown-interactive-block') !== null`, &hasInteractiveBlock),
	)
	if err != nil {
		t.Fatalf("Failed to navigate: %v", err)
	}

	if !hasInteractiveBlock {
		var htmlContent string
		chromedp.Run(ctx, chromedp.OuterHTML("html", &htmlContent))
		t.Logf("HTML: %s", htmlContent[:min(2000, len(htmlContent))])
		t.Fatal("Page did not load correctly - no interactive block found")
	}

	// Wait for WebSocket to render the table
	var tableRendered bool
	err = chromedp.Run(ctx,
		chromedp.Sleep(2*time.Second),
		chromedp.Evaluate(`document.querySelector('[lvt-source] table') !== null`, &tableRendered),
	)
	if err != nil {
		t.Fatalf("Failed to wait for table: %v", err)
	}

	if !tableRendered {
		var htmlContent string
		chromedp.Run(ctx, chromedp.OuterHTML("html", &htmlContent))
		t.Logf("HTML (first 2000 chars): %s", htmlContent[:min(2000, len(htmlContent))])
		t.Logf("Console logs: %v", consoleLogs)
		t.Fatal("Table was not rendered by WebSocket - table not found in lvt-source container")
	}
	t.Log("Page loaded and table rendered via WebSocket")

	// Test 2: Verify user data is rendered (using data attributes instead of classes)
	var rowCount int
	var firstUserName string
	var firstUserEmail string
	err = chromedp.Run(ctx,
		chromedp.Evaluate(`document.querySelectorAll('tr[data-user-id]').length`, &rowCount),
		chromedp.Evaluate(`
			(() => {
				const row = document.querySelector('tr[data-user-id]');
				const nameCell = row ? row.querySelector('td:nth-child(2)') : null;
				return nameCell ? nameCell.textContent : '';
			})()
		`, &firstUserName),
		chromedp.Evaluate(`
			(() => {
				const row = document.querySelector('tr[data-user-id]');
				const emailCell = row ? row.querySelector('td:nth-child(3)') : null;
				return emailCell ? emailCell.textContent : '';
			})()
		`, &firstUserEmail),
	)
	if err != nil {
		t.Fatalf("Failed to check user data: %v", err)
	}

	if rowCount == 0 {
		t.Logf("Console logs: %v", consoleLogs)
		t.Fatal("No user rows found - data was not fetched from REST API")
	}
	t.Logf("Found %d user rows", rowCount)

	if rowCount != 3 {
		t.Fatalf("Expected 3 users from REST API, got %d", rowCount)
	}
	t.Log("Correct number of users rendered")

	// Verify first user data
	if !strings.Contains(firstUserName, "Alice") {
		t.Fatalf("Expected first user name to be 'Alice', got: %s", firstUserName)
	}
	t.Log("First user name is Alice")

	if !strings.Contains(firstUserEmail, "alice@example.com") {
		t.Fatalf("Expected first user email to be 'alice@example.com', got: %s", firstUserEmail)
	}
	t.Log("First user email is alice@example.com")

	// Test 3: Verify Refresh button exists
	var refreshButtonExists bool
	err = chromedp.Run(ctx,
		chromedp.Evaluate(`document.querySelector('button[lvt-click="Refresh"]') !== null`, &refreshButtonExists),
	)
	if err != nil {
		t.Fatalf("Failed to check refresh button: %v", err)
	}

	if !refreshButtonExists {
		t.Fatal("Refresh button not found")
	}
	t.Log("Refresh button exists")

	// Test 4: Click refresh and verify data is still present (re-fetched)
	err = chromedp.Run(ctx,
		chromedp.Click(`button[lvt-click="Refresh"]`, chromedp.ByQuery),
		chromedp.Sleep(2*time.Second),
	)
	if err != nil {
		t.Fatalf("Failed to click refresh: %v", err)
	}
	t.Log("Clicked refresh button")

	// Verify data is still present after refresh
	var rowCountAfterRefresh int
	err = chromedp.Run(ctx,
		chromedp.Evaluate(`document.querySelectorAll('tr[data-user-id]').length`, &rowCountAfterRefresh),
	)
	if err != nil {
		t.Fatalf("Failed to check rows after refresh: %v", err)
	}

	if rowCountAfterRefresh != 3 {
		t.Logf("Console logs: %v", consoleLogs)
		t.Fatalf("Expected 3 rows after refresh, got %d", rowCountAfterRefresh)
	}
	t.Log("Data persisted after refresh")

	t.Log("All lvt-source REST API tests passed!")
}
