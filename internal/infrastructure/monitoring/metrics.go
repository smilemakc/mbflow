package monitoring

import (
	"sync"
	"time"
)

// MetricsCollector collects execution metrics for workflows and nodes.
// It tracks execution counts, durations, success/failure rates, and AI API usage.
type MetricsCollector struct {
	// workflowMetrics stores metrics per workflow
	workflowMetrics map[string]*WorkflowMetrics
	// nodeMetrics stores metrics per node ID
	nodeMetrics map[string]*NodeMetrics
	// aiMetrics stores AI API usage metrics
	aiMetrics *AIMetrics
	// mu protects concurrent access
	mu sync.RWMutex
}

// WorkflowMetrics represents metrics for a workflow.
type WorkflowMetrics struct {
	WorkflowID      string        `json:"workflow_id"`
	ExecutionCount  int           `json:"execution_count"`
	SuccessCount    int           `json:"success_count"`
	FailureCount    int           `json:"failure_count"`
	TotalDuration   time.Duration `json:"total_duration"`
	AverageDuration time.Duration `json:"average_duration"`
	MinDuration     time.Duration `json:"min_duration"`
	MaxDuration     time.Duration `json:"max_duration"`
	LastExecutionAt time.Time     `json:"last_execution_at"`
}

// NodeMetrics represents metrics for a specific node instance.
type NodeMetrics struct {
	NodeID          string        `json:"node_id"`
	NodeType        string        `json:"node_type"`
	NodeName        string        `json:"node_name"`
	ExecutionCount  int           `json:"execution_count"`
	SuccessCount    int           `json:"success_count"`
	FailureCount    int           `json:"failure_count"`
	RetryCount      int           `json:"retry_count"`
	TotalDuration   time.Duration `json:"total_duration"`
	AverageDuration time.Duration `json:"average_duration"`
	MinDuration     time.Duration `json:"min_duration"`
	MaxDuration     time.Duration `json:"max_duration"`
}

// AIMetrics represents AI API usage metrics.
type AIMetrics struct {
	TotalRequests    int           `json:"total_requests"`
	TotalTokens      int           `json:"total_tokens"`
	PromptTokens     int           `json:"prompt_tokens"`
	CompletionTokens int           `json:"completion_tokens"`
	EstimatedCostUSD float64       `json:"estimated_cost_usd"`
	AverageLatency   time.Duration `json:"average_latency"`
	mu               sync.RWMutex
}

// NewMetricsCollector creates a new MetricsCollector.
func NewMetricsCollector() *MetricsCollector {
	return &MetricsCollector{
		workflowMetrics: make(map[string]*WorkflowMetrics),
		nodeMetrics:     make(map[string]*NodeMetrics),
		aiMetrics:       &AIMetrics{},
	}
}

// RecordWorkflowExecution records metrics for a workflow execution.
func (mc *MetricsCollector) RecordWorkflowExecution(workflowID string, duration time.Duration, success bool) {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	metrics, ok := mc.workflowMetrics[workflowID]
	if !ok {
		metrics = &WorkflowMetrics{
			WorkflowID:  workflowID,
			MinDuration: duration,
			MaxDuration: duration,
		}
		mc.workflowMetrics[workflowID] = metrics
	}

	metrics.ExecutionCount++
	if success {
		metrics.SuccessCount++
	} else {
		metrics.FailureCount++
	}

	metrics.TotalDuration += duration
	metrics.AverageDuration = metrics.TotalDuration / time.Duration(metrics.ExecutionCount)
	metrics.LastExecutionAt = time.Now()

	if duration < metrics.MinDuration {
		metrics.MinDuration = duration
	}
	if duration > metrics.MaxDuration {
		metrics.MaxDuration = duration
	}
}

// RecordNodeExecution records metrics for a node execution.
func (mc *MetricsCollector) RecordNodeExecution(nodeID, nodeType, nodeName string, duration time.Duration, success bool, isRetry bool) {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	metrics, ok := mc.nodeMetrics[nodeID]
	if !ok {
		metrics = &NodeMetrics{
			NodeID:      nodeID,
			NodeType:    nodeType,
			NodeName:    nodeName,
			MinDuration: duration,
			MaxDuration: duration,
		}
		mc.nodeMetrics[nodeID] = metrics
	}

	metrics.ExecutionCount++
	if success {
		metrics.SuccessCount++
	} else {
		metrics.FailureCount++
	}
	if isRetry {
		metrics.RetryCount++
	}

	metrics.TotalDuration += duration
	metrics.AverageDuration = metrics.TotalDuration / time.Duration(metrics.ExecutionCount)

	if duration < metrics.MinDuration {
		metrics.MinDuration = duration
	}
	if duration > metrics.MaxDuration {
		metrics.MaxDuration = duration
	}
}

