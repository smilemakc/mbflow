package mbflow

import (
	"context"
	"time"

	executor "mbflow/internal/application/executor"
	"mbflow/internal/domain"
	"mbflow/internal/infrastructure/monitoring"
)

// Workflow represents a workflow process.
type Workflow interface {
	ID() string
	Name() string
	Version() string
	Spec() map[string]any
	CreatedAt() time.Time
}

// Execution represents a workflow execution instance.
type Execution interface {
	ID() string
	WorkflowID() string
	Status() string
	StartedAt() time.Time
	FinishedAt() *time.Time
}

// Node represents a step in a workflow.
type Node interface {
	ID() string
	WorkflowID() string
	Type() string
	Name() string
	Config() map[string]any
}

// Edge represents a connection between nodes.
type Edge interface {
	ID() string
	WorkflowID() string
	FromNodeID() string
	ToNodeID() string
	Type() string
	Config() map[string]any
}

// Trigger represents a trigger for starting a workflow.
type Trigger interface {
	ID() string
	WorkflowID() string
	Type() string
	Config() map[string]any
}

// Event represents a system event.
type Event interface {
	EventID() string
	EventType() string
	WorkflowID() string
	ExecutionID() string
	WorkflowName() string
	NodeID() string
	Timestamp() time.Time
	Payload() []byte
	Metadata() map[string]string
}

// WorkflowRepository defines the interface for workflow operations.
type WorkflowRepository interface {
	SaveWorkflow(ctx context.Context, w Workflow) error
	GetWorkflow(ctx context.Context, id string) (Workflow, error)
	ListWorkflows(ctx context.Context) ([]Workflow, error)
}

// ExecutionRepository defines the interface for execution operations.
type ExecutionRepository interface {
	SaveExecution(ctx context.Context, e Execution) error
	GetExecution(ctx context.Context, id string) (Execution, error)
	ListExecutions(ctx context.Context) ([]Execution, error)
}

// EventRepository defines the interface for event operations.
type EventRepository interface {
	AppendEvent(ctx context.Context, e Event) error
	ListEventsByExecution(ctx context.Context, executionID string) ([]Event, error)
}

// NodeRepository defines the interface for node operations.
type NodeRepository interface {
	SaveNode(ctx context.Context, n Node) error
	GetNode(ctx context.Context, id string) (Node, error)
	ListNodes(ctx context.Context, workflowID string) ([]Node, error)
}

// EdgeRepository defines the interface for edge operations.
type EdgeRepository interface {
	SaveEdge(ctx context.Context, e Edge) error
	GetEdge(ctx context.Context, id string) (Edge, error)
	ListEdges(ctx context.Context, workflowID string) ([]Edge, error)
}

// TriggerRepository defines the interface for trigger operations.
type TriggerRepository interface {
	SaveTrigger(ctx context.Context, t Trigger) error
	GetTrigger(ctx context.Context, id string) (Trigger, error)
	ListTriggers(ctx context.Context, workflowID string) ([]Trigger, error)
}

// Storage combines all repositories.
type Storage interface {
	WorkflowRepository
	ExecutionRepository
	EventRepository
	NodeRepository
	EdgeRepository
	TriggerRepository
}

// Executor represents a workflow executor.
// It provides methods for executing workflows and nodes.
type Executor interface {
	// ExecuteWorkflow executes a complete workflow.
	// If edges are provided, uses graph-based traversal with parallel execution support.
	// If edges are empty, falls back to sequential execution for backward compatibility.
	ExecuteWorkflow(ctx context.Context, workflowID, executionID string, nodes []ExecutorNodeConfig, edges []ExecutorEdgeConfig, initialVariables map[string]interface{}) (ExecutorState, error)

	// ExecuteNode executes a single node
	ExecuteNode(ctx context.Context, state ExecutorState, nodeConfig ExecutorNodeConfig) error

	// AddObserver adds an execution observer
	AddObserver(observer ExecutionObserver)

	// GetMetrics returns execution metrics
	GetMetrics() ExecutorMetrics
}

// ExecutorNodeConfig represents the configuration for executing a node.
type ExecutorNodeConfig = executor.NodeConfig

// ExecutorEdgeConfig represents the configuration for an edge in the workflow graph.
type ExecutorEdgeConfig = executor.EdgeConfig

