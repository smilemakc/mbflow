package engine

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/smilemakc/mbflow/internal/application/observer"
	"github.com/smilemakc/mbflow/internal/domain/repository"
	storagemodels "github.com/smilemakc/mbflow/internal/infrastructure/storage/models"
	"github.com/smilemakc/mbflow/pkg/executor"
	"github.com/smilemakc/mbflow/pkg/models"
)

// ExecutionManager manages workflow execution lifecycle
type ExecutionManager struct {
	executorManager executor.Manager
	workflowRepo    repository.WorkflowRepository
	executionRepo   repository.ExecutionRepository
	eventRepo       repository.EventRepository
	dagExecutor     *DAGExecutor
	observerManager *observer.ObserverManager
}

// NewExecutionManager creates a new execution manager
func NewExecutionManager(
	executorManager executor.Manager,
	workflowRepo repository.WorkflowRepository,
	executionRepo repository.ExecutionRepository,
	eventRepo repository.EventRepository,
	observerManager *observer.ObserverManager,
) *ExecutionManager {
	nodeExecutor := NewNodeExecutor(executorManager)
	dagExecutor := NewDAGExecutor(nodeExecutor, observerManager)

	return &ExecutionManager{
		executorManager: executorManager,
		workflowRepo:    workflowRepo,
		executionRepo:   executionRepo,
		eventRepo:       eventRepo,
		dagExecutor:     dagExecutor,
		observerManager: observerManager,
	}
}

// Execute executes a workflow
func (em *ExecutionManager) Execute(
	ctx context.Context,
	workflowID string,
	input map[string]interface{},
	opts *ExecutionOptions,
) (*models.Execution, error) {
	// Use default options if not provided
	if opts == nil {
		opts = DefaultExecutionOptions()
	}

	// 1. Load workflow
	workflowUUID, err := uuid.Parse(workflowID)
	if err != nil {
		return nil, fmt.Errorf("invalid workflow ID: %w", err)
	}

	workflowModel, err := em.workflowRepo.FindByIDWithRelations(ctx, workflowUUID)
	if err != nil {
		return nil, fmt.Errorf("failed to load workflow: %w", err)
	}

	// Convert storage model to domain model
	workflow := WorkflowModelToDomain(workflowModel)

	// 2. Create execution record
	execution := &models.Execution{
		ID:           uuid.New().String(),
		WorkflowID:   workflow.ID,
		WorkflowName: workflow.Name,
		Status:       models.ExecutionStatusRunning,
		Input:        input,
		Variables:    em.mergeVariables(workflow.Variables, opts.Variables),
		StartedAt:    time.Now(),
	}

	// Convert to storage model and save execution
	executionModel := ExecutionDomainToModel(execution)
	if err := em.executionRepo.Create(ctx, executionModel); err != nil {
		return nil, fmt.Errorf("failed to create execution: %w", err)
	}

	// Notify execution started
	if em.observerManager != nil {
		event := observer.Event{
			Type:        observer.EventTypeExecutionStarted,
			ExecutionID: execution.ID,
			WorkflowID:  execution.WorkflowID,
			Timestamp:   execution.StartedAt,
			Status:      string(execution.Status),
			Input:       execution.Input,
			Variables:   execution.Variables,
		}
		em.observerManager.Notify(ctx, event)
	}

	// 3. Build execution state
	execState := NewExecutionState(
		execution.ID,
		workflow.ID,
		workflow,
		input,
		execution.Variables,
	)

	// 4. Execute DAG
	execErr := em.dagExecutor.Execute(ctx, execState, opts)

	// 5. Update execution with results
	now := time.Now()
	execution.CompletedAt = &now
	execution.Duration = execution.CalculateDuration()

	if execErr != nil {
		execution.Status = models.ExecutionStatusFailed
		execution.Error = execErr.Error()
	} else {
		execution.Status = models.ExecutionStatusCompleted
		// Set output to final node's output
		execution.Output = em.getFinalOutput(execState)
	}

	// Build node executions (need workflow model for UUID mapping)
	execution.NodeExecutions = em.buildNodeExecutions(execState, workflow, workflowModel)

	// Convert to storage model and update execution
	executionModel = ExecutionDomainToModel(execution)
	if err := em.executionRepo.Update(ctx, executionModel); err != nil {
		return nil, fmt.Errorf("failed to update execution: %w", err)
	}

	// Notify execution completion
	if em.observerManager != nil {
		duration := execution.Duration
		eventType := observer.EventTypeExecutionCompleted
		if execErr != nil {
			eventType = observer.EventTypeExecutionFailed
		}

		event := observer.Event{
			Type:        eventType,
			ExecutionID: execution.ID,
			WorkflowID:  execution.WorkflowID,
			Timestamp:   time.Now(),
			Status:      string(execution.Status),
			Output:      execution.Output,
			DurationMs:  &duration,
			Variables:   execution.Variables,
		}

		if execErr != nil {
			event.Error = execErr
		}

		em.observerManager.Notify(ctx, event)
	}

	return execution, execErr
}

