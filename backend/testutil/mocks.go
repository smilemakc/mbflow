package testutil

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// SetupOpenAIMock creates a mock OpenAI API server for testing
func SetupOpenAIMock(t *testing.T) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		// Default response for chat completions
		response := map[string]interface{}{
			"id":      "chatcmpl-test-123",
			"object":  "chat.completion",
			"created": 1234567890,
			"model":   "gpt-4",
			"choices": []map[string]interface{}{
				{
					"index": 0,
					"message": map[string]interface{}{
						"role":    "assistant",
						"content": "Mocked LLM response",
					},
					"finish_reason": "stop",
				},
			},
			"usage": map[string]interface{}{
				"prompt_tokens":     10,
				"completion_tokens": 20,
				"total_tokens":      30,
			},
		}

		json.NewEncoder(w).Encode(response)
	}))
}

// SetupOpenAIToolCallMock creates a mock OpenAI server that returns tool calls
func SetupOpenAIToolCallMock(t *testing.T, toolCalls []map[string]interface{}) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		response := map[string]interface{}{
			"id":      "chatcmpl-test-123",
			"object":  "chat.completion",
			"created": 1234567890,
			"model":   "gpt-4",
			"choices": []map[string]interface{}{
				{
					"index": 0,
					"message": map[string]interface{}{
						"role":       "assistant",
						"content":    nil,
						"tool_calls": toolCalls,
					},
					"finish_reason": "tool_calls",
				},
			},
			"usage": map[string]interface{}{
				"prompt_tokens":     10,
				"completion_tokens": 20,
				"total_tokens":      30,
			},
		}

		json.NewEncoder(w).Encode(response)
	}))
}

// SetupAnthropicMock creates a mock Anthropic API server for testing
func SetupAnthropicMock(t *testing.T) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		response := map[string]interface{}{
			"id":   "msg_test_123",
			"type": "message",
			"role": "assistant",
			"content": []map[string]interface{}{
				{
					"type": "text",
					"text": "Mocked Anthropic response",
				},
			},
			"model": "claude-3-5-sonnet-20241022",
			"usage": map[string]interface{}{
				"input_tokens":  10,
				"output_tokens": 20,
			},
		}

		json.NewEncoder(w).Encode(response)
	}))
}

// SetupAnthropicToolCallMock creates a mock Anthropic server that returns tool calls
func SetupAnthropicToolCallMock(t *testing.T, toolUse map[string]interface{}) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		response := map[string]interface{}{
			"id":   "msg_test_123",
			"type": "message",
			"role": "assistant",
			"content": []map[string]interface{}{
				{
					"type":  "tool_use",
					"id":    toolUse["id"],
					"name":  toolUse["name"],
					"input": toolUse["input"],
				},
			},
			"model": "claude-3-5-sonnet-20241022",
			"usage": map[string]interface{}{
				"input_tokens":  10,
				"output_tokens": 20,
			},
		}

		json.NewEncoder(w).Encode(response)
	}))
}

// SetupTelegramMock creates a mock Telegram Bot API server for testing
func SetupTelegramMock(t *testing.T) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		// Default response for sendMessage
		response := map[string]interface{}{
			"ok": true,
			"result": map[string]interface{}{
				"message_id": 123,
				"chat": map[string]interface{}{
					"id":   456,
					"type": "private",
				},
				"text": "Mocked message",
			},
		}

		json.NewEncoder(w).Encode(response)
	}))
}

// SetupTelegramErrorMock creates a mock Telegram server that returns errors
func SetupTelegramErrorMock(t *testing.T, errorCode int, description string) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		response := map[string]interface{}{
			"ok":          false,
			"error_code":  errorCode,
			"description": description,
		}

		json.NewEncoder(w).Encode(response)
	}))
}

// SetupCustomMock creates a custom mock server with a provided handler
func SetupCustomMock(t *testing.T, handler http.HandlerFunc) *httptest.Server {
	return httptest.NewServer(handler)
}