// NodeToConfig converts a domain Node to ExecutorNodeConfig for execution.
// This function extracts execution-relevant information (ID, Type, Config) from a domain Node entity.
// The workflowID and name fields are omitted as they are not needed for execution.
func NodeToConfig(node Node) ExecutorNodeConfig {
	return ExecutorNodeConfig{
		NodeID:   node.ID(),
		NodeType: node.Type(),
		Config:   node.Config(),
	}
}

// NodesToConfigs converts a slice of domain Nodes to ExecutorNodeConfigs for execution.
// This is useful when loading nodes from storage and preparing them for workflow execution.
func NodesToConfigs(nodes []Node) []ExecutorNodeConfig {
	configs := make([]ExecutorNodeConfig, len(nodes))
	for i, node := range nodes {
		configs[i] = NodeToConfig(node)
	}
	return configs
}

// EdgeToConfig converts a domain Edge to ExecutorEdgeConfig for execution.
// This function extracts execution-relevant information (FromNodeID, ToNodeID, Type, Config) from a domain Edge entity.
// The edge ID and workflowID are omitted as they are not needed for execution.
func EdgeToConfig(edge Edge) ExecutorEdgeConfig {
	return ExecutorEdgeConfig{
		FromNodeID: edge.FromNodeID(),
		ToNodeID:   edge.ToNodeID(),
		EdgeType:   edge.Type(),
		Config:     edge.Config(),
	}
}

// EdgesToConfigs converts a slice of domain Edges to ExecutorEdgeConfigs for execution.
// This is useful when loading edges from storage and preparing them for workflow execution.
func EdgesToConfigs(edges []Edge) []ExecutorEdgeConfig {
	configs := make([]ExecutorEdgeConfig, len(edges))
	for i, edge := range edges {
		configs[i] = EdgeToConfig(edge)
	}
	return configs
}

// ExecutorState represents the state of a workflow execution.
type ExecutorState interface {
	// ExecutionID returns the execution ID
	ExecutionID() string

	// WorkflowID returns the workflow ID
	WorkflowID() string

	// Status returns the current status
	Status() string

	// GetVariable retrieves a variable
	GetVariable(key string) (interface{}, bool)

	// GetAllVariables returns all variables
	GetAllVariables() map[string]interface{}

	// GetExecutionDuration returns the execution duration
	GetExecutionDuration() string
}

// ExecutionObserver defines the interface for observing workflow execution events.
// This is the public interface that uses the Node interface.
type ExecutionObserver interface {
	// OnExecutionStarted is called when a workflow execution starts
	OnExecutionStarted(workflowID, executionID string)

	// OnExecutionCompleted is called when a workflow execution completes successfully
	OnExecutionCompleted(workflowID, executionID string, duration time.Duration)

	// OnExecutionFailed is called when a workflow execution fails
	OnExecutionFailed(workflowID, executionID string, err error, duration time.Duration)

	// OnNodeStarted is called when a node starts executing
	// node can be nil if only config is available
	OnNodeStarted(executionID string, node Node, attemptNumber int)

	// OnNodeCompleted is called when a node completes successfully
	// node can be nil if only config is available
	OnNodeCompleted(executionID string, node Node, output interface{}, duration time.Duration)

	// OnNodeFailed is called when a node fails
	// node can be nil if only config is available
	OnNodeFailed(executionID string, node Node, err error, duration time.Duration, willRetry bool)

	// OnNodeRetrying is called when a node is being retried
	// node can be nil if only config is available
	OnNodeRetrying(executionID string, node Node, attemptNumber int, delay time.Duration)

	// OnVariableSet is called when a variable is set in the execution context
	OnVariableSet(executionID, key string, value interface{})
}

// ExecutorMetrics provides execution metrics.
type ExecutorMetrics interface {
	// GetWorkflowMetrics returns metrics for a workflow
	GetWorkflowMetrics(workflowID string) map[string]interface{}

	// GetNodeMetrics returns metrics for a node type
	GetNodeMetrics(nodeType string) map[string]interface{}

	// GetAIMetrics returns AI API usage metrics
	GetAIMetrics() map[string]interface{}

	// GetSummary returns a summary of all metrics
	GetSummary() map[string]interface{}
}

