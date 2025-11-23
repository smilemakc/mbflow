package mbflow

import (
	"context"
	"time"

	"mbflow/internal/application/executor"
	"mbflow/internal/domain"
	"mbflow/internal/infrastructure/monitoring"
)

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

	// Register OpenAI executor (API key can come from node config or context)
	// Always register to allow API key from node config or execution context
	engine.RegisterExecutor(executor.NewOpenAICompletionExecutorWithMetrics(config.OpenAIAPIKey, metrics))
	engine.RegisterExecutor(executor.NewOpenAIResponsesExecutorWithMetrics(config.OpenAIAPIKey, metrics))

	return &workflowExecutor{
		engine:  engine,
		metrics: metrics,
	}
}

// ExecuteWorkflow implements Executor.
func (we *workflowExecutor) ExecuteWorkflow(ctx context.Context, workflowID, executionID string, nodes []ExecutorNodeConfig, edges []ExecutorEdgeConfig, initialVariables map[string]interface{}) (ExecutorState, error) {
	// Execute workflow
	state, err := we.engine.ExecuteWorkflow(ctx, workflowID, executionID, nodes, edges, initialVariables)

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

	// Execute node
	return we.engine.ExecuteNode(ctx, execCtx, nodeConfig)
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

func (ma *metricsAdapter) GetWorkflowMetrics(workflowID string) *WorkflowMetrics {
	return ma.metrics.GetWorkflowMetrics(workflowID)
}

func (ma *metricsAdapter) GetNodeMetrics(nodeType string) *monitoring.NodeMetrics {
	return ma.metrics.GetNodeMetricsByType(nodeType)
}

func (ma *metricsAdapter) GetNodeMetricsByID(nodeID string) *monitoring.NodeMetrics {
	return ma.metrics.GetNodeMetricsByID(nodeID)
}

func (ma *metricsAdapter) GetAIMetrics() *AIMetrics {
	return ma.metrics.GetAIMetrics()
}

func (ma *metricsAdapter) GetSummary() *MetricsSummary {
	return ma.metrics.GetSummary()
}

// observerAdapter adapts public ExecutionObserver to internal observer.
type observerAdapter struct {
	observer ExecutionObserver
}

func (oa *observerAdapter) OnExecutionStarted(workflowID, executionID string) {
	oa.observer.OnExecutionStarted(workflowID, executionID)
}

func (oa *observerAdapter) OnExecutionCompleted(workflowID, executionID string, duration time.Duration) {
	oa.observer.OnExecutionCompleted(workflowID, executionID, duration)
}

func (oa *observerAdapter) OnExecutionFailed(workflowID, executionID string, err error, duration time.Duration) {
	oa.observer.OnExecutionFailed(workflowID, executionID, err, duration)
}

func (oa *observerAdapter) OnNodeStarted(executionID string, node *domain.Node, attemptNumber int) {
	// Convert domain.Node to public Node interface
	var publicNode Node
	if node != nil {
		publicNode = &nodeAdapter{node: node}
	}
	oa.observer.OnNodeStarted(executionID, publicNode, attemptNumber)
}

func (oa *observerAdapter) OnNodeCompleted(executionID string, node *domain.Node, output interface{}, duration time.Duration) {
	var publicNode Node
	if node != nil {
		publicNode = &nodeAdapter{node: node}
	}
	oa.observer.OnNodeCompleted(executionID, publicNode, output, duration)
}

func (oa *observerAdapter) OnNodeFailed(executionID string, node *domain.Node, err error, duration time.Duration, willRetry bool) {
	var publicNode Node
	if node != nil {
		publicNode = &nodeAdapter{node: node}
	}
	oa.observer.OnNodeFailed(executionID, publicNode, err, duration, willRetry)
}

func (oa *observerAdapter) OnNodeRetrying(executionID string, node *domain.Node, attemptNumber int, delay time.Duration) {
	var publicNode Node
	if node != nil {
		publicNode = &nodeAdapter{node: node}
	}
	oa.observer.OnNodeRetrying(executionID, publicNode, attemptNumber, delay)
}

func (oa *observerAdapter) OnVariableSet(executionID, key string, value interface{}) {
	oa.observer.OnVariableSet(executionID, key, value)
}

func (oa *observerAdapter) OnNodeCallbackStarted(executionID string, node *domain.Node) {
	var publicNode Node
	if node != nil {
		publicNode = &nodeAdapter{node: node}
	}
	oa.observer.OnNodeCallbackStarted(executionID, publicNode)
}

func (oa *observerAdapter) OnNodeCallbackCompleted(executionID string, node *domain.Node, err error, duration time.Duration) {
	var publicNode Node
	if node != nil {
		publicNode = &nodeAdapter{node: node}
	}
	oa.observer.OnNodeCallbackCompleted(executionID, publicNode, err, duration)
}

// metricsObserverAdapter adapts monitoring observer to collect metrics.
type metricsObserverAdapter struct {
	metrics *monitoring.MetricsCollector
}

func (moa *metricsObserverAdapter) OnExecutionStarted(workflowID, executionID string) {}

func (moa *metricsObserverAdapter) OnExecutionCompleted(workflowID, executionID string, duration time.Duration) {
	moa.metrics.RecordWorkflowExecution(workflowID, duration, true)
}

func (moa *metricsObserverAdapter) OnExecutionFailed(workflowID, executionID string, err error, duration time.Duration) {
	moa.metrics.RecordWorkflowExecution(workflowID, duration, false)
}

func (moa *metricsObserverAdapter) OnNodeStarted(executionID string, node *domain.Node, attemptNumber int) {
}

func (moa *metricsObserverAdapter) OnNodeCompleted(executionID string, node *domain.Node, output interface{}, duration time.Duration) {
	if node != nil {
		moa.metrics.RecordNodeExecution(node.ID(), node.Type(), node.Name(), duration, true, false)
	}
}

func (moa *metricsObserverAdapter) OnNodeFailed(executionID string, node *domain.Node, err error, duration time.Duration, willRetry bool) {
	if node != nil {
		moa.metrics.RecordNodeExecution(node.ID(), node.Type(), node.Name(), duration, false, willRetry)
	}
}

func (moa *metricsObserverAdapter) OnNodeRetrying(executionID string, node *domain.Node, attemptNumber int, delay time.Duration) {
}

func (moa *metricsObserverAdapter) OnVariableSet(executionID, key string, value interface{}) {}

func (moa *metricsObserverAdapter) OnNodeCallbackStarted(executionID string, node *domain.Node) {
}

func (moa *metricsObserverAdapter) OnNodeCallbackCompleted(executionID string, node *domain.Node, err error, duration time.Duration) {
}

// nodeAdapter adapts domain.Node to public Node interface.
type nodeAdapter struct {
	node *domain.Node
}

func (na *nodeAdapter) ID() string {
	return na.node.ID()
}

func (na *nodeAdapter) WorkflowID() string {
	return na.node.WorkflowID()
}

func (na *nodeAdapter) Type() string {
	return na.node.Type()
}

func (na *nodeAdapter) Name() string {
	return na.node.Name()
}

func (na *nodeAdapter) Config() map[string]any {
	return na.node.Config()
}
