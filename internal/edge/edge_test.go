package edge

import (
	"context"
	"github.com/stretchr/testify/assert"
	n "mbflow/internal/node"
	"testing"
)

func TestDirect_Traverse(t *testing.T) {
	d := NewDirect("A", "B")
	ok, in, err := d.Traverse(context.Background(), n.NodeOutput{Data: 42})
	assert.NoError(t, err)
	assert.True(t, ok)
	assert.Equal(t, 42, in.Data.(int))
}

func TestConditional_Traverse(t *testing.T) {
	c := NewConditional("A", "B", func(out n.NodeOutput) (bool, error) { return out.Data.(int) > 10, nil })
	ok, _, _ := c.Traverse(context.Background(), n.NodeOutput{Data: 5})
	assert.False(t, ok)
	ok, _, _ = c.Traverse(context.Background(), n.NodeOutput{Data: 15})
	assert.True(t, ok)
}
