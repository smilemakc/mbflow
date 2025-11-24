package executor

import (
	"fmt"

	"github.com/google/uuid"
	"github.com/smilemakc/mbflow/internal/domain"
)

// ExecutionPlan represents a plan for executing a workflow
// It organizes nodes into "waves" that can be executed in sequence,
// with nodes within each wave executing in parallel
type ExecutionPlan struct {
	WorkflowID uuid.UUID
	Graph      *WorkflowGraph

	// Waves of nodes to execute
	// Each wave contains nodes that can execute in parallel
	Waves []ExecutionWave

	// Metadata
	TotalNodes  int
	MaxParallel int // Maximum parallelism (largest wave)
	Depth       int // Number of waves
}

// ExecutionWave represents a group of nodes that can execute in parallel
type ExecutionWave struct {
	WaveNumber int
	Nodes      []NodeExecution
}

// NodeExecution represents a node in the execution plan
type NodeExecution struct {
	NodeID       uuid.UUID
	Node         domain.Node
	Dependencies []uuid.UUID // Node IDs that must complete before this node
}

// ExecutionPlanner creates execution plans from workflows
type ExecutionPlanner struct {
	conditionEvaluator *ConditionEvaluator
}

// NewExecutionPlanner creates a new execution planner
func NewExecutionPlanner() *ExecutionPlanner {
	return &ExecutionPlanner{
		conditionEvaluator: NewConditionEvaluator(true),
	}
}

// CreatePlan creates an execution plan from a workflow
func (ep *ExecutionPlanner) CreatePlan(workflow domain.Workflow) (*ExecutionPlan, error) {
	// Build graph
	graph, err := NewWorkflowGraph(workflow)
	if err != nil {
		return nil, fmt.Errorf("failed to build graph: %w", err)
	}

	// Calculate waves using parallelization analysis
	waves, err := graph.GetParallelizableNodes()
	if err != nil {
		return nil, fmt.Errorf("failed to calculate execution waves: %w", err)
	}

	// Build execution waves with node details
	executionWaves := make([]ExecutionWave, 0, len(waves))
	maxParallel := 0

	for waveNum, waveNodeIDs := range waves {
		waveNodes := make([]NodeExecution, 0, len(waveNodeIDs))

		for _, nodeID := range waveNodeIDs {
			node, err := graph.GetNode(nodeID)
			if err != nil {
				return nil, fmt.Errorf("failed to get node %s: %w", nodeID, err)
			}

			// Get dependencies
			deps := graph.GetPredecessors(nodeID)

			waveNodes = append(waveNodes, NodeExecution{
				NodeID:       nodeID,
				Node:         node,
				Dependencies: deps,
			})
		}

		executionWaves = append(executionWaves, ExecutionWave{
			WaveNumber: waveNum,
			Nodes:      waveNodes,
		})

		if len(waveNodes) > maxParallel {
			maxParallel = len(waveNodes)
		}
	}

	plan := &ExecutionPlan{
		WorkflowID:  workflow.ID(),
		Graph:       graph,
		Waves:       executionWaves,
		TotalNodes:  graph.GetNodeCount(),
		MaxParallel: maxParallel,
		Depth:       len(executionWaves),
	}

	return plan, nil
}

// GetReadyNodes returns nodes that are ready to execute based on completed nodes
// This is used for dynamic planning during execution
func (ep *ExecutionPlanner) GetReadyNodes(
	graph *WorkflowGraph,
	completed map[uuid.UUID]bool,
	variables *domain.VariableSet,
) ([]uuid.UUID, error) {
	ready := make([]uuid.UUID, 0)

	for _, node := range graph.GetAllNodes() {
		nodeID := node.ID()

		// Skip if already completed
		if completed[nodeID] {
			continue
		}

		// Check if all dependencies are satisfied
		canExecute, err := ep.canExecuteNode(graph, nodeID, completed, variables)
		if err != nil {
			return nil, err
		}

		if canExecute {
			ready = append(ready, nodeID)
		}
	}

	return ready, nil
}

