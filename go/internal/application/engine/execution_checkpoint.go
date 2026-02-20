package engine

import (
	"encoding/json"
	"fmt"
	"time"

	pkgengine "github.com/smilemakc/mbflow/go/pkg/engine"
	"github.com/smilemakc/mbflow/go/pkg/models"
)

// ExecutionCheckpoint represents a snapshot of execution state at a specific wave.
type ExecutionCheckpoint struct {
	ExecutionID    string                                `json:"execution_id"`
	WorkflowID     string                                `json:"workflow_id"`
	WaveIndex      int                                   `json:"wave_index"`
	Timestamp      time.Time                             `json:"timestamp"`
	CompletedNodes []string                              `json:"completed_nodes"`
	NodeOutputs    map[string]any                        `json:"node_outputs"`
	NodeStatuses   map[string]models.NodeExecutionStatus `json:"node_statuses"`
	Variables      map[string]any                        `json:"variables"`
}

// CreateCheckpoint creates a checkpoint from current execution state.
func CreateCheckpoint(execState *pkgengine.ExecutionState, waveIndex int) *ExecutionCheckpoint {
	completedNodes := []string{}
	outputs := make(map[string]any)
	statuses := make(map[string]models.NodeExecutionStatus)
	variables := make(map[string]any)

	for _, node := range execState.Workflow.Nodes {
		if status, ok := execState.GetNodeStatus(node.ID); ok {
			statuses[node.ID] = status
			if status == models.NodeExecutionStatusCompleted {
				completedNodes = append(completedNodes, node.ID)
			}
		}
		if output, ok := execState.GetNodeOutput(node.ID); ok {
			outputs[node.ID] = output
		}
	}

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

// RestoreFromCheckpoint restores execution state from a checkpoint.
func RestoreFromCheckpoint(checkpoint *ExecutionCheckpoint, workflow *models.Workflow, input map[string]any) *pkgengine.ExecutionState {
	execState := pkgengine.NewExecutionState(
		checkpoint.ExecutionID,
		checkpoint.WorkflowID,
		workflow,
		input,
		checkpoint.Variables,
	)

	for k, v := range checkpoint.NodeOutputs {
		execState.SetNodeOutput(k, v)
	}
	for k, v := range checkpoint.NodeStatuses {
		execState.SetNodeStatus(k, v)
	}

	return execState
}

// Serialize converts checkpoint to JSON.
func (cp *ExecutionCheckpoint) Serialize() ([]byte, error) {
	return json.Marshal(cp)
}

// DeserializeCheckpoint creates a checkpoint from JSON.
func DeserializeCheckpoint(data []byte) (*ExecutionCheckpoint, error) {
	var cp ExecutionCheckpoint
	if err := json.Unmarshal(data, &cp); err != nil {
		return nil, fmt.Errorf("failed to deserialize checkpoint: %w", err)
	}
	return &cp, nil
}

// ValidateCheckpoint validates that a checkpoint is compatible with a workflow.
func ValidateCheckpoint(checkpoint *ExecutionCheckpoint, workflow *models.Workflow) error {
	if checkpoint.WorkflowID != workflow.ID {
		return fmt.Errorf("checkpoint workflow ID (%s) does not match workflow ID (%s)", checkpoint.WorkflowID, workflow.ID)
	}

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

// GetNextWaveIndex returns the wave index to resume from.
func (cp *ExecutionCheckpoint) GetNextWaveIndex() int {
	return cp.WaveIndex + 1
}

// IsNodeCompleted checks if a node was completed in this checkpoint.
func (cp *ExecutionCheckpoint) IsNodeCompleted(nodeID string) bool {
	for _, id := range cp.CompletedNodes {
		if id == nodeID {
			return true
		}
	}
	return false
}

// CheckpointManager manages checkpoint storage and retrieval.
type CheckpointManager struct {
	checkpoints map[string]*ExecutionCheckpoint
}

// NewCheckpointManager creates a new checkpoint manager.
func NewCheckpointManager() *CheckpointManager {
	return &CheckpointManager{
		checkpoints: make(map[string]*ExecutionCheckpoint),
	}
}

// SaveCheckpoint stores a checkpoint.
func (cm *CheckpointManager) SaveCheckpoint(checkpoint *ExecutionCheckpoint) {
	cm.checkpoints[checkpoint.ExecutionID] = checkpoint
}

// GetCheckpoint retrieves the latest checkpoint for an execution.
func (cm *CheckpointManager) GetCheckpoint(executionID string) (*ExecutionCheckpoint, bool) {
	cp, ok := cm.checkpoints[executionID]
	return cp, ok
}

// DeleteCheckpoint removes a checkpoint.
func (cm *CheckpointManager) DeleteCheckpoint(executionID string) {
	delete(cm.checkpoints, executionID)
}

// ListCheckpoints returns all checkpoints.
func (cm *CheckpointManager) ListCheckpoints() []*ExecutionCheckpoint {
	checkpoints := make([]*ExecutionCheckpoint, 0, len(cm.checkpoints))
	for _, cp := range cm.checkpoints {
		checkpoints = append(checkpoints, cp)
	}
	return checkpoints
}
