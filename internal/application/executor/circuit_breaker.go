package executor

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/smilemakc/mbflow/internal/domain"
)

// CircuitState represents the state of a circuit breaker
type CircuitState int

const (
	// StateClosed - circuit is closed, requests pass through normally
	StateClosed CircuitState = iota

	// StateOpen - circuit is open, requests fail immediately
	StateOpen

	// StateHalfOpen - circuit is testing if the service has recovered
	StateHalfOpen
)

// String returns string representation of circuit state
func (s CircuitState) String() string {
	switch s {
	case StateClosed:
		return "closed"
	case StateOpen:
		return "open"
	case StateHalfOpen:
		return "half-open"
	default:
		return "unknown"
	}
}

// CircuitBreakerConfig holds circuit breaker configuration
type CircuitBreakerConfig struct {
	// FailureThreshold is the number of consecutive failures before opening the circuit
	FailureThreshold int

	// SuccessThreshold is the number of consecutive successes in half-open state before closing
	SuccessThreshold int

	// Timeout is how long the circuit stays open before transitioning to half-open
	Timeout time.Duration

	// MaxConcurrentRequests in half-open state
	MaxConcurrentRequests int
}

// DefaultCircuitBreakerConfig returns default configuration
func DefaultCircuitBreakerConfig() CircuitBreakerConfig {
	return CircuitBreakerConfig{
		FailureThreshold:      5,
		SuccessThreshold:      2,
		Timeout:               60 * time.Second,
		MaxConcurrentRequests: 1,
	}
}

// CircuitBreaker implements the circuit breaker pattern
type CircuitBreaker struct {
	mu sync.RWMutex

	config CircuitBreakerConfig
	state  CircuitState

	// Counters
	consecutiveFailures  int
	consecutiveSuccesses int
	totalFailures        int
	totalSuccesses       int

	// Timing
	lastFailureTime time.Time
	lastStateChange time.Time
	openedAt        time.Time

	// Half-open concurrency control
	halfOpenRequests int
}

// NewCircuitBreaker creates a new circuit breaker
func NewCircuitBreaker(config CircuitBreakerConfig) *CircuitBreaker {
	return &CircuitBreaker{
		config:          config,
		state:           StateClosed,
		lastStateChange: time.Now(),
	}
}

// Execute executes a function with circuit breaker protection
func (cb *CircuitBreaker) Execute(ctx context.Context, fn func() error) error {
	// Check if circuit allows execution
	if err := cb.beforeRequest(); err != nil {
		return err
	}

	// Execute function
	err := fn()

	// Record result
	cb.afterRequest(err)

	return err
}

// beforeRequest checks if the circuit breaker allows the request
func (cb *CircuitBreaker) beforeRequest() error {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	switch cb.state {
	case StateClosed:
		// Allow request
		return nil

	case StateOpen:
		// Check if timeout has elapsed
		if time.Since(cb.openedAt) >= cb.config.Timeout {
			// Transition to half-open
			cb.setState(StateHalfOpen)
			cb.halfOpenRequests = 1
			return nil
		}

		// Circuit is still open
		return &CircuitBreakerOpenError{
			OpenedAt: cb.openedAt,
			Timeout:  cb.config.Timeout,
		}

	case StateHalfOpen:
		// Check if we can allow more requests
		if cb.halfOpenRequests >= cb.config.MaxConcurrentRequests {
			return &CircuitBreakerOpenError{
				OpenedAt: cb.openedAt,
				Timeout:  cb.config.Timeout,
			}
		}

		cb.halfOpenRequests++
		return nil

	default:
		return errors.New("unknown circuit breaker state")
	}
}

// afterRequest records the result of a request
func (cb *CircuitBreaker) afterRequest(err error) {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	if cb.state == StateHalfOpen {
		cb.halfOpenRequests--
	}

	if err != nil {
		cb.onFailure()
	} else {
		cb.onSuccess()
	}
}

// onFailure handles a failed request
func (cb *CircuitBreaker) onFailure() {
	cb.consecutiveFailures++
	cb.consecutiveSuccesses = 0
	cb.totalFailures++
	cb.lastFailureTime = time.Now()

	switch cb.state {
	case StateClosed:
		// Check if we should open the circuit
		if cb.consecutiveFailures >= cb.config.FailureThreshold {
			cb.setState(StateOpen)
			cb.openedAt = time.Now()
		}

	case StateHalfOpen:
		// Any failure in half-open state immediately opens the circuit
		cb.setState(StateOpen)
		cb.openedAt = time.Now()
	}
}

// onSuccess handles a successful request
func (cb *CircuitBreaker) onSuccess() {
	cb.consecutiveSuccesses++
	cb.consecutiveFailures = 0
	cb.totalSuccesses++

	if cb.state == StateHalfOpen {
		// Check if we should close the circuit
		if cb.consecutiveSuccesses >= cb.config.SuccessThreshold {
			cb.setState(StateClosed)
		}
	}
}

// setState changes the circuit state
func (cb *CircuitBreaker) setState(newState CircuitState) {
	if cb.state != newState {
		cb.state = newState
		cb.lastStateChange = time.Now()

		// Reset counters on state change
		if newState == StateClosed {
			cb.consecutiveFailures = 0
			cb.consecutiveSuccesses = 0
		}
	}
}

