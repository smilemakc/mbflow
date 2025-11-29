package domain

import (
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
)

// Execution is an aggregate root that represents a workflow execution instance.
// It uses Event Sourcing pattern - all state changes are recorded as events,
// and the current state is derived by replaying those events.
// This enables full audit trail, debugging, and state recovery.
type Execution interface {
	// Identity
	ID() uuid.UUID
	WorkflowID() uuid.UUID
	TriggerID() uuid.UUID

	// State
	Phase() ExecutionPhase
	StartedAt() time.Time
	FinishedAt() *time.Time
	Duration() time.Duration

	// Variables
	Variables() *VariableSet
	GetVariable(key string) (any, bool)
	GlobalVariables() *VariableSet
	SetGlobalVariable(key string, value any) error
	GetNodeOutput(nodeID uuid.UUID) (*VariableSet, bool)
	SetNodeOutput(nodeID uuid.UUID, output map[string]any) error

	// Node states
	GetNodeState(nodeID uuid.UUID) (*NodeExecutionState, bool)
	GetAllNodeStates() map[uuid.UUID]*NodeExecutionState

	// Error handling
	Error() string
	HasError() bool

	// Event sourcing
	GetUncommittedEvents() []Event
	MarkEventsAsCommitted()
	ApplyEvent(event Event) error

	// Commands - These generate events
	Start(triggerID uuid.UUID, initialVariables map[string]any) error
	StartNode(nodeID uuid.UUID, nodeName string, nodeType NodeType, inputVars map[string]any) error
	CompleteNode(nodeID uuid.UUID, nodeName string, nodeType NodeType, output map[string]any, duration time.Duration) error
	FailNode(nodeID uuid.UUID, nodeName string, nodeType NodeType, errorMsg string, retryCount int) error
	SkipNode(nodeID uuid.UUID, nodeName string, reason string) error
	SetVariable(key string, value any, scope VariableScope, nodeID uuid.UUID) error
	Complete(finalVariables map[string]any) error
	Fail(errorMsg string, failedNodeID uuid.UUID) error
}

// execution is the internal implementation of Execution aggregate
type execution struct {
	mu sync.RWMutex

	// Identity
	id         uuid.UUID
	workflowID uuid.UUID
	triggerID  uuid.UUID

	// Current state (derived from events)
	phase      ExecutionPhase
	startedAt  time.Time
	finishedAt *time.Time
	error      string

	// Execution context
	variables       *VariableSet               // All variables (flattened view for backward compatibility)
	globalVariables *VariableSet               // Global context variables (read-only for nodes)
	nodeOutputs     map[uuid.UUID]*VariableSet // Per-node output tracking
	nodeStates      map[uuid.UUID]*NodeExecutionState

	// Event sourcing
	version           int64 // Current version (sequence number of last applied event)
	uncommittedEvents []Event
}

// NewExecution creates a new Execution instance.
// If id is uuid.Nil, a new UUID will be generated automatically.
func NewExecution(id, workflowID uuid.UUID) (Execution, error) {
	if id == uuid.Nil {
		id = uuid.New()
	}

	if workflowID == uuid.Nil {
		return nil, NewDomainError(
			ErrCodeInvalidInput,
			"workflow ID cannot be nil",
			nil,
		)
	}

	return &execution{
		id:                id,
		workflowID:        workflowID,
		phase:             ExecutionPhasePlanning,
		variables:         NewVariableSet(nil),
		globalVariables:   NewVariableSet(nil),
		nodeOutputs:       make(map[uuid.UUID]*VariableSet),
		nodeStates:        make(map[uuid.UUID]*NodeExecutionState),
		version:           0,
		uncommittedEvents: make([]Event, 0),
	}, nil
}

// RebuildFromEvents reconstructs an Execution from its event history
func RebuildFromEvents(id, workflowID uuid.UUID, events []Event) (Execution, error) {
	exec, err := NewExecution(id, workflowID)
	if err != nil {
		return nil, err
	}

	// Apply all events to rebuild state
	for _, event := range events {
		if err := exec.ApplyEvent(event); err != nil {
			return nil, fmt.Errorf("failed to apply event %s: %w", event.EventID(), err)
		}
	}

	// Clear uncommitted events after rebuilding (these are historical)
	impl := exec.(*execution)
	impl.uncommittedEvents = make([]Event, 0)

	return exec, nil
}

