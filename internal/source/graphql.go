// Package source provides data source implementations for lvt-source blocks.
package source

import (
	"fmt"
	"net/http"
	"os"
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
