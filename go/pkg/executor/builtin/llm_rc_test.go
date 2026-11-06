package builtin

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/smilemakc/mbflow/go/pkg/models"
)

// rcRecordingServer serves a fixed response and records every request body.
type rcRecordingServer struct {
	*httptest.Server
	mu      sync.Mutex
	bodies  []map[string]any
	paths   []string
	headers []http.Header
}

func rcNewRecordingServer(t *testing.T, status int, response string) *rcRecordingServer {
	t.Helper()
	rs := &rcRecordingServer{}
	rs.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		data, _ := io.ReadAll(r.Body)
		var body map[string]any
		_ = json.Unmarshal(data, &body)
		rs.mu.Lock()
		rs.bodies = append(rs.bodies, body)
		rs.paths = append(rs.paths, r.URL.Path)
		rs.headers = append(rs.headers, r.Header.Clone())
		rs.mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_, _ = w.Write([]byte(response))
	}))
	t.Cleanup(rs.Close)
	return rs
}

func (rs *rcRecordingServer) lastBody() map[string]any {
	rs.mu.Lock()
	defer rs.mu.Unlock()
	if len(rs.bodies) == 0 {
		return nil
	}
	return rs.bodies[len(rs.bodies)-1]
}

// rcScriptedProvider returns the scripted responses in order.
type rcScriptedProvider struct {
	mu        sync.Mutex
	responses []*models.LLMResponse
	err       error
	requests  []*models.LLMRequest
}

func (p *rcScriptedProvider) Execute(_ context.Context, req *models.LLMRequest) (*models.LLMResponse, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	cp := *req
	p.requests = append(p.requests, &cp)
	if p.err != nil {
		return nil, p.err
	}
	idx := len(p.requests) - 1
	if idx >= len(p.responses) {
		idx = len(p.responses) - 1
	}
	return p.responses[idx], nil
}

// --- OpenAI Responses provider ---

const rcResponsesBody = `{
  "id": "resp_1", "object": "response", "created_at": 1700000000, "status": "completed",
  "model": "gpt-5",
  "output": [
    {"id": "msg_1", "type": "message", "status": "completed", "role": "assistant",
     "content": [{"type": "output_text", "text": "Hello there",
                  "annotations": [{"type": "url_citation", "start_index": 1, "end_index": 5,
                                   "url": "https://example.com", "title": "Ex"}]}]},
    {"id": "fc_1", "type": "function_call", "status": "completed", "call_id": "call_1",
     "name": "lookup", "arguments": "{\"q\":1}"},
    {"id": "ws_1", "type": "web_search_call", "status": "completed", "queries": ["golang"],
     "results": [{"url": "https://go.dev"}]}
  ],
  "error": {"code": "warn"},
  "incomplete_details": {"reason": "none"},
  "reasoning": {"effort": "high", "summary": "thought"},
  "usage": {"input_tokens": 3, "output_tokens": 4, "total_tokens": 7}
}`

func TestRC_OpenAIResponsesProvider_ExecuteFullResponse(t *testing.T) {
	srv := rcNewRecordingServer(t, http.StatusOK, rcResponsesBody)

	exec := NewLLMExecutor()
	out, err := exec.Execute(context.Background(), map[string]any{
		"provider":             "openai-responses",
		"model":                "gpt-5",
		"api_key":              "sk-rc",
		"base_url":             srv.URL,
		"org_id":               "org-rc",
		"prompt":               "ignored because input is set",
		"input":                []any{map[string]any{"role": "user", "content": "hi"}},
		"instructions":         "be nice",
		"max_tokens":           50,
		"temperature":          0.5,
		"top_p":                0.9,
		"previous_response_id": "resp_0",
		"background":           true,
		"max_tool_calls":       float64(3),
		"store":                false,
		"reasoning":            map[string]any{"effort": "high"},
		"tools": []any{map[string]any{"type": "function", "function": map[string]any{
			"name": "lookup", "description": "Look up", "parameters": map[string]any{"type": "object"},
		}}},
		"hosted_tools": []any{
			map[string]any{"type": "web_search_preview", "domains": []any{"go.dev"}, "search_context_size": "high"},
			map[string]any{"type": "file_search", "vector_store_ids": []any{"vs_1"}, "max_num_results": float64(5),
				"ranking_options": map[string]any{"ranker": "auto"}},
			map[string]any{"type": "code_interpreter"},
		},
		"response_format": map[string]any{"type": "json_schema", "json_schema": map[string]any{
			"name": "answer", "description": "d", "schema": map[string]any{"type": "object"}, "strict": true,
		}},
	}, nil)
	require.NoError(t, err)

	body := srv.lastBody()
	require.NotNil(t, body)
	assert.Equal(t, "/responses", srv.paths[0])
	assert.Equal(t, "Bearer sk-rc", srv.headers[0].Get("Authorization"))
	assert.Equal(t, "org-rc", srv.headers[0].Get("OpenAI-Organization"))
	assert.Equal(t, "gpt-5", body["model"])
	assert.Equal(t, "be nice", body["instructions"])
	assert.Equal(t, float64(50), body["max_output_tokens"])
	assert.Equal(t, 0.5, body["temperature"])
	assert.Equal(t, 0.9, body["top_p"])
	assert.Equal(t, "resp_0", body["previous_response_id"])
	assert.Equal(t, true, body["background"])
	assert.Equal(t, float64(3), body["max_tool_calls"])
	assert.Equal(t, false, body["store"])
	assert.Equal(t, map[string]any{"effort": "high"}, body["reasoning"])
	assert.IsType(t, []any{}, body["input"])
	tools := body["tools"].([]any)
	require.Len(t, tools, 4)
	assert.Equal(t, "lookup", tools[0].(map[string]any)["name"])
	assert.Equal(t, []any{"go.dev"}, tools[1].(map[string]any)["domains"])
	assert.Equal(t, "high", tools[1].(map[string]any)["search_context_size"])
	assert.Equal(t, []any{"vs_1"}, tools[2].(map[string]any)["vector_store_ids"])
	assert.Equal(t, float64(5), tools[2].(map[string]any)["max_num_results"])
	assert.NotNil(t, tools[2].(map[string]any)["ranking_options"])
	assert.Equal(t, "code_interpreter", tools[3].(map[string]any)["type"])
	format := body["text"].(map[string]any)["format"].(map[string]any)
	assert.Equal(t, "json_schema", format["type"])

	result := out.(map[string]any)
	assert.Equal(t, "Hello there", result["content_raw"])
	assert.Equal(t, "resp_1", result["response_id"])
	assert.Equal(t, "completed", result["status"])
	assert.Equal(t, "completed", result["finish_reason"])
	usage := result["usage"].(map[string]any)
	assert.Equal(t, 7, usage["total_tokens"])
	items := result["output_items"].([]map[string]any)
	require.Len(t, items, 3)
	assert.Equal(t, "assistant", items[0]["role"])
	content := items[0]["content"].([]map[string]any)
	ann := content[0]["annotations"].([]map[string]any)
	assert.Equal(t, "https://example.com", ann[0]["url"])
	assert.Equal(t, "call_1", items[1]["call_id"])
	assert.Equal(t, "lookup", items[1]["name"])
	assert.Equal(t, `{"q":1}`, items[1]["arguments"])
	assert.Equal(t, []string{"golang"}, items[2]["queries"])
	assert.NotNil(t, items[2]["results"])
	toolCalls := result["tool_calls"].([]map[string]any)
	require.Len(t, toolCalls, 1)
	assert.Equal(t, "call_1", toolCalls[0]["id"])
	assert.Contains(t, result, "error")
	assert.Equal(t, map[string]any{"reason": "none"}, result["incomplete_details"])
	assert.Equal(t, map[string]any{"effort": "high", "summary": "thought"}, result["reasoning"])
}

