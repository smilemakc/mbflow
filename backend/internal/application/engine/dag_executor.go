package engine

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/expr-lang/expr"
	"github.com/smilemakc/mbflow/internal/application/observer"
	"github.com/smilemakc/mbflow/pkg/models"
)

// DAGExecutor executes workflow nodes in topological order with wave-based parallelism
type DAGExecutor struct {
	nodeExecutor    *NodeExecutor
	observerManager *observer.ObserverManager
	conditionCache  *ConditionCache // Cache for compiled edge conditions
}

// NewDAGExecutor creates a new DAG executor
func NewDAGExecutor(nodeExecutor *NodeExecutor, observerManager *observer.ObserverManager) *DAGExecutor {
	return &DAGExecutor{
		nodeExecutor:    nodeExecutor,
		observerManager: observerManager,
		conditionCache:  NewConditionCache(100), // Cache up to 100 compiled conditions
	}
}

// Execute executes the workflow DAG
func (de *DAGExecutor) Execute(
	ctx context.Context,
	execState *ExecutionState,
	opts *ExecutionOptions,
) error {
	// 1. Build DAG from workflow
	dag := buildDAG(execState.Workflow)

	// 2. Perform topological sort to get execution waves
	waves, err := topologicalSort(dag)
	if err != nil {
		return fmt.Errorf("DAG validation failed: %w", err)
	}

	// 3. Execute waves sequentially, nodes in parallel within wave
	for waveIdx, wave := range waves {
		if err := de.executeWave(ctx, execState, wave, waveIdx, opts); err != nil {
			return fmt.Errorf("wave %d execution failed: %w", waveIdx, err)
		}
	}

	return nil
}

