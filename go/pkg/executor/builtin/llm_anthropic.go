package builtin

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/smilemakc/mbflow/go/pkg/models"
	"github.com/smilemakc/mbflow/go/pkg/netguard"
)

const (
	anthropicDefaultBaseURL   = "https://api.anthropic.com"
	anthropicAPIVersion       = "2023-06-01"
	anthropicDefaultMaxTokens = 4096
	// anthropicStructuredTool is the tool forced to produce structured output
	// when no schema name is given.
	anthropicStructuredTool = "structured_output"
)

// AnthropicProvider implements the LLM provider for the Anthropic Messages API.
type AnthropicProvider struct {
	apiKey  string
	baseURL string
	client  *http.Client
}

// NewAnthropicProvider creates an Anthropic provider. baseURL defaults to
// https://api.anthropic.com (without the /v1 suffix).
func NewAnthropicProvider(apiKey, baseURL string) (*AnthropicProvider, error) {
	if apiKey == "" {
		return nil, fmt.Errorf("api_key is required for Anthropic provider")
	}
	if baseURL == "" {
		baseURL = anthropicDefaultBaseURL
	}
	return &AnthropicProvider{
		apiKey:  apiKey,
		baseURL: strings.TrimSuffix(strings.TrimSuffix(baseURL, "/"), "/v1"),
		client:  &http.Client{Timeout: 300 * time.Second},
	}, nil
}