func TestRC_OpenAIResponsesProvider_PromptAndInstructionFallback(t *testing.T) {
	srv := rcNewRecordingServer(t, http.StatusOK, `{"id":"r","status":"completed","output":[]}`)
	p, err := NewOpenAIResponsesProvider("k", srv.URL, "")
	require.NoError(t, err)

	resp, err := p.Execute(context.Background(), &models.LLMRequest{
		Model: "m", Prompt: "hello", Instruction: "sys",
		Reasoning: &models.LLMReasoningInfo{}, // empty reasoning is omitted
	})
	require.NoError(t, err)
	assert.Equal(t, "r", resp.ResponseID)
	body := srv.lastBody()
	assert.Equal(t, "hello", body["input"])
	assert.Equal(t, "sys", body["instructions"])
	assert.NotContains(t, body, "reasoning")
	assert.NotContains(t, body, "tools")
	assert.Empty(t, srv.headers[0].Get("OpenAI-Organization"))
}

func TestRC_OpenAIResponsesProvider_History(t *testing.T) {
	srv := rcNewRecordingServer(t, http.StatusOK, `{"id":"r","status":"completed","output":[]}`)
	p, err := NewOpenAIResponsesProvider("k", srv.URL, "")
	require.NoError(t, err)

	_, err = p.Execute(context.Background(), &models.LLMRequest{
		Model:       "m",
		Instruction: "ignored when history is present",
		Messages: []models.LLMMessage{
			{Role: "user", Content: "q"},
			{Role: "assistant", Content: "thinking", ToolCalls: []models.LLMToolCall{
				{ID: "c1", Function: models.LLMFunctionCall{Name: "f", Arguments: "{}"}},
			}},
			{Role: "tool", ToolCallID: "c1", Content: "42"},
		},
	})
	require.NoError(t, err)
	body := srv.lastBody()
	assert.NotContains(t, body, "instructions")
	input := body["input"].([]any)
	require.Len(t, input, 4)
	assert.Equal(t, "function_call", input[2].(map[string]any)["type"])
	assert.Equal(t, "function_call_output", input[3].(map[string]any)["type"])
}

func TestRC_OpenAIResponsesProvider_Errors(t *testing.T) {
	_, err := NewOpenAIResponsesProvider("", "", "")
	require.Error(t, err)

	p, err := NewOpenAIResponsesProvider("k", "", "")
	require.NoError(t, err)
	assert.Equal(t, "https://api.openai.com/v1", p.baseURL)

	t.Run("structured api error", func(t *testing.T) {
		srv := rcNewRecordingServer(t, http.StatusUnauthorized,
			`{"error":{"code":"invalid_api_key","message":"bad key","type":"auth"}}`)
		p, _ := NewOpenAIResponsesProvider("k", srv.URL, "")
		_, err := p.Execute(context.Background(), &models.LLMRequest{Model: "m", Prompt: "x"})
		var llmErr *models.LLMError
		require.ErrorAs(t, err, &llmErr)
		assert.Equal(t, "invalid_api_key", llmErr.Code)
		assert.Equal(t, "bad key", llmErr.Message)
		assert.Equal(t, models.LLMProviderOpenAIResponses, llmErr.Provider)
	})
	t.Run("plain api error", func(t *testing.T) {
		srv := rcNewRecordingServer(t, http.StatusBadGateway, `upstream down`)
		p, _ := NewOpenAIResponsesProvider("k", srv.URL, "")
		_, err := p.Execute(context.Background(), &models.LLMRequest{Model: "m", Prompt: "x"})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "status 502")
		assert.Contains(t, err.Error(), "upstream down")
	})
	t.Run("malformed body", func(t *testing.T) {
		srv := rcNewRecordingServer(t, http.StatusOK, `{not json`)
		p, _ := NewOpenAIResponsesProvider("k", srv.URL, "")
		_, err := p.Execute(context.Background(), &models.LLMRequest{Model: "m", Prompt: "x"})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to parse response")
	})
	t.Run("bad url", func(t *testing.T) {
		p, _ := NewOpenAIResponsesProvider("k", "://bad", "")
		_, err := p.Execute(context.Background(), &models.LLMRequest{Model: "m", Prompt: "x"})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to create request")
	})
	t.Run("unreachable", func(t *testing.T) {
		srv := httptest.NewServer(http.NotFoundHandler())
		url := srv.URL
		srv.Close()
		p, _ := NewOpenAIResponsesProvider("k", url, "")
		_, err := p.Execute(context.Background(), &models.LLMRequest{Model: "m", Prompt: "x"})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "request failed")
	})
	t.Run("unmarshalable request", func(t *testing.T) {
		p, _ := NewOpenAIResponsesProvider("k", "http://127.0.0.1:1", "")
		_, err := p.Execute(context.Background(), &models.LLMRequest{Model: "m", Input: make(chan int)})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to marshal request")
	})
}

