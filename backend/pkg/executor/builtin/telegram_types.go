package builtin

// telegramAPIResponse represents the raw API response structure from Telegram Bot API.
type telegramAPIResponse struct {
	OK          bool                    `json:"ok"`
	Result      *telegramMessage        `json:"result,omitempty"`
	Description string                  `json:"description,omitempty"`
	ErrorCode   int                     `json:"error_code,omitempty"`
	Parameters  *telegramResponseParams `json:"parameters,omitempty"`
}

// telegramMessage represents a Telegram message structure.
type telegramMessage struct {
	MessageID int `json:"message_id"`
	Date      int `json:"date"`
	Chat      struct {
		ID int64 `json:"id"`
	} `json:"chat"`
	Text     string         `json:"text,omitempty"`
	Caption  string         `json:"caption,omitempty"`
	Photo    []telegramFile `json:"photo,omitempty"`
	Document *telegramFile  `json:"document,omitempty"`
	Audio    *telegramFile  `json:"audio,omitempty"`
	Video    *telegramFile  `json:"video,omitempty"`
}

// telegramFile represents file information in Telegram API.
type telegramFile struct {
	FileID       string `json:"file_id"`
	FileUniqueID string `json:"file_unique_id"`
	FileSize     int    `json:"file_size,omitempty"`
	FileName     string `json:"file_name,omitempty"`
}

// telegramResponseParams contains additional response parameters.
type telegramResponseParams struct {
	RetryAfter int `json:"retry_after,omitempty"` // For rate limiting (429)
}

// TelegramRequest represents a processed request ready for API call.
type TelegramRequest struct {
	BotToken string
	ChatID   string

	MessageType string
	Text        string
	ParseMode   string

	DisableWebPagePreview bool
	DisableNotification   bool
	ProtectContent        bool

	ReplyToMessageID int
	MessageThreadID  int

	// Media fields
	FileSource string
	FileData   string
	FileName   string

	Timeout int // Timeout in seconds
}

// TelegramResponse represents the output from Telegram executor.
type TelegramResponse struct {
	Success   bool  `json:"success"`
	MessageID int   `json:"message_id,omitempty"`
	ChatID    int64 `json:"chat_id,omitempty"`
	Date      int   `json:"date,omitempty"`

	MessageType string `json:"message_type"`
	Text        string `json:"text,omitempty"`
	Caption     string `json:"caption,omitempty"`

	// File info (for media messages)
	FileID       string `json:"file_id,omitempty"`
	FileUniqueID string `json:"file_unique_id,omitempty"`
	FileSize     int    `json:"file_size,omitempty"`

	// Error information
	Error     string `json:"error,omitempty"`
	ErrorCode int    `json:"error_code,omitempty"`

	// Request metadata
	DurationMS int64 `json:"duration_ms"`
}
