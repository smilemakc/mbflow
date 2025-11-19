package executor

import (
	"fmt"
	"log"
	"strings"

	"github.com/expr-lang/expr"
)

// WorkflowGraph represents the structure of a workflow with nodes and edges.
// It provides methods for graph traversal and identifying parallel execution opportunities.
type WorkflowGraph struct {
	// nodes maps node ID to node configuration
	nodes map[string]*NodeConfig

	// edges maps from node ID to list of target node IDs
	forwardEdges map[string][]string

	// reverseEdges maps to node ID to list of source node IDs
	reverseEdges map[string][]string

	// nodeConfigs is the original list of node configs
	nodeConfigs []NodeConfig

	// edgeConfigs maps edge key (fromNodeID:toNodeID) to edge configuration
	edgeConfigs map[string]EdgeConfig
}

// EdgeConfig represents the configuration for an edge in the workflow graph.
type EdgeConfig struct {
	FromNodeID string
	ToNodeID   string
	EdgeType   string
	Config     map[string]any
}

// NewWorkflowGraph creates a new WorkflowGraph from nodes and edges.
func NewWorkflowGraph(nodes []NodeConfig, edges []EdgeConfig) *WorkflowGraph {
	graph := &WorkflowGraph{
		nodes:        make(map[string]*NodeConfig),
		forwardEdges: make(map[string][]string),
		reverseEdges: make(map[string][]string),
		nodeConfigs:  nodes,
		edgeConfigs:  make(map[string]EdgeConfig),
	}

	// Build node map
	for i := range nodes {
		graph.nodes[nodes[i].NodeID] = &nodes[i]
	}

	// Build edge maps and store edge configurations
	for _, edge := range edges {
		// Forward edge: from -> to
		graph.forwardEdges[edge.FromNodeID] = append(graph.forwardEdges[edge.FromNodeID], edge.ToNodeID)

		// Reverse edge: to <- from
		graph.reverseEdges[edge.ToNodeID] = append(graph.reverseEdges[edge.ToNodeID], edge.FromNodeID)

		// Store edge configuration with key "fromNodeID:toNodeID"
		edgeKey := fmt.Sprintf("%s:%s", edge.FromNodeID, edge.ToNodeID)
		graph.edgeConfigs[edgeKey] = edge
	}

	return graph
}

// GetNode returns the node configuration for a given node ID.
func (g *WorkflowGraph) GetNode(nodeID string) (*NodeConfig, bool) {
	node, ok := g.nodes[nodeID]
	return node, ok
}

// GetAllNodes returns all node configurations.
func (g *WorkflowGraph) GetAllNodes() []NodeConfig {
	return g.nodeConfigs
}

// GetNextNodes returns all nodes that can be reached from the given node.
func (g *WorkflowGraph) GetNextNodes(nodeID string) []string {
	return g.forwardEdges[nodeID]
}

// GetPreviousNodes returns all nodes that lead to the given node.
func (g *WorkflowGraph) GetPreviousNodes(nodeID string) []string {
	return g.reverseEdges[nodeID]
}

// GetEdgeConfig returns the edge configuration for a given from-to node pair.
func (g *WorkflowGraph) GetEdgeConfig(fromNodeID, toNodeID string) (*EdgeConfig, bool) {
	edgeKey := fmt.Sprintf("%s:%s", fromNodeID, toNodeID)
	edge, ok := g.edgeConfigs[edgeKey]
	if !ok {
		return nil, false
	}
	return &edge, true
}

// GetEntryNodes returns all nodes that have no incoming edges (entry points).
func (g *WorkflowGraph) GetEntryNodes() []string {
	var entryNodes []string
	for nodeID := range g.nodes {
		if len(g.reverseEdges[nodeID]) == 0 {
			entryNodes = append(entryNodes, nodeID)
		}
	}
	return entryNodes
}

// GetExitNodes returns all nodes that have no outgoing edges (exit points).
func (g *WorkflowGraph) GetExitNodes() []string {
	var exitNodes []string
	for nodeID := range g.nodes {
		if len(g.forwardEdges[nodeID]) == 0 {
			exitNodes = append(exitNodes, nodeID)
		}
	}
	return exitNodes
}

// IsJoinNode checks if a node is a join node (has multiple incoming edges).
func (g *WorkflowGraph) IsJoinNode(nodeID string) bool {
	return len(g.reverseEdges[nodeID]) > 1
}

// IsForkNode checks if a node is a fork node (has multiple outgoing edges).
func (g *WorkflowGraph) IsForkNode(nodeID string) bool {
	return len(g.forwardEdges[nodeID]) > 1
}

