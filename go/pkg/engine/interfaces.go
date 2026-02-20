package engine

import (
	"context"

	"github.com/smilemakc/mbflow/go/pkg/models"
)

// ExecutionRunner executes workflows and manages their lifecycle.
// This interface abstracts the workflow execution engine for use by the SDK
// and other consumers without requiring internal package imports.
type ExecutionRunner interface {
	// Execute starts a new workflow execution with the given input.
	Execute(ctx context.Context, workflow *models.Workflow, input map[string]any, opts *ExecutionOptions) (*models.Execution, error)

	// GetExecution retrieves an execution by ID.
	GetExecution(ctx context.Context, executionID string) (*models.Execution, error)

	// CancelExecution cancels a running execution.
	CancelExecution(ctx context.Context, executionID string) error
}

// StandaloneExecutor executes workflows without persistence.
// This is useful for testing, demos, and simple automation scripts.
type StandaloneExecutor interface {
	// ExecuteStandalone executes a workflow synchronously without persistence.
	// All execution happens in-memory and no data is stored to a database.
	ExecuteStandalone(ctx context.Context, workflow *models.Workflow, input map[string]any, opts *ExecutionOptions) (*models.Execution, error)
}

// ObserverManager manages execution event observers.
// It allows registration of observers that receive notifications
// about execution events (start, complete, fail, etc.).
type ObserverManager interface {
	// Notify sends an event to all registered observers.
	Notify(ctx context.Context, event *Event) error

	// Register adds an observer to receive events.
	Register(observer Observer) error

	// Unregister removes an observer.
	Unregister(name string) error

	// Count returns the number of registered observers.
	Count() int
}

// Observer receives execution events.
type Observer interface {
	// Name returns a unique identifier for the observer.
	Name() string

	// OnEvent is called when an execution event occurs.
	OnEvent(ctx context.Context, event *Event) error
}

// Event represents an execution event.
type Event struct {
	// Type is the kind of event (e.g., "execution.started", "node.completed")
	Type string

	// ExecutionID is the ID of the execution this event relates to
	ExecutionID string

	// WorkflowID is the ID of the workflow being executed
	WorkflowID string

	// NodeID is the ID of the node (for node-level events)
	NodeID string

	// Status is the current status
	Status string

	// Error contains error information if applicable
	Error string

	// Metadata contains additional event-specific data
	Metadata map[string]any
}

// ConditionEvaluator evaluates edge conditions.
// Simple impl: string matching. Full impl: expr-lang with caching.
type ConditionEvaluator interface {
	// Evaluate evaluates a condition expression against node output.
	// Returns true if the condition passes.
	Evaluate(condition string, nodeOutput any) (bool, error)
}

// ExecutionNotifier receives execution lifecycle events.
// NoOp impl: for standalone. Observer impl: for full engine.
type ExecutionNotifier interface {
	// Notify sends an execution event.
	Notify(ctx context.Context, event ExecutionEvent)
}

// EventType constants for execution events.
const (
	EventTypeExecutionStarted         = "execution.started"
	EventTypeExecutionCompleted       = "execution.completed"
	EventTypeExecutionFailed          = "execution.failed"
	EventTypeExecutionCancelled       = "execution.cancelled"
	EventTypeWaveStarted              = "wave.started"
	EventTypeWaveCompleted            = "wave.completed"
	EventTypeNodeStarted              = "node.started"
	EventTypeNodeCompleted            = "node.completed"
	EventTypeNodeFailed               = "node.failed"
	EventTypeNodeSkipped              = "node.skipped"
	EventTypeNodeRetrying             = "node.retrying"
	EventTypeLoopIteration            = "loop.iteration"
	EventTypeLoopExhausted            = "loop.exhausted"
	EventTypeSubWorkflowProgress      = "sub_workflow.progress"
	EventTypeSubWorkflowItemCompleted = "sub_workflow.item_completed"
	EventTypeSubWorkflowItemFailed    = "sub_workflow.item_failed"
)
