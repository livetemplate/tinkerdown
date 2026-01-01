package source

import (
	"errors"
	"testing"
)

func TestSourceErrorError(t *testing.T) {
	err := &SourceError{
		Source:    "test-source",
		Operation: "fetch",
		Err:       errors.New("connection refused"),
		Retryable: true,
	}

	msg := err.Error()
	expected := `source "test-source" fetch failed: connection refused`
	if msg != expected {
		t.Errorf("expected %q, got %q", expected, msg)
	}
}

func TestSourceErrorUnwrap(t *testing.T) {
	underlying := errors.New("underlying error")
	err := &SourceError{
		Source:    "test",
		Operation: "op",
		Err:       underlying,
	}

	if !errors.Is(err, underlying) {
		t.Error("SourceError should unwrap to underlying error")
	}
}

func TestConnectionErrorError(t *testing.T) {
	err := &ConnectionError{
		Source:  "pg-source",
		Address: "localhost:5432",
		Err:     errors.New("no route to host"),
	}

	msg := err.Error()
	expected := `source "pg-source": connection to localhost:5432 failed: no route to host`
	if msg != expected {
		t.Errorf("expected %q, got %q", expected, msg)
	}
}

func TestTimeoutErrorError(t *testing.T) {
	err := &TimeoutError{
		Source:    "rest-api",
		Operation: "fetch",
		Duration:  "30s",
	}

	msg := err.Error()
	expected := `source "rest-api": fetch timed out after 30s`
	if msg != expected {
		t.Errorf("expected %q, got %q", expected, msg)
	}
}

func TestValidationErrorError(t *testing.T) {
	tests := []struct {
		name     string
		err      *ValidationError
		expected string
	}{
		{
			name: "with field",
			err: &ValidationError{
				Source: "json-source",
				Field:  "file",
				Reason: "file not found",
			},
			expected: `source "json-source": invalid file: file not found`,
		},
		{
			name: "without field",
			err: &ValidationError{
				Source: "csv-source",
				Reason: "invalid format",
			},
			expected: `source "csv-source": validation failed: invalid format`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.err.Error() != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, tt.err.Error())
			}
		})
	}
}

func TestHTTPErrorError(t *testing.T) {
	err := &HTTPError{
		Source:     "api",
		StatusCode: 404,
		Status:     "404 Not Found",
		Body:       "resource not found",
	}

	msg := err.Error()
	expected := `source "api": HTTP 404 404 Not Found: resource not found`
	if msg != expected {
		t.Errorf("expected %q, got %q", expected, msg)
	}
}

func TestHTTPErrorIsRetryable(t *testing.T) {
	tests := []struct {
		statusCode int
		retryable  bool
	}{
		{200, false},
		{400, false},
		{401, false},
		{403, false},
		{404, false},
		{429, true}, // Too Many Requests
		{500, true}, // Internal Server Error
		{502, true}, // Bad Gateway
		{503, true}, // Service Unavailable
		{504, true}, // Gateway Timeout
	}

	for _, tt := range tests {
		err := &HTTPError{StatusCode: tt.statusCode}
		if err.IsRetryable() != tt.retryable {
			t.Errorf("status %d: expected retryable=%v, got %v", tt.statusCode, tt.retryable, err.IsRetryable())
		}
	}
}

func TestCircuitOpenErrorError(t *testing.T) {
	err := &CircuitOpenError{Source: "flaky-api"}

	msg := err.Error()
	expected := `source "flaky-api": circuit breaker open, service temporarily unavailable`
	if msg != expected {
		t.Errorf("expected %q, got %q", expected, msg)
	}
}

func TestNewSourceError(t *testing.T) {
	// Test with context.DeadlineExceeded
	err := NewSourceError("test", "fetch", errors.New("context deadline exceeded"))
	if !err.Retryable {
		t.Error("deadline exceeded should be retryable")
	}

	// Test with regular error
	err = NewSourceError("test", "parse", errors.New("invalid json"))
	if err.Retryable {
		t.Error("parse error should not be retryable")
	}
}

func TestUserFriendlyMessage(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected string
	}{
		{
			name:     "connection error",
			err:      &ConnectionError{Source: "db", Address: "localhost", Err: errors.New("refused")},
			expected: "Could not connect to data source. Please check your connection.",
		},
		{
			name:     "timeout error",
			err:      &TimeoutError{Source: "api", Operation: "fetch", Duration: "30s"},
			expected: "Request timed out. Please try again.",
		},
		{
			name:     "circuit open error",
			err:      &CircuitOpenError{Source: "service"},
			expected: "Service temporarily unavailable. Please try again later.",
		},
		{
			name:     "http 401 error",
			err:      &HTTPError{Source: "api", StatusCode: 401},
			expected: "Authentication required.",
		},
		{
			name:     "http 403 error",
			err:      &HTTPError{Source: "api", StatusCode: 403},
			expected: "Access denied.",
		},
		{
			name:     "http 404 error",
			err:      &HTTPError{Source: "api", StatusCode: 404},
			expected: "Resource not found.",
		},
		{
			name:     "http 429 error",
			err:      &HTTPError{Source: "api", StatusCode: 429},
			expected: "Too many requests. Please slow down.",
		},
		{
			name:     "http 500 error",
			err:      &HTTPError{Source: "api", StatusCode: 500},
			expected: "Server error. Please try again later.",
		},
		{
			name:     "validation error",
			err:      &ValidationError{Source: "csv", Field: "file", Reason: "not found"},
			expected: "Invalid data: not found",
		},
		{
			name:     "graphql error",
			err:      &GraphQLError{Source: "github", Message: "Field 'user' not found", Path: []string{"query", "user"}},
			expected: "GraphQL query failed: Field 'user' not found",
		},
		{
			name:     "generic error",
			err:      errors.New("something went wrong"),
			expected: "Failed to load data. Please try again.",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := UserFriendlyMessage(tt.err)
			if msg != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, msg)
			}
		})
	}
}
