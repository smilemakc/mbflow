package executor

import (
	"context"
	"fmt"
	"math"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/smilemakc/mbflow/internal/domain"
	"github.com/smilemakc/mbflow/internal/infrastructure/monitoring"
)

// WorkflowEngine is the main execution engine that orchestrates workflow execution
// using a three-phase architecture: Plan → Execute → Finalize
type WorkflowEngine struct {
	// Dependencies
	eventStore        domain.EventStore
	observerManager   *monitoring.ObserverManager
	planner           *ExecutionPlanner
	evaluator         *ConditionEvaluator
	templateProcessor *TemplateProcessor
	variableBinder    *VariableBinder

	// Node executors registry
	nodeExecutors map[domain.NodeType]NodeExecutor

	// Configuration
	config EngineConfig
}

// EngineConfig holds configuration for the workflow engine
type EngineConfig struct {
	// Parallelism
	MaxParallelNodes int
	EnableParallel   bool

	// Error handling
	DefaultErrorStrategy domain.ErrorStrategy

	// Retry
	EnableRetry       bool
	DefaultMaxRetries int
	DefaultRetryDelay time.Duration

	// Circuit breaker
	EnableCircuitBreaker bool

	// Timeouts
	NodeExecutionTimeout     time.Duration
	WorkflowExecutionTimeout time.Duration

	// Monitoring
	EnableMetrics bool
	EnableTracing bool

	// Templating
	EnableTemplating    bool
	DefaultTemplateMode string
}

// DefaultEngineConfig returns default configuration
func DefaultEngineConfig() EngineConfig {
	return EngineConfig{
		MaxParallelNodes:         10,
		EnableParallel:           true,
		DefaultErrorStrategy:     domain.ErrorStrategyFailFast,
		EnableRetry:              true,
		DefaultMaxRetries:        3,
		DefaultRetryDelay:        time.Second,
		EnableCircuitBreaker:     false,
		NodeExecutionTimeout:     5 * time.Minute,
		WorkflowExecutionTimeout: 30 * time.Minute,
		EnableMetrics:            true,
		EnableTracing:            false,
		EnableTemplating:         true,
		DefaultTemplateMode:      TemplateModeLenient,
	}
}

// NewWorkflowEngine creates a new workflow execution engine
func NewWorkflowEngine(eventStore domain.EventStore, observerManager *monitoring.ObserverManager, config EngineConfig) *WorkflowEngine {
	evaluator := NewConditionEvaluator(true)
	engine := &WorkflowEngine{
		eventStore:        eventStore,
		observerManager:   observerManager,
		planner:           NewExecutionPlanner(),
		evaluator:         evaluator,
		templateProcessor: NewTemplateProcessor(evaluator),
		variableBinder:    NewVariableBinder(evaluator),
		nodeExecutors:     make(map[domain.NodeType]NodeExecutor),
		config:            config,
	}

	// Register default node executors
	engine.registerDefaultExecutors()

	return engine
}

// RegisterNodeExecutor registers a custom node executor
func (e *WorkflowEngine) RegisterNodeExecutor(nodeType domain.NodeType, executor NodeExecutor) {
	e.nodeExecutors[nodeType] = executor
}

// registerDefaultExecutors registers built-in node executors
func (e *WorkflowEngine) registerDefaultExecutors() {
	RegisterDefaultExecutors(e)
}

