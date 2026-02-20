package models

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/smilemakc/mbflow/pkg/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test JSONBMap Type Operations

func TestJSONBMap_Value_Serialization(t *testing.T) {
	data := JSONBMap{
		"name":   "test",
		"count":  float64(42),
		"active": true,
	}

	value, err := data.Value()
	require.NoError(t, err)

	// Should return JSON string
	str, ok := value.(string)
	require.True(t, ok, "Value should return string")
	assert.Contains(t, str, "name")
	assert.Contains(t, str, "test")
}

func TestJSONBMap_Value_NilMap(t *testing.T) {
	var data JSONBMap

	value, err := data.Value()
	require.NoError(t, err)
	assert.Nil(t, value, "Nil map should serialize to nil")
}

func TestJSONBMap_Scan_Deserialization(t *testing.T) {
	jsonBytes := []byte(`{"name":"test","count":42,"active":true}`)

	var data JSONBMap
	err := data.Scan(jsonBytes)

	require.NoError(t, err)
	assert.Equal(t, "test", data["name"])
	assert.Equal(t, float64(42), data["count"])
	assert.Equal(t, true, data["active"])
}

func TestJSONBMap_Scan_NilValue(t *testing.T) {
	var data JSONBMap
	err := data.Scan(nil)

	require.NoError(t, err)
	assert.NotNil(t, data, "Scanning nil should create empty map")
	assert.Len(t, data, 0)
}

func TestJSONBMap_Scan_EmptyBytes(t *testing.T) {
	var data JSONBMap
	err := data.Scan([]byte{})

	require.NoError(t, err)
	assert.NotNil(t, data)
	assert.Len(t, data, 0)
}

func TestJSONBMap_GetString(t *testing.T) {
	data := JSONBMap{
		"name": "John Doe",
		"age":  float64(30),
	}

	assert.Equal(t, "John Doe", data.GetString("name"))
	assert.Equal(t, "", data.GetString("age"), "Should return empty string for non-string type")
	assert.Equal(t, "", data.GetString("missing"), "Should return empty string for missing key")
}

func TestJSONBMap_GetInt(t *testing.T) {
	data := JSONBMap{
		"count": float64(42),
		"name":  "test",
	}

	assert.Equal(t, 42, data.GetInt("count"))
	assert.Equal(t, 0, data.GetInt("name"), "Should return 0 for non-numeric type")
	assert.Equal(t, 0, data.GetInt("missing"), "Should return 0 for missing key")
}

func TestJSONBMap_GetFloat(t *testing.T) {
	data := JSONBMap{
		"price": float64(19.99),
		"name":  "item",
	}

	assert.Equal(t, 19.99, data.GetFloat("price"))
	assert.Equal(t, 0.0, data.GetFloat("name"), "Should return 0.0 for non-numeric type")
	assert.Equal(t, 0.0, data.GetFloat("missing"), "Should return 0.0 for missing key")
}

func TestJSONBMap_GetBool(t *testing.T) {
	data := JSONBMap{
		"active": true,
		"count":  float64(42),
	}

	assert.True(t, data.GetBool("active"))
	assert.False(t, data.GetBool("count"), "Should return false for non-bool type")
	assert.False(t, data.GetBool("missing"), "Should return false for missing key")
}

func TestJSONBMap_GetMap(t *testing.T) {
	data := JSONBMap{
		"user": map[string]any{
			"name": "John",
			"age":  float64(30),
		},
		"count": float64(42),
	}

	userMap := data.GetMap("user")
	assert.Equal(t, "John", userMap["name"])
	assert.Equal(t, float64(30), userMap["age"])

	emptyMap := data.GetMap("count")
	assert.Empty(t, emptyMap, "Should return empty map for non-map type")

	missingMap := data.GetMap("missing")
	assert.NotNil(t, missingMap, "Should return non-nil empty map for missing key")
	assert.Empty(t, missingMap)
}

func TestJSONBMap_SetAndHas(t *testing.T) {
	data := make(JSONBMap)

	assert.False(t, data.Has("key"), "Should not have key initially")

	data.Set("key", "value")
	assert.True(t, data.Has("key"), "Should have key after Set")
	assert.Equal(t, "value", data["key"])
}

func TestJSONBMap_Delete(t *testing.T) {
	data := JSONBMap{
		"key1": "value1",
		"key2": "value2",
	}

	data.Delete("key1")
	assert.False(t, data.Has("key1"), "Deleted key should not exist")
	assert.True(t, data.Has("key2"), "Other keys should remain")
}

