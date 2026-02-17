package builtin

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/smilemakc/mbflow/pkg/executor"
)

// TelegramDownloadExecutor downloads files from Telegram by file_id.
type TelegramDownloadExecutor struct {
	*executor.BaseExecutor
	httpClient *http.Client
	baseURL    string // For testing purposes
	mu         sync.RWMutex
}

// NewTelegramDownloadExecutor creates a new Telegram download executor.
func NewTelegramDownloadExecutor() *TelegramDownloadExecutor {
	return &TelegramDownloadExecutor{
		BaseExecutor: executor.NewBaseExecutor("telegram_download"),
		httpClient: &http.Client{
			Timeout: 60 * time.Second, // Longer timeout for file downloads
		},
		baseURL: "https://api.telegram.org",
	}
}

// getFileResponse represents the response from getFile API call.
type getFileResponse struct {
	OK          bool   `json:"ok"`
	Description string `json:"description,omitempty"`
	ErrorCode   int    `json:"error_code,omitempty"`
	Result      *struct {
		FileID       string `json:"file_id"`
		FileUniqueID string `json:"file_unique_id"`
		FileSize     int64  `json:"file_size,omitempty"`
		FilePath     string `json:"file_path,omitempty"`
	} `json:"result,omitempty"`
}

// Execute downloads a file from Telegram.
//
// Config:
//   - bot_token: Telegram bot token (required)
//   - file_id: File ID to download (required)
//   - output_format: "base64" (default) or "url"
//   - timeout: Request timeout in seconds (default: 60)
//
// Output:
//   - success: true/false
//   - file_data: Base64 encoded content (if output_format=base64)
//   - file_url: Direct download URL (always provided)
//   - file_path: File path on Telegram servers
//   - file_size: Size in bytes
//   - file_id: Original file_id
//   - file_unique_id: Unique file identifier
//   - duration_ms: Execution time
func (e *TelegramDownloadExecutor) Execute(ctx context.Context, config map[string]any, input any) (any, error) {
	startTime := time.Now()

	// Get required fields
	botToken, err := e.GetString(config, "bot_token")
	if err != nil {
		return nil, fmt.Errorf("bot_token is required: %w", err)
	}

	fileID, err := e.GetString(config, "file_id")
	if err != nil {
		return nil, fmt.Errorf("file_id is required: %w", err)
	}

	outputFormat := e.GetStringDefault(config, "output_format", "base64")
	timeout := e.GetIntDefault(config, "timeout", 60)

	// Step 1: Get file path from Telegram API
	fileInfo, err := e.getFile(ctx, botToken, fileID, time.Duration(timeout)*time.Second)
	if err != nil {
		return nil, fmt.Errorf("failed to get file info: %w", err)
	}

	// Build file URL
	fileURL := fmt.Sprintf("%s/file/bot%s/%s", e.baseURL, botToken, fileInfo.FilePath)

	result := map[string]any{
		"success":        true,
		"file_id":        fileInfo.FileID,
		"file_unique_id": fileInfo.FileUniqueID,
		"file_path":      fileInfo.FilePath,
		"file_url":       fileURL,
		"file_size":      fileInfo.FileSize,
		"duration_ms":    time.Since(startTime).Milliseconds(),
	}

	// Step 2: Download file content if base64 format requested
	if outputFormat == "base64" {
		fileData, err := e.downloadFile(ctx, fileURL, time.Duration(timeout)*time.Second)
		if err != nil {
			return nil, fmt.Errorf("failed to download file: %w", err)
		}
		result["file_data"] = base64.StdEncoding.EncodeToString(fileData)
		result["duration_ms"] = time.Since(startTime).Milliseconds()
	}

	return result, nil
}

// getFile calls Telegram's getFile API to get file path.
func (e *TelegramDownloadExecutor) getFile(ctx context.Context, botToken, fileID string, timeout time.Duration) (*struct {
	FileID       string
	FileUniqueID string
	FileSize     int64
	FilePath     string
}, error) {
	reqCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	url := fmt.Sprintf("%s/bot%s/getFile?file_id=%s", e.baseURL, botToken, fileID)

	req, err := http.NewRequestWithContext(reqCtx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := e.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var apiResp getFileResponse
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if !apiResp.OK {
		return nil, fmt.Errorf("telegram API error: %s (code: %d)", apiResp.Description, apiResp.ErrorCode)
	}

	if apiResp.Result == nil || apiResp.Result.FilePath == "" {
		return nil, fmt.Errorf("file path not returned by Telegram API")
	}

	return &struct {
		FileID       string
		FileUniqueID string
		FileSize     int64
		FilePath     string
	}{
		FileID:       apiResp.Result.FileID,
		FileUniqueID: apiResp.Result.FileUniqueID,
		FileSize:     apiResp.Result.FileSize,
		FilePath:     apiResp.Result.FilePath,
	}, nil
}

// downloadFile downloads file content from the given URL.
func (e *TelegramDownloadExecutor) downloadFile(ctx context.Context, url string, timeout time.Duration) ([]byte, error) {
	reqCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(reqCtx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := e.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("download failed with status: %d", resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read file content: %w", err)
	}

	return data, nil
}

// Validate validates the Telegram download executor configuration.
func (e *TelegramDownloadExecutor) Validate(config map[string]any) error {
	// Validate required fields
	if err := e.ValidateRequired(config, "bot_token", "file_id"); err != nil {
		return err
	}

	// Validate output_format
	outputFormat := e.GetStringDefault(config, "output_format", "base64")
	validFormats := map[string]bool{"base64": true, "url": true}
	if !validFormats[outputFormat] {
		return fmt.Errorf("invalid output_format: %s (must be: base64, url)", outputFormat)
	}

	// Validate timeout
	if timeout := e.GetIntDefault(config, "timeout", 60); timeout < 1 || timeout > 300 {
		return fmt.Errorf("timeout must be between 1 and 300 seconds")
	}

	return nil
}
