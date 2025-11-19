package engine

import (
	"context"
	"github.com/stretchr/testify/assert"
	e "mbflow/internal/edge"
	n "mbflow/internal/node"
	"testing"
)

type echoNode struct{ id string }

func (eN *echoNode) ID() string                    { return eN.id }
func (eN *echoNode) Name() string                  { return "echo" }
func (eN *echoNode) Version() string               { return "1.0" }
func (eN *echoNode) Validate(in n.NodeInput) error { return nil }
func (eN *echoNode) InputSchema() n.Schema         { return n.Schema{} }
func (eN *echoNode) OutputSchema() n.Schema        { return n.Schema{} }
func (eN *echoNode) Execute(ctx context.Context, in n.NodeInput) (n.NodeOutput, error) {
	return n.NodeOutput{Data: in.Data}, nil
}

func TestExecutor_SequentialPropagation(t *testing.T) {
	g := NewGraph()
	g.AddNode("A")
	g.AddNode("B")
	g.AddEdge("A", "B")
	nodes := map[string]n.Node{
		"A": &echoNode{id: "A"},
		"B": &echoNode{id: "B"},
	}
	edges := []e.Edge{e.NewDirect("A", "B")}
	ex := NewExecutor(g, nodes, edges)
	res, err := ex.Execute(context.Background(), map[string]n.NodeInput{
		"A": {Data: 7},
	})
	assert.NoError(t, err)
	assert.Equal(t, 7, res.Outputs["B"].Data.(int))
}
