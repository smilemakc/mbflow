package engine

import (
	"fmt"

	"github.com/smilemakc/mbflow/pkg/models"
)

// DAG represents workflow graph with indexed lookups.
type DAG struct {
	Nodes    map[string]*models.Node
	Edges    map[string][]string // nodeID -> []childNodeIDs
	InDegree map[string]int      // nodeID -> number of parents
	Index    *DAGIndex           // Indexed lookups for O(1) access
}

// DAGIndex provides O(1) lookups for common operations.
type DAGIndex struct {
	ParentsByNode map[string][]*models.Node // nodeID -> parent nodes
	EdgesByTarget map[string][]*models.Edge // nodeID -> incoming edges
	EdgesBySource map[string][]*models.Edge // nodeID -> outgoing edges
	NodesByID     map[string]*models.Node   // nodeID -> node (fast lookup)
}

// BuildDAG builds a DAG from workflow with indexed lookups.
func BuildDAG(workflow *models.Workflow) *DAG {
	dag := &DAG{
		Nodes:    make(map[string]*models.Node),
		Edges:    make(map[string][]string),
		InDegree: make(map[string]int),
		Index: &DAGIndex{
			ParentsByNode: make(map[string][]*models.Node),
			EdgesByTarget: make(map[string][]*models.Edge),
			EdgesBySource: make(map[string][]*models.Edge),
			NodesByID:     make(map[string]*models.Node),
		},
	}

	for _, node := range workflow.Nodes {
		dag.Nodes[node.ID] = node
		dag.InDegree[node.ID] = 0
		dag.Index.NodesByID[node.ID] = node
		dag.Index.ParentsByNode[node.ID] = []*models.Node{}
	}

	for _, edge := range workflow.Edges {
		dag.Edges[edge.From] = append(dag.Edges[edge.From], edge.To)
		dag.InDegree[edge.To]++

		dag.Index.EdgesByTarget[edge.To] = append(dag.Index.EdgesByTarget[edge.To], edge)
		dag.Index.EdgesBySource[edge.From] = append(dag.Index.EdgesBySource[edge.From], edge)

		if parentNode := dag.Index.NodesByID[edge.From]; parentNode != nil {
			dag.Index.ParentsByNode[edge.To] = append(dag.Index.ParentsByNode[edge.To], parentNode)
		}
	}

	return dag
}

// TopologicalSort performs Kahn's algorithm and returns execution waves
// (groups of nodes that can be executed in parallel).
func TopologicalSort(dag *DAG) ([][]*models.Node, error) {
	inDegree := make(map[string]int)
	for k, v := range dag.InDegree {
		inDegree[k] = v
	}

	waves := [][]*models.Node{}
	processed := 0

	for processed < len(dag.Nodes) {
		wave := []*models.Node{}

		for nodeID, degree := range inDegree {
			if degree == 0 {
				if node, ok := dag.Nodes[nodeID]; ok {
					wave = append(wave, node)
				}
			}
		}

		if len(wave) == 0 {
			return nil, fmt.Errorf("cycle detected in workflow graph")
		}

		for _, node := range wave {
			delete(inDegree, node.ID)
			processed++

			for _, childID := range dag.Edges[node.ID] {
				inDegree[childID]--
			}
		}

		waves = append(waves, wave)
	}

	return waves, nil
}

// FlattenWaves converts wave-based topology to flat sequential order.
func FlattenWaves(waves [][]*models.Node) []string {
	var result []string
	for _, wave := range waves {
		for _, node := range wave {
			result = append(result, node.ID)
		}
	}
	return result
}

// FindLeafNodes finds nodes with no outgoing edges.
func FindLeafNodes(workflow *models.Workflow) []*models.Node {
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

// GetNodeByID returns a node by its ID.
func GetNodeByID(workflow *models.Workflow, nodeID string) *models.Node {
	for _, node := range workflow.Nodes {
		if node.ID == nodeID {
			return node
		}
	}
	return nil
}

// SortNodesByPriority sorts nodes by priority (higher priority first).
func SortNodesByPriority(nodes []*models.Node) []*models.Node {
	sorted := make([]*models.Node, len(nodes))
	copy(sorted, nodes)

	for i := 1; i < len(sorted); i++ {
		key := sorted[i]
		keyPriority := GetNodePriority(key)
		j := i - 1

		for j >= 0 && GetNodePriority(sorted[j]) < keyPriority {
			sorted[j+1] = sorted[j]
			j--
		}
		sorted[j+1] = key
	}

	return sorted
}
