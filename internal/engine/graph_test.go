package engine

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestGraph_ValidateDAG_OK(t *testing.T) {
	g := NewGraph()
	g.AddNode("A")
	g.AddNode("B")
	g.AddNode("C")
	g.AddEdge("A", "B")
	g.AddEdge("B", "C")
	err := g.ValidateDAG()
	assert.NoError(t, err)
}

func TestGraph_ValidateDAG_Cycle(t *testing.T) {
	g := NewGraph()
	g.AddNode("A")
	g.AddNode("B")
	g.AddEdge("A", "B")
	g.AddEdge("B", "A")
	err := g.ValidateDAG()
	assert.Error(t, err)
}

func TestGraph_TopologicalSort(t *testing.T) {
	g := NewGraph()
	g.AddNode("A")
	g.AddNode("B")
	g.AddNode("C")
	g.AddEdge("A", "B")
	g.AddEdge("A", "C")
	order, err := g.TopologicalSort()
	assert.NoError(t, err)
	assert.Len(t, order, 3)
	assert.Equal(t, "A", order[0])
}
