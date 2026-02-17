package builtin

import (
	"context"
	"regexp"
	"strings"
	"time"

	"github.com/smilemakc/mbflow/pkg/executor"
)

// TelegramParseExecutor parses Telegram updates and extracts structured data.
type TelegramParseExecutor struct {
	*executor.BaseExecutor
}

// NewTelegramParseExecutor creates a new Telegram parse executor.
func NewTelegramParseExecutor() *TelegramParseExecutor {
	return &TelegramParseExecutor{
		BaseExecutor: executor.NewBaseExecutor("telegram_parse"),
	}
}

// TelegramFileInfo contains extracted file information.
type TelegramFileInfo struct {
	Type         string `json:"type"`
	FileID       string `json:"file_id"`
	FileUniqueID string `json:"file_unique_id"`
	FileSize     int    `json:"file_size,omitempty"`
	FileName     string `json:"file_name,omitempty"`
	MimeType     string `json:"mime_type,omitempty"`
	Width        int    `json:"width,omitempty"`
	Height       int    `json:"height,omitempty"`
	Duration     int    `json:"duration,omitempty"`
}

// Execute parses a Telegram update and extracts structured data.
//
// Config:
//   - extract_files: Extract all files from message (default: true)
//   - extract_commands: Parse /commands from text (default: true)
//   - extract_entities: Extract URLs, emails, mentions (default: false)
//
// Input: Telegram update from webhook (as map)
//
// Output:
//   - update_type: "message" | "edited_message" | "callback_query" | "inline_query" | etc.
//   - message_type: "text" | "photo" | "document" | "audio" | "video" | "voice" | "location" | "contact"
//   - text: Message text or caption
//   - command: Extracted command (e.g., "/start")
//   - command_args: Array of command arguments
//   - files: Array of file info objects
//   - callback_data: Raw callback data string (for callback_query)
//   - user: User info object
//   - chat: Chat info object
//   - reply_to_message_id: ID of replied message
func (e *TelegramParseExecutor) Execute(ctx context.Context, config map[string]any, input any) (any, error) {
	startTime := time.Now()

	extractFiles := e.GetBoolDefault(config, "extract_files", true)
	extractCommands := e.GetBoolDefault(config, "extract_commands", true)
	extractEntities := e.GetBoolDefault(config, "extract_entities", false)

	inputMap, ok := input.(map[string]any)
	if !ok {
		// If input is nil or not a map, return empty parsed result
		return map[string]any{
			"update_type":  "unknown",
			"message_type": "unknown",
			"duration_ms":  time.Since(startTime).Milliseconds(),
		}, nil
	}

	result := map[string]any{
		"duration_ms": time.Since(startTime).Milliseconds(),
	}

	// Determine update type and get message data
	updateType, messageData := e.determineUpdateType(inputMap)
	result["update_type"] = updateType

	// Extract user and chat info
	if user := e.extractUser(inputMap, updateType); user != nil {
		result["user"] = user
	}
	if chat := e.extractChat(inputMap, updateType); chat != nil {
		result["chat"] = chat
	}

	// Handle different update types
	switch updateType {
	case "callback_query":
		if cbQuery, ok := inputMap["callback_query"].(map[string]any); ok {
			if data, ok := cbQuery["data"].(string); ok {
				result["callback_data"] = data
			}
			if cbID, ok := cbQuery["id"].(string); ok {
				result["callback_query_id"] = cbID
			}
		}

	case "inline_query":
		if inlineQuery, ok := inputMap["inline_query"].(map[string]any); ok {
			if query, ok := inlineQuery["query"].(string); ok {
				result["query"] = query
			}
			if queryID, ok := inlineQuery["id"].(string); ok {
				result["inline_query_id"] = queryID
			}
		}

	default:
		// Message-like updates
		if messageData != nil {
			// Determine message type
			messageType := e.determineMessageType(messageData)
			result["message_type"] = messageType

			// Extract text
			text := e.extractText(messageData)
			if text != "" {
				result["text"] = text
			}

			// Extract command if enabled
			if extractCommands && text != "" {
				command, args := e.extractCommand(text)
				if command != "" {
					result["command"] = command
					result["command_args"] = args
				}
			}

			// Extract files if enabled
			if extractFiles {
				files := e.extractFiles(messageData)
				if len(files) > 0 {
					result["files"] = files
				}
			}

			// Extract entities if enabled
			if extractEntities {
				entities := e.extractEntities(messageData)
				if len(entities) > 0 {
					result["entities"] = entities
				}
			}

			// Extract reply info
			if replyTo, ok := messageData["reply_to_message"].(map[string]any); ok {
				if msgID, ok := replyTo["message_id"].(float64); ok {
					result["reply_to_message_id"] = int(msgID)
				}
			}

			// Extract message_id
			if msgID, ok := messageData["message_id"].(float64); ok {
				result["message_id"] = int(msgID)
			}
		}
	}

	result["duration_ms"] = time.Since(startTime).Milliseconds()
	return result, nil
}

