package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

// EventModel represents an immutable event in the event sourcing log
type EventModel struct {
	bun.BaseModel `bun:"table:mbflow_events,alias:ev"`

	ID          uuid.UUID `bun:"id,pk,type:uuid,default:uuid_generate_v4()" json:"id"`
	ExecutionID uuid.UUID `bun:"execution_id,notnull,type:uuid" json:"execution_id" validate:"required"`
	EventType   string    `bun:"event_type,notnull" json:"event_type" validate:"required,max=100"`
	Sequence    int64     `bun:"sequence,notnull,autoincrement" json:"sequence"`
	Payload     JSONBMap  `bun:"payload,type:jsonb,notnull,default:'{}'" json:"payload"`
	CreatedAt   time.Time `bun:"created_at,notnull,default:current_timestamp" json:"created_at"`

	// Relationships
	Execution *ExecutionModel `bun:"rel:belongs-to,join:execution_id=id" json:"execution,omitempty"`
}

// TableName returns the table name for EventModel
func (EventModel) TableName() string {
	return "mbflow_events"
}

// BeforeInsert hook to set timestamp
func (e *EventModel) BeforeInsert(ctx interface{}) error {
	e.CreatedAt = time.Now()
	if e.ID == uuid.Nil {
		e.ID = uuid.New()
	}
	if e.Payload == nil {
		e.Payload = make(JSONBMap)
	}
	return nil
}

// Common event types (dot notation)
const (
	EventTypeExecutionStarted   = "execution.started"
	EventTypeExecutionCompleted = "execution.completed"
	EventTypeExecutionFailed    = "execution.failed"
	EventTypeWorkflowCancelled  = "workflow.cancelled"
	EventTypeWorkflowPaused     = "workflow.paused"
	EventTypeWorkflowResumed    = "workflow.resumed"
	EventTypeNodeStarted        = "node.started"
	EventTypeNodeCompleted      = "node.completed"
	EventTypeNodeFailed         = "node.failed"
	EventTypeNodeSkipped        = "node.skipped"
	EventTypeNodeRetrying       = "node.retrying"
	EventTypeWaveStarted        = "wave.started"
	EventTypeWaveCompleted      = "wave.completed"
	EventTypeConditionEvaluated = "condition.evaluated"
	EventTypeVariableSet        = "variable.set"
	EventTypeErrorOccurred      = "error.occurred"
	EventTypeStateChanged       = "state.changed"
)

// IsWorkflowEvent returns true if event is a workflow-level event
func (e *EventModel) IsWorkflowEvent() bool {
	switch e.EventType {
	case EventTypeExecutionStarted, EventTypeExecutionCompleted, EventTypeExecutionFailed,
		EventTypeWorkflowCancelled, EventTypeWorkflowPaused, EventTypeWorkflowResumed:
		return true
	}
	return false
}

// IsNodeEvent returns true if event is a node-level event
func (e *EventModel) IsNodeEvent() bool {
	switch e.EventType {
	case EventTypeNodeStarted, EventTypeNodeCompleted, EventTypeNodeFailed,
		EventTypeNodeSkipped, EventTypeNodeRetrying:
		return true
	}
	return false
}
