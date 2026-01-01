// Package source provides data source implementations for lvt-source blocks.
package source

import (
	"errors"
	"fmt"
	"net"
	"strings"
)

// SourceError wraps errors with source context
type SourceError struct {
	Source    string // Source name (e.g., "users")
	Operation string // Operation that failed (e.g., "fetch", "connect")
	Err       error  // Underlying error
	Retryable bool   // Whether this error is retryable
}

func (e *SourceError) Error() string {
	if e.Operation != "" {
		return fmt.Sprintf("source %q %s failed: %v", e.Source, e.Operation, e.Err)
	}
	return fmt.Sprintf("source %q: %v", e.Source, e.Err)
}

func (e *SourceError) Unwrap() error {
	return e.Err
}

// IsRetryable returns true if the error is retryable
func (e *SourceError) IsRetryable() bool {
	return e.Retryable
}

// ConnectionError represents a connection failure
type ConnectionError struct {
	Source  string
	Address string
	Err     error
}

func (e *ConnectionError) Error() string {
	return fmt.Sprintf("source %q: connection to %s failed: %v", e.Source, e.Address, e.Err)
}

func (e *ConnectionError) Unwrap() error {
	return e.Err
}

// TimeoutError represents a timeout
type TimeoutError struct {
	Source    string
	Operation string
	Duration  string
}

func (e *TimeoutError) Error() string {
	return fmt.Sprintf("source %q: %s timed out after %s", e.Source, e.Operation, e.Duration)
}

// ValidationError represents invalid data or configuration
type ValidationError struct {
	Source string
	Field  string
	Reason string
}

func (e *ValidationError) Error() string {
	if e.Field != "" {
		return fmt.Sprintf("source %q: invalid %s: %s", e.Source, e.Field, e.Reason)
	}
	return fmt.Sprintf("source %q: validation failed: %s", e.Source, e.Reason)
}

// HTTPError represents an HTTP error response
type HTTPError struct {
	Source     string
	StatusCode int
	Status     string
	Body       string
}

func (e *HTTPError) Error() string {
	if e.Body != "" {
		return fmt.Sprintf("source %q: HTTP %d %s: %s", e.Source, e.StatusCode, e.Status, e.Body)
	}
	return fmt.Sprintf("source %q: HTTP %d %s", e.Source, e.StatusCode, e.Status)
}

// IsRetryable returns true for 5xx errors and 429 (rate limit)
func (e *HTTPError) IsRetryable() bool {
	return e.StatusCode >= 500 || e.StatusCode == 429
}

// GraphQLError represents an error returned by a GraphQL API
type GraphQLError struct {
	Source  string
	Message string
	Path    []string
}

func (e *GraphQLError) Error() string {
	if len(e.Path) > 0 {
		return fmt.Sprintf("source %q: graphql error at %v: %s", e.Source, e.Path, e.Message)
	}
	return fmt.Sprintf("source %q: graphql error: %s", e.Source, e.Message)
}

// CircuitOpenError indicates the circuit breaker is open
type CircuitOpenError struct {
	Source string
}

func (e *CircuitOpenError) Error() string {
	return fmt.Sprintf("source %q: circuit breaker open, service temporarily unavailable", e.Source)
}

// NewSourceError creates a SourceError with retryable detection
func NewSourceError(source, operation string, err error) *SourceError {
	return &SourceError{
		Source:    source,
		Operation: operation,
		Err:       err,
		Retryable: isRetryableError(err),
	}
}

// isRetryableError checks if an error is retryable
func isRetryableError(err error) bool {
	if err == nil {
		return false
	}

	// Check for specific retryable error types
	var httpErr *HTTPError
	if errors.As(err, &httpErr) {
		return httpErr.IsRetryable()
	}

	var connErr *ConnectionError
	if errors.As(err, &connErr) {
		return true // Connection errors are always retryable
	}

	var timeoutErr *TimeoutError
	if errors.As(err, &timeoutErr) {
		return true // Timeouts are retryable
	}

	// Check for network errors
	var netErr net.Error
	if errors.As(err, &netErr) {
		return netErr.Timeout() || netErr.Temporary()
	}

	// Check for common transient error messages
	errStr := strings.ToLower(err.Error())
	transientPatterns := []string{
		"connection refused",
		"connection reset",
		"no such host",
		"timeout",
		"deadline exceeded",
		"temporary failure",
		"try again",
		"service unavailable",
		"bad gateway",
		"gateway timeout",
	}
	for _, pattern := range transientPatterns {
		if strings.Contains(errStr, pattern) {
			return true
		}
	}

	return false
}

// UserFriendlyMessage returns a user-friendly error message for templates
func UserFriendlyMessage(err error) string {
	if err == nil {
		return ""
	}

	// Check specific error types
	var circuitErr *CircuitOpenError
	if errors.As(err, &circuitErr) {
		return "Service temporarily unavailable. Please try again later."
	}

	var httpErr *HTTPError
	if errors.As(err, &httpErr) {
		switch {
		case httpErr.StatusCode == 401:
			return "Authentication required."
		case httpErr.StatusCode == 403:
			return "Access denied."
		case httpErr.StatusCode == 404:
			return "Resource not found."
		case httpErr.StatusCode == 429:
			return "Too many requests. Please slow down."
		case httpErr.StatusCode >= 500:
			return "Server error. Please try again later."
		default:
			return fmt.Sprintf("Request failed (HTTP %d).", httpErr.StatusCode)
		}
	}

	var timeoutErr *TimeoutError
	if errors.As(err, &timeoutErr) {
		return "Request timed out. Please try again."
	}

	var connErr *ConnectionError
	if errors.As(err, &connErr) {
		return "Could not connect to data source. Please check your connection."
	}

	var validationErr *ValidationError
	if errors.As(err, &validationErr) {
		return fmt.Sprintf("Invalid data: %s", validationErr.Reason)
	}

	var gqlErr *GraphQLError
	if errors.As(err, &gqlErr) {
		return fmt.Sprintf("GraphQL query failed: %s", gqlErr.Message)
	}

	// Generic fallback
	return "Failed to load data. Please try again."
}
