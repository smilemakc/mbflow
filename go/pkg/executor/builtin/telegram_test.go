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

func TestTelegramExecutor_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  map[string]any
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid text message config",
			config: map[string]any{
				"bot_token":    "123456:ABC-DEF",
				"chat_id":      "-1001234567890",
				"message_type": "text",
				"text":         "Hello, World!",
			},
			wantErr: false,
		},
		{
			name: "missing bot_token",
			config: map[string]any{
				"chat_id":      "-1001234567890",
				"message_type": "text",
				"text":         "Hello",
			},
			wantErr: true,
			errMsg:  "bot_token",
		},
		{
			name: "missing chat_id",
			config: map[string]any{
				"bot_token":    "123456:ABC-DEF",
				"message_type": "text",
				"text":         "Hello",
			},
			wantErr: true,
			errMsg:  "chat_id",
		},
		{
			name: "missing message_type",
			config: map[string]any{
				"bot_token": "123456:ABC-DEF",
				"chat_id":   "-1001234567890",
				"text":      "Hello",
			},
			wantErr: true,
			errMsg:  "message_type",
		},
		{
			name: "invalid bot_token format",
			config: map[string]any{
				"bot_token":    "invalid-token",
				"chat_id":      "-1001234567890",
				"message_type": "text",
				"text":         "Hello",
			},
			wantErr: true,
			errMsg:  "invalid bot_token format",
		},
		{
			name: "invalid message_type",
			config: map[string]any{
				"bot_token":    "123456:ABC-DEF",
				"chat_id":      "-1001234567890",
				"message_type": "invalid",
			},
			wantErr: true,
			errMsg:  "invalid message_type",
		},
		{
			name: "text message without text",
			config: map[string]any{
				"bot_token":    "123456:ABC-DEF",
				"chat_id":      "-1001234567890",
				"message_type": "text",
			},
			wantErr: true,
			errMsg:  "text is required",
		},
		{
			name: "valid photo config with file_id",
			config: map[string]any{
				"bot_token":    "123456:ABC-DEF",
				"chat_id":      "-1001234567890",
				"message_type": "photo",
				"file_source":  "file_id",
				"file_data":    "AgACAgIAAxkBAAIC...",
			},
			wantErr: false,
		},
		{
			name: "valid photo config with url",
			config: map[string]any{
				"bot_token":    "123456:ABC-DEF",
				"chat_id":      "-1001234567890",
				"message_type": "photo",
				"file_source":  "url",
				"file_data":    "https://example.com/image.jpg",
			},
			wantErr: false,
		},
		{
			name: "valid document config with base64",
			config: map[string]any{
				"bot_token":    "123456:ABC-DEF",
				"chat_id":      "-1001234567890",
				"message_type": "document",
				"file_source":  "base64",
				"file_data":    "SGVsbG8gV29ybGQh",
				"file_name":    "document.txt",
			},
			wantErr: false,
		},
		{
			name: "media message without file_source",
			config: map[string]any{
				"bot_token":    "123456:ABC-DEF",
				"chat_id":      "-1001234567890",
				"message_type": "document",
				"file_data":    "SGVsbG8gV29ybGQh",
			},
			wantErr: true,
			errMsg:  "file_source",
		},
		{
			name: "media message without file_data",
			config: map[string]any{
				"bot_token":    "123456:ABC-DEF",
				"chat_id":      "-1001234567890",
				"message_type": "document",
				"file_source":  "url",
			},
			wantErr: true,
			errMsg:  "file_data",
		},
		{
			name: "invalid file_source",
			config: map[string]any{
				"bot_token":    "123456:ABC-DEF",
				"chat_id":      "-1001234567890",
				"message_type": "photo",
				"file_source":  "invalid",
				"file_data":    "data",
			},
			wantErr: true,
			errMsg:  "invalid file_source",
		},
		{
			name: "invalid parse_mode",
			config: map[string]any{
				"bot_token":    "123456:ABC-DEF",
				"chat_id":      "-1001234567890",
				"message_type": "text",
				"text":         "Hello",
				"parse_mode":   "InvalidMode",
			},
			wantErr: true,
			errMsg:  "invalid parse_mode",
		},
		{
			name: "valid parse_mode Markdown",
			config: map[string]any{
				"bot_token":    "123456:ABC-DEF",
				"chat_id":      "-1001234567890",
				"message_type": "text",
				"text":         "*Bold* text",
				"parse_mode":   "Markdown",
			},
			wantErr: false,
		},
		{
			name: "valid parse_mode HTML",
			config: map[string]any{
				"bot_token":    "123456:ABC-DEF",
				"chat_id":      "-1001234567890",
				"message_type": "text",
				"text":         "<b>Bold</b> text",
				"parse_mode":   "HTML",
			},
			wantErr: false,
		},
		{
			name: "timeout too small",
			config: map[string]any{
				"bot_token":    "123456:ABC-DEF",
				"chat_id":      "-1001234567890",
				"message_type": "text",
				"text":         "Hello",
				"timeout":      0,
			},
			wantErr: true,
			errMsg:  "timeout must be between",
		},
		{
			name: "timeout too large",
			config: map[string]any{
				"bot_token":    "123456:ABC-DEF",
				"chat_id":      "-1001234567890",
				"message_type": "text",
				"text":         "Hello",
				"timeout":      400,
			},
			wantErr: true,
			errMsg:  "timeout must be between",
		},
	}

	executor := NewTelegramExecutor()

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

