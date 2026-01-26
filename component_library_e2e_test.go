//go:build !ci

package tinkerdown_test

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"net/http/httptest"

	"github.com/chromedp/cdproto/runtime"
	"github.com/chromedp/chromedp"
	tinkerdown "github.com/livetemplate/tinkerdown"
	"github.com/livetemplate/tinkerdown/internal/config"
	"github.com/livetemplate/tinkerdown/internal/server"
)

// TestAutoTableRendering tests both simple and rich table auto-generation modes
func TestAutoTableRendering(t *testing.T) {
	// Load config from test example
	cfg, err := config.LoadFromDir("examples/component-library-test")
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Create test server
	srv := server.NewWithConfig("examples/component-library-test", cfg)
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

	// Navigate and get the page content
	var htmlContent string
	err = chromedp.Run(ctx,
		chromedp.Navigate(url+"/"),
		chromedp.Sleep(15*time.Second), // Wait for all blocks to compile and initialize
		chromedp.OuterHTML("html", &htmlContent),
	)
	if err != nil {
		t.Fatalf("Failed to navigate: %v", err)
	}

	t.Logf("HTML content length: %d", len(htmlContent))

	// Test 1: Simple table renders with <thead> and <tbody>
	t.Run("simple_table_structure", func(t *testing.T) {
		if !strings.Contains(htmlContent, "<thead>") {
			t.Logf("HTML content (first 5000 chars): %s", htmlContent[:min(5000, len(htmlContent))])
			t.Fatal("Simple table missing <thead>")
		}
		if !strings.Contains(htmlContent, "<tbody>") {
			t.Fatal("Simple table missing <tbody>")
		}
		t.Log("Simple table structure verified")
	})

	// Test 2: Simple table renders user data
	t.Run("simple_table_data", func(t *testing.T) {
		if !strings.Contains(htmlContent, "Alice") {
			t.Logf("HTML content: %s", htmlContent)
			t.Fatal("Alice user not found in table")
		}
		if !strings.Contains(htmlContent, "alice@example.com") {
			t.Fatal("Alice email not found in table")
		}
		if !strings.Contains(htmlContent, "Bob") {
			t.Fatal("Bob user not found in table")
		}
		t.Log("User data rendered correctly")
	})

	// Test 3: Simple table with actions renders action buttons
	t.Run("simple_table_actions", func(t *testing.T) {
		if !strings.Contains(htmlContent, "lvt-click=\"delete\"") {
			t.Logf("Console logs: %v", consoleLogs)
			t.Fatal("Delete action button not found")
		}
		if !strings.Contains(htmlContent, "lvt-click=\"edit\"") {
			t.Fatal("Edit action button not found")
		}
		if !strings.Contains(htmlContent, ">Delete</button>") {
			t.Fatal("Delete button label not found")
		}
		t.Log("Action buttons rendered correctly")
	})

	// Test 4: Empty state message renders
	t.Run("empty_state_message", func(t *testing.T) {
		if !strings.Contains(htmlContent, "No users found") {
			t.Logf("HTML content: %s", htmlContent)
			t.Fatal("Empty state message 'No users found' not rendered")
		}
		t.Log("Empty state message rendered correctly")
	})

	// Test 5: Rich mode with lvt-datatable renders datatable component
	// NOTE: Rich mode is tested via unit tests in TestAutoTableGeneration.
	// The E2E example focuses on simple tables due to external SVG styling issues with datatable.
	t.Run("rich_datatable_component", func(t *testing.T) {
		t.Skip("Rich datatable E2E test skipped - external component has SVG sizing issues")
	})

	// Test 6: Rich mode has sorting capability
	t.Run("rich_datatable_sorting", func(t *testing.T) {
		t.Skip("Rich datatable E2E test skipped - external component has SVG sizing issues")
	})

	// Test 7: Select dropdown renders with options
	t.Run("select_dropdown", func(t *testing.T) {
		if !strings.Contains(htmlContent, "test-select") {
			t.Fatal("Select dropdown not found")
		}
		if !strings.Contains(htmlContent, "value=\"US\"") {
			t.Logf("HTML content: %s", htmlContent)
			t.Fatal("US option not found in select")
		}
		if !strings.Contains(htmlContent, "United States") {
			t.Fatal("United States label not found in select")
		}
		t.Log("Select dropdown rendered correctly")
	})

	// Test 8: Multiple simple tables on the same page
	t.Run("multiple_tables", func(t *testing.T) {
		tableCount := strings.Count(htmlContent, "<table")

		// We expect at least 4 simple tables (explicit columns, actions, empty state, auto-discovery)
		if tableCount < 4 {
			t.Logf("Expected at least 4 tables, found %d", tableCount)
			t.Fatal("Not enough tables rendered")
		}
		t.Logf("Found %d simple tables", tableCount)
	})

	// Test 9: Simple string array list (ul)
	t.Run("simple_list", func(t *testing.T) {
		if !strings.Contains(htmlContent, "test-tags") {
			t.Fatal("Tags list not found")
		}
		// Check for list items from tags.json
		if !strings.Contains(htmlContent, "<li>") {
			t.Fatal("No list items found")
		}
		t.Log("Simple list rendered correctly")
	})

	// Test 10: Object list with field
	t.Run("list_with_field", func(t *testing.T) {
		if !strings.Contains(htmlContent, "test-tasks") {
			t.Fatal("Tasks list not found")
		}
		t.Log("List with field rendered correctly")
	})

	// Test 11: List with actions
	t.Run("list_with_actions", func(t *testing.T) {
		if !strings.Contains(htmlContent, "test-actions-list") {
			t.Fatal("Actions list not found")
		}
		t.Log("List with actions rendered correctly")
	})

	// Test 12: Ordered list with empty state
	t.Run("ordered_list_empty", func(t *testing.T) {
		if !strings.Contains(htmlContent, "test-empty-list") {
			t.Fatal("Empty list not found")
		}
		if !strings.Contains(htmlContent, "No items available") {
			t.Fatal("Empty state message not found")
		}
		t.Log("Ordered list with empty state rendered correctly")
	})

	t.Log("All auto-rendering tests passed!")
}

