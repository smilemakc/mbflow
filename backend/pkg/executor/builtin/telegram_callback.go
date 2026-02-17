package builtin

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/smilemakc/mbflow/pkg/executor"
)

// TelegramCallbackExecutor answers callback queries from inline keyboard buttons.
type TelegramCallbackExecutor struct {
	*executor.BaseExecutor
	httpClient *http.Client
	baseURL    string // For testing purposes
	mu         sync.RWMutex
}

// NewTelegramCallbackExecutor creates a new Telegram callback executor.
func NewTelegramCallbackExecutor() *TelegramCallbackExecutor {
	return &TelegramCallbackExecutor{
		BaseExecutor: executor.NewBaseExecutor("telegram_callback"),
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		baseURL: "https://api.telegram.org",
	}
}

// answerCallbackResponse represents the API response.
type answerCallbackResponse struct {
	OK          bool   `json:"ok"`
	Description string `json:"description,omitempty"`
	ErrorCode   int    `json:"error_code,omitempty"`
	Result      bool   `json:"result,omitempty"`
}

// Execute answers a callback query.
//
// Config:
//   - bot_token: Telegram bot token (required)
//   - callback_query_id: ID of the callback query to answer (required)
//   - text: Notification text (optional, max 200 chars)
//   - show_alert: Show as alert instead of notification (default: false)
//   - url: URL to open (for game buttons only)
//   - cache_time: Cache time in seconds (default: 0)
//   - timeout: Request timeout in seconds (default: 30)
//
// Output:
//   - success: true/false
//   - duration_ms: Execution time
func (e *TelegramCallbackExecutor) Execute(ctx context.Context, config map[string]any, input any) (any, error) {
	startTime := time.Now()

	// Get required fields
	botToken, err := e.GetString(config, "bot_token")
	if err != nil {
		return nil, fmt.Errorf("bot_token is required: %w", err)
	}

	callbackQueryID, err := e.GetString(config, "callback_query_id")
	if err != nil {
		return nil, fmt.Errorf("callback_query_id is required: %w", err)
	}

	// Get optional fields
	text := e.GetStringDefault(config, "text", "")
	showAlert := e.GetBoolDefault(config, "show_alert", false)
	url := e.GetStringDefault(config, "url", "")
	cacheTime := e.GetIntDefault(config, "cache_time", 0)
	timeout := e.GetIntDefault(config, "timeout", 30)

	// Build request payload
	payload := map[string]any{
		"callback_query_id": callbackQueryID,
	}

	if text != "" {
		// Telegram limit: 200 characters
		if len(text) > 200 {
			text = text[:200]
		}
		payload["text"] = text
	}

	if showAlert {
		payload["show_alert"] = true
	}

	if url != "" {
		payload["url"] = url
	}

	if cacheTime > 0 {
		payload["cache_time"] = cacheTime
	}

	// Execute API request
	apiURL := fmt.Sprintf("%s/bot%s/answerCallbackQuery", e.baseURL, botToken)
	response, err := e.executeRequest(ctx, apiURL, payload, time.Duration(timeout)*time.Second)
	if err != nil {
		return nil, fmt.Errorf("failed to answer callback query: %w", err)
	}

	return map[string]any{
		"success":     response.OK,
		"duration_ms": time.Since(startTime).Milliseconds(),
	}, nil
}

// executeRequest sends the API request.
func (e *TelegramCallbackExecutor) executeRequest(ctx context.Context, url string, payload map[string]any, timeout time.Duration) (*answerCallbackResponse, error) {
	reqCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal payload: %w", err)
	}

	req, err := http.NewRequestWithContext(reqCtx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := e.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var apiResp answerCallbackResponse
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if !apiResp.OK {
		return nil, fmt.Errorf("telegram API error: %s (code: %d)", apiResp.Description, apiResp.ErrorCode)
	}

	return &apiResp, nil
}

// Validate validates the Telegram callback executor configuration.
func (e *TelegramCallbackExecutor) Validate(config map[string]any) error {
	// Validate required fields
	if err := e.ValidateRequired(config, "bot_token", "callback_query_id"); err != nil {
		return err
	}

	// Validate timeout
	if timeout := e.GetIntDefault(config, "timeout", 30); timeout < 1 || timeout > 300 {
		return fmt.Errorf("timeout must be between 1 and 300 seconds")
	}

	// Validate cache_time
	if cacheTime := e.GetIntDefault(config, "cache_time", 0); cacheTime < 0 {
		return fmt.Errorf("cache_time cannot be negative")
	}

	return nil
}