func TestJSONBMap_Clone(t *testing.T) {
	original := JSONBMap{
		"name": "test",
		"nested": map[string]any{
			"value": float64(42),
		},
	}

	cloned := original.Clone()

	// Verify clone has same values
	assert.Equal(t, original["name"], cloned["name"])

	// Modify clone
	cloned.Set("name", "modified")

	// Original should be unchanged
	assert.Equal(t, "test", original["name"])
	assert.Equal(t, "modified", cloned["name"])
}

func TestJSONBMap_Clone_NilMap(t *testing.T) {
	var original JSONBMap

	cloned := original.Clone()
	assert.NotNil(t, cloned, "Clone of nil map should return non-nil empty map")
	assert.Empty(t, cloned)
}

// Test StringArray Type Operations

func TestStringArray_Value_Serialization(t *testing.T) {
	array := StringArray{"tag1", "tag2", "tag3"}

	value, err := array.Value()
	require.NoError(t, err)

	// Should return PostgreSQL array format: {"tag1","tag2","tag3"}
	str, ok := value.(string)
	require.True(t, ok, "Value should return string")
	assert.Equal(t, `{"tag1","tag2","tag3"}`, str)
}

func TestStringArray_Value_EmptyArray(t *testing.T) {
	array := StringArray{}

	value, err := array.Value()
	require.NoError(t, err)
	assert.Equal(t, "{}", value, "Empty array should serialize to {}")
}

func TestStringArray_Value_NilArray(t *testing.T) {
	var array StringArray

	value, err := array.Value()
	require.NoError(t, err)
	assert.Equal(t, "{}", value, "Nil array should serialize to {}")
}

func TestStringArray_Scan_Deserialization(t *testing.T) {
	// PostgreSQL array format
	pgArray := []byte(`{"tag1","tag2","tag3"}`)

	var array StringArray
	err := array.Scan(pgArray)

	require.NoError(t, err)
	assert.Len(t, array, 3)
	assert.Equal(t, "tag1", array[0])
	assert.Equal(t, "tag2", array[1])
	assert.Equal(t, "tag3", array[2])
}

func TestStringArray_Scan_EmptyArray(t *testing.T) {
	var array StringArray
	err := array.Scan([]byte("{}"))

	require.NoError(t, err)
	assert.Empty(t, array)
}

func TestStringArray_Scan_NilValue(t *testing.T) {
	var array StringArray
	err := array.Scan(nil)

	require.NoError(t, err)
	assert.NotNil(t, array, "Scanning nil should create empty array")
	assert.Empty(t, array)
}

func TestStringArray_Scan_StringValue(t *testing.T) {
	// PostgreSQL might return string instead of []byte
	var array StringArray
	err := array.Scan(`{"a","b","c"}`)

	require.NoError(t, err)
	assert.Len(t, array, 3)
	assert.Equal(t, "a", array[0])
}

// Test Workflow Mappers

func TestWorkflowToStorage_BasicConversion(t *testing.T) {
	domainWorkflow := &models.Workflow{
		Name:        "Test Workflow",
		Description: "Test Description",
		Version:     1,
		Status:      models.WorkflowStatusActive,
		Variables: map[string]any{
			"key": "value",
		},
		Metadata: map[string]any{
			"author": "test",
		},
		Tags: []string{"tag1", "tag2"},
	}

	workflowID := uuid.New()
	storageWorkflow := WorkflowToStorage(domainWorkflow, workflowID)

	assert.Equal(t, workflowID, storageWorkflow.ID)
	assert.Equal(t, "Test Workflow", storageWorkflow.Name)
	assert.Equal(t, "Test Description", storageWorkflow.Description)
	assert.Equal(t, 1, storageWorkflow.Version)
	assert.Equal(t, "active", storageWorkflow.Status)
	assert.Equal(t, "value", storageWorkflow.Variables.GetString("key"))
	assert.Equal(t, "test", storageWorkflow.Metadata.GetString("author"))

	// Tags should be stored in metadata
	tags, ok := storageWorkflow.Metadata["tags"].([]string)
	require.True(t, ok, "Tags should be in metadata")
	assert.Equal(t, []string{"tag1", "tag2"}, tags)
}

func TestWorkflowToStorage_WithNodes(t *testing.T) {
	domainWorkflow := &models.Workflow{
		Name: "Workflow with Nodes",
		Nodes: []*models.Node{
			{
				ID:   "node1",
				Name: "HTTP Node",
				Type: "http",
				Config: map[string]any{
					"url": "https://api.example.com",
				},
				Position: &models.Position{X: 100, Y: 200},
			},
		},
	}

	workflowID := uuid.New()
	storageWorkflow := WorkflowToStorage(domainWorkflow, workflowID)

	require.Len(t, storageWorkflow.Nodes, 1)
	node := storageWorkflow.Nodes[0]
	assert.Equal(t, "node1", node.NodeID)
	assert.Equal(t, "HTTP Node", node.Name)
	assert.Equal(t, "http", node.Type)
	assert.Equal(t, "https://api.example.com", node.Config.GetString("url"))
	assert.Equal(t, 100, node.Position.GetInt("x"))
	assert.Equal(t, 200, node.Position.GetInt("y"))
}

