package executor

import (
	"context"
	"fmt"
	"sync"
	"time"

	"mbflow/internal/domain/errors"
	"mbflow/internal/infrastructure/monitoring"
)

// WorkflowEngine is the main execution engine for workflows.
// It orchestrates node execution, manages state, handles retries, and monitors execution.
type WorkflowEngine struct {
	// executors maps node types to their executors
	executors map[string]NodeExecutor

	// retryPolicy defines the retry behavior
	retryPolicy *RetryPolicy

	// observerManager manages execution observers
	observerManager *monitoring.ObserverManager

	// mu protects concurrent access
	mu sync.RWMutex
}

// EngineConfig configures the workflow engine.
type EngineConfig struct {
	// OpenAIAPIKey is the API key for OpenAI
	OpenAIAPIKey string

	// RetryPolicy defines the retry behavior
	RetryPolicy *RetryPolicy

	// EnableMonitoring enables monitoring and logging
	EnableMonitoring bool

	// VerboseLogging enables verbose logging
	VerboseLogging bool
}

// NewWorkflowEngine creates a new WorkflowEngine.
func NewWorkflowEngine(config *EngineConfig) *WorkflowEngine {
	if config == nil {
		config = &EngineConfig{}
	}

	if config.RetryPolicy == nil {
		config.RetryPolicy = DefaultRetryPolicy()
	}

	engine := &WorkflowEngine{
		executors:       make(map[string]NodeExecutor),
		retryPolicy:     config.RetryPolicy,
		observerManager: monitoring.NewObserverManager(),
	}

	// Register default executors
	if config.OpenAIAPIKey != "" {
		engine.RegisterExecutor(NewOpenAICompletionExecutor(config.OpenAIAPIKey))
	}
	engine.RegisterExecutor(NewHTTPRequestExecutor())
	engine.RegisterExecutor(NewConditionalRouterExecutor())
	engine.RegisterExecutor(NewDataMergerExecutor())
	engine.RegisterExecutor(NewDataAggregatorExecutor())
	engine.RegisterExecutor(NewScriptExecutorExecutor())

	// Setup monitoring if enabled
	if config.EnableMonitoring {
		logger := monitoring.NewExecutionLogger("WorkflowEngine", config.VerboseLogging)
		metrics := monitoring.NewMetricsCollector()

		observer := monitoring.NewCompositeObserver(logger, metrics, nil)
		engine.observerManager.AddObserver(observer)
	}

	return engine
}

// RegisterExecutor registers a node executor.
func (e *WorkflowEngine) RegisterExecutor(executor NodeExecutor) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.executors[executor.Type()] = executor
}

// AddObserver adds an execution observer.
func (e *WorkflowEngine) AddObserver(observer monitoring.ExecutionObserver) {
	e.observerManager.AddObserver(observer)
}

// ExecuteWorkflow executes a complete workflow.
// This is a simplified implementation that executes nodes in order.
// A full implementation would need to handle the workflow graph, parallel execution, etc.
func (e *WorkflowEngine) ExecuteWorkflow(ctx context.Context, workflowID, executionID string, nodes []NodeConfig, initialVariables map[string]interface{}) (*ExecutionState, error) {
	// Create execution state
	state := NewExecutionState(executionID, workflowID)

	// Set initial variables
	for k, v := range initialVariables {
		state.SetVariable(k, v)
	}

	// Create execution context
	execCtx := NewExecutionContext(ctx, state)

	// Notify observers
	e.observerManager.NotifyExecutionStarted(workflowID, executionID)
	state.SetStatus(ExecutionStatusRunning)

	// Execute nodes in sequence (simplified - real implementation would follow the graph)
	for _, nodeConfig := range nodes {
		// Check context cancellation
		select {
		case <-ctx.Done():
			state.SetStatus(ExecutionStatusCancelled)
			return state, ctx.Err()
		default:
		}

		// Execute node
		if err := e.ExecuteNode(ctx, execCtx, nodeConfig); err != nil {
			// Mark execution as failed
			state.SetStatus(ExecutionStatusFailed)
			state.Error = err

			duration := state.GetExecutionDuration()
			e.observerManager.NotifyExecutionFailed(workflowID, executionID, err, duration)

			return state, err
		}
	}

	// Mark execution as completed
	state.SetStatus(ExecutionStatusCompleted)
	duration := state.GetExecutionDuration()
	e.observerManager.NotifyExecutionCompleted(workflowID, executionID, duration)

	return state, nil
}

// NodeConfig represents the configuration for executing a node.
type NodeConfig struct {
	NodeID   string
	NodeType string
	Config   map[string]any
}

