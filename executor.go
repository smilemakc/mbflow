package mbflow

import (
	"context"

	"github.com/smilemakc/mbflow/internal/application/executor"
	"github.com/smilemakc/mbflow/internal/domain"
	"github.com/smilemakc/mbflow/internal/infrastructure/monitoring"
	"github.com/smilemakc/mbflow/internal/infrastructure/storage"
)

// ========== Type Aliases for Executor Components ==========

type (
	// Core executor types
	NodeExecutor        = executor.NodeExecutor
	NodeExecutionInputs = executor.NodeExecutionInputs
	WorkflowEngine      = executor.WorkflowEngine
	EngineConfig        = executor.EngineConfig

	// Retry and resilience
	RetryPolicy          = executor.RetryPolicy
	CircuitBreaker       = executor.CircuitBreaker
	CircuitBreakerConfig = executor.CircuitBreakerConfig

	// Error handling
	ErrorStrategy = executor.ErrorStrategy
	NodeError     = executor.NodeError

	// Join and parallel execution
	JoinEvaluator    = executor.JoinEvaluator
	JoinBranchStatus = executor.JoinBranchStatus

	// Planning
	ExecutionPlan = executor.ExecutionPlan
	ExecutionWave = executor.ExecutionWave
	WorkflowGraph = executor.WorkflowGraph
)

// Executor is a public façade around the internal WorkflowEngine.
// It wires the event store, optional observers/metrics, and exposes a simplified API.
type Executor struct {
	engine    *executor.WorkflowEngine
	store     domain.EventStore
	observers *monitoring.ObserverManager
}

// ExecutorOption configures the public Executor.
type ExecutorOption func(*executorConfig)

type executorConfig struct {
	engineConfig executor.EngineConfig
	eventStore   domain.EventStore
	observers    []monitoring.ExecutionObserver
}

// WithEngineConfig overrides the default engine configuration.
func WithEngineConfig(cfg executor.EngineConfig) ExecutorOption {
	return func(c *executorConfig) {
		c.engineConfig = cfg
	}
}

// WithEventStore injects a custom event store (e.g., Postgres).
func WithEventStore(store domain.EventStore) ExecutorOption {
	return func(c *executorConfig) {
		c.eventStore = store
	}
}

// WithObserver registers an execution observer that will receive streamed events.
func WithObserver(observer monitoring.ExecutionObserver) ExecutorOption {
	return func(c *executorConfig) {
		c.observers = append(c.observers, observer)
	}
}

// NewExecutor creates a new public executor with sensible defaults:
// - In-memory event store
// - Default engine configuration
// - Optional observers wired through an observed event store (streaming)
func NewExecutor(opts ...ExecutorOption) *Executor {
	cfg := executorConfig{
		engineConfig: executor.DefaultEngineConfig(),
		eventStore:   storage.NewMemoryEventStore(),
	}

	for _, opt := range opts {
		opt(&cfg)
	}

	observerManager := monitoring.NewObserverManager()
	for _, obs := range cfg.observers {
		observerManager.AddObserver(obs)
	}

	return &Executor{
		engine:    executor.NewWorkflowEngine(cfg.eventStore, observerManager, cfg.engineConfig),
		store:     cfg.eventStore,
		observers: observerManager,
	}
}

// ExecuteWorkflow runs the workflow with the provided trigger and initial variables.
// The execution result (domain.Execution) can be inspected or rebuilt later from the event store.
func (e *Executor) ExecuteWorkflow(
	ctx context.Context,
	workflow domain.Workflow,
	trigger domain.Trigger,
	initialVariables map[string]any,
) (domain.Execution, error) {
	return e.engine.ExecuteWorkflow(ctx, workflow, trigger, initialVariables)
}

// EventStore returns the underlying event store (observed wrapper).
func (e *Executor) EventStore() domain.EventStore {
	return e.store
}

// AddObserver registers an observer that will receive streamed events.
func (e *Executor) AddObserver(observer monitoring.ExecutionObserver) {
	e.observers.AddObserver(observer)
}

// Observers exposes the observer manager for advanced scenarios.
func (e *Executor) Observers() *monitoring.ObserverManager {
	return e.observers
}

// Deprecated: prefer NewExecutor which wires observers and event store automatically.
func NewWorkflowEngine(config EngineConfig) *WorkflowEngine {
	return executor.NewWorkflowEngine(storage.NewMemoryEventStore(), nil, config)
}

// Ensure MetricsCollector implements ExecutorMetrics
var _ ExecutorMetrics = (*monitoring.MetricsCollector)(nil)

// ExecutionObserver is a callback for workflow execution events.
type ExecutionObserver = monitoring.ExecutionObserver

// ========== Public Helper Functions ==========

// RegisterDefaultExecutors registers all built-in node executors with the engine
func RegisterDefaultExecutors(engine *WorkflowEngine) {
	executor.RegisterDefaultExecutors(engine)
}

// DefaultEngineConfig returns the default engine configuration
func DefaultEngineConfig() EngineConfig {
	return executor.DefaultEngineConfig()
}