// executeWave executes all nodes in a wave in parallel with priority support and error handling
func (de *DAGExecutor) executeWave(
	ctx context.Context,
	execState *ExecutionState,
	wave []*models.Node,
	waveIdx int,
	opts *ExecutionOptions,
) error {
	waveStartTime := time.Now()

	// Check for cancellation before wave
	select {
	case <-ctx.Done():
		return fmt.Errorf("execution cancelled before wave %d: %w", waveIdx, ctx.Err())
	default:
	}

	// Sort nodes by priority (higher priority first)
	sortedWave := sortNodesByPriority(wave)

	// Notify wave started
	nodeCount := len(sortedWave)
	de.safeNotify(ctx, observer.Event{
		Type:        observer.EventTypeWaveStarted,
		ExecutionID: execState.ExecutionID,
		WorkflowID:  execState.WorkflowID,
		Timestamp:   waveStartTime,
		Status:      "running",
		WaveIndex:   &waveIdx,
		NodeCount:   &nodeCount,
	})

	var wg sync.WaitGroup
	errChan := make(chan error, len(sortedWave))
	var errMu sync.Mutex
	var collectedErrors []error

	// Limit parallelism if configured
	semaphore := make(chan struct{}, opts.MaxParallelism)
	if opts.MaxParallelism <= 0 {
		// Unlimited parallelism
		semaphore = make(chan struct{}, len(sortedWave))
	}

	for _, node := range sortedWave {
		wg.Add(1)
		go func(n *models.Node) {
			defer wg.Done()

			// Check for cancellation
			select {
			case <-ctx.Done():
				execState.SetNodeStatus(n.ID, models.NodeExecutionStatusSkipped)
				de.safeNotify(ctx, observer.Event{
					Type:        observer.EventTypeNodeSkipped,
					ExecutionID: execState.ExecutionID,
					WorkflowID:  execState.WorkflowID,
					Timestamp:   time.Now(),
					Status:      "skipped",
					NodeID:      &n.ID,
					NodeName:    &n.Name,
					NodeType:    &n.Type,
					Message:     ptrString("execution cancelled"),
				})
				return
			default:
			}

			// Acquire semaphore
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			// Check if node should be executed based on incoming edge conditions
			shouldExec, skipReason := de.shouldExecuteNode(execState, n)
			if !shouldExec {
				// Skip this node - mark as skipped
				execState.SetNodeStatus(n.ID, models.NodeExecutionStatusSkipped)

				// Notify node skipped
				de.safeNotify(ctx, observer.Event{
					Type:        observer.EventTypeNodeSkipped,
					ExecutionID: execState.ExecutionID,
					WorkflowID:  execState.WorkflowID,
					Timestamp:   time.Now(),
					Status:      "skipped",
					NodeID:      &n.ID,
					NodeName:    &n.Name,
					NodeType:    &n.Type,
					Message:     &skipReason,
				})
				return
			}

			// Execute node
			if err := de.executeNode(ctx, execState, n, opts); err != nil {
				nodeErr := fmt.Errorf("node %s failed: %w", n.ID, err)
				errChan <- nodeErr

				if opts.ContinueOnError {
					// Collect error but continue
					errMu.Lock()
					collectedErrors = append(collectedErrors, nodeErr)
					errMu.Unlock()
				}
			}
		}(node)
	}

	wg.Wait()
	close(errChan)

	// Check for errors
	if !opts.ContinueOnError {
		// Fail-fast mode: return first error
		for err := range errChan {
			if err != nil {
				return err
			}
		}
	} else {
		// Continue-on-error mode: drain channel and aggregate errors
		for err := range errChan {
			if err != nil {
				errMu.Lock()
				if !containsError(collectedErrors, err) {
					collectedErrors = append(collectedErrors, err)
				}
				errMu.Unlock()
			}
		}
	}

	// Notify wave completed
	waveDuration := time.Since(waveStartTime).Milliseconds()
	status := "completed"
	if len(collectedErrors) > 0 {
		status = "completed_with_errors"
	}

	de.safeNotify(ctx, observer.Event{
		Type:        observer.EventTypeWaveCompleted,
		ExecutionID: execState.ExecutionID,
		WorkflowID:  execState.WorkflowID,
		Timestamp:   time.Now(),
		Status:      status,
		WaveIndex:   &waveIdx,
		DurationMs:  &waveDuration,
	})

	// Return aggregated errors if continue-on-error is enabled
	if opts.ContinueOnError && len(collectedErrors) > 0 {
		return &AggregatedError{
			Message: fmt.Sprintf("wave %d completed with %d error(s)", waveIdx, len(collectedErrors)),
			Errors:  collectedErrors,
		}
	}

	return nil
}

// AggregatedError contains multiple errors from continue-on-error mode
type AggregatedError struct {
	Message string
	Errors  []error
}

func (ae *AggregatedError) Error() string {
	if len(ae.Errors) == 0 {
		return ae.Message
	}
	return fmt.Sprintf("%s: %v", ae.Message, ae.Errors)
}

// sortNodesByPriority sorts nodes by priority (higher priority first)
func sortNodesByPriority(nodes []*models.Node) []*models.Node {
	sorted := make([]*models.Node, len(nodes))
	copy(sorted, nodes)

	// Simple insertion sort by priority
	for i := 1; i < len(sorted); i++ {
		key := sorted[i]
		keyPriority := getNodePriority(key)
		j := i - 1

		for j >= 0 && getNodePriority(sorted[j]) < keyPriority {
			sorted[j+1] = sorted[j]
			j--
		}
		sorted[j+1] = key
	}

	return sorted
}

// containsError checks if an error is already in the slice
func containsError(errors []error, target error) bool {
	for _, err := range errors {
		if err.Error() == target.Error() {
			return true
		}
	}
	return false
}

