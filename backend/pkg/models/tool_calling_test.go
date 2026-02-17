package models

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ==================== ToolCallMode Tests ====================

func TestToolCallMode_Constants(t *testing.T) {
	assert.Equal(t, ToolCallMode("auto"), ToolCallModeAuto)
	assert.Equal(t, ToolCallMode("manual"), ToolCallModeManual)
}

// ==================== DefaultToolCallConfig Tests ====================

func TestDefaultToolCallConfig_Success(t *testing.T) {
	config := DefaultToolCallConfig()

	require.NotNil(t, config)
	assert.Equal(t, ToolCallModeManual, config.Mode)
	assert.Equal(t, 10, config.MaxIterations)
	assert.Equal(t, 30, config.TimeoutPerTool)
	assert.Equal(t, 300, config.TotalTimeout)
	assert.False(t, config.StopOnToolFailure)
}

func TestDefaultToolCallConfig_Independent(t *testing.T) {
	// Verify each call returns a new instance
	config1 := DefaultToolCallConfig()
	config2 := DefaultToolCallConfig()

	require.NotNil(t, config1)
	require.NotNil(t, config2)

	// Modify config1
	config1.MaxIterations = 20
	config1.Mode = ToolCallModeAuto

	// Verify config2 is not affected
	assert.Equal(t, 10, config2.MaxIterations)
	assert.Equal(t, ToolCallModeManual, config2.Mode)
}

// ==================== ToolCallConfig JSON Tests ====================

func TestToolCallConfig_JSONMarshaling(t *testing.T) {
	config := &ToolCallConfig{
		Mode:              ToolCallModeAuto,
		MaxIterations:     15,
		TimeoutPerTool:    60,
		TotalTimeout:      600,
		StopOnToolFailure: true,
	}

	data, err := json.Marshal(config)
	require.NoError(t, err)

	var unmarshaled ToolCallConfig
	err = json.Unmarshal(data, &unmarshaled)
	require.NoError(t, err)

	assert.Equal(t, config.Mode, unmarshaled.Mode)
	assert.Equal(t, config.MaxIterations, unmarshaled.MaxIterations)
	assert.Equal(t, config.TimeoutPerTool, unmarshaled.TimeoutPerTool)
	assert.Equal(t, config.TotalTimeout, unmarshaled.TotalTimeout)
	assert.Equal(t, config.StopOnToolFailure, unmarshaled.StopOnToolFailure)
}

func TestToolCallConfig_AutoMode(t *testing.T) {
	config := &ToolCallConfig{
		Mode:              ToolCallModeAuto,
		MaxIterations:     5,
		TimeoutPerTool:    10,
		TotalTimeout:      100,
		StopOnToolFailure: true,
	}

	data, err := json.Marshal(config)
	require.NoError(t, err)

	assert.Contains(t, string(data), `"mode":"auto"`)
	assert.Contains(t, string(data), `"max_iterations":5`)
}

func TestToolCallConfig_ManualMode(t *testing.T) {
	config := &ToolCallConfig{
		Mode:              ToolCallModeManual,
		MaxIterations:     10,
		TimeoutPerTool:    30,
		TotalTimeout:      300,
		StopOnToolFailure: false,
	}

	data, err := json.Marshal(config)
	require.NoError(t, err)

	assert.Contains(t, string(data), `"mode":"manual"`)
}

// ==================== LLMMessage Tests ====================

func TestLLMMessage_UserMessage(t *testing.T) {
	msg := LLMMessage{
		Role:    "user",
		Content: "Hello, how are you?",
	}

	data, err := json.Marshal(msg)
	require.NoError(t, err)

	var unmarshaled LLMMessage
	err = json.Unmarshal(data, &unmarshaled)
	require.NoError(t, err)

	assert.Equal(t, "user", unmarshaled.Role)
	assert.Equal(t, "Hello, how are you?", unmarshaled.Content)
}

func TestLLMMessage_AssistantMessage(t *testing.T) {
	msg := LLMMessage{
		Role:    "assistant",
		Content: "I'm doing well, thank you!",
	}

	data, err := json.Marshal(msg)
	require.NoError(t, err)

	var unmarshaled LLMMessage
	err = json.Unmarshal(data, &unmarshaled)
	require.NoError(t, err)

	assert.Equal(t, "assistant", unmarshaled.Role)
	assert.Equal(t, "I'm doing well, thank you!", unmarshaled.Content)
}

