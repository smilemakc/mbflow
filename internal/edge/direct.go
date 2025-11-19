package edge

import (
	"context"
	n "mbflow/internal/node"
)

type Direct struct {
	from string
	to   string
}

func NewDirect(from, to string) *Direct { return &Direct{from: from, to: to} }

func (d *Direct) From() string { return d.from }
func (d *Direct) To() string   { return d.to }
func (d *Direct) Type() Type   { return TypeDirect }

func (d *Direct) Traverse(_ context.Context, output n.NodeOutput) (bool, n.NodeInput, error) {
	return true, n.NodeInput{Data: output.Data, Metadata: output.Metadata}, nil
}