func TestTelegramExecutor_Execute_TextMessage(t *testing.T) {
	// Mock HTTP server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Contains(t, r.URL.Path, "/sendMessage")
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

		var payload map[string]any
		err := json.NewDecoder(r.Body).Decode(&payload)
		require.NoError(t, err)

		assert.Equal(t, "123456", payload["chat_id"])
		assert.Equal(t, "Test message", payload["text"])
		assert.Equal(t, "Markdown", payload["parse_mode"])

		// Return mock response
		response := telegramAPIResponse{
			OK: true,
			Result: &telegramMessage{
				MessageID: 42,
				Date:      1234567890,
				Chat: struct {
					ID int64 `json:"id"`
				}{ID: 123456},
				Text: "Test message",
			},
		}
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	executor := NewTelegramExecutor()
	executor.baseURL = server.URL // Override base URL for testing

	config := map[string]any{
		"bot_token":    "test-token",
		"chat_id":      "123456",
		"message_type": "text",
		"text":         "Test message",
		"parse_mode":   "Markdown",
	}

	// Parse config to get request
	req, err := executor.parseConfig(config)
	require.NoError(t, err)

	result, err := executor.sendTextMessage(context.Background(), req)

	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, result.Success)
	assert.Equal(t, 42, result.MessageID)
	assert.Equal(t, int64(123456), result.ChatID)
	assert.Equal(t, "Test message", result.Text)
}

func TestTelegramExecutor_Execute_TextMessage_WithOptions(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var payload map[string]any
		json.NewDecoder(r.Body).Decode(&payload)

		assert.Equal(t, true, payload["disable_web_page_preview"])
		assert.Equal(t, true, payload["disable_notification"])
		assert.Equal(t, true, payload["protect_content"])
		assert.Equal(t, float64(100), payload["reply_to_message_id"])

		response := telegramAPIResponse{
			OK: true,
			Result: &telegramMessage{
				MessageID: 43,
				Date:      1234567890,
				Chat: struct {
					ID int64 `json:"id"`
				}{ID: 123456},
				Text: "Test",
			},
		}
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	executor := NewTelegramExecutor()
	executor.baseURL = server.URL

	config := map[string]any{
		"bot_token":                "test-token",
		"chat_id":                  "123456",
		"message_type":             "text",
		"text":                     "Test",
		"disable_web_page_preview": true,
		"disable_notification":     true,
		"protect_content":          true,
		"reply_to_message_id":      100,
	}

	req, err := executor.parseConfig(config)
	require.NoError(t, err)

	result, err := executor.sendTextMessage(context.Background(), req)

	require.NoError(t, err)
	assert.True(t, result.Success)
}

func TestTelegramExecutor_Execute_APIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := telegramAPIResponse{
			OK:          false,
			Description: "Forbidden: bot was blocked by the user",
			ErrorCode:   403,
		}
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	executor := NewTelegramExecutor()
	executor.baseURL = server.URL

	config := map[string]any{
		"bot_token":    "test-token",
		"chat_id":      "123456",
		"message_type": "text",
		"text":         "Test",
	}

	req, err := executor.parseConfig(config)
	require.NoError(t, err)

	result, err := executor.sendTextMessage(context.Background(), req)

	require.NoError(t, err) // Executor returns error in response, not as error
	assert.False(t, result.Success)
	assert.Equal(t, 403, result.ErrorCode)
	assert.Equal(t, "Forbidden: bot was blocked by the user", result.Error)
}

