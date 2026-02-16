package engine

import (
	"encoding/json"
	"sync"
	"time"

	"github.com/smilemakc/mbflow/pkg/models"
)

// ExecutionState tracks runtime state of workflow execution.
// Thread-safe via RWMutex. Used by both standalone and full engine modes.
type ExecutionState struct {
	ExecutionID string
	WorkflowID  string
	Workflow    *models.Workflow
	Input       map[string]interface{}
	Variables   map[string]interface{}
	Resources   map[string]interface{} // alias -> resource data for template resolution

	// Node execution tracking
	NodeOutputs         map[string]interface{}                // nodeID -> output
	NodeInputs          map[string]interface{}                // nodeID -> input (passed to executor)
	NodeErrors          map[string]error                      // nodeID -> error
	NodeStatus          map[string]models.NodeExecutionStatus // nodeID -> status
	NodeStartTimes      map[string]time.Time                  // nodeID -> start time
	NodeEndTimes        map[string]time.Time                  // nodeID -> end time
	NodeConfigs         map[string]map[string]interface{}     // nodeID -> original config
	NodeResolvedConfigs map[string]map[string]interface{}     // nodeID -> resolved config

	// Loop tracking
	LoopIterations map[string]int         // edgeID -> iteration count
	LoopInputs     map[string]interface{} // nodeID -> loop input override

	mu sync.RWMutex
}

// NewExecutionState creates a new execution state.
func NewExecutionState(executionID, workflowID string, workflow *models.Workflow, input, variables map[string]interface{}) *ExecutionState {
	return &ExecutionState{
		ExecutionID:         executionID,
		WorkflowID:          workflowID,
		Workflow:            workflow,
		Input:               input,
		Variables:           variables,
		Resources:           make(map[string]interface{}),
		NodeOutputs:         make(map[string]interface{}),
		NodeInputs:          make(map[string]interface{}),
		NodeErrors:          make(map[string]error),
		NodeStatus:          make(map[string]models.NodeExecutionStatus),
		NodeStartTimes:      make(map[string]time.Time),
		NodeEndTimes:        make(map[string]time.Time),
		NodeConfigs:         make(map[string]map[string]interface{}),
		NodeResolvedConfigs: make(map[string]map[string]interface{}),
		LoopIterations:      make(map[string]int),
		LoopInputs:          make(map[string]interface{}),
	}
}

// SetNodeOutput safely sets node output.
func (es *ExecutionState) SetNodeOutput(nodeID string, output interface{}) {
	es.mu.Lock()
	defer es.mu.Unlock()
	es.NodeOutputs[nodeID] = output
}

// GetNodeOutput safely gets node output.
func (es *ExecutionState) GetNodeOutput(nodeID string) (interface{}, bool) {
	es.mu.RLock()
	defer es.mu.RUnlock()
	output, ok := es.NodeOutputs[nodeID]
	return output, ok
}

// SetNodeError safely sets node error.
func (es *ExecutionState) SetNodeError(nodeID string, err error) {
	es.mu.Lock()
	defer es.mu.Unlock()
	es.NodeErrors[nodeID] = err
}

// GetNodeError safely gets node error.
func (es *ExecutionState) GetNodeError(nodeID string) (error, bool) {
	es.mu.RLock()
	defer es.mu.RUnlock()
	err, ok := es.NodeErrors[nodeID]
	return err, ok
}

// SetNodeStatus safely sets node status.
func (es *ExecutionState) SetNodeStatus(nodeID string, status models.NodeExecutionStatus) {
	es.mu.Lock()
	defer es.mu.Unlock()
	es.NodeStatus[nodeID] = status
}

// GetNodeStatus safely gets node status.
func (es *ExecutionState) GetNodeStatus(nodeID string) (models.NodeExecutionStatus, bool) {
	es.mu.RLock()
	defer es.mu.RUnlock()
	status, ok := es.NodeStatus[nodeID]
	return status, ok
}

// SetNodeStartTime safely sets node start time.
func (es *ExecutionState) SetNodeStartTime(nodeID string, t time.Time) {
	es.mu.Lock()
	defer es.mu.Unlock()
	es.NodeStartTimes[nodeID] = t
}

// GetNodeStartTime safely gets node start time.
func (es *ExecutionState) GetNodeStartTime(nodeID string) (time.Time, bool) {
	es.mu.RLock()
	defer es.mu.RUnlock()
	t, ok := es.NodeStartTimes[nodeID]
	return t, ok
}

// SetNodeEndTime safely sets node end time.
func (es *ExecutionState) SetNodeEndTime(nodeID string, t time.Time) {
	es.mu.Lock()
	defer es.mu.Unlock()
	es.NodeEndTimes[nodeID] = t
}

