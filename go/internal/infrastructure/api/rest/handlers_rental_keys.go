package rest

import (
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/smilemakc/mbflow/go/internal/application/rentalkey"
	"github.com/smilemakc/mbflow/go/internal/infrastructure/logger"
	"github.com/smilemakc/mbflow/go/pkg/models"
)

// RentalKeyHandlers handles user-facing rental key HTTP requests.
// Users can only view their own rental keys and usage, never the actual API key value.
type RentalKeyHandlers struct {
	provider *rentalkey.Provider
	logger   *logger.Logger
}

// NewRentalKeyHandlers creates a new RentalKeyHandlers instance
func NewRentalKeyHandlers(provider *rentalkey.Provider, log *logger.Logger) *RentalKeyHandlers {
	return &RentalKeyHandlers{
		provider: provider,
		logger:   log,
	}
}

// ============================================================================
// Response types
// ============================================================================

// RentalKeyResponse represents a rental key in API response (without API key value)
type RentalKeyResponse struct {
	ID          string         `json:"id"`
	Name        string         `json:"name"`
	Description string         `json:"description,omitempty"`
	Status      string         `json:"status"`
	Provider    string         `json:"provider"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	LastUsedAt  *time.Time     `json:"last_used_at,omitempty"`
	Metadata    map[string]any `json:"metadata,omitempty"`

	// Usage limits
	DailyRequestLimit *int   `json:"daily_request_limit,omitempty"`
	MonthlyTokenLimit *int64 `json:"monthly_token_limit,omitempty"`

	// Current usage
	RequestsToday   int   `json:"requests_today"`
	TokensThisMonth int64 `json:"tokens_this_month"`

	// Aggregated statistics
	TotalRequests int64                   `json:"total_requests"`
	TotalUsage    MultimodalUsageResponse `json:"total_usage"`
	TotalCost     float64                 `json:"total_cost"`
}

// MultimodalUsageResponse represents multimodal token usage
type MultimodalUsageResponse struct {
	// Text
	PromptTokens     int64 `json:"prompt_tokens"`
	CompletionTokens int64 `json:"completion_tokens"`

	// Image
	ImageInputTokens  int64 `json:"image_input_tokens"`
	ImageOutputTokens int64 `json:"image_output_tokens"`

	// Audio
	AudioInputTokens  int64 `json:"audio_input_tokens"`
	AudioOutputTokens int64 `json:"audio_output_tokens"`

	// Video
	VideoInputTokens  int64 `json:"video_input_tokens"`
	VideoOutputTokens int64 `json:"video_output_tokens"`

	// Total (computed)
	Total int64 `json:"total"`
}

// UsageRecordResponse represents a single usage record
type UsageRecordResponse struct {
	ID             string                  `json:"id"`
	Model          string                  `json:"model"`
	Usage          MultimodalUsageResponse `json:"usage"`
	EstimatedCost  float64                 `json:"estimated_cost"`
	ExecutionID    string                  `json:"execution_id,omitempty"`
	WorkflowID     string                  `json:"workflow_id,omitempty"`
	NodeID         string                  `json:"node_id,omitempty"`
	Status         string                  `json:"status"`
	ErrorMessage   string                  `json:"error_message,omitempty"`
	ResponseTimeMs int                     `json:"response_time_ms,omitempty"`
	CreatedAt      time.Time               `json:"created_at"`
}

// ============================================================================
// User Handlers
// ============================================================================

// ListRentalKeys returns all rental keys for the current user
// GET /api/v1/rental-keys
func (h *RentalKeyHandlers) ListRentalKeys(c *gin.Context) {
	userID, ok := GetUserID(c)
	if !ok {
		respondError(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	providerFilter := c.Query("provider")

	var keys []*models.RentalKeyResource
	var err error

	if providerFilter != "" {
		keys, err = h.provider.GetKeysByProvider(c.Request.Context(), userID, models.LLMProviderType(providerFilter))
	} else {
		keys, err = h.provider.GetKeysByOwner(c.Request.Context(), userID)
	}

	if err != nil {
		h.logger.Error("Failed to list rental keys", "error", err, "user_id", userID)
		respondError(c, http.StatusInternalServerError, "failed to list rental keys")
		return
	}

	response := make([]RentalKeyResponse, len(keys))
	for i, key := range keys {
		response[i] = h.toResponse(key)
	}

	c.JSON(http.StatusOK, gin.H{"rental_keys": response})
}

// GetRentalKey returns a specific rental key by ID (without API key value)
// GET /api/v1/rental-keys/:id
func (h *RentalKeyHandlers) GetRentalKey(c *gin.Context) {
	userID, ok := GetUserID(c)
	if !ok {
		respondError(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	keyID, ok := getParam(c, "id")
	if !ok {
		return
	}

	key, err := h.provider.GetKey(c.Request.Context(), keyID)
	if err != nil {
		if errors.Is(err, models.ErrRentalKeyNotFound) || errors.Is(err, models.ErrResourceNotFound) {
			respondError(c, http.StatusNotFound, "rental key not found")
			return
		}
		h.logger.Error("Failed to get rental key", "error", err, "key_id", keyID)
		respondError(c, http.StatusInternalServerError, "failed to get rental key")
		return
	}

	if key.OwnerID != userID {
		respondError(c, http.StatusForbidden, "access denied")
		return
	}

	c.JSON(http.StatusOK, h.toResponse(key))
}

// GetRentalKeyUsage returns usage history for a rental key
// GET /api/v1/rental-keys/:id/usage
func (h *RentalKeyHandlers) GetRentalKeyUsage(c *gin.Context) {
	userID, ok := GetUserID(c)
	if !ok {
		respondError(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	keyID, ok := getParam(c, "id")
	if !ok {
		return
	}

	// Verify ownership
	key, err := h.provider.GetKey(c.Request.Context(), keyID)
	if err != nil {
		if errors.Is(err, models.ErrRentalKeyNotFound) || errors.Is(err, models.ErrResourceNotFound) {
			respondError(c, http.StatusNotFound, "rental key not found")
			return
		}
		h.logger.Error("Failed to get rental key", "error", err, "key_id", keyID)
		respondError(c, http.StatusInternalServerError, "failed to get rental key")
		return
	}

	if key.OwnerID != userID {
		respondError(c, http.StatusForbidden, "access denied")
		return
	}

	limit := getQueryInt(c, "limit", 50)
	offset := getQueryInt(c, "offset", 0)

	// Cap limit to prevent abuse
	if limit > 100 {
		limit = 100
	}

	usage, err := h.provider.GetUsageHistory(c.Request.Context(), keyID, limit, offset)
	if err != nil {
		h.logger.Error("Failed to get usage history", "error", err, "key_id", keyID)
		respondError(c, http.StatusInternalServerError, "failed to get usage history")
		return
	}

	response := make([]UsageRecordResponse, len(usage))
	for i, record := range usage {
		response[i] = h.toUsageRecordResponse(record)
	}

	c.JSON(http.StatusOK, gin.H{
		"usage":  response,
		"limit":  limit,
		"offset": offset,
	})
}

// GetRentalKeyUsageSummary returns aggregated usage statistics
// GET /api/v1/rental-keys/:id/summary
func (h *RentalKeyHandlers) GetRentalKeyUsageSummary(c *gin.Context) {
	userID, ok := GetUserID(c)
	if !ok {
		respondError(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	keyID, ok := getParam(c, "id")
	if !ok {
		return
	}

	// Verify ownership
	key, err := h.provider.GetKey(c.Request.Context(), keyID)
	if err != nil {
		if errors.Is(err, models.ErrRentalKeyNotFound) || errors.Is(err, models.ErrResourceNotFound) {
			respondError(c, http.StatusNotFound, "rental key not found")
			return
		}
		h.logger.Error("Failed to get rental key", "error", err, "key_id", keyID)
		respondError(c, http.StatusInternalServerError, "failed to get rental key")
		return
	}

	if key.OwnerID != userID {
		respondError(c, http.StatusForbidden, "access denied")
		return
	}

	summary, err := h.provider.GetUsageSummary(c.Request.Context(), keyID)
	if err != nil {
		h.logger.Error("Failed to get usage summary", "error", err, "key_id", keyID)
		respondError(c, http.StatusInternalServerError, "failed to get usage summary")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"summary": gin.H{
			"total_requests": summary.TotalRequests,
			"total_cost":     summary.TotalCost,
			"total_usage":    h.toMultimodalUsageResponse(summary.TotalUsage),
		},
	})
}

// ============================================================================
// Helper methods
// ============================================================================

func (h *RentalKeyHandlers) toResponse(key *models.RentalKeyResource) RentalKeyResponse {
	return RentalKeyResponse{
		ID:                key.ID,
		Name:              key.Name,
		Description:       key.Description,
		Status:            string(key.Status),
		Provider:          string(key.Provider),
		CreatedAt:         key.CreatedAt,
		UpdatedAt:         key.UpdatedAt,
		LastUsedAt:        key.LastUsedAt,
		Metadata:          key.Metadata,
		DailyRequestLimit: key.DailyRequestLimit,
		MonthlyTokenLimit: key.MonthlyTokenLimit,
		RequestsToday:     key.RequestsToday,
		TokensThisMonth:   key.TokensThisMonth,
		TotalRequests:     key.TotalRequests,
		TotalUsage:        h.toMultimodalUsageResponse(&key.TotalUsage),
		TotalCost:         key.TotalCost,
	}
}

func (h *RentalKeyHandlers) toMultimodalUsageResponse(usage *models.MultimodalUsage) MultimodalUsageResponse {
	if usage == nil {
		return MultimodalUsageResponse{}
	}
	return MultimodalUsageResponse{
		PromptTokens:      usage.PromptTokens,
		CompletionTokens:  usage.CompletionTokens,
		ImageInputTokens:  usage.ImageInputTokens,
		ImageOutputTokens: usage.ImageOutputTokens,
		AudioInputTokens:  usage.AudioInputTokens,
		AudioOutputTokens: usage.AudioOutputTokens,
		VideoInputTokens:  usage.VideoInputTokens,
		VideoOutputTokens: usage.VideoOutputTokens,
		Total:             usage.TotalTokens(),
	}
}

func (h *RentalKeyHandlers) toUsageRecordResponse(record *models.RentalKeyUsageRecord) UsageRecordResponse {
	return UsageRecordResponse{
		ID:             record.ID,
		Model:          record.Model,
		Usage:          h.toMultimodalUsageResponse(&record.Usage),
		EstimatedCost:  record.EstimatedCost,
		ExecutionID:    record.ExecutionID,
		WorkflowID:     record.WorkflowID,
		NodeID:         record.NodeID,
		Status:         record.Status,
		ErrorMessage:   record.ErrorMessage,
		ResponseTimeMs: record.ResponseTimeMs,
		CreatedAt:      record.CreatedAt,
	}
}
