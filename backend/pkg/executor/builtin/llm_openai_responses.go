package builtin

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/smilemakc/mbflow/pkg/models"
)

// OpenAIResponsesProvider implements the LLM provider for OpenAI Responses API.
// This provider supports GPT-5, o3-mini, gpt-4.1+ and newer reasoning models.
type OpenAIResponsesProvider struct {
	apiKey  string
	baseURL string
	orgID   string
	client  *http.Client
}

// NewOpenAIResponsesProvider creates a new OpenAI Responses API provider.
func NewOpenAIResponsesProvider(apiKey, baseURL, orgID string) (*OpenAIResponsesProvider, error) {
	if apiKey == "" {
		return nil, fmt.Errorf("api_key is required for OpenAI Responses provider")
	}

	if baseURL == "" {
		baseURL = "https://api.openai.com/v1"
	}

	return &OpenAIResponsesProvider{
		apiKey:  apiKey,
		baseURL: baseURL,
		orgID:   orgID,
		client: &http.Client{
			Timeout: 120 * time.Second,
		},
	}, nil
}

// Execute executes an LLM request using OpenAI Responses API.
func (p *OpenAIResponsesProvider) Execute(ctx context.Context, req *models.LLMRequest) (*models.LLMResponse, error) {
	// Build request body for Responses API
	reqBody := p.buildRequestBody(req)

	// Marshal to JSON
	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create HTTP request to /responses endpoint
	httpReq, err := http.NewRequestWithContext(ctx, "POST", p.baseURL+"/responses", bytes.NewReader(jsonData))
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
					Provider: models.LLMProviderOpenAIResponses,
					Code:     fmt.Sprintf("%v", errorData["code"]),
					Message:  fmt.Sprintf("%v", errorData["message"]),
					Type:     fmt.Sprintf("%v", errorData["type"]),
				}
			}
		}
		return nil, fmt.Errorf("OpenAI Responses API error (status %d): %s", resp.StatusCode, string(respBody))
	}

	// Parse response
	var apiResp openAIResponsesAPIResponse
	if err := json.Unmarshal(respBody, &apiResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	// Convert to our model
	return p.convertResponse(&apiResp), nil
}

// buildRequestBody builds the OpenAI Responses API request body.
func (p *OpenAIResponsesProvider) buildRequestBody(req *models.LLMRequest) map[string]any {
	body := map[string]any{
		"model": req.Model,
	}

	// Input (can be string or structured array)
	if req.Input != nil {
		body["input"] = req.Input
	} else if req.Prompt != "" {
		// Fallback to prompt for simple text input
		body["input"] = req.Prompt
	}

	// Instructions (system message)
	if req.Instructions != "" {
		body["instructions"] = req.Instructions
	} else if req.Instruction != "" {
		body["instructions"] = req.Instruction
	}

	// Optional parameters
	if req.MaxTokens > 0 {
		body["max_output_tokens"] = req.MaxTokens
	}
	if req.Temperature > 0 {
		body["temperature"] = req.Temperature
	}
	if req.TopP > 0 {
		body["top_p"] = req.TopP
	}
	if req.PreviousResponseID != "" {
		body["previous_response_id"] = req.PreviousResponseID
	}
	if req.Background {
		body["background"] = true
	}
	if req.MaxToolCalls > 0 {
		body["max_tool_calls"] = req.MaxToolCalls
	}
	if req.Store != nil {
		body["store"] = *req.Store
	}

	// Reasoning configuration (for o3-mini, etc.)
	if req.Reasoning != nil {
		reasoning := map[string]any{}
		if req.Reasoning.Effort != "" {
			reasoning["effort"] = req.Reasoning.Effort
		}
		if len(reasoning) > 0 {
			body["reasoning"] = reasoning
		}
	}

	// Tools - combine function tools and hosted tools
	var tools []map[string]any

	// Add function tools
	if len(req.Tools) > 0 {
		for _, tool := range req.Tools {
			tools = append(tools, map[string]any{
				"type":        "function",
				"name":        tool.Function.Name,
				"description": tool.Function.Description,
				"parameters":  tool.Function.Parameters,
			})
		}
	}

	// Add hosted tools (web_search, file_search, code_interpreter)
	if len(req.HostedTools) > 0 {
		for _, htool := range req.HostedTools {
			toolDef := map[string]any{
				"type": htool.Type,
			}

			switch htool.Type {
			case "web_search_preview":
				if len(htool.Domains) > 0 {
					toolDef["domains"] = htool.Domains
				}
				if htool.SearchContextSize != "" {
					toolDef["search_context_size"] = htool.SearchContextSize
				}
			case "file_search":
				if len(htool.VectorStoreIDs) > 0 {
					toolDef["vector_store_ids"] = htool.VectorStoreIDs
				}
				if htool.MaxNumResults > 0 {
					toolDef["max_num_results"] = htool.MaxNumResults
				}
				if htool.RankingOptions != nil {
					toolDef["ranking_options"] = htool.RankingOptions
				}
			case "code_interpreter":
				// No additional config for code_interpreter
			}

			tools = append(tools, toolDef)
		}
	}

	if len(tools) > 0 {
		body["tools"] = tools
	}

	// Response format (structured outputs)
	if req.ResponseFormat != nil {
		body["text"] = map[string]any{
			"format": p.buildResponseFormat(req.ResponseFormat),
		}
	}

	return body
}

