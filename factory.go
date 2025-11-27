package mbflow

import (
	"github.com/google/uuid"
	"github.com/smilemakc/mbflow/internal/domain"
)

// NewWorkflow creates a new workflow.
// If id is uuid.Nil, a new UUID will be generated automatically.
// Note: Use WorkflowBuilder for a more convenient way to build workflows.
func NewWorkflow(name, version, description string, spec map[string]any) (Workflow, error) {
	return domain.NewWorkflow(name, version, description, spec)
}

// NewExecution creates a new workflow execution.
// If id is uuid.Nil, a new UUID will be generated automatically.
func NewExecution(id, workflowID uuid.UUID) (Execution, error) {
	return domain.NewExecution(id, workflowID)
}

// WorkflowBuilder provides a fluent interface for building workflows.
// This is the recommended way to create workflows with nodes, edges, and triggers.
//
// Example:
//
//	workflow, err := mbflow.NewWorkflowBuilder("My Workflow", "1.0").
//		WithDescription("A sample workflow").
//		AddNode(mbflow.NodeTypeHTTPRequest, "Fetch Data", map[string]any{"url": "https://api.example.com"}).
//		AddNode(mbflow.NodeTypeTransform, "Process Data", map[string]any{"script": "..."}).
//		AddEdge("Fetch Data", "Process Data", "direct", nil).
//		AddTrigger("manual", nil).
//		Build()
type WorkflowBuilder struct {
	workflow domain.Workflow
	err      error
}

// NewWorkflowBuilder creates a new workflow builder.
// name - the workflow name (required)
// version - the workflow version (required)
func NewWorkflowBuilder(name, version string) *WorkflowBuilder {
	wf, err := domain.NewWorkflow(name, version, "", make(map[string]any))
	if err != nil {
		return &WorkflowBuilder{err: err}
	}
	return &WorkflowBuilder{
		workflow: wf,
	}
}

// WithID sets a specific workflow ID.
// If not called, a random UUID will be generated.
func (b *WorkflowBuilder) WithID(id uuid.UUID) *WorkflowBuilder {
	if b.err != nil {
		return b
	}
	// Recreate workflow with specific ID
	wf, err := domain.RestoreWorkflow(id, b.workflow.Name(), b.workflow.Version(), b.workflow.Description(), b.workflow.Spec())
	if err != nil {
		b.err = err
		return b
	}
	b.workflow = wf
	return b
}

// WithDescription sets the workflow description.
func (b *WorkflowBuilder) WithDescription(description string) *WorkflowBuilder {
	if b.err != nil {
		return b
	}
	// Recreate workflow with description
	wf, err := domain.RestoreWorkflow(b.workflow.ID(), b.workflow.Name(), b.workflow.Version(), description, b.workflow.Spec())
	if err != nil {
		b.err = err
		return b
	}
	// Copy nodes, edges, triggers from old workflow
	b.workflow = wf
	return b
}

// WithSpec sets the workflow specification (metadata).
func (b *WorkflowBuilder) WithSpec(spec map[string]any) *WorkflowBuilder {
	if b.err != nil {
		return b
	}
	// Recreate workflow with spec
	wf, err := domain.RestoreWorkflow(b.workflow.ID(), b.workflow.Name(), b.workflow.Version(), b.workflow.Description(), spec)
	if err != nil {
		b.err = err
		return b
	}
	b.workflow = wf
	return b
}

// UseNode adds an existing Node to the workflow, updating the builder's state with any encountered errors.
func (b *WorkflowBuilder) UseNode(node domain.Node) *WorkflowBuilder {
	if err := b.workflow.UseNode(node); err != nil {
		b.err = err
	}
	return b
}

// AddNode adds a node to the workflow.
// nodeType - the type of node (use NodeType constants from mbflow package)
// name - unique name for the node
// config - node-specific configuration
func (b *WorkflowBuilder) AddNode(nodeType string, name string, config map[string]any) *WorkflowBuilder {
	if b.err != nil {
		return b
	}

	// Convert string to NodeType
	nt := domain.NodeType(nodeType)
	_, err := b.workflow.AddNode(nt, name, config)
	if err != nil {
		b.err = err
	}
	return b
}

