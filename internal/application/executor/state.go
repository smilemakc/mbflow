package executor

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/smilemakc/mbflow/internal/domain"
)

// ExecutionState represents the state of a workflow execution.
// It manages the execution context, variables, and node states.
type ExecutionState struct {
	// ExecutionID is the unique identifier for this execution
	ExecutionID uuid.UUID
	// WorkflowID is the ID of the workflow being executed
	WorkflowID uuid.UUID
	// Status is the current status of the execution
	Status ExecutionStatus
	// Variables stores the execution variables (output from nodes)
	Variables map[string]interface{}
	// NodeStates tracks the state of each node
	NodeStates map[uuid.UUID]*NodeState
	// StartedAt is when the execution started
	StartedAt time.Time
	// FinishedAt is when the execution finished (nil if still running)
	FinishedAt *time.Time
	// Error stores any execution error
	Error error
	// repository is optional; when set, state will be persisted on every change
	// repository domain.ExecutionStateRepository
	// ctx is the context for persistence operations
	ctx context.Context
	// mu protects concurrent access to the state
	mu sync.RWMutex
}

// ExecutionStatus represents the status of an execution.
type ExecutionStatus string

const (
	// ExecutionStatusPending indicates the execution is pending
	ExecutionStatusPending ExecutionStatus = "pending"
	// ExecutionStatusRunning indicates the execution is running
	ExecutionStatusRunning ExecutionStatus = "running"
	// ExecutionStatusCompleted indicates the execution completed successfully
	ExecutionStatusCompleted ExecutionStatus = "completed"
	// ExecutionStatusFailed indicates the execution failed
	ExecutionStatusFailed ExecutionStatus = "failed"
	// ExecutionStatusCancelled indicates the execution was cancelled
	ExecutionStatusCancelled ExecutionStatus = "cancelled"
)

// NodeState represents the state of a single node execution.
type NodeState struct {
	// NodeID is the ID of the node
	NodeID uuid.UUID
	// Status is the current status of the node
	Status NodeStatus
	// StartedAt is when the node started executing
	StartedAt *time.Time
	// FinishedAt is when the node finished executing
	FinishedAt *time.Time
	// Output is the output from the node
	Output interface{}
	// Error stores any node execution error
	Error error
	// AttemptNumber is the current attempt number (for retries)
	AttemptNumber int
	// MaxAttempts is the maximum number of attempts allowed
	MaxAttempts int
}

// NodeStatus represents the status of a node execution.
type NodeStatus string

const (
	// NodeStatusPending indicates the node is pending execution
	NodeStatusPending NodeStatus = "pending"
	// NodeStatusRunning indicates the node is currently executing
	NodeStatusRunning NodeStatus = "running"
	// NodeStatusCompleted indicates the node completed successfully
	NodeStatusCompleted NodeStatus = "completed"
	// NodeStatusFailed indicates the node failed
	NodeStatusFailed NodeStatus = "failed"
	// NodeStatusSkipped indicates the node was skipped
	NodeStatusSkipped NodeStatus = "skipped"
	// NodeStatusRetrying indicates the node is being retried
	NodeStatusRetrying NodeStatus = "retrying"
)

// NewExecutionState creates a new ExecutionState.
func NewExecutionState(executionID, workflowID uuid.UUID) *ExecutionState {
	return &ExecutionState{
		ExecutionID: executionID,
		WorkflowID:  workflowID,
		Status:      ExecutionStatusPending,
		Variables:   make(map[string]interface{}),
		NodeStates:  make(map[uuid.UUID]*NodeState),
		StartedAt:   time.Now(),
	}
}

// NewExecutionStateWithRepository creates a new ExecutionState with persistence support.
func NewExecutionStateWithRepository(ctx context.Context, executionID, workflowID uuid.UUID, repository any) *ExecutionState {
	return &ExecutionState{
		ExecutionID: executionID,
		WorkflowID:  workflowID,
		Status:      ExecutionStatusPending,
		Variables:   make(map[string]interface{}),
		NodeStates:  make(map[uuid.UUID]*NodeState),
		StartedAt:   time.Now(),
		// repository:  repository,
		ctx: ctx,
	}
}

// SetStatus sets the execution status.
func (s *ExecutionState) SetStatus(status ExecutionStatus) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Status = status
	if status == ExecutionStatusCompleted || status == ExecutionStatusFailed || status == ExecutionStatusCancelled {
		now := time.Now()
		s.FinishedAt = &now
	}
	s.persist()
}

// GetStatus returns the current execution status.
func (s *ExecutionState) GetStatus() ExecutionStatus {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.Status
}

// GetStatusString returns the current execution status as string.
func (s *ExecutionState) GetStatusString() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return string(s.Status)
}

// GetExecutionID returns the execution ID.
func (s *ExecutionState) GetExecutionID() uuid.UUID {
	return s.ExecutionID
}

// GetWorkflowID returns the workflow ID.
func (s *ExecutionState) GetWorkflowID() uuid.UUID {
	return s.WorkflowID
}

