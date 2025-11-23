package mbflow

import (
	"mbflow/internal/application/executor"
	"mbflow/internal/infrastructure/monitoring"
)

type NodeExecutor = executor.NodeExecutor

// WorkflowEngine represents a workflow executor.
// It provides methods for executing workflows and nodes.
type WorkflowEngine = executor.WorkflowEngine

type EngineConfig = executor.EngineConfig

type RetryPolicy = executor.RetryPolicy

// NewWorkflowEngine creates a new workflow executor.
func NewWorkflowEngine(config *EngineConfig) *WorkflowEngine {
	if config == nil {
		config = &EngineConfig{
			RetryPolicy:      executor.DefaultRetryPolicy(),
			EnableMonitoring: true,
		}
	}
	return executor.NewWorkflowEngine(config)
}

// Ensure MetricsCollector implements ExecutorMetrics
var _ ExecutorMetrics = (*monitoring.MetricsCollector)(nil)

// ExecutionObserver is a callback for workflow execution events.
type ExecutionObserver = monitoring.ExecutionObserver
