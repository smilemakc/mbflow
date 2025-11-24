package storage

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/smilemakc/mbflow/internal/domain"
)

// MemoryStore is an in-memory implementation of the Storage interface
// Suitable for development, testing, and simple use cases
type MemoryStore struct {
	mu              sync.RWMutex
	workflows       map[uuid.UUID]domain.Workflow
	executionStates map[uuid.UUID]*domain.ExecutionState

	// Event store for event sourcing
	eventStore *MemoryEventStore

	// Deprecated: These fields are kept for backward compatibility
	// New code should use event sourcing via eventStore
	executions map[uuid.UUID]domain.Execution
	events     []domain.Event
	nodes      map[uuid.UUID]domain.Node
	edges      map[uuid.UUID]domain.Edge
	triggers   map[uuid.UUID]domain.Trigger
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		workflows:       make(map[uuid.UUID]domain.Workflow),
		executions:      make(map[uuid.UUID]domain.Execution),
		executionStates: make(map[uuid.UUID]*domain.ExecutionState),
		eventStore:      NewMemoryEventStore(),
		nodes:           make(map[uuid.UUID]domain.Node),
		edges:           make(map[uuid.UUID]domain.Edge),
		triggers:        make(map[uuid.UUID]domain.Trigger),
	}
}

func (s *MemoryStore) SaveWorkflow(ctx context.Context, w domain.Workflow) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.workflows[w.ID()] = w
	return nil
}

func (s *MemoryStore) GetWorkflow(ctx context.Context, id uuid.UUID) (domain.Workflow, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	w, ok := s.workflows[id]
	if !ok {
		return nil, fmt.Errorf("workflow not found")
	}
	return w, nil
}

func (s *MemoryStore) ListWorkflows(ctx context.Context) ([]domain.Workflow, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]domain.Workflow, 0, len(s.workflows))
	for _, w := range s.workflows {
		out = append(out, w)
	}
	return out, nil
}

func (s *MemoryStore) SaveExecution(ctx context.Context, x domain.Execution) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.executions[x.ID()] = x
	return nil
}

func (s *MemoryStore) GetExecution(ctx context.Context, id uuid.UUID) (domain.Execution, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	x, ok := s.executions[id]
	if !ok {
		return nil, fmt.Errorf("execution not found")
	}
	return x, nil
}

func (s *MemoryStore) ListExecutions(ctx context.Context) ([]domain.Execution, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]domain.Execution, 0, len(s.executions))
	for _, x := range s.executions {
		out = append(out, x)
	}
	return out, nil
}

func (s *MemoryStore) AppendEvent(ctx context.Context, ev domain.Event) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.events = append(s.events, ev)
	return nil
}

func (s *MemoryStore) ListEventsByExecution(ctx context.Context, executionID uuid.UUID) ([]domain.Event, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var out []domain.Event
	for _, ev := range s.events {
		if ev.ExecutionID() == executionID {
			out = append(out, ev)
		}
	}
	return out, nil
}

func (s *MemoryStore) SaveNode(ctx context.Context, n domain.Node) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.nodes[n.ID()] = n
	return nil
}

func (s *MemoryStore) GetNode(ctx context.Context, id uuid.UUID) (domain.Node, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	n, ok := s.nodes[id]
	if !ok {
		return nil, fmt.Errorf("node not found")
	}
	return n, nil
}

func (s *MemoryStore) ListNodes(ctx context.Context, workflowID uuid.UUID) ([]domain.Node, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var out []domain.Node
	for _, n := range s.nodes {
		out = append(out, n)
	}
	return out, nil
}

func (s *MemoryStore) SaveEdge(ctx context.Context, e domain.Edge) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.edges[e.ID()] = e
	return nil
}

func (s *MemoryStore) GetEdge(ctx context.Context, id uuid.UUID) (domain.Edge, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	e, ok := s.edges[id]
	if !ok {
		return nil, fmt.Errorf("edge not found")
	}
	return e, nil
}

func (s *MemoryStore) ListEdges(ctx context.Context, workflowID uuid.UUID) ([]domain.Edge, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var out []domain.Edge
	for _, e := range s.edges {
		out = append(out, e)
	}
	return out, nil
}

func (s *MemoryStore) SaveTrigger(ctx context.Context, t domain.Trigger) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.triggers[t.ID()] = t
	return nil
}

func (s *MemoryStore) GetTrigger(ctx context.Context, id uuid.UUID) (domain.Trigger, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	t, ok := s.triggers[id]
	if !ok {
		return nil, fmt.Errorf("trigger not found")
	}
	return t, nil
}

