package engine

import (
	"context"
	"sync"
	"time"
)

// EphemeralStreamRegistry is an in-memory registry for ephemeral execution streams.
// It maps execution IDs to their EphemeralNotifiers, allowing StreamEvents to connect
// even when persist_execution=false (no DB row exists).
type EphemeralStreamRegistry struct {
	mu      sync.RWMutex
	streams map[string]*ephemeralStream
	ttl     time.Duration
}

type ephemeralStream struct {
	notifier   *EphemeralNotifier
	createdAt  time.Time
	terminalAt *time.Time
}

// NewEphemeralStreamRegistry creates a registry with the given TTL for cleanup
// after terminal events.
func NewEphemeralStreamRegistry(ttl time.Duration) *EphemeralStreamRegistry {
	return &EphemeralStreamRegistry{
		streams: make(map[string]*ephemeralStream),
		ttl:     ttl,
	}
}

// Register adds an ephemeral execution to the registry.
func (r *EphemeralStreamRegistry) Register(executionID string, notifier *EphemeralNotifier) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.streams[executionID] = &ephemeralStream{
		notifier:  notifier,
		createdAt: time.Now(),
	}
}

// Get returns the EphemeralNotifier for the given execution ID.
func (r *EphemeralStreamRegistry) Get(executionID string) (*EphemeralNotifier, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	s, ok := r.streams[executionID]
	if !ok {
		return nil, false
	}
	return s.notifier, true
}

// IsTerminal reports whether the execution is currently marked terminal.
func (r *EphemeralStreamRegistry) IsTerminal(executionID string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	s, ok := r.streams[executionID]
	if !ok {
		return false
	}
	return s.terminalAt != nil
}

// MarkTerminal marks an execution as having reached a terminal state.
// The entry stays in the registry for TTL duration to allow late subscribers.
func (r *EphemeralStreamRegistry) MarkTerminal(executionID string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if s, ok := r.streams[executionID]; ok {
		now := time.Now()
		s.terminalAt = &now
	}
}

// StartCleanup runs a background goroutine that periodically removes expired entries.
// It blocks until ctx is cancelled.
func (r *EphemeralStreamRegistry) StartCleanup(ctx context.Context) {
	ticker := time.NewTicker(r.ttl)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			r.Cleanup()
		}
	}
}

// Cleanup removes entries that have been terminal for longer than TTL.
func (r *EphemeralStreamRegistry) Cleanup() {
	r.mu.Lock()
	defer r.mu.Unlock()
	now := time.Now()
	for id, s := range r.streams {
		if s.terminalAt != nil && now.Sub(*s.terminalAt) > r.ttl {
			delete(r.streams, id)
		}
	}
}