func TestWorkflowToStorage_WithEdges(t *testing.T) {
	domainWorkflow := &models.Workflow{
		Name: "Workflow with Edges",
		Edges: []*models.Edge{
			{
				ID:        "edge1",
				From:      "node1",
				To:        "node2",
				Condition: "output.success == true",
			},
		},
	}

	workflowID := uuid.New()
	storageWorkflow := WorkflowToStorage(domainWorkflow, workflowID)

	require.Len(t, storageWorkflow.Edges, 1)
	edge := storageWorkflow.Edges[0]
	assert.Equal(t, "edge1", edge.EdgeID)
	assert.Equal(t, "node1", edge.FromNodeID)
	assert.Equal(t, "node2", edge.ToNodeID)
	assert.Equal(t, "output.success == true", edge.Condition.GetString("expression"))
}

func TestWorkflowFromStorage_BasicConversion(t *testing.T) {
	workflowID := uuid.New()
	storageWorkflow := &WorkflowModel{
		ID:          workflowID,
		Name:        "Test Workflow",
		Description: "Test Description",
		Version:     1,
		Status:      "active",
		Variables: JSONBMap{
			"key": "value",
		},
		Metadata: JSONBMap{
			"author": "test",
			"tags":   []any{"tag1", "tag2"},
		},
	}

	domainWorkflow := WorkflowFromStorage(storageWorkflow)

	assert.Equal(t, workflowID.String(), domainWorkflow.ID)
	assert.Equal(t, "Test Workflow", domainWorkflow.Name)
	assert.Equal(t, "Test Description", domainWorkflow.Description)
	assert.Equal(t, 1, domainWorkflow.Version)
	assert.Equal(t, models.WorkflowStatusActive, domainWorkflow.Status)
	assert.Equal(t, "value", domainWorkflow.Variables["key"])
	assert.Equal(t, "test", domainWorkflow.Metadata["author"])

	// Tags should be extracted from metadata
	assert.Len(t, domainWorkflow.Tags, 2)
	assert.Equal(t, "tag1", domainWorkflow.Tags[0])
	assert.Equal(t, "tag2", domainWorkflow.Tags[1])
}

func TestWorkflowFromStorage_WithNodes(t *testing.T) {
	storageWorkflow := &WorkflowModel{
		ID:   uuid.New(),
		Name: "Workflow with Nodes",
		Nodes: []*NodeModel{
			{
				NodeID: "node1",
				Name:   "HTTP Node",
				Type:   "http",
				Config: JSONBMap{
					"url": "https://api.example.com",
				},
				Position: JSONBMap{
					"x": float64(100),
					"y": float64(200),
				},
			},
		},
	}

	domainWorkflow := WorkflowFromStorage(storageWorkflow)

	require.Len(t, domainWorkflow.Nodes, 1)
	node := domainWorkflow.Nodes[0]
	assert.Equal(t, "node1", node.ID)
	assert.Equal(t, "HTTP Node", node.Name)
	assert.Equal(t, "http", node.Type)
	assert.Equal(t, "https://api.example.com", node.Config["url"])
	assert.NotNil(t, node.Position)
	assert.Equal(t, 100.0, node.Position.X)
	assert.Equal(t, 200.0, node.Position.Y)
}

func TestWorkflowRoundTrip_PreservesData(t *testing.T) {
	// Create domain workflow
	originalWorkflow := &models.Workflow{
		Name:        "Round Trip Test",
		Description: "Testing round trip conversion",
		Version:     2,
		Status:      models.WorkflowStatusDraft,
		Tags:        []string{"test", "roundtrip"},
		Variables: map[string]any{
			"var1": "value1",
			"var2": float64(42),
		},
		Metadata: map[string]any{
			"key": "value",
		},
		Nodes: []*models.Node{
			{
				ID:   "node1",
				Name: "Test Node",
				Type: "http",
				Config: map[string]any{
					"url": "https://example.com",
				},
			},
		},
		Edges: []*models.Edge{
			{
				ID:   "edge1",
				From: "node1",
				To:   "node2",
			},
		},
	}

	// Convert to storage and back
	workflowID := uuid.New()
	storageWorkflow := WorkflowToStorage(originalWorkflow, workflowID)
	convertedWorkflow := WorkflowFromStorage(storageWorkflow)

	// Verify data preservation
	assert.Equal(t, originalWorkflow.Name, convertedWorkflow.Name)
	assert.Equal(t, originalWorkflow.Description, convertedWorkflow.Description)
	assert.Equal(t, originalWorkflow.Version, convertedWorkflow.Version)
	assert.Equal(t, originalWorkflow.Status, convertedWorkflow.Status)
	assert.Equal(t, originalWorkflow.Tags, convertedWorkflow.Tags)
	assert.Len(t, convertedWorkflow.Nodes, 1)
	assert.Len(t, convertedWorkflow.Edges, 1)
}

