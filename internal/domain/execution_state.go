package domain

import (
	"time"
)

// ExecutionStateStatus represents the status of an execution state.
type ExecutionStateStatus string

const (
	// ExecutionStateStatusPending indicates the execution is pending
	ExecutionStateStatusPending ExecutionStateStatus = "pending"
	// ExecutionStateStatusRunning indicates the execution is running
	ExecutionStateStatusRunning ExecutionStateStatus = "running"
	// ExecutionStateStatusCompleted indicates the execution completed successfully
	ExecutionStateStatusCompleted ExecutionStateStatus = "completed"
	// ExecutionStateStatusFailed indicates the execution failed
	ExecutionStateStatusFailed ExecutionStateStatus = "failed"
	// ExecutionStateStatusCancelled indicates the execution was cancelled
	ExecutionStateStatusCancelled ExecutionStateStatus = "cancelled"
)

// NodeStateStatus represents the status of a node execution.
type NodeStateStatus string

const (
	// NodeStateStatusPending indicates the node is pending execution
	NodeStateStatusPending NodeStateStatus = "pending"
	// NodeStateStatusRunning indicates the node is currently executing
	NodeStateStatusRunning NodeStateStatus = "running"
	// NodeStateStatusCompleted indicates the node completed successfully
	NodeStateStatusCompleted NodeStateStatus = "completed"
	// NodeStateStatusFailed indicates the node failed
	NodeStateStatusFailed NodeStateStatus = "failed"
	// NodeStateStatusSkipped indicates the node was skipped
	NodeStateStatusSkipped NodeStateStatus = "skipped"
	// NodeStateStatusRetrying indicates the node is being retried
	NodeStateStatusRetrying NodeStateStatus = "retrying"
)

// NodeState is a value object that represents the state of a single node execution.
// It encapsulates all information about a node's execution status, timing, output, and errors.
// This is part of the ExecutionState aggregate and is used to track individual node progress.
type NodeState struct {
	nodeID        string
	status        NodeStateStatus
	startedAt     *time.Time
	finishedAt    *time.Time
	output        interface{}
	errorMessage  string
	attemptNumber int
	maxAttempts   int
}

// NewNodeState creates a new NodeState instance.
func NewNodeState(nodeID string, status NodeStateStatus, startedAt, finishedAt *time.Time, output interface{}, errorMessage string, attemptNumber, maxAttempts int) *NodeState {
	return &NodeState{
		nodeID:        nodeID,
		status:        status,
		startedAt:     startedAt,
		finishedAt:    finishedAt,
		output:        output,
		errorMessage:  errorMessage,
		attemptNumber: attemptNumber,
		maxAttempts:   maxAttempts,
	}
}

// ReconstructNodeState reconstructs a NodeState from persistence.
func ReconstructNodeState(nodeID string, status NodeStateStatus, startedAt, finishedAt *time.Time, output interface{}, errorMessage string, attemptNumber, maxAttempts int) *NodeState {
	return &NodeState{
		nodeID:        nodeID,
		status:        status,
		startedAt:     startedAt,
		finishedAt:    finishedAt,
		output:        output,
		errorMessage:  errorMessage,
		attemptNumber: attemptNumber,
		maxAttempts:   maxAttempts,
	}
}

// NodeID returns the node ID.
func (ns *NodeState) NodeID() string {
	return ns.nodeID
}

// Status returns the node status.
func (ns *NodeState) Status() NodeStateStatus {
	return ns.status
}

// StartedAt returns when the node started executing.
func (ns *NodeState) StartedAt() *time.Time {
	return ns.startedAt
}

// FinishedAt returns when the node finished executing.
func (ns *NodeState) FinishedAt() *time.Time {
	return ns.finishedAt
}

// Output returns the output from the node.
func (ns *NodeState) Output() interface{} {
	return ns.output
}

// ErrorMessage returns the error message if the node failed.
func (ns *NodeState) ErrorMessage() string {
	return ns.errorMessage
}

// AttemptNumber returns the current attempt number.
func (ns *NodeState) AttemptNumber() int {
	return ns.attemptNumber
}

// MaxAttempts returns the maximum number of attempts allowed.
func (ns *NodeState) MaxAttempts() int {
	return ns.maxAttempts
}

