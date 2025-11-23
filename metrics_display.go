package mbflow

import "fmt"

// ANSI colors & styles
const (
	colorReset  = "\033[0m"
	colorBlue   = "\033[34m"
	colorCyan   = "\033[36m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
	colorRed    = "\033[31m"
	bold        = "\033[1m"
)

// DisplayMetrics prints execution metrics in a formatted, human-readable way.
// This is a helper function designed for examples, demos, and debugging.
//
// Parameters:
//   - metrics: The ExecutorMetrics instance to display (from executor.GetMetrics())
//   - workflowID: The workflow ID to show metrics for (pass empty string to skip workflow metrics)
//   - nodeIDs: List of node IDs to show metrics for (e.g., []string{"node-1", "node-2"})
//   - showAIMetrics: Whether to display AI API usage metrics
//
// Example usage:
//
//	executor := mbflow.NewExecutor(&mbflow.ExecutorConfig{...})
//	// ... execute workflow ...
//	nodeIDs := []string{"start", "process", "end"}
//	mbflow.DisplayMetrics(executor.GetMetrics(), workflowID, nodeIDs, true)
func DisplayMetrics(metrics ExecutorMetrics, workflowID string, nodeIDs []string, showAIMetrics bool) {
	title := func(text string) {
		fmt.Printf("\n%s%s=== %s ===%s\n\n", bold, colorBlue, text, colorReset)
	}

	section := func(text string) {
		fmt.Printf("%s%s%s\n", bold, text, colorReset)
	}

	kv := func(label string, value any) {
		fmt.Printf("  %s%-22s%s: %v\n", colorCyan, label, colorReset, value)
	}

	title("Execution Metrics")

	// Summary
	summary := metrics.GetSummary()
	if summary != nil {
		section("Summary:")
		kv("Total Workflows", summary.TotalWorkflows)
		kv("Total Executions", summary.TotalExecutions)
		kv("Total Successes", fmt.Sprintf("%s%d%s", colorGreen, summary.TotalSuccesses, colorReset))
		kv("Total Failures", fmt.Sprintf("%s%d%s", colorRed, summary.TotalFailures, colorReset))
		kv("Success Rate", fmt.Sprintf("%.2f%%", summary.OverallSuccessRate*100))
		kv("Node Executions", summary.TotalNodeExecutions)
		kv("Node Retries", summary.TotalNodeRetries)
		kv("AI Requests", summary.TotalAIRequests)
		kv("AI Tokens", summary.TotalAITokens)
		kv("AI Cost (USD)", fmt.Sprintf("$%.4f", summary.EstimatedAICostUSD))
	}

	// Workflow metrics
	if workflowID != "" {
		workflowMetrics := metrics.GetWorkflowMetrics(workflowID)
		if workflowMetrics != nil {
			section("\nWorkflow Metrics:")
			kv("Workflow ID", workflowMetrics.WorkflowID)
			kv("Execution Count", workflowMetrics.ExecutionCount)
			kv("Success Count", fmt.Sprintf("%s%d%s", colorGreen, workflowMetrics.SuccessCount, colorReset))
			kv("Failure Count", fmt.Sprintf("%s%d%s", colorRed, workflowMetrics.FailureCount, colorReset))
			kv("Avg Duration", workflowMetrics.AverageDuration)
			kv("Min Duration", workflowMetrics.MinDuration)
			kv("Max Duration", workflowMetrics.MaxDuration)
		}
	}

	// Node metrics
	if len(nodeIDs) > 0 {
		section("\nNode Metrics:")
		for _, nodeID := range nodeIDs {
			nodeMetrics := metrics.GetNodeMetricsByID(nodeID)
			if nodeMetrics != nil {
				displayName := nodeMetrics.NodeName
				if displayName == "" {
					displayName = nodeID
				}
				fmt.Printf("\n  %s%s%s (%s)\n", bold, displayName, colorReset, nodeMetrics.NodeType)
				kv("Node ID", nodeMetrics.NodeID)
				kv("Execution Count", nodeMetrics.ExecutionCount)
				kv("Success Count", fmt.Sprintf("%s%d%s", colorGreen, nodeMetrics.SuccessCount, colorReset))
				kv("Failure Count", fmt.Sprintf("%s%d%s", colorRed, nodeMetrics.FailureCount, colorReset))
				kv("Retry Count", fmt.Sprintf("%s%d%s", colorYellow, nodeMetrics.RetryCount, colorReset))
				kv("Avg Duration", nodeMetrics.AverageDuration)
			}
		}
	}

	// AI metrics
	if showAIMetrics {
		aiMetrics := metrics.GetAIMetrics()
		if aiMetrics != nil {
			section("\nAI API Metrics:")
			kv("Total Requests", aiMetrics.TotalRequests)
			kv("Total Tokens", aiMetrics.TotalTokens)
			kv("Prompt Tokens", aiMetrics.PromptTokens)
			kv("Completion Tokens", aiMetrics.CompletionTokens)
			kv("Estimated Cost", fmt.Sprintf("$%.4f", aiMetrics.EstimatedCostUSD))
			kv("Avg Latency", aiMetrics.AverageLatency)
		}
	}

	fmt.Println()
}
