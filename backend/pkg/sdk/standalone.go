package sdk

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/smilemakc/mbflow/internal/application/engine"
	"github.com/smilemakc/mbflow/pkg/models"
)

// ExecuteWorkflowStandalone executes a workflow in standalone mode without persistence.
// This is useful for:
//   - Examples and demos
//   - Testing workflows before deploying to production
//   - Simple automation scripts that don't need execution history
//   - Embedded scenarios where you want to execute workflows in-memory
//
// The workflow is executed synchronously and returns the final result.
// No data is persisted to any database - everything runs in-memory.
func (c *Client) ExecuteWorkflowStandalone(
	ctx context.Context,
	workflow *models.Workflow,
	input map[string]interface{},
	opts *engine.ExecutionOptions,
) (*models.Execution, error) {
	if err := c.checkClosed(); err != nil {
		return nil, err
	}

	if workflow == nil {
		return nil, fmt.Errorf("workflow is required")
	}

	// Only available in embedded mode
	if c.config.Mode != ModeEmbedded {
		return nil, fmt.Errorf("standalone execution only available in embedded mode")
	}

	if c.executorManager == nil {
		return nil, fmt.Errorf("executor manager not initialized")
	}

	// Use default options if not provided
	if opts == nil {
		opts = engine.DefaultExecutionOptions()
	}

	// Set workflow ID if not set
	if workflow.ID == "" {
		workflow.ID = uuid.New().String()
	}

	// Validate input
	if input == nil {
		input = make(map[string]interface{})
	}

	// Create execution record
	execution := &models.Execution{
		ID:           uuid.New().String(),
		WorkflowID:   workflow.ID,
		WorkflowName: workflow.Name,
		Status:       models.ExecutionStatusRunning,
		Input:        input,
		Variables:    mergeVariables(workflow.Variables, opts.Variables),
		StartedAt:    time.Now(),
	}

	// Create execution state
	execState := engine.NewExecutionState(
		execution.ID,
		workflow.ID,
		workflow,
		input,
		execution.Variables,
	)

	// Create node executor and DAG executor
	nodeExecutor := engine.NewNodeExecutor(c.executorManager)
	dagExecutor := engine.NewDAGExecutor(nodeExecutor)

	// Execute DAG
	execErr := dagExecutor.Execute(ctx, execState, opts)

	// Update execution with results
	now := time.Now()
	execution.CompletedAt = &now
	execution.Duration = execution.CalculateDuration()

	if execErr != nil {
		execution.Status = models.ExecutionStatusFailed
		execution.Error = execErr.Error()
	} else {
		execution.Status = models.ExecutionStatusCompleted
		// Set output to final node's output
		execution.Output = getFinalOutput(execState, workflow)
	}

	// Build node executions
	execution.NodeExecutions = buildNodeExecutions(execState, workflow)

	return execution, execErr
}

// mergeVariables merges workflow and execution variables.
// Execution variables override workflow variables.
func mergeVariables(
	workflowVars map[string]interface{},
	executionVars map[string]interface{},
) map[string]interface{} {
	merged := make(map[string]interface{})

	// Copy workflow variables
	for k, v := range workflowVars {
		merged[k] = v
	}

	// Execution variables override workflow variables
	for k, v := range executionVars {
		merged[k] = v
	}

	return merged
}

// getFinalOutput gets output from leaf nodes (nodes with no outgoing edges)
func getFinalOutput(execState *engine.ExecutionState, workflow *models.Workflow) map[string]interface{} {
	// Find leaf nodes (nodes with no outgoing edges)
	leafNodes := findLeafNodes(workflow)

	if len(leafNodes) == 0 {
		return nil
	}

	// If single leaf, return its output
	if len(leafNodes) == 1 {
		if output, ok := execState.GetNodeOutput(leafNodes[0].ID); ok {
			if outputMap, ok := output.(map[string]interface{}); ok {
				return outputMap
			}
		}
	}

	// Multiple leaves - merge outputs namespaced by node ID
	merged := make(map[string]interface{})
	for _, node := range leafNodes {
		if output, ok := execState.GetNodeOutput(node.ID); ok {
			merged[node.ID] = output
		}
	}

	return merged
}

// findLeafNodes finds nodes with no outgoing edges
func findLeafNodes(workflow *models.Workflow) []*models.Node {
	hasOutgoing := make(map[string]bool)
	for _, edge := range workflow.Edges {
		hasOutgoing[edge.From] = true
	}

	leaves := []*models.Node{}
	for _, node := range workflow.Nodes {
		if !hasOutgoing[node.ID] {
			leaves = append(leaves, node)
		}
	}

	return leaves
}

// buildNodeExecutions builds NodeExecution records from execution state
func buildNodeExecutions(
	execState *engine.ExecutionState,
	workflow *models.Workflow,
) []*models.NodeExecution {
	nodeExecs := make([]*models.NodeExecution, 0, len(workflow.Nodes))

	for _, node := range workflow.Nodes {
		nodeExec := &models.NodeExecution{
			ID:          uuid.New().String(),
			ExecutionID: execState.ExecutionID,
			NodeID:      node.ID,
			NodeName:    node.Name,
			NodeType:    node.Type,
		}

		// Get status
		if status, ok := execState.GetNodeStatus(node.ID); ok {
			nodeExec.Status = status
		}

		// Get output
		if output, ok := execState.GetNodeOutput(node.ID); ok {
			if outputMap, ok := output.(map[string]interface{}); ok {
				nodeExec.Output = outputMap
			}
		}

		// Get error
		if err, ok := execState.GetNodeError(node.ID); ok {
			nodeExec.Error = err.Error()
		}

		nodeExecs = append(nodeExecs, nodeExec)
	}

	return nodeExecs
}
