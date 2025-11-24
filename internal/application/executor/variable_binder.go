package executor

import (
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/smilemakc/mbflow/internal/domain"
)

// NodeExecutionInputs encapsulates all input data for a node execution
type NodeExecutionInputs struct {
	// Variables contains scoped inputs from parent nodes (already bound/namespaced)
	Variables *domain.VariableSet

	// GlobalContext contains global context variables (read-only for nodes)
	GlobalContext *domain.VariableSet

	// ParentOutputs contains raw outputs from each parent node (for advanced use cases)
	ParentOutputs map[uuid.UUID]*domain.VariableSet

	// ExecutionID is the ID of the current execution
	ExecutionID uuid.UUID

	// WorkflowID is the ID of the workflow being executed
	WorkflowID uuid.UUID
}

// VariableBinder handles binding parent outputs to node inputs
type VariableBinder struct {
	evaluator *ConditionEvaluator
}

// NewVariableBinder creates a new VariableBinder
func NewVariableBinder(evaluator *ConditionEvaluator) *VariableBinder {
	return &VariableBinder{
		evaluator: evaluator,
	}
}

// BindInputs creates NodeExecutionInputs by binding parent outputs to node inputs
func (vb *VariableBinder) BindInputs(
	node domain.Node,
	graph *WorkflowGraph,
	execution domain.Execution,
) (*NodeExecutionInputs, error) {

	// Get parent nodes from graph
	parentNodeIDs := graph.GetPredecessors(node.ID())

	// Collect parent outputs
	parentOutputs := make(map[uuid.UUID]*domain.VariableSet)
	parentNodes := make(map[uuid.UUID]domain.Node)

	for _, parentID := range parentNodeIDs {
		if output, ok := execution.GetNodeOutput(parentID); ok {
			parentOutputs[parentID] = output
			if parentNode, err := graph.GetNode(parentID); err == nil {
				parentNodes[parentID] = parentNode
			}
		}
	}

	// Start with empty variable set
	scopedVars := domain.NewVariableSet(nil)

	// Apply binding based on configuration
	bindingConfig := node.InputBindingConfig()

	if bindingConfig.AutoBind {
		// Auto-bind: merge parent outputs with collision handling
		if err := vb.autoBindParents(scopedVars, parentOutputs, parentNodes, bindingConfig.CollisionStrategy); err != nil {
			return nil, err
		}
	}

	// Apply explicit mappings (overrides auto-bind)
	if len(bindingConfig.Mappings) > 0 {
		if err := vb.applyExplicitMappings(scopedVars, parentOutputs, parentNodes, bindingConfig.Mappings); err != nil {
			return nil, err
		}
	}

	// Get additional sources from edges (edge-based variable passing)
	additionalSources, err := vb.getAdditionalSources(node, graph, parentNodeIDs)
	if err != nil {
		return nil, fmt.Errorf("failed to get additional sources: %w", err)
	}

	// Bind additional sources (always namespaced)
	if len(additionalSources) > 0 {
		if err := vb.bindAdditionalSources(scopedVars, additionalSources, execution); err != nil {
			return nil, fmt.Errorf("failed to bind additional sources: %w", err)
		}
	}

	// Validate against input schema if exists
	if schema := node.IOSchema(); schema != nil && schema.Inputs != nil {
		if err := schema.Inputs.Validate(scopedVars.All()); err != nil {
			return nil, fmt.Errorf("input validation failed for node %s: %w", node.Name(), err)
		}
	}

	return &NodeExecutionInputs{
		Variables:     scopedVars,
		GlobalContext: execution.GlobalVariables(),
		ParentOutputs: parentOutputs,
		ExecutionID:   execution.ID(),
		WorkflowID:    execution.WorkflowID(),
	}, nil
}

// autoBindParents merges parent outputs with collision handling
func (vb *VariableBinder) autoBindParents(
	target *domain.VariableSet,
	parentOutputs map[uuid.UUID]*domain.VariableSet,
	parentNodes map[uuid.UUID]domain.Node,
	strategy domain.CollisionStrategy,
) error {

	if len(parentOutputs) == 0 {
		// No parents - node sees only global context
		return nil
	}

	// Always apply collision strategy (including for single parent)
	// This ensures consistent namespacing regardless of parent count
	return vb.mergeMultipleParents(target, parentOutputs, parentNodes, strategy)
}

// mergeMultipleParents handles merging outputs from multiple parents
func (vb *VariableBinder) mergeMultipleParents(
	target *domain.VariableSet,
	parentOutputs map[uuid.UUID]*domain.VariableSet,
	parentNodes map[uuid.UUID]domain.Node,
	strategy domain.CollisionStrategy,
) error {

	switch strategy {
	case domain.CollisionStrategyNamespaceByParent:
		// Store entire parent output under parent node name
		// This allows accessing parent data as: parent_name.field_name
		for parentID, output := range parentOutputs {
			parentNode := parentNodes[parentID]
			nodeName := parentNode.Name()

			// Store the entire output map under the parent node name
			if err := target.Set(nodeName, output.All()); err != nil {
				return err
			}
		}

	case domain.CollisionStrategyCollect:
		// Collect colliding keys into arrays
		collected := make(map[string][]any)

		for _, output := range parentOutputs {
			for key, value := range output.All() {
				collected[key] = append(collected[key], value)
			}
		}

		for key, values := range collected {
			if len(values) == 1 {
				_ = target.Set(key, values[0])
			} else {
				_ = target.Set(key, values)
			}
		}

	case domain.CollisionStrategyError:
		// Fail on any collision
		keys := make(map[string]int)
		for _, output := range parentOutputs {
			for key := range output.All() {
				keys[key]++
				if keys[key] > 1 {
					return domain.NewDomainError(
						domain.ErrCodeValidationFailed,
						fmt.Sprintf("collision detected on key '%s' from multiple parents", key),
						nil,
					)
				}
			}
		}

		// No collisions - merge directly
		for _, output := range parentOutputs {
			if err := target.Merge(output); err != nil {
				return err
			}
		}

	default:
		return domain.NewDomainError(
			domain.ErrCodeInvalidInput,
			fmt.Sprintf("unknown collision strategy: %s", strategy),
			nil,
		)
	}

	return nil
}

