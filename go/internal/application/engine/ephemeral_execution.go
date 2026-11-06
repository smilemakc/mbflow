package engine

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/smilemakc/mbflow/go/internal/domain/repository"
	storagemodels "github.com/smilemakc/mbflow/go/internal/infrastructure/storage/models"
	pkgengine "github.com/smilemakc/mbflow/go/pkg/engine"
	"github.com/smilemakc/mbflow/go/pkg/models"
)

// ExecuteEphemeral executes an inline workflow without requiring a stored workflow definition.
func (em *ExecutionManager) ExecuteEphemeral(ctx context.Context, opts *EphemeralExecutionOptions) (*models.Execution, error) {
	if opts.Workflow == nil {
		return nil, fmt.Errorf("workflow is required for ephemeral execution")
	}

	if err := opts.Workflow.Validate(); err != nil {
		return nil, fmt.Errorf("invalid workflow: %w", err)
	}

	// Resources (and decrypted credentials) are loaded before anything is
	// recorded, so a credential the caller may not use is rejected up front.
	secrets := pkgengine.NewSecretRedactor()
	refs := ephemeralResourceRefs(opts.Workflow, opts.CredentialIDs)
	ownerID := ephemeralOwnerID(opts)
	if len(refs) > 0 && ownerID == "" {
		return nil, fmt.Errorf("%w: credentials and resources of an inline execution require an owner: "+
			"run it as an authenticated user, or set workflow.created_by to the owner's user ID", ErrResourceUnavailable)
	}
	resources, err := em.loadResources(ctx, refs, ownerID, opts.Workflow, secrets)
	if err != nil {
		return nil, err
	}
	rt := &ephemeralRuntime{resources: resources, secrets: secrets}

	execution := em.buildEphemeralExecution(opts)

	redactor := NewEventRedactor()
	ephNotifier := NewEphemeralNotifier(em.observerManager, redactor)
	ephNotifier.notPersisted = !opts.PersistExecution
	if !opts.PersistExecution {
		em.notPersisted.Store(execution.ID, struct{}{})
	}

	if em.ephemeralRegistry != nil {
		em.ephemeralRegistry.Register(execution.ID, ephNotifier)
	}

	webhookNames := em.registerEphemeralWebhookObservers(execution.ID, opts.Webhooks)

	dagExecutor := em.buildEphemeralDAGExecutor(ephNotifier)
	pkgOpts := convertEphemeralToPkgOptions(opts)

	if opts.Mode == "sync" {
		return em.executeEphemeralSync(ctx, execution, opts, rt, dagExecutor, pkgOpts, webhookNames)
	}
	return em.executeEphemeralAsync(ctx, execution, opts, rt, dagExecutor, pkgOpts, webhookNames)
}

// ephemeralRuntime carries what an inline execution loaded before it started.
type ephemeralRuntime struct {
	resources map[string]any
	secrets   *pkgengine.SecretRedactor
}

// newState returns the execution state of an inline execution with its
// resources and secrets attached.
func (rt *ephemeralRuntime) newState(execution *models.Execution, workflow *models.Workflow) *pkgengine.ExecutionState {
	execState := pkgengine.NewExecutionState(
		execution.ID,
		workflow.ID,
		workflow,
		execution.Input,
		execution.Variables,
	)
	if rt != nil {
		if rt.resources != nil {
			execState.Resources = rt.resources
		}
		execState.Secrets = rt.secrets
	}
	return execState
}

// ephemeralOwnerID returns the user an inline execution runs as: the caller
// recorded by the API, else the workflow's created_by (service API callers
// acting on behalf of a user).
func ephemeralOwnerID(opts *EphemeralExecutionOptions) string {
	if opts.CreatedBy != nil && *opts.CreatedBy != uuid.Nil {
		return opts.CreatedBy.String()
	}
	return opts.Workflow.CreatedBy
}

func (em *ExecutionManager) buildEphemeralExecution(opts *EphemeralExecutionOptions) *models.Execution {
	variables := pkgengine.MergeVariables(opts.Workflow.Variables, opts.Variables)

	input := opts.Input
	if input == nil {
		input = make(map[string]any)
	}

	return &models.Execution{
		ID:             uuid.New().String(),
		WorkflowSource: "inline",
		WorkflowName:   opts.Workflow.Name,
		Status:         models.ExecutionStatusPending,
		Input:          input,
		Variables:      variables,
		StrictMode:     opts.StrictMode,
		StartedAt:      time.Now(),
	}
}

func (em *ExecutionManager) buildEphemeralDAGExecutor(notifier pkgengine.ExecutionNotifier) *pkgengine.DAGExecutor {
	nodeExecutor := pkgengine.NewNodeExecutor(em.executorManager)
	condEvaluator := pkgengine.NewExprConditionEvaluator()
	workflowLoader := pkgengine.NewNilWorkflowLoader()
	return pkgengine.NewDAGExecutor(nodeExecutor, condEvaluator, notifier, workflowLoader)
}

