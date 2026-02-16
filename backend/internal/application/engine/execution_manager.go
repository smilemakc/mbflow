package engine

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/smilemakc/mbflow/internal/application/observer"
	"github.com/smilemakc/mbflow/internal/domain/repository"
	storagemodels "github.com/smilemakc/mbflow/internal/infrastructure/storage/models"
	pkgengine "github.com/smilemakc/mbflow/pkg/engine"
	"github.com/smilemakc/mbflow/pkg/executor"
	"github.com/smilemakc/mbflow/pkg/models"
)

// ExecutionManager manages workflow execution lifecycle.
type ExecutionManager struct {
	executorManager executor.Manager
	workflowRepo    repository.WorkflowRepository
	executionRepo   repository.ExecutionRepository
	eventRepo       repository.EventRepository
	resourceRepo    repository.ResourceRepository
	dagExecutor     *pkgengine.DAGExecutor
	observerManager *observer.ObserverManager
}

// NewExecutionManager creates a new execution manager.
func NewExecutionManager(
	executorManager executor.Manager,
	workflowRepo repository.WorkflowRepository,
	executionRepo repository.ExecutionRepository,
	eventRepo repository.EventRepository,
	resourceRepo repository.ResourceRepository,
	observerManager *observer.ObserverManager,
) *ExecutionManager {
	nodeExecutor := pkgengine.NewNodeExecutor(executorManager)
	notifier := NewObserverNotifier(observerManager)
	condEvaluator := pkgengine.NewExprConditionEvaluator()
	dagExecutor := pkgengine.NewDAGExecutor(nodeExecutor, condEvaluator, notifier, pkgengine.NewNilWorkflowLoader())

	return &ExecutionManager{
		executorManager: executorManager,
		workflowRepo:    workflowRepo,
		executionRepo:   executionRepo,
		eventRepo:       eventRepo,
		resourceRepo:    resourceRepo,
		dagExecutor:     dagExecutor,
		observerManager: observerManager,
	}
}

// Execute executes a workflow synchronously (blocks until completion).
func (em *ExecutionManager) Execute(
	ctx context.Context,
	workflowID string,
	input map[string]interface{},
	opts *ExecutionOptions,
) (*models.Execution, error) {
	execution, workflow, workflowModel, err := em.prepareExecution(ctx, workflowID, input, opts, models.ExecutionStatusRunning)
	if err != nil {
		return nil, err
	}

	em.notifyExecutionStarted(ctx, execution)

	execState, execErr := em.executeWorkflowDAG(ctx, execution, workflow, opts)

	if err := em.finalizeExecution(ctx, execution, workflow, workflowModel, execState, execErr); err != nil {
		return nil, err
	}

	return execution, execErr
}

// ExecuteAsync executes a workflow asynchronously.
func (em *ExecutionManager) ExecuteAsync(
	ctx context.Context,
	workflowID string,
	input map[string]interface{},
	opts *ExecutionOptions,
) (*models.Execution, error) {
	execution, workflow, workflowModel, err := em.prepareExecution(ctx, workflowID, input, opts, models.ExecutionStatusPending)
	if err != nil {
		return nil, err
	}

	go func() {
		bgCtx := context.Background()

		execution.Status = models.ExecutionStatusRunning
		executionModel := storagemodels.ExecutionDomainToModel(execution)
		if err := em.executionRepo.Update(bgCtx, executionModel); err != nil {
			em.notifyExecutionError(bgCtx, execution, fmt.Errorf("failed to update execution status: %w", err))
			return
		}

		em.notifyExecutionStarted(bgCtx, execution)

		execState, execErr := em.executeWorkflowDAG(bgCtx, execution, workflow, opts)

		if err := em.finalizeExecution(bgCtx, execution, workflow, workflowModel, execState, execErr); err != nil {
			em.notifyExecutionError(bgCtx, execution, fmt.Errorf("failed to finalize execution: %w", err))
			return
		}
	}()

	return execution, nil
}

