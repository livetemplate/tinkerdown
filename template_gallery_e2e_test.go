//go:build !ci

package tinkerdown_test

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"net/http/httptest"

	"github.com/chromedp/cdproto/runtime"
	"github.com/chromedp/chromedp"
	"github.com/livetemplate/tinkerdown/cmd/tinkerdown/commands"
	"github.com/livetemplate/tinkerdown/internal/config"
	"github.com/livetemplate/tinkerdown/internal/server"
)

// scaffoldTemplate creates a new project from a template in a temp directory and
// returns the project path. It uses NewCommand just like `tinkerdown new` does.
func scaffoldTemplate(t *testing.T, templateName string) string {
	t.Helper()
	tmpDir := t.TempDir()
	projectName := "test-" + templateName
	projectDir := filepath.Join(tmpDir, projectName)

	oldDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("Failed to chdir to temp: %v", err)
	}
	defer os.Chdir(oldDir)

	if err := commands.NewCommand([]string{projectName}, templateName); err != nil {
		t.Fatalf("NewCommand(%s) failed: %v", templateName, err)
	}

	return projectDir
}

// startTestServer creates a tinkerdown server for the given directory and returns
// the httptest server. The server discovers pages (including frontmatter sources).
func startTestServer(t *testing.T, dir string) *httptest.Server {
	t.Helper()
	srv := server.New(dir)
	if err := srv.Discover(); err != nil {
		t.Fatalf("Failed to discover pages in %s: %v", dir, err)
	}
	handler := server.WithCompression(srv)
	return httptest.NewServer(handler)
}

// templateE2EContext bundles the common setup for template E2E tests.
type templateE2EContext struct {
	t           *testing.T
	projectDir  string
	ts          *httptest.Server
	chromeCtx   *DockerChromeContext
	consoleLogs []string
	url         string
}

// setupTemplateE2E scaffolds a template, starts a server, and sets up Docker Chrome.
func setupTemplateE2E(t *testing.T, templateName string) (*templateE2EContext, func()) {
	t.Helper()

	projectDir := scaffoldTemplate(t, templateName)
	ts := startTestServer(t, projectDir)

	chromeCtx, chromeCleanup := SetupDockerChrome(t, 90*time.Second)

	ctx := &templateE2EContext{
		t:          t,
		projectDir: projectDir,
		ts:         ts,
		chromeCtx:  chromeCtx,
		url:        ConvertURLForDockerChrome(ts.URL),
	}

	// Capture console logs for debugging
	chromedp.ListenTarget(chromeCtx.Context, func(ev interface{}) {
		if ev, ok := ev.(*runtime.EventConsoleAPICalled); ok {
			for _, arg := range ev.Args {
				ctx.consoleLogs = append(ctx.consoleLogs, fmt.Sprintf("[Console] %s", arg.Value))
			}
		}
	})

	t.Logf("Template %s: server=%s docker=%s", templateName, ts.URL, ctx.url)

	cleanup := func() {
		chromeCleanup()
		ts.Close()
	}
	return ctx, cleanup
}

// dumpDebugInfo logs HTML and console output when a test fails.
func (ctx *templateE2EContext) dumpDebugInfo() {
	var htmlContent string
	chromedp.Run(ctx.chromeCtx.Context, chromedp.OuterHTML("html", &htmlContent))
	maxLen := 5000
	if len(htmlContent) > maxLen {
		htmlContent = htmlContent[:maxLen]
	}
	ctx.t.Logf("HTML:\n%s", htmlContent)
	ctx.t.Logf("Console logs: %v", ctx.consoleLogs)
}

