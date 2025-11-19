package mbflow

import (
	"context"

	"mbflow/internal/domain"
)

// storageAdapter adapts internal storage to the public interface.
type storageAdapter struct {
	store domain.Storage
}

// Workflow methods

func (a *storageAdapter) SaveWorkflow(ctx context.Context, w Workflow) error {
	// Convert public interface to domain entity
	domainWorkflow, ok := w.(*domain.Workflow)
	if !ok {
		// If it's not our domain entity, create a new one
		domainWorkflow = domain.ReconstructWorkflow(w.ID(), w.Name(), w.Version(), w.Spec(), w.CreatedAt())
	}
	return a.store.SaveWorkflow(ctx, domainWorkflow)
}

func (a *storageAdapter) GetWorkflow(ctx context.Context, id string) (Workflow, error) {
	return a.store.GetWorkflow(ctx, id)
}

func (a *storageAdapter) ListWorkflows(ctx context.Context) ([]Workflow, error) {
	workflows, err := a.store.ListWorkflows(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]Workflow, len(workflows))
	for i, w := range workflows {
		result[i] = w
	}
	return result, nil
}

// Execution methods

func (a *storageAdapter) SaveExecution(ctx context.Context, e Execution) error {
	domainExecution := unwrapExecution(e)
	return a.store.SaveExecution(ctx, domainExecution)
}

func (a *storageAdapter) GetExecution(ctx context.Context, id string) (Execution, error) {
	exec, err := a.store.GetExecution(ctx, id)
	if err != nil {
		return nil, err
	}
	return wrapExecution(exec), nil
}

func (a *storageAdapter) ListExecutions(ctx context.Context) ([]Execution, error) {
	executions, err := a.store.ListExecutions(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]Execution, len(executions))
	for i, e := range executions {
		result[i] = wrapExecution(e)
	}
	return result, nil
}

// Event methods

func (a *storageAdapter) AppendEvent(ctx context.Context, e Event) error {
	domainEvent, ok := e.(*domain.Event)
	if !ok {
		domainEvent = domain.ReconstructEvent(e.EventID(), e.EventType(), e.WorkflowID(), e.ExecutionID(), e.WorkflowName(), e.NodeID(), e.Timestamp(), e.Payload(), e.Metadata())
	}
	return a.store.AppendEvent(ctx, domainEvent)
}

func (a *storageAdapter) ListEventsByExecution(ctx context.Context, executionID string) ([]Event, error) {
	events, err := a.store.ListEventsByExecution(ctx, executionID)
	if err != nil {
		return nil, err
	}
	result := make([]Event, len(events))
	for i, e := range events {
		result[i] = e
	}
	return result, nil
}

// Node methods

func (a *storageAdapter) SaveNode(ctx context.Context, n Node) error {
	domainNode, ok := n.(*domain.Node)
	if !ok {
		domainNode = domain.NewNode(n.ID(), n.WorkflowID(), n.Type(), n.Name(), n.Config())
	}
	return a.store.SaveNode(ctx, domainNode)
}

func (a *storageAdapter) GetNode(ctx context.Context, id string) (Node, error) {
	return a.store.GetNode(ctx, id)
}

func (a *storageAdapter) ListNodes(ctx context.Context, workflowID string) ([]Node, error) {
	nodes, err := a.store.ListNodes(ctx, workflowID)
	if err != nil {
		return nil, err
	}
	result := make([]Node, len(nodes))
	for i, n := range nodes {
		result[i] = n
	}
	return result, nil
}

// Edge methods

func (a *storageAdapter) SaveEdge(ctx context.Context, e Edge) error {
	domainEdge, ok := e.(*domain.Edge)
	if !ok {
		domainEdge = domain.NewEdge(e.ID(), e.WorkflowID(), e.FromNodeID(), e.ToNodeID(), e.Type(), e.Config())
	}
	return a.store.SaveEdge(ctx, domainEdge)
}

func (a *storageAdapter) GetEdge(ctx context.Context, id string) (Edge, error) {
	return a.store.GetEdge(ctx, id)
}

func (a *storageAdapter) ListEdges(ctx context.Context, workflowID string) ([]Edge, error) {
	edges, err := a.store.ListEdges(ctx, workflowID)
	if err != nil {
		return nil, err
	}
	result := make([]Edge, len(edges))
	for i, e := range edges {
		result[i] = e
	}
	return result, nil
}

// Trigger methods

func (a *storageAdapter) SaveTrigger(ctx context.Context, t Trigger) error {
	domainTrigger, ok := t.(*domain.Trigger)
	if !ok {
		domainTrigger = domain.NewTrigger(t.ID(), t.WorkflowID(), t.Type(), t.Config())
	}
	return a.store.SaveTrigger(ctx, domainTrigger)
}

func (a *storageAdapter) GetTrigger(ctx context.Context, id string) (Trigger, error) {
	return a.store.GetTrigger(ctx, id)
}

func (a *storageAdapter) ListTriggers(ctx context.Context, workflowID string) ([]Trigger, error) {
	triggers, err := a.store.ListTriggers(ctx, workflowID)
	if err != nil {
		return nil, err
	}
	result := make([]Trigger, len(triggers))
	for i, t := range triggers {
		result[i] = t
	}
	return result, nil
}