// executeNode executes a single node with timeout and retry support
func (de *DAGExecutor) executeNode(
	ctx context.Context,
	execState *ExecutionState,
	node *models.Node,
	opts *ExecutionOptions,
) error {
	nodeStartTime := time.Now()

	// Check for cancellation before starting
	select {
	case <-ctx.Done():
		return fmt.Errorf("execution cancelled before node start: %w", ctx.Err())
	default:
	}

	// Mark as running and record start time
	execState.SetNodeStatus(node.ID, models.NodeExecutionStatusRunning)
	execState.SetNodeStartTime(node.ID, nodeStartTime)

	// Notify node started (with error recovery)
	de.safeNotify(ctx, observer.Event{
		Type:        observer.EventTypeNodeStarted,
		ExecutionID: execState.ExecutionID,
		WorkflowID:  execState.WorkflowID,
		Timestamp:   nodeStartTime,
		Status:      "running",
		NodeID:      &node.ID,
		NodeName:    &node.Name,
		NodeType:    &node.Type,
	})

	// Create node-specific context with timeout
	nodeCtx := ctx
	nodeTimeoutMs := getNodeTimeout(node)
	if nodeTimeoutMs > 0 {
		var cancel context.CancelFunc
		nodeCtx, cancel = context.WithTimeout(ctx, time.Duration(nodeTimeoutMs)*time.Millisecond)
		defer cancel()
	} else if opts.NodeTimeout > 0 {
		var cancel context.CancelFunc
		nodeCtx, cancel = context.WithTimeout(ctx, opts.NodeTimeout)
		defer cancel()
	}

	// Get parent nodes
	parentNodes := getParentNodes(execState.Workflow, node)

	// Prepare node context
	nodeExecCtx := PrepareNodeContext(execState, node, parentNodes, opts)

	// Execute node with retry policy
	var execResult *NodeExecutionResult
	var execErr error

	retryPolicy := opts.RetryPolicy
	if retryPolicy == nil {
		retryPolicy = NoRetryPolicy()
	}

	// Setup retry callback to update observer
	retryPolicy.OnRetry = func(attempt int, err error) {
		de.safeNotify(ctx, observer.Event{
			Type:        observer.EventTypeNodeStarted, // Reuse started event for retry
			ExecutionID: execState.ExecutionID,
			WorkflowID:  execState.WorkflowID,
			Timestamp:   time.Now(),
			Status:      "retrying",
			NodeID:      &node.ID,
			NodeName:    &node.Name,
			NodeType:    &node.Type,
			Error:       err,
		})
	}

	execErr = retryPolicy.Execute(nodeCtx, func() error {
		result, err := de.nodeExecutor.Execute(nodeCtx, nodeExecCtx)
		if err == nil {
			execResult = result
		}
		return err
	})

	// Check if execution was successful
	if execErr != nil {
		nodeEndTime := time.Now()
		// Store error and mark as failed
		execState.SetNodeError(node.ID, execErr)
		execState.SetNodeStatus(node.ID, models.NodeExecutionStatusFailed)
		execState.SetNodeEndTime(node.ID, nodeEndTime)

		// Notify node failed
		nodeDuration := time.Since(nodeStartTime).Milliseconds()
		de.safeNotify(ctx, observer.Event{
			Type:        observer.EventTypeNodeFailed,
			ExecutionID: execState.ExecutionID,
			WorkflowID:  execState.WorkflowID,
			Timestamp:   time.Now(),
			Status:      "failed",
			NodeID:      &node.ID,
			NodeName:    &node.Name,
			NodeType:    &node.Type,
			Error:       execErr,
			DurationMs:  &nodeDuration,
		})

		return execErr
	}

	nodeEndTime := time.Now()

	// Check output size if limit is set
	if opts.MaxOutputSize > 0 {
		outputSize := estimateSize(execResult.Output)
		if outputSize > opts.MaxOutputSize {
			err := fmt.Errorf("node output size (%d bytes) exceeds limit (%d bytes)", outputSize, opts.MaxOutputSize)
			execState.SetNodeError(node.ID, err)
			execState.SetNodeStatus(node.ID, models.NodeExecutionStatusFailed)
			execState.SetNodeEndTime(node.ID, nodeEndTime)
			return err
		}
	}

	// Store execution result with metadata
	execState.SetNodeOutput(node.ID, execResult.Output)
	execState.SetNodeInput(node.ID, execResult.Input)
	execState.SetNodeConfig(node.ID, execResult.Config)
	execState.SetNodeResolvedConfig(node.ID, execResult.ResolvedConfig)
	execState.SetNodeStatus(node.ID, models.NodeExecutionStatusCompleted)
	execState.SetNodeEndTime(node.ID, nodeEndTime)

	// Check total memory usage
	if opts.MaxTotalMemory > 0 {
		totalMemory := execState.GetTotalMemoryUsage()
		if totalMemory > opts.MaxTotalMemory {
			// Log warning but don't fail (could implement cleanup here)
			de.safeNotify(ctx, observer.Event{
				Type:        observer.EventTypeNodeCompleted,
				ExecutionID: execState.ExecutionID,
				WorkflowID:  execState.WorkflowID,
				Timestamp:   time.Now(),
				Status:      "warning",
				NodeID:      &node.ID,
				Message:     ptrString(fmt.Sprintf("Total memory usage (%d) exceeds limit (%d)", totalMemory, opts.MaxTotalMemory)),
			})
		}
	}

	// Notify node completed
	nodeDuration := time.Since(nodeStartTime).Milliseconds()
	event := observer.Event{
		Type:        observer.EventTypeNodeCompleted,
		ExecutionID: execState.ExecutionID,
		WorkflowID:  execState.WorkflowID,
		Timestamp:   time.Now(),
		Status:      "completed",
		NodeID:      &node.ID,
		NodeName:    &node.Name,
		NodeType:    &node.Type,
		DurationMs:  &nodeDuration,
	}

	event.Output = toMapInterface(execResult.Output)

	de.safeNotify(ctx, event)

	return nil
}

