package rest

import (
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/smilemakc/mbflow/internal/application/servicekey"
	"github.com/smilemakc/mbflow/internal/domain/repository"
	"github.com/smilemakc/mbflow/internal/infrastructure/logger"
	"github.com/smilemakc/mbflow/pkg/models"
)

// ServiceKeyAdminHandlers handles admin service key operations
type ServiceKeyAdminHandlers struct {
	service *servicekey.Service
	logger  *logger.Logger
}

// NewServiceKeyAdminHandlers creates a new ServiceKeyAdminHandlers instance
func NewServiceKeyAdminHandlers(service *servicekey.Service, log *logger.Logger) *ServiceKeyAdminHandlers {
	return &ServiceKeyAdminHandlers{
		service: service,
		logger:  log,
	}
}

// ============================================================================
// Request/Response types
// ============================================================================

// CreateServiceKeyRequest represents request to create a service key
type CreateServiceKeyRequest struct {
	UserID        string `json:"user_id" binding:"required,uuid"`
	Name          string `json:"name" binding:"required,min=1,max=255"`
	Description   string `json:"description" binding:"max=1000"`
	ExpiresInDays *int   `json:"expires_in_days"`
}

// CreateServiceKeyResponse returns created key with plain text (shown only once!)
type CreateServiceKeyResponse struct {
	ID          string     `json:"id"`
	UserID      string     `json:"user_id"`
	Name        string     `json:"name"`
	Description string     `json:"description,omitempty"`
	Key         string     `json:"key"`
	KeyPrefix   string     `json:"key_prefix"`
	Status      string     `json:"status"`
	ExpiresAt   *time.Time `json:"expires_at,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	CreatedBy   string     `json:"created_by"`
	Warning     string     `json:"warning"`
}

// ServiceKeyResponse represents a service key in API response
type ServiceKeyResponse struct {
	ID          string     `json:"id"`
	UserID      string     `json:"user_id"`
	Name        string     `json:"name"`
	Description string     `json:"description,omitempty"`
	KeyPrefix   string     `json:"key_prefix"`
	Status      string     `json:"status"`
	LastUsedAt  *time.Time `json:"last_used_at,omitempty"`
	UsageCount  int64      `json:"usage_count"`
	ExpiresAt   *time.Time `json:"expires_at,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	CreatedBy   string     `json:"created_by"`
}

// ============================================================================
// Admin Handlers
// ============================================================================