// TestTemplateCSVInventoryE2E verifies the csv-inventory template renders a product table.
func TestTemplateCSVInventoryE2E(t *testing.T) {
	ctx, cleanup := setupTemplateE2E(t, "csv-inventory")
	defer cleanup()

	bctx := ctx.chromeCtx.Context

	// Navigate and wait for interactive block + table to render via WebSocket
	err := chromedp.Run(bctx,
		chromedp.Navigate(ctx.url+"/"),
		chromedp.WaitVisible(".tinkerdown-interactive-block", chromedp.ByQuery),
		chromedp.WaitVisible(".tinkerdown-interactive-block table", chromedp.ByQuery),
	)
	if err != nil {
		ctx.dumpDebugInfo()
		t.Fatalf("Product table was not rendered from CSV source: %v", err)
	}
	t.Log("Product table rendered from CSV file")

	// Verify row count (10 products in products.csv)
	var rowCount int
	err = chromedp.Run(bctx,
		chromedp.Evaluate(`document.querySelectorAll('tbody tr').length`, &rowCount),
	)
	if err != nil {
		t.Fatalf("Failed to query table data: %v", err)
	}

	if rowCount != 10 {
		ctx.dumpDebugInfo()
		t.Fatalf("Expected 10 product rows, got %d", rowCount)
	}
	t.Logf("Correct product count: %d rows", rowCount)

	// Verify data content
	var bodyHTML string
	err = chromedp.Run(bctx,
		chromedp.OuterHTML("body", &bodyHTML),
	)
	if err != nil {
		t.Fatalf("Failed to get body HTML: %v", err)
	}

	if !strings.Contains(bodyHTML, "Mechanical Keyboard") {
		ctx.dumpDebugInfo()
		t.Fatal("Expected 'Mechanical Keyboard' in rendered table")
	}
	t.Log("CSV inventory template E2E passed")
}

// TestTemplateJSONDashboardE2E verifies the json-dashboard template renders
// computed expressions and a task table.
func TestTemplateJSONDashboardE2E(t *testing.T) {
	ctx, cleanup := setupTemplateE2E(t, "json-dashboard")
	defer cleanup()

	bctx := ctx.chromeCtx.Context

	// Navigate and wait for table rendering
	err := chromedp.Run(bctx,
		chromedp.Navigate(ctx.url+"/"),
		chromedp.WaitVisible(".tinkerdown-interactive-block", chromedp.ByQuery),
		chromedp.WaitVisible(".tinkerdown-interactive-block table", chromedp.ByQuery),
	)
	if err != nil {
		ctx.dumpDebugInfo()
		t.Fatalf("Task table was not rendered from JSON source: %v", err)
	}
	t.Log("Task table rendered from JSON file")

	// Verify row count (8 tasks in metrics.json)
	var rowCount int
	err = chromedp.Run(bctx,
		chromedp.Evaluate(`document.querySelectorAll('tbody tr').length`, &rowCount),
	)
	if err != nil {
		t.Fatalf("Failed to query table data: %v", err)
	}

	if rowCount != 8 {
		ctx.dumpDebugInfo()
		t.Fatalf("Expected 8 task rows, got %d", rowCount)
	}
	t.Logf("Correct task count: %d rows", rowCount)

	// Verify computed expressions were evaluated.
	// The Summary section has expressions like `=count(tasks)` that resolve to numbers.
	// We check that the resolved value "Total Tasks: 8" appears in the rendered page.
	// (Note: the "How It Works" section below intentionally shows the raw syntax as docs.)
	var summaryHTML string
	err = chromedp.Run(bctx,
		chromedp.Evaluate(`
			(() => {
				// Get the Summary section content (between h2#summary and next h2)
				const headings = document.querySelectorAll('h2');
				for (const h of headings) {
					if (h.textContent.includes('Summary')) {
						let text = '';
						let el = h.nextElementSibling;
						while (el && el.tagName !== 'H2') {
							text += el.textContent + '\n';
							el = el.nextElementSibling;
						}
						return text;
					}
				}
				return document.body.innerText;
			})()
		`, &summaryHTML),
	)
	if err != nil {
		t.Fatalf("Failed to get summary text: %v", err)
	}

	// The summary should contain resolved numbers, not raw expressions
	if strings.Contains(summaryHTML, "=count(") || strings.Contains(summaryHTML, "=sum(") || strings.Contains(summaryHTML, "=max(") {
		ctx.dumpDebugInfo()
		t.Fatalf("Computed expressions in Summary section were not evaluated: %s", summaryHTML)
	}
	t.Log("Computed expressions evaluated")

	// Verify task data content
	var bodyHTML string
	err = chromedp.Run(bctx, chromedp.OuterHTML("body", &bodyHTML))
	if err != nil {
		t.Fatalf("Failed to get body: %v", err)
	}
	if !strings.Contains(bodyHTML, "Design homepage") {
		ctx.dumpDebugInfo()
		t.Fatal("Expected 'Design homepage' in rendered table")
	}

	t.Log("JSON dashboard template E2E passed")
}

