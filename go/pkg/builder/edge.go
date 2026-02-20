package builder

import (
	"fmt"

	"github.com/smilemakc/mbflow/go/pkg/models"
)

// EdgeBuilder builds edge definitions.
type EdgeBuilder struct {
	id           string
	from         string
	to           string
	condition    string
	sourceHandle string
	loop         *models.LoopConfig
	metadata     map[string]any
	err          error
}

// EdgeOption is a function that configures an EdgeBuilder.
type EdgeOption func(*EdgeBuilder) error

// NewEdge creates a new edge builder.
// Edge ID is auto-generated as "edge_{from}_{to}" unless overridden.
func NewEdge(from, to string, opts ...EdgeOption) *EdgeBuilder {
	eb := &EdgeBuilder{
		from:     from,
		to:       to,
		metadata: make(map[string]any),
	}

	// Auto-generate edge ID
	eb.id = fmt.Sprintf("edge_%s_%s", from, to)

	for _, opt := range opts {
		if err := opt(eb); err != nil {
			eb.err = err
			return eb
		}
	}

	return eb
}

// Build constructs the final Edge.
func (eb *EdgeBuilder) Build() (*models.Edge, error) {
	if eb.err != nil {
		return nil, eb.err
	}

	edge := &models.Edge{
		ID:           eb.id,
		From:         eb.from,
		To:           eb.to,
		SourceHandle: eb.sourceHandle,
		Condition:    eb.condition,
		Loop:         eb.loop,
		Metadata:     eb.metadata,
	}

	if err := edge.Validate(); err != nil {
		return nil, err
	}

	return edge, nil
}

// WithEdgeID sets a custom edge ID.
func WithEdgeID(id string) EdgeOption {
	return func(eb *EdgeBuilder) error {
		if id == "" {
			return fmt.Errorf("edge ID cannot be empty")
		}
		eb.id = id
		return nil
	}
}

// WithCondition sets an edge condition (expr-lang expression).
func WithCondition(condition string) EdgeOption {
	return func(eb *EdgeBuilder) error {
		eb.condition = condition
		return nil
	}
}

// WhenTrue creates a conditional edge that activates when the condition is true.
// This is a convenience wrapper around WithCondition.
func WhenTrue(condition string) EdgeOption {
	return func(eb *EdgeBuilder) error {
		if condition == "" {
			return fmt.Errorf("condition cannot be empty")
		}
		eb.condition = condition
		return nil
	}
}

// WhenFalse creates a conditional edge that activates when the condition is false.
// This is a convenience wrapper that negates the condition.
func WhenFalse(condition string) EdgeOption {
	return func(eb *EdgeBuilder) error {
		if condition == "" {
			return fmt.Errorf("condition cannot be empty")
		}
		eb.condition = fmt.Sprintf("!(%s)", condition)
		return nil
	}
}

// WhenEqual creates a conditional edge that activates when field equals value.
// Example: WhenEqual("output.status", "success")
func WhenEqual(field, value string) EdgeOption {
	return func(eb *EdgeBuilder) error {
		if field == "" {
			return fmt.Errorf("field cannot be empty")
		}
		if value == "" {
			return fmt.Errorf("value cannot be empty")
		}
		eb.condition = fmt.Sprintf("%s == %q", field, value)
		return nil
	}
}

// WithEdgeMetadata adds edge metadata.
func WithEdgeMetadata(key string, value any) EdgeOption {
	return func(eb *EdgeBuilder) error {
		if key == "" {
			return fmt.Errorf("metadata key cannot be empty")
		}
		eb.metadata[key] = value
		return nil
	}
}

// WithSourceHandle sets the source handle for conditional routing.
// Used with conditional nodes to specify which branch (e.g. "true" or "false") this edge comes from.
func WithSourceHandle(handle string) EdgeOption {
	return func(eb *EdgeBuilder) error {
		if handle == "" {
			return fmt.Errorf("source handle cannot be empty")
		}
		eb.sourceHandle = handle
		return nil
	}
}

// FromTrueBranch creates an edge from the "true" branch of a conditional node.
func FromTrueBranch() EdgeOption {
	return func(eb *EdgeBuilder) error {
		eb.sourceHandle = "true"
		return nil
	}
}

// FromFalseBranch creates an edge from the "false" branch of a conditional node.
func FromFalseBranch() EdgeOption {
	return func(eb *EdgeBuilder) error {
		eb.sourceHandle = "false"
		return nil
	}
}

// WithLoop marks this edge as a loop (back) edge with the specified max iterations.
// Loop edges are excluded from topological sort and enable controlled re-execution of wave ranges.
func WithLoop(maxIterations int) EdgeOption {
	return func(eb *EdgeBuilder) error {
		if maxIterations <= 0 {
			return fmt.Errorf("max iterations must be > 0")
		}
		eb.loop = &models.LoopConfig{MaxIterations: maxIterations}
		return nil
	}
}
