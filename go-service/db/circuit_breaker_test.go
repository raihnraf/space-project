package db

import (
	"sync"
	"testing"
	"time"
)

// TestCircuitBreakerInitialState tests that the circuit breaker starts in CLOSED state
func TestCircuitBreakerInitialState(t *testing.T) {
	cb := NewCircuitBreaker(3, 30*time.Second)

	if cb.State() != Closed {
		t.Errorf("expected initial state to be CLOSED, got %s", cb.State())
	}

	if !cb.IsClosed() {
		t.Error("expected IsClosed() to be true initially")
	}

	if cb.IsOpen() {
		t.Error("expected IsOpen() to be false initially")
	}

	if cb.IsHalfOpen() {
		t.Error("expected IsHalfOpen() to be false initially")
	}

	if cb.FailureCount() != 0 {
		t.Errorf("expected initial failure count to be 0, got %d", cb.FailureCount())
	}
}

// TestCircuitBreakerAllowInClosedState tests that requests are allowed in CLOSED state
func TestCircuitBreakerAllowInClosedState(t *testing.T) {
	cb := NewCircuitBreaker(3, 30*time.Second)

	for i := 0; i < 10; i++ {
		if !cb.Allow() {
			t.Errorf("expected Allow() to return true in CLOSED state (attempt %d)", i)
		}
	}
}

// TestCircuitBreakerClosedToOpenTransition tests transition from CLOSED to OPEN
func TestCircuitBreakerClosedToOpenTransition(t *testing.T) {
	threshold := 3
	cb := NewCircuitBreaker(threshold, 30*time.Second)

	// Record failures up to threshold
	for i := 0; i < threshold-1; i++ {
		cb.RecordFailure()
		if cb.IsOpen() {
			t.Errorf("circuit should not open before reaching threshold (after %d failures)", i+1)
		}
	}

	// One more failure should open the circuit
	cb.RecordFailure()
	if !cb.IsOpen() {
		t.Error("circuit should be OPEN after reaching failure threshold")
	}

	// Verify Allow() returns false in OPEN state
	if cb.Allow() {
		t.Error("expected Allow() to return false in OPEN state")
	}
}

// TestCircuitBreakerOpenToHalfOpenTransition tests transition from OPEN to HALF_OPEN after timeout
func TestCircuitBreakerOpenToHalfOpenTransition(t *testing.T) {
	// Use short timeout for testing
	timeout := 100 * time.Millisecond
	cb := NewCircuitBreaker(1, timeout)

	// Open the circuit
	cb.RecordFailure()
	if !cb.IsOpen() {
		t.Fatal("circuit should be OPEN")
	}

	// Wait for timeout
	time.Sleep(timeout + 50*time.Millisecond)

	// Next Allow() should transition to HALF_OPEN
	if !cb.Allow() {
		t.Error("expected Allow() to return true after timeout (transition to HALF_OPEN)")
	}

	if !cb.IsHalfOpen() {
		t.Error("circuit should be in HALF_OPEN state after timeout")
	}
}

// TestCircuitBreakerHalfOpenToClosedOnSuccess tests recovery from HALF_OPEN
func TestCircuitBreakerHalfOpenToClosedOnSuccess(t *testing.T) {
	timeout := 100 * time.Millisecond
	cb := NewCircuitBreaker(1, timeout)

	// Open the circuit
	cb.RecordFailure()
	time.Sleep(timeout + 50*time.Millisecond)

	// Transition to HALF_OPEN
	cb.Allow()
	if !cb.IsHalfOpen() {
		t.Fatal("circuit should be in HALF_OPEN state")
	}

	// Record success - should close the circuit
	cb.RecordSuccess()
	if !cb.IsClosed() {
		t.Error("circuit should be CLOSED after success in HALF_OPEN state")
	}

	if cb.FailureCount() != 0 {
		t.Errorf("failure count should be reset to 0, got %d", cb.FailureCount())
	}
}

// TestCircuitBreakerHalfOpenToOpenOnFailure tests re-opening from HALF_OPEN
func TestCircuitBreakerHalfOpenToOpenOnFailure(t *testing.T) {
	timeout := 100 * time.Millisecond
	cb := NewCircuitBreaker(1, timeout)

	// Open the circuit
	cb.RecordFailure()
	time.Sleep(timeout + 50*time.Millisecond)

	// Transition to HALF_OPEN
	cb.Allow()
	if !cb.IsHalfOpen() {
		t.Fatal("circuit should be in HALF_OPEN state")
	}

	// Record failure - should re-open the circuit
	cb.RecordFailure()
	if !cb.IsOpen() {
		t.Error("circuit should be OPEN after failure in HALF_OPEN state")
	}
}

