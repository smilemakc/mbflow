package engine

import (
	"context"
	"fmt"
	"reflect"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/smilemakc/mbflow/pkg/models"
)

const (
	NodeTypeSubWorkflow       = "sub_workflow"
	SubWorkflowDefaultItemVar = "item"
	SubWorkflowDefaultOnError = "fail_fast"
	SubWorkflowOnErrorCollect = "collect_partial"
)

// subWorkflowConfig holds parsed configuration for a sub_workflow node.
type subWorkflowConfig struct {
	WorkflowID     string
	ForEach        string
	ItemVar        string
	MaxParallelism int
	OnError        string
	TimeoutPerItem time.Duration
}

// subWorkflowItemResult holds the result of a single child execution.
type subWorkflowItemResult struct {
	Index       int    `json:"index"`
	Status      string `json:"status"`
	ExecutionID string `json:"execution_id"`
	Output      any    `json:"output,omitempty"`
	Error       string `json:"error,omitempty"`
	DurationMs  int64  `json:"duration_ms,omitempty"`
}

// executeSubWorkflow handles fan-out execution of sub-workflow nodes.
func (de *DAGExecutor) executeSubWorkflow(
	ctx context.Context,
	execState *ExecutionState,
	node *models.Node,
	opts *ExecutionOptions,
) error {
	cfg, err := parseSubWorkflowConfig(node)
	if err != nil {
		return fmt.Errorf("invalid sub_workflow config: %w", err)
	}

	// 1. Evaluate for_each expression to get items array
	parentNodes := GetRegularParentNodes(execState.Workflow, node)
	nodeCtx := PrepareNodeContext(execState, node, parentNodes, opts)
	items, err := evaluateForEach(cfg.ForEach, nodeCtx.DirectParentOutput)
	if err != nil {
		return fmt.Errorf("for_each evaluation failed: %w", err)
	}

	// 2. Load child workflow
	childWF, err := de.workflowLoader.LoadWorkflow(ctx, cfg.WorkflowID)
	if err != nil {
		return fmt.Errorf("failed to load child workflow %s: %w", cfg.WorkflowID, err)
	}

	// 3. Handle empty array
	if len(items) == 0 {
		output := map[string]any{
			"items":   []any{},
			"summary": map[string]any{"total": 0, "completed": 0, "failed": 0},
		}
		execState.SetNodeOutput(node.ID, output)
		execState.SetNodeStatus(node.ID, models.NodeExecutionStatusCompleted)
		return nil
	}

	// 4. Execute children in parallel
	results := make([]subWorkflowItemResult, len(items))
	var completed, failed int64

	maxPar := cfg.MaxParallelism
	if maxPar <= 0 {
		maxPar = len(items)
	}
	semaphore := make(chan struct{}, maxPar)

	var wg sync.WaitGroup
	var firstErr error
	var errOnce sync.Once
	cancelCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	for i, item := range items {
		wg.Add(1)
		go func(idx int, itm any) {
			defer wg.Done()

			select {
			case <-cancelCtx.Done():
				results[idx] = subWorkflowItemResult{
					Index:  idx,
					Status: "cancelled",
				}
				return
			case semaphore <- struct{}{}:
				defer func() { <-semaphore }()
			}

			result := de.executeSubWorkflowItem(cancelCtx, execState, node, childWF, cfg, idx, len(items), itm, opts)
			results[idx] = result

			if result.Status == "completed" {
				newCompleted := atomic.AddInt64(&completed, 1)
				de.safeNotify(ctx, ExecutionEvent{
					Type:                  EventTypeSubWorkflowItemCompleted,
					ExecutionID:           execState.ExecutionID,
					WorkflowID:            execState.WorkflowID,
					NodeID:                node.ID,
					Timestamp:             time.Now(),
					SubWorkflowTotal:      len(items),
					SubWorkflowCompleted:  int(newCompleted),
					SubWorkflowFailed:     int(atomic.LoadInt64(&failed)),
					SubWorkflowItemIndex:  idx,
					SubWorkflowItemExecID: result.ExecutionID,
					DurationMs:            result.DurationMs,
				})
			} else if result.Status == "failed" {
				newFailed := atomic.AddInt64(&failed, 1)
				de.safeNotify(ctx, ExecutionEvent{
					Type:                  EventTypeSubWorkflowItemFailed,
					ExecutionID:           execState.ExecutionID,
					WorkflowID:            execState.WorkflowID,
					NodeID:                node.ID,
					Timestamp:             time.Now(),
					Status:                "failed",
					SubWorkflowTotal:      len(items),
					SubWorkflowCompleted:  int(atomic.LoadInt64(&completed)),
					SubWorkflowFailed:     int(newFailed),
					SubWorkflowItemIndex:  idx,
					SubWorkflowItemExecID: result.ExecutionID,
					Message:               result.Error,
				})

				if cfg.OnError == SubWorkflowDefaultOnError {
					errOnce.Do(func() {
						firstErr = fmt.Errorf("child %d failed: %s", idx, result.Error)
						cancel()
					})
				}
			}

			// Progress event
			de.safeNotify(ctx, ExecutionEvent{
				Type:                 EventTypeSubWorkflowProgress,
				ExecutionID:          execState.ExecutionID,
				WorkflowID:           execState.WorkflowID,
				NodeID:               node.ID,
				Timestamp:            time.Now(),
				SubWorkflowTotal:     len(items),
				SubWorkflowCompleted: int(atomic.LoadInt64(&completed)),
				SubWorkflowFailed:    int(atomic.LoadInt64(&failed)),
			})
		}(i, item)
	}

	wg.Wait()

	// 5. Build output
	itemOutputs := make([]any, len(results))
	for i, r := range results {
		itemOutputs[i] = map[string]any{
			"index":        r.Index,
			"status":       r.Status,
			"execution_id": r.ExecutionID,
			"output":       r.Output,
			"error":        r.Error,
			"duration_ms":  r.DurationMs,
		}
	}

	finalCompleted := int(atomic.LoadInt64(&completed))
	finalFailed := int(atomic.LoadInt64(&failed))

	output := map[string]any{
		"items": itemOutputs,
		"summary": map[string]any{
			"total":     len(items),
			"completed": finalCompleted,
			"failed":    finalFailed,
		},
	}

	execState.SetNodeOutput(node.ID, output)
	execState.SetNodeInput(node.ID, nodeCtx.DirectParentOutput)
	execState.SetNodeConfig(node.ID, node.Config)

	if cfg.OnError == SubWorkflowDefaultOnError && firstErr != nil {
		execState.SetNodeStatus(node.ID, models.NodeExecutionStatusFailed)
		execState.SetNodeError(node.ID, firstErr)
		return firstErr
	}

	execState.SetNodeStatus(node.ID, models.NodeExecutionStatusCompleted)
	return nil
}

