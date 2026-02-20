package builtin

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/smilemakc/mbflow/go/pkg/executor"
)

// TelegramExecutor executes Telegram Bot API requests.
type TelegramExecutor struct {
	*executor.BaseExecutor
	httpClient *http.Client
	baseURL    string // For testing purposes
	mu         sync.RWMutex
}

// NewTelegramExecutor creates a new Telegram executor.
func NewTelegramExecutor() *TelegramExecutor {
	return &TelegramExecutor{
		BaseExecutor: executor.NewBaseExecutor("telegram"),
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		baseURL: "https://api.telegram.org",
	}
}

// Execute executes a Telegram request.
//
// Template Resolution (Automatic):
// Like LLMExecutor, this executor is automatically wrapped with TemplateExecutorWrapper.
// Templates in config are resolved BEFORE this method is called.
//
// Example workflow configuration:
//
//	config: {
//	  "bot_token": "{{env.telegram_bot_token}}",
//	  "chat_id": "{{env.telegram_chat_id}}",
//	  "message_type": "text",
//	  "text": "Workflow {{input.workflow_name}} completed with status: {{input.status}}",
//	  "parse_mode": "Markdown"
//	}
//
// After template resolution:
//
//	config: {
//	  "bot_token": "123456:ABC-DEF...",
//	  "chat_id": "-1001234567890",
//	  "message_type": "text",
//	  "text": "Workflow data-processing completed with status: success",
//	  "parse_mode": "Markdown"
//	}
func (e *TelegramExecutor) Execute(ctx context.Context, config map[string]any, input any) (any, error) {
	startTime := time.Now()

	// Parse and validate config
	req, err := e.parseConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to parse telegram config: %w", err)
	}

	// Execute request based on message type
	var response *TelegramResponse
	switch req.MessageType {
	case "text":
		response, err = e.sendTextMessage(ctx, req)
	case "photo":
		response, err = e.sendPhoto(ctx, req)
	case "document":
		response, err = e.sendDocument(ctx, req)
	case "audio":
		response, err = e.sendAudio(ctx, req)
	case "video":
		response, err = e.sendVideo(ctx, req)
	default:
		return nil, fmt.Errorf("unsupported message_type: %s", req.MessageType)
	}

	if err != nil {
		return nil, fmt.Errorf("telegram execution failed: %w", err)
	}

	// Add execution metadata
	response.DurationMS = time.Since(startTime).Milliseconds()

	// Convert to output map
	return e.responseToMap(response), nil
}

// Validate validates the Telegram executor configuration.
func (e *TelegramExecutor) Validate(config map[string]any) error {
	// Validate required fields
	if err := e.ValidateRequired(config, "bot_token", "chat_id", "message_type"); err != nil {
		return err
	}

	// Validate bot_token format
	botToken, err := e.GetString(config, "bot_token")
	if err != nil {
		return err
	}
	if !e.isValidBotToken(botToken) {
		return fmt.Errorf("invalid bot_token format (expected: <bot_id>:<token>)")
	}

	// Validate chat_id
	chatID, err := e.GetString(config, "chat_id")
	if err != nil {
		return err
	}
	if chatID == "" {
		return fmt.Errorf("chat_id cannot be empty")
	}

	// Validate message_type
	messageType, err := e.GetString(config, "message_type")
	if err != nil {
		return err
	}
	validTypes := map[string]bool{
		"text": true, "photo": true, "document": true, "audio": true, "video": true,
	}
	if !validTypes[messageType] {
		return fmt.Errorf("invalid message_type: %s (must be: text, photo, document, audio, video)", messageType)
	}

	// Validate message content based on type
	if messageType == "text" {
		if _, err := e.GetString(config, "text"); err != nil {
			return fmt.Errorf("text is required for message_type=text")
		}
	} else {
		// Media types require file_source and file_data
		if err := e.ValidateRequired(config, "file_source", "file_data"); err != nil {
			return fmt.Errorf("file_source and file_data required for media messages: %w", err)
		}

		fileSource, _ := e.GetString(config, "file_source")
		validSources := map[string]bool{"base64": true, "url": true, "file_id": true}
		if !validSources[fileSource] {
			return fmt.Errorf("invalid file_source: %s (must be: base64, url, file_id)", fileSource)
		}
	}

	// Validate parse_mode if present
	if parseMode, ok := config["parse_mode"].(string); ok && parseMode != "" {
		validModes := map[string]bool{"Markdown": true, "MarkdownV2": true, "HTML": true}
		if !validModes[parseMode] {
			return fmt.Errorf("invalid parse_mode: %s (must be: Markdown, MarkdownV2, HTML)", parseMode)
		}
	}

	// Validate timeout
	if timeout := e.GetIntDefault(config, "timeout", 30); timeout < 1 || timeout > 300 {
		return fmt.Errorf("timeout must be between 1 and 300 seconds")
	}

	return nil
}