// buildResponseFormat builds the response format for structured outputs.
func (p *OpenAIResponsesProvider) buildResponseFormat(format *models.LLMResponseFormat) map[string]any {
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

// convertResponse converts OpenAI Responses API response to our model.
func (p *OpenAIResponsesProvider) convertResponse(resp *openAIResponsesAPIResponse) *models.LLMResponse {
	response := &models.LLMResponse{
		ResponseID:   resp.ID,
		Model:        resp.Model,
		Status:       resp.Status,
		FinishReason: resp.Status, // Map status to finish_reason for compatibility
		CreatedAt:    time.Unix(resp.CreatedAt, 0),
	}

	// Handle error
	if resp.Error != nil {
		response.Error = &models.LLMError{
			Provider: models.LLMProviderOpenAIResponses,
			Message:  fmt.Sprintf("%v", resp.Error),
		}
	}

	// Handle incomplete details
	if resp.IncompleteDetails != nil {
		response.IncompleteDetails = resp.IncompleteDetails
	}

	// Handle reasoning
	if resp.Reasoning != nil {
		response.Reasoning = &models.LLMReasoningInfo{
			Effort:  resp.Reasoning.Effort,
			Summary: resp.Reasoning.Summary,
		}
	}

	// Handle usage
	if resp.Usage != nil {
		response.Usage = models.LLMUsage{
			PromptTokens:     resp.Usage.InputTokens,
			CompletionTokens: resp.Usage.OutputTokens,
			TotalTokens:      resp.Usage.TotalTokens,
		}
	}

	// Parse polymorphic output items
	response.OutputItems = make([]models.LLMOutputItem, 0, len(resp.Output))

	for _, item := range resp.Output {
		outputItem := models.LLMOutputItem{
			ID:     item.ID,
			Type:   item.Type,
			Status: item.Status,
		}

		switch item.Type {
		case "message":
			outputItem.Role = item.Role
			if len(item.Content) > 0 {
				outputItem.Content = make([]models.LLMOutputContent, 0, len(item.Content))
				for _, content := range item.Content {
					contentPart := models.LLMOutputContent{
						Type: content.Type,
						Text: content.Text,
					}

					// Parse annotations
					if len(content.Annotations) > 0 {
						contentPart.Annotations = make([]models.LLMOutputAnnotation, 0, len(content.Annotations))
						for _, ann := range content.Annotations {
							annotation := models.LLMOutputAnnotation{
								Type:       ann.Type,
								StartIndex: ann.StartIndex,
								EndIndex:   ann.EndIndex,
								URL:        ann.URL,
								Title:      ann.Title,
								Index:      ann.Index,
								FileID:     ann.FileID,
								Filename:   ann.Filename,
							}
							contentPart.Annotations = append(contentPart.Annotations, annotation)
						}
					}

					outputItem.Content = append(outputItem.Content, contentPart)
				}

				// Extract first text content for backward compatibility
				if response.Content == "" && len(item.Content) > 0 && item.Content[0].Type == "output_text" {
					response.Content = item.Content[0].Text
				}
			}

		case "function_call":
			outputItem.CallID = item.CallID
			outputItem.Name = item.Name
			outputItem.Arguments = item.Arguments

			// Add to legacy ToolCalls for backward compatibility
			response.ToolCalls = append(response.ToolCalls, models.LLMToolCall{
				ID:   item.CallID,
				Type: "function",
				Function: models.LLMFunctionCall{
					Name:      item.Name,
					Arguments: item.Arguments,
				},
			})

		case "web_search_call", "file_search_call":
			outputItem.Queries = item.Queries
			outputItem.Results = item.Results
		}

		response.OutputItems = append(response.OutputItems, outputItem)
	}

	return response
}

// --- OpenAI Responses API response types ---

type openAIResponsesAPIResponse struct {
	ID                 string               `json:"id"`
	Object             string               `json:"object"`
	CreatedAt          int64                `json:"created_at"`
	Status             string               `json:"status"`
	Model              string               `json:"model"`
	Output             []responseOutputItem `json:"output"`
	PreviousResponseID string               `json:"previous_response_id"`
	Error              any                  `json:"error"`
	IncompleteDetails  map[string]any       `json:"incomplete_details"`
	Reasoning          *reasoningInfo       `json:"reasoning"`
	Usage              *usageInfo           `json:"usage"`
}

type responseOutputItem struct {
	ID        string              `json:"id"`
	Type      string              `json:"type"`
	Status    string              `json:"status"`
	Role      string              `json:"role,omitempty"`
	Content   []outputContentPart `json:"content,omitempty"`
	CallID    string              `json:"call_id,omitempty"`
	Name      string              `json:"name,omitempty"`
	Arguments string              `json:"arguments,omitempty"`
	Queries   []string            `json:"queries,omitempty"`
	Results   any                 `json:"results,omitempty"`
}

type outputContentPart struct {
	Type        string             `json:"type"`
	Text        string             `json:"text,omitempty"`
	Annotations []outputAnnotation `json:"annotations,omitempty"`
}

type outputAnnotation struct {
	Type       string `json:"type"`
	StartIndex int    `json:"start_index,omitempty"`
	EndIndex   int    `json:"end_index,omitempty"`
	URL        string `json:"url,omitempty"`
	Title      string `json:"title,omitempty"`
	Index      int    `json:"index,omitempty"`
	FileID     string `json:"file_id,omitempty"`
	Filename   string `json:"filename,omitempty"`
}

type reasoningInfo struct {
	Effort  string `json:"effort"`
	Summary string `json:"summary"`
}

type usageInfo struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
	TotalTokens  int `json:"total_tokens"`
}
