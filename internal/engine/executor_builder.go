package engine

import (
	e "mbflow/internal/edge"
	n "mbflow/internal/node"
)

type ExecutorBuilder struct {
	g     *Graph
	nodes map[string]n.Node
	edges []e.Edge
}

func NewExecutorBuilder() *ExecutorBuilder {
	return &ExecutorBuilder{g: NewGraph(), nodes: make(map[string]n.Node)}
}

func (b *ExecutorBuilder) Graph(g *Graph) *ExecutorBuilder { b.g = g; return b }

func (b *ExecutorBuilder) WithNode(node n.Node) *ExecutorBuilder {
	b.nodes[node.ID()] = node
	b.g.AddNode(node.ID())
	return b
}

func (b *ExecutorBuilder) WithEdge(edge e.Edge) *ExecutorBuilder {
	b.edges = append(b.edges, edge)
	b.g.AddEdge(edge.From(), edge.To())
	return b
}

func (b *ExecutorBuilder) Build() *Executor { return NewExecutor(b.g, b.nodes, b.edges) }