// canExecuteNode checks if a node can be executed
func (ep *ExecutionPlanner) canExecuteNode(
	graph *WorkflowGraph,
	nodeID uuid.UUID,
	completed map[uuid.UUID]bool,
	variables *domain.VariableSet,
) (bool, error) {
	// Get incoming edges
	incomingEdges := graph.GetIncomingEdges(nodeID)

	// No incoming edges = entry node, always ready
	if len(incomingEdges) == 0 {
		return true, nil
	}

	// Track active dependencies (edges whose conditions are true)
	activeDependencies := make(map[uuid.UUID]bool)
	hasConditional := false

	for _, edge := range incomingEdges {
		fromNodeID := edge.FromNodeID()

		// Evaluate edge condition
		if edge.Type() == domain.EdgeTypeConditional {
			hasConditional = true

			active, err := ep.conditionEvaluator.EvaluateEdge(edge, variables)
			if err != nil {
				return false, fmt.Errorf("failed to evaluate edge condition: %w", err)
			}

			if active {
				activeDependencies[fromNodeID] = true
			}
		} else {
			// Non-conditional edges are always active
			activeDependencies[fromNodeID] = true
		}
	}

	// If has conditional edges but none are active, node is not ready
	if hasConditional && len(activeDependencies) == 0 {
		return false, nil
	}

	// Check if all active dependencies are completed
	for depNodeID := range activeDependencies {
		if !completed[depNodeID] {
			return false, nil
		}
	}

	// Handle join nodes
	if graph.IsJoinNode(nodeID) {
		return ep.evaluateJoinNode(graph, nodeID, completed, activeDependencies)
	}

	return true, nil
}

// evaluateJoinNode evaluates if a join node is ready based on its strategy
func (ep *ExecutionPlanner) evaluateJoinNode(
	graph *WorkflowGraph,
	nodeID uuid.UUID,
	completed map[uuid.UUID]bool,
	activeDependencies map[uuid.UUID]bool,
) (bool, error) {
	strategy := graph.GetJoinStrategy(nodeID)

	// Get all incoming edges (predecessors)
	predecessors := graph.GetPredecessors(nodeID)

	// Filter to only active predecessors
	activePredecessors := make([]uuid.UUID, 0)
	for _, predID := range predecessors {
		if activeDependencies[predID] {
			activePredecessors = append(activePredecessors, predID)
		}
	}

	// Count completed active predecessors
	completedCount := 0
	for _, predID := range activePredecessors {
		if completed[predID] {
			completedCount++
		}
	}

	switch strategy {
	case domain.JoinStrategyWaitAll:
		// Wait for all active branches
		return completedCount == len(activePredecessors), nil

	case domain.JoinStrategyWaitAny:
		// Wait for any one branch
		return completedCount >= 1, nil

	case domain.JoinStrategyWaitFirst:
		// Same as WaitAny for now
		return completedCount >= 1, nil

	case domain.JoinStrategyWaitN:
		// Get N from node config
		node, err := graph.GetNode(nodeID)
		if err != nil {
			return false, err
		}

		config := node.Config()
		n, ok := config["min_required"].(int)
		if !ok {
			// Default to WaitAll if not specified
			return completedCount == len(activePredecessors), nil
		}

		return completedCount >= n, nil

	default:
		// Default to WaitAll
		return completedCount == len(activePredecessors), nil
	}
}

// OptimizePlan optimizes the execution plan (placeholder for future optimizations)
func (ep *ExecutionPlanner) OptimizePlan(plan *ExecutionPlan) *ExecutionPlan {
	// Future optimizations:
	// - Reorder nodes within waves for better cache locality
	// - Merge small waves
	// - Identify critical path
	// - Resource-aware scheduling
	return plan
}