// SetVariable sets a variable in the execution context.
func (s *ExecutionState) SetVariable(key string, value interface{}) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Variables[key] = value

	s.persist()
}

// GetVariable retrieves a variable from the execution context.
func (s *ExecutionState) GetVariable(key string) (interface{}, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	val, ok := s.Variables[key]
	return val, ok
}

// GetAllVariables returns a copy of all variables.
func (s *ExecutionState) GetAllVariables() map[string]interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()

	vars := make(map[string]interface{}, len(s.Variables))
	for k, v := range s.Variables {
		vars[k] = v
	}
	return vars
}

// SetNodeState sets the state for a node.
func (s *ExecutionState) SetNodeState(nodeID uuid.UUID, state *NodeState) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.NodeStates[nodeID] = state

	s.persist()
}

// GetNodeState retrieves the state for a node.
func (s *ExecutionState) GetNodeState(nodeID uuid.UUID) (*NodeState, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	state, ok := s.NodeStates[nodeID]
	return state, ok
}

// InitializeNodeState initializes the state for a node if it doesn't exist.
func (s *ExecutionState) InitializeNodeState(nodeID uuid.UUID, maxAttempts int) *NodeState {
	s.mu.Lock()
	defer s.mu.Unlock()
	if state, ok := s.NodeStates[nodeID]; ok {
		return state
	}

	state := &NodeState{
		NodeID:        nodeID,
		Status:        NodeStatusPending,
		AttemptNumber: 0,
		MaxAttempts:   maxAttempts,
	}
	s.NodeStates[nodeID] = state

	s.persist()
	return state
}

// MarkNodeStarted marks a node as started.
func (s *ExecutionState) MarkNodeStarted(nodeID uuid.UUID) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if state, ok := s.NodeStates[nodeID]; ok {
		now := time.Now()
		state.StartedAt = &now
		state.Status = NodeStatusRunning
		state.AttemptNumber++
	}

	s.persist()
}

// MarkNodeCompleted marks a node as completed with output.
func (s *ExecutionState) MarkNodeCompleted(nodeID uuid.UUID, output interface{}) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if state, ok := s.NodeStates[nodeID]; ok {
		now := time.Now()
		state.FinishedAt = &now
		state.Status = NodeStatusCompleted
		state.Output = output
		state.Error = nil
	}

	s.persist()
}

// MarkNodeFailed marks a node as failed with an error.
func (s *ExecutionState) MarkNodeFailed(nodeID uuid.UUID, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if state, ok := s.NodeStates[nodeID]; ok {
		now := time.Now()
		state.FinishedAt = &now
		state.Status = NodeStatusFailed
		state.Error = err
	}

	s.persist()
}

// MarkNodeRetrying marks a node as retrying.
func (s *ExecutionState) MarkNodeRetrying(nodeID uuid.UUID) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if state, ok := s.NodeStates[nodeID]; ok {
		state.Status = NodeStatusRetrying
		state.FinishedAt = nil
	}

	s.persist()
}

// CanRetryNode checks if a node can be retried.
func (s *ExecutionState) CanRetryNode(nodeID uuid.UUID) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if state, ok := s.NodeStates[nodeID]; ok {
		return state.AttemptNumber < state.MaxAttempts
	}
	return false
}

// GetExecutionDuration returns the duration of the execution.
func (s *ExecutionState) GetExecutionDuration() time.Duration {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.FinishedAt != nil {
		return s.FinishedAt.Sub(s.StartedAt)
	}
	return time.Since(s.StartedAt)
}

// GetNodeDuration returns the duration of a node execution.
func (s *ExecutionState) GetNodeDuration(nodeID uuid.UUID) time.Duration {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if state, ok := s.NodeStates[nodeID]; ok {
		if state.StartedAt != nil {
			if state.FinishedAt != nil {
				return state.FinishedAt.Sub(*state.StartedAt)
			}
			return time.Since(*state.StartedAt)
		}
	}
	return 0
}

// Clone creates a deep copy of the execution state.
func (s *ExecutionState) Clone() *ExecutionState {
	s.mu.RLock()
	defer s.mu.RUnlock()

	clone := &ExecutionState{
		ExecutionID: s.ExecutionID,
		WorkflowID:  s.WorkflowID,
		Status:      s.Status,
		Variables:   make(map[string]interface{}),
		NodeStates:  make(map[uuid.UUID]*NodeState),
		StartedAt:   s.StartedAt,
		FinishedAt:  s.FinishedAt,
		Error:       s.Error,
	}

	// Copy variables
	for k, v := range s.Variables {
		clone.Variables[k] = v
	}

	// Copy node states
	for k, v := range s.NodeStates {
		nodeState := &NodeState{
			NodeID:        v.NodeID,
			Status:        v.Status,
			StartedAt:     v.StartedAt,
			FinishedAt:    v.FinishedAt,
			Output:        v.Output,
			Error:         v.Error,
			AttemptNumber: v.AttemptNumber,
			MaxAttempts:   v.MaxAttempts,
		}
		clone.NodeStates[k] = nodeState
	}

	return clone
}