// CreateServiceKey creates a new service key
// POST /api/v1/admin/service-keys
func (h *ServiceKeyAdminHandlers) CreateServiceKey(c *gin.Context) {
	adminID, ok := GetUserIDAsUUID(c)
	if !ok {
		respondError(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req CreateServiceKeyRequest
	if err := bindJSON(c, &req); err != nil {
		return
	}

	userID, err := uuid.Parse(req.UserID)
	if err != nil {
		respondError(c, http.StatusBadRequest, "invalid user_id format")
		return
	}

	result, err := h.service.CreateKey(c.Request.Context(), userID, req.Name, req.Description, adminID, req.ExpiresInDays)
	if err != nil {
		if errors.Is(err, models.ErrServiceKeyLimitReached) {
			respondError(c, http.StatusConflict, "service key limit reached for user")
			return
		}
		h.logger.Error("Failed to create service key",
			"error", err,
			"admin_id", adminID,
			"user_id", req.UserID,
		)
		respondError(c, http.StatusInternalServerError, "failed to create service key")
		return
	}

	h.logger.Info("Service key created",
		"key_id", result.Key.ID,
		"admin_id", adminID,
		"user_id", req.UserID,
		"name", req.Name,
	)

	response := CreateServiceKeyResponse{
		ID:          result.Key.ID,
		UserID:      result.Key.UserID,
		Name:        result.Key.Name,
		Description: result.Key.Description,
		Key:         result.PlainKey,
		KeyPrefix:   result.Key.KeyPrefix,
		Status:      string(result.Key.Status),
		ExpiresAt:   result.Key.ExpiresAt,
		CreatedAt:   result.Key.CreatedAt,
		CreatedBy:   result.Key.CreatedBy,
		Warning:     "Save this key securely - it will not be shown again!",
	}

	respondJSON(c, http.StatusCreated, response)
}

// ListServiceKeys returns all service keys with optional filtering
// GET /api/v1/admin/service-keys
func (h *ServiceKeyAdminHandlers) ListServiceKeys(c *gin.Context) {
	limit := getQueryInt(c, "limit", 50)
	offset := getQueryInt(c, "offset", 0)

	if limit > 100 {
		limit = 100
	}

	filter := repository.ServiceKeyFilter{
		Limit:  limit,
		Offset: offset,
	}

	if userIDStr := c.Query("user_id"); userIDStr != "" {
		if userID, err := uuid.Parse(userIDStr); err == nil {
			filter.UserID = &userID
		}
	}
	if status := c.Query("status"); status != "" {
		filter.Status = &status
	}
	if createdByStr := c.Query("created_by"); createdByStr != "" {
		if createdBy, err := uuid.Parse(createdByStr); err == nil {
			filter.CreatedBy = &createdBy
		}
	}

	keys, total, err := h.service.ListAll(c.Request.Context(), filter)
	if err != nil {
		h.logger.Error("Failed to list service keys", "error", err)
		respondError(c, http.StatusInternalServerError, "failed to list service keys")
		return
	}

	response := make([]ServiceKeyResponse, len(keys))
	for i, key := range keys {
		response[i] = h.toResponse(key)
	}

	c.JSON(http.StatusOK, gin.H{
		"service_keys": response,
		"total":        total,
		"limit":        limit,
		"offset":       offset,
	})
}

// GetServiceKey returns a specific service key by ID
// GET /api/v1/admin/service-keys/:id
func (h *ServiceKeyAdminHandlers) GetServiceKey(c *gin.Context) {
	keyID, ok := getParam(c, "id")
	if !ok {
		return
	}

	keyUUID, err := uuid.Parse(keyID)
	if err != nil {
		respondError(c, http.StatusBadRequest, "invalid service key ID")
		return
	}

	key, err := h.service.GetByID(c.Request.Context(), keyUUID)
	if err != nil {
		if errors.Is(err, models.ErrServiceKeyNotFound) {
			respondError(c, http.StatusNotFound, "service key not found")
			return
		}
		h.logger.Error("Failed to get service key", "error", err, "key_id", keyID)
		respondError(c, http.StatusInternalServerError, "failed to get service key")
		return
	}

	respondJSON(c, http.StatusOK, h.toResponse(key))
}

// DeleteServiceKey soft-deletes a service key
// DELETE /api/v1/admin/service-keys/:id
func (h *ServiceKeyAdminHandlers) DeleteServiceKey(c *gin.Context) {
	adminID, ok := GetUserID(c)
	if !ok {
		respondError(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	keyID, ok := getParam(c, "id")
	if !ok {
		return
	}

	keyUUID, err := uuid.Parse(keyID)
	if err != nil {
		respondError(c, http.StatusBadRequest, "invalid service key ID")
		return
	}

	if err := h.service.DeleteKey(c.Request.Context(), keyUUID); err != nil {
		if errors.Is(err, models.ErrServiceKeyNotFound) {
			respondError(c, http.StatusNotFound, "service key not found")
			return
		}
		h.logger.Error("Failed to delete service key", "error", err, "key_id", keyID)
		respondError(c, http.StatusInternalServerError, "failed to delete service key")
		return
	}

	h.logger.Info("Service key deleted",
		"key_id", keyID,
		"admin_id", adminID,
	)

	c.JSON(http.StatusOK, gin.H{"message": "service key deleted successfully"})
}

// RevokeServiceKey revokes a service key
// POST /api/v1/admin/service-keys/:id/revoke
func (h *ServiceKeyAdminHandlers) RevokeServiceKey(c *gin.Context) {
	adminID, ok := GetUserID(c)
	if !ok {
		respondError(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	keyID, ok := getParam(c, "id")
	if !ok {
		return
	}

	keyUUID, err := uuid.Parse(keyID)
	if err != nil {
		respondError(c, http.StatusBadRequest, "invalid service key ID")
		return
	}

	if err := h.service.RevokeKey(c.Request.Context(), keyUUID); err != nil {
		if errors.Is(err, models.ErrServiceKeyNotFound) {
			respondError(c, http.StatusNotFound, "service key not found")
			return
		}
		h.logger.Error("Failed to revoke service key", "error", err, "key_id", keyID)
		respondError(c, http.StatusInternalServerError, "failed to revoke service key")
		return
	}

	h.logger.Info("Service key revoked",
		"key_id", keyID,
		"admin_id", adminID,
	)

	c.JSON(http.StatusOK, gin.H{"message": "service key revoked successfully"})
}

// ============================================================================
// Helper methods
// ============================================================================

func (h *ServiceKeyAdminHandlers) toResponse(key *models.ServiceKey) ServiceKeyResponse {
	return ServiceKeyResponse{
		ID:          key.ID,
		UserID:      key.UserID,
		Name:        key.Name,
		Description: key.Description,
		KeyPrefix:   key.KeyPrefix,
		Status:      string(key.Status),
		LastUsedAt:  key.LastUsedAt,
		UsageCount:  key.UsageCount,
		ExpiresAt:   key.ExpiresAt,
		CreatedAt:   key.CreatedAt,
		CreatedBy:   key.CreatedBy,
	}
}
