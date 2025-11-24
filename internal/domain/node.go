package domain

import (
	"github.com/google/uuid"
)

// Node represents a step in a workflow.
// Node is an entity that is part of the Workflow aggregate.
// It defines the configuration and metadata for a single processing unit within a workflow.
type Node interface {
	ID() uuid.UUID
	Type() NodeType
	Name() string
	Config() map[string]any
	IOSchema() *NodeIOSchema
	InputBindingConfig() *InputBindingConfig
}

// node is the internal implementation of Node entity.
// It is managed by the Workflow aggregate and has no independent lifecycle.
type node struct {
	id            uuid.UUID
	nodeType      NodeType
	name          string
	config        map[string]any
	ioSchema      *NodeIOSchema
	bindingConfig *InputBindingConfig
}

// RestoreNode creates a Node instance for reconstruction from persistence.
// This function is used internally for rebuilding the aggregate from storage.
func RestoreNode(id uuid.UUID, nodeType NodeType, name string, config map[string]any) Node {
	n := &node{
		id:       id,
		nodeType: nodeType,
		name:     name,
		config:   config,
	}

	// Extract schema and binding config from config if present
	if ioSchema, ok := config["_io_schema"].(*NodeIOSchema); ok {
		n.ioSchema = ioSchema
	}
	if bindingConfig, ok := config["_binding_config"].(*InputBindingConfig); ok {
		n.bindingConfig = bindingConfig
	}

	return n
}

// NewNode creates and returns a new Node instance with a specified type, name, and configuration map.
func NewNode(nodeType NodeType, name string, config map[string]any) Node {
	return RestoreNode(uuid.New(), nodeType, name, config)
}

// ID returns the node ID.
func (n *node) ID() uuid.UUID {
	return n.id
}

// Type returns the type of the node.
func (n *node) Type() NodeType {
	return n.nodeType
}

// Name returns the name of the node.
func (n *node) Name() string {
	return n.name
}

// Config returns the configuration of the node.
func (n *node) Config() map[string]any {
	return n.config
}

// IOSchema returns the input/output schema of the node.
func (n *node) IOSchema() *NodeIOSchema {
	return n.ioSchema
}

// InputBindingConfig returns the input binding configuration.
// Returns a default configuration if not explicitly set.
func (n *node) InputBindingConfig() *InputBindingConfig {
	if n.bindingConfig == nil {
		// Default: auto-bind with namespace collision strategy
		return &InputBindingConfig{
			AutoBind:          true,
			Mappings:          make(map[string]string),
			CollisionStrategy: CollisionStrategyNamespaceByParent,
		}
	}
	return n.bindingConfig
}