func TestRC_OpenAIResponsesProvider_BuildResponseFormatText(t *testing.T) {
	p := &OpenAIResponsesProvider{}
	assert.Equal(t, map[string]any{"type": "text"}, p.buildResponseFormat(&models.LLMResponseFormat{Type: "text"}))
	got := p.buildResponseFormat(&models.LLMResponseFormat{Type: "json_schema",
		JSONSchema: &models.LLMJSONSchema{Name: "n", Schema: map[string]any{"type": "object"}, Strict: true}})
	assert.Equal(t, "n", got["name"])
	assert.Equal(t, map[string]any{"type": "object"}, got["schema"])
	assert.Equal(t, true, got["strict"])
	assert.NotContains(t, got, "json_schema")
	assert.NotContains(t, got, "description")
}

// --- OpenAI chat completions provider ---

func TestRC_OpenAIProvider_ExecuteMultimodalAndParams(t *testing.T) {
	srv := rcNewRecordingServer(t, http.StatusOK, `{
	  "id":"chat_1","created":1700000000,"model":"gpt-4o",
	  "choices":[{"index":0,"message":{"role":"assistant","content":"{\"ok\":true}",
	    "tool_calls":[{"id":"t1","type":"function","function":{"name":"f","arguments":"{}"}}]},
	    "finish_reason":"tool_calls"}],
	  "usage":{"prompt_tokens":1,"completion_tokens":2,"total_tokens":3}}`)

	exec := NewLLMExecutor()
	out, err := exec.Execute(context.Background(), map[string]any{
		"provider":          "openai",
		"model":             "gpt-4o",
		"api_key":           "sk-rc",
		"base_url":          srv.URL,
		"org_id":            "org-1",
		"prompt":            "describe",
		"instruction":       "system!",
		"max_tokens":        10,
		"temperature":       0.2,
		"top_p":             0.3,
		"frequency_penalty": 0.4,
		"presence_penalty":  0.5,
		"stop_sequences":    []any{"END", 1},
		"image_url":         []any{"https://img/1.png"},
		"image_id":          []any{"img-1"},
		"file_id":           []any{"file-1"},
		"files": []any{
			map[string]any{"data": "aGk=", "mime_type": "image/png", "name": "a.png"},
			map[string]any{"data": "aGk=", "mime_type": "image/jpeg", "detail": "high"},
			map[string]any{"data": "JVBERg==", "mime_type": "application/pdf", "name": "doc.pdf"},
		},
		"response_format": map[string]any{"type": "json_object"},
	}, nil)
	require.NoError(t, err)

	body := srv.lastBody()
	assert.Equal(t, "/chat/completions", srv.paths[0])
	assert.Equal(t, "org-1", srv.headers[0].Get("OpenAI-Organization"))
	assert.Equal(t, float64(10), body["max_tokens"])
	assert.Equal(t, 0.2, body["temperature"])
	assert.Equal(t, 0.3, body["top_p"])
	assert.Equal(t, 0.4, body["frequency_penalty"])
	assert.Equal(t, 0.5, body["presence_penalty"])
	assert.Equal(t, []any{"END"}, body["stop"])
	assert.Equal(t, map[string]any{"type": "json_object"}, body["response_format"])
	msgs := body["messages"].([]any)
	require.Len(t, msgs, 2)
	assert.Equal(t, "system", msgs[0].(map[string]any)["role"])
	parts := msgs[1].(map[string]any)["content"].([]any)
	require.Len(t, parts, 5)
	assert.Equal(t, "text", parts[0].(map[string]any)["type"])
	assert.Equal(t, "https://img/1.png", parts[1].(map[string]any)["image_url"].(map[string]any)["url"])
	img := parts[2].(map[string]any)["image_url"].(map[string]any)
	assert.Equal(t, "data:image/png;base64,aGk=", img["url"])
	assert.Equal(t, "auto", img["detail"])
	assert.Equal(t, "high", parts[3].(map[string]any)["image_url"].(map[string]any)["detail"])
	assert.Equal(t, "file", parts[4].(map[string]any)["type"])

	result := out.(map[string]any)
	assert.Equal(t, map[string]any{"ok": true}, result["content"], "json_object content is parsed")
	assert.Equal(t, `{"ok":true}`, result["content_raw"])
	assert.Equal(t, "tool_calls", result["finish_reason"])
	assert.Len(t, result["tool_calls"], 1)
}

