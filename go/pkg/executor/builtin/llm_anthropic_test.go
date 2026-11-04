package builtin

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/smilemakc/mbflow/go/pkg/models"
)

const anthropicTextResponse = `{
  "id": "msg_1", "type": "message", "role": "assistant", "model": "claude-test",
  "content": [{"type": "text", "text": "Hello"}, {"type": "text", "text": "world"}],
  "stop_reason": "end_turn",
  "usage": {"input_tokens": 10, "output_tokens": 5}
}`

func TestNewAnthropicProvider(t *testing.T) {
	_, err := NewAnthropicProvider("", "")
	require.Error(t, err)

	p, err := NewAnthropicProvider("k", "")
	require.NoError(t, err)
	assert.Equal(t, anthropicDefaultBaseURL, p.baseURL)

	p, err = NewAnthropicProvider("k", "https://proxy.example/v1/")
	require.NoError(t, err)
	assert.Equal(t, "https://proxy.example", p.baseURL, "a trailing /v1 is not duplicated")
}

func TestAnthropicProvider_TextRequest(t *testing.T) {
	srv := rcNewRecordingServer(t, 200, anthropicTextResponse)
	p, err := NewAnthropicProvider("secret", srv.URL)
	require.NoError(t, err)

	resp, err := p.Execute(context.Background(), &models.LLMRequest{
		Model:         "claude-test",
		Instruction:   "Be brief",
		Prompt:        "Say hello",
		Temperature:   0.2,
		TopP:          0.9,
		StopSequences: []string{"END"},
		Files: []models.LLMFileAttachment{
			{MimeType: "image/png", Data: "aW1n"},
			{MimeType: "application/pdf", Data: "cGRm"},
			{MimeType: "text/plain", Data: "x"},
		},
		ImageURLs: []string{"https://example.com/a.png"},
	})
	require.NoError(t, err)

	assert.Equal(t, "Hello\nworld", resp.Content)
	assert.Equal(t, "stop", resp.FinishReason)
	assert.Equal(t, "msg_1", resp.ResponseID)
	assert.Equal(t, models.LLMUsage{PromptTokens: 10, CompletionTokens: 5, TotalTokens: 15}, resp.Usage)

	assert.Equal(t, "/v1/messages", srv.paths[0])
	h := srv.headers[0]
	assert.Equal(t, "secret", h.Get("x-api-key"))
	assert.Equal(t, anthropicAPIVersion, h.Get("anthropic-version"))

	body := srv.lastBody()
	assert.Equal(t, "claude-test", body["model"])
	assert.EqualValues(t, anthropicDefaultMaxTokens, body["max_tokens"])
	assert.Equal(t, "Be brief", body["system"])
	assert.EqualValues(t, 0.2, body["temperature"])
	assert.EqualValues(t, 0.9, body["top_p"])
	assert.Equal(t, []any{"END"}, body["stop_sequences"])
	assert.NotContains(t, body, "tools")

	msgs := body["messages"].([]any)
	require.Len(t, msgs, 1)
	content := msgs[0].(map[string]any)["content"].([]any)
	require.Len(t, content, 4, "image, pdf, image url, text; unsupported files are skipped")
	assert.Equal(t, "image", content[0].(map[string]any)["type"])
	assert.Equal(t, "document", content[1].(map[string]any)["type"])
	assert.Equal(t, map[string]any{"type": "url", "url": "https://example.com/a.png"}, content[2].(map[string]any)["source"])
	assert.Equal(t, map[string]any{"type": "text", "text": "Say hello"}, content[3])
}

