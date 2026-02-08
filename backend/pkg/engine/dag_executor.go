package engine

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/smilemakc/mbflow/pkg/models"
)

// DAGExecutor executes workflow nodes in topological order with wave-based parallelism.
// Uses ConditionEvaluator and ExecutionNotifier interfaces for pluggable behavior.
type DAGExecutor struct {
	nodeExecutor       *NodeExecutor
	conditionEvaluator ConditionEvaluator
	notifier           ExecutionNotifier
}

// NewDAGExecutor creates a new DAG executor.
func NewDAGExecutor(nodeExecutor *NodeExecutor, conditionEvaluator ConditionEvaluator, notifier ExecutionNotifier) *DAGExecutor {
	return &DAGExecutor{
		nodeExecutor:       nodeExecutor,
		conditionEvaluator: conditionEvaluator,
		notifier:           notifier,
	}
}

// Execute executes the workflow DAG.
func (de *DAGExecutor) Execute(
	ctx context.Context,
	execState *ExecutionState,
	opts *ExecutionOptions,
) error {
	dag := BuildDAG(execState.Workflow)

	waves, err := TopologicalSort(dag)
	if err != nil {
		return fmt.Errorf("DAG validation failed: %w", err)
	}

	waveIdx := 0
	for waveIdx < len(waves) {
		if err := ctx.Err(); err != nil {
			return fmt.Errorf("execution cancelled: %w", err)
		}

		if err := de.executeWave(ctx, execState, waves[waveIdx], waveIdx, opts); err != nil {
			return fmt.Errorf("wave %d execution failed: %w", waveIdx, err)
		}

		if jumpTarget := de.processLoopEdges(ctx, execState, dag, waves, waveIdx); jumpTarget >= 0 {
			waveIdx = jumpTarget
			continue
		}

		waveIdx++
	}

	return nil
}

