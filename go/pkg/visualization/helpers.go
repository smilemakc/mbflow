package visualization

import (
	"fmt"
	"os"

	"github.com/smilemakc/mbflow/go/pkg/models"
)

// RenderWorkflow is a convenience function to render a workflow in the specified format.
// Supported formats: "mermaid", "ascii".
// If opts is nil, default options will be used.
func RenderWorkflow(workflow *models.Workflow, format string, opts *RenderOptions) (string, error) {
	if opts == nil {
		opts = DefaultRenderOptions()
	}

	var renderer Renderer
	switch format {
	case "mermaid":
		renderer = NewMermaidRenderer()
	case "ascii":
		renderer = NewASCIIRenderer()
	default:
		return "", fmt.Errorf("unsupported format: %s (supported: mermaid, ascii)", format)
	}

	return renderer.Render(workflow, opts)
}

// PrintWorkflow prints a workflow diagram to stdout in the specified format.
// Supported formats: "mermaid", "ascii".
func PrintWorkflow(workflow *models.Workflow, format string, opts *RenderOptions) error {
	diagram, err := RenderWorkflow(workflow, format, opts)
	if err != nil {
		return err
	}

	fmt.Fprintln(os.Stdout, diagram)
	return nil
}

// SaveWorkflowToFile saves a workflow diagram to a file.
func SaveWorkflowToFile(workflow *models.Workflow, format string, filename string, opts *RenderOptions) error {
	diagram, err := RenderWorkflow(workflow, format, opts)
	if err != nil {
		return err
	}

	return os.WriteFile(filename, []byte(diagram), 0644)
}
