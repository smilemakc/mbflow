package rest

import (
	"crypto/hmac"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/smilemakc/mbflow/go/internal/application/trigger"
	"github.com/smilemakc/mbflow/go/internal/infrastructure/logger"
	"github.com/smilemakc/mbflow/go/pkg/models"
)

// TelegramWebhookHandlers provides HTTP handlers for Telegram Bot webhook
type TelegramWebhookHandlers struct {
	webhookRegistry *trigger.WebhookRegistry
	logger          *logger.Logger
}

// NewTelegramWebhookHandlers creates a new TelegramWebhookHandlers instance
func NewTelegramWebhookHandlers(webhookRegistry *trigger.WebhookRegistry, log *logger.Logger) *TelegramWebhookHandlers {
	return &TelegramWebhookHandlers{
		webhookRegistry: webhookRegistry,
		logger:          log,
	}
}

// TelegramUpdate represents an incoming Telegram update
// See: https://core.telegram.org/bots/api#update
type TelegramUpdate struct {
	UpdateID          int                    `json:"update_id"`
	Message           *TelegramMessage       `json:"message,omitempty"`
	EditedMessage     *TelegramMessage       `json:"edited_message,omitempty"`
	ChannelPost       *TelegramMessage       `json:"channel_post,omitempty"`
	EditedChannelPost *TelegramMessage       `json:"edited_channel_post,omitempty"`
	CallbackQuery     *TelegramCallbackQuery `json:"callback_query,omitempty"`
	InlineQuery       *TelegramInlineQuery   `json:"inline_query,omitempty"`
}

// TelegramMessage represents a Telegram message
type TelegramMessage struct {
	MessageID      int               `json:"message_id"`
	From           *TelegramUser     `json:"from,omitempty"`
	Chat           TelegramChat      `json:"chat"`
	Date           int               `json:"date"`
	Text           string            `json:"text,omitempty"`
	Caption        string            `json:"caption,omitempty"`
	Photo          []TelegramPhoto   `json:"photo,omitempty"`
	Document       *TelegramDocument `json:"document,omitempty"`
	Audio          *TelegramAudio    `json:"audio,omitempty"`
	Video          *TelegramVideo    `json:"video,omitempty"`
	Voice          *TelegramVoice    `json:"voice,omitempty"`
	Location       *TelegramLocation `json:"location,omitempty"`
	Contact        *TelegramContact  `json:"contact,omitempty"`
	ReplyToMessage *TelegramMessage  `json:"reply_to_message,omitempty"`
}

// TelegramUser represents a Telegram user
type TelegramUser struct {
	ID           int    `json:"id"`
	IsBot        bool   `json:"is_bot"`
	FirstName    string `json:"first_name"`
	LastName     string `json:"last_name,omitempty"`
	Username     string `json:"username,omitempty"`
	LanguageCode string `json:"language_code,omitempty"`
}

// TelegramChat represents a Telegram chat
type TelegramChat struct {
	ID        int64  `json:"id"`
	Type      string `json:"type"` // "private", "group", "supergroup", "channel"
	Title     string `json:"title,omitempty"`
	Username  string `json:"username,omitempty"`
	FirstName string `json:"first_name,omitempty"`
	LastName  string `json:"last_name,omitempty"`
}

// TelegramPhoto represents a photo size
type TelegramPhoto struct {
	FileID       string `json:"file_id"`
	FileUniqueID string `json:"file_unique_id"`
	Width        int    `json:"width"`
	Height       int    `json:"height"`
	FileSize     int    `json:"file_size,omitempty"`
}

// TelegramDocument represents a document
type TelegramDocument struct {
	FileID       string `json:"file_id"`
	FileUniqueID string `json:"file_unique_id"`
	FileName     string `json:"file_name,omitempty"`
	MimeType     string `json:"mime_type,omitempty"`
	FileSize     int    `json:"file_size,omitempty"`
}

// TelegramAudio represents an audio file
type TelegramAudio struct {
	FileID       string `json:"file_id"`
	FileUniqueID string `json:"file_unique_id"`
	Duration     int    `json:"duration"`
	Performer    string `json:"performer,omitempty"`
	Title        string `json:"title,omitempty"`
	MimeType     string `json:"mime_type,omitempty"`
	FileSize     int    `json:"file_size,omitempty"`
}

