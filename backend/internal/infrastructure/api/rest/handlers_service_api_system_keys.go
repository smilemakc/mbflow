package rest

import (
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/smilemakc/mbflow/internal/application/systemkey"
	"github.com/smilemakc/mbflow/internal/domain/repository"
	"github.com/smilemakc/mbflow/internal/infrastructure/logger"
	"github.com/smilemakc/mbflow/pkg/models"
)

type ServiceAPISystemKeyHandlers struct {
	service *systemkey.Service
	logger  *logger.Logger
}

func NewServiceAPISystemKeyHandlers(service *systemkey.Service, log *logger.Logger) *ServiceAPISystemKeyHandlers {
	return &ServiceAPISystemKeyHandlers{
		service: service,
		logger:  log,
	}
}

type CreateSystemKeyRequest struct {
	Name          string `json:"name" binding:"required,min=1,max=255"`
	Description   string `json:"description" binding:"max=1000"`
	ServiceName   string `json:"service_name" binding:"required,min=1,max=100"`
	ExpiresInDays *int   `json:"expires_in_days"`
}

type CreateSystemKeyResponse struct {
	ID          string     `json:"id"`
	Name        string     `json:"name"`
	Description string     `json:"description,omitempty"`
	ServiceName string     `json:"service_name"`
	Key         string     `json:"key"`
	KeyPrefix   string     `json:"key_prefix"`
	Status      string     `json:"status"`
	ExpiresAt   *time.Time `json:"expires_at,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	CreatedBy   string     `json:"created_by"`
	Warning     string     `json:"warning"`
}

type SystemKeyResponse struct {
	ID          string     `json:"id"`
	Name        string     `json:"name"`
	Description string     `json:"description,omitempty"`
	ServiceName string     `json:"service_name"`
	KeyPrefix   string     `json:"key_prefix"`
	Status      string     `json:"status"`
	LastUsedAt  *time.Time `json:"last_used_at,omitempty"`
	UsageCount  int64      `json:"usage_count"`
	ExpiresAt   *time.Time `json:"expires_at,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	CreatedBy   string     `json:"created_by"`
}

