package engine

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/expr-lang/expr"
	"github.com/smilemakc/mbflow/internal/application/observer"
	"github.com/smilemakc/mbflow/pkg/models"
)

// DAGExecutor executes workflow nodes in topological order with wave-based parallelism
type DAGExecutor struct {
	nodeExecutor    *NodeExecutor
	observerManager *observer.ObserverManager
}

// NewDAGExecutor creates a new DAG executor
func NewDAGExecutor(nodeExecutor *NodeExecutor, observerManager *observer.ObserverManager) *DAGExecutor {
	return &DAGExecutor{
		nodeExecutor:    nodeExecutor,
		observerManager: observerManager,
	}
}

// Execute executes the workflow DAG
func (de *DAGExecutor) Execute(
	ctx context.Context,
	execState *ExecutionState,
	opts *ExecutionOptions,
) error {
	// 1. Build DAG from workflow
	dag := buildDAG(execState.Workflow)

	// 2. Perform topological sort to get execution waves
	waves, err := topologicalSort(dag)
	if err != nil {
		return fmt.Errorf("DAG validation failed: %w", err)
	}

	// 3. Execute waves sequentially, nodes in parallel within wave
	for waveIdx, wave := range waves {
		if err := de.executeWave(ctx, execState, wave, waveIdx, opts); err != nil {
			return fmt.Errorf("wave %d execution failed: %w", waveIdx, err)
		}
	}

	return nil
}

// executeWave executes all nodes in a wave in parallel
func (de *DAGExecutor) executeWave(
	ctx context.Context,
	execState *ExecutionState,
	wave []*models.Node,
	waveIdx int,
	opts *ExecutionOptions,
) error {
	waveStartTime := time.Now()

	// Notify wave started
	if de.observerManager != nil {
		nodeCount := len(wave)
		event := observer.Event{
			Type:        observer.EventTypeWaveStarted,
			ExecutionID: execState.ExecutionID,
			WorkflowID:  execState.WorkflowID,
			Timestamp:   waveStartTime,
			Status:      "running",
			WaveIndex:   &waveIdx,
			NodeCount:   &nodeCount,
		}
		de.observerManager.Notify(ctx, event)
	}

	var wg sync.WaitGroup
	errChan := make(chan error, len(wave))

	// Limit parallelism if configured
	semaphore := make(chan struct{}, opts.MaxParallelism)
	if opts.MaxParallelism <= 0 {
		// Unlimited parallelism
		semaphore = make(chan struct{}, len(wave))
	}

	for _, node := range wave {
		wg.Add(1)
		go func(n *models.Node) {
			defer wg.Done()

			// Acquire semaphore
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			// Check if node should be executed based on incoming edge conditions
			shouldExec, skipReason := de.shouldExecuteNode(execState, n)
			if !shouldExec {
				// Skip this node - mark as skipped
				execState.SetNodeStatus(n.ID, models.NodeExecutionStatusSkipped)

				// Notify node skipped
				if de.observerManager != nil {
					event := observer.Event{
						Type:        observer.EventTypeNodeSkipped,
						ExecutionID: execState.ExecutionID,
						WorkflowID:  execState.WorkflowID,
						Timestamp:   time.Now(),
						Status:      "skipped",
						NodeID:      &n.ID,
						NodeName:    &n.Name,
						NodeType:    &n.Type,
					}
					if skipReason != "" {
						event.Message = &skipReason
					}
					de.observerManager.Notify(ctx, event)
				}
				return
			}

			// Execute node
			if err := de.executeNode(ctx, execState, n, opts); err != nil {
				errChan <- fmt.Errorf("node %s failed: %w", n.ID, err)
			}
		}(node)
	}

	wg.Wait()
	close(errChan)

	// Check for errors
	for err := range errChan {
		if err != nil {
			return err
		}
	}

	// Notify wave completed
	if de.observerManager != nil {
		waveDuration := time.Since(waveStartTime).Milliseconds()
		event := observer.Event{
			Type:        observer.EventTypeWaveCompleted,
			ExecutionID: execState.ExecutionID,
			WorkflowID:  execState.WorkflowID,
			Timestamp:   time.Now(),
			Status:      "completed",
			WaveIndex:   &waveIdx,
			DurationMs:  &waveDuration,
		}
		de.observerManager.Notify(ctx, event)
	}

	return nil
}