// safeNotify wraps observer notifications with panic recovery
func (de *DAGExecutor) safeNotify(ctx context.Context, event observer.Event) {
	if de.observerManager == nil {
		return
	}

	defer func() {
		if r := recover(); r != nil {
			// Log the panic but don't crash execution
			fmt.Printf("Observer notification panicked: %v\n", r)
		}
	}()

	de.observerManager.Notify(ctx, event)
}

// ptrString returns a pointer to a string
func ptrString(s string) *string {
	return &s
}

// DAG represents workflow graph with indexed lookups
type DAG struct {
	Nodes    map[string]*models.Node
	Edges    map[string][]string // nodeID -> []childNodeIDs
	InDegree map[string]int      // nodeID -> number of parents
	Index    *DAGIndex           // Indexed lookups for O(1) access
}

// DAGIndex provides O(1) lookups for common operations
type DAGIndex struct {
	ParentsByNode map[string][]*models.Node // nodeID -> parent nodes
	EdgesByTarget map[string][]*models.Edge // nodeID -> incoming edges
	EdgesBySource map[string][]*models.Edge // nodeID -> outgoing edges
	NodesByID     map[string]*models.Node   // nodeID -> node (fast lookup)
}

// buildDAG builds DAG from workflow with indexed lookups
func buildDAG(workflow *models.Workflow) *DAG {
	dag := &DAG{
		Nodes:    make(map[string]*models.Node),
		Edges:    make(map[string][]string),
		InDegree: make(map[string]int),
		Index: &DAGIndex{
			ParentsByNode: make(map[string][]*models.Node),
			EdgesByTarget: make(map[string][]*models.Edge),
			EdgesBySource: make(map[string][]*models.Edge),
			NodesByID:     make(map[string]*models.Node),
		},
	}

	// Add nodes
	for _, node := range workflow.Nodes {
		dag.Nodes[node.ID] = node
		dag.InDegree[node.ID] = 0
		dag.Index.NodesByID[node.ID] = node
		dag.Index.ParentsByNode[node.ID] = []*models.Node{} // Initialize empty slice
	}

	// Add edges and build parent index
	for _, edge := range workflow.Edges {
		dag.Edges[edge.From] = append(dag.Edges[edge.From], edge.To)
		dag.InDegree[edge.To]++

		// Index edges by target and source
		dag.Index.EdgesByTarget[edge.To] = append(dag.Index.EdgesByTarget[edge.To], edge)
		dag.Index.EdgesBySource[edge.From] = append(dag.Index.EdgesBySource[edge.From], edge)

		// Build parent relationships
		if parentNode := dag.Index.NodesByID[edge.From]; parentNode != nil {
			dag.Index.ParentsByNode[edge.To] = append(dag.Index.ParentsByNode[edge.To], parentNode)
		}
	}

	return dag
}

