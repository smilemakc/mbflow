package engine

import (
	"sync"
	"time"

	"github.com/smilemakc/mbflow/internal/application/observer"
	"github.com/smilemakc/mbflow/pkg/models"
)

// ExecutionOptions configures execution behavior
type ExecutionOptions struct {
	StrictMode      bool                      // Fail on missing template variables
	MaxParallelism  int                       // Max concurrent nodes per wave (0 = unlimited)
	Timeout         time.Duration             // Overall execution timeout
	NodeTimeout     time.Duration             // Per-node execution timeout
	Variables       map[string]interface{}    // Runtime execution variables (override workflow vars)
	ObserverManager *observer.ObserverManager // Optional observer manager for execution events
}

// ExecutionState tracks runtime state of workflow execution
type ExecutionState struct {
	ExecutionID string
	WorkflowID  string
	Workflow    *models.Workflow
	Input       map[string]interface{}
	Variables   map[string]interface{} // Merged workflow + execution vars

	// Node execution tracking
	NodeOutputs    map[string]interface{}                // nodeID -> output
	NodeErrors     map[string]error                      // nodeID -> error
	NodeStatus     map[string]models.NodeExecutionStatus // nodeID -> status
	NodeStartTimes map[string]time.Time                  // nodeID -> start time
	NodeEndTimes   map[string]time.Time                  // nodeID -> end time

	// Synchronization
	mu sync.RWMutex
}

// NodeContext holds context for single node execution
type NodeContext struct {
	ExecutionID        string
	NodeID             string
	Node               *models.Node
	WorkflowVariables  map[string]interface{} // Variables from workflow definition
	ExecutionVariables map[string]interface{} // Runtime variables (override workflow vars)
	DirectParentOutput map[string]interface{} // Output from immediate parent (for {{input.field}})
	StrictMode         bool                   // Fail on missing template variables
}

// DefaultExecutionOptions returns default execution options
func DefaultExecutionOptions() *ExecutionOptions {
	return &ExecutionOptions{
		StrictMode:     false,
		MaxParallelism: 10,
		Timeout:        5 * time.Minute,
		NodeTimeout:    1 * time.Minute,
		Variables:      make(map[string]interface{}),
	}
}

// NewExecutionState creates a new execution state
func NewExecutionState(executionID, workflowID string, workflow *models.Workflow, input, variables map[string]interface{}) *ExecutionState {
	return &ExecutionState{
		ExecutionID:    executionID,
		WorkflowID:     workflowID,
		Workflow:       workflow,
		Input:          input,
		Variables:      variables,
		NodeOutputs:    make(map[string]interface{}),
		NodeErrors:     make(map[string]error),
		NodeStatus:     make(map[string]models.NodeExecutionStatus),
		NodeStartTimes: make(map[string]time.Time),
		NodeEndTimes:   make(map[string]time.Time),
	}
}

// SetNodeOutput safely sets node output
func (es *ExecutionState) SetNodeOutput(nodeID string, output interface{}) {
	es.mu.Lock()
	defer es.mu.Unlock()
	es.NodeOutputs[nodeID] = output
}

// SetNodeError safely sets node error
func (es *ExecutionState) SetNodeError(nodeID string, err error) {
	es.mu.Lock()
	defer es.mu.Unlock()
	es.NodeErrors[nodeID] = err
}

// SetNodeStatus safely sets node status
func (es *ExecutionState) SetNodeStatus(nodeID string, status models.NodeExecutionStatus) {
	es.mu.Lock()
	defer es.mu.Unlock()
	es.NodeStatus[nodeID] = status
}

// GetNodeOutput safely gets node output
func (es *ExecutionState) GetNodeOutput(nodeID string) (interface{}, bool) {
	es.mu.RLock()
	defer es.mu.RUnlock()
	output, ok := es.NodeOutputs[nodeID]
	return output, ok
}

// GetNodeError safely gets node error
func (es *ExecutionState) GetNodeError(nodeID string) (error, bool) {
	es.mu.RLock()
	defer es.mu.RUnlock()
	err, ok := es.NodeErrors[nodeID]
	return err, ok
}

// GetNodeStatus safely gets node status
func (es *ExecutionState) GetNodeStatus(nodeID string) (models.NodeExecutionStatus, bool) {
	es.mu.RLock()
	defer es.mu.RUnlock()
	status, ok := es.NodeStatus[nodeID]
	return status, ok
}

// SetNodeStartTime safely sets node start time
func (es *ExecutionState) SetNodeStartTime(nodeID string, startTime time.Time) {
	es.mu.Lock()
	defer es.mu.Unlock()
	es.NodeStartTimes[nodeID] = startTime
}

// SetNodeEndTime safely sets node end time
func (es *ExecutionState) SetNodeEndTime(nodeID string, endTime time.Time) {
	es.mu.Lock()
	defer es.mu.Unlock()
	es.NodeEndTimes[nodeID] = endTime
}

// GetNodeStartTime safely gets node start time
func (es *ExecutionState) GetNodeStartTime(nodeID string) (time.Time, bool) {
	es.mu.RLock()
	defer es.mu.RUnlock()
	startTime, ok := es.NodeStartTimes[nodeID]
	return startTime, ok
}

// GetNodeEndTime safely gets node end time
func (es *ExecutionState) GetNodeEndTime(nodeID string) (time.Time, bool) {
	es.mu.RLock()
	defer es.mu.RUnlock()
	endTime, ok := es.NodeEndTimes[nodeID]
	return endTime, ok
}