// State returns the current state
func (cb *CircuitBreaker) State() CircuitState {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return cb.state
}

// Stats returns circuit breaker statistics
func (cb *CircuitBreaker) Stats() map[string]interface{} {
	cb.mu.RLock()
	defer cb.mu.RUnlock()

	stats := map[string]interface{}{
		"state":                 cb.state.String(),
		"consecutive_failures":  cb.consecutiveFailures,
		"consecutive_successes": cb.consecutiveSuccesses,
		"total_failures":        cb.totalFailures,
		"total_successes":       cb.totalSuccesses,
		"last_state_change":     cb.lastStateChange.Format(time.RFC3339),
	}

	if cb.state == StateOpen {
		stats["opened_at"] = cb.openedAt.Format(time.RFC3339)
		stats["time_until_half_open"] = cb.config.Timeout - time.Since(cb.openedAt)
	}

	return stats
}

// Reset resets the circuit breaker to closed state
func (cb *CircuitBreaker) Reset() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.state = StateClosed
	cb.consecutiveFailures = 0
	cb.consecutiveSuccesses = 0
	cb.halfOpenRequests = 0
	cb.lastStateChange = time.Now()
}

// CircuitBreakerOpenError is returned when circuit breaker is open
type CircuitBreakerOpenError struct {
	OpenedAt time.Time
	Timeout  time.Duration
}

func (e *CircuitBreakerOpenError) Error() string {
	timeRemaining := e.Timeout - time.Since(e.OpenedAt)
	return fmt.Sprintf("circuit breaker is open, retry in %v", timeRemaining)
}

// CircuitBreakerExecutor wraps a node executor with circuit breaker protection
type CircuitBreakerExecutor struct {
	executor       NodeExecutor
	circuitBreaker *CircuitBreaker
}

// NewCircuitBreakerExecutor creates a new circuit breaker executor
func NewCircuitBreakerExecutor(executor NodeExecutor, config CircuitBreakerConfig) *CircuitBreakerExecutor {
	return &CircuitBreakerExecutor{
		executor:       executor,
		circuitBreaker: NewCircuitBreaker(config),
	}
}

// Execute executes the node with circuit breaker protection
func (cbe *CircuitBreakerExecutor) Execute(
	ctx context.Context,
	node domain.Node,
	inputs *NodeExecutionInputs,
) (map[string]any, error) {
	var output map[string]any
	var execErr error

	// Execute with circuit breaker
	err := cbe.circuitBreaker.Execute(ctx, func() error {
		output, execErr = cbe.executor.Execute(ctx, node, inputs)
		return execErr
	})

	if err != nil {
		// Circuit breaker prevented execution
		return nil, err
	}

	return output, execErr
}

// GetCircuitBreaker returns the underlying circuit breaker
func (cbe *CircuitBreakerExecutor) GetCircuitBreaker() *CircuitBreaker {
	return cbe.circuitBreaker
}

// CircuitBreakerRegistry manages circuit breakers for different services/nodes
type CircuitBreakerRegistry struct {
	mu       sync.RWMutex
	breakers map[string]*CircuitBreaker
	config   CircuitBreakerConfig
}

// NewCircuitBreakerRegistry creates a new registry
func NewCircuitBreakerRegistry(config CircuitBreakerConfig) *CircuitBreakerRegistry {
	return &CircuitBreakerRegistry{
		breakers: make(map[string]*CircuitBreaker),
		config:   config,
	}
}

// Get gets or creates a circuit breaker for a key
func (cbr *CircuitBreakerRegistry) Get(key string) *CircuitBreaker {
	cbr.mu.RLock()
	cb, exists := cbr.breakers[key]
	cbr.mu.RUnlock()

	if exists {
		return cb
	}

	cbr.mu.Lock()
	defer cbr.mu.Unlock()

	// Double-check after acquiring write lock
	cb, exists = cbr.breakers[key]
	if exists {
		return cb
	}

	// Create new circuit breaker
	cb = NewCircuitBreaker(cbr.config)
	cbr.breakers[key] = cb

	return cb
}

// Reset resets a specific circuit breaker
func (cbr *CircuitBreakerRegistry) Reset(key string) {
	cbr.mu.RLock()
	cb, exists := cbr.breakers[key]
	cbr.mu.RUnlock()

	if exists {
		cb.Reset()
	}
}

// ResetAll resets all circuit breakers
func (cbr *CircuitBreakerRegistry) ResetAll() {
	cbr.mu.RLock()
	defer cbr.mu.RUnlock()

	for _, cb := range cbr.breakers {
		cb.Reset()
	}
}

// GetStats returns statistics for all circuit breakers
func (cbr *CircuitBreakerRegistry) GetStats() map[string]map[string]interface{} {
	cbr.mu.RLock()
	defer cbr.mu.RUnlock()

	stats := make(map[string]map[string]interface{})
	for key, cb := range cbr.breakers {
		stats[key] = cb.Stats()
	}

	return stats
}
