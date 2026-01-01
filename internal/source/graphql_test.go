// internal/source/graphql_test.go
package source

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