// determineUpdateType identifies the update type from Telegram webhook payload.
func (e *TelegramParseExecutor) determineUpdateType(input map[string]any) (string, map[string]any) {
	// Check for pre-parsed update_type (from webhook handler)
	if updateType, ok := input["update_type"].(string); ok {
		// Return the message data based on type
		switch updateType {
		case "message":
			if msg, ok := input["message"].(map[string]any); ok {
				return updateType, msg
			}
		case "edited_message":
			if msg, ok := input["edited_message"].(map[string]any); ok {
				return updateType, msg
			}
		case "channel_post":
			if msg, ok := input["channel_post"].(map[string]any); ok {
				return updateType, msg
			}
		case "edited_channel_post":
			if msg, ok := input["edited_channel_post"].(map[string]any); ok {
				return updateType, msg
			}
		case "callback_query":
			return updateType, nil
		case "inline_query":
			return updateType, nil
		}
		return updateType, nil
	}

	// Check each possible field
	if _, ok := input["callback_query"].(map[string]any); ok {
		return "callback_query", nil
	}
	if _, ok := input["inline_query"].(map[string]any); ok {
		return "inline_query", nil
	}
	if msg, ok := input["edited_message"].(map[string]any); ok {
		return "edited_message", msg
	}
	if msg, ok := input["channel_post"].(map[string]any); ok {
		return "channel_post", msg
	}
	if msg, ok := input["edited_channel_post"].(map[string]any); ok {
		return "edited_channel_post", msg
	}
	if msg, ok := input["message"].(map[string]any); ok {
		return "message", msg
	}

	return "unknown", nil
}

// determineMessageType identifies the type of message content.
func (e *TelegramParseExecutor) determineMessageType(msg map[string]any) string {
	// Check for media types first
	if _, ok := msg["photo"].([]any); ok {
		return "photo"
	}
	if _, ok := msg["document"].(map[string]any); ok {
		return "document"
	}
	if _, ok := msg["audio"].(map[string]any); ok {
		return "audio"
	}
	if _, ok := msg["video"].(map[string]any); ok {
		return "video"
	}
	if _, ok := msg["voice"].(map[string]any); ok {
		return "voice"
	}
	if _, ok := msg["video_note"].(map[string]any); ok {
		return "video_note"
	}
	if _, ok := msg["sticker"].(map[string]any); ok {
		return "sticker"
	}
	if _, ok := msg["animation"].(map[string]any); ok {
		return "animation"
	}
	if _, ok := msg["location"].(map[string]any); ok {
		return "location"
	}
	if _, ok := msg["contact"].(map[string]any); ok {
		return "contact"
	}
	if _, ok := msg["poll"].(map[string]any); ok {
		return "poll"
	}
	if _, ok := msg["text"].(string); ok {
		return "text"
	}

	return "unknown"
}

// extractText gets text from message or caption.
func (e *TelegramParseExecutor) extractText(msg map[string]any) string {
	if text, ok := msg["text"].(string); ok {
		return text
	}
	if caption, ok := msg["caption"].(string); ok {
		return caption
	}
	return ""
}

// extractCommand parses bot command from text.
func (e *TelegramParseExecutor) extractCommand(text string) (string, []string) {
	if !strings.HasPrefix(text, "/") {
		return "", nil
	}

	parts := strings.Fields(text)
	if len(parts) == 0 {
		return "", nil
	}

	// Handle /command@botname format
	command := parts[0]
	if atIndex := strings.Index(command, "@"); atIndex > 0 {
		command = command[:atIndex]
	}

	args := []string{}
	if len(parts) > 1 {
		args = parts[1:]
	}

	return command, args
}

// extractFiles extracts all file info from message.
func (e *TelegramParseExecutor) extractFiles(msg map[string]any) []map[string]any {
	var files []map[string]any

	// Photo (array of sizes)
	if photos, ok := msg["photo"].([]any); ok && len(photos) > 0 {
		// Get largest photo (last in array)
		if photo, ok := photos[len(photos)-1].(map[string]any); ok {
			files = append(files, e.extractFileInfo("photo", photo))
		}
	}

	// Document
	if doc, ok := msg["document"].(map[string]any); ok {
		files = append(files, e.extractFileInfo("document", doc))
	}

	// Audio
	if audio, ok := msg["audio"].(map[string]any); ok {
		files = append(files, e.extractFileInfo("audio", audio))
	}

	// Video
	if video, ok := msg["video"].(map[string]any); ok {
		files = append(files, e.extractFileInfo("video", video))
	}

	// Voice
	if voice, ok := msg["voice"].(map[string]any); ok {
		files = append(files, e.extractFileInfo("voice", voice))
	}

	// Video note
	if videoNote, ok := msg["video_note"].(map[string]any); ok {
		files = append(files, e.extractFileInfo("video_note", videoNote))
	}

	// Sticker
	if sticker, ok := msg["sticker"].(map[string]any); ok {
		files = append(files, e.extractFileInfo("sticker", sticker))
	}

	// Animation
	if animation, ok := msg["animation"].(map[string]any); ok {
		files = append(files, e.extractFileInfo("animation", animation))
	}

	return files
}

