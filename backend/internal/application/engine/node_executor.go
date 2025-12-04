package engine

import (
	"context"
	"fmt"

	"github.com/smilemakc/mbflow/pkg/executor"
	"github.com/smilemakc/mbflow/pkg/models"
)

// NodeExecutor executes a single node with automatic template resolution
type NodeExecutor struct {
	executorManager executor.Manager
}

// NewNodeExecutor creates a new node executor
func NewNodeExecutor(manager executor.Manager) *NodeExecutor {
	return &NodeExecutor{
		executorManager: manager,
	}
}

// NodeExecutionResult contains the result of node execution along with metadata
type NodeExecutionResult struct {
	Output         interface{}            // Execution result
	Input          interface{}            // Input passed to executor
	Config         map[string]interface{} // Original config (before template resolution)
	ResolvedConfig map[string]interface{} // Resolved config (after template resolution)
}

// Execute executes a single node with automatic template resolution.
//
// This is the CRITICAL integration point where TemplateExecutorWrapper is applied.
//
// Flow:
//  1. Get base executor from registry
//  2. Build ExecutionContextData from node context
//  3. Create template engine from ExecutionContextData
//  4. Resolve templates in config to get ResolvedConfig
//  5. Execute with resolved config
//  6. Return NodeExecutionResult with metadata
func (ne *NodeExecutor) Execute(ctx context.Context, nodeCtx *NodeContext) (*NodeExecutionResult, error) {
	// 1. Get base executor from registry
	baseExecutor, err := ne.executorManager.Get(nodeCtx.Node.Type)
	if err != nil {
		return nil, fmt.Errorf("executor not found for type %s: %w", nodeCtx.Node.Type, err)
	}

	// 2. Build ExecutionContextData for template resolution
	execCtxData := &executor.ExecutionContextData{
		WorkflowVariables:  nodeCtx.WorkflowVariables,
		ExecutionVariables: nodeCtx.ExecutionVariables,
		ParentNodeOutput:   nodeCtx.DirectParentOutput, // ⭐ Key: output from immediate parent
		StrictMode:         nodeCtx.StrictMode,
	}

	// 3. Create template engine from execution context
	templateEngine := executor.NewTemplateEngine(execCtxData)

	// 4. Resolve templates in config ⭐ THIS IS WHERE WE GET RESOLVED CONFIG
	resolvedConfig, err := templateEngine.ResolveConfig(nodeCtx.Node.Config)
	if err != nil {
		return nil, fmt.Errorf("template resolution failed: %w", err)
	}

	// 5. Execute with resolved config
	//   - {{input.field}} → already resolved in resolvedConfig
	//   - {{env.var}} → already resolved in resolvedConfig
	output, err := baseExecutor.Execute(ctx, resolvedConfig, nodeCtx.DirectParentOutput)
	if err != nil {
		return nil, fmt.Errorf("node execution failed: %w", err)
	}

	// 6. Return result with metadata
	return &NodeExecutionResult{
		Output:         output,
		Input:          nodeCtx.DirectParentOutput,
		Config:         nodeCtx.Node.Config, // Original config
		ResolvedConfig: resolvedConfig,      // Resolved config
	}, nil
}

// PrepareNodeContext builds NodeContext from execution state and node.
//
// This function handles:
//   - Single parent: merges parent output with execution input (parent output takes precedence)
//   - Multiple parents: merges outputs by parent node ID (namespace collision avoidance)
//   - No parents: uses execution input
func PrepareNodeContext(
	execState *ExecutionState,
	node *models.Node,
	parentNodes []*models.Node,
	opts *ExecutionOptions,
) *NodeContext {
	// Get direct parent output (for nodes with single parent)
	var directParentOutput map[string]interface{}

	if len(parentNodes) == 1 {
		// Single parent - merge execution input with parent output
		// This allows child nodes to access both execution input and parent output
		directParentOutput = make(map[string]interface{})

		// First, copy execution input
		for k, v := range execState.Input {
			directParentOutput[k] = v
		}

		// Then, overlay parent output (takes precedence)
		parentID := parentNodes[0].ID
		if output, ok := execState.GetNodeOutput(parentID); ok {
			if outputMap, ok := output.(map[string]interface{}); ok {
				for k, v := range outputMap {
					directParentOutput[k] = v
				}
			}
		}
	} else if len(parentNodes) > 1 {
		// Multiple parents - merge outputs with namespace by parent ID
		directParentOutput = mergeParentOutputs(execState, parentNodes)
	} else {
		// No parents - use execution input
		directParentOutput = execState.Input
	}

	return &NodeContext{
		ExecutionID:        execState.ExecutionID,
		NodeID:             node.ID,
		Node:               node,
		WorkflowVariables:  execState.Workflow.Variables,
		ExecutionVariables: execState.Variables,
		DirectParentOutput: directParentOutput,
		StrictMode:         opts.StrictMode,
	}
}

// mergeParentOutputs merges outputs from multiple parent nodes.
//
// To avoid namespace collisions, outputs are namespaced by parent node ID:
//
//	{
//	  "parent1-id": {parent1 output},
//	  "parent2-id": {parent2 output}
//	}
//
// Access in templates:
//
//	{{input.parent1-id.field}}
//	{{input.parent2-id.data}}
func mergeParentOutputs(execState *ExecutionState, parentNodes []*models.Node) map[string]interface{} {
	merged := make(map[string]interface{})

	for _, parent := range parentNodes {
		if output, ok := execState.GetNodeOutput(parent.ID); ok {
			// Namespace outputs by parent node ID to avoid collisions
			merged[parent.ID] = output
		}
	}

	return merged
}
