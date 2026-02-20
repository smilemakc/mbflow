package builtin

import (
	"context"
	"testing"

	"github.com/smilemakc/mbflow/go/pkg/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestGeminiProvider_NewGeminiProvider tests the constructor
func TestGeminiProvider_NewGeminiProvider(t *testing.T) {
	tests := []struct {
		name        string
		apiKey      string
		baseURL     string
		wantErr     bool
		errMsg      string
		expectedURL string
	}{
		{
			name:        "success with API key only (default baseURL)",
			apiKey:      "test-api-key",
			baseURL:     "",
			wantErr:     false,
			expectedURL: "https://generativelanguage.googleapis.com/v1beta",
		},
		{
			name:        "success with API key and custom baseURL",
			apiKey:      "test-api-key",
			baseURL:     "https://custom-proxy.example.com/v1",
			wantErr:     false,
			expectedURL: "https://custom-proxy.example.com/v1",
		},
		{
			name:    "error when API key is empty",
			apiKey:  "",
			baseURL: "",
			wantErr: true,
			errMsg:  "api_key is required for Gemini provider",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider, err := NewGeminiProvider(tt.apiKey, tt.baseURL)

			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
				assert.Nil(t, provider)
			} else {
				require.NoError(t, err)
				require.NotNil(t, provider)
				assert.Equal(t, tt.apiKey, provider.apiKey)
				assert.Equal(t, tt.expectedURL, provider.baseURL)
				assert.NotNil(t, provider.client)
			}
		})
	}
}

// TestGeminiProvider_BuildRequestBody tests request building
func TestGeminiProvider_BuildRequestBody(t *testing.T) {
	provider, err := NewGeminiProvider("test-key", "")
	require.NoError(t, err)

	tests := []struct {
		name     string
		req      *models.LLMRequest
		validate func(t *testing.T, body map[string]any)
	}{
		{
			name: "basic text prompt",
			req: &models.LLMRequest{
				Prompt: "Hello, how are you?",
			},
			validate: func(t *testing.T, body map[string]any) {
				contents, ok := body["contents"].([]map[string]any)
				require.True(t, ok)
				require.Len(t, contents, 1)

				assert.Equal(t, "user", contents[0]["role"])

				parts, ok := contents[0]["parts"].([]map[string]any)
				require.True(t, ok)
				require.Len(t, parts, 1)
				assert.Equal(t, "Hello, how are you?", parts[0]["text"])
			},
		},
		{
			name: "with system instruction",
			req: &models.LLMRequest{
				Instruction: "You are a helpful assistant",
				Prompt:      "Hello",
			},
			validate: func(t *testing.T, body map[string]any) {
				systemInstruction, ok := body["systemInstruction"].(map[string]any)
				require.True(t, ok)

				parts, ok := systemInstruction["parts"].([]map[string]any)
				require.True(t, ok)
				require.Len(t, parts, 1)
				assert.Equal(t, "You are a helpful assistant", parts[0]["text"])
			},
		},
		{
			name: "with generation config params",
			req: &models.LLMRequest{
				Prompt:        "Test",
				Temperature:   0.7,
				TopP:          0.9,
				MaxTokens:     1000,
				StopSequences: []string{"END", "STOP"},
			},
			validate: func(t *testing.T, body map[string]any) {
				generationConfig, ok := body["generationConfig"].(map[string]any)
				require.True(t, ok)

				assert.Equal(t, 0.7, generationConfig["temperature"])
				assert.Equal(t, 0.9, generationConfig["topP"])
				assert.Equal(t, 1000, generationConfig["maxOutputTokens"])

				stopSeqs, ok := generationConfig["stopSequences"].([]string)
				require.True(t, ok)
				assert.Equal(t, []string{"END", "STOP"}, stopSeqs)
			},
		},
		{
			name: "with JSON mode response format",
			req: &models.LLMRequest{
				Prompt: "Generate JSON",
				ResponseFormat: &models.LLMResponseFormat{
					Type: "json_object",
				},
			},
			validate: func(t *testing.T, body map[string]any) {
				generationConfig, ok := body["generationConfig"].(map[string]any)
				require.True(t, ok)
				assert.Equal(t, "application/json", generationConfig["responseMimeType"])
			},
		},
		{
			name: "with JSON schema response format",
			req: &models.LLMRequest{
				Prompt: "Generate structured data",
				ResponseFormat: &models.LLMResponseFormat{
					Type: "json_schema",
					JSONSchema: &models.LLMJSONSchema{
						Name: "user_schema",
						Schema: map[string]any{
							"type": "object",
							"properties": map[string]any{
								"name": map[string]any{"type": "string"},
								"age":  map[string]any{"type": "integer"},
							},
						},
					},
				},
			},
			validate: func(t *testing.T, body map[string]any) {
				generationConfig, ok := body["generationConfig"].(map[string]any)
				require.True(t, ok)
				assert.Equal(t, "application/json", generationConfig["responseMimeType"])

				responseSchema, ok := generationConfig["responseSchema"].(map[string]any)
				require.True(t, ok)
				assert.Equal(t, "object", responseSchema["type"])

				properties, ok := responseSchema["properties"].(map[string]any)
				require.True(t, ok)
				assert.Contains(t, properties, "name")
				assert.Contains(t, properties, "age")
			},
		},
		{
			name: "with tools (function calling)",
			req: &models.LLMRequest{
				Prompt: "What's the weather?",
				Tools: []models.LLMTool{
					{
						Type: "function",
						Function: models.LLMFunctionTool{
							Name:        "get_weather",
							Description: "Get weather for a location",
							Parameters: map[string]any{
								"type": "object",
								"properties": map[string]any{
									"location": map[string]any{"type": "string"},
								},
							},
						},
					},
					{
						Type: "function",
						Function: models.LLMFunctionTool{
							Name:        "get_time",
							Description: "Get current time",
							Parameters: map[string]any{
								"type": "object",
							},
						},
					},
				},
			},
			validate: func(t *testing.T, body map[string]any) {
				tools, ok := body["tools"].([]map[string]any)
				require.True(t, ok)
				require.Len(t, tools, 1)

				functionDeclarations, ok := tools[0]["functionDeclarations"].([]map[string]any)
				require.True(t, ok)
				require.Len(t, functionDeclarations, 2)

				assert.Equal(t, "get_weather", functionDeclarations[0]["name"])
				assert.Equal(t, "Get weather for a location", functionDeclarations[0]["description"])
				assert.NotNil(t, functionDeclarations[0]["parameters"])

				assert.Equal(t, "get_time", functionDeclarations[1]["name"])
				assert.Equal(t, "Get current time", functionDeclarations[1]["description"])
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body := provider.buildRequestBody(tt.req)
			require.NotNil(t, body)
			tt.validate(t, body)
		})
	}
}