// Execute sends req to the Messages API.
func (p *AnthropicProvider) Execute(ctx context.Context, req *models.LLMRequest) (*models.LLMResponse, error) {
	body, structuredTool := p.buildRequestBody(req)

	jsonData, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, p.baseURL+"/v1/messages", bytes.NewReader(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-api-key", p.apiKey)
	httpReq.Header.Set("anthropic-version", anthropicAPIVersion)

	resp, err := netguard.Guard(p.client).Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	respBody, err := netguard.ReadAllLimited(resp.Body, netguard.MaxResponseBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		var errResp anthropicErrorResponse
		if err := json.Unmarshal(respBody, &errResp); err == nil && errResp.Error.Message != "" {
			return nil, &models.LLMError{
				Provider: models.LLMProviderAnthropic,
				Code:     fmt.Sprintf("%d", resp.StatusCode),
				Message:  errResp.Error.Message,
				Type:     errResp.Error.Type,
			}
		}
		return nil, fmt.Errorf("anthropic API error (status %d): %s", resp.StatusCode, string(respBody))
	}

	var apiResp anthropicMessageResponse
	if err := json.Unmarshal(respBody, &apiResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}
	return p.convertResponse(&apiResp, req, structuredTool), nil
}

// buildRequestBody builds the Messages API body. It returns the name of the
// tool forced to produce structured output, if any.
func (p *AnthropicProvider) buildRequestBody(req *models.LLMRequest) (map[string]any, string) {
	maxTokens := req.MaxTokens
	if maxTokens <= 0 {
		maxTokens = anthropicDefaultMaxTokens
	}
	body := map[string]any{
		"model":      req.Model,
		"max_tokens": maxTokens,
	}

	var system []string
	var messages []map[string]any
	if len(req.Messages) > 0 {
		system, messages = p.buildHistory(req)
	} else {
		if req.Instruction != "" {
			system = append(system, req.Instruction)
		}
		messages = []map[string]any{{"role": "user", "content": p.userContent(req, req.Prompt)}}
	}

	if req.Temperature > 0 {
		body["temperature"] = req.Temperature
	}
	if req.TopP > 0 {
		body["top_p"] = req.TopP
	}
	if len(req.StopSequences) > 0 {
		body["stop_sequences"] = req.StopSequences
	}

	tools := make([]map[string]any, 0, len(req.Tools))
	for _, t := range req.Tools {
		tools = append(tools, map[string]any{
			"name":         t.Function.Name,
			"description":  t.Function.Description,
			"input_schema": anthropicInputSchema(t.Function.Parameters),
		})
	}

	// Structured output: a JSON schema is enforced by forcing a tool whose
	// input is the schema (when the request has no tools of its own); plain
	// JSON is requested through the system prompt.
	structuredTool := ""
	if f := req.ResponseFormat; f != nil {
		switch {
		case f.Type == "json_schema" && f.JSONSchema != nil && len(tools) == 0:
			structuredTool = f.JSONSchema.Name
			if structuredTool == "" {
				structuredTool = anthropicStructuredTool
			}
			description := f.JSONSchema.Description
			if description == "" {
				description = "Return the answer in the required structure."
			}
			tools = append(tools, map[string]any{
				"name":         structuredTool,
				"description":  description,
				"input_schema": anthropicInputSchema(f.JSONSchema.Schema),
			})
			body["tool_choice"] = map[string]any{"type": "tool", "name": structuredTool}
		case f.Type == "json_object" || f.Type == "json_schema":
			instruction := "Respond with a single valid JSON object and nothing else."
			if f.JSONSchema != nil && f.JSONSchema.Schema != nil {
				if schema, err := json.Marshal(f.JSONSchema.Schema); err == nil {
					instruction += " It must match this JSON schema: " + string(schema)
				}
			}
			system = append(system, instruction)
		}
	}

	if len(tools) > 0 {
		body["tools"] = tools
	}
	if len(system) > 0 {
		body["system"] = strings.Join(system, "\n\n")
	}
	body["messages"] = messages
	return body, structuredTool
}

// anthropicInputSchema returns a valid input_schema (Anthropic requires an
// object schema).
func anthropicInputSchema(schema map[string]any) map[string]any {
	if len(schema) == 0 {
		return map[string]any{"type": "object", "properties": map[string]any{}}
	}
	if _, ok := schema["type"]; !ok {
		out := make(map[string]any, len(schema)+1)
		for k, v := range schema {
			out[k] = v
		}
		out["type"] = "object"
		return out
	}
	return schema
}

// buildHistory converts req.Messages (auto tool-calling mode). System messages
// form the system prompt, the attachments of the request go to the first user
// message, and results of the calls of one turn are sent together.
func (p *AnthropicProvider) buildHistory(req *models.LLMRequest) (system []string, messages []map[string]any) {
	attached := false
	var results []map[string]any
	flush := func() {
		if len(results) > 0 {
			messages = append(messages, map[string]any{"role": "user", "content": results})
			results = nil
		}
	}
	for _, msg := range req.Messages {
		if msg.Role != "tool" {
			flush()
		}
		switch msg.Role {
		case "system":
			if msg.Content != "" {
				system = append(system, msg.Content)
			}
		case "assistant":
			var content []map[string]any
			if msg.Content != "" {
				content = append(content, map[string]any{"type": "text", "text": msg.Content})
			}
			for _, tc := range msg.ToolCalls {
				input := map[string]any{}
				if tc.Function.Arguments != "" {
					_ = json.Unmarshal([]byte(tc.Function.Arguments), &input)
				}
				content = append(content, map[string]any{
					"type":  "tool_use",
					"id":    tc.ID,
					"name":  tc.Function.Name,
					"input": input,
				})
			}
			if len(content) > 0 {
				messages = append(messages, map[string]any{"role": "assistant", "content": content})
			}
		case "tool":
			results = append(results, map[string]any{
				"type":        "tool_result",
				"tool_use_id": msg.ToolCallID,
				"content":     msg.Content,
			})
		default:
			content := []map[string]any{{"type": "text", "text": msg.Content}}
			if !attached {
				attached = true
				content = p.userContent(req, msg.Content)
			}
			messages = append(messages, map[string]any{"role": "user", "content": content})
		}
	}
	flush()
	return system, messages
}

// userContent builds the content blocks of a user message: attached images and
// PDFs first (as Anthropic recommends), then the text.
func (p *AnthropicProvider) userContent(req *models.LLMRequest, text string) []map[string]any {
	var content []map[string]any
	for _, file := range req.Files {
		switch {
		case file.IsImage():
			content = append(content, map[string]any{
				"type":   "image",
				"source": map[string]any{"type": "base64", "media_type": file.MimeType, "data": file.Data},
			})
		case file.IsPDF():
			content = append(content, map[string]any{
				"type":   "document",
				"source": map[string]any{"type": "base64", "media_type": file.MimeType, "data": file.Data},
			})
		}
	}
	for _, imageURL := range req.ImageURLs {
		content = append(content, map[string]any{
			"type":   "image",
			"source": map[string]any{"type": "url", "url": imageURL},
		})
	}
	if text != "" || len(content) == 0 {
		content = append(content, map[string]any{"type": "text", "text": text})
	}
	return content
}

// convertResponse maps a Messages API response to LLMResponse. When
// structuredTool is set, the input of that forced tool call is the content.
func (p *AnthropicProvider) convertResponse(resp *anthropicMessageResponse, req *models.LLMRequest, structuredTool string) *models.LLMResponse {
	var texts []string
	var toolCalls []models.LLMToolCall
	structured := ""
	for _, block := range resp.Content {
		switch block.Type {
		case "text":
			if block.Text != "" {
				texts = append(texts, block.Text)
			}
		case "tool_use":
			args := string(block.Input)
			if args == "" || args == "null" {
				args = "{}"
			}
			if structuredTool != "" && block.Name == structuredTool {
				structured = args
				continue
			}
			toolCalls = append(toolCalls, models.LLMToolCall{
				ID:       block.ID,
				Type:     "function",
				Function: models.LLMFunctionCall{Name: block.Name, Arguments: args},
			})
		}
	}

	content := strings.Join(texts, "\n")
	finishReason := p.normalizeStopReason(resp.StopReason)
	if structured != "" {
		content = structured
		finishReason = "stop"
	}
	if len(toolCalls) > 0 {
		finishReason = "tool_calls"
	}

	model := resp.Model
	if model == "" {
		model = req.Model
	}
	return &models.LLMResponse{
		Content:      content,
		ResponseID:   resp.ID,
		Model:        model,
		FinishReason: finishReason,
		CreatedAt:    time.Now(),
		Usage: models.LLMUsage{
			PromptTokens:     resp.Usage.InputTokens,
			CompletionTokens: resp.Usage.OutputTokens,
			TotalTokens:      resp.Usage.InputTokens + resp.Usage.OutputTokens,
		},
		ToolCalls: toolCalls,
	}
}

// normalizeStopReason maps Anthropic stop reasons to the common finish reasons.
func (p *AnthropicProvider) normalizeStopReason(reason string) string {
	switch reason {
	case "end_turn", "stop_sequence":
		return "stop"
	case "max_tokens":
		return "length"
	case "tool_use":
		return "tool_calls"
	case "refusal":
		return "content_filter"
	default:
		return reason
	}
}

type anthropicMessageResponse struct {
	ID         string                  `json:"id"`
	Model      string                  `json:"model"`
	Content    []anthropicContentBlock `json:"content"`
	StopReason string                  `json:"stop_reason"`
	Usage      anthropicUsage          `json:"usage"`
}

type anthropicContentBlock struct {
	Type  string          `json:"type"`
	Text  string          `json:"text,omitempty"`
	ID    string          `json:"id,omitempty"`
	Name  string          `json:"name,omitempty"`
	Input json.RawMessage `json:"input,omitempty"`
}

type anthropicUsage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
}

type anthropicErrorResponse struct {
	Error struct {
		Type    string `json:"type"`
		Message string `json:"message"`
	} `json:"error"`
}