// mergeVariables merges workflow and execution variables.
// Execution variables override workflow variables.
func (em *ExecutionManager) mergeVariables(
	workflowVars map[string]interface{},
	executionVars map[string]interface{},
) map[string]interface{} {
	merged := make(map[string]interface{})

	// Copy workflow variables
	for k, v := range workflowVars {
		merged[k] = v
	}

	// Execution variables override workflow variables
	for k, v := range executionVars {
		merged[k] = v
	}

	return merged
}

// getFinalOutput gets output from leaf nodes (nodes with no outgoing edges)
func (em *ExecutionManager) getFinalOutput(execState *ExecutionState) map[string]interface{} {
	// Find leaf nodes (nodes with no outgoing edges)
	leafNodes := em.findLeafNodes(execState.Workflow)

	if len(leafNodes) == 0 {
		return nil
	}

	// If single leaf, return its output
	if len(leafNodes) == 1 {
		if output, ok := execState.GetNodeOutput(leafNodes[0].ID); ok {
			if outputMap, ok := output.(map[string]interface{}); ok {
				return outputMap
			}
		}
	}

	// Multiple leaves - merge outputs namespaced by node ID
	merged := make(map[string]interface{})
	for _, node := range leafNodes {
		if output, ok := execState.GetNodeOutput(node.ID); ok {
			merged[node.ID] = output
		}
	}

	return merged
}

// findLeafNodes finds nodes with no outgoing edges
func (em *ExecutionManager) findLeafNodes(workflow *models.Workflow) []*models.Node {
	hasOutgoing := make(map[string]bool)
	for _, edge := range workflow.Edges {
		hasOutgoing[edge.From] = true
	}

	leaves := []*models.Node{}
	for _, node := range workflow.Nodes {
		if !hasOutgoing[node.ID] {
			leaves = append(leaves, node)
		}
	}

	return leaves
}

// buildNodeExecutions builds NodeExecution records from execution state
func (em *ExecutionManager) buildNodeExecutions(
	execState *ExecutionState,
	workflow *models.Workflow,
	workflowModel *storagemodels.WorkflowModel,
) []*models.NodeExecution {
	// Build map from logical ID to UUID
	logicalToUUID := make(map[string]string)
	for _, nodeModel := range workflowModel.Nodes {
		logicalToUUID[nodeModel.NodeID] = nodeModel.ID.String()
	}

	nodeExecs := make([]*models.NodeExecution, 0, len(workflow.Nodes))

	for _, node := range workflow.Nodes {
		// Get the UUID for this logical node ID
		nodeUUID, ok := logicalToUUID[node.ID]
		if !ok {
			// Skip nodes that don't have a UUID mapping
			continue
		}

		nodeExec := &models.NodeExecution{
			ID:          uuid.New().String(),
			ExecutionID: execState.ExecutionID,
			NodeID:      nodeUUID, // Use UUID instead of logical ID
			NodeName:    node.Name,
			NodeType:    node.Type,
		}

		// Get status
		if status, ok := execState.GetNodeStatus(node.ID); ok {
			nodeExec.Status = status
		}

		// Get output
		if output, ok := execState.GetNodeOutput(node.ID); ok {
			if outputMap, ok := output.(map[string]interface{}); ok {
				nodeExec.Output = outputMap
			}
		}

		// Get error
		if err, ok := execState.GetNodeError(node.ID); ok {
			nodeExec.Error = err.Error()
		}

		// Get timestamps
		if startTime, ok := execState.GetNodeStartTime(node.ID); ok {
			nodeExec.StartedAt = startTime
		}
		if endTime, ok := execState.GetNodeEndTime(node.ID); ok {
			nodeExec.CompletedAt = &endTime
		}

		nodeExecs = append(nodeExecs, nodeExec)
	}

	return nodeExecs
}