// Test Node Mappers

func TestNodeFromStorage_WithPosition(t *testing.T) {
	storageNode := &NodeModel{
		NodeID: "node1",
		Name:   "Test Node",
		Type:   "http",
		Config: JSONBMap{
			"url": "https://api.example.com",
		},
		Position: JSONBMap{
			"x": float64(150),
			"y": float64(250),
		},
	}

	domainNode := NodeFromStorage(storageNode)

	assert.Equal(t, "node1", domainNode.ID)
	assert.Equal(t, "Test Node", domainNode.Name)
	assert.Equal(t, "http", domainNode.Type)
	assert.Equal(t, "https://api.example.com", domainNode.Config["url"])
	require.NotNil(t, domainNode.Position)
	assert.Equal(t, 150.0, domainNode.Position.X)
	assert.Equal(t, 250.0, domainNode.Position.Y)
}

func TestNodeFromStorage_WithoutPosition(t *testing.T) {
	storageNode := &NodeModel{
		NodeID: "node1",
		Name:   "Test Node",
		Type:   "http",
		Config: JSONBMap{},
	}

	domainNode := NodeFromStorage(storageNode)

	assert.Nil(t, domainNode.Position, "Position should be nil when not provided")
}

// Test Edge Mappers

func TestEdgeFromStorage_WithCondition(t *testing.T) {
	storageEdge := &EdgeModel{
		EdgeID:     "edge1",
		FromNodeID: "node1",
		ToNodeID:   "node2",
		Condition: JSONBMap{
			"expression": "output.success == true",
		},
	}

	domainEdge := EdgeFromStorage(storageEdge)

	assert.Equal(t, "edge1", domainEdge.ID)
	assert.Equal(t, "node1", domainEdge.From)
	assert.Equal(t, "node2", domainEdge.To)
	assert.Equal(t, "output.success == true", domainEdge.Condition)
}

func TestEdgeFromStorage_WithoutCondition(t *testing.T) {
	storageEdge := &EdgeModel{
		EdgeID:     "edge1",
		FromNodeID: "node1",
		ToNodeID:   "node2",
	}

	domainEdge := EdgeFromStorage(storageEdge)

	assert.Empty(t, domainEdge.Condition, "Condition should be empty when not provided")
}

func TestEdgeFromStorage_WithSourceHandle(t *testing.T) {
	storageEdge := &EdgeModel{
		EdgeID:       "edge1",
		FromNodeID:   "check",
		ToNodeID:     "fix",
		SourceHandle: "false",
	}

	domainEdge := EdgeFromStorage(storageEdge)

	assert.Equal(t, "false", domainEdge.SourceHandle)
	assert.Nil(t, domainEdge.Loop)
}

func TestEdgeFromStorage_WithLoop(t *testing.T) {
	storageEdge := &EdgeModel{
		EdgeID:     "loop1",
		FromNodeID: "fix",
		ToNodeID:   "check",
		Loop:       JSONBMap{"max_iterations": float64(3)},
	}

	domainEdge := EdgeFromStorage(storageEdge)

	require.NotNil(t, domainEdge.Loop, "Loop should not be nil")
	assert.Equal(t, 3, domainEdge.Loop.MaxIterations)
	assert.Empty(t, domainEdge.Condition, "Loop edges should not have conditions")
}

func TestEdgeFromStorage_WithNilLoop(t *testing.T) {
	storageEdge := &EdgeModel{
		EdgeID:     "edge1",
		FromNodeID: "node1",
		ToNodeID:   "node2",
	}

	domainEdge := EdgeFromStorage(storageEdge)

	assert.Nil(t, domainEdge.Loop, "Loop should be nil when not provided")
}

func TestEdgeToStorage_WithLoopAndSourceHandle(t *testing.T) {
	workflowID := uuid.New()
	domainEdge := &models.Edge{
		ID:           "loop1",
		From:         "fix",
		To:           "check",
		SourceHandle: "",
		Loop:         &models.LoopConfig{MaxIterations: 5},
	}

	storageEdge := EdgeToStorage(domainEdge, workflowID)

	assert.Equal(t, "loop1", storageEdge.EdgeID)
	assert.Equal(t, workflowID, storageEdge.WorkflowID)
	assert.Equal(t, "fix", storageEdge.FromNodeID)
	assert.Equal(t, "check", storageEdge.ToNodeID)
	require.NotNil(t, storageEdge.Loop)
	assert.Equal(t, 5, storageEdge.Loop["max_iterations"])
}

