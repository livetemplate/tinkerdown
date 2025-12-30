package source

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/livetemplate/tinkerdown/internal/config"
)

// RestSource fetches data from a REST API endpoint
type RestSource struct {
	name           string
	url            string
	method         string
	headers        map[string]string
	client         *http.Client
	retryConfig    RetryConfig
	circuitBreaker *CircuitBreaker
}

// NewRestSource creates a new REST API source
func NewRestSource(name, url string, options map[string]string) (*RestSource, error) {
	return NewRestSourceWithConfig(name, url, options, config.SourceConfig{})
}

// NewRestSourceWithConfig creates a new REST API source with full configuration
func NewRestSourceWithConfig(name, url string, options map[string]string, cfg config.SourceConfig) (*RestSource, error) {
	if url == "" {
		return nil, &ValidationError{Source: name, Field: "url", Reason: "url is required"}
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

	return &RestSource{
		name:           name,
		url:            url,
		method:         method,
		headers:        headers,
		retryConfig:    retryConfig,
		circuitBreaker: circuitBreaker,
		client: &http.Client{
			Timeout: timeout,
		},
	}, nil
}

// Name returns the source identifier
func (s *RestSource) Name() string {
	return s.name
}

// Fetch makes an HTTP request and parses JSON response with retry and circuit breaker
func (s *RestSource) Fetch(ctx context.Context) ([]map[string]interface{}, error) {
	// Use circuit breaker + retry
	return s.circuitBreaker.Execute(ctx, func(ctx context.Context) ([]map[string]interface{}, error) {
		return WithRetry(ctx, s.name, s.retryConfig, func(ctx context.Context) ([]map[string]interface{}, error) {
			return s.doFetch(ctx)
		})
	})
}

// doFetch performs the actual HTTP request
func (s *RestSource) doFetch(ctx context.Context) ([]map[string]interface{}, error) {
	// Create request with context
	req, err := http.NewRequestWithContext(ctx, s.method, s.url, nil)
	if err != nil {
		return nil, &SourceError{
			Source:    s.name,
			Operation: "create request",
			Err:       err,
			Retryable: false,
		}
	}

	// Set headers
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

	// Check status code
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return nil, &HTTPError{
			Source:     s.name,
			StatusCode: resp.StatusCode,
			Status:     resp.Status,
			Body:       strings.TrimSpace(string(body)),
		}
	}

	// Read response body with size limit to prevent OOM
	const maxResponseSize = 10 * 1024 * 1024 // 10MB
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseSize))
	if err != nil {
		return nil, &SourceError{
			Source:    s.name,
			Operation: "read response",
			Err:       err,
			Retryable: false,
		}
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

	return nil, &ValidationError{Source: s.name, Reason: "could not parse response as JSON"}
}

// Close is a no-op for REST sources
func (s *RestSource) Close() error {
	return nil
}
