package source

import (
	"context"
	"log"
	"sync"
	"time"
)

// CircuitState represents the state of a circuit breaker
type CircuitState int

const (
	CircuitClosed   CircuitState = iota // Normal operation, requests allowed
	CircuitOpen                         // Failures exceeded threshold, requests blocked
	CircuitHalfOpen                     // Testing if service recovered
)

func (s CircuitState) String() string {
	switch s {
	case CircuitClosed:
		return "closed"
	case CircuitOpen:
		return "open"
	case CircuitHalfOpen:
		return "half-open"
	default:
		return "unknown"
	}
}

// CircuitBreakerConfig configures the circuit breaker
type CircuitBreakerConfig struct {
	FailureThreshold int           // Number of failures to open circuit (default: 5)
	SuccessThreshold int           // Number of successes in half-open to close (default: 2)
	Timeout          time.Duration // Time to wait before half-open (default: 30s)
	FailureWindow    time.Duration // Window to count failures (default: 1 minute)
	EnableLog        bool          // Whether to log state changes
}

// DefaultCircuitBreakerConfig returns the default circuit breaker configuration
func DefaultCircuitBreakerConfig() CircuitBreakerConfig {
	return CircuitBreakerConfig{
		FailureThreshold: 5,
		SuccessThreshold: 2,
		Timeout:          30 * time.Second,
		FailureWindow:    1 * time.Minute,
		EnableLog:        true,
	}
}

// CircuitBreaker implements the circuit breaker pattern
type CircuitBreaker struct {
	name   string
	config CircuitBreakerConfig

	mu              sync.RWMutex
	state           CircuitState
	failures        []time.Time // Recent failure timestamps
	successes       int         // Consecutive successes in half-open state
	lastStateChange time.Time
}

// NewCircuitBreaker creates a new circuit breaker
func NewCircuitBreaker(name string, config CircuitBreakerConfig) *CircuitBreaker {
	return &CircuitBreaker{
		name:            name,
		config:          config,
		state:           CircuitClosed,
		failures:        make([]time.Time, 0),
		lastStateChange: time.Now(),
	}
}

// Execute runs a function through the circuit breaker
func (cb *CircuitBreaker) Execute(ctx context.Context, fn RetryableFunc) ([]map[string]interface{}, error) {
	if !cb.canExecute() {
		return nil, &CircuitOpenError{Source: cb.name}
	}

	result, err := fn(ctx)

	cb.recordResult(err)

	return result, err
}

// canExecute checks if a request should be allowed
func (cb *CircuitBreaker) canExecute() bool {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	now := time.Now()

	switch cb.state {
	case CircuitClosed:
		return true

	case CircuitOpen:
		// Check if timeout has passed
		if now.Sub(cb.lastStateChange) >= cb.config.Timeout {
			cb.transitionTo(CircuitHalfOpen)
			return true
		}
		return false

	case CircuitHalfOpen:
		// Allow limited requests to test recovery
		return true

	default:
		return true
	}
}

// recordResult updates the circuit breaker based on request result
func (cb *CircuitBreaker) recordResult(err error) {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	now := time.Now()

	if err == nil {
		cb.recordSuccess()
	} else if shouldRetry(err) {
		// Only count retryable errors as failures
		cb.recordFailure(now)
	}
}

// recordSuccess handles a successful request
func (cb *CircuitBreaker) recordSuccess() {
	switch cb.state {
	case CircuitHalfOpen:
		cb.successes++
		if cb.successes >= cb.config.SuccessThreshold {
			cb.transitionTo(CircuitClosed)
		}
	case CircuitClosed:
		// Reset failure count on success
		cb.failures = cb.failures[:0]
	}
}

// recordFailure handles a failed request
func (cb *CircuitBreaker) recordFailure(now time.Time) {
	// Add failure timestamp
	cb.failures = append(cb.failures, now)

	// Remove old failures outside the window
	cutoff := now.Add(-cb.config.FailureWindow)
	newFailures := cb.failures[:0]
	for _, t := range cb.failures {
		if t.After(cutoff) {
			newFailures = append(newFailures, t)
		}
	}
	cb.failures = newFailures

	switch cb.state {
	case CircuitClosed:
		if len(cb.failures) >= cb.config.FailureThreshold {
			cb.transitionTo(CircuitOpen)
		}
	case CircuitHalfOpen:
		// Any failure in half-open goes back to open
		cb.transitionTo(CircuitOpen)
	}
}

// transitionTo changes the circuit state
func (cb *CircuitBreaker) transitionTo(newState CircuitState) {
	if cb.state == newState {
		return
	}

	oldState := cb.state
	cb.state = newState
	cb.lastStateChange = time.Now()

	// Reset counters on state change
	cb.successes = 0

	if cb.config.EnableLog {
		log.Printf("[circuit/%s] State changed: %s -> %s", cb.name, oldState, newState)
	}
}

// State returns the current circuit state
func (cb *CircuitBreaker) State() CircuitState {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return cb.state
}

// Reset forces the circuit breaker to closed state
func (cb *CircuitBreaker) Reset() {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	cb.state = CircuitClosed
	cb.failures = cb.failures[:0]
	cb.successes = 0
	cb.lastStateChange = time.Now()
}

