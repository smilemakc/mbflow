package grpc

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/smilemakc/mbflow/go/internal/application/serviceapi"
	"github.com/smilemakc/mbflow/go/pkg/models"
)

// --- toProtoWorkflow tests ---

func TestToProtoWorkflow_ShouldReturnNil_WhenInputIsNil(t *testing.T) {
	result := toProtoWorkflow(nil)

	assert.Nil(t, result)
}

func TestToProtoWorkflow_ShouldConvertFullWorkflow_WhenAllFieldsPopulated(t *testing.T) {
	now := time.Date(2025, 1, 15, 10, 30, 0, 0, time.UTC)
	updated := time.Date(2025, 1, 16, 12, 0, 0, 0, time.UTC)

	workflow := &models.Workflow{
		ID:          "wf-123",
		Name:        "Test Workflow",
		Description: "A test workflow",
		Version:     3,
		Status:      models.WorkflowStatusActive,
		CreatedBy:   "user-456",
		CreatedAt:   now,
		UpdatedAt:   updated,
		Variables:   map[string]any{"env": "production"},
		Metadata:    map[string]any{"priority": "high"},
		Nodes: []*models.Node{
			{
				ID:   "node-1",
				Name: "Start",
				Type: "start",
				Config: map[string]any{
					"timeout": float64(30),
				},
				Position: &models.Position{X: 100.0, Y: 200.0},
			},
			{
				ID:   "node-2",
				Name: "Process",
				Type: "transform",
			},
		},
		Edges: []*models.Edge{
			{
				ID:        "edge-1",
				From:      "node-1",
				To:        "node-2",
				Condition: "next",
				Metadata:  map[string]any{"label": "next"},
			},
		},
	}

	result := toProtoWorkflow(workflow)

	require.NotNil(t, result)
	assert.Equal(t, "wf-123", result.Id)
	assert.Equal(t, "Test Workflow", result.Name)
	assert.Equal(t, "A test workflow", result.Description)
	assert.Equal(t, "active", result.Status)
	assert.Equal(t, int32(3), result.Version)
	assert.Equal(t, "user-456", result.CreatedBy)
	assert.Equal(t, now.Unix(), result.CreatedAt.AsTime().Unix())
	assert.Equal(t, updated.Unix(), result.UpdatedAt.AsTime().Unix())

	// Variables
	require.NotNil(t, result.Variables)
	varsMap := result.Variables.AsMap()
	assert.Equal(t, "production", varsMap["env"])

	// Metadata
	require.NotNil(t, result.Metadata)
	metaMap := result.Metadata.AsMap()
	assert.Equal(t, "high", metaMap["priority"])

	// Nodes
	require.Len(t, result.Nodes, 2)
	assert.Equal(t, "node-1", result.Nodes[0].Id)
	assert.Equal(t, "Start", result.Nodes[0].Name)
	assert.Equal(t, "start", result.Nodes[0].Type)
	require.NotNil(t, result.Nodes[0].Config)
	assert.Equal(t, float64(30), result.Nodes[0].Config.AsMap()["timeout"])
	require.NotNil(t, result.Nodes[0].Position)
	posMap := result.Nodes[0].Position.AsMap()
	assert.Equal(t, float64(100), posMap["x"])
	assert.Equal(t, float64(200), posMap["y"])

	assert.Equal(t, "node-2", result.Nodes[1].Id)
	assert.Equal(t, "Process", result.Nodes[1].Name)
	assert.Equal(t, "transform", result.Nodes[1].Type)
	assert.Nil(t, result.Nodes[1].Position)

	// Edges
	require.Len(t, result.Edges, 1)
	assert.Equal(t, "edge-1", result.Edges[0].Id)
	assert.Equal(t, "node-1", result.Edges[0].From)
	assert.Equal(t, "node-2", result.Edges[0].To)
	require.NotNil(t, result.Edges[0].Condition)
	assert.Equal(t, "next", result.Edges[0].Condition.AsMap()["expression"])
}

