# GraphQL Source Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add a GraphQL source type that fetches data from GraphQL APIs using external query files.

**Architecture:** HTTP POST to GraphQL endpoint with query + variables, extract array from nested response using dot-path, reuse existing retry/circuit breaker infrastructure.

**Tech Stack:** Go, net/http, encoding/json, existing source patterns

---

## Task 1: Add Config Fields

**Files:**
- Modify: `internal/config/config.go`

**Step 1: Add new fields to SourceConfig struct**

Find the SourceConfig struct and add these fields after the existing ones:

```go
// Add after Path field (around line 37):
QueryFile   string                 `yaml:"query_file,omitempty"`   // For graphql: path to .graphql file
Variables   map[string]interface{} `yaml:"variables,omitempty"`    // For graphql: query variables
ResultPath  string                 `yaml:"result_path,omitempty"`  // For graphql: dot-path to extract array
```

**Step 2: Run existing config tests**

Run: `GOWORK=off go test ./internal/config/... -v`
Expected: PASS (no breaking changes)

**Step 3: Commit**

```bash
git add internal/config/config.go
git commit -m "feat(config): add GraphQL source fields"
```

---

## Task 2: Path Extraction Utility

**Files:**
- Create: `internal/source/graphql.go`
- Create: `internal/source/graphql_test.go`

**Step 1: Write failing tests for extractPath**

```go
// internal/source/graphql_test.go
package source

import (
	"testing"
)

func TestExtractPath_Simple(t *testing.T) {
	data := map[string]interface{}{
		"users": []interface{}{
			map[string]interface{}{"name": "Alice"},
			map[string]interface{}{"name": "Bob"},
		},
	}

	result, err := extractPath(data, "users")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 2 {
		t.Errorf("expected 2 items, got %d", len(result))
	}
	if result[0]["name"] != "Alice" {
		t.Errorf("expected Alice, got %v", result[0]["name"])
	}
}

func TestExtractPath_Nested(t *testing.T) {
	data := map[string]interface{}{
		"repository": map[string]interface{}{
			"issues": map[string]interface{}{
				"nodes": []interface{}{
					map[string]interface{}{"title": "Bug"},
					map[string]interface{}{"title": "Feature"},
				},
			},
		},
	}

	result, err := extractPath(data, "repository.issues.nodes")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 2 {
		t.Errorf("expected 2 items, got %d", len(result))
	}
	if result[0]["title"] != "Bug" {
		t.Errorf("expected Bug, got %v", result[0]["title"])
	}
}

func TestExtractPath_NotFound(t *testing.T) {
	data := map[string]interface{}{
		"users": []interface{}{},
	}

	_, err := extractPath(data, "nonexistent.path")
	if err == nil {
		t.Error("expected error for nonexistent path")
	}
}

func TestExtractPath_NotArray(t *testing.T) {
	data := map[string]interface{}{
		"user": map[string]interface{}{"name": "Alice"},
	}

	_, err := extractPath(data, "user")
	if err == nil {
		t.Error("expected error when path doesn't resolve to array")
	}
}

func TestExtractPath_EmptyPath(t *testing.T) {
	data := map[string]interface{}{
		"items": []interface{}{
			map[string]interface{}{"id": 1},
		},
	}

	_, err := extractPath(data, "")
	if err == nil {
		t.Error("expected error for empty path")
	}
}
```

**Step 2: Run tests to verify they fail**

Run: `GOWORK=off go test ./internal/source/... -v -run TestExtractPath`
Expected: FAIL (extractPath not defined)

**Step 3: Implement extractPath function**

```go
// internal/source/graphql.go
package source

import (
	"fmt"
	"strings"
)

// extractPath extracts an array from nested data using dot-notation path.
// Example: "repository.issues.nodes" extracts data["repository"]["issues"]["nodes"]
func extractPath(data map[string]interface{}, path string) ([]map[string]interface{}, error) {
	if path == "" {
		return nil, fmt.Errorf("result_path is required")
	}

	parts := strings.Split(path, ".")
	current := interface{}(data)

	for _, part := range parts {
		switch v := current.(type) {
		case map[string]interface{}:
			var ok bool
			current, ok = v[part]
			if !ok {
				return nil, fmt.Errorf("path '%s' not found at '%s'", path, part)
			}
		default:
			return nil, fmt.Errorf("path '%s' cannot traverse non-object at '%s'", path, part)
		}
	}

	// Convert to []map[string]interface{}
	arr, ok := current.([]interface{})
	if !ok {
		return nil, fmt.Errorf("path '%s' does not resolve to an array", path)
	}

	result := make([]map[string]interface{}, 0, len(arr))
	for _, item := range arr {
		if m, ok := item.(map[string]interface{}); ok {
			result = append(result, m)
		}
	}

	return result, nil
}
```

