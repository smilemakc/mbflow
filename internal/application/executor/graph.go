package executor

import (
	"fmt"

	"github.com/google/uuid"
	"github.com/smilemakc/mbflow/internal/domain"
)

// WorkflowGraph represents a directed graph built from a workflow's nodes and edges.
// It provides efficient graph traversal and analysis operations.
type WorkflowGraph struct {
	workflowID uuid.UUID

	// Node storage
	nodes    map[uuid.UUID]domain.Node
	nodeList []domain.Node

	// Edge storage
	edges        []domain.Edge
	forwardEdges map[uuid.UUID][]domain.Edge // from node ID -> outgoing edges
	reverseEdges map[uuid.UUID][]domain.Edge // to node ID -> incoming edges

	// Cached analysis results
	entryNodes []uuid.UUID
	exitNodes  []uuid.UUID
	hasCycles  bool
	validated  bool
}

// NewWorkflowGraph creates a new WorkflowGraph from a Workflow aggregate
func NewWorkflowGraph(workflow domain.Workflow) (*WorkflowGraph, error) {
	nodes := workflow.GetAllNodes()
	edges := workflow.GetAllEdges()

	if len(nodes) == 0 {
		return nil, domain.NewDomainError(
			domain.ErrCodeValidationFailed,
			"workflow has no nodes",
			nil,
		)
	}

	graph := &WorkflowGraph{
		workflowID:   workflow.ID(),
		nodes:        make(map[uuid.UUID]domain.Node),
		nodeList:     nodes,
		edges:        edges,
		forwardEdges: make(map[uuid.UUID][]domain.Edge),
		reverseEdges: make(map[uuid.UUID][]domain.Edge),
	}

	// Build node map
	for _, node := range nodes {
		graph.nodes[node.ID()] = node
	}

	// Build edge maps
	for _, edge := range edges {
		graph.forwardEdges[edge.FromNodeID()] = append(
			graph.forwardEdges[edge.FromNodeID()],
			edge,
		)
		graph.reverseEdges[edge.ToNodeID()] = append(
			graph.reverseEdges[edge.ToNodeID()],
			edge,
		)
	}

	// Perform validation
	if err := graph.Validate(); err != nil {
		return nil, err
	}

	return graph, nil
}

// Validate validates the graph structure
func (g *WorkflowGraph) Validate() error {
	if g.validated {
		return nil
	}

	// Check for orphaned nodes (nodes with no incoming or outgoing edges)
	// Allow entry and exit nodes, but warn about completely isolated nodes
	for nodeID, node := range g.nodes {
		hasIncoming := len(g.reverseEdges[nodeID]) > 0
		hasOutgoing := len(g.forwardEdges[nodeID]) > 0

		if !hasIncoming && !hasOutgoing && len(g.nodes) > 1 {
			return domain.NewDomainError(
				domain.ErrCodeValidationFailed,
				fmt.Sprintf("node '%s' (%s) is isolated (no edges)", node.Name(), nodeID),
				nil,
			)
		}
	}

	// Check for cycles
	if g.detectCycles() {
		g.hasCycles = true
		return domain.NewDomainError(
			domain.ErrCodeCyclicDependency,
			"workflow graph contains cycles",
			nil,
		)
	}

	// Validate edge data sources (include_outputs_from configuration)
	for _, edge := range g.edges {
		if err := g.ValidateEdgeDataSources(edge); err != nil {
			return err
		}
	}

	// Identify entry and exit nodes
	g.entryNodes = g.findEntryNodes()
	g.exitNodes = g.findExitNodes()

	if len(g.entryNodes) == 0 {
		return domain.NewDomainError(
			domain.ErrCodeValidationFailed,
			"workflow has no entry nodes (all nodes have incoming edges)",
			nil,
		)
	}

	if len(g.exitNodes) == 0 {
		return domain.NewDomainError(
			domain.ErrCodeValidationFailed,
			"workflow has no exit nodes (all nodes have outgoing edges)",
			nil,
		)
	}

	g.validated = true
	return nil
}

// GetNode returns a node by ID
func (g *WorkflowGraph) GetNode(nodeID uuid.UUID) (domain.Node, error) {
	node, exists := g.nodes[nodeID]
	if !exists {
		return nil, domain.NewDomainError(
			domain.ErrCodeNotFound,
			fmt.Sprintf("node %s not found in graph", nodeID),
			nil,
		)
	}
	return node, nil
}

// GetAllNodes returns all nodes
func (g *WorkflowGraph) GetAllNodes() []domain.Node {
	return g.nodeList
}

