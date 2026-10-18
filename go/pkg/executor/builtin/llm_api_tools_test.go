package builtin

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/smilemakc/mbflow/go/pkg/models"
)

func toolCallResponse(id, name, arguments string) *models.LLMResponse {
	return &models.LLMResponse{FinishReason: "tool_calls", ToolCalls: []models.LLMToolCall{
		{ID: id, Type: "function", Function: models.LLMFunctionCall{Name: name, Arguments: arguments}},
	}}
}

func apiToolsConfig(provider string, operationIDs ...any) map[string]any {
	return map[string]any{
		"provider":         provider,
		"model":            "gpt-test",
		"api_key":          "sk-test",
		"prompt":           "Add Rex to the main store",
		"tool_call_config": map[string]any{"mode": "auto"},
		"api_tools":        []any{map[string]any{"client_id": "client-1", "operation_ids": operationIDs}},
	}
}

func llmWithAPIClient(client *models.APIClient) *LLMExecutor {
	llm := NewLLMExecutor()
	llm.toolRegistry().SetOpenAPIExecutor(NewAPIOperationExecutor(fakeAPIClients{"client-1": client}, nil))
	return llm
}

// fakeOpenAI serves chat completions: the first request gets a call of the
// tool, later ones the final answer. It records the request bodies.
func fakeOpenAI(t *testing.T, toolName, arguments string) (*httptest.Server, func() []map[string]any) {
	t.Helper()
	var mu sync.Mutex
	var requests []map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/chat/completions", r.URL.Path)
		assert.Equal(t, "Bearer sk-test", r.Header.Get("Authorization"))
		var body map[string]any
		if !assert.NoError(t, json.NewDecoder(r.Body).Decode(&body)) {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		mu.Lock()
		requests = append(requests, body)
		n := len(requests)
		mu.Unlock()

		message := map[string]any{"role": "assistant", "content": "Rex is in store 7."}
		finish := "stop"
		if n == 1 {
			message = map[string]any{"role": "assistant", "content": nil, "tool_calls": []any{map[string]any{
				"id": "call_1", "type": "function", "function": map[string]any{"name": toolName, "arguments": arguments},
			}}}
			finish = "tool_calls"
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id": fmt.Sprintf("chatcmpl-%d", n), "object": "chat.completion", "created": 1, "model": "gpt-test",
			"choices": []any{map[string]any{"index": 0, "message": message, "finish_reason": finish}},
			"usage":   map[string]any{"prompt_tokens": 10, "completion_tokens": 5, "total_tokens": 15},
		})
	}))
	t.Cleanup(server.Close)
	return server, func() []map[string]any {
		mu.Lock()
		defer mu.Unlock()
		return append([]map[string]any(nil), requests...)
	}
}

func TestLLMExecutor_AutoMode_OpenAIProviderCallsAPITool(t *testing.T) {
	api, captured := captureServer(t, http.StatusCreated, `{"id": 7, "name": "Rex"}`)
	llm := llmWithAPIClient(petClient(api.URL))
	openAI, requests := fakeOpenAI(t, "createPet", `{"path_params":{"storeId":"main"},"body":{"name":"Rex"}}`)

	config := apiToolsConfig("openai", "createPet")
	config["base_url"] = openAI.URL
	config["instruction"] = "You manage a pet store."
	out, err := llm.Execute(ownerContext(testOwner), config, nil)
	require.NoError(t, err)

	assert.Equal(t, "POST", captured.method)
	assert.Equal(t, "/stores/main/pets", captured.path)
	assert.JSONEq(t, `{"name":"Rex"}`, captured.body)

	sent := requests()
	require.Len(t, sent, 2)
	tools := sent[0]["tools"].([]any)
	require.Len(t, tools, 1)
	function := tools[0].(map[string]any)["function"].(map[string]any)
	assert.Equal(t, "createPet", function["name"])
	assert.Equal(t, `POST /stores/{storeId}/pets (API "Pets")`, function["description"])
	assert.Contains(t, function["parameters"].(map[string]any)["properties"], "path_params")

	messages := sent[1]["messages"].([]any)
	require.Len(t, messages, 4)
	assert.Equal(t, map[string]any{"role": "system", "content": "You manage a pet store."}, messages[0])
	assert.Equal(t, map[string]any{"role": "user", "content": "Add Rex to the main store"}, messages[1])
	call := messages[2].(map[string]any)
	assert.Equal(t, "assistant", call["role"])
	assert.Nil(t, call["content"])
	assert.Equal(t, []any{map[string]any{"id": "call_1", "type": "function", "function": map[string]any{
		"name": "createPet", "arguments": `{"path_params":{"storeId":"main"},"body":{"name":"Rex"}}`,
	}}}, call["tool_calls"])
	result := messages[3].(map[string]any)
	assert.Equal(t, "tool", result["role"])
	assert.Equal(t, "call_1", result["tool_call_id"])
	assert.JSONEq(t, `{"status":201,"body":{"id":7,"name":"Rex"}}`, result["content"].(string))

	output := out.(map[string]any)
	assert.Equal(t, "Rex is in store 7.", output["content"])
	assert.Equal(t, "finish", output["stopped_reason"])
	assert.Equal(t, 2, output["total_iterations"])
	executions := output["tool_executions"].([]any)
	require.Len(t, executions, 1)
	assert.Equal(t, "createPet", executions[0].(map[string]any)["function_name"])
	assert.NotContains(t, executions[0], "error")
}