// TestAutoTableGeneration tests the autoGenerateTableTemplate function directly
func TestAutoTableGeneration(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		contains    []string
		notContains []string
	}{
		{
			name:  "simple mode with explicit columns",
			input: `<table lvt-source="users" lvt-columns="name:Name,email:Email"></table>`,
			contains: []string{
				"<thead>",
				"<th>Name</th>",
				"<th>Email</th>",
				"{{range .Data}}",
				"{{.Name}}",
				"{{.Email}}",
			},
			notContains: []string{
				"lvt-columns=",
				"data-datatable",
				"lvt:datatable",
			},
		},
		{
			name:  "simple mode with actions",
			input: `<table lvt-source="users" lvt-columns="name:Name" lvt-actions="delete:Delete"></table>`,
			contains: []string{
				"<th>Actions</th>",
				`lvt-click="delete"`,
				`lvt-data-id="{{.Id}}"`,
				">Delete</button>",
			},
			notContains: []string{
				"lvt-actions=",
				"data-datatable",
			},
		},
		{
			name:  "simple mode with empty state",
			input: `<table lvt-source="users" lvt-columns="name:Name" lvt-empty="No items"></table>`,
			contains: []string{
				"No items",
				"{{if not .Data}}",
			},
			notContains: []string{
				"lvt-empty=",
			},
		},
		{
			name:  "rich mode with lvt-datatable",
			input: `<table lvt-source="users" lvt-columns="name:Name" lvt-datatable></table>`,
			contains: []string{
				`{{template "lvt:datatable:default:v1" .Table}}`,
			},
			notContains: []string{
				"<thead>",
				"<tbody>",
				"lvt-datatable",
			},
		},
		{
			name:  "preserves existing content",
			input: `<table lvt-source="users"><tbody>{{range .Data}}<tr><td>custom</td></tr>{{end}}</tbody></table>`,
			contains: []string{
				"custom",
				"<tbody>",
			},
			notContains: []string{
				"<thead>",
			},
		},
		{
			name:  "auto-discovery when no columns",
			input: `<table lvt-source="users"></table>`,
			contains: []string{
				"{{if .Data}}",
				"{{range $key, $_ := index .Data 0}}",
				"{{$key}}",
				"{{$value}}",
			},
		},
		{
			name:  "preserves extra attributes",
			input: `<table lvt-source="users" lvt-columns="name:Name" class="my-table" id="users-table"></table>`,
			contains: []string{
				`class="my-table"`,
				`id="users-table"`,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Parse the input as a page to trigger autoGenerateTableTemplate
			page, err := tinkerdown.ParseString(fmt.Sprintf("---\ntitle: test\nsources:\n  users:\n    type: json\n    file: test.json\n---\n```lvt\n%s\n```", tt.input))
			if err != nil {
				t.Fatalf("Failed to parse: %v", err)
			}

			// Get the generated content from the interactive block
			var generatedContent string
			for _, block := range page.InteractiveBlocks {
				generatedContent = block.Content
				break
			}

			t.Logf("Generated content:\n%s", generatedContent)

			// Check contains
			for _, want := range tt.contains {
				if !strings.Contains(generatedContent, want) {
					t.Errorf("Expected generated content to contain %q", want)
				}
			}

			// Check notContains
			for _, notWant := range tt.notContains {
				if strings.Contains(generatedContent, notWant) {
					t.Errorf("Expected generated content NOT to contain %q", notWant)
				}
			}
		})
	}
}

