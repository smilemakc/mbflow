package executor

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/smilemakc/mbflow/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Helper function to create a mock node
func createMockNode(nodeType domain.NodeType, config map[string]any) domain.Node {
	return domain.RestoreNode(
		uuid.New(),
		nodeType,
		"test-node",
		config,
	)
}

// Helper function to create NodeExecutionInputs from variables
func createNodeInputs(variables *domain.VariableSet) *NodeExecutionInputs {
	globalContext := domain.NewVariableSet(nil)
	globalContext.SetReadOnly(true)

	return &NodeExecutionInputs{
		Variables:     variables,
		GlobalContext: globalContext,
		ParentOutputs: make(map[uuid.UUID]*domain.VariableSet),
		ExecutionID:   uuid.New(),
		WorkflowID:    uuid.New(),
	}
}

// TestTransformNodeExecutor tests the TransformNodeExecutor
func TestTransformNodeExecutor(t *testing.T) {
	t.Run("Simple expression transformation", func(t *testing.T) {
		executor := NewTransformNodeExecutor()
		config := map[string]any{
			"transformations": map[string]interface{}{
				"result": "a + b",
			},
		}
		node := createMockNode(domain.NodeTypeTransform, config)
		variables := domain.NewVariableSet(nil)

		_ = variables.Set("a", 10)
		_ = variables.Set("b", 20)

		result, err := executor.Execute(context.Background(), node, createNodeInputs(variables))

		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, 30, result["result"])
	})

	t.Run("String concatenation", func(t *testing.T) {
		executor := NewTransformNodeExecutor()
		config := map[string]any{
			"transformations": map[string]interface{}{
				"fullname": "first + ' ' + last",
			},
		}
		node := createMockNode(domain.NodeTypeTransform, config)
		variables := domain.NewVariableSet(nil)

		_ = variables.Set("first", "John")
		_ = variables.Set("last", "Doe")

		result, err := executor.Execute(context.Background(), node, createNodeInputs(variables))

		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, "John Doe", result["fullname"])
	})

	t.Run("Multiple transformations", func(t *testing.T) {
		executor := NewTransformNodeExecutor()
		config := map[string]any{
			"transformations": map[string]interface{}{
				"sum":     "a + b",
				"product": "a * b",
			},
		}
		node := createMockNode(domain.NodeTypeTransform, config)
		variables := domain.NewVariableSet(nil)

		_ = variables.Set("a", 5)
		_ = variables.Set("b", 3)

		result, err := executor.Execute(context.Background(), node, createNodeInputs(variables))

		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, 8, result["sum"])
		assert.Equal(t, 15, result["product"])
	})

	t.Run("Invalid expression returns error", func(t *testing.T) {
		executor := NewTransformNodeExecutor()
		config := map[string]any{
			"transformations": map[string]interface{}{
				"result": "invalid syntax +++",
			},
		}
		node := createMockNode(domain.NodeTypeTransform, config)
		variables := domain.NewVariableSet(nil)

		result, err := executor.Execute(context.Background(), node, createNodeInputs(variables))

		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid syntax")
		assert.Nil(t, result)
	})
}