// ExecutionState is a domain aggregate that manages the complete state of a workflow execution.
// It encapsulates execution variables, node states, status, and timing information.
// This aggregate is the root entity for execution state management and ensures
// consistency of the execution context throughout the workflow lifecycle.
// ExecutionState is separate from Execution entity which serves as a lightweight execution record.
type ExecutionState struct {
	executionID string
	workflowID  string
	status      ExecutionStateStatus
	variables   map[string]interface{}
	nodeStates  map[string]*NodeState
	startedAt   time.Time
	finishedAt  *time.Time
	errorMsg    string
}

// NewExecutionState creates a new ExecutionState instance.
func NewExecutionState(executionID, workflowID string) *ExecutionState {
	return &ExecutionState{
		executionID: executionID,
		workflowID:  workflowID,
		status:      ExecutionStateStatusPending,
		variables:   make(map[string]interface{}),
		nodeStates:  make(map[string]*NodeState),
		startedAt:   time.Now(),
	}
}

// ReconstructExecutionState reconstructs an ExecutionState from persistence.
func ReconstructExecutionState(executionID, workflowID string, status ExecutionStateStatus, variables map[string]interface{}, nodeStates map[string]*NodeState, startedAt time.Time, finishedAt *time.Time, errorMsg string) *ExecutionState {
	if variables == nil {
		variables = make(map[string]interface{})
	}
	if nodeStates == nil {
		nodeStates = make(map[string]*NodeState)
	}
	return &ExecutionState{
		executionID: executionID,
		workflowID:  workflowID,
		status:      status,
		variables:   variables,
		nodeStates:  nodeStates,
		startedAt:   startedAt,
		finishedAt:  finishedAt,
		errorMsg:    errorMsg,
	}
}

// ExecutionID returns the unique identifier for this execution.
func (es *ExecutionState) ExecutionID() string {
	return es.executionID
}

// WorkflowID returns the ID of the workflow being executed.
func (es *ExecutionState) WorkflowID() string {
	return es.workflowID
}

// Status returns the current status of the execution.
func (es *ExecutionState) Status() ExecutionStateStatus {
	return es.status
}

// Variables returns a copy of all execution variables.
func (es *ExecutionState) Variables() map[string]interface{} {
	vars := make(map[string]interface{}, len(es.variables))
	for k, v := range es.variables {
		vars[k] = v
	}
	return vars
}

// GetVariable retrieves a variable by key.
func (es *ExecutionState) GetVariable(key string) (interface{}, bool) {
	val, ok := es.variables[key]
	return val, ok
}

// SetVariable sets a variable in the execution context.
func (es *ExecutionState) SetVariable(key string, value interface{}) {
	if es.variables == nil {
		es.variables = make(map[string]interface{})
	}
	es.variables[key] = value
}

// NodeStates returns a copy of all node states.
func (es *ExecutionState) NodeStates() map[string]*NodeState {
	states := make(map[string]*NodeState, len(es.nodeStates))
	for k, v := range es.nodeStates {
		states[k] = v
	}
	return states
}

// GetNodeState retrieves the state for a node.
func (es *ExecutionState) GetNodeState(nodeID string) (*NodeState, bool) {
	state, ok := es.nodeStates[nodeID]
	return state, ok
}

// SetNodeState sets the state for a node.
func (es *ExecutionState) SetNodeState(nodeID string, state *NodeState) {
	if es.nodeStates == nil {
		es.nodeStates = make(map[string]*NodeState)
	}
	es.nodeStates[nodeID] = state
}

// StartedAt returns when the execution started.
func (es *ExecutionState) StartedAt() time.Time {
	return es.startedAt
}

// FinishedAt returns when the execution finished (nil if still running).
func (es *ExecutionState) FinishedAt() *time.Time {
	return es.finishedAt
}

// ErrorMessage returns the error message if the execution failed.
func (es *ExecutionState) ErrorMessage() string {
	return es.errorMsg
}

// SetStatus sets the execution status.
func (es *ExecutionState) SetStatus(status ExecutionStateStatus) {
	es.status = status
	if status == ExecutionStateStatusCompleted || status == ExecutionStateStatusFailed || status == ExecutionStateStatusCancelled {
		now := time.Now()
		es.finishedAt = &now
	}
}

// SetError sets the error message.
func (es *ExecutionState) SetError(errorMsg string) {
	es.errorMsg = errorMsg
}

// SetFinishedAt sets the finished timestamp.
func (es *ExecutionState) SetFinishedAt(finishedAt *time.Time) {
	es.finishedAt = finishedAt
}
