package engine

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/smilemakc/mbflow/pkg/executor"
	"github.com/smilemakc/mbflow/pkg/models"
)

// standaloneExecutor implements StandaloneExecutor using the unified DAGExecutor.
type standaloneExecutor struct {
	dagExecutor *DAGExecutor
}

// NewStandaloneExecutor creates a new standalone executor that runs workflows
// in-memory without persistence. Uses SimpleConditionEvaluator and NoOpNotifier.
func NewStandaloneExecutor(executorManager executor.Manager) StandaloneExecutor {
	nodeExecutor := NewNodeExecutor(executorManager)
	return &standaloneExecutor{
		dagExecutor: NewDAGExecutor(
			nodeExecutor,
			NewExprConditionEvaluator(),
			NewNoOpNotifier(),
			NewNilWorkflowLoader(),
		),
	}
}

// ExecuteStandalone executes a workflow synchronously without persistence.
func (e *standaloneExecutor) ExecuteStandalone(
	ctx context.Context,
	workflow *models.Workflow,
	input map[string]any,
	opts *ExecutionOptions,
) (*models.Execution, error) {
	if workflow == nil {
		return nil, fmt.Errorf("workflow is required")
	}

	if opts == nil {
		opts = DefaultExecutionOptions()
	}

	if workflow.ID == "" {
		workflow.ID = uuid.New().String()
	}

	if input == nil {
		input = make(map[string]any)
	}

	// Apply timeout
	if opts.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, opts.Timeout)
		defer cancel()
	}

	execution := &models.Execution{
		ID:           uuid.New().String(),
		WorkflowID:   workflow.ID,
		WorkflowName: workflow.Name,
		Status:       models.ExecutionStatusRunning,
		Input:        input,
		Variables:    MergeVariables(workflow.Variables, opts.Variables),
		StartedAt:    time.Now(),
	}

	state := NewExecutionState(execution.ID, workflow.ID, workflow, input, execution.Variables)

	execErr := e.dagExecutor.Execute(ctx, state, opts)

	now := time.Now()
	execution.CompletedAt = &now
	execution.Duration = execution.CalculateDuration()

	if execErr != nil {
		execution.Status = models.ExecutionStatusFailed
		execution.Error = execErr.Error()
	} else {
		execution.Status = models.ExecutionStatusCompleted
		execution.Output = getFinalOutputFromState(state, workflow)
	}

	execution.NodeExecutions = buildNodeExecutionsFromState(state, workflow)

	return execution, execErr
}

// getFinalOutputFromState gets output from leaf nodes.
func getFinalOutputFromState(state *ExecutionState, workflow *models.Workflow) map[string]any {
	leafNodes := FindLeafNodes(workflow)

	if len(leafNodes) == 0 {
		return nil
	}

	if len(leafNodes) == 1 {
		if output, ok := state.GetNodeOutput(leafNodes[0].ID); ok {
			return ToMapInterface(output)
		}
	}

	merged := make(map[string]any)
	for _, node := range leafNodes {
		if output, ok := state.GetNodeOutput(node.ID); ok {
			merged[node.ID] = output
		}
	}

	return merged
}

// buildNodeExecutionsFromState builds NodeExecution records from execution state.
func buildNodeExecutionsFromState(state *ExecutionState, workflow *models.Workflow) []*models.NodeExecution {
	nodeExecs := make([]*models.NodeExecution, 0, len(workflow.Nodes))

	for _, node := range workflow.Nodes {
		nodeExec := &models.NodeExecution{
			ID:          uuid.New().String(),
			ExecutionID: state.ExecutionID,
			NodeID:      node.ID,
			NodeName:    node.Name,
			NodeType:    node.Type,
		}

		if status, ok := state.GetNodeStatus(node.ID); ok {
			nodeExec.Status = status
		}

		if input, ok := state.GetNodeInput(node.ID); ok {
			nodeExec.Input = ToMapInterface(input)
		}

		if output, ok := state.GetNodeOutput(node.ID); ok {
			nodeExec.Output = ToMapInterface(output)
		}

		if config, ok := state.GetNodeConfig(node.ID); ok {
			nodeExec.Config = config
		}

		if resolvedConfig, ok := state.GetNodeResolvedConfig(node.ID); ok {
			nodeExec.ResolvedConfig = resolvedConfig
		}

		if err, ok := state.GetNodeError(node.ID); ok {
			nodeExec.Error = err.Error()
		}

		if startTime, ok := state.GetNodeStartTime(node.ID); ok {
			nodeExec.StartedAt = startTime
		}
		if endTime, ok := state.GetNodeEndTime(node.ID); ok {
			nodeExec.CompletedAt = &endTime
		}

		nodeExecs = append(nodeExecs, nodeExec)
	}

	return nodeExecs
}