// ExecuteWorkflow executes a workflow with the given trigger and initial variables
// This is the main entry point for workflow execution
func (e *WorkflowEngine) ExecuteWorkflow(
	ctx context.Context,
	workflow domain.Workflow,
	trigger domain.Trigger,
	initialVariables map[string]any,
) (domain.Execution, error) {
	// Generate execution ID
	executionID := uuid.New()

	// Create execution aggregate
	execution, err := domain.NewExecution(executionID, workflow.ID())
	if err != nil {
		return nil, fmt.Errorf("failed to create execution: %w", err)
	}

	// Phase 1: Planning
	plan, err := e.planExecution(ctx, workflow, execution)
	if err != nil {
		return nil, fmt.Errorf("planning phase failed: %w", err)
	}

	// Phase 2: Execute
	err = e.executeWorkflow(ctx, workflow, execution, trigger, plan, initialVariables)
	if err != nil {
		// Execution phase failed - finalize with error
		_ = e.finalizeExecution(ctx, execution, err)
		return execution, err
	}

	// Phase 3: Finalize
	err = e.finalizeExecution(ctx, execution, nil)
	if err != nil {
		return execution, fmt.Errorf("finalization phase failed: %w", err)
	}

	return execution, nil
}

// planExecution - Phase 1: Planning
// Validates workflow, builds graph, creates execution plan
func (e *WorkflowEngine) planExecution(
	ctx context.Context,
	workflow domain.Workflow,
	execution domain.Execution,
) (*ExecutionPlan, error) {
	// Validate workflow
	if err := workflow.Validate(); err != nil {
		return nil, fmt.Errorf("workflow validation failed: %w", err)
	}

	// Create execution plan
	plan, err := e.planner.CreatePlan(workflow)
	if err != nil {
		return nil, fmt.Errorf("failed to create execution plan: %w", err)
	}

	// Validate plan
	if err := e.planner.ValidatePlan(plan); err != nil {
		return nil, fmt.Errorf("execution plan validation failed: %w", err)
	}

	return plan, nil
}

// executeWorkflow - Phase 2: Execute
// Executes nodes according to the plan
func (e *WorkflowEngine) executeWorkflow(
	ctx context.Context,
	workflow domain.Workflow,
	execution domain.Execution,
	trigger domain.Trigger,
	plan *ExecutionPlan,
	initialVariables map[string]any,
) error {
	// Check trigger condition
	if !trigger.IsActive() || !trigger.ShouldTrigger(initialVariables) {
		return domain.NewDomainError(
			domain.ErrCodeValidationFailed,
			"trigger condition not met",
			nil,
		)
	}
	// Start execution
	if err := execution.Start(trigger.ID(), initialVariables); err != nil {
		return fmt.Errorf("failed to start execution: %w", err)
	}

	// Notify observers
	if e.observerManager != nil {
		e.observerManager.NotifyExecutionStarted(workflow.ID().String(), execution.ID().String())
	}

	// Persist start event
	if err := e.persistEvents(ctx, execution); err != nil {
		return fmt.Errorf("failed to persist start event: %w", err)
	}

	// Execute workflow using wave-based execution
	if e.config.EnableParallel {
		return e.executeWaves(ctx, execution, plan)
	}

	// Fallback to sequential execution
	return e.executeSequential(ctx, execution, plan)
}

// executeWaves executes nodes in waves (parallel execution within each wave)
func (e *WorkflowEngine) executeWaves(
	ctx context.Context,
	execution domain.Execution,
	plan *ExecutionPlan,
) error {
	for waveNum, wave := range plan.Waves {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		// Execute wave in parallel
		if err := e.executeWave(ctx, execution, wave, plan.Graph); err != nil {
			return fmt.Errorf("wave %d failed: %w", waveNum, err)
		}

		// Persist events after each wave
		if err := e.persistEvents(ctx, execution); err != nil {
			return fmt.Errorf("failed to persist events after wave %d: %w", waveNum, err)
		}
	}

	return nil
}

