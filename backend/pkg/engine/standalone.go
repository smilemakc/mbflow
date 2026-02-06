package engine

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/smilemakc/mbflow/pkg/executor"
	"github.com/smilemakc/mbflow/pkg/models"
)

// standaloneExecutor implements StandaloneExecutor for in-memory workflow execution.
type standaloneExecutor struct {
	executorManager executor.Manager
}

// NewStandaloneExecutor creates a new standalone executor that runs workflows
// in-memory without persistence. This is useful for testing, demos, and
// simple automation scripts.
func NewStandaloneExecutor(executorManager executor.Manager) StandaloneExecutor {
	return &standaloneExecutor{
		executorManager: executorManager,
	}
}

// ExecuteStandalone executes a workflow synchronously without persistence.
func (e *standaloneExecutor) ExecuteStandalone(
	ctx context.Context,
	workflow *models.Workflow,
	input map[string]interface{},
	opts *ExecutionOptions,
) (*models.Execution, error) {
	if workflow == nil {
		return nil, fmt.Errorf("workflow is required")
	}

	if e.executorManager == nil {
		return nil, fmt.Errorf("executor manager not initialized")
	}

	// Use default options if not provided
	if opts == nil {
		opts = DefaultExecutionOptions()
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
	state := newExecutionState(execution.ID, workflow.ID, workflow, input, execution.Variables)

	// Execute workflow
	execErr := e.executeDAG(ctx, state, opts)

	// Update execution with results
	now := time.Now()
	execution.CompletedAt = &now
	execution.Duration = execution.CalculateDuration()

	if execErr != nil {
		execution.Status = models.ExecutionStatusFailed
		execution.Error = execErr.Error()
	} else {
		execution.Status = models.ExecutionStatusCompleted
		execution.Output = getFinalOutput(state, workflow)
	}

	// Build node executions
	execution.NodeExecutions = buildNodeExecutions(state, workflow)

	return execution, execErr
}

// executeDAG executes the workflow DAG.
func (e *standaloneExecutor) executeDAG(ctx context.Context, state *executionState, opts *ExecutionOptions) error {
	// Apply timeout if set
	if opts.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, opts.Timeout)
		defer cancel()
	}

	// Get execution order (topological sort)
	order, err := topologicalSort(state.workflow)
	if err != nil {
		return fmt.Errorf("invalid workflow DAG: %w", err)
	}

	// Execute nodes in order
	for _, nodeID := range order {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		node := getNodeByID(state.workflow, nodeID)
		if node == nil {
			return fmt.Errorf("node not found: %s", nodeID)
		}

		// Check if node should be executed (edge conditions)
		shouldExecute, err := e.shouldExecuteNode(state, node)
		if err != nil {
			return fmt.Errorf("condition evaluation failed for node %s: %w", nodeID, err)
		}

		if !shouldExecute {
			state.setNodeStatus(nodeID, models.NodeExecutionStatusSkipped)
			continue
		}

		// Execute node
		if err := e.executeNode(ctx, state, node, opts); err != nil {
			if opts.ContinueOnError {
				state.setNodeError(nodeID, err)
				continue
			}
			return err
		}
	}

	return nil
}

// shouldExecuteNode checks if a node should be executed based on edge conditions.
func (e *standaloneExecutor) shouldExecuteNode(state *executionState, node *models.Node) (bool, error) {
	// Find incoming edges
	var incomingEdges []*models.Edge
	for _, edge := range state.workflow.Edges {
		if edge.To == node.ID {
			incomingEdges = append(incomingEdges, edge)
		}
	}

	// If no incoming edges, always execute (start node)
	if len(incomingEdges) == 0 {
		return true, nil
	}

	// Check if at least one incoming edge's condition is met
	for _, edge := range incomingEdges {
		// If no condition, check if source node completed
		if edge.Condition == "" {
			status, _ := state.getNodeStatus(edge.From)
			if status == models.NodeExecutionStatusCompleted {
				return true, nil
			}
			continue
		}

		// Evaluate condition
		sourceOutput, _ := state.getNodeOutput(edge.From)
		result, err := evaluateCondition(edge.Condition, sourceOutput)
		if err != nil {
			return false, err
		}
		if result {
			return true, nil
		}
	}

	return false, nil
}

