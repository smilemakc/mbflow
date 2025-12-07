package engine

import (
	"sync"
	"time"

	"github.com/smilemakc/mbflow/internal/application/observer"
	"github.com/smilemakc/mbflow/pkg/models"
)

// ExecutionOptions configures execution behavior
type ExecutionOptions struct {
	StrictMode       bool                      // Fail on missing template variables
	MaxParallelism   int                       // Max concurrent nodes per wave (0 = unlimited)
	Timeout          time.Duration             // Overall execution timeout
	NodeTimeout      time.Duration             // Per-node execution timeout
	Variables        map[string]interface{}    // Runtime execution variables (override workflow vars)
	ObserverManager  *observer.ObserverManager // Optional observer manager for execution events
	RetryPolicy      *RetryPolicy              // Retry policy for node execution failures
	ContinueOnError  bool                      // Continue executing other nodes even if some fail
	MaxOutputSize    int64                     // Maximum size of node output in bytes (0 = unlimited)
	MaxTotalMemory   int64                     // Maximum total memory for all node outputs (0 = unlimited)
	EnableMemoryOpts bool                      // Enable automatic memory optimization
}

// ExecutionState tracks runtime state of workflow execution
type ExecutionState struct {
	ExecutionID string
	WorkflowID  string
	Workflow    *models.Workflow
	Input       map[string]interface{}
	Variables   map[string]interface{} // Merged workflow + execution vars

	// Node execution tracking
	NodeOutputs         map[string]interface{}                // nodeID -> output
	NodeInputs          map[string]interface{}                // nodeID -> input (passed to executor)
	NodeErrors          map[string]error                      // nodeID -> error
	NodeStatus          map[string]models.NodeExecutionStatus // nodeID -> status
	NodeStartTimes      map[string]time.Time                  // nodeID -> start time
	NodeEndTimes        map[string]time.Time                  // nodeID -> end time
	NodeConfigs         map[string]map[string]interface{}     // nodeID -> original config (before template resolution)
	NodeResolvedConfigs map[string]map[string]interface{}     // nodeID -> resolved config (after template resolution)

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
		StrictMode:       false,
		MaxParallelism:   10,
		Timeout:          5 * time.Minute,
		NodeTimeout:      1 * time.Minute,
		Variables:        make(map[string]interface{}),
		RetryPolicy:      NoRetryPolicy(), // No retries by default
		ContinueOnError:  false,           // Fail fast by default
		MaxOutputSize:    0,               // Unlimited by default
		MaxTotalMemory:   0,               // Unlimited by default
		EnableMemoryOpts: false,
	}
}

// NewExecutionState creates a new execution state
func NewExecutionState(executionID, workflowID string, workflow *models.Workflow, input, variables map[string]interface{}) *ExecutionState {
	return &ExecutionState{
		ExecutionID:         executionID,
		WorkflowID:          workflowID,
		Workflow:            workflow,
		Input:               input,
		Variables:           variables,
		NodeOutputs:         make(map[string]interface{}),
		NodeInputs:          make(map[string]interface{}),
		NodeErrors:          make(map[string]error),
		NodeStatus:          make(map[string]models.NodeExecutionStatus),
		NodeStartTimes:      make(map[string]time.Time),
		NodeEndTimes:        make(map[string]time.Time),
		NodeConfigs:         make(map[string]map[string]interface{}),
		NodeResolvedConfigs: make(map[string]map[string]interface{}),
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

// SetNodeInput safely sets node input
func (es *ExecutionState) SetNodeInput(nodeID string, input interface{}) {
	es.mu.Lock()
	defer es.mu.Unlock()
	es.NodeInputs[nodeID] = input
}

// GetNodeInput safely gets node input
func (es *ExecutionState) GetNodeInput(nodeID string) (interface{}, bool) {
	es.mu.RLock()
	defer es.mu.RUnlock()
	input, ok := es.NodeInputs[nodeID]
	return input, ok
}

// SetNodeConfig safely sets node original config
func (es *ExecutionState) SetNodeConfig(nodeID string, config map[string]interface{}) {
	es.mu.Lock()
	defer es.mu.Unlock()
	es.NodeConfigs[nodeID] = config
}

// GetNodeConfig safely gets node original config
func (es *ExecutionState) GetNodeConfig(nodeID string) (map[string]interface{}, bool) {
	es.mu.RLock()
	defer es.mu.RUnlock()
	config, ok := es.NodeConfigs[nodeID]
	return config, ok
}

// SetNodeResolvedConfig safely sets node resolved config
func (es *ExecutionState) SetNodeResolvedConfig(nodeID string, config map[string]interface{}) {
	es.mu.Lock()
	defer es.mu.Unlock()
	es.NodeResolvedConfigs[nodeID] = config
}

// GetNodeResolvedConfig safely gets node resolved config
func (es *ExecutionState) GetNodeResolvedConfig(nodeID string) (map[string]interface{}, bool) {
	es.mu.RLock()
	defer es.mu.RUnlock()
	config, ok := es.NodeResolvedConfigs[nodeID]
	return config, ok
}

// ClearNodeOutput removes output for a specific node (for memory optimization)
func (es *ExecutionState) ClearNodeOutput(nodeID string) {
	es.mu.Lock()
	defer es.mu.Unlock()
	delete(es.NodeOutputs, nodeID)
}

// GetTotalMemoryUsage estimates total memory used by node outputs (rough estimate)
func (es *ExecutionState) GetTotalMemoryUsage() int64 {
	es.mu.RLock()
	defer es.mu.RUnlock()

	var total int64
	for _, output := range es.NodeOutputs {
		total += estimateSize(output)
	}
	return total
}

// estimateSize provides a rough estimate of memory size for an interface{}
func estimateSize(v interface{}) int64 {
	switch val := v.(type) {
	case string:
		return int64(len(val))
	case []byte:
		return int64(len(val))
	case map[string]interface{}:
		var size int64
		for k, v := range val {
			size += int64(len(k)) + estimateSize(v)
		}
		return size
	case []interface{}:
		var size int64
		for _, item := range val {
			size += estimateSize(item)
		}
		return size
	default:
		// Rough estimate for other types
		return 64
	}
}