// executeWave executes all nodes in a wave in parallel.
func (de *DAGExecutor) executeWave(
	ctx context.Context,
	execState *ExecutionState,
	wave []*models.Node,
	waveIdx int,
	opts *ExecutionOptions,
) error {
	waveStartTime := time.Now()

	select {
	case <-ctx.Done():
		return fmt.Errorf("execution cancelled before wave %d: %w", waveIdx, ctx.Err())
	default:
	}

	sortedWave := SortNodesByPriority(wave)

	nodeCount := len(sortedWave)
	de.safeNotify(ctx, ExecutionEvent{
		Type:        EventTypeWaveStarted,
		ExecutionID: execState.ExecutionID,
		WorkflowID:  execState.WorkflowID,
		Timestamp:   waveStartTime,
		Status:      "running",
		WaveIndex:   waveIdx,
		NodeCount:   nodeCount,
	})

	var wg sync.WaitGroup
	errChan := make(chan error, len(sortedWave))
	var errMu sync.Mutex
	var collectedErrors []error

	maxParallelism := opts.MaxParallelism
	if maxParallelism <= 0 {
		maxParallelism = opts.MaxConcurrency
	}
	if maxParallelism <= 0 {
		maxParallelism = len(sortedWave)
	}
	semaphore := make(chan struct{}, maxParallelism)

	for _, node := range sortedWave {
		wg.Add(1)
		go func(n *models.Node) {
			defer wg.Done()

			select {
			case <-ctx.Done():
				execState.SetNodeStatus(n.ID, models.NodeExecutionStatusSkipped)
				de.safeNotify(ctx, ExecutionEvent{
					Type:        EventTypeNodeSkipped,
					ExecutionID: execState.ExecutionID,
					WorkflowID:  execState.WorkflowID,
					Timestamp:   time.Now(),
					Status:      "skipped",
					NodeID:      n.ID,
					NodeName:    n.Name,
					NodeType:    n.Type,
					Message:     "execution cancelled",
				})
				return
			default:
			}

			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			shouldExec, skipReason := de.shouldExecuteNode(execState, n)
			if !shouldExec {
				execState.SetNodeStatus(n.ID, models.NodeExecutionStatusSkipped)
				de.safeNotify(ctx, ExecutionEvent{
					Type:        EventTypeNodeSkipped,
					ExecutionID: execState.ExecutionID,
					WorkflowID:  execState.WorkflowID,
					Timestamp:   time.Now(),
					Status:      "skipped",
					NodeID:      n.ID,
					NodeName:    n.Name,
					NodeType:    n.Type,
					Message:     skipReason,
				})
				return
			}

			if err := de.executeNode(ctx, execState, n, opts); err != nil {
				nodeErr := fmt.Errorf("node %s failed: %w", n.ID, err)
				errChan <- nodeErr

				if opts.ContinueOnError {
					errMu.Lock()
					collectedErrors = append(collectedErrors, nodeErr)
					errMu.Unlock()
				}
			}
		}(node)
	}

	wg.Wait()
	close(errChan)

	if !opts.ContinueOnError {
		for err := range errChan {
			if err != nil {
				return err
			}
		}
	} else {
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

	waveDuration := time.Since(waveStartTime).Milliseconds()
	status := "completed"
	if len(collectedErrors) > 0 {
		status = "completed_with_errors"
	}

	de.safeNotify(ctx, ExecutionEvent{
		Type:        EventTypeWaveCompleted,
		ExecutionID: execState.ExecutionID,
		WorkflowID:  execState.WorkflowID,
		Timestamp:   time.Now(),
		Status:      status,
		WaveIndex:   waveIdx,
		DurationMs:  waveDuration,
	})

	if opts.ContinueOnError && len(collectedErrors) > 0 {
		return fmt.Errorf("wave %d completed with %d error(s): %w", waveIdx, len(collectedErrors), errors.Join(collectedErrors...))
	}

	return nil
}

// containsError checks if an error is already in the slice.
func containsError(errs []error, target error) bool {
	for _, err := range errs {
		if err.Error() == target.Error() {
			return true
		}
	}
	return false
}

// executeNode executes a single node with timeout and retry support.
func (de *DAGExecutor) executeNode(
	ctx context.Context,
	execState *ExecutionState,
	node *models.Node,
	opts *ExecutionOptions,
) error {
	nodeStartTime := time.Now()

	select {
	case <-ctx.Done():
		return fmt.Errorf("execution cancelled before node start: %w", ctx.Err())
	default:
	}

	execState.SetNodeStatus(node.ID, models.NodeExecutionStatusRunning)
	execState.SetNodeStartTime(node.ID, nodeStartTime)

	de.safeNotify(ctx, ExecutionEvent{
		Type:        EventTypeNodeStarted,
		ExecutionID: execState.ExecutionID,
		WorkflowID:  execState.WorkflowID,
		Timestamp:   nodeStartTime,
		Status:      "running",
		NodeID:      node.ID,
		NodeName:    node.Name,
		NodeType:    node.Type,
	})

	// Create node-specific context with timeout
	nodeCtx := ctx
	nodeTimeoutMs := GetNodeTimeout(node)
	if nodeTimeoutMs > 0 {
		var cancel context.CancelFunc
		nodeCtx, cancel = context.WithTimeout(ctx, time.Duration(nodeTimeoutMs)*time.Millisecond)
		defer cancel()
	} else if opts.NodeTimeout > 0 {
		var cancel context.CancelFunc
		nodeCtx, cancel = context.WithTimeout(ctx, opts.NodeTimeout)
		defer cancel()
	}

	parentNodes := GetRegularParentNodes(execState.Workflow, node)
	nodeExecCtx := PrepareNodeContext(execState, node, parentNodes, opts)

	// Execute node with retry policy
	var execResult *NodeExecutionResult
	var execErr error

	retryPolicy := convertRetryPolicy(opts.RetryPolicy)

	retryPolicy.OnRetry = func(attempt int, err error) {
		de.safeNotify(ctx, ExecutionEvent{
			Type:        EventTypeNodeRetrying,
			ExecutionID: execState.ExecutionID,
			WorkflowID:  execState.WorkflowID,
			Timestamp:   time.Now(),
			Status:      "retrying",
			NodeID:      node.ID,
			NodeName:    node.Name,
			NodeType:    node.Type,
			Error:       err,
		})
	}

	execErr = retryPolicy.Execute(nodeCtx, func() error {
		result, err := de.nodeExecutor.Execute(nodeCtx, nodeExecCtx)
		if result != nil {
			execResult = result
		}
		return err
	})

	if execErr != nil {
		nodeEndTime := time.Now()
		execState.SetNodeError(node.ID, execErr)
		execState.SetNodeStatus(node.ID, models.NodeExecutionStatusFailed)
		execState.SetNodeEndTime(node.ID, nodeEndTime)

		if execResult != nil {
			execState.SetNodeInput(node.ID, execResult.Input)
			execState.SetNodeConfig(node.ID, execResult.Config)
			execState.SetNodeResolvedConfig(node.ID, execResult.ResolvedConfig)
		}

		nodeDuration := time.Since(nodeStartTime).Milliseconds()
		de.safeNotify(ctx, ExecutionEvent{
			Type:        EventTypeNodeFailed,
			ExecutionID: execState.ExecutionID,
			WorkflowID:  execState.WorkflowID,
			Timestamp:   time.Now(),
			Status:      "failed",
			NodeID:      node.ID,
			NodeName:    node.Name,
			NodeType:    node.Type,
			Error:       execErr,
			DurationMs:  nodeDuration,
		})

		return execErr
	}

	nodeEndTime := time.Now()

	// Check output size
	if opts.MaxOutputSize > 0 {
		outputSize := EstimateSize(execResult.Output)
		if outputSize > opts.MaxOutputSize {
			err := fmt.Errorf("node output size (%d bytes) exceeds limit (%d bytes)", outputSize, opts.MaxOutputSize)
			execState.SetNodeError(node.ID, err)
			execState.SetNodeStatus(node.ID, models.NodeExecutionStatusFailed)
			execState.SetNodeEndTime(node.ID, nodeEndTime)
			return err
		}
	}

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
			de.safeNotify(ctx, ExecutionEvent{
				Type:        EventTypeNodeCompleted,
				ExecutionID: execState.ExecutionID,
				WorkflowID:  execState.WorkflowID,
				Timestamp:   time.Now(),
				Status:      "warning",
				NodeID:      node.ID,
				Message:     fmt.Sprintf("Total memory usage (%d) exceeds limit (%d)", totalMemory, opts.MaxTotalMemory),
			})
		}
	}

	nodeDuration := time.Since(nodeStartTime).Milliseconds()
	de.safeNotify(ctx, ExecutionEvent{
		Type:        EventTypeNodeCompleted,
		ExecutionID: execState.ExecutionID,
		WorkflowID:  execState.WorkflowID,
		Timestamp:   time.Now(),
		Status:      "completed",
		NodeID:      node.ID,
		NodeName:    node.Name,
		NodeType:    node.Type,
		DurationMs:  nodeDuration,
		Output:      ToMapInterface(execResult.Output),
	})

	return nil
}

