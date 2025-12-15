package models

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ==================== LLMFileAttachment Tests ====================

func TestLLMFileAttachment_IsImage(t *testing.T) {
	tests := []struct {
		name     string
		mimeType string
		expected bool
	}{
		{"JPEG image", "image/jpeg", true},
		{"PNG image", "image/png", true},
		{"GIF image", "image/gif", true},
		{"WebP image", "image/webp", true},
		{"PDF not image", "application/pdf", false},
		{"Plain text not image", "text/plain", false},
		{"Empty mime type", "", false},
		{"Unknown image type", "image/bmp", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			attachment := &LLMFileAttachment{
				MimeType: tt.mimeType,
			}

			result := attachment.IsImage()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestLLMFileAttachment_IsPDF(t *testing.T) {
	tests := []struct {
		name     string
		mimeType string
		expected bool
	}{
		{"PDF file", "application/pdf", true},
		{"JPEG not PDF", "image/jpeg", false},
		{"PNG not PDF", "image/png", false},
		{"Empty mime type", "", false},
		{"Text file not PDF", "text/plain", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			attachment := &LLMFileAttachment{
				MimeType: tt.mimeType,
			}

			result := attachment.IsPDF()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestLLMFileAttachment_IsSupported(t *testing.T) {
	tests := []struct {
		name     string
		mimeType string
		expected bool
	}{
		// Images - supported
		{"JPEG supported", "image/jpeg", true},
		{"PNG supported", "image/png", true},
		{"GIF supported", "image/gif", true},
		{"WebP supported", "image/webp", true},

		// PDF - supported
		{"PDF supported", "application/pdf", true},

		// Unsupported types
		{"BMP not supported", "image/bmp", false},
		{"SVG not supported", "image/svg+xml", false},
		{"Text not supported", "text/plain", false},
		{"JSON not supported", "application/json", false},
		{"Empty not supported", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			attachment := &LLMFileAttachment{
				MimeType: tt.mimeType,
			}

			result := attachment.IsSupported()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestLLMFileAttachment_AllMethods(t *testing.T) {
	// Test all methods together with different file types
	tests := []struct {
		name        string
		attachment  LLMFileAttachment
		isImage     bool
		isPDF       bool
		isSupported bool
	}{
		{
			name: "JPEG image file",
			attachment: LLMFileAttachment{
				Data:     "base64data",
				MimeType: "image/jpeg",
				Name:     "photo.jpg",
				Detail:   "high",
			},
			isImage:     true,
			isPDF:       false,
			isSupported: true,
		},
		{
			name: "PNG image file",
			attachment: LLMFileAttachment{
				Data:     "base64data",
				MimeType: "image/png",
				Name:     "screenshot.png",
				Detail:   "auto",
			},
			isImage:     true,
			isPDF:       false,
			isSupported: true,
		},
		{
			name: "PDF document",
			attachment: LLMFileAttachment{
				Data:     "base64data",
				MimeType: "application/pdf",
				Name:     "document.pdf",
			},
			isImage:     false,
			isPDF:       true,
			isSupported: true,
		},
		{
			name: "Unsupported text file",
			attachment: LLMFileAttachment{
				Data:     "base64data",
				MimeType: "text/plain",
				Name:     "notes.txt",
			},
			isImage:     false,
			isPDF:       false,
			isSupported: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.isImage, tt.attachment.IsImage(), "IsImage mismatch")
			assert.Equal(t, tt.isPDF, tt.attachment.IsPDF(), "IsPDF mismatch")
			assert.Equal(t, tt.isSupported, tt.attachment.IsSupported(), "IsSupported mismatch")
		})
	}
}

// ==================== LLMError Tests ====================

func TestLLMError_Error(t *testing.T) {
	tests := []struct {
		name     string
		err      LLMError
		expected string
	}{
		{
			name: "OpenAI error",
			err: LLMError{
				Provider: LLMProviderOpenAI,
				Code:     "invalid_api_key",
				Message:  "Invalid API key provided",
			},
			expected: "LLM error (openai): Invalid API key provided",
		},
		{
			name: "Anthropic error",
			err: LLMError{
				Provider: LLMProviderAnthropic,
				Code:     "rate_limit_exceeded",
				Message:  "Rate limit exceeded",
			},
			expected: "LLM error (anthropic): Rate limit exceeded",
		},
		{
			name: "OpenAI Responses error",
			err: LLMError{
				Provider: LLMProviderOpenAIResponses,
				Code:     "model_not_found",
				Message:  "The model 'gpt-5' does not exist",
			},
			expected: "LLM error (openai-responses): The model 'gpt-5' does not exist",
		},
		{
			name: "Error with type and param",
			err: LLMError{
				Provider: LLMProviderOpenAI,
				Code:     "invalid_request_error",
				Message:  "Invalid parameter value",
				Type:     "invalid_request",
				Param:    "temperature",
			},
			expected: "LLM error (openai): Invalid parameter value",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.err.Error()
			assert.Equal(t, tt.expected, result)
		})
	}
}

// ==================== LLMRequest JSON Marshaling Tests ====================

func TestLLMRequest_JSONMarshaling(t *testing.T) {
	request := LLMRequest{
		Provider:    LLMProviderOpenAI,
		Model:       "gpt-4",
		Instruction: "You are a helpful assistant",
		Prompt:      "Hello, world!",
		MaxTokens:   100,
		Temperature: 0.7,
		TopP:        0.9,
		Tools: []LLMTool{
			{
				Type: "function",
				Function: LLMFunctionTool{
					Name:        "get_weather",
					Description: "Get weather for location",
					Parameters: map[string]interface{}{
						"type": "object",
						"properties": map[string]interface{}{
							"location": map[string]interface{}{
								"type": "string",
							},
						},
					},
				},
			},
		},
		ResponseFormat: &LLMResponseFormat{
			Type: "json_object",
		},
	}

	// Marshal to JSON
	data, err := json.Marshal(request)
	require.NoError(t, err)
	assert.NotEmpty(t, data)

	// Unmarshal back
	var unmarshaled LLMRequest
	err = json.Unmarshal(data, &unmarshaled)
	require.NoError(t, err)

	// Verify fields
	assert.Equal(t, request.Provider, unmarshaled.Provider)
	assert.Equal(t, request.Model, unmarshaled.Model)
	assert.Equal(t, request.Instruction, unmarshaled.Instruction)
	assert.Equal(t, request.Prompt, unmarshaled.Prompt)
	assert.Equal(t, request.MaxTokens, unmarshaled.MaxTokens)
	assert.Equal(t, request.Temperature, unmarshaled.Temperature)
	assert.Equal(t, request.TopP, unmarshaled.TopP)
	assert.Len(t, unmarshaled.Tools, 1)
	assert.Equal(t, "get_weather", unmarshaled.Tools[0].Function.Name)
	assert.Equal(t, "json_object", unmarshaled.ResponseFormat.Type)
}

func TestLLMRequest_WithFileAttachments(t *testing.T) {
	request := LLMRequest{
		Provider: LLMProviderOpenAI,
		Model:    "gpt-4-vision-preview",
		Prompt:   "What's in this image?",
		Files: []LLMFileAttachment{
			{
				Data:     "base64encodedimage",
				MimeType: "image/jpeg",
				Name:     "photo.jpg",
				Detail:   "high",
			},
			{
				Data:     "base64encodedpdf",
				MimeType: "application/pdf",
				Name:     "document.pdf",
			},
		},
	}

	data, err := json.Marshal(request)
	require.NoError(t, err)

	var unmarshaled LLMRequest
	err = json.Unmarshal(data, &unmarshaled)
	require.NoError(t, err)

	assert.Len(t, unmarshaled.Files, 2)
	assert.Equal(t, "image/jpeg", unmarshaled.Files[0].MimeType)
	assert.Equal(t, "application/pdf", unmarshaled.Files[1].MimeType)
	assert.True(t, unmarshaled.Files[0].IsImage())
	assert.True(t, unmarshaled.Files[1].IsPDF())
}

// ==================== LLMResponse JSON Marshaling Tests ====================

func TestLLMResponse_JSONMarshaling(t *testing.T) {
	now := time.Now()
	response := LLMResponse{
		Content:      "Hello! How can I help you?",
		ResponseID:   "resp_123",
		Model:        "gpt-4",
		FinishReason: "stop",
		CreatedAt:    now,
		Usage: LLMUsage{
			PromptTokens:     10,
			CompletionTokens: 8,
			TotalTokens:      18,
		},
	}

	data, err := json.Marshal(response)
	require.NoError(t, err)

	var unmarshaled LLMResponse
	err = json.Unmarshal(data, &unmarshaled)
	require.NoError(t, err)

	assert.Equal(t, response.Content, unmarshaled.Content)
	assert.Equal(t, response.ResponseID, unmarshaled.ResponseID)
	assert.Equal(t, response.Model, unmarshaled.Model)
	assert.Equal(t, response.FinishReason, unmarshaled.FinishReason)
	assert.Equal(t, response.Usage.PromptTokens, unmarshaled.Usage.PromptTokens)
	assert.Equal(t, response.Usage.CompletionTokens, unmarshaled.Usage.CompletionTokens)
	assert.Equal(t, response.Usage.TotalTokens, unmarshaled.Usage.TotalTokens)
}

func TestLLMResponse_WithToolCalls(t *testing.T) {
	response := LLMResponse{
		Content:      "",
		Model:        "gpt-4",
		FinishReason: "tool_calls",
		CreatedAt:    time.Now(),
		ToolCalls: []LLMToolCall{
			{
				ID:   "call_123",
				Type: "function",
				Function: LLMFunctionCall{
					Name:      "get_weather",
					Arguments: `{"location":"Paris"}`,
				},
			},
			{
				ID:   "call_456",
				Type: "function",
				Function: LLMFunctionCall{
					Name:      "get_time",
					Arguments: `{"timezone":"UTC"}`,
				},
			},
		},
		Usage: LLMUsage{
			PromptTokens:     20,
			CompletionTokens: 15,
			TotalTokens:      35,
		},
	}

	data, err := json.Marshal(response)
	require.NoError(t, err)

	var unmarshaled LLMResponse
	err = json.Unmarshal(data, &unmarshaled)
	require.NoError(t, err)

	assert.Len(t, unmarshaled.ToolCalls, 2)
	assert.Equal(t, "call_123", unmarshaled.ToolCalls[0].ID)
	assert.Equal(t, "get_weather", unmarshaled.ToolCalls[0].Function.Name)
	assert.Equal(t, `{"location":"Paris"}`, unmarshaled.ToolCalls[0].Function.Arguments)
	assert.Equal(t, "call_456", unmarshaled.ToolCalls[1].ID)
	assert.Equal(t, "get_time", unmarshaled.ToolCalls[1].Function.Name)
}

// ==================== LLMProvider Tests ====================

func TestLLMProvider_Constants(t *testing.T) {
	// Verify provider constants exist
	assert.Equal(t, LLMProvider("openai"), LLMProviderOpenAI)
	assert.Equal(t, LLMProvider("openai-responses"), LLMProviderOpenAIResponses)
	assert.Equal(t, LLMProvider("anthropic"), LLMProviderAnthropic)
}

// ==================== LLMUsage Tests ====================

func TestLLMUsage_JSONMarshaling(t *testing.T) {
	usage := LLMUsage{
		PromptTokens:     100,
		CompletionTokens: 50,
		TotalTokens:      150,
	}

	data, err := json.Marshal(usage)
	require.NoError(t, err)

	var unmarshaled LLMUsage
	err = json.Unmarshal(data, &unmarshaled)
	require.NoError(t, err)

	assert.Equal(t, usage.PromptTokens, unmarshaled.PromptTokens)
	assert.Equal(t, usage.CompletionTokens, unmarshaled.CompletionTokens)
	assert.Equal(t, usage.TotalTokens, unmarshaled.TotalTokens)
}

// ==================== LLMResponseFormat Tests ====================

func TestLLMResponseFormat_JSONObject(t *testing.T) {
	format := LLMResponseFormat{
		Type: "json_object",
	}

	data, err := json.Marshal(format)
	require.NoError(t, err)

	var unmarshaled LLMResponseFormat
	err = json.Unmarshal(data, &unmarshaled)
	require.NoError(t, err)

	assert.Equal(t, "json_object", unmarshaled.Type)
	assert.Nil(t, unmarshaled.JSONSchema)
}

func TestLLMResponseFormat_JSONSchema(t *testing.T) {
	format := LLMResponseFormat{
		Type: "json_schema",
		JSONSchema: &LLMJSONSchema{
			Name:        "user_profile",
			Description: "User profile schema",
			Schema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"name": map[string]interface{}{
						"type": "string",
					},
					"age": map[string]interface{}{
						"type": "integer",
					},
				},
				"required": []string{"name"},
			},
			Strict: true,
		},
	}

	data, err := json.Marshal(format)
	require.NoError(t, err)

	var unmarshaled LLMResponseFormat
	err = json.Unmarshal(data, &unmarshaled)
	require.NoError(t, err)

	assert.Equal(t, "json_schema", unmarshaled.Type)
	assert.NotNil(t, unmarshaled.JSONSchema)
	assert.Equal(t, "user_profile", unmarshaled.JSONSchema.Name)
	assert.Equal(t, "User profile schema", unmarshaled.JSONSchema.Description)
	assert.True(t, unmarshaled.JSONSchema.Strict)
	assert.NotNil(t, unmarshaled.JSONSchema.Schema)
}

// ==================== LLMHostedTool Tests ====================

func TestLLMHostedTool_WebSearch(t *testing.T) {
	tool := LLMHostedTool{
		Type:              "web_search_preview",
		Domains:           []string{"example.com", "test.com"},
		SearchContextSize: "medium",
	}

	data, err := json.Marshal(tool)
	require.NoError(t, err)

	var unmarshaled LLMHostedTool
	err = json.Unmarshal(data, &unmarshaled)
	require.NoError(t, err)

	assert.Equal(t, "web_search_preview", unmarshaled.Type)
	assert.Len(t, unmarshaled.Domains, 2)
	assert.Equal(t, "medium", unmarshaled.SearchContextSize)
}

func TestLLMHostedTool_FileSearch(t *testing.T) {
	tool := LLMHostedTool{
		Type:           "file_search",
		VectorStoreIDs: []string{"vs_123", "vs_456"},
		MaxNumResults:  5,
	}

	data, err := json.Marshal(tool)
	require.NoError(t, err)

	var unmarshaled LLMHostedTool
	err = json.Unmarshal(data, &unmarshaled)
	require.NoError(t, err)

	assert.Equal(t, "file_search", unmarshaled.Type)
	assert.Len(t, unmarshaled.VectorStoreIDs, 2)
	assert.Equal(t, 5, unmarshaled.MaxNumResults)
}

func TestLLMHostedTool_CodeInterpreter(t *testing.T) {
	tool := LLMHostedTool{
		Type: "code_interpreter",
	}

	data, err := json.Marshal(tool)
	require.NoError(t, err)

	var unmarshaled LLMHostedTool
	err = json.Unmarshal(data, &unmarshaled)
	require.NoError(t, err)

	assert.Equal(t, "code_interpreter", unmarshaled.Type)
}

// ==================== LLMOutputItem Tests ====================

func TestLLMOutputItem_Message(t *testing.T) {
	item := LLMOutputItem{
		ID:     "item_123",
		Type:   "message",
		Status: "completed",
		Role:   "assistant",
		Content: []LLMOutputContent{
			{
				Type: "output_text",
				Text: "Hello! How can I help you?",
			},
		},
	}

	data, err := json.Marshal(item)
	require.NoError(t, err)

	var unmarshaled LLMOutputItem
	err = json.Unmarshal(data, &unmarshaled)
	require.NoError(t, err)

	assert.Equal(t, "item_123", unmarshaled.ID)
	assert.Equal(t, "message", unmarshaled.Type)
	assert.Equal(t, "completed", unmarshaled.Status)
	assert.Equal(t, "assistant", unmarshaled.Role)
	assert.Len(t, unmarshaled.Content, 1)
	assert.Equal(t, "output_text", unmarshaled.Content[0].Type)
	assert.Equal(t, "Hello! How can I help you?", unmarshaled.Content[0].Text)
}

func TestLLMOutputItem_FunctionCall(t *testing.T) {
	item := LLMOutputItem{
		ID:        "item_456",
		Type:      "function_call",
		Status:    "completed",
		CallID:    "call_789",
		Name:      "get_weather",
		Arguments: `{"location":"Paris"}`,
	}

	data, err := json.Marshal(item)
	require.NoError(t, err)

	var unmarshaled LLMOutputItem
	err = json.Unmarshal(data, &unmarshaled)
	require.NoError(t, err)

	assert.Equal(t, "item_456", unmarshaled.ID)
	assert.Equal(t, "function_call", unmarshaled.Type)
	assert.Equal(t, "call_789", unmarshaled.CallID)
	assert.Equal(t, "get_weather", unmarshaled.Name)
	assert.Equal(t, `{"location":"Paris"}`, unmarshaled.Arguments)
}

// ==================== Complex Integration Tests ====================

func TestLLM_ComplexRequestResponse(t *testing.T) {
	// Create a complex LLM request with all features
	request := LLMRequest{
		Provider:    LLMProviderOpenAIResponses,
		Model:       "gpt-5",
		Instruction: "You are a helpful assistant with web search capabilities",
		Prompt:      "Search for the latest news about AI",
		MaxTokens:   1000,
		Temperature: 0.8,
		Files: []LLMFileAttachment{
			{
				Data:     "base64image",
				MimeType: "image/png",
				Name:     "chart.png",
				Detail:   "high",
			},
		},
		Tools: []LLMTool{
			{
				Type: "function",
				Function: LLMFunctionTool{
					Name:        "search_database",
					Description: "Search internal database",
					Parameters: map[string]interface{}{
						"type": "object",
						"properties": map[string]interface{}{
							"query": map[string]interface{}{"type": "string"},
						},
					},
				},
			},
		},
		HostedTools: []LLMHostedTool{
			{
				Type:              "web_search_preview",
				SearchContextSize: "large",
			},
		},
	}

	// Marshal request
	requestData, err := json.Marshal(request)
	require.NoError(t, err)

	// Unmarshal request
	var unmarshaledRequest LLMRequest
	err = json.Unmarshal(requestData, &unmarshaledRequest)
	require.NoError(t, err)

	assert.Equal(t, request.Provider, unmarshaledRequest.Provider)
	assert.Equal(t, request.Model, unmarshaledRequest.Model)
	assert.Len(t, unmarshaledRequest.Files, 1)
	assert.Len(t, unmarshaledRequest.Tools, 1)
	assert.Len(t, unmarshaledRequest.HostedTools, 1)

	// Create a complex response
	response := LLMResponse{
		Content:      "Here are the latest AI news...",
		ResponseID:   "resp_789",
		Model:        "gpt-5",
		FinishReason: "stop",
		Status:       "completed",
		CreatedAt:    time.Now(),
		Usage: LLMUsage{
			PromptTokens:     150,
			CompletionTokens: 200,
			TotalTokens:      350,
		},
		OutputItems: []LLMOutputItem{
			{
				ID:     "item_1",
				Type:   "message",
				Status: "completed",
				Role:   "assistant",
				Content: []LLMOutputContent{
					{
						Type: "output_text",
						Text: "Based on my web search...",
						Annotations: []LLMOutputAnnotation{
							{
								Type:       "url_citation",
								StartIndex: 10,
								EndIndex:   20,
								URL:        "https://example.com/article",
								Title:      "AI News",
							},
						},
					},
				},
			},
		},
	}

	// Marshal response
	responseData, err := json.Marshal(response)
	require.NoError(t, err)

	// Unmarshal response
	var unmarshaledResponse LLMResponse
	err = json.Unmarshal(responseData, &unmarshaledResponse)
	require.NoError(t, err)

	assert.Equal(t, response.Content, unmarshaledResponse.Content)
	assert.Equal(t, response.ResponseID, unmarshaledResponse.ResponseID)
	assert.Equal(t, response.Status, unmarshaledResponse.Status)
	assert.Len(t, unmarshaledResponse.OutputItems, 1)
	assert.Len(t, unmarshaledResponse.OutputItems[0].Content[0].Annotations, 1)
}
