package observer

import (
	"context"
	"fmt"
	"sync"

	"github.com/smilemakc/mbflow/go/internal/infrastructure/logger"
)

// ObserverManager manages multiple observers with non-blocking notifications
type ObserverManager struct {
	observers  []Observer
	logger     *logger.Logger
	mu         sync.RWMutex
	bufferSize int // Buffer size for async notification channel
}

// ManagerOption configures ObserverManager
type ManagerOption func(*ObserverManager)

// WithLogger sets the logger for the manager
func WithLogger(l *logger.Logger) ManagerOption {
	return func(m *ObserverManager) {
		m.logger = l
	}
}

// WithBufferSize sets the async notification buffer size
func WithBufferSize(size int) ManagerOption {
	return func(m *ObserverManager) {
		m.bufferSize = size
	}
}

// NewObserverManager creates a new observer manager
func NewObserverManager(opts ...ManagerOption) *ObserverManager {
	mgr := &ObserverManager{
		observers:  make([]Observer, 0),
		bufferSize: 100, // Default buffer size
	}

	for _, opt := range opts {
		opt(mgr)
	}

	return mgr
}

// Register adds an observer to the manager
func (m *ObserverManager) Register(observer Observer) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Check for duplicate names
	for _, obs := range m.observers {
		if obs.Name() == observer.Name() {
			return fmt.Errorf("observer with name %q already registered", observer.Name())
		}
	}

	m.observers = append(m.observers, observer)
	return nil
}

// Unregister removes an observer by name
func (m *ObserverManager) Unregister(name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	for i, obs := range m.observers {
		if obs.Name() == name {
			m.observers = append(m.observers[:i], m.observers[i+1:]...)
			return nil
		}
	}

	return fmt.Errorf("observer %q not found", name)
}

// Notify sends an event to all registered observers (NON-BLOCKING)
// Each observer runs in its own goroutine, errors are logged but don't propagate
// IMPORTANT: Uses context.WithoutCancel to ensure notifications complete even if caller context is canceled
func (m *ObserverManager) Notify(ctx context.Context, event Event) {
	m.mu.RLock()
	// Copy observers slice to avoid holding lock during notification
	observersCopy := make([]Observer, len(m.observers))
	copy(observersCopy, m.observers)
	m.mu.RUnlock()

	// Decouple observer notifications from caller's context lifecycle using WithoutCancel.
	// This ensures that even if workflow execution context is canceled, observers can complete their work
	// (e.g., saving execution.completed events to database).
	// WithoutCancel preserves all context values (trace IDs, request IDs, etc.) but removes cancelation.
	observerCtx := context.WithoutCancel(ctx)

	// Notify each observer in parallel (non-blocking)
	for _, obs := range observersCopy {
		go m.notifyObserver(observerCtx, obs, event)
	}
}

// notifyObserver notifies a single observer with error recovery
func (m *ObserverManager) notifyObserver(ctx context.Context, obs Observer, event Event) {
	// Recover from panics
	defer func() {
		if r := recover(); r != nil {
			if m.logger != nil {
				m.logger.ErrorContext(ctx, "Observer panic recovered",
					"observer", obs.Name(),
					"event_type", string(event.Type),
					"panic", r,
				)
			}
		}
	}()

	// Check filter
	filter := obs.Filter()
	if filter != nil && !filter.ShouldNotify(event) {
		return // Event filtered out
	}

	// Call observer
	if err := obs.OnEvent(ctx, event); err != nil {
		if m.logger != nil {
			m.logger.ErrorContext(ctx, "Observer notification failed",
				"observer", obs.Name(),
				"event_type", string(event.Type),
				"error", err,
			)
		}
	}
}

// Count returns the number of registered observers
func (m *ObserverManager) Count() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.observers)
}