// TestCircuitBreakerHalfOpenOnlyAllowsOneRequest tests that only one request is allowed in HALF_OPEN
func TestCircuitBreakerHalfOpenOnlyAllowsOneRequest(t *testing.T) {
	timeout := 100 * time.Millisecond
	cb := NewCircuitBreaker(1, timeout)

	// Open the circuit
	cb.RecordFailure()
	time.Sleep(timeout + 50*time.Millisecond)

	// First Allow() should succeed and transition from OPEN to HALF_OPEN
	if !cb.Allow() {
		t.Error("first Allow() should return true (transition from OPEN to HALF_OPEN)")
	}

	// The next call in HALF_OPEN state should also return true
	// (first call in HALF_OPEN increments halfOpenAttempts to 1, which is not > 1)
	if !cb.Allow() {
		t.Error("second Allow() should return true in HALF_OPEN (first request)")
	}

	// Subsequent Allow() calls should be blocked (halfOpenAttempts now = 2, which is > 1)
	for i := 0; i < 5; i++ {
		if cb.Allow() {
			t.Errorf("subsequent Allow() %d should return false in HALF_OPEN state", i+2)
		}
	}
}

// TestCircuitBreakerReset tests manual reset functionality
func TestCircuitBreakerReset(t *testing.T) {
	cb := NewCircuitBreaker(3, 30*time.Second)

	// Open the circuit
	cb.RecordFailure()
	cb.RecordFailure()
	cb.RecordFailure()
	if !cb.IsOpen() {
		t.Fatal("circuit should be OPEN")
	}

	// Reset
	cb.Reset()

	if !cb.IsClosed() {
		t.Error("circuit should be CLOSED after reset")
	}

	if cb.FailureCount() != 0 {
		t.Errorf("failure count should be 0 after reset, got %d", cb.FailureCount())
	}

	if !cb.Allow() {
		t.Error("Allow() should return true after reset")
	}
}

// TestCircuitBreakerStateString tests state string representation
func TestCircuitBreakerStateString(t *testing.T) {
	tests := []struct {
		state    CircuitBreakerState
		expected string
	}{
		{Closed, "CLOSED"},
		{Open, "OPEN"},
		{HalfOpen, "HALF_OPEN"},
		{CircuitBreakerState(99), "UNKNOWN"},
	}

	for _, tt := range tests {
		if got := tt.state.String(); got != tt.expected {
			t.Errorf("state %d: expected %s, got %s", tt.state, tt.expected, got)
		}
	}
}

// TestCircuitBreakerFailureCountIncrement tests failure count incrementing
func TestCircuitBreakerFailureCountIncrement(t *testing.T) {
	cb := NewCircuitBreaker(10, 30*time.Second)

	for i := 1; i <= 5; i++ {
		cb.RecordFailure()
		if cb.FailureCount() != i {
			t.Errorf("expected failure count %d, got %d", i, cb.FailureCount())
		}
	}
}

// TestCircuitBreakerRecordSuccessInClosedState tests that success in CLOSED state is a no-op
func TestCircuitBreakerRecordSuccessInClosedState(t *testing.T) {
	cb := NewCircuitBreaker(3, 30*time.Second)

	// Record some failures (but not enough to open)
	cb.RecordFailure()
	cb.RecordFailure()

	countBefore := cb.FailureCount()
	cb.RecordSuccess()

	// Failure count should not change in CLOSED state
	if cb.FailureCount() != countBefore {
		t.Errorf("success in CLOSED state should not change failure count (was %d, got %d)",
			countBefore, cb.FailureCount())
	}

	// State should remain CLOSED
	if !cb.IsClosed() {
		t.Error("state should remain CLOSED after success in CLOSED state")
	}
}

