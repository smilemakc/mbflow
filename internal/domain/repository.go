package domain

import (
	"context"

	"github.com/google/uuid"
)

// WorkflowRepository defines the repository interface for workflow persistence.
// Since Workflow is now an aggregate root that owns Node, Edge, and Trigger,
// the repository handles persisting the entire aggregate.
type WorkflowRepository interface {
	// SaveWorkflow persists a workflow with all its child entities (nodes, edges, triggers)
	SaveWorkflow(ctx context.Context, workflow Workflow) error

	// GetWorkflow retrieves a workflow with all its child entities
	GetWorkflow(ctx context.Context, id uuid.UUID) (Workflow, error)

	// GetWorkflowByName retrieves a workflow by name and version
	GetWorkflowByName(ctx context.Context, name, version string) (Workflow, error)

	// ListWorkflows returns all workflows
	ListWorkflows(ctx context.Context) ([]Workflow, error)

	// DeleteWorkflow removes a workflow and all its child entities
	DeleteWorkflow(ctx context.Context, id uuid.UUID) error

	// WorkflowExists checks if a workflow exists
	WorkflowExists(ctx context.Context, id uuid.UUID) (bool, error)
}

// ExecutionRepository defines the interface for execution persistence.
// With Event Sourcing, executions are rebuilt from events.
type ExecutionRepository interface {
	// GetExecution retrieves an execution by rebuilding it from events
	GetExecution(ctx context.Context, id uuid.UUID) (Execution, error)

	// ListExecutions returns all executions for a workflow
	ListExecutionsByWorkflow(ctx context.Context, workflowID uuid.UUID) ([]Execution, error)

	// ListAllExecutions returns all executions (paginated)
	ListAllExecutions(ctx context.Context, limit, offset int) ([]Execution, error)

	// SaveSnapshot optionally saves a snapshot of execution state for performance
	// This is optional optimization - primary source of truth is events
	SaveSnapshot(ctx context.Context, execution Execution) error

	// GetSnapshot retrieves the latest snapshot if available
	GetSnapshot(ctx context.Context, id uuid.UUID) (Execution, error)
}

// EventStore defines the interface for event sourcing persistence.
// This is the primary storage mechanism for execution state.
type EventStore interface {
	// AppendEvent appends a single event to the event stream
	AppendEvent(ctx context.Context, event Event) error

	// AppendEvents appends multiple events atomically
	AppendEvents(ctx context.Context, events []Event) error

	// GetEvents retrieves all events for an execution
	GetEvents(ctx context.Context, executionID uuid.UUID) ([]Event, error)

	// GetEventsSince retrieves events after a specific sequence number
	GetEventsSince(ctx context.Context, executionID uuid.UUID, sequenceNumber int64) ([]Event, error)

	// GetEventsByType retrieves events of a specific type
	GetEventsByType(ctx context.Context, executionID uuid.UUID, eventType EventType) ([]Event, error)

	// GetEventsByWorkflow retrieves all events for a workflow (all executions)
	GetEventsByWorkflow(ctx context.Context, workflowID uuid.UUID) ([]Event, error)

	// GetEventCount returns the number of events for an execution
	GetEventCount(ctx context.Context, executionID uuid.UUID) (int64, error)
}

// Storage is the unified repository interface that combines all persistence operations.
// This provides a single point of access to all domain entity storage.
type Storage interface {
	WorkflowRepository
	ExecutionRepository
	EventStore

	// Transaction support for atomic operations
	BeginTransaction(ctx context.Context) (context.Context, error)
	CommitTransaction(ctx context.Context) error
	RollbackTransaction(ctx context.Context) error

	// Health check
	Ping(ctx context.Context) error

	// Close closes the storage connection
	Close() error
}

// QueryOptions provides options for querying repositories
type QueryOptions struct {
	Limit      int
	Offset     int
	SortBy     string
	SortOrder  string // "asc" or "desc"
	Filters    map[string]interface{}
	IncludeAll bool // Include child entities, events, etc.
}

// ExecutionQuery provides advanced querying capabilities for executions
type ExecutionQuery interface {
	// FindExecutionsByPhase finds executions in a specific phase
	FindExecutionsByPhase(ctx context.Context, phase ExecutionPhase) ([]Execution, error)

	// FindExecutionsByWorkflowAndPhase finds executions for a workflow in a specific phase
	FindExecutionsByWorkflowAndPhase(ctx context.Context, workflowID uuid.UUID, phase ExecutionPhase) ([]Execution, error)

	// FindFailedExecutions finds all failed executions
	FindFailedExecutions(ctx context.Context, workflowID uuid.UUID) ([]Execution, error)

	// FindExecutionsInProgress finds all running executions
	FindExecutionsInProgress(ctx context.Context) ([]Execution, error)

	// GetExecutionStatistics returns statistics about executions
	GetExecutionStatistics(ctx context.Context, workflowID uuid.UUID) (map[string]interface{}, error)
}