func TestEdgeToStorage_WithSourceHandle(t *testing.T) {
	workflowID := uuid.New()
	domainEdge := &models.Edge{
		ID:           "cond1",
		From:         "check",
		To:           "ok",
		SourceHandle: "true",
	}

	storageEdge := EdgeToStorage(domainEdge, workflowID)

	assert.Equal(t, "true", storageEdge.SourceHandle)
	assert.Nil(t, storageEdge.Loop)
}

func TestEdgeModelToDomain_WithLoop(t *testing.T) {
	em := &EdgeModel{
		EdgeID:     "loop1",
		FromNodeID: "fix",
		ToNodeID:   "gen",
		Loop:       JSONBMap{"max_iterations": float64(2)},
	}

	edge := EdgeModelToDomain(em)

	require.NotNil(t, edge)
	require.NotNil(t, edge.Loop)
	assert.Equal(t, 2, edge.Loop.MaxIterations)
}

func TestEdgeModelToDomain_WithSourceHandle(t *testing.T) {
	em := &EdgeModel{
		EdgeID:       "cond1",
		FromNodeID:   "check",
		ToNodeID:     "fix",
		SourceHandle: "false",
	}

	edge := EdgeModelToDomain(em)

	require.NotNil(t, edge)
	assert.Equal(t, "false", edge.SourceHandle)
}

func TestEdgeRoundTrip_LoopAndSourceHandle(t *testing.T) {
	workflowID := uuid.New()
	original := &models.Edge{
		ID:           "loop1",
		From:         "regen",
		To:           "gen",
		SourceHandle: "",
		Loop:         &models.LoopConfig{MaxIterations: 3},
	}

	// Domain → Storage
	storageEdge := EdgeToStorage(original, workflowID)

	// Storage → Domain (via EdgeFromStorage)
	restored := EdgeFromStorage(storageEdge)

	assert.Equal(t, original.ID, restored.ID)
	assert.Equal(t, original.From, restored.From)
	assert.Equal(t, original.To, restored.To)
	require.NotNil(t, restored.Loop)
	assert.Equal(t, original.Loop.MaxIterations, restored.Loop.MaxIterations)
}

func TestEdgeRoundTrip_SourceHandle(t *testing.T) {
	workflowID := uuid.New()
	original := &models.Edge{
		ID:           "cond1",
		From:         "check",
		To:           "ok",
		SourceHandle: "true",
		Condition:    "",
	}

	storageEdge := EdgeToStorage(original, workflowID)
	restored := EdgeFromStorage(storageEdge)

	assert.Equal(t, "true", restored.SourceHandle)
	assert.Nil(t, restored.Loop)
}

// Test ExecutionModel Helper Methods

func TestExecutionModel_StatusCheckers(t *testing.T) {
	tests := []struct {
		name        string
		status      string
		isPending   bool
		isRunning   bool
		isCompleted bool
		isFailed    bool
		isCancelled bool
		isPaused    bool
		isTerminal  bool
	}{
		{"pending", "pending", true, false, false, false, false, false, false},
		{"running", "running", false, true, false, false, false, false, false},
		{"completed", "completed", false, false, true, false, false, false, true},
		{"failed", "failed", false, false, false, true, false, false, true},
		{"cancelled", "cancelled", false, false, false, false, true, false, true},
		{"paused", "paused", false, false, false, false, false, true, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			exec := &ExecutionModel{Status: tt.status}

			assert.Equal(t, tt.isPending, exec.IsPending())
			assert.Equal(t, tt.isRunning, exec.IsRunning())
			assert.Equal(t, tt.isCompleted, exec.IsCompleted())
			assert.Equal(t, tt.isFailed, exec.IsFailed())
			assert.Equal(t, tt.isCancelled, exec.IsCancelled())
			assert.Equal(t, tt.isPaused, exec.IsPaused())
			assert.Equal(t, tt.isTerminal, exec.IsTerminal())
		})
	}
}

func TestExecutionModel_Duration(t *testing.T) {
	t.Run("with both timestamps", func(t *testing.T) {
		start := time.Now().Add(-5 * time.Minute)
		end := time.Now()
		exec := &ExecutionModel{
			StartedAt:   &start,
			CompletedAt: &end,
		}

		duration := exec.Duration()
		require.NotNil(t, duration)
		assert.True(t, *duration >= 4*time.Minute && *duration <= 6*time.Minute)
	})

	t.Run("without started timestamp", func(t *testing.T) {
		end := time.Now()
		exec := &ExecutionModel{
			CompletedAt: &end,
		}

		duration := exec.Duration()
		assert.Nil(t, duration)
	})

	t.Run("without completed timestamp", func(t *testing.T) {
		start := time.Now()
		exec := &ExecutionModel{
			StartedAt: &start,
		}

		duration := exec.Duration()
		assert.Nil(t, duration)
	})
}