func TestRC_OpenAIProvider_UserContentSkipsUnsupportedAndEmptyText(t *testing.T) {
	p := &OpenAIProvider{}
	got := p.buildUserContent(&models.LLMRequest{Files: []models.LLMFileAttachment{
		{Data: "x", MimeType: "text/plain"},
		{Data: "x", MimeType: "image/webp"},
	}})
	parts := got.([]map[string]any)
	require.Len(t, parts, 1)
	assert.Equal(t, "image_url", parts[0]["type"])

	assert.Equal(t, "just text", p.buildUserContent(&models.LLMRequest{Prompt: "just text"}))
}

func TestRC_OpenAIProvider_BuildResponseFormat(t *testing.T) {
	p := &OpenAIProvider{}
	got := p.buildResponseFormat(&models.LLMResponseFormat{Type: "json_schema", JSONSchema: &models.LLMJSONSchema{
		Name: "x", Description: "d", Schema: map[string]any{"type": "object"}, Strict: true,
	}})
	js := got["json_schema"].(map[string]any)
	assert.Equal(t, "x", js["name"])
	assert.Equal(t, true, js["strict"])
}

func TestRC_OpenAIProvider_Errors(t *testing.T) {
	_, err := NewOpenAIProvider("", "", "")
	require.Error(t, err)
	p, err := NewOpenAIProvider("k", "", "")
	require.NoError(t, err)
	assert.Equal(t, "https://api.openai.com/v1", p.baseURL)

	t.Run("structured", func(t *testing.T) {
		srv := rcNewRecordingServer(t, http.StatusTooManyRequests,
			`{"error":{"code":"rate","message":"slow down","type":"limit"}}`)
		p, _ := NewOpenAIProvider("k", srv.URL, "")
		_, err := p.Execute(context.Background(), &models.LLMRequest{Model: "m", Prompt: "x"})
		var llmErr *models.LLMError
		require.ErrorAs(t, err, &llmErr)
		assert.Equal(t, "slow down", llmErr.Message)
	})
	t.Run("plain", func(t *testing.T) {
		srv := rcNewRecordingServer(t, http.StatusInternalServerError, `oops`)
		p, _ := NewOpenAIProvider("k", srv.URL, "")
		_, err := p.Execute(context.Background(), &models.LLMRequest{Model: "m", Prompt: "x"})
		require.ErrorContains(t, err, "status 500")
	})
	t.Run("malformed", func(t *testing.T) {
		srv := rcNewRecordingServer(t, http.StatusOK, `[`)
		p, _ := NewOpenAIProvider("k", srv.URL, "")
		_, err := p.Execute(context.Background(), &models.LLMRequest{Model: "m", Prompt: "x"})
		require.ErrorContains(t, err, "failed to parse response")
	})
	t.Run("no choices", func(t *testing.T) {
		srv := rcNewRecordingServer(t, http.StatusOK, `{"model":"m","choices":[]}`)
		p, _ := NewOpenAIProvider("k", srv.URL, "")
		resp, err := p.Execute(context.Background(), &models.LLMRequest{Model: "m", Prompt: "x"})
		require.NoError(t, err)
		assert.Equal(t, "error", resp.FinishReason)
	})
	t.Run("bad url", func(t *testing.T) {
		p, _ := NewOpenAIProvider("k", "://bad", "")
		_, err := p.Execute(context.Background(), &models.LLMRequest{Model: "m", Prompt: "x"})
		require.ErrorContains(t, err, "failed to create request")
	})
	t.Run("unreachable", func(t *testing.T) {
		srv := httptest.NewServer(http.NotFoundHandler())
		url := srv.URL
		srv.Close()
		p, _ := NewOpenAIProvider("k", url, "")
		_, err := p.Execute(context.Background(), &models.LLMRequest{Model: "m", Prompt: "x"})
		require.ErrorContains(t, err, "request failed")
	})
	t.Run("unmarshalable", func(t *testing.T) {
		p, _ := NewOpenAIProvider("k", "http://127.0.0.1:1", "")
		_, err := p.Execute(context.Background(), &models.LLMRequest{Model: "m", Prompt: "x",
			Tools: []models.LLMTool{{Function: models.LLMFunctionTool{Name: "f", Parameters: map[string]any{"c": make(chan int)}}}}})
		require.ErrorContains(t, err, "failed to marshal request")
	})
}

// --- Gemini provider ---

func TestRC_GeminiProvider_Execute(t *testing.T) {
	srv := rcNewRecordingServer(t, http.StatusOK, `{
	  "candidates":[{"content":{"role":"model","parts":[{"text":"hola"},
	     {"functionCall":{"name":"lookup","args":{"q":"x"}}}]},"finishReason":"STOP"}],
	  "usageMetadata":{"promptTokenCount":2,"candidatesTokenCount":3,"totalTokenCount":5},
	  "modelVersion":"gemini-2.0","responseId":"g1"}`)

	exec := NewLLMExecutor()
	out, err := exec.Execute(context.Background(), map[string]any{
		"provider": "gemini",
		"model":    "gemini-2.0",
		"api_key":  "g-key",
		"base_url": srv.URL,
		"prompt":   "hi",
	}, nil)
	require.NoError(t, err)
	assert.Equal(t, "/models/gemini-2.0:generateContent", srv.paths[0])
	assert.Equal(t, "g-key", srv.headers[0].Get("x-goog-api-key"))
	result := out.(map[string]any)
	assert.Equal(t, "hola", result["content"])
	assert.Equal(t, 5, result["usage"].(map[string]any)["total_tokens"])
	assert.Len(t, result["tool_calls"], 1)
}

