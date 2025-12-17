package rest

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/smilemakc/mbflow/internal/application/rentalkey"
	"github.com/smilemakc/mbflow/internal/domain/repository"
	"github.com/smilemakc/mbflow/internal/infrastructure/logger"
	"github.com/smilemakc/mbflow/pkg/models"
)

// RentalKeyAdminHandlers handles admin rental key HTTP requests.
// Admins can create, update, and manage rental keys for any user.
type RentalKeyAdminHandlers struct {
	adminService *rentalkey.AdminService
	logger       *logger.Logger
}

// NewRentalKeyAdminHandlers creates a new RentalKeyAdminHandlers instance
func NewRentalKeyAdminHandlers(adminService *rentalkey.AdminService, log *logger.Logger) *RentalKeyAdminHandlers {
	return &RentalKeyAdminHandlers{
		adminService: adminService,
		logger:       log,
	}
}

// ============================================================================
// Request types
// ============================================================================

// CreateRentalKeyRequest represents request to create a rental key
type CreateRentalKeyRequest struct {
	OwnerID           string                 `json:"owner_id" binding:"required,uuid"`
	Name              string                 `json:"name" binding:"required,min=1,max=255"`
	Description       string                 `json:"description" binding:"max=1000"`
	Provider          string                 `json:"provider" binding:"required,oneof=openai anthropic google_ai"`
	APIKey            string                 `json:"api_key" binding:"required"`
	ProviderConfig    map[string]interface{} `json:"provider_config,omitempty"`
	DailyRequestLimit *int                   `json:"daily_request_limit,omitempty"`
	MonthlyTokenLimit *int64                 `json:"monthly_token_limit,omitempty"`
	PricingPlanID     string                 `json:"pricing_plan_id,omitempty"`
	ProvisionerType   string                 `json:"provisioner_type,omitempty"`
}

// UpdateRentalKeyRequest represents request to update a rental key
type UpdateRentalKeyRequest struct {
	Name              *string                `json:"name,omitempty"`
	Description       *string                `json:"description,omitempty"`
	Status            *string                `json:"status,omitempty"`
	DailyRequestLimit *int                   `json:"daily_request_limit,omitempty"`
	MonthlyTokenLimit *int64                 `json:"monthly_token_limit,omitempty"`
	ProviderConfig    map[string]interface{} `json:"provider_config,omitempty"`
}

// RotateKeyRequest represents request to rotate API key
type RotateKeyRequest struct {
	NewAPIKey string `json:"new_api_key" binding:"required"`
}

// AdminRentalKeyResponse includes additional admin fields
type AdminRentalKeyResponse struct {
	RentalKeyResponse
	OwnerID         string `json:"owner_id"`
	PricingPlanID   string `json:"pricing_plan_id,omitempty"`
	CreatedBy       string `json:"created_by,omitempty"`
	ProvisionerType string `json:"provisioner_type"`
}

// ============================================================================
// Admin Handlers
// ============================================================================