// topologicalSort performs topological sort using Kahn's algorithm
// and returns execution waves (nodes that can be executed in parallel)
func topologicalSort(dag *DAG) ([][]*models.Node, error) {
	// Copy in-degree map to avoid modifying original
	inDegree := make(map[string]int)
	for k, v := range dag.InDegree {
		inDegree[k] = v
	}

	waves := [][]*models.Node{}
	processed := 0

	for processed < len(dag.Nodes) {
		wave := []*models.Node{}

		// Find all nodes with in-degree 0
		for nodeID, degree := range inDegree {
			if degree == 0 {
				if node, ok := dag.Nodes[nodeID]; ok {
					wave = append(wave, node)
				}
			}
		}

		if len(wave) == 0 {
			// No nodes with in-degree 0 but graph not fully processed
			// This means there's a cycle
			return nil, fmt.Errorf("cycle detected in workflow graph")
		}

		// Process wave
		for _, node := range wave {
			delete(inDegree, node.ID)
			processed++

			// Decrease in-degree of children
			for _, childID := range dag.Edges[node.ID] {
				inDegree[childID]--
			}
		}

		waves = append(waves, wave)
	}

	return waves, nil
}

// getParentNodes returns parent nodes for a given node using helpers
func getParentNodes(workflow *models.Workflow, node *models.Node) []*models.Node {
	parents := []*models.Node{}
	incomingEdges := collectIncomingEdges(workflow.Edges, node.ID)

	for _, edge := range incomingEdges {
		if parentNode := findNodeByID(workflow.Nodes, edge.From); parentNode != nil {
			parents = append(parents, parentNode)
		}
	}

	return parents
}

// shouldExecuteNode checks if a node should be executed based on incoming edge conditions.
// Returns (shouldExecute, skipReason).
// A node is executed if AT LEAST ONE incoming edge passes all checks (OR semantics):
// - Source node was executed (not skipped)
// - Edge condition evaluates to true (or no condition)
// - SourceHandle routing passes (for conditional nodes)
func (de *DAGExecutor) shouldExecuteNode(
	execState *ExecutionState,
	node *models.Node,
) (bool, string) {
	workflow := execState.Workflow

	// Find all incoming edges to this node using helper
	incomingEdges := collectIncomingEdges(workflow.Edges, node.ID)

	// If no incoming edges, execute the node (start node)
	if len(incomingEdges) == 0 {
		return true, ""
	}

	// Check if at least one incoming edge allows execution
	hasValidPath := false
	allSkipReasons := []string{}

	for _, edge := range incomingEdges {
		// Find source node using helper
		sourceNode := findNodeByID(workflow.Nodes, edge.From)

		if sourceNode == nil {
			continue
		}

		// Check if source node was skipped
		sourceStatus, _ := execState.GetNodeStatus(sourceNode.ID)
		if sourceStatus == models.NodeExecutionStatusSkipped {
			allSkipReasons = append(allSkipReasons, fmt.Sprintf("parent %s skipped", sourceNode.ID))
			continue
		}

		// Check if source node was executed
		if sourceStatus != models.NodeExecutionStatusCompleted {
			// Parent not yet executed or failed - this shouldn't happen in wave execution
			allSkipReasons = append(allSkipReasons, fmt.Sprintf("parent %s not completed (%s)", sourceNode.ID, sourceStatus))
			continue
		}

		// Evaluate edge condition if present
		if edge.Condition != "" {
			passed, err := de.evaluateEdgeCondition(edge, execState, sourceNode)
			if err != nil {
				allSkipReasons = append(allSkipReasons, fmt.Sprintf("edge from %s: condition error: %v", sourceNode.ID, err))
				continue
			}
			if !passed {
				allSkipReasons = append(allSkipReasons, fmt.Sprintf("edge from %s: condition '%s' is false", sourceNode.ID, edge.Condition))
				continue
			}
		}

		// Check for sourceHandle-based routing from conditional nodes
		if sourceNode.Type == NodeTypeConditional && edge.SourceHandle != "" {
			passed, err := evaluateSourceHandleCondition(edge, execState, sourceNode)
			if err != nil {
				allSkipReasons = append(allSkipReasons, fmt.Sprintf("edge from %s: sourceHandle error: %v", sourceNode.ID, err))
				continue
			}
			if !passed {
				allSkipReasons = append(allSkipReasons, fmt.Sprintf("edge from %s: conditional branch '%s' not active", sourceNode.ID, edge.SourceHandle))
				continue
			}
		}

		// This edge passes all checks - node should execute
		hasValidPath = true
		break
	}

	if hasValidPath {
		return true, ""
	}

	// No valid path found - skip with combined reason
	skipReason := "no valid incoming path"
	if len(allSkipReasons) > 0 {
		skipReason = fmt.Sprintf("no valid incoming path: %v", allSkipReasons)
	}
	return false, skipReason
}

