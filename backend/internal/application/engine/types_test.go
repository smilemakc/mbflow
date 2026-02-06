package engine

import (
	"testing"
	"time"

	pkgengine "github.com/smilemakc/mbflow/pkg/engine"
	"github.com/smilemakc/mbflow/pkg/models"
)

func TestDefaultExecutionOptions(t *testing.T) {
	opts := DefaultExecutionOptions()

	if opts == nil {
		t.Fatal("DefaultExecutionOptions() returned nil")
	}

	if opts.StrictMode != false {
		t.Errorf("expected StrictMode = false, got %v", opts.StrictMode)
	}

	if opts.MaxParallelism != 10 {
		t.Errorf("expected MaxParallelism = 10, got %d", opts.MaxParallelism)
	}

	if opts.Timeout != 5*time.Minute {
		t.Errorf("expected Timeout = 5m, got %v", opts.Timeout)
	}

	if opts.NodeTimeout != 1*time.Minute {
		t.Errorf("expected NodeTimeout = 1m, got %v", opts.NodeTimeout)
	}

	if opts.Variables == nil {
		t.Error("Variables map is nil")
	}
}

func TestNewExecutionState(t *testing.T) {
	workflow := &models.Workflow{
		ID:   "wf-123",
		Name: "Test Workflow",
		Nodes: []*models.Node{
			{ID: "node-1", Name: "Node 1", Type: "test"},
		},
	}

	input := map[string]interface{}{
		"key": "value",
	}

	variables := map[string]interface{}{
		"var1": "val1",
	}

	execState := pkgengine.NewExecutionState("exec-123", "wf-123", workflow, input, variables)

	if execState == nil {
		t.Fatal("NewExecutionState() returned nil")
	}

	if execState.ExecutionID != "exec-123" {
		t.Errorf("expected ExecutionID = exec-123, got %s", execState.ExecutionID)
	}

	if execState.WorkflowID != "wf-123" {
		t.Errorf("expected WorkflowID = wf-123, got %s", execState.WorkflowID)
	}

	if execState.Workflow != workflow {
		t.Error("Workflow not set correctly")
	}

	if execState.Input["key"] != "value" {
		t.Error("Input not set correctly")
	}

	if execState.Variables["var1"] != "val1" {
		t.Error("Variables not set correctly")
	}

	if execState.NodeOutputs == nil {
		t.Error("NodeOutputs map is nil")
	}

	if execState.NodeErrors == nil {
		t.Error("NodeErrors map is nil")
	}

	if execState.NodeStatus == nil {
		t.Error("NodeStatus map is nil")
	}
}

func TestExecutionState_SetAndGetNodeOutput(t *testing.T) {
	execState := pkgengine.NewExecutionState("exec-1", "wf-1", &models.Workflow{}, nil, nil)

	output := map[string]interface{}{
		"result": "success",
	}

	execState.SetNodeOutput("node-1", output)

	retrieved, ok := execState.GetNodeOutput("node-1")
	if !ok {
		t.Error("expected to find node output")
	}

	retrievedMap, ok := retrieved.(map[string]interface{})
	if !ok {
		t.Error("output is not a map")
	}

	if retrievedMap["result"] != "success" {
		t.Errorf("expected result=success, got %v", retrievedMap["result"])
	}

	_, ok = execState.GetNodeOutput("non-existent")
	if ok {
		t.Error("expected not to find non-existent node output")
	}
}

func TestExecutionState_SetAndGetNodeError(t *testing.T) {
	execState := pkgengine.NewExecutionState("exec-1", "wf-1", &models.Workflow{}, nil, nil)

	err := models.ErrNodeExecutionFailed

	execState.SetNodeError("node-1", err)

	retrieved, ok := execState.GetNodeError("node-1")
	if !ok {
		t.Error("expected to find node error")
	}

	if retrieved != err {
		t.Errorf("expected error %v, got %v", err, retrieved)
	}

	_, ok = execState.GetNodeError("non-existent")
	if ok {
		t.Error("expected not to find non-existent node error")
	}
}

func TestExecutionState_SetAndGetNodeStatus(t *testing.T) {
	execState := pkgengine.NewExecutionState("exec-1", "wf-1", &models.Workflow{}, nil, nil)

	status := models.NodeExecutionStatusCompleted

	execState.SetNodeStatus("node-1", status)

	retrieved, ok := execState.GetNodeStatus("node-1")
	if !ok {
		t.Error("expected to find node status")
	}

	if retrieved != status {
		t.Errorf("expected status %v, got %v", status, retrieved)
	}

	_, ok = execState.GetNodeStatus("non-existent")
	if ok {
		t.Error("expected not to find non-existent node status")
	}
}

