package domain

import "github.com/google/uuid"

// Edge is a domain entity that represents a connection between two nodes in a workflow.
// It defines the control flow and data transformation rules between workflow steps.
// Edge is an entity that is part of the Workflow aggregate.
type Edge interface {
	ID() uuid.UUID
	FromNodeID() uuid.UUID
	ToNodeID() uuid.UUID
	Type() EdgeType
	Config() map[string]any
}

// edge is the internal implementation of Edge entity.
// It is managed by the Workflow aggregate and has no independent lifecycle.
type edge struct {
	id         uuid.UUID
	fromNodeID uuid.UUID
	toNodeID   uuid.UUID
	edgeType   EdgeType
	config     map[string]any
}

// RestoreEdge creates an Edge instance for reconstruction from persistence.
// This function is used internally for rebuilding the aggregate from storage.
func RestoreEdge(id, fromNodeID, toNodeID uuid.UUID, edgeType EdgeType, config map[string]any) Edge {
	return &edge{
		id:         id,
		fromNodeID: fromNodeID,
		toNodeID:   toNodeID,
		edgeType:   edgeType,
		config:     config,
	}
}

// NewEdge creates a new Edge connecting two nodes with a specified type and configuration.
// It generates a unique ID for the Edge.
func NewEdge(fromNodeID, toNodeID uuid.UUID, edgeType EdgeType, config map[string]any) Edge {
	return RestoreEdge(uuid.New(), fromNodeID, toNodeID, edgeType, config)
}

// ID returns the edge ID.
func (e *edge) ID() uuid.UUID {
	return e.id
}

// FromNodeID returns the source node ID.
func (e *edge) FromNodeID() uuid.UUID {
	return e.fromNodeID
}

// ToNodeID returns the destination node ID.
func (e *edge) ToNodeID() uuid.UUID {
	return e.toNodeID
}

// Type returns the type of the edge.
func (e *edge) Type() EdgeType {
	return e.edgeType
}

// Config returns the configuration of the edge.
func (e *edge) Config() map[string]any {
	return e.config
}