func TestRC_GeminiProvider_Errors(t *testing.T) {
	t.Run("structured", func(t *testing.T) {
		srv := rcNewRecordingServer(t, http.StatusBadRequest,
			`{"error":{"code":400,"message":"bad model","status":"INVALID_ARGUMENT"}}`)
		p, _ := NewGeminiProvider("k", srv.URL)
		_, err := p.Execute(context.Background(), &models.LLMRequest{Model: "m", Prompt: "x"})
		var llmErr *models.LLMError
		require.ErrorAs(t, err, &llmErr)
		assert.Equal(t, "400", llmErr.Code)
		assert.Equal(t, "INVALID_ARGUMENT", llmErr.Type)
	})
	t.Run("plain", func(t *testing.T) {
		srv := rcNewRecordingServer(t, http.StatusServiceUnavailable, `busy`)
		p, _ := NewGeminiProvider("k", srv.URL)
		_, err := p.Execute(context.Background(), &models.LLMRequest{Model: "m", Prompt: "x"})
		require.ErrorContains(t, err, "status 503")
	})
	t.Run("malformed", func(t *testing.T) {
		srv := rcNewRecordingServer(t, http.StatusOK, `nope`)
		p, _ := NewGeminiProvider("k", srv.URL)
		_, err := p.Execute(context.Background(), &models.LLMRequest{Model: "m", Prompt: "x"})
		require.ErrorContains(t, err, "failed to parse response")
	})
	t.Run("no candidates", func(t *testing.T) {
		srv := rcNewRecordingServer(t, http.StatusOK, `{"candidates":[]}`)
		p, _ := NewGeminiProvider("k", srv.URL)
		resp, err := p.Execute(context.Background(), &models.LLMRequest{Model: "m", Prompt: "x"})
		require.NoError(t, err)
		assert.Equal(t, "m", resp.Model)
		assert.Equal(t, "error", resp.FinishReason)
	})
	t.Run("bad url", func(t *testing.T) {
		p, _ := NewGeminiProvider("k", "://bad")
		_, err := p.Execute(context.Background(), &models.LLMRequest{Model: "m", Prompt: "x"})
		require.ErrorContains(t, err, "failed to create request")
	})
	t.Run("unreachable", func(t *testing.T) {
		srv := httptest.NewServer(http.NotFoundHandler())
		url := srv.URL
		srv.Close()
		p, _ := NewGeminiProvider("k", url)
		_, err := p.Execute(context.Background(), &models.LLMRequest{Model: "m", Prompt: "x"})
		require.ErrorContains(t, err, "request failed")
	})
	t.Run("unmarshalable", func(t *testing.T) {
		p, _ := NewGeminiProvider("k", "http://127.0.0.1:1")
		_, err := p.Execute(context.Background(), &models.LLMRequest{Model: "m", Prompt: "x",
			Tools: []models.LLMTool{{Function: models.LLMFunctionTool{Name: "f", Parameters: map[string]any{"c": make(chan int)}}}}})
		require.ErrorContains(t, err, "failed to marshal request")
	})
}

func TestRC_GeminiFunctionResponse(t *testing.T) {
	assert.Equal(t, map[string]any{"a": float64(1)}, geminiFunctionResponse(`{"a":1}`))
	assert.Equal(t, map[string]any{"result": []any{float64(1)}}, geminiFunctionResponse(`[1]`))
	assert.Equal(t, map[string]any{"result": "plain"}, geminiFunctionResponse(`plain`))
}

// --- LLMExecutor config parsing / validation ---

func TestRC_LLMExecutor_ParseConfigErrors(t *testing.T) {
	exec := NewLLMExecutor()
	base := func(k string, v any) map[string]any {
		return map[string]any{"provider": "openai", "model": "m", "prompt": "p", "api_key": "k", k: v}
	}
	cases := map[string]struct {
		cfg  map[string]any
		want string
	}{
		"file not object":        {base("files", []any{"x"}), "file 0 is not a valid object"},
		"file no data":           {base("files", []any{map[string]any{"mime_type": "image/png"}}), "data is required"},
		"file no mime":           {base("files", []any{map[string]any{"data": "x"}}), "mime_type is required"},
		"file unsupported":       {base("files", []any{map[string]any{"data": "x", "mime_type": "text/csv"}}), "unsupported mime_type"},
		"tool not object":        {base("tools", []any{1}), "tool 0 is not a valid object"},
		"tool wrong type":        {base("tools", []any{map[string]any{"type": "x"}}), "only 'function' type"},
		"tool missing function":  {base("tools", []any{map[string]any{"type": "function"}}), "missing function definition"},
		"tool missing name":      {base("tools", []any{map[string]any{"type": "function", "function": map[string]any{}}}), "function name is required"},
		"format no type":         {base("response_format", map[string]any{}), "response_format type is required"},
		"format no schema":       {base("response_format", map[string]any{"type": "json_schema"}), "json_schema is required"},
		"format no schema name":  {base("response_format", map[string]any{"type": "json_schema", "json_schema": map[string]any{}}), "json_schema name is required"},
		"hosted not object":      {base("hosted_tools", []any{"x"}), "hosted tool 0 is not a valid object"},
		"hosted no type":         {base("hosted_tools", []any{map[string]any{}}), "missing type"},
		"hosted unsupported":     {base("hosted_tools", []any{map[string]any{"type": "dalle"}}), "unsupported type dalle"},
		"message not object":     {base("messages", []any{"x"}), "message 0 is not a valid object"},
		"function not object":    {base("functions", []any{"x"}), "function 0 is not a valid object"},
		"api tools not array":    {base("api_tools", "x"), "api_tools must be an array"},
		"api tools no client":    {base("api_tools", []any{map[string]any{}}), "client_id is required"},
		"api tools no ops":       {base("api_tools", []any{map[string]any{"client_id": "c"}}), "operation_ids must list"},
		"api tools item invalid": {base("api_tools", []any{1}), "api_tools[0] must be an object"},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			_, err := exec.Execute(context.Background(), tc.cfg, nil)
			require.Error(t, err)
			assert.Contains(t, err.Error(), tc.want)
		})
	}
}