// TestGeminiProvider_BuildUserContent tests multimodal content building
func TestGeminiProvider_BuildUserContent(t *testing.T) {
	provider, err := NewGeminiProvider("test-key", "")
	require.NoError(t, err)

	tests := []struct {
		name     string
		req      *models.LLMRequest
		validate func(t *testing.T, parts []map[string]any)
	}{
		{
			name: "text only",
			req: &models.LLMRequest{
				Prompt: "Describe this",
			},
			validate: func(t *testing.T, parts []map[string]any) {
				require.Len(t, parts, 1)
				assert.Equal(t, "Describe this", parts[0]["text"])
			},
		},
		{
			name: "text with base64 image file (inline_data)",
			req: &models.LLMRequest{
				Prompt: "What's in this image?",
				Files: []models.LLMFileAttachment{
					{
						MimeType: "image/png",
						Data:     "base64encodeddata==",
					},
				},
			},
			validate: func(t *testing.T, parts []map[string]any) {
				require.Len(t, parts, 2)

				// First part is text
				assert.Equal(t, "What's in this image?", parts[0]["text"])

				// Second part is inline_data
				inlineData, ok := parts[1]["inline_data"].(map[string]any)
				require.True(t, ok)
				assert.Equal(t, "image/png", inlineData["mime_type"])
				assert.Equal(t, "base64encodeddata==", inlineData["data"])
			},
		},
		{
			name: "text with image URL (file_data)",
			req: &models.LLMRequest{
				Prompt:    "Analyze these images",
				ImageURLs: []string{"https://example.com/image.jpg", "https://example.com/photo.png"},
			},
			validate: func(t *testing.T, parts []map[string]any) {
				require.Len(t, parts, 3)

				// First part is text
				assert.Equal(t, "Analyze these images", parts[0]["text"])

				// Second part is JPEG file_data
				fileData1, ok := parts[1]["file_data"].(map[string]any)
				require.True(t, ok)
				assert.Equal(t, "image/jpeg", fileData1["mime_type"])
				assert.Equal(t, "https://example.com/image.jpg", fileData1["file_uri"])

				// Third part is PNG file_data
				fileData2, ok := parts[2]["file_data"].(map[string]any)
				require.True(t, ok)
				assert.Equal(t, "image/png", fileData2["mime_type"])
				assert.Equal(t, "https://example.com/photo.png", fileData2["file_uri"])
			},
		},
		{
			name: "multiple image URLs with different formats",
			req: &models.LLMRequest{
				Prompt: "Compare",
				ImageURLs: []string{
					"https://example.com/photo.GIF",
					"https://example.com/image.webp",
					"https://example.com/unknown.xyz",
				},
			},
			validate: func(t *testing.T, parts []map[string]any) {
				require.Len(t, parts, 4)

				// Check GIF
				fileData1, ok := parts[1]["file_data"].(map[string]any)
				require.True(t, ok)
				assert.Equal(t, "image/gif", fileData1["mime_type"])

				// Check WEBP
				fileData2, ok := parts[2]["file_data"].(map[string]any)
				require.True(t, ok)
				assert.Equal(t, "image/webp", fileData2["mime_type"])

				// Check unknown (defaults to JPEG)
				fileData3, ok := parts[3]["file_data"].(map[string]any)
				require.True(t, ok)
				assert.Equal(t, "image/jpeg", fileData3["mime_type"])
			},
		},
		{
			name: "mixed: text, base64 image, and URL images",
			req: &models.LLMRequest{
				Prompt: "Compare all these images",
				Files: []models.LLMFileAttachment{
					{
						MimeType: "image/jpeg",
						Data:     "inlinebase64data",
					},
				},
				ImageURLs: []string{"https://example.com/remote.png"},
			},
			validate: func(t *testing.T, parts []map[string]any) {
				require.Len(t, parts, 3)

				// Text
				assert.Equal(t, "Compare all these images", parts[0]["text"])

				// Inline data
				inlineData, ok := parts[1]["inline_data"].(map[string]any)
				require.True(t, ok)
				assert.Equal(t, "image/jpeg", inlineData["mime_type"])

				// File data
				fileData, ok := parts[2]["file_data"].(map[string]any)
				require.True(t, ok)
				assert.Equal(t, "image/png", fileData["mime_type"])
			},
		},
		{
			name: "non-image file is skipped",
			req: &models.LLMRequest{
				Prompt: "Process file",
				Files: []models.LLMFileAttachment{
					{
						MimeType: "application/pdf",
						Data:     "pdfdata",
					},
				},
			},
			validate: func(t *testing.T, parts []map[string]any) {
				// Only text part, PDF is not supported as image
				require.Len(t, parts, 1)
				assert.Equal(t, "Process file", parts[0]["text"])
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parts := provider.buildUserContent(tt.req)
			require.NotNil(t, parts)
			tt.validate(t, parts)
		})
	}
}

