package executor

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/smilemakc/mbflow/internal/domain"
)

// JoinEvaluator evaluates join node conditions and determines when they should execute
type JoinEvaluator struct {
	mu sync.RWMutex

	// Track completion status of incoming branches for each join node
	branchStatus map[uuid.UUID]*JoinBranchStatus
}

// JoinBranchStatus tracks the status of branches leading to a join node
type JoinBranchStatus struct {
	JoinNodeID uuid.UUID
	Strategy   domain.JoinStrategy

	// Branch tracking
	IncomingBranches  []uuid.UUID                  // Node IDs of incoming branches
	CompletedBranches map[uuid.UUID]bool           // Which branches have completed
	BranchOutputs     map[uuid.UUID]map[string]any // Outputs from each branch
	BranchErrors      map[uuid.UUID]error          // Errors from branches

	// Timing
	FirstCompletionTime *time.Time
	LastCompletionTime  *time.Time

	// Configuration
	MinRequired int // For WaitN strategy

	// State
	Triggered bool
}

// NewJoinEvaluator creates a new join evaluator
func NewJoinEvaluator() *JoinEvaluator {
	return &JoinEvaluator{
		branchStatus: make(map[uuid.UUID]*JoinBranchStatus),
	}
}

// RegisterJoinNode registers a join node with its incoming branches
func (je *JoinEvaluator) RegisterJoinNode(
	joinNodeID uuid.UUID,
	strategy domain.JoinStrategy,
	incomingBranches []uuid.UUID,
	minRequired int,
) {
	je.mu.Lock()
	defer je.mu.Unlock()

	je.branchStatus[joinNodeID] = &JoinBranchStatus{
		JoinNodeID:        joinNodeID,
		Strategy:          strategy,
		IncomingBranches:  incomingBranches,
		CompletedBranches: make(map[uuid.UUID]bool),
		BranchOutputs:     make(map[uuid.UUID]map[string]any),
		BranchErrors:      make(map[uuid.UUID]error),
		MinRequired:       minRequired,
		Triggered:         false,
	}
}

// MarkBranchCompleted marks a branch as completed
func (je *JoinEvaluator) MarkBranchCompleted(
	joinNodeID uuid.UUID,
	branchNodeID uuid.UUID,
	output map[string]any,
	err error,
) error {
	je.mu.Lock()
	defer je.mu.Unlock()

	status, exists := je.branchStatus[joinNodeID]
	if !exists {
		return fmt.Errorf("join node %s not registered", joinNodeID)
	}

	// Check if this is a valid incoming branch
	isValid := false
	for _, incomingID := range status.IncomingBranches {
		if incomingID == branchNodeID {
			isValid = true
			break
		}
	}

	if !isValid {
		return fmt.Errorf("node %s is not an incoming branch for join node %s",
			branchNodeID, joinNodeID)
	}

	// Mark as completed
	status.CompletedBranches[branchNodeID] = true
	status.BranchOutputs[branchNodeID] = output

	if err != nil {
		status.BranchErrors[branchNodeID] = err
	}

	// Track timing
	now := time.Now()
	if status.FirstCompletionTime == nil {
		status.FirstCompletionTime = &now
	}
	status.LastCompletionTime = &now

	return nil
}

// ShouldTriggerJoin checks if a join node should be triggered
func (je *JoinEvaluator) ShouldTriggerJoin(joinNodeID uuid.UUID) (bool, error) {
	je.mu.RLock()
	defer je.mu.RUnlock()

	status, exists := je.branchStatus[joinNodeID]
	if !exists {
		return false, fmt.Errorf("join node %s not registered", joinNodeID)
	}

	// Already triggered
	if status.Triggered {
		return false, nil
	}

	// Evaluate based on strategy
	completedCount := len(status.CompletedBranches)
	totalBranches := len(status.IncomingBranches)

	switch status.Strategy {
	case domain.JoinStrategyWaitAll:
		// Wait for all branches to complete
		return completedCount == totalBranches, nil

	case domain.JoinStrategyWaitAny:
		// Trigger as soon as any branch completes
		return completedCount >= 1, nil

	case domain.JoinStrategyWaitFirst:
		// Trigger on the first completion (same as WaitAny for now)
		return completedCount >= 1, nil

	case domain.JoinStrategyWaitN:
		// Trigger when N branches complete
		return completedCount >= status.MinRequired, nil

	default:
		return false, fmt.Errorf("unknown join strategy: %s", status.Strategy)
	}
}