// TelegramVideo represents a video file
type TelegramVideo struct {
	FileID       string `json:"file_id"`
	FileUniqueID string `json:"file_unique_id"`
	Width        int    `json:"width"`
	Height       int    `json:"height"`
	Duration     int    `json:"duration"`
	MimeType     string `json:"mime_type,omitempty"`
	FileSize     int    `json:"file_size,omitempty"`
}

// TelegramVoice represents a voice message
type TelegramVoice struct {
	FileID       string `json:"file_id"`
	FileUniqueID string `json:"file_unique_id"`
	Duration     int    `json:"duration"`
	MimeType     string `json:"mime_type,omitempty"`
	FileSize     int    `json:"file_size,omitempty"`
}

// TelegramLocation represents a location
type TelegramLocation struct {
	Longitude float64 `json:"longitude"`
	Latitude  float64 `json:"latitude"`
}

// TelegramContact represents a contact
type TelegramContact struct {
	PhoneNumber string `json:"phone_number"`
	FirstName   string `json:"first_name"`
	LastName    string `json:"last_name,omitempty"`
	UserID      int    `json:"user_id,omitempty"`
}

// TelegramCallbackQuery represents a callback query from an inline button
type TelegramCallbackQuery struct {
	ID              string           `json:"id"`
	From            TelegramUser     `json:"from"`
	Message         *TelegramMessage `json:"message,omitempty"`
	InlineMessageID string           `json:"inline_message_id,omitempty"`
	Data            string           `json:"data,omitempty"`
	GameShortName   string           `json:"game_short_name,omitempty"`
}

// TelegramInlineQuery represents an inline query
type TelegramInlineQuery struct {
	ID       string            `json:"id"`
	From     TelegramUser      `json:"from"`
	Query    string            `json:"query"`
	Offset   string            `json:"offset"`
	ChatType string            `json:"chat_type,omitempty"`
	Location *TelegramLocation `json:"location,omitempty"`
}

// HandleTelegramWebhook handles POST /api/v1/webhooks/telegram/{trigger_id}
func (h *TelegramWebhookHandlers) HandleTelegramWebhook(c *gin.Context) {
	triggerID := c.Param("trigger_id")
	if triggerID == "" {
		respondError(c, http.StatusBadRequest, "trigger_id is required")
		return
	}

	// Get trigger to check secret_token
	trig, exists := h.webhookRegistry.GetWebhook(triggerID)
	if !exists {
		h.logger.Error("Telegram webhook trigger not found", "trigger_id", triggerID)
		respondError(c, http.StatusNotFound, "webhook trigger not found")
		return
	}

	if !trig.Enabled {
		h.logger.Error("Telegram webhook trigger is disabled", "trigger_id", triggerID)
		respondError(c, http.StatusForbidden, "webhook trigger is disabled")
		return
	}

	// Read raw body for signature validation
	bodyBytes, err := io.ReadAll(c.Request.Body)
	if err != nil {
		h.logger.Error("Failed to read request body", "error", err, "trigger_id", triggerID)
		respondError(c, http.StatusBadRequest, "failed to read request body")
		return
	}

	// Validate Telegram signature if secret_token is configured
	if secretToken, ok := trig.Config["secret_token"].(string); ok && secretToken != "" {
		if err := h.validateTelegramSignature(c, bodyBytes, secretToken); err != nil {
			h.logger.Error("Telegram signature validation failed", "error", err, "trigger_id", triggerID)
			respondError(c, http.StatusUnauthorized, "signature validation failed")
			return
		}
	}

	// Parse Telegram update
	var update TelegramUpdate
	if err := json.Unmarshal(bodyBytes, &update); err != nil {
		h.logger.Error("Failed to parse Telegram update", "error", err, "trigger_id", triggerID)
		respondError(c, http.StatusBadRequest, "invalid Telegram update format")
		return
	}

	// Convert update to workflow input
	input := h.convertUpdateToInput(&update, trig)

	// Get source IP
	sourceIP := getSourceIP(c)

	// Execute webhook
	executionID, err := h.webhookRegistry.ExecuteWebhook(
		c.Request.Context(),
		triggerID,
		input,
		extractHeaders(c),
		sourceIP,
	)
	if err != nil {
		statusCode := http.StatusInternalServerError
		errorMsg := err.Error()

		if strings.Contains(errorMsg, "not found") {
			statusCode = http.StatusNotFound
		} else if strings.Contains(errorMsg, "disabled") {
			statusCode = http.StatusForbidden
		} else if strings.Contains(errorMsg, "rate limit exceeded") {
			statusCode = http.StatusTooManyRequests
		}

		h.logger.Error("Failed to execute Telegram webhook", "error", err, "trigger_id", triggerID, "update_id", update.UpdateID)
		respondError(c, statusCode, errorMsg)
		return
	}

	h.logger.Info("Telegram webhook executed successfully",
		"trigger_id", triggerID,
		"execution_id", executionID,
		"update_id", update.UpdateID,
	)

	// Return 200 OK (Telegram expects 200)
	c.JSON(http.StatusOK, gin.H{
		"ok":           true,
		"execution_id": executionID,
	})
}