// GetNodeCount returns the number of nodes
func (g *WorkflowGraph) GetNodeCount() int {
	return len(g.nodes)
}

// GetOutgoingEdges returns all outgoing edges from a node
func (g *WorkflowGraph) GetOutgoingEdges(nodeID uuid.UUID) []domain.Edge {
	edges, exists := g.forwardEdges[nodeID]
	if !exists {
		return []domain.Edge{}
	}
	return edges
}

// GetIncomingEdges returns all incoming edges to a node
func (g *WorkflowGraph) GetIncomingEdges(nodeID uuid.UUID) []domain.Edge {
	edges, exists := g.reverseEdges[nodeID]
	if !exists {
		return []domain.Edge{}
	}
	return edges
}

// GetSuccessors returns IDs of nodes that can be reached from the given node
func (g *WorkflowGraph) GetSuccessors(nodeID uuid.UUID) []uuid.UUID {
	edges := g.GetOutgoingEdges(nodeID)
	successors := make([]uuid.UUID, 0, len(edges))
	for _, edge := range edges {
		successors = append(successors, edge.ToNodeID())
	}
	return successors
}

// GetPredecessors returns IDs of nodes that lead to the given node
func (g *WorkflowGraph) GetPredecessors(nodeID uuid.UUID) []uuid.UUID {
	edges := g.GetIncomingEdges(nodeID)
	predecessors := make([]uuid.UUID, 0, len(edges))
	for _, edge := range edges {
		predecessors = append(predecessors, edge.FromNodeID())
	}
	return predecessors
}

// GetEntryNodes returns all entry nodes (no incoming edges)
func (g *WorkflowGraph) GetEntryNodes() []uuid.UUID {
	if g.entryNodes != nil {
		return g.entryNodes
	}
	return g.findEntryNodes()
}

// GetExitNodes returns all exit nodes (no outgoing edges)
func (g *WorkflowGraph) GetExitNodes() []uuid.UUID {
	if g.exitNodes != nil {
		return g.exitNodes
	}
	return g.findExitNodes()
}

// findEntryNodes identifies nodes with no incoming edges
func (g *WorkflowGraph) findEntryNodes() []uuid.UUID {
	var entries []uuid.UUID
	for nodeID := range g.nodes {
		if len(g.reverseEdges[nodeID]) == 0 {
			entries = append(entries, nodeID)
		}
	}
	return entries
}

// findExitNodes identifies nodes with no outgoing edges
func (g *WorkflowGraph) findExitNodes() []uuid.UUID {
	var exits []uuid.UUID
	for nodeID := range g.nodes {
		if len(g.forwardEdges[nodeID]) == 0 {
			exits = append(exits, nodeID)
		}
	}
	return exits
}

// IsJoinNode checks if a node is a join point (multiple incoming edges)
func (g *WorkflowGraph) IsJoinNode(nodeID uuid.UUID) bool {
	return len(g.reverseEdges[nodeID]) > 1
}

// IsForkNode checks if a node is a fork point (multiple outgoing edges)
func (g *WorkflowGraph) IsForkNode(nodeID uuid.UUID) bool {
	return len(g.forwardEdges[nodeID]) > 1
}

// GetJoinStrategy returns the join strategy for a join node
func (g *WorkflowGraph) GetJoinStrategy(nodeID uuid.UUID) domain.JoinStrategy {
	node, exists := g.nodes[nodeID]
	if !exists {
		return domain.JoinStrategyWaitAll // Default
	}

	// Check node config for join strategy
	config := node.Config()
	if strategyStr, ok := config["join_strategy"].(string); ok {
		return domain.JoinStrategy(strategyStr)
	}

	// Default: wait for all incoming branches
	return domain.JoinStrategyWaitAll
}

// detectCycles performs cycle detection using DFS
func (g *WorkflowGraph) detectCycles() bool {
	visited := make(map[uuid.UUID]bool)
	recStack := make(map[uuid.UUID]bool)

	for nodeID := range g.nodes {
		if !visited[nodeID] {
			if g.hasCycleDFS(nodeID, visited, recStack) {
				return true
			}
		}
	}

	return false
}

