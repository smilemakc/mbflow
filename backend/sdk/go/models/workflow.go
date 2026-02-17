package models

import "time"

// WorkflowStatus represents the lifecycle state of a workflow.
type WorkflowStatus string

const (
	WorkflowStatusDraft    WorkflowStatus = "draft"
	WorkflowStatusActive   WorkflowStatus = "active"
	WorkflowStatusInactive WorkflowStatus = "inactive"
	WorkflowStatusArchived WorkflowStatus = "archived"
)

// Workflow represents a complete workflow definition with its DAG structure.
type Workflow struct {
	ID          string         `json:"id"`
	Name        string         `json:"name"`
	Description string         `json:"description,omitempty"`
	Version     int            `json:"version"`
	Status      WorkflowStatus `json:"status"`
	Tags        []string       `json:"tags,omitempty"`
	Nodes       []*Node        `json:"nodes"`
	Edges       []*Edge        `json:"edges"`
	Variables   map[string]any `json:"variables,omitempty"`
	Metadata    map[string]any `json:"metadata,omitempty"`
	CreatedBy   string         `json:"created_by,omitempty"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
}

// Node represents a single node in the workflow DAG.
type Node struct {
	ID          string         `json:"id"`
	Name        string         `json:"name"`
	Type        string         `json:"type"`
	Description string         `json:"description,omitempty"`
	Config      map[string]any `json:"config"`
	Position    *Position      `json:"position,omitempty"`
	Metadata    map[string]any `json:"metadata,omitempty"`
}

// Position represents the visual position of a node in the editor.
type Position struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

// Edge represents a directed connection between two nodes in the DAG.
type Edge struct {
	ID           string         `json:"id"`
	From         string         `json:"from"`
	To           string         `json:"to"`
	SourceHandle string         `json:"source_handle,omitempty"`
	Condition    string         `json:"condition,omitempty"`
	Loop         *LoopConfig    `json:"loop,omitempty"`
	Metadata     map[string]any `json:"metadata,omitempty"`
}

// LoopConfig configures a loop edge that allows controlled re-execution of a wave range.
type LoopConfig struct {
	MaxIterations int `json:"max_iterations"`
}
