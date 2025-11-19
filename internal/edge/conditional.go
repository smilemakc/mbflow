package edge

import (
	"context"
	n "mbflow/internal/node"
)

type ConditionFunc func(output n.NodeOutput) (bool, error)

type Conditional struct {
	from      string
	to        string
	condition ConditionFunc
}

func NewConditional(from, to string, cond ConditionFunc) *Conditional {
	return &Conditional{from: from, to: to, condition: cond}
}

func (c *Conditional) From() string { return c.from }
func (c *Conditional) To() string   { return c.to }
func (c *Conditional) Type() Type   { return TypeConditional }

func (c *Conditional) Traverse(_ context.Context, output n.NodeOutput) (bool, n.NodeInput, error) {
	ok, err := c.condition(output)
	if err != nil {
		return false, n.NodeInput{}, err
	}
	if !ok {
		return false, n.NodeInput{}, nil
	}
	return true, n.NodeInput{Data: output.Data, Metadata: output.Metadata}, nil
}
