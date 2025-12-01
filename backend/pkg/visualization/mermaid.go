package visualization

import (
	"fmt"
	"strings"

	"github.com/smilemakc/mbflow/pkg/models"
)

// MermaidRenderer renders workflows as Mermaid flowchart diagrams.
type MermaidRenderer struct{}

// NewMermaidRenderer creates a new Mermaid renderer.
func NewMermaidRenderer() *MermaidRenderer {
	return &MermaidRenderer{}
}

// Format returns the format identifier.
func (r *MermaidRenderer) Format() string {
	return "mermaid"
}

// Render converts a workflow into Mermaid flowchart syntax.
func (r *MermaidRenderer) Render(workflow *models.Workflow, opts *RenderOptions) (string, error) {
	if workflow == nil {
		return "", fmt.Errorf("workflow is nil")
	}

	if opts == nil {
		opts = DefaultRenderOptions()
	}

	var sb strings.Builder

	// Write header
	sb.WriteString("flowchart ")
	sb.WriteString(opts.Direction)
	sb.WriteString("\n")

	// Write nodes
	for _, node := range workflow.Nodes {
		sb.WriteString("    ")
		sb.WriteString(r.renderNode(node, opts))
		sb.WriteString("\n")
	}

	// Write edges
	if len(workflow.Edges) > 0 {
		sb.WriteString("\n")
		for _, edge := range workflow.Edges {
			sb.WriteString("    ")
			sb.WriteString(r.renderEdge(edge, opts))
			sb.WriteString("\n")
		}
	}

	return sb.String(), nil
}

// renderNode formats a single node based on its type.
func (r *MermaidRenderer) renderNode(node *models.Node, opts *RenderOptions) string {
	label := r.buildNodeLabel(node, opts)

	// Choose shape based on node type
	switch node.Type {
	case "http":
		// Rectangle: ["label"]
		return fmt.Sprintf(`%s["%s"]`, node.ID, label)
	case "llm":
		// Stadium: (["label"])
		return fmt.Sprintf(`%s(["%s"])`, node.ID, label)
	case "transform":
		// Trapezoid: [/"label"/]
		return fmt.Sprintf(`%s[/"%s"/]`, node.ID, label)
	case "conditional":
		// Diamond: {"label"}
		return fmt.Sprintf(`%s{"%s"}`, node.ID, label)
	case "merge":
		// Hexagon: {{"label"}}
		return fmt.Sprintf(`%s{{"%s"}}`, node.ID, label)
	default:
		// Default rectangle for custom types
		return fmt.Sprintf(`%s["%s"]`, node.ID, label)
	}
}

// buildNodeLabel constructs the node label with type prefix and configuration.
func (r *MermaidRenderer) buildNodeLabel(node *models.Node, opts *RenderOptions) string {
	var parts []string

	// Add type prefix
	typePrefix := r.getNodeTypePrefix(node)
	if typePrefix != "" {
		parts = append(parts, typePrefix)
	}

	// Add node name
	if node.Name != "" {
		parts = append(parts, node.Name)
	} else {
		parts = append(parts, node.ID)
	}

	label := strings.Join(parts, ": ")

	// Add configuration details if requested
	if opts.ShowConfig && len(node.Config) > 0 {
		configStr := r.extractKeyConfig(node)
		if configStr != "" {
			label = label + "<br/>" + configStr
		}
	}

	// Escape special characters for Mermaid
	label = strings.ReplaceAll(label, `"`, `&quot;`)

	return label
}

// getNodeTypePrefix returns a human-readable prefix for the node type.
func (r *MermaidRenderer) getNodeTypePrefix(node *models.Node) string {
	switch node.Type {
	case "http":
		method, ok := node.Config["method"].(string)
		if ok && method != "" {
			return "HTTP: " + method
		}
		return "HTTP"
	case "llm":
		provider, _ := node.Config["provider"].(string)
		if provider == "" {
			provider = "LLM"
		}
		return provider
	case "transform":
		return "Transform"
	case "conditional":
		return "If"
	case "merge":
		return "Merge"
	default:
		return strings.ToUpper(node.Type)
	}
}

// extractKeyConfig extracts key configuration parameters for display.
func (r *MermaidRenderer) extractKeyConfig(node *models.Node) string {
	switch node.Type {
	case "http":
		url, _ := node.Config["url"].(string)
		return url
	case "llm":
		model, _ := node.Config["model"].(string)
		if model != "" {
			return model
		}
	case "transform":
		transformType, _ := node.Config["type"].(string)
		if transformType != "" {
			return transformType
		}
	}
	return ""
}

// renderEdge formats an edge connection.
func (r *MermaidRenderer) renderEdge(edge *models.Edge, opts *RenderOptions) string {
	// Check if edge has a condition
	if opts.ShowConditions && edge.Condition != "" {
		// Labeled arrow: from -->|condition| to
		return fmt.Sprintf("%s -->|%s| %s", edge.From, edge.Condition, edge.To)
	}

	// Simple arrow: from --> to
	return fmt.Sprintf("%s --> %s", edge.From, edge.To)
}
