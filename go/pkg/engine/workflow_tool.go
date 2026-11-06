package engine

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"

	"github.com/smilemakc/mbflow/go/pkg/executor"
)

// maxWorkflowToolDepth limits how deeply workflows run as LLM tools may nest
// (an agent in a tool workflow can call tool workflows of its own).
const maxWorkflowToolDepth = 5

type workflowToolDepthKey struct{}

// ErrWorkflowToolDepth is returned when workflows run as tools nest too deeply.
var ErrWorkflowToolDepth = errors.New("sub-workflow tools nested too deeply")

// ExecuteWorkflow runs a workflow to completion for an LLM tool call (a
// "sub_workflow" function) and returns the output of its terminal node(s),
// like the items of a sub_workflow node. It must be called from a node
// execution: the workflow has to belong to the owner of the calling workflow,
// and it inherits the caller's resources (credentials) and variables.
func (de *DAGExecutor) ExecuteWorkflow(ctx context.Context, workflowID string, input map[string]any) (any, error) {
	caller, ok := executor.GetExecutionContext(ctx)
	if !ok {
		return nil, errors.New("sub-workflow tool called outside of a workflow execution")
	}
	if de.workflowLoader == nil {
		return nil, errors.New("workflow loader not configured")
	}

	depth, _ := ctx.Value(workflowToolDepthKey{}).(int)
	if depth >= maxWorkflowToolDepth {
		return nil, fmt.Errorf("%w (limit %d)", ErrWorkflowToolDepth, maxWorkflowToolDepth)
	}
	ctx = context.WithValue(ctx, workflowToolDepthKey{}, depth+1)

	wf, err := de.workflowLoader.LoadWorkflow(ctx, workflowID)
	if err != nil {
		return nil, fmt.Errorf("failed to load workflow %s: %w", workflowID, err)
	}
	// Same rule as the sub_workflow node: otherwise a user could run (and read
	// the output of) someone else's workflow by ID.
	if wf.CreatedBy != caller.OwnerID {
		return nil, fmt.Errorf("failed to load workflow %s: %w", workflowID, ErrSubWorkflowNotAccessible)
	}

	cloned, err := wf.Clone()
	if err != nil {
		return nil, fmt.Errorf("failed to clone workflow %s: %w", workflowID, err)
	}

	state := NewExecutionState(uuid.New().String(), cloned.ID, cloned, input, caller.ExecutionVariables)
	state.ParentExecutionID = caller.ExecutionID
	state.ParentNodeID = caller.NodeID
	if caller.Resources != nil {
		state.Resources = caller.Resources
	}

	if err := de.Execute(ctx, state, DefaultExecutionOptions()); err != nil {
		return nil, err
	}
	return collectChildOutput(state), nil
}
