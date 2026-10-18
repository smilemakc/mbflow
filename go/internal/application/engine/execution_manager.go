package engine

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"runtime/debug"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/smilemakc/mbflow/go/internal/application/observer"
	"github.com/smilemakc/mbflow/go/internal/domain/repository"
	storagemodels "github.com/smilemakc/mbflow/go/internal/infrastructure/storage/models"
	pkgengine "github.com/smilemakc/mbflow/go/pkg/engine"
	"github.com/smilemakc/mbflow/go/pkg/executor"
	"github.com/smilemakc/mbflow/go/pkg/models"
	"github.com/smilemakc/mbflow/go/pkg/netguard"
)

// ExecutionManager manages workflow execution lifecycle.
type ExecutionManager struct {
	executorManager   executor.Manager
	workflowRepo      repository.WorkflowRepository
	executionRepo     repository.ExecutionRepository
	eventRepo         repository.EventRepository
	resourceRepo      repository.ResourceRepository
	dagExecutor       *pkgengine.DAGExecutor
	observerManager   *observer.ObserverManager
	ephemeralRegistry *EphemeralStreamRegistry

	// notPersisted holds IDs of running ephemeral executions that have no
	// database record; their events must not reach storage observers.
	notPersisted sync.Map

	// Background executions are tracked so Shutdown can drain them.
	inflight   sync.WaitGroup
	rootOnce   sync.Once
	rootCtx    context.Context
	rootCancel context.CancelFunc
}

// runContext returns the root context for background executions. It is
// initialized lazily so zero-value managers (e.g. in tests) work too.
func (em *ExecutionManager) runContext() context.Context {
	em.rootOnce.Do(func() {
		em.rootCtx, em.rootCancel = context.WithCancel(context.Background())
	})
	return em.rootCtx
}

// NewExecutionManager creates a new execution manager.
func NewExecutionManager(
	executorManager executor.Manager,
	workflowRepo repository.WorkflowRepository,
	executionRepo repository.ExecutionRepository,
	eventRepo repository.EventRepository,
	resourceRepo repository.ResourceRepository,
	observerManager *observer.ObserverManager,
	ephemeralRegistry ...*EphemeralStreamRegistry,
) *ExecutionManager {
	nodeExecutor := pkgengine.NewNodeExecutor(executorManager)
	notifier := NewObserverNotifier(observerManager)
	condEvaluator := pkgengine.NewExprConditionEvaluator()
	workflowLoader := NewRepositoryWorkflowLoader(workflowRepo)
	dagExecutor := pkgengine.NewDAGExecutor(nodeExecutor, condEvaluator, notifier, workflowLoader)

	em := &ExecutionManager{
		executorManager: executorManager,
		workflowRepo:    workflowRepo,
		executionRepo:   executionRepo,
		eventRepo:       eventRepo,
		resourceRepo:    resourceRepo,
		dagExecutor:     dagExecutor,
		observerManager: observerManager,
	}

	if len(ephemeralRegistry) > 0 && ephemeralRegistry[0] != nil {
		em.ephemeralRegistry = ephemeralRegistry[0]
	}

	return em
}

// ObserverManager returns the observer manager used for execution events.
func (em *ExecutionManager) ObserverManager() *observer.ObserverManager {
	return em.observerManager
}

// HasEphemeralExecution reports whether an ephemeral execution is currently tracked in memory.
func (em *ExecutionManager) HasEphemeralExecution(executionID string) bool {
	if em.ephemeralRegistry == nil {
		return false
	}
	_, ok := em.ephemeralRegistry.Get(executionID)
	return ok
}

// IsEphemeralExecutionTerminal reports whether an in-memory ephemeral execution reached a terminal state.
func (em *ExecutionManager) IsEphemeralExecutionTerminal(executionID string) bool {
	if em.ephemeralRegistry == nil {
		return false
	}
	return em.ephemeralRegistry.IsTerminal(executionID)
}

