package resilience

import (
	"context"
	"errors"
	"sync"
	"time"
)

// ErrCircuitOpen indicates the downstream dependency is temporarily blocked by the breaker.
var ErrCircuitOpen = errors.New("circuit breaker is open")

type circuitState int

const (
	stateClosed circuitState = iota
	stateOpen
	stateHalfOpen
)

// CircuitBreaker is a small in-process breaker for outbound dependency calls.
type CircuitBreaker struct {
	mu               sync.Mutex
	failureThreshold uint32
	openTimeout      time.Duration
	now              func() time.Time

	state              circuitState
	consecutiveFailure uint32
	openUntil          time.Time
	halfOpenInFlight   bool
}

// NewCircuitBreaker creates a new breaker with the given settings.
func NewCircuitBreaker(failureThreshold uint32, openTimeout time.Duration) *CircuitBreaker {
	if failureThreshold == 0 {
		failureThreshold = 5
	}
	if openTimeout <= 0 {
		openTimeout = 30 * time.Second
	}

	return &CircuitBreaker{
		failureThreshold: failureThreshold,
		openTimeout:      openTimeout,
		now:              time.Now,
	}
}

// Execute runs fn when the breaker allows it, updating breaker state from the result.
func (b *CircuitBreaker) Execute(ctx context.Context, fn func(context.Context) error) error {
	if err := b.beforeRequest(); err != nil {
		return err
	}

	err := fn(ctx)
	b.afterRequest(err)
	return err
}

func (b *CircuitBreaker) beforeRequest() error {
	b.mu.Lock()
	defer b.mu.Unlock()

	now := b.now()

	switch b.state {
	case stateOpen:
		if now.Before(b.openUntil) {
			return ErrCircuitOpen
		}
		b.state = stateHalfOpen
		b.halfOpenInFlight = true
		return nil
	case stateHalfOpen:
		if b.halfOpenInFlight {
			return ErrCircuitOpen
		}
		b.halfOpenInFlight = true
		return nil
	default:
		return nil
	}
}

func (b *CircuitBreaker) afterRequest(err error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if err == nil {
		b.state = stateClosed
		b.consecutiveFailure = 0
		b.halfOpenInFlight = false
		return
	}

	switch b.state {
	case stateHalfOpen:
		b.trip()
	case stateClosed:
		b.consecutiveFailure++
		if b.consecutiveFailure >= b.failureThreshold {
			b.trip()
		}
	case stateOpen:
		// no-op: requests should not run while open
	}

	b.halfOpenInFlight = false
}

func (b *CircuitBreaker) trip() {
	b.state = stateOpen
	b.openUntil = b.now().Add(b.openTimeout)
	b.consecutiveFailure = 0
	b.halfOpenInFlight = false
}