// GetNodeEndTime safely gets node end time.
func (es *ExecutionState) GetNodeEndTime(nodeID string) (time.Time, bool) {
	es.mu.RLock()
	defer es.mu.RUnlock()
	t, ok := es.NodeEndTimes[nodeID]
	return t, ok
}

// SetNodeInput safely sets node input.
func (es *ExecutionState) SetNodeInput(nodeID string, input interface{}) {
	es.mu.Lock()
	defer es.mu.Unlock()
	es.NodeInputs[nodeID] = input
}

// GetNodeInput safely gets node input.
func (es *ExecutionState) GetNodeInput(nodeID string) (interface{}, bool) {
	es.mu.RLock()
	defer es.mu.RUnlock()
	input, ok := es.NodeInputs[nodeID]
	return input, ok
}

// SetNodeConfig safely sets node original config.
func (es *ExecutionState) SetNodeConfig(nodeID string, config map[string]interface{}) {
	es.mu.Lock()
	defer es.mu.Unlock()
	es.NodeConfigs[nodeID] = config
}

// GetNodeConfig safely gets node original config.
func (es *ExecutionState) GetNodeConfig(nodeID string) (map[string]interface{}, bool) {
	es.mu.RLock()
	defer es.mu.RUnlock()
	config, ok := es.NodeConfigs[nodeID]
	return config, ok
}

// SetNodeResolvedConfig safely sets node resolved config.
func (es *ExecutionState) SetNodeResolvedConfig(nodeID string, config map[string]interface{}) {
	es.mu.Lock()
	defer es.mu.Unlock()
	es.NodeResolvedConfigs[nodeID] = config
}

// GetNodeResolvedConfig safely gets node resolved config.
func (es *ExecutionState) GetNodeResolvedConfig(nodeID string) (map[string]interface{}, bool) {
	es.mu.RLock()
	defer es.mu.RUnlock()
	config, ok := es.NodeResolvedConfigs[nodeID]
	return config, ok
}

// GetLoopIteration returns the current iteration count for a loop edge.
func (es *ExecutionState) GetLoopIteration(edgeID string) int {
	es.mu.RLock()
	defer es.mu.RUnlock()
	return es.LoopIterations[edgeID]
}

// IncrementLoopIteration increments and returns the new iteration count for a loop edge.
func (es *ExecutionState) IncrementLoopIteration(edgeID string) int {
	es.mu.Lock()
	defer es.mu.Unlock()
	es.LoopIterations[edgeID]++
	return es.LoopIterations[edgeID]
}

// SetLoopInput sets a loop input override for a node.
func (es *ExecutionState) SetLoopInput(nodeID string, input interface{}) {
	es.mu.Lock()
	defer es.mu.Unlock()
	es.LoopInputs[nodeID] = input
}

// GetLoopInput returns the loop input for a node, if any.
func (es *ExecutionState) GetLoopInput(nodeID string) (interface{}, bool) {
	es.mu.RLock()
	defer es.mu.RUnlock()
	input, ok := es.LoopInputs[nodeID]
	return input, ok
}

// ClearLoopInput removes the loop input for a node.
func (es *ExecutionState) ClearLoopInput(nodeID string) {
	es.mu.Lock()
	defer es.mu.Unlock()
	delete(es.LoopInputs, nodeID)
}

// ResetNodeForLoop clears all execution state for a node so it can be re-executed in a loop.
func (es *ExecutionState) ResetNodeForLoop(nodeID string) {
	es.mu.Lock()
	defer es.mu.Unlock()
	delete(es.NodeOutputs, nodeID)
	delete(es.NodeInputs, nodeID)
	delete(es.NodeErrors, nodeID)
	delete(es.NodeStatus, nodeID)
	delete(es.NodeStartTimes, nodeID)
	delete(es.NodeEndTimes, nodeID)
	delete(es.NodeConfigs, nodeID)
	delete(es.NodeResolvedConfigs, nodeID)
}

// ClearNodeOutput removes output for a specific node (for memory optimization).
func (es *ExecutionState) ClearNodeOutput(nodeID string) {
	es.mu.Lock()
	defer es.mu.Unlock()
	delete(es.NodeOutputs, nodeID)
}

// GetTotalMemoryUsage estimates total memory used by node outputs.
func (es *ExecutionState) GetTotalMemoryUsage() int64 {
	es.mu.RLock()
	defer es.mu.RUnlock()

	var total int64
	for _, output := range es.NodeOutputs {
		total += EstimateSize(output)
	}
	return total
}

// ToMapInterface converts any value to map[string]interface{}.
// Fast path for already-map values, JSON roundtrip for structs.
func ToMapInterface(v interface{}) map[string]interface{} {
	if v == nil {
		return nil
	}
	if m, ok := v.(map[string]interface{}); ok {
		return m
	}
	data, err := json.Marshal(v)
	if err != nil {
		return map[string]interface{}{"value": v}
	}
	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		return map[string]interface{}{"value": v}
	}
	return result
}
