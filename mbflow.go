package mbflow

import (
	"context"
	"time"
)

// Workflow represents a workflow process.
type Workflow interface {
	ID() string
	Name() string
	Version() string
	Spec() []byte
	CreatedAt() time.Time
}

// Execution represents a workflow execution instance.
type Execution interface {
	ID() string
	WorkflowID() string
	Status() string
	StartedAt() time.Time
	FinishedAt() *time.Time
}

// Node represents a step in a workflow.
type Node interface {
	ID() string
	WorkflowID() string
	Type() string
	Name() string
	Config() map[string]any
}

// Edge represents a connection between nodes.
type Edge interface {
	ID() string
	WorkflowID() string
	FromNodeID() string
	ToNodeID() string
	Type() string
	Config() map[string]any
}

// Trigger represents a trigger for starting a workflow.
type Trigger interface {
	ID() string
	WorkflowID() string
	Type() string
	Config() map[string]any
}

// Event represents a system event.
type Event interface {
	EventID() string
	EventType() string
	WorkflowID() string
	ExecutionID() string
	WorkflowName() string
	NodeID() string
	Timestamp() time.Time
	Payload() []byte
	Metadata() map[string]string
}

// WorkflowRepository defines the interface for workflow operations.
type WorkflowRepository interface {
	SaveWorkflow(ctx context.Context, w Workflow) error
	GetWorkflow(ctx context.Context, id string) (Workflow, error)
	ListWorkflows(ctx context.Context) ([]Workflow, error)
}

// ExecutionRepository defines the interface for execution operations.
type ExecutionRepository interface {
	SaveExecution(ctx context.Context, e Execution) error
	GetExecution(ctx context.Context, id string) (Execution, error)
	ListExecutions(ctx context.Context) ([]Execution, error)
}

// EventRepository defines the interface for event operations.
type EventRepository interface {
	AppendEvent(ctx context.Context, e Event) error
	ListEventsByExecution(ctx context.Context, executionID string) ([]Event, error)
}

// NodeRepository defines the interface for node operations.
type NodeRepository interface {
	SaveNode(ctx context.Context, n Node) error
	GetNode(ctx context.Context, id string) (Node, error)
	ListNodes(ctx context.Context, workflowID string) ([]Node, error)
}

// EdgeRepository defines the interface for edge operations.
type EdgeRepository interface {
	SaveEdge(ctx context.Context, e Edge) error
	GetEdge(ctx context.Context, id string) (Edge, error)
	ListEdges(ctx context.Context, workflowID string) ([]Edge, error)
}

// TriggerRepository defines the interface for trigger operations.
type TriggerRepository interface {
	SaveTrigger(ctx context.Context, t Trigger) error
	GetTrigger(ctx context.Context, id string) (Trigger, error)
	ListTriggers(ctx context.Context, workflowID string) ([]Trigger, error)
}

// Storage combines all repositories.
type Storage interface {
	WorkflowRepository
	ExecutionRepository
	EventRepository
	NodeRepository
	EdgeRepository
	TriggerRepository
}