func (em *ExecutionManager) executeEphemeralSync(
	ctx context.Context,
	execution *models.Execution,
	opts *EphemeralExecutionOptions,
	rt *ephemeralRuntime,
	dagExecutor *pkgengine.DAGExecutor,
	pkgOpts *pkgengine.ExecutionOptions,
	webhookNames []string,
) (*models.Execution, error) {
	defer em.unregisterWebhookObservers(webhookNames)
	defer em.markEphemeralTerminal(execution.ID)

	release, err := em.acquireSlot(ctx)
	if err != nil {
		return nil, fmt.Errorf("waiting for an execution slot: %w", err)
	}
	defer release()

	// execCtx is cancelled by CancelExecution (persisted executions only);
	// ctx stays valid so the final state can still be persisted.
	execCtx, cancelExec := context.WithCancelCause(ctx)
	defer cancelExec(nil)

	if opts.PersistExecution {
		if err := em.createEphemeralExecution(ctx, execution, opts.Workflow, opts.CreatedBy); err != nil {
			return nil, fmt.Errorf("failed to persist ephemeral execution: %w", err)
		}
		defer em.active.add(execution.ID, cancelExec)()
	}

	execution.Status = models.ExecutionStatusRunning

	if opts.PersistExecution {
		if err := em.updateEphemeralExecution(ctx, execution); err != nil {
			if errors.Is(err, repository.ErrExecutionAlreadyCancelled) {
				em.abandonQueuedExecution(ctx, execution, true, errCancelRequested)
				return execution, errCancelRequested
			}
			return nil, fmt.Errorf("failed to persist ephemeral execution status: %w", err)
		}
	}

	em.notifyExecutionStarted(ctx, execution)

	execState := rt.newState(execution, opts.Workflow)

	execErr := execState.Secrets.Error(dagExecutor.Execute(execCtx, execState, pkgOpts))

	em.finalizeEphemeralExecution(execCtx, execution, execState, opts.Workflow, execErr)

	if opts.PersistExecution {
		if err := em.persistFinalState(ctx, execution); err != nil {
			return nil, fmt.Errorf("failed to persist ephemeral execution: %w", err)
		}
	}

	em.notifyExecutionCompletion(ctx, execution, execErr)

	return execution, execErr
}

func (em *ExecutionManager) executeEphemeralAsync(
	ctx context.Context,
	execution *models.Execution,
	opts *EphemeralExecutionOptions,
	rt *ephemeralRuntime,
	dagExecutor *pkgengine.DAGExecutor,
	pkgOpts *pkgengine.ExecutionOptions,
	webhookNames []string,
) (*models.Execution, error) {
	// execCtx is cancelled when shutdown times out or by CancelExecution
	// (persisted executions only); bgCtx stays valid so the final state can
	// still be persisted.
	runCtx := em.runContext()
	execCtx, cancelExec := context.WithCancelCause(runCtx)

	untrack := func() {}
	if opts.PersistExecution {
		if err := em.createEphemeralExecution(ctx, execution, opts.Workflow, opts.CreatedBy); err != nil {
			cancelExec(nil)
			em.unregisterWebhookObservers(webhookNames)
			em.markEphemeralTerminal(execution.ID)
			return nil, fmt.Errorf("failed to persist ephemeral execution: %w", err)
		}
		untrack = em.active.add(execution.ID, cancelExec)
	}

	// The goroutine below mutates execution; callers get a snapshot so they
	// can serialize it without racing.
	snapshot := *execution

	em.inflight.Add(1)
	go func() {
		defer em.inflight.Done()
		defer cancelExec(nil)
		defer untrack()
		defer em.unregisterWebhookObservers(webhookNames)

		bgCtx := context.WithoutCancel(runCtx)
		defer em.recoverExecution(bgCtx, execution, opts.PersistExecution, func() {
			em.markEphemeralTerminal(execution.ID)
		})

		// The execution stays pending until a slot frees up.
		release, err := em.acquireSlot(execCtx)
		if err != nil {
			em.abandonQueuedExecution(bgCtx, execution, opts.PersistExecution, queuedCancelCause(execCtx, err))
			em.markEphemeralTerminal(execution.ID)
			return
		}
		defer release()

		execution.Status = models.ExecutionStatusRunning

		if opts.PersistExecution {
			if err := em.updateEphemeralExecution(bgCtx, execution); err != nil {
				if errors.Is(err, repository.ErrExecutionAlreadyCancelled) {
					em.abandonQueuedExecution(bgCtx, execution, true, errCancelRequested)
					em.markEphemeralTerminal(execution.ID)
					return
				}
				em.notifyExecutionError(bgCtx, execution, fmt.Errorf("failed to update execution status: %w", err))
				em.markEphemeralTerminal(execution.ID)
				return
			}
		}

		em.notifyExecutionStarted(bgCtx, execution)

		execState := rt.newState(execution, opts.Workflow)

		execErr := execState.Secrets.Error(dagExecutor.Execute(execCtx, execState, pkgOpts))

		em.finalizeEphemeralExecution(execCtx, execution, execState, opts.Workflow, execErr)

		if opts.PersistExecution {
			if err := em.persistFinalState(bgCtx, execution); err != nil {
				em.notifyExecutionError(bgCtx, execution, fmt.Errorf("failed to update execution: %w", err))
				em.markEphemeralTerminal(execution.ID)
				return
			}
		}

		em.notifyExecutionCompletion(bgCtx, execution, execErr)
		em.markEphemeralTerminal(execution.ID)
	}()

	return &snapshot, nil
}

