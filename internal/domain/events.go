package domain

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// EventType defines the type of domain event
type EventType string

const (
	// Execution lifecycle events
	EventTypeExecutionStarted   EventType = "execution.started"
	EventTypeExecutionCompleted EventType = "execution.completed"
	EventTypeExecutionFailed    EventType = "execution.failed"
	EventTypeExecutionPaused    EventType = "execution.paused"
	EventTypeExecutionResumed   EventType = "execution.resumed"
	EventTypeExecutionCancelled EventType = "execution.cancelled"

	// Node lifecycle events
	EventTypeNodeStarted   EventType = "node.started"
	EventTypeNodeCompleted EventType = "node.completed"
	EventTypeNodeFailed    EventType = "node.failed"
	EventTypeNodeSkipped   EventType = "node.skipped"
	EventTypeNodeRetrying  EventType = "node.retrying"

	// Variable events
	EventTypeVariableSet     EventType = "variable.set"
	EventTypeVariableUpdated EventType = "variable.updated"
	EventTypeVariableDeleted EventType = "variable.deleted"

	// Edge evaluation events
	EventTypeEdgeEvaluated EventType = "edge.evaluated"
	EventTypeEdgeTraversed EventType = "edge.traversed"
	EventTypeEdgeSkipped   EventType = "edge.skipped"
)

// Event represents an immutable domain event in the event sourcing model.
// Events are the source of truth for the execution state and enable replay and audit.
type Event interface {
	// Identity
	EventID() uuid.UUID
	EventType() EventType
	AggregateID() uuid.UUID // The Execution ID
	Timestamp() time.Time
	SequenceNumber() int64

	// Context
	ExecutionID() uuid.UUID
	WorkflowID() uuid.UUID
	NodeID() uuid.UUID // May be uuid.Nil for execution-level events

	// Data
	Data() map[string]any
	Metadata() map[string]string

	// Serialization
	ToJSON() ([]byte, error)
}

// BaseEvent is the base implementation of Event
type BaseEvent struct {
	eventID        uuid.UUID
	eventType      EventType
	aggregateID    uuid.UUID // Execution ID
	timestamp      time.Time
	sequenceNumber int64
	executionID    uuid.UUID
	workflowID     uuid.UUID
	nodeID         uuid.UUID
	data           map[string]any
	metadata       map[string]string
}

// NewEvent creates a new base event
func NewEvent(
	eventType EventType,
	aggregateID uuid.UUID,
	sequenceNumber int64,
	workflowID uuid.UUID,
	nodeID uuid.UUID,
	data map[string]any,
	metadata map[string]string,
) Event {
	if data == nil {
		data = make(map[string]any)
	}
	if metadata == nil {
		metadata = make(map[string]string)
	}

	return &BaseEvent{
		eventID:        uuid.New(),
		eventType:      eventType,
		aggregateID:    aggregateID,
		timestamp:      time.Now(),
		sequenceNumber: sequenceNumber,
		executionID:    aggregateID, // aggregateID is the execution ID
		workflowID:     workflowID,
		nodeID:         nodeID,
		data:           data,
		metadata:       metadata,
	}
}

// ReconstructEvent reconstructs an event from persistence
func ReconstructEvent(
	eventID uuid.UUID,
	eventType EventType,
	aggregateID uuid.UUID,
	timestamp time.Time,
	sequenceNumber int64,
	workflowID uuid.UUID,
	nodeID uuid.UUID,
	data map[string]any,
	metadata map[string]string,
) Event {
	return &BaseEvent{
		eventID:        eventID,
		eventType:      eventType,
		aggregateID:    aggregateID,
		timestamp:      timestamp,
		sequenceNumber: sequenceNumber,
		workflowID:     workflowID,
		nodeID:         nodeID,
		data:           data,
		metadata:       metadata,
	}
}

func (e *BaseEvent) EventID() uuid.UUID {
	return e.eventID
}

func (e *BaseEvent) EventType() EventType {
	return e.eventType
}

func (e *BaseEvent) AggregateID() uuid.UUID {
	return e.aggregateID
}

func (e *BaseEvent) Timestamp() time.Time {
	return e.timestamp
}

func (e *BaseEvent) SequenceNumber() int64 {
	return e.sequenceNumber
}

func (e *BaseEvent) ExecutionID() uuid.UUID {
	return e.executionID
}

func (e *BaseEvent) WorkflowID() uuid.UUID {
	return e.workflowID
}

func (e *BaseEvent) NodeID() uuid.UUID {
	return e.nodeID
}

func (e *BaseEvent) Data() map[string]any {
	return e.data
}

func (e *BaseEvent) Metadata() map[string]string {
	return e.metadata
}

func (e *BaseEvent) ToJSON() ([]byte, error) {
	return json.Marshal(map[string]any{
		"event_id":        e.eventID,
		"event_type":      e.eventType,
		"aggregate_id":    e.aggregateID,
		"timestamp":       e.timestamp,
		"sequence_number": e.sequenceNumber,
		"workflow_id":     e.workflowID,
		"node_id":         e.nodeID,
		"data":            e.data,
		"metadata":        e.metadata,
	})
}

// Event Factory Functions - Create strongly-typed events