// TestHTTPNodeExecutor tests the HTTPNodeExecutor
func TestHTTPNodeExecutor(t *testing.T) {
	t.Run("Successful GET request", func(t *testing.T) {
		// Create test server
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "GET", r.Method)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]string{"message": "success"})
		}))
		defer server.Close()

		executor := NewHTTPNodeExecutor()
		config := map[string]any{
			"url":    server.URL,
			"method": "GET",
		}
		node := createMockNode(domain.NodeTypeHTTP, config)
		variables := domain.NewVariableSet(nil)

		result, err := executor.Execute(context.Background(), node, createNodeInputs(variables))

		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, "success", result["message"])
		assert.Equal(t, 200, result["status_code"])
	})

	t.Run("POST request with body", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "POST", r.Method)

			var body map[string]interface{}
			json.NewDecoder(r.Body).Decode(&body)
			assert.Equal(t, "test", body["key"])

			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(map[string]string{"status": "created"})
		}))
		defer server.Close()

		executor := NewHTTPNodeExecutor()
		config := map[string]any{
			"url":    server.URL,
			"method": "POST",
			"body": map[string]interface{}{
				"key": "test",
			},
		}
		node := createMockNode(domain.NodeTypeHTTP, config)
		variables := domain.NewVariableSet(nil)

		result, err := executor.Execute(context.Background(), node, createNodeInputs(variables))

		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, "created", result["status"])
	})

	t.Run("Request with headers", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "Bearer token123", r.Header.Get("Authorization"))
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]string{"authenticated": "true"})
		}))
		defer server.Close()

		executor := NewHTTPNodeExecutor()
		config := map[string]any{
			"url":    server.URL,
			"method": "GET",
			"headers": map[string]interface{}{
				"Authorization": "Bearer token123",
			},
		}
		node := createMockNode(domain.NodeTypeHTTP, config)
		variables := domain.NewVariableSet(nil)

		result, err := executor.Execute(context.Background(), node, createNodeInputs(variables))

		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, "true", result["authenticated"])
	})

	t.Run("HTTP error status returns error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(map[string]string{"error": "not found"})
		}))
		defer server.Close()

		executor := NewHTTPNodeExecutor()
		config := map[string]any{
			"url": server.URL,
		}
		node := createMockNode(domain.NodeTypeHTTP, config)
		variables := domain.NewVariableSet(nil)

		result, err := executor.Execute(context.Background(), node, createNodeInputs(variables))

		require.Error(t, err)
		require.NotNil(t, result)
		assert.Equal(t, 404, result["status_code"])
	})

	t.Run("Missing URL returns error", func(t *testing.T) {
		executor := NewHTTPNodeExecutor()
		config := map[string]any{}
		node := createMockNode(domain.NodeTypeHTTP, config)
		variables := domain.NewVariableSet(nil)

		result, err := executor.Execute(context.Background(), node, createNodeInputs(variables))

		require.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "url not specified")
	})
}

// TestJSONParserExecutor tests the JSONParserExecutor
func TestJSONParserExecutor(t *testing.T) {
	t.Run("Parse JSON string", func(t *testing.T) {
		executor := NewJSONParserExecutor()
		config := map[string]any{
			"input_key": "json_data",
		}
		node := createMockNode(domain.NodeTypeJSONParser, config)
		variables := domain.NewVariableSet(nil)

		_ = variables.Set("json_data", `{"name": "John", "age": 30}`)

		result, err := executor.Execute(context.Background(), node, createNodeInputs(variables))

		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, "John", result["name"])
		assert.Equal(t, float64(30), result["age"])
	})

	t.Run("Parse JSON bytes", func(t *testing.T) {
		executor := NewJSONParserExecutor()
		config := map[string]any{
			"input_key": "json_data",
		}
		node := createMockNode(domain.NodeTypeJSONParser, config)
		variables := domain.NewVariableSet(nil)

		_ = variables.Set("json_data", []byte(`{"status": "ok"}`))

		result, err := executor.Execute(context.Background(), node, createNodeInputs(variables))

		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, "ok", result["status"])
	})

	t.Run("Already parsed map passes through", func(t *testing.T) {
		executor := NewJSONParserExecutor()
		config := map[string]any{
			"input_key": "json_data",
		}
		node := createMockNode(domain.NodeTypeJSONParser, config)
		variables := domain.NewVariableSet(nil)

		originalMap := map[string]any{"key": "value"}
		_ = variables.Set("json_data", originalMap)

		result, err := executor.Execute(context.Background(), node, createNodeInputs(variables))

		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, originalMap, result)
	})

	t.Run("Invalid JSON returns error", func(t *testing.T) {
		executor := NewJSONParserExecutor()
		config := map[string]any{
			"input_key": "json_data",
		}
		node := createMockNode(domain.NodeTypeJSONParser, config)
		variables := domain.NewVariableSet(nil)

		_ = variables.Set("json_data", "not valid json")

		result, err := executor.Execute(context.Background(), node, createNodeInputs(variables))

		require.Error(t, err)
		assert.Nil(t, result)
	})

	t.Run("Missing input key returns error", func(t *testing.T) {
		executor := NewJSONParserExecutor()
		config := map[string]any{
			"input_key": "missing_key",
		}
		node := createMockNode(domain.NodeTypeJSONParser, config)
		variables := domain.NewVariableSet(nil)

		result, err := executor.Execute(context.Background(), node, createNodeInputs(variables))

		require.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "not found")
	})
}