// hasCycleDFS performs DFS-based cycle detection
func (g *WorkflowGraph) hasCycleDFS(nodeID uuid.UUID, visited, recStack map[uuid.UUID]bool) bool {
	visited[nodeID] = true
	recStack[nodeID] = true

	for _, edge := range g.forwardEdges[nodeID] {
		nextNodeID := edge.ToNodeID()
		if !visited[nextNodeID] {
			if g.hasCycleDFS(nextNodeID, visited, recStack) {
				return true
			}
		} else if recStack[nextNodeID] {
			return true
		}
	}

	recStack[nodeID] = false
	return false
}

// TopologicalSort returns nodes in topological order
func (g *WorkflowGraph) TopologicalSort() ([]uuid.UUID, error) {
	if g.hasCycles {
		return nil, domain.NewDomainError(
			domain.ErrCodeCyclicDependency,
			"cannot perform topological sort on graph with cycles",
			nil,
		)
	}

	// Kahn's algorithm
	inDegree := make(map[uuid.UUID]int)
	for nodeID := range g.nodes {
		inDegree[nodeID] = len(g.reverseEdges[nodeID])
	}

	queue := make([]uuid.UUID, 0)
	for nodeID, degree := range inDegree {
		if degree == 0 {
			queue = append(queue, nodeID)
		}
	}

	result := make([]uuid.UUID, 0, len(g.nodes))

	for len(queue) > 0 {
		nodeID := queue[0]
		queue = queue[1:]
		result = append(result, nodeID)

		for _, edge := range g.forwardEdges[nodeID] {
			nextNodeID := edge.ToNodeID()
			inDegree[nextNodeID]--
			if inDegree[nextNodeID] == 0 {
				queue = append(queue, nextNodeID)
			}
		}
	}

	if len(result) != len(g.nodes) {
		return nil, domain.NewDomainError(
			domain.ErrCodeCyclicDependency,
			"topological sort failed - graph may have cycles",
			nil,
		)
	}

	return result, nil
}

// GetParallelizableNodes identifies nodes that can be executed in parallel
// Returns a slice of "waves" where each wave contains nodes that can execute concurrently
func (g *WorkflowGraph) GetParallelizableNodes() ([][]uuid.UUID, error) {
	// Use topological sort to get execution order
	sorted, err := g.TopologicalSort()
	if err != nil {
		return nil, err
	}

	// Group nodes into waves based on their depth in the graph
	waves := make([][]uuid.UUID, 0)
	processed := make(map[uuid.UUID]bool)

	for len(processed) < len(g.nodes) {
		wave := make([]uuid.UUID, 0)

		// Find nodes whose dependencies are all processed
		for _, nodeID := range sorted {
			if processed[nodeID] {
				continue
			}

			// Check if all predecessors are processed
			predecessors := g.GetPredecessors(nodeID)
			allProcessed := true
			for _, predID := range predecessors {
				if !processed[predID] {
					allProcessed = false
					break
				}
			}

			if allProcessed {
				wave = append(wave, nodeID)
			}
		}

		if len(wave) == 0 {
			break // Should not happen if graph is valid
		}

		waves = append(waves, wave)

		// Mark wave nodes as processed
		for _, nodeID := range wave {
			processed[nodeID] = true
		}
	}

	return waves, nil
}

// GetDepth returns the maximum depth of the graph (longest path from entry to exit)
func (g *WorkflowGraph) GetDepth() int {
	depths := make(map[uuid.UUID]int)

	// Initialize entry nodes with depth 0
	for _, nodeID := range g.GetEntryNodes() {
		depths[nodeID] = 0
	}

	// Topological order ensures we process nodes after their dependencies
	sorted, err := g.TopologicalSort()
	if err != nil {
		return 0
	}

	maxDepth := 0
	for _, nodeID := range sorted {
		currentDepth := depths[nodeID]

		// Update depth for successors
		for _, edge := range g.forwardEdges[nodeID] {
			nextNodeID := edge.ToNodeID()
			newDepth := currentDepth + 1
			if newDepth > depths[nextNodeID] {
				depths[nextNodeID] = newDepth
			}
			if newDepth > maxDepth {
				maxDepth = newDepth
			}
		}
	}

	return maxDepth
}

// GetNodeByName returns a node by its name
func (g *WorkflowGraph) GetNodeByName(name string) (domain.Node, error) {
	for _, node := range g.nodes {
		if node.Name() == name {
			return node, nil
		}
	}
	return nil, domain.NewDomainError(
		domain.ErrCodeNotFound,
		fmt.Sprintf("node '%s' not found in graph", name),
		nil,
	)
}

