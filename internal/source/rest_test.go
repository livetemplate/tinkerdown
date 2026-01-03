package source

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewRestSourceValidation(t *testing.T) {
	tests := []struct {
		name      string
		srcName   string
		url       string
		options   map[string]string
		wantErr   bool
		errReason string
	}{
		{
			name:    "valid url",
			srcName: "test",
			url:     "http://localhost:8080/api",
			options: nil,
			wantErr: false,
		},
		{
			name:      "empty url",
			srcName:   "test",
			url:       "",
			options:   nil,
			wantErr:   true,
			errReason: "url is required",
		},
		{
			name:    "url with env var",
			srcName: "test",
			url:     "${TEST_API_URL}/data",
			options: nil,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			src, err := NewRestSource(tt.srcName, tt.url, tt.options)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errReason)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.srcName, src.Name())
		})
	}
}

func TestNewRestSourceOptions(t *testing.T) {
	tests := []struct {
		name           string
		options        map[string]string
		expectedMethod string
	}{
		{
			name:           "default method is GET",
			options:        nil,
			expectedMethod: "GET",
		},
		{
			name:           "explicit GET method",
			options:        map[string]string{"method": "get"},
			expectedMethod: "GET",
		},
		{
			name:           "POST method",
			options:        map[string]string{"method": "post"},
			expectedMethod: "POST",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var receivedMethod string
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				receivedMethod = r.Method
				w.Header().Set("Content-Type", "application/json")
				w.Write([]byte("[]"))
			}))
			defer server.Close()

			src, err := NewRestSource("test", server.URL, tt.options)
			require.NoError(t, err)

			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			_, err = src.Fetch(ctx)
			require.NoError(t, err)
			assert.Equal(t, tt.expectedMethod, receivedMethod)
		})
	}
}

func TestRestSourceFetchJSON(t *testing.T) {
	// Create test server that returns JSON array
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]map[string]interface{}{
			{"id": 1, "name": "Alice"},
			{"id": 2, "name": "Bob"},
		})
	}))
	defer server.Close()

	src, err := NewRestSource("test", server.URL, nil)
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	data, err := src.Fetch(ctx)
	require.NoError(t, err)
	require.Len(t, data, 2)

	assert.Equal(t, float64(1), data[0]["id"])
	assert.Equal(t, "Alice", data[0]["name"])
	assert.Equal(t, float64(2), data[1]["id"])
	assert.Equal(t, "Bob", data[1]["name"])
}

func TestRestSourceFetchSingleObject(t *testing.T) {
	// Create test server that returns single JSON object
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status": "ok",
			"count":  42,
		})
	}))
	defer server.Close()

	src, err := NewRestSource("test", server.URL, nil)
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	data, err := src.Fetch(ctx)
	require.NoError(t, err)
	require.Len(t, data, 1)

	assert.Equal(t, "ok", data[0]["status"])
	assert.Equal(t, float64(42), data[0]["count"])
}

func TestRestSourceFetchDataWrapper(t *testing.T) {
	// Create test server that returns wrapped data (common API pattern)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": true,
			"data": []map[string]interface{}{
				{"id": 1, "title": "First"},
				{"id": 2, "title": "Second"},
			},
		})
	}))
	defer server.Close()

	src, err := NewRestSource("test", server.URL, nil)
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	data, err := src.Fetch(ctx)
	require.NoError(t, err)
	require.Len(t, data, 2)

	assert.Equal(t, float64(1), data[0]["id"])
	assert.Equal(t, "First", data[0]["title"])
}

func TestRestSourceFetchResultsWrapper(t *testing.T) {
	// Create test server that returns results wrapper (another common pattern)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"count": 2,
			"results": []map[string]interface{}{
				{"id": 1, "name": "First"},
				{"id": 2, "name": "Second"},
			},
		})
	}))
	defer server.Close()

	src, err := NewRestSource("test", server.URL, nil)
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	data, err := src.Fetch(ctx)
	require.NoError(t, err)
	require.Len(t, data, 2)

	assert.Equal(t, float64(1), data[0]["id"])
	assert.Equal(t, "First", data[0]["name"])
}