// parseConfig parses executor config into TelegramRequest.
func (e *TelegramExecutor) parseConfig(config map[string]any) (*TelegramRequest, error) {
	req := &TelegramRequest{}

	// Required fields
	req.BotToken, _ = e.GetString(config, "bot_token")
	req.ChatID, _ = e.GetString(config, "chat_id")
	req.MessageType, _ = e.GetString(config, "message_type")

	// Text content
	req.Text = e.GetStringDefault(config, "text", "")
	req.ParseMode = e.GetStringDefault(config, "parse_mode", "")

	// Flags
	req.DisableWebPagePreview = e.GetBoolDefault(config, "disable_web_page_preview", false)
	req.DisableNotification = e.GetBoolDefault(config, "disable_notification", false)
	req.ProtectContent = e.GetBoolDefault(config, "protect_content", false)

	// Optional IDs
	req.ReplyToMessageID = e.GetIntDefault(config, "reply_to_message_id", 0)
	req.MessageThreadID = e.GetIntDefault(config, "message_thread_id", 0)

	// Media fields
	if req.MessageType != "text" {
		req.FileSource = e.GetStringDefault(config, "file_source", "")
		req.FileData = e.GetStringDefault(config, "file_data", "")
		req.FileName = e.GetStringDefault(config, "file_name", "")
	}

	// Timeout
	req.Timeout = e.GetIntDefault(config, "timeout", 30)

	return req, nil
}

// sendTextMessage sends a text message via Telegram Bot API.
func (e *TelegramExecutor) sendTextMessage(ctx context.Context, req *TelegramRequest) (*TelegramResponse, error) {
	if req.Text == "" {
		return nil, fmt.Errorf("text cannot be empty")
	}

	// Build API URL
	apiURL := fmt.Sprintf("%s/bot%s/sendMessage", e.baseURL, req.BotToken)

	// Build request body
	payload := map[string]any{
		"chat_id": req.ChatID,
		"text":    req.Text,
	}

	// Add optional parameters
	if req.ParseMode != "" {
		payload["parse_mode"] = req.ParseMode
	}
	if req.DisableWebPagePreview {
		payload["disable_web_page_preview"] = true
	}
	if req.DisableNotification {
		payload["disable_notification"] = true
	}
	if req.ProtectContent {
		payload["protect_content"] = true
	}
	if req.ReplyToMessageID > 0 {
		payload["reply_to_message_id"] = req.ReplyToMessageID
	}
	if req.MessageThreadID > 0 {
		payload["message_thread_id"] = req.MessageThreadID
	}

	// Execute HTTP request
	return e.executeAPIRequest(ctx, apiURL, payload, time.Duration(req.Timeout)*time.Second)
}

// sendPhoto sends a photo via Telegram Bot API.
func (e *TelegramExecutor) sendPhoto(ctx context.Context, req *TelegramRequest) (*TelegramResponse, error) {
	apiURL := fmt.Sprintf("%s/bot%s/sendPhoto", e.baseURL, req.BotToken)

	// Handle different file sources
	switch req.FileSource {
	case "file_id":
		return e.sendMediaByFileID(ctx, apiURL, req, "photo")
	case "url":
		return e.sendMediaByURL(ctx, apiURL, req, "photo")
	case "base64":
		return e.sendMediaByBase64(ctx, apiURL, req, "photo")
	default:
		return nil, fmt.Errorf("unsupported file_source: %s", req.FileSource)
	}
}

// sendDocument sends a document via Telegram Bot API.
func (e *TelegramExecutor) sendDocument(ctx context.Context, req *TelegramRequest) (*TelegramResponse, error) {
	apiURL := fmt.Sprintf("%s/bot%s/sendDocument", e.baseURL, req.BotToken)

	switch req.FileSource {
	case "file_id":
		return e.sendMediaByFileID(ctx, apiURL, req, "document")
	case "url":
		return e.sendMediaByURL(ctx, apiURL, req, "document")
	case "base64":
		return e.sendMediaByBase64(ctx, apiURL, req, "document")
	default:
		return nil, fmt.Errorf("unsupported file_source: %s", req.FileSource)
	}
}

// sendAudio sends audio via Telegram Bot API.
func (e *TelegramExecutor) sendAudio(ctx context.Context, req *TelegramRequest) (*TelegramResponse, error) {
	apiURL := fmt.Sprintf("%s/bot%s/sendAudio", e.baseURL, req.BotToken)

	switch req.FileSource {
	case "file_id":
		return e.sendMediaByFileID(ctx, apiURL, req, "audio")
	case "url":
		return e.sendMediaByURL(ctx, apiURL, req, "audio")
	case "base64":
		return e.sendMediaByBase64(ctx, apiURL, req, "audio")
	default:
		return nil, fmt.Errorf("unsupported file_source: %s", req.FileSource)
	}
}

