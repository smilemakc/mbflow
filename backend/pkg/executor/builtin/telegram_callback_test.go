package builtin

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTelegramCallbackExecutor_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  map[string]any
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid config minimal",
			config: map[string]any{
				"bot_token":         "123456:ABC-DEF",
				"callback_query_id": "callback-123",
			},
			wantErr: false,
		},
		{
			name: "valid config with all options",
			config: map[string]any{
				"bot_token":         "123456:ABC-DEF",
				"callback_query_id": "callback-123",
				"text":              "Processing...",
				"show_alert":        true,
				"cache_time":        60,
			},
			wantErr: false,
		},
		{
			name: "missing bot_token",
			config: map[string]any{
				"callback_query_id": "callback-123",
			},
			wantErr: true,
			errMsg:  "bot_token",
		},
		{
			name: "missing callback_query_id",
			config: map[string]any{
				"bot_token": "123456:ABC-DEF",
			},
			wantErr: true,
			errMsg:  "callback_query_id",
		},
		{
			name: "timeout too small",
			config: map[string]any{
				"bot_token":         "123456:ABC-DEF",
				"callback_query_id": "callback-123",
				"timeout":           0,
			},
			wantErr: true,
			errMsg:  "timeout must be between",
		},
		{
			name: "negative cache_time",
			config: map[string]any{
				"bot_token":         "123456:ABC-DEF",
				"callback_query_id": "callback-123",
				"cache_time":        -1,
			},
			wantErr: true,
			errMsg:  "cache_time cannot be negative",
		},
	}

	executor := NewTelegramCallbackExecutor()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := executor.Validate(tt.config)
			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestTelegramCallbackExecutor_Execute_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Contains(t, r.URL.Path, "/answerCallbackQuery")
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

		var payload map[string]any
		json.NewDecoder(r.Body).Decode(&payload)

		assert.Equal(t, "callback-123", payload["callback_query_id"])
		assert.Equal(t, "Done!", payload["text"])

		response := answerCallbackResponse{
			OK:     true,
			Result: true,
		}
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	executor := NewTelegramCallbackExecutor()
	executor.baseURL = server.URL

	result, err := executor.Execute(context.Background(), map[string]any{
		"bot_token":         "test-token",
		"callback_query_id": "callback-123",
		"text":              "Done!",
	}, nil)

	require.NoError(t, err)
	resultMap := result.(map[string]any)

	assert.True(t, resultMap["success"].(bool))
	assert.GreaterOrEqual(t, resultMap["duration_ms"].(int64), int64(0))
}

func TestTelegramCallbackExecutor_Execute_WithAlert(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var payload map[string]any
		json.NewDecoder(r.Body).Decode(&payload)

		assert.Equal(t, true, payload["show_alert"])
		assert.Equal(t, "Warning!", payload["text"])

		response := answerCallbackResponse{
			OK:     true,
			Result: true,
		}
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	executor := NewTelegramCallbackExecutor()
	executor.baseURL = server.URL

	result, err := executor.Execute(context.Background(), map[string]any{
		"bot_token":         "test-token",
		"callback_query_id": "callback-123",
		"text":              "Warning!",
		"show_alert":        true,
	}, nil)

	require.NoError(t, err)
	assert.True(t, result.(map[string]any)["success"].(bool))
}

func TestTelegramCallbackExecutor_Execute_WithCacheTime(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var payload map[string]any
		json.NewDecoder(r.Body).Decode(&payload)

		assert.Equal(t, float64(300), payload["cache_time"])

		response := answerCallbackResponse{OK: true, Result: true}
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	executor := NewTelegramCallbackExecutor()
	executor.baseURL = server.URL

	result, err := executor.Execute(context.Background(), map[string]any{
		"bot_token":         "test-token",
		"callback_query_id": "callback-123",
		"cache_time":        300,
	}, nil)

	require.NoError(t, err)
	assert.True(t, result.(map[string]any)["success"].(bool))
}

func TestTelegramCallbackExecutor_Execute_TextTruncation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var payload map[string]any
		json.NewDecoder(r.Body).Decode(&payload)

		text := payload["text"].(string)
		// Should be truncated to 200 chars
		assert.LessOrEqual(t, len(text), 200)

		response := answerCallbackResponse{OK: true, Result: true}
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	executor := NewTelegramCallbackExecutor()
	executor.baseURL = server.URL

	// Create text longer than 200 chars
	longText := ""
	for i := 0; i < 250; i++ {
		longText += "x"
	}

	result, err := executor.Execute(context.Background(), map[string]any{
		"bot_token":         "test-token",
		"callback_query_id": "callback-123",
		"text":              longText,
	}, nil)

	require.NoError(t, err)
	assert.True(t, result.(map[string]any)["success"].(bool))
}

func TestTelegramCallbackExecutor_Execute_APIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := answerCallbackResponse{
			OK:          false,
			Description: "Bad Request: query is too old",
			ErrorCode:   400,
		}
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	executor := NewTelegramCallbackExecutor()
	executor.baseURL = server.URL

	_, err := executor.Execute(context.Background(), map[string]any{
		"bot_token":         "test-token",
		"callback_query_id": "old-callback",
	}, nil)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "query is too old")
}

func TestTelegramCallbackExecutor_Execute_EmptyResponse(t *testing.T) {
	// Test with minimal payload (no text)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var payload map[string]any
		json.NewDecoder(r.Body).Decode(&payload)

		// Should only have callback_query_id
		assert.Equal(t, "callback-123", payload["callback_query_id"])
		assert.NotContains(t, payload, "text")
		assert.NotContains(t, payload, "show_alert")

		response := answerCallbackResponse{OK: true, Result: true}
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	executor := NewTelegramCallbackExecutor()
	executor.baseURL = server.URL

	result, err := executor.Execute(context.Background(), map[string]any{
		"bot_token":         "test-token",
		"callback_query_id": "callback-123",
	}, nil)

	require.NoError(t, err)
	assert.True(t, result.(map[string]any)["success"].(bool))
}