// TestDataMergerExecutor tests the DataMergerExecutor
func TestDataMergerExecutor(t *testing.T) {
	t.Run("Merge with overwrite strategy", func(t *testing.T) {
		executor := NewDataMergerExecutor()
		config := map[string]any{
			"sources":  []interface{}{"data1", "data2"},
			"strategy": "overwrite",
		}
		node := createMockNode(domain.NodeTypeDataMerger, config)
		variables := domain.NewVariableSet(nil)

		_ = variables.Set("data1", map[string]any{"a": 1, "b": 2})
		_ = variables.Set("data2", map[string]any{"b": 3, "c": 4})

		result, err := executor.Execute(context.Background(), node, createNodeInputs(variables))

		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, 1, result["a"])
		assert.Equal(t, 3, result["b"]) // Overwritten by data2
		assert.Equal(t, 4, result["c"])
	})

	t.Run("Merge with keep_first strategy", func(t *testing.T) {
		executor := NewDataMergerExecutor()
		config := map[string]any{
			"sources":  []interface{}{"data1", "data2"},
			"strategy": "keep_first",
		}
		node := createMockNode(domain.NodeTypeDataMerger, config)
		variables := domain.NewVariableSet(nil)

		_ = variables.Set("data1", map[string]any{"a": 1, "b": 2})
		_ = variables.Set("data2", map[string]any{"b": 3, "c": 4})

		result, err := executor.Execute(context.Background(), node, createNodeInputs(variables))

		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, 1, result["a"])
		assert.Equal(t, 2, result["b"]) // Kept from data1
		assert.Equal(t, 4, result["c"])
	})

	t.Run("Merge with collect strategy", func(t *testing.T) {
		executor := NewDataMergerExecutor()
		config := map[string]any{
			"sources":  []interface{}{"data1", "data2"},
			"strategy": "collect",
		}
		node := createMockNode(domain.NodeTypeDataMerger, config)
		variables := domain.NewVariableSet(nil)

		_ = variables.Set("data1", map[string]any{"a": 1})
		_ = variables.Set("data2", map[string]any{"a": 2})

		result, err := executor.Execute(context.Background(), node, createNodeInputs(variables))

		require.NoError(t, err)
		require.NotNil(t, result)

		collected, ok := result["a"].([]any)
		require.True(t, ok)
		assert.Len(t, collected, 2)
		assert.Contains(t, collected, 1)
		assert.Contains(t, collected, 2)
	})

	t.Run("Missing sources returns error", func(t *testing.T) {
		executor := NewDataMergerExecutor()
		config := map[string]any{}
		node := createMockNode(domain.NodeTypeDataMerger, config)
		variables := domain.NewVariableSet(nil)

		result, err := executor.Execute(context.Background(), node, createNodeInputs(variables))

		require.Error(t, err)
		assert.Nil(t, result)
	})
}