// sendVideo sends video via Telegram Bot API.
func (e *TelegramExecutor) sendVideo(ctx context.Context, req *TelegramRequest) (*TelegramResponse, error) {
	apiURL := fmt.Sprintf("%s/bot%s/sendVideo", e.baseURL, req.BotToken)

	switch req.FileSource {
	case "file_id":
		return e.sendMediaByFileID(ctx, apiURL, req, "video")
	case "url":
		return e.sendMediaByURL(ctx, apiURL, req, "video")
	case "base64":
		return e.sendMediaByBase64(ctx, apiURL, req, "video")
	default:
		return nil, fmt.Errorf("unsupported file_source: %s", req.FileSource)
	}
}

// sendMediaByFileID sends media using existing file_id.
func (e *TelegramExecutor) sendMediaByFileID(ctx context.Context, apiURL string, req *TelegramRequest, mediaField string) (*TelegramResponse, error) {
	payload := map[string]any{
		"chat_id":  req.ChatID,
		mediaField: req.FileData,
	}
	e.addMediaOptions(payload, req)
	return e.executeAPIRequest(ctx, apiURL, payload, time.Duration(req.Timeout)*time.Second)
}

// sendMediaByURL sends media from URL.
func (e *TelegramExecutor) sendMediaByURL(ctx context.Context, apiURL string, req *TelegramRequest, mediaField string) (*TelegramResponse, error) {
	payload := map[string]any{
		"chat_id":  req.ChatID,
		mediaField: req.FileData,
	}
	e.addMediaOptions(payload, req)
	return e.executeAPIRequest(ctx, apiURL, payload, time.Duration(req.Timeout)*time.Second)
}

// sendMediaByBase64 sends media from base64 encoded data.
func (e *TelegramExecutor) sendMediaByBase64(ctx context.Context, apiURL string, req *TelegramRequest, mediaField string) (*TelegramResponse, error) {
	fileBytes, err := base64.StdEncoding.DecodeString(req.FileData)
	if err != nil {
		return nil, fmt.Errorf("failed to decode base64: %w", err)
	}

	return e.sendMediaMultipart(ctx, apiURL, req, fileBytes, mediaField)
}

// sendMediaMultipart sends media using multipart/form-data.
func (e *TelegramExecutor) sendMediaMultipart(ctx context.Context, apiURL string, req *TelegramRequest, fileBytes []byte, mediaField string) (*TelegramResponse, error) {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	writer.WriteField("chat_id", req.ChatID)

	// Determine file name and extension
	fileName := req.FileName
	if fileName == "" {
		fileName = e.getDefaultFileName(mediaField)
	}

	part, err := writer.CreateFormFile(mediaField, fileName)
	if err != nil {
		return nil, fmt.Errorf("failed to create form file: %w", err)
	}

	if _, err := part.Write(fileBytes); err != nil {
		return nil, fmt.Errorf("failed to write file data: %w", err)
	}

	// Add optional fields
	if req.Text != "" {
		writer.WriteField("caption", req.Text)
	}
	if req.ParseMode != "" {
		writer.WriteField("parse_mode", req.ParseMode)
	}
	if req.DisableNotification {
		writer.WriteField("disable_notification", "true")
	}
	if req.ProtectContent {
		writer.WriteField("protect_content", "true")
	}
	if req.ReplyToMessageID > 0 {
		writer.WriteField("reply_to_message_id", strconv.Itoa(req.ReplyToMessageID))
	}
	if req.MessageThreadID > 0 {
		writer.WriteField("message_thread_id", strconv.Itoa(req.MessageThreadID))
	}

	writer.Close()

	return e.executeMultipartRequest(ctx, apiURL, body, writer.FormDataContentType(), time.Duration(req.Timeout)*time.Second)
}

// addMediaOptions adds common media options to payload.
func (e *TelegramExecutor) addMediaOptions(payload map[string]any, req *TelegramRequest) {
	if req.Text != "" {
		payload["caption"] = req.Text
	}
	if req.ParseMode != "" {
		payload["parse_mode"] = req.ParseMode
	}
	if req.DisableNotification {
		payload["disable_notification"] = true
	}
	if req.ProtectContent {
		payload["protect_content"] = true
	}
	if req.ReplyToMessageID > 0 {
		payload["reply_to_message_id"] = req.ReplyToMessageID
	}
	if req.MessageThreadID > 0 {
		payload["message_thread_id"] = req.MessageThreadID
	}
}