// validateTelegramSignature validates the Telegram webhook signature
// See: https://core.telegram.org/bots/api#setwebhook
func (h *TelegramWebhookHandlers) validateTelegramSignature(c *gin.Context, body []byte, secretToken string) error {
	// Get X-Telegram-Bot-Api-Secret-Token header
	providedToken := c.GetHeader("X-Telegram-Bot-Api-Secret-Token")
	if providedToken == "" {
		return fmt.Errorf("missing X-Telegram-Bot-Api-Secret-Token header")
	}

	// Simple constant-time comparison
	if !hmac.Equal([]byte(providedToken), []byte(secretToken)) {
		return fmt.Errorf("invalid secret token")
	}

	return nil
}

// convertUpdateToInput converts Telegram update to workflow input
func (h *TelegramWebhookHandlers) convertUpdateToInput(update *TelegramUpdate, trig *models.Trigger) map[string]any {
	input := make(map[string]any)

	// Add default trigger input
	if defaultInput, ok := trig.Config["input"].(map[string]any); ok {
		for k, v := range defaultInput {
			input[k] = v
		}
	}

	// Add update ID
	input["update_id"] = update.UpdateID

	// Determine update type and add relevant data
	updateType := ""
	var updateData any

	switch {
	case update.Message != nil:
		updateType = "message"
		updateData = h.messageToMap(update.Message)
	case update.EditedMessage != nil:
		updateType = "edited_message"
		updateData = h.messageToMap(update.EditedMessage)
	case update.ChannelPost != nil:
		updateType = "channel_post"
		updateData = h.messageToMap(update.ChannelPost)
	case update.EditedChannelPost != nil:
		updateType = "edited_channel_post"
		updateData = h.messageToMap(update.EditedChannelPost)
	case update.CallbackQuery != nil:
		updateType = "callback_query"
		updateData = h.callbackQueryToMap(update.CallbackQuery)
	case update.InlineQuery != nil:
		updateType = "inline_query"
		updateData = h.inlineQueryToMap(update.InlineQuery)
	}

	input["update_type"] = updateType
	if updateData != nil {
		input[updateType] = updateData
	}

	// Add convenience fields for common use cases
	if update.Message != nil {
		input["text"] = update.Message.Text
		input["chat_id"] = update.Message.Chat.ID
		if update.Message.From != nil {
			input["user_id"] = update.Message.From.ID
			input["username"] = update.Message.From.Username
			input["first_name"] = update.Message.From.FirstName
		}
	}

	return input
}