func TestExecutionModel_MarkStarted(t *testing.T) {
	exec := &ExecutionModel{Status: "pending"}

	exec.MarkStarted()

	assert.Equal(t, "running", exec.Status)
	assert.NotNil(t, exec.StartedAt)
	assert.WithinDuration(t, time.Now(), *exec.StartedAt, time.Second)
}

func TestExecutionModel_MarkCompleted(t *testing.T) {
	exec := &ExecutionModel{Status: "running"}

	exec.MarkCompleted()

	assert.Equal(t, "completed", exec.Status)
	assert.NotNil(t, exec.CompletedAt)
	assert.WithinDuration(t, time.Now(), *exec.CompletedAt, time.Second)
}

func TestExecutionModel_MarkFailed(t *testing.T) {
	exec := &ExecutionModel{Status: "running"}

	exec.MarkFailed("execution error")

	assert.Equal(t, "failed", exec.Status)
	assert.NotNil(t, exec.CompletedAt)
	assert.Equal(t, "execution error", exec.Error)
	assert.WithinDuration(t, time.Now(), *exec.CompletedAt, time.Second)
}

func TestExecutionModel_MarkCancelled(t *testing.T) {
	exec := &ExecutionModel{Status: "running"}

	exec.MarkCancelled()

	assert.Equal(t, "cancelled", exec.Status)
	assert.NotNil(t, exec.CompletedAt)
	assert.WithinDuration(t, time.Now(), *exec.CompletedAt, time.Second)
}

// Test NodeExecutionModel Helper Methods

func TestNodeExecutionModel_StatusCheckers(t *testing.T) {
	tests := []struct {
		name        string
		status      string
		isPending   bool
		isRunning   bool
		isCompleted bool
		isFailed    bool
		isSkipped   bool
		isRetrying  bool
		isTerminal  bool
	}{
		{"pending", "pending", true, false, false, false, false, false, false},
		{"running", "running", false, true, false, false, false, false, false},
		{"completed", "completed", false, false, true, false, false, false, true},
		{"failed", "failed", false, false, false, true, false, false, true},
		{"skipped", "skipped", false, false, false, false, true, false, true},
		{"retrying", "retrying", false, false, false, false, false, true, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			node := &NodeExecutionModel{Status: tt.status}

			assert.Equal(t, tt.isPending, node.IsPending())
			assert.Equal(t, tt.isRunning, node.IsRunning())
			assert.Equal(t, tt.isCompleted, node.IsCompleted())
			assert.Equal(t, tt.isFailed, node.IsFailed())
			assert.Equal(t, tt.isSkipped, node.IsSkipped())
			assert.Equal(t, tt.isRetrying, node.IsRetrying())
			assert.Equal(t, tt.isTerminal, node.IsTerminal())
		})
	}
}

func TestNodeExecutionModel_Duration(t *testing.T) {
	t.Run("with both timestamps", func(t *testing.T) {
		start := time.Now().Add(-2 * time.Minute)
		end := time.Now()
		node := &NodeExecutionModel{
			StartedAt:   &start,
			CompletedAt: &end,
		}

		duration := node.Duration()
		require.NotNil(t, duration)
		assert.True(t, *duration >= time.Minute && *duration <= 3*time.Minute)
	})

	t.Run("without timestamps", func(t *testing.T) {
		node := &NodeExecutionModel{}
		assert.Nil(t, node.Duration())
	})
}

func TestNodeExecutionModel_StateTransitions(t *testing.T) {
	t.Run("MarkStarted", func(t *testing.T) {
		node := &NodeExecutionModel{Status: "pending"}
		node.MarkStarted()

		assert.Equal(t, "running", node.Status)
		assert.NotNil(t, node.StartedAt)
	})

	t.Run("MarkCompleted", func(t *testing.T) {
		node := &NodeExecutionModel{Status: "running"}
		node.MarkCompleted()

		assert.Equal(t, "completed", node.Status)
		assert.NotNil(t, node.CompletedAt)
	})

	t.Run("MarkFailed", func(t *testing.T) {
		node := &NodeExecutionModel{Status: "running"}
		node.MarkFailed("node error")

		assert.Equal(t, "failed", node.Status)
		assert.NotNil(t, node.CompletedAt)
		assert.Equal(t, "node error", node.Error)
	})

	t.Run("MarkSkipped", func(t *testing.T) {
		node := &NodeExecutionModel{Status: "pending"}
		node.MarkSkipped()

		assert.Equal(t, "skipped", node.Status)
	})

	t.Run("MarkRetrying", func(t *testing.T) {
		node := &NodeExecutionModel{Status: "failed", RetryCount: 0}
		node.MarkRetrying()

		assert.Equal(t, "retrying", node.Status)
		assert.Equal(t, 1, node.RetryCount)
	})
}

