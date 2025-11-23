package monitoring

import (
	"fmt"
	"sync"
	"time"

	"mbflow/internal/domain"
)

// ExecutionObserver defines the interface for observing workflow execution events.
// Implementations can use this to monitor, log, or react to execution events.
type ExecutionObserver interface {
	// OnExecutionStarted is called when a workflow execution starts
	OnExecutionStarted(workflowID, executionID string)

	// OnExecutionCompleted is called when a workflow execution completes successfully
	OnExecutionCompleted(workflowID, executionID string, duration time.Duration)

	// OnExecutionFailed is called when a workflow execution fails
	OnExecutionFailed(workflowID, executionID string, err error, duration time.Duration)

	// OnNodeStarted is called when a node starts executing
	// node can be nil if only config is available
	OnNodeStarted(executionID string, node *domain.Node, attemptNumber int)

	// OnNodeCompleted is called when a node completes successfully
	// node can be nil if only config is available
	OnNodeCompleted(executionID string, node *domain.Node, output interface{}, duration time.Duration)

	// OnNodeFailed is called when a node fails
	// node can be nil if only config is available
	OnNodeFailed(executionID string, node *domain.Node, err error, duration time.Duration, willRetry bool)

	// OnNodeRetrying is called when a node is being retried
	// node can be nil if only config is available
	OnNodeRetrying(executionID string, node *domain.Node, attemptNumber int, delay time.Duration)

	// OnVariableSet is called when a variable is set in the execution context
	OnVariableSet(executionID, key string, value interface{})

	// OnNodeCallbackStarted is called when a node callback starts processing
	OnNodeCallbackStarted(executionID string, node *domain.Node)

	// OnNodeCallbackCompleted is called when a node callback completes
	// err is nil if the callback succeeded, non-nil if it failed
	OnNodeCallbackCompleted(executionID string, node *domain.Node, err error, duration time.Duration)
}

// ObserverManager manages multiple observers and notifies them of events.
// It implements the observer pattern for workflow execution monitoring.
type ObserverManager struct {
	observers []ExecutionObserver
	mu        sync.RWMutex
}

// NewObserverManager creates a new ObserverManager.
func NewObserverManager() *ObserverManager {
	return &ObserverManager{
		observers: make([]ExecutionObserver, 0),
	}
}

// AddObserver adds an observer to the manager.
func (om *ObserverManager) AddObserver(observer ExecutionObserver) {
	om.mu.Lock()
	defer om.mu.Unlock()
	om.observers = append(om.observers, observer)
}

// RemoveObserver removes an observer from the manager.
func (om *ObserverManager) RemoveObserver(observer ExecutionObserver) {
	om.mu.Lock()
	defer om.mu.Unlock()

	for i, obs := range om.observers {
		if obs == observer {
			om.observers = append(om.observers[:i], om.observers[i+1:]...)
			return
		}
	}
}

// NotifyExecutionStarted notifies all observers that an execution has started.
func (om *ObserverManager) NotifyExecutionStarted(workflowID, executionID string) {
	om.mu.RLock()
	defer om.mu.RUnlock()

	for _, observer := range om.observers {
		observer.OnExecutionStarted(workflowID, executionID)
	}
}

// NotifyExecutionCompleted notifies all observers that an execution has completed.
func (om *ObserverManager) NotifyExecutionCompleted(workflowID, executionID string, duration time.Duration) {
	om.mu.RLock()
	defer om.mu.RUnlock()

	for _, observer := range om.observers {
		observer.OnExecutionCompleted(workflowID, executionID, duration)
	}
}

// NotifyExecutionFailed notifies all observers that an execution has failed.
func (om *ObserverManager) NotifyExecutionFailed(workflowID, executionID string, err error, duration time.Duration) {
	om.mu.RLock()
	defer om.mu.RUnlock()

	for _, observer := range om.observers {
		observer.OnExecutionFailed(workflowID, executionID, err, duration)
	}
}

// NotifyNodeStarted notifies all observers that a node has started.
func (om *ObserverManager) NotifyNodeStarted(executionID string, node *domain.Node, attemptNumber int) {
	om.mu.RLock()
	defer om.mu.RUnlock()

	for _, observer := range om.observers {
		observer.OnNodeStarted(executionID, node, attemptNumber)
	}
}

// NotifyNodeCompleted notifies all observers that a node has completed.
func (om *ObserverManager) NotifyNodeCompleted(executionID string, node *domain.Node, output interface{}, duration time.Duration) {
	om.mu.RLock()
	defer om.mu.RUnlock()

	for _, observer := range om.observers {
		observer.OnNodeCompleted(executionID, node, output, duration)
	}
}

