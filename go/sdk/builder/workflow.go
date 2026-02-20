package builder

import (
	"fmt"

	"github.com/smilemakc/mbflow/go/sdk/models"
)

// WorkflowBuilder constructs a Workflow using a fluent API.
type WorkflowBuilder struct {
	workflow   *models.Workflow
	nodes      map[string]bool
	nodeOrder  []string
	edges      []*models.Edge
	err        error
	autoLayout bool
}

func NewWorkflow(name string, opts ...WorkflowOption) *WorkflowBuilder {
	b := &WorkflowBuilder{
		workflow: &models.Workflow{
			Name:   name,
			Status: models.WorkflowStatusDraft,
		},
		nodes: make(map[string]bool),
	}
	for _, o := range opts {
		o(b)
	}
	return b
}

func (b *WorkflowBuilder) AddNode(id, name, nodeType string, opts ...NodeOption) *WorkflowBuilder {
	if b.err != nil {
		return b
	}
	if b.nodes[id] {
		b.err = fmt.Errorf("duplicate node ID: %s", id)
		return b
	}

	node := &models.Node{
		ID:   id,
		Name: name,
		Type: nodeType,
	}
	for _, o := range opts {
		o(node)
	}

	b.workflow.Nodes = append(b.workflow.Nodes, node)
	b.nodes[id] = true
	b.nodeOrder = append(b.nodeOrder, id)
	return b
}

func (b *WorkflowBuilder) Connect(from, to string, opts ...EdgeOption) *WorkflowBuilder {
	if b.err != nil {
		return b
	}
	edge := &models.Edge{
		ID:   fmt.Sprintf("edge_%s_%s", from, to),
		From: from,
		To:   to,
	}
	for _, o := range opts {
		o(edge)
	}
	b.edges = append(b.edges, edge)
	return b
}

func (b *WorkflowBuilder) ConnectWithCondition(from, to, condition string) *WorkflowBuilder {
	return b.Connect(from, to, WithCondition(condition))
}

func (b *WorkflowBuilder) ConnectFromHandle(from, to, handle string) *WorkflowBuilder {
	return b.Connect(from, to, WithSourceHandle(handle))
}

func (b *WorkflowBuilder) WithAutoLayout() *WorkflowBuilder {
	b.autoLayout = true
	return b
}

func (b *WorkflowBuilder) Build() (*models.Workflow, error) {
	if b.err != nil {
		return nil, b.err
	}

	for _, edge := range b.edges {
		if !b.nodes[edge.From] {
			return nil, fmt.Errorf("edge %s references unknown source node: %s", edge.ID, edge.From)
		}
		if !b.nodes[edge.To] {
			return nil, fmt.Errorf("edge %s references unknown target node: %s", edge.ID, edge.To)
		}
	}

	b.workflow.Edges = b.edges

	if b.autoLayout {
		b.applyAutoLayout()
	}

	return b.workflow, nil
}

func (b *WorkflowBuilder) MustBuild() *models.Workflow {
	wf, err := b.Build()
	if err != nil {
		panic(fmt.Sprintf("MustBuild: %v", err))
	}
	return wf
}

func (b *WorkflowBuilder) applyAutoLayout() {
	const (
		xStart = 100.0
		yStart = 100.0
		yStep  = 150.0
	)
	for i, node := range b.workflow.Nodes {
		if node.Position == nil {
			node.Position = &models.Position{
				X: xStart,
				Y: yStart + float64(i)*yStep,
			}
		}
	}
}