func TestLLMMessage_AssistantWithToolCalls(t *testing.T) {
	msg := LLMMessage{
		Role:    "assistant",
		Content: "",
		ToolCalls: []LLMToolCall{
			{
				ID:   "call_123",
				Type: "function",
				Function: LLMFunctionCall{
					Name:      "get_weather",
					Arguments: `{"location":"Paris"}`,
				},
			},
		},
	}

	data, err := json.Marshal(msg)
	require.NoError(t, err)

	var unmarshaled LLMMessage
	err = json.Unmarshal(data, &unmarshaled)
	require.NoError(t, err)

	assert.Equal(t, "assistant", unmarshaled.Role)
	assert.Len(t, unmarshaled.ToolCalls, 1)
	assert.Equal(t, "call_123", unmarshaled.ToolCalls[0].ID)
	assert.Equal(t, "get_weather", unmarshaled.ToolCalls[0].Function.Name)
}

func TestLLMMessage_ToolMessage(t *testing.T) {
	msg := LLMMessage{
		Role:       "tool",
		Content:    `{"temperature": 22, "conditions": "sunny"}`,
		ToolCallID: "call_123",
		Name:       "get_weather",
	}

	data, err := json.Marshal(msg)
	require.NoError(t, err)

	var unmarshaled LLMMessage
	err = json.Unmarshal(data, &unmarshaled)
	require.NoError(t, err)

	assert.Equal(t, "tool", unmarshaled.Role)
	assert.Equal(t, "call_123", unmarshaled.ToolCallID)
	assert.Equal(t, "get_weather", unmarshaled.Name)
	assert.Contains(t, unmarshaled.Content, "temperature")
}

func TestLLMMessage_SystemMessage(t *testing.T) {
	msg := LLMMessage{
		Role:    "system",
		Content: "You are a helpful assistant.",
	}

	data, err := json.Marshal(msg)
	require.NoError(t, err)

	var unmarshaled LLMMessage
	err = json.Unmarshal(data, &unmarshaled)
	require.NoError(t, err)

	assert.Equal(t, "system", unmarshaled.Role)
	assert.Equal(t, "You are a helpful assistant.", unmarshaled.Content)
}

func TestLLMMessage_WithMetadata(t *testing.T) {
	msg := LLMMessage{
		Role:    "assistant",
		Content: "Response",
		Metadata: map[string]any{
			"model":       "gpt-4",
			"temperature": 0.7,
		},
	}

	data, err := json.Marshal(msg)
	require.NoError(t, err)

	var unmarshaled LLMMessage
	err = json.Unmarshal(data, &unmarshaled)
	require.NoError(t, err)

	assert.NotNil(t, unmarshaled.Metadata)
	assert.Equal(t, "gpt-4", unmarshaled.Metadata["model"])
}

// ==================== FunctionType Tests ====================

func TestFunctionType_Constants(t *testing.T) {
	assert.Equal(t, FunctionType("builtin"), FunctionTypeBuiltin)
	assert.Equal(t, FunctionType("sub_workflow"), FunctionTypeSubWorkflow)
	assert.Equal(t, FunctionType("custom_code"), FunctionTypeCustomCode)
	assert.Equal(t, FunctionType("openapi"), FunctionTypeOpenAPI)
}

// ==================== FunctionDefinition Tests ====================

func TestFunctionDefinition_Builtin(t *testing.T) {
	def := FunctionDefinition{
		Type:        FunctionTypeBuiltin,
		Name:        "http_request",
		Description: "Make HTTP request",
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"url": map[string]any{"type": "string"},
			},
		},
		BuiltinName: "http",
	}

	data, err := json.Marshal(def)
	require.NoError(t, err)

	var unmarshaled FunctionDefinition
	err = json.Unmarshal(data, &unmarshaled)
	require.NoError(t, err)

	assert.Equal(t, FunctionTypeBuiltin, unmarshaled.Type)
	assert.Equal(t, "http_request", unmarshaled.Name)
	assert.Equal(t, "http", unmarshaled.BuiltinName)
}

func TestFunctionDefinition_SubWorkflow(t *testing.T) {
	def := FunctionDefinition{
		Type:        FunctionTypeSubWorkflow,
		Name:        "process_order",
		Description: "Process customer order",
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"order_id": map[string]any{"type": "string"},
			},
		},
		WorkflowID: "wf_123",
		InputMapping: map[string]string{
			"order_id": "input.order.id",
		},
		OutputExtractor: ".result.status",
	}

	data, err := json.Marshal(def)
	require.NoError(t, err)

	var unmarshaled FunctionDefinition
	err = json.Unmarshal(data, &unmarshaled)
	require.NoError(t, err)

	assert.Equal(t, FunctionTypeSubWorkflow, unmarshaled.Type)
	assert.Equal(t, "wf_123", unmarshaled.WorkflowID)
	assert.NotNil(t, unmarshaled.InputMapping)
	assert.Equal(t, "input.order.id", unmarshaled.InputMapping["order_id"])
	assert.Equal(t, ".result.status", unmarshaled.OutputExtractor)
}

