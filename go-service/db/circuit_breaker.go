package db

import (
	"log"
	"sync"
	"time"
)

// CircuitBreakerState represents the state of the circuit breaker
type CircuitBreakerState int

const (
	// Closed means requests are allowed through
	Closed CircuitBreakerState = iota
	// Open means requests are blocked (circuit is "tripped")
	Open
	// HalfOpen means one request is allowed through to test if the service has recovered
	HalfOpen
)

// String returns the string representation of the circuit breaker state
func (s CircuitBreakerState) String() string {
	switch s {
	case Closed:
		return "CLOSED"
	case Open:
		return "OPEN"
	case HalfOpen:
		return "HALF_OPEN"
	default:
		return "UNKNOWN"
	}
}

// CircuitBreaker implements the circuit breaker pattern
// It prevents cascading failures by blocking requests to a failing service
// after a threshold of consecutive failures is reached.
//
// States:
// - CLOSED: Requests pass through normally. Failures increment the counter.
// - OPEN: Requests are blocked. After timeout, transitions to HALF_OPEN.
// - HALF_OPEN: One request is allowed. Success closes circuit, failure reopens it.
type CircuitBreaker struct {
	mu                sync.Mutex
	state             CircuitBreakerState
	failureCount      int
	failureThreshold  int
	lastFailureTime   time.Time
	timeout           time.Duration
	halfOpenAttempts  int
}

// NewCircuitBreaker creates a new circuit breaker with the given threshold and timeout
// threshold: number of consecutive failures before opening the circuit
// timeout: how long to wait before transitioning from OPEN to HALF_OPEN
func NewCircuitBreaker(threshold int, timeout time.Duration) *CircuitBreaker {
	return &CircuitBreaker{
		state:            Closed,
		failureThreshold: threshold,
		timeout:          timeout,
	}
}

// Allow returns true if a request should be allowed through the circuit breaker
// It handles state transitions and implements the circuit breaker logic:
// - CLOSED: Always allow
// - OPEN: Allow only if timeout has passed (transition to HALF_OPEN)
// - HALF_OPEN: Allow one request to test recovery
func (cb *CircuitBreaker) Allow() bool {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	switch cb.state {
	case Closed:
		return true

	case Open:
		// Check if we should transition to HALF_OPEN
		if time.Since(cb.lastFailureTime) >= cb.timeout {
			log.Printf("CircuitBreaker: OPEN -> HALF_OPEN (timeout elapsed)")
			cb.state = HalfOpen
			cb.halfOpenAttempts = 0
			return true
		}
		return false

	case HalfOpen:
		// Allow one request through to test if service has recovered
		cb.halfOpenAttempts++
		if cb.halfOpenAttempts > 1 {
			// Only one request allowed in HALF_OPEN state
			return false
		}
		return true

	default:
		return false
	}
}

// RecordSuccess records a successful request
// It closes the circuit if we're in HALF_OPEN state (service recovered)
func (cb *CircuitBreaker) RecordSuccess() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	if cb.state == HalfOpen {
		log.Printf("CircuitBreaker: HALF_OPEN -> CLOSED (service recovered)")
		cb.state = Closed
		cb.failureCount = 0
		cb.halfOpenAttempts = 0
	}
}

// RecordFailure records a failed request
// It increments the failure counter and opens the circuit if threshold is reached
func (cb *CircuitBreaker) RecordFailure() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.failureCount++
	cb.lastFailureTime = time.Now()

	// Log the failure count
	log.Printf("CircuitBreaker: Failure recorded (count: %d/%d, state: %s)",
		cb.failureCount, cb.failureThreshold, cb.state)

	// Open the circuit if we've reached the threshold
	if cb.state == Closed && cb.failureCount >= cb.failureThreshold {
		log.Printf("CircuitBreaker: CLOSED -> OPEN (threshold reached)")
		cb.state = Open
	} else if cb.state == HalfOpen {
		// Service not recovered, go back to OPEN
		log.Printf("CircuitBreaker: HALF_OPEN -> OPEN (service still failing)")
		cb.state = Open
		cb.halfOpenAttempts = 0
	}
}

// Reset manually resets the circuit breaker to CLOSED state
// This can be used for testing or manual intervention
func (cb *CircuitBreaker) Reset() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	log.Printf("CircuitBreaker: Manually reset to CLOSED")
	cb.state = Closed
	cb.failureCount = 0
	cb.lastFailureTime = time.Time{}
	cb.halfOpenAttempts = 0
}

// State returns the current state of the circuit breaker
func (cb *CircuitBreaker) State() CircuitBreakerState {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	return cb.state
}

// FailureCount returns the current failure count
func (cb *CircuitBreaker) FailureCount() int {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	return cb.failureCount
}

// IsOpen returns true if the circuit breaker is in OPEN state
func (cb *CircuitBreaker) IsOpen() bool {
	return cb.State() == Open
}

// IsClosed returns true if the circuit breaker is in CLOSED state
func (cb *CircuitBreaker) IsClosed() bool {
	return cb.State() == Closed
}

// IsHalfOpen returns true if the circuit breaker is in HALF_OPEN state
func (cb *CircuitBreaker) IsHalfOpen() bool {
	return cb.State() == HalfOpen
}