// executeNode executes a single node
func (de *DAGExecutor) executeNode(
	ctx context.Context,
	execState *ExecutionState,
	node *models.Node,
	opts *ExecutionOptions,
) error {
	nodeStartTime := time.Now()

	// Mark as running and record start time
	execState.SetNodeStatus(node.ID, models.NodeExecutionStatusRunning)
	execState.SetNodeStartTime(node.ID, nodeStartTime)

	// Notify node started
	if de.observerManager != nil {
		event := observer.Event{
			Type:        observer.EventTypeNodeStarted,
			ExecutionID: execState.ExecutionID,
			WorkflowID:  execState.WorkflowID,
			Timestamp:   nodeStartTime,
			Status:      "running",
			NodeID:      &node.ID,
			NodeName:    &node.Name,
			NodeType:    &node.Type,
		}
		de.observerManager.Notify(ctx, event)
	}

	// Get parent nodes
	parentNodes := getParentNodes(execState.Workflow, node)

	// Prepare node context
	nodeCtx := PrepareNodeContext(execState, node, parentNodes, opts)

	// Execute node with template resolution via NodeExecutor
	execResult, err := de.nodeExecutor.Execute(ctx, nodeCtx)

	if err != nil {
		nodeEndTime := time.Now()
		// Store error and mark as failed
		execState.SetNodeError(node.ID, err)
		execState.SetNodeStatus(node.ID, models.NodeExecutionStatusFailed)
		execState.SetNodeEndTime(node.ID, nodeEndTime)

		// Notify node failed
		if de.observerManager != nil {
			nodeDuration := time.Since(nodeStartTime).Milliseconds()
			event := observer.Event{
				Type:        observer.EventTypeNodeFailed,
				ExecutionID: execState.ExecutionID,
				WorkflowID:  execState.WorkflowID,
				Timestamp:   time.Now(),
				Status:      "failed",
				NodeID:      &node.ID,
				NodeName:    &node.Name,
				NodeType:    &node.Type,
				Error:       err,
				DurationMs:  &nodeDuration,
			}
			de.observerManager.Notify(ctx, event)
		}

		return err
	}

	nodeEndTime := time.Now()
	// Store execution result with metadata
	execState.SetNodeOutput(node.ID, execResult.Output)
	execState.SetNodeInput(node.ID, execResult.Input)
	execState.SetNodeConfig(node.ID, execResult.Config)
	execState.SetNodeResolvedConfig(node.ID, execResult.ResolvedConfig)
	execState.SetNodeStatus(node.ID, models.NodeExecutionStatusCompleted)
	execState.SetNodeEndTime(node.ID, nodeEndTime)

	// Notify node completed
	if de.observerManager != nil {
		nodeDuration := time.Since(nodeStartTime).Milliseconds()
		event := observer.Event{
			Type:        observer.EventTypeNodeCompleted,
			ExecutionID: execState.ExecutionID,
			WorkflowID:  execState.WorkflowID,
			Timestamp:   time.Now(),
			Status:      "completed",
			NodeID:      &node.ID,
			NodeName:    &node.Name,
			NodeType:    &node.Type,
			DurationMs:  &nodeDuration,
		}

		// Add output if it's a map
		if outputMap, ok := execResult.Output.(map[string]interface{}); ok {
			event.Output = outputMap
		}

		de.observerManager.Notify(ctx, event)
	}

	return nil
}

// DAG represents workflow graph
type DAG struct {
	Nodes    map[string]*models.Node
	Edges    map[string][]string // nodeID -> []childNodeIDs
	InDegree map[string]int      // nodeID -> number of parents
}

// buildDAG builds DAG from workflow
func buildDAG(workflow *models.Workflow) *DAG {
	dag := &DAG{
		Nodes:    make(map[string]*models.Node),
		Edges:    make(map[string][]string),
		InDegree: make(map[string]int),
	}

	// Add nodes
	for _, node := range workflow.Nodes {
		dag.Nodes[node.ID] = node
		dag.InDegree[node.ID] = 0
	}

	// Add edges
	for _, edge := range workflow.Edges {
		dag.Edges[edge.From] = append(dag.Edges[edge.From], edge.To)
		dag.InDegree[edge.To]++
	}

	return dag
}

// topologicalSort performs topological sort using Kahn's algorithm
// and returns execution waves (nodes that can be executed in parallel)
func topologicalSort(dag *DAG) ([][]*models.Node, error) {
	// Copy in-degree map to avoid modifying original
	inDegree := make(map[string]int)
	for k, v := range dag.InDegree {
		inDegree[k] = v
	}

	waves := [][]*models.Node{}
	processed := 0

	for processed < len(dag.Nodes) {
		wave := []*models.Node{}

		// Find all nodes with in-degree 0
		for nodeID, degree := range inDegree {
			if degree == 0 {
				if node, ok := dag.Nodes[nodeID]; ok {
					wave = append(wave, node)
				}
			}
		}

		if len(wave) == 0 {
			// No nodes with in-degree 0 but graph not fully processed
			// This means there's a cycle
			return nil, fmt.Errorf("cycle detected in workflow graph")
		}

		// Process wave
		for _, node := range wave {
			delete(inDegree, node.ID)
			processed++

			// Decrease in-degree of children
			for _, childID := range dag.Edges[node.ID] {
				inDegree[childID]--
			}
		}

		waves = append(waves, wave)
	}

	return waves, nil
}

