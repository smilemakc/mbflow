package edge

import (
	"context"
	n "mbflow/internal/node"
)

type Type string

const (
	TypeDirect      Type = "direct"
	TypeConditional Type = "conditional"
)

type Edge interface {
	From() string
	To() string
	Type() Type
	Traverse(ctx context.Context, output n.NodeOutput) (proceed bool, transformed n.NodeInput, err error)
}
