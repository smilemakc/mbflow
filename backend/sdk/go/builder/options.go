package builder

import "github.com/smilemakc/mbflow/sdk/go/models"

// WorkflowOption configures a WorkflowBuilder.
type WorkflowOption func(*WorkflowBuilder)

func WithDescription(desc string) WorkflowOption {
	return func(b *WorkflowBuilder) { b.workflow.Description = desc }
}

func WithStatus(status models.WorkflowStatus) WorkflowOption {
	return func(b *WorkflowBuilder) { b.workflow.Status = status }
}

func WithVariable(key string, value any) WorkflowOption {
	return func(b *WorkflowBuilder) {
		if b.workflow.Variables == nil {
			b.workflow.Variables = make(map[string]any)
		}
		b.workflow.Variables[key] = value
	}
}

func WithTag(tags ...string) WorkflowOption {
	return func(b *WorkflowBuilder) {
		b.workflow.Tags = append(b.workflow.Tags, tags...)
	}
}

func WithMetadata(key string, value any) WorkflowOption {
	return func(b *WorkflowBuilder) {
		if b.workflow.Metadata == nil {
			b.workflow.Metadata = make(map[string]any)
		}
		b.workflow.Metadata[key] = value
	}
}

// NodeOption configures a node.
type NodeOption func(n *models.Node)

func WithConfig(key string, value any) NodeOption {
	return func(n *models.Node) {
		if n.Config == nil {
			n.Config = make(map[string]any)
		}
		n.Config[key] = value
	}
}

func WithNodeDescription(desc string) NodeOption {
	return func(n *models.Node) { n.Description = desc }
}

func WithPosition(x, y float64) NodeOption {
	return func(n *models.Node) {
		n.Position = &models.Position{X: x, Y: y}
	}
}

// EdgeOption configures an edge.
type EdgeOption func(e *models.Edge)

func WithCondition(expr string) EdgeOption {
	return func(e *models.Edge) { e.Condition = expr }
}

func WithSourceHandle(handle string) EdgeOption {
	return func(e *models.Edge) { e.SourceHandle = handle }
}

func WithLoop(maxIterations int) EdgeOption {
	return func(e *models.Edge) {
		e.Loop = &models.LoopConfig{MaxIterations: maxIterations}
	}
}