// NotifyNodeFailed notifies all observers that a node has failed.
func (om *ObserverManager) NotifyNodeFailed(executionID string, node *domain.Node, err error, duration time.Duration, willRetry bool) {
	om.mu.RLock()
	defer om.mu.RUnlock()

	for _, observer := range om.observers {
		observer.OnNodeFailed(executionID, node, err, duration, willRetry)
	}
}

// NotifyNodeRetrying notifies all observers that a node is being retried.
func (om *ObserverManager) NotifyNodeRetrying(executionID string, node *domain.Node, attemptNumber int, delay time.Duration) {
	om.mu.RLock()
	defer om.mu.RUnlock()

	for _, observer := range om.observers {
		observer.OnNodeRetrying(executionID, node, attemptNumber, delay)
	}
}

// NotifyVariableSet notifies all observers that a variable has been set.
func (om *ObserverManager) NotifyVariableSet(executionID, key string, value interface{}) {
	om.mu.RLock()
	defer om.mu.RUnlock()

	for _, observer := range om.observers {
		observer.OnVariableSet(executionID, key, value)
	}
}

// NotifyNodeCallbackStarted notifies all observers that a node callback has started.
func (om *ObserverManager) NotifyNodeCallbackStarted(executionID string, node *domain.Node) {
	om.mu.RLock()
	defer om.mu.RUnlock()

	for _, observer := range om.observers {
		observer.OnNodeCallbackStarted(executionID, node)
	}
}

// NotifyNodeCallbackCompleted notifies all observers that a node callback has completed.
func (om *ObserverManager) NotifyNodeCallbackCompleted(executionID string, node *domain.Node, err error, duration time.Duration) {
	om.mu.RLock()
	defer om.mu.RUnlock()

	for _, observer := range om.observers {
		observer.OnNodeCallbackCompleted(executionID, node, err, duration)
	}
}

// CompositeObserver combines logging, metrics, and tracing into a single observer.
// This is a convenience implementation that integrates all monitoring components.
type CompositeObserver struct {
	logger  LegacyExecutionLogger
	metrics *MetricsCollector
	trace   *ExecutionTrace
}

// NewCompositeObserver creates a new CompositeObserver.
// The logger parameter should implement LegacyExecutionLogger for full compatibility.
// If the logger only implements ExecutionLogger (the minimal interface),
// you can wrap it with a legacy adapter or use the Log method directly.
func NewCompositeObserver(logger LegacyExecutionLogger, metrics *MetricsCollector, trace *ExecutionTrace) *CompositeObserver {
	return &CompositeObserver{
		logger:  logger,
		metrics: metrics,
		trace:   trace,
	}
}

// OnExecutionStarted implements ExecutionObserver.
func (co *CompositeObserver) OnExecutionStarted(workflowID, executionID string) {
	if co.logger != nil {
		co.logger.LogExecutionStarted(workflowID, executionID)
	}
	if co.trace != nil {
		co.trace.AddEvent("execution_started", "", "", "Workflow execution started", nil, nil)
	}
}

// OnExecutionCompleted implements ExecutionObserver.
func (co *CompositeObserver) OnExecutionCompleted(workflowID, executionID string, duration time.Duration) {
	if co.logger != nil {
		co.logger.LogExecutionCompleted(workflowID, executionID, duration)
	}
	if co.metrics != nil {
		co.metrics.RecordWorkflowExecution(workflowID, duration, true)
	}
	if co.trace != nil {
		co.trace.AddEvent("execution_completed", "", "", "Workflow execution completed",
			map[string]interface{}{"duration": duration}, nil)
	}
}

// OnExecutionFailed implements ExecutionObserver.
func (co *CompositeObserver) OnExecutionFailed(workflowID, executionID string, err error, duration time.Duration) {
	if co.logger != nil {
		co.logger.LogExecutionFailed(workflowID, executionID, err, duration)
	}
	if co.metrics != nil {
		co.metrics.RecordWorkflowExecution(workflowID, duration, false)
	}
	if co.trace != nil {
		co.trace.AddEvent("execution_failed", "", "", "Workflow execution failed",
			map[string]interface{}{"duration": duration}, err)
	}
}