func TestFunctionDefinition_CustomCode_JavaScript(t *testing.T) {
	def := FunctionDefinition{
		Type:        FunctionTypeCustomCode,
		Name:        "calculate_total",
		Description: "Calculate order total",
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"items": map[string]any{"type": "array"},
			},
		},
		Language: "javascript",
		Code:     "return items.reduce((sum, item) => sum + item.price, 0);",
	}

	data, err := json.Marshal(def)
	require.NoError(t, err)

	var unmarshaled FunctionDefinition
	err = json.Unmarshal(data, &unmarshaled)
	require.NoError(t, err)

	assert.Equal(t, FunctionTypeCustomCode, unmarshaled.Type)
	assert.Equal(t, "javascript", unmarshaled.Language)
	assert.Contains(t, unmarshaled.Code, "reduce")
}

func TestFunctionDefinition_CustomCode_Python(t *testing.T) {
	def := FunctionDefinition{
		Type:        FunctionTypeCustomCode,
		Name:        "analyze_data",
		Description: "Analyze dataset",
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"data": map[string]any{"type": "array"},
			},
		},
		Language: "python",
		Code:     "return sum(data) / len(data)",
	}

	data, err := json.Marshal(def)
	require.NoError(t, err)

	var unmarshaled FunctionDefinition
	err = json.Unmarshal(data, &unmarshaled)
	require.NoError(t, err)

	assert.Equal(t, FunctionTypeCustomCode, unmarshaled.Type)
	assert.Equal(t, "python", unmarshaled.Language)
	assert.Contains(t, unmarshaled.Code, "sum")
}

func TestFunctionDefinition_OpenAPI(t *testing.T) {
	def := FunctionDefinition{
		Type:        FunctionTypeOpenAPI,
		Name:        "create_user",
		Description: "Create new user via API",
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"name":  map[string]any{"type": "string"},
				"email": map[string]any{"type": "string"},
			},
		},
		OpenAPISpec: "https://api.example.com/openapi.json",
		OperationID: "createUser",
		BaseURL:     "https://api.example.com",
		AuthConfig: map[string]any{
			"type":  "bearer",
			"token": "sk-123",
		},
	}

	data, err := json.Marshal(def)
	require.NoError(t, err)

	var unmarshaled FunctionDefinition
	err = json.Unmarshal(data, &unmarshaled)
	require.NoError(t, err)

	assert.Equal(t, FunctionTypeOpenAPI, unmarshaled.Type)
	assert.Equal(t, "https://api.example.com/openapi.json", unmarshaled.OpenAPISpec)
	assert.Equal(t, "createUser", unmarshaled.OperationID)
	assert.Equal(t, "https://api.example.com", unmarshaled.BaseURL)
	assert.NotNil(t, unmarshaled.AuthConfig)
	assert.Equal(t, "bearer", unmarshaled.AuthConfig["type"])
}

// ==================== ToolExecutionResult Tests ====================

func TestToolExecutionResult_Success(t *testing.T) {
	result := ToolExecutionResult{
		ToolCallID:    "call_123",
		FunctionName:  "get_weather",
		Result:        map[string]any{"temperature": 22, "conditions": "sunny"},
		Error:         "",
		ExecutionTime: 150,
		Metadata: map[string]any{
			"cache_hit": false,
		},
	}

	data, err := json.Marshal(result)
	require.NoError(t, err)

	var unmarshaled ToolExecutionResult
	err = json.Unmarshal(data, &unmarshaled)
	require.NoError(t, err)

	assert.Equal(t, "call_123", unmarshaled.ToolCallID)
	assert.Equal(t, "get_weather", unmarshaled.FunctionName)
	assert.NotNil(t, unmarshaled.Result)
	assert.Empty(t, unmarshaled.Error)
	assert.Equal(t, int64(150), unmarshaled.ExecutionTime)
}

