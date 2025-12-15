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

// Execute executes a workflow synchronously (blocks until completion)
func (em *ExecutionManager) Execute(
	ctx context.Context,
	workflowID string,
	input map[string]interface{},
	opts *ExecutionOptions,
) (*models.Execution, error) {
	// Prepare execution (load workflow, create record)
	execution, workflow, workflowModel, err := em.prepareExecution(ctx, workflowID, input, opts, models.ExecutionStatusRunning)
	if err != nil {
		return nil, err
	}

	// Notify execution started
	em.notifyExecutionStarted(ctx, execution)

	// Execute workflow DAG
	execState, execErr := em.executeWorkflowDAG(ctx, execution, workflow, opts)

	// Finalize execution (update status, save results)
	if err := em.finalizeExecution(ctx, execution, workflow, workflowModel, execState, execErr); err != nil {
		return nil, err
	}

	return execution, execErr
}

// ExecuteAsync executes a workflow asynchronously.
// It creates the execution record immediately and returns it,
// while the actual workflow execution happens in a background goroutine.
func (em *ExecutionManager) ExecuteAsync(
	ctx context.Context,
	workflowID string,
	input map[string]interface{},
	opts *ExecutionOptions,
) (*models.Execution, error) {
	// Prepare execution (load workflow, create record with PENDING status)
	execution, workflow, workflowModel, err := em.prepareExecution(ctx, workflowID, input, opts, models.ExecutionStatusPending)
	if err != nil {
		return nil, err
	}

	// Execute workflow in background goroutine
	go func() {
		bgCtx := context.Background()

		// Update status to running
		execution.Status = models.ExecutionStatusRunning
		executionModel := ExecutionDomainToModel(execution)
		if err := em.executionRepo.Update(bgCtx, executionModel); err != nil {
			em.notifyExecutionError(bgCtx, execution, fmt.Errorf("failed to update execution status: %w", err))
			return
		}

		// Notify execution started
		em.notifyExecutionStarted(bgCtx, execution)

		// Build execution state
		execState, execErr := em.executeWorkflowDAG(bgCtx, execution, workflow, opts)

		// Finalize execution
		if err := em.finalizeExecution(bgCtx, execution, workflow, workflowModel, execState, execErr); err != nil {
			em.notifyExecutionError(bgCtx, execution, fmt.Errorf("failed to finalize execution: %w", err))
			return
		}
	}()

	// Return execution immediately
	return execution, nil
}

// prepareExecution loads workflow and creates execution record
func (em *ExecutionManager) prepareExecution(
	ctx context.Context,
	workflowID string,
	input map[string]interface{},
	opts *ExecutionOptions,
	initialStatus models.ExecutionStatus,
) (*models.Execution, *models.Workflow, *storagemodels.WorkflowModel, error) {
	// Use default options if not provided
	if opts == nil {
		opts = DefaultExecutionOptions()
	}

	// Load workflow
	workflowUUID, err := uuid.Parse(workflowID)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("invalid workflow ID: %w", err)
	}

	workflowModel, err := em.workflowRepo.FindByIDWithRelations(ctx, workflowUUID)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to load workflow: %w", err)
	}

	workflow := WorkflowModelToDomain(workflowModel)

	// Create execution record
	execution := &models.Execution{
		ID:           uuid.New().String(),
		WorkflowID:   workflow.ID,
		WorkflowName: workflow.Name,
		Status:       initialStatus,
		Input:        input,
		Variables:    em.mergeVariables(workflow.Variables, opts.Variables),
		StartedAt:    time.Now(),
	}

	// Save execution
	executionModel := ExecutionDomainToModel(execution)
	if err := em.executionRepo.Create(ctx, executionModel); err != nil {
		return nil, nil, nil, fmt.Errorf("failed to create execution: %w", err)
	}

	return execution, workflow, workflowModel, nil
}

// executeWorkflowDAG executes the workflow DAG and returns execution state
func (em *ExecutionManager) executeWorkflowDAG(
	ctx context.Context,
	execution *models.Execution,
	workflow *models.Workflow,
	opts *ExecutionOptions,
) (*ExecutionState, error) {
	// Build execution state
	execState := NewExecutionState(
		execution.ID,
		workflow.ID,
		workflow,
		execution.Input,
		execution.Variables,
	)

	// Execute DAG
	execErr := em.dagExecutor.Execute(ctx, execState, opts)

	return execState, execErr
}

// finalizeExecution updates execution with results and saves to database
func (em *ExecutionManager) finalizeExecution(
	ctx context.Context,
	execution *models.Execution,
	workflow *models.Workflow,
	workflowModel *storagemodels.WorkflowModel,
	execState *ExecutionState,
	execErr error,
) error {
	// Update execution with results
	now := time.Now()
	execution.CompletedAt = &now
	execution.Duration = execution.CalculateDuration()

	if execErr != nil {
		execution.Status = models.ExecutionStatusFailed
		execution.Error = execErr.Error()
	} else {
		execution.Status = models.ExecutionStatusCompleted
		execution.Output = em.getFinalOutput(execState)
	}

	// Build node executions
	execution.NodeExecutions = em.buildNodeExecutions(execState, workflow, workflowModel)

	// Update execution in database
	executionModel := ExecutionDomainToModel(execution)
	if err := em.executionRepo.Update(ctx, executionModel); err != nil {
		return fmt.Errorf("failed to update execution: %w", err)
	}

	// Notify execution completion
	em.notifyExecutionCompletion(ctx, execution, execErr)

	return nil
}

// notifyExecutionStarted sends execution started event
func (em *ExecutionManager) notifyExecutionStarted(ctx context.Context, execution *models.Execution) {
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
}

// notifyExecutionCompletion sends execution completion event
func (em *ExecutionManager) notifyExecutionCompletion(ctx context.Context, execution *models.Execution, execErr error) {
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
}

// notifyExecutionError sends execution error event
func (em *ExecutionManager) notifyExecutionError(ctx context.Context, execution *models.Execution, err error) {
	if em.observerManager != nil {
		event := observer.Event{
			Type:        observer.EventTypeExecutionFailed,
			ExecutionID: execution.ID,
			WorkflowID:  execution.WorkflowID,
			Timestamp:   time.Now(),
			Error:       err,
		}
		em.observerManager.Notify(ctx, event)
	}
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
			return toMapInterface(output)
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

		// Get input
		if input, ok := execState.GetNodeInput(node.ID); ok {
			nodeExec.Input = toMapInterface(input)
		}

		// Get output
		if output, ok := execState.GetNodeOutput(node.ID); ok {
			nodeExec.Output = toMapInterface(output)
		}

		// Get config (original)
		if config, ok := execState.GetNodeConfig(node.ID); ok {
			nodeExec.Config = config
		}

		// Get resolved config
		if resolvedConfig, ok := execState.GetNodeResolvedConfig(node.ID); ok {
			nodeExec.ResolvedConfig = resolvedConfig
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