// ExecuteNode executes a single node with retry logic.
func (e *WorkflowEngine) ExecuteNode(ctx context.Context, execCtx *ExecutionContext, nodeConfig NodeConfig) error {
	// Get executor for node type
	e.mu.RLock()
	executor, ok := e.executors[nodeConfig.NodeType]
	e.mu.RUnlock()

	if !ok {
		return errors.NewNodeExecutionError(
			execCtx.State().WorkflowID,
			execCtx.State().ExecutionID,
			nodeConfig.NodeID,
			nodeConfig.NodeType,
			1,
			fmt.Sprintf("no executor registered for node type '%s'", nodeConfig.NodeType),
			nil,
			false,
		)
	}

	// Create retry executor
	retryExecutor := NewRetryExecutor(e.retryPolicy)

	// Execute with retry logic
	err := retryExecutor.ExecuteWithCallback(
		ctx,
		func(attemptNumber int) error {
			// Mark node as started
			execCtx.State().MarkNodeStarted(nodeConfig.NodeID)
			e.observerManager.NotifyNodeStarted(execCtx.State().ExecutionID, nodeConfig.NodeID, nodeConfig.NodeType, attemptNumber)

			startTime := time.Now()

			// Execute the node
			output, execErr := executor.Execute(ctx, execCtx, nodeConfig.NodeID, nodeConfig.Config)

			duration := time.Since(startTime)

			if execErr != nil {
				// Mark node as failed
				execCtx.State().MarkNodeFailed(nodeConfig.NodeID, execErr)

				// Check if we should retry
				willRetry := e.retryPolicy.ShouldRetry(execErr, attemptNumber)

				e.observerManager.NotifyNodeFailed(
					execCtx.State().ExecutionID,
					nodeConfig.NodeID,
					nodeConfig.NodeType,
					execErr,
					duration,
					willRetry,
				)

				return execErr
			}

			// Mark node as completed
			execCtx.State().MarkNodeCompleted(nodeConfig.NodeID, output)
			e.observerManager.NotifyNodeCompleted(
				execCtx.State().ExecutionID,
				nodeConfig.NodeID,
				nodeConfig.NodeType,
				output,
				duration,
			)

			return nil
		},
		func(attemptNumber int, err error, delay time.Duration) {
			// Callback before retry
			execCtx.State().MarkNodeRetrying(nodeConfig.NodeID)
			e.observerManager.NotifyNodeRetrying(
				execCtx.State().ExecutionID,
				nodeConfig.NodeID,
				attemptNumber+1,
				delay,
			)
		},
	)

	return err
}

// ExecuteNodeSimple executes a single node without retry logic.
// This is useful for testing or when you want to handle retries externally.
func (e *WorkflowEngine) ExecuteNodeSimple(ctx context.Context, execCtx *ExecutionContext, nodeConfig NodeConfig) (interface{}, error) {
	// Get executor for node type
	e.mu.RLock()
	executor, ok := e.executors[nodeConfig.NodeType]
	e.mu.RUnlock()

	if !ok {
		return nil, errors.NewNodeExecutionError(
			execCtx.State().WorkflowID,
			execCtx.State().ExecutionID,
			nodeConfig.NodeID,
			nodeConfig.NodeType,
			1,
			fmt.Sprintf("no executor registered for node type '%s'", nodeConfig.NodeType),
			nil,
			false,
		)
	}

	// Initialize node state
	execCtx.State().InitializeNodeState(nodeConfig.NodeID, 1)

	// Mark node as started
	execCtx.State().MarkNodeStarted(nodeConfig.NodeID)
	e.observerManager.NotifyNodeStarted(execCtx.State().ExecutionID, nodeConfig.NodeID, nodeConfig.NodeType, 1)

	startTime := time.Now()

	// Execute the node
	output, err := executor.Execute(ctx, execCtx, nodeConfig.NodeID, nodeConfig.Config)

	duration := time.Since(startTime)

	if err != nil {
		// Mark node as failed
		execCtx.State().MarkNodeFailed(nodeConfig.NodeID, err)
		e.observerManager.NotifyNodeFailed(
			execCtx.State().ExecutionID,
			nodeConfig.NodeID,
			nodeConfig.NodeType,
			err,
			duration,
			false,
		)
		return nil, err
	}

	// Mark node as completed
	execCtx.State().MarkNodeCompleted(nodeConfig.NodeID, output)
	e.observerManager.NotifyNodeCompleted(
		execCtx.State().ExecutionID,
		nodeConfig.NodeID,
		nodeConfig.NodeType,
		output,
		duration,
	)

	return output, nil
}

// GetExecutor returns the executor for a given node type.
func (e *WorkflowEngine) GetExecutor(nodeType string) (NodeExecutor, bool) {
	e.mu.RLock()
	defer e.mu.RUnlock()
	executor, ok := e.executors[nodeType]
	return executor, ok
}

// GetRegisteredExecutors returns all registered executor types.
func (e *WorkflowEngine) GetRegisteredExecutors() []string {
	e.mu.RLock()
	defer e.mu.RUnlock()

	types := make([]string, 0, len(e.executors))
	for t := range e.executors {
		types = append(types, t)
	}
	return types
}
