package rest

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/smilemakc/mbflow/go/internal/application/auth"
	"github.com/smilemakc/mbflow/go/internal/application/servicekey"
	"github.com/smilemakc/mbflow/go/pkg/models"
	pkgmodels "github.com/smilemakc/mbflow/go/pkg/models"
)

const (
	// Context keys for auth data
	ContextKeyUserID       = "user_id"
	ContextKeyUser         = "user"
	ContextKeyClaims       = "claims"
	ContextKeyToken        = "token"
	ContextKeyIsAdmin      = "is_admin"
	ContextKeyPermissions  = "permissions"
	ContextKeyAuthMethod   = "auth_method"
	ContextKeyServiceKeyID = "service_key_id"
)

// AuthMiddleware provides authentication and authorization middleware
type AuthMiddleware struct {
	providerManager   *auth.ProviderManager
	authService       *auth.Service
	serviceKeyService *servicekey.Service
}

// NewAuthMiddleware creates a new authentication middleware
func NewAuthMiddleware(pm *auth.ProviderManager, authService *auth.Service, serviceKeyService *servicekey.Service) *AuthMiddleware {
	return &AuthMiddleware{
		providerManager:   pm,
		authService:       authService,
		serviceKeyService: serviceKeyService,
	}
}

// RequireAuth middleware that requires valid authentication
func (m *AuthMiddleware) RequireAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		token, err := m.extractToken(c)
		if err != nil {
			respondError(c, http.StatusUnauthorized, "authentication required")
			c.Abort()
			return
		}

		// Check if it's a service key (starts with "sk_")
		if strings.HasPrefix(token, "sk_") && m.serviceKeyService != nil {
			serviceKey, err := m.serviceKeyService.ValidateKey(c.Request.Context(), token)
			if err != nil {
				if errors.Is(err, models.ErrServiceKeyRevoked) {
					respondError(c, http.StatusUnauthorized, "service key has been revoked")
				} else if errors.Is(err, models.ErrServiceKeyExpired) {
					respondError(c, http.StatusUnauthorized, "service key has expired")
				} else {
					respondError(c, http.StatusUnauthorized, "invalid service key")
				}
				c.Abort()
				return
			}

			// Set context values from service key
			c.Set(ContextKeyUserID, serviceKey.UserID)
			c.Set(ContextKeyIsAdmin, false)
			c.Set(ContextKeyAuthMethod, "service_key")
			c.Set(ContextKeyServiceKeyID, serviceKey.ID)

			c.Next()
			return
		}

		// Otherwise validate as JWT token
		claims, err := m.providerManager.ValidateToken(c.Request.Context(), token)
		if err != nil {
			if errors.Is(err, auth.ErrExpiredToken) {
				respondError(c, http.StatusUnauthorized, "token expired")
			} else {
				respondError(c, http.StatusUnauthorized, "invalid token")
			}
			c.Abort()
			return
		}

		// Set context values
		c.Set(ContextKeyUserID, claims.UserID)
		c.Set(ContextKeyClaims, claims)
		c.Set(ContextKeyToken, token)
		c.Set(ContextKeyIsAdmin, claims.IsAdmin)
		c.Set(ContextKeyAuthMethod, "jwt")

		c.Next()
	}
}

// OptionalAuth middleware that allows unauthenticated requests but sets user context if authenticated
func (m *AuthMiddleware) OptionalAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		token, err := m.extractToken(c)
		if err != nil {
			// No token provided, continue without auth
			c.Next()
			return
		}

		claims, err := m.providerManager.ValidateToken(c.Request.Context(), token)
		if err != nil {
			// Invalid token, continue without auth
			c.Next()
			return
		}

		// Set context values
		c.Set(ContextKeyUserID, claims.UserID)
		c.Set(ContextKeyClaims, claims)
		c.Set(ContextKeyToken, token)
		c.Set(ContextKeyIsAdmin, claims.IsAdmin)

		c.Next()
	}
}

// RequireAdmin middleware that requires admin privileges
func (m *AuthMiddleware) RequireAdmin() gin.HandlerFunc {
	return func(c *gin.Context) {
		// First ensure authentication
		token, err := m.extractToken(c)
		if err != nil {
			respondError(c, http.StatusUnauthorized, "authentication required")
			c.Abort()
			return
		}

		claims, err := m.providerManager.ValidateToken(c.Request.Context(), token)
		if err != nil {
			respondError(c, http.StatusUnauthorized, "invalid token")
			c.Abort()
			return
		}

		// Check admin status
		if !claims.IsAdmin {
			respondError(c, http.StatusForbidden, "admin privileges required")
			c.Abort()
			return
		}

		// Set context values
		c.Set(ContextKeyUserID, claims.UserID)
		c.Set(ContextKeyClaims, claims)
		c.Set(ContextKeyToken, token)
		c.Set(ContextKeyIsAdmin, true)

		c.Next()
	}
}

