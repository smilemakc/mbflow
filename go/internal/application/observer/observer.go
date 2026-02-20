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
	Input     map[string]any // Input data (for node.started)
	Output    map[string]any // Output data (for node.completed)
	Variables map[string]any // Execution variables

	// Performance metrics
	DurationMs *int64 // Duration in milliseconds (for completed/failed events)
	RetryCount *int   // Retry count (future)

	// Additional metadata
	Metadata map[string]any // Additional context
	Message  *string        // Optional message (for skipped nodes, etc)
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
	EventTypeNodeSkipped        EventType = "node.skipped"
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

// ExecutionIDFilter filters events by execution ID
type ExecutionIDFilter struct {
	executionID string
}

// NewExecutionIDFilter creates a filter that only passes events for a specific execution
func NewExecutionIDFilter(executionID string) EventFilter {
	return &ExecutionIDFilter{executionID: executionID}
}

// ShouldNotify returns true if the event belongs to the target execution
func (f *ExecutionIDFilter) ShouldNotify(event Event) bool {
	return event.ExecutionID == f.executionID
}

// NodeIDFilter filters events by node IDs.
// Non-node events (execution.*, wave.*) always pass through.
type NodeIDFilter struct {
	allowedNodeIDs map[string]bool
}

// NewNodeIDFilter creates a filter for specific node IDs.
// Returns nil if no IDs provided (nil filter = all events).
func NewNodeIDFilter(nodeIDs ...string) EventFilter {
	if len(nodeIDs) == 0 {
		return nil
	}
	m := make(map[string]bool, len(nodeIDs))
	for _, id := range nodeIDs {
		m[id] = true
	}
	return &NodeIDFilter{allowedNodeIDs: m}
}

// ShouldNotify returns true for non-node events or events matching allowed node IDs
func (f *NodeIDFilter) ShouldNotify(event Event) bool {
	if event.NodeID == nil {
		return true // Non-node events always pass
	}
	return f.allowedNodeIDs[*event.NodeID]
}

// CompoundEventFilter combines multiple filters with AND logic.
// All sub-filters must pass for the event to be notified.
type CompoundEventFilter struct {
	filters []EventFilter
}

// NewCompoundEventFilter creates a filter that requires all sub-filters to pass.
// Nil filters are ignored. Returns nil if no valid filters remain.
func NewCompoundEventFilter(filters ...EventFilter) EventFilter {
	nonNil := make([]EventFilter, 0, len(filters))
	for _, f := range filters {
		if f != nil {
			nonNil = append(nonNil, f)
		}
	}
	if len(nonNil) == 0 {
		return nil
	}
	if len(nonNil) == 1 {
		return nonNil[0]
	}
	return &CompoundEventFilter{filters: nonNil}
}

// ShouldNotify returns true only if all sub-filters pass
func (f *CompoundEventFilter) ShouldNotify(event Event) bool {
	for _, filter := range f.filters {
		if !filter.ShouldNotify(event) {
			return false
		}
	}
	return true
}