// DefaultRetryPolicy returns a sensible default retry policy
func DefaultRetryPolicy() *RetryPolicy {
	return executor.DefaultRetryPolicy()
}

// NoRetryPolicy returns a policy that disables retries
func NoRetryPolicy() *RetryPolicy {
	return executor.NoRetryPolicy()
}

// DefaultCircuitBreakerConfig returns default circuit breaker configuration
func DefaultCircuitBreakerConfig() CircuitBreakerConfig {
	return executor.DefaultCircuitBreakerConfig()
}

// NewCircuitBreaker creates a new circuit breaker
func NewCircuitBreaker(config CircuitBreakerConfig) *CircuitBreaker {
	return executor.NewCircuitBreaker(config)
}

// ========== Factory Functions ==========

// NewFailFastStrategy creates a fail-fast error strategy
func NewFailFastStrategy() ErrorStrategy {
	return executor.NewFailFastStrategy()
}

// NewContinueOnErrorStrategy creates a continue-on-error strategy
func NewContinueOnErrorStrategy() ErrorStrategy {
	return executor.NewContinueOnErrorStrategy()
}

// NewBestEffortStrategy creates a best-effort strategy
func NewBestEffortStrategy() ErrorStrategy {
	return executor.NewBestEffortStrategy()
}

// NewRequireNStrategy creates a require-N strategy
func NewRequireNStrategy(minRequired int) ErrorStrategy {
	return executor.NewRequireNStrategy(minRequired)
}

// NewJoinEvaluator creates a new join evaluator
func NewJoinEvaluator() *JoinEvaluator {
	return executor.NewJoinEvaluator()
}

// ========== Advanced Executor Creation ==========

// ExecutorBuilder provides a fluent interface for building executors with advanced configuration
type ExecutorBuilder struct {
	config        EngineConfig
	eventStore    domain.EventStore
	observers     []monitoring.ExecutionObserver
	nodeExecutors map[domain.NodeType]NodeExecutor
}

// NewExecutorBuilder creates a new executor builder with default configuration
func NewExecutorBuilder() *ExecutorBuilder {
	return &ExecutorBuilder{
		config:        DefaultEngineConfig(),
		eventStore:    storage.NewMemoryEventStore(),
		observers:     make([]monitoring.ExecutionObserver, 0),
		nodeExecutors: make(map[domain.NodeType]NodeExecutor),
	}
}

// WithConfig sets the engine configuration
func (b *ExecutorBuilder) WithConfig(config EngineConfig) *ExecutorBuilder {
	b.config = config
	return b
}

// WithEventStore sets the event store
func (b *ExecutorBuilder) WithEventStore(store domain.EventStore) *ExecutorBuilder {
	b.eventStore = store
	return b
}

// WithObserver adds an execution observer
func (b *ExecutorBuilder) WithObserver(observer monitoring.ExecutionObserver) *ExecutorBuilder {
	b.observers = append(b.observers, observer)
	return b
}

// WithNodeExecutor registers a custom node executor
func (b *ExecutorBuilder) WithNodeExecutor(nodeType domain.NodeType, exec NodeExecutor) *ExecutorBuilder {
	b.nodeExecutors[nodeType] = exec
	return b
}

// EnableParallelExecution enables parallel node execution
func (b *ExecutorBuilder) EnableParallelExecution(maxParallel int) *ExecutorBuilder {
	b.config.EnableParallel = true
	b.config.MaxParallelNodes = maxParallel
	return b
}

// EnableRetry enables retry with the given policy
func (b *ExecutorBuilder) EnableRetry(maxRetries int) *ExecutorBuilder {
	b.config.EnableRetry = true
	b.config.DefaultMaxRetries = maxRetries
	return b
}

// EnableCircuitBreaker enables circuit breaker protection
func (b *ExecutorBuilder) EnableCircuitBreaker() *ExecutorBuilder {
	b.config.EnableCircuitBreaker = true
	return b
}

// EnableMetrics enables metrics collection
func (b *ExecutorBuilder) EnableMetrics() *ExecutorBuilder {
	b.config.EnableMetrics = true
	return b
}

// EnableTracing enables distributed tracing
func (b *ExecutorBuilder) EnableTracing() *ExecutorBuilder {
	b.config.EnableTracing = true
	return b
}

// Build creates the executor
func (b *ExecutorBuilder) Build() *Executor {
	if b.eventStore == nil {
		b.eventStore = storage.NewMemoryEventStore()
	}
	// Create observer manager
	observerManager := monitoring.NewObserverManager()
	for _, obs := range b.observers {
		observerManager.AddObserver(obs)
	}

	// Create engine with observer manager
	engine := executor.NewWorkflowEngine(b.eventStore, observerManager, b.config)

	// Register default executors
	RegisterDefaultExecutors(engine)

	// Register custom executors
	for nodeType, exec := range b.nodeExecutors {
		engine.RegisterNodeExecutor(nodeType, exec)
	}

	return &Executor{
		engine:    engine,
		store:     b.eventStore,
		observers: observerManager,
	}
}
