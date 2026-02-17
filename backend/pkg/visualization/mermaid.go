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

	// Write config block if theme variables are set
	if len(opts.ThemeVariables) > 0 || opts.Direction == "elk" {
		sb.WriteString("---\n")
		sb.WriteString("config:\n")

		// Layout configuration (elk is more adaptive for complex graphs)
		if opts.Direction == "elk" {
			sb.WriteString("  layout: elk\n")
		}

		// Theme configuration
		if len(opts.ThemeVariables) > 0 {
			sb.WriteString("  theme: base\n")
			sb.WriteString("  themeVariables:\n")
			for key, value := range opts.ThemeVariables {
				sb.WriteString(fmt.Sprintf("    %s: \"%s\"\n", key, value))
			}
		}

		sb.WriteString("---\n")
	}

	// Write header
	sb.WriteString("flowchart ")
	if opts.Direction != "elk" {
		sb.WriteString(opts.Direction)
	} else {
		sb.WriteString("TB") // Default direction for elk layout
	}
	sb.WriteString("\n")

	// Write nodes
	for _, node := range workflow.Nodes {
		sb.WriteString("    ")
		sb.WriteString(r.renderNode(node, opts))
		sb.WriteString("\n")
	}

	// Write edges (optimized with & for parallel connections)
	if len(workflow.Edges) > 0 {
		sb.WriteString("\n")

		// Group edges by source node for compact syntax
		edgesBySource := make(map[string][]*models.Edge)
		for _, edge := range workflow.Edges {
			edgesBySource[edge.From] = append(edgesBySource[edge.From], edge)
		}

		// Render grouped edges
		for sourceID, edges := range edgesBySource {
			if len(edges) == 1 {
				// Single edge - simple syntax
				sb.WriteString("    ")
				sb.WriteString(r.renderEdge(edges[0], opts))
				sb.WriteString("\n")
			} else {
				// Multiple edges from same source
				// Check if all edges have no conditions
				allNoConditions := true
				for _, edge := range edges {
					if (opts.ShowConditions && edge.Condition != "") || edge.IsLoop() {
						allNoConditions = false
						break
					}
				}

				if allNoConditions {
					// Compact: source --> target1 & target2 & target3
					sb.WriteString("    ")
					sb.WriteString(sourceID)
					sb.WriteString(" --> ")
					for i, edge := range edges {
						if i > 0 {
							sb.WriteString(" & ")
						}
						sb.WriteString(edge.To)
					}
					sb.WriteString("\n")
				} else {
					// With conditions - render separately
					for _, edge := range edges {
						sb.WriteString("    ")
						sb.WriteString(r.renderEdge(edge, opts))
						sb.WriteString("\n")
					}
				}
			}
		}
	}

	// Add styling for node types
	if opts.ShowConfig {
		sb.WriteString(r.renderNodeStyles())

		// Apply classes to nodes
		sb.WriteString("\n")
		sb.WriteString(r.applyNodeClasses(workflow))
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
		switch provider {
		case "openai":
			return "OpenAI"
		case "openai-responses":
			return "OpenAI Responses"
		case "anthropic":
			return "Anthropic"
		default:
			if provider != "" {
				return strings.ToUpper(provider)
			}
			return "LLM"
		}
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
		var parts []string

		// Add model
		model, _ := node.Config["model"].(string)
		if model != "" {
			parts = append(parts, model)
		}

		// For Responses API, show special features
		provider, _ := node.Config["provider"].(string)
		if provider == "openai-responses" {
			// Show hosted tools if present
			if hostedTools, ok := node.Config["hosted_tools"].([]any); ok && len(hostedTools) > 0 {
				var tools []string
				for _, tool := range hostedTools {
					if toolMap, ok := tool.(map[string]any); ok {
						if toolType, ok := toolMap["type"].(string); ok {
							switch toolType {
							case "web_search_preview":
								tools = append(tools, "🌐 Web Search")
							case "file_search":
								tools = append(tools, "📄 File Search")
							case "code_interpreter":
								tools = append(tools, "💻 Code")
							default:
								tools = append(tools, toolType)
							}
						}
					}
				}
				if len(tools) > 0 {
					parts = append(parts, strings.Join(tools, ", "))
				}
			}

			// Show reasoning effort if present
			if reasoning, ok := node.Config["reasoning"].(map[string]any); ok {
				if effort, ok := reasoning["effort"].(string); ok && effort != "" {
					parts = append(parts, "💭 "+effort+" reasoning")
				}
			}
		}

		return strings.Join(parts, "<br/>")
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
	// Loop edges use dotted lines with iteration label
	if edge.IsLoop() {
		label := fmt.Sprintf("loop (max %d)", edge.Loop.MaxIterations)
		return fmt.Sprintf(`%s -. "%s" .-> %s`, edge.From, label, edge.To)
	}

	// Check if edge has a condition
	if opts.ShowConditions && edge.Condition != "" {
		// Escape HTML entities in condition
		condition := r.escapeHTML(edge.Condition)
		// Labeled arrow with escaped condition
		return fmt.Sprintf(`%s -- "%s" --> %s`, edge.From, condition, edge.To)
	}

	// Simple arrow: from --> to
	return fmt.Sprintf("%s --> %s", edge.From, edge.To)
}

