package livepage_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/chromedp/cdproto/runtime"
	"github.com/chromedp/chromedp"
	"github.com/livetemplate/livepage/internal/server"
)

// TestPlaygroundPageLoads tests that the playground page loads correctly.
func TestPlaygroundPageLoads(t *testing.T) {
	// Create test server
	srv := server.New("examples/autopersist-test")
	if err := srv.Discover(); err != nil {
		t.Fatalf("Failed to discover pages: %v", err)
	}

	ts := httptest.NewServer(srv)
	defer ts.Close()

	// Test the playground page loads
	resp, err := http.Get(ts.URL + "/playground")
	if err != nil {
		t.Fatalf("Failed to fetch playground page: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Expected status 200, got %d", resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)
	bodyStr := string(body)

	// Verify key elements are present
	if !strings.Contains(bodyStr, "LivePage Playground") {
		t.Fatal("Playground page missing title")
	}
	if !strings.Contains(bodyStr, "editor") {
		t.Fatal("Playground page missing editor element")
	}
	if !strings.Contains(bodyStr, "previewFrame") {
		t.Fatal("Playground page missing preview frame")
	}

	t.Log("✓ Playground page loads correctly")
}

// TestPlaygroundRenderAPI tests the /playground/render API endpoint.
func TestPlaygroundRenderAPI(t *testing.T) {
	// Create test server
	srv := server.New("examples/autopersist-test")
	if err := srv.Discover(); err != nil {
		t.Fatalf("Failed to discover pages: %v", err)
	}

	ts := httptest.NewServer(srv)
	defer ts.Close()

	// The lvt block needs lvt-persist to auto-generate a server block
	markdown := `---
title: "Test App"
---

# Test App

` + "```lvt" + `
<div class="p-4">
    <h1 class="text-xl">Hello World</h1>
    <form lvt-submit="save" lvt-persist="items">
        <input type="text" name="title" required>
        <button type="submit">Add</button>
    </form>
    {{range .Items}}
    <div>{{.Title}}</div>
    {{end}}
</div>
` + "```"

	// Send render request
	reqBody, _ := json.Marshal(map[string]string{
		"markdown": markdown,
	})

	resp, err := http.Post(ts.URL+"/playground/render", "application/json", bytes.NewReader(reqBody))
	if err != nil {
		t.Fatalf("Failed to send render request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("Expected status 200, got %d: %s", resp.StatusCode, string(body))
	}

	// Parse response
	var renderResp struct {
		SessionID string `json:"sessionId"`
		Error     string `json:"error"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&renderResp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if renderResp.SessionID == "" {
		t.Fatal("Expected session ID in response")
	}
	t.Logf("✓ Received session ID: %s", renderResp.SessionID)

	// Test preview endpoint
	previewResp, err := http.Get(ts.URL + "/playground/preview/" + renderResp.SessionID)
	if err != nil {
		t.Fatalf("Failed to fetch preview: %v", err)
	}
	defer previewResp.Body.Close()

	if previewResp.StatusCode != http.StatusOK {
		t.Fatalf("Expected preview status 200, got %d", previewResp.StatusCode)
	}

	previewBody, _ := io.ReadAll(previewResp.Body)
	previewStr := string(previewBody)

	// The initial HTML has the static title and markdown heading
	// The interactive content (Hello World) is rendered via WebSocket
	if !strings.Contains(previewStr, "Test App") {
		t.Fatal("Preview missing page title")
	}
	if !strings.Contains(previewStr, "livepage-interactive-block") {
		t.Fatal("Preview missing interactive block container")
	}
	// Check that PicoCSS and LivePage client are included
	if !strings.Contains(previewStr, "picocss") {
		t.Fatal("Preview missing PicoCSS")
	}
	if !strings.Contains(previewStr, "livepage-client.js") {
		t.Fatal("Preview missing LivePage client JS")
	}

	t.Log("✓ Preview renders correctly")
}

// TestPlaygroundRenderAPIValidation tests validation in the render API.
func TestPlaygroundRenderAPIValidation(t *testing.T) {
	// Create test server
	srv := server.New("examples/autopersist-test")
	if err := srv.Discover(); err != nil {
		t.Fatalf("Failed to discover pages: %v", err)
	}

	ts := httptest.NewServer(srv)
	defer ts.Close()

	tests := []struct {
		name       string
		body       string
		wantStatus int
	}{
		{
			name:       "empty body",
			body:       `{}`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "empty markdown",
			body:       `{"markdown": ""}`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "whitespace only",
			body:       `{"markdown": "   "}`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "invalid json",
			body:       `{invalid}`,
			wantStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := http.Post(ts.URL+"/playground/render", "application/json", strings.NewReader(tt.body))
			if err != nil {
				t.Fatalf("Failed to send request: %v", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != tt.wantStatus {
				body, _ := io.ReadAll(resp.Body)
				t.Fatalf("Expected status %d, got %d: %s", tt.wantStatus, resp.StatusCode, string(body))
			}
		})
	}
	t.Log("✓ Validation tests passed")
}

// TestPlaygroundSessionExpiry tests that invalid session IDs return 404.
func TestPlaygroundSessionExpiry(t *testing.T) {
	// Create test server
	srv := server.New("examples/autopersist-test")
	if err := srv.Discover(); err != nil {
		t.Fatalf("Failed to discover pages: %v", err)
	}

	ts := httptest.NewServer(srv)
	defer ts.Close()

	// Try to access non-existent session
	resp, err := http.Get(ts.URL + "/playground/preview/nonexistent-session-id")
	if err != nil {
		t.Fatalf("Failed to fetch preview: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("Expected status 404, got %d", resp.StatusCode)
	}

	t.Log("✓ Invalid session returns 404")
}

// TestPlaygroundE2E performs an end-to-end browser test of the playground.
func TestPlaygroundE2E(t *testing.T) {
	// Create test server
	srv := server.New("examples/autopersist-test")
	if err := srv.Discover(); err != nil {
		t.Fatalf("Failed to discover pages: %v", err)
	}

	ts := httptest.NewServer(srv)
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

	// Collect console logs
	var consoleLogs []string
	chromedp.ListenTarget(ctx, func(ev interface{}) {
		if ev, ok := ev.(*runtime.EventConsoleAPICalled); ok {
			for _, arg := range ev.Args {
				consoleLogs = append(consoleLogs, fmt.Sprintf("[Console] %s", arg.Value))
			}
		}
	})

	t.Logf("Test server URL: %s", ts.URL)

	// Test 1: Navigate to playground
	var htmlContent string
	var hasEditor bool
	var hasPreviewFrame bool
	err := chromedp.Run(ctx,
		chromedp.Navigate(ts.URL+"/playground"),
		chromedp.Sleep(500*time.Millisecond),
		chromedp.Evaluate(`document.querySelector('#editor') !== null`, &hasEditor),
		chromedp.Evaluate(`document.querySelector('#previewFrame') !== null`, &hasPreviewFrame),
	)
	if err != nil {
		t.Fatalf("Failed to navigate: %v", err)
	}

	if !hasEditor {
		chromedp.Run(ctx, chromedp.OuterHTML("html", &htmlContent))
		t.Logf("HTML: %s", htmlContent[:min(2000, len(htmlContent))])
		t.Fatal("Playground editor not found")
	}
	t.Log("✓ Playground editor found")

	if !hasPreviewFrame {
		t.Fatal("Playground preview frame not found")
	}
	t.Log("✓ Playground preview frame found")

	// Test 2: Enter markdown and run preview
	// The lvt block needs lvt-persist to auto-generate a server block
	testMarkdown := `---
title: "E2E Test App"
---

# E2E Test

` + "```lvt" + `
<div class="test-content p-4">
    <h1>E2E Test Content</h1>
    <p class="test-message">This is a test message</p>
    <form lvt-submit="save" lvt-persist="testdata">
        <input type="text" name="content" required>
        <button type="submit">Add</button>
    </form>
    {{range .Testdata}}
    <div>{{.Content}}</div>
    {{end}}
</div>
` + "```"

	err = chromedp.Run(ctx,
		// Clear and set editor content
		chromedp.SetValue("#editor", testMarkdown, chromedp.ByQuery),
		chromedp.Sleep(200*time.Millisecond),
	)
	if err != nil {
		t.Fatalf("Failed to set editor content: %v", err)
	}
	t.Log("✓ Editor content set")

	// Test 3: Click Run button
	err = chromedp.Run(ctx,
		chromedp.Click("#runBtn", chromedp.ByQuery),
		chromedp.Sleep(3*time.Second), // Wait for render and preview load
	)
	if err != nil {
		t.Fatalf("Failed to click Run button: %v", err)
	}
	t.Log("✓ Run button clicked")

	// Test 4: Verify preview loaded (check iframe has src)
	var previewSrc string
	var previewHidden bool
	err = chromedp.Run(ctx,
		chromedp.AttributeValue("#previewFrame", "src", &previewSrc, nil),
		chromedp.Evaluate(`document.querySelector('#previewFrame').classList.contains('hidden')`, &previewHidden),
	)
	if err != nil {
		t.Fatalf("Failed to check preview: %v", err)
	}

	if previewSrc == "" {
		t.Logf("Console logs: %v", consoleLogs)
		t.Fatal("Preview iframe has no src - preview did not load")
	}
	t.Logf("✓ Preview src: %s", previewSrc)

	if previewHidden {
		t.Fatal("Preview iframe is hidden - should be visible")
	}
	t.Log("✓ Preview iframe is visible")

	// Test 5: Verify preview URL contains session ID
	if !strings.Contains(previewSrc, "/playground/preview/") {
		t.Fatalf("Preview src doesn't contain expected path: %s", previewSrc)
	}
	t.Log("✓ Preview URL is correct")

	t.Log("✅ All playground E2E tests passed!")
}