func TestRC_LLMExecutor_ParseConfigFull(t *testing.T) {
	exec := NewLLMExecutor()
	req, err := exec.parseConfig(map[string]any{
		"provider":       "openai-responses",
		"model":          "m",
		"prompt":         "p",
		"max_tool_calls": 4,
		"hosted_tools": []any{
			map[string]any{"type": "file_search", "max_num_results": 7},
		},
		"messages": []any{
			map[string]any{"role": "assistant", "content": "c", "tool_call_id": "x", "name": "n",
				"tool_calls": []any{"skip", map[string]any{"id": "1", "type": "function",
					"function": map[string]any{"name": "f", "arguments": "{}"}}}},
		},
		"functions": []any{map[string]any{
			"type": "sub_workflow", "name": "fn", "description": "d",
			"parameters": map[string]any{"type": "object"}, "builtin_name": "b", "workflow_id": "wf",
			"input_mapping": map[string]any{"a": "b", "bad": 1}, "output_extractor": ".x",
			"language": "python", "code": "pass", "client_id": "c", "operation_id": "o",
		}},
		"tool_call_config": map[string]any{
			"mode": "manual", "auto_execute_tools": true, "max_iterations": 3,
			"timeout_per_tool": 2, "total_timeout": 9, "stop_on_tool_failure": true,
		},
	})
	require.NoError(t, err)
	assert.Equal(t, 4, req.MaxToolCalls)
	assert.Equal(t, 7, req.HostedTools[0].MaxNumResults)
	require.Len(t, req.Messages, 1)
	msg := req.Messages[0]
	assert.Equal(t, "x", msg.ToolCallID)
	assert.Equal(t, "n", msg.Name)
	require.Len(t, msg.ToolCalls, 1)
	assert.Equal(t, "f", msg.ToolCalls[0].Function.Name)
	fn := req.Functions[0]
	assert.Equal(t, map[string]string{"a": "b"}, fn.InputMapping)
	assert.Equal(t, ".x", fn.OutputExtractor)
	assert.Equal(t, "python", fn.Language)
	assert.Equal(t, "pass", fn.Code)
	assert.Equal(t, "c", fn.ClientID)
	assert.Equal(t, "o", fn.OperationID)
	assert.Equal(t, models.ToolCallModeAuto, req.ToolCallConfig.Mode)
	assert.Equal(t, 3, req.ToolCallConfig.MaxIterations)
	assert.Equal(t, 2, req.ToolCallConfig.TimeoutPerTool)
	assert.Equal(t, 9, req.ToolCallConfig.TotalTimeout)
	assert.True(t, req.ToolCallConfig.StopOnToolFailure)
}

func TestRC_LLMExecutor_Validate(t *testing.T) {
	exec := NewLLMExecutor()
	ok := func() map[string]any {
		return map[string]any{"provider": "openai", "model": "m", "prompt": "p", "api_key": "k"}
	}
	require.NoError(t, exec.Validate(ok()))

	cases := map[string]struct {
		mut  func(map[string]any)
		want string
	}{
		"provider wrong type": {func(c map[string]any) { c["provider"] = 5 }, ""},
		"model wrong type":    {func(c map[string]any) { c["model"] = 5 }, ""},
		"empty model":         {func(c map[string]any) { c["model"] = "" }, ""},
		"negative max tokens": {func(c map[string]any) { c["max_tokens"] = -1 }, "max_tokens"},
		"bad format type":     {func(c map[string]any) { c["response_format"] = map[string]any{"type": "xml"} }, "invalid response_format type"},
		"format no schema": {func(c map[string]any) {
			c["response_format"] = map[string]any{"type": "json_schema"}
		}, "json_schema is required"},
		"format schema missing": {func(c map[string]any) {
			c["response_format"] = map[string]any{"type": "json_schema", "json_schema": map[string]any{"name": "n"}}
		}, "json_schema schema is required"},
		"format no name": {func(c map[string]any) {
			c["response_format"] = map[string]any{"type": "json_schema", "json_schema": map[string]any{}}
		}, "json_schema name is required"},
		"tool not object":      {func(c map[string]any) { c["tools"] = []any{1} }, "not a valid object"},
		"tool no function":     {func(c map[string]any) { c["tools"] = []any{map[string]any{"type": "function"}} }, "missing function definition"},
		"tool bad type":        {func(c map[string]any) { c["tools"] = []any{map[string]any{"type": "x"}} }, "only 'function'"},
		"api tools invalid":    {func(c map[string]any) { c["api_tools"] = "x" }, "api_tools must be an array"},
		"top_p out of range":   {func(c map[string]any) { c["top_p"] = 1.5 }, "top_p"},
		"temperature too high": {func(c map[string]any) { c["temperature"] = 2.5 }, "temperature"},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			cfg := ok()
			tc.mut(cfg)
			err := exec.Validate(cfg)
			require.Error(t, err)
			if tc.want != "" {
				assert.Contains(t, err.Error(), tc.want)
			}
		})
	}
}

func TestRC_LLMExecutor_ProviderSelection(t *testing.T) {
	exec := NewLLMExecutor()
	assert.False(t, exec.hasProvider(models.LLMProviderOpenAI))
	exec.RegisterProvider(models.LLMProviderOpenAI, &rcScriptedProvider{})
	assert.True(t, exec.hasProvider(models.LLMProviderOpenAI))

	_, err := exec.getOrCreateProvider(&models.LLMRequest{Provider: "unknown"})
	require.ErrorContains(t, err, "unsupported provider")

	anthropic, err := exec.getOrCreateProvider(&models.LLMRequest{Provider: models.LLMProviderAnthropic,
		ProviderConfig: map[string]any{"api_key": "k"}})
	require.NoError(t, err)
	assert.IsType(t, &AnthropicProvider{}, anthropic)

	p, err := exec.getOrCreateProvider(&models.LLMRequest{Provider: models.LLMProviderOpenAIResponses,
		ProviderConfig: map[string]any{"api_key": "k"}})
	require.NoError(t, err)
	assert.IsType(t, &OpenAIResponsesProvider{}, p)
}

