// Package visualization provides workflow diagram rendering in various formats.
//
// The package supports rendering MBFlow workflows as:
//   - Mermaid flowchart diagrams (for documentation and GitHub)
//   - ASCII tree graphs (for console output)
//
// Example usage:
//
//	renderer := visualization.NewMermaidRenderer()
//	opts := visualization.DefaultRenderOptions()
//	diagram, err := renderer.Render(workflow, opts)
package visualization

import (
	"github.com/smilemakc/mbflow/pkg/models"
)

// Renderer is the interface for rendering workflows in different formats.
type Renderer interface {
	// Render converts a workflow into the target format.
	Render(workflow *models.Workflow, opts *RenderOptions) (string, error)

	// Format returns the format identifier (e.g., "mermaid", "ascii").
	Format() string
}

// RenderOptions configures how workflows are rendered.
type RenderOptions struct {
	// ShowConfig controls whether node configuration details are displayed.
	ShowConfig bool

	// ShowConditions controls whether edge conditions are displayed.
	ShowConditions bool

	// ShowDescription controls whether node descriptions are displayed.
	ShowDescription bool

	// UseColor enables ANSI color codes (ASCII renderer only).
	UseColor bool

	// CompactMode reduces the output size (ASCII renderer only).
	CompactMode bool

	// Direction sets the diagram flow direction (Mermaid renderer only).
	// Valid values: "TB" (top-bottom), "LR" (left-right), "RL" (right-left), "BT" (bottom-top).
	Direction string

	// ThemeVariables allows customizing Mermaid theme (Mermaid renderer only).
	ThemeVariables map[string]string
}

// DefaultRenderOptions returns the default rendering options.
func DefaultRenderOptions() *RenderOptions {
	return &RenderOptions{
		ShowConfig:      true,
		ShowConditions:  true,
		ShowDescription: false,
		UseColor:        true, // Will be auto-detected based on TTY
		CompactMode:     false,
		Direction:       "TB", // Top to Bottom
		ThemeVariables:  nil,
	}
}
