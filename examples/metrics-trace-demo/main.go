package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"mbflow/internal/application/executor"
	"mbflow/internal/infrastructure/monitoring"
)

// This demo demonstrates the complete workflow for collecting, storing, and exporting
// metrics and execution traces using MetricsCollector and ExecutionTrace.

func main() {
	fmt.Println("=== Metrics & Trace Collection Demo ===\n")

	// Create output directories
	metricsDir := "./output/metrics"
	tracesDir := "./output/traces"

	if err := os.MkdirAll(metricsDir, 0755); err != nil {
		log.Fatal(err)
	}
	if err := os.MkdirAll(tracesDir, 0755); err != nil {
		log.Fatal(err)
	}

	// Demo 1: Basic metrics collection
	demoBasicMetrics()

	// Demo 2: Execution trace collection
	demoExecutionTrace()

	// Demo 3: Combined metrics and traces with workflow execution
	demoWorkflowWithMetricsAndTraces(metricsDir, tracesDir)

	// Demo 4: Metrics persistence
	demoMetricsPersistence(metricsDir)

	// Demo 5: Trace persistence and export
	demoTracePersistence(tracesDir)

	// Demo 6: Analyzing saved metrics and traces
	demoAnalyzeData(metricsDir, tracesDir)

	fmt.Println("\n=== Demo Complete ===")
	fmt.Printf("Metrics saved to: %s\n", metricsDir)
	fmt.Printf("Traces saved to: %s\n", tracesDir)
}

// demoBasicMetrics demonstrates basic metrics collection
func demoBasicMetrics() {
	fmt.Println("--- 1. Basic Metrics Collection ---")

	metrics := monitoring.NewMetricsCollector()

	// Simulate workflow executions
	metrics.RecordWorkflowExecution("workflow-1", 250*time.Millisecond, true)
	metrics.RecordWorkflowExecution("workflow-1", 300*time.Millisecond, true)
	metrics.RecordWorkflowExecution("workflow-1", 280*time.Millisecond, false)

	// Simulate node executions
	metrics.RecordNodeExecution("http", 100*time.Millisecond, true, false)
	metrics.RecordNodeExecution("http", 120*time.Millisecond, false, false)
	metrics.RecordNodeExecution("http", 110*time.Millisecond, true, true) // retry
	metrics.RecordNodeExecution("transform", 50*time.Millisecond, true, false)

	// Simulate AI requests
	metrics.RecordAIRequest(500, 200, 2*time.Second)
	metrics.RecordAIRequest(600, 250, 2500*time.Millisecond)

	// Get and display summary
	summary := metrics.GetSummary()
	fmt.Printf("Total Workflows: %d\n", summary.TotalWorkflows)
	fmt.Printf("Total Executions: %d\n", summary.TotalExecutions)
	fmt.Printf("Success Rate: %.2f%%\n", summary.OverallSuccessRate*100)
	fmt.Printf("Total Node Executions: %d\n", summary.TotalNodeExecutions)
	fmt.Printf("Total AI Requests: %d\n", summary.TotalAIRequests)
	fmt.Printf("Estimated AI Cost: $%.4f\n", summary.EstimatedAICostUSD)
	fmt.Println()
}

// demoExecutionTrace demonstrates execution trace collection
func demoExecutionTrace() {
	fmt.Println("--- 2. Execution Trace Collection ---")

	trace := monitoring.NewExecutionTrace("exec-123", "workflow-1")

	// Add execution events
	trace.AddEvent("execution_started", "", "", "Workflow execution started", nil, nil)

	// Simulate node execution
	trace.AddEvent("node_started", "node-1", "http", "HTTP node started",
		map[string]interface{}{"url": "https://api.example.com"}, nil)

	time.Sleep(100 * time.Millisecond)

	trace.AddEvent("node_completed", "node-1", "http", "HTTP node completed",
		map[string]interface{}{"status": 200, "duration": "100ms"}, nil)

	// Simulate variable set
	trace.AddEvent("variable_set", "", "", "Variable set",
		map[string]interface{}{"key": "response_data", "value": "..."}, nil)

	// Simulate another node with error
	trace.AddEvent("node_started", "node-2", "transform", "Transform node started", nil, nil)
	trace.AddEvent("node_failed", "node-2", "transform", "Transform node failed",
		map[string]interface{}{"duration": "50ms"},
		fmt.Errorf("invalid expression"))

	// Retry
	trace.AddEvent("node_retrying", "node-2", "transform", "Retrying node",
		map[string]interface{}{"attempt": 2, "delay": "1s"}, nil)

	time.Sleep(50 * time.Millisecond)

	trace.AddEvent("node_completed", "node-2", "transform", "Transform node completed",
		map[string]interface{}{"duration": "45ms"}, nil)

	trace.AddEvent("execution_completed", "", "", "Workflow execution completed",
		map[string]interface{}{"total_duration": "200ms"}, nil)

	// Display trace
	fmt.Println(trace.String())

	// Display summary
	summary := trace.GetSummary()
	fmt.Printf("Summary:\n")
	fmt.Printf("  Total Events: %d\n", summary.TotalEvents)
	fmt.Printf("  Error Count: %d\n", summary.ErrorCount)
	fmt.Printf("  Duration: %v\n", summary.Duration)
	fmt.Printf("  Unique Nodes: %d\n", len(summary.NodeIDs))
	fmt.Println()
}