// RecordAIRequest records metrics for an AI API request.
func (mc *MetricsCollector) RecordAIRequest(promptTokens, completionTokens int, latency time.Duration) {
	mc.aiMetrics.mu.Lock()
	defer mc.aiMetrics.mu.Unlock()

	mc.aiMetrics.TotalRequests++
	mc.aiMetrics.PromptTokens += promptTokens
	mc.aiMetrics.CompletionTokens += completionTokens
	mc.aiMetrics.TotalTokens += promptTokens + completionTokens

	// Simple cost estimation (GPT-4 pricing as of 2024)
	// $0.03 per 1K prompt tokens, $0.06 per 1K completion tokens
	promptCost := float64(promptTokens) / 1000.0 * 0.03
	completionCost := float64(completionTokens) / 1000.0 * 0.06
	mc.aiMetrics.EstimatedCostUSD += promptCost + completionCost

	// Update average latency
	totalLatency := time.Duration(mc.aiMetrics.TotalRequests-1) * mc.aiMetrics.AverageLatency
	mc.aiMetrics.AverageLatency = (totalLatency + latency) / time.Duration(mc.aiMetrics.TotalRequests)
}

// GetWorkflowMetrics returns metrics for a specific workflow.
func (mc *MetricsCollector) GetWorkflowMetrics(workflowID string) *WorkflowMetrics {
	mc.mu.RLock()
	defer mc.mu.RUnlock()

	if metrics, ok := mc.workflowMetrics[workflowID]; ok {
		// Return a copy
		c := *metrics
		return &c
	}
	return nil
}

// GetAllWorkflowMetrics returns metrics for all workflows.
func (mc *MetricsCollector) GetAllWorkflowMetrics() map[string]*WorkflowMetrics {
	mc.mu.RLock()
	defer mc.mu.RUnlock()

	result := make(map[string]*WorkflowMetrics)
	for k, v := range mc.workflowMetrics {
		c := *v
		result[k] = &c
	}
	return result
}

// GetNodeMetricsByID returns metrics for a specific node ID.
func (mc *MetricsCollector) GetNodeMetricsByID(nodeID string) *NodeMetrics {
	mc.mu.RLock()
	defer mc.mu.RUnlock()

	if metrics, ok := mc.nodeMetrics[nodeID]; ok {
		c := *metrics
		return &c
	}
	return nil
}

// GetNodeMetrics returns aggregated metrics for a specific node type.
// This method satisfies the ExecutorMetrics interface.
func (mc *MetricsCollector) GetNodeMetrics(nodeType string) *NodeMetrics {
	return mc.GetNodeMetricsByType(nodeType)
}

// GetNodeMetricsByType returns aggregated metrics for a specific node type.
func (mc *MetricsCollector) GetNodeMetricsByType(nodeType string) *NodeMetrics {
	mc.mu.RLock()
	defer mc.mu.RUnlock()

	aggregated := &NodeMetrics{
		NodeType: nodeType,
	}

	found := false
	for _, m := range mc.nodeMetrics {
		if m.NodeType == nodeType {
			if !found {
				aggregated.MinDuration = m.MinDuration
				aggregated.MaxDuration = m.MaxDuration
				found = true
			}

			aggregated.ExecutionCount += m.ExecutionCount
			aggregated.SuccessCount += m.SuccessCount
			aggregated.FailureCount += m.FailureCount
			aggregated.RetryCount += m.RetryCount
			aggregated.TotalDuration += m.TotalDuration

			if m.MinDuration < aggregated.MinDuration {
				aggregated.MinDuration = m.MinDuration
			}
			if m.MaxDuration > aggregated.MaxDuration {
				aggregated.MaxDuration = m.MaxDuration
			}
		}
	}

	if !found {
		return nil
	}

	if aggregated.ExecutionCount > 0 {
		aggregated.AverageDuration = aggregated.TotalDuration / time.Duration(aggregated.ExecutionCount)
	}

	return aggregated
}

// GetAllNodeMetrics returns metrics for all nodes.
func (mc *MetricsCollector) GetAllNodeMetrics() map[string]*NodeMetrics {
	mc.mu.RLock()
	defer mc.mu.RUnlock()

	result := make(map[string]*NodeMetrics)
	for k, v := range mc.nodeMetrics {
		c := *v
		result[k] = &c
	}
	return result
}

