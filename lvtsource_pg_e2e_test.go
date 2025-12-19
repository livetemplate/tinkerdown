package livepage_test

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"net/http/httptest"

	"github.com/chromedp/cdproto/runtime"
	"github.com/chromedp/chromedp"
	_ "github.com/lib/pq"
	"github.com/livetemplate/livepage/internal/config"
	"github.com/livetemplate/livepage/internal/server"
)

// TestLvtSourcePostgres tests the lvt-source functionality with PostgreSQL
// This test verifies that:
// 1. lvt-source="users" fetches data from the configured PostgreSQL query
// 2. The data is rendered in the template
// 3. The Refresh action re-fetches data
func TestLvtSourcePostgres(t *testing.T) {
	// Check if Docker is available
	if _, err := exec.LookPath("docker"); err != nil {
		t.Skip("Docker not available, skipping PostgreSQL E2E test")
	}

	// Start PostgreSQL container
	containerID, dbURL := startPostgresContainer(t)
	defer stopPostgresContainer(t, containerID)

	// Wait for PostgreSQL to be ready and set up test data
	setupTestDatabase(t, dbURL)

	// Set DATABASE_URL for the test
	originalDBURL := os.Getenv("DATABASE_URL")
	os.Setenv("DATABASE_URL", dbURL)
	defer func() {
		if originalDBURL != "" {
			os.Setenv("DATABASE_URL", originalDBURL)
		} else {
			os.Unsetenv("DATABASE_URL")
		}
	}()

	// Load config from test example
	cfg, err := config.LoadFromDir("examples/lvt-source-pg-test")
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Verify source is configured
	if cfg.Sources == nil {
		t.Fatal("No sources configured in livepage.yaml")
	}
	userSource, ok := cfg.Sources["users"]
	if !ok {
		t.Fatal("users source not found in config")
	}
	if userSource.Type != "pg" {
		t.Fatalf("Expected pg source type, got: %s", userSource.Type)
	}
	t.Logf("Source config: type=%s, query=%s", userSource.Type, userSource.Query)

	// Create test server
	srv := server.NewWithConfig("examples/lvt-source-pg-test", cfg)
	if err := srv.Discover(); err != nil {
		t.Fatalf("Failed to discover pages: %v", err)
	}

	handler := server.WithCompression(srv)
	ts := httptest.NewServer(handler)
	defer ts.Close()

	// Setup chromedp
	allocCtx, cancel := chromedp.NewExecAllocator(context.Background(),
		append(chromedp.DefaultExecAllocatorOptions[:],
			chromedp.Flag("headless", true),
		)...)
	defer cancel()

	ctx, cancel := chromedp.NewContext(allocCtx, chromedp.WithLogf(t.Logf))
	defer cancel()

	ctx, cancel = context.WithTimeout(ctx, 60*time.Second)
	defer cancel()

	// Store console logs for debugging
	var consoleLogs []string
	chromedp.ListenTarget(ctx, func(ev interface{}) {
		if ev, ok := ev.(*runtime.EventConsoleAPICalled); ok {
			for _, arg := range ev.Args {
				consoleLogs = append(consoleLogs, fmt.Sprintf("[Console] %s", arg.Value))
			}
		}
	})

	t.Logf("Test server URL: %s", ts.URL)

	// Test 1: Navigate and wait for WebSocket to render content
	var hasInteractiveBlock bool
	err = chromedp.Run(ctx,
		chromedp.Navigate(ts.URL+"/"),
		chromedp.Sleep(3*time.Second),
		chromedp.Evaluate(`document.querySelector('.livepage-interactive-block') !== null`, &hasInteractiveBlock),
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
		t.Fatal("No user rows found - data was not fetched from PostgreSQL")
	}
	t.Logf("Found %d user rows", rowCount)

	if rowCount != 3 {
		t.Fatalf("Expected 3 users from database, got %d", rowCount)
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

	t.Log("All lvt-source PostgreSQL tests passed!")
}

// startPostgresContainer starts a PostgreSQL container and returns its ID and connection URL
func startPostgresContainer(t *testing.T) (string, string) {
	t.Helper()

	// Use a unique container name based on test name
	containerName := fmt.Sprintf("livepage-pg-test-%d", time.Now().UnixNano())

	// Start PostgreSQL container
	cmd := exec.Command("docker", "run", "-d",
		"--name", containerName,
		"-e", "POSTGRES_PASSWORD=testpass",
		"-e", "POSTGRES_USER=testuser",
		"-e", "POSTGRES_DB=testdb",
		"-p", "0:5432", // Let Docker assign a random port
		"postgres:15-alpine",
	)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to start PostgreSQL container: %v\nOutput: %s", err, output)
	}

	containerID := strings.TrimSpace(string(output))
	t.Logf("Started PostgreSQL container: %s", containerID[:12])

	// Get the mapped port
	cmd = exec.Command("docker", "port", containerID, "5432")
	output, err = cmd.CombinedOutput()
	if err != nil {
		stopPostgresContainer(t, containerID)
		t.Fatalf("Failed to get container port: %v\nOutput: %s", err, output)
	}

	// Parse port from output (e.g., "0.0.0.0:55432" or ":::55432")
	portLine := strings.TrimSpace(string(output))
	parts := strings.Split(portLine, ":")
	port := parts[len(parts)-1]

	dbURL := fmt.Sprintf("postgres://testuser:testpass@localhost:%s/testdb?sslmode=disable", port)
	t.Logf("PostgreSQL URL: %s", dbURL)

	return containerID, dbURL
}

// stopPostgresContainer stops and removes the PostgreSQL container
func stopPostgresContainer(t *testing.T, containerID string) {
	t.Helper()

	cmd := exec.Command("docker", "rm", "-f", containerID)
	if err := cmd.Run(); err != nil {
		t.Logf("Warning: failed to stop container: %v", err)
	} else {
		t.Logf("Stopped PostgreSQL container: %s", containerID[:12])
	}
}

// setupTestDatabase waits for PostgreSQL to be ready and creates test data
func setupTestDatabase(t *testing.T, dbURL string) {
	t.Helper()

	// Wait for PostgreSQL to be ready
	var db *sql.DB
	var err error
	for i := 0; i < 30; i++ {
		db, err = sql.Open("postgres", dbURL)
		if err == nil {
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			err = db.PingContext(ctx)
			cancel()
			if err == nil {
				break
			}
			db.Close()
		}
		time.Sleep(time.Second)
	}
	if err != nil {
		t.Fatalf("PostgreSQL not ready after 30 seconds: %v", err)
	}
	defer db.Close()

	// Create test table and data
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS users (
			id SERIAL PRIMARY KEY,
			name VARCHAR(255) NOT NULL,
			email VARCHAR(255) NOT NULL
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create table: %v", err)
	}

	// Insert test data
	_, err = db.Exec(`
		INSERT INTO users (name, email) VALUES
			('Alice', 'alice@example.com'),
			('Bob', 'bob@example.com'),
			('Charlie', 'charlie@example.com')
	`)
	if err != nil {
		t.Fatalf("Failed to insert test data: %v", err)
	}

	t.Log("Test database setup complete")
}