**Step 4: Run tests to verify they pass**

Run: `GOWORK=off go test ./internal/source/... -v -run TestExtractPath`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/source/graphql.go internal/source/graphql_test.go
git commit -m "feat(source): add extractPath utility for GraphQL responses"
```

---

## Task 3: GraphQL Error Type

**Files:**
- Modify: `internal/source/errors.go`
- Modify: `internal/source/graphql_test.go`

**Step 1: Add GraphQLError type to errors.go**

```go
// Add to internal/source/errors.go after existing error types:

// GraphQLError represents an error returned by a GraphQL API
type GraphQLError struct {
	Source  string
	Message string
	Path    []string
}

func (e *GraphQLError) Error() string {
	if len(e.Path) > 0 {
		return fmt.Sprintf("graphql error in source '%s' at %v: %s", e.Source, e.Path, e.Message)
	}
	return fmt.Sprintf("graphql error in source '%s': %s", e.Source, e.Message)
}
```

**Step 2: Run existing error tests**

Run: `GOWORK=off go test ./internal/source/... -v -run TestError`
Expected: PASS

**Step 3: Commit**

```bash
git add internal/source/errors.go
git commit -m "feat(source): add GraphQLError type"
```

---

## Task 4: GraphQL Source Struct and Constructor

**Files:**
- Modify: `internal/source/graphql.go`
- Modify: `internal/source/graphql_test.go`

**Step 1: Write failing test for NewGraphQLSource**

```go
// Add to internal/source/graphql_test.go:

func TestNewGraphQLSource_Valid(t *testing.T) {
	cfg := config.SourceConfig{
		Type:       "graphql",
		URL:        "https://api.example.com/graphql",
		QueryFile:  "queries/test.graphql",
		ResultPath: "data.users",
	}

	src, err := NewGraphQLSource("test", cfg, "/tmp")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if src.Name() != "test" {
		t.Errorf("expected name 'test', got '%s'", src.Name())
	}
}

func TestNewGraphQLSource_MissingURL(t *testing.T) {
	cfg := config.SourceConfig{
		Type:       "graphql",
		QueryFile:  "queries/test.graphql",
		ResultPath: "data.users",
	}

	_, err := NewGraphQLSource("test", cfg, "/tmp")
	if err == nil {
		t.Error("expected error for missing URL")
	}
}

func TestNewGraphQLSource_MissingQueryFile(t *testing.T) {
	cfg := config.SourceConfig{
		Type:       "graphql",
		URL:        "https://api.example.com/graphql",
		ResultPath: "data.users",
	}

	_, err := NewGraphQLSource("test", cfg, "/tmp")
	if err == nil {
		t.Error("expected error for missing query_file")
	}
}

func TestNewGraphQLSource_MissingResultPath(t *testing.T) {
	cfg := config.SourceConfig{
		Type:       "graphql",
		URL:        "https://api.example.com/graphql",
		QueryFile:  "queries/test.graphql",
	}

	_, err := NewGraphQLSource("test", cfg, "/tmp")
	if err == nil {
		t.Error("expected error for missing result_path")
	}
}
```

**Step 2: Run tests to verify they fail**

Run: `GOWORK=off go test ./internal/source/... -v -run TestNewGraphQLSource`
Expected: FAIL (NewGraphQLSource not defined)

**Step 3: Implement GraphQLSource struct and constructor**

```go
// Add to internal/source/graphql.go after extractPath:

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/livetemplate/tinkerdown/internal/config"
)

// GraphQLSource fetches data from a GraphQL API endpoint
type GraphQLSource struct {
	name           string
	url            string
	queryFile      string
	variables      map[string]interface{}
	resultPath     string
	headers        map[string]string
	client         *http.Client
	retryConfig    RetryConfig
	circuitBreaker *CircuitBreaker
	siteDir        string
}

