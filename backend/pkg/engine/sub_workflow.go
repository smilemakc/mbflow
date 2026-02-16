package engine

import (
	"context"
	"fmt"

	"github.com/smilemakc/mbflow/pkg/models"
)

// executeSubWorkflow handles fan-out execution of sub-workflow nodes.
// It evaluates the for_each expression, spawns N child executions, and collects results.
func (de *DAGExecutor) executeSubWorkflow(
	ctx context.Context,
	execState *ExecutionState,
	node *models.Node,
	opts *ExecutionOptions,
) error {
	return fmt.Errorf("sub_workflow execution not yet implemented")
}