func TestToProtoWorkflow_ShouldHandleNilFields_WhenOptionalFieldsAreNil(t *testing.T) {
	now := time.Date(2025, 1, 15, 10, 30, 0, 0, time.UTC)

	workflow := &models.Workflow{
		ID:        "wf-minimal",
		Name:      "Minimal",
		Status:    models.WorkflowStatusDraft,
		Version:   1,
		CreatedAt: now,
		UpdatedAt: now,
		// Variables, Metadata, Nodes, Edges all nil
	}

	result := toProtoWorkflow(workflow)

	require.NotNil(t, result)
	assert.Equal(t, "wf-minimal", result.Id)
	assert.Equal(t, "Minimal", result.Name)
	assert.Equal(t, "draft", result.Status)
	assert.Equal(t, int32(1), result.Version)
	assert.Nil(t, result.Variables)
	assert.Nil(t, result.Metadata)
	assert.Nil(t, result.Nodes)
	assert.Nil(t, result.Edges)
	assert.Empty(t, result.Description)
	assert.Empty(t, result.CreatedBy)
}

func TestToProtoWorkflow_ShouldHandleEmptySlices_WhenNodesAndEdgesEmpty(t *testing.T) {
	now := time.Now()

	workflow := &models.Workflow{
		ID:        "wf-empty",
		Name:      "Empty",
		Status:    models.WorkflowStatusInactive,
		Version:   1,
		CreatedAt: now,
		UpdatedAt: now,
		Nodes:     []*models.Node{},
		Edges:     []*models.Edge{},
	}

	result := toProtoWorkflow(workflow)

	require.NotNil(t, result)
	assert.Len(t, result.Nodes, 0)
	assert.Len(t, result.Edges, 0)
}

// --- toProtoExecution tests ---

func TestToProtoExecution_ShouldReturnNil_WhenInputIsNil(t *testing.T) {
	result := toProtoExecution(nil)

	assert.Nil(t, result)
}

func TestToProtoExecution_ShouldConvertFullExecution_WhenAllFieldsPopulated(t *testing.T) {
	startedAt := time.Date(2025, 2, 1, 8, 0, 0, 0, time.UTC)
	completedAt := time.Date(2025, 2, 1, 8, 5, 0, 0, time.UTC)
	nodeStarted := time.Date(2025, 2, 1, 8, 1, 0, 0, time.UTC)
	nodeCompleted := time.Date(2025, 2, 1, 8, 2, 0, 0, time.UTC)

	execution := &models.Execution{
		ID:          "exec-001",
		WorkflowID:  "wf-123",
		Status:      models.ExecutionStatusCompleted,
		Error:       "",
		StartedAt:   startedAt,
		CompletedAt: &completedAt,
		Duration:    300000,
		Input:       map[string]any{"param": "value"},
		Output:      map[string]any{"result": "success"},
		NodeExecutions: []*models.NodeExecution{
			{
				ID:          "ne-001",
				NodeID:      "node-1",
				NodeName:    "Start",
				NodeType:    "start",
				Status:      models.NodeExecutionStatusCompleted,
				Error:       "",
				StartedAt:   nodeStarted,
				CompletedAt: &nodeCompleted,
				Duration:    60000,
				Input:       map[string]any{"in": "data"},
				Output:      map[string]any{"out": "result"},
			},
		},
	}

	result := toProtoExecution(execution)

	require.NotNil(t, result)
	assert.Equal(t, "exec-001", result.Id)
	assert.Equal(t, "wf-123", result.WorkflowId)
	assert.Equal(t, "completed", result.Status)
	assert.Empty(t, result.Error)
	assert.Equal(t, int64(300000), result.DurationMs)
	assert.Equal(t, startedAt.Unix(), result.StartedAt.AsTime().Unix())
	assert.Equal(t, completedAt.Unix(), result.CompletedAt.AsTime().Unix())
	assert.Equal(t, startedAt.Unix(), result.CreatedAt.AsTime().Unix())

	// Input/Output
	require.NotNil(t, result.Input)
	assert.Equal(t, "value", result.Input.AsMap()["param"])
	require.NotNil(t, result.Output)
	assert.Equal(t, "success", result.Output.AsMap()["result"])

	// NodeExecutions
	require.Len(t, result.NodeExecutions, 1)
	ne := result.NodeExecutions[0]
	assert.Equal(t, "ne-001", ne.Id)
	assert.Equal(t, "node-1", ne.NodeId)
	assert.Equal(t, "Start", ne.NodeName)
	assert.Equal(t, "start", ne.NodeType)
	assert.Equal(t, "completed", ne.Status)
	assert.Empty(t, ne.Error)
	assert.Equal(t, int64(60000), ne.DurationMs)
	assert.Equal(t, nodeStarted.Unix(), ne.StartedAt.AsTime().Unix())
	assert.Equal(t, nodeCompleted.Unix(), ne.CompletedAt.AsTime().Unix())
	require.NotNil(t, ne.Input)
	assert.Equal(t, "data", ne.Input.AsMap()["in"])
	require.NotNil(t, ne.Output)
	assert.Equal(t, "result", ne.Output.AsMap()["out"])
}