func TestToolExecutionResult_Error(t *testing.T) {
	result := ToolExecutionResult{
		ToolCallID:    "call_456",
		FunctionName:  "search_database",
		Result:        nil,
		Error:         "Database connection timeout",
		ExecutionTime: 5000,
	}

	data, err := json.Marshal(result)
	require.NoError(t, err)

	var unmarshaled ToolExecutionResult
	err = json.Unmarshal(data, &unmarshaled)
	require.NoError(t, err)

	assert.Equal(t, "call_456", unmarshaled.ToolCallID)
	assert.Equal(t, "search_database", unmarshaled.FunctionName)
	assert.Nil(t, unmarshaled.Result)
	assert.Equal(t, "Database connection timeout", unmarshaled.Error)
	assert.Equal(t, int64(5000), unmarshaled.ExecutionTime)
}

func TestToolExecutionResult_WithMetadata(t *testing.T) {
	result := ToolExecutionResult{
		ToolCallID:    "call_789",
		FunctionName:  "http_request",
		Result:        map[string]any{"status": 200},
		ExecutionTime: 250,
		Metadata: map[string]any{
			"retries":       2,
			"cache_hit":     true,
			"response_size": 1024,
		},
	}

	data, err := json.Marshal(result)
	require.NoError(t, err)

	var unmarshaled ToolExecutionResult
	err = json.Unmarshal(data, &unmarshaled)
	require.NoError(t, err)

	assert.NotNil(t, unmarshaled.Metadata)
	assert.Equal(t, float64(2), unmarshaled.Metadata["retries"])
	assert.Equal(t, true, unmarshaled.Metadata["cache_hit"])
}

// ==================== ConversationHistory Tests ====================

func TestConversationHistory_Empty(t *testing.T) {
	history := ConversationHistory{
		Messages:        []LLMMessage{},
		ToolExecutions:  []ToolExecutionResult{},
		TotalIterations: 0,
	}

	data, err := json.Marshal(history)
	require.NoError(t, err)

	var unmarshaled ConversationHistory
	err = json.Unmarshal(data, &unmarshaled)
	require.NoError(t, err)

	assert.Empty(t, unmarshaled.Messages)
	assert.Empty(t, unmarshaled.ToolExecutions)
	assert.Equal(t, 0, unmarshaled.TotalIterations)
}

func TestConversationHistory_SimpleConversation(t *testing.T) {
	history := ConversationHistory{
		Messages: []LLMMessage{
			{
				Role:    "user",
				Content: "What's the weather in Paris?",
			},
			{
				Role:    "assistant",
				Content: "Let me check that for you.",
			},
		},
		TotalIterations: 1,
	}

	data, err := json.Marshal(history)
	require.NoError(t, err)

	var unmarshaled ConversationHistory
	err = json.Unmarshal(data, &unmarshaled)
	require.NoError(t, err)

	assert.Len(t, unmarshaled.Messages, 2)
	assert.Equal(t, "user", unmarshaled.Messages[0].Role)
	assert.Equal(t, "assistant", unmarshaled.Messages[1].Role)
	assert.Equal(t, 1, unmarshaled.TotalIterations)
}

func TestConversationHistory_WithToolCalls(t *testing.T) {
	history := ConversationHistory{
		Messages: []LLMMessage{
			{
				Role:    "user",
				Content: "What's the weather in Paris?",
			},
			{
				Role:    "assistant",
				Content: "",
				ToolCalls: []LLMToolCall{
					{
						ID:   "call_123",
						Type: "function",
						Function: LLMFunctionCall{
							Name:      "get_weather",
							Arguments: `{"location":"Paris"}`,
						},
					},
				},
			},
			{
				Role:       "tool",
				Content:    `{"temperature": 22, "conditions": "sunny"}`,
				ToolCallID: "call_123",
				Name:       "get_weather",
			},
			{
				Role:    "assistant",
				Content: "The weather in Paris is sunny with a temperature of 22°C.",
			},
		},
		ToolExecutions: []ToolExecutionResult{
			{
				ToolCallID:    "call_123",
				FunctionName:  "get_weather",
				Result:        map[string]any{"temperature": 22, "conditions": "sunny"},
				ExecutionTime: 150,
			},
		},
		TotalIterations: 1,
	}

	data, err := json.Marshal(history)
	require.NoError(t, err)

	var unmarshaled ConversationHistory
	err = json.Unmarshal(data, &unmarshaled)
	require.NoError(t, err)

	assert.Len(t, unmarshaled.Messages, 4)
	assert.Len(t, unmarshaled.ToolExecutions, 1)
	assert.Equal(t, 1, unmarshaled.TotalIterations)

	// Verify tool call message
	assert.Equal(t, "assistant", unmarshaled.Messages[1].Role)
	assert.Len(t, unmarshaled.Messages[1].ToolCalls, 1)
	assert.Equal(t, "call_123", unmarshaled.Messages[1].ToolCalls[0].ID)

	// Verify tool response message
	assert.Equal(t, "tool", unmarshaled.Messages[2].Role)
	assert.Equal(t, "call_123", unmarshaled.Messages[2].ToolCallID)
	assert.Equal(t, "get_weather", unmarshaled.Messages[2].Name)

	// Verify tool execution result
	assert.Equal(t, "call_123", unmarshaled.ToolExecutions[0].ToolCallID)
	assert.Equal(t, "get_weather", unmarshaled.ToolExecutions[0].FunctionName)
}