// messageToMap converts TelegramMessage to map
func (h *TelegramWebhookHandlers) messageToMap(msg *TelegramMessage) map[string]any {
	result := map[string]any{
		"message_id": msg.MessageID,
		"date":       msg.Date,
		"chat": map[string]any{
			"id":    msg.Chat.ID,
			"type":  msg.Chat.Type,
			"title": msg.Chat.Title,
		},
	}

	if msg.From != nil {
		result["from"] = map[string]any{
			"id":         msg.From.ID,
			"is_bot":     msg.From.IsBot,
			"first_name": msg.From.FirstName,
			"last_name":  msg.From.LastName,
			"username":   msg.From.Username,
		}
	}

	if msg.Text != "" {
		result["text"] = msg.Text
	}

	if msg.Caption != "" {
		result["caption"] = msg.Caption
	}

	if len(msg.Photo) > 0 {
		photos := make([]map[string]any, len(msg.Photo))
		for i, p := range msg.Photo {
			photos[i] = map[string]any{
				"file_id":        p.FileID,
				"file_unique_id": p.FileUniqueID,
				"width":          p.Width,
				"height":         p.Height,
				"file_size":      p.FileSize,
			}
		}
		result["photo"] = photos
	}

	if msg.Document != nil {
		result["document"] = map[string]any{
			"file_id":        msg.Document.FileID,
			"file_unique_id": msg.Document.FileUniqueID,
			"file_name":      msg.Document.FileName,
			"mime_type":      msg.Document.MimeType,
			"file_size":      msg.Document.FileSize,
		}
	}

	if msg.Audio != nil {
		result["audio"] = map[string]any{
			"file_id":        msg.Audio.FileID,
			"file_unique_id": msg.Audio.FileUniqueID,
			"duration":       msg.Audio.Duration,
			"performer":      msg.Audio.Performer,
			"title":          msg.Audio.Title,
			"mime_type":      msg.Audio.MimeType,
			"file_size":      msg.Audio.FileSize,
		}
	}

	if msg.Video != nil {
		result["video"] = map[string]any{
			"file_id":        msg.Video.FileID,
			"file_unique_id": msg.Video.FileUniqueID,
			"width":          msg.Video.Width,
			"height":         msg.Video.Height,
			"duration":       msg.Video.Duration,
			"mime_type":      msg.Video.MimeType,
			"file_size":      msg.Video.FileSize,
		}
	}

	if msg.Voice != nil {
		result["voice"] = map[string]any{
			"file_id":        msg.Voice.FileID,
			"file_unique_id": msg.Voice.FileUniqueID,
			"duration":       msg.Voice.Duration,
			"mime_type":      msg.Voice.MimeType,
			"file_size":      msg.Voice.FileSize,
		}
	}

	if msg.Location != nil {
		result["location"] = map[string]any{
			"longitude": msg.Location.Longitude,
			"latitude":  msg.Location.Latitude,
		}
	}

	if msg.Contact != nil {
		result["contact"] = map[string]any{
			"phone_number": msg.Contact.PhoneNumber,
			"first_name":   msg.Contact.FirstName,
			"last_name":    msg.Contact.LastName,
			"user_id":      msg.Contact.UserID,
		}
	}

	return result
}

// callbackQueryToMap converts TelegramCallbackQuery to map
func (h *TelegramWebhookHandlers) callbackQueryToMap(query *TelegramCallbackQuery) map[string]any {
	result := map[string]any{
		"id":   query.ID,
		"data": query.Data,
		"from": map[string]any{
			"id":         query.From.ID,
			"is_bot":     query.From.IsBot,
			"first_name": query.From.FirstName,
			"last_name":  query.From.LastName,
			"username":   query.From.Username,
		},
	}

	if query.Message != nil {
		result["message"] = h.messageToMap(query.Message)
	}

	if query.InlineMessageID != "" {
		result["inline_message_id"] = query.InlineMessageID
	}

	return result
}

// inlineQueryToMap converts TelegramInlineQuery to map
func (h *TelegramWebhookHandlers) inlineQueryToMap(query *TelegramInlineQuery) map[string]any {
	result := map[string]any{
		"id":     query.ID,
		"query":  query.Query,
		"offset": query.Offset,
		"from": map[string]any{
			"id":         query.From.ID,
			"is_bot":     query.From.IsBot,
			"first_name": query.From.FirstName,
			"last_name":  query.From.LastName,
			"username":   query.From.Username,
		},
	}

	if query.ChatType != "" {
		result["chat_type"] = query.ChatType
	}

	if query.Location != nil {
		result["location"] = map[string]any{
			"longitude": query.Location.Longitude,
			"latitude":  query.Location.Latitude,
		}
	}

	return result
}

// extractHeaders extracts request headers into a map
func extractHeaders(c *gin.Context) map[string]string {
	headers := make(map[string]string)
	for key, values := range c.Request.Header {
		if len(values) > 0 {
			headers[key] = values[0]
		}
	}
	return headers
}
