package builder

import (
	"fmt"

	"github.com/smilemakc/mbflow/go/pkg/models"
)

// RelativePosition positions a node relative to another node.
// The workflow builder must have the reference node already added.
// Note: This creates a closure that will be evaluated during Build().
func RelativePosition(refNodeID string, offsetX, offsetY float64) NodeOption {
	return func(nb *NodeBuilder) error {
		// We can't resolve the reference node position here since it may not be built yet.
		// Store it in metadata and resolve during workflow build if needed.
		// For now, we'll just store a marker and calculate position later.

		// Simple implementation: just store the offset in metadata
		// A more sophisticated implementation would resolve this during workflow Build()
		if refNodeID == "" {
			return fmt.Errorf("reference node ID cannot be empty")
		}

		// Store reference info in metadata for potential future use
		if nb.metadata == nil {
			nb.metadata = make(map[string]any)
		}
		nb.metadata["_position_ref"] = map[string]any{
			"ref_node": refNodeID,
			"offset_x": offsetX,
			"offset_y": offsetY,
		}

		// For now, set a default position
		// In a full implementation, this would be resolved during workflow.Build()
		nb.position = &models.Position{
			X: offsetX,
			Y: offsetY,
		}

		return nil
	}
}

// AutoLayoutPosition returns a position option that will be automatically
// calculated based on the node's order in the workflow.
// This is handled by the WorkflowBuilder when WithAutoLayout() is enabled.
func AutoLayoutPosition() NodeOption {
	return func(nb *NodeBuilder) error {
		// Don't set position - let WorkflowBuilder handle it
		return nil
	}
}