// Test TriggerModel Helper Methods

func TestTriggerModel_TypeCheckers(t *testing.T) {
	tests := []struct {
		name        string
		triggerType string
		isManual    bool
		isCron      bool
		isWebhook   bool
		isEvent     bool
		isInterval  bool
	}{
		{"manual", "manual", true, false, false, false, false},
		{"cron", "cron", false, true, false, false, false},
		{"webhook", "webhook", false, false, true, false, false},
		{"event", "event", false, false, false, true, false},
		{"interval", "interval", false, false, false, false, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			trigger := &TriggerModel{Type: tt.triggerType}

			assert.Equal(t, tt.isManual, trigger.IsManual())
			assert.Equal(t, tt.isCron, trigger.IsCron())
			assert.Equal(t, tt.isWebhook, trigger.IsWebhook())
			assert.Equal(t, tt.isEvent, trigger.IsEvent())
			assert.Equal(t, tt.isInterval, trigger.IsInterval())
		})
	}
}

func TestTriggerModel_MarkTriggered(t *testing.T) {
	trigger := &TriggerModel{}

	trigger.MarkTriggered()

	assert.NotNil(t, trigger.LastTriggeredAt)
	assert.WithinDuration(t, time.Now(), *trigger.LastTriggeredAt, time.Second)
}

func TestTriggerModel_GetCronExpression(t *testing.T) {
	t.Run("with expression", func(t *testing.T) {
		trigger := &TriggerModel{
			Type: "cron",
			Config: JSONBMap{
				"expression": "0 0 * * * *",
			},
		}

		expr := trigger.GetCronExpression()
		assert.Equal(t, "0 0 * * * *", expr)
	})

	t.Run("without expression", func(t *testing.T) {
		trigger := &TriggerModel{
			Type:   "cron",
			Config: JSONBMap{},
		}

		expr := trigger.GetCronExpression()
		assert.Empty(t, expr)
	})

	t.Run("wrong type", func(t *testing.T) {
		trigger := &TriggerModel{
			Type: "webhook",
			Config: JSONBMap{
				"expression": "0 0 * * * *",
			},
		}

		expr := trigger.GetCronExpression()
		assert.Empty(t, expr)
	})
}

func TestTriggerModel_GetWebhookURL(t *testing.T) {
	t.Run("with url", func(t *testing.T) {
		trigger := &TriggerModel{
			Type: "webhook",
			Config: JSONBMap{
				"url": "/webhook/test",
			},
		}

		url := trigger.GetWebhookURL()
		assert.Equal(t, "/webhook/test", url)
	})

	t.Run("without url", func(t *testing.T) {
		trigger := &TriggerModel{
			Type:   "webhook",
			Config: JSONBMap{},
		}

		url := trigger.GetWebhookURL()
		assert.Empty(t, url)
	})

	t.Run("wrong type", func(t *testing.T) {
		trigger := &TriggerModel{
			Type: "cron",
			Config: JSONBMap{
				"url": "/webhook/test",
			},
		}

		url := trigger.GetWebhookURL()
		assert.Empty(t, url)
	})
}

func TestTriggerModel_GetIntervalDuration(t *testing.T) {
	t.Run("with seconds", func(t *testing.T) {
		trigger := &TriggerModel{
			Type: "interval",
			Config: JSONBMap{
				"seconds": float64(300), // 5 minutes
			},
		}

		duration := trigger.GetIntervalDuration()
		assert.Equal(t, 5*time.Minute, duration)
	})

	t.Run("with different seconds", func(t *testing.T) {
		trigger := &TriggerModel{
			Type: "interval",
			Config: JSONBMap{
				"seconds": float64(30),
			},
		}

		duration := trigger.GetIntervalDuration()
		assert.Equal(t, 30*time.Second, duration)
	})

	t.Run("without seconds", func(t *testing.T) {
		trigger := &TriggerModel{
			Type:   "interval",
			Config: JSONBMap{},
		}

		duration := trigger.GetIntervalDuration()
		assert.Equal(t, time.Duration(0), duration)
	})

	t.Run("wrong type", func(t *testing.T) {
		trigger := &TriggerModel{
			Type: "cron",
			Config: JSONBMap{
				"seconds": float64(30),
			},
		}

		duration := trigger.GetIntervalDuration()
		assert.Equal(t, time.Duration(0), duration)
	})
}

// Test WorkflowModel Helper Methods