func TestExecutionState_Concurrent(t *testing.T) {
	execState := pkgengine.NewExecutionState("exec-1", "wf-1", &models.Workflow{}, nil, nil)
	done := make(chan bool)

	go func() {
		for i := 0; i < 100; i++ {
			execState.SetNodeOutput("node-1", map[string]interface{}{"count": i})
		}
		done <- true
	}()

	go func() {
		for i := 0; i < 100; i++ {
			execState.GetNodeOutput("node-1")
		}
		done <- true
	}()

	go func() {
		for i := 0; i < 100; i++ {
			execState.SetNodeStatus("node-1", models.NodeExecutionStatusRunning)
		}
		done <- true
	}()

	go func() {
		for i := 0; i < 100; i++ {
			execState.GetNodeStatus("node-1")
		}
		done <- true
	}()

	for i := 0; i < 4; i++ {
		<-done
	}

	_, ok := execState.GetNodeOutput("node-1")
	if !ok {
		t.Error("execution state corrupted after concurrent access")
	}
}

func TestNodeContext_Structure(t *testing.T) {
	nodeCtx := &pkgengine.NodeContext{
		ExecutionID: "exec-123",
		NodeID:      "node-1",
		Node: &models.Node{
			ID:   "node-1",
			Name: "Test Node",
			Type: "http",
			Config: map[string]interface{}{
				"url": "https://api.example.com",
			},
		},
		WorkflowVariables: map[string]interface{}{
			"apiKey": "secret",
		},
		ExecutionVariables: map[string]interface{}{
			"overrideVar": "override",
		},
		DirectParentOutput: map[string]interface{}{
			"data": "parent-data",
		},
		StrictMode: true,
	}

	if nodeCtx.ExecutionID != "exec-123" {
		t.Errorf("expected ExecutionID = exec-123, got %s", nodeCtx.ExecutionID)
	}

	if nodeCtx.NodeID != "node-1" {
		t.Errorf("expected NodeID = node-1, got %s", nodeCtx.NodeID)
	}

	if nodeCtx.Node == nil {
		t.Error("Node is nil")
	}

	if nodeCtx.WorkflowVariables["apiKey"] != "secret" {
		t.Error("WorkflowVariables not set correctly")
	}

	if nodeCtx.ExecutionVariables["overrideVar"] != "override" {
		t.Error("ExecutionVariables not set correctly")
	}

	if nodeCtx.DirectParentOutput["data"] != "parent-data" {
		t.Error("DirectParentOutput not set correctly")
	}

	if !nodeCtx.StrictMode {
		t.Error("expected StrictMode = true")
	}
}

func TestExecutionOptions_CustomValues(t *testing.T) {
	opts := &ExecutionOptions{
		StrictMode:     true,
		MaxParallelism: 5,
		Timeout:        10 * time.Minute,
		NodeTimeout:    2 * time.Minute,
		Variables: map[string]interface{}{
			"custom": "value",
		},
	}

	if opts.StrictMode != true {
		t.Error("expected StrictMode = true")
	}

	if opts.MaxParallelism != 5 {
		t.Errorf("expected MaxParallelism = 5, got %d", opts.MaxParallelism)
	}

	if opts.Timeout != 10*time.Minute {
		t.Errorf("expected Timeout = 10m, got %v", opts.Timeout)
	}

	if opts.NodeTimeout != 2*time.Minute {
		t.Errorf("expected NodeTimeout = 2m, got %v", opts.NodeTimeout)
	}

	if opts.Variables["custom"] != "value" {
		t.Error("Variables not set correctly")
	}
}

func TestExecutionState_GetNodeStartTime(t *testing.T) {
	workflow := &models.Workflow{ID: "wf-1", Name: "Test"}
	state := pkgengine.NewExecutionState("exec-1", "wf-1", workflow, nil, nil)
	nodeID := "node1"

	_, ok := state.GetNodeStartTime(nodeID)
	if ok {
		t.Error("start time should not exist initially")
	}

	now := time.Now()
	state.SetNodeStartTime(nodeID, now)
	startTime, ok := state.GetNodeStartTime(nodeID)
	if !ok {
		t.Error("start time should exist after setting")
	}
	if startTime.Unix() != now.Unix() {
		t.Errorf("expected start time %v, got %v", now, startTime)
	}
}

func TestExecutionState_GetNodeEndTime(t *testing.T) {
	workflow := &models.Workflow{ID: "wf-1", Name: "Test"}
	state := pkgengine.NewExecutionState("exec-1", "wf-1", workflow, nil, nil)
	nodeID := "node1"

	_, ok := state.GetNodeEndTime(nodeID)
	if ok {
		t.Error("end time should not exist initially")
	}

	now := time.Now()
	state.SetNodeEndTime(nodeID, now)
	endTime, ok := state.GetNodeEndTime(nodeID)
	if !ok {
		t.Error("end time should exist after setting")
	}
	if endTime.Unix() != now.Unix() {
		t.Errorf("expected end time %v, got %v", now, endTime)
	}
}