// OnNodeStarted implements ExecutionObserver.
func (co *CompositeObserver) OnNodeStarted(executionID string, node *domain.Node, attemptNumber int) {
	if co.logger != nil {
		co.logger.LogNodeStarted(executionID, node, attemptNumber)
	}
	if co.trace != nil {
		nodeID := ""
		nodeType := ""
		if node != nil {
			nodeID = node.ID()
			nodeType = node.Type()
		}
		co.trace.AddEvent("node_started", nodeID, nodeType, "Node execution started",
			map[string]interface{}{"attempt": attemptNumber}, nil)
	}
}

// OnNodeCompleted implements ExecutionObserver.
func (co *CompositeObserver) OnNodeCompleted(executionID string, node *domain.Node, output interface{}, duration time.Duration) {
	if co.logger != nil {
		co.logger.LogNodeCompleted(executionID, node, duration)
	}
	if co.metrics != nil {
		nodeType := ""
		if node != nil {
			nodeType = node.Type()
		}
		co.metrics.RecordNodeExecution(nodeType, duration, true, false)
	}
	if co.trace != nil {
		nodeID := ""
		nodeType := ""
		if node != nil {
			nodeID = node.ID()
			nodeType = node.Type()
		}
		co.trace.AddEvent("node_completed", nodeID, nodeType, "Node execution completed",
			map[string]interface{}{"duration": duration}, nil)
	}
}

// OnNodeFailed implements ExecutionObserver.
func (co *CompositeObserver) OnNodeFailed(executionID string, node *domain.Node, err error, duration time.Duration, willRetry bool) {
	if co.logger != nil {
		co.logger.LogNodeFailed(executionID, node, err, duration, willRetry)
	}
	if co.metrics != nil {
		nodeType := ""
		if node != nil {
			nodeType = node.Type()
		}
		co.metrics.RecordNodeExecution(nodeType, duration, false, false)
	}
	if co.trace != nil {
		nodeID := ""
		nodeType := ""
		if node != nil {
			nodeID = node.ID()
			nodeType = node.Type()
		}
		co.trace.AddEvent("node_failed", nodeID, nodeType, "Node execution failed",
			map[string]interface{}{"duration": duration, "will_retry": willRetry}, err)
	}
}

// OnNodeRetrying implements ExecutionObserver.
func (co *CompositeObserver) OnNodeRetrying(executionID string, node *domain.Node, attemptNumber int, delay time.Duration) {
	if co.logger != nil {
		co.logger.LogNodeRetrying(executionID, node, attemptNumber, delay)
	}
	if co.trace != nil {
		nodeID := ""
		if node != nil {
			nodeID = node.ID()
		}
		co.trace.AddEvent("node_retrying", nodeID, "", "Node being retried",
			map[string]interface{}{"attempt": attemptNumber, "delay": delay}, nil)
	}
}

// OnVariableSet implements ExecutionObserver.
func (co *CompositeObserver) OnVariableSet(executionID, key string, value interface{}) {
	if co.logger != nil {
		co.logger.LogVariableSet(executionID, key, value)
	}
	if co.trace != nil {
		co.trace.AddEvent("variable_set", "", "", "Variable set",
			map[string]interface{}{"key": key, "value": value}, nil)
	}
}

// OnNodeCallbackStarted implements ExecutionObserver.
func (co *CompositeObserver) OnNodeCallbackStarted(executionID string, node *domain.Node) {
	if co.logger != nil {
		nodeID := ""
		if node != nil {
			nodeID = node.ID()
		}
		co.logger.LogInfo(executionID, fmt.Sprintf("Callback started for node %s", nodeID))
	}
	if co.trace != nil {
		nodeID := ""
		nodeType := ""
		if node != nil {
			nodeID = node.ID()
			nodeType = node.Type()
		}
		co.trace.AddEvent("node_callback_started", nodeID, nodeType, "Node callback started", nil, nil)
	}
}

// OnNodeCallbackCompleted implements ExecutionObserver.
func (co *CompositeObserver) OnNodeCallbackCompleted(executionID string, node *domain.Node, err error, duration time.Duration) {
	if co.logger != nil {
		nodeID := ""
		if node != nil {
			nodeID = node.ID()
		}
		if err != nil {
			co.logger.LogError(executionID, fmt.Sprintf("Callback failed for node %s (duration: %v)", nodeID, duration), err)
		} else {
			co.logger.LogInfo(executionID, fmt.Sprintf("Callback completed for node %s (duration: %v)", nodeID, duration))
		}
	}
	if co.trace != nil {
		nodeID := ""
		nodeType := ""
		if node != nil {
			nodeID = node.ID()
			nodeType = node.Type()
		}
		co.trace.AddEvent("node_callback_completed", nodeID, nodeType, "Node callback completed",
			map[string]interface{}{"duration": duration}, err)
	}
}