// executeSubWorkflowItem executes a single child workflow for one array item.
func (de *DAGExecutor) executeSubWorkflowItem(
	ctx context.Context,
	parentState *ExecutionState,
	parentNode *models.Node,
	childWF *models.Workflow,
	cfg *subWorkflowConfig,
	index int,
	totalItems int,
	item any,
	opts *ExecutionOptions,
) subWorkflowItemResult {
	startTime := time.Now()
	childExecID := uuid.New().String()

	result := subWorkflowItemResult{
		Index:       index,
		ExecutionID: childExecID,
	}

	// Clone child workflow
	clonedWF, err := childWF.Clone()
	if err != nil {
		result.Status = "failed"
		result.Error = fmt.Sprintf("failed to clone workflow: %v", err)
		return result
	}

	// Build child input
	childInput := map[string]any{
		cfg.ItemVar: item,
		"index":     index,
		"total":     totalItems,
	}
	// Inherit parent execution input as context
	for k, v := range parentState.Input {
		if _, exists := childInput[k]; !exists {
			childInput[k] = v
		}
	}

	// Create child execution state
	childState := NewExecutionState(childExecID, clonedWF.ID, clonedWF, childInput, parentState.Variables)
	childState.ParentExecutionID = parentState.ExecutionID
	childState.ParentNodeID = parentNode.ID
	idx := index
	childState.ItemIndex = &idx
	childState.Resources = parentState.Resources

	// Apply per-item timeout
	execCtx := ctx
	if cfg.TimeoutPerItem > 0 {
		var cancel context.CancelFunc
		execCtx, cancel = context.WithTimeout(ctx, cfg.TimeoutPerItem)
		defer cancel()
	}

	// Execute child workflow
	err = de.Execute(execCtx, childState, opts)

	result.DurationMs = time.Since(startTime).Milliseconds()

	if err != nil {
		result.Status = "failed"
		result.Error = err.Error()
		return result
	}

	// Collect output from last node(s) of child workflow
	result.Status = "completed"
	result.Output = collectChildOutput(childState)
	return result
}

