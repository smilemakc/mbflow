package observer

import (
	"context"
	"time"
)

// Observer is the core interface for workflow execution event observation
type Observer interface {
	// OnEvent is called when any execution event occurs
	OnEvent(ctx context.Context, event Event) error

	// Name returns the observer's unique identifier
	Name() string

	// Filter returns the event filter for this observer (nil = all events)
	Filter() EventFilter
}

// Event represents a workflow execution event with complete context
type Event struct {
	// Event metadata
	Type        EventType // Event type (execution.started, node.completed, etc)
	ExecutionID string    // Execution UUID
	WorkflowID  string    // Workflow UUID
	Timestamp   time.Time // Event timestamp

	// Context-specific fields (populated based on event type)
	NodeID    *string // Node ID (for node events)
	NodeName  *string // Node name
	NodeType  *string // Node type (http, llm, transform, etc)
	WaveIndex *int    // Wave index (for wave events)
	NodeCount *int    // Number of nodes in wave (for wave.started)

	// Status and results
	Status string // Current status (running, completed, failed)
	Error  error  // Error if any

	// Data payload (for detailed event data)
	Input     map[string]interface{} // Input data (for node.started)
	Output    map[string]interface{} // Output data (for node.completed)
	Variables map[string]interface{} // Execution variables

	// Performance metrics
	DurationMs *int64 // Duration in milliseconds (for completed/failed events)
	RetryCount *int   // Retry count (future)

	// Additional metadata
	Metadata map[string]interface{} // Additional context
}

// EventType represents the type of execution event (dot notation)
type EventType string

const (
	EventTypeExecutionStarted   EventType = "execution.started"
	EventTypeExecutionCompleted EventType = "execution.completed"
	EventTypeExecutionFailed    EventType = "execution.failed"
	EventTypeWaveStarted        EventType = "wave.started"
	EventTypeWaveCompleted      EventType = "wave.completed"
	EventTypeNodeStarted        EventType = "node.started"
	EventTypeNodeCompleted      EventType = "node.completed"
	EventTypeNodeFailed         EventType = "node.failed"
	EventTypeNodeRetrying       EventType = "node.retrying"
)

// EventFilter defines filtering criteria for events
type EventFilter interface {
	ShouldNotify(event Event) bool
}

// EventTypeFilter filters events by type
type EventTypeFilter struct {
	allowedTypes map[EventType]bool
}

// NewEventTypeFilter creates a filter for specific event types
// If no types specified, allows all events
func NewEventTypeFilter(types ...EventType) EventFilter {
	if len(types) == 0 {
		return nil // nil filter = all events
	}

	filter := &EventTypeFilter{
		allowedTypes: make(map[EventType]bool),
	}
	for _, t := range types {
		filter.allowedTypes[t] = true
	}
	return filter
}

// ShouldNotify checks if the event should trigger notification
func (f *EventTypeFilter) ShouldNotify(event Event) bool {
	if f == nil || len(f.allowedTypes) == 0 {
		return true // No filter = all events
	}
	return f.allowedTypes[event.Type]
}
