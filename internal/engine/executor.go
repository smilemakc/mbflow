package engine

import (
	"context"
	"errors"
	e "mbflow/internal/edge"
	n "mbflow/internal/node"
)

type ExecutionResult struct {
	Outputs map[string]n.NodeOutput
}

type Executor struct {
	graph *Graph
	nodes map[string]n.Node
	edges []e.Edge
}

func NewExecutor(graph *Graph, nodes map[string]n.Node, edges []e.Edge) *Executor {
	return &Executor{graph: graph, nodes: nodes, edges: edges}
}

func (ex *Executor) Execute(ctx context.Context, inputs map[string]n.NodeInput) (ExecutionResult, error) {
	if err := ex.graph.ValidateDAG(); err != nil {
		return ExecutionResult{}, err
	}
	order, err := ex.graph.TopologicalSort()
	if err != nil {
		return ExecutionResult{}, err
	}
	outputs := make(map[string]n.NodeOutput)
	// Execute sequentially per MVP
	for _, nodeID := range order {
		node, ok := ex.nodes[nodeID]
		if !ok {
			return ExecutionResult{}, errors.New("node not found: " + nodeID)
		}
		input := inputs[nodeID]
		if err := node.Validate(input); err != nil {
			return ExecutionResult{}, err
		}
		out, err := node.Execute(ctx, input)
		if err != nil {
			return ExecutionResult{}, err
		}
		outputs[nodeID] = out
		// propagate along outgoing edges
		for _, ed := range ex.edges {
			if ed.From() != nodeID {
				continue
			}
			proceed, transformed, err := ed.Traverse(ctx, out)
			if err != nil {
				return ExecutionResult{}, err
			}
			if !proceed {
				continue
			}
			inputs[ed.To()] = transformed
		}
	}
	return ExecutionResult{Outputs: outputs}, nil
}