func TestToProtoExecution_ShouldHandleNilCompletedAt_WhenExecutionRunning(t *testing.T) {
	startedAt := time.Date(2025, 2, 1, 8, 0, 0, 0, time.UTC)

	execution := &models.Execution{
		ID:         "exec-002",
		WorkflowID: "wf-123",
		Status:     models.ExecutionStatusRunning,
		StartedAt:  startedAt,
		// CompletedAt is nil
	}

	result := toProtoExecution(execution)

	require.NotNil(t, result)
	assert.Equal(t, "running", result.Status)
	assert.Nil(t, result.CompletedAt)
	assert.Nil(t, result.Input)
	assert.Nil(t, result.Output)
	assert.Nil(t, result.NodeExecutions)
}

func TestToProtoExecution_ShouldPreserveErrorField_WhenExecutionFailed(t *testing.T) {
	startedAt := time.Date(2025, 2, 1, 8, 0, 0, 0, time.UTC)
	completedAt := time.Date(2025, 2, 1, 8, 0, 30, 0, time.UTC)

	execution := &models.Execution{
		ID:          "exec-003",
		WorkflowID:  "wf-123",
		Status:      models.ExecutionStatusFailed,
		Error:       "node timeout exceeded",
		StartedAt:   startedAt,
		CompletedAt: &completedAt,
		Duration:    30000,
	}

	result := toProtoExecution(execution)

	require.NotNil(t, result)
	assert.Equal(t, "failed", result.Status)
	assert.Equal(t, "node timeout exceeded", result.Error)
}

func TestToProtoExecution_ShouldHandleNodeExecutionWithNilCompletedAt(t *testing.T) {
	startedAt := time.Now()

	execution := &models.Execution{
		ID:         "exec-004",
		WorkflowID: "wf-123",
		Status:     models.ExecutionStatusRunning,
		StartedAt:  startedAt,
		NodeExecutions: []*models.NodeExecution{
			{
				ID:        "ne-002",
				NodeID:    "node-1",
				NodeName:  "Processing",
				NodeType:  "transform",
				Status:    models.NodeExecutionStatusRunning,
				StartedAt: startedAt,
				// CompletedAt nil
			},
		},
	}

	result := toProtoExecution(execution)

	require.NotNil(t, result)
	require.Len(t, result.NodeExecutions, 1)
	assert.Nil(t, result.NodeExecutions[0].CompletedAt)
	assert.Equal(t, "running", result.NodeExecutions[0].Status)
}

// --- toProtoTrigger tests ---

func TestToProtoTrigger_ShouldReturnNil_WhenInputIsNil(t *testing.T) {
	result := toProtoTrigger(nil)

	assert.Nil(t, result)
}