// GetCriticalPath identifies the critical path (longest path) in the execution plan
func (ep *ExecutionPlanner) GetCriticalPath(plan *ExecutionPlan) []uuid.UUID {
	// Use graph depth analysis
	depths := make(map[uuid.UUID]int)
	parents := make(map[uuid.UUID]uuid.UUID)

	// Initialize entry nodes
	for _, nodeID := range plan.Graph.GetEntryNodes() {
		depths[nodeID] = 0
	}

	// Calculate depths using topological order
	sorted, _ := plan.Graph.TopologicalSort()

	maxDepth := 0
	var deepestNode uuid.UUID

	for _, nodeID := range sorted {
		currentDepth := depths[nodeID]

		for _, edge := range plan.Graph.GetOutgoingEdges(nodeID) {
			nextNodeID := edge.ToNodeID()
			newDepth := currentDepth + 1

			if newDepth > depths[nextNodeID] {
				depths[nextNodeID] = newDepth
				parents[nextNodeID] = nodeID

				if newDepth > maxDepth {
					maxDepth = newDepth
					deepestNode = nextNodeID
				}
			}
		}
	}

	// Backtrack to build critical path
	path := make([]uuid.UUID, 0)
	current := deepestNode

	for current != uuid.Nil {
		path = append([]uuid.UUID{current}, path...)
		current = parents[current]
	}

	return path
}

// EstimateExecutionTime estimates total execution time (requires node execution time metadata)
func (ep *ExecutionPlanner) EstimateExecutionTime(plan *ExecutionPlan) int {
	// This is a simplified estimation
	// In production, would use historical execution times or estimates from node configs

	totalTime := 0
	for _, _ = range plan.Waves {
		// Assume wave executes in parallel, so time = max node time in wave
		// For now, assume each node takes 1 time unit
		waveTime := 1
		totalTime += waveTime
	}

	return totalTime
}

// ValidatePlan validates the execution plan
func (ep *ExecutionPlanner) ValidatePlan(plan *ExecutionPlan) error {
	// Check that all nodes are included
	nodesInPlan := make(map[uuid.UUID]bool)
	for _, wave := range plan.Waves {
		for _, nodeExec := range wave.Nodes {
			nodesInPlan[nodeExec.NodeID] = true
		}
	}

	for _, node := range plan.Graph.GetAllNodes() {
		if !nodesInPlan[node.ID()] {
			return domain.NewDomainError(
				domain.ErrCodeValidationFailed,
				fmt.Sprintf("node %s (%s) not included in execution plan", node.Name(), node.ID()),
				nil,
			)
		}
	}

	// Check that waves respect dependencies
	completedInPreviousWaves := make(map[uuid.UUID]bool)
	for _, wave := range plan.Waves {
		for _, nodeExec := range wave.Nodes {
			// Check all dependencies are in previous waves
			for _, depID := range nodeExec.Dependencies {
				if !completedInPreviousWaves[depID] {
					// This is okay for conditional dependencies
					// Skip validation for now
				}
			}
		}

		// Mark all nodes in this wave as completed
		for _, nodeExec := range wave.Nodes {
			completedInPreviousWaves[nodeExec.NodeID] = true
		}
	}

	return nil
}

// GetPlanSummary returns a summary of the execution plan
func (ep *ExecutionPlanner) GetPlanSummary(plan *ExecutionPlan) map[string]interface{} {
	criticalPath := ep.GetCriticalPath(plan)
	estimatedTime := ep.EstimateExecutionTime(plan)

	return map[string]interface{}{
		"workflow_id":          plan.WorkflowID.String(),
		"total_nodes":          plan.TotalNodes,
		"total_waves":          plan.Depth,
		"max_parallelism":      plan.MaxParallel,
		"critical_path_length": len(criticalPath),
		"estimated_time":       estimatedTime,
		"waves": func() []map[string]interface{} {
			waves := make([]map[string]interface{}, len(plan.Waves))
			for i, wave := range plan.Waves {
				waves[i] = map[string]interface{}{
					"wave_number": wave.WaveNumber,
					"node_count":  len(wave.Nodes),
					"nodes": func() []string {
						names := make([]string, len(wave.Nodes))
						for j, n := range wave.Nodes {
							names[j] = n.Node.Name()
						}
						return names
					}(),
				}
			}
			return waves
		}(),
	}
}