// TestDataAggregatorExecutor tests the DataAggregatorExecutor
func TestDataAggregatorExecutor(t *testing.T) {
	t.Run("Sum aggregation", func(t *testing.T) {
		executor := NewDataAggregatorExecutor()
		config := map[string]any{
			"input_key": "numbers",
			"function":  "sum",
		}
		node := createMockNode(domain.NodeTypeDataAggregator, config)
		variables := domain.NewVariableSet(nil)

		_ = variables.Set("numbers", []any{1.0, 2.0, 3.0, 4.0, 5.0})

		result, err := executor.Execute(context.Background(), node, createNodeInputs(variables))

		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, 15.0, result["result"])
		assert.Equal(t, 5, result["count"])
	})

	t.Run("Count aggregation", func(t *testing.T) {
		executor := NewDataAggregatorExecutor()
		config := map[string]any{
			"input_key": "items",
			"function":  "count",
		}
		node := createMockNode(domain.NodeTypeDataAggregator, config)
		variables := domain.NewVariableSet(nil)

		_ = variables.Set("items", []any{"a", "b", "c"})

		result, err := executor.Execute(context.Background(), node, createNodeInputs(variables))

		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, 3, result["result"])
	})

	t.Run("Average aggregation", func(t *testing.T) {
		executor := NewDataAggregatorExecutor()
		config := map[string]any{
			"input_key": "numbers",
			"function":  "avg",
		}
		node := createMockNode(domain.NodeTypeDataAggregator, config)
		variables := domain.NewVariableSet(nil)

		_ = variables.Set("numbers", []any{10.0, 20.0, 30.0})

		result, err := executor.Execute(context.Background(), node, createNodeInputs(variables))

		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, 20.0, result["result"])
	})

	t.Run("Min aggregation", func(t *testing.T) {
		executor := NewDataAggregatorExecutor()
		config := map[string]any{
			"input_key": "numbers",
			"function":  "min",
		}
		node := createMockNode(domain.NodeTypeDataAggregator, config)
		variables := domain.NewVariableSet(nil)

		_ = variables.Set("numbers", []any{5.0, 2.0, 8.0, 1.0})

		result, err := executor.Execute(context.Background(), node, createNodeInputs(variables))

		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, 1.0, result["result"])
	})

	t.Run("Max aggregation", func(t *testing.T) {
		executor := NewDataAggregatorExecutor()
		config := map[string]any{
			"input_key": "numbers",
			"function":  "max",
		}
		node := createMockNode(domain.NodeTypeDataAggregator, config)
		variables := domain.NewVariableSet(nil)

		_ = variables.Set("numbers", []any{5.0, 2.0, 8.0, 1.0})

		result, err := executor.Execute(context.Background(), node, createNodeInputs(variables))

		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, 8.0, result["result"])
	})

	t.Run("Collect aggregation", func(t *testing.T) {
		executor := NewDataAggregatorExecutor()
		config := map[string]any{
			"input_key": "items",
			"function":  "collect",
		}
		node := createMockNode(domain.NodeTypeDataAggregator, config)
		variables := domain.NewVariableSet(nil)

		items := []any{"a", "b", "c"}
		_ = variables.Set("items", items)

		result, err := executor.Execute(context.Background(), node, createNodeInputs(variables))

		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, items, result["result"])
	})

	t.Run("Non-array input returns error", func(t *testing.T) {
		executor := NewDataAggregatorExecutor()
		config := map[string]any{
			"input_key": "not_array",
			"function":  "sum",
		}
		node := createMockNode(domain.NodeTypeDataAggregator, config)
		variables := domain.NewVariableSet(nil)

		_ = variables.Set("not_array", "string value")

		result, err := executor.Execute(context.Background(), node, createNodeInputs(variables))

		require.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "not an array")
	})

	t.Run("Unknown function returns error", func(t *testing.T) {
		executor := NewDataAggregatorExecutor()
		config := map[string]any{
			"input_key": "numbers",
			"function":  "unknown",
		}
		node := createMockNode(domain.NodeTypeDataAggregator, config)
		variables := domain.NewVariableSet(nil)

		_ = variables.Set("numbers", []any{1.0, 2.0})

		result, err := executor.Execute(context.Background(), node, createNodeInputs(variables))

		require.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "unknown aggregation function")
	})

	// Test field extraction mode
	t.Run("Field extraction from simple variables", func(t *testing.T) {
		executor := NewDataAggregatorExecutor()
		config := map[string]any{
			"fields": map[string]interface{}{
				"name":   "user_name",
				"email":  "user_email",
				"status": "user_status",
			},
		}
		node := createMockNode(domain.NodeTypeDataAggregator, config)
		variables := domain.NewVariableSet(nil)

		_ = variables.Set("user_name", "John Doe")
		_ = variables.Set("user_email", "john@example.com")
		_ = variables.Set("user_status", "active")

		result, err := executor.Execute(context.Background(), node, createNodeInputs(variables))

		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, "John Doe", result["name"])
		assert.Equal(t, "john@example.com", result["email"])
		assert.Equal(t, "active", result["status"])
	})

	t.Run("Field extraction with nested paths", func(t *testing.T) {
		executor := NewDataAggregatorExecutor()
		config := map[string]any{
			"fields": map[string]interface{}{
				"repo_name":        "parsed_data.name",
				"stars":            "parsed_data.stargazers_count",
				"repo_description": "parsed_data.description",
			},
		}
		node := createMockNode(domain.NodeTypeDataAggregator, config)
		variables := domain.NewVariableSet(nil)

		_ = variables.Set("parsed_data", map[string]any{
			"name":             "go",
			"stargazers_count": 130991.0,
			"description":      "The Go programming language",
			"forks":            12345,
		})

		result, err := executor.Execute(context.Background(), node, createNodeInputs(variables))

		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, "go", result["repo_name"])
		assert.Equal(t, 130991.0, result["stars"])
		assert.Equal(t, "The Go programming language", result["repo_description"])
		assert.NotContains(t, result, "forks") // Only specified fields should be extracted
	})

	t.Run("Field extraction with missing source fields", func(t *testing.T) {
		executor := NewDataAggregatorExecutor()
		config := map[string]any{
			"fields": map[string]interface{}{
				"name":  "user_name",
				"email": "user_email",
			},
		}
		node := createMockNode(domain.NodeTypeDataAggregator, config)
		variables := domain.NewVariableSet(nil)

		_ = variables.Set("user_name", "John Doe")
		// user_email is missing

		result, err := executor.Execute(context.Background(), node, createNodeInputs(variables))

		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, "John Doe", result["name"])
		assert.NotContains(t, result, "email") // Missing fields should not be included
	})
}