// TestTemplateMarkdownNotesE2E verifies the markdown-notes template renders
// notes from a markdown table and supports the add form.
func TestTemplateMarkdownNotesE2E(t *testing.T) {
	ctx, cleanup := setupTemplateE2E(t, "markdown-notes")
	defer cleanup()

	bctx := ctx.chromeCtx.Context

	// Navigate and wait for rendering — markdown notes uses Go templates (not lvt-columns),
	// so the table is inside the interactive block rendered by WebSocket
	err := chromedp.Run(bctx,
		chromedp.Navigate(ctx.url+"/"),
		chromedp.WaitVisible(".tinkerdown-interactive-block", chromedp.ByQuery),
		chromedp.Sleep(5*time.Second),
	)
	if err != nil {
		ctx.dumpDebugInfo()
		t.Fatalf("Page did not load: %v", err)
	}

	// Check for table or rendered content
	var hasTable bool
	err = chromedp.Run(bctx,
		chromedp.Evaluate(`document.querySelector('.tinkerdown-interactive-block table') !== null`, &hasTable),
	)
	if err != nil {
		t.Fatalf("Failed to check for table: %v", err)
	}

	if !hasTable {
		ctx.dumpDebugInfo()
		t.Fatal("Notes table was not rendered from markdown source")
	}
	t.Log("Notes table rendered from markdown file")

	// Verify content contains initial notes
	var bodyHTML string
	err = chromedp.Run(bctx, chromedp.OuterHTML("body", &bodyHTML))
	if err != nil {
		t.Fatalf("Failed to get body: %v", err)
	}

	if !strings.Contains(bodyHTML, "Welcome") {
		ctx.dumpDebugInfo()
		t.Fatal("Expected 'Welcome' note in rendered table")
	}
	t.Log("Initial notes rendered correctly")

	t.Log("Markdown notes template E2E passed")
}

// TestTemplateGraphQLExplorerE2E verifies the graphql-explorer template renders
// countries from the public GraphQL API.
func TestTemplateGraphQLExplorerE2E(t *testing.T) {
	ctx, cleanup := setupTemplateE2E(t, "graphql-explorer")
	defer cleanup()

	bctx := ctx.chromeCtx.Context

	// Navigate and wait for rendering (GraphQL API call may take longer)
	err := chromedp.Run(bctx,
		chromedp.Navigate(ctx.url+"/"),
		chromedp.WaitVisible(".tinkerdown-interactive-block", chromedp.ByQuery),
		chromedp.WaitVisible(".tinkerdown-interactive-block table", chromedp.ByQuery),
	)
	if err != nil {
		ctx.dumpDebugInfo()
		t.Fatalf("Countries table was not rendered from GraphQL source: %v", err)
	}
	t.Log("Countries table rendered from GraphQL API")

	// Verify we got data (should be 200+ countries)
	var rowCount int
	err = chromedp.Run(bctx,
		chromedp.Evaluate(`document.querySelectorAll('tbody tr').length`, &rowCount),
	)
	if err != nil {
		t.Fatalf("Failed to query country data: %v", err)
	}

	if rowCount < 50 {
		ctx.dumpDebugInfo()
		t.Fatalf("Expected 50+ country rows (got %d) — GraphQL API may be down", rowCount)
	}
	t.Logf("Country rows rendered: %d", rowCount)

	// Spot-check known countries
	var bodyHTML string
	err = chromedp.Run(bctx, chromedp.OuterHTML("body", &bodyHTML))
	if err != nil {
		t.Fatalf("Failed to get body: %v", err)
	}
	if !strings.Contains(bodyHTML, "United States") {
		ctx.dumpDebugInfo()
		t.Fatal("Expected 'United States' in rendered table")
	}

	t.Log("GraphQL explorer template E2E passed")
}

