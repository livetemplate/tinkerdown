package source

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestCircuitBreakerInitialState(t *testing.T) {
	cb := NewCircuitBreaker("test", DefaultCircuitBreakerConfig())
	if cb.State() != CircuitClosed {
		t.Errorf("expected initial state to be Closed, got %v", cb.State())
	}
}

func TestCircuitBreakerSuccessfulRequests(t *testing.T) {
	cfg := CircuitBreakerConfig{
		FailureThreshold: 3,
		SuccessThreshold: 2,
		Timeout:          100 * time.Millisecond,
		FailureWindow:    1 * time.Minute,
	}
	cb := NewCircuitBreaker("test", cfg)

	// Successful requests should keep circuit closed
	for i := 0; i < 10; i++ {
		result, err := cb.Execute(context.Background(), func(ctx context.Context) ([]map[string]interface{}, error) {
			return []map[string]interface{}{{"i": i}}, nil
		})
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if len(result) != 1 {
			t.Errorf("unexpected result: %v", result)
		}
	}

	if cb.State() != CircuitClosed {
		t.Errorf("expected circuit to remain Closed, got %v", cb.State())
	}
}

func TestCircuitBreakerOpensOnFailures(t *testing.T) {
	cfg := CircuitBreakerConfig{
		FailureThreshold: 3,
		SuccessThreshold: 2,
		Timeout:          100 * time.Millisecond,
		FailureWindow:    1 * time.Minute,
	}
	cb := NewCircuitBreaker("test", cfg)

	// Generate failures
	for i := 0; i < 3; i++ {
		cb.Execute(context.Background(), func(ctx context.Context) ([]map[string]interface{}, error) {
			return nil, &SourceError{Source: "test", Retryable: true}
		})
	}

	if cb.State() != CircuitOpen {
		t.Errorf("expected circuit to be Open after %d failures, got %v", 3, cb.State())
	}

	// Requests should be blocked
	_, err := cb.Execute(context.Background(), func(ctx context.Context) ([]map[string]interface{}, error) {
		t.Error("function should not be called when circuit is open")
		return nil, nil
	})

	var circuitErr *CircuitOpenError
	if !errors.As(err, &circuitErr) {
		t.Errorf("expected CircuitOpenError, got %T", err)
	}
}

func TestCircuitBreakerTransitionsToHalfOpen(t *testing.T) {
	cfg := CircuitBreakerConfig{
		FailureThreshold: 2,
		SuccessThreshold: 2,
		Timeout:          50 * time.Millisecond, // Short timeout for testing
		FailureWindow:    1 * time.Minute,
	}
	cb := NewCircuitBreaker("test", cfg)

	// Open the circuit
	for i := 0; i < 2; i++ {
		cb.Execute(context.Background(), func(ctx context.Context) ([]map[string]interface{}, error) {
			return nil, &SourceError{Source: "test", Retryable: true}
		})
	}

	if cb.State() != CircuitOpen {
		t.Errorf("expected circuit to be Open, got %v", cb.State())
	}

	// Wait for timeout
	time.Sleep(60 * time.Millisecond)

	// Next request should transition to half-open
	cb.Execute(context.Background(), func(ctx context.Context) ([]map[string]interface{}, error) {
		return []map[string]interface{}{}, nil
	})

	// State should be half-open or closed (depending on success)
	state := cb.State()
	if state != CircuitHalfOpen && state != CircuitClosed {
		t.Errorf("expected circuit to be HalfOpen or Closed, got %v", state)
	}
}

func TestCircuitBreakerClosesAfterSuccesses(t *testing.T) {
	cfg := CircuitBreakerConfig{
		FailureThreshold: 2,
		SuccessThreshold: 2,
		Timeout:          50 * time.Millisecond,
		FailureWindow:    1 * time.Minute,
	}
	cb := NewCircuitBreaker("test", cfg)

	// Open the circuit
	for i := 0; i < 2; i++ {
		cb.Execute(context.Background(), func(ctx context.Context) ([]map[string]interface{}, error) {
			return nil, &SourceError{Source: "test", Retryable: true}
		})
	}

	// Wait for timeout to transition to half-open
	time.Sleep(60 * time.Millisecond)

	// Successful requests should close the circuit
	for i := 0; i < 2; i++ {
		cb.Execute(context.Background(), func(ctx context.Context) ([]map[string]interface{}, error) {
			return []map[string]interface{}{}, nil
		})
	}

	if cb.State() != CircuitClosed {
		t.Errorf("expected circuit to be Closed after successful requests, got %v", cb.State())
	}
}