// escapeHTML escapes HTML special characters for Mermaid labels.
func (r *MermaidRenderer) escapeHTML(text string) string {
	text = strings.ReplaceAll(text, "&", "&amp;")
	text = strings.ReplaceAll(text, "<", "&lt;")
	text = strings.ReplaceAll(text, ">", "&gt;")
	text = strings.ReplaceAll(text, `"`, "&quot;")
	return text
}

// renderNodeStyles generates CSS styling for different node types.
func (r *MermaidRenderer) renderNodeStyles() string {
	var sb strings.Builder
	sb.WriteString("\n")
	sb.WriteString("    %% Node type styles\n")
	sb.WriteString("    classDef httpNode fill:#D0E6FF,stroke:#1A73E8,stroke-width:2px,color:#000\n")
	sb.WriteString("    classDef llmNode fill:#E8D9FF,stroke:#8E57FF,stroke-width:2px,color:#000\n")
	sb.WriteString("    classDef transformNode fill:#FFE5C2,stroke:#F7931A,stroke-width:2px,color:#000\n")
	sb.WriteString("    classDef conditionalNode fill:#DFF7E3,stroke:#34A853,stroke-width:2px,color:#000\n")
	sb.WriteString("    classDef mergeNode fill:#FFD9E6,stroke:#EA4C89,stroke-width:2px,color:#000\n")
	return sb.String()
}

// applyNodeClasses applies CSS classes to nodes based on their type.
func (r *MermaidRenderer) applyNodeClasses(workflow *models.Workflow) string {
	var sb strings.Builder

	// Group nodes by type for cleaner output
	nodesByType := make(map[string][]string)
	for _, node := range workflow.Nodes {
		className := r.getNodeClassName(node.Type)
		if className != "" {
			nodesByType[className] = append(nodesByType[className], node.ID)
		}
	}

	// Apply classes
	for className, nodeIDs := range nodesByType {
		if len(nodeIDs) > 0 {
			sb.WriteString("    class ")
			for i, nodeID := range nodeIDs {
				if i > 0 {
					sb.WriteString(",")
				}
				sb.WriteString(nodeID)
			}
			sb.WriteString(" ")
			sb.WriteString(className)
			sb.WriteString("\n")
		}
	}

	return sb.String()
}

// getNodeClassName returns the CSS class name for a node type.
func (r *MermaidRenderer) getNodeClassName(nodeType string) string {
	switch nodeType {
	case "http":
		return "httpNode"
	case "llm":
		return "llmNode"
	case "transform":
		return "transformNode"
	case "conditional":
		return "conditionalNode"
	case "merge":
		return "mergeNode"
	default:
		return "" // No custom styling for unknown types
	}
}
