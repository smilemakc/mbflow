package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

// EventModel represents an immutable event in the event sourcing log
type EventModel struct {
	bun.BaseModel `bun:"table:events,alias:ev"`

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
	return "events"
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

// Common event types
const (
	EventTypeWorkflowStarted    = "workflow_started"
	EventTypeWorkflowCompleted  = "workflow_completed"
	EventTypeWorkflowFailed     = "workflow_failed"
	EventTypeWorkflowCancelled  = "workflow_cancelled"
	EventTypeWorkflowPaused     = "workflow_paused"
	EventTypeWorkflowResumed    = "workflow_resumed"
	EventTypeNodeStarted        = "node_started"
	EventTypeNodeCompleted      = "node_completed"
	EventTypeNodeFailed         = "node_failed"
	EventTypeNodeSkipped        = "node_skipped"
	EventTypeNodeRetrying       = "node_retrying"
	EventTypeWaveStarted        = "wave_started"
	EventTypeWaveCompleted      = "wave_completed"
	EventTypeConditionEvaluated = "condition_evaluated"
	EventTypeVariableSet        = "variable_set"
	EventTypeErrorOccurred      = "error_occurred"
	EventTypeStateChanged       = "state_changed"
)

// IsWorkflowEvent returns true if event is a workflow-level event
func (e *EventModel) IsWorkflowEvent() bool {
	switch e.EventType {
	case EventTypeWorkflowStarted, EventTypeWorkflowCompleted, EventTypeWorkflowFailed,
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
