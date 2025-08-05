package llmtracer

import (
	"sync"
	"time"
)

// CircuitBreakerState represents the current state of the circuit breaker
type CircuitBreakerState int

const (
	// StateClosed means the circuit is functioning normally
	StateClosed CircuitBreakerState = iota
	// StateOpen means the circuit is open and requests will fail fast
	StateOpen
	// StateHalfOpen means the circuit is testing if the backend has recovered
	StateHalfOpen
)

// CircuitBreaker implements the circuit breaker pattern for storage operations
type CircuitBreaker struct {
	maxFailures  int
	resetTimeout time.Duration

	mu           sync.Mutex
	state        CircuitBreakerState
	failures     int
	lastFailTime time.Time
	successCount int
}

// NewCircuitBreaker creates a new circuit breaker
func NewCircuitBreaker(maxFailures int, resetTimeout time.Duration) *CircuitBreaker {
	return &CircuitBreaker{
		maxFailures:  maxFailures,
		resetTimeout: resetTimeout,
		state:        StateClosed,
	}
}

// Call executes the given function if the circuit breaker allows it
func (cb *CircuitBreaker) Call(fn func() error) error {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	// Check if we should transition from Open to HalfOpen
	if cb.state == StateOpen {
		if time.Since(cb.lastFailTime) > cb.resetTimeout {
			cb.state = StateHalfOpen
			cb.successCount = 0
		} else {
			return ErrCircuitOpen
		}
	}

	// Execute the function
	err := fn()

	// Update state based on result
	if err != nil {
		cb.recordFailure()
	} else {
		cb.recordSuccess()
	}

	return err
}

// recordFailure records a failure and potentially opens the circuit
func (cb *CircuitBreaker) recordFailure() {
	cb.failures++
	cb.lastFailTime = time.Now()

	if cb.failures >= cb.maxFailures {
		cb.state = StateOpen
	}
}

// recordSuccess records a success and potentially closes the circuit
func (cb *CircuitBreaker) recordSuccess() {
	if cb.state == StateHalfOpen {
		cb.successCount++
		// After 2 successful calls in half-open state, close the circuit
		if cb.successCount >= 2 {
			cb.state = StateClosed
			cb.failures = 0
		}
	} else if cb.state == StateClosed {
		// Reset failure count on success in closed state
		cb.failures = 0
	}
}

// GetState returns the current state of the circuit breaker
func (cb *CircuitBreaker) GetState() CircuitBreakerState {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	// Check for state transition
	if cb.state == StateOpen && time.Since(cb.lastFailTime) > cb.resetTimeout {
		cb.state = StateHalfOpen
		cb.successCount = 0
	}

	return cb.state
}

// IsOpen returns true if the circuit breaker is open
func (cb *CircuitBreaker) IsOpen() bool {
	return cb.GetState() == StateOpen
}