// TestConditionalRouteExecutor tests the ConditionalRouteExecutor
func TestConditionalRouteExecutor(t *testing.T) {
	t.Run("First matching route is selected", func(t *testing.T) {
		executor := NewConditionalRouteExecutor()
		config := map[string]any{
			"routes": []interface{}{
				map[string]interface{}{
					"name":      "route_a",
					"condition": "value > 10",
				},
				map[string]interface{}{
					"name":      "route_b",
					"condition": "value > 5",
				},
			},
		}
		node := createMockNode(domain.NodeTypeConditionalRoute, config)
		variables := domain.NewVariableSet(nil)

		_ = variables.Set("value", 15)

		result, err := executor.Execute(context.Background(), node, createNodeInputs(variables))

		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, "route_a", result["selected_route"])
	})

	t.Run("Second route selected when first fails", func(t *testing.T) {
		executor := NewConditionalRouteExecutor()
		config := map[string]any{
			"routes": []interface{}{
				map[string]interface{}{
					"name":      "route_a",
					"condition": "value > 20",
				},
				map[string]interface{}{
					"name":      "route_b",
					"condition": "value > 5",
				},
			},
		}
		node := createMockNode(domain.NodeTypeConditionalRoute, config)
		variables := domain.NewVariableSet(nil)

		_ = variables.Set("value", 10)

		result, err := executor.Execute(context.Background(), node, createNodeInputs(variables))

		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, "route_b", result["selected_route"])
	})

	t.Run("Default route when no conditions match", func(t *testing.T) {
		executor := NewConditionalRouteExecutor()
		config := map[string]any{
			"routes": []interface{}{
				map[string]interface{}{
					"name":      "route_a",
					"condition": "value > 20",
				},
			},
			"default_route": "fallback",
		}
		node := createMockNode(domain.NodeTypeConditionalRoute, config)
		variables := domain.NewVariableSet(nil)

		_ = variables.Set("value", 5)

		result, err := executor.Execute(context.Background(), node, createNodeInputs(variables))

		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, "fallback", result["selected_route"])
	})

	t.Run("Default 'default' when no route matches and no default specified", func(t *testing.T) {
		executor := NewConditionalRouteExecutor()
		config := map[string]any{
			"routes": []interface{}{
				map[string]interface{}{
					"name":      "route_a",
					"condition": "value > 20",
				},
			},
		}
		node := createMockNode(domain.NodeTypeConditionalRoute, config)
		variables := domain.NewVariableSet(nil)

		_ = variables.Set("value", 5)

		result, err := executor.Execute(context.Background(), node, createNodeInputs(variables))

		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, "default", result["selected_route"])
	})

	t.Run("Expression condition route", func(t *testing.T) {
		executor := NewConditionalRouteExecutor()
		config := map[string]any{
			"routes": []interface{}{
				map[string]interface{}{
					"name":      "route_a",
					"condition": "ai_response['intent']['confidence'] > 0.5",
				},
				map[string]interface{}{
					"name":      "route_b",
					"condition": "value > 5",
				},
			},
		}

		node := createMockNode(domain.NodeTypeConditionalRoute, config)
		variables := domain.NewVariableSet(nil)
		_ = variables.Set("ai_response", map[string]any{
			"intent": map[string]any{
				"confidence": 0.7,
			},
		})

		result, err := executor.Execute(context.Background(), node, createNodeInputs(variables))

		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, "route_a", result["selected_route"])

	})
}

