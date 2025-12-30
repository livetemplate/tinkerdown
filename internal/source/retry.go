package source

import (
	"context"
	"errors"
	"log"
	"math"
	"math/rand"
	"time"
)

// RetryConfig configures retry behavior
type RetryConfig struct {
	MaxRetries  int           // Maximum number of retry attempts (default: 3)
	BaseDelay   time.Duration // Initial delay between retries (default: 100ms)
	MaxDelay    time.Duration // Maximum delay between retries (default: 5s)
	Multiplier  float64       // Delay multiplier for exponential backoff (default: 2.0)
	EnableLog   bool          // Whether to log retry attempts
}

// DefaultRetryConfig returns the default retry configuration
func DefaultRetryConfig() RetryConfig {
	return RetryConfig{
		MaxRetries: 3,
		BaseDelay:  100 * time.Millisecond,
		MaxDelay:   5 * time.Second,
		Multiplier: 2.0,
		EnableLog:  true,
	}
}

// RetryableFunc is a function that can be retried
type RetryableFunc func(ctx context.Context) ([]map[string]interface{}, error)

// WithRetry wraps a function with retry logic
func WithRetry(ctx context.Context, source string, cfg RetryConfig, fn RetryableFunc) ([]map[string]interface{}, error) {
	var lastErr error

	for attempt := 0; attempt <= cfg.MaxRetries; attempt++ {
		// Check context before each attempt
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}

		result, err := fn(ctx)
		if err == nil {
			if attempt > 0 && cfg.EnableLog {
				log.Printf("[source/%s] Succeeded on attempt %d", source, attempt+1)
			}
			return result, nil
		}

		lastErr = err

		// Check if error is retryable
		if !shouldRetry(err) {
			if cfg.EnableLog {
				log.Printf("[source/%s] Non-retryable error: %v", source, err)
			}
			return nil, err
		}

		// Don't sleep after the last attempt
		if attempt < cfg.MaxRetries {
			delay := calculateDelay(attempt, cfg)
			if cfg.EnableLog {
				log.Printf("[source/%s] Attempt %d failed (%v), retrying in %v...", source, attempt+1, err, delay)
			}

			select {
			case <-time.After(delay):
				// Continue to next attempt
			case <-ctx.Done():
				return nil, ctx.Err()
			}
		}
	}

	if cfg.EnableLog {
		log.Printf("[source/%s] All %d attempts failed", source, cfg.MaxRetries+1)
	}

	// If lastErr is already a SourceError, update its Retryable flag instead of wrapping
	var sourceErr *SourceError
	if errors.As(lastErr, &sourceErr) {
		sourceErr.Retryable = false // Already exhausted retries
		return nil, lastErr
	}

	return nil, &SourceError{
		Source:    source,
		Operation: "fetch",
		Err:       lastErr,
		Retryable: false, // Already exhausted retries
	}
}

// shouldRetry determines if an error should be retried
func shouldRetry(err error) bool {
	if err == nil {
		return false
	}

	// Check for SourceError with Retryable flag
	var sourceErr *SourceError
	if errors.As(err, &sourceErr) {
		return sourceErr.Retryable
	}

	// Check for HTTPError
	var httpErr *HTTPError
	if errors.As(err, &httpErr) {
		return httpErr.IsRetryable()
	}

	// Check for circuit open (don't retry)
	var circuitErr *CircuitOpenError
	if errors.As(err, &circuitErr) {
		return false
	}

	// Check for validation errors (don't retry)
	var validationErr *ValidationError
	if errors.As(err, &validationErr) {
		return false
	}

	// Use generic retryable check
	return isRetryableError(err)
}

// calculateDelay computes the delay for the given attempt using exponential backoff with jitter
func calculateDelay(attempt int, cfg RetryConfig) time.Duration {
	// Exponential backoff: baseDelay * multiplier^attempt
	delay := float64(cfg.BaseDelay) * math.Pow(cfg.Multiplier, float64(attempt))

	// Cap at max delay
	if delay > float64(cfg.MaxDelay) {
		delay = float64(cfg.MaxDelay)
	}

	// Add jitter: randomize between 80% and 120% of delay to prevent thundering herd
	jitter := 0.8 + rand.Float64()*0.4 // Range: [0.8, 1.2)
	delay *= jitter

	return time.Duration(delay)
}
