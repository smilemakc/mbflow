package engine

import "time"

// ExecutionEvent represents a lifecycle event during workflow execution.
// Used by ExecutionNotifier implementations to track execution progress.
type ExecutionEvent struct {
	Type        string
	ExecutionID string
	WorkflowID  string
	NodeID      string
	NodeName    string
	NodeType    string
	WaveIndex   int
	NodeCount   int
	Status      string
	Error       error
	Output      interface{}
	DurationMs  int64
	Message     string
	Timestamp   time.Time
	Input       map[string]interface{}
	Variables   map[string]interface{}
}
