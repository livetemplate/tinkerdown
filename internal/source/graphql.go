// Package source provides data source implementations for lvt-source blocks.
package source

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

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
			Message string        `json:"message"`
			Path    []interface{} `json:"path"`
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
		// Convert path to []string (handles both string and numeric indices)
		pathStrings := make([]string, 0, len(gqlResp.Errors[0].Path))
		for _, p := range gqlResp.Errors[0].Path {
			pathStrings = append(pathStrings, fmt.Sprint(p))
		}
		return nil, &GraphQLError{
			Source:  s.name,
			Message: gqlResp.Errors[0].Message,
			Path:    pathStrings,
		}
	}

	// Extract data using result_path
	if gqlResp.Data == nil {
		return []map[string]interface{}{}, nil
	}

	return extractPath(gqlResp.Data, s.resultPath)
}

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
	skippedCount := 0
	for _, item := range arr {
		if m, ok := item.(map[string]interface{}); ok {
			result = append(result, m)
		} else {
			skippedCount++
		}
	}

	if skippedCount > 0 {
		log.Printf("[graphql] warning: extractPath skipped %d non-object items at path '%s'", skippedCount, path)
	}

	return result, nil
}
