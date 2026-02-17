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
	Output      any
	DurationMs  int64
	Message     string
	Timestamp   time.Time
	Input       map[string]any
	Variables   map[string]any

	// Loop-related fields
	LoopEdgeID    string `json:"-"`
	LoopIteration int    `json:"-"`
	LoopMaxIter   int    `json:"-"`

	// Sub-workflow related fields
	SubWorkflowTotal      int    `json:"sub_workflow_total,omitempty"`
	SubWorkflowCompleted  int    `json:"sub_workflow_completed,omitempty"`
	SubWorkflowFailed     int    `json:"sub_workflow_failed,omitempty"`
	SubWorkflowRunning    int    `json:"sub_workflow_running,omitempty"`
	SubWorkflowItemIndex  int    `json:"sub_workflow_item_index,omitempty"`
	SubWorkflowItemExecID string `json:"sub_workflow_item_exec_id,omitempty"`
}