// ID returns the execution ID
func (e *execution) ID() uuid.UUID {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.id
}

// WorkflowID returns the workflow ID
func (e *execution) WorkflowID() uuid.UUID {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.workflowID
}

// TriggerID returns the trigger ID that started this execution
func (e *execution) TriggerID() uuid.UUID {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.triggerID
}

// Phase returns the current execution phase
func (e *execution) Phase() ExecutionPhase {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.phase
}

// StartedAt returns the start timestamp
func (e *execution) StartedAt() time.Time {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.startedAt
}

// FinishedAt returns the finish timestamp
func (e *execution) FinishedAt() *time.Time {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.finishedAt
}

// Duration returns the execution duration
func (e *execution) Duration() time.Duration {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.calculateDuration()
}

// calculateDuration is an internal helper that calculates duration without locking
// Used by methods that already hold the lock
func (e *execution) calculateDuration() time.Duration {
	if e.startedAt.IsZero() {
		return 0
	}
	if e.finishedAt == nil {
		return time.Since(e.startedAt)
	}
	return e.finishedAt.Sub(e.startedAt)
}

// Variables returns the variable set
func (e *execution) Variables() *VariableSet {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.variables
}

// GetVariable gets a variable value
func (e *execution) GetVariable(key string) (any, bool) {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.variables.Get(key)
}

// GlobalVariables returns the global context variables (read-only for nodes)
func (e *execution) GlobalVariables() *VariableSet {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.globalVariables
}

// SetGlobalVariable sets a global variable (should only be called during initialization)
func (e *execution) SetGlobalVariable(key string, value any) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.globalVariables.Set(key, value)
}

// GetNodeOutput gets the output variables from a specific node
func (e *execution) GetNodeOutput(nodeID uuid.UUID) (*VariableSet, bool) {
	e.mu.RLock()
	defer e.mu.RUnlock()
	output, exists := e.nodeOutputs[nodeID]
	return output, exists
}

// SetNodeOutput sets the output variables for a specific node
func (e *execution) SetNodeOutput(nodeID uuid.UUID, output map[string]any) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	// Create a new VariableSet for this node's output
	varSet := NewVariableSet(nil)
	for k, v := range output {
		if err := varSet.Set(k, v); err != nil {
			return err
		}
	}
	e.nodeOutputs[nodeID] = varSet

	// Also merge into global variables for backward compatibility
	for k, v := range output {
		_ = e.variables.Set(k, v)
	}

	return nil
}

// GetNodeState gets the state of a node
func (e *execution) GetNodeState(nodeID uuid.UUID) (*NodeExecutionState, bool) {
	e.mu.RLock()
	defer e.mu.RUnlock()
	state, exists := e.nodeStates[nodeID]
	return state, exists
}

// GetAllNodeStates returns all node states
func (e *execution) GetAllNodeStates() map[uuid.UUID]*NodeExecutionState {
	e.mu.RLock()
	defer e.mu.RUnlock()

	result := make(map[uuid.UUID]*NodeExecutionState, len(e.nodeStates))
	for k, v := range e.nodeStates {
		result[k] = v.Clone()
	}
	return result
}

// Error returns the error message
func (e *execution) Error() string {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.error
}

// HasError returns true if execution has an error
func (e *execution) HasError() bool {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.error != ""
}

// GetUncommittedEvents returns events that haven't been persisted yet
func (e *execution) GetUncommittedEvents() []Event {
	e.mu.RLock()
	defer e.mu.RUnlock()

	result := make([]Event, len(e.uncommittedEvents))
	copy(result, e.uncommittedEvents)
	return result
}

// MarkEventsAsCommitted clears the uncommitted events list
func (e *execution) MarkEventsAsCommitted() {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.uncommittedEvents = make([]Event, 0)
}

// raiseEvent adds an event to uncommitted list and applies it
func (e *execution) raiseEvent(event Event) error {
	// Apply the event to update state
	if err := e.applyEventInternal(event); err != nil {
		return err
	}

	// Add to uncommitted events
	e.uncommittedEvents = append(e.uncommittedEvents, event)

	return nil
}

