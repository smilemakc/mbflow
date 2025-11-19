package domain

import (
	"time"
)

// Event represents a significant occurrence within the system.
type Event struct {
	eventID      string
	eventType    string
	workflowID   string
	executionID  string
	workflowName string
	nodeID       string
	timestamp    time.Time
	payload      []byte
	metadata     map[string]string
}

// NewEvent creates a new Event instance.
func NewEvent(eventID, eventType, workflowID, executionID, workflowName, nodeID string, payload []byte, metadata map[string]string) *Event {
	return &Event{
		eventID:      eventID,
		eventType:    eventType,
		workflowID:   workflowID,
		executionID:  executionID,
		workflowName: workflowName,
		nodeID:       nodeID,
		timestamp:    time.Now(),
		payload:      payload,
		metadata:     metadata,
	}
}

// ReconstructEvent reconstructs an Event from persistence.
func ReconstructEvent(eventID, eventType, workflowID, executionID, workflowName, nodeID string, timestamp time.Time, payload []byte, metadata map[string]string) *Event {
	return &Event{
		eventID:      eventID,
		eventType:    eventType,
		workflowID:   workflowID,
		executionID:  executionID,
		workflowName: workflowName,
		nodeID:       nodeID,
		timestamp:    timestamp,
		payload:      payload,
		metadata:     metadata,
	}
}

// EventID returns the unique identifier of the event.
func (e *Event) EventID() string {
	return e.eventID
}

// EventType returns the type of the event.
func (e *Event) EventType() string {
	return e.eventType
}

// WorkflowID returns the ID of the associated workflow.
func (e *Event) WorkflowID() string {
	return e.workflowID
}

// ExecutionID returns the ID of the associated execution.
func (e *Event) ExecutionID() string {
	return e.executionID
}

// WorkflowName returns the name of the associated workflow.
func (e *Event) WorkflowName() string {
	return e.workflowName
}

// NodeID returns the ID of the associated node, if any.
func (e *Event) NodeID() string {
	return e.nodeID
}

// Timestamp returns when the event occurred.
func (e *Event) Timestamp() time.Time {
	return e.timestamp
}

// Payload returns the event data.
func (e *Event) Payload() []byte {
	return e.payload
}

// Metadata returns additional metadata associated with the event.
func (e *Event) Metadata() map[string]string {
	return e.metadata
}
