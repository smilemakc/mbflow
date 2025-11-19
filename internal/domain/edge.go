package domain

// Edge is a domain entity that represents a connection between two nodes in a workflow.
// It defines the control flow and data transformation rules between workflow steps.
// Edges are immutable entities that are part of a Workflow aggregate.
type Edge struct {
	id         string
	workflowID string
	fromNodeID string
	toNodeID   string
	edgeType   string
	config     map[string]any
}

// NewEdge creates a new Edge instance.
func NewEdge(id, workflowID, fromNodeID, toNodeID, edgeType string, config map[string]any) *Edge {
	return &Edge{
		id:         id,
		workflowID: workflowID,
		fromNodeID: fromNodeID,
		toNodeID:   toNodeID,
		edgeType:   edgeType,
		config:     config,
	}
}

// ID returns the edge ID.
func (e *Edge) ID() string {
	return e.id
}

// WorkflowID returns the workflow ID this edge belongs to.
func (e *Edge) WorkflowID() string {
	return e.workflowID
}

// FromNodeID returns the source node ID.
func (e *Edge) FromNodeID() string {
	return e.fromNodeID
}

// ToNodeID returns the destination node ID.
func (e *Edge) ToNodeID() string {
	return e.toNodeID
}

// Type returns the type of the edge.
func (e *Edge) Type() string {
	return e.edgeType
}

// Config returns the configuration of the edge.
func (e *Edge) Config() map[string]any {
	return e.config
}
