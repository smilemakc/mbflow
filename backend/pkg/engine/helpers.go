package engine

import (
	"github.com/smilemakc/mbflow/pkg/models"
)

// FindNodeByID finds a node by ID in a slice of nodes.
func FindNodeByID(nodes []*models.Node, nodeID string) *models.Node {
	for _, node := range nodes {
		if node.ID == nodeID {
			return node
		}
	}
	return nil
}

// CollectIncomingEdges collects all edges that have the given node as target.
func CollectIncomingEdges(edges []*models.Edge, targetNodeID string) []*models.Edge {
	var incoming []*models.Edge
	for _, edge := range edges {
		if edge.To == targetNodeID {
			incoming = append(incoming, edge)
		}
	}
	return incoming
}

// CollectOutgoingEdges collects all edges that have the given node as source.
func CollectOutgoingEdges(edges []*models.Edge, sourceNodeID string) []*models.Edge {
	var outgoing []*models.Edge
	for _, edge := range edges {
		if edge.From == sourceNodeID {
			outgoing = append(outgoing, edge)
		}
	}
	return outgoing
}

// CollectRegularIncomingEdges collects all non-loop edges that have the given node as target.
func CollectRegularIncomingEdges(edges []*models.Edge, targetNodeID string) []*models.Edge {
	var incoming []*models.Edge
	for _, edge := range edges {
		if edge.To == targetNodeID && !edge.IsLoop() {
			incoming = append(incoming, edge)
		}
	}
	return incoming
}

// GetRegularParentNodes returns parent nodes connected via non-loop edges.
func GetRegularParentNodes(workflow *models.Workflow, node *models.Node) []*models.Node {
	parents := []*models.Node{}
	incomingEdges := CollectRegularIncomingEdges(workflow.Edges, node.ID)

	for _, edge := range incomingEdges {
		if parentNode := FindNodeByID(workflow.Nodes, edge.From); parentNode != nil {
			parents = append(parents, parentNode)
		}
	}

	return parents
}

// GetNodePriority extracts priority from node metadata, returns default if not found.
func GetNodePriority(node *models.Node) int {
	if node.Metadata == nil {
		return DefaultNodePriority
	}

	if priority, ok := node.Metadata["priority"]; ok {
		switch p := priority.(type) {
		case int:
			return p
		case float64:
			return int(p)
		case int64:
			return int(p)
		}
	}

	return DefaultNodePriority
}

// GetNodeTimeout extracts timeout from node config, returns 0 if not found.
func GetNodeTimeout(node *models.Node) int64 {
	if node.Config == nil {
		return 0
	}

	if timeout, ok := node.Config["timeout"]; ok {
		switch t := timeout.(type) {
		case int:
			return int64(t)
		case int64:
			return t
		case float64:
			return int64(t)
		}
	}

	return 0
}

// EstimateSize provides a rough estimate of memory size for an interface{}.
func EstimateSize(v interface{}) int64 {
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
			size += int64(len(k)) + EstimateSize(v)
		}
		return size
	case []interface{}:
		var size int64
		for _, item := range val {
			size += EstimateSize(item)
		}
		return size
	default:
		return 64
	}
}

// MergeVariables merges workflow and execution variables.
// Execution variables override workflow variables.
func MergeVariables(workflowVars, executionVars map[string]interface{}) map[string]interface{} {
	merged := make(map[string]interface{})
	for k, v := range workflowVars {
		merged[k] = v
	}
	for k, v := range executionVars {
		merged[k] = v
	}
	return merged
}

// GetParentNodes returns parent nodes for a given node.
func GetParentNodes(workflow *models.Workflow, node *models.Node) []*models.Node {
	parents := []*models.Node{}
	incomingEdges := CollectIncomingEdges(workflow.Edges, node.ID)

	for _, edge := range incomingEdges {
		if parentNode := FindNodeByID(workflow.Nodes, edge.From); parentNode != nil {
			parents = append(parents, parentNode)
		}
	}

	return parents
}

// PtrString returns a pointer to a string.
func PtrString(s string) *string {
	return &s
}
