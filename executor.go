package mbflow

import (
	"context"
	"time"

	"mbflow/internal/application/executor"
	"mbflow/internal/infrastructure/monitoring"
)

// Executor represents a workflow executor.
// It provides methods for executing workflows and nodes.
type Executor interface {
	// ExecuteWorkflow executes a complete workflow
	ExecuteWorkflow(ctx context.Context, workflowID, executionID string, nodes []ExecutorNodeConfig, initialVariables map[string]interface{}) (ExecutorState, error)

	// ExecuteNode executes a single node
	ExecuteNode(ctx context.Context, state ExecutorState, nodeConfig ExecutorNodeConfig) error

	// AddObserver adds an execution observer
	AddObserver(observer ExecutionObserver)

	// GetMetrics returns execution metrics
	GetMetrics() ExecutorMetrics
}

// ExecutorNodeConfig represents the configuration for executing a node.
type ExecutorNodeConfig struct {
	NodeID   string
	NodeType string
	Config   map[string]any
}

// ExecutorState represents the state of a workflow execution.
type ExecutorState interface {
	// ExecutionID returns the execution ID
	ExecutionID() string

	// WorkflowID returns the workflow ID
	WorkflowID() string

	// Status returns the current status
	Status() string

	// GetVariable retrieves a variable
	GetVariable(key string) (interface{}, bool)

	// GetAllVariables returns all variables
	GetAllVariables() map[string]interface{}

	// GetExecutionDuration returns the execution duration
	GetExecutionDuration() string
}

// ExecutionObserver defines the interface for observing workflow execution events.
type ExecutionObserver interface {
	// OnExecutionStarted is called when a workflow execution starts
	OnExecutionStarted(workflowID, executionID string)

	// OnExecutionCompleted is called when a workflow execution completes successfully
	OnExecutionCompleted(workflowID, executionID string, duration string)

	// OnExecutionFailed is called when a workflow execution fails
	OnExecutionFailed(workflowID, executionID string, err error, duration string)

	// OnNodeStarted is called when a node starts executing
	OnNodeStarted(executionID, nodeID, nodeType string, attemptNumber int)

	// OnNodeCompleted is called when a node completes successfully
	OnNodeCompleted(executionID, nodeID, nodeType string, output interface{}, duration string)

	// OnNodeFailed is called when a node fails
	OnNodeFailed(executionID, nodeID, nodeType string, err error, duration string, willRetry bool)
}

// ExecutorMetrics provides execution metrics.
type ExecutorMetrics interface {
	// GetWorkflowMetrics returns metrics for a workflow
	GetWorkflowMetrics(workflowID string) map[string]interface{}

	// GetNodeMetrics returns metrics for a node type
	GetNodeMetrics(nodeType string) map[string]interface{}

	// GetAIMetrics returns AI API usage metrics
	GetAIMetrics() map[string]interface{}

	// GetSummary returns a summary of all metrics
	GetSummary() map[string]interface{}
}

// ExecutorConfig configures the workflow executor.
type ExecutorConfig struct {
	// OpenAIAPIKey is the API key for OpenAI
	OpenAIAPIKey string

	// MaxRetryAttempts is the maximum number of retry attempts
	MaxRetryAttempts int

	// EnableMonitoring enables monitoring and logging
	EnableMonitoring bool

	// VerboseLogging enables verbose logging
	VerboseLogging bool
}

// workflowExecutor is the internal implementation of Executor.
type workflowExecutor struct {
	engine  *executor.WorkflowEngine
	metrics *monitoring.MetricsCollector
}

// NewExecutor creates a new workflow executor.
func NewExecutor(config *ExecutorConfig) Executor {
	if config == nil {
		config = &ExecutorConfig{
			MaxRetryAttempts: 3,
			EnableMonitoring: true,
		}
	}

	// Create retry policy
	retryPolicy := executor.DefaultRetryPolicy()
	if config.MaxRetryAttempts > 0 {
		retryPolicy.MaxAttempts = config.MaxRetryAttempts
	}

	// Create engine config
	engineConfig := &executor.EngineConfig{
		OpenAIAPIKey:     config.OpenAIAPIKey,
		RetryPolicy:      retryPolicy,
		EnableMonitoring: config.EnableMonitoring,
		VerboseLogging:   config.VerboseLogging,
	}

	// Create engine
	engine := executor.NewWorkflowEngine(engineConfig)

	// Create metrics collector
	metrics := monitoring.NewMetricsCollector()

	// Add metrics observer
	if config.EnableMonitoring {
		observer := &metricsObserverAdapter{metrics: metrics}
		engine.AddObserver(observer)
	}

	return &workflowExecutor{
		engine:  engine,
		metrics: metrics,
	}
}