// prepareExecution loads workflow and creates execution record.
func (em *ExecutionManager) prepareExecution(
	ctx context.Context,
	workflowID string,
	input map[string]interface{},
	opts *ExecutionOptions,
	initialStatus models.ExecutionStatus,
) (*models.Execution, *models.Workflow, *storagemodels.WorkflowModel, error) {
	if opts == nil {
		opts = DefaultExecutionOptions()
	}

	workflowUUID, err := uuid.Parse(workflowID)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("invalid workflow ID: %w", err)
	}

	workflowModel, err := em.workflowRepo.FindByIDWithRelations(ctx, workflowUUID)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to load workflow: %w", err)
	}

	workflow := storagemodels.WorkflowModelToDomain(workflowModel)

	execution := &models.Execution{
		ID:           uuid.New().String(),
		WorkflowID:   workflow.ID,
		WorkflowName: workflow.Name,
		Status:       initialStatus,
		Input:        input,
		Variables:    pkgengine.MergeVariables(workflow.Variables, opts.Variables),
		StartedAt:    time.Now(),
	}

	executionModel := storagemodels.ExecutionDomainToModel(execution)
	if err := em.executionRepo.Create(ctx, executionModel); err != nil {
		return nil, nil, nil, fmt.Errorf("failed to create execution: %w", err)
	}

	return execution, workflow, workflowModel, nil
}

// executeWorkflowDAG executes the workflow DAG and returns execution state.
func (em *ExecutionManager) executeWorkflowDAG(
	ctx context.Context,
	execution *models.Execution,
	workflow *models.Workflow,
	opts *ExecutionOptions,
) (*pkgengine.ExecutionState, error) {
	execState := pkgengine.NewExecutionState(
		execution.ID,
		workflow.ID,
		workflow,
		execution.Input,
		execution.Variables,
	)

	// Load and validate workflow resources
	if len(workflow.Resources) > 0 {
		resourceMap, err := em.loadAndValidateResources(ctx, workflow)
		if err != nil {
			return execState, err
		}
		execState.Resources = resourceMap
	}

	// Convert internal options to pkg options
	pkgOpts := convertToPkgOptions(opts)

	execErr := em.dagExecutor.Execute(ctx, execState, pkgOpts)

	return execState, execErr
}

// finalizeExecution updates execution with results and saves to database.
func (em *ExecutionManager) finalizeExecution(
	ctx context.Context,
	execution *models.Execution,
	workflow *models.Workflow,
	workflowModel *storagemodels.WorkflowModel,
	execState *pkgengine.ExecutionState,
	execErr error,
) error {
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

	execution.NodeExecutions = em.buildNodeExecutions(execState, workflow, workflowModel)

	executionModel := storagemodels.ExecutionDomainToModel(execution)
	if err := em.executionRepo.Update(ctx, executionModel); err != nil {
		return fmt.Errorf("failed to update execution: %w", err)
	}

	em.notifyExecutionCompletion(ctx, execution, execErr)

	return nil
}

// notifyExecutionStarted sends execution started event.
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

// notifyExecutionCompletion sends execution completion event.
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

// notifyExecutionError sends execution error event.
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

// getFinalOutput gets output from leaf nodes.
func (em *ExecutionManager) getFinalOutput(execState *pkgengine.ExecutionState) map[string]interface{} {
	leafNodes := pkgengine.FindLeafNodes(execState.Workflow)

	if len(leafNodes) == 0 {
		return nil
	}

	if len(leafNodes) == 1 {
		if output, ok := execState.GetNodeOutput(leafNodes[0].ID); ok {
			return pkgengine.ToMapInterface(output)
		}
	}

	merged := make(map[string]interface{})
	for _, node := range leafNodes {
		if output, ok := execState.GetNodeOutput(node.ID); ok {
			merged[node.ID] = output
		}
	}

	return merged
}