// NewExecutionStartedEvent creates an event for execution start
func NewExecutionStartedEvent(
	executionID, workflowID uuid.UUID,
	sequenceNumber int64,
	triggerID uuid.UUID,
	initialVariables map[string]any,
) Event {
	return NewEvent(
		EventTypeExecutionStarted,
		executionID,
		sequenceNumber,
		workflowID,
		uuid.Nil,
		map[string]any{
			"trigger_id":        triggerID,
			"initial_variables": initialVariables,
			"execution_phase":   ExecutionPhaseExecuting,
		},
		nil,
	)
}

// NewExecutionCompletedEvent creates an event for successful execution completion
func NewExecutionCompletedEvent(
	executionID, workflowID uuid.UUID,
	sequenceNumber int64,
	finalVariables map[string]any,
	duration time.Duration,
) Event {
	return NewEvent(
		EventTypeExecutionCompleted,
		executionID,
		sequenceNumber,
		workflowID,
		uuid.Nil,
		map[string]any{
			"final_variables": finalVariables,
			"duration_ms":     duration.Milliseconds(),
			"execution_phase": ExecutionPhaseCompleted,
		},
		nil,
	)
}

// NewExecutionFailedEvent creates an event for execution failure
func NewExecutionFailedEvent(
	executionID, workflowID uuid.UUID,
	sequenceNumber int64,
	errorMessage string,
	failedNodeID uuid.UUID,
) Event {
	return NewEvent(
		EventTypeExecutionFailed,
		executionID,
		sequenceNumber,
		workflowID,
		failedNodeID,
		map[string]any{
			"error":           errorMessage,
			"failed_node_id":  failedNodeID,
			"execution_phase": ExecutionPhaseFailed,
		},
		nil,
	)
}

// NewNodeStartedEvent creates an event for node execution start
func NewNodeStartedEvent(
	executionID, workflowID, nodeID uuid.UUID,
	sequenceNumber int64,
	nodeName string,
	nodeType NodeType,
	inputVariables map[string]any,
) Event {
	return NewEvent(
		EventTypeNodeStarted,
		executionID,
		sequenceNumber,
		workflowID,
		nodeID,
		map[string]any{
			"node_name":       nodeName,
			"node_type":       nodeType,
			"input_variables": inputVariables,
			"node_status":     NodeStatusRunning,
		},
		nil,
	)
}

// NewNodeCompletedEvent creates an event for successful node completion
func NewNodeCompletedEvent(
	executionID, workflowID, nodeID uuid.UUID,
	sequenceNumber int64,
	nodeName string,
	nodeType NodeType,
	output map[string]any,
	duration time.Duration,
) Event {
	return NewEvent(
		EventTypeNodeCompleted,
		executionID,
		sequenceNumber,
		workflowID,
		nodeID,
		map[string]any{
			"node_name":   nodeName,
			"node_type":   nodeType,
			"output":      output,
			"duration_ms": duration.Milliseconds(),
			"node_status": NodeStatusCompleted,
		},
		nil,
	)
}

// NewNodeFailedEvent creates an event for node execution failure
func NewNodeFailedEvent(
	executionID, workflowID, nodeID uuid.UUID,
	sequenceNumber int64,
	nodeName string,
	nodeType NodeType,
	errorMessage string,
	retryCount int,
) Event {
	return NewEvent(
		EventTypeNodeFailed,
		executionID,
		sequenceNumber,
		workflowID,
		nodeID,
		map[string]any{
			"node_name":   nodeName,
			"node_type":   nodeType,
			"error":       errorMessage,
			"retry_count": retryCount,
			"node_status": NodeStatusFailed,
		},
		nil,
	)
}

// NewNodeSkippedEvent creates an event for skipped node
func NewNodeSkippedEvent(
	executionID, workflowID, nodeID uuid.UUID,
	sequenceNumber int64,
	nodeName string,
	reason string,
) Event {
	return NewEvent(
		EventTypeNodeSkipped,
		executionID,
		sequenceNumber,
		workflowID,
		nodeID,
		map[string]any{
			"node_name":   nodeName,
			"reason":      reason,
			"node_status": NodeStatusSkipped,
		},
		nil,
	)
}

// NewVariableSetEvent creates an event for variable setting
func NewVariableSetEvent(
	executionID, workflowID, nodeID uuid.UUID,
	sequenceNumber int64,
	variableName string,
	value any,
	scope string,
) Event {
	return NewEvent(
		EventTypeVariableSet,
		executionID,
		sequenceNumber,
		workflowID,
		nodeID,
		map[string]any{
			"variable_name": variableName,
			"value":         value,
			"scope":         scope,
			"value_type":    InferType(value).String(),
		},
		nil,
	)
}

// NewEdgeEvaluatedEvent creates an event for edge condition evaluation
func NewEdgeEvaluatedEvent(
	executionID, workflowID uuid.UUID,
	sequenceNumber int64,
	edgeID, fromNodeID, toNodeID uuid.UUID,
	condition string,
	result bool,
) Event {
	return NewEvent(
		EventTypeEdgeEvaluated,
		executionID,
		sequenceNumber,
		workflowID,
		fromNodeID,
		map[string]any{
			"edge_id":      edgeID,
			"from_node_id": fromNodeID,
			"to_node_id":   toNodeID,
			"condition":    condition,
			"result":       result,
		},
		nil,
	)
}

// EventApplier is an interface for entities that can apply events to rebuild state
type EventApplier interface {
	ApplyEvent(event Event) error
}