func TestToProtoTrigger_ShouldConvertFullTrigger_WhenAllFieldsPopulated(t *testing.T) {
	createdAt := time.Date(2025, 3, 1, 9, 0, 0, 0, time.UTC)
	updatedAt := time.Date(2025, 3, 2, 10, 0, 0, 0, time.UTC)
	lastRun := time.Date(2025, 3, 3, 11, 0, 0, 0, time.UTC)

	trigger := &models.Trigger{
		ID:          "trig-001",
		WorkflowID:  "wf-123",
		Name:        "Nightly Build",
		Description: "Runs every night at midnight",
		Type:        models.TriggerTypeCron,
		Config: map[string]any{
			"schedule": "0 0 * * *",
			"timezone": "UTC",
		},
		Enabled:   true,
		CreatedAt: createdAt,
		UpdatedAt: updatedAt,
		LastRun:   &lastRun,
	}

	result := toProtoTrigger(trigger)

	require.NotNil(t, result)
	assert.Equal(t, "trig-001", result.Id)
	assert.Equal(t, "wf-123", result.WorkflowId)
	assert.Equal(t, "Nightly Build", result.Name)
	assert.Equal(t, "Runs every night at midnight", result.Description)
	assert.Equal(t, "cron", result.Type)
	assert.True(t, result.Enabled)
	assert.Equal(t, createdAt.Unix(), result.CreatedAt.AsTime().Unix())
	assert.Equal(t, updatedAt.Unix(), result.UpdatedAt.AsTime().Unix())
	assert.Equal(t, lastRun.Unix(), result.LastRun.AsTime().Unix())

	require.NotNil(t, result.Config)
	configMap := result.Config.AsMap()
	assert.Equal(t, "0 0 * * *", configMap["schedule"])
	assert.Equal(t, "UTC", configMap["timezone"])
}

func TestToProtoTrigger_ShouldHandleNilLastRun_WhenTriggerNeverRan(t *testing.T) {
	now := time.Now()

	trigger := &models.Trigger{
		ID:         "trig-002",
		WorkflowID: "wf-456",
		Name:       "Manual Trigger",
		Type:       models.TriggerTypeManual,
		Enabled:    false,
		CreatedAt:  now,
		UpdatedAt:  now,
		// LastRun nil, Config nil
	}

	result := toProtoTrigger(trigger)

	require.NotNil(t, result)
	assert.Equal(t, "trig-002", result.Id)
	assert.Equal(t, "manual", result.Type)
	assert.False(t, result.Enabled)
	assert.Nil(t, result.LastRun)
	assert.Nil(t, result.Config)
}

// --- toProtoCredential tests ---

func TestToProtoCredential_ShouldReturnNil_WhenInputIsNil(t *testing.T) {
	result := toProtoCredential(nil)

	assert.Nil(t, result)
}

func TestToProtoCredential_ShouldConvertFullCredential_WhenAllFieldsPopulated(t *testing.T) {
	createdAt := time.Date(2025, 4, 1, 14, 0, 0, 0, time.UTC)
	updatedAt := time.Date(2025, 4, 2, 15, 0, 0, 0, time.UTC)
	expiresAt := time.Date(2026, 4, 1, 14, 0, 0, 0, time.UTC)
	lastUsedAt := time.Date(2025, 4, 3, 16, 0, 0, 0, time.UTC)

	cred := &serviceapi.CredentialInfo{
		ID:             "cred-001",
		Name:           "API Key",
		Description:    "Production API key",
		Status:         "active",
		CredentialType: "api_key",
		Provider:       "openai",
		ExpiresAt:      &expiresAt,
		LastUsedAt:     &lastUsedAt,
		UsageCount:     42,
		CreatedAt:      createdAt,
		UpdatedAt:      updatedAt,
		Fields:         []string{"api_key", "org_id"},
	}

	result := toProtoCredential(cred)

	require.NotNil(t, result)
	assert.Equal(t, "cred-001", result.Id)
	assert.Equal(t, "API Key", result.Name)
	assert.Equal(t, "Production API key", result.Description)
	assert.Equal(t, "active", result.Status)
	assert.Equal(t, "api_key", result.CredentialType)
	assert.Equal(t, "openai", result.Provider)
	assert.Equal(t, int64(42), result.UsageCount)
	assert.Equal(t, createdAt.Unix(), result.CreatedAt.AsTime().Unix())
	assert.Equal(t, updatedAt.Unix(), result.UpdatedAt.AsTime().Unix())
	assert.Equal(t, expiresAt.Unix(), result.ExpiresAt.AsTime().Unix())
	assert.Equal(t, lastUsedAt.Unix(), result.LastUsedAt.AsTime().Unix())
	assert.Equal(t, []string{"api_key", "org_id"}, result.Fields)
}

