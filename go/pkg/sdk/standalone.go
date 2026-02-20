package sdk

import (
	"context"
	"fmt"

	"github.com/smilemakc/mbflow/go/pkg/engine"
	"github.com/smilemakc/mbflow/go/pkg/models"
)

// ExecuteWorkflowStandalone executes a workflow in standalone mode without persistence.
// This is useful for:
//   - Examples and demos
//   - Testing workflows before deploying to production
//   - Simple automation scripts that don't need execution history
//   - Embedded scenarios where you want to execute workflows in-memory
//
// The workflow is executed synchronously and returns the final result.
// No data is persisted to any database - everything runs in-memory.
func (c *Client) ExecuteWorkflowStandalone(
	ctx context.Context,
	workflow *models.Workflow,
	input map[string]any,
	opts *engine.ExecutionOptions,
) (*models.Execution, error) {
	if err := c.checkClosed(); err != nil {
		return nil, err
	}

	if workflow == nil {
		return nil, fmt.Errorf("workflow is required")
	}

	// Only available in embedded mode
	if c.config.Mode != ModeEmbedded {
		return nil, fmt.Errorf("standalone execution only available in embedded mode")
	}

	if c.standaloneExecutor == nil {
		return nil, fmt.Errorf("standalone executor not initialized")
	}

	// Use the standalone executor from pkg/engine
	return c.standaloneExecutor.ExecuteStandalone(ctx, workflow, input, opts)
}
