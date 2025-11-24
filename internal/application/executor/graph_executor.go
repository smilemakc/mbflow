package executor

import (
	"context"
	"fmt"

	"github.com/smilemakc/mbflow/internal/domain"
)

// ExecutorRegistry is a legacy interface kept for backward compatibility.
// New code should rely on WorkflowEngine and the node executor registry there.
type ExecutorRegistry interface {
	GetExecutor(nodeType domain.NodeType) (NodeExecutor, bool)
}

// GraphExecutor is a deprecated wrapper kept to avoid breaking older callers.
// Use WorkflowEngine.ExecuteWorkflow instead.
type GraphExecutor struct {
	planner *ExecutionPlanner
}

// NewGraphExecutor creates a legacy graph executor.
// It only builds plans; execution is not supported and will return an error.
func NewGraphExecutor(_ *EngineConfig, _ ExecutorRegistry) *GraphExecutor {
	return &GraphExecutor{
		planner: NewExecutionPlanner(),
	}
}

// BuildExecutionPlan builds an execution plan from the given workflow.
func (e *GraphExecutor) BuildExecutionPlan(
	_ context.Context,
	workflow domain.Workflow,
	_ domain.Trigger,
	_ map[string]any,
) (*ExecutionPlan, error) {
	return e.planner.CreatePlan(workflow)
}

// Execute is no longer implemented in the legacy GraphExecutor.
// Callers should migrate to WorkflowEngine.ExecuteWorkflow.
func (e *GraphExecutor) Execute(ctx context.Context, plan *ExecutionPlan) (*ExecutionState, error) {
	return nil, fmt.Errorf("GraphExecutor is deprecated; use WorkflowEngine.ExecuteWorkflow (ctx: %v, plan depth: %d)", ctx.Err(), len(plan.Waves))
}