// TestScriptExecutorNode tests the ScriptExecutorNode
func TestScriptExecutorNode(t *testing.T) {
	t.Run("Execute simple expr script", func(t *testing.T) {
		executor := NewScriptExecutorNode()
		config := map[string]any{
			"script":   "a * 2 + b",
			"language": "expr",
		}
		node := createMockNode(domain.NodeTypeScriptExecutor, config)
		variables := domain.NewVariableSet(nil)

		_ = variables.Set("a", 5)
		_ = variables.Set("b", 3)

		result, err := executor.Execute(context.Background(), node, createNodeInputs(variables))

		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, 13, result["result"])
	})

	t.Run("Execute expr script with default language", func(t *testing.T) {
		executor := NewScriptExecutorNode()
		config := map[string]any{
			"script": "x + y",
		}
		node := createMockNode(domain.NodeTypeScriptExecutor, config)
		variables := domain.NewVariableSet(nil)

		_ = variables.Set("x", 10)
		_ = variables.Set("y", 20)

		result, err := executor.Execute(context.Background(), node, createNodeInputs(variables))

		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, 30, result["result"])
	})

	t.Run("Invalid script returns error", func(t *testing.T) {
		executor := NewScriptExecutorNode()
		config := map[string]any{
			"script": "invalid ++ syntax",
		}
		node := createMockNode(domain.NodeTypeScriptExecutor, config)
		variables := domain.NewVariableSet(nil)

		result, err := executor.Execute(context.Background(), node, createNodeInputs(variables))

		require.Error(t, err)
		assert.Nil(t, result)
	})

	t.Run("Unsupported language returns error", func(t *testing.T) {
		executor := NewScriptExecutorNode()
		config := map[string]any{
			"script":   "print('hello')",
			"language": "python",
		}
		node := createMockNode(domain.NodeTypeScriptExecutor, config)
		variables := domain.NewVariableSet(nil)

		result, err := executor.Execute(context.Background(), node, createNodeInputs(variables))

		require.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "unsupported script language")
	})

	t.Run("Missing script returns error", func(t *testing.T) {
		executor := NewScriptExecutorNode()
		config := map[string]any{}
		node := createMockNode(domain.NodeTypeScriptExecutor, config)
		variables := domain.NewVariableSet(nil)

		result, err := executor.Execute(context.Background(), node, createNodeInputs(variables))

		require.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "script not specified")
	})
}

// TestParallelNodeExecutor tests the ParallelNodeExecutor
func TestParallelNodeExecutor(t *testing.T) {
	t.Run("Returns empty map", func(t *testing.T) {
		executor := NewParallelNodeExecutor()
		node := createMockNode(domain.NodeTypeParallel, map[string]any{})
		variables := domain.NewVariableSet(nil)

		result, err := executor.Execute(context.Background(), node, createNodeInputs(variables))

		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Empty(t, result)
	})
}

// TestGetNestedValue tests the getNestedValue helper function
func TestGetNestedValue(t *testing.T) {
	t.Run("Simple key access", func(t *testing.T) {
		data := map[string]interface{}{
			"name": "John",
		}

		result := getNestedValue(data, "name")

		assert.Equal(t, "John", result)
	})

	t.Run("Nested key access", func(t *testing.T) {
		data := map[string]interface{}{
			"user": map[string]interface{}{
				"name":  "John",
				"email": "john@example.com",
			},
		}

		result := getNestedValue(data, "user.name")

		assert.Equal(t, "John", result)
	})

	t.Run("Deep nested key access", func(t *testing.T) {
		data := map[string]interface{}{
			"company": map[string]interface{}{
				"employee": map[string]interface{}{
					"details": map[string]interface{}{
						"name": "Alice",
					},
				},
			},
		}

		result := getNestedValue(data, "company.employee.details.name")

		assert.Equal(t, "Alice", result)
	})

	t.Run("Missing key returns nil", func(t *testing.T) {
		data := map[string]interface{}{
			"name": "John",
		}

		result := getNestedValue(data, "missing")

		assert.Nil(t, result)
	})

	t.Run("Missing nested key returns nil", func(t *testing.T) {
		data := map[string]interface{}{
			"user": map[string]interface{}{
				"name": "John",
			},
		}

		result := getNestedValue(data, "user.email")

		assert.Nil(t, result)
	})

	t.Run("Non-map intermediate value returns nil", func(t *testing.T) {
		data := map[string]interface{}{
			"user": "not a map",
		}

		result := getNestedValue(data, "user.name")

		assert.Nil(t, result)
	})
}