// NewGraphQLSource creates a new GraphQL API source
func NewGraphQLSource(name string, cfg config.SourceConfig, siteDir string) (*GraphQLSource, error) {
	if cfg.URL == "" {
		return nil, &ValidationError{Source: name, Field: "url", Reason: "url is required"}
	}
	if cfg.QueryFile == "" {
		return nil, &ValidationError{Source: name, Field: "query_file", Reason: "query_file is required"}
	}
	if cfg.ResultPath == "" {
		return nil, &ValidationError{Source: name, Field: "result_path", Reason: "result_path is required"}
	}

	// Expand environment variables in URL
	url := os.ExpandEnv(cfg.URL)

	// Expand environment variables in variables
	variables := make(map[string]interface{})
	for k, v := range cfg.Variables {
		if s, ok := v.(string); ok {
			variables[k] = os.ExpandEnv(s)
		} else {
			variables[k] = v
		}
	}

	// Parse headers from options (same as REST source)
	headers := make(map[string]string)
	if cfg.Options != nil {
		if headerStr := cfg.Options["headers"]; headerStr != "" {
			for _, h := range strings.Split(headerStr, ",") {
				parts := strings.SplitN(h, ":", 2)
				if len(parts) == 2 {
					key := strings.TrimSpace(parts[0])
					value := strings.TrimSpace(parts[1])
					headers[key] = os.ExpandEnv(value)
				}
			}
		}
		if authHeader := cfg.Options["auth_header"]; authHeader != "" {
			headers["Authorization"] = os.ExpandEnv(authHeader)
		}
		if apiKey := cfg.Options["api_key"]; apiKey != "" {
			headers["X-API-Key"] = os.ExpandEnv(apiKey)
		}
	}

	// Get timeout from config or default
	timeout := cfg.GetTimeout()

	// Build retry config
	retryConfig := RetryConfig{
		MaxRetries: cfg.GetRetryMaxRetries(),
		BaseDelay:  cfg.GetRetryBaseDelay(),
		MaxDelay:   cfg.GetRetryMaxDelay(),
		Multiplier: 2.0,
		EnableLog:  true,
	}

	// Create circuit breaker
	cbConfig := DefaultCircuitBreakerConfig()
	circuitBreaker := NewCircuitBreaker(name, cbConfig)

	return &GraphQLSource{
		name:           name,
		url:            url,
		queryFile:      cfg.QueryFile,
		variables:      variables,
		resultPath:     cfg.ResultPath,
		headers:        headers,
		retryConfig:    retryConfig,
		circuitBreaker: circuitBreaker,
		siteDir:        siteDir,
		client: &http.Client{
			Timeout: timeout,
		},
	}, nil
}

// Name returns the source identifier
func (s *GraphQLSource) Name() string {
	return s.name
}

// Close is a no-op for GraphQL sources
func (s *GraphQLSource) Close() error {
	return nil
}
```

**Step 4: Add config import to graphql_test.go**

```go
// Update imports in internal/source/graphql_test.go:
import (
	"testing"

	"github.com/livetemplate/tinkerdown/internal/config"
)
```

**Step 5: Run tests to verify they pass**

Run: `GOWORK=off go test ./internal/source/... -v -run TestNewGraphQLSource`
Expected: PASS

**Step 6: Commit**

```bash
git add internal/source/graphql.go internal/source/graphql_test.go
git commit -m "feat(source): add GraphQLSource struct and constructor"
```

---

## Task 5: Implement Fetch Method

**Files:**
- Modify: `internal/source/graphql.go`
- Modify: `internal/source/graphql_test.go`

**Step 1: Write failing test for Fetch with mock server**

```go
// Add to internal/source/graphql_test.go:

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/livetemplate/tinkerdown/internal/config"
)

