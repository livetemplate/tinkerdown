package source

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestWithRetrySuccess(t *testing.T) {
	cfg := RetryConfig{
		MaxRetries: 3,
		BaseDelay:  10 * time.Millisecond,
		MaxDelay:   100 * time.Millisecond,
		Multiplier: 2.0,
	}

	calls := 0
	result, err := WithRetry(context.Background(), "test", cfg, func(ctx context.Context) ([]map[string]interface{}, error) {
		calls++
		return []map[string]interface{}{{"key": "value"}}, nil
	})

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if calls != 1 {
		t.Errorf("expected 1 call, got %d", calls)
	}
	if len(result) != 1 || result[0]["key"] != "value" {
		t.Errorf("unexpected result: %v", result)
	}
}

func TestWithRetryRetryableError(t *testing.T) {
	cfg := RetryConfig{
		MaxRetries: 3,
		BaseDelay:  1 * time.Millisecond,
		MaxDelay:   10 * time.Millisecond,
		Multiplier: 2.0,
	}

	calls := 0
	_, err := WithRetry(context.Background(), "test", cfg, func(ctx context.Context) ([]map[string]interface{}, error) {
		calls++
		// Return a retryable error
		return nil, &SourceError{Source: "test", Operation: "fetch", Err: errors.New("timeout"), Retryable: true}
	})

	if err == nil {
		t.Error("expected error")
	}
	// Should be called MaxRetries + 1 times (initial + retries)
	if calls != 4 {
		t.Errorf("expected 4 calls (1 initial + 3 retries), got %d", calls)
	}
}

func TestWithRetryNonRetryableError(t *testing.T) {
	cfg := RetryConfig{
		MaxRetries: 3,
		BaseDelay:  1 * time.Millisecond,
		MaxDelay:   10 * time.Millisecond,
		Multiplier: 2.0,
	}

	calls := 0
	_, err := WithRetry(context.Background(), "test", cfg, func(ctx context.Context) ([]map[string]interface{}, error) {
		calls++
		// Return a non-retryable error
		return nil, &SourceError{Source: "test", Operation: "parse", Err: errors.New("invalid json"), Retryable: false}
	})

	if err == nil {
		t.Error("expected error")
	}
	// Should only be called once (no retries for non-retryable errors)
	if calls != 1 {
		t.Errorf("expected 1 call (no retries), got %d", calls)
	}
}

func TestWithRetryEventualSuccess(t *testing.T) {
	cfg := RetryConfig{
		MaxRetries: 3,
		BaseDelay:  1 * time.Millisecond,
		MaxDelay:   10 * time.Millisecond,
		Multiplier: 2.0,
	}

	calls := 0
	result, err := WithRetry(context.Background(), "test", cfg, func(ctx context.Context) ([]map[string]interface{}, error) {
		calls++
		if calls < 3 {
			// Fail first 2 times with retryable error
			return nil, &SourceError{Source: "test", Operation: "fetch", Err: errors.New("temporary"), Retryable: true}
		}
		// Succeed on 3rd attempt
		return []map[string]interface{}{{"success": true}}, nil
	})

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if calls != 3 {
		t.Errorf("expected 3 calls, got %d", calls)
	}
	if len(result) != 1 || result[0]["success"] != true {
		t.Errorf("unexpected result: %v", result)
	}
}

func TestWithRetryContextCanceled(t *testing.T) {
	cfg := RetryConfig{
		MaxRetries: 10,
		BaseDelay:  100 * time.Millisecond,
		MaxDelay:   1 * time.Second,
		Multiplier: 2.0,
	}

	ctx, cancel := context.WithCancel(context.Background())
	calls := 0

	// Cancel after first call
	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	_, err := WithRetry(ctx, "test", cfg, func(ctx context.Context) ([]map[string]interface{}, error) {
		calls++
		return nil, &SourceError{Source: "test", Operation: "fetch", Err: errors.New("fail"), Retryable: true}
	})

	if err == nil {
		t.Error("expected error")
	}
	// Should stop retrying when context is canceled
	if calls > 2 {
		t.Errorf("expected at most 2 calls before context cancel, got %d", calls)
	}
}

func TestWithRetryHTTPRetryable(t *testing.T) {
	cfg := RetryConfig{
		MaxRetries: 2,
		BaseDelay:  1 * time.Millisecond,
		MaxDelay:   10 * time.Millisecond,
		Multiplier: 2.0,
	}

	tests := []struct {
		name          string
		statusCode    int
		expectedCalls int
	}{
		{"500 should retry", 500, 3},
		{"502 should retry", 502, 3},
		{"503 should retry", 503, 3},
		{"429 should retry", 429, 3},
		{"400 should not retry", 400, 1},
		{"401 should not retry", 401, 1},
		{"404 should not retry", 404, 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			calls := 0
			WithRetry(context.Background(), "test", cfg, func(ctx context.Context) ([]map[string]interface{}, error) {
				calls++
				return nil, &HTTPError{Source: "test", StatusCode: tt.statusCode, Status: "error"}
			})
			if calls != tt.expectedCalls {
				t.Errorf("expected %d calls, got %d", tt.expectedCalls, calls)
			}
		})
	}
}

func TestShouldRetry(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "retryable source error",
			err:      &SourceError{Retryable: true},
			expected: true,
		},
		{
			name:     "non-retryable source error",
			err:      &SourceError{Retryable: false},
			expected: false,
		},
		{
			name:     "http 500",
			err:      &HTTPError{StatusCode: 500},
			expected: true,
		},
		{
			name:     "http 404",
			err:      &HTTPError{StatusCode: 404},
			expected: false,
		},
		{
			name:     "connection error",
			err:      &ConnectionError{Source: "test"},
			expected: true,
		},
		{
			name:     "timeout error",
			err:      &TimeoutError{Source: "test"},
			expected: true,
		},
		{
			name:     "circuit open error",
			err:      &CircuitOpenError{Source: "test"},
			expected: false,
		},
		{
			name:     "validation error",
			err:      &ValidationError{Source: "test"},
			expected: false,
		},
		{
			name:     "generic error",
			err:      errors.New("unknown"),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if shouldRetry(tt.err) != tt.expected {
				t.Errorf("expected shouldRetry=%v for %T", tt.expected, tt.err)
			}
		})
	}
}