// executeWave executes all nodes in a wave in parallel
func (e *WorkflowEngine) executeWave(
	ctx context.Context,
	execution domain.Execution,
	wave ExecutionWave,
	graph *WorkflowGraph,
) error {
	// Limit parallelism
	maxParallel := e.config.MaxParallelNodes
	if len(wave.Nodes) < maxParallel {
		maxParallel = len(wave.Nodes)
	}

	// Create semaphore for limiting concurrent executions
	semaphore := make(chan struct{}, maxParallel)

	var wg sync.WaitGroup
	errChan := make(chan error, len(wave.Nodes))

	for _, nodeExec := range wave.Nodes {
		wg.Add(1)

		go func(ne NodeExecution) {
			defer wg.Done()

			// Acquire semaphore
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			// Execute node
			if err := e.executeNode(ctx, execution, ne, graph); err != nil {
				errChan <- err
			}
		}(nodeExec)
	}

	// Wait for all nodes to complete
	wg.Wait()
	close(errChan)

	// Check for errors
	var errors []error
	for err := range errChan {
		errors = append(errors, err)
	}

	if len(errors) > 0 {
		// Handle based on error strategy
		return e.handleWaveErrors(errors)
	}

	return nil
}

// executeNode executes a single node
func (e *WorkflowEngine) executeNode(
	ctx context.Context,
	execution domain.Execution,
	nodeExec NodeExecution,
	graph *WorkflowGraph,
) error {
	node := nodeExec.Node
	nodeID := node.ID()
	nodeName := node.Name()
	workflowID := execution.WorkflowID().String()
	// Check if node should be skipped (conditional edges)
	shouldExecute, err := e.shouldExecuteNode(execution, nodeID, graph)
	if err != nil {
		return err
	}

	if !shouldExecute {
		// Skip node
		return execution.SkipNode(nodeID, node.Name(), "conditional edge evaluated to false")
	}

	// Get node executor
	executor, exists := e.nodeExecutors[node.Type()]
	if !exists {
		return domain.NewDomainError(
			domain.ErrCodeNotFound,
			fmt.Sprintf("no executor registered for node type %s", node.Type()),
			nil,
		)
	}

	// Bind inputs using VariableBinder
	nodeInputs, err := e.variableBinder.BindInputs(node, graph, execution)
	if err != nil {
		return fmt.Errorf("failed to bind inputs for node %s: %w", node.Name(), err)
	}

	// Start node execution (store bound inputs in event)
	inputVars := nodeInputs.Variables.All()
	if err := execution.StartNode(nodeID, node.Name(), node.Type(), inputVars); err != nil {
		return err
	}

	// Notify observers
	if e.observerManager != nil {
		e.observerManager.NotifyNodeStarted(workflowID, execution.ID().String(), node, 1)
	}

	// Preprocess node config with templating (using scoped variables)
	if e.config.EnableTemplating {
		templateConfig := extractTemplateConfig(node.Config(), e.config.DefaultTemplateMode)

		// Merge scoped + global for templating
		templateVars := nodeInputs.Variables.Clone()
		_ = templateVars.Merge(nodeInputs.GlobalContext)

		processedConfig, err := e.templateProcessor.ProcessMap(
			node.Config(),
			templateVars.All(),
			templateConfig,
		)
		if err != nil {
			return fmt.Errorf("template processing failed for node %s: %w", node.Name(), err)
		}
		node = cloneNodeWithConfig(node, processedConfig)
	}

	// Execute node with timeout
	execCtx, cancel := context.WithTimeout(ctx, e.config.NodeExecutionTimeout)
	defer cancel()

	startTime := time.Now()
	output, err := executor.Execute(execCtx, node, nodeInputs)
	duration := time.Since(startTime)

	if err != nil {
		// Node execution failed
		if err := execution.FailNode(nodeID, node.Name(), node.Type(), err.Error(), 0); err != nil {
			return err
		}

		// Notify observers
		if e.observerManager != nil {
			e.observerManager.NotifyNodeFailed(workflowID, execution.ID().String(), node, err, duration, false)
		}

		// Check if we should retry (check both global config and per-node config)
		if e.config.EnableRetry {
			retryConfig := GetRetryConfig(node)
			if retryConfig.Enabled {
				return e.retryNode(ctx, execution, nodeExec, executor, graph)
			}
		}

		return fmt.Errorf("node %s failed: %w", node.Name(), err)
	}

	// Filter output to schema if defined
	if schema := node.IOSchema(); schema != nil && schema.Outputs != nil {
		output = e.filterOutputToSchema(output, schema.Outputs)
	}

	// Node execution succeeded
	if err := execution.CompleteNode(nodeID, node.Name(), node.Type(), output, duration); err != nil {
		return err
	}

	// Notify observers
	if e.observerManager != nil {
		e.observerManager.NotifyNodeCompleted(workflowID, execution.ID().String(), node, output, duration)
	}

	// Store node output separately
	if err := execution.SetNodeOutput(nodeID, output); err != nil {
		return err
	}
	if err := execution.Variables().Set(nodeName, output); err != nil {
		return err
	}

	return nil
}