func TestLLMExecutor_AutoMode_APIToolErrorGoesToModel(t *testing.T) {
	api, _ := captureServer(t, http.StatusConflict, `{"message": "Rex already exists"}`)
	llm := llmWithAPIClient(petClient(api.URL))

	var lastToolMessage models.LLMMessage
	calls := 0
	llm.RegisterProvider("mock", &MockLLMProvider{ExecuteFn: func(_ context.Context, req *models.LLMRequest) (*models.LLMResponse, error) {
		calls++
		if calls == 1 {
			return toolCallResponse("call-1", "createPet", `{"path_params":{"storeId":"main"},"body":{"name":"Rex"}}`), nil
		}
		lastToolMessage = req.Messages[len(req.Messages)-1]
		return &models.LLMResponse{Content: "Rex is already there.", FinishReason: "stop"}, nil
	}})

	out, err := llm.Execute(ownerContext(testOwner), apiToolsConfig("mock", "createPet"), nil)
	require.NoError(t, err)
	assert.Equal(t, "tool", lastToolMessage.Role)
	assert.Equal(t, `Error: HTTP 409: {"message":"Rex already exists"}`, lastToolMessage.Content)
	assert.Equal(t, "Rex is already there.", out.(map[string]any)["content"])

	config := apiToolsConfig("mock", "createPet")
	config["tool_call_config"] = map[string]any{"mode": "auto", "stop_on_tool_failure": true}
	calls = 0
	_, err = llm.Execute(ownerContext(testOwner), config, nil)
	assert.ErrorContains(t, err, "HTTP 409")
}

func TestLLMExecutor_AutoMode_APIToolNamesStayUnique(t *testing.T) {
	functions := models.NewFunctionRegistry()
	functions.Register("health", func(map[string]any) (any, error) { return "ok", nil })
	llm := NewLLMExecutor()
	llm.SetToolCallingRegistry(NewToolCallingRegistry(functions))
	llm.toolRegistry().SetOpenAPIExecutor(NewAPIOperationExecutor(fakeAPIClients{"client-1": petClient("https://pets.test")}, nil))

	var tools []models.LLMTool
	llm.RegisterProvider("mock", &MockLLMProvider{ExecuteFn: func(_ context.Context, req *models.LLMRequest) (*models.LLMResponse, error) {
		tools = req.Tools
		return &models.LLMResponse{Content: "ok", FinishReason: "stop"}, nil
	}})

	config := apiToolsConfig("mock", "health")
	config["functions"] = []any{map[string]any{"type": "builtin", "name": "health", "builtin_name": "health", "description": "Built-in"}}
	_, err := llm.Execute(ownerContext(testOwner), config, nil)
	require.NoError(t, err)
	require.Len(t, tools, 2)
	assert.Equal(t, "health", tools[0].Function.Name, "configured functions keep their names")
	assert.Equal(t, "health_2", tools[1].Function.Name)
}