// ApplyEvent applies an event to the execution state (for rebuilding from history)
func (e *execution) ApplyEvent(event Event) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.applyEventInternal(event)
}

// applyEventInternal applies an event without locking (internal use)
func (e *execution) applyEventInternal(event Event) error {
	// Increment version
	e.version = event.SequenceNumber()

	// Apply event based on type
	switch event.EventType() {
	case EventTypeExecutionStarted:
		return e.applyExecutionStarted(event)
	case EventTypeExecutionCompleted:
		return e.applyExecutionCompleted(event)
	case EventTypeExecutionFailed:
		return e.applyExecutionFailed(event)
	case EventTypeNodeStarted:
		return e.applyNodeStarted(event)
	case EventTypeNodeCompleted:
		return e.applyNodeCompleted(event)
	case EventTypeNodeFailed:
		return e.applyNodeFailed(event)
	case EventTypeNodeSkipped:
		return e.applyNodeSkipped(event)
	case EventTypeVariableSet:
		return e.applyVariableSet(event)
	default:
		// Unknown event type - log but don't fail
		return nil
	}
}

// Command implementations - These generate events

// Start starts the execution
func (e *execution) Start(triggerID uuid.UUID, initialVariables map[string]any) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if e.phase != ExecutionPhasePlanning {
		return NewDomainError(
			ErrCodeInvalidState,
			fmt.Sprintf("cannot start execution in phase %s", e.phase),
			nil,
		)
	}

	event := NewExecutionStartedEvent(
		e.id,
		e.workflowID,
		e.version+1,
		triggerID,
		initialVariables,
	)

	return e.raiseEvent(event)
}

// StartNode starts a node execution
func (e *execution) StartNode(nodeID uuid.UUID, nodeName string, nodeType NodeType, inputVars map[string]any) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if e.phase != ExecutionPhaseExecuting {
		return NewDomainError(
			ErrCodeInvalidState,
			fmt.Sprintf("cannot start node in phase %s", e.phase),
			nil,
		)
	}

	event := NewNodeStartedEvent(
		e.id,
		e.workflowID,
		nodeID,
		e.version+1,
		nodeName,
		nodeType,
		inputVars,
	)

	return e.raiseEvent(event)
}

// CompleteNode marks a node as completed
func (e *execution) CompleteNode(nodeID uuid.UUID, nodeName string, nodeType NodeType, output map[string]any, duration time.Duration) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	event := NewNodeCompletedEvent(
		e.id,
		e.workflowID,
		nodeID,
		e.version+1,
		nodeName,
		nodeType,
		output,
		duration,
	)

	return e.raiseEvent(event)
}

// FailNode marks a node as failed
func (e *execution) FailNode(nodeID uuid.UUID, nodeName string, nodeType NodeType, errorMsg string, retryCount int) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	event := NewNodeFailedEvent(
		e.id,
		e.workflowID,
		nodeID,
		e.version+1,
		nodeName,
		nodeType,
		errorMsg,
		retryCount,
	)

	return e.raiseEvent(event)
}

// SkipNode marks a node as skipped
func (e *execution) SkipNode(nodeID uuid.UUID, nodeName string, reason string) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	event := NewNodeSkippedEvent(
		e.id,
		e.workflowID,
		nodeID,
		e.version+1,
		nodeName,
		reason,
	)

	return e.raiseEvent(event)
}

// SetVariable sets a variable
func (e *execution) SetVariable(key string, value any, scope VariableScope, nodeID uuid.UUID) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	event := NewVariableSetEvent(
		e.id,
		e.workflowID,
		nodeID,
		e.version+1,
		key,
		value,
		string(scope),
	)

	return e.raiseEvent(event)
}

// Complete marks the execution as completed
func (e *execution) Complete(finalVariables map[string]any) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if e.phase.IsTerminal() {
		return NewDomainError(
			ErrCodeInvalidState,
			fmt.Sprintf("execution already in terminal phase %s", e.phase),
			nil,
		)
	}

	event := NewExecutionCompletedEvent(
		e.id,
		e.workflowID,
		e.version+1,
		finalVariables,
		e.calculateDuration(),
	)

	return e.raiseEvent(event)
}