func TestGraphQLSource_Fetch(t *testing.T) {
	// Create mock GraphQL server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request
		if r.Method != "POST" {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("expected application/json content type")
		}

		// Return mock response
		resp := map[string]interface{}{
			"data": map[string]interface{}{
				"users": []interface{}{
					map[string]interface{}{"id": "1", "name": "Alice"},
					map[string]interface{}{"id": "2", "name": "Bob"},
				},
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	// Create temp query file
	tmpDir := t.TempDir()
	queryPath := filepath.Join(tmpDir, "query.graphql")
	os.WriteFile(queryPath, []byte("query { users { id name } }"), 0644)

	cfg := config.SourceConfig{
		Type:       "graphql",
		URL:        server.URL,
		QueryFile:  "query.graphql",
		ResultPath: "users",
	}

	src, err := NewGraphQLSource("test", cfg, tmpDir)
	if err != nil {
		t.Fatalf("failed to create source: %v", err)
	}

	result, err := src.Fetch(context.Background())
	if err != nil {
		t.Fatalf("fetch failed: %v", err)
	}

	if len(result) != 2 {
		t.Errorf("expected 2 users, got %d", len(result))
	}
	if result[0]["name"] != "Alice" {
		t.Errorf("expected Alice, got %v", result[0]["name"])
	}
}

func TestGraphQLSource_FetchWithVariables(t *testing.T) {
	var receivedVars map[string]interface{}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Query     string                 `json:"query"`
			Variables map[string]interface{} `json:"variables"`
		}
		json.NewDecoder(r.Body).Decode(&body)
		receivedVars = body.Variables

		resp := map[string]interface{}{
			"data": map[string]interface{}{
				"user": []interface{}{
					map[string]interface{}{"id": "1", "name": "Alice"},
				},
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	tmpDir := t.TempDir()
	queryPath := filepath.Join(tmpDir, "query.graphql")
	os.WriteFile(queryPath, []byte("query($id: ID!) { user(id: $id) { id name } }"), 0644)

	cfg := config.SourceConfig{
		Type:       "graphql",
		URL:        server.URL,
		QueryFile:  "query.graphql",
		ResultPath: "user",
		Variables: map[string]interface{}{
			"id": "123",
		},
	}

	src, err := NewGraphQLSource("test", cfg, tmpDir)
	if err != nil {
		t.Fatalf("failed to create source: %v", err)
	}

	_, err = src.Fetch(context.Background())
	if err != nil {
		t.Fatalf("fetch failed: %v", err)
	}

	if receivedVars["id"] != "123" {
		t.Errorf("expected id=123, got %v", receivedVars["id"])
	}
}

func TestGraphQLSource_FetchGraphQLError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]interface{}{
			"data": nil,
			"errors": []interface{}{
				map[string]interface{}{
					"message": "Not found",
					"path":    []interface{}{"user"},
				},
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	tmpDir := t.TempDir()
	queryPath := filepath.Join(tmpDir, "query.graphql")
	os.WriteFile(queryPath, []byte("query { user { id } }"), 0644)

	cfg := config.SourceConfig{
		Type:       "graphql",
		URL:        server.URL,
		QueryFile:  "query.graphql",
		ResultPath: "user",
	}

	src, err := NewGraphQLSource("test", cfg, tmpDir)
	if err != nil {
		t.Fatalf("failed to create source: %v", err)
	}

	_, err = src.Fetch(context.Background())
	if err == nil {
		t.Error("expected error for GraphQL error response")
	}

	gqlErr, ok := err.(*GraphQLError)
	if !ok {
		t.Errorf("expected GraphQLError, got %T", err)
	} else if gqlErr.Message != "Not found" {
		t.Errorf("expected 'Not found', got '%s'", gqlErr.Message)
	}
}

func TestGraphQLSource_FetchQueryFileNotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("should not make request if query file not found")
	}))
	defer server.Close()

	cfg := config.SourceConfig{
		Type:       "graphql",
		URL:        server.URL,
		QueryFile:  "nonexistent.graphql",
		ResultPath: "users",
	}

	src, err := NewGraphQLSource("test", cfg, "/tmp")
	if err != nil {
		t.Fatalf("failed to create source: %v", err)
	}

	_, err = src.Fetch(context.Background())
	if err == nil {
		t.Error("expected error for missing query file")
	}
}
```

**Step 2: Run tests to verify they fail**

Run: `GOWORK=off go test ./internal/source/... -v -run "TestGraphQLSource_Fetch"`
Expected: FAIL (Fetch not implemented)

**Step 3: Implement Fetch method**

```go
// Add to internal/source/graphql.go after Close():

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/livetemplate/tinkerdown/internal/config"
)

// Fetch makes a GraphQL request and parses the response
func (s *GraphQLSource) Fetch(ctx context.Context) ([]map[string]interface{}, error) {
	// Use circuit breaker + retry
	return s.circuitBreaker.Execute(ctx, func(ctx context.Context) ([]map[string]interface{}, error) {
		return WithRetry(ctx, s.name, s.retryConfig, func(ctx context.Context) ([]map[string]interface{}, error) {
			return s.doFetch(ctx)
		})
	})
}