func TestTelegramExecutor_Execute_PhotoByFileID(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Contains(t, r.URL.Path, "/sendPhoto")

		var payload map[string]any
		json.NewDecoder(r.Body).Decode(&payload)

		assert.Equal(t, "AgACAgIAAxkBAAIC...", payload["photo"])
		assert.Equal(t, "123456", payload["chat_id"])
		assert.Equal(t, "Photo caption", payload["caption"])

		response := telegramAPIResponse{
			OK: true,
			Result: &telegramMessage{
				MessageID: 50,
				Date:      1234567890,
				Chat: struct {
					ID int64 `json:"id"`
				}{ID: 123456},
				Caption: "Photo caption",
				Photo: []telegramFile{
					{FileID: "AgACAgIAAxkBAAIC...", FileUniqueID: "AQADAgAD...", FileSize: 12345},
				},
			},
		}
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	executor := NewTelegramExecutor()
	executor.baseURL = server.URL

	config := map[string]any{
		"bot_token":    "test-token",
		"chat_id":      "123456",
		"message_type": "photo",
		"file_source":  "file_id",
		"file_data":    "AgACAgIAAxkBAAIC...",
		"text":         "Photo caption",
	}

	req, err := executor.parseConfig(config)
	require.NoError(t, err)

	result, err := executor.sendPhoto(context.Background(), req)

	require.NoError(t, err)
	assert.True(t, result.Success)
	assert.Equal(t, 50, result.MessageID)
	assert.Equal(t, "photo", result.MessageType)
	assert.Equal(t, "Photo caption", result.Caption)
	assert.Equal(t, "AgACAgIAAxkBAAIC...", result.FileID)
	assert.Equal(t, 12345, result.FileSize)
}

func TestTelegramExecutor_Execute_PhotoByURL(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var payload map[string]any
		json.NewDecoder(r.Body).Decode(&payload)

		assert.Equal(t, "https://example.com/image.jpg", payload["photo"])

		response := telegramAPIResponse{
			OK: true,
			Result: &telegramMessage{
				MessageID: 51,
				Date:      1234567890,
				Chat: struct {
					ID int64 `json:"id"`
				}{ID: 123456},
				Photo: []telegramFile{
					{FileID: "NewFileID", FileUniqueID: "UniqueID", FileSize: 54321},
				},
			},
		}
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	executor := NewTelegramExecutor()
	executor.baseURL = server.URL

	config := map[string]any{
		"bot_token":    "test-token",
		"chat_id":      "123456",
		"message_type": "photo",
		"file_source":  "url",
		"file_data":    "https://example.com/image.jpg",
	}

	req, err := executor.parseConfig(config)
	require.NoError(t, err)

	result, err := executor.sendPhoto(context.Background(), req)

	require.NoError(t, err)
	assert.True(t, result.Success)
	assert.Equal(t, "photo", result.MessageType)
	assert.Equal(t, "NewFileID", result.FileID)
}

func TestTelegramExecutor_Execute_DocumentByBase64(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Contains(t, r.URL.Path, "/sendDocument")
		assert.Contains(t, r.Header.Get("Content-Type"), "multipart/form-data")

		// Parse multipart form
		err := r.ParseMultipartForm(10 << 20)
		require.NoError(t, err)

		assert.Equal(t, "123456", r.FormValue("chat_id"))
		assert.Equal(t, "Document caption", r.FormValue("caption"))

		// Check file exists
		_, header, err := r.FormFile("document")
		require.NoError(t, err)
		assert.Equal(t, "test.txt", header.Filename)

		response := telegramAPIResponse{
			OK: true,
			Result: &telegramMessage{
				MessageID: 60,
				Date:      1234567890,
				Chat: struct {
					ID int64 `json:"id"`
				}{ID: 123456},
				Caption: "Document caption",
				Document: &telegramFile{
					FileID:       "BQACAgIAAxkBAAID...",
					FileUniqueID: "UniqueDocID",
					FileSize:     1024,
					FileName:     "test.txt",
				},
			},
		}
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	executor := NewTelegramExecutor()
	executor.baseURL = server.URL

	// "Hello World!" in base64
	base64Data := "SGVsbG8gV29ybGQh"

	config := map[string]any{
		"bot_token":    "test-token",
		"chat_id":      "123456",
		"message_type": "document",
		"file_source":  "base64",
		"file_data":    base64Data,
		"file_name":    "test.txt",
		"text":         "Document caption",
	}

	req, err := executor.parseConfig(config)
	require.NoError(t, err)

	result, err := executor.sendDocument(context.Background(), req)

	require.NoError(t, err)
	assert.True(t, result.Success)
	assert.Equal(t, "document", result.MessageType)
	assert.Equal(t, "BQACAgIAAxkBAAID...", result.FileID)
	assert.Equal(t, 1024, result.FileSize)
}

