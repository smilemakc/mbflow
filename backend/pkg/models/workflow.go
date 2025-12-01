package models

import (
	"encoding/json"
	"fmt"
	"time"
)

// Workflow represents a complete workflow definition with its DAG structure.
type Workflow struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Description string                 `json:"description,omitempty"`
	Version     int                    `json:"version"`
	Status      WorkflowStatus         `json:"status"`
	Tags        []string               `json:"tags,omitempty"`
	Nodes       []*Node                `json:"nodes"`
	Edges       []*Edge                `json:"edges"`
	Variables   map[string]interface{} `json:"variables,omitempty"` // Workflow-level variables for template substitution
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
}

// WorkflowStatus represents the status of a workflow.
type WorkflowStatus string

const (
	WorkflowStatusDraft    WorkflowStatus = "draft"
	WorkflowStatusActive   WorkflowStatus = "active"
	WorkflowStatusInactive WorkflowStatus = "inactive"
	WorkflowStatusArchived WorkflowStatus = "archived"
)

// Node represents a single node in the workflow DAG.
type Node struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Type        string                 `json:"type"`
	Description string                 `json:"description,omitempty"`
	Config      map[string]interface{} `json:"config"`
	Position    *Position              `json:"position,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// Position represents the visual position of a node in the editor.
type Position struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

// Edge represents a directed edge between two nodes in the DAG.
type Edge struct {
	ID        string                 `json:"id"`
	From      string                 `json:"from"`
	To        string                 `json:"to"`
	Condition string                 `json:"condition,omitempty"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// Validate validates the workflow structure.
func (w *Workflow) Validate() error {
	if w.Name == "" {
		return &ValidationError{Field: "name", Message: "name is required"}
	}

	if len(w.Nodes) == 0 {
		return &ValidationError{Field: "nodes", Message: "at least one node is required"}
	}

	// Validate nodes
	nodeIDs := make(map[string]bool)
	for _, node := range w.Nodes {
		if err := node.Validate(); err != nil {
			return err
		}

		if nodeIDs[node.ID] {
			return &ValidationError{Field: "nodes", Message: fmt.Sprintf("duplicate node ID: %s", node.ID)}
		}
		nodeIDs[node.ID] = true
	}

	// Validate edges
	for _, edge := range w.Edges {
		if err := edge.Validate(); err != nil {
			return err
		}

		if !nodeIDs[edge.From] {
			return &ValidationError{Field: "edges", Message: fmt.Sprintf("edge references non-existent source node: %s", edge.From)}
		}

		if !nodeIDs[edge.To] {
			return &ValidationError{Field: "edges", Message: fmt.Sprintf("edge references non-existent target node: %s", edge.To)}
		}
	}

	return nil
}

// Validate validates the node structure.
func (n *Node) Validate() error {
	if n.ID == "" {
		return &ValidationError{Field: "id", Message: "node ID is required"}
	}

	if n.Name == "" {
		return &ValidationError{Field: "name", Message: "node name is required"}
	}

	if n.Type == "" {
		return &ValidationError{Field: "type", Message: "node type is required"}
	}

	return nil
}

// Validate validates the edge structure.
func (e *Edge) Validate() error {
	if e.ID == "" {
		return &ValidationError{Field: "id", Message: "edge ID is required"}
	}

	if e.From == "" {
		return &ValidationError{Field: "from", Message: "edge source is required"}
	}

	if e.To == "" {
		return &ValidationError{Field: "to", Message: "edge target is required"}
	}

	if e.From == e.To {
		return &ValidationError{Field: "edge", Message: "self-loop edges are not allowed"}
	}

	return nil
}

// GetNode returns a node by ID.
func (w *Workflow) GetNode(nodeID string) (*Node, error) {
	for _, node := range w.Nodes {
		if node.ID == nodeID {
			return node, nil
		}
	}
	return nil, ErrNodeNotFound
}

// GetEdge returns an edge by ID.
func (w *Workflow) GetEdge(edgeID string) (*Edge, error) {
	for _, edge := range w.Edges {
		if edge.ID == edgeID {
			return edge, nil
		}
	}
	return nil, ErrEdgeNotFound
}

// AddNode adds a node to the workflow.
func (w *Workflow) AddNode(node *Node) error {
	if err := node.Validate(); err != nil {
		return err
	}

	// Check for duplicate ID
	for _, n := range w.Nodes {
		if n.ID == node.ID {
			return &ValidationError{Field: "id", Message: "node ID already exists"}
		}
	}

	w.Nodes = append(w.Nodes, node)
	w.UpdatedAt = time.Now()
	return nil
}

// AddEdge adds an edge to the workflow.
func (w *Workflow) AddEdge(edge *Edge) error {
	if err := edge.Validate(); err != nil {
		return err
	}

	// Verify nodes exist
	if _, err := w.GetNode(edge.From); err != nil {
		return &ValidationError{Field: "from", Message: "source node does not exist"}
	}

	if _, err := w.GetNode(edge.To); err != nil {
		return &ValidationError{Field: "to", Message: "target node does not exist"}
	}

	// Check for duplicate ID
	for _, e := range w.Edges {
		if e.ID == edge.ID {
			return &ValidationError{Field: "id", Message: "edge ID already exists"}
		}
	}

	w.Edges = append(w.Edges, edge)
	w.UpdatedAt = time.Now()
	return nil
}

// RemoveNode removes a node from the workflow and its associated edges.
func (w *Workflow) RemoveNode(nodeID string) error {
	// Find and remove the node
	found := false
	for i, node := range w.Nodes {
		if node.ID == nodeID {
			w.Nodes = append(w.Nodes[:i], w.Nodes[i+1:]...)
			found = true
			break
		}
	}

	if !found {
		return ErrNodeNotFound
	}

	// Remove associated edges
	var edges []*Edge
	for _, edge := range w.Edges {
		if edge.From != nodeID && edge.To != nodeID {
			edges = append(edges, edge)
		}
	}
	w.Edges = edges

	w.UpdatedAt = time.Now()
	return nil
}

// RemoveEdge removes an edge from the workflow.
func (w *Workflow) RemoveEdge(edgeID string) error {
	for i, edge := range w.Edges {
		if edge.ID == edgeID {
			w.Edges = append(w.Edges[:i], w.Edges[i+1:]...)
			w.UpdatedAt = time.Now()
			return nil
		}
	}
	return ErrEdgeNotFound
}

// Clone creates a deep copy of the workflow.
func (w *Workflow) Clone() (*Workflow, error) {
	data, err := json.Marshal(w)
	if err != nil {
		return nil, err
	}

	var clone Workflow
	if err := json.Unmarshal(data, &clone); err != nil {
		return nil, err
	}

	return &clone, nil
}
