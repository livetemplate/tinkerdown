package server

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/livetemplate/tinkerdown/internal/config"
	"github.com/livetemplate/tinkerdown/internal/source"
)

// mockSource is a simple mock source for testing
type mockSource struct {
	name     string
	data     []map[string]interface{}
	readonly bool
	writes   []mockWrite
}

type mockWrite struct {
	action string
	data   map[string]interface{}
}

func (m *mockSource) Name() string { return m.name }

func (m *mockSource) Fetch(ctx context.Context) ([]map[string]interface{}, error) {
	return m.data, nil
}

func (m *mockSource) Close() error { return nil }

func (m *mockSource) WriteItem(ctx context.Context, action string, data map[string]interface{}) error {
	m.writes = append(m.writes, mockWrite{action: action, data: data})
	return nil
}

func (m *mockSource) IsReadonly() bool { return m.readonly }

// Ensure mockSource implements WritableSource
var _ source.WritableSource = (*mockSource)(nil)

func TestAPIHandler_Get(t *testing.T) {
	cfg := &config.Config{
		Sources: map[string]config.SourceConfig{
			"tasks": {Type: "json"},
		},
	}

	handler := NewAPIHandler(cfg, t.TempDir())
	defer handler.Close()

	// Inject a mock source directly
	handler.mu.Lock()
	handler.sources["tasks"] = &mockSource{
		name: "tasks",
		data: []map[string]interface{}{
			{"id": "1", "text": "Task 1", "done": false},
			{"id": "2", "text": "Task 2", "done": true},
		},
	}
	handler.mu.Unlock()

	// Test GET request
	req := httptest.NewRequest("GET", "/api/sources/tasks", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	data, ok := response["data"].([]interface{})
	if !ok {
		t.Fatalf("Expected data to be an array")
	}
	if len(data) != 2 {
		t.Errorf("Expected 2 items, got %d", len(data))
	}

	if response["total"].(float64) != 2 {
		t.Errorf("Expected total=2, got %v", response["total"])
	}
}

func TestAPIHandler_GetWithFilter(t *testing.T) {
	cfg := &config.Config{
		Sources: map[string]config.SourceConfig{
			"tasks": {Type: "json"},
		},
	}

	handler := NewAPIHandler(cfg, t.TempDir())
	defer handler.Close()

	// Inject a mock source
	handler.mu.Lock()
	handler.sources["tasks"] = &mockSource{
		name: "tasks",
		data: []map[string]interface{}{
			{"id": "1", "text": "Task 1", "done": "false"},
			{"id": "2", "text": "Task 2", "done": "true"},
			{"id": "3", "text": "Task 3", "done": "false"},
		},
	}
	handler.mu.Unlock()

	// Test GET request with filter
	req := httptest.NewRequest("GET", "/api/sources/tasks?filter=done=false", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	data, ok := response["data"].([]interface{})
	if !ok {
		t.Fatalf("Expected data to be an array")
	}
	if len(data) != 2 {
		t.Errorf("Expected 2 items with done=false, got %d", len(data))
	}
}

func TestAPIHandler_GetWithPagination(t *testing.T) {
	cfg := &config.Config{
		Sources: map[string]config.SourceConfig{
			"tasks": {Type: "json"},
		},
	}

	handler := NewAPIHandler(cfg, t.TempDir())
	defer handler.Close()

	// Inject a mock source
	handler.mu.Lock()
	handler.sources["tasks"] = &mockSource{
		name: "tasks",
		data: []map[string]interface{}{
			{"id": "1", "text": "Task 1"},
			{"id": "2", "text": "Task 2"},
			{"id": "3", "text": "Task 3"},
			{"id": "4", "text": "Task 4"},
			{"id": "5", "text": "Task 5"},
		},
	}
	handler.mu.Unlock()

	// Test GET request with pagination
	req := httptest.NewRequest("GET", "/api/sources/tasks?limit=2&offset=1", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	data, ok := response["data"].([]interface{})
	if !ok {
		t.Fatalf("Expected data to be an array")
	}
	if len(data) != 2 {
		t.Errorf("Expected 2 items, got %d", len(data))
	}

	// Total should still be 5
	if response["total"].(float64) != 5 {
		t.Errorf("Expected total=5, got %v", response["total"])
	}

	// Check we got the right items (offset=1, so starting from Task 2)
	firstItem := data[0].(map[string]interface{})
	if firstItem["id"] != "2" {
		t.Errorf("Expected first item id=2, got %v", firstItem["id"])
	}
}

func TestAPIHandler_Post(t *testing.T) {
	cfg := &config.Config{
		Sources: map[string]config.SourceConfig{
			"tasks": {Type: "json"},
		},
	}

	handler := NewAPIHandler(cfg, t.TempDir())
	defer handler.Close()

	// Inject a mock writable source
	mockSrc := &mockSource{
		name:     "tasks",
		data:     []map[string]interface{}{},
		readonly: false,
	}
	handler.mu.Lock()
	handler.sources["tasks"] = mockSrc
	handler.mu.Unlock()

	// Test POST request
	body := strings.NewReader(`{"text": "New Task", "done": false}`)
	req := httptest.NewRequest("POST", "/api/sources/tasks", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("Expected status 201, got %d: %s", w.Code, w.Body.String())
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if response["success"] != true {
		t.Error("Expected success=true")
	}

	// Check write was recorded
	if len(mockSrc.writes) != 1 {
		t.Errorf("Expected 1 write, got %d", len(mockSrc.writes))
	}
	if mockSrc.writes[0].action != "add" {
		t.Errorf("Expected action=add, got %s", mockSrc.writes[0].action)
	}
}

func TestAPIHandler_Put(t *testing.T) {
	cfg := &config.Config{
		Sources: map[string]config.SourceConfig{
			"tasks": {Type: "json"},
		},
	}

	handler := NewAPIHandler(cfg, t.TempDir())
	defer handler.Close()

	// Inject a mock writable source
	mockSrc := &mockSource{
		name:     "tasks",
		data:     []map[string]interface{}{{"id": "1", "text": "Task 1"}},
		readonly: false,
	}
	handler.mu.Lock()
	handler.sources["tasks"] = mockSrc
	handler.mu.Unlock()

	// Test PUT request
	body := strings.NewReader(`{"done": true}`)
	req := httptest.NewRequest("PUT", "/api/sources/tasks/1", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	// Check write was recorded
	if len(mockSrc.writes) != 1 {
		t.Errorf("Expected 1 write, got %d", len(mockSrc.writes))
	}
	if mockSrc.writes[0].action != "update" {
		t.Errorf("Expected action=update, got %s", mockSrc.writes[0].action)
	}
	if mockSrc.writes[0].data["id"] != "1" {
		t.Errorf("Expected id=1, got %v", mockSrc.writes[0].data["id"])
	}
}

func TestAPIHandler_Delete(t *testing.T) {
	cfg := &config.Config{
		Sources: map[string]config.SourceConfig{
			"tasks": {Type: "json"},
		},
	}

	handler := NewAPIHandler(cfg, t.TempDir())
	defer handler.Close()

	// Inject a mock writable source
	mockSrc := &mockSource{
		name:     "tasks",
		data:     []map[string]interface{}{{"id": "1", "text": "Task 1"}},
		readonly: false,
	}
	handler.mu.Lock()
	handler.sources["tasks"] = mockSrc
	handler.mu.Unlock()

	// Test DELETE request
	req := httptest.NewRequest("DELETE", "/api/sources/tasks/1", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	// Check write was recorded
	if len(mockSrc.writes) != 1 {
		t.Errorf("Expected 1 write, got %d", len(mockSrc.writes))
	}
	if mockSrc.writes[0].action != "delete" {
		t.Errorf("Expected action=delete, got %s", mockSrc.writes[0].action)
	}
}

func TestAPIHandler_PostInvalidJSON(t *testing.T) {
	cfg := &config.Config{
		Sources: map[string]config.SourceConfig{
			"tasks": {Type: "json"},
		},
	}

	handler := NewAPIHandler(cfg, t.TempDir())
	defer handler.Close()

	// Inject a mock writable source
	handler.mu.Lock()
	handler.sources["tasks"] = &mockSource{
		name:     "tasks",
		data:     []map[string]interface{}{},
		readonly: false,
	}
	handler.mu.Unlock()

	tests := []struct {
		name string
		body string
	}{
		{"malformed JSON", `{"text": "missing closing brace"`},
		{"non-JSON content", `this is not json`},
		{"empty body", ``},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("POST", "/api/sources/tasks", strings.NewReader(tt.body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, req)

			if w.Code != http.StatusBadRequest {
				t.Errorf("Expected status 400 for %s, got %d: %s", tt.name, w.Code, w.Body.String())
			}

			var response map[string]string
			if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
				t.Fatalf("Failed to parse error response: %v", err)
			}
			if response["error"] != "invalid JSON body" {
				t.Errorf("Expected error 'invalid JSON body', got %s", response["error"])
			}
		})
	}
}

func TestAPIHandler_PutInvalidJSON(t *testing.T) {
	cfg := &config.Config{
		Sources: map[string]config.SourceConfig{
			"tasks": {Type: "json"},
		},
	}

	handler := NewAPIHandler(cfg, t.TempDir())
	defer handler.Close()

	// Inject a mock writable source
	handler.mu.Lock()
	handler.sources["tasks"] = &mockSource{
		name:     "tasks",
		data:     []map[string]interface{}{{"id": "1"}},
		readonly: false,
	}
	handler.mu.Unlock()

	tests := []struct {
		name string
		body string
	}{
		{"malformed JSON", `{"done": true`},
		{"non-JSON content", `not json at all`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("PUT", "/api/sources/tasks/1", strings.NewReader(tt.body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, req)

			if w.Code != http.StatusBadRequest {
				t.Errorf("Expected status 400 for %s, got %d: %s", tt.name, w.Code, w.Body.String())
			}

			var response map[string]string
			if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
				t.Fatalf("Failed to parse error response: %v", err)
			}
			if response["error"] != "invalid JSON body" {
				t.Errorf("Expected error 'invalid JSON body', got %s", response["error"])
			}
		})
	}
}

func TestAPIHandler_ReadonlySource(t *testing.T) {
	cfg := &config.Config{
		Sources: map[string]config.SourceConfig{
			"tasks": {Type: "json"},
		},
	}

	handler := NewAPIHandler(cfg, t.TempDir())
	defer handler.Close()

	// Inject a mock read-only source
	handler.mu.Lock()
	handler.sources["tasks"] = &mockSource{
		name:     "tasks",
		data:     []map[string]interface{}{},
		readonly: true,
	}
	handler.mu.Unlock()

	// Test POST request should fail
	body := strings.NewReader(`{"text": "New Task"}`)
	req := httptest.NewRequest("POST", "/api/sources/tasks", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("Expected status 403, got %d", w.Code)
	}
}

func TestAPIHandler_SourceNotFound(t *testing.T) {
	cfg := &config.Config{
		Sources: map[string]config.SourceConfig{},
	}

	handler := NewAPIHandler(cfg, t.TempDir())
	defer handler.Close()

	// Test GET request for non-existent source
	req := httptest.NewRequest("GET", "/api/sources/nonexistent", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d", w.Code)
	}
}

func TestCORSMiddleware(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	t.Run("specific origin", func(t *testing.T) {
		// Test with specific origin (not wildcard)
		wrapped := CORSMiddleware([]string{"http://localhost:3000"})(handler)

		req := httptest.NewRequest("GET", "/api/sources/test", nil)
		req.Header.Set("Origin", "http://localhost:3000")
		w := httptest.NewRecorder()

		wrapped.ServeHTTP(w, req)

		if w.Header().Get("Access-Control-Allow-Origin") != "http://localhost:3000" {
			t.Errorf("Expected specific origin header, got %s", w.Header().Get("Access-Control-Allow-Origin"))
		}
	})

	t.Run("wildcard origin", func(t *testing.T) {
		// Test with wildcard - should use "*" header
		wrapped := CORSMiddleware([]string{"*"})(handler)

		req := httptest.NewRequest("GET", "/api/sources/test", nil)
		req.Header.Set("Origin", "http://example.com")
		w := httptest.NewRecorder()

		wrapped.ServeHTTP(w, req)

		if w.Header().Get("Access-Control-Allow-Origin") != "*" {
			t.Errorf("Expected wildcard '*' header, got %s", w.Header().Get("Access-Control-Allow-Origin"))
		}
	})

	t.Run("disallowed origin", func(t *testing.T) {
		wrapped := CORSMiddleware([]string{"http://localhost:3000"})(handler)

		req := httptest.NewRequest("GET", "/api/sources/test", nil)
		req.Header.Set("Origin", "http://evil.com")
		w := httptest.NewRecorder()

		wrapped.ServeHTTP(w, req)

		if w.Header().Get("Access-Control-Allow-Origin") != "" {
			t.Errorf("Expected no CORS header for disallowed origin, got %s", w.Header().Get("Access-Control-Allow-Origin"))
		}
	})
}

func TestCORSMiddleware_Preflight(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	wrapped := CORSMiddleware([]string{"*"})(handler)

	// Test OPTIONS preflight request
	req := httptest.NewRequest("OPTIONS", "/api/sources/test", nil)
	req.Header.Set("Origin", "http://example.com")
	w := httptest.NewRecorder()

	wrapped.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200 for preflight, got %d", w.Code)
	}
	if w.Header().Get("Access-Control-Allow-Methods") == "" {
		t.Error("Expected Access-Control-Allow-Methods header")
	}
	// Verify wildcard is used in preflight response
	if w.Header().Get("Access-Control-Allow-Origin") != "*" {
		t.Errorf("Expected wildcard '*' header in preflight, got %s", w.Header().Get("Access-Control-Allow-Origin"))
	}
}

func TestRateLimitMiddleware(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// Very low rate limit for testing: 1 request per second, burst of 1
	wrapped := RateLimitMiddleware(1, 1)(handler)

	// First request should succeed
	req1 := httptest.NewRequest("GET", "/api/sources/test", nil)
	w1 := httptest.NewRecorder()
	wrapped.ServeHTTP(w1, req1)

	if w1.Code != http.StatusOK {
		t.Errorf("Expected first request to succeed, got %d", w1.Code)
	}

	// Second request should be rate limited
	req2 := httptest.NewRequest("GET", "/api/sources/test", nil)
	w2 := httptest.NewRecorder()
	wrapped.ServeHTTP(w2, req2)

	if w2.Code != http.StatusTooManyRequests {
		t.Errorf("Expected second request to be rate limited, got %d", w2.Code)
	}
}

func TestApplyFilter(t *testing.T) {
	data := []map[string]interface{}{
		{"id": "1", "status": "active", "count": 10.0},
		{"id": "2", "status": "inactive", "count": 20.0},
		{"id": "3", "status": "active", "count": 30.0},
	}

	tests := []struct {
		filter   string
		expected int
	}{
		{"status=active", 2},
		{"status=inactive", 1},
		{"status!=active", 1},
		{"count=10", 1},
		{"nonexistent=value", 0},
	}

	for _, tt := range tests {
		t.Run(tt.filter, func(t *testing.T) {
			result := applyFilter(data, tt.filter)
			if len(result) != tt.expected {
				t.Errorf("filter %q: expected %d items, got %d", tt.filter, tt.expected, len(result))
			}
		})
	}
}

func TestPaginate(t *testing.T) {
	data := []map[string]interface{}{
		{"id": "1"}, {"id": "2"}, {"id": "3"}, {"id": "4"}, {"id": "5"},
	}

	tests := []struct {
		name     string
		offset   int
		limit    int
		expected int
		firstID  string
	}{
		{"no pagination", 0, 0, 5, "1"},
		{"limit only", 0, 3, 3, "1"},
		{"offset and limit", 2, 2, 2, "3"},
		{"offset near end", 4, 10, 1, "5"},
		{"offset past end", 10, 10, 0, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := paginate(data, tt.offset, tt.limit)
			if len(result) != tt.expected {
				t.Errorf("offset=%d, limit=%d: expected %d items, got %d", tt.offset, tt.limit, tt.expected, len(result))
			}
			if tt.expected > 0 && result[0]["id"] != tt.firstID {
				t.Errorf("offset=%d, limit=%d: expected first id=%s, got %v", tt.offset, tt.limit, tt.firstID, result[0]["id"])
			}
		})
	}
}

func TestAuthMiddleware(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"success": true}`))
	})

	t.Run("no auth configured passes through", func(t *testing.T) {
		wrapped := AuthMiddleware(nil)(handler)

		req := httptest.NewRequest("GET", "/api/sources/test", nil)
		w := httptest.NewRecorder()

		wrapped.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected request to pass through without auth, got %d", w.Code)
		}
	})

	t.Run("valid API key with X-API-Key header", func(t *testing.T) {
		cfg := &config.AuthConfig{APIKey: "secret-key-123"}
		wrapped := AuthMiddleware(cfg)(handler)

		req := httptest.NewRequest("GET", "/api/sources/test", nil)
		req.Header.Set("X-API-Key", "secret-key-123")
		w := httptest.NewRecorder()

		wrapped.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected valid API key to succeed, got %d", w.Code)
		}
	})

	t.Run("invalid API key", func(t *testing.T) {
		cfg := &config.AuthConfig{APIKey: "secret-key-123"}
		wrapped := AuthMiddleware(cfg)(handler)

		req := httptest.NewRequest("GET", "/api/sources/test", nil)
		req.Header.Set("X-API-Key", "wrong-key")
		w := httptest.NewRecorder()

		wrapped.ServeHTTP(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Errorf("Expected invalid API key to return 401, got %d", w.Code)
		}

		var response map[string]string
		if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
			t.Fatalf("Failed to parse error response: %v", err)
		}
		if response["error"] != "invalid API key" {
			t.Errorf("Expected 'invalid API key' error, got %s", response["error"])
		}
	})

	t.Run("missing API key", func(t *testing.T) {
		cfg := &config.AuthConfig{APIKey: "secret-key-123"}
		wrapped := AuthMiddleware(cfg)(handler)

		req := httptest.NewRequest("GET", "/api/sources/test", nil)
		w := httptest.NewRecorder()

		wrapped.ServeHTTP(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Errorf("Expected missing API key to return 401, got %d", w.Code)
		}

		var response map[string]string
		if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
			t.Fatalf("Failed to parse error response: %v", err)
		}
		if response["error"] != "authentication required" {
			t.Errorf("Expected 'authentication required' error, got %s", response["error"])
		}
	})

	t.Run("valid Bearer token", func(t *testing.T) {
		cfg := &config.AuthConfig{APIKey: "secret-key-123", HeaderName: "Authorization"}
		wrapped := AuthMiddleware(cfg)(handler)

		req := httptest.NewRequest("GET", "/api/sources/test", nil)
		req.Header.Set("Authorization", "Bearer secret-key-123")
		w := httptest.NewRecorder()

		wrapped.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected valid Bearer token to succeed, got %d", w.Code)
		}
	})

	t.Run("invalid Bearer format", func(t *testing.T) {
		cfg := &config.AuthConfig{APIKey: "secret-key-123", HeaderName: "Authorization"}
		wrapped := AuthMiddleware(cfg)(handler)

		req := httptest.NewRequest("GET", "/api/sources/test", nil)
		req.Header.Set("Authorization", "Basic secret-key-123")
		w := httptest.NewRecorder()

		wrapped.ServeHTTP(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Errorf("Expected invalid Bearer format to return 401, got %d", w.Code)
		}

		var response map[string]string
		if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
			t.Fatalf("Failed to parse error response: %v", err)
		}
		if !strings.Contains(response["error"], "Bearer") {
			t.Errorf("Expected error about Bearer format, got %s", response["error"])
		}
	})

	t.Run("custom header name", func(t *testing.T) {
		cfg := &config.AuthConfig{APIKey: "my-token", HeaderName: "X-Custom-Token"}
		wrapped := AuthMiddleware(cfg)(handler)

		req := httptest.NewRequest("GET", "/api/sources/test", nil)
		req.Header.Set("X-Custom-Token", "my-token")
		w := httptest.NewRecorder()

		wrapped.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected custom header to work, got %d", w.Code)
		}
	})

	t.Run("multiple keys with different permissions", func(t *testing.T) {
		cfg := &config.AuthConfig{
			Keys: []config.APIKeyConfig{
				{Name: "readonly", Key: "ro-key", Permissions: []config.Permission{config.PermRead}},
				{Name: "admin", Key: "admin-key", Permissions: []config.Permission{config.PermRead, config.PermWrite, config.PermDelete}},
			},
		}
		wrapped := AuthMiddleware(cfg)(handler)

		// readonly key should authenticate
		req := httptest.NewRequest("GET", "/api/sources/test", nil)
		req.Header.Set("X-API-Key", "ro-key")
		w := httptest.NewRecorder()
		wrapped.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Errorf("Expected readonly key to authenticate, got %d", w.Code)
		}

		// admin key should authenticate
		req = httptest.NewRequest("GET", "/api/sources/test", nil)
		req.Header.Set("X-API-Key", "admin-key")
		w = httptest.NewRecorder()
		wrapped.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Errorf("Expected admin key to authenticate, got %d", w.Code)
		}

		// unknown key should fail
		req = httptest.NewRequest("GET", "/api/sources/test", nil)
		req.Header.Set("X-API-Key", "unknown")
		w = httptest.NewRecorder()
		wrapped.ServeHTTP(w, req)
		if w.Code != http.StatusUnauthorized {
			t.Errorf("Expected unknown key to return 401, got %d", w.Code)
		}
	})
}

func TestMethodPermissionMiddleware(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	tests := []struct {
		name       string
		method     string
		perms      []config.Permission
		wantStatus int
	}{
		{"GET with read", "GET", []config.Permission{config.PermRead}, http.StatusOK},
		{"GET without read", "GET", []config.Permission{config.PermWrite}, http.StatusForbidden},
		{"POST with write", "POST", []config.Permission{config.PermWrite}, http.StatusOK},
		{"POST without write", "POST", []config.Permission{config.PermRead}, http.StatusForbidden},
		{"DELETE with delete", "DELETE", []config.Permission{config.PermDelete}, http.StatusOK},
		{"DELETE without delete", "DELETE", []config.Permission{config.PermRead}, http.StatusForbidden},
		{"PUT with write", "PUT", []config.Permission{config.PermWrite}, http.StatusOK},
		{"OPTIONS bypasses check", "OPTIONS", nil, http.StatusOK},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wrapped := MethodPermissionMiddleware()(handler)

			req := httptest.NewRequest(tt.method, "/api/sources/test", nil)
			if tt.perms != nil {
				ctx := context.WithValue(req.Context(), ctxKeyPermissions, tt.perms)
				req = req.WithContext(ctx)
			}
			w := httptest.NewRecorder()
			wrapped.ServeHTTP(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("method=%s perms=%v: got %d, want %d", tt.method, tt.perms, w.Code, tt.wantStatus)
			}
		})
	}
}

func TestSecureCompare(t *testing.T) {
	tests := []struct {
		name     string
		a        string
		b        string
		expected bool
	}{
		{"equal strings", "secret", "secret", true},
		{"different strings", "secret", "wrong", false},
		{"different lengths", "short", "longer", false},
		{"empty strings", "", "", true},
		{"one empty", "secret", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := secureCompare(tt.a, tt.b)
			if result != tt.expected {
				t.Errorf("secureCompare(%q, %q) = %v, want %v", tt.a, tt.b, result, tt.expected)
			}
		})
	}
}
