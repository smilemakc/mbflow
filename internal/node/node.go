package node

import (
	"context"
)

type Schema map[string]any

type NodeInput struct {
	Data     any
	Metadata map[string]string
}

type NodeOutput struct {
	Data     any
	Metadata map[string]string
}

type Node interface {
	ID() string
	Name() string
	Version() string
	Execute(ctx context.Context, input NodeInput) (NodeOutput, error)
	Validate(input NodeInput) error
	InputSchema() Schema
	OutputSchema() Schema
}