// demoWorkflowWithMetricsAndTraces demonstrates using metrics and traces with actual workflow execution
func demoWorkflowWithMetricsAndTraces(metricsDir, tracesDir string) {
	fmt.Println("--- 3. Workflow Execution with Metrics & Traces ---")

	// Create metrics collector and trace
	metrics := monitoring.NewMetricsCollector()
	trace := monitoring.NewExecutionTrace("exec-real-001", "workflow-demo")

	// Create console logger
	logger := monitoring.NewConsoleLogger(monitoring.ConsoleLoggerConfig{
		Prefix:  "DEMO",
		Verbose: false,
		Writer:  os.Stdout,
	})

	// Create composite observer that combines all three
	observer := monitoring.NewCompositeObserver(logger, metrics, trace)

	// Create workflow engine
	engine := executor.NewWorkflowEngine(&executor.EngineConfig{
		EnableMonitoring: false,
		VerboseLogging:   false,
	})

	// Add our composite observer
	engine.AddObserver(observer)

	// Define a simple workflow
	nodes := []executor.NodeConfig{
		{
			NodeID:   "start",
			NodeType: "start",
			Config:   map[string]interface{}{},
		},
		{
			NodeID:   "transform-1",
			NodeType: "transform",
			Config: map[string]interface{}{
				"expression": "user_count * 2",
			},
		},
		{
			NodeID:   "end",
			NodeType: "end",
			Config:   map[string]interface{}{},
		},
	}

	edges := []executor.EdgeConfig{
		{
			FromNodeID: "start",
			ToNodeID:   "transform-1",
			EdgeType:   "direct",
		},
		{
			FromNodeID: "transform-1",
			ToNodeID:   "end",
			EdgeType:   "direct",
		},
	}

	// Execute workflow
	ctx := context.Background()
	initialVars := map[string]interface{}{
		"user_count": 42,
	}

	_, err := engine.ExecuteWorkflow(ctx, "workflow-demo", "exec-real-001", nodes, edges, initialVars)
	if err != nil {
		log.Printf("Workflow execution failed: %v", err)
	}

	// Display collected metrics
	fmt.Println("\nCollected Metrics:")
	workflowMetrics := metrics.GetWorkflowMetrics("workflow-demo")
	if workflowMetrics != nil {
		fmt.Printf("  Executions: %d\n", workflowMetrics.ExecutionCount)
		fmt.Printf("  Success: %d, Failures: %d\n", workflowMetrics.SuccessCount, workflowMetrics.FailureCount)
		fmt.Printf("  Avg Duration: %v\n", workflowMetrics.AverageDuration)
	}

	// Display trace info
	fmt.Println("\nTrace Summary:")
	traceSummary := trace.GetSummary()
	fmt.Printf("  Total Events: %d\n", traceSummary.TotalEvents)
	fmt.Printf("  Duration: %v\n", traceSummary.Duration)
	fmt.Printf("  Has Errors: %v\n", trace.HasErrors())

	// Save metrics and trace
	metricsSnapshot := metrics.Snapshot()
	metricsFile, _ := monitoring.SaveMetricsToFileWithTimestamp(metricsSnapshot, metricsDir, "workflow-demo")
	fmt.Printf("\nMetrics saved to: %s\n", metricsFile)

	traceFile, _ := monitoring.SaveTraceToFileWithTimestamp(trace, tracesDir)
	fmt.Printf("Trace saved to: %s\n", traceFile)
	fmt.Println()
}