// applyExplicitMappings applies user-defined mappings
func (vb *VariableBinder) applyExplicitMappings(
	target *domain.VariableSet,
	parentOutputs map[uuid.UUID]*domain.VariableSet,
	parentNodes map[uuid.UUID]domain.Node,
	mappings map[string]string,
) error {

	for targetKey, sourcePath := range mappings {
		value, err := vb.resolveSourcePath(sourcePath, parentOutputs, parentNodes)
		if err != nil {
			return fmt.Errorf("failed to resolve mapping '%s' -> '%s': %w", targetKey, sourcePath, err)
		}
		if err := target.Set(targetKey, value); err != nil {
			return err
		}
	}

	return nil
}

// getAdditionalSources extracts additional data source nodes from incoming edges' config.
// Returns a map of node ID to node for nodes specified in include_outputs_from,
// excluding nodes that are already direct parents.
func (vb *VariableBinder) getAdditionalSources(
	node domain.Node,
	graph *WorkflowGraph,
	directParentIDs []uuid.UUID,
) (map[uuid.UUID]domain.Node, error) {
	additionalSources := make(map[uuid.UUID]domain.Node)

	// Create set of direct parent IDs for quick lookup
	directParents := make(map[uuid.UUID]bool)
	for _, parentID := range directParentIDs {
		directParents[parentID] = true
	}

	// Check all incoming edges
	incomingEdges := graph.GetIncomingEdges(node.ID())
	for _, edge := range incomingEdges {
		config := edge.Config()
		if config == nil {
			continue
		}

		// Extract include_outputs_from
		includeOutputsFrom, ok := config["include_outputs_from"]
		if !ok {
			continue
		}

		// Convert to string slice
		var nodeNames []string
		switch v := includeOutputsFrom.(type) {
		case []string:
			nodeNames = v
		case []interface{}:
			nodeNames = make([]string, len(v))
			for i, item := range v {
				str, ok := item.(string)
				if !ok {
					return nil, fmt.Errorf("include_outputs_from contains non-string value at index %d", i)
				}
				nodeNames[i] = str
			}
		default:
			return nil, fmt.Errorf("include_outputs_from must be a string array")
		}

		// Resolve each node name to node object
		for _, nodeName := range nodeNames {
			sourceNode, err := graph.GetNodeByName(nodeName)
			if err != nil {
				return nil, fmt.Errorf("node '%s' from include_outputs_from not found: %w", nodeName, err)
			}

			sourceNodeID := sourceNode.ID()

			// Skip if already a direct parent
			if directParents[sourceNodeID] {
				continue
			}

			// Add to additional sources
			additionalSources[sourceNodeID] = sourceNode
		}
	}

	return additionalSources, nil
}

// bindAdditionalSources binds outputs from additional source nodes to the target VariableSet.
// All variables are namespaced by the source node name (format: nodename_variablename).
func (vb *VariableBinder) bindAdditionalSources(
	target *domain.VariableSet,
	additionalSources map[uuid.UUID]domain.Node,
	execution domain.Execution,
) error {
	for nodeID, node := range additionalSources {
		// Get node output from execution
		output, ok := execution.GetNodeOutput(nodeID)
		if !ok {
			return fmt.Errorf("node '%s' referenced in include_outputs_from has not executed yet", node.Name())
		}

		// Store entire output under node name (consistent with parent binding)
		nodeName := node.Name()
		if err := target.Set(nodeName, output.All()); err != nil {
			return fmt.Errorf("failed to set variable '%s': %w", nodeName, err)
		}
	}

	return nil
}

// resolveSourcePath resolves a source path like "parent_node.field" or just "field"
func (vb *VariableBinder) resolveSourcePath(
	path string,
	parentOutputs map[uuid.UUID]*domain.VariableSet,
	parentNodes map[uuid.UUID]domain.Node,
) (any, error) {
	// Parse path: "node_name.field" or just "field"
	parts := strings.SplitN(path, ".", 2)

	if len(parts) == 2 {
		// Path includes node name: "node_name.field"
		nodeName := parts[0]
		fieldName := parts[1]

		// Find the parent node by name
		for parentID, parentNode := range parentNodes {
			if parentNode.Name() == nodeName {
				output := parentOutputs[parentID]
				if value, ok := output.Get(fieldName); ok {
					return value, nil
				}
				return nil, domain.NewDomainError(
					domain.ErrCodeNotFound,
					fmt.Sprintf("field '%s' not found in parent node '%s' output", fieldName, nodeName),
					nil,
				)
			}
		}

		return nil, domain.NewDomainError(
			domain.ErrCodeNotFound,
			fmt.Sprintf("parent node '%s' not found", nodeName),
			nil,
		)
	}

	// Simple field name - try to find it in any parent
	fieldName := parts[0]
	for parentID := range parentNodes {
		output := parentOutputs[parentID]
		if value, ok := output.Get(fieldName); ok {
			return value, nil
		}
	}

	return nil, domain.NewDomainError(
		domain.ErrCodeNotFound,
		fmt.Sprintf("field '%s' not found in any parent output", fieldName),
		nil,
	)
}
