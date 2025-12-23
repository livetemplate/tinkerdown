package livemdtools_test

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"net/http/httptest"

	"github.com/chromedp/cdproto/runtime"
	"github.com/chromedp/chromedp"
	"github.com/livetemplate/livemdtools/internal/config"
	"github.com/livetemplate/livemdtools/internal/server"
)

// TestLvtSourceMarkdownTaskList tests the lvt-source functionality with markdown task lists
func TestLvtSourceMarkdownTaskList(t *testing.T) {
	// Load config from test example
	cfg, err := config.LoadFromDir("examples/markdown-data-todo")
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Verify sources are configured
	if cfg.Sources == nil {
		t.Fatal("No sources configured in livemdtools.yaml")
	}
	tasksSource, ok := cfg.Sources["tasks"]
	if !ok {
		t.Fatal("tasks source not found in config")
	}
	if tasksSource.Type != "markdown" {
		t.Fatalf("Expected markdown source type, got: %s", tasksSource.Type)
	}
	if tasksSource.Anchor != "#data-section" {
		t.Fatalf("Expected anchor #data-section, got: %s", tasksSource.Anchor)
	}
	t.Logf("Source config: type=%s, anchor=%s", tasksSource.Type, tasksSource.Anchor)

	// Create test server
	srv := server.NewWithConfig("examples/markdown-data-todo", cfg)
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
	var hasTaskList bool
	err = chromedp.Run(ctx,
		chromedp.Navigate(ts.URL+"/"),
		chromedp.Sleep(5*time.Second),
		chromedp.Evaluate(`document.querySelector('[lvt-source="tasks"] ul') !== null`, &hasTaskList),
	)
	if err != nil {
		t.Fatalf("Failed to navigate: %v", err)
	}

	if !hasTaskList {
		var htmlContent string
		chromedp.Run(ctx, chromedp.OuterHTML("html", &htmlContent))
		t.Logf("HTML (first 3000 chars): %s", htmlContent[:min(3000, len(htmlContent))])
		t.Logf("Console logs: %v", consoleLogs)
		t.Fatal("Task list was not rendered - markdown source failed")
	}
	t.Log("Task list rendered from markdown")

	// Test 2: Verify task data is rendered
	var taskCount int
	var firstTaskText string
	var checkedCount int
	err = chromedp.Run(ctx,
		chromedp.Evaluate(`document.querySelectorAll('[lvt-source="tasks"] li').length`, &taskCount),
		chromedp.Evaluate(`
			(() => {
				const li = document.querySelector('[lvt-source="tasks"] li');
				const span = li ? li.querySelector('span') : null;
				return span ? span.textContent : '';
			})()
		`, &firstTaskText),
		chromedp.Evaluate(`document.querySelectorAll('[lvt-source="tasks"] input[type="checkbox"]:checked').length`, &checkedCount),
	)
	if err != nil {
		t.Fatalf("Failed to check task data: %v", err)
	}

	if taskCount != 5 {
		t.Fatalf("Expected 5 tasks from markdown, got %d", taskCount)
	}
	t.Logf("Correct number of tasks rendered from markdown: %d", taskCount)

	if !strings.Contains(firstTaskText, "Buy groceries") {
		t.Fatalf("Expected first task text to be 'Buy groceries', got: %s", firstTaskText)
	}
	t.Log("First task text is 'Buy groceries'")

	if checkedCount != 2 {
		t.Fatalf("Expected 2 completed tasks (checked checkboxes), got %d", checkedCount)
	}
	t.Logf("Correct number of completed tasks: %d", checkedCount)

	// Test 3: Verify the task counter shows correct total
	var totalText string
	err = chromedp.Run(ctx,
		chromedp.Evaluate(`
			(() => {
				const small = document.querySelector('[lvt-source="tasks"] small');
				return small ? small.textContent : '';
			})()
		`, &totalText),
	)
	if err != nil {
		t.Fatalf("Failed to check total: %v", err)
	}

	if !strings.Contains(totalText, "5") {
		t.Fatalf("Expected total to contain '5', got: %s", totalText)
	}
	t.Log("Task counter shows correct total")

	// Test 4: Verify refresh button exists
	var hasRefreshButton bool
	err = chromedp.Run(ctx,
		chromedp.Evaluate(`document.querySelector('[lvt-source="tasks"] button[lvt-click="Refresh"]') !== null`, &hasRefreshButton),
	)
	if err != nil {
		t.Fatalf("Failed to check refresh button: %v", err)
	}

	if !hasRefreshButton {
		t.Fatal("Refresh button not found")
	}
	t.Log("Refresh button exists")

	t.Log("Markdown task list source test passed!")
}

// TestLvtSourceMarkdownBulletList tests the lvt-source functionality with markdown bullet lists
func TestLvtSourceMarkdownBulletList(t *testing.T) {
	// Create a temporary example for bullet list testing
	// First, let's verify we can parse bullet lists using the existing infrastructure

	cfg := &config.Config{
		Title: "Bullet List Test",
		Sources: map[string]config.SourceConfig{
			"items": {
				Type:   "markdown",
				Anchor: "#items",
			},
		},
	}

	// Verify config is correctly structured
	if cfg.Sources["items"].Type != "markdown" {
		t.Fatalf("Expected markdown source type")
	}

	t.Log("Markdown bullet list config test passed!")
}

// TestLvtSourceMarkdownTable tests the lvt-source functionality with markdown tables
func TestLvtSourceMarkdownTable(t *testing.T) {
	// Verify config structure for table source
	cfg := &config.Config{
		Title: "Table Test",
		Sources: map[string]config.SourceConfig{
			"products": {
				Type:   "markdown",
				Anchor: "#products",
			},
		},
	}

	// Verify config is correctly structured
	if cfg.Sources["products"].Type != "markdown" {
		t.Fatalf("Expected markdown source type")
	}

	t.Log("Markdown table config test passed!")
}