// AddNodeWithConfig adds a node to the workflow using a structured config.
// This is a type-safe alternative to AddNode that accepts NodeConfig structs
// instead of map[string]any.
//
// Example:
//
//	builder.AddNodeWithConfig(mbflow.NodeTypeHTTPRequest, "fetch", &mbflow.HTTPRequestConfig{
//	    URL: "https://api.example.com",
//	    Method: "GET",
//	    OutputKey: "response",
//	})
func (b *WorkflowBuilder) AddNodeWithConfig(nodeType string, name string, config NodeConfig) *WorkflowBuilder {
	if b.err != nil {
		return b
	}

	// Convert structured config to map[string]any
	configMap, err := config.ToMap()
	if err != nil {
		b.err = err
		return b
	}

	// Use the existing AddNode method
	return b.AddNode(nodeType, name, configMap)
}

// AddEdge adds an edge between two nodes identified by their names.
// fromNodeName - name of the source node
// toNodeName - name of the destination node
// edgeType - type of edge ("direct", "conditional", "fork", "join")
// config - edge-specific configuration (e.g., condition for conditional edges)
func (b *WorkflowBuilder) AddEdge(fromNodeName, toNodeName, edgeType string, config map[string]any) *WorkflowBuilder {
	if b.err != nil {
		return b
	}

	// Find nodes by name
	var fromNodeID, toNodeID uuid.UUID
	for _, n := range b.workflow.GetAllNodes() {
		if n.Name() == fromNodeName {
			fromNodeID = n.ID()
		}
		if n.Name() == toNodeName {
			toNodeID = n.ID()
		}
	}

	if fromNodeID == uuid.Nil {
		b.err = domain.NewDomainError(domain.ErrCodeNotFound, "from node not found: "+fromNodeName, nil)
		return b
	}
	if toNodeID == uuid.Nil {
		b.err = domain.NewDomainError(domain.ErrCodeNotFound, "to node not found: "+toNodeName, nil)
		return b
	}

	// Convert string to EdgeType
	et := domain.EdgeType(edgeType)

	_, err := b.workflow.AddEdge(fromNodeID, toNodeID, et, config)
	if err != nil {
		b.err = err
	}
	return b
}

// AddEdgeWithDataSources adds an edge with additional upstream data sources.
// This is a convenience wrapper around AddEdge that sets the include_outputs_from config.
// The target node will receive namespaced variables from the specified additional source nodes
// in addition to its direct parent outputs.
//
// Example:
//
//	builder.AddEdgeWithDataSources("enhance_content", "aggregate", "direct",
//	                               []string{"generate_content", "analyze_quality"})
//
// The "aggregate" node will receive:
//   - Variables from "enhance_content" (direct parent) - auto-merged based on collision strategy
//   - Variables from "generate_content" - namespaced as "generate_content_*"
//   - Variables from "analyze_quality" - namespaced as "analyze_quality_*"
func (b *WorkflowBuilder) AddEdgeWithDataSources(
	fromNodeName, toNodeName, edgeType string,
	additionalSources []string,
) *WorkflowBuilder {
	config := map[string]any{
		"include_outputs_from": additionalSources,
	}
	return b.AddEdge(fromNodeName, toNodeName, edgeType, config)
}

// AddTrigger adds a trigger to the workflow.
// triggerType - type of trigger ("manual", "auto", "http", "schedule", "event")
// config - trigger-specific configuration
func (b *WorkflowBuilder) AddTrigger(triggerType string, config map[string]any) *WorkflowBuilder {
	if b.err != nil {
		return b
	}

	// Convert string to TriggerType
	tt := domain.TriggerType(triggerType)

	_, err := b.workflow.AddTrigger(tt, config)
	if err != nil {
		b.err = err
	}
	return b
}

// Build finalizes the workflow and returns it.
// It validates the workflow structure before returning.
func (b *WorkflowBuilder) Build() (Workflow, error) {
	if b.err != nil {
		return nil, b.err
	}
	// Validate workflow
	if err := b.workflow.Validate(); err != nil {
		return nil, err
	}

	return b.workflow, nil
}

// BuildWithoutValidation finalizes the workflow without validation.
// Use this if you want to save an incomplete workflow for later editing.
func (b *WorkflowBuilder) BuildWithoutValidation() (Workflow, error) {
	if b.err != nil {
		return nil, b.err
	}

	return b.workflow, nil
}
