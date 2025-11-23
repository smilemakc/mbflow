package mbflow

import (
	"github.com/google/uuid"
)

// RelationshipBuilder provides a fluent interface for building edges between workflow nodes.
// It simplifies the creation of edges by providing type-safe methods for different edge types.
//
// Example usage:
//
//	edges := NewRelationshipBuilder(workflowID).
//	    Direct(startNode, processNode).
//	    Parallel(processNode, checkNode).
//	    Parallel(processNode, validateNode).
//	    Join(checkNode, mergeNode).
//	    Join(validateNode, mergeNode).
//	    Conditional(mergeNode, endNode, "status == 'success'").
//	    Build()
type RelationshipBuilder struct {
	workflowID string
	edges      []Edge
}

// NewRelationshipBuilder creates a new RelationshipBuilder for the specified workflow.
func NewRelationshipBuilder(workflowID string) *RelationshipBuilder {
	return &RelationshipBuilder{
		workflowID: workflowID,
		edges:      make([]Edge, 0),
	}
}

// Direct adds a direct edge from one node to another.
// Direct edges are the default sequential flow between nodes.
func (rb *RelationshipBuilder) Direct(from, to Node) *RelationshipBuilder {
	edge := NewEdge(
		uuid.NewString(),
		rb.workflowID,
		from.ID(),
		to.ID(),
		"direct",
		map[string]any{},
	)
	rb.edges = append(rb.edges, edge)
	return rb
}

// Parallel adds a parallel edge from one node to another.
// Parallel edges allow multiple nodes to execute concurrently.
func (rb *RelationshipBuilder) Parallel(from, to Node) *RelationshipBuilder {
	edge := NewEdge(
		uuid.NewString(),
		rb.workflowID,
		from.ID(),
		to.ID(),
		"parallel",
		map[string]any{},
	)
	rb.edges = append(rb.edges, edge)
	return rb
}

// Join adds a join edge from one node to another.
// Join edges are used to synchronize parallel execution branches.
// The target node waits for all incoming join edges to complete before executing.
func (rb *RelationshipBuilder) Join(from, to Node) *RelationshipBuilder {
	edge := NewEdge(
		uuid.NewString(),
		rb.workflowID,
		from.ID(),
		to.ID(),
		"join",
		map[string]any{},
	)
	rb.edges = append(rb.edges, edge)
	return rb
}

// Conditional adds a conditional edge from one node to another with a condition expression.
// The edge is followed only if the condition evaluates to true.
// Condition expressions use expr syntax and have access to the workflow execution context.
//
// Example conditions:
//   - "status == 'approved'"
//   - "amount > 1000"
//   - "inquiry_type == 'billing'"
func (rb *RelationshipBuilder) Conditional(from, to Node, condition string) *RelationshipBuilder {
	config := map[string]any{
		"condition": condition,
	}
	edge := NewEdge(
		uuid.NewString(),
		rb.workflowID,
		from.ID(),
		to.ID(),
		"conditional",
		config,
	)
	rb.edges = append(rb.edges, edge)
	return rb
}

// Build returns the constructed slice of edges.
// Call this method at the end of the builder chain to get the final edges.
func (rb *RelationshipBuilder) Build() []Edge {
	return rb.edges
}