// Execute executes a workflow synchronously (blocks until completion).
func (em *ExecutionManager) Execute(
	ctx context.Context,
	workflowID string,
	input map[string]any,
	opts *ExecutionOptions,
) (*models.Execution, error) {
	execution, workflow, workflowModel, err := em.prepareExecution(ctx, workflowID, input, opts, models.ExecutionStatusRunning)
	if err != nil {
		return nil, err
	}

	// Register per-execution webhook observers
	webhookNames := em.registerWebhookObservers(execution.ID, opts)
	defer em.unregisterWebhookObservers(webhookNames)

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
	input map[string]any,
	opts *ExecutionOptions,
) (*models.Execution, error) {
	execution, workflow, workflowModel, err := em.prepareExecution(ctx, workflowID, input, opts, models.ExecutionStatusPending)
	if err != nil {
		return nil, err
	}

	// The goroutine below mutates execution; callers get a snapshot so they
	// can serialize it without racing.
	snapshot := *execution

	em.inflight.Add(1)
	go func() {
		defer em.inflight.Done()

		// Register per-execution webhook observers
		webhookNames := em.registerWebhookObservers(execution.ID, opts)
		defer em.unregisterWebhookObservers(webhookNames)

		// runCtx is cancelled when shutdown times out; bgCtx stays valid so the
		// final state can still be persisted.
		runCtx := em.runContext()
		bgCtx := context.WithoutCancel(runCtx)
		defer em.recoverExecution(bgCtx, execution, true, nil)

		execution.Status = models.ExecutionStatusRunning
		executionModel := storagemodels.ExecutionDomainToModel(execution)
		if err := em.executionRepo.Update(bgCtx, executionModel); err != nil {
			em.notifyExecutionError(bgCtx, execution, fmt.Errorf("failed to update execution status: %w", err))
			return
		}

		em.notifyExecutionStarted(bgCtx, execution)

		execState, execErr := em.executeWorkflowDAG(runCtx, execution, workflow, opts)

		if err := em.finalizeExecution(bgCtx, execution, workflow, workflowModel, execState, execErr); err != nil {
			em.notifyExecutionError(bgCtx, execution, fmt.Errorf("failed to finalize execution: %w", err))
			return
		}
	}()

	return &snapshot, nil
}

// prepareExecution loads workflow and creates execution record.
func (em *ExecutionManager) prepareExecution(
	ctx context.Context,
	workflowID string,
	input map[string]any,
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
		ID:             uuid.New().String(),
		WorkflowID:     workflow.ID,
		WorkflowName:   workflow.Name,
		WorkflowSource: "stored",
		Status:         initialStatus,
		Input:          input,
		Variables:      pkgengine.MergeVariables(workflow.Variables, opts.Variables),
		StartedAt:      time.Now(),
	}

	executionModel := storagemodels.ExecutionDomainToModel(execution)
	if opts.TriggerID != "" {
		if triggerID, err := uuid.Parse(opts.TriggerID); err == nil {
			executionModel.TriggerID = &triggerID
		}
	}
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
			Metadata:    em.eventMetadata(execution.ID),
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
			Metadata:    em.eventMetadata(execution.ID),
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
			Metadata:    em.eventMetadata(execution.ID),
		}
		em.observerManager.Notify(ctx, event)
	}
}

// eventMetadata returns metadata for execution-level events; events of
// non-persisted executions are flagged so storage observers skip them.
func (em *ExecutionManager) eventMetadata(executionID string) map[string]any {
	if _, ok := em.notPersisted.Load(executionID); ok {
		return map[string]any{observer.MetadataKeyNotPersisted: true}
	}
	return nil
}

// getFinalOutput gets output from leaf nodes.
func (em *ExecutionManager) getFinalOutput(execState *pkgengine.ExecutionState) map[string]any {
	leafNodes := pkgengine.FindLeafNodes(execState.Workflow)

	if len(leafNodes) == 0 {
		return nil
	}

	if len(leafNodes) == 1 {
		if output, ok := execState.GetNodeOutput(leafNodes[0].ID); ok {
			return pkgengine.ToMapInterface(output)
		}
	}

	merged := make(map[string]any)
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
) (map[string]any, error) {
	resourceMap := make(map[string]any)

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

		resourceMap[wr.Alias] = map[string]any{
			"id":          resource.GetID(),
			"name":        resource.GetName(),
			"type":        string(resource.GetType()),
			"access_type": wr.AccessType,
		}
	}

	return resourceMap, nil
}

