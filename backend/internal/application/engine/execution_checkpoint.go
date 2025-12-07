package engine

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/smilemakc/mbflow/pkg/models"
)

// ExecutionCheckpoint represents a snapshot of execution state at a specific wave
type ExecutionCheckpoint struct {
	ExecutionID    string                                `json:"execution_id"`
	WorkflowID     string                                `json:"workflow_id"`
	WaveIndex      int                                   `json:"wave_index"`
	Timestamp      time.Time                             `json:"timestamp"`
	CompletedNodes []string                              `json:"completed_nodes"`
	NodeOutputs    map[string]interface{}                `json:"node_outputs"`
	NodeStatuses   map[string]models.NodeExecutionStatus `json:"node_statuses"`
	Variables      map[string]interface{}                `json:"variables"`
}

// CreateCheckpoint creates a checkpoint from current execution state
func CreateCheckpoint(execState *ExecutionState, waveIndex int) *ExecutionCheckpoint {
	execState.mu.RLock()
	defer execState.mu.RUnlock()

	// Collect completed nodes
	completedNodes := []string{}
	for nodeID, status := range execState.NodeStatus {
		if status == models.NodeExecutionStatusCompleted {
			completedNodes = append(completedNodes, nodeID)
		}
	}

	// Deep copy outputs and statuses
	outputs := make(map[string]interface{})
	for k, v := range execState.NodeOutputs {
		outputs[k] = v
	}

	statuses := make(map[string]models.NodeExecutionStatus)
	for k, v := range execState.NodeStatus {
		statuses[k] = v
	}

	variables := make(map[string]interface{})
	for k, v := range execState.Variables {
		variables[k] = v
	}

	return &ExecutionCheckpoint{
		ExecutionID:    execState.ExecutionID,
		WorkflowID:     execState.WorkflowID,
		WaveIndex:      waveIndex,
		Timestamp:      time.Now(),
		CompletedNodes: completedNodes,
		NodeOutputs:    outputs,
		NodeStatuses:   statuses,
		Variables:      variables,
	}
}

// RestoreFromCheckpoint restores execution state from a checkpoint
func RestoreFromCheckpoint(checkpoint *ExecutionCheckpoint, workflow *models.Workflow, input map[string]interface{}) *ExecutionState {
	execState := NewExecutionState(
		checkpoint.ExecutionID,
		checkpoint.WorkflowID,
		workflow,
		input,
		checkpoint.Variables,
	)

	// Restore node outputs and statuses
	execState.mu.Lock()
	for k, v := range checkpoint.NodeOutputs {
		execState.NodeOutputs[k] = v
	}
	for k, v := range checkpoint.NodeStatuses {
		execState.NodeStatus[k] = v
	}
	execState.mu.Unlock()

	return execState
}

// Serialize converts checkpoint to JSON
func (cp *ExecutionCheckpoint) Serialize() ([]byte, error) {
	return json.Marshal(cp)
}

// DeserializeCheckpoint creates a checkpoint from JSON
func DeserializeCheckpoint(data []byte) (*ExecutionCheckpoint, error) {
	var cp ExecutionCheckpoint
	if err := json.Unmarshal(data, &cp); err != nil {
		return nil, fmt.Errorf("failed to deserialize checkpoint: %w", err)
	}
	return &cp, nil
}

// ValidateCheckpoint validates that a checkpoint is compatible with a workflow
func ValidateCheckpoint(checkpoint *ExecutionCheckpoint, workflow *models.Workflow) error {
	if checkpoint.WorkflowID != workflow.ID {
		return fmt.Errorf("checkpoint workflow ID (%s) does not match workflow ID (%s)", checkpoint.WorkflowID, workflow.ID)
	}

	// Verify that all completed nodes exist in workflow
	nodeIDs := make(map[string]bool)
	for _, node := range workflow.Nodes {
		nodeIDs[node.ID] = true
	}

	for _, nodeID := range checkpoint.CompletedNodes {
		if !nodeIDs[nodeID] {
			return fmt.Errorf("checkpoint references non-existent node: %s", nodeID)
		}
	}

	return nil
}

// GetNextWaveIndex returns the wave index to resume from
func (cp *ExecutionCheckpoint) GetNextWaveIndex() int {
	return cp.WaveIndex + 1
}

// IsNodeCompleted checks if a node was completed in this checkpoint
func (cp *ExecutionCheckpoint) IsNodeCompleted(nodeID string) bool {
	for _, id := range cp.CompletedNodes {
		if id == nodeID {
			return true
		}
	}
	return false
}

// CheckpointManager manages checkpoint storage and retrieval
type CheckpointManager struct {
	checkpoints map[string]*ExecutionCheckpoint // executionID -> latest checkpoint
}

// NewCheckpointManager creates a new checkpoint manager
func NewCheckpointManager() *CheckpointManager {
	return &CheckpointManager{
		checkpoints: make(map[string]*ExecutionCheckpoint),
	}
}

// SaveCheckpoint stores a checkpoint
func (cm *CheckpointManager) SaveCheckpoint(checkpoint *ExecutionCheckpoint) {
	cm.checkpoints[checkpoint.ExecutionID] = checkpoint
}

// GetCheckpoint retrieves the latest checkpoint for an execution
func (cm *CheckpointManager) GetCheckpoint(executionID string) (*ExecutionCheckpoint, bool) {
	cp, ok := cm.checkpoints[executionID]
	return cp, ok
}

// DeleteCheckpoint removes a checkpoint
func (cm *CheckpointManager) DeleteCheckpoint(executionID string) {
	delete(cm.checkpoints, executionID)
}

// ListCheckpoints returns all checkpoints
func (cm *CheckpointManager) ListCheckpoints() []*ExecutionCheckpoint {
	checkpoints := make([]*ExecutionCheckpoint, 0, len(cm.checkpoints))
	for _, cp := range cm.checkpoints {
		checkpoints = append(checkpoints, cp)
	}
	return checkpoints
}