// evaluateEdgeCondition evaluates the condition expression on an edge using cache.
// Returns true if the condition passes, false otherwise.
func (de *DAGExecutor) evaluateEdgeCondition(
	edge *models.Edge,
	execState *ExecutionState,
	sourceNode *models.Node,
) (bool, error) {
	condition := edge.Condition
	if condition == "" {
		return true, nil // No condition = always pass
	}

	// Get output from source node
	output, _ := execState.GetNodeOutput(sourceNode.ID)

	// Prepare environment for expression evaluation
	env := map[string]interface{}{
		"output": output,
		"node":   sourceNode.ID,
	}

	// Compile and cache the expression
	program, err := de.conditionCache.CompileAndCache(condition, env)
	if err != nil {
		return false, fmt.Errorf("failed to compile edge condition: %w", err)
	}

	// Execute the compiled program
	result, err := expr.Run(program, env)
	if err != nil {
		return false, fmt.Errorf("failed to evaluate edge condition: %w", err)
	}

	if boolResult, ok := result.(bool); ok {
		return boolResult, nil
	}

	return false, fmt.Errorf("edge condition must return boolean, got: %T", result)
}

// evaluateSourceHandleCondition checks if the edge's sourceHandle matches
// the output of a conditional node.
// For conditional nodes, output is typically a boolean (true/false).
func evaluateSourceHandleCondition(
	edge *models.Edge,
	execState *ExecutionState,
	sourceNode *models.Node,
) (bool, error) {
	// Get output from conditional node
	output, ok := execState.GetNodeOutput(sourceNode.ID)
	if !ok {
		return false, fmt.Errorf("conditional node %s has no output", sourceNode.ID)
	}

	// Conditional nodes return boolean
	if boolOutput, ok := output.(bool); ok {
		switch edge.SourceHandle {
		case SourceHandleTrue:
			return boolOutput, nil
		case SourceHandleFalse:
			return !boolOutput, nil
		default:
			// Unknown handle - let it pass
			return true, nil
		}
	}

	// If output is a map, check for "result" key
	if mapOutput, ok := output.(map[string]interface{}); ok {
		if result, exists := mapOutput["result"]; exists {
			if boolResult, ok := result.(bool); ok {
				switch edge.SourceHandle {
				case SourceHandleTrue:
					return boolResult, nil
				case SourceHandleFalse:
					return !boolResult, nil
				}
			}
		}
	}

	// Can't determine - default to pass
	return true, nil
}