// Fail marks the execution as failed
func (e *execution) Fail(errorMsg string, failedNodeID uuid.UUID) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if e.phase.IsTerminal() {
		return NewDomainError(
			ErrCodeInvalidState,
			fmt.Sprintf("execution already in terminal phase %s", e.phase),
			nil,
		)
	}

	event := NewExecutionFailedEvent(
		e.id,
		e.workflowID,
		e.version+1,
		errorMsg,
		failedNodeID,
	)

	return e.raiseEvent(event)
}

// Event application methods

func (e *execution) applyExecutionStarted(event Event) error {
	data := event.Data()

	if triggerID, ok := data["trigger_id"].(string); ok {
		e.triggerID = uuid.MustParse(triggerID)
	}

	if initialVars, ok := data["initial_variables"].(map[string]any); ok {
		for k, v := range initialVars {
			// Set in both global and execution variables
			_ = e.globalVariables.Set(k, v)
			_ = e.variables.Set(k, v)
		}
	}

	// Mark global variables as read-only after initialization
	e.globalVariables.SetReadOnly(true)

	e.phase = ExecutionPhaseExecuting
	e.startedAt = event.Timestamp()

	return nil
}

func (e *execution) applyExecutionCompleted(event Event) error {
	e.phase = ExecutionPhaseCompleted
	t := event.Timestamp()
	e.finishedAt = &t

	return nil
}

func (e *execution) applyExecutionFailed(event Event) error {
	data := event.Data()

	if errorMsg, ok := data["error"].(string); ok {
		e.error = errorMsg
	}

	e.phase = ExecutionPhaseFailed
	t := event.Timestamp()
	e.finishedAt = &t

	return nil
}

func (e *execution) applyNodeStarted(event Event) error {
	nodeID := event.NodeID()
	data := event.Data()

	nodeName, _ := data["node_name"].(string)
	nodeTypeStr, _ := data["node_type"].(string)
	nodeType := NodeType(nodeTypeStr)

	// Create or get node state
	state, exists := e.nodeStates[nodeID]
	if !exists {
		state = NewNodeExecutionState(nodeID, nodeName, nodeType)
		e.nodeStates[nodeID] = state
	}

	state.Start()

	return nil
}

func (e *execution) applyNodeCompleted(event Event) error {
	nodeID := event.NodeID()
	data := event.Data()

	state, exists := e.nodeStates[nodeID]
	if !exists {
		return NewDomainError(
			ErrCodeNotFound,
			fmt.Sprintf("node state not found for node %s", nodeID),
			nil,
		)
	}

	output, _ := data["output"].(map[string]any)
	state.Complete(output)

	return nil
}

func (e *execution) applyNodeFailed(event Event) error {
	nodeID := event.NodeID()
	data := event.Data()

	state, exists := e.nodeStates[nodeID]
	if !exists {
		return NewDomainError(
			ErrCodeNotFound,
			fmt.Sprintf("node state not found for node %s", nodeID),
			nil,
		)
	}

	errorMsg, _ := data["error"].(string)
	state.Fail(errorMsg)

	if retryCount, ok := data["retry_count"].(int); ok {
		for i := 0; i < retryCount; i++ {
			state.IncrementRetry()
		}
	}

	return nil
}

func (e *execution) applyNodeSkipped(event Event) error {
	nodeID := event.NodeID()
	data := event.Data()

	reason, _ := data["reason"].(string)

	state, exists := e.nodeStates[nodeID]
	if !exists {
		// Create state if it doesn't exist
		nodeName, _ := data["node_name"].(string)
		nodeType, _ := data["node_type"].(string)
		if nodeType == "" {
			nodeType = string(NodeTypeTransform)
		}
		state = NewNodeExecutionState(nodeID, nodeName, NodeType(nodeType))
		e.nodeStates[nodeID] = state
	}

	state.Skip(reason)

	return nil
}

func (e *execution) applyVariableSet(event Event) error {
	data := event.Data()

	varName, _ := data["variable_name"].(string)
	value := data["value"]
	scope, _ := data["scope"].(string)

	nodeID := event.NodeID()

	// Set variable in appropriate scope
	if scope == string(ScopeNode) && nodeID != uuid.Nil {
		if state, exists := e.nodeStates[nodeID]; exists {
			state.SetVariable(varName, value)
		}
	} else {
		// Execution-level or workflow-level variables
		_ = e.variables.Set(varName, value)
	}

	return nil
}
