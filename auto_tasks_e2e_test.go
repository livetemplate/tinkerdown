//go:build !ci

package tinkerdown_test

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"net/http/httptest"

	"github.com/chromedp/cdproto/runtime"
	"github.com/chromedp/chromedp"
	"github.com/livetemplate/tinkerdown/internal/server"
)

// autoTasksConsoleLogs provides thread-safe console log capture for auto-tasks tests
type autoTasksConsoleLogs struct {
	mu   sync.RWMutex
	logs []string
}

func (l *autoTasksConsoleLogs) append(msg string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.logs = append(l.logs, msg)
}

func (l *autoTasksConsoleLogs) get() []string {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return append([]string{}, l.logs...)
}

// createAutoTasksExample creates a temp directory with a zero-config markdown file
// containing task list sections (no frontmatter, no separate data file).
func createAutoTasksExample(t *testing.T) (string, func()) {
	t.Helper()

	tempDir, err := os.MkdirTemp("", "auto-tasks-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	// Zero-config markdown: just headings + task lists
	content := `# My Day

## Morning Tasks
- [ ] Make coffee
- [x] Exercise
- [ ] Check email

## Evening Tasks
- [ ] Cook dinner
- [ ] Read book
`

	indexPath := filepath.Join(tempDir, "index.md")
	if err := os.WriteFile(indexPath, []byte(content), 0644); err != nil {
		os.RemoveAll(tempDir)
		t.Fatalf("Failed to write index.md: %v", err)
	}

	cleanup := func() {
		os.RemoveAll(tempDir)
	}

	return tempDir, cleanup
}

// autoTasksTestContext holds test infrastructure for auto-tasks E2E tests
type autoTasksTestContext struct {
	Server      *httptest.Server
	ChromeCtx   *DockerChromeContext
	URL         string
	ConsoleLogs *autoTasksConsoleLogs
	srv         *server.Server
}

func setupAutoTasksTest(t *testing.T, exampleDir string, enableWatch bool) (*autoTasksTestContext, func()) {
	t.Helper()

	srv := server.New(exampleDir)
	if err := srv.Discover(); err != nil {
		t.Fatalf("Failed to discover pages: %v", err)
	}

	if enableWatch {
		if err := srv.EnableWatch(true); err != nil {
			t.Fatalf("Failed to enable file watching: %v", err)
		}
	}

	handler := server.WithCompression(srv)
	ts := httptest.NewServer(handler)

	chromeCtx, chromeCleanup := SetupDockerChrome(t, 90*time.Second)

	url := ConvertURLForDockerChrome(ts.URL)

	consoleLogs := &autoTasksConsoleLogs{}
	chromedp.ListenTarget(chromeCtx.Context, func(ev any) {
		if ev, ok := ev.(*runtime.EventConsoleAPICalled); ok {
			for _, arg := range ev.Args {
				consoleLogs.append(fmt.Sprintf("[Console] %s", arg.Value))
			}
		}
	})

	testCtx := &autoTasksTestContext{
		Server:      ts,
		ChromeCtx:   chromeCtx,
		URL:         url,
		ConsoleLogs: consoleLogs,
		srv:         srv,
	}

	cleanup := func() {
		chromeCleanup()
		if enableWatch {
			srv.StopWatch()
		}
		ts.Close()
	}

	return testCtx, cleanup
}

// TestAutoTasks_BasicToggle verifies that plain markdown checkboxes render as
// interactive elements and can be toggled, with changes persisted to the file.
func TestAutoTasks_BasicToggle(t *testing.T) {
	tempDir, tempCleanup := createAutoTasksExample(t)
	defer tempCleanup()

	testCtx, cleanup := setupAutoTasksTest(t, tempDir, false)
	defer cleanup()

	ctx := testCtx.ChromeCtx.Context
	t.Logf("Test server URL: %s", testCtx.Server.URL)
	t.Logf("Temp dir: %s", tempDir)

	// Navigate and wait for content to render
	var hasAutoSource bool
	err := chromedp.Run(ctx,
		chromedp.Navigate(testCtx.URL+"/"),
		chromedp.Sleep(15*time.Second), // Wait for plugin compilation
		chromedp.Evaluate(`document.querySelector('[lvt-source="_auto_morning-tasks"]') !== null`, &hasAutoSource),
	)
	if err != nil {
		t.Fatalf("Failed to navigate: %v", err)
	}

	if !hasAutoSource {
		var htmlContent string
		chromedp.Run(ctx, chromedp.OuterHTML("html", &htmlContent))
		t.Logf("HTML (first 3000 chars): %s", htmlContent[:min(3000, len(htmlContent))])
		t.Logf("Console logs: %v", testCtx.ConsoleLogs.get())
		t.Fatal("Auto-generated task list was not rendered")
	}
	t.Log("Auto-generated task list rendered successfully")

	// Verify checkboxes exist
	var checkboxCount int
	err = chromedp.Run(ctx,
		chromedp.Evaluate(`document.querySelectorAll('[lvt-source="_auto_morning-tasks"] input[type="checkbox"]').length`, &checkboxCount),
	)
	if err != nil {
		t.Fatalf("Failed to count checkboxes: %v", err)
	}

	if checkboxCount != 3 {
		t.Fatalf("Expected 3 checkboxes in morning tasks, got %d", checkboxCount)
	}
	t.Log("Found 3 checkboxes in morning tasks section")

	// Get initial state of "Make coffee" (should be unchecked)
	// Content-based IDs are deterministic via FNV hash
	var makeChecked bool
	err = chromedp.Run(ctx,
		chromedp.Evaluate(`
			(() => {
				const checkboxes = document.querySelectorAll('[lvt-source="_auto_morning-tasks"] input[type="checkbox"]');
				// First checkbox is "Make coffee" (unchecked)
				return checkboxes.length > 0 ? checkboxes[0].checked : false;
			})()
		`, &makeChecked),
	)
	if err != nil {
		t.Fatalf("Failed to get initial state: %v", err)
	}
	if makeChecked {
		t.Fatal("'Make coffee' should start unchecked")
	}
	t.Log("'Make coffee' starts unchecked")

	// Click the first checkbox to toggle it
	err = chromedp.Run(ctx,
		chromedp.Click(`[lvt-source="_auto_morning-tasks"] input[type="checkbox"]`, chromedp.ByQuery),
		chromedp.Sleep(3*time.Second), // Wait for WebSocket response + file write
	)
	if err != nil {
		t.Fatalf("Failed to click checkbox: %v", err)
	}
	t.Log("Clicked first checkbox")

	// Verify the checkbox is now checked in the UI
	var afterChecked bool
	err = chromedp.Run(ctx,
		chromedp.Evaluate(`
			(() => {
				const checkboxes = document.querySelectorAll('[lvt-source="_auto_morning-tasks"] input[type="checkbox"]');
				return checkboxes.length > 0 ? checkboxes[0].checked : false;
			})()
		`, &afterChecked),
	)
	if err != nil {
		t.Fatalf("Failed to get state after toggle: %v", err)
	}
	if !afterChecked {
		t.Logf("Console logs: %v", testCtx.ConsoleLogs.get())
		t.Fatal("Checkbox should be checked after toggle")
	}
	t.Log("Checkbox is now checked in UI")

	// Verify the markdown file was updated on disk
	content, err := os.ReadFile(filepath.Join(tempDir, "index.md"))
	if err != nil {
		t.Fatalf("Failed to read index.md: %v", err)
	}

	fileStr := string(content)
	if !strings.Contains(fileStr, "- [x] Make coffee") {
		t.Logf("File content:\n%s", fileStr)
		t.Fatal("File should contain checked 'Make coffee' task")
	}
	t.Log("File was updated with checked task")

	// Verify the heading is preserved in the file
	if !strings.Contains(fileStr, "## Morning Tasks") {
		t.Fatal("Heading '## Morning Tasks' should still exist in the file")
	}
	t.Log("Headings preserved in file")

	t.Log("BasicToggle test passed!")
}

// TestAutoTasks_AddTask verifies that the auto-generated add form works:
// type a task name, submit, and verify it appears in both UI and file.
func TestAutoTasks_AddTask(t *testing.T) {
	tempDir, tempCleanup := createAutoTasksExample(t)
	defer tempCleanup()

	testCtx, cleanup := setupAutoTasksTest(t, tempDir, false)
	defer cleanup()

	ctx := testCtx.ChromeCtx.Context

	// Navigate and wait
	err := chromedp.Run(ctx,
		chromedp.Navigate(testCtx.URL+"/"),
		chromedp.Sleep(15*time.Second),
		chromedp.WaitVisible(`[lvt-source="_auto_morning-tasks"]`, chromedp.ByQuery),
	)
	if err != nil {
		t.Fatalf("Failed to navigate: %v", err)
	}
	t.Log("Page loaded with auto-tasks")

	// Type new task text and submit
	err = chromedp.Run(ctx,
		chromedp.SendKeys(`[lvt-source="_auto_morning-tasks"] input[name="text"]`, "Walk the dog", chromedp.ByQuery),
		chromedp.Click(`[lvt-source="_auto_morning-tasks"] button[type="submit"]`, chromedp.ByQuery),
		chromedp.Sleep(3*time.Second), // Wait for WebSocket + file write
	)
	if err != nil {
		t.Fatalf("Failed to add task: %v", err)
	}
	t.Log("Submitted new task 'Walk the dog'")

	// Verify new task appears in UI
	var newTaskCount int
	err = chromedp.Run(ctx,
		chromedp.Evaluate(`document.querySelectorAll('[lvt-source="_auto_morning-tasks"] input[type="checkbox"]').length`, &newTaskCount),
	)
	if err != nil {
		t.Fatalf("Failed to count tasks after add: %v", err)
	}

	if newTaskCount != 4 {
		t.Logf("Console logs: %v", testCtx.ConsoleLogs.get())
		t.Fatalf("Expected 4 tasks after adding, got %d", newTaskCount)
	}
	t.Log("New task appears in UI (4 total)")

	// Verify file contains the new task
	content, err := os.ReadFile(filepath.Join(tempDir, "index.md"))
	if err != nil {
		t.Fatalf("Failed to read index.md: %v", err)
	}

	if !strings.Contains(string(content), "Walk the dog") {
		t.Logf("File content:\n%s", string(content))
		t.Fatal("File should contain 'Walk the dog' task")
	}
	t.Log("New task persisted to file")

	t.Log("AddTask test passed!")
}

// TestAutoTasks_MultipleSections verifies that two independent task sections
// render and operate independently.
func TestAutoTasks_MultipleSections(t *testing.T) {
	tempDir, tempCleanup := createAutoTasksExample(t)
	defer tempCleanup()

	testCtx, cleanup := setupAutoTasksTest(t, tempDir, false)
	defer cleanup()

	ctx := testCtx.ChromeCtx.Context

	// Navigate and wait
	err := chromedp.Run(ctx,
		chromedp.Navigate(testCtx.URL+"/"),
		chromedp.Sleep(15*time.Second),
	)
	if err != nil {
		t.Fatalf("Failed to navigate: %v", err)
	}

	// Verify both sections rendered
	var hasMorning, hasEvening bool
	err = chromedp.Run(ctx,
		chromedp.Evaluate(`document.querySelector('[lvt-source="_auto_morning-tasks"]') !== null`, &hasMorning),
		chromedp.Evaluate(`document.querySelector('[lvt-source="_auto_evening-tasks"]') !== null`, &hasEvening),
	)
	if err != nil {
		t.Fatalf("Failed to check sections: %v", err)
	}

	if !hasMorning {
		var html string
		chromedp.Run(ctx, chromedp.OuterHTML("html", &html))
		t.Logf("HTML (first 3000 chars): %s", html[:min(3000, len(html))])
		t.Fatal("Morning tasks section not found")
	}
	if !hasEvening {
		t.Fatal("Evening tasks section not found")
	}
	t.Log("Both task sections rendered independently")

	// Count tasks in each section
	var morningCount, eveningCount int
	err = chromedp.Run(ctx,
		chromedp.Evaluate(`document.querySelectorAll('[lvt-source="_auto_morning-tasks"] input[type="checkbox"]').length`, &morningCount),
		chromedp.Evaluate(`document.querySelectorAll('[lvt-source="_auto_evening-tasks"] input[type="checkbox"]').length`, &eveningCount),
	)
	if err != nil {
		t.Fatalf("Failed to count tasks: %v", err)
	}

	if morningCount != 3 {
		t.Fatalf("Expected 3 morning tasks, got %d", morningCount)
	}
	if eveningCount != 2 {
		t.Fatalf("Expected 2 evening tasks, got %d", eveningCount)
	}
	t.Logf("Morning: %d tasks, Evening: %d tasks", morningCount, eveningCount)

	t.Log("MultipleSections test passed!")
}

// TestAutoTasks_PersistAcrossReload verifies that toggled state survives a page reload.
func TestAutoTasks_PersistAcrossReload(t *testing.T) {
	tempDir, tempCleanup := createAutoTasksExample(t)
	defer tempCleanup()

	testCtx, cleanup := setupAutoTasksTest(t, tempDir, false)
	defer cleanup()

	ctx := testCtx.ChromeCtx.Context

	// Navigate and wait
	err := chromedp.Run(ctx,
		chromedp.Navigate(testCtx.URL+"/"),
		chromedp.Sleep(15*time.Second),
		chromedp.WaitVisible(`[lvt-source="_auto_morning-tasks"]`, chromedp.ByQuery),
	)
	if err != nil {
		t.Fatalf("Failed to navigate: %v", err)
	}

	// Toggle first checkbox
	err = chromedp.Run(ctx,
		chromedp.Click(`[lvt-source="_auto_morning-tasks"] input[type="checkbox"]`, chromedp.ByQuery),
		chromedp.Sleep(3*time.Second),
	)
	if err != nil {
		t.Fatalf("Failed to toggle: %v", err)
	}
	t.Log("Toggled first checkbox")

	// Reload the page
	err = chromedp.Run(ctx,
		chromedp.Navigate(testCtx.URL+"/"),
		chromedp.Sleep(15*time.Second),
		chromedp.WaitVisible(`[lvt-source="_auto_morning-tasks"]`, chromedp.ByQuery),
	)
	if err != nil {
		t.Fatalf("Failed to reload: %v", err)
	}
	t.Log("Page reloaded")

	// Verify the toggled state persists
	var isChecked bool
	err = chromedp.Run(ctx,
		chromedp.Evaluate(`
			(() => {
				const checkboxes = document.querySelectorAll('[lvt-source="_auto_morning-tasks"] input[type="checkbox"]');
				return checkboxes.length > 0 ? checkboxes[0].checked : false;
			})()
		`, &isChecked),
	)
	if err != nil {
		t.Fatalf("Failed to check persisted state: %v", err)
	}

	if !isChecked {
		t.Logf("Console logs: %v", testCtx.ConsoleLogs.get())
		t.Fatal("Checkbox should remain checked after reload")
	}
	t.Log("Checkbox state persisted across reload")

	t.Log("PersistAcrossReload test passed!")
}

// TestAutoTasks_NoFullReload verifies that toggling a checkbox does NOT trigger
// a full browser reload (only a WebSocket data update).
func TestAutoTasks_NoFullReload(t *testing.T) {
	tempDir, tempCleanup := createAutoTasksExample(t)
	defer tempCleanup()

	// Enable file watching — this is where the watcher fix matters
	testCtx, cleanup := setupAutoTasksTest(t, tempDir, true)
	defer cleanup()

	ctx := testCtx.ChromeCtx.Context

	// Navigate and wait
	err := chromedp.Run(ctx,
		chromedp.Navigate(testCtx.URL+"/"),
		chromedp.Sleep(15*time.Second),
		chromedp.WaitVisible(`[lvt-source="_auto_morning-tasks"]`, chromedp.ByQuery),
	)
	if err != nil {
		t.Fatalf("Failed to navigate: %v", err)
	}

	// Inject a marker into the DOM that would be lost on full page reload
	err = chromedp.Run(ctx,
		chromedp.Evaluate(`
			(() => {
				window.__autoTasksReloadMarker = 'not-reloaded';
				return true;
			})()
		`, nil),
	)
	if err != nil {
		t.Fatalf("Failed to set marker: %v", err)
	}
	t.Log("Reload marker set")

	// Toggle checkbox
	err = chromedp.Run(ctx,
		chromedp.Click(`[lvt-source="_auto_morning-tasks"] input[type="checkbox"]`, chromedp.ByQuery),
		chromedp.Sleep(5*time.Second), // Wait for watcher + refresh cycle
	)
	if err != nil {
		t.Fatalf("Failed to toggle: %v", err)
	}
	t.Log("Toggled checkbox with file watcher active")

	// Check that the marker is still present (no full reload happened)
	var marker string
	err = chromedp.Run(ctx,
		chromedp.Evaluate(`window.__autoTasksReloadMarker || 'reloaded'`, &marker),
	)
	if err != nil {
		t.Fatalf("Failed to check marker: %v", err)
	}

	if marker != "not-reloaded" {
		t.Logf("Console logs: %v", testCtx.ConsoleLogs.get())
		t.Fatalf("Expected marker 'not-reloaded', got %q — full page reload occurred!", marker)
	}
	t.Log("No full page reload — watcher fix working correctly")

	// Verify the checkbox was still toggled successfully
	var isChecked bool
	err = chromedp.Run(ctx,
		chromedp.Evaluate(`
			(() => {
				const checkboxes = document.querySelectorAll('[lvt-source="_auto_morning-tasks"] input[type="checkbox"]');
				return checkboxes.length > 0 ? checkboxes[0].checked : false;
			})()
		`, &isChecked),
	)
	if err != nil {
		t.Fatalf("Failed to verify toggle: %v", err)
	}

	if !isChecked {
		t.Fatal("Checkbox should be checked even without full reload")
	}
	t.Log("Checkbox toggled via WebSocket data refresh (no reload)")

	t.Log("NoFullReload test passed!")
}