// TestGeminiProvider_ConvertResponse tests response parsing
func TestGeminiProvider_ConvertResponse(t *testing.T) {
	provider, err := NewGeminiProvider("test-key", "")
	require.NoError(t, err)

	tests := []struct {
		name     string
		resp     *geminiGenerateContentResponse
		req      *models.LLMRequest
		validate func(t *testing.T, result *models.LLMResponse)
	}{
		{
			name: "normal text response",
			resp: &geminiGenerateContentResponse{
				Candidates: []geminiCandidate{
					{
						Content: geminiContent{
							Role: "model",
							Parts: []geminiPart{
								{Text: "Hello! I'm doing great, thank you for asking."},
							},
						},
						FinishReason: "STOP",
					},
				},
				UsageMetadata: geminiUsageMetadata{
					PromptTokenCount:     10,
					CandidatesTokenCount: 15,
					TotalTokenCount:      25,
				},
				ModelVersion: "gemini-2.5-flash",
				ResponseID:   "resp-123",
			},
			req: &models.LLMRequest{Model: "gemini-2.5-flash"},
			validate: func(t *testing.T, result *models.LLMResponse) {
				assert.Equal(t, "Hello! I'm doing great, thank you for asking.", result.Content)
				assert.Equal(t, "stop", result.FinishReason)
				assert.Equal(t, "gemini-2.5-flash", result.Model)
				assert.Equal(t, "resp-123", result.ResponseID)
				assert.Equal(t, 10, result.Usage.PromptTokens)
				assert.Equal(t, 15, result.Usage.CompletionTokens)
				assert.Equal(t, 25, result.Usage.TotalTokens)
				assert.Empty(t, result.ToolCalls)
			},
		},
		{
			name: "response with tool calls",
			resp: &geminiGenerateContentResponse{
				Candidates: []geminiCandidate{
					{
						Content: geminiContent{
							Role: "model",
							Parts: []geminiPart{
								{
									FunctionCall: &geminiFunctionCall{
										Name: "get_weather",
										Args: map[string]any{
											"location": "London",
											"units":    "celsius",
										},
									},
								},
							},
						},
						FinishReason: "STOP",
					},
				},
				UsageMetadata: geminiUsageMetadata{
					PromptTokenCount:     20,
					CandidatesTokenCount: 10,
					TotalTokenCount:      30,
				},
				ModelVersion: "gemini-2.5-flash",
			},
			req: &models.LLMRequest{Model: "gemini-2.5-flash"},
			validate: func(t *testing.T, result *models.LLMResponse) {
				assert.Empty(t, result.Content)
				assert.Equal(t, "tool_calls", result.FinishReason)
				assert.Len(t, result.ToolCalls, 1)

				toolCall := result.ToolCalls[0]
				assert.Equal(t, "get_weather", toolCall.ID)
				assert.Equal(t, "function", toolCall.Type)
				assert.Equal(t, "get_weather", toolCall.Function.Name)

				// Verify arguments are JSON-encoded
				assert.Contains(t, toolCall.Function.Arguments, "London")
				assert.Contains(t, toolCall.Function.Arguments, "celsius")
			},
		},
		{
			name: "response with multiple tool calls",
			resp: &geminiGenerateContentResponse{
				Candidates: []geminiCandidate{
					{
						Content: geminiContent{
							Role: "model",
							Parts: []geminiPart{
								{
									FunctionCall: &geminiFunctionCall{
										Name: "get_weather",
										Args: map[string]any{"location": "London"},
									},
								},
								{
									FunctionCall: &geminiFunctionCall{
										Name: "get_time",
										Args: map[string]any{"timezone": "UTC"},
									},
								},
							},
						},
						FinishReason: "STOP",
					},
				},
				UsageMetadata: geminiUsageMetadata{
					TotalTokenCount: 40,
				},
				ModelVersion: "gemini-2.5-flash",
			},
			req: &models.LLMRequest{Model: "gemini-2.5-flash"},
			validate: func(t *testing.T, result *models.LLMResponse) {
				assert.Equal(t, "tool_calls", result.FinishReason)
				assert.Len(t, result.ToolCalls, 2)

				assert.Equal(t, "get_weather", result.ToolCalls[0].Function.Name)
				assert.Equal(t, "get_time", result.ToolCalls[1].Function.Name)
			},
		},
		{
			name: "response with text and tool call",
			resp: &geminiGenerateContentResponse{
				Candidates: []geminiCandidate{
					{
						Content: geminiContent{
							Role: "model",
							Parts: []geminiPart{
								{Text: "Let me check the weather for you."},
								{
									FunctionCall: &geminiFunctionCall{
										Name: "get_weather",
										Args: map[string]any{"location": "Paris"},
									},
								},
							},
						},
						FinishReason: "STOP",
					},
				},
				UsageMetadata: geminiUsageMetadata{
					TotalTokenCount: 35,
				},
				ModelVersion: "gemini-2.5-flash",
			},
			req: &models.LLMRequest{Model: "gemini-2.5-flash"},
			validate: func(t *testing.T, result *models.LLMResponse) {
				assert.Equal(t, "Let me check the weather for you.", result.Content)
				assert.Equal(t, "tool_calls", result.FinishReason)
				assert.Len(t, result.ToolCalls, 1)
			},
		},
		{
			name: "empty candidates (error model)",
			resp: &geminiGenerateContentResponse{
				Candidates:    []geminiCandidate{},
				UsageMetadata: geminiUsageMetadata{},
				ModelVersion:  "gemini-2.5-flash",
			},
			req: &models.LLMRequest{Model: "gemini-2.5-flash"},
			validate: func(t *testing.T, result *models.LLMResponse) {
				assert.Empty(t, result.Content)
				assert.Equal(t, "error", result.FinishReason)
				assert.Equal(t, "gemini-2.5-flash", result.Model)
				assert.Empty(t, result.ToolCalls)
			},
		},
		{
			name: "empty candidates with no ModelVersion (uses request model)",
			resp: &geminiGenerateContentResponse{
				Candidates:    []geminiCandidate{},
				UsageMetadata: geminiUsageMetadata{},
				ModelVersion:  "",
			},
			req: &models.LLMRequest{Model: "gemini-2.5-flash-fallback"},
			validate: func(t *testing.T, result *models.LLMResponse) {
				assert.Equal(t, "error", result.FinishReason)
				assert.Equal(t, "gemini-2.5-flash-fallback", result.Model)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := provider.convertResponse(tt.resp, tt.req)
			require.NotNil(t, result)
			tt.validate(t, result)
		})
	}
}