// executeNode executes a single node.
func (e *standaloneExecutor) executeNode(ctx context.Context, state *executionState, node *models.Node, opts *ExecutionOptions) error {
	state.setNodeStatus(node.ID, models.NodeExecutionStatusRunning)

	// Get executor for node type
	exec, err := e.executorManager.Get(node.Type)
	if err != nil {
		state.setNodeStatus(node.ID, models.NodeExecutionStatusFailed)
		state.setNodeError(node.ID, err)
		return fmt.Errorf("executor not found for type %s: %w", node.Type, err)
	}

	// Build input from parent nodes
	input := buildNodeInput(state, node)

	// Apply node timeout if configured
	nodeCtx := ctx
	if timeout, ok := node.Config["timeout"].(int); ok && timeout > 0 {
		var cancel context.CancelFunc
		nodeCtx, cancel = context.WithTimeout(ctx, time.Duration(timeout)*time.Millisecond)
		defer cancel()
	}

	// Execute with retry if configured
	var output interface{}
	var execErr error

	maxAttempts := 1
	if opts.RetryPolicy != nil && opts.RetryPolicy.MaxAttempts > 0 {
		maxAttempts = opts.RetryPolicy.MaxAttempts
	}

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		output, execErr = exec.Execute(nodeCtx, node.Config, input)
		if execErr == nil {
			break
		}

		if attempt < maxAttempts && opts.RetryPolicy != nil {
			delay := calculateRetryDelay(opts.RetryPolicy, attempt)
			select {
			case <-time.After(delay):
				continue
			case <-ctx.Done():
				return ctx.Err()
			}
		}
	}

	if execErr != nil {
		state.setNodeStatus(node.ID, models.NodeExecutionStatusFailed)
		state.setNodeError(node.ID, execErr)
		return fmt.Errorf("node %s execution failed: %w", node.ID, execErr)
	}

	// Check output size limit
	if opts.MaxOutputSize > 0 {
		if size := estimateSize(output); size > opts.MaxOutputSize {
			err := fmt.Errorf("output size %d exceeds limit %d", size, opts.MaxOutputSize)
			state.setNodeStatus(node.ID, models.NodeExecutionStatusFailed)
			state.setNodeError(node.ID, err)
			return err
		}
	}

	state.setNodeOutput(node.ID, output)
	state.setNodeStatus(node.ID, models.NodeExecutionStatusCompleted)

	return nil
}

// buildNodeInput builds input for a node from its parent outputs.
func buildNodeInput(state *executionState, node *models.Node) interface{} {
	// Find parent nodes
	var parentOutputs []interface{}
	for _, edge := range state.workflow.Edges {
		if edge.To == node.ID {
			if output, ok := state.getNodeOutput(edge.From); ok {
				parentOutputs = append(parentOutputs, output)
			}
		}
	}

	// If no parents, use workflow input
	if len(parentOutputs) == 0 {
		return state.input
	}

	// Single parent - pass its output directly
	if len(parentOutputs) == 1 {
		return parentOutputs[0]
	}

	// Multiple parents - merge outputs
	merged := make(map[string]interface{})
	for i, output := range parentOutputs {
		if outputMap, ok := output.(map[string]interface{}); ok {
			for k, v := range outputMap {
				merged[k] = v
			}
		} else {
			merged[fmt.Sprintf("input_%d", i)] = output
		}
	}

	return merged
}

// calculateRetryDelay calculates the delay before a retry attempt.
func calculateRetryDelay(policy *RetryPolicy, attempt int) time.Duration {
	if policy == nil {
		return 0
	}

	delay := policy.InitialDelay

	switch policy.BackoffStrategy {
	case BackoffConstant:
		// Use initial delay
	case BackoffLinear:
		delay = policy.InitialDelay * time.Duration(attempt)
	case BackoffExponential:
		delay = policy.InitialDelay * time.Duration(1<<uint(attempt-1))
	}

	if policy.MaxDelay > 0 && delay > policy.MaxDelay {
		delay = policy.MaxDelay
	}

	return delay
}

// estimateSize estimates the size of a value in bytes.
func estimateSize(v interface{}) int64 {
	switch val := v.(type) {
	case nil:
		return 0
	case string:
		return int64(len(val))
	case []byte:
		return int64(len(val))
	case map[string]interface{}:
		var size int64
		for k, v := range val {
			size += int64(len(k)) + estimateSize(v)
		}
		return size
	case []interface{}:
		var size int64
		for _, item := range val {
			size += estimateSize(item)
		}
		return size
	default:
		return 8 // Approximate size for primitives
	}
}