// registerWebhookObservers creates and registers per-execution webhook observers.
// Returns observer names for cleanup via unregisterWebhookObservers.
func (em *ExecutionManager) registerWebhookObservers(executionID string, opts *ExecutionOptions) []string {
	if em.observerManager == nil || opts == nil || len(opts.Webhooks) == 0 {
		return nil
	}

	names := make([]string, 0, len(opts.Webhooks))
	for i, wh := range opts.Webhooks {
		name := fmt.Sprintf("webhook_%s_%d", executionID, i)

		// Build compound filter: execution ID + event types + node IDs
		filters := []observer.EventFilter{
			observer.NewExecutionIDFilter(executionID),
		}

		if len(wh.Events) > 0 {
			eventTypes := make([]observer.EventType, len(wh.Events))
			for j, e := range wh.Events {
				eventTypes[j] = observer.EventType(e)
			}
			filters = append(filters, observer.NewEventTypeFilter(eventTypes...))
		}

		if len(wh.NodeIDs) > 0 {
			filters = append(filters, observer.NewNodeIDFilter(wh.NodeIDs...))
		}

		compoundFilter := observer.NewCompoundEventFilter(filters...)

		// SSRF pre-flight: reject callback URLs pointing at disallowed networks
		// (no-op under the permissive default policy).
		checkCtx, cancelCheck := context.WithTimeout(context.Background(), 5*time.Second)
		checkErr := netguard.Default().CheckURL(checkCtx, wh.URL)
		cancelCheck()
		if checkErr != nil {
			continue
		}

		obs := observer.NewHTTPCallbackObserver(
			wh.URL,
			observer.WithHTTPName(name),
			observer.WithHTTPHeaders(wh.Headers),
			observer.WithHTTPFilter(compoundFilter),
			// User-supplied URL: every connection (incl. redirects and DNS
			// re-resolution) goes through the SSRF-guarded transport.
			observer.WithHTTPClient(netguard.DefaultClient(10*time.Second)),
		)

		if err := em.observerManager.Register(obs); err != nil {
			// Log but don't fail the execution for a bad webhook
			continue
		}
		names = append(names, name)
	}

	return names
}

// unregisterWebhookObservers removes per-execution observers from the global ObserverManager.
func (em *ExecutionManager) unregisterWebhookObservers(names []string) {
	if em.observerManager == nil || len(names) == 0 {
		return
	}
	for _, name := range names {
		_ = em.observerManager.Unregister(name)
	}
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

// Shutdown waits for in-flight background executions to finish. If ctx expires
// first, running executions are cancelled (and recorded as cancelled) and
// Shutdown waits briefly for them to persist their final state.
func (em *ExecutionManager) Shutdown(ctx context.Context) error {
	em.runContext()

	done := make(chan struct{})
	go func() {
		em.inflight.Wait()
		close(done)
	}()

	select {
	case <-done:
		em.rootCancel()
		return nil
	case <-ctx.Done():
	}

	em.rootCancel()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
	}
	return fmt.Errorf("executions still running at shutdown deadline: %w", ctx.Err())
}

// recoverExecution converts a panic in a background execution into a failed
// execution instead of crashing the process. onPanic runs after the state is
// updated (may be nil). persist controls whether the failure is stored.
func (em *ExecutionManager) recoverExecution(ctx context.Context, execution *models.Execution, persist bool, onPanic func()) {
	r := recover()
	if r == nil {
		return
	}

	slog.Error("recovered panic in workflow execution",
		"execution_id", execution.ID, "panic", r, "stack", string(debug.Stack()))

	now := time.Now()
	execution.Status = models.ExecutionStatusFailed
	execution.Error = fmt.Sprintf("internal error: execution panicked: %v", r)
	execution.CompletedAt = &now
	if persist {
		if err := em.executionRepo.Update(ctx, storagemodels.ExecutionDomainToModel(execution)); err != nil {
			slog.Error("failed to persist panicked execution", "execution_id", execution.ID, "error", err)
		}
	}
	em.notifyExecutionError(ctx, execution, errors.New(execution.Error))
	if onPanic != nil {
		onPanic()
	}
}
