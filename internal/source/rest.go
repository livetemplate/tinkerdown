package source

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

// RestSource fetches data from a REST API endpoint
type RestSource struct {
	name    string
	url     string
	method  string
	headers map[string]string
	client  *http.Client
}

// NewRestSource creates a new REST API source
func NewRestSource(name, url string, options map[string]string) (*RestSource, error) {
	if url == "" {
		return nil, fmt.Errorf("rest source %q: url is required", name)
	}

	// Expand environment variables in URL
	url = os.ExpandEnv(url)

	method := "GET"
	if options != nil && options["method"] != "" {
		method = strings.ToUpper(options["method"])
	}

	// Parse headers from options (format: "key1:value1,key2:value2")
	headers := make(map[string]string)
	if options != nil && options["headers"] != "" {
		for _, h := range strings.Split(options["headers"], ",") {
			parts := strings.SplitN(h, ":", 2)
			if len(parts) == 2 {
				key := strings.TrimSpace(parts[0])
				value := strings.TrimSpace(parts[1])
				// Expand environment variables in header values
				headers[key] = os.ExpandEnv(value)
			}
		}
	}

	// Check for common auth headers from environment
	if options != nil {
		if authHeader := options["auth_header"]; authHeader != "" {
			headers["Authorization"] = os.ExpandEnv(authHeader)
		}
		if apiKey := options["api_key"]; apiKey != "" {
			headers["X-API-Key"] = os.ExpandEnv(apiKey)
		}
	}

	return &RestSource{
		name:    name,
		url:     url,
		method:  method,
		headers: headers,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}, nil
}

// Name returns the source identifier
func (s *RestSource) Name() string {
	return s.name
}

// Fetch makes an HTTP request and parses JSON response
func (s *RestSource) Fetch(ctx context.Context) ([]map[string]interface{}, error) {
	// Create request with context
	req, err := http.NewRequestWithContext(ctx, s.method, s.url, nil)
	if err != nil {
		return nil, fmt.Errorf("rest source %q: failed to create request: %w", s.name, err)
	}

	// Set headers
	req.Header.Set("Accept", "application/json")
	for key, value := range s.headers {
		req.Header.Set(key, value)
	}

	// Execute request
	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("rest source %q: request failed: %w", s.name, err)
	}
	defer resp.Body.Close()

	// Check status code
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return nil, fmt.Errorf("rest source %q: HTTP %d: %s", s.name, resp.StatusCode, string(body))
	}

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("rest source %q: failed to read response: %w", s.name, err)
	}

	// Parse JSON response
	return s.parseJSON(body)
}

// parseJSON handles both array and object JSON responses
func (s *RestSource) parseJSON(data []byte) ([]map[string]interface{}, error) {
	data = []byte(strings.TrimSpace(string(data)))

	if len(data) == 0 {
		return []map[string]interface{}{}, nil
	}

	// Try parsing as array first
	var arr []map[string]interface{}
	if err := json.Unmarshal(data, &arr); err == nil {
		return arr, nil
	}

	// Try parsing as single object
	var obj map[string]interface{}
	if err := json.Unmarshal(data, &obj); err == nil {
		// Check if the object has a "data" field that's an array (common API pattern)
		if dataField, ok := obj["data"]; ok {
			if dataArr, ok := dataField.([]interface{}); ok {
				results := make([]map[string]interface{}, 0, len(dataArr))
				for _, item := range dataArr {
					if itemMap, ok := item.(map[string]interface{}); ok {
						results = append(results, itemMap)
					}
				}
				if len(results) > 0 {
					return results, nil
				}
			}
		}
		// Check for "results" field (another common pattern)
		if resultsField, ok := obj["results"]; ok {
			if resultsArr, ok := resultsField.([]interface{}); ok {
				results := make([]map[string]interface{}, 0, len(resultsArr))
				for _, item := range resultsArr {
					if itemMap, ok := item.(map[string]interface{}); ok {
						results = append(results, itemMap)
					}
				}
				if len(results) > 0 {
					return results, nil
				}
			}
		}
		// Return as single-item array
		return []map[string]interface{}{obj}, nil
	}

	return nil, fmt.Errorf("rest source %q: could not parse response as JSON", s.name)
}

// Close is a no-op for REST sources
func (s *RestSource) Close() error {
	return nil
}
