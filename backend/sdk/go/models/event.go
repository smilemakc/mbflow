package models

import "time"

// Event represents an immutable record in the execution event log.
type Event struct {
	ID          string         `json:"id"`
	ExecutionID string         `json:"execution_id"`
	EventType   string         `json:"event_type"`
	Sequence    int64          `json:"sequence"`
	Payload     map[string]any `json:"payload,omitempty"`
	CreatedAt   time.Time      `json:"created_at"`
}

const (
	EventTypeExecutionStarted   = "execution.started"
	EventTypeExecutionCompleted = "execution.completed"
	EventTypeExecutionFailed    = "execution.failed"
	EventTypeExecutionCancelled = "execution.cancelled"
	EventTypeExecutionPaused    = "execution.paused"
	EventTypeExecutionResumed   = "execution.resumed"

	EventTypeNodeStarted   = "node.started"
	EventTypeNodeCompleted = "node.completed"
	EventTypeNodeFailed    = "node.failed"
	EventTypeNodeSkipped   = "node.skipped"
	EventTypeNodeRetrying  = "node.retrying"

	EventTypeWaveStarted   = "wave.started"
	EventTypeWaveCompleted = "wave.completed"

	EventTypeConditionEvaluated = "condition.evaluated"
	EventTypeVariableSet        = "variable.set"
	EventTypeErrorOccurred      = "error.occurred"
	EventTypeStateChanged       = "state.changed"
)