// MarkJoinTriggered marks a join node as triggered
func (je *JoinEvaluator) MarkJoinTriggered(joinNodeID uuid.UUID) error {
	je.mu.Lock()
	defer je.mu.Unlock()

	status, exists := je.branchStatus[joinNodeID]
	if !exists {
		return fmt.Errorf("join node %s not registered", joinNodeID)
	}

	status.Triggered = true
	return nil
}

// GetJoinInput collects and merges input from completed branches
func (je *JoinEvaluator) GetJoinInput(joinNodeID uuid.UUID) (map[string]any, error) {
	je.mu.RLock()
	defer je.mu.RUnlock()

	status, exists := je.branchStatus[joinNodeID]
	if !exists {
		return nil, fmt.Errorf("join node %s not registered", joinNodeID)
	}

	// Collect outputs from completed branches
	mergedInput := make(map[string]any)

	// Create array of branch outputs
	branchOutputs := make([]map[string]any, 0, len(status.CompletedBranches))
	for branchID := range status.CompletedBranches {
		if output, ok := status.BranchOutputs[branchID]; ok {
			branchOutputs = append(branchOutputs, output)

			// Merge into combined output
			for k, v := range output {
				mergedInput[k] = v
			}
		}
	}

	// Add metadata about the join
	mergedInput["_join_branch_count"] = len(status.CompletedBranches)
	mergedInput["_join_branches"] = branchOutputs
	mergedInput["_join_strategy"] = string(status.Strategy)

	if status.FirstCompletionTime != nil {
		mergedInput["_join_first_completion"] = status.FirstCompletionTime.Format(time.RFC3339)
	}
	if status.LastCompletionTime != nil {
		mergedInput["_join_last_completion"] = status.LastCompletionTime.Format(time.RFC3339)
	}

	return mergedInput, nil
}

// GetJoinStatus returns the current status of a join node
func (je *JoinEvaluator) GetJoinStatus(joinNodeID uuid.UUID) (*JoinBranchStatus, error) {
	je.mu.RLock()
	defer je.mu.RUnlock()

	status, exists := je.branchStatus[joinNodeID]
	if !exists {
		return nil, fmt.Errorf("join node %s not registered", joinNodeID)
	}

	// Return a copy to prevent external modification
	statusCopy := &JoinBranchStatus{
		JoinNodeID:        status.JoinNodeID,
		Strategy:          status.Strategy,
		IncomingBranches:  make([]uuid.UUID, len(status.IncomingBranches)),
		CompletedBranches: make(map[uuid.UUID]bool),
		BranchOutputs:     make(map[uuid.UUID]map[string]any),
		BranchErrors:      make(map[uuid.UUID]error),
		MinRequired:       status.MinRequired,
		Triggered:         status.Triggered,
	}

	copy(statusCopy.IncomingBranches, status.IncomingBranches)

	for k, v := range status.CompletedBranches {
		statusCopy.CompletedBranches[k] = v
	}

	for k, v := range status.BranchOutputs {
		statusCopy.BranchOutputs[k] = v
	}

	for k, v := range status.BranchErrors {
		statusCopy.BranchErrors[k] = v
	}

	if status.FirstCompletionTime != nil {
		t := *status.FirstCompletionTime
		statusCopy.FirstCompletionTime = &t
	}

	if status.LastCompletionTime != nil {
		t := *status.LastCompletionTime
		statusCopy.LastCompletionTime = &t
	}

	return statusCopy, nil
}

