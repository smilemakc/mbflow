package engine

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
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

	execution := em.buildEphemeralExecution(opts)

	redactor := NewEventRedactor()
	ephNotifier := NewEphemeralNotifier(em.observerManager, redactor)

	if em.ephemeralRegistry != nil {
		em.ephemeralRegistry.Register(execution.ID, ephNotifier)
	}

	webhookNames := em.registerEphemeralWebhookObservers(execution.ID, opts.Webhooks)

	dagExecutor := em.buildEphemeralDAGExecutor(ephNotifier)
	pkgOpts := convertEphemeralToPkgOptions(opts)

	if opts.Mode == "sync" {
		return em.executeEphemeralSync(ctx, execution, opts, dagExecutor, pkgOpts, webhookNames)
	}
	return em.executeEphemeralAsync(ctx, execution, opts, dagExecutor, pkgOpts, webhookNames)
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
	dagExecutor *pkgengine.DAGExecutor,
	pkgOpts *pkgengine.ExecutionOptions,
	webhookNames []string,
) (*models.Execution, error) {
	defer em.unregisterWebhookObservers(webhookNames)
	defer em.markEphemeralTerminal(execution.ID)

	if opts.PersistExecution {
		if err := em.createEphemeralExecution(ctx, execution, opts.Workflow); err != nil {
			return nil, fmt.Errorf("failed to persist ephemeral execution: %w", err)
		}
	}

	execution.Status = models.ExecutionStatusRunning

	if opts.PersistExecution {
		if err := em.updateEphemeralExecution(ctx, execution); err != nil {
			return nil, fmt.Errorf("failed to persist ephemeral execution status: %w", err)
		}
	}

	em.notifyExecutionStarted(ctx, execution)

	execState := pkgengine.NewExecutionState(
		execution.ID,
		opts.Workflow.ID,
		opts.Workflow,
		execution.Input,
		execution.Variables,
	)

	execErr := dagExecutor.Execute(ctx, execState, pkgOpts)

	em.finalizeEphemeralExecution(execution, execState, opts.Workflow, execErr)

	if opts.PersistExecution {
		if err := em.updateEphemeralExecution(ctx, execution); err != nil {
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
	dagExecutor *pkgengine.DAGExecutor,
	pkgOpts *pkgengine.ExecutionOptions,
	webhookNames []string,
) (*models.Execution, error) {
	if opts.PersistExecution {
		if err := em.createEphemeralExecution(ctx, execution, opts.Workflow); err != nil {
			em.markEphemeralTerminal(execution.ID)
			return nil, fmt.Errorf("failed to persist ephemeral execution: %w", err)
		}
	}

	go func() {
		defer em.unregisterWebhookObservers(webhookNames)

		bgCtx := context.Background()

		execution.Status = models.ExecutionStatusRunning

		if opts.PersistExecution {
			if err := em.updateEphemeralExecution(bgCtx, execution); err != nil {
				em.notifyExecutionError(bgCtx, execution, fmt.Errorf("failed to update execution status: %w", err))
				em.markEphemeralTerminal(execution.ID)
				return
			}
		}

		em.notifyExecutionStarted(bgCtx, execution)

		execState := pkgengine.NewExecutionState(
			execution.ID,
			opts.Workflow.ID,
			opts.Workflow,
			execution.Input,
			execution.Variables,
		)

		execErr := dagExecutor.Execute(bgCtx, execState, pkgOpts)

		em.finalizeEphemeralExecution(execution, execState, opts.Workflow, execErr)

		if opts.PersistExecution {
			if err := em.updateEphemeralExecution(bgCtx, execution); err != nil {
				em.notifyExecutionError(bgCtx, execution, fmt.Errorf("failed to update execution: %w", err))
				em.markEphemeralTerminal(execution.ID)
				return
			}
		}

		em.notifyExecutionCompletion(bgCtx, execution, execErr)
		em.markEphemeralTerminal(execution.ID)
	}()

	return execution, nil
}

func (em *ExecutionManager) finalizeEphemeralExecution(
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
	} else {
		execution.Status = models.ExecutionStatusCompleted
		execution.Output = em.getFinalOutput(execState)
	}

	execution.NodeExecutions = buildEphemeralNodeExecutions(execState, workflow)
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

func (em *ExecutionManager) createEphemeralExecution(ctx context.Context, execution *models.Execution, workflow *models.Workflow) error {
	executionModel := storagemodels.ExecutionDomainToModel(execution)
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