// filterOutputToSchema filters output to only include keys declared in the schema
func (e *WorkflowEngine) filterOutputToSchema(
	output map[string]any,
	schema *domain.VariableSchema,
) map[string]any {
	filtered := make(map[string]any)

	for key, value := range output {
		if _, exists := schema.GetDefinition(key); exists {
			// Key is in schema - include it
			filtered[key] = value
		}
	}

	return filtered
}

// shouldExecuteNode checks if a node should be executed based on conditional edges
func (e *WorkflowEngine) shouldExecuteNode(
	execution domain.Execution,
	nodeID uuid.UUID,
	graph *WorkflowGraph,
) (bool, error) {
	// Get incoming edges
	incomingEdges := graph.GetIncomingEdges(nodeID)
	if len(incomingEdges) == 0 {
		// Entry node - always execute
		return true, nil
	}

	// Check all incoming edges
	hasConditional := false
	anyConditionTrue := false

	for _, edge := range incomingEdges {
		if edge.Type() == domain.EdgeTypeConditional {
			hasConditional = true

			// Evaluate condition
			result, err := e.evaluator.EvaluateEdge(edge, execution.Variables())
			if err != nil {
				return false, err
			}

			if result {
				anyConditionTrue = true
			}
		} else {
			// Non-conditional edge - execute
			return true, nil
		}
	}

	// If has conditional edges, at least one must be true
	if hasConditional && !anyConditionTrue {
		return false, nil
	}

	return true, nil
}

// retryNode retries a failed node execution
func (e *WorkflowEngine) retryNode(
	ctx context.Context,
	execution domain.Execution,
	nodeExec NodeExecution,
	executor NodeExecutor,
	graph *WorkflowGraph,
) error {
	node := nodeExec.Node
	nodeID := node.ID()
	workflowID := execution.WorkflowID().String()
	// Get retry configuration from node config
	retryConfig := GetRetryConfig(node)
	if !retryConfig.Enabled {
		// Retry not enabled for this node
		return fmt.Errorf("node %s failed and retry is not enabled", node.Name())
	}

	// Create retry policy from config
	policy := CreateRetryPolicy(retryConfig)

	// Attempt retries
	var lastErr error
	for attempt := 1; attempt <= policy.MaxAttempts; attempt++ {
		// Calculate delay
		delay := e.calculateRetryDelay(policy, attempt)

		// Notify observers about retry
		if e.observerManager != nil && attempt > 1 {
			e.observerManager.NotifyNodeRetrying(workflowID, execution.ID().String(), node, attempt, delay)
		}

		// Wait before retry
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(delay):
			// Continue with retry
		}

		// Bind inputs for retry
		nodeInputs, bindErr := e.variableBinder.BindInputs(node, graph, execution)
		if bindErr != nil {
			lastErr = fmt.Errorf("failed to bind inputs: %w", bindErr)
			continue
		}

		// Retry node execution
		startTime := time.Now()
		output, err := executor.Execute(ctx, node, nodeInputs)
		duration := time.Since(startTime)

		if err == nil {
			// Retry succeeded
			if err := execution.CompleteNode(nodeID, node.Name(), node.Type(), output, duration); err != nil {
				return err
			}

			// Notify observers
			if e.observerManager != nil {
				e.observerManager.NotifyNodeCompleted(workflowID, execution.ID().String(), node, output, duration)
			}

			// Store output in variables if configured
			if outputKey, ok := node.Config()["output_key"].(string); ok && outputKey != "" {
				if err := execution.SetVariable(outputKey, output, domain.ScopeExecution, uuid.Nil); err != nil {
					return err
				}
			}

			return nil
		}

		lastErr = err

		// Update failure with retry count
		if err := execution.FailNode(nodeID, node.Name(), node.Type(), err.Error(), attempt); err != nil {
			return err
		}

		// Notify observers
		if e.observerManager != nil {
			willRetry := attempt < policy.MaxAttempts
			e.observerManager.NotifyNodeFailed(workflowID, execution.ID().String(), node, err, duration, willRetry)
		}
	}

	// All retries exhausted
	return fmt.Errorf("node %s failed after %d retry attempts: %w", node.Name(), policy.MaxAttempts, lastErr)
}