// processLoopEdges checks if any loop edge should fire after the current wave.
// Returns the wave index to jump to, or -1 if no loop fires.
func (de *DAGExecutor) processLoopEdges(
	ctx context.Context,
	execState *ExecutionState,
	dag *DAG,
	waves [][]*models.Node,
	currentWave int,
) int {
	for _, edge := range dag.LoopEdges {
		// Check if loop source is in the current wave and completed
		sourceWave := findNodeWave(waves, edge.From)
		if sourceWave != currentWave {
			continue
		}

		sourceStatus, _ := execState.GetNodeStatus(edge.From)
		if sourceStatus != models.NodeExecutionStatusCompleted {
			continue
		}

		maxIter := edge.Loop.MaxIterations
		currentIter := execState.GetLoopIteration(edge.ID)

		if currentIter >= maxIter {
			de.safeNotify(ctx, ExecutionEvent{
				Type:          EventTypeLoopExhausted,
				ExecutionID:   execState.ExecutionID,
				WorkflowID:    execState.WorkflowID,
				Timestamp:     time.Now(),
				NodeID:        edge.From,
				LoopEdgeID:    edge.ID,
				LoopIteration: currentIter,
				LoopMaxIter:   maxIter,
				Message:       fmt.Sprintf("loop %s exhausted after %d iterations", edge.ID, maxIter),
			})
			continue
		}

		// Fire the loop
		newIter := execState.IncrementLoopIteration(edge.ID)

		// Set loop input: output of source becomes input of target
		if output, ok := execState.GetNodeOutput(edge.From); ok {
			execState.SetLoopInput(edge.To, output)
		}

		targetWave := findNodeWave(waves, edge.To)
		if targetWave < 0 {
			continue
		}

		de.resetWaveRange(execState, waves, targetWave, currentWave)

		de.safeNotify(ctx, ExecutionEvent{
			Type:          EventTypeLoopIteration,
			ExecutionID:   execState.ExecutionID,
			WorkflowID:    execState.WorkflowID,
			Timestamp:     time.Now(),
			NodeID:        edge.To,
			LoopEdgeID:    edge.ID,
			LoopIteration: newIter,
			LoopMaxIter:   maxIter,
			Message:       fmt.Sprintf("loop %s iteration %d/%d: jumping from wave %d to wave %d", edge.ID, newIter, maxIter, currentWave, targetWave),
		})

		return targetWave
	}

	return -1
}