func TestCircuitBreakerReopensOnHalfOpenFailure(t *testing.T) {
	cfg := CircuitBreakerConfig{
		FailureThreshold: 2,
		SuccessThreshold: 2,
		Timeout:          50 * time.Millisecond,
		FailureWindow:    1 * time.Minute,
	}
	cb := NewCircuitBreaker("test", cfg)

	// Open the circuit
	for i := 0; i < 2; i++ {
		cb.Execute(context.Background(), func(ctx context.Context) ([]map[string]interface{}, error) {
			return nil, &SourceError{Source: "test", Retryable: true}
		})
	}

	// Wait for timeout
	time.Sleep(60 * time.Millisecond)

	// Failure in half-open should reopen circuit
	cb.Execute(context.Background(), func(ctx context.Context) ([]map[string]interface{}, error) {
		return nil, &SourceError{Source: "test", Retryable: true}
	})

	if cb.State() != CircuitOpen {
		t.Errorf("expected circuit to reopen on half-open failure, got %v", cb.State())
	}
}

func TestCircuitBreakerReset(t *testing.T) {
	cfg := CircuitBreakerConfig{
		FailureThreshold: 2,
		SuccessThreshold: 2,
		Timeout:          1 * time.Minute,
		FailureWindow:    1 * time.Minute,
	}
	cb := NewCircuitBreaker("test", cfg)

	// Open the circuit
	for i := 0; i < 2; i++ {
		cb.Execute(context.Background(), func(ctx context.Context) ([]map[string]interface{}, error) {
			return nil, &SourceError{Source: "test", Retryable: true}
		})
	}

	if cb.State() != CircuitOpen {
		t.Errorf("expected circuit to be Open, got %v", cb.State())
	}

	// Reset should close the circuit
	cb.Reset()

	if cb.State() != CircuitClosed {
		t.Errorf("expected circuit to be Closed after reset, got %v", cb.State())
	}

	// Should be able to execute again
	_, err := cb.Execute(context.Background(), func(ctx context.Context) ([]map[string]interface{}, error) {
		return []map[string]interface{}{}, nil
	})

	if err != nil {
		t.Errorf("unexpected error after reset: %v", err)
	}
}

func TestCircuitBreakerNonRetryableErrorsDontCount(t *testing.T) {
	cfg := CircuitBreakerConfig{
		FailureThreshold: 2,
		SuccessThreshold: 2,
		Timeout:          1 * time.Minute,
		FailureWindow:    1 * time.Minute,
	}
	cb := NewCircuitBreaker("test", cfg)

	// Non-retryable errors shouldn't open the circuit
	for i := 0; i < 10; i++ {
		cb.Execute(context.Background(), func(ctx context.Context) ([]map[string]interface{}, error) {
			return nil, &ValidationError{Source: "test", Reason: "invalid"}
		})
	}

	if cb.State() != CircuitClosed {
		t.Errorf("expected circuit to remain Closed for non-retryable errors, got %v", cb.State())
	}
}

func TestCircuitBreakerFailureWindowExpiry(t *testing.T) {
	cfg := CircuitBreakerConfig{
		FailureThreshold: 3,
		SuccessThreshold: 2,
		Timeout:          1 * time.Minute,
		FailureWindow:    50 * time.Millisecond, // Short window for testing
	}
	cb := NewCircuitBreaker("test", cfg)

	// Add some failures
	cb.Execute(context.Background(), func(ctx context.Context) ([]map[string]interface{}, error) {
		return nil, &SourceError{Source: "test", Retryable: true}
	})

	// Wait for failure window to expire
	time.Sleep(60 * time.Millisecond)

	// These failures should start fresh (old ones expired)
	for i := 0; i < 2; i++ {
		cb.Execute(context.Background(), func(ctx context.Context) ([]map[string]interface{}, error) {
			return nil, &SourceError{Source: "test", Retryable: true}
		})
	}

	// Circuit should still be closed (only 2 failures in window)
	if cb.State() != CircuitClosed {
		t.Errorf("expected circuit to remain Closed with expired failures, got %v", cb.State())
	}
}

func TestCircuitStateString(t *testing.T) {
	tests := []struct {
		state    CircuitState
		expected string
	}{
		{CircuitClosed, "closed"},
		{CircuitOpen, "open"},
		{CircuitHalfOpen, "half-open"},
		{CircuitState(99), "unknown"},
	}

	for _, tt := range tests {
		if tt.state.String() != tt.expected {
			t.Errorf("expected %q, got %q", tt.expected, tt.state.String())
		}
	}
}

func TestDefaultCircuitBreakerConfig(t *testing.T) {
	cfg := DefaultCircuitBreakerConfig()

	if cfg.FailureThreshold != 5 {
		t.Errorf("expected FailureThreshold=5, got %d", cfg.FailureThreshold)
	}
	if cfg.SuccessThreshold != 2 {
		t.Errorf("expected SuccessThreshold=2, got %d", cfg.SuccessThreshold)
	}
	if cfg.Timeout != 30*time.Second {
		t.Errorf("expected Timeout=30s, got %v", cfg.Timeout)
	}
	if cfg.FailureWindow != 1*time.Minute {
		t.Errorf("expected FailureWindow=1m, got %v", cfg.FailureWindow)
	}
}