func (em *ExecutionManager) finalizeEphemeralExecution(
	execCtx context.Context,
	execution *models.Execution,
	execState *pkgengine.ExecutionState,
	workflow *models.Workflow,
	execErr error,
) {
	now := time.Now()
	execution.CompletedAt = &now
	execution.Duration = execution.CalculateDuration()

	if execErr != nil {
		if errors.Is(execErr, context.Canceled) || errors.Is(execErr, context.DeadlineExceeded) {
			execution.Status = models.ExecutionStatusCancelled
		} else {
			execution.Status = models.ExecutionStatusFailed
		}
		execution.Error = execErr.Error()
		if cancelled, msg := cancellationReason(execCtx); cancelled {
			execution.Status = models.ExecutionStatusCancelled
			execution.Error = msg
		}
	} else {
		execution.Status = models.ExecutionStatusCompleted
		execution.Output = em.getFinalOutput(execState)
	}

	execution.NodeExecutions = buildEphemeralNodeExecutions(execState, workflow)
	redactExecution(execution, execState)
}

func buildEphemeralNodeExecutions(execState *pkgengine.ExecutionState, workflow *models.Workflow) []*models.NodeExecution {
	nodeExecs := make([]*models.NodeExecution, 0, len(workflow.Nodes))

	for _, node := range workflow.Nodes {
		nodeExec := &models.NodeExecution{
			ID:          uuid.New().String(),
			ExecutionID: execState.ExecutionID,
			NodeID:      node.ID,
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

func (em *ExecutionManager) createEphemeralExecution(ctx context.Context, execution *models.Execution, workflow *models.Workflow, createdBy *uuid.UUID) error {
	executionModel := storagemodels.ExecutionDomainToModel(execution)
	executionModel.CreatedBy = createdBy
	snapshot, err := serializeWorkflowSnapshot(workflow)
	if err != nil {
		return fmt.Errorf("serialize workflow snapshot: %w", err)
	}
	executionModel.WorkflowSnapshot = snapshot
	return em.executionRepo.Create(ctx, executionModel)
}

func (em *ExecutionManager) updateEphemeralExecution(ctx context.Context, execution *models.Execution) error {
	executionModel := storagemodels.ExecutionDomainToModel(execution)
	return em.executionRepo.Update(ctx, executionModel)
}

func serializeWorkflowSnapshot(workflow *models.Workflow) (storagemodels.JSONBMap, error) {
	snapshotBytes, err := json.Marshal(workflow)
	if err != nil {
		return nil, fmt.Errorf("marshal workflow snapshot: %w", err)
	}

	var snapshot storagemodels.JSONBMap
	if err := json.Unmarshal(snapshotBytes, &snapshot); err != nil {
		return nil, fmt.Errorf("unmarshal workflow snapshot: %w", err)
	}
	return snapshot, nil
}

func (em *ExecutionManager) registerEphemeralWebhookObservers(executionID string, webhooks []WebhookSubscription) []string {
	if em.observerManager == nil || len(webhooks) == 0 {
		return nil
	}

	ephOpts := &ExecutionOptions{Webhooks: webhooks}
	return em.registerWebhookObservers(executionID, ephOpts)
}

func (em *ExecutionManager) markEphemeralTerminal(executionID string) {
	em.notPersisted.Delete(executionID)
	if em.ephemeralRegistry != nil {
		em.ephemeralRegistry.MarkTerminal(executionID)
	}
}

func convertEphemeralToPkgOptions(opts *EphemeralExecutionOptions) *pkgengine.ExecutionOptions {
	pkgOpts := pkgengine.DefaultExecutionOptions()

	if opts.Timeout > 0 {
		pkgOpts.Timeout = opts.Timeout
	}
	if opts.NodeTimeout > 0 {
		pkgOpts.NodeTimeout = opts.NodeTimeout
	}

	pkgOpts.StrictMode = opts.StrictMode
	pkgOpts.ContinueOnError = opts.ContinueOnError

	return pkgOpts
}