// getParentNodes returns parent nodes for a given node
func getParentNodes(workflow *models.Workflow, node *models.Node) []*models.Node {
	parents := []*models.Node{}

	for _, edge := range workflow.Edges {
		if edge.To == node.ID {
			// Find parent node by ID
			for _, n := range workflow.Nodes {
				if n.ID == edge.From {
					parents = append(parents, n)
					break
				}
			}
		}
	}

	return parents
}

// shouldExecuteNode checks if a node should be executed based on incoming edge conditions.
// Returns (shouldExecute, skipReason).
// A node is skipped if ANY incoming edge has a condition that evaluates to false.
func (de *DAGExecutor) shouldExecuteNode(
	execState *ExecutionState,
	node *models.Node,
) (bool, string) {
	workflow := execState.Workflow

	// Find all incoming edges to this node
	for _, edge := range workflow.Edges {
		if edge.To != node.ID {
			continue
		}

		// Find source node
		var sourceNode *models.Node
		for _, n := range workflow.Nodes {
			if n.ID == edge.From {
				sourceNode = n
				break
			}
		}

		if sourceNode == nil {
			continue
		}

		// Check if source node was skipped - if so, skip downstream nodes too
		sourceStatus, _ := execState.GetNodeStatus(sourceNode.ID)
		if sourceStatus == models.NodeExecutionStatusSkipped {
			return false, fmt.Sprintf("parent node %s was skipped", sourceNode.ID)
		}

		// Evaluate edge condition if present
		if edge.Condition != "" {
			passed, err := evaluateEdgeCondition(edge, execState, sourceNode)
			if err != nil {
				// On error, skip with message
				return false, fmt.Sprintf("edge condition error: %v", err)
			}
			if !passed {
				return false, fmt.Sprintf("edge condition '%s' is false", edge.Condition)
			}
		}

		// Check for sourceHandle-based routing from conditional nodes
		if sourceNode.Type == "conditional" && edge.SourceHandle != "" {
			passed, err := evaluateSourceHandleCondition(edge, execState, sourceNode)
			if err != nil {
				return false, fmt.Sprintf("sourceHandle evaluation error: %v", err)
			}
			if !passed {
				return false, fmt.Sprintf("conditional branch '%s' not active", edge.SourceHandle)
			}
		}
	}

	return true, ""
}

// evaluateEdgeCondition evaluates the condition expression on an edge.
// Returns true if the condition passes, false otherwise.
func evaluateEdgeCondition(
	edge *models.Edge,
	execState *ExecutionState,
	sourceNode *models.Node,
) (bool, error) {
	condition := edge.Condition
	if condition == "" {
		return true, nil // No condition = always pass
	}

	// Get output from source node
	output, _ := execState.GetNodeOutput(sourceNode.ID)

	// Prepare environment for expression evaluation
	env := map[string]interface{}{
		"output": output,
		"node":   sourceNode.ID,
	}

	// Compile and execute expression
	program, err := expr.Compile(condition, expr.Env(env), expr.AsBool())
	if err != nil {
		return false, fmt.Errorf("failed to compile edge condition: %w", err)
	}

	result, err := expr.Run(program, env)
	if err != nil {
		return false, fmt.Errorf("failed to evaluate edge condition: %w", err)
	}

	if boolResult, ok := result.(bool); ok {
		return boolResult, nil
	}

	return false, fmt.Errorf("edge condition must return boolean, got: %T", result)
}

// evaluateSourceHandleCondition checks if the edge's sourceHandle matches
// the output of a conditional node.
// For conditional nodes, output is typically a boolean (true/false).
func evaluateSourceHandleCondition(
	edge *models.Edge,
	execState *ExecutionState,
	sourceNode *models.Node,
) (bool, error) {
	// Get output from conditional node
	output, ok := execState.GetNodeOutput(sourceNode.ID)
	if !ok {
		return false, fmt.Errorf("conditional node %s has no output", sourceNode.ID)
	}

	// Conditional nodes return boolean
	if boolOutput, ok := output.(bool); ok {
		switch edge.SourceHandle {
		case "true":
			return boolOutput, nil
		case "false":
			return !boolOutput, nil
		default:
			// Unknown handle - let it pass
			return true, nil
		}
	}

	// If output is a map, check for "result" key
	if mapOutput, ok := output.(map[string]interface{}); ok {
		if result, exists := mapOutput["result"]; exists {
			if boolResult, ok := result.(bool); ok {
				switch edge.SourceHandle {
				case "true":
					return boolResult, nil
				case "false":
					return !boolResult, nil
				}
			}
		}
	}

	// Can't determine - default to pass
	return true, nil
}