// GetParallelBranches returns all nodes that share the same parent node (can execute in parallel).
// This identifies nodes that can be executed concurrently after a fork.
func (g *WorkflowGraph) GetParallelBranches(parentNodeID string) []string {
	return g.forwardEdges[parentNodeID]
}

// normalizeStringValues recursively normalizes string values in a map by trimming whitespace.
// This ensures string comparisons in conditions work correctly even if values have trailing whitespace.
func normalizeStringValues(data interface{}) interface{} {
	switch v := data.(type) {
	case string:
		return strings.TrimSpace(v)
	case map[string]interface{}:
		normalized := make(map[string]interface{})
		for k, val := range v {
			normalized[k] = normalizeStringValues(val)
		}
		return normalized
	case []interface{}:
		normalized := make([]interface{}, len(v))
		for i, val := range v {
			normalized[i] = normalizeStringValues(val)
		}
		return normalized
	default:
		return v
	}
}

// evaluateCondition evaluates a condition expression using the expr library.
// The condition can access variables from the variables map, including nested fields using dot notation.
// String values are normalized (trimmed) before evaluation to handle whitespace issues.
// Returns true if condition is true, false if false, and error if condition is invalid or variable is missing.
func evaluateCondition(condition string, variables map[string]interface{}) (bool, error) {
	if condition == "" {
		return false, fmt.Errorf("condition is empty")
	}

	// Normalize string values in variables to handle whitespace issues
	normalizedVars := normalizeStringValues(variables).(map[string]interface{})

	// For expr library, we need to pass variables in a way that allows direct access
	// We'll create a wrapper that makes map keys accessible as variables
	// Compile with a map type environment
	envType := map[string]interface{}{}

	// Compile the expression - expr will allow accessing map keys directly
	program, err := expr.Compile(condition, expr.Env(envType), expr.AsBool())
	if err != nil {
		// If compilation fails, try without Env (allows dynamic variables)
		program, err = expr.Compile(condition, expr.AsBool())
		if err != nil {
			return false, fmt.Errorf("failed to compile condition '%s': %w", condition, err)
		}
	}

	// Run the compiled program with actual variable values
	// expr library will make map keys accessible as variables when passed to Run()
	result, err := expr.Run(program, normalizedVars)
	if err != nil {
		// Provide more detailed error message with variable values
		var varInfo []string
		for k, v := range normalizedVars {
			if strVal, ok := v.(string); ok && len(strVal) < 100 {
				varInfo = append(varInfo, fmt.Sprintf("%s=%q", k, strVal))
			}
		}
		if len(varInfo) > 0 {
			return false, fmt.Errorf("failed to evaluate condition '%s' with variables [%s]: %w", condition, strings.Join(varInfo, ", "), err)
		}
		return false, fmt.Errorf("failed to evaluate condition '%s': %w", condition, err)
	}

	// Convert result to boolean (expr.AsBool() should ensure this, but check anyway)
	resultBool, ok := result.(bool)
	if !ok {
		return false, fmt.Errorf("condition '%s' did not return a boolean value, got %T", condition, result)
	}

	return resultBool, nil
}

