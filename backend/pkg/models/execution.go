package models

import (
	"time"
)

// Execution represents a single workflow execution instance.
type Execution struct {
	ID             string                 `json:"id"`
	WorkflowID     string                 `json:"workflow_id"`
	WorkflowName   string                 `json:"workflow_name,omitempty"`
	Status         ExecutionStatus        `json:"status"`
	Input          map[string]interface{} `json:"input,omitempty"`
	Output         map[string]interface{} `json:"output,omitempty"`
	Error          string                 `json:"error,omitempty"`
	NodeExecutions []*NodeExecution       `json:"node_executions,omitempty"`
	Variables      map[string]interface{} `json:"variables,omitempty"`   // Runtime variables that override workflow variables
	StrictMode     bool                   `json:"strict_mode,omitempty"` // If true, missing template variables cause execution to fail
	StartedAt      time.Time              `json:"started_at"`
	CompletedAt    *time.Time             `json:"completed_at,omitempty"`
	Duration       int64                  `json:"duration,omitempty"` // milliseconds
	TriggeredBy    string                 `json:"triggered_by,omitempty"`
	Metadata       map[string]interface{} `json:"metadata,omitempty"`
}

// ExecutionStatus represents the status of an execution.
type ExecutionStatus string

const (
	ExecutionStatusPending   ExecutionStatus = "pending"
	ExecutionStatusRunning   ExecutionStatus = "running"
	ExecutionStatusCompleted ExecutionStatus = "completed"
	ExecutionStatusFailed    ExecutionStatus = "failed"
	ExecutionStatusCancelled ExecutionStatus = "cancelled"
	ExecutionStatusTimeout   ExecutionStatus = "timeout"
)

// NodeExecution represents the execution of a single node within a workflow execution.
type NodeExecution struct {
	ID             string                 `json:"id"`
	ExecutionID    string                 `json:"execution_id"`
	NodeID         string                 `json:"node_id"`
	NodeName       string                 `json:"node_name,omitempty"`
	NodeType       string                 `json:"node_type,omitempty"`
	Status         NodeExecutionStatus    `json:"status"`
	Input          map[string]interface{} `json:"input,omitempty"`           // Input data passed to the node executor
	Output         map[string]interface{} `json:"output,omitempty"`          // Output data from node execution
	Config         map[string]interface{} `json:"config,omitempty"`          // Original node configuration (before template resolution)
	ResolvedConfig map[string]interface{} `json:"resolved_config,omitempty"` // Configuration after template resolution (final config used by executor)
	Error          string                 `json:"error,omitempty"`
	StartedAt      time.Time              `json:"started_at"`
	CompletedAt    *time.Time             `json:"completed_at,omitempty"`
	Duration       int64                  `json:"duration,omitempty"` // milliseconds
	RetryCount     int                    `json:"retry_count,omitempty"`
	Metadata       map[string]interface{} `json:"metadata,omitempty"`
}

// NodeExecutionStatus represents the status of a node execution.
type NodeExecutionStatus string

const (
	NodeExecutionStatusPending   NodeExecutionStatus = "pending"
	NodeExecutionStatusRunning   NodeExecutionStatus = "running"
	NodeExecutionStatusCompleted NodeExecutionStatus = "completed"
	NodeExecutionStatusFailed    NodeExecutionStatus = "failed"
	NodeExecutionStatusSkipped   NodeExecutionStatus = "skipped"
	NodeExecutionStatusCancelled NodeExecutionStatus = "cancelled"
)

// IsTerminal returns true if the execution status is terminal (completed, failed, cancelled, timeout).
func (s ExecutionStatus) IsTerminal() bool {
	return s == ExecutionStatusCompleted ||
		s == ExecutionStatusFailed ||
		s == ExecutionStatusCancelled ||
		s == ExecutionStatusTimeout
}

// IsTerminal returns true if the node execution status is terminal.
func (s NodeExecutionStatus) IsTerminal() bool {
	return s == NodeExecutionStatusCompleted ||
		s == NodeExecutionStatusFailed ||
		s == NodeExecutionStatusSkipped ||
		s == NodeExecutionStatusCancelled
}

// GetNodeExecution returns a node execution by node ID.
func (e *Execution) GetNodeExecution(nodeID string) (*NodeExecution, error) {
	for _, ne := range e.NodeExecutions {
		if ne.NodeID == nodeID {
			return ne, nil
		}
	}
	return nil, ErrNodeNotFound
}

// CalculateDuration calculates the execution duration in milliseconds.
func (e *Execution) CalculateDuration() int64 {
	if e.CompletedAt == nil {
		return time.Since(e.StartedAt).Milliseconds()
	}
	return e.CompletedAt.Sub(e.StartedAt).Milliseconds()
}

// CalculateDuration calculates the node execution duration in milliseconds.
func (ne *NodeExecution) CalculateDuration() int64 {
	if ne.CompletedAt == nil {
		return time.Since(ne.StartedAt).Milliseconds()
	}
	return ne.CompletedAt.Sub(ne.StartedAt).Milliseconds()
}

// GetSuccessRate returns the success rate of node executions as a percentage.
func (e *Execution) GetSuccessRate() float64 {
	if len(e.NodeExecutions) == 0 {
		return 0
	}

	completed := 0
	for _, ne := range e.NodeExecutions {
		if ne.Status == NodeExecutionStatusCompleted {
			completed++
		}
	}

	return float64(completed) / float64(len(e.NodeExecutions)) * 100
}

// GetFailedNodes returns a list of failed node executions.
func (e *Execution) GetFailedNodes() []*NodeExecution {
	var failed []*NodeExecution
	for _, ne := range e.NodeExecutions {
		if ne.Status == NodeExecutionStatusFailed {
			failed = append(failed, ne)
		}
	}
	return failed
}
