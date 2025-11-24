package domain

import (
	"time"

	"github.com/google/uuid"
)

// NodeExecutionState represents the execution state of a single node within a workflow execution.
// This is a value object that tracks the lifecycle and metadata of node execution.
type NodeExecutionState struct {
	nodeID     uuid.UUID
	nodeName   string
	nodeType   NodeType
	status     NodeStatus
	startedAt  *time.Time
	finishedAt *time.Time
	retryCount int
	error      string
	output     map[string]any
	variables  map[string]any // Node-local variables
}

// NewNodeExecutionState creates a new node execution state in pending status
func NewNodeExecutionState(nodeID uuid.UUID, nodeName string, nodeType NodeType) *NodeExecutionState {
	return &NodeExecutionState{
		nodeID:     nodeID,
		nodeName:   nodeName,
		nodeType:   nodeType,
		status:     NodeStatusPending,
		retryCount: 0,
		output:     make(map[string]any),
		variables:  make(map[string]any),
	}
}

// NodeID returns the node ID
func (ns *NodeExecutionState) NodeID() uuid.UUID {
	return ns.nodeID
}

// NodeName returns the node name
func (ns *NodeExecutionState) NodeName() string {
	return ns.nodeName
}

// NodeType returns the node type
func (ns *NodeExecutionState) NodeType() NodeType {
	return ns.nodeType
}

// Status returns the current status
func (ns *NodeExecutionState) Status() NodeStatus {
	return ns.status
}

// StartedAt returns the start time
func (ns *NodeExecutionState) StartedAt() *time.Time {
	return ns.startedAt
}

// FinishedAt returns the finish time
func (ns *NodeExecutionState) FinishedAt() *time.Time {
	return ns.finishedAt
}

// RetryCount returns the number of retries
func (ns *NodeExecutionState) RetryCount() int {
	return ns.retryCount
}

// Error returns the error message if failed
func (ns *NodeExecutionState) Error() string {
	return ns.error
}

// Output returns the node output
func (ns *NodeExecutionState) Output() map[string]any {
	return ns.output
}

// Variables returns node-local variables
func (ns *NodeExecutionState) Variables() map[string]any {
	return ns.variables
}

// Duration returns the execution duration
func (ns *NodeExecutionState) Duration() time.Duration {
	if ns.startedAt == nil {
		return 0
	}
	if ns.finishedAt == nil {
		return time.Since(*ns.startedAt)
	}
	return ns.finishedAt.Sub(*ns.startedAt)
}

// IsTerminal returns true if the node is in a terminal state
func (ns *NodeExecutionState) IsTerminal() bool {
	return ns.status.IsTerminal()
}

// Start marks the node as running
func (ns *NodeExecutionState) Start() {
	now := time.Now()
	ns.status = NodeStatusRunning
	ns.startedAt = &now
}

// Complete marks the node as completed with output
func (ns *NodeExecutionState) Complete(output map[string]any) {
	now := time.Now()
	ns.status = NodeStatusCompleted
	ns.finishedAt = &now
	if output != nil {
		ns.output = output
	}
}

// Fail marks the node as failed with an error
func (ns *NodeExecutionState) Fail(errorMsg string) {
	now := time.Now()
	ns.status = NodeStatusFailed
	ns.finishedAt = &now
	ns.error = errorMsg
}

// Skip marks the node as skipped
func (ns *NodeExecutionState) Skip(reason string) {
	now := time.Now()
	ns.status = NodeStatusSkipped
	ns.finishedAt = &now
	ns.error = reason
}

// IncrementRetry increments the retry counter
func (ns *NodeExecutionState) IncrementRetry() {
	ns.retryCount++
}

// SetVariable sets a node-local variable
func (ns *NodeExecutionState) SetVariable(key string, value any) {
	ns.variables[key] = value
}

// GetVariable gets a node-local variable
func (ns *NodeExecutionState) GetVariable(key string) (any, bool) {
	value, exists := ns.variables[key]
	return value, exists
}

// Clone creates a copy of the node state
func (ns *NodeExecutionState) Clone() *NodeExecutionState {
	clone := &NodeExecutionState{
		nodeID:     ns.nodeID,
		nodeName:   ns.nodeName,
		nodeType:   ns.nodeType,
		status:     ns.status,
		retryCount: ns.retryCount,
		error:      ns.error,
		output:     make(map[string]any),
		variables:  make(map[string]any),
	}

	if ns.startedAt != nil {
		t := *ns.startedAt
		clone.startedAt = &t
	}

	if ns.finishedAt != nil {
		t := *ns.finishedAt
		clone.finishedAt = &t
	}

	for k, v := range ns.output {
		clone.output[k] = v
	}

	for k, v := range ns.variables {
		clone.variables[k] = v
	}

	return clone
}

// ToMap converts the node state to a map for serialization
func (ns *NodeExecutionState) ToMap() map[string]any {
	result := map[string]any{
		"node_id":     ns.nodeID.String(),
		"node_name":   ns.nodeName,
		"node_type":   ns.nodeType.String(),
		"status":      ns.status.String(),
		"retry_count": ns.retryCount,
		"output":      ns.output,
		"variables":   ns.variables,
	}

	if ns.startedAt != nil {
		result["started_at"] = ns.startedAt.Format(time.RFC3339)
	}

	if ns.finishedAt != nil {
		result["finished_at"] = ns.finishedAt.Format(time.RFC3339)
		result["duration_ms"] = ns.Duration().Milliseconds()
	}

	if ns.error != "" {
		result["error"] = ns.error
	}

	return result
}
