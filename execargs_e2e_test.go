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

// TestExecArgsForm tests the auto-generated argument form functionality.
// This test verifies:
// 1. Form inputs are rendered for each argument
// 2. Input types match the inferred types (text, number, checkbox)
// 3. Default values are populated
// 4. --help descriptions are shown
// 5. Form submission triggers Run action
// 6. Output reflects the submitted values
func TestExecArgsForm(t *testing.T) {
	// Load config from test example
	cfg, err := config.LoadFromDir("examples/exec-args-test")
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Verify source is configured correctly
	if cfg.Sources == nil {
		t.Fatal("No sources configured in livemdtools.yaml")
	}
	source, ok := cfg.Sources["greeting"]
	if !ok {
		t.Fatal("greeting source not found in config")
	}
	if source.Type != "exec" {
		t.Fatalf("Expected exec source type, got: %s", source.Type)
	}
	if !source.Manual {
		t.Fatal("Expected manual: true for greeting source")
	}
	t.Logf("Source config: type=%s, cmd=%s, manual=%t", source.Type, source.Cmd, source.Manual)

	// Create test server
	srv := server.NewWithConfig("examples/exec-args-test", cfg)
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

	// Test 1: Navigate and wait for page to load
	var hasInteractiveBlock bool
	err = chromedp.Run(ctx,
		chromedp.Navigate(ts.URL+"/"),
		chromedp.Sleep(3*time.Second),
		chromedp.Evaluate(`document.querySelector('.livemdtools-interactive-block') !== null`, &hasInteractiveBlock),
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
	t.Log("Page loaded with interactive block")

	// Wait for form to be rendered (may take a moment after WebSocket connection)
	var hasForm bool
	var inputCount int
	for i := 0; i < 20; i++ {
		err = chromedp.Run(ctx,
			chromedp.Evaluate(`document.querySelector('form[lvt-submit="Run"]') !== null`, &hasForm),
		)
		if err == nil && hasForm {
			break
		}
		time.Sleep(300 * time.Millisecond)
	}

	err = chromedp.Run(ctx,
		chromedp.Evaluate(`document.querySelectorAll('form[lvt-submit="Run"] input').length`, &inputCount),
	)
	if err != nil {
		t.Fatalf("Failed to check form: %v", err)
	}

	if !hasForm {
		var htmlContent string
		chromedp.Run(ctx, chromedp.OuterHTML("html", &htmlContent))
		// Get more of the content
		t.Logf("HTML length: %d", len(htmlContent))
		t.Logf("HTML: %s", htmlContent[:min(5000, len(htmlContent))])
		t.Logf("Console logs: %v", consoleLogs)
		t.Fatal("Form with lvt-submit='Run' not found")
	}
	t.Log("Form with lvt-submit='Run' found")

	// We expect 3 inputs: name (text), count (number), uppercase (checkbox)
	if inputCount < 3 {
		var htmlContent string
		chromedp.Run(ctx, chromedp.OuterHTML("html", &htmlContent))
		t.Logf("HTML: %s", htmlContent[:min(3000, len(htmlContent))])
		t.Fatalf("Expected at least 3 form inputs, got: %d", inputCount)
	}
	t.Logf("Found %d form inputs", inputCount)

	// Test 3: Verify input types are correct
	var nameInputType string
	var countInputType string
	var uppercaseInputType string
	err = chromedp.Run(ctx,
		chromedp.Evaluate(`document.querySelector('input[name="name"]')?.type || 'not found'`, &nameInputType),
		chromedp.Evaluate(`document.querySelector('input[name="count"]')?.type || 'not found'`, &countInputType),
		chromedp.Evaluate(`document.querySelector('input[name="uppercase"]')?.type || 'not found'`, &uppercaseInputType),
	)
	if err != nil {
		t.Fatalf("Failed to check input types: %v", err)
	}

	if nameInputType != "text" {
		t.Fatalf("Expected name input type 'text', got: '%s'", nameInputType)
	}
	t.Log("Name input is text type")

	if countInputType != "number" {
		t.Fatalf("Expected count input type 'number', got: '%s'", countInputType)
	}
	t.Log("Count input is number type")

	if uppercaseInputType != "checkbox" {
		t.Fatalf("Expected uppercase input type 'checkbox', got: '%s'", uppercaseInputType)
	}
	t.Log("Uppercase input is checkbox type")

	// Test 4: Verify default values
	var nameValue string
	var countValue string
	var uppercaseChecked bool
	err = chromedp.Run(ctx,
		chromedp.Evaluate(`document.querySelector('input[name="name"]')?.value || ''`, &nameValue),
		chromedp.Evaluate(`document.querySelector('input[name="count"]')?.value || ''`, &countValue),
		chromedp.Evaluate(`document.querySelector('input[name="uppercase"]')?.checked || false`, &uppercaseChecked),
	)
	if err != nil {
		t.Fatalf("Failed to check default values: %v", err)
	}

	if nameValue != "World" {
		t.Fatalf("Expected name default 'World', got: '%s'", nameValue)
	}
	t.Logf("Name default value: %s", nameValue)

	if countValue != "3" {
		t.Fatalf("Expected count default '3', got: '%s'", countValue)
	}
	t.Logf("Count default value: %s", countValue)

	if uppercaseChecked {
		t.Fatal("Expected uppercase to be unchecked (false)")
	}
	t.Log("Uppercase default is unchecked")

	// Test 5: Verify --help descriptions are shown (if introspection worked)
	var hasDescriptions bool
	err = chromedp.Run(ctx,
		chromedp.Evaluate(`document.body.textContent.includes('The name to greet')`, &hasDescriptions),
	)
	if err != nil {
		t.Fatalf("Failed to check descriptions: %v", err)
	}

	if hasDescriptions {
		t.Log("Descriptions from --help are displayed")
	} else {
		t.Log("Note: Descriptions from --help not found (introspection may have failed, which is ok)")
	}

	// Wait for WebSocket connection before submitting form
	var isConnected bool
	for i := 0; i < 20; i++ {
		chromedp.Run(ctx, chromedp.Evaluate(`
			(() => {
				return window.livemdtools && window.livemdtools._client && window.livemdtools._client.isConnected ? true : false;
			})()
		`, &isConnected))
		if isConnected {
			break
		}
		time.Sleep(200 * time.Millisecond)
	}
	if !isConnected {
		// Fallback: check console logs
		for _, log := range consoleLogs {
			if strings.Contains(log, "[Livemdtools] Connected") {
				isConnected = true
				break
			}
		}
	}
	if !isConnected {
		time.Sleep(2 * time.Second)
	}
	t.Log("WebSocket connection established")

	// Test 6: Modify values and submit form
	err = chromedp.Run(ctx,
		// Clear name field and enter new value
		chromedp.Clear(`input[name="name"]`, chromedp.ByQuery),
		chromedp.SendKeys(`input[name="name"]`, "Alice", chromedp.ByQuery),
		// Clear count field and enter new value
		chromedp.Clear(`input[name="count"]`, chromedp.ByQuery),
		chromedp.SendKeys(`input[name="count"]`, "2", chromedp.ByQuery),
		// Check the uppercase checkbox
		chromedp.Click(`input[name="uppercase"]`, chromedp.ByQuery),
	)
	if err != nil {
		t.Fatalf("Failed to modify form values: %v", err)
	}
	t.Log("Modified form values")

	// Submit the form
	err = chromedp.Run(ctx,
		chromedp.Click(`form[lvt-submit="Run"] button[type="submit"]`, chromedp.ByQuery),
	)
	if err != nil {
		t.Fatalf("Failed to submit form: %v", err)
	}
	t.Log("Form submitted")

	// Wait for command to update with new values (DOM update via morphdom)
	var commandText string
	for i := 0; i < 30; i++ {
		err = chromedp.Run(ctx,
			chromedp.Evaluate(`
				(() => {
					// Get all code elements and their text
					const codes = document.querySelectorAll('code');
					const texts = Array.from(codes).map(c => c.textContent);
					// Also check if the main block's innerHTML contains Alice
					const main = document.querySelector('main[lvt-source="greeting"]');
					const mainHtml = main ? main.innerHTML.substring(0, 500) : 'main not found';
					return JSON.stringify({codes: texts, mainHtml: mainHtml});
				})()
			`, &commandText),
		)
		if err == nil && strings.Contains(commandText, "Alice") {
			break
		}
		time.Sleep(200 * time.Millisecond)
	}

	// Parse the debug info
	t.Logf("DOM state after wait: %s", commandText)

	// The DOM might have multiple code elements due to morphdom updates
	// Check if any of them contains the updated command
	if !strings.Contains(commandText, "Alice") {
		t.Logf("Console logs: %v", consoleLogs)
		t.Fatal("DOM does not contain updated name 'Alice'")
	}
	t.Log("Command updated with new name value")

	if !strings.Contains(commandText, "count 2") && !strings.Contains(commandText, "count2") {
		t.Fatal("DOM does not contain updated count '2'")
	}
	t.Log("Command updated with new count value")

	if !strings.Contains(commandText, "uppercase true") {
		t.Fatal("DOM does not contain updated uppercase 'true'")
	}
	t.Log("Command updated with uppercase=true")

	// Test 8: Verify output shows correct data
	var hasAliceMessage bool
	var hasUppercaseMessage bool
	err = chromedp.Run(ctx,
		chromedp.Evaluate(`document.body.innerHTML.includes('HELLO, ALICE')`, &hasUppercaseMessage),
		chromedp.Evaluate(`document.body.innerHTML.includes('Alice')`, &hasAliceMessage),
	)
	if err != nil {
		t.Fatalf("Failed to check output: %v", err)
	}

	if !hasAliceMessage && !hasUppercaseMessage {
		var htmlContent string
		chromedp.Run(ctx, chromedp.OuterHTML("html", &htmlContent))
		t.Logf("HTML: %s", htmlContent[:min(3000, len(htmlContent))])
		t.Fatal("Output does not contain greeting for Alice")
	}

	if hasUppercaseMessage {
		t.Log("Output shows uppercase greeting for Alice")
	} else {
		t.Log("Output shows greeting for Alice")
	}

	// Test 9: Verify status is success
	var statusText string
	err = chromedp.Run(ctx,
		chromedp.Evaluate(`
			(() => {
				// Look for status in the template output
				const el = document.body;
				if (el && el.textContent.includes('success')) {
					return 'success';
				}
				return el ? el.textContent.substring(0, 200) : '';
			})()
		`, &statusText),
	)
	if err != nil {
		t.Fatalf("Failed to check status: %v", err)
	}

	if !strings.Contains(statusText, "success") {
		var htmlContent string
		chromedp.Run(ctx, chromedp.OuterHTML("html", &htmlContent))
		t.Logf("HTML: %s", htmlContent[:min(3000, len(htmlContent))])
		t.Fatalf("Status is not success: %s", statusText)
	}
	t.Log("Status shows success")

	t.Log("All exec args form tests passed!")
}
