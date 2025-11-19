package storage

import (
	"context"
	"fmt"
	"sync"
	"time"

	"mbflow/internal/domain"
)

type MemoryStore struct {
	mu              sync.RWMutex
	workflows       map[string]*domain.Workflow
	executions      map[string]*domain.Execution
	executionStates map[string]*domain.ExecutionState
	events          []*domain.Event
	nodes           map[string]*domain.Node
	edges           map[string]*domain.Edge
	triggers        map[string]*domain.Trigger
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		workflows:       make(map[string]*domain.Workflow),
		executions:      make(map[string]*domain.Execution),
		executionStates: make(map[string]*domain.ExecutionState),
		nodes:           make(map[string]*domain.Node),
		edges:           make(map[string]*domain.Edge),
		triggers:        make(map[string]*domain.Trigger),
	}
}

func (s *MemoryStore) SaveWorkflow(ctx context.Context, w *domain.Workflow) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.workflows[w.ID()] = w
	return nil
}

func (s *MemoryStore) GetWorkflow(ctx context.Context, id string) (*domain.Workflow, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	w, ok := s.workflows[id]
	if !ok {
		return nil, fmt.Errorf("workflow not found")
	}
	return w, nil
}

func (s *MemoryStore) ListWorkflows(ctx context.Context) ([]*domain.Workflow, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*domain.Workflow, 0, len(s.workflows))
	for _, w := range s.workflows {
		out = append(out, w)
	}
	return out, nil
}

func (s *MemoryStore) SaveExecution(ctx context.Context, x *domain.Execution) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.executions[x.ID()] = x
	return nil
}

func (s *MemoryStore) GetExecution(ctx context.Context, id string) (*domain.Execution, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	x, ok := s.executions[id]
	if !ok {
		return nil, fmt.Errorf("execution not found")
	}
	return x, nil
}

func (s *MemoryStore) ListExecutions(ctx context.Context) ([]*domain.Execution, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*domain.Execution, 0, len(s.executions))
	for _, x := range s.executions {
		out = append(out, x)
	}
	return out, nil
}

func (s *MemoryStore) AppendEvent(ctx context.Context, ev *domain.Event) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.events = append(s.events, ev)
	return nil
}

func (s *MemoryStore) ListEventsByExecution(ctx context.Context, executionID string) ([]*domain.Event, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var out []*domain.Event
	for _, ev := range s.events {
		if ev.ExecutionID() == executionID {
			out = append(out, ev)
		}
	}
	return out, nil
}

func (s *MemoryStore) SaveNode(ctx context.Context, n *domain.Node) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.nodes[n.ID()] = n
	return nil
}

func (s *MemoryStore) GetNode(ctx context.Context, id string) (*domain.Node, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	n, ok := s.nodes[id]
	if !ok {
		return nil, fmt.Errorf("node not found")
	}
	return n, nil
}

func (s *MemoryStore) ListNodes(ctx context.Context, workflowID string) ([]*domain.Node, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var out []*domain.Node
	for _, n := range s.nodes {
		if n.WorkflowID() == workflowID {
			out = append(out, n)
		}
	}
	return out, nil
}

func (s *MemoryStore) SaveEdge(ctx context.Context, e *domain.Edge) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.edges[e.ID()] = e
	return nil
}

func (s *MemoryStore) GetEdge(ctx context.Context, id string) (*domain.Edge, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	e, ok := s.edges[id]
	if !ok {
		return nil, fmt.Errorf("edge not found")
	}
	return e, nil
}

func (s *MemoryStore) ListEdges(ctx context.Context, workflowID string) ([]*domain.Edge, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var out []*domain.Edge
	for _, e := range s.edges {
		if e.WorkflowID() == workflowID {
			out = append(out, e)
		}
	}
	return out, nil
}

func (s *MemoryStore) SaveTrigger(ctx context.Context, t *domain.Trigger) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.triggers[t.ID()] = t
	return nil
}

func (s *MemoryStore) GetTrigger(ctx context.Context, id string) (*domain.Trigger, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	t, ok := s.triggers[id]
	if !ok {
		return nil, fmt.Errorf("trigger not found")
	}
	return t, nil
}

func (s *MemoryStore) ListTriggers(ctx context.Context, workflowID string) ([]*domain.Trigger, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var out []*domain.Trigger
	for _, t := range s.triggers {
		if t.WorkflowID() == workflowID {
			out = append(out, t)
		}
	}
	return out, nil
}

// ExecutionState storage methods

func (s *MemoryStore) SaveExecutionState(ctx context.Context, state *domain.ExecutionState) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Create a deep copy to avoid external modifications
	// Serialize and deserialize to ensure clean copy with JSON handling
	stateCopy := s.cloneExecutionState(state)
	s.executionStates[state.ExecutionID()] = stateCopy
	return nil
}

func (s *MemoryStore) GetExecutionState(ctx context.Context, executionID string) (*domain.ExecutionState, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	state, ok := s.executionStates[executionID]
	if !ok {
		return nil, fmt.Errorf("execution state not found")
	}
	// Return a copy to avoid external modifications
	return s.cloneExecutionState(state), nil
}

func (s *MemoryStore) DeleteExecutionState(ctx context.Context, executionID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.executionStates, executionID)
	return nil
}

// cloneExecutionState creates a deep copy of ExecutionState
// This ensures external modifications don't affect stored state
func (s *MemoryStore) cloneExecutionState(state *domain.ExecutionState) *domain.ExecutionState {
	// Copy variables
	variables := make(map[string]interface{})
	for k, v := range state.Variables() {
		variables[k] = v
	}

	// Copy node states
	nodeStates := make(map[string]*domain.NodeState)
	for nodeID, ns := range state.NodeStates() {
		var startedAt, finishedAt *time.Time
		if ns.StartedAt() != nil {
			t := *ns.StartedAt()
			startedAt = &t
		}
		if ns.FinishedAt() != nil {
			t := *ns.FinishedAt()
			finishedAt = &t
		}

		nodeStates[nodeID] = domain.ReconstructNodeState(
			ns.NodeID(),
			ns.Status(),
			startedAt,
			finishedAt,
			ns.Output(),
			ns.ErrorMessage(),
			ns.AttemptNumber(),
			ns.MaxAttempts(),
		)
	}

	var finishedAt *time.Time
	if state.FinishedAt() != nil {
		t := *state.FinishedAt()
		finishedAt = &t
	}

	return domain.ReconstructExecutionState(
		state.ExecutionID(),
		state.WorkflowID(),
		state.Status(),
		variables,
		nodeStates,
		state.StartedAt(),
		finishedAt,
		state.ErrorMessage(),
	)
}