// Reset clears all join status (useful for workflow restart)
func (je *JoinEvaluator) Reset() {
	je.mu.Lock()
	defer je.mu.Unlock()

	je.branchStatus = make(map[uuid.UUID]*JoinBranchStatus)
}

// JoinExecutor executes join nodes by merging branch outputs
type JoinExecutor struct {
	evaluator *JoinEvaluator
}

// NewJoinExecutor creates a new join executor
func NewJoinExecutor(evaluator *JoinEvaluator) *JoinExecutor {
	return &JoinExecutor{
		evaluator: evaluator,
	}
}

// Execute executes a join node
func (je *JoinExecutor) Execute(
	ctx context.Context,
	node domain.Node,
	inputs *NodeExecutionInputs,
) (map[string]any, error) {
	// Get merged input from completed branches
	input, err := je.evaluator.GetJoinInput(node.ID())
	if err != nil {
		return nil, fmt.Errorf("failed to get join input: %w", err)
	}

	// Mark join as triggered
	if err := je.evaluator.MarkJoinTriggered(node.ID()); err != nil {
		return nil, fmt.Errorf("failed to mark join as triggered: %w", err)
	}

	// Join nodes simply pass through the merged input
	// The actual merging logic can be customized via node config

	mergeStrategy := "last_wins" // Default
	if strategy, ok := node.Config()["merge_strategy"].(string); ok {
		mergeStrategy = strategy
	}

	switch mergeStrategy {
	case "last_wins":
		// Already done in GetJoinInput - later values overwrite earlier ones
		return input, nil

	case "collect_all":
		// All branch outputs are already in _join_branches
		return input, nil

	case "first_only":
		// Return only the first branch's output
		if branches, ok := input["_join_branches"].([]map[string]any); ok && len(branches) > 0 {
			return branches[0], nil
		}
		return input, nil

	default:
		// Unknown strategy - use last_wins
		return input, nil
	}
}

// ParallelBranchExecutor executes multiple branches in parallel
type ParallelBranchExecutor struct {
	executor      NodeExecutor
	joinEvaluator *JoinEvaluator
}

// NewParallelBranchExecutor creates a new parallel branch executor
func NewParallelBranchExecutor(executor NodeExecutor, joinEvaluator *JoinEvaluator) *ParallelBranchExecutor {
	return &ParallelBranchExecutor{
		executor:      executor,
		joinEvaluator: joinEvaluator,
	}
}

// ExecuteBranches executes multiple branches in parallel
func (pbe *ParallelBranchExecutor) ExecuteBranches(
	ctx context.Context,
	branches []domain.Node,
	joinNodeID uuid.UUID,
	inputs *NodeExecutionInputs,
) error {
	var wg sync.WaitGroup
	errChan := make(chan error, len(branches))

	for _, branch := range branches {
		wg.Add(1)

		go func(node domain.Node) {
			defer wg.Done()

			// Execute branch
			output, err := pbe.executor.Execute(ctx, node, inputs)

			// Report completion to join evaluator
			if markErr := pbe.joinEvaluator.MarkBranchCompleted(joinNodeID, node.ID(), output, err); markErr != nil {
				errChan <- markErr
				return
			}

			if err != nil {
				errChan <- fmt.Errorf("branch %s failed: %w", node.Name(), err)
			}
		}(branch)
	}

	// Wait for all branches
	wg.Wait()
	close(errChan)

	// Collect errors
	var errors []error
	for err := range errChan {
		errors = append(errors, err)
	}

	if len(errors) > 0 {
		// Return first error (strategy can be customized)
		return errors[0]
	}

	return nil
}

// JoinNodeConfig holds configuration for a join node
type JoinNodeConfig struct {
	Strategy      domain.JoinStrategy
	MinRequired   int
	MergeStrategy string // "last_wins", "collect_all", "first_only"
	Timeout       time.Duration
}