func TestAnthropicProvider_ToolCalls(t *testing.T) {
	srv := rcNewRecordingServer(t, 200, `{
	  "id": "msg_2", "model": "claude-test",
	  "content": [
	    {"type": "text", "text": "Let me check."},
	    {"type": "tool_use", "id": "toolu_1", "name": "get_weather", "input": {"city": "Paris"}},
	    {"type": "tool_use", "id": "toolu_2", "name": "now"}
	  ],
	  "stop_reason": "tool_use",
	  "usage": {"input_tokens": 3, "output_tokens": 4}
	}`)
	p, _ := NewAnthropicProvider("k", srv.URL)

	resp, err := p.Execute(context.Background(), &models.LLMRequest{
		Model:     "claude-test",
		Prompt:    "Weather?",
		MaxTokens: 100,
		Tools: []models.LLMTool{
			{Type: "function", Function: models.LLMFunctionTool{
				Name: "get_weather", Description: "Weather by city",
				Parameters: map[string]any{"type": "object", "properties": map[string]any{"city": map[string]any{"type": "string"}}},
			}},
			{Type: "function", Function: models.LLMFunctionTool{Name: "now"}},
		},
	})
	require.NoError(t, err)

	assert.Equal(t, "tool_calls", resp.FinishReason)
	require.Len(t, resp.ToolCalls, 2)
	assert.Equal(t, "toolu_1", resp.ToolCalls[0].ID)
	assert.Equal(t, "get_weather", resp.ToolCalls[0].Function.Name)
	assert.JSONEq(t, `{"city":"Paris"}`, resp.ToolCalls[0].Function.Arguments)
	assert.Equal(t, "{}", resp.ToolCalls[1].Function.Arguments)

	body := srv.lastBody()
	assert.EqualValues(t, 100, body["max_tokens"])
	tools := body["tools"].([]any)
	require.Len(t, tools, 2)
	assert.Equal(t, "get_weather", tools[0].(map[string]any)["name"])
	assert.Equal(t, map[string]any{"type": "object", "properties": map[string]any{}}, tools[1].(map[string]any)["input_schema"],
		"tools without parameters get an empty object schema")
}

func TestAnthropicProvider_History(t *testing.T) {
	srv := rcNewRecordingServer(t, 200, anthropicTextResponse)
	p, _ := NewAnthropicProvider("k", srv.URL)

	_, err := p.Execute(context.Background(), &models.LLMRequest{
		Model: "claude-test",
		Files: []models.LLMFileAttachment{{MimeType: "image/jpeg", Data: "aW1n"}},
		Messages: []models.LLMMessage{
			{Role: "system", Content: "sys"},
			{Role: "user", Content: "question"},
			{Role: "assistant", Content: "thinking", ToolCalls: []models.LLMToolCall{
				{ID: "t1", Function: models.LLMFunctionCall{Name: "a", Arguments: `{"x":1}`}},
				{ID: "t2", Function: models.LLMFunctionCall{Name: "b"}},
			}},
			{Role: "tool", ToolCallID: "t1", Name: "a", Content: `{"ok":true}`},
			{Role: "tool", ToolCallID: "t2", Name: "b", Content: "done"},
			{Role: "assistant"},
			{Role: "user", Content: "thanks"},
		},
	})
	require.NoError(t, err)

	body := srv.lastBody()
	assert.Equal(t, "sys", body["system"])
	msgs := body["messages"].([]any)
	require.Len(t, msgs, 4, "empty assistant messages are dropped, tool results grouped")

	first := msgs[0].(map[string]any)
	assert.Equal(t, "user", first["role"])
	firstContent := first["content"].([]any)
	require.Len(t, firstContent, 2, "attachments go to the first user message")
	assert.Equal(t, "image", firstContent[0].(map[string]any)["type"])

	assistant := msgs[1].(map[string]any)
	assert.Equal(t, "assistant", assistant["role"])
	blocks := assistant["content"].([]any)
	require.Len(t, blocks, 3)
	assert.Equal(t, map[string]any{"type": "tool_use", "id": "t1", "name": "a", "input": map[string]any{"x": float64(1)}}, blocks[1])
	assert.Equal(t, map[string]any{}, blocks[2].(map[string]any)["input"])

	results := msgs[2].(map[string]any)
	assert.Equal(t, "user", results["role"])
	resultBlocks := results["content"].([]any)
	require.Len(t, resultBlocks, 2)
	assert.Equal(t, map[string]any{"type": "tool_result", "tool_use_id": "t2", "content": "done"}, resultBlocks[1])

	last := msgs[3].(map[string]any)
	assert.Equal(t, []any{map[string]any{"type": "text", "text": "thanks"}}, last["content"])
}

