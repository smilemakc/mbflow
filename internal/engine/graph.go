package engine

import (
	"errors"
)

type NodeRef struct {
	ID string
}

type EdgeRef struct {
	From string
	To   string
}

type Graph struct {
	Nodes map[string]NodeRef
	Edges []EdgeRef
	out   map[string][]string
	in    map[string][]string
}

func NewGraph() *Graph {
	return &Graph{Nodes: make(map[string]NodeRef), out: make(map[string][]string), in: make(map[string][]string)}
}

func (g *Graph) AddNode(id string) {
	g.Nodes[id] = NodeRef{ID: id}
}

func (g *Graph) AddEdge(from, to string) {
	g.Edges = append(g.Edges, EdgeRef{From: from, To: to})
	g.out[from] = append(g.out[from], to)
	g.in[to] = append(g.in[to], from)
}

func (g *Graph) ValidateDAG() error {
	// Kahn's algorithm cycle detection
	indeg := make(map[string]int)
	for id := range g.Nodes {
		indeg[id] = 0
	}
	for _, e := range g.Edges {
		indeg[e.To]++
	}
	q := make([]string, 0)
	for id, d := range indeg {
		if d == 0 {
			q = append(q, id)
		}
	}
	visited := 0
	for len(q) > 0 {
		n := q[0]
		q = q[1:]
		visited++
		for _, m := range g.out[n] {
			indeg[m]--
			if indeg[m] == 0 {
				q = append(q, m)
			}
		}
	}
	if visited != len(g.Nodes) {
		return errors.New("graph has cycles")
	}
	return nil
}

func (g *Graph) TopologicalSort() ([]string, error) {
	indeg := make(map[string]int)
	for id := range g.Nodes {
		indeg[id] = 0
	}
	for _, e := range g.Edges {
		indeg[e.To]++
	}
	q := make([]string, 0)
	for id, d := range indeg {
		if d == 0 {
			q = append(q, id)
		}
	}
	order := make([]string, 0, len(g.Nodes))
	for len(q) > 0 {
		n := q[0]
		q = q[1:]
		order = append(order, n)
		for _, m := range g.out[n] {
			indeg[m]--
			if indeg[m] == 0 {
				q = append(q, m)
			}
		}
	}
	if len(order) != len(g.Nodes) {
		return nil, errors.New("graph has cycles")
	}
	return order, nil
}