// resetWaveRange resets execution state for all nodes in waves [from, to].
func (de *DAGExecutor) resetWaveRange(
	execState *ExecutionState,
	waves [][]*models.Node,
	from, to int,
) {
	for waveIdx := from; waveIdx <= to; waveIdx++ {
		if waveIdx < 0 || waveIdx >= len(waves) {
			continue
		}
		for _, node := range waves[waveIdx] {
			execState.ResetNodeForLoop(node.ID)
		}
	}
}

// safeNotify wraps notifications with panic recovery.
func (de *DAGExecutor) safeNotify(ctx context.Context, event ExecutionEvent) {
	if de.notifier == nil {
		return
	}

	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("Notifier panicked: %v\n", r)
		}
	}()

	de.notifier.Notify(ctx, event)
}

// shouldExecuteNode checks if a node should be executed based on incoming edge conditions.
// A node is executed if AT LEAST ONE incoming edge passes all checks (OR semantics).
func (de *DAGExecutor) shouldExecuteNode(
	execState *ExecutionState,
	node *models.Node,
) (bool, string) {
	// If node has loop input, it should always execute
	if _, hasLoopInput := execState.GetLoopInput(node.ID); hasLoopInput {
		return true, ""
	}

	workflow := execState.Workflow

	incomingEdges := CollectRegularIncomingEdges(workflow.Edges, node.ID)

	if len(incomingEdges) == 0 {
		return true, ""
	}

	hasValidPath := false
	allSkipReasons := []string{}

	for _, edge := range incomingEdges {
		sourceNode := FindNodeByID(workflow.Nodes, edge.From)
		if sourceNode == nil {
			continue
		}

		sourceStatus, _ := execState.GetNodeStatus(sourceNode.ID)
		if sourceStatus == models.NodeExecutionStatusSkipped {
			allSkipReasons = append(allSkipReasons, fmt.Sprintf("parent %s skipped", sourceNode.ID))
			continue
		}

		if sourceStatus != models.NodeExecutionStatusCompleted {
			allSkipReasons = append(allSkipReasons, fmt.Sprintf("parent %s not completed (%s)", sourceNode.ID, sourceStatus))
			continue
		}

		// Evaluate edge condition
		if edge.Condition != "" {
			output, _ := execState.GetNodeOutput(sourceNode.ID)
			passed, err := de.conditionEvaluator.Evaluate(edge.Condition, output)
			if err != nil {
				allSkipReasons = append(allSkipReasons, fmt.Sprintf("edge from %s: condition error: %v", sourceNode.ID, err))
				continue
			}
			if !passed {
				allSkipReasons = append(allSkipReasons, fmt.Sprintf("edge from %s: condition '%s' is false", sourceNode.ID, edge.Condition))
				continue
			}
		}

		// Check sourceHandle routing for conditional nodes
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

		hasValidPath = true
		break
	}

	if hasValidPath {
		return true, ""
	}

	skipReason := "no valid incoming path"
	if len(allSkipReasons) > 0 {
		skipReason = fmt.Sprintf("no valid incoming path: %v", allSkipReasons)
	}
	return false, skipReason
}

// evaluateSourceHandleCondition checks if the edge's sourceHandle matches
// the output of a conditional node.
func evaluateSourceHandleCondition(
	edge *models.Edge,
	execState *ExecutionState,
	sourceNode *models.Node,
) (bool, error) {
	output, ok := execState.GetNodeOutput(sourceNode.ID)
	if !ok {
		return false, fmt.Errorf("conditional node %s has no output", sourceNode.ID)
	}

	if boolOutput, ok := output.(bool); ok {
		switch edge.SourceHandle {
		case SourceHandleTrue:
			return boolOutput, nil
		case SourceHandleFalse:
			return !boolOutput, nil
		default:
			return true, nil
		}
	}

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

	return true, nil
}

// convertRetryPolicy converts pkg/engine RetryPolicy to InternalRetryPolicy.
func convertRetryPolicy(rp *RetryPolicy) *InternalRetryPolicy {
	if rp == nil {
		return NoInternalRetryPolicy()
	}

	strategy := InternalBackoffConstant
	switch rp.BackoffStrategy {
	case BackoffLinear:
		strategy = InternalBackoffLinear
	case BackoffExponential:
		strategy = InternalBackoffExponential
	}

	return &InternalRetryPolicy{
		MaxAttempts:     rp.MaxAttempts,
		InitialDelay:    rp.InitialDelay,
		MaxDelay:        rp.MaxDelay,
		BackoffStrategy: strategy,
		RetryableErrors: rp.RetryOn,
	}
}
