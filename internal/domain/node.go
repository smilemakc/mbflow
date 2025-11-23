package domain

import (
	"fmt"

	"github.com/google/uuid"
)

// Node is a domain entity that represents a step in a workflow definition.
// It defines the configuration and metadata for a single processing unit within a workflow.
// Nodes are immutable entities that are part of a Workflow aggregate.
type Node struct {
	id         string
	workflowID string
	nodeType   string
	name       string
	config     map[string]any
}

// NodeConfig holds the configuration for creating a new Node with UUID validation.
type NodeConfig struct {
	// ID is the node ID (will be validated as UUID)
	ID string `json:"id"`
	// WorkflowID is the workflow ID this node belongs to (will be validated as UUID)
	WorkflowID string `json:"workflow_id,omitempty"`
	// Type is the node type (e.g., "http-request", "openai-completion")
	Type string `json:"type"`
	// Name is the display name for the node
	Name string `json:"name"`
	// Config holds the node-specific configuration
	Config map[string]any `json:"config,omitempty"`
}

// NewNode creates a new Node instance from NodeConfig with UUID validation.
// Returns an error if ID or WorkflowID are not valid UUIDs.
func NewNode(cfg NodeConfig) (*Node, error) {
	// Validate ID is a valid UUID
	if _, err := uuid.Parse(cfg.ID); err != nil {
		return nil, fmt.Errorf("invalid node ID '%s': must be a valid UUID: %w", cfg.ID, err)
	}
	if cfg.WorkflowID != "" {
		// Validate WorkflowID is a valid UUID
		if _, err := uuid.Parse(cfg.WorkflowID); err != nil {
			return nil, fmt.Errorf("invalid workflow ID '%s': must be a valid UUID: %w", cfg.WorkflowID, err)
		}
	}

	// Validate required fields
	if cfg.Type == "" {
		return nil, fmt.Errorf("node type cannot be empty")
	}

	return &Node{
		id:         cfg.ID,
		workflowID: cfg.WorkflowID,
		nodeType:   cfg.Type,
		name:       cfg.Name,
		config:     cfg.Config,
	}, nil
}

// ID returns the node ID.
func (n *Node) ID() string {
	return n.id
}

// WorkflowID returns the workflow ID this node belongs to.
func (n *Node) WorkflowID() string {
	return n.workflowID
}

// Type returns the type of the node.
func (n *Node) Type() string {
	return n.nodeType
}

// Name returns the name of the node.
func (n *Node) Name() string {
	return n.name
}

// Config returns the configuration of the node.
func (n *Node) Config() map[string]any {
	return n.config
}