func TestAnthropicProvider_StructuredOutput(t *testing.T) {
	schema := map[string]any{"type": "object", "properties": map[string]any{"score": map[string]any{"type": "number"}}}

	t.Run("json_schema forces a tool", func(t *testing.T) {
		srv := rcNewRecordingServer(t, 200, `{
		  "id": "m", "model": "claude-test",
		  "content": [{"type": "tool_use", "id": "x", "name": "rating", "input": {"score": 7}}],
		  "stop_reason": "tool_use", "usage": {}
		}`)
		p, _ := NewAnthropicProvider("k", srv.URL)
		resp, err := p.Execute(context.Background(), &models.LLMRequest{
			Model: "claude-test", Prompt: "rate",
			ResponseFormat: &models.LLMResponseFormat{Type: "json_schema", JSONSchema: &models.LLMJSONSchema{Name: "rating", Schema: schema}},
		})
		require.NoError(t, err)
		assert.JSONEq(t, `{"score":7}`, resp.Content)
		assert.Equal(t, "stop", resp.FinishReason)
		assert.Empty(t, resp.ToolCalls)

		body := srv.lastBody()
		assert.Equal(t, map[string]any{"type": "tool", "name": "rating"}, body["tool_choice"])
		tool := body["tools"].([]any)[0].(map[string]any)
		assert.Equal(t, "rating", tool["name"])
		assert.NotEmpty(t, tool["description"])
	})

	t.Run("json_schema without a name", func(t *testing.T) {
		srv := rcNewRecordingServer(t, 200, anthropicTextResponse)
		p, _ := NewAnthropicProvider("k", srv.URL)
		_, err := p.Execute(context.Background(), &models.LLMRequest{
			Model: "claude-test", Prompt: "rate",
			ResponseFormat: &models.LLMResponseFormat{Type: "json_schema", JSONSchema: &models.LLMJSONSchema{Description: "d", Schema: schema}},
		})
		require.NoError(t, err)
		body := srv.lastBody()
		assert.Equal(t, anthropicStructuredTool, body["tool_choice"].(map[string]any)["name"])
		assert.Equal(t, "d", body["tools"].([]any)[0].(map[string]any)["description"])
	})

	t.Run("json_schema with tools uses the system prompt", func(t *testing.T) {
		srv := rcNewRecordingServer(t, 200, anthropicTextResponse)
		p, _ := NewAnthropicProvider("k", srv.URL)
		_, err := p.Execute(context.Background(), &models.LLMRequest{
			Model: "claude-test", Prompt: "rate", Instruction: "base",
			Tools:          []models.LLMTool{{Function: models.LLMFunctionTool{Name: "lookup"}}},
			ResponseFormat: &models.LLMResponseFormat{Type: "json_schema", JSONSchema: &models.LLMJSONSchema{Name: "rating", Schema: schema}},
		})
		require.NoError(t, err)
		body := srv.lastBody()
		assert.NotContains(t, body, "tool_choice")
		assert.Contains(t, body["system"], "base\n\nRespond with a single valid JSON object")
		assert.Contains(t, body["system"], `"score"`)
	})

	t.Run("json_object", func(t *testing.T) {
		srv := rcNewRecordingServer(t, 200, anthropicTextResponse)
		p, _ := NewAnthropicProvider("k", srv.URL)
		_, err := p.Execute(context.Background(), &models.LLMRequest{
			Model: "claude-test", Prompt: "rate",
			ResponseFormat: &models.LLMResponseFormat{Type: "json_object"},
		})
		require.NoError(t, err)
		assert.Equal(t, "Respond with a single valid JSON object and nothing else.", srv.lastBody()["system"])
	})
}

