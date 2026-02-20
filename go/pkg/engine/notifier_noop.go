package engine

import "context"

// NoOpNotifier is an ExecutionNotifier that does nothing.
// Used by standalone executor where no event notifications are needed.
type NoOpNotifier struct{}

// NewNoOpNotifier creates a new NoOpNotifier.
func NewNoOpNotifier() *NoOpNotifier {
	return &NoOpNotifier{}
}

// Notify does nothing.
func (n *NoOpNotifier) Notify(ctx context.Context, event ExecutionEvent) {}
