package builtin

import (
	"context"
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