func TestRC_LLMExecutor_ProviderErrorAndInputDirect(t *testing.T) {
	exec := NewLLMExecutor()
	prov := &rcScriptedProvider{err: errors.New("boom")}
	exec.RegisterProvider(models.LLMProviderOpenAI, prov)
	_, err := exec.Execute(context.Background(), map[string]any{
		"provider": "openai", "model": "m", "prompt": "p", "use_input_directly": true}, map[string]any{"a": 1})
	require.ErrorContains(t, err, "LLM execution failed")
	require.Len(t, prov.requests, 1)
	assert.Equal(t, map[string]any{"a": 1}, prov.requests[0].Input)
}

func TestRC_LLMExecutor_ResponseToMapAllFields(t *testing.T) {
	exec := NewLLMExecutor()
	resp := &models.LLMResponse{
		Content:  "not json",
		Metadata: map[string]any{"m": 1},
		Messages: []models.LLMMessage{{
			Role: "assistant", Content: "x", ToolCallID: "t", Name: "n", Metadata: map[string]any{"k": "v"},
			ToolCalls: []models.LLMToolCall{{ID: "1", Type: "function", Function: models.LLMFunctionCall{Name: "f"}}},
		}},
		ToolExecutions: []models.ToolExecutionResult{{ToolCallID: "1", FunctionName: "f", Result: 1, Error: "e",
			Metadata: map[string]any{"x": 1}}},
		TotalIterations: 2,
		StoppedReason:   "finish",
	}
	out := exec.responseToMap(resp, &models.LLMResponseFormat{Type: "json_object"})
	assert.Equal(t, "not json", out["content"], "unparsable JSON content stays a string")
	assert.Equal(t, map[string]any{"m": 1}, out["metadata"])
	msg := out["messages"].([]any)[0].(map[string]any)
	assert.Equal(t, "t", msg["tool_call_id"])
	assert.Equal(t, "n", msg["name"])
	assert.NotNil(t, msg["metadata"])
	assert.Len(t, msg["tool_calls"], 1)
	te := out["tool_executions"].([]any)[0].(map[string]any)
	assert.Equal(t, "e", te["error"])
	assert.NotNil(t, te["metadata"])
	assert.Equal(t, 2, out["total_iterations"])
	assert.Equal(t, "finish", out["stopped_reason"])
}

func TestRC_LLMExecutor_ToJSON(t *testing.T) {
	exec := NewLLMExecutor()
	assert.Contains(t, exec.toJSON(map[string]any{"a": 1}), `"a": 1`)
	assert.Contains(t, exec.toJSON(make(chan int)), "error:")
}

func TestRC_LLMExecutor_FormatToolResult(t *testing.T) {
	exec := NewLLMExecutor()
	assert.Equal(t, "Error: x", exec.formatToolResult(models.ToolExecutionResult{Error: "x"}))
	assert.Equal(t, `{"a":1}`, exec.formatToolResult(models.ToolExecutionResult{Result: map[string]int{"a": 1}}))
	assert.Contains(t, exec.formatToolResult(models.ToolExecutionResult{Result: make(chan int)}), "Error formatting result")
}

// --- Auto tool calling ---

func rcAutoConfig(extra map[string]any) map[string]any {
	cfg := map[string]any{
		"provider": "openai", "model": "m", "prompt": "do it", "instruction": "sys",
		"functions": []any{map[string]any{"type": "builtin", "name": "add", "builtin_name": "add"}},
	}
	for k, v := range extra {
		cfg[k] = v
	}
	return cfg
}

func rcRegistryWithAdd() *ToolCallingRegistry {
	fr := models.NewFunctionRegistry()
	fr.Register("add", func(args map[string]any) (any, error) {
		a, _ := args["a"].(float64)
		b, _ := args["b"].(float64)
		if a < 0 {
			return nil, errors.New("negative")
		}
		return a + b, nil
	})
	return NewToolCallingRegistry(fr)
}

func TestRC_LLMExecutor_AutoToolCalling_MaxIterations(t *testing.T) {
	exec := NewLLMExecutor()
	exec.SetToolCallingRegistry(rcRegistryWithAdd())
	prov := &rcScriptedProvider{responses: []*models.LLMResponse{
		toolCallResponse("c1", "add", `{"a":1,"b":2}`),
	}}
	exec.RegisterProvider(models.LLMProviderOpenAI, prov)

	out, err := exec.Execute(context.Background(), rcAutoConfig(map[string]any{
		"tool_call_config": map[string]any{"mode": "auto", "max_iterations": float64(2),
			"total_timeout": float64(30), "timeout_per_tool": float64(5)},
	}), nil)
	require.NoError(t, err)
	result := out.(map[string]any)
	assert.Equal(t, "max_iterations", result["stopped_reason"])
	assert.Equal(t, 2, result["total_iterations"])
	execs := result["tool_executions"].([]any)
	require.Len(t, execs, 2)
	assert.Equal(t, float64(3), execs[0].(map[string]any)["result"])
	// The first request carries the system instruction and the prompt.
	first := prov.requests[0]
	require.GreaterOrEqual(t, len(first.Messages), 2)
	assert.Equal(t, "system", first.Messages[0].Role)
	require.Len(t, first.Tools, 1)
}