func TestToProtoCredential_ShouldHandleNilOptionalTimestamps_WhenNoExpiryOrLastUsed(t *testing.T) {
	now := time.Now()

	cred := &serviceapi.CredentialInfo{
		ID:             "cred-002",
		Name:           "OAuth Token",
		Status:         "active",
		CredentialType: "oauth2",
		Provider:       "github",
		UsageCount:     0,
		CreatedAt:      now,
		UpdatedAt:      now,
		Fields:         []string{"access_token"},
		// ExpiresAt nil, LastUsedAt nil
	}

	result := toProtoCredential(cred)

	require.NotNil(t, result)
	assert.Equal(t, "cred-002", result.Id)
	assert.Nil(t, result.ExpiresAt)
	assert.Nil(t, result.LastUsedAt)
	assert.Equal(t, int64(0), result.UsageCount)
}

func TestToProtoCredential_ShouldHandleEmptyFields_WhenNoFieldKeys(t *testing.T) {
	now := time.Now()

	cred := &serviceapi.CredentialInfo{
		ID:             "cred-003",
		Name:           "Empty Cred",
		Status:         "inactive",
		CredentialType: "api_key",
		CreatedAt:      now,
		UpdatedAt:      now,
		Fields:         []string{},
	}

	result := toProtoCredential(cred)

	require.NotNil(t, result)
	assert.Empty(t, result.Fields)
}

func TestToProtoCredential_ShouldHandleNilFields_WhenFieldsNotSet(t *testing.T) {
	now := time.Now()

	cred := &serviceapi.CredentialInfo{
		ID:             "cred-004",
		Name:           "No Fields",
		Status:         "active",
		CredentialType: "api_key",
		CreatedAt:      now,
		UpdatedAt:      now,
		// Fields nil
	}

	result := toProtoCredential(cred)

	require.NotNil(t, result)
	assert.Nil(t, result.Fields)
}

// --- mapToStruct tests ---

func TestMapToStruct_ShouldReturnNil_WhenMapIsNil(t *testing.T) {
	result := mapToStruct(nil)

	assert.Nil(t, result)
}

func TestMapToStruct_ShouldConvertValidMap_WhenMapHasStringKeys(t *testing.T) {
	m := map[string]any{
		"name":    "test",
		"count":   float64(42),
		"enabled": true,
	}

	result := mapToStruct(m)

	require.NotNil(t, result)
	asMap := result.AsMap()
	assert.Equal(t, "test", asMap["name"])
	assert.Equal(t, float64(42), asMap["count"])
	assert.Equal(t, true, asMap["enabled"])
}

func TestMapToStruct_ShouldHandleEmptyMap_WhenMapHasNoEntries(t *testing.T) {
	m := map[string]any{}

	result := mapToStruct(m)

	require.NotNil(t, result)
	assert.Empty(t, result.AsMap())
}

func TestMapToStruct_ShouldHandleNestedMap_WhenMapContainsSubMaps(t *testing.T) {
	m := map[string]any{
		"nested": map[string]any{
			"key": "value",
		},
	}

	result := mapToStruct(m)

	require.NotNil(t, result)
	asMap := result.AsMap()
	nested, ok := asMap["nested"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "value", nested["key"])
}

func TestMapToStruct_ShouldHandleListValues_WhenMapContainsSlices(t *testing.T) {
	m := map[string]any{
		"tags": []any{"a", "b", "c"},
	}

	result := mapToStruct(m)

	require.NotNil(t, result)
	asMap := result.AsMap()
	tags, ok := asMap["tags"].([]any)
	require.True(t, ok)
	assert.Len(t, tags, 3)
	assert.Equal(t, "a", tags[0])
}