// calculateRetryDelay calculates the delay before the next retry using exponential backoff
func (e *WorkflowEngine) calculateRetryDelay(policy *RetryPolicy, attempt int) time.Duration {
	// Calculate exponential delay
	delay := float64(policy.InitialDelay) * math.Pow(policy.Multiplier, float64(attempt-1))

	// Apply max delay cap
	if delay > float64(policy.MaxDelay) {
		delay = float64(policy.MaxDelay)
	}

	// Add jitter if enabled
	if policy.Jitter {
		jitterAmount := delay * 0.1 // 10% jitter
		jitter := (2*float64(time.Now().UnixNano()%1000)/1000 - 1) * jitterAmount
		delay += jitter
	}

	return time.Duration(delay)
}

// executeSequential executes nodes sequentially (fallback when parallel is disabled)
func (e *WorkflowEngine) executeSequential(
	ctx context.Context,
	execution domain.Execution,
	plan *ExecutionPlan,
) error {
	// Get topological order
	order, err := plan.Graph.TopologicalSort()
	if err != nil {
		return err
	}

	// Execute nodes in order
	for _, nodeID := range order {
		node, err := plan.Graph.GetNode(nodeID)
		if err != nil {
			return err
		}

		nodeExec := NodeExecution{
			NodeID:       nodeID,
			Node:         node,
			Dependencies: plan.Graph.GetPredecessors(nodeID),
		}

		if err := e.executeNode(ctx, execution, nodeExec, plan.Graph); err != nil {
			return err
		}

		// Persist events after each node
		if err := e.persistEvents(ctx, execution); err != nil {
			return err
		}
	}

	return nil
}

// handleWaveErrors handles errors that occurred during wave execution
func (e *WorkflowEngine) handleWaveErrors(errors []error) error {
	if len(errors) == 0 {
		return nil
	}

	// Based on error strategy
	switch e.config.DefaultErrorStrategy {
	case domain.ErrorStrategyFailFast:
		// Return first error
		return errors[0]

	case domain.ErrorStrategyContinueOnError:
		// Collect all errors
		return fmt.Errorf("wave execution encountered %d errors: %v", len(errors), errors)

	case domain.ErrorStrategyBestEffort:
		// Log errors but continue
		// For now, just return nil
		return nil

	default:
		return errors[0]
	}
}

// finalizeExecution - Phase 3: Finalize
// Completes execution, runs compensations if needed
func (e *WorkflowEngine) finalizeExecution(
	ctx context.Context,
	execution domain.Execution,
	executionErr error,
) error {
	if executionErr != nil {
		// Execution failed - mark as failed
		if err := execution.Fail(executionErr.Error(), uuid.Nil); err != nil {
			return err
		}

		// Notify observers
		if e.observerManager != nil {
			duration := time.Since(execution.StartedAt())
			e.observerManager.NotifyExecutionFailed(execution.WorkflowID().String(), execution.ID().String(), executionErr, duration)
		}
	} else {
		// Execution succeeded - mark as completed
		finalVars := execution.Variables().All()
		if err := execution.Complete(finalVars); err != nil {
			return err
		}

		// Notify observers
		if e.observerManager != nil {
			duration := time.Since(execution.StartedAt())
			e.observerManager.NotifyExecutionCompleted(execution.WorkflowID().String(), execution.ID().String(), duration)
		}
	}

	// Persist final events
	if err := e.persistEvents(ctx, execution); err != nil {
		return err
	}

	return nil
}

