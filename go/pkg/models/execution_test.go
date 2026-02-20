package models

import (
	"testing"
	"time"
)

func TestExecutionStatus_IsTerminal(t *testing.T) {
	tests := []struct {
		name     string
		status   ExecutionStatus
		expected bool
	}{
		{"completed is terminal", ExecutionStatusCompleted, true},
		{"failed is terminal", ExecutionStatusFailed, true},
		{"cancelled is terminal", ExecutionStatusCancelled, true},
		{"timeout is terminal", ExecutionStatusTimeout, true},
		{"pending is not terminal", ExecutionStatusPending, false},
		{"running is not terminal", ExecutionStatusRunning, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.status.IsTerminal(); got != tt.expected {
				t.Errorf("IsTerminal() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestNodeExecutionStatus_IsTerminal(t *testing.T) {
	tests := []struct {
		name     string
		status   NodeExecutionStatus
		expected bool
	}{
		{"completed is terminal", NodeExecutionStatusCompleted, true},
		{"failed is terminal", NodeExecutionStatusFailed, true},
		{"skipped is terminal", NodeExecutionStatusSkipped, true},
		{"cancelled is terminal", NodeExecutionStatusCancelled, true},
		{"pending is not terminal", NodeExecutionStatusPending, false},
		{"running is not terminal", NodeExecutionStatusRunning, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.status.IsTerminal(); got != tt.expected {
				t.Errorf("IsTerminal() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestExecution_GetNodeExecution(t *testing.T) {
	nodeExecs := []*NodeExecution{
		{ID: "ne-1", NodeID: "node-1", Status: NodeExecutionStatusCompleted},
		{ID: "ne-2", NodeID: "node-2", Status: NodeExecutionStatusCompleted},
		{ID: "ne-3", NodeID: "node-3", Status: NodeExecutionStatusFailed},
	}

	execution := &Execution{
		ID:             "exec-1",
		WorkflowID:     "wf-1",
		Status:         ExecutionStatusCompleted,
		NodeExecutions: nodeExecs,
	}

	tests := []struct {
		name    string
		nodeID  string
		wantErr bool
		wantNE  string
	}{
		{
			name:    "find existing node execution",
			nodeID:  "node-2",
			wantErr: false,
			wantNE:  "ne-2",
		},
		{
			name:    "node execution not found",
			nodeID:  "non-existent",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ne, err := execution.GetNodeExecution(tt.nodeID)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				if err != ErrNodeNotFound {
					t.Errorf("expected ErrNodeNotFound, got %v", err)
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if ne == nil {
					t.Fatal("node execution is nil")
				}
				if ne.ID != tt.wantNE {
					t.Errorf("expected node execution ID %s, got %s", tt.wantNE, ne.ID)
				}
			}
		})
	}
}

func TestExecution_CalculateDuration(t *testing.T) {
	startTime := time.Now().Add(-5 * time.Second)
	completedTime := startTime.Add(3 * time.Second)

	tests := []struct {
		name      string
		execution *Execution
		wantRange [2]int64 // min and max expected duration in ms
	}{
		{
			name: "completed execution",
			execution: &Execution{
				StartedAt:   startTime,
				CompletedAt: &completedTime,
			},
			wantRange: [2]int64{2900, 3100}, // ~3000ms with 100ms tolerance
		},
		{
			name: "running execution",
			execution: &Execution{
				StartedAt: time.Now().Add(-2 * time.Second),
			},
			wantRange: [2]int64{1900, 2100}, // ~2000ms with 100ms tolerance
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			duration := tt.execution.CalculateDuration()
			if duration < tt.wantRange[0] || duration > tt.wantRange[1] {
				t.Errorf("CalculateDuration() = %d, want between %d and %d",
					duration, tt.wantRange[0], tt.wantRange[1])
			}
		})
	}
}

func TestNodeExecution_CalculateDuration(t *testing.T) {
	startTime := time.Now().Add(-2 * time.Second)
	completedTime := startTime.Add(1 * time.Second)

	tests := []struct {
		name      string
		nodeExec  *NodeExecution
		wantRange [2]int64
	}{
		{
			name: "completed node execution",
			nodeExec: &NodeExecution{
				StartedAt:   startTime,
				CompletedAt: &completedTime,
			},
			wantRange: [2]int64{900, 1100}, // ~1000ms
		},
		{
			name: "running node execution",
			nodeExec: &NodeExecution{
				StartedAt: time.Now().Add(-500 * time.Millisecond),
			},
			wantRange: [2]int64{400, 600}, // ~500ms
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			duration := tt.nodeExec.CalculateDuration()
			if duration < tt.wantRange[0] || duration > tt.wantRange[1] {
				t.Errorf("CalculateDuration() = %d, want between %d and %d",
					duration, tt.wantRange[0], tt.wantRange[1])
			}
		})
	}
}

func TestExecution_GetSuccessRate(t *testing.T) {
	tests := []struct {
		name      string
		execution *Execution
		wantRate  float64
	}{
		{
			name: "all completed",
			execution: &Execution{
				NodeExecutions: []*NodeExecution{
					{Status: NodeExecutionStatusCompleted},
					{Status: NodeExecutionStatusCompleted},
					{Status: NodeExecutionStatusCompleted},
				},
			},
			wantRate: 100.0,
		},
		{
			name: "50% success rate",
			execution: &Execution{
				NodeExecutions: []*NodeExecution{
					{Status: NodeExecutionStatusCompleted},
					{Status: NodeExecutionStatusCompleted},
					{Status: NodeExecutionStatusFailed},
					{Status: NodeExecutionStatusFailed},
				},
			},
			wantRate: 50.0,
		},
		{
			name: "all failed",
			execution: &Execution{
				NodeExecutions: []*NodeExecution{
					{Status: NodeExecutionStatusFailed},
					{Status: NodeExecutionStatusFailed},
				},
			},
			wantRate: 0.0,
		},
		{
			name: "no node executions",
			execution: &Execution{
				NodeExecutions: []*NodeExecution{},
			},
			wantRate: 0.0,
		},
		{
			name: "mixed statuses",
			execution: &Execution{
				NodeExecutions: []*NodeExecution{
					{Status: NodeExecutionStatusCompleted},
					{Status: NodeExecutionStatusFailed},
					{Status: NodeExecutionStatusSkipped},
					{Status: NodeExecutionStatusCancelled},
				},
			},
			wantRate: 25.0, // Only completed counts as success
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rate := tt.execution.GetSuccessRate()
			if rate != tt.wantRate {
				t.Errorf("GetSuccessRate() = %v, want %v", rate, tt.wantRate)
			}
		})
	}
}

func TestExecution_GetFailedNodes(t *testing.T) {
	tests := []struct {
		name          string
		execution     *Execution
		expectedCount int
	}{
		{
			name: "some failed nodes",
			execution: &Execution{
				NodeExecutions: []*NodeExecution{
					{NodeID: "node-1", Status: NodeExecutionStatusCompleted},
					{NodeID: "node-2", Status: NodeExecutionStatusFailed},
					{NodeID: "node-3", Status: NodeExecutionStatusFailed},
					{NodeID: "node-4", Status: NodeExecutionStatusCompleted},
				},
			},
			expectedCount: 2,
		},
		{
			name: "no failed nodes",
			execution: &Execution{
				NodeExecutions: []*NodeExecution{
					{NodeID: "node-1", Status: NodeExecutionStatusCompleted},
					{NodeID: "node-2", Status: NodeExecutionStatusCompleted},
				},
			},
			expectedCount: 0,
		},
		{
			name: "all failed nodes",
			execution: &Execution{
				NodeExecutions: []*NodeExecution{
					{NodeID: "node-1", Status: NodeExecutionStatusFailed},
					{NodeID: "node-2", Status: NodeExecutionStatusFailed},
				},
			},
			expectedCount: 2,
		},
		{
			name: "no node executions",
			execution: &Execution{
				NodeExecutions: []*NodeExecution{},
			},
			expectedCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			failed := tt.execution.GetFailedNodes()
			if len(failed) != tt.expectedCount {
				t.Errorf("GetFailedNodes() returned %d nodes, want %d", len(failed), tt.expectedCount)
			}
			// Verify all returned nodes are actually failed
			for _, ne := range failed {
				if ne.Status != NodeExecutionStatusFailed {
					t.Errorf("GetFailedNodes() returned node with status %s, expected failed", ne.Status)
				}
			}
		})
	}
}

func TestExecutionStatus_Constants(t *testing.T) {
	statuses := []ExecutionStatus{
		ExecutionStatusPending,
		ExecutionStatusRunning,
		ExecutionStatusCompleted,
		ExecutionStatusFailed,
		ExecutionStatusCancelled,
		ExecutionStatusTimeout,
	}

	expectedValues := []string{
		"pending",
		"running",
		"completed",
		"failed",
		"cancelled",
		"timeout",
	}

	for i, status := range statuses {
		if string(status) != expectedValues[i] {
			t.Errorf("expected status %s, got %s", expectedValues[i], string(status))
		}
	}
}

func TestNodeExecutionStatus_Constants(t *testing.T) {
	statuses := []NodeExecutionStatus{
		NodeExecutionStatusPending,
		NodeExecutionStatusRunning,
		NodeExecutionStatusCompleted,
		NodeExecutionStatusFailed,
		NodeExecutionStatusSkipped,
		NodeExecutionStatusCancelled,
	}

	expectedValues := []string{
		"pending",
		"running",
		"completed",
		"failed",
		"skipped",
		"cancelled",
	}

	for i, status := range statuses {
		if string(status) != expectedValues[i] {
			t.Errorf("expected status %s, got %s", expectedValues[i], string(status))
		}
	}
}
