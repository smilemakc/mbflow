package domain

import (
	"context"
)

// WorkflowRepository defines the interface for storing and retrieving workflows.
type WorkflowRepository interface {
	SaveWorkflow(ctx context.Context, w *Workflow) error
	GetWorkflow(ctx context.Context, id string) (*Workflow, error)
	ListWorkflows(ctx context.Context) ([]*Workflow, error)
}

// ExecutionRepository defines the interface for storing and retrieving executions.
type ExecutionRepository interface {
	SaveExecution(ctx context.Context, e *Execution) error
	GetExecution(ctx context.Context, id string) (*Execution, error)
	ListExecutions(ctx context.Context) ([]*Execution, error)
}

// EventRepository defines the interface for storing and retrieving events.
type EventRepository interface {
	AppendEvent(ctx context.Context, e *Event) error
	ListEventsByExecution(ctx context.Context, executionID string) ([]*Event, error)
}

// NodeRepository defines the interface for storing and retrieving nodes.
type NodeRepository interface {
	SaveNode(ctx context.Context, n *Node) error
	GetNode(ctx context.Context, id string) (*Node, error)
	ListNodes(ctx context.Context, workflowID string) ([]*Node, error)
}

// EdgeRepository defines the interface for storing and retrieving edges.
type EdgeRepository interface {
	SaveEdge(ctx context.Context, e *Edge) error
	GetEdge(ctx context.Context, id string) (*Edge, error)
	ListEdges(ctx context.Context, workflowID string) ([]*Edge, error)
}

// TriggerRepository defines the interface for storing and retrieving triggers.
type TriggerRepository interface {
	SaveTrigger(ctx context.Context, t *Trigger) error
	GetTrigger(ctx context.Context, id string) (*Trigger, error)
	ListTriggers(ctx context.Context, workflowID string) ([]*Trigger, error)
}

// Storage is an aggregate interface combining all repositories.
type Storage interface {
	WorkflowRepository
	ExecutionRepository
	EventRepository
	NodeRepository
	EdgeRepository
	TriggerRepository
}
