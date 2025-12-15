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

func TestTelegramDownloadExecutor_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  map[string]interface{}
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid config with base64",
			config: map[string]interface{}{
				"bot_token":     "123456:ABC-DEF",
				"file_id":       "AgACAgIAAxkBAAIC...",
				"output_format": "base64",
			},
			wantErr: false,
		},
		{
			name: "valid config with url",
			config: map[string]interface{}{
				"bot_token":     "123456:ABC-DEF",
				"file_id":       "AgACAgIAAxkBAAIC...",
				"output_format": "url",
			},
			wantErr: false,
		},
		{
			name: "valid config default format",
			config: map[string]interface{}{
				"bot_token": "123456:ABC-DEF",
				"file_id":   "AgACAgIAAxkBAAIC...",
			},
			wantErr: false,
		},
		{
			name: "missing bot_token",
			config: map[string]interface{}{
				"file_id": "AgACAgIAAxkBAAIC...",
			},
			wantErr: true,
			errMsg:  "bot_token",
		},
		{
			name: "missing file_id",
			config: map[string]interface{}{
				"bot_token": "123456:ABC-DEF",
			},
			wantErr: true,
			errMsg:  "file_id",
		},
		{
			name: "invalid output_format",
			config: map[string]interface{}{
				"bot_token":     "123456:ABC-DEF",
				"file_id":       "AgACAgIAAxkBAAIC...",
				"output_format": "invalid",
			},
			wantErr: true,
			errMsg:  "invalid output_format",
		},
		{
			name: "timeout too small",
			config: map[string]interface{}{
				"bot_token": "123456:ABC-DEF",
				"file_id":   "AgACAgIAAxkBAAIC...",
				"timeout":   0,
			},
			wantErr: true,
			errMsg:  "timeout must be between",
		},
		{
			name: "timeout too large",
			config: map[string]interface{}{
				"bot_token": "123456:ABC-DEF",
				"file_id":   "AgACAgIAAxkBAAIC...",
				"timeout":   400,
			},
			wantErr: true,
			errMsg:  "timeout must be between",
		},
	}

	executor := NewTelegramDownloadExecutor()

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

func TestTelegramDownloadExecutor_Execute_URLFormat(t *testing.T) {
	// Mock server that simulates Telegram's getFile API
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/bottest-token/getFile" {
			response := getFileResponse{
				OK: true,
				Result: &struct {
					FileID       string `json:"file_id"`
					FileUniqueID string `json:"file_unique_id"`
					FileSize     int64  `json:"file_size,omitempty"`
					FilePath     string `json:"file_path,omitempty"`
				}{
					FileID:       "test-file-id",
					FileUniqueID: "test-unique-id",
					FileSize:     1024,
					FilePath:     "documents/file.txt",
				},
			}
			json.NewEncoder(w).Encode(response)
			return
		}
		http.NotFound(w, r)
	}))
	defer server.Close()

	executor := NewTelegramDownloadExecutor()
	executor.baseURL = server.URL

	result, err := executor.Execute(context.Background(), map[string]interface{}{
		"bot_token":     "test-token",
		"file_id":       "test-file-id",
		"output_format": "url",
	}, nil)

	require.NoError(t, err)
	resultMap := result.(map[string]interface{})

	assert.True(t, resultMap["success"].(bool))
	assert.Equal(t, "test-file-id", resultMap["file_id"])
	assert.Equal(t, "test-unique-id", resultMap["file_unique_id"])
	assert.Equal(t, int64(1024), resultMap["file_size"])
	assert.Equal(t, "documents/file.txt", resultMap["file_path"])
	assert.Contains(t, resultMap["file_url"], "/file/bottest-token/documents/file.txt")
	assert.NotContains(t, resultMap, "file_data") // No base64 data for url format
}

func TestTelegramDownloadExecutor_Execute_Base64Format(t *testing.T) {
	fileContent := []byte("Hello, Telegram!")

	// Mock server for getFile and file download
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/bottest-token/getFile":
			response := getFileResponse{
				OK: true,
				Result: &struct {
					FileID       string `json:"file_id"`
					FileUniqueID string `json:"file_unique_id"`
					FileSize     int64  `json:"file_size,omitempty"`
					FilePath     string `json:"file_path,omitempty"`
				}{
					FileID:       "test-file-id",
					FileUniqueID: "test-unique-id",
					FileSize:     int64(len(fileContent)),
					FilePath:     "documents/test.txt",
				},
			}
			json.NewEncoder(w).Encode(response)
		case "/file/bottest-token/documents/test.txt":
			w.Write(fileContent)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	executor := NewTelegramDownloadExecutor()
	executor.baseURL = server.URL

	result, err := executor.Execute(context.Background(), map[string]interface{}{
		"bot_token":     "test-token",
		"file_id":       "test-file-id",
		"output_format": "base64",
	}, nil)

	require.NoError(t, err)
	resultMap := result.(map[string]interface{})

	assert.True(t, resultMap["success"].(bool))
	assert.Equal(t, "SGVsbG8sIFRlbGVncmFtIQ==", resultMap["file_data"]) // base64 of "Hello, Telegram!"
}

func TestTelegramDownloadExecutor_Execute_APIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := getFileResponse{
			OK:          false,
			Description: "Bad Request: file is too big",
			ErrorCode:   400,
		}
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	executor := NewTelegramDownloadExecutor()
	executor.baseURL = server.URL

	_, err := executor.Execute(context.Background(), map[string]interface{}{
		"bot_token": "test-token",
		"file_id":   "large-file-id",
	}, nil)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "file is too big")
}

func TestTelegramDownloadExecutor_Execute_FileNotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := getFileResponse{
			OK:          false,
			Description: "Bad Request: invalid file_id",
			ErrorCode:   400,
		}
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	executor := NewTelegramDownloadExecutor()
	executor.baseURL = server.URL

	_, err := executor.Execute(context.Background(), map[string]interface{}{
		"bot_token": "test-token",
		"file_id":   "invalid-file-id",
	}, nil)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid file_id")
}

func TestTelegramDownloadExecutor_Execute_DownloadError(t *testing.T) {
	// Server that returns file info but fails on download
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/bottest-token/getFile" {
			response := getFileResponse{
				OK: true,
				Result: &struct {
					FileID       string `json:"file_id"`
					FileUniqueID string `json:"file_unique_id"`
					FileSize     int64  `json:"file_size,omitempty"`
					FilePath     string `json:"file_path,omitempty"`
				}{
					FileID:   "test-file-id",
					FilePath: "documents/test.txt",
				},
			}
			json.NewEncoder(w).Encode(response)
			return
		}
		// File download fails
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	executor := NewTelegramDownloadExecutor()
	executor.baseURL = server.URL

	_, err := executor.Execute(context.Background(), map[string]interface{}{
		"bot_token":     "test-token",
		"file_id":       "test-file-id",
		"output_format": "base64",
	}, nil)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "download failed")
}