// persistEvents persists uncommitted events from execution
func (e *WorkflowEngine) persistEvents(ctx context.Context, execution domain.Execution) error {
	events := execution.GetUncommittedEvents()
	if len(events) == 0 {
		return nil
	}

	// Persist events atomically
	if err := e.eventStore.AppendEvents(ctx, events); err != nil {
		return fmt.Errorf("failed to persist events: %w", err)
	}

	// Mark events as committed
	execution.MarkEventsAsCommitted()

	return nil
}

// GetExecution retrieves an execution by ID (rebuilds from events)
func (e *WorkflowEngine) GetExecution(ctx context.Context, executionID, workflowID uuid.UUID) (domain.Execution, error) {
	// Get events
	events, err := e.eventStore.GetEvents(ctx, executionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get events: %w", err)
	}

	if len(events) == 0 {
		return nil, domain.NewDomainError(
			domain.ErrCodeNotFound,
			fmt.Sprintf("execution %s not found", executionID),
			nil,
		)
	}

	// Rebuild execution from events
	execution, err := domain.RebuildFromEvents(executionID, workflowID, events)
	if err != nil {
		return nil, fmt.Errorf("failed to rebuild execution: %w", err)
	}

	return execution, nil
}

// extractTemplateConfig extracts template configuration from node config
func extractTemplateConfig(config map[string]any, defaultMode string) TemplateConfig {
	templateConfig := TemplateConfig{
		StrictMode: defaultMode == TemplateModeStrict,
		Fields:     nil, // Empty means all fields
	}

	// Check if node has template_config
	if tc, ok := config["template_config"].(map[string]any); ok {
		// Extract mode
		if mode, ok := tc["mode"].(string); ok {
			templateConfig.StrictMode = mode == TemplateModeStrict
		}

		// Extract fields
		if fields, ok := tc["fields"].([]interface{}); ok {
			strFields := make([]string, 0, len(fields))
			for _, f := range fields {
				if str, ok := f.(string); ok {
					strFields = append(strFields, str)
				}
			}
			templateConfig.Fields = strFields
		}
	}

	return templateConfig
}

// cloneNodeWithConfig creates a new node with processed config
// This preserves the node's identity but uses the templated config
func cloneNodeWithConfig(node domain.Node, processedConfig map[string]any) domain.Node {
	return &templateNode{
		original:        node,
		processedConfig: processedConfig,
	}
}

// templateNode wraps a node with processed config
type templateNode struct {
	original        domain.Node
	processedConfig map[string]any
}

func (tn *templateNode) ID() uuid.UUID {
	return tn.original.ID()
}

func (tn *templateNode) Type() domain.NodeType {
	return tn.original.Type()
}

func (tn *templateNode) Name() string {
	return tn.original.Name()
}

func (tn *templateNode) Config() map[string]any {
	return tn.processedConfig
}

func (tn *templateNode) IOSchema() *domain.NodeIOSchema {
	return tn.original.IOSchema()
}

func (tn *templateNode) InputBindingConfig() *domain.InputBindingConfig {
	return tn.original.InputBindingConfig()
}

// NodeExecutor defines the interface for node executors
type NodeExecutor interface {
	Execute(ctx context.Context, node domain.Node, inputs *NodeExecutionInputs) (map[string]any, error)
}

// NoOpExecutor is a no-operation executor for start/end nodes
type NoOpExecutor struct{}

func (e *NoOpExecutor) Execute(ctx context.Context, node domain.Node, variables *domain.VariableSet) (map[string]any, error) {
	return make(map[string]any), nil
}
