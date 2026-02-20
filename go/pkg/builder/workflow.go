package builder

import (
	"fmt"

	"github.com/smilemakc/mbflow/go/pkg/models"
)

// WorkflowBuilder builds workflow definitions fluently.
type WorkflowBuilder struct {
	workflow      *models.Workflow
	nodes         map[string]*NodeBuilder // Track node builders by ID
	nodeOrder     []string                // Track insertion order
	edges         []*EdgeBuilder
	err           error // Accumulate errors
	strictMode    bool
	autoLayout    bool
	layoutCounter int
}

// WorkflowOption is a function that configures a WorkflowBuilder.
type WorkflowOption func(*WorkflowBuilder) error

// NewWorkflow creates a new workflow builder with the given name.
func NewWorkflow(name string, opts ...WorkflowOption) *WorkflowBuilder {
	wb := &WorkflowBuilder{
		workflow: &models.Workflow{
			Name:      name,
			Status:    models.WorkflowStatusDraft,
			Variables: make(map[string]any),
			Metadata:  make(map[string]any),
			Nodes:     make([]*models.Node, 0),
			Edges:     make([]*models.Edge, 0),
		},
		nodes:     make(map[string]*NodeBuilder),
		nodeOrder: make([]string, 0),
		edges:     make([]*EdgeBuilder, 0),
	}

	for _, opt := range opts {
		if err := opt(wb); err != nil {
			wb.err = err
			return wb
		}
	}

	return wb
}

// WithDescription sets the workflow description.
func WithDescription(desc string) WorkflowOption {
	return func(wb *WorkflowBuilder) error {
		wb.workflow.Description = desc
		return nil
	}
}

// WithStatus sets the workflow status.
func WithStatus(status models.WorkflowStatus) WorkflowOption {
	return func(wb *WorkflowBuilder) error {
		wb.workflow.Status = status
		return nil
	}
}

// WithVariable adds a workflow variable.
func WithVariable(key string, value any) WorkflowOption {
	return func(wb *WorkflowBuilder) error {
		if key == "" {
			return fmt.Errorf("variable key cannot be empty")
		}
		wb.workflow.Variables[key] = value
		return nil
	}
}

// WithVariables sets multiple workflow variables.
func WithVariables(vars map[string]any) WorkflowOption {
	return func(wb *WorkflowBuilder) error {
		for k, v := range vars {
			wb.workflow.Variables[k] = v
		}
		return nil
	}
}

// WithTags sets workflow tags.
func WithTags(tags ...string) WorkflowOption {
	return func(wb *WorkflowBuilder) error {
		wb.workflow.Tags = tags
		return nil
	}
}

// WithMetadata adds workflow metadata.
func WithMetadata(key string, value any) WorkflowOption {
	return func(wb *WorkflowBuilder) error {
		if key == "" {
			return fmt.Errorf("metadata key cannot be empty")
		}
		wb.workflow.Metadata[key] = value
		return nil
	}
}

// WithStrictValidation enables strict validation mode.
// In strict mode, all node configs are validated upfront.
func WithStrictValidation() WorkflowOption {
	return func(wb *WorkflowBuilder) error {
		wb.strictMode = true
		return nil
	}
}

// WithAutoLayout enables automatic layout of nodes.
// Nodes will be positioned in a horizontal flow with 200px spacing.
func WithAutoLayout() WorkflowOption {
	return func(wb *WorkflowBuilder) error {
		wb.autoLayout = true
		return nil
	}
}

// AddNode adds a node to the workflow using a NodeBuilder.
func (wb *WorkflowBuilder) AddNode(nodeBuilder *NodeBuilder) *WorkflowBuilder {
	if wb.err != nil {
		return wb
	}

	if nodeBuilder == nil {
		wb.err = fmt.Errorf("node builder cannot be nil")
		return wb
	}

	if nodeBuilder.id == "" {
		wb.err = fmt.Errorf("node must have an ID")
		return wb
	}

	if _, exists := wb.nodes[nodeBuilder.id]; exists {
		wb.err = fmt.Errorf("duplicate node ID: %s", nodeBuilder.id)
		return wb
	}

	// Auto-layout if enabled and no position set
	if wb.autoLayout && nodeBuilder.position == nil {
		nodeBuilder.position = &models.Position{
			X: float64(wb.layoutCounter * 200),
			Y: 100,
		}
		wb.layoutCounter++
	}

	wb.nodes[nodeBuilder.id] = nodeBuilder
	wb.nodeOrder = append(wb.nodeOrder, nodeBuilder.id)
	return wb
}

// Connect creates an edge between two nodes.
func (wb *WorkflowBuilder) Connect(fromID, toID string, opts ...EdgeOption) *WorkflowBuilder {
	if wb.err != nil {
		return wb
	}

	eb := NewEdge(fromID, toID, opts...)
	wb.edges = append(wb.edges, eb)
	return wb
}

// Build validates and constructs the final Workflow.
func (wb *WorkflowBuilder) Build() (*models.Workflow, error) {
	if wb.err != nil {
		return nil, wb.err
	}

	// Build all nodes in insertion order
	nodes := make([]*models.Node, 0, len(wb.nodes))
	for _, id := range wb.nodeOrder {
		nb := wb.nodes[id]
		node, err := nb.Build()
		if err != nil {
			return nil, fmt.Errorf("node %s: %w", id, err)
		}
		nodes = append(nodes, node)
	}
	wb.workflow.Nodes = nodes

	// Build all edges
	edges := make([]*models.Edge, 0, len(wb.edges))
	for i, eb := range wb.edges {
		edge, err := eb.Build()
		if err != nil {
			return nil, fmt.Errorf("edge %d: %w", i, err)
		}
		edges = append(edges, edge)
	}
	wb.workflow.Edges = edges

	// Validate workflow structure
	if err := wb.workflow.Validate(); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	return wb.workflow, nil
}

// MustBuild builds and panics on error.
// Useful for examples and tests.
func (wb *WorkflowBuilder) MustBuild() *models.Workflow {
	wf, err := wb.Build()
	if err != nil {
		panic(err)
	}
	return wf
}