// IsAncestor checks if ancestorID is an ancestor of descendantID in the graph.
// Uses BFS traversal to check if descendantID is reachable from ancestorID.
// Returns false if nodes are the same or if no path exists.
func (g *WorkflowGraph) IsAncestor(ancestorID, descendantID uuid.UUID) bool {
	// Self-reference check
	if ancestorID == descendantID {
		return false
	}

	// BFS to check reachability
	visited := make(map[uuid.UUID]bool)
	queue := []uuid.UUID{ancestorID}
	visited[ancestorID] = true

	for len(queue) > 0 {
		currentID := queue[0]
		queue = queue[1:]

		// Check all successors
		for _, edge := range g.forwardEdges[currentID] {
			nextID := edge.ToNodeID()

			// Found path to descendant
			if nextID == descendantID {
				return true
			}

			// Continue BFS if not visited
			if !visited[nextID] {
				visited[nextID] = true
				queue = append(queue, nextID)
			}
		}
	}

	// No path found
	return false
}

// ValidateEdgeDataSources validates that nodes referenced in edge's include_outputs_from
// configuration exist and are ancestors of the target node.
func (g *WorkflowGraph) ValidateEdgeDataSources(edge domain.Edge) error {
	config := edge.Config()
	if config == nil {
		return nil
	}

	// Check if include_outputs_from is present
	includeOutputsFrom, ok := config["include_outputs_from"]
	if !ok {
		return nil // No additional sources specified
	}

	// Convert to string slice
	var nodeNames []string
	switch v := includeOutputsFrom.(type) {
	case []string:
		nodeNames = v
	case []interface{}:
		nodeNames = make([]string, len(v))
		for i, item := range v {
			str, ok := item.(string)
			if !ok {
				return domain.NewDomainError(
					domain.ErrCodeValidationFailed,
					fmt.Sprintf("include_outputs_from contains non-string value at index %d", i),
					nil,
				)
			}
			nodeNames[i] = str
		}
	default:
		return domain.NewDomainError(
			domain.ErrCodeValidationFailed,
			"include_outputs_from must be a string array",
			nil,
		)
	}

	// Validate each referenced node
	targetNodeID := edge.ToNodeID()
	for _, nodeName := range nodeNames {
		// Check node exists
		sourceNode, err := g.GetNodeByName(nodeName)
		if err != nil {
			return domain.NewDomainError(
				domain.ErrCodeValidationFailed,
				fmt.Sprintf("node '%s' in include_outputs_from not found in workflow", nodeName),
				nil,
			)
		}

		sourceNodeID := sourceNode.ID()

		// Check for self-reference
		if sourceNodeID == targetNodeID {
			return domain.NewDomainError(
				domain.ErrCodeValidationFailed,
				fmt.Sprintf("node '%s' cannot include outputs from itself", nodeName),
				nil,
			)
		}

		// Check that source is an ancestor of target
		if !g.IsAncestor(sourceNodeID, targetNodeID) {
			targetNode, _ := g.GetNode(targetNodeID)
			return domain.NewDomainError(
				domain.ErrCodeValidationFailed,
				fmt.Sprintf("node '%s' in include_outputs_from is not an ancestor of target node '%s'",
					nodeName, targetNode.Name()),
				nil,
			)
		}
	}

	return nil
}

// Clone creates a copy of the graph
func (g *WorkflowGraph) Clone() *WorkflowGraph {
	clone := &WorkflowGraph{
		workflowID:   g.workflowID,
		nodes:        make(map[uuid.UUID]domain.Node, len(g.nodes)),
		nodeList:     make([]domain.Node, len(g.nodeList)),
		edges:        make([]domain.Edge, len(g.edges)),
		forwardEdges: make(map[uuid.UUID][]domain.Edge),
		reverseEdges: make(map[uuid.UUID][]domain.Edge),
		hasCycles:    g.hasCycles,
		validated:    g.validated,
	}

	// Copy nodes
	for k, v := range g.nodes {
		clone.nodes[k] = v
	}
	copy(clone.nodeList, g.nodeList)

	// Copy edges
	copy(clone.edges, g.edges)
	for k, v := range g.forwardEdges {
		clone.forwardEdges[k] = append([]domain.Edge{}, v...)
	}
	for k, v := range g.reverseEdges {
		clone.reverseEdges[k] = append([]domain.Edge{}, v...)
	}

	// Copy entry/exit nodes
	if g.entryNodes != nil {
		clone.entryNodes = append([]uuid.UUID{}, g.entryNodes...)
	}
	if g.exitNodes != nil {
		clone.exitNodes = append([]uuid.UUID{}, g.exitNodes...)
	}

	return clone
}