func TestMapToStruct_ShouldReturnNil_WhenMapContainsInvalidValues(t *testing.T) {
	// structpb.NewStruct fails for values that cannot be represented in protobuf struct
	m := map[string]any{
		"channel": make(chan int),
	}

	result := mapToStruct(m)

	assert.Nil(t, result)
}

// --- structToMap tests ---

func TestStructToMap_ShouldReturnNil_WhenStructIsNil(t *testing.T) {
	result := structToMap(nil)

	assert.Nil(t, result)
}

func TestStructToMap_ShouldConvertValidStruct_WhenStructHasFields(t *testing.T) {
	s, err := structpb.NewStruct(map[string]any{
		"name":  "workflow",
		"count": float64(10),
	})
	require.NoError(t, err)

	result := structToMap(s)

	require.NotNil(t, result)
	assert.Equal(t, "workflow", result["name"])
	assert.Equal(t, float64(10), result["count"])
}

func TestStructToMap_ShouldHandleEmptyStruct_WhenStructHasNoFields(t *testing.T) {
	s, err := structpb.NewStruct(map[string]any{})
	require.NoError(t, err)

	result := structToMap(s)

	require.NotNil(t, result)
	assert.Empty(t, result)
}

func TestStructToMap_ShouldHandleNestedStruct_WhenStructContainsSubStructs(t *testing.T) {
	s, err := structpb.NewStruct(map[string]any{
		"outer": map[string]any{
			"inner": "deep",
		},
	})
	require.NoError(t, err)

	result := structToMap(s)

	require.NotNil(t, result)
	outer, ok := result["outer"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "deep", outer["inner"])
}

// --- optionalTimestamp tests ---

func TestOptionalTimestamp_ShouldReturnNil_WhenTimestampIsNil(t *testing.T) {
	result := optionalTimestamp(nil)

	assert.Nil(t, result)
}

func TestOptionalTimestamp_ShouldReturnTimePointer_WhenTimestampIsValid(t *testing.T) {
	expected := time.Date(2025, 6, 15, 12, 30, 0, 0, time.UTC)
	ts := timestamppb.New(expected)

	result := optionalTimestamp(ts)

	require.NotNil(t, result)
	assert.Equal(t, expected.Unix(), result.Unix())
}

func TestOptionalTimestamp_ShouldReturnZeroTime_WhenTimestampIsEpoch(t *testing.T) {
	epoch := time.Unix(0, 0).UTC()
	ts := timestamppb.New(epoch)

	result := optionalTimestamp(ts)

	require.NotNil(t, result)
	assert.Equal(t, epoch.Unix(), result.Unix())
}

func TestOptionalTimestamp_ShouldPreserveNanoseconds_WhenTimestampHasNanos(t *testing.T) {
	expected := time.Date(2025, 6, 15, 12, 30, 45, 123456789, time.UTC)
	ts := timestamppb.New(expected)

	result := optionalTimestamp(ts)

	require.NotNil(t, result)
	assert.Equal(t, expected.UnixNano(), result.UnixNano())
}

// --- mapToStruct / structToMap roundtrip ---

func TestMapToStructAndStructToMap_ShouldRoundTrip_WhenValidMap(t *testing.T) {
	original := map[string]any{
		"string_val": "hello",
		"number_val": float64(3.14),
		"bool_val":   true,
		"null_val":   nil,
	}

	s := mapToStruct(original)
	require.NotNil(t, s)

	result := structToMap(s)

	require.NotNil(t, result)
	assert.Equal(t, "hello", result["string_val"])
	assert.Equal(t, float64(3.14), result["number_val"])
	assert.Equal(t, true, result["bool_val"])
	assert.Nil(t, result["null_val"])
}

// --- toProtoWorkflows / toProtoExecutions / toProtoTriggers / toProtoCredentials ---

func TestToProtoWorkflows_ShouldConvertSlice_WhenMultipleWorkflows(t *testing.T) {
	now := time.Now()
	workflows := []*models.Workflow{
		{ID: "wf-1", Name: "First", Status: models.WorkflowStatusActive, CreatedAt: now, UpdatedAt: now},
		{ID: "wf-2", Name: "Second", Status: models.WorkflowStatusDraft, CreatedAt: now, UpdatedAt: now},
	}

	result := toProtoWorkflows(workflows)

	require.Len(t, result, 2)
	assert.Equal(t, "wf-1", result[0].Id)
	assert.Equal(t, "wf-2", result[1].Id)
}

