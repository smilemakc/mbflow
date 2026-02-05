package rest

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/smilemakc/mbflow/internal/application/systemkey"
	"github.com/smilemakc/mbflow/internal/infrastructure/logger"
	"github.com/smilemakc/mbflow/internal/infrastructure/storage"
	"github.com/smilemakc/mbflow/pkg/models"
)

const (
	ContextKeySystemKeyID  = "system_key_id"
	ContextKeyServiceName  = "service_name"
	ContextKeyImpersonated = "impersonated"
)

type SystemAuthMiddleware struct {
	systemKeyService *systemkey.Service
	userRepo         *storage.UserRepository
	systemUserID     string
	logger           *logger.Logger
}

func NewSystemAuthMiddleware(service *systemkey.Service, userRepo *storage.UserRepository, systemUserID string, log *logger.Logger) *SystemAuthMiddleware {
	return &SystemAuthMiddleware{
		systemKeyService: service,
		userRepo:         userRepo,
		systemUserID:     systemUserID,
		logger:           log,
	}
}

func (m *SystemAuthMiddleware) RequireSystemAccess() gin.HandlerFunc {
	return func(c *gin.Context) {
		token, err := m.extractSystemKey(c)
		if err != nil {
			respondError(c, http.StatusUnauthorized, "system key required")
			c.Abort()
			return
		}

		if !strings.HasPrefix(token, models.SystemKeyPrefix) {
			respondError(c, http.StatusUnauthorized, "system key required")
			c.Abort()
			return
		}

		key, err := m.systemKeyService.ValidateKey(c.Request.Context(), token)
		if err != nil {
			if errors.Is(err, models.ErrSystemKeyRevoked) {
				respondError(c, http.StatusUnauthorized, "system key has been revoked")
			} else if errors.Is(err, models.ErrSystemKeyExpired) {
				respondError(c, http.StatusUnauthorized, "system key has expired")
			} else {
				respondError(c, http.StatusUnauthorized, "invalid system key")
			}
			c.Abort()
			return
		}

		c.Set(ContextKeyAuthMethod, "system_key")
		c.Set(ContextKeySystemKeyID, key.ID)
		c.Set(ContextKeyServiceName, key.ServiceName)
		c.Set(ContextKeyIsAdmin, true)

		c.Next()
	}
}

func (m *SystemAuthMiddleware) HandleImpersonation() gin.HandlerFunc {
	return func(c *gin.Context) {
		onBehalfOf := c.GetHeader("X-On-Behalf-Of")

		if onBehalfOf != "" {
			userUUID, err := uuid.Parse(onBehalfOf)
			if err != nil {
				respondError(c, http.StatusBadRequest, "invalid user ID format")
				c.Abort()
				return
			}

			user, err := m.userRepo.FindByID(c.Request.Context(), userUUID)
			if err != nil {
				m.logger.Error("Failed to check user existence for impersonation", "error", err, "user_id", userUUID)
				respondError(c, http.StatusInternalServerError, "failed to validate user")
				c.Abort()
				return
			}

			if user == nil {
				respondError(c, http.StatusUnprocessableEntity, "user not found for impersonation")
				c.Abort()
				return
			}

			c.Set(ContextKeyUserID, userUUID.String())
			c.Set(ContextKeyImpersonated, true)
		} else {
			c.Set(ContextKeyUserID, m.systemUserID)
			c.Set(ContextKeyImpersonated, false)
		}

		c.Next()
	}
}

func (m *SystemAuthMiddleware) extractSystemKey(c *gin.Context) (string, error) {
	if systemKey := c.GetHeader("X-System-Key"); systemKey != "" {
		return systemKey, nil
	}

	authHeader := c.GetHeader("Authorization")
	if authHeader != "" {
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) == 2 && strings.EqualFold(parts[0], "Bearer") {
			if strings.HasPrefix(parts[1], models.SystemKeyPrefix) {
				return parts[1], nil
			}
		}
	}

	return "", errors.New("no system key provided")
}