func TestTelegramExecutor_isValidBotToken(t *testing.T) {
	executor := NewTelegramExecutor()

	tests := []struct {
		name  string
		token string
		valid bool
	}{
		{"valid token", "123456789:ABCdefGHIjklMNOpqrsTUVwxyz", true},
		{"missing colon", "123456789ABCdefGHI", false},
		{"non-numeric bot_id", "abc:ABCdefGHI", false},
		{"empty token part", "123456789:", false},
		{"empty bot_id", ":ABCdefGHI", false},
		{"multiple colons", "123:456:ABC", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := executor.isValidBotToken(tt.token)
			assert.Equal(t, tt.valid, result)
		})
	}
}

func TestTelegramExecutor_getDefaultFileName(t *testing.T) {
	executor := NewTelegramExecutor()

	tests := []struct {
		mediaField   string
		expectedName string
	}{
		{"photo", "photo.jpg"},
		{"document", "document.pdf"},
		{"audio", "audio.mp3"},
		{"video", "video.mp4"},
		{"unknown", "file.bin"},
	}

	for _, tt := range tests {
		t.Run(tt.mediaField, func(t *testing.T) {
			result := executor.getDefaultFileName(tt.mediaField)
			assert.Equal(t, tt.expectedName, result)
		})
	}
}

func TestTelegramExecutor_ResponseToMap(t *testing.T) {
	executor := NewTelegramExecutor()

	t.Run("success response with text", func(t *testing.T) {
		response := &TelegramResponse{
			Success:     true,
			MessageID:   42,
			ChatID:      123456,
			Date:        1234567890,
			MessageType: "text",
			Text:        "Hello",
			DurationMS:  100,
		}

		result := executor.responseToMap(response)

		assert.True(t, result["success"].(bool))
		assert.Equal(t, 42, result["message_id"])
		assert.Equal(t, int64(123456), result["chat_id"])
		assert.Equal(t, 1234567890, result["date"])
		assert.Equal(t, "text", result["message_type"])
		assert.Equal(t, "Hello", result["text"])
		assert.Equal(t, int64(100), result["duration_ms"])
	})

	t.Run("success response with photo", func(t *testing.T) {
		response := &TelegramResponse{
			Success:      true,
			MessageID:    43,
			ChatID:       123456,
			MessageType:  "photo",
			Caption:      "Photo caption",
			FileID:       "AgACAgIAAxkBAAIC...",
			FileUniqueID: "AQADAgAD...",
			FileSize:     12345,
			DurationMS:   200,
		}

		result := executor.responseToMap(response)

		assert.True(t, result["success"].(bool))
		assert.Equal(t, "photo", result["message_type"])
		assert.Equal(t, "Photo caption", result["caption"])
		assert.Equal(t, "AgACAgIAAxkBAAIC...", result["file_id"])
		assert.Equal(t, "AQADAgAD...", result["file_unique_id"])
		assert.Equal(t, 12345, result["file_size"])
	})

	t.Run("error response", func(t *testing.T) {
		response := &TelegramResponse{
			Success:    false,
			Error:      "Forbidden: bot was blocked",
			ErrorCode:  403,
			DurationMS: 50,
		}

		result := executor.responseToMap(response)

		assert.False(t, result["success"].(bool))
		assert.Equal(t, "Forbidden: bot was blocked", result["error"])
		assert.Equal(t, 403, result["error_code"])
		assert.Equal(t, int64(50), result["duration_ms"])
	})
}
