package builtin

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/smilemakc/mbflow/go/pkg/models"
)

// OpenAIProvider implements the LLM provider for OpenAI using direct HTTP calls.
type OpenAIProvider struct {
	apiKey  string
	baseURL string
	orgID   string
	client  *http.Client
}

// NewOpenAIProvider creates a new OpenAI provider with the given configuration.
func NewOpenAIProvider(apiKey, baseURL, orgID string) (*OpenAIProvider, error) {
	if apiKey == "" {
		return nil, fmt.Errorf("api_key is required for OpenAI provider")
	}

	if baseURL == "" {
		baseURL = "https://api.openai.com/v1"
	}

	return &OpenAIProvider{
		apiKey:  apiKey,
		baseURL: baseURL,
		orgID:   orgID,
		client: &http.Client{
			Timeout: 120 * time.Second,
		},
	}, nil
}

// Execute executes an LLM request using OpenAI.
func (p *OpenAIProvider) Execute(ctx context.Context, req *models.LLMRequest) (*models.LLMResponse, error) {
	// Build request body
	reqBody := p.buildRequestBody(req)

	// Marshal to JSON
	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create HTTP request
	httpReq, err := http.NewRequestWithContext(ctx, "POST", p.baseURL+"/chat/completions", bytes.NewReader(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+p.apiKey)

	if p.orgID != "" {
		httpReq.Header.Set("OpenAI-Organization", p.orgID)
	}

	// Execute request
	resp, err := p.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	// Read response body
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// Check for errors
	if resp.StatusCode != http.StatusOK {
		var errorResp map[string]any
		if err := json.Unmarshal(respBody, &errorResp); err == nil {
			if errorData, ok := errorResp["error"].(map[string]any); ok {
				return nil, &models.LLMError{
					Provider: models.LLMProviderOpenAI,
					Code:     fmt.Sprintf("%v", errorData["code"]),
					Message:  fmt.Sprintf("%v", errorData["message"]),
					Type:     fmt.Sprintf("%v", errorData["type"]),
				}
			}
		}
		return nil, fmt.Errorf("OpenAI API error (status %d): %s", resp.StatusCode, string(respBody))
	}

	// Parse response
	var apiResp openAIChatCompletionResponse
	if err := json.Unmarshal(respBody, &apiResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	// Convert to our model
	return p.convertResponse(&apiResp), nil
}

// buildRequestBody builds the OpenAI API request body.
func (p *OpenAIProvider) buildRequestBody(req *models.LLMRequest) map[string]any {
	body := map[string]any{
		"model": req.Model,
	}

	// Build messages
	messages := []map[string]any{}

	// System message (instruction)
	if req.Instruction != "" {
		messages = append(messages, map[string]any{
			"role":    "system",
			"content": req.Instruction,
		})
	}

	// User message with multimodal support
	userContent := p.buildUserContent(req)
	messages = append(messages, map[string]any{
		"role":    "user",
		"content": userContent,
	})

	body["messages"] = messages

	// Optional parameters
	if req.MaxTokens > 0 {
		body["max_tokens"] = req.MaxTokens
	}
	if req.Temperature > 0 {
		body["temperature"] = req.Temperature
	}
	if req.TopP > 0 {
		body["top_p"] = req.TopP
	}
	if req.FrequencyPenalty != 0 {
		body["frequency_penalty"] = req.FrequencyPenalty
	}
	if req.PresencePenalty != 0 {
		body["presence_penalty"] = req.PresencePenalty
	}
	if len(req.StopSequences) > 0 {
		body["stop"] = req.StopSequences
	}

	// Tools (function calling)
	if len(req.Tools) > 0 {
		tools := p.buildTools(req.Tools)
		body["tools"] = tools
	}

	// Response format
	if req.ResponseFormat != nil {
		body["response_format"] = p.buildResponseFormat(req.ResponseFormat)
	}

	return body
}

// buildUserContent builds the user message content with multimodal support.
func (p *OpenAIProvider) buildUserContent(req *models.LLMRequest) any {
	// If no images or files, just return text
	if len(req.ImageURLs) == 0 && len(req.ImageIDs) == 0 && len(req.Files) == 0 {
		return req.Prompt
	}

	// Build multimodal content array
	content := []map[string]any{}

	// Add text
	if req.Prompt != "" {
		content = append(content, map[string]any{
			"type": "text",
			"text": req.Prompt,
		})
	}

	// Add images from URLs
	for _, imageURL := range req.ImageURLs {
		content = append(content, map[string]any{
			"type": "image_url",
			"image_url": map[string]any{
				"url": imageURL,
			},
		})
	}

	// Add base64 encoded files
	for _, file := range req.Files {
		if !file.IsSupported() {
			continue // Skip unsupported file types
		}

		if file.IsImage() {
			// Images use image_url with data URL format
			detail := file.Detail
			if detail == "" {
				detail = "auto"
			}
			content = append(content, map[string]any{
				"type": "image_url",
				"image_url": map[string]any{
					"url":    "data:" + file.MimeType + ";base64," + file.Data,
					"detail": detail,
				},
			})
		} else if file.IsPDF() {
			// PDFs use file content type (supported by gpt-4o, gpt-4o-mini)
			content = append(content, map[string]any{
				"type": "file",
				"file": map[string]any{
					"filename": file.Name,
					"file_data": map[string]any{
						"mime_type": file.MimeType,
						"data":      file.Data,
					},
				},
			})
		}
	}

	// Note: Image IDs and File IDs would require additional API calls to get URLs
	// For simplicity, we're omitting them here. In production, you'd fetch the file URLs.

	return content
}

// buildTools builds the tools array for function calling.
func (p *OpenAIProvider) buildTools(tools []models.LLMTool) []map[string]any {
	result := make([]map[string]any, len(tools))

	for i, tool := range tools {
		result[i] = map[string]any{
			"type": "function",
			"function": map[string]any{
				"name":        tool.Function.Name,
				"description": tool.Function.Description,
				"parameters":  tool.Function.Parameters,
			},
		}
	}

	return result
}

// buildResponseFormat builds the response_format parameter.
func (p *OpenAIProvider) buildResponseFormat(format *models.LLMResponseFormat) map[string]any {
	result := map[string]any{
		"type": format.Type,
	}

	if format.Type == "json_schema" && format.JSONSchema != nil {
		result["json_schema"] = map[string]any{
			"name":        format.JSONSchema.Name,
			"description": format.JSONSchema.Description,
			"schema":      format.JSONSchema.Schema,
			"strict":      format.JSONSchema.Strict,
		}
	}

	return result
}

// convertResponse converts OpenAI API response to our model.
func (p *OpenAIProvider) convertResponse(resp *openAIChatCompletionResponse) *models.LLMResponse {
	if len(resp.Choices) == 0 {
		return &models.LLMResponse{
			Model:        resp.Model,
			FinishReason: "error",
			CreatedAt:    time.Now(),
		}
	}

	choice := resp.Choices[0]

	response := &models.LLMResponse{
		Content:      choice.Message.Content,
		ResponseID:   resp.ID,
		Model:        resp.Model,
		FinishReason: choice.FinishReason,
		CreatedAt:    time.Unix(resp.Created, 0),
		Usage: models.LLMUsage{
			PromptTokens:     resp.Usage.PromptTokens,
			CompletionTokens: resp.Usage.CompletionTokens,
			TotalTokens:      resp.Usage.TotalTokens,
		},
	}

	// Convert tool calls
	if len(choice.Message.ToolCalls) > 0 {
		response.ToolCalls = make([]models.LLMToolCall, len(choice.Message.ToolCalls))
		for i, tc := range choice.Message.ToolCalls {
			response.ToolCalls[i] = models.LLMToolCall{
				ID:   tc.ID,
				Type: tc.Type,
				Function: models.LLMFunctionCall{
					Name:      tc.Function.Name,
					Arguments: tc.Function.Arguments,
				},
			}
		}
	}

	return response
}

// OpenAI API response types
type openAIChatCompletionResponse struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	Model   string `json:"model"`
	Choices []struct {
		Index   int `json:"index"`
		Message struct {
			Role      string `json:"role"`
			Content   string `json:"content"`
			ToolCalls []struct {
				ID       string `json:"id"`
				Type     string `json:"type"`
				Function struct {
					Name      string `json:"name"`
					Arguments string `json:"arguments"`
				} `json:"function"`
			} `json:"tool_calls,omitempty"`
		} `json:"message"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
}