// TestAutoSelectGeneration tests the select auto-generation function
func TestAutoSelectGeneration(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		contains []string
	}{
		{
			name:  "with value and label fields",
			input: `<select lvt-source="countries" lvt-value="code" lvt-label="name"></select>`,
			contains: []string{
				"{{range .Data}}",
				`value="{{.Code}}"`,
				"{{.Name}}",
			},
		},
		{
			name:  "default fields",
			input: `<select lvt-source="items"></select>`,
			contains: []string{
				"{{range .Data}}",
				`value="{{.Id}}"`,
				"{{.Name}}",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Parse the input as a page to trigger autoGenerateSelectTemplate
			page, err := tinkerdown.ParseString(fmt.Sprintf("---\ntitle: test\nsources:\n  countries:\n    type: json\n    file: test.json\n  items:\n    type: json\n    file: test.json\n---\n```lvt\n%s\n```", tt.input))
			if err != nil {
				t.Fatalf("Failed to parse: %v", err)
			}

			// Get the generated content from the interactive block
			var generatedContent string
			for _, block := range page.InteractiveBlocks {
				generatedContent = block.Content
				break
			}

			t.Logf("Generated content:\n%s", generatedContent)

			// Check contains
			for _, want := range tt.contains {
				if !strings.Contains(generatedContent, want) {
					t.Errorf("Expected generated content to contain %q", want)
				}
			}
		})
	}
}