// demoMetricsPersistence demonstrates automatic metrics persistence
func demoMetricsPersistence(metricsDir string) {
	fmt.Println("--- 4. Metrics Persistence ---")

	metrics := monitoring.NewMetricsCollector()

	// Create persistence manager with 5-second save interval
	persistence := monitoring.NewMetricsPersistence(metrics, metricsDir, 5*time.Second)
	persistence.SetFilePrefix("auto-metrics")
	persistence.SetRetention(5) // Keep last 5 snapshots

	// Start automatic persistence
	persistence.Start()
	defer persistence.Stop()

	// Simulate some workflow executions over time
	for i := 0; i < 3; i++ {
		metrics.RecordWorkflowExecution("workflow-persistent",
			time.Duration(200+i*50)*time.Millisecond, true)
		metrics.RecordNodeExecution("http",
			time.Duration(100+i*10)*time.Millisecond, true, false)

		fmt.Printf("Recorded execution %d\n", i+1)
		time.Sleep(1 * time.Second)
	}

	// Save immediately
	savedFile, err := persistence.SaveNow()
	if err != nil {
		log.Printf("Failed to save metrics: %v", err)
	} else {
		fmt.Printf("Metrics snapshot saved to: %s\n", savedFile)
	}

	fmt.Println()
}

// demoTracePersistence demonstrates trace persistence
func demoTracePersistence(tracesDir string) {
	fmt.Println("--- 5. Trace Persistence ---")

	// Create multiple traces
	traces := make([]*monitoring.ExecutionTrace, 3)
	for i := 0; i < 3; i++ {
		execID := fmt.Sprintf("exec-%03d", i+1)
		trace := monitoring.NewExecutionTrace(execID, "workflow-batch")

		trace.AddEvent("execution_started", "", "", "Started", nil, nil)
		trace.AddEvent("node_started", "node-1", "http", "Processing", nil, nil)
		time.Sleep(10 * time.Millisecond)
		trace.AddEvent("node_completed", "node-1", "http", "Completed", nil, nil)
		trace.AddEvent("execution_completed", "", "", "Finished", nil, nil)

		traces[i] = trace
	}

	// Create persistence manager
	tracePersistence := monitoring.NewTracePersistence(tracesDir)

	// Save individual traces
	for _, trace := range traces {
		savedFile, err := tracePersistence.SaveTrace(trace)
		if err != nil {
			log.Printf("Failed to save trace: %v", err)
		} else {
			fmt.Printf("Trace saved: %s\n", savedFile)
		}
	}

	// Export all traces to a single text file
	textFile := tracesDir + "/all-traces.txt"
	if err := monitoring.ExportTracesAsText(traces, textFile); err != nil {
		log.Printf("Failed to export traces: %v", err)
	} else {
		fmt.Printf("All traces exported to: %s\n", textFile)
	}

	fmt.Println()
}

// demoAnalyzeData demonstrates loading and analyzing saved data
func demoAnalyzeData(metricsDir, tracesDir string) {
	fmt.Println("--- 6. Analyzing Saved Data ---")

	// This is a simplified demo - in production, you would scan directories
	// and load the most recent files

	fmt.Println("Metrics Analysis:")
	fmt.Println("  • Metrics are saved as JSON files in:", metricsDir)
	fmt.Println("  • Each file contains: workflow metrics, node metrics, AI metrics, and summary")
	fmt.Println("  • You can load metrics with monitoring.LoadMetricsFromFile(path)")
	fmt.Println()

	fmt.Println("Trace Analysis:")
	fmt.Println("  • Traces are saved as JSON files in:", tracesDir)
	fmt.Println("  • Each file contains: execution ID, workflow ID, and all events")
	fmt.Println("  • You can load traces with monitoring.LoadTraceFromFile(path)")
	fmt.Println("  • Text export available for human-readable analysis")
	fmt.Println()

	fmt.Println("Example Analysis Queries:")
	fmt.Println("  • Find slowest workflows by analyzing average duration")
	fmt.Println("  • Identify most failed node types")
	fmt.Println("  • Calculate AI API costs over time")
	fmt.Println("  • Debug failed executions using trace events")
	fmt.Println("  • Monitor retry patterns and error frequencies")
	fmt.Println()
}