// collectChildOutput gathers output from completed nodes of a child execution.
func collectChildOutput(state *ExecutionState) any {
	// Find terminal nodes (nodes with no outgoing edges)
	outgoing := make(map[string]bool)
	for _, edge := range state.Workflow.Edges {
		if !edge.IsLoop() {
			outgoing[edge.From] = true
		}
	}

	outputs := make(map[string]any)
	for _, node := range state.Workflow.Nodes {
		if !outgoing[node.ID] {
			if output, ok := state.GetNodeOutput(node.ID); ok {
				outputs[node.ID] = output
			}
		}
	}

	// If single terminal node, unwrap
	if len(outputs) == 1 {
		for _, v := range outputs {
			return v
		}
	}
	return outputs
}

// parseSubWorkflowConfig extracts and validates sub_workflow config from node.
func parseSubWorkflowConfig(node *models.Node) (*subWorkflowConfig, error) {
	cfg := &subWorkflowConfig{
		ItemVar: SubWorkflowDefaultItemVar,
		OnError: SubWorkflowDefaultOnError,
	}

	wfID, ok := node.Config["workflow_id"].(string)
	if !ok || wfID == "" {
		return nil, fmt.Errorf("workflow_id is required")
	}
	cfg.WorkflowID = wfID

	forEach, ok := node.Config["for_each"].(string)
	if !ok || forEach == "" {
		return nil, fmt.Errorf("for_each is required")
	}
	cfg.ForEach = forEach

	if iv, ok := node.Config["item_var"].(string); ok && iv != "" {
		cfg.ItemVar = iv
	}

	if mp, ok := node.Config["max_parallelism"]; ok {
		switch v := mp.(type) {
		case float64:
			cfg.MaxParallelism = int(v)
		case int:
			cfg.MaxParallelism = v
		}
	}

	if oe, ok := node.Config["on_error"].(string); ok {
		cfg.OnError = oe
	}

	if tp, ok := node.Config["timeout_per_item"]; ok {
		switch v := tp.(type) {
		case float64:
			cfg.TimeoutPerItem = time.Duration(int(v)) * time.Millisecond
		case int:
			cfg.TimeoutPerItem = time.Duration(v) * time.Millisecond
		}
	}

	return cfg, nil
}

// evaluateForEach evaluates the for_each expression and returns items as a slice.
func evaluateForEach(expression string, input map[string]any) ([]any, error) {
	// Navigate dot-separated path: "input.cells" -> input["cells"]
	parts := splitDotPath(expression)

	var current any = input
	for _, part := range parts {
		if part == "input" {
			continue // "input" prefix refers to the input map itself
		}
		m, ok := current.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("cannot navigate path %q: not a map at %q", expression, part)
		}
		current, ok = m[part]
		if !ok {
			return nil, fmt.Errorf("path %q: key %q not found", expression, part)
		}
	}

	// Convert to []any
	return toSlice(current)
}

// splitDotPath splits "input.cells" into ["input", "cells"].
func splitDotPath(path string) []string {
	var parts []string
	current := ""
	for _, c := range path {
		if c == '.' {
			if current != "" {
				parts = append(parts, current)
			}
			current = ""
		} else {
			current += string(c)
		}
	}
	if current != "" {
		parts = append(parts, current)
	}
	return parts
}

// toSlice converts various array types to []any.
func toSlice(val any) ([]any, error) {
	if val == nil {
		return nil, fmt.Errorf("for_each value is nil")
	}

	// Direct []any
	if s, ok := val.([]any); ok {
		return s, nil
	}

	// Use reflection for typed slices
	rv := reflect.ValueOf(val)
	if rv.Kind() == reflect.Slice || rv.Kind() == reflect.Array {
		result := make([]any, rv.Len())
		for i := 0; i < rv.Len(); i++ {
			result[i] = rv.Index(i).Interface()
		}
		return result, nil
	}

	return nil, fmt.Errorf("for_each must return an array, got: %T", val)
}
