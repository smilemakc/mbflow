package domain

import (
	"context"
)

// WorkflowRepository defines the repository interface for workflow persistence.
// Repositories abstract data access and provide a collection-like interface for domain entities.
// This is part of the infrastructure layer but the interface belongs to the domain layer.
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

// ExecutionStateRepository defines the interface for storing and retrieving execution states.
type ExecutionStateRepository interface {
	SaveExecutionState(ctx context.Context, state *ExecutionState) error
	GetExecutionState(ctx context.Context, executionID string) (*ExecutionState, error)
	DeleteExecutionState(ctx context.Context, executionID string) error
}

// Storage is an aggregate repository interface that combines all domain repositories.
// This interface provides a unified access point to all persistence operations
// for domain entities, following the repository pattern from DDD.
type Storage interface {
	WorkflowRepository
	ExecutionRepository
	EventRepository
	NodeRepository
	EdgeRepository
	TriggerRepository
	ExecutionStateRepository
}