// CreateRentalKey creates a new rental key
// POST /api/v1/admin/rental-keys
func (h *RentalKeyAdminHandlers) CreateRentalKey(c *gin.Context) {
	adminID, ok := GetUserID(c)
	if !ok {
		respondError(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req CreateRentalKeyRequest
	if err := bindJSON(c, &req); err != nil {
		return
	}

	// Determine provisioner type
	provisionerType := models.ProvisionerTypeManual
	if req.ProvisionerType != "" {
		provisionerType = models.ProvisionerType(req.ProvisionerType)
	}

	createReq := &rentalkey.CreateKeyRequest{
		OwnerID:           req.OwnerID,
		Name:              req.Name,
		Description:       req.Description,
		Provider:          models.LLMProviderType(req.Provider),
		PlainAPIKey:       req.APIKey,
		ProviderConfig:    req.ProviderConfig,
		DailyRequestLimit: req.DailyRequestLimit,
		MonthlyTokenLimit: req.MonthlyTokenLimit,
		PricingPlanID:     req.PricingPlanID,
		CreatedBy:         adminID,
		ProvisionerType:   provisionerType,
	}

	key, err := h.adminService.CreateKey(c.Request.Context(), createReq)
	if err != nil {
		h.logger.Error("Failed to create rental key",
			"error", err,
			"admin_id", adminID,
			"owner_id", req.OwnerID,
			"provider", req.Provider,
		)
		respondError(c, http.StatusInternalServerError, "failed to create rental key")
		return
	}

	h.logger.Info("Rental key created",
		"key_id", key.ID,
		"admin_id", adminID,
		"owner_id", req.OwnerID,
		"provider", req.Provider,
	)

	c.JSON(http.StatusCreated, h.toAdminResponse(key))
}

// ListAllRentalKeys returns all rental keys with optional filtering
// GET /api/v1/admin/rental-keys
func (h *RentalKeyAdminHandlers) ListAllRentalKeys(c *gin.Context) {
	limit := getQueryInt(c, "limit", 50)
	offset := getQueryInt(c, "offset", 0)

	// Cap limit
	if limit > 100 {
		limit = 100
	}

	filter := repository.RentalKeyFilter{
		Limit:  limit,
		Offset: offset,
	}

	// Optional filters
	if provider := c.Query("provider"); provider != "" {
		p := models.LLMProviderType(provider)
		filter.Provider = &p
	}
	if status := c.Query("status"); status != "" {
		s := models.ResourceStatus(status)
		filter.Status = &s
	}
	if ownerID := c.Query("owner_id"); ownerID != "" {
		filter.OwnerID = &ownerID
	}
	if createdBy := c.Query("created_by"); createdBy != "" {
		filter.CreatedBy = &createdBy
	}

	keys, total, err := h.adminService.ListAllKeys(c.Request.Context(), filter)
	if err != nil {
		h.logger.Error("Failed to list rental keys", "error", err)
		respondError(c, http.StatusInternalServerError, "failed to list rental keys")
		return
	}

	response := make([]AdminRentalKeyResponse, len(keys))
	for i, key := range keys {
		response[i] = h.toAdminResponse(key)
	}

	c.JSON(http.StatusOK, gin.H{
		"rental_keys": response,
		"total":       total,
		"limit":       limit,
		"offset":      offset,
	})
}

// GetRentalKey returns a specific rental key by ID
// GET /api/v1/admin/rental-keys/:id
func (h *RentalKeyAdminHandlers) GetRentalKey(c *gin.Context) {
	keyID, ok := getParam(c, "id")
	if !ok {
		return
	}

	key, err := h.adminService.GetKey(c.Request.Context(), keyID)
	if err != nil {
		if errors.Is(err, models.ErrRentalKeyNotFound) || errors.Is(err, models.ErrResourceNotFound) {
			respondError(c, http.StatusNotFound, "rental key not found")
			return
		}
		h.logger.Error("Failed to get rental key", "error", err, "key_id", keyID)
		respondError(c, http.StatusInternalServerError, "failed to get rental key")
		return
	}

	c.JSON(http.StatusOK, h.toAdminResponse(key))
}

// UpdateRentalKey updates a rental key
// PUT /api/v1/admin/rental-keys/:id
func (h *RentalKeyAdminHandlers) UpdateRentalKey(c *gin.Context) {
	adminID, ok := GetUserID(c)
	if !ok {
		respondError(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	keyID, ok := getParam(c, "id")
	if !ok {
		return
	}

	var req UpdateRentalKeyRequest
	if err := bindJSON(c, &req); err != nil {
		return
	}

	updateReq := &rentalkey.UpdateKeyRequest{
		Name:              req.Name,
		Description:       req.Description,
		DailyRequestLimit: req.DailyRequestLimit,
		MonthlyTokenLimit: req.MonthlyTokenLimit,
		ProviderConfig:    req.ProviderConfig,
	}

	if req.Status != nil {
		status := models.ResourceStatus(*req.Status)
		updateReq.Status = &status
	}

	key, err := h.adminService.UpdateKey(c.Request.Context(), keyID, updateReq)
	if err != nil {
		if errors.Is(err, models.ErrRentalKeyNotFound) || errors.Is(err, models.ErrResourceNotFound) {
			respondError(c, http.StatusNotFound, "rental key not found")
			return
		}
		h.logger.Error("Failed to update rental key", "error", err, "key_id", keyID)
		respondError(c, http.StatusInternalServerError, "failed to update rental key")
		return
	}

	h.logger.Info("Rental key updated",
		"key_id", keyID,
		"admin_id", adminID,
	)

	c.JSON(http.StatusOK, h.toAdminResponse(key))
}

// RotateRentalKeyAPIKey rotates the API key for a rental key
// POST /api/v1/admin/rental-keys/:id/rotate-key
func (h *RentalKeyAdminHandlers) RotateRentalKeyAPIKey(c *gin.Context) {
	adminID, ok := GetUserID(c)
	if !ok {
		respondError(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	keyID, ok := getParam(c, "id")
	if !ok {
		return
	}

	var req RotateKeyRequest
	if err := bindJSON(c, &req); err != nil {
		return
	}

	err := h.adminService.RotateKey(c.Request.Context(), keyID, req.NewAPIKey)
	if err != nil {
		if errors.Is(err, models.ErrRentalKeyNotFound) || errors.Is(err, models.ErrResourceNotFound) {
			respondError(c, http.StatusNotFound, "rental key not found")
			return
		}
		h.logger.Error("Failed to rotate API key", "error", err, "key_id", keyID)
		respondError(c, http.StatusInternalServerError, "failed to rotate API key")
		return
	}

	h.logger.Info("Rental key API rotated",
		"key_id", keyID,
		"admin_id", adminID,
	)

	c.JSON(http.StatusOK, gin.H{"message": "API key rotated successfully"})
}

// DeleteRentalKey soft-deletes a rental key
// DELETE /api/v1/admin/rental-keys/:id
func (h *RentalKeyAdminHandlers) DeleteRentalKey(c *gin.Context) {
	adminID, ok := GetUserID(c)
	if !ok {
		respondError(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	keyID, ok := getParam(c, "id")
	if !ok {
		return
	}

	err := h.adminService.DeleteKey(c.Request.Context(), keyID)
	if err != nil {
		if errors.Is(err, models.ErrRentalKeyNotFound) || errors.Is(err, models.ErrResourceNotFound) {
			respondError(c, http.StatusNotFound, "rental key not found")
			return
		}
		h.logger.Error("Failed to delete rental key", "error", err, "key_id", keyID)
		respondError(c, http.StatusInternalServerError, "failed to delete rental key")
		return
	}

	h.logger.Info("Rental key deleted",
		"key_id", keyID,
		"admin_id", adminID,
	)

	c.JSON(http.StatusOK, gin.H{"message": "rental key deleted successfully"})
}

// ResetDailyUsage resets daily request counters for all rental keys
// POST /api/v1/admin/rental-keys/reset-daily
func (h *RentalKeyAdminHandlers) ResetDailyUsage(c *gin.Context) {
	adminID, ok := GetUserID(c)
	if !ok {
		respondError(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	err := h.adminService.ResetDailyUsage(c.Request.Context())
	if err != nil {
		h.logger.Error("Failed to reset daily usage", "error", err)
		respondError(c, http.StatusInternalServerError, "failed to reset daily usage")
		return
	}

	h.logger.Info("Daily usage reset",
		"admin_id", adminID,
	)

	c.JSON(http.StatusOK, gin.H{"message": "daily usage reset successfully"})
}

// ResetMonthlyUsage resets monthly token counters for all rental keys
// POST /api/v1/admin/rental-keys/reset-monthly
func (h *RentalKeyAdminHandlers) ResetMonthlyUsage(c *gin.Context) {
	adminID, ok := GetUserID(c)
	if !ok {
		respondError(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	err := h.adminService.ResetMonthlyUsage(c.Request.Context())
	if err != nil {
		h.logger.Error("Failed to reset monthly usage", "error", err)
		respondError(c, http.StatusInternalServerError, "failed to reset monthly usage")
		return
	}

	h.logger.Info("Monthly usage reset",
		"admin_id", adminID,
	)

	c.JSON(http.StatusOK, gin.H{"message": "monthly usage reset successfully"})
}

// ============================================================================
// Helper methods
// ============================================================================

func (h *RentalKeyAdminHandlers) toAdminResponse(key *models.RentalKeyResource) AdminRentalKeyResponse {
	return AdminRentalKeyResponse{
		RentalKeyResponse: RentalKeyResponse{
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
			TotalUsage:        toMultimodalUsageResponse(&key.TotalUsage),
			TotalCost:         key.TotalCost,
		},
		OwnerID:         key.OwnerID,
		PricingPlanID:   key.PricingPlanID,
		CreatedBy:       key.CreatedBy,
		ProvisionerType: string(key.ProvisionerType),
	}
}

func toMultimodalUsageResponse(usage *models.MultimodalUsage) MultimodalUsageResponse {
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
