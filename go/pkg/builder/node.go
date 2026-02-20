package builder

import (
	"fmt"

	"github.com/smilemakc/mbflow/go/pkg/models"
)

// NodeBuilder builds node definitions.
type NodeBuilder struct {
	id          string
	name        string
	nodeType    string
	description string
	config      map[string]any
	position    *models.Position
	metadata    map[string]any
	err         error
}

// NodeOption is a function that configures a NodeBuilder.
type NodeOption func(*NodeBuilder) error

// NewNode creates a new node builder.
func NewNode(id, nodeType, name string, opts ...NodeOption) *NodeBuilder {
	nb := &NodeBuilder{
		id:       id,
		nodeType: nodeType,
		name:     name,
		config:   make(map[string]any),
		metadata: make(map[string]any),
	}

	for _, opt := range opts {
		if err := opt(nb); err != nil {
			nb.err = err
			return nb
		}
	}

	return nb
}

// Build constructs the final Node.
func (nb *NodeBuilder) Build() (*models.Node, error) {
	if nb.err != nil {
		return nil, nb.err
	}

	node := &models.Node{
		ID:          nb.id,
		Name:        nb.name,
		Type:        nb.nodeType,
		Description: nb.description,
		Config:      nb.config,
		Position:    nb.position,
		Metadata:    nb.metadata,
	}

	if err := node.Validate(); err != nil {
		return nil, err
	}

	return node, nil
}

// WithNodeDescription sets the node description.
func WithNodeDescription(desc string) NodeOption {
	return func(nb *NodeBuilder) error {
		nb.description = desc
		return nil
	}
}

// WithPosition sets the node position (absolute coordinates).
func WithPosition(x, y float64) NodeOption {
	return func(nb *NodeBuilder) error {
		nb.position = &models.Position{X: x, Y: y}
		return nil
	}
}

// GridPosition calculates position in a grid layout.
// Uses 200px spacing for both X and Y.
func GridPosition(row, col int) NodeOption {
	return func(nb *NodeBuilder) error {
		if row < 0 || col < 0 {
			return fmt.Errorf("grid position row and col must be non-negative")
		}
		nb.position = &models.Position{
			X: float64(col * 200),
			Y: float64(row * 200),
		}
		return nil
	}
}

// WithNodeMetadata adds node metadata.
func WithNodeMetadata(key string, value any) NodeOption {
	return func(nb *NodeBuilder) error {
		if key == "" {
			return fmt.Errorf("metadata key cannot be empty")
		}
		nb.metadata[key] = value
		return nil
	}
}

// WithConfig sets the raw config map.
// This is an escape hatch for advanced use cases.
func WithConfig(config map[string]any) NodeOption {
	return func(nb *NodeBuilder) error {
		nb.config = config
		return nil
	}
}

// WithConfigValue sets a single config value.
func WithConfigValue(key string, value any) NodeOption {
	return func(nb *NodeBuilder) error {
		if key == "" {
			return fmt.Errorf("config key cannot be empty")
		}
		nb.config[key] = value
		return nil
	}
}

// NewSubWorkflowNode creates a sub_workflow node for fan-out execution.
func NewSubWorkflowNode(id, name, workflowID string, opts ...NodeOption) *NodeBuilder {
	nb := NewNode(id, "sub_workflow", name)
	nb.config["workflow_id"] = workflowID
	for _, opt := range opts {
		if err := opt(nb); err != nil {
			nb.err = err
			return nb
		}
	}
	return nb
}

// WithForEach sets the for_each expression for fan-out.
func WithForEach(expression string) NodeOption {
	return func(nb *NodeBuilder) error {
		nb.config["for_each"] = expression
		return nil
	}
}

// WithItemVar sets the variable name for the current item in child input.
func WithItemVar(varName string) NodeOption {
	return func(nb *NodeBuilder) error {
		nb.config["item_var"] = varName
		return nil
	}
}

// WithMaxParallelism sets the max concurrent child executions.
func WithMaxParallelism(n int) NodeOption {
	return func(nb *NodeBuilder) error {
		nb.config["max_parallelism"] = n
		return nil
	}
}

// WithOnError sets the error handling strategy: "fail_fast" or "collect_partial".
func WithOnError(strategy string) NodeOption {
	return func(nb *NodeBuilder) error {
		nb.config["on_error"] = strategy
		return nil
	}
}

// WithWorkflowID sets the target workflow ID (alternative to constructor parameter).
func WithWorkflowID(workflowID string) NodeOption {
	return func(nb *NodeBuilder) error {
		nb.config["workflow_id"] = workflowID
		return nil
	}
}
