package engine

import (
	"context"
	"fmt"
	"sync"
	"time"

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
	result, err := de.nodeExecutor.Execute(ctx, nodeCtx)

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
	// Store result and mark as completed
	execState.SetNodeOutput(node.ID, result)
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
		if outputMap, ok := result.(map[string]interface{}); ok {
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