// extractFileInfo extracts common file info from a file object.
func (e *TelegramParseExecutor) extractFileInfo(fileType string, file map[string]any) map[string]any {
	info := map[string]any{
		"type": fileType,
	}

	if fileID, ok := file["file_id"].(string); ok {
		info["file_id"] = fileID
	}
	if fileUniqueID, ok := file["file_unique_id"].(string); ok {
		info["file_unique_id"] = fileUniqueID
	}
	if fileSize, ok := file["file_size"].(float64); ok {
		info["file_size"] = int(fileSize)
	}
	if fileName, ok := file["file_name"].(string); ok {
		info["file_name"] = fileName
	}
	if mimeType, ok := file["mime_type"].(string); ok {
		info["mime_type"] = mimeType
	}
	if width, ok := file["width"].(float64); ok {
		info["width"] = int(width)
	}
	if height, ok := file["height"].(float64); ok {
		info["height"] = int(height)
	}
	if duration, ok := file["duration"].(float64); ok {
		info["duration"] = int(duration)
	}

	return info
}

// extractUser extracts user info from update.
func (e *TelegramParseExecutor) extractUser(input map[string]any, updateType string) map[string]any {
	var from map[string]any

	switch updateType {
	case "callback_query":
		if cbQuery, ok := input["callback_query"].(map[string]any); ok {
			from, _ = cbQuery["from"].(map[string]any)
		}
	case "inline_query":
		if inlineQuery, ok := input["inline_query"].(map[string]any); ok {
			from, _ = inlineQuery["from"].(map[string]any)
		}
	default:
		// For message-like updates
		for _, key := range []string{"message", "edited_message", "channel_post", "edited_channel_post"} {
			if msg, ok := input[key].(map[string]any); ok {
				from, _ = msg["from"].(map[string]any)
				break
			}
		}
	}

	if from == nil {
		return nil
	}

	user := map[string]any{}
	if id, ok := from["id"].(float64); ok {
		user["id"] = int64(id)
	}
	if username, ok := from["username"].(string); ok {
		user["username"] = username
	}
	if firstName, ok := from["first_name"].(string); ok {
		user["first_name"] = firstName
	}
	if lastName, ok := from["last_name"].(string); ok {
		user["last_name"] = lastName
	}
	if langCode, ok := from["language_code"].(string); ok {
		user["language_code"] = langCode
	}
	if isBot, ok := from["is_bot"].(bool); ok {
		user["is_bot"] = isBot
	}

	return user
}

// extractChat extracts chat info from update.
func (e *TelegramParseExecutor) extractChat(input map[string]any, updateType string) map[string]any {
	var chatData map[string]any

	switch updateType {
	case "callback_query":
		if cbQuery, ok := input["callback_query"].(map[string]any); ok {
			if msg, ok := cbQuery["message"].(map[string]any); ok {
				chatData, _ = msg["chat"].(map[string]any)
			}
		}
	default:
		for _, key := range []string{"message", "edited_message", "channel_post", "edited_channel_post"} {
			if msg, ok := input[key].(map[string]any); ok {
				chatData, _ = msg["chat"].(map[string]any)
				break
			}
		}
	}

	if chatData == nil {
		return nil
	}

	chat := map[string]any{}
	if id, ok := chatData["id"].(float64); ok {
		chat["id"] = int64(id)
	}
	if chatType, ok := chatData["type"].(string); ok {
		chat["type"] = chatType
	}
	if title, ok := chatData["title"].(string); ok {
		chat["title"] = title
	}
	if username, ok := chatData["username"].(string); ok {
		chat["username"] = username
	}

	return chat
}

// extractEntities extracts URLs, emails, and mentions from message entities.
func (e *TelegramParseExecutor) extractEntities(msg map[string]any) map[string]any {
	entities := map[string]any{
		"urls":     []string{},
		"emails":   []string{},
		"mentions": []string{},
	}

	text := e.extractText(msg)
	if text == "" {
		return entities
	}

	// Get entities array from message
	msgEntities, ok := msg["entities"].([]any)
	if !ok {
		msgEntities, _ = msg["caption_entities"].([]any)
	}

	var urls, emails, mentions []string

	for _, entity := range msgEntities {
		entityMap, ok := entity.(map[string]any)
		if !ok {
			continue
		}

		entityType, _ := entityMap["type"].(string)
		offset := int(entityMap["offset"].(float64))
		length := int(entityMap["length"].(float64))

		if offset+length > len(text) {
			continue
		}

		value := text[offset : offset+length]

		switch entityType {
		case "url":
			urls = append(urls, value)
		case "email":
			emails = append(emails, value)
		case "mention":
			mentions = append(mentions, value)
		case "text_mention":
			// User mention without username
			mentions = append(mentions, value)
		}
	}

	// Also extract URLs using regex for cases without entities
	if len(urls) == 0 {
		urlRegex := regexp.MustCompile(`https?://[^\s]+`)
		urls = urlRegex.FindAllString(text, -1)
	}

	entities["urls"] = urls
	entities["emails"] = emails
	entities["mentions"] = mentions

	return entities
}

// Validate validates the Telegram parse executor configuration.
func (e *TelegramParseExecutor) Validate(config map[string]any) error {
	// All config options are optional with defaults
	return nil
}
