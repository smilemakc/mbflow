package engine

import (
	"context"
	"sync/atomic"

	"github.com/smilemakc/mbflow/go/internal/application/observer"
	pkgengine "github.com/smilemakc/mbflow/go/pkg/engine"
)

// EphemeralNotifier wraps ObserverManager with sequence numbering and redaction.
// Implements pkgengine.ExecutionNotifier.
type EphemeralNotifier struct {
	manager  *observer.ObserverManager
	redactor *EventRedactor
	seq      atomic.Int64
}

// NewEphemeralNotifier creates a new EphemeralNotifier.
func NewEphemeralNotifier(manager *observer.ObserverManager, redactor *EventRedactor) *EphemeralNotifier {
	return &EphemeralNotifier{
		manager:  manager,
		redactor: redactor,
	}
}

// Notify converts an ExecutionEvent, redacts sensitive data, stamps a monotonic sequence number, and dispatches.
func (n *EphemeralNotifier) Notify(ctx context.Context, event pkgengine.ExecutionEvent) {
	if n.manager == nil {
		return
	}

	seq := n.seq.Add(1)

	obsEvent := convertToObserverEvent(event)

	if n.redactor != nil && obsEvent.Variables != nil {
		obsEvent.Variables = n.redactor.RedactMap(obsEvent.Variables)
	}

	if obsEvent.Metadata == nil {
		obsEvent.Metadata = make(map[string]any)
	}
	obsEvent.Metadata["sequence"] = seq

	n.manager.Notify(ctx, obsEvent)
}
