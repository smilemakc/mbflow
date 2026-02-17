package builtin

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/smilemakc/mbflow/pkg/models"
)

// GeminiProvider implements the LLM provider for Google Gemini using direct HTTP calls.
type GeminiProvider struct {
	apiKey  string
	baseURL string
	client  *http.Client
}

// NewGeminiProvider creates a new Gemini provider with the given configuration.
func NewGeminiProvider(apiKey, baseURL string) (*GeminiProvider, error) {
	if apiKey == "" {
		return nil, fmt.Errorf("api_key is required for Gemini provider")
	}

	if baseURL == "" {
		baseURL = "https://generativelanguage.googleapis.com/v1beta"
	}

	return &GeminiProvider{
		apiKey:  apiKey,
		baseURL: baseURL,
		client: &http.Client{
			Timeout: 120 * time.Second,
		},
	}, nil
}

// Execute executes an LLM request using Gemini.
func (p *GeminiProvider) Execute(ctx context.Context, req *models.LLMRequest) (*models.LLMResponse, error) {
	// Build request body
	reqBody := p.buildRequestBody(req)

	// Marshal to JSON
	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create HTTP request
	url := fmt.Sprintf("%s/models/%s:generateContent", p.baseURL, req.Model)
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-goog-api-key", p.apiKey)

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
		var errorResp geminiErrorResponse
		if err := json.Unmarshal(respBody, &errorResp); err == nil && errorResp.Error.Message != "" {
			return nil, &models.LLMError{
				Provider: models.LLMProviderGemini,
				Code:     fmt.Sprintf("%d", errorResp.Error.Code),
				Message:  errorResp.Error.Message,
				Type:     errorResp.Error.Status,
			}
		}
		return nil, fmt.Errorf("Gemini API error (status %d): %s", resp.StatusCode, string(respBody))
	}

	// Parse response
	var apiResp geminiGenerateContentResponse
	if err := json.Unmarshal(respBody, &apiResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	// Convert to our model
	return p.convertResponse(&apiResp, req), nil
}

// buildRequestBody builds the Gemini API request body.
func (p *GeminiProvider) buildRequestBody(req *models.LLMRequest) map[string]any {
	body := map[string]any{}

	// System instruction
	if req.Instruction != "" {
		body["systemInstruction"] = map[string]any{
			"parts": []map[string]any{
				{
					"text": req.Instruction,
				},
			},
		}
	}

	// Build contents (user message with multimodal support)
	userContent := p.buildUserContent(req)
	body["contents"] = []map[string]any{
		{
			"role":  "user",
			"parts": userContent,
		},
	}

	// Build generation config
	generationConfig := map[string]any{}

	if req.Temperature > 0 {
		generationConfig["temperature"] = req.Temperature
	}
	if req.TopP > 0 {
		generationConfig["topP"] = req.TopP
	}
	if req.MaxTokens > 0 {
		generationConfig["maxOutputTokens"] = req.MaxTokens
	}
	if len(req.StopSequences) > 0 {
		generationConfig["stopSequences"] = req.StopSequences
	}

	// Response format handling
	if req.ResponseFormat != nil {
		if req.ResponseFormat.Type == "json_object" {
			generationConfig["responseMimeType"] = "application/json"
		} else if req.ResponseFormat.Type == "json_schema" && req.ResponseFormat.JSONSchema != nil {
			generationConfig["responseMimeType"] = "application/json"
			generationConfig["responseSchema"] = req.ResponseFormat.JSONSchema.Schema
		}
	}

	if len(generationConfig) > 0 {
		body["generationConfig"] = generationConfig
	}

	// Tools (function calling)
	if len(req.Tools) > 0 {
		tools := p.buildTools(req.Tools)
		body["tools"] = tools
	}

	return body
}

// buildUserContent builds the user message content with multimodal support.
func (p *GeminiProvider) buildUserContent(req *models.LLMRequest) []map[string]any {
	parts := []map[string]any{}

	// Add text
	if req.Prompt != "" {
		parts = append(parts, map[string]any{
			"text": req.Prompt,
		})
	}

	// Add base64 encoded files (inline_data)
	for _, file := range req.Files {
		if !file.IsSupported() {
			continue
		}

		if file.IsImage() {
			parts = append(parts, map[string]any{
				"inline_data": map[string]any{
					"mime_type": file.MimeType,
					"data":      file.Data,
				},
			})
		}
	}

	// Add images from URLs (file_data)
	for _, imageURL := range req.ImageURLs {
		mimeType := "image/jpeg"
		if strings.HasSuffix(strings.ToLower(imageURL), ".png") {
			mimeType = "image/png"
		} else if strings.HasSuffix(strings.ToLower(imageURL), ".gif") {
			mimeType = "image/gif"
		} else if strings.HasSuffix(strings.ToLower(imageURL), ".webp") {
			mimeType = "image/webp"
		}

		parts = append(parts, map[string]any{
			"file_data": map[string]any{
				"mime_type": mimeType,
				"file_uri":  imageURL,
			},
		})
	}

	return parts
}

