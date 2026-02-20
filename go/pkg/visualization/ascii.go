package visualization

import (
	"fmt"
	"os"
	"strings"

	"github.com/smilemakc/mbflow/go/pkg/models"
	"golang.org/x/term"
)

// ASCIIRenderer renders workflows as ASCII tree graphs.
type ASCIIRenderer struct{}

// NewASCIIRenderer creates a new ASCII renderer.
func NewASCIIRenderer() *ASCIIRenderer {
	return &ASCIIRenderer{}
}

// Format returns the format identifier.
func (r *ASCIIRenderer) Format() string {
	return "ascii"
}

// ANSI color codes
const (
	colorReset  = "\033[0m"
	colorRed    = "\033[31m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
	colorBlue   = "\033[34m"
	colorCyan   = "\033[36m"
	colorWhite  = "\033[37m"
)

// Box drawing characters
const (
	branchChar     = "├── "
	lastBranchChar = "└── "
	verticalChar   = "│   "
	emptyChar      = "    "
)

// Render converts a workflow into ASCII tree format.
func (r *ASCIIRenderer) Render(workflow *models.Workflow, opts *RenderOptions) (string, error) {
	if workflow == nil {
		return "", fmt.Errorf("workflow is nil")
	}

	if opts == nil {
		opts = DefaultRenderOptions()
	}

	// Auto-detect TTY for color support
	if opts.UseColor {
		opts.UseColor = isTerminal()
	}

	var sb strings.Builder

	// Write workflow title
	title := workflow.Name
	if title == "" {
		title = workflow.ID
	}
	sb.WriteString(r.colorize(title, colorCyan, opts.UseColor))
	sb.WriteString("\n\n")

	// Build graph structure
	graph := r.buildGraph(workflow)

	// Find root nodes (nodes with no incoming edges)
	rootNodes := r.findRootNodes(workflow)

	if len(rootNodes) == 0 && len(workflow.Nodes) > 0 {
		// If no root nodes found (cycle or all connected), use the first node
		rootNodes = []*models.Node{workflow.Nodes[0]}
	}

	// Render each root node and its children
	visited := make(map[string]bool)
	for i, root := range rootNodes {
		isLast := i == len(rootNodes)-1
		r.renderNode(&sb, root, graph, "", isLast, visited, opts)
	}

	return sb.String(), nil
}

// graphNode represents a node in the adjacency list.
type graphNode struct {
	Node     *models.Node
	Children []*graphEdge
}

// graphEdge represents an edge with optional condition.
type graphEdge struct {
	Target    *models.Node
	Condition string
}

// buildGraph creates an adjacency list representation of the workflow.
func (r *ASCIIRenderer) buildGraph(workflow *models.Workflow) map[string]*graphNode {
	graph := make(map[string]*graphNode)

	// Initialize graph nodes
	for _, node := range workflow.Nodes {
		graph[node.ID] = &graphNode{
			Node:     node,
			Children: []*graphEdge{},
		}
	}

	// Add edges
	for _, edge := range workflow.Edges {
		if parent, ok := graph[edge.From]; ok {
			if child, ok := graph[edge.To]; ok {
				parent.Children = append(parent.Children, &graphEdge{
					Target:    child.Node,
					Condition: edge.Condition,
				})
			}
		}
	}

	return graph
}

// findRootNodes finds nodes with no incoming edges.
func (r *ASCIIRenderer) findRootNodes(workflow *models.Workflow) []*models.Node {
	hasIncoming := make(map[string]bool)

	// Mark all nodes that have incoming edges
	for _, edge := range workflow.Edges {
		hasIncoming[edge.To] = true
	}

	// Find nodes without incoming edges
	var roots []*models.Node
	for _, node := range workflow.Nodes {
		if !hasIncoming[node.ID] {
			roots = append(roots, node)
		}
	}

	return roots
}

// renderNode recursively renders a node and its children.
func (r *ASCIIRenderer) renderNode(
	sb *strings.Builder,
	node *models.Node,
	graph map[string]*graphNode,
	prefix string,
	isLast bool,
	visited map[string]bool,
	opts *RenderOptions,
) {
	// Check for cycles
	if visited[node.ID] {
		if prefix != "" {
			if isLast {
				sb.WriteString(prefix + lastBranchChar)
			} else {
				sb.WriteString(prefix + branchChar)
			}
		}
		sb.WriteString(r.colorize("(cycle detected: "+node.ID+")", colorRed, opts.UseColor))
		sb.WriteString("\n")
		return
	}

	visited[node.ID] = true

	// Write node with proper prefix
	if prefix != "" {
		if isLast {
			sb.WriteString(prefix + lastBranchChar)
		} else {
			sb.WriteString(prefix + branchChar)
		}
	}

	sb.WriteString(r.formatNode(node, opts))
	sb.WriteString("\n")

	// Write config details if not in compact mode
	if !opts.CompactMode && opts.ShowConfig {
		configStr := r.extractNodeConfig(node)
		if configStr != "" {
			configPrefix := prefix
			if prefix != "" {
				if isLast {
					configPrefix += emptyChar
				} else {
					configPrefix += verticalChar
				}
			}
			sb.WriteString(configPrefix)
			sb.WriteString(r.colorize("│ "+configStr, colorWhite, opts.UseColor))
			sb.WriteString("\n")
		}
	}

	// Get children
	gNode, ok := graph[node.ID]
	if !ok || len(gNode.Children) == 0 {
		return
	}

	// Prepare prefix for children
	childPrefix := prefix
	if isLast {
		childPrefix += emptyChar
	} else {
		childPrefix += verticalChar
	}

	// Render children
	for i, edge := range gNode.Children {
		isLastChild := i == len(gNode.Children)-1
		r.renderNode(sb, edge.Target, graph, childPrefix, isLastChild, visited, opts)
	}
}

// formatNode formats a node for display.
func (r *ASCIIRenderer) formatNode(node *models.Node, opts *RenderOptions) string {
	if opts.CompactMode {
		// Compact: id (type)
		return fmt.Sprintf("%s %s",
			r.colorize(node.ID, colorGreen, opts.UseColor),
			r.colorize("("+node.Type+")", colorYellow, opts.UseColor))
	}

	// Detailed: [id] Name (type)
	var parts []string

	parts = append(parts, r.colorize("["+node.ID+"]", colorGreen, opts.UseColor))

	if node.Name != "" {
		parts = append(parts, node.Name)
	}

	parts = append(parts, r.colorize("("+node.Type+")", colorYellow, opts.UseColor))

	return strings.Join(parts, " ")
}

// extractNodeConfig extracts key configuration for display.
func (r *ASCIIRenderer) extractNodeConfig(node *models.Node) string {
	switch node.Type {
	case "http":
		method, _ := node.Config["method"].(string)
		url, _ := node.Config["url"].(string)
		if method != "" && url != "" {
			return method + " " + url
		}
		return url
	case "llm":
		provider, _ := node.Config["provider"].(string)
		model, _ := node.Config["model"].(string)
		if provider != "" && model != "" {
			return provider + " / " + model
		}
		return model
	case "transform":
		transformType, _ := node.Config["type"].(string)
		return transformType
	}
	return ""
}

// colorize applies ANSI color codes to text.
func (r *ASCIIRenderer) colorize(text, color string, enabled bool) string {
	if !enabled {
		return text
	}
	return color + text + colorReset
}

// isTerminal checks if stdout is a terminal (for auto-detecting color support).
func isTerminal() bool {
	return term.IsTerminal(int(os.Stdout.Fd()))
}