func TestLLMExecutor_APIToolsConfigErrors(t *testing.T) {
	cases := map[string]struct {
		apiTools any
		err      string
	}{
		"not an array":  {"createPet", "api_tools must be an array"},
		"not an object": {[]any{"createPet"}, "api_tools[0] must be an object"},
		"no client":     {[]any{map[string]any{"operation_ids": []any{"createPet"}}}, "api_tools[0]: client_id is required"},
		"no operations": {[]any{map[string]any{"client_id": "client-1", "operation_ids": []any{" "}}}, "api_tools[0]: operation_ids must list at least one operation"},
	}
	llm := llmWithAPIClient(petClient("https://pets.test"))
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			config := apiToolsConfig("openai", "createPet")
			config["api_tools"] = tc.apiTools
			assert.ErrorContains(t, llm.Validate(config), tc.err)
			_, err := llm.Execute(ownerContext(testOwner), config, nil)
			assert.ErrorContains(t, err, tc.err)
		})
	}

	withoutAPIClients := NewLLMExecutor()
	withoutAPIClients.SetToolCallingRegistry(NewToolCallingRegistry(models.NewFunctionRegistry()))
	withoutAPIClients.RegisterProvider("mock", &MockLLMProvider{})
	_, err := withoutAPIClients.Execute(ownerContext(testOwner), apiToolsConfig("mock", "createPet"), nil)
	assert.ErrorContains(t, err, "API tools are not available")

	_, err = llmWithAPIClient(petClient("https://pets.test")).Execute(ownerContext("someone-else"), apiToolsConfig("openai", "createPet"), nil)
	assert.ErrorIs(t, err, models.ErrAPIClientNotFound, "tools are built only from the owner's clients")
}

func TestLLMExecutor_AutoMode_ToolTimeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case <-r.Context().Done():
		case <-time.After(10 * time.Second):
		}
	}))
	defer server.Close()

	llm := llmWithAPIClient(petClient(server.URL))
	calls := 0
	llm.RegisterProvider("mock", &MockLLMProvider{ExecuteFn: func(_ context.Context, req *models.LLMRequest) (*models.LLMResponse, error) {
		calls++
		if calls == 1 {
			return toolCallResponse("call-1", "health", `{}`), nil
		}
		return &models.LLMResponse{Content: "done", FinishReason: "stop"}, nil
	}})

	config := apiToolsConfig("mock", "health")
	config["tool_call_config"] = map[string]any{"mode": "auto", "timeout_per_tool": 1}
	started := time.Now()
	out, err := llm.Execute(ownerContext(testOwner), config, nil)
	require.NoError(t, err)
	assert.Less(t, time.Since(started), 5*time.Second)

	executions := out.(map[string]any)["tool_executions"].([]any)
	require.Len(t, executions, 1)
	assert.Contains(t, executions[0].(map[string]any)["error"], "deadline exceeded")
}

func TestOpenAIProvider_HistoryKeepsAttachments(t *testing.T) {
	provider, err := NewOpenAIProvider("sk-test", "", "")
	require.NoError(t, err)
	body := provider.buildRequestBody(&models.LLMRequest{
		Model: "gpt-test", Instruction: "ignored", Prompt: "What is this?", ImageURLs: []string{"https://img.test/rex.png"},
		Messages: []models.LLMMessage{
			{Role: "user", Content: "What is this?"},
			{Role: "assistant", Content: "A dog."},
			{Role: "user", Content: "Which breed?"},
		},
	})

	messages := body["messages"].([]map[string]any)
	require.Len(t, messages, 3, "the history replaces the instruction and the prompt")
	assert.Len(t, messages[0]["content"], 2, "the first user message carries the image")
	assert.Equal(t, "A dog.", messages[1]["content"])
	assert.Equal(t, "Which breed?", messages[2]["content"])
}