// String returns a string representation of the execution state.
func (s *ExecutionState) String() string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	completedNodes := 0
	failedNodes := 0
	for _, state := range s.NodeStates {
		if state.Status == NodeStatusCompleted {
			completedNodes++
		} else if state.Status == NodeStatusFailed {
			failedNodes++
		}
	}

	return fmt.Sprintf("Execution[%s] Status=%s Nodes=%d/%d (completed=%d, failed=%d) Duration=%s",
		s.ExecutionID, s.Status, completedNodes+failedNodes, len(s.NodeStates),
		completedNodes, failedNodes, s.GetExecutionDuration())
}

// ExecutionContext provides context for node execution.
// It wraps the execution state and provides helper methods for node executors.
type ExecutionContext struct {
	ctx   context.Context
	state *ExecutionState
}

// NewExecutionContext creates a new ExecutionContext.
func NewExecutionContext(ctx context.Context, state *ExecutionState) *ExecutionContext {
	return &ExecutionContext{
		ctx:   ctx,
		state: state,
	}
}

// Context returns the underlying context.
func (ec *ExecutionContext) Context() context.Context {
	return ec.ctx
}

// State returns the execution state.
func (ec *ExecutionContext) State() *ExecutionState {
	return ec.state
}

// SetVariable sets a variable in the execution context.
func (ec *ExecutionContext) SetVariable(key string, value interface{}) {
	ec.state.SetVariable(key, value)
}

// GetVariable retrieves a variable from the execution context.
func (ec *ExecutionContext) GetVariable(key string) (interface{}, bool) {
	return ec.state.GetVariable(key)
}

// GetAllVariables returns all variables.
func (ec *ExecutionContext) GetAllVariables() map[string]interface{} {
	return ec.state.GetAllVariables()
}

// persist saves the execution state to storage if a repository is configured.
// Errors are logged but do not affect execution.
func (s *ExecutionState) persist() {
	// if s.repository == nil || s.ctx == nil {
	// 	return
	// }

	// Convert to domain ExecutionState
	// domainState := s.toDomainExecutionState()

	// Persist asynchronously to avoid blocking execution
	// go func() {
	// if err := s.repository.SaveExecutionState(s.ctx, domainState); err != nil {
	// 	slog.Warn("Failed to persist execution state",
	// 		"executionID", s.ExecutionID,
	// 		"error", err)
	// }
	// }()
}

// toDomainExecutionState converts the application ExecutionState to domain ExecutionState.
func (s *ExecutionState) toDomainExecutionState() *domain.ExecutionState {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Convert status
	var domainStatus domain.ExecutionStateStatus
	switch s.Status {
	case ExecutionStatusPending:
		domainStatus = domain.ExecutionStateStatusPending
	case ExecutionStatusRunning:
		domainStatus = domain.ExecutionStateStatusRunning
	case ExecutionStatusCompleted:
		domainStatus = domain.ExecutionStateStatusCompleted
	case ExecutionStatusFailed:
		domainStatus = domain.ExecutionStateStatusFailed
	case ExecutionStatusCancelled:
		domainStatus = domain.ExecutionStateStatusCancelled
	default:
		domainStatus = domain.ExecutionStateStatusPending
	}

	// Convert error to string
	errorMsg := ""
	if s.Error != nil {
		errorMsg = s.Error.Error()
	}

	// Convert NodeStates
	nodeStates := make(map[uuid.UUID]*domain.NodeState)
	for nodeID, ns := range s.NodeStates {
		var domainNodeStatus domain.NodeStateStatus
		switch ns.Status {
		case NodeStatusPending:
			domainNodeStatus = domain.NodeStateStatusPending
		case NodeStatusRunning:
			domainNodeStatus = domain.NodeStateStatusRunning
		case NodeStatusCompleted:
			domainNodeStatus = domain.NodeStateStatusCompleted
		case NodeStatusFailed:
			domainNodeStatus = domain.NodeStateStatusFailed
		case NodeStatusSkipped:
			domainNodeStatus = domain.NodeStateStatusSkipped
		case NodeStatusRetrying:
			domainNodeStatus = domain.NodeStateStatusRetrying
		default:
			domainNodeStatus = domain.NodeStateStatusPending
		}

		errorMsgNS := ""
		if ns.Error != nil {
			errorMsgNS = ns.Error.Error()
		}

		nodeStates[nodeID] = domain.ReconstructNodeState(
			ns.NodeID,
			domainNodeStatus,
			ns.StartedAt,
			ns.FinishedAt,
			ns.Output,
			errorMsgNS,
			ns.AttemptNumber,
			ns.MaxAttempts,
		)
	}

	return domain.ReconstructExecutionState(
		s.ExecutionID,
		s.WorkflowID,
		domainStatus,
		s.Variables,
		nodeStates,
		s.StartedAt,
		s.FinishedAt,
		errorMsg,
	)
}