// RequireRole middleware that requires specific roles
func (m *AuthMiddleware) RequireRole(roles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// First ensure authentication
		token, err := m.extractToken(c)
		if err != nil {
			respondError(c, http.StatusUnauthorized, "authentication required")
			c.Abort()
			return
		}

		claims, err := m.providerManager.ValidateToken(c.Request.Context(), token)
		if err != nil {
			respondError(c, http.StatusUnauthorized, "invalid token")
			c.Abort()
			return
		}

		// Admins bypass role check
		if claims.IsAdmin {
			c.Set(ContextKeyUserID, claims.UserID)
			c.Set(ContextKeyClaims, claims)
			c.Set(ContextKeyToken, token)
			c.Set(ContextKeyIsAdmin, true)
			c.Next()
			return
		}

		// Check if user has any of the required roles
		hasRole := false
		for _, requiredRole := range roles {
			for _, userRole := range claims.Roles {
				if strings.EqualFold(userRole, requiredRole) {
					hasRole = true
					break
				}
			}
			if hasRole {
				break
			}
		}

		if !hasRole {
			respondError(c, http.StatusForbidden, "insufficient privileges")
			c.Abort()
			return
		}

		// Set context values
		c.Set(ContextKeyUserID, claims.UserID)
		c.Set(ContextKeyClaims, claims)
		c.Set(ContextKeyToken, token)
		c.Set(ContextKeyIsAdmin, claims.IsAdmin)

		c.Next()
	}
}

// RequirePermission middleware that requires specific permission
func (m *AuthMiddleware) RequirePermission(permission string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// First ensure authentication
		token, err := m.extractToken(c)
		if err != nil {
			respondError(c, http.StatusUnauthorized, "authentication required")
			c.Abort()
			return
		}

		claims, err := m.providerManager.ValidateToken(c.Request.Context(), token)
		if err != nil {
			respondError(c, http.StatusUnauthorized, "invalid token")
			c.Abort()
			return
		}

		// Admins bypass permission check
		if claims.IsAdmin {
			c.Set(ContextKeyUserID, claims.UserID)
			c.Set(ContextKeyClaims, claims)
			c.Set(ContextKeyToken, token)
			c.Set(ContextKeyIsAdmin, true)
			c.Next()
			return
		}

		// Check permission through auth service
		userID, err := uuid.Parse(claims.UserID)
		if err != nil {
			respondError(c, http.StatusUnauthorized, "invalid user ID")
			c.Abort()
			return
		}

		hasPermission, err := m.authService.HasPermission(c.Request.Context(), userID, permission)
		if err != nil || !hasPermission {
			respondError(c, http.StatusForbidden, "permission denied")
			c.Abort()
			return
		}

		// Set context values
		c.Set(ContextKeyUserID, claims.UserID)
		c.Set(ContextKeyClaims, claims)
		c.Set(ContextKeyToken, token)
		c.Set(ContextKeyIsAdmin, claims.IsAdmin)

		c.Next()
	}
}

// extractToken extracts the JWT token from Authorization header, cookie, query param,
// OR service key from X-Service-Key header
func (m *AuthMiddleware) extractToken(c *gin.Context) (string, error) {
	// Check X-Service-Key header first
	if serviceKey := c.GetHeader("X-Service-Key"); serviceKey != "" {
		return serviceKey, nil
	}

	// Try Authorization header
	authHeader := c.GetHeader("Authorization")
	if authHeader != "" {
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) == 2 && strings.EqualFold(parts[0], "Bearer") {
			return parts[1], nil
		}
	}

	// Try cookie
	token, err := c.Cookie("auth_token")
	if err == nil && token != "" {
		return token, nil
	}

	// Try query parameter (for WebSocket connections)
	token = c.Query("token")
	if token != "" {
		return token, nil
	}

	return "", errors.New("no token provided")
}

// GetUserID extracts user ID from gin context
func GetUserID(c *gin.Context) (string, bool) {
	userID, exists := c.Get(ContextKeyUserID)
	if !exists {
		return "", false
	}
	return userID.(string), true
}

// GetUserIDAsUUID extracts user ID from gin context as UUID
func GetUserIDAsUUID(c *gin.Context) (uuid.UUID, bool) {
	userIDStr, ok := GetUserID(c)
	if !ok {
		return uuid.Nil, false
	}
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return uuid.Nil, false
	}
	return userID, true
}

// GetClaims extracts JWT claims from gin context
func GetClaims(c *gin.Context) (*auth.JWTClaims, bool) {
	claims, exists := c.Get(ContextKeyClaims)
	if !exists {
		return nil, false
	}
	return claims.(*auth.JWTClaims), true
}

// GetToken extracts token from gin context
func GetToken(c *gin.Context) (string, bool) {
	token, exists := c.Get(ContextKeyToken)
	if !exists {
		return "", false
	}
	return token.(string), true
}

// IsAdmin checks if the current user is admin
func IsAdmin(c *gin.Context) bool {
	isAdmin, exists := c.Get(ContextKeyIsAdmin)
	if !exists {
		return false
	}
	return isAdmin.(bool)
}

// GetUser extracts full user from gin context
func GetUser(c *gin.Context) (*pkgmodels.User, bool) {
	user, exists := c.Get(ContextKeyUser)
	if !exists {
		return nil, false
	}
	return user.(*pkgmodels.User), true
}

// GetAuthMethod returns the authentication method used
func GetAuthMethod(c *gin.Context) string {
	method, exists := c.Get(ContextKeyAuthMethod)
	if !exists {
		return "jwt"
	}
	return method.(string)
}

// GetServiceKeyID returns the service key ID if authenticated via service key
func GetServiceKeyID(c *gin.Context) (string, bool) {
	id, exists := c.Get(ContextKeyServiceKeyID)
	if !exists {
		return "", false
	}
	return id.(string), true
}

// IsServiceKeyAuth returns true if the request is authenticated via service key
func IsServiceKeyAuth(c *gin.Context) bool {
	return GetAuthMethod(c) == "service_key"
}