// buildNodeExecutions builds NodeExecution records from execution state.
func (em *ExecutionManager) buildNodeExecutions(
	execState *pkgengine.ExecutionState,
	workflow *models.Workflow,
	workflowModel *storagemodels.WorkflowModel,
) []*models.NodeExecution {
	logicalToUUID := make(map[string]string)
	for _, nodeModel := range workflowModel.Nodes {
		logicalToUUID[nodeModel.NodeID] = nodeModel.ID.String()
	}

	nodeExecs := make([]*models.NodeExecution, 0, len(workflow.Nodes))

	for _, node := range workflow.Nodes {
		nodeUUID, ok := logicalToUUID[node.ID]
		if !ok {
			continue
		}

		nodeExec := &models.NodeExecution{
			ID:          uuid.New().String(),
			ExecutionID: execState.ExecutionID,
			NodeID:      nodeUUID,
			NodeName:    node.Name,
			NodeType:    node.Type,
		}

		if status, ok := execState.GetNodeStatus(node.ID); ok {
			nodeExec.Status = status
		}

		if input, ok := execState.GetNodeInput(node.ID); ok {
			nodeExec.Input = pkgengine.ToMapInterface(input)
		}

		if output, ok := execState.GetNodeOutput(node.ID); ok {
			nodeExec.Output = pkgengine.ToMapInterface(output)
		}

		if config, ok := execState.GetNodeConfig(node.ID); ok {
			nodeExec.Config = config
		}

		if resolvedConfig, ok := execState.GetNodeResolvedConfig(node.ID); ok {
			nodeExec.ResolvedConfig = resolvedConfig
		}

		if err, ok := execState.GetNodeError(node.ID); ok {
			nodeExec.Error = err.Error()
		}

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

// loadAndValidateResources loads workflow resources and validates ownership.
func (em *ExecutionManager) loadAndValidateResources(
	ctx context.Context,
	workflow *models.Workflow,
) (map[string]interface{}, error) {
	resourceMap := make(map[string]interface{})

	for _, wr := range workflow.Resources {
		resource, err := em.resourceRepo.GetByID(ctx, wr.ResourceID)
		if err != nil {
			return nil, fmt.Errorf("failed to load resource %s (alias: %s): %w", wr.ResourceID, wr.Alias, err)
		}

		if workflow.CreatedBy != "" && resource.GetOwnerID() != workflow.CreatedBy {
			return nil, fmt.Errorf("resource access denied: resource %s (alias: %s) owner does not match workflow owner",
				wr.ResourceID, wr.Alias)
		}

		if resource.GetStatus() != models.ResourceStatusActive {
			return nil, fmt.Errorf("resource %s (alias: %s) is not active (status: %s)",
				wr.ResourceID, wr.Alias, resource.GetStatus())
		}

		resourceMap[wr.Alias] = map[string]interface{}{
			"id":          resource.GetID(),
			"name":        resource.GetName(),
			"type":        string(resource.GetType()),
			"access_type": wr.AccessType,
		}
	}

	return resourceMap, nil
}

// convertToPkgOptions converts internal ExecutionOptions to pkg ExecutionOptions.
func convertToPkgOptions(opts *ExecutionOptions) *pkgengine.ExecutionOptions {
	if opts == nil {
		return pkgengine.DefaultExecutionOptions()
	}

	pkgOpts := &pkgengine.ExecutionOptions{
		Timeout:          opts.Timeout,
		NodeTimeout:      opts.NodeTimeout,
		ContinueOnError:  opts.ContinueOnError,
		StrictMode:       opts.StrictMode,
		MaxConcurrency:   opts.MaxParallelism,
		MaxParallelism:   opts.MaxParallelism,
		MaxOutputSize:    opts.MaxOutputSize,
		MaxTotalMemory:   opts.MaxTotalMemory,
		EnableMemoryOpts: opts.EnableMemoryOpts,
		Variables:        opts.Variables,
	}

	if opts.RetryPolicy != nil {
		strategy := pkgengine.BackoffConstant
		switch opts.RetryPolicy.BackoffStrategy {
		case BackoffLinear:
			strategy = pkgengine.BackoffLinear
		case BackoffExponential:
			strategy = pkgengine.BackoffExponential
		}
		pkgOpts.RetryPolicy = &pkgengine.RetryPolicy{
			MaxAttempts:     opts.RetryPolicy.MaxAttempts,
			InitialDelay:    opts.RetryPolicy.InitialDelay,
			MaxDelay:        opts.RetryPolicy.MaxDelay,
			BackoffStrategy: strategy,
			RetryOn:         opts.RetryPolicy.RetryableErrors,
		}
	}

	return pkgOpts
}