func TestToProtoWorkflows_ShouldReturnEmptySlice_WhenEmpty(t *testing.T) {
	result := toProtoWorkflows([]*models.Workflow{})

	require.NotNil(t, result)
	assert.Len(t, result, 0)
}

func TestToProtoExecutions_ShouldConvertSlice_WhenMultipleExecutions(t *testing.T) {
	now := time.Now()
	executions := []*models.Execution{
		{ID: "exec-1", WorkflowID: "wf-1", Status: models.ExecutionStatusCompleted, StartedAt: now},
		{ID: "exec-2", WorkflowID: "wf-1", Status: models.ExecutionStatusFailed, StartedAt: now},
	}

	result := toProtoExecutions(executions)

	require.Len(t, result, 2)
	assert.Equal(t, "exec-1", result[0].Id)
	assert.Equal(t, "exec-2", result[1].Id)
}

func TestToProtoTriggers_ShouldConvertSlice_WhenMultipleTriggers(t *testing.T) {
	now := time.Now()
	triggers := []*models.Trigger{
		{ID: "trig-1", WorkflowID: "wf-1", Name: "T1", Type: models.TriggerTypeCron, CreatedAt: now, UpdatedAt: now},
		{ID: "trig-2", WorkflowID: "wf-2", Name: "T2", Type: models.TriggerTypeManual, CreatedAt: now, UpdatedAt: now},
	}

	result := toProtoTriggers(triggers)

	require.Len(t, result, 2)
	assert.Equal(t, "trig-1", result[0].Id)
	assert.Equal(t, "trig-2", result[1].Id)
}

func TestToProtoCredentials_ShouldConvertSlice_WhenMultipleCredentials(t *testing.T) {
	now := time.Now()
	creds := []*serviceapi.CredentialInfo{
		{ID: "cred-1", Name: "Key 1", Status: "active", CredentialType: "api_key", CreatedAt: now, UpdatedAt: now},
		{ID: "cred-2", Name: "Key 2", Status: "expired", CredentialType: "oauth2", CreatedAt: now, UpdatedAt: now},
	}

	result := toProtoCredentials(creds)

	require.Len(t, result, 2)
	assert.Equal(t, "cred-1", result[0].Id)
	assert.Equal(t, "cred-2", result[1].Id)
}

// --- toProtoNode / toProtoEdge nil tests ---

func TestToProtoNode_ShouldReturnNil_WhenInputIsNil(t *testing.T) {
	result := toProtoNode(nil)

	assert.Nil(t, result)
}

func TestToProtoNode_ShouldConvert_WhenNoPositionOrConfig(t *testing.T) {
	node := &models.Node{
		ID:   "n-1",
		Name: "Simple",
		Type: "action",
	}

	result := toProtoNode(node)

	require.NotNil(t, result)
	assert.Equal(t, "n-1", result.Id)
	assert.Equal(t, "Simple", result.Name)
	assert.Equal(t, "action", result.Type)
	assert.Nil(t, result.Config)
	assert.Nil(t, result.Position)
}

func TestToProtoEdge_ShouldReturnNil_WhenInputIsNil(t *testing.T) {
	result := toProtoEdge(nil)

	assert.Nil(t, result)
}

func TestToProtoEdge_ShouldConvert_WhenNoMetadata(t *testing.T) {
	edge := &models.Edge{
		ID:   "e-1",
		From: "n-1",
		To:   "n-2",
	}

	result := toProtoEdge(edge)

	require.NotNil(t, result)
	assert.Equal(t, "e-1", result.Id)
	assert.Equal(t, "n-1", result.From)
	assert.Equal(t, "n-2", result.To)
	assert.Nil(t, result.Condition)
}

// --- toProtoNodeExecution nil test ---

func TestToProtoNodeExecution_ShouldReturnNil_WhenInputIsNil(t *testing.T) {
	result := toProtoNodeExecution(nil)

	assert.Nil(t, result)
}

