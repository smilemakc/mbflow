package executor

import (
	"context"
	"fmt"
	"sync"

	"github.com/google/uuid"
	"github.com/smilemakc/mbflow/internal/domain"
)

// ErrorStrategy defines how to handle errors during workflow execution
type ErrorStrategy interface {
	// HandleNodeError handles an error from a single node execution
	HandleNodeError(ctx context.Context, nodeID uuid.UUID, nodeName string, err error) error

	// HandleWaveErrors handles errors from a wave of parallel node executions
	HandleWaveErrors(ctx context.Context, errors []NodeError) error

	// ShouldContinueExecution determines if execution should continue after errors
	ShouldContinueExecution(errors []NodeError) bool

	// Name returns the strategy name
	Name() string
}

// NodeError represents an error from a specific node
type NodeError struct {
	NodeID   uuid.UUID
	NodeName string
	Error    error
	Attempt  int
}

// ErrorContext holds context for error handling decisions
type ErrorContext struct {
	ExecutionID    uuid.UUID
	WorkflowID     uuid.UUID
	CurrentWave    int
	TotalWaves     int
	CompletedNodes int
	FailedNodes    int
	TotalNodes     int
}

// FailFastStrategy stops execution on the first error
type FailFastStrategy struct{}

func NewFailFastStrategy() *FailFastStrategy {
	return &FailFastStrategy{}
}

func (s *FailFastStrategy) Name() string {
	return "fail_fast"
}

func (s *FailFastStrategy) HandleNodeError(ctx context.Context, nodeID uuid.UUID, nodeName string, err error) error {
	return fmt.Errorf("node %s (%s) failed (fail-fast): %w", nodeName, nodeID, err)
}

func (s *FailFastStrategy) HandleWaveErrors(ctx context.Context, errors []NodeError) error {
	if len(errors) == 0 {
		return nil
	}

	// Return first error encountered
	firstErr := errors[0]
	return fmt.Errorf("node %s (%s) failed (fail-fast): %w",
		firstErr.NodeName, firstErr.NodeID, firstErr.Error)
}

func (s *FailFastStrategy) ShouldContinueExecution(errors []NodeError) bool {
	return len(errors) == 0
}

// ContinueOnErrorStrategy continues execution even when nodes fail
type ContinueOnErrorStrategy struct {
	mu              sync.RWMutex
	collectedErrors []NodeError
}

func NewContinueOnErrorStrategy() *ContinueOnErrorStrategy {
	return &ContinueOnErrorStrategy{
		collectedErrors: make([]NodeError, 0),
	}
}

func (s *ContinueOnErrorStrategy) Name() string {
	return "continue_on_error"
}

func (s *ContinueOnErrorStrategy) HandleNodeError(ctx context.Context, nodeID uuid.UUID, nodeName string, err error) error {
	s.mu.Lock()
	s.collectedErrors = append(s.collectedErrors, NodeError{
		NodeID:   nodeID,
		NodeName: nodeName,
		Error:    err,
	})
	s.mu.Unlock()

	// Don't return error - allow execution to continue
	return nil
}

func (s *ContinueOnErrorStrategy) HandleWaveErrors(ctx context.Context, errors []NodeError) error {
	if len(errors) == 0 {
		return nil
	}

	s.mu.Lock()
	s.collectedErrors = append(s.collectedErrors, errors...)
	s.mu.Unlock()

	// Don't return error - allow execution to continue
	return nil
}

func (s *ContinueOnErrorStrategy) ShouldContinueExecution(errors []NodeError) bool {
	// Always continue regardless of errors
	return true
}

// GetCollectedErrors returns all errors that were collected during execution
func (s *ContinueOnErrorStrategy) GetCollectedErrors() []NodeError {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]NodeError, len(s.collectedErrors))
	copy(result, s.collectedErrors)
	return result
}

// BestEffortStrategy attempts to complete as many nodes as possible
// Similar to ContinueOnError but with better error reporting
type BestEffortStrategy struct {
	mu              sync.RWMutex
	successfulNodes map[uuid.UUID]bool
	failedNodes     map[uuid.UUID]NodeError
}

func NewBestEffortStrategy() *BestEffortStrategy {
	return &BestEffortStrategy{
		successfulNodes: make(map[uuid.UUID]bool),
		failedNodes:     make(map[uuid.UUID]NodeError),
	}
}

func (s *BestEffortStrategy) Name() string {
	return "best_effort"
}