func (s *MemoryStore) ListTriggers(ctx context.Context, workflowID uuid.UUID) ([]domain.Trigger, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var out []domain.Trigger
	for _, t := range s.triggers {
		out = append(out, t)
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

func (s *MemoryStore) GetExecutionState(ctx context.Context, executionID uuid.UUID) (*domain.ExecutionState, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	state, ok := s.executionStates[executionID]
	if !ok {
		return nil, fmt.Errorf("execution state not found")
	}
	// Return a copy to avoid external modifications
	return s.cloneExecutionState(state), nil
}

func (s *MemoryStore) DeleteExecutionState(ctx context.Context, executionID uuid.UUID) error {
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
	nodeStates := make(map[uuid.UUID]*domain.NodeState)
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

// ========== EventStore interface implementation ==========

// AppendEvents appends multiple events atomically (delegates to eventStore)
func (s *MemoryStore) AppendEvents(ctx context.Context, events []domain.Event) error {
	return s.eventStore.AppendEvents(ctx, events)
}

// GetEvents retrieves all events for an execution (delegates to eventStore)
func (s *MemoryStore) GetEvents(ctx context.Context, executionID uuid.UUID) ([]domain.Event, error) {
	return s.eventStore.GetEvents(ctx, executionID)
}

// GetEventsSince retrieves events after a specific sequence number (delegates to eventStore)
func (s *MemoryStore) GetEventsSince(ctx context.Context, executionID uuid.UUID, sequenceNumber int64) ([]domain.Event, error) {
	return s.eventStore.GetEventsSince(ctx, executionID, sequenceNumber)
}

// GetEventsByType retrieves events of a specific type (delegates to eventStore)
func (s *MemoryStore) GetEventsByType(ctx context.Context, executionID uuid.UUID, eventType domain.EventType) ([]domain.Event, error) {
	return s.eventStore.GetEventsByType(ctx, executionID, eventType)
}

// GetEventsByWorkflow retrieves all events for a workflow (delegates to eventStore)
func (s *MemoryStore) GetEventsByWorkflow(ctx context.Context, workflowID uuid.UUID) ([]domain.Event, error) {
	return s.eventStore.GetEventsByWorkflow(ctx, workflowID)
}

// GetEventCount returns the number of events for an execution (delegates to eventStore)
func (s *MemoryStore) GetEventCount(ctx context.Context, executionID uuid.UUID) (int64, error) {
	return s.eventStore.GetEventCount(ctx, executionID)
}

// ========== New Storage interface methods ==========

// GetWorkflowByName retrieves a workflow by name and version
func (s *MemoryStore) GetWorkflowByName(ctx context.Context, name, version string) (domain.Workflow, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, w := range s.workflows {
		if w.Name() == name && w.Version() == version {
			return w, nil
		}
	}

	return nil, domain.NewDomainError(
		domain.ErrCodeNotFound,
		fmt.Sprintf("workflow %s:%s not found", name, version),
		nil,
	)
}

// DeleteWorkflow removes a workflow and all its child entities
func (s *MemoryStore) DeleteWorkflow(ctx context.Context, id uuid.UUID) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.workflows[id]; !exists {
		return domain.NewDomainError(
			domain.ErrCodeNotFound,
			fmt.Sprintf("workflow %s not found", id),
			nil,
		)
	}

	delete(s.workflows, id)
	return nil
}

// WorkflowExists checks if a workflow exists
func (s *MemoryStore) WorkflowExists(ctx context.Context, id uuid.UUID) (bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	_, exists := s.workflows[id]
	return exists, nil
}

// ListExecutionsByWorkflow returns all executions for a workflow
func (s *MemoryStore) ListExecutionsByWorkflow(ctx context.Context, workflowID uuid.UUID) ([]domain.Execution, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]domain.Execution, 0)
	for _, exec := range s.executions {
		if exec.WorkflowID() == workflowID {
			result = append(result, exec)
		}
	}

	return result, nil
}

// ListAllExecutions returns all executions (paginated)
func (s *MemoryStore) ListAllExecutions(ctx context.Context, limit, offset int) ([]domain.Execution, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Convert map to slice
	allExecutions := make([]domain.Execution, 0, len(s.executions))
	for _, exec := range s.executions {
		allExecutions = append(allExecutions, exec)
	}

	// Apply pagination
	start := offset
	if start > len(allExecutions) {
		return []domain.Execution{}, nil
	}

	end := start + limit
	if end > len(allExecutions) {
		end = len(allExecutions)
	}

	return allExecutions[start:end], nil
}

// SaveSnapshot saves a snapshot of execution state for performance
func (s *MemoryStore) SaveSnapshot(ctx context.Context, execution domain.Execution) error {
	// For in-memory store, snapshots are not needed
	// Just save the execution normally
	return s.SaveExecution(ctx, execution)
}

// GetSnapshot retrieves the latest snapshot if available
func (s *MemoryStore) GetSnapshot(ctx context.Context, id uuid.UUID) (domain.Execution, error) {
	// For in-memory store, just return the execution
	return s.GetExecution(ctx, id)
}

// ========== Transaction support ==========

// BeginTransaction begins a new transaction (no-op for memory store)
func (s *MemoryStore) BeginTransaction(ctx context.Context) (context.Context, error) {
	// Memory store doesn't support real transactions
	// Return the same context
	return ctx, nil
}

// CommitTransaction commits the current transaction (no-op for memory store)
func (s *MemoryStore) CommitTransaction(ctx context.Context) error {
	// Memory store doesn't support real transactions
	return nil
}

// RollbackTransaction rolls back the current transaction (no-op for memory store)
func (s *MemoryStore) RollbackTransaction(ctx context.Context) error {
	// Memory store doesn't support real transactions
	return nil
}

// ========== Health check ==========

// Ping checks if the storage is accessible
func (s *MemoryStore) Ping(ctx context.Context) error {
	// Memory store is always accessible
	return nil
}

// Close closes the storage connection (no-op for memory store)
func (s *MemoryStore) Close() error {
	// Nothing to close for memory store
	return nil
}
