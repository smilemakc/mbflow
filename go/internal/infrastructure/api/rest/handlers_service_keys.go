package rest

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/smilemakc/mbflow/go/internal/application/servicekey"
	"github.com/smilemakc/mbflow/go/internal/infrastructure/logger"
	"github.com/smilemakc/mbflow/go/pkg/models"
)

// ServiceKeyHandlers handles user service key operations
type ServiceKeyHandlers struct {
	service *servicekey.Service
	logger  *logger.Logger
}

// NewServiceKeyHandlers creates a new ServiceKeyHandlers instance
func NewServiceKeyHandlers(service *servicekey.Service, log *logger.Logger) *ServiceKeyHandlers {
	return &ServiceKeyHandlers{
		service: service,
		logger:  log,
	}
}

// ListMyServiceKeys returns all service keys for the current user
// GET /api/v1/service-keys
func (h *ServiceKeyHandlers) ListMyServiceKeys(c *gin.Context) {
	userID, ok := GetUserIDAsUUID(c)
	if !ok {
		respondError(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	keys, err := h.service.GetByUserID(c.Request.Context(), userID)
	if err != nil {
		h.logger.Error("Failed to list service keys", "error", err, "user_id", userID)
		respondError(c, http.StatusInternalServerError, "failed to list service keys")
		return
	}

	response := make([]ServiceKeyResponse, len(keys))
	for i, key := range keys {
		response[i] = toServiceKeyResponse(key)
	}

	c.JSON(http.StatusOK, gin.H{"service_keys": response})
}

// GetMyServiceKey returns a specific service key by ID (only if owned by current user)
// GET /api/v1/service-keys/:id
func (h *ServiceKeyHandlers) GetMyServiceKey(c *gin.Context) {
	userID, ok := GetUserIDAsUUID(c)
	if !ok {
		respondError(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	keyIDStr, ok := getParam(c, "id")
	if !ok {
		return
	}

	keyID, err := uuid.Parse(keyIDStr)
	if err != nil {
		respondError(c, http.StatusBadRequest, "invalid service key ID")
		return
	}

	key, err := h.service.GetByID(c.Request.Context(), keyID)
	if err != nil {
		if errors.Is(err, models.ErrServiceKeyNotFound) {
			respondError(c, http.StatusNotFound, "service key not found")
			return
		}
		h.logger.Error("Failed to get service key", "error", err, "key_id", keyID)
		respondError(c, http.StatusInternalServerError, "failed to get service key")
		return
	}

	if key.UserID != userID.String() {
		respondError(c, http.StatusForbidden, "access denied")
		return
	}

	c.JSON(http.StatusOK, toServiceKeyResponse(key))
}

// toServiceKeyResponse converts domain model to response DTO
func toServiceKeyResponse(key *models.ServiceKey) ServiceKeyResponse {
	return ServiceKeyResponse{
		ID:          key.ID,
		UserID:      key.UserID,
		Name:        key.Name,
		Description: key.Description,
		KeyPrefix:   key.KeyPrefix,
		Status:      key.Status,
		LastUsedAt:  key.LastUsedAt,
		UsageCount:  key.UsageCount,
		ExpiresAt:   key.ExpiresAt,
		CreatedAt:   key.CreatedAt,
		CreatedBy:   key.CreatedBy,
	}
}
