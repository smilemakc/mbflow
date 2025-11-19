package node

import (
	"context"
	"github.com/stretchr/testify/assert"
	"testing"
)

type dummyNode struct{ id string }

func (d *dummyNode) ID() string      { return d.id }
func (d *dummyNode) Name() string    { return "dummy" }
func (d *dummyNode) Version() string { return "1.0" }
func (d *dummyNode) Execute(ctx context.Context, input NodeInput) (NodeOutput, error) {
	return NodeOutput{Data: input.Data}, nil
}
func (d *dummyNode) Validate(input NodeInput) error { return nil }
func (d *dummyNode) InputSchema() Schema            { return Schema{} }
func (d *dummyNode) OutputSchema() Schema           { return Schema{} }

func TestRegistry_RegisterAndGet(t *testing.T) {
	r := NewRegistry()
	n := &dummyNode{id: "n1"}
	err := r.Register(n)
	assert.NoError(t, err)
	got, ok := r.GetByID("n1")
	assert.True(t, ok)
	assert.NotNil(t, got)
	assert.Equal(t, "n1", got.ID())
}