func TestExecutionState_GetNodeInput(t *testing.T) {
	workflow := &models.Workflow{ID: "wf-1", Name: "Test"}
	state := pkgengine.NewExecutionState("exec-1", "wf-1", workflow, nil, nil)
	nodeID := "node1"

	_, ok := state.GetNodeInput(nodeID)
	if ok {
		t.Error("input should not exist initially")
	}

	input := map[string]interface{}{
		"user": "john",
		"age":  30,
	}
	state.SetNodeInput(nodeID, input)
	retrieved, ok := state.GetNodeInput(nodeID)
	if !ok {
		t.Error("input should exist after setting")
	}
	retrievedMap := retrieved.(map[string]interface{})
	if retrievedMap["user"] != "john" || retrievedMap["age"] != 30 {
		t.Errorf("retrieved input doesn't match: %v", retrieved)
	}
}

func TestExecutionState_GetNodeConfig(t *testing.T) {
	workflow := &models.Workflow{ID: "wf-1", Name: "Test"}
	state := pkgengine.NewExecutionState("exec-1", "wf-1", workflow, nil, nil)
	nodeID := "node1"

	_, ok := state.GetNodeConfig(nodeID)
	if ok {
		t.Error("config should not exist initially")
	}

	config := map[string]interface{}{
		"url":     "https://api.example.com",
		"timeout": 30,
	}
	state.SetNodeConfig(nodeID, config)
	retrieved, ok := state.GetNodeConfig(nodeID)
	if !ok {
		t.Error("config should exist after setting")
	}
	if retrieved["url"] != "https://api.example.com" {
		t.Errorf("retrieved config doesn't match: %v", retrieved)
	}
}

func TestExecutionState_GetNodeResolvedConfig(t *testing.T) {
	workflow := &models.Workflow{ID: "wf-1", Name: "Test"}
	state := pkgengine.NewExecutionState("exec-1", "wf-1", workflow, nil, nil)
	nodeID := "node1"

	_, ok := state.GetNodeResolvedConfig(nodeID)
	if ok {
		t.Error("resolved config should not exist initially")
	}

	config := map[string]interface{}{
		"url":    "https://api.example.com/users/123",
		"apiKey": "resolved-key-456",
	}
	state.SetNodeResolvedConfig(nodeID, config)
	retrieved, ok := state.GetNodeResolvedConfig(nodeID)
	if !ok {
		t.Error("resolved config should exist after setting")
	}
	if retrieved["url"] != "https://api.example.com/users/123" {
		t.Errorf("retrieved resolved config doesn't match: %v", retrieved)
	}
}

func TestExecutionState_ClearNodeOutput(t *testing.T) {
	workflow := &models.Workflow{ID: "wf-1", Name: "Test"}
	state := pkgengine.NewExecutionState("exec-1", "wf-1", workflow, nil, nil)
	nodeID := "node1"

	output := map[string]interface{}{
		"result": "success",
		"data":   []int{1, 2, 3},
	}
	state.SetNodeOutput(nodeID, output)

	_, ok := state.GetNodeOutput(nodeID)
	if !ok {
		t.Error("output should exist after setting")
	}

	state.ClearNodeOutput(nodeID)

	_, ok = state.GetNodeOutput(nodeID)
	if ok {
		t.Error("output should not exist after clearing")
	}
}

func TestExecutionState_GetTotalMemoryUsage(t *testing.T) {
	workflow := &models.Workflow{ID: "wf-1", Name: "Test"}
	state := pkgengine.NewExecutionState("exec-1", "wf-1", workflow, nil, nil)

	usage := state.GetTotalMemoryUsage()
	if usage != 0 {
		t.Errorf("expected 0 usage for empty state, got %d", usage)
	}

	state.SetNodeOutput("node1", "small string")
	state.SetNodeOutput("node2", map[string]interface{}{
		"key1": "value1",
		"key2": "value2",
	})
	state.SetNodeOutput("node3", []byte{1, 2, 3, 4, 5})

	usage = state.GetTotalMemoryUsage()
	if usage <= 0 {
		t.Error("expected non-zero usage after adding outputs")
	}

	state.ClearNodeOutput("node2")
	newUsage := state.GetTotalMemoryUsage()
	if newUsage >= usage {
		t.Errorf("expected usage to decrease after clearing output, was %d, now %d", usage, newUsage)
	}
}
