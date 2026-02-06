// Package models defines the public domain models for MBFlow.
package models

import (
	"time"
)

// Event represents an immutable event in the execution event log.
// Events track workflow and node execution progress for observability and replay.
type Event struct {
	ID          string                 `json:"id"`
	ExecutionID string                 `json:"execution_id"`
	EventType   string                 `json:"event_type"`
	Sequence    int64                  `json:"sequence"`
	Payload     map[string]interface{} `json:"payload,omitempty"`
	CreatedAt   time.Time              `json:"created_at"`
}

// Event type constants (dot notation for hierarchical categorization).
const (
	// Execution-level events
	EventTypeExecutionStarted   = "execution.started"
	EventTypeExecutionCompleted = "execution.completed"
	EventTypeExecutionFailed    = "execution.failed"
	EventTypeExecutionCancelled = "execution.cancelled"
	EventTypeExecutionPaused    = "execution.paused"
	EventTypeExecutionResumed   = "execution.resumed"

	// Node-level events
	EventTypeNodeStarted   = "node.started"
	EventTypeNodeCompleted = "node.completed"
	EventTypeNodeFailed    = "node.failed"
	EventTypeNodeSkipped   = "node.skipped"
	EventTypeNodeRetrying  = "node.retrying"

	// Wave-level events (parallel execution batches)
	EventTypeWaveStarted   = "wave.started"
	EventTypeWaveCompleted = "wave.completed"

	// Other events
	EventTypeConditionEvaluated = "condition.evaluated"
	EventTypeVariableSet        = "variable.set"
	EventTypeErrorOccurred      = "error.occurred"
	EventTypeStateChanged       = "state.changed"
)

// IsExecutionEvent returns true if the event is an execution-level event.
func (e *Event) IsExecutionEvent() bool {
	switch e.EventType {
	case EventTypeExecutionStarted, EventTypeExecutionCompleted, EventTypeExecutionFailed,
		EventTypeExecutionCancelled, EventTypeExecutionPaused, EventTypeExecutionResumed:
		return true
	}
	return false
}

// IsNodeEvent returns true if the event is a node-level event.
func (e *Event) IsNodeEvent() bool {
	switch e.EventType {
	case EventTypeNodeStarted, EventTypeNodeCompleted, EventTypeNodeFailed,
		EventTypeNodeSkipped, EventTypeNodeRetrying:
		return true
	}
	return false
}

// IsWaveEvent returns true if the event is a wave-level event.
func (e *Event) IsWaveEvent() bool {
	switch e.EventType {
	case EventTypeWaveStarted, EventTypeWaveCompleted:
		return true
	}
	return false
}

// Validate validates the event structure.
func (e *Event) Validate() error {
	if e.ExecutionID == "" {
		return &ValidationError{Field: "execution_id", Message: "execution ID is required"}
	}

	if e.EventType == "" {
		return &ValidationError{Field: "event_type", Message: "event type is required"}
	}

	return nil
}

// GetNodeID extracts the node ID from the event payload if present.
func (e *Event) GetNodeID() string {
	if e.Payload == nil {
		return ""
	}
	if nodeID, ok := e.Payload["node_id"].(string); ok {
		return nodeID
	}
	return ""
}

// GetNodeName extracts the node name from the event payload if present.
func (e *Event) GetNodeName() string {
	if e.Payload == nil {
		return ""
	}
	if nodeName, ok := e.Payload["node_name"].(string); ok {
		return nodeName
	}
	return ""
}

// GetError extracts the error message from the event payload if present.
func (e *Event) GetError() string {
	if e.Payload == nil {
		return ""
	}
	if err, ok := e.Payload["error"].(string); ok {
		return err
	}
	return ""
}
