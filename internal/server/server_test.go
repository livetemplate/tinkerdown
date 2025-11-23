package server

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestMdToPattern(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"index.md", "/"},
		{"counter.md", "/counter"},
		{"tutorials/intro.md", "/tutorials/intro"},
		{"tutorials/index.md", "/tutorials/"},
		{"advanced/state.md", "/advanced/state"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := mdToPattern(tt.input)
			if got != tt.want {
				t.Errorf("mdToPattern(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestServerDiscover(t *testing.T) {
	// Create temp directory with test files
	tmpDir := t.TempDir()

	// Create test markdown files
	files := map[string]string{
		"index.md": `---
title: "Home"
---
# Home Page`,
		"counter.md": `---
title: "Counter"
---
# Counter Tutorial`,
		"tutorials/intro.md": `---
title: "Introduction"
---
# Introduction`,
		"_drafts/draft.md": `---
title: "Draft"
---
# Draft (should be ignored)`,
	}

	for path, content := range files {
		fullPath := filepath.Join(tmpDir, path)
		dir := filepath.Dir(fullPath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatalf("Failed to create directory: %v", err)
		}
		if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to write file: %v", err)
		}
	}

	// Create server and discover
	srv := New(tmpDir)
	if err := srv.Discover(); err != nil {
		t.Fatalf("Discover() error: %v", err)
	}

	// Check discovered routes
	routes := srv.Routes()
	if len(routes) != 3 { // index, counter, tutorials/intro (not _drafts)
		t.Errorf("Expected 3 routes, got %d", len(routes))
		for _, r := range routes {
			t.Logf("  Route: %s -> %s", r.Pattern, r.FilePath)
		}
	}

	// Check specific routes exist
	patterns := make(map[string]bool)
	for _, r := range routes {
		patterns[r.Pattern] = true
	}

	if !patterns["/"] {
		t.Error("Expected route / not found")
	}
	if !patterns["/counter"] {
		t.Error("Expected route /counter not found")
	}
	if !patterns["/tutorials/intro"] {
		t.Error("Expected route /tutorials/intro not found")
	}
	if patterns["/_drafts/draft"] {
		t.Error("Route /_drafts/draft should not exist (underscore prefix)")
	}
}

func TestServerServeHTTP(t *testing.T) {
	// Create temp directory with test file
	tmpDir := t.TempDir()

	content := `---
title: "Test Page"
---
# Test Content

This is a test page.`

	if err := os.WriteFile(filepath.Join(tmpDir, "test.md"), []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write file: %v", err)
	}

	// Create and discover
	srv := New(tmpDir)
	if err := srv.Discover(); err != nil {
		t.Fatalf("Discover() error: %v", err)
	}

	// Test HTTP request
	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()

	srv.ServeHTTP(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Status = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	body := w.Body.String()
	if body == "" {
		t.Error("Response body is empty")
	}

	// Check that HTML contains title
	if !contains(body, "Test Page") {
		t.Error("Response does not contain page title")
	}

	// Check that HTML contains content
	if !contains(body, "Test Content") {
		t.Error("Response does not contain page content")
	}
}

func TestServerNotFound(t *testing.T) {
	tmpDir := t.TempDir()

	srv := New(tmpDir)
	if err := srv.Discover(); err != nil {
		t.Fatalf("Discover() error: %v", err)
	}

	req := httptest.NewRequest("GET", "/nonexistent", nil)
	w := httptest.NewRecorder()

	srv.ServeHTTP(w, req)

	resp := w.Result()
	// Server now redirects unknown pages to home instead of returning 404
	if resp.StatusCode != http.StatusSeeOther && resp.StatusCode != http.StatusNotFound {
		t.Errorf("Status = %d, want %d or %d", resp.StatusCode, http.StatusSeeOther, http.StatusNotFound)
	}
}

func TestSortRoutes(t *testing.T) {
	routes := []*Route{
		{Pattern: "/counter"},
		{Pattern: "/"},
		{Pattern: "/tutorials/intro"},
		{Pattern: "/tutorials/"},
		{Pattern: "/advanced"},
	}

	sortRoutes(routes)

	// Root should be first
	if routes[0].Pattern != "/" {
		t.Errorf("First route = %s, want /", routes[0].Pattern)
	}

	// Find position of directory index
	tutorialsIdx := -1
	for i, r := range routes {
		if r.Pattern == "/tutorials/" {
			tutorialsIdx = i
			break
		}
	}

	// Check that /tutorials/ comes before /tutorials/intro
	introIdx := -1
	for i, r := range routes {
		if r.Pattern == "/tutorials/intro" {
			introIdx = i
			break
		}
	}

	if tutorialsIdx != -1 && introIdx != -1 && tutorialsIdx > introIdx {
		t.Error("/tutorials/ should come before /tutorials/intro")
	}
}

func contains(s, substr string) bool {
	return len(s) > 0 && len(substr) > 0 && len(s) >= len(substr) &&
		(s == substr || findSubstring(s, substr))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