// ExecutorConfig configures the workflow executor.
type ExecutorConfig struct {
	// OpenAIAPIKey is the API key for OpenAI
	OpenAIAPIKey string

	// MaxRetryAttempts is the maximum number of retry attempts
	MaxRetryAttempts int

	// EnableMonitoring enables monitoring and logging
	EnableMonitoring bool

	// VerboseLogging enables verbose logging
	VerboseLogging bool
}

// HTTPCallbackObserver sends execution events to an HTTP callback URL.
// It implements the ExecutionObserver interface and sends POST requests
// with JSON payloads for each execution event.
type HTTPCallbackObserver struct {
	internal *monitoring.HTTPCallbackObserver
}

// HTTPCallbackConfig holds configuration for HTTPCallbackObserver.
type HTTPCallbackConfig = monitoring.HTTPCallbackConfig

// NewHTTPCallbackObserver creates a new HTTPCallbackObserver with the given configuration.
func NewHTTPCallbackObserver(config HTTPCallbackConfig) (*HTTPCallbackObserver, error) {
	internal, err := monitoring.NewHTTPCallbackObserver(config)
	if err != nil {
		return nil, err
	}
	return &HTTPCallbackObserver{internal: internal}, nil
}

// OnExecutionStarted implements ExecutionObserver.
func (o *HTTPCallbackObserver) OnExecutionStarted(workflowID, executionID string) {
	o.internal.OnExecutionStarted(workflowID, executionID)
}

// OnExecutionCompleted implements ExecutionObserver.
func (o *HTTPCallbackObserver) OnExecutionCompleted(workflowID, executionID string, duration time.Duration) {
	o.internal.OnExecutionCompleted(workflowID, executionID, duration)
}

// OnExecutionFailed implements ExecutionObserver.
func (o *HTTPCallbackObserver) OnExecutionFailed(workflowID, executionID string, err error, duration time.Duration) {
	o.internal.OnExecutionFailed(workflowID, executionID, err, duration)
}

// OnNodeStarted implements ExecutionObserver.
func (o *HTTPCallbackObserver) OnNodeStarted(executionID string, node Node, attemptNumber int) {
	var domainNode *domain.Node
	if node != nil {
		// Convert public Node interface to domain.Node
		domainNode = domain.NewNode(
			node.ID(),
			node.WorkflowID(),
			node.Type(),
			node.Name(),
			node.Config(),
		)
	}
	o.internal.OnNodeStarted(executionID, domainNode, attemptNumber)
}

// OnNodeCompleted implements ExecutionObserver.
func (o *HTTPCallbackObserver) OnNodeCompleted(executionID string, node Node, output interface{}, duration time.Duration) {
	var domainNode *domain.Node
	if node != nil {
		domainNode = domain.NewNode(
			node.ID(),
			node.WorkflowID(),
			node.Type(),
			node.Name(),
			node.Config(),
		)
	}
	o.internal.OnNodeCompleted(executionID, domainNode, output, duration)
}

// OnNodeFailed implements ExecutionObserver.
func (o *HTTPCallbackObserver) OnNodeFailed(executionID string, node Node, err error, duration time.Duration, willRetry bool) {
	var domainNode *domain.Node
	if node != nil {
		domainNode = domain.NewNode(
			node.ID(),
			node.WorkflowID(),
			node.Type(),
			node.Name(),
			node.Config(),
		)
	}
	o.internal.OnNodeFailed(executionID, domainNode, err, duration, willRetry)
}

// OnNodeRetrying implements ExecutionObserver.
func (o *HTTPCallbackObserver) OnNodeRetrying(executionID string, node Node, attemptNumber int, delay time.Duration) {
	var domainNode *domain.Node
	if node != nil {
		domainNode = domain.NewNode(
			node.ID(),
			node.WorkflowID(),
			node.Type(),
			node.Name(),
			node.Config(),
		)
	}
	o.internal.OnNodeRetrying(executionID, domainNode, attemptNumber, delay)
}

// OnVariableSet implements ExecutionObserver.
func (o *HTTPCallbackObserver) OnVariableSet(executionID, key string, value interface{}) {
	o.internal.OnVariableSet(executionID, key, value)
}

// SetEnabled enables or disables the observer.
func (o *HTTPCallbackObserver) SetEnabled(enabled bool) {
	o.internal.SetEnabled(enabled)
}

// IsEnabled returns whether the observer is enabled.
func (o *HTTPCallbackObserver) IsEnabled() bool {
	return o.internal.IsEnabled()
}
