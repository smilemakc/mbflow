package models

import "time"

// ExecutionStatus represents the lifecycle state of an execution.
type ExecutionStatus string

const (
	ExecutionStatusPending   ExecutionStatus = "pending"
	ExecutionStatusRunning   ExecutionStatus = "running"
	ExecutionStatusCompleted ExecutionStatus = "completed"
	ExecutionStatusFailed    ExecutionStatus = "failed"
	ExecutionStatusCancelled ExecutionStatus = "cancelled"
	ExecutionStatusTimeout   ExecutionStatus = "timeout"
)

// NodeExecutionStatus represents the lifecycle state of a node execution.
type NodeExecutionStatus string

const (
	NodeExecutionStatusPending   NodeExecutionStatus = "pending"
	NodeExecutionStatusRunning   NodeExecutionStatus = "running"
	NodeExecutionStatusCompleted NodeExecutionStatus = "completed"
	NodeExecutionStatusFailed    NodeExecutionStatus = "failed"
	NodeExecutionStatusSkipped   NodeExecutionStatus = "skipped"
	NodeExecutionStatusCancelled NodeExecutionStatus = "cancelled"
)

// Execution represents a single workflow execution instance.
type Execution struct {
	ID             string           `json:"id"`
	WorkflowID     string           `json:"workflow_id"`
	WorkflowName   string           `json:"workflow_name,omitempty"`
	Status         ExecutionStatus  `json:"status"`
	Input          map[string]any   `json:"input,omitempty"`
	Output         map[string]any   `json:"output,omitempty"`
	Error          string           `json:"error,omitempty"`
	NodeExecutions []*NodeExecution `json:"node_executions,omitempty"`
	Variables      map[string]any   `json:"variables,omitempty"`
	StrictMode     bool             `json:"strict_mode,omitempty"`
	StartedAt      time.Time        `json:"started_at"`
	CompletedAt    *time.Time       `json:"completed_at,omitempty"`
	Duration       int64            `json:"duration,omitempty"`
	TriggeredBy    string           `json:"triggered_by,omitempty"`
	Metadata       map[string]any   `json:"metadata,omitempty"`
}

// NodeExecution represents the execution of a single node within a workflow execution.
type NodeExecution struct {
	ID             string              `json:"id"`
	ExecutionID    string              `json:"execution_id"`
	NodeID         string              `json:"node_id"`
	NodeName       string              `json:"node_name,omitempty"`
	NodeType       string              `json:"node_type,omitempty"`
	Status         NodeExecutionStatus `json:"status"`
	Input          map[string]any      `json:"input,omitempty"`
	Output         map[string]any      `json:"output,omitempty"`
	Config         map[string]any      `json:"config,omitempty"`
	ResolvedConfig map[string]any      `json:"resolved_config,omitempty"`
	Error          string              `json:"error,omitempty"`
	StartedAt      time.Time           `json:"started_at"`
	CompletedAt    *time.Time          `json:"completed_at,omitempty"`
	Duration       int64               `json:"duration,omitempty"`
	RetryCount     int                 `json:"retry_count,omitempty"`
	Metadata       map[string]any      `json:"metadata,omitempty"`
}