func TestRC_LLMExecutor_AutoToolCalling_StopOnFailure(t *testing.T) {
	exec := NewLLMExecutor()
	exec.SetToolCallingRegistry(rcRegistryWithAdd())
	exec.RegisterProvider(models.LLMProviderOpenAI, &rcScriptedProvider{responses: []*models.LLMResponse{
		toolCallResponse("c1", "add", `{"a":-1,"b":2}`),
	}})
	_, err := exec.Execute(context.Background(), rcAutoConfig(map[string]any{
		"tool_call_config": map[string]any{"mode": "auto", "stop_on_tool_failure": true},
	}), nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "negative")
}

func TestRC_LLMExecutor_AutoToolCalling_UnknownFunctionContinues(t *testing.T) {
	exec := NewLLMExecutor()
	exec.SetToolCallingRegistry(rcRegistryWithAdd())
	exec.RegisterProvider(models.LLMProviderOpenAI, &rcScriptedProvider{responses: []*models.LLMResponse{
		toolCallResponse("c1", "missing", `{}`),
		{Content: "done", FinishReason: "stop"},
	}})
	out, err := exec.Execute(context.Background(), rcAutoConfig(map[string]any{
		"messages":         []any{map[string]any{"role": "user", "content": "history"}},
		"tool_call_config": map[string]any{"mode": "auto"},
	}), nil)
	require.NoError(t, err)
	result := out.(map[string]any)
	assert.Equal(t, "done", result["content"])
	te := result["tool_executions"].([]any)[0].(map[string]any)
	assert.Contains(t, te["error"], "function not found: missing")
	msgs := result["messages"].([]any)
	assert.Equal(t, "Error: function not found: missing", msgs[2].(map[string]any)["content"])
}

func TestRC_LLMExecutor_AutoToolCalling_ProviderError(t *testing.T) {
	exec := NewLLMExecutor()
	exec.SetToolCallingRegistry(rcRegistryWithAdd())
	exec.RegisterProvider(models.LLMProviderOpenAI, &rcScriptedProvider{err: errors.New("down")})
	_, err := exec.Execute(context.Background(), rcAutoConfig(map[string]any{
		"tool_call_config": map[string]any{"mode": "auto"},
	}), nil)
	require.ErrorContains(t, err, "LLM call failed at iteration 0")
}

func TestRC_LLMExecutor_AutoToolCalling_KeepsProviderContent(t *testing.T) {
	exec := NewLLMExecutor()
	exec.SetToolCallingRegistry(rcRegistryWithAdd())
	call := toolCallResponse("c", "add", `{"a":1,"b":2}`)
	call.ProviderContent = json.RawMessage(`[{"type":"thinking","signature":"s"}]`)
	prov := &rcScriptedProvider{responses: []*models.LLMResponse{call, {Content: "3", FinishReason: "stop"}}}
	_, err := exec.executeWithToolCalling(context.Background(), &models.LLMRequest{
		Prompt:    "p",
		Functions: []models.FunctionDefinition{{Type: models.FunctionTypeBuiltin, Name: "add", BuiltinName: "add"}},
	}, prov)
	require.NoError(t, err)
	require.Len(t, prov.requests, 2)
	var assistant *models.LLMMessage
	for i, m := range prov.requests[1].Messages {
		if m.Role == "assistant" {
			assistant = &prov.requests[1].Messages[i]
		}
	}
	require.NotNil(t, assistant)
	assert.JSONEq(t, string(call.ProviderContent), string(assistant.ProviderContent))
}

func TestRC_LLMExecutor_AutoToolCalling_NoRegistry(t *testing.T) {
	exec := NewLLMExecutor()
	_, err := exec.executeWithToolCalling(context.Background(), &models.LLMRequest{}, &rcScriptedProvider{})
	require.ErrorContains(t, err, "tool calling registry not configured")
}

func TestRC_LLMExecutor_AutoToolCalling_DefaultsAndTooManyTools(t *testing.T) {
	exec := NewLLMExecutor()
	exec.SetToolCallingRegistry(rcRegistryWithAdd())
	prov := &rcScriptedProvider{responses: []*models.LLMResponse{{Content: "x", FinishReason: "stop"}}}
	resp, err := exec.executeWithToolCalling(context.Background(),
		&models.LLMRequest{Prompt: "p", ToolCallConfig: &models.ToolCallConfig{MaxIterations: 0}}, prov)
	require.NoError(t, err)
	assert.Equal(t, 1, resp.TotalIterations)

	funcs := make([]models.FunctionDefinition, maxToolsPerRequest+1)
	_, err = exec.executeWithToolCalling(context.Background(), &models.LLMRequest{Functions: funcs}, prov)
	require.ErrorContains(t, err, "at most")

	// API tools requested without an OpenAPI executor.
	_, err = exec.executeWithToolCalling(context.Background(), &models.LLMRequest{
		APITools: []models.APIToolRef{{ClientID: "c", OperationIDs: []string{"o"}}}}, prov)
	require.ErrorContains(t, err, "failed to load API tools")
}

func TestRC_LLMExecutor_AutoToolCalling_ZeroIterationsNoResponse(t *testing.T) {
	// A negative max_iterations after defaults is clamped to 10, so "no response"
	// can only be reached with a provider returning a tool call each time; this
	// asserts the clamp by counting provider calls.
	exec := NewLLMExecutor()
	exec.SetToolCallingRegistry(rcRegistryWithAdd())
	prov := &rcScriptedProvider{responses: []*models.LLMResponse{toolCallResponse("c", "add", `{"a":1,"b":1}`)}}
	resp, err := exec.executeWithToolCalling(context.Background(), &models.LLMRequest{
		Prompt:         "p",
		Functions:      []models.FunctionDefinition{{Type: models.FunctionTypeBuiltin, Name: "add", BuiltinName: "add"}},
		ToolCallConfig: &models.ToolCallConfig{MaxIterations: -5},
	}, prov)
	require.NoError(t, err)
	assert.Equal(t, 10, resp.TotalIterations)
	assert.Len(t, prov.requests, 10)
}
