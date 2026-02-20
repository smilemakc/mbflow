package engine

import (
	"context"
	"fmt"

	"github.com/smilemakc/mbflow/go/pkg/executor"
	"github.com/smilemakc/mbflow/go/pkg/models"
)

// NodeExecutor executes a single node with automatic template resolution.
type NodeExecutor struct {
	executorManager executor.Manager
}

// NewNodeExecutor creates a new node executor.
func NewNodeExecutor(manager executor.Manager) *NodeExecutor {
	return &NodeExecutor{
		executorManager: manager,
	}
}

// NodeExecutionResult contains the result of node execution along with metadata.
type NodeExecutionResult struct {
	Output         any
	Input          any
	Config         map[string]any
	ResolvedConfig map[string]any
}

// NodeContext holds context for single node execution.
type NodeContext struct {
	ExecutionID        string
	NodeID             string
	Node               *models.Node
	WorkflowVariables  map[string]any
	ExecutionVariables map[string]any
	DirectParentOutput map[string]any
	Resources          map[string]any
	StrictMode         bool
}

// Execute executes a single node with automatic template resolution.
//
// Flow:
//  1. Get base executor from registry
//  2. Build ExecutionContextData from node context
//  3. Create template engine from ExecutionContextData
//  4. Resolve templates in config to get ResolvedConfig
//  5. Execute with resolved config
//  6. Return NodeExecutionResult with metadata
func (ne *NodeExecutor) Execute(ctx context.Context, nodeCtx *NodeContext) (*NodeExecutionResult, error) {
	baseExecutor, err := ne.executorManager.Get(nodeCtx.Node.Type)
	if err != nil {
		return nil, fmt.Errorf("executor not found for type %s: %w", nodeCtx.Node.Type, err)
	}

	execCtxData := &executor.ExecutionContextData{
		WorkflowVariables:  nodeCtx.WorkflowVariables,
		ExecutionVariables: nodeCtx.ExecutionVariables,
		ParentNodeOutput:   nodeCtx.DirectParentOutput,
		Resources:          nodeCtx.Resources,
		StrictMode:         nodeCtx.StrictMode,
	}

	templateEngine := executor.NewTemplateEngine(execCtxData)

	resolvedConfig, err := templateEngine.ResolveConfig(nodeCtx.Node.Config)
	if err != nil {
		return nil, fmt.Errorf("template resolution failed: %w", err)
	}

	output, err := baseExecutor.Execute(ctx, resolvedConfig, nodeCtx.DirectParentOutput)

	result := &NodeExecutionResult{
		Output:         output,
		Input:          nodeCtx.DirectParentOutput,
		Config:         nodeCtx.Node.Config,
		ResolvedConfig: resolvedConfig,
	}

	if err != nil {
		return result, fmt.Errorf("node execution failed: %w", err)
	}

	return result, nil
}

// PrepareNodeContext builds NodeContext from execution state and node.
//
// Input merging strategy:
//   - No parents: uses execution input
//   - Single parent: merges execution input with parent output (parent output takes precedence)
//   - Multiple parents: merges outputs namespaced by parent node ID
func PrepareNodeContext(
	execState *ExecutionState,
	node *models.Node,
	parentNodes []*models.Node,
	opts *ExecutionOptions,
) *NodeContext {
	var directParentOutput map[string]any

	if loopInput, ok := execState.GetLoopInput(node.ID); ok {
		directParentOutput = make(map[string]any)
		for k, v := range execState.Input {
			directParentOutput[k] = v
		}
		if loopMap, ok := loopInput.(map[string]any); ok {
			for k, v := range loopMap {
				directParentOutput[k] = v
			}
		}
		execState.ClearLoopInput(node.ID)
	} else if len(parentNodes) == 1 {
		directParentOutput = make(map[string]any)

		for k, v := range execState.Input {
			directParentOutput[k] = v
		}

		parentID := parentNodes[0].ID
		if output, ok := execState.GetNodeOutput(parentID); ok {
			if outputMap, ok := output.(map[string]any); ok {
				for k, v := range outputMap {
					directParentOutput[k] = v
				}
			}
		}
	} else if len(parentNodes) > 1 {
		directParentOutput = mergeParentOutputs(execState, parentNodes)
	} else {
		directParentOutput = execState.Input
	}

	return &NodeContext{
		ExecutionID:        execState.ExecutionID,
		NodeID:             node.ID,
		Node:               node,
		WorkflowVariables:  execState.Workflow.Variables,
		ExecutionVariables: execState.Variables,
		DirectParentOutput: directParentOutput,
		Resources:          execState.Resources,
		StrictMode:         opts.StrictMode,
	}
}

// mergeParentOutputs merges outputs from multiple parent nodes.
// Outputs are namespaced by parent node ID to avoid collisions.
func mergeParentOutputs(execState *ExecutionState, parentNodes []*models.Node) map[string]any {
	merged := make(map[string]any)

	for _, parent := range parentNodes {
		if output, ok := execState.GetNodeOutput(parent.ID); ok {
			merged[parent.ID] = output
		}
	}

	return merged
}
