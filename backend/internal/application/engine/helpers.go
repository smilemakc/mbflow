package engine

import (
	"encoding/json"

	"github.com/smilemakc/mbflow/pkg/models"
)

// findNodeByID finds a node by ID in a slice of nodes
func findNodeByID(nodes []*models.Node, nodeID string) *models.Node {
	for _, node := range nodes {
		if node.ID == nodeID {
			return node
		}
	}
	return nil
}

// collectIncomingEdges collects all edges that have the given node as target
func collectIncomingEdges(edges []*models.Edge, targetNodeID string) []*models.Edge {
	var incoming []*models.Edge
	for _, edge := range edges {
		if edge.To == targetNodeID {
			incoming = append(incoming, edge)
		}
	}
	return incoming
}

// collectOutgoingEdges collects all edges that have the given node as source
func collectOutgoingEdges(edges []*models.Edge, sourceNodeID string) []*models.Edge {
	var outgoing []*models.Edge
	for _, edge := range edges {
		if edge.From == sourceNodeID {
			outgoing = append(outgoing, edge)
		}
	}
	return outgoing
}

// getNodePriority extracts priority from node metadata, returns default if not found
func getNodePriority(node *models.Node) int {
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

// getNodeTimeout extracts timeout from node config, returns 0 if not found
func getNodeTimeout(node *models.Node) int64 {
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

// toMapInterface converts any value to map[string]interface{}.
// Fast path for already-map values, JSON roundtrip for structs.
func toMapInterface(v interface{}) map[string]interface{} {
	if v == nil {
		return nil
	}
	if m, ok := v.(map[string]interface{}); ok {
		return m
	}
	data, err := json.Marshal(v)
	if err != nil {
		return map[string]interface{}{"value": v}
	}
	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		return map[string]interface{}{"value": v}
	}
	return result
}