func TestGeminiProvider_BuildRequestBodyWithHistory(t *testing.T) {
	provider, err := NewGeminiProvider("test-key", "")
	require.NoError(t, err)
	body := provider.buildRequestBody(&models.LLMRequest{
		Instruction: "ignored",
		Prompt:      "Find Rex",
		ImageURLs:   []string{"https://img.test/rex.png"},
		Messages: []models.LLMMessage{
			{Role: "system", Content: "You manage a pet store."},
			{Role: "user", Content: "Find Rex"},
			{Role: "assistant", ToolCalls: []models.LLMToolCall{
				{ID: "getPet", Function: models.LLMFunctionCall{Name: "getPet", Arguments: `{"path_params":{"petId":7}}`}, Signature: "sig-1"},
				{ID: "listPets", Function: models.LLMFunctionCall{Name: "listPets"}},
			}},
			{Role: "tool", ToolCallID: "getPet", Name: "getPet", Content: `{"status":200,"body":{"name":"Rex"}}`},
			{Role: "tool", ToolCallID: "listPets", Name: "listPets", Content: "Error: HTTP 500: oops"},
		},
	})

	assert.Equal(t, map[string]any{"parts": []map[string]any{{"text": "You manage a pet store."}}}, body["systemInstruction"])
	contents := body["contents"].([]map[string]any)
	require.Len(t, contents, 3)
	assert.Equal(t, "user", contents[0]["role"])
	assert.Len(t, contents[0]["parts"], 2, "the first user message carries the image")
	assert.Equal(t, map[string]any{"role": "model", "parts": []map[string]any{
		{"functionCall": map[string]any{"name": "getPet", "args": map[string]any{"path_params": map[string]any{"petId": float64(7)}}}, "thoughtSignature": "sig-1"},
		{"functionCall": map[string]any{"name": "listPets", "args": map[string]any{}}},
	}}, contents[1])
	assert.Equal(t, map[string]any{"role": "user", "parts": []map[string]any{
		{"functionResponse": map[string]any{"name": "getPet", "response": map[string]any{"status": float64(200), "body": map[string]any{"name": "Rex"}}}},
		{"functionResponse": map[string]any{"name": "listPets", "response": map[string]any{"result": "Error: HTTP 500: oops"}}},
	}}, contents[2], "results of one turn are sent together")
}

func TestGeminiProvider_KeepsThoughtSignature(t *testing.T) {
	provider, err := NewGeminiProvider("test-key", "")
	require.NoError(t, err)
	var resp geminiGenerateContentResponse
	require.NoError(t, json.Unmarshal([]byte(`{"candidates":[{"content":{"role":"model","parts":[
		{"functionCall":{"name":"getPet","args":{"id":7}},"thoughtSignature":"sig-1"}]},"finishReason":"STOP"}]}`), &resp))

	out := provider.convertResponse(&resp, &models.LLMRequest{Model: "gemini-test"})
	require.Len(t, out.ToolCalls, 1)
	assert.Equal(t, "sig-1", out.ToolCalls[0].Signature)
}

func TestOpenAIResponsesProvider_BuildRequestBodyWithHistory(t *testing.T) {
	provider, err := NewOpenAIResponsesProvider("sk-test", "", "")
	require.NoError(t, err)
	body := provider.buildRequestBody(&models.LLMRequest{
		Model: "gpt-test", Instruction: "You manage a pet store.", Prompt: "Find Rex",
		Messages: []models.LLMMessage{
			{Role: "system", Content: "You manage a pet store."},
			{Role: "user", Content: "Find Rex"},
			{Role: "assistant", Content: "Looking it up.", ToolCalls: []models.LLMToolCall{
				{ID: "call_1", Function: models.LLMFunctionCall{Name: "getPet", Arguments: `{"path_params":{"petId":7}}`}},
			}},
			{Role: "tool", ToolCallID: "call_1", Name: "getPet", Content: `{"status":200}`},
		},
	})

	assert.NotContains(t, body, "instructions", "the history carries the system message")
	assert.Equal(t, []map[string]any{
		{"role": "system", "content": "You manage a pet store."},
		{"role": "user", "content": "Find Rex"},
		{"role": "assistant", "content": "Looking it up."},
		{"type": "function_call", "call_id": "call_1", "name": "getPet", "arguments": `{"path_params":{"petId":7}}`},
		{"type": "function_call_output", "call_id": "call_1", "output": `{"status":200}`},
	}, body["input"])
}