// GetReadyNodes returns all nodes that are ready to execute (all dependencies completed).
// For conditional edges, only active edges (where condition is true) are considered as dependencies.
// Returns error if any conditional edge has an invalid condition or missing variables.
func (g *WorkflowGraph) GetReadyNodes(completedNodes map[string]bool, execCtx *ExecutionContext) ([]string, error) {
	var readyNodes []string

	// Get all variables for condition evaluation
	variables := execCtx.GetAllVariables()

	// Track condition evaluation results to avoid excessive logging
	conditionCache := make(map[string]bool) // key: "fromNodeID:toNodeID:condition"

	for nodeID := range g.nodes {
		// Skip if already completed
		if completedNodes[nodeID] {
			continue
		}

		// Check if all active dependencies are completed
		dependencies := g.reverseEdges[nodeID]
		activeDependencies := make([]string, 0) // Dependencies that are actually active (conditions satisfied)

		for _, depNodeID := range dependencies {
			// Check if this is a conditional edge
			edgeConfig, ok := g.GetEdgeConfig(depNodeID, nodeID)
			if ok && edgeConfig.EdgeType == "conditional" {
				// Parse conditional edge configuration
				conditionalConfig, err := parseConfig[ConditionalEdgeConfig](edgeConfig.Config)
				if err != nil {
					return nil, fmt.Errorf("failed to parse conditional edge config from node '%s' to node '%s': %w", depNodeID, nodeID, err)
				}

				// Validate condition
				if conditionalConfig.Condition == "" {
					return nil, fmt.Errorf("conditional edge from node '%s' to node '%s' has no condition", depNodeID, nodeID)
				}

				// Check cache first to avoid re-evaluation
				cacheKey := fmt.Sprintf("%s:%s:%s", depNodeID, nodeID, conditionalConfig.Condition)
				conditionResult, cached := conditionCache[cacheKey]

				if !cached {
					// Evaluate condition
					var err error
					conditionResult, err = evaluateCondition(conditionalConfig.Condition, variables)
					if err != nil {
						return nil, fmt.Errorf("failed to evaluate condition '%s' for edge from node '%s' to node '%s': %w", conditionalConfig.Condition, depNodeID, nodeID, err)
					}

					// Cache the result
					conditionCache[cacheKey] = conditionResult

					// Log condition evaluation result only once per unique condition
					log.Printf("[WorkflowGraph] Condition evaluation: condition='%s' from_node='%s' to_node='%s' result=%v", conditionalConfig.Condition, depNodeID, nodeID, conditionResult)
					if !conditionResult {
						// Log variable values that might be relevant to the condition (only once)
						var relevantVars []string
						for k, v := range variables {
							if strVal, ok := v.(string); ok && len(strVal) < 100 {
								relevantVars = append(relevantVars, fmt.Sprintf("%s=%q", k, strVal))
							}
						}
						if len(relevantVars) > 0 {
							log.Printf("[WorkflowGraph] Condition false - current variables: %s", strings.Join(relevantVars, ", "))
						}
					}
				}

				// If condition is true, this edge is active and dependency must be completed
				if conditionResult {
					activeDependencies = append(activeDependencies, depNodeID)
				}
				// If condition is false, this edge is not active - ignore it
			} else {
				// For direct edges (and other non-conditional types), always consider as active dependency
				activeDependencies = append(activeDependencies, depNodeID)
			}
		}

		// If there are no active dependencies, node is ready (entry node or all conditional edges inactive)
		if len(activeDependencies) == 0 {
			readyNodes = append(readyNodes, nodeID)
			continue
		}

		// Check if all active dependencies are completed
		allActiveDepsCompleted := true
		for _, depNodeID := range activeDependencies {
			if !completedNodes[depNodeID] {
				allActiveDepsCompleted = false
				break
			}
		}

		// If all active dependencies are completed, node is ready
		if allActiveDepsCompleted {
			readyNodes = append(readyNodes, nodeID)
		}
	}

	return readyNodes, nil
}

// HasCycles checks if the graph contains cycles using DFS.
func (g *WorkflowGraph) HasCycles() bool {
	visited := make(map[string]bool)
	recStack := make(map[string]bool)

	for nodeID := range g.nodes {
		if !visited[nodeID] {
			if g.hasCyclesDFS(nodeID, visited, recStack) {
				return true
			}
		}
	}

	return false
}

// hasCyclesDFS performs DFS to detect cycles.
func (g *WorkflowGraph) hasCyclesDFS(nodeID string, visited, recStack map[string]bool) bool {
	visited[nodeID] = true
	recStack[nodeID] = true

	for _, nextNodeID := range g.forwardEdges[nodeID] {
		if !visited[nextNodeID] {
			if g.hasCyclesDFS(nextNodeID, visited, recStack) {
				return true
			}
		} else if recStack[nextNodeID] {
			return true
		}
	}

	recStack[nodeID] = false
	return false
}

// TopologicalSort returns nodes in topological order (if no cycles exist).
func (g *WorkflowGraph) TopologicalSort() ([]string, error) {
	if g.HasCycles() {
		return nil, fmt.Errorf("graph contains cycles, cannot perform topological sort")
	}

	// Calculate in-degree for each node
	inDegree := make(map[string]int)
	for nodeID := range g.nodes {
		inDegree[nodeID] = len(g.reverseEdges[nodeID])
	}

	// Find all nodes with in-degree 0 (entry nodes)
	queue := make([]string, 0)
	for nodeID, degree := range inDegree {
		if degree == 0 {
			queue = append(queue, nodeID)
		}
	}

	result := make([]string, 0)

	// Process nodes
	for len(queue) > 0 {
		// Get node from queue
		nodeID := queue[0]
		queue = queue[1:]
		result = append(result, nodeID)

		// Reduce in-degree of adjacent nodes
		for _, nextNodeID := range g.forwardEdges[nodeID] {
			inDegree[nextNodeID]--
			if inDegree[nextNodeID] == 0 {
				queue = append(queue, nextNodeID)
			}
		}
	}

	return result, nil
}