// buildTools builds the tools array for function calling.
func (p *GeminiProvider) buildTools(tools []models.LLMTool) []map[string]any {
	functionDeclarations := make([]map[string]any, len(tools))

	for i, tool := range tools {
		functionDeclarations[i] = map[string]any{
			"name":        tool.Function.Name,
			"description": tool.Function.Description,
			"parameters":  tool.Function.Parameters,
		}
	}

	return []map[string]any{
		{
			"functionDeclarations": functionDeclarations,
		},
	}
}

// convertResponse converts Gemini API response to our model.
func (p *GeminiProvider) convertResponse(resp *geminiGenerateContentResponse, req *models.LLMRequest) *models.LLMResponse {
	if len(resp.Candidates) == 0 {
		model := resp.ModelVersion
		if model == "" {
			model = req.Model
		}
		return &models.LLMResponse{
			Model:        model,
			FinishReason: "error",
			CreatedAt:    time.Now(),
		}
	}

	candidate := resp.Candidates[0]

	// Extract text content and tool calls
	var content string
	var toolCalls []models.LLMToolCall

	for _, part := range candidate.Content.Parts {
		if part.Text != "" {
			content = part.Text
		}

		if part.FunctionCall != nil {
			argsJSON, err := json.Marshal(part.FunctionCall.Args)
			if err != nil {
				argsJSON = []byte("{}")
			}

			toolCalls = append(toolCalls, models.LLMToolCall{
				ID:   part.FunctionCall.Name,
				Type: "function",
				Function: models.LLMFunctionCall{
					Name:      part.FunctionCall.Name,
					Arguments: string(argsJSON),
				},
			})
		}
	}

	// Normalize finish reason
	finishReason := p.normalizeFinishReason(candidate.FinishReason)
	if len(toolCalls) > 0 {
		finishReason = "tool_calls"
	}

	// Determine model
	model := resp.ModelVersion
	if model == "" {
		model = req.Model
	}

	response := &models.LLMResponse{
		Content:      content,
		ResponseID:   resp.ResponseID,
		Model:        model,
		FinishReason: finishReason,
		CreatedAt:    time.Now(),
		Usage: models.LLMUsage{
			PromptTokens:     resp.UsageMetadata.PromptTokenCount,
			CompletionTokens: resp.UsageMetadata.CandidatesTokenCount,
			TotalTokens:      resp.UsageMetadata.TotalTokenCount,
		},
		ToolCalls: toolCalls,
	}

	return response
}

// normalizeFinishReason normalizes Gemini finish reasons to our standard format.
func (p *GeminiProvider) normalizeFinishReason(reason string) string {
	switch strings.ToUpper(reason) {
	case "STOP":
		return "stop"
	case "MAX_TOKENS":
		return "length"
	case "SAFETY":
		return "content_filter"
	default:
		return strings.ToLower(reason)
	}
}

// Gemini API response types
type geminiGenerateContentResponse struct {
	Candidates    []geminiCandidate   `json:"candidates"`
	UsageMetadata geminiUsageMetadata `json:"usageMetadata"`
	ModelVersion  string              `json:"modelVersion"`
	ResponseID    string              `json:"responseId"`
}

type geminiCandidate struct {
	Content      geminiContent `json:"content"`
	FinishReason string        `json:"finishReason"`
}

type geminiContent struct {
	Role  string       `json:"role"`
	Parts []geminiPart `json:"parts"`
}

type geminiPart struct {
	Text         string              `json:"text,omitempty"`
	FunctionCall *geminiFunctionCall `json:"functionCall,omitempty"`
}

type geminiFunctionCall struct {
	Name string         `json:"name"`
	Args map[string]any `json:"args"`
}

type geminiUsageMetadata struct {
	PromptTokenCount     int `json:"promptTokenCount"`
	CandidatesTokenCount int `json:"candidatesTokenCount"`
	TotalTokenCount      int `json:"totalTokenCount"`
}

type geminiErrorResponse struct {
	Error geminiError `json:"error"`
}

type geminiError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Status  string `json:"status"`
}