// getDefaultFileName returns default file name for media type.
func (e *TelegramExecutor) getDefaultFileName(mediaField string) string {
	switch mediaField {
	case "photo":
		return "photo.jpg"
	case "document":
		return "document.pdf"
	case "audio":
		return "audio.mp3"
	case "video":
		return "video.mp4"
	default:
		return "file.bin"
	}
}

// executeAPIRequest executes a JSON API request.
func (e *TelegramExecutor) executeAPIRequest(ctx context.Context, url string, payload map[string]any, timeout time.Duration) (*TelegramResponse, error) {
	// Marshal payload
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal payload: %w", err)
	}

	// Create request with timeout context
	reqCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(reqCtx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	// Execute request
	resp, err := e.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	// Parse response
	return e.parseAPIResponse(resp)
}

// executeMultipartRequest executes a multipart/form-data request.
func (e *TelegramExecutor) executeMultipartRequest(ctx context.Context, url string, body *bytes.Buffer, contentType string, timeout time.Duration) (*TelegramResponse, error) {
	reqCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(reqCtx, "POST", url, body)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", contentType)

	resp, err := e.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	return e.parseAPIResponse(resp)
}

// parseAPIResponse parses Telegram API response.
func (e *TelegramExecutor) parseAPIResponse(resp *http.Response) (*TelegramResponse, error) {
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var apiResp telegramAPIResponse
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	// Check for API errors
	if !apiResp.OK {
		errorMsg := apiResp.Description
		if errorMsg == "" {
			errorMsg = "unknown error"
		}

		return &TelegramResponse{
			Success:   false,
			Error:     errorMsg,
			ErrorCode: apiResp.ErrorCode,
		}, nil
	}

	// Success - extract message details
	if apiResp.Result == nil {
		return nil, fmt.Errorf("missing result in successful response")
	}

	return e.buildSuccessResponse(apiResp.Result), nil
}

// buildSuccessResponse builds TelegramResponse from API message.
func (e *TelegramExecutor) buildSuccessResponse(msg *telegramMessage) *TelegramResponse {
	response := &TelegramResponse{
		Success:   true,
		MessageID: msg.MessageID,
		ChatID:    msg.Chat.ID,
		Date:      msg.Date,
		Text:      msg.Text,
		Caption:   msg.Caption,
	}

	// Extract file info based on message type
	if msg.Document != nil {
		response.MessageType = "document"
		response.FileID = msg.Document.FileID
		response.FileUniqueID = msg.Document.FileUniqueID
		response.FileSize = msg.Document.FileSize
	} else if len(msg.Photo) > 0 {
		response.MessageType = "photo"
		// Use largest photo size
		largestPhoto := msg.Photo[len(msg.Photo)-1]
		response.FileID = largestPhoto.FileID
		response.FileUniqueID = largestPhoto.FileUniqueID
		response.FileSize = largestPhoto.FileSize
	} else if msg.Audio != nil {
		response.MessageType = "audio"
		response.FileID = msg.Audio.FileID
		response.FileUniqueID = msg.Audio.FileUniqueID
		response.FileSize = msg.Audio.FileSize
	} else if msg.Video != nil {
		response.MessageType = "video"
		response.FileID = msg.Video.FileID
		response.FileUniqueID = msg.Video.FileUniqueID
		response.FileSize = msg.Video.FileSize
	} else {
		response.MessageType = "text"
	}

	return response
}

// responseToMap converts TelegramResponse to output map.
func (e *TelegramExecutor) responseToMap(response *TelegramResponse) map[string]any {
	result := map[string]any{
		"success":      response.Success,
		"message_type": response.MessageType,
		"duration_ms":  response.DurationMS,
	}

	if response.Success {
		result["message_id"] = response.MessageID
		result["chat_id"] = response.ChatID
		result["date"] = response.Date

		if response.Text != "" {
			result["text"] = response.Text
		}
		if response.Caption != "" {
			result["caption"] = response.Caption
		}

		// Add file info for media messages
		if response.FileID != "" {
			result["file_id"] = response.FileID
			result["file_unique_id"] = response.FileUniqueID
			if response.FileSize > 0 {
				result["file_size"] = response.FileSize
			}
		}
	} else {
		result["error"] = response.Error
		if response.ErrorCode > 0 {
			result["error_code"] = response.ErrorCode
		}
	}

	return result
}

// isValidBotToken validates bot token format (basic check).
func (e *TelegramExecutor) isValidBotToken(token string) bool {
	// Format: <bot_id>:<token>
	// Example: 123456789:ABCdefGHIjklMNOpqrsTUVwxyz
	parts := strings.Split(token, ":")
	if len(parts) != 2 {
		return false
	}

	// Check bot_id is numeric
	if _, err := strconv.Atoi(parts[0]); err != nil {
		return false
	}

	// Check token is not empty
	return len(parts[1]) > 0
}