func (h *ServiceAPISystemKeyHandlers) CreateSystemKey(c *gin.Context) {
	adminID, ok := GetUserIDAsUUID(c)
	if !ok {
		respondError(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req CreateSystemKeyRequest
	if err := bindJSON(c, &req); err != nil {
		return
	}

	result, err := h.service.CreateKey(c.Request.Context(), req.Name, req.Description, req.ServiceName, adminID, req.ExpiresInDays)
	if err != nil {
		if errors.Is(err, models.ErrSystemKeyLimitReached) {
			respondError(c, http.StatusConflict, "system key limit reached")
			return
		}
		h.logger.Error("Failed to create system key",
			"error", err,
			"admin_id", adminID,
			"service_name", req.ServiceName,
		)
		respondError(c, http.StatusInternalServerError, "failed to create system key")
		return
	}

	h.logger.Info("System key created",
		"key_id", result.Key.ID,
		"admin_id", adminID,
		"service_name", req.ServiceName,
		"name", req.Name,
	)

	response := CreateSystemKeyResponse{
		ID:          result.Key.ID,
		Name:        result.Key.Name,
		Description: result.Key.Description,
		ServiceName: result.Key.ServiceName,
		Key:         result.PlainKey,
		KeyPrefix:   result.Key.KeyPrefix,
		Status:      result.Key.Status,
		ExpiresAt:   result.Key.ExpiresAt,
		CreatedAt:   result.Key.CreatedAt,
		CreatedBy:   result.Key.CreatedBy,
		Warning:     "Save this key securely - it will not be shown again!",
	}

	respondJSON(c, http.StatusCreated, response)
}

func (h *ServiceAPISystemKeyHandlers) ListSystemKeys(c *gin.Context) {
	limit := getQueryInt(c, "limit", 50)
	offset := getQueryInt(c, "offset", 0)

	if limit > 100 {
		limit = 100
	}

	filter := repository.SystemKeyFilter{
		Limit:  limit,
		Offset: offset,
	}

	if serviceName := c.Query("service_name"); serviceName != "" {
		filter.ServiceName = &serviceName
	}
	if status := c.Query("status"); status != "" {
		filter.Status = &status
	}

	keys, total, err := h.service.ListAll(c.Request.Context(), filter)
	if err != nil {
		h.logger.Error("Failed to list system keys", "error", err)
		respondError(c, http.StatusInternalServerError, "failed to list system keys")
		return
	}

	response := make([]SystemKeyResponse, len(keys))
	for i, key := range keys {
		response[i] = toSystemKeyResponse(key)
	}

	respondList(c, http.StatusOK, response, int(total), limit, offset)
}

func (h *ServiceAPISystemKeyHandlers) GetSystemKey(c *gin.Context) {
	keyID, ok := getParam(c, "id")
	if !ok {
		return
	}

	keyUUID, err := uuid.Parse(keyID)
	if err != nil {
		respondError(c, http.StatusBadRequest, "invalid system key ID")
		return
	}

	key, err := h.service.GetByID(c.Request.Context(), keyUUID)
	if err != nil {
		if errors.Is(err, models.ErrSystemKeyNotFound) {
			respondError(c, http.StatusNotFound, "system key not found")
			return
		}
		h.logger.Error("Failed to get system key", "error", err, "key_id", keyID)
		respondError(c, http.StatusInternalServerError, "failed to get system key")
		return
	}

	respondJSON(c, http.StatusOK, toSystemKeyResponse(key))
}

func (h *ServiceAPISystemKeyHandlers) DeleteSystemKey(c *gin.Context) {
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
		respondError(c, http.StatusBadRequest, "invalid system key ID")
		return
	}

	if err := h.service.DeleteKey(c.Request.Context(), keyUUID); err != nil {
		if errors.Is(err, models.ErrSystemKeyNotFound) {
			respondError(c, http.StatusNotFound, "system key not found")
			return
		}
		h.logger.Error("Failed to delete system key", "error", err, "key_id", keyID)
		respondError(c, http.StatusInternalServerError, "failed to delete system key")
		return
	}

	h.logger.Info("System key deleted",
		"key_id", keyID,
		"admin_id", adminID,
	)

	respondJSON(c, http.StatusOK, gin.H{"message": "system key deleted successfully"})
}

func (h *ServiceAPISystemKeyHandlers) RevokeSystemKey(c *gin.Context) {
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
		respondError(c, http.StatusBadRequest, "invalid system key ID")
		return
	}

	if err := h.service.RevokeKey(c.Request.Context(), keyUUID); err != nil {
		if errors.Is(err, models.ErrSystemKeyNotFound) {
			respondError(c, http.StatusNotFound, "system key not found")
			return
		}
		h.logger.Error("Failed to revoke system key", "error", err, "key_id", keyID)
		respondError(c, http.StatusInternalServerError, "failed to revoke system key")
		return
	}

	h.logger.Info("System key revoked",
		"key_id", keyID,
		"admin_id", adminID,
	)

	respondJSON(c, http.StatusOK, gin.H{"message": "system key revoked successfully"})
}

func toSystemKeyResponse(key *models.SystemKey) SystemKeyResponse {
	return SystemKeyResponse{
		ID:          key.ID,
		Name:        key.Name,
		Description: key.Description,
		ServiceName: key.ServiceName,
		KeyPrefix:   key.KeyPrefix,
		Status:      key.Status,
		LastUsedAt:  key.LastUsedAt,
		UsageCount:  key.UsageCount,
		ExpiresAt:   key.ExpiresAt,
		CreatedAt:   key.CreatedAt,
		CreatedBy:   key.CreatedBy,
	}
}