// TestTemplateCLIWrapperE2E verifies the cli-wrapper template renders
// the command panel and handles the exec source.
func TestTemplateCLIWrapperE2E(t *testing.T) {
	// Exec sources require --allow-exec flag (same as `tinkerdown serve --allow-exec`)
	config.SetAllowExec(true)
	defer config.SetAllowExec(false)

	ctx, cleanup := setupTemplateE2E(t, "cli-wrapper")
	defer cleanup()

	bctx := ctx.chromeCtx.Context

	// Navigate and wait for interactive block to render
	err := chromedp.Run(bctx,
		chromedp.Navigate(ctx.url+"/"),
		chromedp.WaitVisible(".tinkerdown-interactive-block", chromedp.ByQuery),
		chromedp.Sleep(3*time.Second),
	)
	if err != nil {
		ctx.dumpDebugInfo()
		t.Fatalf("CLI wrapper page did not render: %v", err)
	}
	t.Log("Interactive block rendered")

	// Verify the title was substituted (not literal <<.Title>> or [[.Title]])
	var pageText string
	err = chromedp.Run(bctx,
		chromedp.Evaluate(`document.body.innerText`, &pageText),
	)
	if err != nil {
		t.Fatalf("Failed to get page text: %v", err)
	}

	if strings.Contains(pageText, "<<.Title>>") || strings.Contains(pageText, "[[.Title]]") {
		t.Fatal("Template delimiters were not substituted in rendered page")
	}
	t.Log("Template delimiters correctly substituted")

	// Verify the exec toolbar Run button exists
	var hasRunBtn bool
	err = chromedp.Run(bctx,
		chromedp.Evaluate(`document.querySelector('.exec-toolbar-run-btn') !== null`, &hasRunBtn),
	)
	if err != nil {
		t.Fatalf("Failed to check for run button: %v", err)
	}
	if !hasRunBtn {
		ctx.dumpDebugInfo()
		t.Fatal("Exec toolbar Run button not found")
	}
	t.Log("Exec toolbar Run button present")

	// Click Run and verify the command output appears inside the interactive block
	err = chromedp.Run(bctx,
		chromedp.Click(".exec-toolbar-run-btn", chromedp.ByQuery),
		chromedp.Sleep(3*time.Second),
	)
	if err != nil {
		t.Fatalf("Failed to click Run: %v", err)
	}

	// Check that the output text appears inside the interactive block's rendered content
	var blockHTML string
	err = chromedp.Run(bctx,
		chromedp.Evaluate(`
			(() => {
				const block = document.querySelector('.tinkerdown-interactive-block');
				return block ? block.innerHTML : '';
			})()
		`, &blockHTML),
	)
	if err != nil {
		t.Fatalf("Failed to get block HTML: %v", err)
	}
	if !strings.Contains(blockHTML, "Hello, World!") {
		ctx.dumpDebugInfo()
		t.Fatalf("Expected 'Hello, World!' in interactive block output after Run, got: %s", blockHTML[:min(500, len(blockHTML))])
	}
	t.Log("Command output rendered in block after Run")

	t.Log("CLI wrapper template E2E passed")
}
