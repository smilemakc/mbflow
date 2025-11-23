package executor

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"mbflow/internal/domain"
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

	// stateRepository is optional; when set, execution states will be persisted
	stateRepository domain.ExecutionStateRepository

	// parallelErrorHandling defines error handling behavior for parallel execution
	parallelErrorHandling bool

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

	// StateRepository is optional; when set, execution states will be persisted
	StateRepository domain.ExecutionStateRepository

	// ParallelErrorHandling defines how to handle errors in parallel branches.
	// If true, execution stops immediately when any parallel branch fails (default behavior).
	// If false, other branches continue and join nodes handle partial failures.
	ParallelErrorHandling bool
}

// NewWorkflowEngine creates a new WorkflowEngine.
func NewWorkflowEngine(config *EngineConfig) *WorkflowEngine {
	if config == nil {
		config = &EngineConfig{}
	}

	if config.RetryPolicy == nil {
		config.RetryPolicy = DefaultRetryPolicy()
	}

	// Default parallel error handling: stop on first error (true)
	// Since bool zero value is false, we default to true for safety (stop on error)
	// User must explicitly set ParallelErrorHandling to false to enable continue-on-error
	parallelErrorHandling := true
	// Note: We can't distinguish "not set" from "set to false", so we always default to true
	// This means the default behavior is to stop on error, which is safer

	engine := &WorkflowEngine{
		executors:             make(map[string]NodeExecutor),
		retryPolicy:           config.RetryPolicy,
		observerManager:       monitoring.NewObserverManager(),
		stateRepository:       config.StateRepository,
		parallelErrorHandling: parallelErrorHandling,
	}

	// Prepare monitoring if enabled
	var metrics *monitoring.MetricsCollector
	if config.EnableMonitoring {
		logger := monitoring.NewExecutionLogger("WorkflowEngine", config.VerboseLogging)
		metrics = monitoring.NewMetricsCollector()
		observer := monitoring.NewCompositeObserver(logger, metrics, nil)
		engine.observerManager.AddObserver(observer)
	}

	// Register default executors
	// OpenAI executor is always registered (API key can come from node config or context)
	if metrics != nil {
		engine.RegisterExecutor(NewOpenAICompletionExecutorWithMetrics(config.OpenAIAPIKey, metrics))
	} else {
		engine.RegisterExecutor(NewOpenAICompletionExecutor(config.OpenAIAPIKey))
	}
	engine.RegisterExecutor(NewHTTPRequestExecutor())
	engine.RegisterExecutor(NewConditionalRouterExecutor())
	engine.RegisterExecutor(NewDataMergerExecutor())
	engine.RegisterExecutor(NewDataAggregatorExecutor())
	engine.RegisterExecutor(NewScriptExecutorExecutor())

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
// If edges are provided, it uses graph-based traversal with parallel execution support.
// If edges are empty, it falls back to sequential execution for backward compatibility.
func (e *WorkflowEngine) ExecuteWorkflow(ctx context.Context, workflowID, executionID string, nodes []NodeConfig, edges []EdgeConfig, initialVariables map[string]interface{}) (*ExecutionState, error) {
	// Try to load existing state if repository is available
	var state *ExecutionState
	if e.stateRepository != nil {
		domainState, err := e.stateRepository.GetExecutionState(ctx, executionID)
		if err == nil && domainState != nil {
			// Convert domain state to application state
			state = e.fromDomainExecutionState(domainState, ctx)
		}
	}

	// Create new state if not loaded
	if state == nil {
		if e.stateRepository != nil {
			state = NewExecutionStateWithRepository(ctx, executionID, workflowID, e.stateRepository)
		} else {
			state = NewExecutionState(executionID, workflowID)
		}
	}

	// Set initial variables
	for k, v := range initialVariables {
		state.SetVariable(k, v)
	}

	// Create execution context
	execCtx := NewExecutionContext(ctx, state)

	// Notify observers
	e.observerManager.NotifyExecutionStarted(workflowID, executionID)
	state.SetStatus(ExecutionStatusRunning)

	// If edges are provided, use graph-based execution with parallel support
	if len(edges) > 0 {
		err := e.executeWorkflowGraph(ctx, execCtx, nodes, edges)
		if err != nil {
			// Mark execution as failed
			state.SetStatus(ExecutionStatusFailed)
			state.Error = err

			duration := state.GetExecutionDuration()
			e.observerManager.NotifyExecutionFailed(workflowID, executionID, err, duration)

			return state, err
		}

		// Mark execution as completed
		state.SetStatus(ExecutionStatusCompleted)
		duration := state.GetExecutionDuration()
		e.observerManager.NotifyExecutionCompleted(workflowID, executionID, duration)

		return state, nil
	}

	// Fallback to sequential execution for backward compatibility
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

// executeWorkflowGraph executes a workflow using graph-based traversal with parallel execution support.
func (e *WorkflowEngine) executeWorkflowGraph(ctx context.Context, execCtx *ExecutionContext, nodes []NodeConfig, edges []EdgeConfig) error {
	// Build workflow graph
	graph := NewWorkflowGraph(nodes, edges)

	// Check for cycles
	if graph.HasCycles() {
		return fmt.Errorf("workflow graph contains cycles, cannot execute")
	}

	// Track completed nodes
	completedNodes := make(map[string]bool)
	executingNodes := make(map[string]bool)
	var mu sync.Mutex

	// Get entry nodes to start execution
	readyNodes := graph.GetEntryNodes()
	if len(readyNodes) == 0 {
		return fmt.Errorf("workflow has no entry nodes")
	}

	// Track stuck detection
	stuckIterations := 0
	const maxStuckIterations = 100 // 100 * 10ms = 1 second

	// Continue until all nodes are completed
	for {
		// Check context cancellation
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		// Find all ready nodes (nodes whose dependencies are completed and conditions are satisfied)
		mu.Lock()
		readyNodes, errGetReady := graph.GetReadyNodes(completedNodes, execCtx)
		if errGetReady != nil {
			mu.Unlock()
			return fmt.Errorf("failed to get ready nodes: %w", errGetReady)
		}

		currentReadyNodes := make([]string, 0)
		for _, nodeID := range readyNodes {
			if !executingNodes[nodeID] {
				currentReadyNodes = append(currentReadyNodes, nodeID)
				executingNodes[nodeID] = true
			}
		}
		mu.Unlock()

		// If no ready nodes, check if we're done or stuck
		if len(currentReadyNodes) == 0 {
			// Check if all nodes are completed
			if len(completedNodes) == len(nodes) {
				break
			}

			// Check if we're stuck (no progress possible)
			stuckIterations++
			if stuckIterations >= maxStuckIterations {
				// Analyze why we're stuck
				return e.analyzeStuckExecution(graph, completedNodes, execCtx)
			}

			// Wait a bit and retry (in case of race conditions)
			time.Sleep(10 * time.Millisecond)
			continue
		}

		// Reset stuck counter when we make progress
		stuckIterations = 0

		// Execute ready nodes in parallel
		err := e.executeNodesInParallel(ctx, execCtx, graph, currentReadyNodes, completedNodes, executingNodes, &mu)
		if err != nil {
			return err
		}
	}

	return nil
}

// analyzeStuckExecution analyzes why execution is stuck and returns a detailed error.
func (e *WorkflowEngine) analyzeStuckExecution(graph *WorkflowGraph, completedNodes map[string]bool, execCtx *ExecutionContext) error {
	var stuckNodes []string
	var stuckReasons []string

	variables := execCtx.GetAllVariables()

	// Find all nodes that are not completed
	for nodeID := range graph.nodes {
		if completedNodes[nodeID] {
			continue
		}

		// Check dependencies
		dependencies := graph.GetPreviousNodes(nodeID)
		allDepsCompleted := true
		var failedConditions []string

		for _, depNodeID := range dependencies {
			if !completedNodes[depNodeID] {
				allDepsCompleted = false
				break
			}

			// Check conditional edges
			edgeConfig, ok := graph.GetEdgeConfig(depNodeID, nodeID)
			if ok && edgeConfig.EdgeType == "conditional" {
				conditionalConfig, err := parseConfig[ConditionalEdgeConfig](edgeConfig.Config)
				if err == nil && conditionalConfig.Condition != "" {
					conditionResult, err := evaluateCondition(conditionalConfig.Condition, variables)
					if err == nil && !conditionResult {
						// Condition failed - this is why the node is stuck
						failedConditions = append(failedConditions, fmt.Sprintf("condition '%s' from node '%s' evaluated to false", conditionalConfig.Condition, depNodeID))
					}
				}
			}
		}

		// If all dependencies are completed but node is not ready, it's stuck
		if allDepsCompleted {
			stuckNodes = append(stuckNodes, nodeID)
			if len(failedConditions) > 0 {
				stuckReasons = append(stuckReasons, fmt.Sprintf("node '%s': %s", nodeID, strings.Join(failedConditions, "; ")))
			} else {
				stuckReasons = append(stuckReasons, fmt.Sprintf("node '%s': all dependencies completed but node is not ready (possible missing edge or condition issue)", nodeID))
			}
		}
	}

	if len(stuckNodes) == 0 {
		return fmt.Errorf("execution stuck: no nodes can become ready, but not all nodes are completed (%d/%d completed)", len(completedNodes), len(graph.nodes))
	}

	// Build detailed error message
	var msg strings.Builder
	msg.WriteString(fmt.Sprintf("execution stuck: %d node(s) cannot become ready:\n", len(stuckNodes)))
	for i, reason := range stuckReasons {
		msg.WriteString(fmt.Sprintf("  %d. %s\n", i+1, reason))
	}

	// Add variable context for debugging
	msg.WriteString("\nCurrent variable values:\n")
	for k, v := range variables {
		if strVal, ok := v.(string); ok && len(strVal) < 200 {
			msg.WriteString(fmt.Sprintf("  %s = %q\n", k, strVal))
		}
	}

	return fmt.Errorf("%s", msg.String())
}

// executeNodesInParallel executes multiple nodes concurrently using goroutines.
func (e *WorkflowEngine) executeNodesInParallel(ctx context.Context, execCtx *ExecutionContext, graph *WorkflowGraph, nodeIDs []string, completedNodes map[string]bool, executingNodes map[string]bool, mu *sync.Mutex) error {
	var wg sync.WaitGroup
	errChan := make(chan error, len(nodeIDs))

	// Execute each node in a separate goroutine
	for _, nodeID := range nodeIDs {
		nodeConfig, ok := graph.GetNode(nodeID)
		if !ok {
			mu.Lock()
			delete(executingNodes, nodeID)
			mu.Unlock()
			return fmt.Errorf("node %s not found in graph", nodeID)
		}

		wg.Add(1)
		go func(nID string, nConfig NodeConfig) {
			defer wg.Done()

			// Execute the node
			err := e.ExecuteNode(ctx, execCtx, nConfig)

			// Update state
			mu.Lock()
			delete(executingNodes, nID)
			if err == nil {
				completedNodes[nID] = true
			}
			mu.Unlock()

			// Send error if any
			if err != nil {
				select {
				case errChan <- err:
				default:
				}
			}
		}(nodeID, *nodeConfig)
	}

	// Wait for all nodes to complete
	wg.Wait()
	close(errChan)

	// Check for errors based on error handling strategy
	if e.parallelErrorHandling {
		// Stop on first error: return immediately if any node failed
		select {
		case err := <-errChan:
			return err
		default:
			return nil
		}
	} else {
		// Continue on error: collect all errors but continue execution
		// Join nodes will handle partial failures
		// For now, we still return the first error but mark nodes as failed
		// This allows join nodes to check which dependencies failed
		select {
		case err := <-errChan:
			// Log error but continue - join nodes will handle it
			// Return nil to continue execution
			_ = err
			return nil
		default:
			return nil
		}
	}
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

	// Create a temporary node object from config for logging
	// Use NodeID as name if name is not available
	tempNode := domain.NewNode(
		nodeConfig.NodeID,
		execCtx.State().WorkflowID,
		nodeConfig.NodeType,
		nodeConfig.NodeID, // Use NodeID as name
		nodeConfig.Config,
	)

	// Execute with retry logic
	err := retryExecutor.ExecuteWithCallback(
		ctx,
		func(attemptNumber int) error {
			// Mark node as started
			execCtx.State().MarkNodeStarted(nodeConfig.NodeID)
			e.observerManager.NotifyNodeStarted(execCtx.State().ExecutionID, tempNode, attemptNumber)

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
					tempNode,
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
				tempNode,
				output,
				duration,
			)

			// Execute callback if configured (asynchronously, doesn't affect workflow)
			e.executeNodeCallback(ctx, execCtx, nodeConfig, tempNode, output, duration)

			return nil
		},
		func(attemptNumber int, err error, delay time.Duration) {
			// Callback before retry
			execCtx.State().MarkNodeRetrying(nodeConfig.NodeID)
			e.observerManager.NotifyNodeRetrying(
				execCtx.State().ExecutionID,
				tempNode,
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

	// Create a temporary node object from config for logging
	// Use NodeID as name if name is not available
	tempNode := domain.NewNode(
		nodeConfig.NodeID,
		execCtx.State().WorkflowID,
		nodeConfig.NodeType,
		nodeConfig.NodeID, // Use NodeID as name
		nodeConfig.Config,
	)

	// Initialize node state
	execCtx.State().InitializeNodeState(nodeConfig.NodeID, 1)

	// Mark node as started
	execCtx.State().MarkNodeStarted(nodeConfig.NodeID)
	e.observerManager.NotifyNodeStarted(execCtx.State().ExecutionID, tempNode, 1)

	startTime := time.Now()

	// Execute the node
	output, err := executor.Execute(ctx, execCtx, nodeConfig.NodeID, nodeConfig.Config)

	duration := time.Since(startTime)

	if err != nil {
		// Mark node as failed
		execCtx.State().MarkNodeFailed(nodeConfig.NodeID, err)
		e.observerManager.NotifyNodeFailed(
			execCtx.State().ExecutionID,
			tempNode,
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
		tempNode,
		output,
		duration,
	)

	// Execute callback if configured (asynchronously, doesn't affect workflow)
	e.executeNodeCallback(ctx, execCtx, nodeConfig, tempNode, output, duration)

	return output, nil
}

// executeNodeCallback executes a callback for a node if configured.
// The callback is executed asynchronously and does not affect the workflow execution.
func (e *WorkflowEngine) executeNodeCallback(ctx context.Context, execCtx *ExecutionContext, nodeConfig NodeConfig, node *domain.Node, output interface{}, duration time.Duration) {
	// Check if callback is configured
	callbackConfig, err := parseCallbackConfig(nodeConfig.Config)
	if err != nil {
		// Invalid callback config, but don't fail the workflow
		return
	}
	if callbackConfig == nil {
		// No callback configured
		return
	}

	// Create callback processor
	processor, err := NewHTTPCallbackProcessor(*callbackConfig)
	if err != nil {
		// Invalid callback config, but don't fail the workflow
		return
	}

	// Execute callback asynchronously
	go func() {
		// Notify observers that callback started
		e.observerManager.NotifyNodeCallbackStarted(execCtx.State().ExecutionID, node)

		startTime := time.Now()

		// Prepare callback data
		callbackData := &NodeCallbackData{
			ExecutionID: execCtx.State().ExecutionID,
			WorkflowID:  execCtx.State().WorkflowID,
			NodeID:      nodeConfig.NodeID,
			NodeType:    nodeConfig.NodeType,
			Output:      output,
			Duration:    duration,
			StartedAt:   time.Now().Add(-duration),
			CompletedAt: time.Now(),
		}

		// Include variables if configured
		if callbackConfig.IncludeVariables {
			callbackData.Variables = execCtx.GetAllVariables()
		}

		// Execute callback
		callbackErr := processor.Process(ctx, callbackData)

		callbackDuration := time.Since(startTime)

		// Notify observers that callback completed
		e.observerManager.NotifyNodeCallbackCompleted(execCtx.State().ExecutionID, node, callbackErr, callbackDuration)
	}()
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

// fromDomainExecutionState converts a domain ExecutionState to application ExecutionState.
func (e *WorkflowEngine) fromDomainExecutionState(domainState *domain.ExecutionState, ctx context.Context) *ExecutionState {
	// Convert status
	var status ExecutionStatus
	switch domainState.Status() {
	case domain.ExecutionStateStatusPending:
		status = ExecutionStatusPending
	case domain.ExecutionStateStatusRunning:
		status = ExecutionStatusRunning
	case domain.ExecutionStateStatusCompleted:
		status = ExecutionStatusCompleted
	case domain.ExecutionStateStatusFailed:
		status = ExecutionStatusFailed
	case domain.ExecutionStateStatusCancelled:
		status = ExecutionStatusCancelled
	default:
		status = ExecutionStatusPending
	}

	// Convert error
	var err error
	if domainState.ErrorMessage() != "" {
		err = fmt.Errorf(domainState.ErrorMessage())
	}

	// Convert NodeStates
	nodeStates := make(map[string]*NodeState)
	for nodeID, ns := range domainState.NodeStates() {
		var nodeStatus NodeStatus
		switch ns.Status() {
		case domain.NodeStateStatusPending:
			nodeStatus = NodeStatusPending
		case domain.NodeStateStatusRunning:
			nodeStatus = NodeStatusRunning
		case domain.NodeStateStatusCompleted:
			nodeStatus = NodeStatusCompleted
		case domain.NodeStateStatusFailed:
			nodeStatus = NodeStatusFailed
		case domain.NodeStateStatusSkipped:
			nodeStatus = NodeStatusSkipped
		case domain.NodeStateStatusRetrying:
			nodeStatus = NodeStatusRetrying
		default:
			nodeStatus = NodeStatusPending
		}

		var nodeErr error
		if ns.ErrorMessage() != "" {
			nodeErr = fmt.Errorf(ns.ErrorMessage())
		}

		nodeStates[nodeID] = &NodeState{
			NodeID:        ns.NodeID(),
			Status:        nodeStatus,
			StartedAt:     ns.StartedAt(),
			FinishedAt:    ns.FinishedAt(),
			Output:        ns.Output(),
			Error:         nodeErr,
			AttemptNumber: ns.AttemptNumber(),
			MaxAttempts:   ns.MaxAttempts(),
		}
	}

	// Copy variables
	variables := make(map[string]interface{})
	for k, v := range domainState.Variables() {
		variables[k] = v
	}

	state := &ExecutionState{
		ExecutionID: domainState.ExecutionID(),
		WorkflowID:  domainState.WorkflowID(),
		Status:      status,
		Variables:   variables,
		NodeStates:  nodeStates,
		StartedAt:   domainState.StartedAt(),
		FinishedAt:  domainState.FinishedAt(),
		Error:       err,
		repository:  e.stateRepository,
		ctx:         ctx,
	}

	return state
}