// GetAIMetrics returns AI API usage metrics.
func (mc *MetricsCollector) GetAIMetrics() *AIMetrics {
	mc.aiMetrics.mu.RLock()
	defer mc.aiMetrics.mu.RUnlock()

	// Return a new struct with copied values (not copying the mutex)
	return &AIMetrics{
		TotalRequests:    mc.aiMetrics.TotalRequests,
		TotalTokens:      mc.aiMetrics.TotalTokens,
		PromptTokens:     mc.aiMetrics.PromptTokens,
		CompletionTokens: mc.aiMetrics.CompletionTokens,
		EstimatedCostUSD: mc.aiMetrics.EstimatedCostUSD,
		AverageLatency:   mc.aiMetrics.AverageLatency,
	}
}

// GetSuccessRate returns the success rate for a workflow.
func (mc *MetricsCollector) GetSuccessRate(workflowID string) float64 {
	mc.mu.RLock()
	defer mc.mu.RUnlock()

	if metrics, ok := mc.workflowMetrics[workflowID]; ok {
		if metrics.ExecutionCount == 0 {
			return 0.0
		}
		return float64(metrics.SuccessCount) / float64(metrics.ExecutionCount)
	}
	return 0.0
}

// GetNodeSuccessRate returns the success rate for a node type.
func (mc *MetricsCollector) GetNodeSuccessRate(nodeType string) float64 {
	mc.mu.RLock()
	defer mc.mu.RUnlock()

	if metrics, ok := mc.nodeMetrics[nodeType]; ok {
		if metrics.ExecutionCount == 0 {
			return 0.0
		}
		return float64(metrics.SuccessCount) / float64(metrics.ExecutionCount)
	}
	return 0.0
}

// Reset resets all metrics.
func (mc *MetricsCollector) Reset() {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	mc.workflowMetrics = make(map[string]*WorkflowMetrics)
	mc.nodeMetrics = make(map[string]*NodeMetrics)
	mc.aiMetrics = &AIMetrics{}
}

// Summary returns a summary of all collected metrics.
type MetricsSummary struct {
	TotalWorkflows      int     `json:"total_workflows"`
	TotalExecutions     int     `json:"total_executions"`
	TotalSuccesses      int     `json:"total_successes"`
	TotalFailures       int     `json:"total_failures"`
	OverallSuccessRate  float64 `json:"overall_success_rate"`
	TotalNodeExecutions int     `json:"total_node_executions"`
	TotalNodeRetries    int     `json:"total_node_retries"`
	TotalAIRequests     int     `json:"total_ai_requests"`
	TotalAITokens       int     `json:"total_ai_tokens"`
	EstimatedAICostUSD  float64 `json:"estimated_ai_cost_usd"`
}

// GetSummary returns a summary of all metrics.
func (mc *MetricsCollector) GetSummary() *MetricsSummary {
	mc.mu.RLock()
	defer mc.mu.RUnlock()

	summary := &MetricsSummary{
		TotalWorkflows: len(mc.workflowMetrics),
	}

	for _, wm := range mc.workflowMetrics {
		summary.TotalExecutions += wm.ExecutionCount
		summary.TotalSuccesses += wm.SuccessCount
		summary.TotalFailures += wm.FailureCount
	}

	if summary.TotalExecutions > 0 {
		summary.OverallSuccessRate = float64(summary.TotalSuccesses) / float64(summary.TotalExecutions)
	}

	for _, nm := range mc.nodeMetrics {
		summary.TotalNodeExecutions += nm.ExecutionCount
		summary.TotalNodeRetries += nm.RetryCount
	}

	mc.aiMetrics.mu.RLock()
	summary.TotalAIRequests = mc.aiMetrics.TotalRequests
	summary.TotalAITokens = mc.aiMetrics.TotalTokens
	summary.EstimatedAICostUSD = mc.aiMetrics.EstimatedCostUSD
	mc.aiMetrics.mu.RUnlock()

	return summary
}

// MetricsSnapshot represents a complete snapshot of all metrics at a point in time.
// This structure is used for serialization, persistence, and export.
type MetricsSnapshot struct {
	Timestamp       time.Time                   `json:"timestamp"`
	WorkflowMetrics map[string]*WorkflowMetrics `json:"workflow_metrics,omitempty"`
	NodeMetrics     map[string]*NodeMetrics     `json:"node_metrics,omitempty"`
	AIMetrics       *AIMetrics                  `json:"ai_metrics,omitempty"`
	Summary         *MetricsSummary             `json:"summary"`
}

// Snapshot creates a complete snapshot of all current metrics.
// This is thread-safe and returns a copy of all metrics data.
func (mc *MetricsCollector) Snapshot() *MetricsSnapshot {
	return &MetricsSnapshot{
		Timestamp:       time.Now(),
		WorkflowMetrics: mc.GetAllWorkflowMetrics(),
		NodeMetrics:     mc.GetAllNodeMetrics(),
		AIMetrics:       mc.GetAIMetrics(),
		Summary:         mc.GetSummary(),
	}
}
