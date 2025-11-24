package monitoring

import (
	"sync"
	"time"

	"github.com/smilemakc/mbflow/internal/domain"
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
	OnNodeStarted(workflowID, executionID string, node domain.Node, attemptNumber int)

	// OnNodeCompleted is called when a node completes successfully
	// node can be nil if only config is available
	OnNodeCompleted(workflowID, executionID string, node domain.Node, output any, duration time.Duration)

	// OnNodeFailed is called when a node fails
	// node can be nil if only config is available
	OnNodeFailed(workflowID, executionID string, node domain.Node, err error, duration time.Duration, willRetry bool)

	// OnNodeRetrying is called when a node is being retried
	// node can be nil if only config is available
	OnNodeRetrying(workflowID, executionID string, node domain.Node, attemptNumber int, delay time.Duration)

	// OnVariableSet is called when a variable is set in the execution context
	OnVariableSet(workflowID, executionID, key string, value any)

	// OnNodeCallbackStarted is called when a node callback starts processing
	OnNodeCallbackStarted(workflowID, executionID string, node domain.Node)

	// OnNodeCallbackCompleted is called when a node callback completes
	// err is nil if the callback succeeded, non-nil if it failed
	OnNodeCallbackCompleted(workflowID, executionID string, node domain.Node, err error, duration time.Duration)
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
func (om *ObserverManager) NotifyNodeStarted(workflowID, executionID string, node domain.Node, attemptNumber int) {
	om.mu.RLock()
	defer om.mu.RUnlock()

	for _, observer := range om.observers {
		observer.OnNodeStarted(workflowID, executionID, node, attemptNumber)
	}
}

// NotifyNodeCompleted notifies all observers that a node has completed.
func (om *ObserverManager) NotifyNodeCompleted(workflowID, executionID string, node domain.Node, output interface{}, duration time.Duration) {
	om.mu.RLock()
	defer om.mu.RUnlock()

	for _, observer := range om.observers {
		observer.OnNodeCompleted(workflowID, executionID, node, output, duration)
	}
}

// NotifyNodeFailed notifies all observers that a node has failed.
func (om *ObserverManager) NotifyNodeFailed(workflowID, executionID string, node domain.Node, err error, duration time.Duration, willRetry bool) {
	om.mu.RLock()
	defer om.mu.RUnlock()

	for _, observer := range om.observers {
		observer.OnNodeFailed(workflowID, executionID, node, err, duration, willRetry)
	}
}

// NotifyNodeRetrying notifies all observers that a node is being retried.
func (om *ObserverManager) NotifyNodeRetrying(workflowID, executionID string, node domain.Node, attemptNumber int, delay time.Duration) {
	om.mu.RLock()
	defer om.mu.RUnlock()

	for _, observer := range om.observers {
		observer.OnNodeRetrying(workflowID, executionID, node, attemptNumber, delay)
	}
}

// NotifyVariableSet notifies all observers that a variable has been set.
func (om *ObserverManager) NotifyVariableSet(workflowID, executionID, key string, value interface{}) {
	om.mu.RLock()
	defer om.mu.RUnlock()

	for _, observer := range om.observers {
		observer.OnVariableSet(workflowID, executionID, key, value)
	}
}

// NotifyNodeCallbackStarted notifies all observers that a node callback has started.
func (om *ObserverManager) NotifyNodeCallbackStarted(workflowID, executionID string, node domain.Node) {
	om.mu.RLock()
	defer om.mu.RUnlock()

	for _, observer := range om.observers {
		observer.OnNodeCallbackStarted(workflowID, executionID, node)
	}
}

// NotifyNodeCallbackCompleted notifies all observers that a node callback has completed.
func (om *ObserverManager) NotifyNodeCallbackCompleted(workflowID, executionID string, node domain.Node, err error, duration time.Duration) {
	om.mu.RLock()
	defer om.mu.RUnlock()

	for _, observer := range om.observers {
		observer.OnNodeCallbackCompleted(workflowID, executionID, node, err, duration)
	}
}

// CompositeObserver combines logging, metrics, and tracing into a single observer.
// This is a convenience implementation that integrates all monitoring components.
type CompositeObserver struct {
	logger  ExecutionLogger
	metrics *MetricsCollector
	trace   *ExecutionTrace
}

// NewCompositeObserver creates a new CompositeObserver.
func NewCompositeObserver(logger ExecutionLogger, metrics *MetricsCollector, trace *ExecutionTrace) *CompositeObserver {
	return &CompositeObserver{
		logger:  logger,
		metrics: metrics,
		trace:   trace,
	}
}

// OnExecutionStarted implements ExecutionObserver.
func (co *CompositeObserver) OnExecutionStarted(workflowID, executionID string) {
	if co.logger != nil {
		co.logger.Log(NewExecutionStartedEvent(workflowID, executionID))
	}
	if co.trace != nil {
		co.trace.AddEvent("execution_started", "", "", "Workflow execution started", nil, nil)
	}
}

// OnExecutionCompleted implements ExecutionObserver.
func (co *CompositeObserver) OnExecutionCompleted(workflowID, executionID string, duration time.Duration) {
	if co.logger != nil {
		co.logger.Log(NewExecutionCompletedEvent(workflowID, executionID, duration))
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
		co.logger.Log(NewExecutionFailedEvent(workflowID, executionID, err, duration))
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
func (co *CompositeObserver) OnNodeStarted(workflowID, executionID string, node domain.Node, attemptNumber int) {
	if co.logger != nil {
		co.logger.Log(NewNodeStartedEvent(workflowID, executionID, node, attemptNumber))
	}
	if co.trace != nil {
		nodeID := ""
		var nodeType domain.NodeType
		if node != nil {
			nodeID = node.ID().String()
			nodeType = node.Type()
		}
		co.trace.AddEvent("node_started", nodeID, string(nodeType), "Node execution started",
			map[string]interface{}{"attempt": attemptNumber}, nil)
	}
}

// OnNodeCompleted implements ExecutionObserver.
func (co *CompositeObserver) OnNodeCompleted(workflowID, executionID string, node domain.Node, output interface{}, duration time.Duration) {
	if co.logger != nil {
		co.logger.Log(NewNodeCompletedEvent(workflowID, executionID, node, output, duration))
	}
	if co.metrics != nil {
		nodeID := ""
		var nodeType domain.NodeType
		nodeName := ""
		if node != nil {
			nodeID = node.ID().String()
			nodeType = node.Type()
			nodeName = node.Name()
		}
		co.metrics.RecordNodeExecution(nodeID, string(nodeType), nodeName, duration, true, false)
	}
	if co.trace != nil {
		nodeID := ""
		var nodeType domain.NodeType
		if node != nil {
			nodeID = node.ID().String()
			nodeType = node.Type()
		}
		co.trace.AddEvent("node_completed", nodeID, string(nodeType), "Node execution completed",
			map[string]interface{}{"duration": duration}, nil)
	}
}

// OnNodeFailed implements ExecutionObserver.
func (co *CompositeObserver) OnNodeFailed(workflowID, executionID string, node domain.Node, err error, duration time.Duration, willRetry bool) {
	if co.logger != nil {
		co.logger.Log(NewNodeFailedEvent(workflowID, executionID, node, err, duration, willRetry))
	}
	if co.metrics != nil {
		nodeID := ""
		var nodeType domain.NodeType
		nodeName := ""
		if node != nil {
			nodeID = node.ID().String()
			nodeType = node.Type()
			nodeName = node.Name()
		}
		co.metrics.RecordNodeExecution(nodeID, string(nodeType), nodeName, duration, false, false)
	}
	if co.trace != nil {
		nodeID := ""
		var nodeType domain.NodeType
		if node != nil {
			nodeID = node.ID().String()
			nodeType = node.Type()
		}
		co.trace.AddEvent("node_failed", nodeID, string(nodeType), "Node execution failed",
			map[string]interface{}{"duration": duration, "will_retry": willRetry}, err)
	}
}

// OnNodeRetrying implements ExecutionObserver.
func (co *CompositeObserver) OnNodeRetrying(workflowID, executionID string, node domain.Node, attemptNumber int, delay time.Duration) {
	if co.logger != nil {
		co.logger.Log(NewNodeRetryingEvent(workflowID, executionID, node, attemptNumber, delay))
	}
	if co.trace != nil {
		nodeID := ""
		if node != nil {
			nodeID = node.ID().String()
		}
		co.trace.AddEvent("node_retrying", nodeID, "", "Node being retried",
			map[string]interface{}{"attempt": attemptNumber, "delay": delay}, nil)
	}
}

// OnVariableSet implements ExecutionObserver.
func (co *CompositeObserver) OnVariableSet(workflowID, executionID, key string, value interface{}) {
	if co.logger != nil {
		co.logger.Log(NewVariableSetEvent(workflowID, executionID, key, value))
	}
	if co.trace != nil {
		co.trace.AddEvent("variable_set", "", "", "Variable set",
			map[string]interface{}{"key": key, "value": value}, nil)
	}
}

// OnNodeCallbackStarted implements ExecutionObserver.
func (co *CompositeObserver) OnNodeCallbackStarted(workflowID, executionID string, node domain.Node) {
	if co.logger != nil {
		co.logger.Log(NewNodeCallbackStartedEvent(workflowID, executionID, node))
	}
	if co.trace != nil {
		nodeID := ""
		var nodeType domain.NodeType
		if node != nil {
			nodeID = node.ID().String()
			nodeType = node.Type()
		}
		co.trace.AddEvent("node_callback_started", nodeID, string(nodeType), "Node callback started", nil, nil)
	}
}

// OnNodeCallbackCompleted implements ExecutionObserver.
func (co *CompositeObserver) OnNodeCallbackCompleted(workflowID, executionID string, node domain.Node, err error, duration time.Duration) {
	if co.logger != nil {
		co.logger.Log(NewNodeCallbackCompletedEvent(workflowID, executionID, node, err, duration))
	}
	if co.trace != nil {
		nodeID := ""
		var nodeType domain.NodeType
		if node != nil {
			nodeID = node.ID().String()
			nodeType = node.Type()
		}
		co.trace.AddEvent("node_callback_completed", nodeID, string(nodeType), "Node callback completed",
			map[string]interface{}{"duration": duration}, err)
	}
}