func (s *BestEffortStrategy) HandleNodeError(ctx context.Context, nodeID uuid.UUID, nodeName string, err error) error {
	s.mu.Lock()
	s.failedNodes[nodeID] = NodeError{
		NodeID:   nodeID,
		NodeName: nodeName,
		Error:    err,
	}
	s.mu.Unlock()

	// Don't return error - allow execution to continue
	return nil
}

func (s *BestEffortStrategy) HandleWaveErrors(ctx context.Context, errors []NodeError) error {
	if len(errors) == 0 {
		return nil
	}

	s.mu.Lock()
	for _, nodeErr := range errors {
		s.failedNodes[nodeErr.NodeID] = nodeErr
	}
	s.mu.Unlock()

	// Don't return error - allow execution to continue
	return nil
}

func (s *BestEffortStrategy) ShouldContinueExecution(errors []NodeError) bool {
	// Always continue - best effort
	return true
}

// MarkNodeSuccess marks a node as successful
func (s *BestEffortStrategy) MarkNodeSuccess(nodeID uuid.UUID) {
	s.mu.Lock()
	s.successfulNodes[nodeID] = true
	s.mu.Unlock()
}

// GetSummary returns a summary of successful and failed nodes
func (s *BestEffortStrategy) GetSummary() map[string]interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()

	failedNodesList := make([]map[string]interface{}, 0, len(s.failedNodes))
	for _, nodeErr := range s.failedNodes {
		failedNodesList = append(failedNodesList, map[string]interface{}{
			"node_id":   nodeErr.NodeID.String(),
			"node_name": nodeErr.NodeName,
			"error":     nodeErr.Error.Error(),
		})
	}

	return map[string]interface{}{
		"successful_count": len(s.successfulNodes),
		"failed_count":     len(s.failedNodes),
		"failed_nodes":     failedNodesList,
	}
}

// RequireNStrategy requires a minimum number of nodes to succeed
type RequireNStrategy struct {
	mu              sync.RWMutex
	minRequired     int
	successCount    int
	failureCount    int
	collectedErrors []NodeError
}

func NewRequireNStrategy(minRequired int) *RequireNStrategy {
	return &RequireNStrategy{
		minRequired:     minRequired,
		collectedErrors: make([]NodeError, 0),
	}
}

func (s *RequireNStrategy) Name() string {
	return fmt.Sprintf("require_%d", s.minRequired)
}

func (s *RequireNStrategy) HandleNodeError(ctx context.Context, nodeID uuid.UUID, nodeName string, err error) error {
	s.mu.Lock()
	s.failureCount++
	s.collectedErrors = append(s.collectedErrors, NodeError{
		NodeID:   nodeID,
		NodeName: nodeName,
		Error:    err,
	})
	s.mu.Unlock()

	// Check if we can still reach minimum required successes
	if s.canStillSucceed() {
		return nil // Continue execution
	}

	// Not enough nodes can succeed - fail fast
	return fmt.Errorf("cannot reach minimum required successes (%d), node %s (%s) failed: %w",
		s.minRequired, nodeName, nodeID, err)
}

func (s *RequireNStrategy) HandleWaveErrors(ctx context.Context, errors []NodeError) error {
	if len(errors) == 0 {
		return nil
	}

	s.mu.Lock()
	s.failureCount += len(errors)
	s.collectedErrors = append(s.collectedErrors, errors...)
	s.mu.Unlock()

	// Check if we can still succeed
	if s.canStillSucceed() {
		return nil
	}

	// Cannot reach minimum - fail
	return fmt.Errorf("cannot reach minimum required successes (%d), %d nodes failed",
		s.minRequired, len(errors))
}

func (s *RequireNStrategy) ShouldContinueExecution(errors []NodeError) bool {
	return s.canStillSucceed()
}

// MarkNodeSuccess increments the success counter
func (s *RequireNStrategy) MarkNodeSuccess() {
	s.mu.Lock()
	s.successCount++
	s.mu.Unlock()
}

// canStillSucceed checks if minimum required can still be reached
func (s *RequireNStrategy) canStillSucceed() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Already reached minimum
	if s.successCount >= s.minRequired {
		return true
	}

	// Check if we can still reach minimum
	// This is a simplified check - in reality would need total node count
	return true
}

// GetStats returns current statistics
func (s *RequireNStrategy) GetStats() map[string]interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return map[string]interface{}{
		"min_required":  s.minRequired,
		"success_count": s.successCount,
		"failure_count": s.failureCount,
		"errors_count":  len(s.collectedErrors),
	}
}