// --- toProtoAuditLogEntry tests ---

func TestToProtoAuditLogEntry_ShouldReturnNil_WhenInputIsNil(t *testing.T) {
	result := toProtoAuditLogEntry(nil)

	assert.Nil(t, result)
}

func TestToProtoAuditLogEntry_ShouldConvertFull_WhenAllFieldsPopulated(t *testing.T) {
	createdAt := time.Date(2025, 5, 1, 10, 0, 0, 0, time.UTC)
	resourceID := "res-123"
	impersonatedUserID := "user-789"
	requestBody := `{"name":"test"}`

	log := &models.ServiceAuditLog{
		ID:                 "audit-001",
		SystemKeyID:        "key-001",
		ServiceName:        "workflow-service",
		ImpersonatedUserID: &impersonatedUserID,
		Action:             "create",
		ResourceType:       "workflow",
		ResourceID:         &resourceID,
		RequestMethod:      "POST",
		RequestPath:        "/api/v1/workflows",
		RequestBody:        &requestBody,
		ResponseStatus:     201,
		IPAddress:          "192.168.1.1",
		CreatedAt:          createdAt,
	}

	result := toProtoAuditLogEntry(log)

	require.NotNil(t, result)
	assert.Equal(t, "audit-001", result.Id)
	assert.Equal(t, "key-001", result.SystemKeyId)
	assert.Equal(t, "workflow-service", result.ServiceName)
	assert.Equal(t, "user-789", result.ImpersonatedUserId)
	assert.Equal(t, "create", result.Action)
	assert.Equal(t, "workflow", result.ResourceType)
	assert.Equal(t, "res-123", result.ResourceId)
	assert.Equal(t, "POST", result.Method)
	assert.Equal(t, "/api/v1/workflows", result.Path)
	assert.Equal(t, `{"name":"test"}`, result.RequestBody)
	assert.Equal(t, int32(201), result.ResponseStatus)
	assert.Equal(t, "192.168.1.1", result.ClientIp)
	assert.Equal(t, createdAt.Unix(), result.CreatedAt.AsTime().Unix())
}

func TestToProtoAuditLogEntry_ShouldHandleNilOptionalFields_WhenNotSet(t *testing.T) {
	createdAt := time.Date(2025, 5, 1, 10, 0, 0, 0, time.UTC)

	log := &models.ServiceAuditLog{
		ID:             "audit-002",
		SystemKeyID:    "key-002",
		ServiceName:    "billing-service",
		Action:         "list",
		ResourceType:   "transactions",
		RequestMethod:  "GET",
		RequestPath:    "/api/v1/transactions",
		ResponseStatus: 200,
		IPAddress:      "10.0.0.1",
		CreatedAt:      createdAt,
		// ResourceID, ImpersonatedUserID, RequestBody all nil
	}

	result := toProtoAuditLogEntry(log)

	require.NotNil(t, result)
	assert.Equal(t, "audit-002", result.Id)
	assert.Empty(t, result.ResourceId)
	assert.Empty(t, result.ImpersonatedUserId)
	assert.Empty(t, result.RequestBody)
}

func TestToProtoAuditLogEntries_ShouldConvertSlice_WhenMultipleEntries(t *testing.T) {
	now := time.Now()
	logs := []*models.ServiceAuditLog{
		{ID: "a-1", SystemKeyID: "k-1", ServiceName: "svc", Action: "get", ResourceType: "wf", RequestMethod: "GET", RequestPath: "/", IPAddress: "1.1.1.1", ResponseStatus: 200, CreatedAt: now},
		{ID: "a-2", SystemKeyID: "k-1", ServiceName: "svc", Action: "delete", ResourceType: "wf", RequestMethod: "DELETE", RequestPath: "/1", IPAddress: "1.1.1.1", ResponseStatus: 204, CreatedAt: now},
	}

	result := toProtoAuditLogEntries(logs)

	require.Len(t, result, 2)
	assert.Equal(t, "a-1", result[0].Id)
	assert.Equal(t, "a-2", result[1].Id)
}
