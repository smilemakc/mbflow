package engine

import (
	"sync"

	"github.com/smilemakc/mbflow/pkg/models"
)

// executionState tracks the state of a workflow execution.
type executionState struct {
	executionID string
	workflowID  string
	workflow    *models.Workflow
	input       map[string]interface{}
	variables   map[string]interface{}

	mu           sync.RWMutex
	nodeStatuses map[string]models.NodeExecutionStatus
	nodeOutputs  map[string]interface{}
	nodeErrors   map[string]error
}

// newExecutionState creates a new execution state.
func newExecutionState(
	executionID, workflowID string,
	workflow *models.Workflow,
	input, variables map[string]interface{},
) *executionState {
	return &executionState{
		executionID:  executionID,
		workflowID:   workflowID,
		workflow:     workflow,
		input:        input,
		variables:    variables,
		nodeStatuses: make(map[string]models.NodeExecutionStatus),
		nodeOutputs:  make(map[string]interface{}),
		nodeErrors:   make(map[string]error),
	}
}

func (s *executionState) setNodeStatus(nodeID string, status models.NodeExecutionStatus) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.nodeStatuses[nodeID] = status
}

func (s *executionState) getNodeStatus(nodeID string) (models.NodeExecutionStatus, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	status, ok := s.nodeStatuses[nodeID]
	return status, ok
}

func (s *executionState) setNodeOutput(nodeID string, output interface{}) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.nodeOutputs[nodeID] = output
}

func (s *executionState) getNodeOutput(nodeID string) (interface{}, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	output, ok := s.nodeOutputs[nodeID]
	return output, ok
}

func (s *executionState) setNodeError(nodeID string, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.nodeErrors[nodeID] = err
}

func (s *executionState) getNodeError(nodeID string) (error, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	err, ok := s.nodeErrors[nodeID]
	return err, ok
}