func TestRestSourceFetchEmptyResponse(t *testing.T) {
	// Create test server that returns empty body
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(""))
	}))
	defer server.Close()

	src, err := NewRestSource("test", server.URL, nil)
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	data, err := src.Fetch(ctx)
	require.NoError(t, err)
	assert.Empty(t, data)
}

func TestRestSourceFetch404(t *testing.T) {
	// Create test server that returns 404
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("Not Found"))
	}))
	defer server.Close()

	src, err := NewRestSource("test", server.URL, nil)
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err = src.Fetch(ctx)
	assert.Error(t, err)

	var httpErr *HTTPError
	assert.ErrorAs(t, err, &httpErr)
	assert.Equal(t, 404, httpErr.StatusCode)
}

func TestRestSourceFetch500(t *testing.T) {
	// Create test server that returns 500
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Internal Server Error"))
	}))
	defer server.Close()

	src, err := NewRestSource("test", server.URL, nil)
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err = src.Fetch(ctx)
	assert.Error(t, err)

	var httpErr *HTTPError
	assert.ErrorAs(t, err, &httpErr)
	assert.Equal(t, 500, httpErr.StatusCode)
}

func TestRestSourceFetchInvalidJSON(t *testing.T) {
	// Create test server that returns invalid JSON
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte("not valid json"))
	}))
	defer server.Close()

	src, err := NewRestSource("test", server.URL, nil)
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err = src.Fetch(ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "could not parse response as JSON")
}

func TestRestSourceHeaders(t *testing.T) {
	var receivedHeaders http.Header
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedHeaders = r.Header.Clone()
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte("[]"))
	}))
	defer server.Close()

	// Test with custom headers
	src, err := NewRestSource("test", server.URL, map[string]string{
		"headers": "X-Custom:value1,Authorization:Bearer token123",
	})
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err = src.Fetch(ctx)
	require.NoError(t, err)

	assert.Equal(t, "value1", receivedHeaders.Get("X-Custom"))
	assert.Equal(t, "Bearer token123", receivedHeaders.Get("Authorization"))
	assert.Equal(t, "application/json", receivedHeaders.Get("Accept"))
}

func TestRestSourceAuthHeader(t *testing.T) {
	var receivedHeaders http.Header
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedHeaders = r.Header.Clone()
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte("[]"))
	}))
	defer server.Close()

	// Test with auth_header option
	src, err := NewRestSource("test", server.URL, map[string]string{
		"auth_header": "Bearer my-secret-token",
	})
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err = src.Fetch(ctx)
	require.NoError(t, err)

	assert.Equal(t, "Bearer my-secret-token", receivedHeaders.Get("Authorization"))
}

func TestRestSourceApiKey(t *testing.T) {
	var receivedHeaders http.Header
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedHeaders = r.Header.Clone()
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte("[]"))
	}))
	defer server.Close()

	// Test with api_key option
	src, err := NewRestSource("test", server.URL, map[string]string{
		"api_key": "my-api-key",
	})
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err = src.Fetch(ctx)
	require.NoError(t, err)

	assert.Equal(t, "my-api-key", receivedHeaders.Get("X-API-Key"))
}

func TestRestSourceEnvVarExpansion(t *testing.T) {
	// Create a test server to verify the URL is correct
	var receivedPath string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte("[]"))
	}))
	defer server.Close()

	// Set env var for test using server URL
	oldVal := os.Getenv("TEST_REST_URL")
	os.Setenv("TEST_REST_URL", server.URL)
	defer os.Setenv("TEST_REST_URL", oldVal)

	src, err := NewRestSource("test", "${TEST_REST_URL}/api/data", nil)
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err = src.Fetch(ctx)
	require.NoError(t, err)
	// Verify the path portion was correctly expanded
	assert.Equal(t, "/api/data", receivedPath)
}

func TestRestSourceClose(t *testing.T) {
	src, err := NewRestSource("test", "http://localhost/api", nil)
	require.NoError(t, err)

	err = src.Close()
	assert.NoError(t, err)
}
