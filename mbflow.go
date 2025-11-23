package mbflow

import (
	"context"
	"time"

	executor "github.com/smilemakc/mbflow/internal/application/executor"
	"github.com/smilemakc/mbflow/internal/domain"
	"github.com/smilemakc/mbflow/internal/infrastructure/monitoring"
)

// NodeExecutorType represents the type of a node executor.
// This is a type alias for string, allowing seamless use of string literals
// while providing convenient predefined constants.
type NodeExecutorType = executor.NodeExecutorType

// Node executor type constants.
// These define all available node types in the system.
const (
	// NodeTypeOpenAICompletion represents an OpenAI completion node.
	NodeTypeOpenAICompletion = executor.NodeTypeOpenAICompletion

	// NodeTypeOpenAIResponses represents an OpenAI Responses API node.
	NodeTypeOpenAIResponses = executor.NodeTypeOpenAIResponses

	// NodeTypeHTTPRequest represents an HTTP request node.
	NodeTypeHTTPRequest = executor.NodeTypeHTTPRequest

	// NodeTypeTelegramMessage represents a Telegram message node.
	NodeTypeTelegramMessage = executor.NodeTypeTelegramMessage

	// NodeTypeConditionalRouter represents a conditional routing node.
	NodeTypeConditionalRouter = executor.NodeTypeConditionalRouter

	// NodeTypeDataMerger represents a data merger node.
	NodeTypeDataMerger = executor.NodeTypeDataMerger

	// NodeTypeDataAggregator represents a data aggregator node.
	NodeTypeDataAggregator = executor.NodeTypeDataAggregator

	// NodeTypeScriptExecutor represents a script executor node.
	NodeTypeScriptExecutor = executor.NodeTypeScriptExecutor

	// NodeTypeJSONParser represents a JSON parser node.
	NodeTypeJSONParser = executor.NodeTypeJSONParser
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

// NodeConfig represents the configuration for executing a node.
type NodeConfig = domain.NodeConfig

// ExecutorEdgeConfig represents the configuration for an edge in the workflow graph.
type ExecutorEdgeConfig = executor.EdgeConfig

// NodeToConfig converts a domain Node to NodeConfig for execution.
// This function extracts execution-relevant information (ID, Type, Name, Config) from a domain Node entity.
// The workflowID field is omitted as it is not needed for execution.
func NodeToConfig(node Node) NodeConfig {
	return NodeConfig{
		ID:     node.ID(),
		Type:   node.Type(),
		Name:   node.Name(),
		Config: node.Config(),
	}
}

// NodesToConfigs converts a slice of domain Nodes to ExecutorNodeConfigs for execution.
// This is useful when loading nodes from storage and preparing them for workflow execution.
func NodesToConfigs(nodes []Node) []NodeConfig {
	configs := make([]NodeConfig, len(nodes))
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
	// GetExecutionID returns the execution ID
	GetExecutionID() string

	// GetWorkflowID returns the workflow ID
	GetWorkflowID() string

	// GetStatus returns the current status as string
	GetStatusString() string

	// GetVariable retrieves a variable
	GetVariable(key string) (interface{}, bool)

	// GetAllVariables returns all variables
	GetAllVariables() map[string]interface{}

	// GetExecutionDuration returns the execution duration
	GetExecutionDuration() time.Duration
}

// ExecutorMetrics provides execution metrics.
type ExecutorMetrics interface {
	// GetWorkflowMetrics returns metrics for a workflow
	GetWorkflowMetrics(workflowID string) *WorkflowMetrics

	// GetNodeMetrics returns aggregated metrics for a node type
	GetNodeMetrics(nodeType string) *NodeMetrics

	// GetNodeMetricsByID returns metrics for a specific node ID
	GetNodeMetricsByID(nodeID string) *NodeMetrics

	// GetAIMetrics returns AI API usage metrics
	GetAIMetrics() *AIMetrics

	// GetSummary returns a summary of all metrics
	GetSummary() *MetricsSummary
}

// WorkflowMetrics represents metrics for a workflow.
type WorkflowMetrics = monitoring.WorkflowMetrics

// NodeMetrics represents metrics for a node type.
type NodeMetrics = monitoring.NodeMetrics

// AIMetrics represents AI API usage metrics.
type AIMetrics = monitoring.AIMetrics

// MetricsSummary represents a summary of all collected metrics.
type MetricsSummary = monitoring.MetricsSummary
