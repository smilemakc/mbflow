package engine

type GraphBuilder struct {
	g *Graph
}

func NewGraphBuilder() *GraphBuilder { return &GraphBuilder{g: NewGraph()} }

func (b *GraphBuilder) AddNode(id string) *GraphBuilder {
	b.g.AddNode(id)
	return b
}

func (b *GraphBuilder) AddEdge(from, to string) *GraphBuilder {
	b.g.AddEdge(from, to)
	return b
}

func (b *GraphBuilder) Build() *Graph { return b.g }