func TestWorkflowModel_StatusCheckers(t *testing.T) {
	t.Run("active workflow", func(t *testing.T) {
		workflow := &WorkflowModel{Status: "active"}

		assert.True(t, workflow.IsActive())
		assert.False(t, workflow.IsDraft())
		assert.False(t, workflow.IsDeleted())
	})

	t.Run("draft workflow", func(t *testing.T) {
		workflow := &WorkflowModel{Status: "draft"}

		assert.False(t, workflow.IsActive())
		assert.True(t, workflow.IsDraft())
		assert.False(t, workflow.IsDeleted())
	})

	t.Run("soft-deleted workflow", func(t *testing.T) {
		deletedTime := time.Now()
		workflow := &WorkflowModel{
			Status:    "deleted",
			DeletedAt: &deletedTime,
		}

		assert.False(t, workflow.IsActive())
		assert.False(t, workflow.IsDraft())
		assert.True(t, workflow.IsDeleted())
	})

	t.Run("not deleted workflow", func(t *testing.T) {
		workflow := &WorkflowModel{
			Status:    "active",
			DeletedAt: nil,
		}

		assert.False(t, workflow.IsDeleted())
	})
}

// Test EdgeModel Helper Methods

func TestEdgeModel_IsConditional(t *testing.T) {
	t.Run("with condition", func(t *testing.T) {
		edge := &EdgeModel{
			Condition: JSONBMap{
				"expression": "output.success == true",
			},
		}

		assert.True(t, edge.IsConditional())
	})

	t.Run("without condition", func(t *testing.T) {
		edge := &EdgeModel{
			Condition: JSONBMap{},
		}

		assert.False(t, edge.IsConditional())
	})

	t.Run("with nil condition", func(t *testing.T) {
		edge := &EdgeModel{}

		assert.False(t, edge.IsConditional())
	})
}

// Test EventModel Helper Methods

func TestEventModel_TypeCheckers(t *testing.T) {
	t.Run("workflow event", func(t *testing.T) {
		event := &EventModel{
			EventType: "execution.started",
		}

		assert.True(t, event.IsWorkflowEvent())
		assert.False(t, event.IsNodeEvent())
	})

	t.Run("node event", func(t *testing.T) {
		event := &EventModel{
			EventType: "node.completed",
		}

		assert.False(t, event.IsWorkflowEvent())
		assert.True(t, event.IsNodeEvent())
	})

	t.Run("other event", func(t *testing.T) {
		event := &EventModel{
			EventType: "system.error",
		}

		assert.False(t, event.IsWorkflowEvent())
		assert.False(t, event.IsNodeEvent())
	})
}

// Test FileModel Helper Methods

func TestFileModel_IsExpired(t *testing.T) {
	t.Run("with past expiration", func(t *testing.T) {
		pastTime := time.Now().Add(-1 * time.Hour)
		file := &FileModel{
			ExpiresAt: &pastTime,
		}

		assert.True(t, file.IsExpired())
	})

	t.Run("with future expiration", func(t *testing.T) {
		futureTime := time.Now().Add(1 * time.Hour)
		file := &FileModel{
			ExpiresAt: &futureTime,
		}

		assert.False(t, file.IsExpired())
	})

	t.Run("without expiration", func(t *testing.T) {
		file := &FileModel{}

		assert.False(t, file.IsExpired())
	})
}

// Test NodeModel Helper Methods

func TestNodeModel_GetPosition(t *testing.T) {
	t.Run("with position", func(t *testing.T) {
		node := &NodeModel{
			Position: JSONBMap{
				"x": float64(100),
				"y": float64(200),
			},
		}

		x, y := node.GetPosition()
		assert.Equal(t, float64(100), x)
		assert.Equal(t, float64(200), y)
	})

	t.Run("without position", func(t *testing.T) {
		node := &NodeModel{}

		x, y := node.GetPosition()
		assert.Equal(t, float64(0), x)
		assert.Equal(t, float64(0), y)
	})
}

func TestNodeModel_SetPosition(t *testing.T) {
	node := &NodeModel{}

	node.SetPosition(150, 250)

	assert.Equal(t, 150, node.Position.GetInt("x"))
	assert.Equal(t, 250, node.Position.GetInt("y"))
}

// Test JSONBMap.Get method

func TestJSONBMap_Get(t *testing.T) {
	data := JSONBMap{
		"string": "value",
		"number": float64(42),
		"bool":   true,
	}

	t.Run("existing key", func(t *testing.T) {
		value, exists := data.Get("string")
		assert.True(t, exists)
		assert.Equal(t, "value", value)
	})

	t.Run("missing key", func(t *testing.T) {
		value, exists := data.Get("missing")
		assert.False(t, exists)
		assert.Nil(t, value)
	})
}