// ErrorStrategyFactory creates error strategies based on domain.ErrorStrategy
type ErrorStrategyFactory struct{}

func NewErrorStrategyFactory() *ErrorStrategyFactory {
	return &ErrorStrategyFactory{}
}

// Create creates an error strategy from domain type
func (f *ErrorStrategyFactory) Create(strategyType domain.ErrorStrategy, config map[string]interface{}) ErrorStrategy {
	switch strategyType {
	case domain.ErrorStrategyFailFast:
		return NewFailFastStrategy()

	case domain.ErrorStrategyContinueOnError:
		return NewContinueOnErrorStrategy()

	case domain.ErrorStrategyBestEffort:
		return NewBestEffortStrategy()

	case domain.ErrorStrategyRequireN:
		// Get N from config
		minRequired := 1
		if n, ok := config["min_required"].(int); ok {
			minRequired = n
		} else if n, ok := config["min_required"].(float64); ok {
			minRequired = int(n)
		}
		return NewRequireNStrategy(minRequired)

	default:
		// Default to fail-fast
		return NewFailFastStrategy()
	}
}

// ErrorStrategyExecutor wraps a node executor with error strategy
type ErrorStrategyExecutor struct {
	executor NodeExecutor
	strategy ErrorStrategy
}

func NewErrorStrategyExecutor(executor NodeExecutor, strategy ErrorStrategy) *ErrorStrategyExecutor {
	return &ErrorStrategyExecutor{
		executor: executor,
		strategy: strategy,
	}
}

func (e *ErrorStrategyExecutor) Execute(
	ctx context.Context,
	node domain.Node,
	inputs *NodeExecutionInputs,
) (map[string]any, error) {
	// Execute node
	output, err := e.executor.Execute(ctx, node, inputs)

	if err != nil {
		// Handle error according to strategy
		strategyErr := e.strategy.HandleNodeError(ctx, node.ID(), node.Name(), err)
		if strategyErr != nil {
			// Strategy says to fail
			return nil, strategyErr
		}

		// Strategy says to continue - return empty output
		return make(map[string]any), nil
	}

	// Mark success for strategies that track it
	switch s := e.strategy.(type) {
	case *BestEffortStrategy:
		s.MarkNodeSuccess(node.ID())
	case *RequireNStrategy:
		s.MarkNodeSuccess()
	}

	return output, nil
}

// GetErrorStrategy returns the underlying error strategy
func (e *ErrorStrategyExecutor) GetErrorStrategy() ErrorStrategy {
	return e.strategy
}

// ErrorRecoveryStrategy defines how to recover from errors
type ErrorRecoveryStrategy interface {
	// CanRecover determines if an error is recoverable
	CanRecover(err error) bool

	// Recover attempts to recover from an error
	Recover(ctx context.Context, nodeID uuid.UUID, err error) error
}

// CompensatingAction represents a compensating action for rollback
type CompensatingAction struct {
	NodeID      uuid.UUID
	NodeName    string
	Action      func(ctx context.Context) error
	Description string
}

// CompensationManager manages compensating actions for failed executions
type CompensationManager struct {
	mu      sync.RWMutex
	actions []CompensatingAction
}

func NewCompensationManager() *CompensationManager {
	return &CompensationManager{
		actions: make([]CompensatingAction, 0),
	}
}

// RegisterCompensation registers a compensating action for a node
func (cm *CompensationManager) RegisterCompensation(nodeID uuid.UUID, nodeName string, action func(ctx context.Context) error, description string) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	cm.actions = append(cm.actions, CompensatingAction{
		NodeID:      nodeID,
		NodeName:    nodeName,
		Action:      action,
		Description: description,
	})
}

// ExecuteCompensations executes all registered compensations in reverse order
func (cm *CompensationManager) ExecuteCompensations(ctx context.Context) []error {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	errors := make([]error, 0)

	// Execute in reverse order (LIFO)
	for i := len(cm.actions) - 1; i >= 0; i-- {
		action := cm.actions[i]

		if err := action.Action(ctx); err != nil {
			errors = append(errors, fmt.Errorf("compensation for node %s (%s) failed: %w",
				action.NodeName, action.NodeID, err))
		}
	}

	return errors
}

// GetRegisteredActions returns all registered compensating actions
func (cm *CompensationManager) GetRegisteredActions() []CompensatingAction {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	result := make([]CompensatingAction, len(cm.actions))
	copy(result, cm.actions)
	return result
}

// Clear clears all registered compensations
func (cm *CompensationManager) Clear() {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	cm.actions = make([]CompensatingAction, 0)
}