// ExecuteWorkflow implements Executor.
func (we *workflowExecutor) ExecuteWorkflow(ctx context.Context, workflowID, executionID string, nodes []ExecutorNodeConfig, initialVariables map[string]interface{}) (ExecutorState, error) {
	// Convert to internal node configs
	internalNodes := make([]executor.NodeConfig, len(nodes))
	for i, n := range nodes {
		internalNodes[i] = executor.NodeConfig{
			NodeID:   n.NodeID,
			NodeType: n.NodeType,
			Config:   n.Config,
		}
	}

	// Execute workflow
	state, err := we.engine.ExecuteWorkflow(ctx, workflowID, executionID, internalNodes, initialVariables)

	return &executorStateAdapter{state: state}, err
}

// ExecuteNode implements Executor.
func (we *workflowExecutor) ExecuteNode(ctx context.Context, state ExecutorState, nodeConfig ExecutorNodeConfig) error {
	// Convert state
	stateAdapter, ok := state.(*executorStateAdapter)
	if !ok {
		return nil
	}

	// Create execution context
	execCtx := executor.NewExecutionContext(ctx, stateAdapter.state)

	// Convert node config
	internalConfig := executor.NodeConfig{
		NodeID:   nodeConfig.NodeID,
		NodeType: nodeConfig.NodeType,
		Config:   nodeConfig.Config,
	}

	// Execute node
	return we.engine.ExecuteNode(ctx, execCtx, internalConfig)
}

// AddObserver implements Executor.
func (we *workflowExecutor) AddObserver(observer ExecutionObserver) {
	adapter := &observerAdapter{observer: observer}
	we.engine.AddObserver(adapter)
}

// GetMetrics implements Executor.
func (we *workflowExecutor) GetMetrics() ExecutorMetrics {
	return &metricsAdapter{metrics: we.metrics}
}

// executorStateAdapter adapts internal ExecutionState to public ExecutorState.
type executorStateAdapter struct {
	state *executor.ExecutionState
}

func (esa *executorStateAdapter) ExecutionID() string {
	return esa.state.ExecutionID
}

func (esa *executorStateAdapter) WorkflowID() string {
	return esa.state.WorkflowID
}

func (esa *executorStateAdapter) Status() string {
	return string(esa.state.GetStatus())
}

func (esa *executorStateAdapter) GetVariable(key string) (interface{}, bool) {
	return esa.state.GetVariable(key)
}

func (esa *executorStateAdapter) GetAllVariables() map[string]interface{} {
	return esa.state.GetAllVariables()
}

func (esa *executorStateAdapter) GetExecutionDuration() string {
	return esa.state.GetExecutionDuration().String()
}

// metricsAdapter adapts internal MetricsCollector to public ExecutorMetrics.
type metricsAdapter struct {
	metrics *monitoring.MetricsCollector
}

func (ma *metricsAdapter) GetWorkflowMetrics(workflowID string) map[string]interface{} {
	metrics := ma.metrics.GetWorkflowMetrics(workflowID)
	if metrics == nil {
		return nil
	}

	return map[string]interface{}{
		"workflow_id":      metrics.WorkflowID,
		"execution_count":  metrics.ExecutionCount,
		"success_count":    metrics.SuccessCount,
		"failure_count":    metrics.FailureCount,
		"average_duration": metrics.AverageDuration.String(),
		"min_duration":     metrics.MinDuration.String(),
		"max_duration":     metrics.MaxDuration.String(),
	}
}

func (ma *metricsAdapter) GetNodeMetrics(nodeType string) map[string]interface{} {
	metrics := ma.metrics.GetNodeMetrics(nodeType)
	if metrics == nil {
		return nil
	}

	return map[string]interface{}{
		"node_type":        metrics.NodeType,
		"execution_count":  metrics.ExecutionCount,
		"success_count":    metrics.SuccessCount,
		"failure_count":    metrics.FailureCount,
		"retry_count":      metrics.RetryCount,
		"average_duration": metrics.AverageDuration.String(),
	}
}