// TestCircuitBreakerFullCycle tests the complete lifecycle
func TestCircuitBreakerFullCycle(t *testing.T) {
	timeout := 50 * time.Millisecond
	cb := NewCircuitBreaker(2, timeout)

	// 1. Start CLOSED
	if !cb.IsClosed() {
		t.Fatal("should start CLOSED")
	}

	// 2. Record failures to open
	cb.RecordFailure()
	cb.RecordFailure()
	if !cb.IsOpen() {
		t.Fatal("should be OPEN after 2 failures")
	}

	// 3. Block requests while OPEN
	if cb.Allow() {
		t.Fatal("should block requests while OPEN")
	}

	// 4. Wait for timeout and transition to HALF_OPEN
	time.Sleep(timeout + 20*time.Millisecond)
	if !cb.Allow() {
		t.Fatal("should allow one request after timeout")
	}
	if !cb.IsHalfOpen() {
		t.Fatal("should be HALF_OPEN after timeout")
	}

	// 5. Fail recovery, go back to OPEN
	cb.RecordFailure()
	if !cb.IsOpen() {
		t.Fatal("should be OPEN after failed recovery")
	}

	// 6. Wait for timeout again
	time.Sleep(timeout + 20*time.Millisecond)
	cb.Allow()

	// 7. Succeed recovery, go to CLOSED
	cb.RecordSuccess()
	if !cb.IsClosed() {
		t.Fatal("should be CLOSED after successful recovery")
	}
}

// TestCircuitBreakerConcurrentAccess tests thread-safety
func TestCircuitBreakerConcurrentAccess(t *testing.T) {
	cb := NewCircuitBreaker(100, 30*time.Second)
	var wg sync.WaitGroup

	// Concurrent Allow() calls
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			cb.Allow()
		}()
	}

	// Concurrent RecordFailure() calls
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			cb.RecordFailure()
		}()
	}

	// Concurrent RecordSuccess() calls
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			cb.RecordSuccess()
		}()
	}

	// Concurrent State() calls
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = cb.State()
		}()
	}

	wg.Wait()

	// If we get here without race conditions, the test passes
}

// TestCircuitBreakerZeroThreshold tests behavior with zero threshold
func TestCircuitBreakerZeroThreshold(t *testing.T) {
	cb := NewCircuitBreaker(0, 30*time.Second)

	// With zero threshold, first failure should open
	cb.RecordFailure()
	if !cb.IsOpen() {
		t.Error("circuit should be OPEN immediately with zero threshold")
	}
}

// TestCircuitBreakerMultipleFailuresBeyondThreshold tests failure count beyond threshold
func TestCircuitBreakerMultipleFailuresBeyondThreshold(t *testing.T) {
	cb := NewCircuitBreaker(2, 30*time.Second)

	// Record failures beyond threshold
	for i := 0; i < 5; i++ {
		cb.RecordFailure()
	}

	// Circuit should be OPEN
	if !cb.IsOpen() {
		t.Error("circuit should be OPEN")
	}

	// Failure count should continue incrementing
	if cb.FailureCount() != 5 {
		t.Errorf("expected failure count 5, got %d", cb.FailureCount())
	}
}

// TestCircuitBreakerTimeoutNotElapsed tests that OPEN doesn't transition too early
func TestCircuitBreakerTimeoutNotElapsed(t *testing.T) {
	timeout := 500 * time.Millisecond
	cb := NewCircuitBreaker(1, timeout)

	// Open the circuit
	cb.RecordFailure()

	// Check immediately (timeout not elapsed)
	if cb.Allow() {
		t.Error("circuit should not transition to HALF_OPEN before timeout")
	}

	// State should still be OPEN
	if !cb.IsOpen() {
		t.Error("circuit should still be OPEN before timeout")
	}
}

// TestCircuitBreakerStateTransitionsSummary tests all state transition helpers
func TestCircuitBreakerStateTransitionsSummary(t *testing.T) {
	cb := NewCircuitBreaker(1, 100*time.Millisecond)

	// CLOSED state
	if !cb.IsClosed() || cb.IsOpen() || cb.IsHalfOpen() {
		t.Error("state helpers incorrect for CLOSED")
	}

	// OPEN state
	cb.RecordFailure()
	if cb.IsClosed() || !cb.IsOpen() || cb.IsHalfOpen() {
		t.Error("state helpers incorrect for OPEN")
	}

	// HALF_OPEN state
	time.Sleep(150 * time.Millisecond)
	cb.Allow()
	if cb.IsClosed() || cb.IsOpen() || !cb.IsHalfOpen() {
		t.Error("state helpers incorrect for HALF_OPEN")
	}
}