func TestConversationHistory_MultipleIterations(t *testing.T) {
	history := ConversationHistory{
		Messages: []LLMMessage{
			{Role: "user", Content: "Question 1"},
			{Role: "assistant", Content: "Answer 1"},
			{Role: "user", Content: "Question 2"},
			{Role: "assistant", Content: "Answer 2"},
			{Role: "user", Content: "Question 3"},
			{Role: "assistant", Content: "Answer 3"},
		},
		TotalIterations: 3,
	}

	data, err := json.Marshal(history)
	require.NoError(t, err)

	var unmarshaled ConversationHistory
	err = json.Unmarshal(data, &unmarshaled)
	require.NoError(t, err)

	assert.Len(t, unmarshaled.Messages, 6)
	assert.Equal(t, 3, unmarshaled.TotalIterations)
}

// ==================== Complex Integration Tests ====================

func TestToolCalling_CompleteWorkflow(t *testing.T) {
	// Create a complete tool calling configuration
	config := &ToolCallConfig{
		Mode:              ToolCallModeAuto,
		MaxIterations:     5,
		TimeoutPerTool:    30,
		TotalTimeout:      300,
		StopOnToolFailure: true,
	}

	// Create function definitions
	functions := []FunctionDefinition{
		{
			Type:        FunctionTypeBuiltin,
			Name:        "get_weather",
			Description: "Get weather information",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"location": map[string]any{"type": "string"},
				},
			},
			BuiltinName: "weather_api",
		},
		{
			Type:        FunctionTypeSubWorkflow,
			Name:        "process_data",
			Description: "Process data with sub-workflow",
			WorkflowID:  "wf_data_processor",
			InputMapping: map[string]string{
				"data": "input.raw_data",
			},
			OutputExtractor: ".result",
		},
	}

	// Create conversation history
	history := ConversationHistory{
		Messages: []LLMMessage{
			{Role: "user", Content: "What's the weather in Paris and process the data?"},
			{
				Role: "assistant",
				ToolCalls: []LLMToolCall{
					{
						ID:   "call_1",
						Type: "function",
						Function: LLMFunctionCall{
							Name:      "get_weather",
							Arguments: `{"location":"Paris"}`,
						},
					},
					{
						ID:   "call_2",
						Type: "function",
						Function: LLMFunctionCall{
							Name:      "process_data",
							Arguments: `{"data":[1,2,3]}`,
						},
					},
				},
			},
		},
		ToolExecutions: []ToolExecutionResult{
			{
				ToolCallID:    "call_1",
				FunctionName:  "get_weather",
				Result:        map[string]any{"temperature": 22},
				ExecutionTime: 150,
			},
			{
				ToolCallID:    "call_2",
				FunctionName:  "process_data",
				Result:        map[string]any{"sum": 6},
				ExecutionTime: 200,
			},
		},
		TotalIterations: 1,
	}

	// Marshal everything
	configData, err := json.Marshal(config)
	require.NoError(t, err)

	functionsData, err := json.Marshal(functions)
	require.NoError(t, err)

	historyData, err := json.Marshal(history)
	require.NoError(t, err)

	// Unmarshal and verify
	var unmarshaledConfig ToolCallConfig
	err = json.Unmarshal(configData, &unmarshaledConfig)
	require.NoError(t, err)
	assert.Equal(t, ToolCallModeAuto, unmarshaledConfig.Mode)

	var unmarshaledFunctions []FunctionDefinition
	err = json.Unmarshal(functionsData, &unmarshaledFunctions)
	require.NoError(t, err)
	assert.Len(t, unmarshaledFunctions, 2)

	var unmarshaledHistory ConversationHistory
	err = json.Unmarshal(historyData, &unmarshaledHistory)
	require.NoError(t, err)
	assert.Len(t, unmarshaledHistory.Messages, 2)
	assert.Len(t, unmarshaledHistory.ToolExecutions, 2)
}