// evaluateCondition evaluates a simple condition expression.
func evaluateCondition(condition string, output interface{}) (bool, error) {
	// For now, implement a simple evaluator
	// In production, this should use a proper expression evaluator
	if condition == "" || condition == "true" {
		return true, nil
	}
	if condition == "false" {
		return false, nil
	}

	// Basic condition support - just check if output exists
	if output != nil {
		return true, nil
	}

	return false, nil
}

// mergeVariables merges workflow and execution variables.
func mergeVariables(workflowVars, executionVars map[string]interface{}) map[string]interface{} {
	merged := make(map[string]interface{})
	for k, v := range workflowVars {
		merged[k] = v
	}
	for k, v := range executionVars {
		merged[k] = v
	}
	return merged
}

// topologicalSort returns node IDs in execution order (respecting dependencies).
func topologicalSort(workflow *models.Workflow) ([]string, error) {
	// Build adjacency list and in-degree map
	inDegree := make(map[string]int)
	adjacency := make(map[string][]string)

	// Initialize all nodes with 0 in-degree
	for _, node := range workflow.Nodes {
		inDegree[node.ID] = 0
		adjacency[node.ID] = []string{}
	}

	// Build graph from edges
	for _, edge := range workflow.Edges {
		adjacency[edge.From] = append(adjacency[edge.From], edge.To)
		inDegree[edge.To]++
	}

	// Kahn's algorithm
	var queue []string
	for nodeID, degree := range inDegree {
		if degree == 0 {
			queue = append(queue, nodeID)
		}
	}

	var order []string
	for len(queue) > 0 {
		// Pop from queue
		nodeID := queue[0]
		queue = queue[1:]
		order = append(order, nodeID)

		// Reduce in-degree of neighbors
		for _, neighbor := range adjacency[nodeID] {
			inDegree[neighbor]--
			if inDegree[neighbor] == 0 {
				queue = append(queue, neighbor)
			}
		}
	}

	// Check for cycles
	if len(order) != len(workflow.Nodes) {
		return nil, fmt.Errorf("workflow contains a cycle")
	}

	return order, nil
}

// getNodeByID returns a node by its ID.
func getNodeByID(workflow *models.Workflow, nodeID string) *models.Node {
	for _, node := range workflow.Nodes {
		if node.ID == nodeID {
			return node
		}
	}
	return nil
}

// getFinalOutput gets output from leaf nodes.
func getFinalOutput(state *executionState, workflow *models.Workflow) map[string]interface{} {
	leafNodes := findLeafNodes(workflow)

	if len(leafNodes) == 0 {
		return nil
	}

	if len(leafNodes) == 1 {
		if output, ok := state.getNodeOutput(leafNodes[0].ID); ok {
			if outputMap, ok := output.(map[string]interface{}); ok {
				return outputMap
			}
		}
	}

	merged := make(map[string]interface{})
	for _, node := range leafNodes {
		if output, ok := state.getNodeOutput(node.ID); ok {
			merged[node.ID] = output
		}
	}

	return merged
}

// findLeafNodes finds nodes with no outgoing edges.
func findLeafNodes(workflow *models.Workflow) []*models.Node {
	hasOutgoing := make(map[string]bool)
	for _, edge := range workflow.Edges {
		hasOutgoing[edge.From] = true
	}

	var leaves []*models.Node
	for _, node := range workflow.Nodes {
		if !hasOutgoing[node.ID] {
			leaves = append(leaves, node)
		}
	}

	return leaves
}

// buildNodeExecutions builds NodeExecution records from execution state.
func buildNodeExecutions(state *executionState, workflow *models.Workflow) []*models.NodeExecution {
	nodeExecs := make([]*models.NodeExecution, 0, len(workflow.Nodes))

	for _, node := range workflow.Nodes {
		nodeExec := &models.NodeExecution{
			ID:          uuid.New().String(),
			ExecutionID: state.executionID,
			NodeID:      node.ID,
			NodeName:    node.Name,
			NodeType:    node.Type,
		}

		if status, ok := state.getNodeStatus(node.ID); ok {
			nodeExec.Status = status
		}

		if output, ok := state.getNodeOutput(node.ID); ok {
			if outputMap, ok := output.(map[string]interface{}); ok {
				nodeExec.Output = outputMap
			}
		}

		if err, ok := state.getNodeError(node.ID); ok {
			nodeExec.Error = err.Error()
		}

		nodeExecs = append(nodeExecs, nodeExec)
	}

	return nodeExecs
}