// doFetch performs the actual GraphQL request
func (s *GraphQLSource) doFetch(ctx context.Context) ([]map[string]interface{}, error) {
	// Read query from file
	queryPath := filepath.Join(s.siteDir, s.queryFile)
	queryBytes, err := os.ReadFile(queryPath)
	if err != nil {
		return nil, &SourceError{
			Source:    s.name,
			Operation: "read query file",
			Err:       err,
			Retryable: false,
		}
	}

	// Build request body
	body := map[string]interface{}{
		"query": string(queryBytes),
	}
	if len(s.variables) > 0 {
		body["variables"] = s.variables
	}

	bodyBytes, err := json.Marshal(body)
	if err != nil {
		return nil, &SourceError{
			Source:    s.name,
			Operation: "marshal request",
			Err:       err,
			Retryable: false,
		}
	}

	// Create request with context
	req, err := http.NewRequestWithContext(ctx, "POST", s.url, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, &SourceError{
			Source:    s.name,
			Operation: "create request",
			Err:       err,
			Retryable: false,
		}
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	for key, value := range s.headers {
		req.Header.Set(key, value)
	}

	// Execute request
	resp, err := s.client.Do(req)
	if err != nil {
		return nil, NewSourceError(s.name, "request", err)
	}
	defer resp.Body.Close()

	// Check HTTP status code
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		bodyBytes, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return nil, &HTTPError{
			Source:     s.name,
			StatusCode: resp.StatusCode,
			Status:     resp.Status,
			Body:       strings.TrimSpace(string(bodyBytes)),
		}
	}

	// Read response body with size limit
	const maxResponseSize = 10 * 1024 * 1024 // 10MB
	respBody, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseSize))
	if err != nil {
		return nil, &SourceError{
			Source:    s.name,
			Operation: "read response",
			Err:       err,
			Retryable: false,
		}
	}

	// Parse GraphQL response
	var gqlResp struct {
		Data   map[string]interface{} `json:"data"`
		Errors []struct {
			Message string   `json:"message"`
			Path    []string `json:"path"`
		} `json:"errors"`
	}

	if err := json.Unmarshal(respBody, &gqlResp); err != nil {
		return nil, &ValidationError{
			Source: s.name,
			Reason: "could not parse response as JSON",
		}
	}

	// Check for GraphQL errors
	if len(gqlResp.Errors) > 0 {
		return nil, &GraphQLError{
			Source:  s.name,
			Message: gqlResp.Errors[0].Message,
			Path:    gqlResp.Errors[0].Path,
		}
	}

	// Extract data using result_path
	if gqlResp.Data == nil {
		return []map[string]interface{}{}, nil
	}

	return extractPath(gqlResp.Data, s.resultPath)
}
```

**Step 4: Run tests to verify they pass**

Run: `GOWORK=off go test ./internal/source/... -v -run "TestGraphQLSource"`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/source/graphql.go internal/source/graphql_test.go
git commit -m "feat(source): implement GraphQL Fetch method"
```

---

## Task 6: Register GraphQL Source Type

**Files:**
- Modify: `internal/source/source.go`

**Step 1: Add graphql case to createSource**

Find the `createSource` function and add the graphql case:

```go
// Add after the "wasm" case in createSource():
case "graphql":
	return NewGraphQLSource(name, cfg, siteDir)
```

**Step 2: Run all source tests**

Run: `GOWORK=off go test ./internal/source/... -v`
Expected: PASS

**Step 3: Commit**

```bash
git add internal/source/source.go
git commit -m "feat(source): register GraphQL source type"
```

---

## Task 7: E2E Test with Public API

**Files:**
- Create: `examples/lvt-source-graphql-test/index.md`
- Create: `examples/lvt-source-graphql-test/tinkerdown.yaml`
- Create: `examples/lvt-source-graphql-test/queries/countries.graphql`
- Create: `lvtsource_graphql_e2e_test.go`

**Step 1: Create example app**

```yaml
# examples/lvt-source-graphql-test/tinkerdown.yaml
title: GraphQL Source Test
sources:
  countries:
    type: graphql
    url: https://countries.trevorblades.com/graphql
    query_file: queries/countries.graphql
    result_path: countries
```

```graphql
# examples/lvt-source-graphql-test/queries/countries.graphql
query {
  countries {
    code
    name
    emoji
  }
}
```