func (ma *metricsAdapter) GetAIMetrics() map[string]interface{} {
	metrics := ma.metrics.GetAIMetrics()
	if metrics == nil {
		return nil
	}

	return map[string]interface{}{
		"total_requests":     metrics.TotalRequests,
		"total_tokens":       metrics.TotalTokens,
		"prompt_tokens":      metrics.PromptTokens,
		"completion_tokens":  metrics.CompletionTokens,
		"estimated_cost_usd": metrics.EstimatedCostUSD,
		"average_latency_ms": metrics.AverageLatency.Milliseconds(),
	}
}

func (ma *metricsAdapter) GetSummary() map[string]interface{} {
	summary := ma.metrics.GetSummary()

	return map[string]interface{}{
		"total_workflows":       summary.TotalWorkflows,
		"total_executions":      summary.TotalExecutions,
		"total_successes":       summary.TotalSuccesses,
		"total_failures":        summary.TotalFailures,
		"overall_success_rate":  summary.OverallSuccessRate,
		"total_node_executions": summary.TotalNodeExecutions,
		"total_node_retries":    summary.TotalNodeRetries,
		"total_ai_requests":     summary.TotalAIRequests,
		"total_ai_tokens":       summary.TotalAITokens,
		"estimated_ai_cost_usd": summary.EstimatedAICostUSD,
	}
}

// observerAdapter adapts public ExecutionObserver to internal observer.
type observerAdapter struct {
	observer ExecutionObserver
}

func (oa *observerAdapter) OnExecutionStarted(workflowID, executionID string) {
	oa.observer.OnExecutionStarted(workflowID, executionID)
}

func (oa *observerAdapter) OnExecutionCompleted(workflowID, executionID string, duration time.Duration) {
	oa.observer.OnExecutionCompleted(workflowID, executionID, duration.String())
}

func (oa *observerAdapter) OnExecutionFailed(workflowID, executionID string, err error, duration time.Duration) {
	oa.observer.OnExecutionFailed(workflowID, executionID, err, duration.String())
}

func (oa *observerAdapter) OnNodeStarted(executionID, nodeID, nodeType string, attemptNumber int) {
	oa.observer.OnNodeStarted(executionID, nodeID, nodeType, attemptNumber)
}

func (oa *observerAdapter) OnNodeCompleted(executionID, nodeID, nodeType string, output interface{}, duration time.Duration) {
	oa.observer.OnNodeCompleted(executionID, nodeID, nodeType, output, duration.String())
}

func (oa *observerAdapter) OnNodeFailed(executionID, nodeID, nodeType string, err error, duration time.Duration, willRetry bool) {
	oa.observer.OnNodeFailed(executionID, nodeID, nodeType, err, duration.String(), willRetry)
}

func (oa *observerAdapter) OnNodeRetrying(executionID, nodeID string, attemptNumber int, delay time.Duration) {
	// Not exposed in public API
}

func (oa *observerAdapter) OnVariableSet(executionID, key string, value interface{}) {
	// Not exposed in public API
}

// metricsObserverAdapter adapts monitoring observer to collect metrics.
type metricsObserverAdapter struct {
	metrics *monitoring.MetricsCollector
}

func (moa *metricsObserverAdapter) OnExecutionStarted(workflowID, executionID string) {}

func (moa *metricsObserverAdapter) OnExecutionCompleted(workflowID, executionID string, duration time.Duration) {
}

func (moa *metricsObserverAdapter) OnExecutionFailed(workflowID, executionID string, err error, duration time.Duration) {
}

func (moa *metricsObserverAdapter) OnNodeStarted(executionID, nodeID, nodeType string, attemptNumber int) {
}

func (moa *metricsObserverAdapter) OnNodeCompleted(executionID, nodeID, nodeType string, output interface{}, duration time.Duration) {
}

func (moa *metricsObserverAdapter) OnNodeFailed(executionID, nodeID, nodeType string, err error, duration time.Duration, willRetry bool) {
}

func (moa *metricsObserverAdapter) OnNodeRetrying(executionID, nodeID string, attemptNumber int, delay time.Duration) {
}

func (moa *metricsObserverAdapter) OnVariableSet(executionID, key string, value interface{}) {}
