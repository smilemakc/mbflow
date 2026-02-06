package engine

import (
	"context"

	"github.com/smilemakc/mbflow/internal/application/observer"
	pkgengine "github.com/smilemakc/mbflow/pkg/engine"
)

// ObserverNotifier adapts ExecutionNotifier to observer.ObserverManager.
type ObserverNotifier struct {
	manager *observer.ObserverManager
}

// NewObserverNotifier creates a new ObserverNotifier.
func NewObserverNotifier(manager *observer.ObserverManager) *ObserverNotifier {
	return &ObserverNotifier{manager: manager}
}

// Notify converts an ExecutionEvent to an observer.Event and delegates.
func (n *ObserverNotifier) Notify(ctx context.Context, event pkgengine.ExecutionEvent) {
	if n.manager == nil {
		return
	}

	obsEvent := observer.Event{
		Type:        observer.EventType(event.Type),
		ExecutionID: event.ExecutionID,
		WorkflowID:  event.WorkflowID,
		Timestamp:   event.Timestamp,
		Status:      event.Status,
		Error:       event.Error,
		Output:      pkgengine.ToMapInterface(event.Output),
	}

	if event.NodeID != "" {
		obsEvent.NodeID = &event.NodeID
	}
	if event.NodeName != "" {
		obsEvent.NodeName = &event.NodeName
	}
	if event.NodeType != "" {
		obsEvent.NodeType = &event.NodeType
	}
	if event.WaveIndex > 0 || event.Type == pkgengine.EventTypeWaveStarted || event.Type == pkgengine.EventTypeWaveCompleted {
		obsEvent.WaveIndex = &event.WaveIndex
	}
	if event.NodeCount > 0 {
		obsEvent.NodeCount = &event.NodeCount
	}
	if event.DurationMs > 0 {
		obsEvent.DurationMs = &event.DurationMs
	}
	if event.Message != "" {
		obsEvent.Message = &event.Message
	}
	if event.Input != nil {
		obsEvent.Input = event.Input
	}
	if event.Variables != nil {
		obsEvent.Variables = event.Variables
	}

	n.manager.Notify(ctx, obsEvent)
}