// GetJoinConfig extracts join configuration from node config
func GetJoinConfig(node domain.Node) *JoinNodeConfig {
	config := node.Config()

	joinConfig := &JoinNodeConfig{
		Strategy:      domain.JoinStrategyWaitAll, // Default
		MinRequired:   1,
		MergeStrategy: "last_wins",
		Timeout:       0,
	}

	// Get strategy
	if strategy, ok := config["join_strategy"].(string); ok {
		joinConfig.Strategy = domain.JoinStrategy(strategy)
	}

	// Get min required
	if minReq, ok := config["min_required"].(int); ok {
		joinConfig.MinRequired = minReq
	} else if minReq, ok := config["min_required"].(float64); ok {
		joinConfig.MinRequired = int(minReq)
	}

	// Get merge strategy
	if mergeStrategy, ok := config["merge_strategy"].(string); ok {
		joinConfig.MergeStrategy = mergeStrategy
	}

	// Get timeout
	if timeout, ok := config["timeout"].(string); ok {
		if d, err := time.ParseDuration(timeout); err == nil {
			joinConfig.Timeout = d
		}
	} else if timeoutMs, ok := config["timeout_ms"].(float64); ok {
		joinConfig.Timeout = time.Duration(timeoutMs) * time.Millisecond
	}

	return joinConfig
}

// ForkNode represents a node that spawns parallel branches
type ForkNode struct {
	nodeID      uuid.UUID
	branches    []uuid.UUID
	joinNodeID  uuid.UUID
	parallelism int // Max parallel branches (0 = unlimited)
}

// ForkExecutor executes fork nodes that spawn parallel branches
type ForkExecutor struct {
	graph *WorkflowGraph
}

// NewForkExecutor creates a new fork executor
func NewForkExecutor(graph *WorkflowGraph) *ForkExecutor {
	return &ForkExecutor{
		graph: graph,
	}
}

// Execute executes a fork node
func (fe *ForkExecutor) Execute(
	ctx context.Context,
	node domain.Node,
	variables *domain.VariableSet,
) (map[string]any, error) {
	// Fork nodes just pass through their input
	// The actual branching is handled by the execution engine based on graph structure

	// Get outgoing edges to determine branches
	outgoingEdges := fe.graph.GetOutgoingEdges(node.ID())

	output := map[string]any{
		"_fork_branch_count": len(outgoingEdges),
		"_fork_node_id":      node.ID().String(),
	}

	// Check if parallelism is limited
	if maxParallel, ok := node.Config()["max_parallel"].(int); ok {
		output["_fork_max_parallel"] = maxParallel
	} else if maxParallel, ok := node.Config()["max_parallel"].(float64); ok {
		output["_fork_max_parallel"] = int(maxParallel)
	}

	return output, nil
}

// SynchronizationBarrier implements a synchronization primitive for parallel execution
type SynchronizationBarrier struct {
	mu sync.Mutex
	cv *sync.Cond

	expected int
	arrived  int
	released bool
}

// NewSynchronizationBarrier creates a new barrier
func NewSynchronizationBarrier(expected int) *SynchronizationBarrier {
	sb := &SynchronizationBarrier{
		expected: expected,
		arrived:  0,
		released: false,
	}
	sb.cv = sync.NewCond(&sb.mu)
	return sb
}

// Arrive marks one participant as arrived
func (sb *SynchronizationBarrier) Arrive() {
	sb.mu.Lock()
	defer sb.mu.Unlock()

	sb.arrived++

	if sb.arrived >= sb.expected {
		sb.released = true
		sb.cv.Broadcast()
	}
}

// Wait waits for all participants to arrive
func (sb *SynchronizationBarrier) Wait(ctx context.Context) error {
	sb.mu.Lock()
	defer sb.mu.Unlock()

	// Already released
	if sb.released {
		return nil
	}

	// Wait with context support
	done := make(chan struct{})
	go func() {
		sb.cv.Wait()
		close(done)
	}()

	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// Reset resets the barrier for reuse
func (sb *SynchronizationBarrier) Reset() {
	sb.mu.Lock()
	defer sb.mu.Unlock()

	sb.arrived = 0
	sb.released = false
}