// TestAutoListGeneration tests the list auto-generation function
func TestAutoListGeneration(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		contains []string
	}{
		{
			name:  "simple string array",
			input: `<ul lvt-source="tags"></ul>`,
			contains: []string{
				"{{range .Data}}",
				"<li>",
				"{{.}}",
				"</li>",
			},
		},
		{
			name:  "with field",
			input: `<ul lvt-source="tasks" lvt-field="title"></ul>`,
			contains: []string{
				"{{range .Data}}",
				"<li>",
				"{{.Title}}",
				"</li>",
			},
		},
		{
			name:  "with actions",
			input: `<ul lvt-source="tasks" lvt-field="title" lvt-actions="delete:Delete"></ul>`,
			contains: []string{
				"{{range .Data}}",
				"{{.Title}}",
				`lvt-click="delete"`,
				`lvt-data-id="{{.Id}}"`,
				">Delete</button>",
			},
		},
		{
			name:  "with empty state",
			input: `<ul lvt-source="items" lvt-empty="No items yet"></ul>`,
			contains: []string{
				"{{if not .Data}}",
				"No items yet",
				"{{else}}",
				"{{range .Data}}",
				"{{end}}",
			},
		},
		{
			name:  "ordered list",
			input: `<ol lvt-source="steps"></ol>`,
			contains: []string{
				"<ol",
				"{{range .Data}}",
				"<li>",
				"{{.}}",
				"</li>",
				"</ol>",
			},
		},
		{
			name:  "preserves other attributes",
			input: `<ul lvt-source="tags" class="my-list" id="tag-list"></ul>`,
			contains: []string{
				`class="my-list"`,
				`id="tag-list"`,
				"{{range .Data}}",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Parse the input as a page to trigger autoGenerateListTemplate
			page, err := tinkerdown.ParseString(fmt.Sprintf("---\ntitle: test\nsources:\n  tags:\n    type: json\n    file: test.json\n  tasks:\n    type: json\n    file: test.json\n  items:\n    type: json\n    file: test.json\n  steps:\n    type: json\n    file: test.json\n---\n```lvt\n%s\n```", tt.input))
			if err != nil {
				t.Fatalf("Failed to parse: %v", err)
			}

			// Get the generated content from the interactive block
			var generatedContent string
			for _, block := range page.InteractiveBlocks {
				generatedContent = block.Content
				break
			}

			t.Logf("Generated content:\n%s", generatedContent)

			// Check contains
			for _, want := range tt.contains {
				if !strings.Contains(generatedContent, want) {
					t.Errorf("Expected generated content to contain %q", want)
				}
			}
		})
	}
}

// TestXSSPrevention verifies that user-provided strings are HTML-escaped
func TestXSSPrevention(t *testing.T) {
	tests := []struct {
		name          string
		input         string
		shouldContain []string // Escaped versions that should be in table cells/headers
	}{
		{
			name:  "XSS in empty message is escaped",
			input: `<table lvt-source="users" lvt-columns="name:Name" lvt-empty="<script>alert(1)</script>"></table>`,
			shouldContain: []string{
				"&lt;script&gt;alert(1)&lt;/script&gt;", // Escaped in empty state td
			},
		},
		{
			name:  "script tag in column label is escaped",
			input: `<table lvt-source="users" lvt-columns="name:Test<b>Bold</b>"></table>`,
			shouldContain: []string{
				"<th>Test&lt;b&gt;Bold&lt;/b&gt;</th>", // Escaped in th
			},
		},
		{
			name:  "ampersand in column label is escaped",
			input: `<table lvt-source="users" lvt-columns="name:Name & Title"></table>`,
			shouldContain: []string{
				"<th>Name &amp; Title</th>", // Escaped in th
			},
		},
		{
			name:  "quotes in action label are escaped",
			input: `<table lvt-source="users" lvt-columns="name:Name" lvt-actions="delete:Delete &quot;Item&quot;"></table>`,
			shouldContain: []string{
				"Delete &amp;quot;Item&amp;quot;</button>", // Escaped in button
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			page, err := tinkerdown.ParseString(fmt.Sprintf("---\ntitle: test\nsources:\n  users:\n    type: json\n    file: test.json\n---\n```lvt\n%s\n```", tt.input))
			if err != nil {
				t.Fatalf("Failed to parse: %v", err)
			}

			var generatedContent string
			for _, block := range page.InteractiveBlocks {
				generatedContent = block.Content
				break
			}

			t.Logf("Generated content:\n%s", generatedContent)

			// Verify escaped versions ARE present
			for _, want := range tt.shouldContain {
				if !strings.Contains(generatedContent, want) {
					t.Errorf("Expected escaped content to contain %q", want)
				}
			}
		})
	}
}

// Legacy test for backward compatibility
func TestComponentLibrary(t *testing.T) {
	// This test is kept for backward compatibility but delegates to the new test
	t.Run("delegates to TestAutoTableRendering", func(t *testing.T) {
		t.Skip("See TestAutoTableRendering for comprehensive tests")
	})
}
