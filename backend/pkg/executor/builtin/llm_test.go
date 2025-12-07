package builtin

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/smilemakc/mbflow/internal/application/template"
	"github.com/smilemakc/mbflow/pkg/executor"
	"github.com/smilemakc/mbflow/pkg/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockLLMProvider is a mock LLM provider for testing.
type MockLLMProvider struct {
	ExecuteFn func(ctx context.Context, req *models.LLMRequest) (*models.LLMResponse, error)
}

func (m *MockLLMProvider) Execute(ctx context.Context, req *models.LLMRequest) (*models.LLMResponse, error) {
	if m.ExecuteFn != nil {
		return m.ExecuteFn(ctx, req)
	}
	return &models.LLMResponse{
		Content:      "Mock response",
		ResponseID:   "mock-response-id",
		Model:        req.Model,
		FinishReason: "stop",
		Usage: models.LLMUsage{
			PromptTokens:     10,
			CompletionTokens: 20,
			TotalTokens:      30,
		},
		CreatedAt: time.Now(),
	}, nil
}

func TestLLMExecutor_Validate(t *testing.T) {
	executor := NewLLMExecutor()
	executor.RegisterProvider("mock", &MockLLMProvider{})

	tests := []struct {
		name    string
		config  map[string]interface{}
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid basic config",
			config: map[string]interface{}{
				"provider": "openai",
				"model":    "gpt-4",
				"prompt":   "Hello",
				"api_key":  "sk-test",
			},
			wantErr: false,
		},
		{
			name: "missing provider",
			config: map[string]interface{}{
				"model":   "gpt-4",
				"prompt":  "Hello",
				"api_key": "sk-test",
			},
			wantErr: true,
			errMsg:  "required field missing: provider",
		},
		{
			name: "missing model",
			config: map[string]interface{}{
				"provider": "openai",
				"prompt":   "Hello",
				"api_key":  "sk-test",
			},
			wantErr: true,
			errMsg:  "required field missing: model",
		},
		{
			name: "missing prompt",
			config: map[string]interface{}{
				"provider": "openai",
				"model":    "gpt-4",
				"api_key":  "sk-test",
			},
			wantErr: true,
			errMsg:  "required field missing: prompt",
		},
		{
			name: "missing api_key",
			config: map[string]interface{}{
				"provider": "openai",
				"model":    "gpt-4",
				"prompt":   "Hello",
			},
			wantErr: true,
			errMsg:  "required field missing: api_key",
		},
		{
			name: "invalid temperature",
			config: map[string]interface{}{
				"provider":    "openai",
				"model":       "gpt-4",
				"prompt":      "Hello",
				"api_key":     "sk-test",
				"temperature": 3.0,
			},
			wantErr: true,
			errMsg:  "temperature must be between 0 and 2",
		},
		{
			name: "invalid top_p",
			config: map[string]interface{}{
				"provider": "openai",
				"model":    "gpt-4",
				"prompt":   "Hello",
				"api_key":  "sk-test",
				"top_p":    1.5,
			},
			wantErr: true,
			errMsg:  "top_p must be between 0 and 1",
		},
		{
			name: "valid with all parameters",
			config: map[string]interface{}{
				"provider":          "openai",
				"model":             "gpt-4",
				"instruction":       "You are a helpful assistant",
				"prompt":            "Hello",
				"api_key":           "sk-test",
				"max_tokens":        1000,
				"temperature":       0.7,
				"top_p":             0.9,
				"frequency_penalty": 0.5,
				"presence_penalty":  0.5,
			},
			wantErr: false,
		},
		{
			name: "valid with response format",
			config: map[string]interface{}{
				"provider": "openai",
				"model":    "gpt-4",
				"prompt":   "Hello",
				"api_key":  "sk-test",
				"response_format": map[string]interface{}{
					"type": "json_object",
				},
			},
			wantErr: false,
		},
		{
			name: "valid with json_schema",
			config: map[string]interface{}{
				"provider": "openai",
				"model":    "gpt-4",
				"prompt":   "Hello",
				"api_key":  "sk-test",
				"response_format": map[string]interface{}{
					"type": "json_schema",
					"json_schema": map[string]interface{}{
						"name": "user_schema",
						"schema": map[string]interface{}{
							"type": "object",
							"properties": map[string]interface{}{
								"name": map[string]interface{}{"type": "string"},
							},
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "invalid json_schema - missing name",
			config: map[string]interface{}{
				"provider": "openai",
				"model":    "gpt-4",
				"prompt":   "Hello",
				"api_key":  "sk-test",
				"response_format": map[string]interface{}{
					"type": "json_schema",
					"json_schema": map[string]interface{}{
						"schema": map[string]interface{}{
							"type": "object",
						},
					},
				},
			},
			wantErr: true,
			errMsg:  "json_schema name is required",
		},
		{
			name: "valid with tools",
			config: map[string]interface{}{
				"provider": "openai",
				"model":    "gpt-4",
				"prompt":   "Hello",
				"api_key":  "sk-test",
				"tools": []interface{}{
					map[string]interface{}{
						"type": "function",
						"function": map[string]interface{}{
							"name":        "get_weather",
							"description": "Get weather",
							"parameters": map[string]interface{}{
								"type": "object",
								"properties": map[string]interface{}{
									"location": map[string]interface{}{"type": "string"},
								},
							},
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "invalid tools - missing name",
			config: map[string]interface{}{
				"provider": "openai",
				"model":    "gpt-4",
				"prompt":   "Hello",
				"api_key":  "sk-test",
				"tools": []interface{}{
					map[string]interface{}{
						"type": "function",
						"function": map[string]interface{}{
							"description": "Get weather",
						},
					},
				},
			},
			wantErr: true,
			errMsg:  "function name is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := executor.Validate(tt.config)
			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestLLMExecutor_Execute_BasicRequest(t *testing.T) {
	executor := NewLLMExecutor()

	mockProvider := &MockLLMProvider{
		ExecuteFn: func(ctx context.Context, req *models.LLMRequest) (*models.LLMResponse, error) {
			assert.Equal(t, "gpt-4", req.Model)
			assert.Equal(t, "You are helpful", req.Instruction)
			assert.Equal(t, "Hello!", req.Prompt)
			assert.Equal(t, 100, req.MaxTokens)
			assert.Equal(t, 0.7, req.Temperature)

			return &models.LLMResponse{
				Content:      "Hi there!",
				ResponseID:   "resp-123",
				Model:        "gpt-4",
				FinishReason: "stop",
				Usage: models.LLMUsage{
					PromptTokens:     5,
					CompletionTokens: 3,
					TotalTokens:      8,
				},
				CreatedAt: time.Now(),
			}, nil
		},
	}

	executor.RegisterProvider("mock", mockProvider)

	config := map[string]interface{}{
		"provider":    "mock",
		"model":       "gpt-4",
		"instruction": "You are helpful",
		"prompt":      "Hello!",
		"max_tokens":  100,
		"temperature": 0.7,
	}

	result, err := executor.Execute(context.Background(), config, nil)
	require.NoError(t, err)
	require.NotNil(t, result)

	resultMap, ok := result.(map[string]interface{})
	require.True(t, ok)

	assert.Equal(t, "Hi there!", resultMap["content"])
	assert.Equal(t, "resp-123", resultMap["response_id"])
	assert.Equal(t, "gpt-4", resultMap["model"])
	assert.Equal(t, "stop", resultMap["finish_reason"])

	usage, ok := resultMap["usage"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, 5, usage["prompt_tokens"])
	assert.Equal(t, 3, usage["completion_tokens"])
	assert.Equal(t, 8, usage["total_tokens"])
}

func TestLLMExecutor_Execute_WithToolCalls(t *testing.T) {
	executor := NewLLMExecutor()

	mockProvider := &MockLLMProvider{
		ExecuteFn: func(ctx context.Context, req *models.LLMRequest) (*models.LLMResponse, error) {
			assert.Len(t, req.Tools, 1)
			assert.Equal(t, "get_weather", req.Tools[0].Function.Name)

			return &models.LLMResponse{
				Content:      "",
				ResponseID:   "resp-456",
				Model:        "gpt-4",
				FinishReason: "tool_calls",
				Usage: models.LLMUsage{
					PromptTokens:     10,
					CompletionTokens: 5,
					TotalTokens:      15,
				},
				ToolCalls: []models.LLMToolCall{
					{
						ID:   "call-123",
						Type: "function",
						Function: models.LLMFunctionCall{
							Name:      "get_weather",
							Arguments: `{"location":"London"}`,
						},
					},
				},
				CreatedAt: time.Now(),
			}, nil
		},
	}

	executor.RegisterProvider("mock", mockProvider)

	config := map[string]interface{}{
		"provider": "mock",
		"model":    "gpt-4",
		"prompt":   "What's the weather in London?",
		"tools": []interface{}{
			map[string]interface{}{
				"type": "function",
				"function": map[string]interface{}{
					"name":        "get_weather",
					"description": "Get weather for a location",
					"parameters": map[string]interface{}{
						"type": "object",
						"properties": map[string]interface{}{
							"location": map[string]interface{}{"type": "string"},
						},
					},
				},
			},
		},
	}

	result, err := executor.Execute(context.Background(), config, nil)
	require.NoError(t, err)
	require.NotNil(t, result)

	resultMap, ok := result.(map[string]interface{})
	require.True(t, ok)

	assert.Equal(t, "tool_calls", resultMap["finish_reason"])

	toolCalls, ok := resultMap["tool_calls"].([]map[string]interface{})
	require.True(t, ok)
	require.Len(t, toolCalls, 1)

	assert.Equal(t, "call-123", toolCalls[0]["id"])
	assert.Equal(t, "function", toolCalls[0]["type"])

	function, ok := toolCalls[0]["function"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "get_weather", function["name"])
	assert.Equal(t, `{"location":"London"}`, function["arguments"])
}

func TestLLMExecutor_Execute_WithResponseFormat(t *testing.T) {
	executor := NewLLMExecutor()

	mockProvider := &MockLLMProvider{
		ExecuteFn: func(ctx context.Context, req *models.LLMRequest) (*models.LLMResponse, error) {
			assert.NotNil(t, req.ResponseFormat)
			assert.Equal(t, "json_schema", req.ResponseFormat.Type)
			assert.NotNil(t, req.ResponseFormat.JSONSchema)
			assert.Equal(t, "user_info", req.ResponseFormat.JSONSchema.Name)

			return &models.LLMResponse{
				Content:      `{"name":"John","age":30}`,
				ResponseID:   "resp-789",
				Model:        "gpt-4",
				FinishReason: "stop",
				Usage: models.LLMUsage{
					PromptTokens:     15,
					CompletionTokens: 10,
					TotalTokens:      25,
				},
				CreatedAt: time.Now(),
			}, nil
		},
	}

	executor.RegisterProvider("mock", mockProvider)

	config := map[string]interface{}{
		"provider": "mock",
		"model":    "gpt-4",
		"prompt":   "Extract user info",
		"response_format": map[string]interface{}{
			"type": "json_schema",
			"json_schema": map[string]interface{}{
				"name": "user_info",
				"schema": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"name": map[string]interface{}{"type": "string"},
						"age":  map[string]interface{}{"type": "integer"},
					},
				},
				"strict": true,
			},
		},
	}

	result, err := executor.Execute(context.Background(), config, nil)
	require.NoError(t, err)
	require.NotNil(t, result)

	resultMap, ok := result.(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, `{"name":"John","age":30}`, resultMap["content"])
}

func TestLLMExecutor_Execute_WithMultimodal(t *testing.T) {
	executor := NewLLMExecutor()

	mockProvider := &MockLLMProvider{
		ExecuteFn: func(ctx context.Context, req *models.LLMRequest) (*models.LLMResponse, error) {
			assert.Len(t, req.ImageURLs, 1)
			assert.Equal(t, "https://example.com/image.jpg", req.ImageURLs[0])

			return &models.LLMResponse{
				Content:      "This is a picture of a cat",
				ResponseID:   "resp-multimodal",
				Model:        "gpt-4-vision",
				FinishReason: "stop",
				Usage: models.LLMUsage{
					PromptTokens:     50,
					CompletionTokens: 20,
					TotalTokens:      70,
				},
				CreatedAt: time.Now(),
			}, nil
		},
	}

	executor.RegisterProvider("mock", mockProvider)

	config := map[string]interface{}{
		"provider": "mock",
		"model":    "gpt-4-vision",
		"prompt":   "Describe this image",
		"image_url": []interface{}{
			"https://example.com/image.jpg",
		},
	}

	result, err := executor.Execute(context.Background(), config, nil)
	require.NoError(t, err)
	require.NotNil(t, result)

	resultMap, ok := result.(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "This is a picture of a cat", resultMap["content"])
}

func TestLLMExecutor_ParseConfig(t *testing.T) {
	executor := NewLLMExecutor()

	config := map[string]interface{}{
		"provider":          "openai",
		"model":             "gpt-4",
		"instruction":       "You are helpful",
		"prompt":            "Hello",
		"max_tokens":        1000,
		"temperature":       0.7,
		"top_p":             0.9,
		"frequency_penalty": 0.5,
		"presence_penalty":  0.3,
		"vector_store_id":   "vs-123",
		"image_url":         []interface{}{"https://example.com/image.jpg"},
		"stop_sequences":    []interface{}{"END"},
	}

	req, err := executor.parseConfig(config)
	require.NoError(t, err)

	assert.Equal(t, models.LLMProviderOpenAI, req.Provider)
	assert.Equal(t, "gpt-4", req.Model)
	assert.Equal(t, "You are helpful", req.Instruction)
	assert.Equal(t, "Hello", req.Prompt)
	assert.Equal(t, 1000, req.MaxTokens)
	assert.Equal(t, 0.7, req.Temperature)
	assert.Equal(t, 0.9, req.TopP)
	assert.Equal(t, 0.5, req.FrequencyPenalty)
	assert.Equal(t, 0.3, req.PresencePenalty)
	assert.Equal(t, "vs-123", req.VectorStoreID)
	assert.Equal(t, []string{"https://example.com/image.jpg"}, req.ImageURLs)
	assert.Equal(t, []string{"END"}, req.StopSequences)
}

func TestLLMExecutor_ParseTools(t *testing.T) {
	executor := NewLLMExecutor()

	toolsConfig := []interface{}{
		map[string]interface{}{
			"type": "function",
			"function": map[string]interface{}{
				"name":        "get_weather",
				"description": "Get weather for a location",
				"parameters": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"location": map[string]interface{}{"type": "string"},
					},
					"required": []interface{}{"location"},
				},
			},
		},
	}

	tools, err := executor.parseTools(toolsConfig)
	require.NoError(t, err)
	require.Len(t, tools, 1)

	assert.Equal(t, "function", tools[0].Type)
	assert.Equal(t, "get_weather", tools[0].Function.Name)
	assert.Equal(t, "Get weather for a location", tools[0].Function.Description)
	assert.NotNil(t, tools[0].Function.Parameters)
}

func TestLLMExecutor_UnsupportedProvider(t *testing.T) {
	executor := NewLLMExecutor()

	config := map[string]interface{}{
		"provider": "unsupported",
		"model":    "gpt-4",
		"prompt":   "Hello",
		"api_key":  "sk-test",
	}

	err := executor.Validate(config)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported LLM provider")
}

func TestLLMExecutor_WithInputTemplates(t *testing.T) {
	exec := NewLLMExecutor()

	mockProvider := &MockLLMProvider{
		ExecuteFn: func(ctx context.Context, req *models.LLMRequest) (*models.LLMResponse, error) {
			// Проверяем что prompt был резолвлен из input
			assert.Equal(t, "Summarize this article: Long article text...", req.Prompt)
			assert.Equal(t, "Use English style", req.Instruction)

			return &models.LLMResponse{
				Content:      "Article summary",
				Model:        "gpt-4",
				FinishReason: "stop",
				Usage: models.LLMUsage{
					PromptTokens:     50,
					CompletionTokens: 20,
					TotalTokens:      70,
				},
				CreatedAt: time.Now(),
			}, nil
		},
	}

	exec.RegisterProvider("mock", mockProvider)

	// Конфигурация с input шаблонами
	config := map[string]interface{}{
		"provider":    "mock",
		"model":       "gpt-4",
		"prompt":      "Summarize this article: {{input.article}}",
		"instruction": "Use {{input.language}} style",
	}

	// Input от предыдущей ноды
	inputData := map[string]interface{}{
		"article":  "Long article text...",
		"language": "English",
	}

	// Создаём template engine с input vars
	varCtx := template.NewVariableContext()
	varCtx.InputVars = inputData

	engine := template.NewEngine(varCtx, template.TemplateOptions{})
	wrappedExec := executor.NewTemplateExecutorWrapper(exec, engine)

	// Выполняем
	result, err := wrappedExec.Execute(context.Background(), config, inputData)
	require.NoError(t, err)

	resultMap, ok := result.(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "Article summary", resultMap["content"])
	assert.Equal(t, "gpt-4", resultMap["model"])
	assert.Equal(t, "stop", resultMap["finish_reason"])

	usage, ok := resultMap["usage"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, 50, usage["prompt_tokens"])
	assert.Equal(t, 20, usage["completion_tokens"])
	assert.Equal(t, 70, usage["total_tokens"])
}

func TestLLMExecutor_WithInputDirectly(t *testing.T) {
	exec := NewLLMExecutor()

	// Expected structured input for Responses API
	expectedInput := map[string]interface{}{
		"messages": []interface{}{
			map[string]interface{}{"role": "user", "content": "Hello"},
			map[string]interface{}{"role": "assistant", "content": "Hi there!"},
			map[string]interface{}{"role": "user", "content": "How are you?"},
		},
	}

	mockProvider := &MockLLMProvider{
		ExecuteFn: func(ctx context.Context, req *models.LLMRequest) (*models.LLMResponse, error) {
			// Verify that input was passed directly to the request
			require.NotNil(t, req.Input, "Input should be passed to LLM request")

			inputMap, ok := req.Input.(map[string]interface{})
			require.True(t, ok, "Input should be a map")

			messages, ok := inputMap["messages"].([]interface{})
			require.True(t, ok, "Input should contain messages array")
			require.Len(t, messages, 3, "Should have 3 messages")

			return &models.LLMResponse{
				Content:      "I'm doing great, thanks!",
				ResponseID:   "resp-input-test",
				Model:        "gpt-4",
				FinishReason: "stop",
				Usage: models.LLMUsage{
					PromptTokens:     25,
					CompletionTokens: 10,
					TotalTokens:      35,
				},
				CreatedAt: time.Now(),
			}, nil
		},
	}

	exec.RegisterProvider("mock", mockProvider)

	// Config with use_input_directly flag
	config := map[string]interface{}{
		"provider":           "mock",
		"model":              "gpt-4",
		"prompt":             "Continue the conversation",
		"use_input_directly": true,
	}

	// Execute with structured input
	result, err := exec.Execute(context.Background(), config, expectedInput)
	require.NoError(t, err)
	require.NotNil(t, result)

	resultMap, ok := result.(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "I'm doing great, thanks!", resultMap["content"])
	assert.Equal(t, "resp-input-test", resultMap["response_id"])
}

func TestLLMExecutor_WithoutInputDirectly(t *testing.T) {
	exec := NewLLMExecutor()

	mockProvider := &MockLLMProvider{
		ExecuteFn: func(ctx context.Context, req *models.LLMRequest) (*models.LLMResponse, error) {
			// When use_input_directly is false (default), Input should be nil
			assert.Nil(t, req.Input, "Input should not be passed when use_input_directly is false")

			return &models.LLMResponse{
				Content:      "Response without direct input",
				ResponseID:   "resp-no-input",
				Model:        "gpt-4",
				FinishReason: "stop",
				Usage: models.LLMUsage{
					PromptTokens:     10,
					CompletionTokens: 5,
					TotalTokens:      15,
				},
				CreatedAt: time.Now(),
			}, nil
		},
	}

	exec.RegisterProvider("mock", mockProvider)

	// Config WITHOUT use_input_directly flag (default behavior)
	config := map[string]interface{}{
		"provider": "mock",
		"model":    "gpt-4",
		"prompt":   "Simple prompt",
	}

	inputData := map[string]interface{}{
		"some_data": "This should not be passed directly",
	}

	// Execute
	result, err := exec.Execute(context.Background(), config, inputData)
	require.NoError(t, err)
	require.NotNil(t, result)

	resultMap, ok := result.(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "Response without direct input", resultMap["content"])
}

func TestLLMExecutor_WithExplicitInputInConfig(t *testing.T) {
	exec := NewLLMExecutor()

	explicitInput := "Explicit input from config"

	mockProvider := &MockLLMProvider{
		ExecuteFn: func(ctx context.Context, req *models.LLMRequest) (*models.LLMResponse, error) {
			// When config has explicit "input" field, it should take precedence
			require.NotNil(t, req.Input, "Input should be set from config")
			assert.Equal(t, explicitInput, req.Input, "Input should match config value")

			return &models.LLMResponse{
				Content:      "Response with explicit config input",
				ResponseID:   "resp-explicit",
				Model:        "gpt-4",
				FinishReason: "stop",
				Usage: models.LLMUsage{
					PromptTokens:     15,
					CompletionTokens: 8,
					TotalTokens:      23,
				},
				CreatedAt: time.Now(),
			}, nil
		},
	}

	exec.RegisterProvider("mock", mockProvider)

	// Config with explicit "input" field
	config := map[string]interface{}{
		"provider": "mock",
		"model":    "gpt-4",
		"prompt":   "Process this",
		"input":    explicitInput, // Explicit input in config
	}

	paramInput := map[string]interface{}{
		"this": "should be ignored",
	}

	// Execute - explicit config input should take precedence
	result, err := exec.Execute(context.Background(), config, paramInput)
	require.NoError(t, err)
	require.NotNil(t, result)

	resultMap, ok := result.(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "Response with explicit config input", resultMap["content"])
}

func TestLLMExecutor_InputPriorityOrder(t *testing.T) {
	exec := NewLLMExecutor()

	tests := []struct {
		name           string
		configInput    interface{}
		useInputDirect bool
		paramInput     interface{}
		expectedInput  interface{}
	}{
		{
			name:           "explicit config input takes precedence over use_input_directly",
			configInput:    "config value",
			useInputDirect: true,
			paramInput:     "param value",
			expectedInput:  "config value",
		},
		{
			name:           "use_input_directly when no config input",
			configInput:    nil,
			useInputDirect: true,
			paramInput:     "param value",
			expectedInput:  "param value",
		},
		{
			name:           "no input when use_input_directly is false",
			configInput:    nil,
			useInputDirect: false,
			paramInput:     "param value",
			expectedInput:  nil,
		},
		{
			name:           "explicit config input even when use_input_directly is false",
			configInput:    "config value",
			useInputDirect: false,
			paramInput:     "param value",
			expectedInput:  "config value",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockProvider := &MockLLMProvider{
				ExecuteFn: func(ctx context.Context, req *models.LLMRequest) (*models.LLMResponse, error) {
					assert.Equal(t, tt.expectedInput, req.Input, "Input should match expected value")
					return &models.LLMResponse{
						Content:      "test response",
						Model:        "gpt-4",
						FinishReason: "stop",
						Usage:        models.LLMUsage{TotalTokens: 10},
						CreatedAt:    time.Now(),
					}, nil
				},
			}

			exec.RegisterProvider("mock", mockProvider)

			config := map[string]interface{}{
				"provider": "mock",
				"model":    "gpt-4",
				"prompt":   "test",
			}

			if tt.configInput != nil {
				config["input"] = tt.configInput
			}

			if tt.useInputDirect {
				config["use_input_directly"] = true
			}

			_, err := exec.Execute(context.Background(), config, tt.paramInput)
			require.NoError(t, err)
		})
	}
}

// ==================== AUTO MODE TOOL CALLING TESTS ====================

func TestLLMExecutor_AutoMode_SingleToolCall(t *testing.T) {
	exec := NewLLMExecutor()

	// Setup mock function registry
	funcRegistry := models.NewFunctionRegistry()
	funcRegistry.Register("get_weather", func(args map[string]interface{}) (interface{}, error) {
		location := args["location"].(string)
		return map[string]interface{}{
			"location":    location,
			"temperature": 22,
			"condition":   "sunny",
		}, nil
	})

	registry := NewToolCallingRegistry(funcRegistry)
	exec.SetToolCallingRegistry(registry)

	// Mock provider that returns tool call first, then final answer
	callCount := 0
	mockProvider := &MockLLMProvider{
		ExecuteFn: func(ctx context.Context, req *models.LLMRequest) (*models.LLMResponse, error) {
			callCount++

			if callCount == 1 {
				// First call: LLM requests tool call
				return &models.LLMResponse{
					Content:      "",
					ResponseID:   "resp-1",
					Model:        "gpt-4",
					FinishReason: "tool_calls",
					Usage:        models.LLMUsage{TotalTokens: 50},
					ToolCalls: []models.LLMToolCall{
						{
							ID:   "call-123",
							Type: "function",
							Function: models.LLMFunctionCall{
								Name:      "get_weather",
								Arguments: `{"location":"London"}`,
							},
						},
					},
					CreatedAt: time.Now(),
				}, nil
			}

			// Second call: LLM returns final answer after tool execution
			assert.Len(t, req.Messages, 3) // system/user, assistant with tool_calls, tool result
			lastMsg := req.Messages[len(req.Messages)-1]
			assert.Equal(t, "tool", lastMsg.Role)
			assert.Equal(t, "call-123", lastMsg.ToolCallID)

			return &models.LLMResponse{
				Content:      "The weather in London is sunny with 22°C",
				ResponseID:   "resp-2",
				Model:        "gpt-4",
				FinishReason: "stop",
				Usage:        models.LLMUsage{TotalTokens: 75},
				CreatedAt:    time.Now(),
			}, nil
		},
	}

	exec.RegisterProvider("mock", mockProvider)

	config := map[string]interface{}{
		"provider": "mock",
		"model":    "gpt-4",
		"prompt":   "What's the weather in London?",
		"tool_call_config": map[string]interface{}{
			"mode":           "auto",
			"max_iterations": 10,
		},
		"functions": []interface{}{
			map[string]interface{}{
				"type":         "builtin",
				"name":         "get_weather",
				"description":  "Get weather for a location",
				"builtin_name": "get_weather",
				"parameters": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"location": map[string]interface{}{"type": "string"},
					},
				},
			},
		},
	}

	result, err := exec.Execute(context.Background(), config, nil)
	require.NoError(t, err)
	require.NotNil(t, result)

	resultMap, ok := result.(map[string]interface{})
	require.True(t, ok)

	assert.Equal(t, "The weather in London is sunny with 22°C", resultMap["content"])
	assert.Equal(t, "stop", resultMap["finish_reason"])
	assert.Equal(t, "finish", resultMap["stopped_reason"])
	assert.Equal(t, 2, resultMap["total_iterations"])

	// Check messages history
	messagesRaw, ok := resultMap["messages"].([]interface{})
	require.True(t, ok, "messages should be an array")
	assert.GreaterOrEqual(t, len(messagesRaw), 3) // user, assistant with tool_calls, tool result

	// Check tool executions
	toolExecsRaw, ok := resultMap["tool_executions"].([]interface{})
	require.True(t, ok, "tool_executions should be an array")
	assert.Len(t, toolExecsRaw, 1)

	toolExec0 := toolExecsRaw[0].(map[string]interface{})
	assert.Equal(t, "call-123", toolExec0["tool_call_id"])
	assert.Equal(t, "get_weather", toolExec0["function_name"])
	assert.NotNil(t, toolExec0["result"])
}

func TestLLMExecutor_AutoMode_MultipleIterations(t *testing.T) {
	exec := NewLLMExecutor()

	// Setup registry with two functions
	funcRegistry := models.NewFunctionRegistry()
	funcRegistry.Register("get_weather", func(args map[string]interface{}) (interface{}, error) {
		return map[string]interface{}{"temperature": 22, "condition": "sunny"}, nil
	})
	funcRegistry.Register("get_time", func(args map[string]interface{}) (interface{}, error) {
		return map[string]interface{}{"time": "14:30"}, nil
	})

	registry := NewToolCallingRegistry(funcRegistry)
	exec.SetToolCallingRegistry(registry)

	callCount := 0
	mockProvider := &MockLLMProvider{
		ExecuteFn: func(ctx context.Context, req *models.LLMRequest) (*models.LLMResponse, error) {
			callCount++

			if callCount == 1 {
				// First iteration: request weather
				return &models.LLMResponse{
					Content:      "",
					Model:        "gpt-4",
					FinishReason: "tool_calls",
					Usage:        models.LLMUsage{TotalTokens: 50},
					ToolCalls: []models.LLMToolCall{
						{
							ID:   "call-1",
							Type: "function",
							Function: models.LLMFunctionCall{
								Name:      "get_weather",
								Arguments: `{"location":"London"}`,
							},
						},
					},
					CreatedAt: time.Now(),
				}, nil
			} else if callCount == 2 {
				// Second iteration: request time
				return &models.LLMResponse{
					Content:      "",
					Model:        "gpt-4",
					FinishReason: "tool_calls",
					Usage:        models.LLMUsage{TotalTokens: 70},
					ToolCalls: []models.LLMToolCall{
						{
							ID:   "call-2",
							Type: "function",
							Function: models.LLMFunctionCall{
								Name:      "get_time",
								Arguments: `{}`,
							},
						},
					},
					CreatedAt: time.Now(),
				}, nil
			}

			// Third iteration: final answer
			return &models.LLMResponse{
				Content:      "It's 14:30 and sunny with 22°C",
				Model:        "gpt-4",
				FinishReason: "stop",
				Usage:        models.LLMUsage{TotalTokens: 90},
				CreatedAt:    time.Now(),
			}, nil
		},
	}

	exec.RegisterProvider("mock", mockProvider)

	config := map[string]interface{}{
		"provider": "mock",
		"model":    "gpt-4",
		"prompt":   "What's the weather and time?",
		"tool_call_config": map[string]interface{}{
			"mode": "auto",
		},
		"functions": []interface{}{
			map[string]interface{}{
				"type":         "builtin",
				"name":         "get_weather",
				"builtin_name": "get_weather",
			},
			map[string]interface{}{
				"type":         "builtin",
				"name":         "get_time",
				"builtin_name": "get_time",
			},
		},
	}

	result, err := exec.Execute(context.Background(), config, nil)
	require.NoError(t, err)

	resultMap, ok := result.(map[string]interface{})
	require.True(t, ok)

	assert.Equal(t, "It's 14:30 and sunny with 22°C", resultMap["content"])
	assert.Equal(t, 3, resultMap["total_iterations"])

	toolExecsRaw, ok := resultMap["tool_executions"].([]interface{})
	require.True(t, ok, "tool_executions should be an array")
	assert.Len(t, toolExecsRaw, 2) // Two tool calls
}

func TestLLMExecutor_AutoMode_MaxIterations(t *testing.T) {
	exec := NewLLMExecutor()

	funcRegistry := models.NewFunctionRegistry()
	funcRegistry.Register("infinite_tool", func(args map[string]interface{}) (interface{}, error) {
		return map[string]interface{}{"status": "ok"}, nil
	})

	registry := NewToolCallingRegistry(funcRegistry)
	exec.SetToolCallingRegistry(registry)

	// Mock that always returns tool calls (infinite loop scenario)
	mockProvider := &MockLLMProvider{
		ExecuteFn: func(ctx context.Context, req *models.LLMRequest) (*models.LLMResponse, error) {
			return &models.LLMResponse{
				Content:      "",
				Model:        "gpt-4",
				FinishReason: "tool_calls",
				Usage:        models.LLMUsage{TotalTokens: 50},
				ToolCalls: []models.LLMToolCall{
					{
						ID:   "call-infinite",
						Type: "function",
						Function: models.LLMFunctionCall{
							Name:      "infinite_tool",
							Arguments: `{}`,
						},
					},
				},
				CreatedAt: time.Now(),
			}, nil
		},
	}

	exec.RegisterProvider("mock", mockProvider)

	config := map[string]interface{}{
		"provider": "mock",
		"model":    "gpt-4",
		"prompt":   "Test infinite loop",
		"tool_call_config": map[string]interface{}{
			"mode":           "auto",
			"max_iterations": 3,
		},
		"functions": []interface{}{
			map[string]interface{}{
				"type":         "builtin",
				"name":         "infinite_tool",
				"builtin_name": "infinite_tool",
			},
		},
	}

	result, err := exec.Execute(context.Background(), config, nil)
	require.NoError(t, err)

	resultMap, ok := result.(map[string]interface{})
	require.True(t, ok)

	assert.Equal(t, 3, resultMap["total_iterations"])
	assert.Equal(t, "max_iterations", resultMap["stopped_reason"])

	toolExecsRaw, ok := resultMap["tool_executions"].([]interface{})
	require.True(t, ok, "tool_executions should be an array")
	assert.Len(t, toolExecsRaw, 3) // Should have 3 tool executions
}

func TestLLMExecutor_AutoMode_StopOnToolFailure(t *testing.T) {
	exec := NewLLMExecutor()

	funcRegistry := models.NewFunctionRegistry()
	funcRegistry.Register("failing_tool", func(args map[string]interface{}) (interface{}, error) {
		return nil, fmt.Errorf("tool execution failed")
	})

	registry := NewToolCallingRegistry(funcRegistry)
	exec.SetToolCallingRegistry(registry)

	mockProvider := &MockLLMProvider{
		ExecuteFn: func(ctx context.Context, req *models.LLMRequest) (*models.LLMResponse, error) {
			return &models.LLMResponse{
				Content:      "",
				Model:        "gpt-4",
				FinishReason: "tool_calls",
				Usage:        models.LLMUsage{TotalTokens: 50},
				ToolCalls: []models.LLMToolCall{
					{
						ID:   "call-fail",
						Type: "function",
						Function: models.LLMFunctionCall{
							Name:      "failing_tool",
							Arguments: `{}`,
						},
					},
				},
				CreatedAt: time.Now(),
			}, nil
		},
	}

	exec.RegisterProvider("mock", mockProvider)

	config := map[string]interface{}{
		"provider": "mock",
		"model":    "gpt-4",
		"prompt":   "Test tool failure",
		"tool_call_config": map[string]interface{}{
			"mode":                 "auto",
			"stop_on_tool_failure": true,
		},
		"functions": []interface{}{
			map[string]interface{}{
				"type":         "builtin",
				"name":         "failing_tool",
				"builtin_name": "failing_tool",
			},
		},
	}

	_, err := exec.Execute(context.Background(), config, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "tool execution failed")
}

func TestLLMExecutor_ParseToolCallConfig(t *testing.T) {
	exec := NewLLMExecutor()

	tests := []struct {
		name     string
		config   map[string]interface{}
		expected *models.ToolCallConfig
	}{
		{
			name: "full config",
			config: map[string]interface{}{
				"mode":                 "auto",
				"max_iterations":       5.0,
				"timeout_per_tool":     30.0,
				"total_timeout":        300.0,
				"stop_on_tool_failure": true,
			},
			expected: &models.ToolCallConfig{
				Mode:              models.ToolCallModeAuto,
				MaxIterations:     5,
				TimeoutPerTool:    30,
				TotalTimeout:      300,
				StopOnToolFailure: true,
			},
		},
		{
			name: "backward compatibility - auto_execute_tools",
			config: map[string]interface{}{
				"auto_execute_tools": true,
			},
			expected: &models.ToolCallConfig{
				Mode:              models.ToolCallModeAuto,
				MaxIterations:     10,
				TimeoutPerTool:    30,
				TotalTimeout:      300,
				StopOnToolFailure: false,
			},
		},
		{
			name:   "default config",
			config: map[string]interface{}{},
			expected: &models.ToolCallConfig{
				Mode:              models.ToolCallModeManual,
				MaxIterations:     10,
				TimeoutPerTool:    30,
				TotalTimeout:      300,
				StopOnToolFailure: false,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := exec.parseToolCallConfig(tt.config)
			require.NoError(t, err)
			assert.Equal(t, tt.expected.Mode, result.Mode)
			assert.Equal(t, tt.expected.MaxIterations, result.MaxIterations)
			assert.Equal(t, tt.expected.TimeoutPerTool, result.TimeoutPerTool)
			assert.Equal(t, tt.expected.TotalTimeout, result.TotalTimeout)
			assert.Equal(t, tt.expected.StopOnToolFailure, result.StopOnToolFailure)
		})
	}
}

func TestLLMExecutor_ParseFunctions(t *testing.T) {
	exec := NewLLMExecutor()

	functionsConfig := []interface{}{
		map[string]interface{}{
			"type":         "builtin",
			"name":         "get_weather",
			"description":  "Get weather",
			"builtin_name": "get_weather",
			"parameters": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"location": map[string]interface{}{"type": "string"},
				},
			},
		},
		map[string]interface{}{
			"type":        "sub_workflow",
			"name":        "process_data",
			"description": "Process data via workflow",
			"workflow_id": "workflow-123",
			"input_mapping": map[string]interface{}{
				"data":   "input_data",
				"format": "output_format",
			},
			"output_extractor": ".result",
		},
	}

	functions, err := exec.parseFunctions(functionsConfig)
	require.NoError(t, err)
	require.Len(t, functions, 2)

	// Check builtin function
	assert.Equal(t, models.FunctionTypeBuiltin, functions[0].Type)
	assert.Equal(t, "get_weather", functions[0].Name)
	assert.Equal(t, "get_weather", functions[0].BuiltinName)

	// Check sub-workflow function
	assert.Equal(t, models.FunctionTypeSubWorkflow, functions[1].Type)
	assert.Equal(t, "process_data", functions[1].Name)
	assert.Equal(t, "workflow-123", functions[1].WorkflowID)
	assert.Equal(t, "input_data", functions[1].InputMapping["data"])
	assert.Equal(t, ".result", functions[1].OutputExtractor)
}

func TestLLMExecutor_ConvertFunctionsToTools(t *testing.T) {
	exec := NewLLMExecutor()

	functions := []models.FunctionDefinition{
		{
			Name:        "get_weather",
			Description: "Get weather for a location",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"location": map[string]interface{}{"type": "string"},
				},
			},
		},
	}

	tools, err := exec.convertFunctionsToTools(functions)
	require.NoError(t, err)
	require.Len(t, tools, 1)

	assert.Equal(t, "function", tools[0].Type)
	assert.Equal(t, "get_weather", tools[0].Function.Name)
	assert.Equal(t, "Get weather for a location", tools[0].Function.Description)
	assert.NotNil(t, tools[0].Function.Parameters)
}

func TestLLMExecutor_AutoMode_WithoutRegistry(t *testing.T) {
	exec := NewLLMExecutor()
	// Don't set registry

	mockProvider := &MockLLMProvider{}
	exec.RegisterProvider("mock", mockProvider)

	config := map[string]interface{}{
		"provider": "mock",
		"model":    "gpt-4",
		"prompt":   "Test",
		"tool_call_config": map[string]interface{}{
			"mode": "auto",
		},
		"functions": []interface{}{
			map[string]interface{}{
				"type": "builtin",
				"name": "test_func",
			},
		},
	}

	_, err := exec.Execute(context.Background(), config, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "tool calling registry not configured")
}