// TestGeminiProvider_NormalizeFinishReason tests finish reason mapping
func TestGeminiProvider_NormalizeFinishReason(t *testing.T) {
	provider, err := NewGeminiProvider("test-key", "")
	require.NoError(t, err)

	tests := []struct {
		name     string
		reason   string
		expected string
	}{
		{
			name:     "STOP -> stop",
			reason:   "STOP",
			expected: "stop",
		},
		{
			name:     "stop (lowercase already) -> stop",
			reason:   "stop",
			expected: "stop",
		},
		{
			name:     "MAX_TOKENS -> length",
			reason:   "MAX_TOKENS",
			expected: "length",
		},
		{
			name:     "SAFETY -> content_filter",
			reason:   "SAFETY",
			expected: "content_filter",
		},
		{
			name:     "unknown uppercase -> lowercase",
			reason:   "RECITATION",
			expected: "recitation",
		},
		{
			name:     "unknown mixed case -> lowercase",
			reason:   "OtherReason",
			expected: "otherreason",
		},
		{
			name:     "empty string -> empty string",
			reason:   "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := provider.normalizeFinishReason(tt.reason)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestGeminiProvider_Execute_WithMockProvider tests integration through executor
func TestGeminiProvider_Execute_WithMockProvider(t *testing.T) {
	executor := NewLLMExecutor()

	// Create a mock Gemini provider
	mockProvider := &MockLLMProvider{
		ExecuteFn: func(ctx context.Context, req *models.LLMRequest) (*models.LLMResponse, error) {
			// Verify request was parsed correctly
			assert.Equal(t, "gemini-2.5-flash", req.Model)
			assert.Equal(t, "You are helpful", req.Instruction)
			assert.Equal(t, "Hello, Gemini!", req.Prompt)
			assert.Equal(t, 0.8, req.Temperature)
			assert.Equal(t, 500, req.MaxTokens)

			// Verify response format
			require.NotNil(t, req.ResponseFormat)
			assert.Equal(t, "json_object", req.ResponseFormat.Type)

			return &models.LLMResponse{
				Content:      `{"response": "Hello from Gemini"}`,
				ResponseID:   "gemini-resp-456",
				Model:        "gemini-2.5-flash",
				FinishReason: "stop",
				Usage: models.LLMUsage{
					PromptTokens:     15,
					CompletionTokens: 10,
					TotalTokens:      25,
				},
			}, nil
		},
	}

	// Register as gemini provider
	executor.RegisterProvider("gemini", mockProvider)

	config := map[string]any{
		"provider":    "gemini",
		"model":       "gemini-2.5-flash",
		"instruction": "You are helpful",
		"prompt":      "Hello, Gemini!",
		"temperature": 0.8,
		"max_tokens":  500,
		"response_format": map[string]any{
			"type": "json_object",
		},
	}

	result, err := executor.Execute(context.Background(), config, nil)
	require.NoError(t, err)
	require.NotNil(t, result)

	resultMap, ok := result.(map[string]any)
	require.True(t, ok)

	assert.Equal(t, `{"response": "Hello from Gemini"}`, resultMap["content"])
	assert.Equal(t, "gemini-resp-456", resultMap["response_id"])
	assert.Equal(t, "gemini-2.5-flash", resultMap["model"])
	assert.Equal(t, "stop", resultMap["finish_reason"])

	usage, ok := resultMap["usage"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, 15, usage["prompt_tokens"])
	assert.Equal(t, 10, usage["completion_tokens"])
	assert.Equal(t, 25, usage["total_tokens"])
}

// TestGeminiProvider_BuildRequestBody_EmptyPrompt tests handling of empty prompt
func TestGeminiProvider_BuildRequestBody_EmptyPrompt(t *testing.T) {
	provider, err := NewGeminiProvider("test-key", "")
	require.NoError(t, err)

	req := &models.LLMRequest{
		Prompt: "",
		Files: []models.LLMFileAttachment{
			{
				MimeType: "image/png",
				Data:     "imagedata",
			},
		},
	}

	body := provider.buildRequestBody(req)
	require.NotNil(t, body)

	contents, ok := body["contents"].([]map[string]any)
	require.True(t, ok)
	require.Len(t, contents, 1)

	parts, ok := contents[0]["parts"].([]map[string]any)
	require.True(t, ok)

	// Should only have the image part, no text part
	require.Len(t, parts, 1)
	_, hasInlineData := parts[0]["inline_data"]
	assert.True(t, hasInlineData, "Should have inline_data for image")
}

// TestGeminiProvider_BuildRequestBody_NoGenerationConfigWhenEmpty tests that generationConfig is omitted when empty
func TestGeminiProvider_BuildRequestBody_NoGenerationConfigWhenEmpty(t *testing.T) {
	provider, err := NewGeminiProvider("test-key", "")
	require.NoError(t, err)

	req := &models.LLMRequest{
		Prompt: "Simple prompt",
		// No temperature, topP, maxTokens, stopSequences, or responseFormat
	}

	body := provider.buildRequestBody(req)
	require.NotNil(t, body)

	// generationConfig should not be present
	_, hasGenerationConfig := body["generationConfig"]
	assert.False(t, hasGenerationConfig, "generationConfig should be omitted when empty")
}

// TestGeminiProvider_BuildTools tests tool building
func TestGeminiProvider_BuildTools(t *testing.T) {
	provider, err := NewGeminiProvider("test-key", "")
	require.NoError(t, err)

	tools := []models.LLMTool{
		{
			Type: "function",
			Function: models.LLMFunctionTool{
				Name:        "search_web",
				Description: "Search the web for information",
				Parameters: map[string]any{
					"type": "object",
					"properties": map[string]any{
						"query": map[string]any{"type": "string"},
					},
					"required": []string{"query"},
				},
			},
		},
		{
			Type: "function",
			Function: models.LLMFunctionTool{
				Name:        "calculate",
				Description: "Perform a calculation",
				Parameters: map[string]any{
					"type": "object",
					"properties": map[string]any{
						"expression": map[string]any{"type": "string"},
					},
				},
			},
		},
	}

	result := provider.buildTools(tools)
	require.Len(t, result, 1)

	functionDeclarations, ok := result[0]["functionDeclarations"].([]map[string]any)
	require.True(t, ok)
	require.Len(t, functionDeclarations, 2)

	// First function
	assert.Equal(t, "search_web", functionDeclarations[0]["name"])
	assert.Equal(t, "Search the web for information", functionDeclarations[0]["description"])
	params0, ok := functionDeclarations[0]["parameters"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "object", params0["type"])

	// Second function
	assert.Equal(t, "calculate", functionDeclarations[1]["name"])
	assert.Equal(t, "Perform a calculation", functionDeclarations[1]["description"])
	params1, ok := functionDeclarations[1]["parameters"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "object", params1["type"])
}