func TestAnthropicProvider_Errors(t *testing.T) {
	t.Run("api error", func(t *testing.T) {
		srv := rcNewRecordingServer(t, 401, `{"type":"error","error":{"type":"authentication_error","message":"invalid x-api-key"}}`)
		p, _ := NewAnthropicProvider("k", srv.URL)
		_, err := p.Execute(context.Background(), &models.LLMRequest{Model: "m", Prompt: "p"})
		var llmErr *models.LLMError
		require.ErrorAs(t, err, &llmErr)
		assert.Equal(t, models.LLMProviderAnthropic, llmErr.Provider)
		assert.Equal(t, "401", llmErr.Code)
		assert.Equal(t, "authentication_error", llmErr.Type)
	})

	t.Run("non-json error", func(t *testing.T) {
		srv := rcNewRecordingServer(t, 502, `bad gateway`)
		p, _ := NewAnthropicProvider("k", srv.URL)
		_, err := p.Execute(context.Background(), &models.LLMRequest{Model: "m", Prompt: "p"})
		require.ErrorContains(t, err, "status 502")
	})

	t.Run("invalid response", func(t *testing.T) {
		srv := rcNewRecordingServer(t, 200, `{not json`)
		p, _ := NewAnthropicProvider("k", srv.URL)
		_, err := p.Execute(context.Background(), &models.LLMRequest{Model: "m", Prompt: "p"})
		require.ErrorContains(t, err, "failed to parse response")
	})

	t.Run("bad url", func(t *testing.T) {
		p, _ := NewAnthropicProvider("k", "://bad")
		_, err := p.Execute(context.Background(), &models.LLMRequest{Model: "m", Prompt: "p"})
		require.ErrorContains(t, err, "failed to create request")
	})

	t.Run("connection refused", func(t *testing.T) {
		p, _ := NewAnthropicProvider("k", "http://127.0.0.1:1")
		_, err := p.Execute(context.Background(), &models.LLMRequest{Model: "m", Prompt: "p"})
		require.ErrorContains(t, err, "request failed")
	})

	t.Run("unmarshalable request", func(t *testing.T) {
		p, _ := NewAnthropicProvider("k", "http://127.0.0.1:1")
		_, err := p.Execute(context.Background(), &models.LLMRequest{Model: "m", Prompt: "p",
			Tools: []models.LLMTool{{Function: models.LLMFunctionTool{Name: "t", Parameters: map[string]any{"type": "object", "bad": make(chan int)}}}}})
		require.ErrorContains(t, err, "failed to marshal request")
	})
}

func TestAnthropicProvider_StopReasonsAndSchema(t *testing.T) {
	p := &AnthropicProvider{}
	for in, want := range map[string]string{
		"end_turn": "stop", "stop_sequence": "stop", "max_tokens": "length",
		"tool_use": "tool_calls", "refusal": "content_filter", "pause_turn": "pause_turn",
	} {
		assert.Equal(t, want, p.normalizeStopReason(in), in)
	}

	withType := anthropicInputSchema(map[string]any{"properties": map[string]any{}})
	assert.Equal(t, "object", withType["type"])

	resp := p.convertResponse(&anthropicMessageResponse{}, &models.LLMRequest{Model: "fallback"}, "")
	assert.Equal(t, "fallback", resp.Model)
}

func TestLLMExecutor_AnthropicEndToEnd(t *testing.T) {
	srv := rcNewRecordingServer(t, 200, anthropicTextResponse)
	exec := NewLLMExecutor()

	out, err := exec.Execute(context.Background(), map[string]any{
		"provider": "anthropic",
		"model":    "claude-test",
		"prompt":   "Say hello",
		"api_key":  "k",
		"base_url": srv.URL,
	}, nil)
	require.NoError(t, err)

	data, _ := json.Marshal(out)
	assert.Contains(t, string(data), "Hello")
	assert.Equal(t, "/v1/messages", srv.paths[0])
}

func TestResponseFormatConfig(t *testing.T) {
	m, ok := responseFormatConfig(map[string]any{"type": "json_schema"})
	assert.True(t, ok)
	assert.Equal(t, "json_schema", m["type"])

	m, ok = responseFormatConfig("json")
	assert.True(t, ok)
	assert.Equal(t, map[string]any{"type": "json_object"}, m)

	m, ok = responseFormatConfig("text")
	assert.True(t, ok)
	assert.Equal(t, map[string]any{"type": "text"}, m)

	_, ok = responseFormatConfig("")
	assert.False(t, ok)
	_, ok = responseFormatConfig(42)
	assert.False(t, ok)

	// An unknown shorthand is rejected by validation.
	err := NewLLMExecutor().Validate(map[string]any{
		"provider": "anthropic", "model": "m", "prompt": "p", "api_key": "k", "response_format": "yaml",
	})
	require.ErrorContains(t, err, "invalid response_format type")
}