```markdown
# examples/lvt-source-graphql-test/index.md
---
title: GraphQL Countries
---

# Countries from GraphQL API

\`\`\`lvt
<table lvt-source="countries" lvt-columns="code:Code,name:Country,emoji:Flag" lvt-empty="Loading countries...">
</table>
\`\`\`
```

**Step 2: Write E2E test**

```go
// lvtsource_graphql_e2e_test.go
package tinkerdown_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/chromedp/chromedp"
)

func TestGraphQLSourceE2E(t *testing.T) {
	// Start server
	srv, cleanup := startTestServer(t, "examples/lvt-source-graphql-test")
	defer cleanup()

	// Create browser context
	ctx, cancel := chromedp.NewContext(context.Background())
	defer cancel()

	ctx, cancel = context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	var tableHTML string
	var countryCount int

	err := chromedp.Run(ctx,
		chromedp.Navigate(srv.URL),
		chromedp.WaitVisible("table", chromedp.ByQuery),
		chromedp.Sleep(2*time.Second), // Wait for data to load
		chromedp.OuterHTML("table", &tableHTML, chromedp.ByQuery),
		chromedp.Evaluate(`document.querySelectorAll("table tbody tr").length`, &countryCount),
	)

	if err != nil {
		t.Fatalf("browser test failed: %v", err)
	}

	// Verify table has data
	if countryCount < 10 {
		t.Errorf("expected at least 10 countries, got %d", countryCount)
	}

	// Verify headers exist
	if !strings.Contains(tableHTML, "<th>Code</th>") {
		t.Error("missing Code header")
	}
	if !strings.Contains(tableHTML, "<th>Country</th>") {
		t.Error("missing Country header")
	}
	if !strings.Contains(tableHTML, "<th>Flag</th>") {
		t.Error("missing Flag header")
	}

	t.Logf("GraphQL source loaded %d countries", countryCount)
}
```

**Step 3: Run E2E test**

Run: `GOWORK=off go test . -v -run TestGraphQLSourceE2E -timeout 60s`
Expected: PASS

**Step 4: Commit**

```bash
git add examples/lvt-source-graphql-test/ lvtsource_graphql_e2e_test.go
git commit -m "test: add GraphQL source E2E test with public countries API"
```

---

## Task 8: Update Documentation

**Files:**
- Modify: `docs/sources/graphql.md`
- Modify: `docs/guides/data-sources.md`
- Modify: `docs/reference/config.md`

**Step 1: Update graphql.md with complete documentation**

Reference the design doc at `docs/plans/2025-12-31-graphql-source-design.md` and ensure `docs/sources/graphql.md` has complete examples.

**Step 2: Add GraphQL to data-sources.md overview**

Add a GraphQL section with basic example.

**Step 3: Add new config fields to config.md**

Document `query_file`, `variables`, and `result_path` fields.

**Step 4: Commit**

```bash
git add docs/
git commit -m "docs: add GraphQL source documentation"
```

---

## Task 9: Update Roadmap

**Files:**
- Modify: `ROADMAP.md`

**Step 1: Mark 4.1 GraphQL source as completed**

Update the roadmap to mark the GraphQL source task as done and move it to Recently Completed.

**Step 2: Commit**

```bash
git add ROADMAP.md
git commit -m "docs: mark 4.1 GraphQL source as completed"
```

---

## Task 10: Final Verification

**Step 1: Run all tests**

Run: `GOWORK=off go test ./... -v`
Expected: All PASS

**Step 2: Build binary**

Run: `GOWORK=off go build -o tinkerdown ./cmd/tinkerdown`
Expected: Success

**Step 3: Manual test with example**

Run: `./tinkerdown serve examples/lvt-source-graphql-test`
Open browser, verify countries table loads.

**Step 4: Create PR**

```bash
git push -u origin feature/graphql-source
gh pr create --title "feat: implement GraphQL source (Phase 4.1)" --body "$(cat <<'EOF'
## Summary
- Adds GraphQL source type for fetching data from GraphQL APIs
- Query files stored externally (.graphql)
- Variables with environment expansion
- Result path extraction for nested responses
- Reuses REST auth pattern via options
- Full retry/circuit breaker support

## Test plan
- [x] Unit tests for extractPath
- [x] Unit tests for constructor validation
- [x] Unit tests for Fetch with mock server
- [x] E2E test with public countries API
- [x] Manual verification with example app

🤖 Generated with [Claude Code](https://claude.com/claude-code)
EOF
)"
```
