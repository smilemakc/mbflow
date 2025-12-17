package rest

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/smilemakc/mbflow/internal/application/auth"
)

// AuthHandlers contains handlers for authentication endpoints
type AuthHandlers struct {
	authService     *auth.Service
	providerManager *auth.ProviderManager
	rateLimiter     *LoginRateLimiter
}

// NewAuthHandlers creates new authentication handlers
func NewAuthHandlers(authService *auth.Service, pm *auth.ProviderManager, rateLimiter *LoginRateLimiter) *AuthHandlers {
	return &AuthHandlers{
		authService:     authService,
		providerManager: pm,
		rateLimiter:     rateLimiter,
	}
}

// RegisterRequest represents a registration request
type RegisterRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=8"`
	FullName string `json:"full_name"`
}

// LoginRequest represents a login request
type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

// RefreshRequest represents a token refresh request
type RefreshRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

// ChangePasswordRequest represents a password change request
type ChangePasswordRequest struct {
	OldPassword string `json:"old_password" binding:"required"`
	NewPassword string `json:"new_password" binding:"required,min=8"`
}

// AuthResponse represents an authentication response
type AuthResponse struct {
	User         interface{} `json:"user"`
	AccessToken  string      `json:"access_token"`
	RefreshToken string      `json:"refresh_token,omitempty"`
	ExpiresIn    int         `json:"expires_in"`
	TokenType    string      `json:"token_type"`
}

// HandleRegister handles user registration
// POST /api/v1/auth/register
func (h *AuthHandlers) HandleRegister(c *gin.Context) {
	var req RegisterRequest
	if err := bindJSON(c, &req); err != nil {
		return
	}

	result, err := h.authService.Register(c.Request.Context(), &auth.RegisterRequest{
		Email:    req.Email,
		Password: req.Password,
		FullName: req.FullName,
	})
	if err != nil {
		respondAPIErrorWithRequestID(c, err)
		return
	}

	respondJSON(c, http.StatusCreated, AuthResponse{
		User:         result.User,
		AccessToken:  result.AccessToken,
		RefreshToken: result.RefreshToken,
		ExpiresIn:    result.ExpiresIn,
		TokenType:    result.TokenType,
	})
}

// HandleLogin handles user login
// POST /api/v1/auth/login
func (h *AuthHandlers) HandleLogin(c *gin.Context) {
	var req LoginRequest
	if err := bindJSON(c, &req); err != nil {
		return
	}

	clientIP := c.ClientIP()

	// Check if IP is rate limited
	if h.rateLimiter != nil && h.rateLimiter.IsBlocked(clientIP) {
		respondErrorWithDetails(c, http.StatusTooManyRequests, "too many login attempts", "RATE_LIMIT_EXCEEDED", map[string]interface{}{
			"retry_after": 900, // 15 minutes
		})
		return
	}

	result, err := h.authService.Login(c.Request.Context(), &auth.LoginRequest{
		Email:    req.Email,
		Password: req.Password,
	}, clientIP, c.GetHeader("User-Agent"))

	if err != nil {
		if h.rateLimiter != nil {
			h.rateLimiter.RecordFailedAttempt(clientIP)
		}

		if err == auth.ErrInvalidCredentials && h.rateLimiter != nil {
			remaining := h.rateLimiter.GetRemainingAttempts(clientIP)
			apiErr := TranslateError(err)
			if apiErr.Details == nil {
				apiErr.Details = make(map[string]interface{})
			}
			apiErr.Details["remaining_attempts"] = remaining
			apiErr.Details["request_id"] = GetRequestID(c)
			c.JSON(apiErr.HTTPStatus, apiErr)
		} else {
			respondAPIErrorWithRequestID(c, err)
		}
		return
	}

	// Reset rate limit on successful login
	if h.rateLimiter != nil {
		h.rateLimiter.RecordSuccessfulLogin(clientIP)
	}

	respondJSON(c, http.StatusOK, AuthResponse{
		User:         result.User,
		AccessToken:  result.AccessToken,
		RefreshToken: result.RefreshToken,
		ExpiresIn:    result.ExpiresIn,
		TokenType:    result.TokenType,
	})
}

func (h *AuthHandlers) HandleLogout(c *gin.Context) {
	token, ok := GetToken(c)
	if !ok {
		respondAPIError(c, ErrUnauthorized)
		return
	}

	if err := h.authService.Logout(c.Request.Context(), token); err != nil {
		respondAPIErrorWithRequestID(c, err)
		return
	}

	respondJSON(c, http.StatusOK, gin.H{"message": "logged out successfully"})
}

// HandleRefresh handles token refresh
// POST /api/v1/auth/refresh
func (h *AuthHandlers) HandleRefresh(c *gin.Context) {
	var req RefreshRequest
	if err := bindJSON(c, &req); err != nil {
		return
	}

	result, err := h.authService.RefreshToken(c.Request.Context(), req.RefreshToken)
	if err != nil {
		respondAPIErrorWithRequestID(c, err)
		return
	}

	respondJSON(c, http.StatusOK, AuthResponse{
		User:         result.User,
		AccessToken:  result.AccessToken,
		RefreshToken: result.RefreshToken,
		ExpiresIn:    result.ExpiresIn,
		TokenType:    result.TokenType,
	})
}

func (h *AuthHandlers) HandleGetMe(c *gin.Context) {
	userID, ok := GetUserID(c)
	if !ok {
		respondAPIError(c, ErrUnauthorized)
		return
	}

	user, err := h.authService.GetCurrentUser(c.Request.Context(), userID)
	if err != nil {
		respondAPIErrorWithRequestID(c, err)
		return
	}

	respondJSON(c, http.StatusOK, gin.H{"user": user})
}

// HandleChangePassword handles password change
// POST /api/v1/auth/password
func (h *AuthHandlers) HandleChangePassword(c *gin.Context) {
	userID, ok := GetUserIDAsUUID(c)
	if !ok {
		respondAPIError(c, ErrUnauthorized)
		return
	}

	var req ChangePasswordRequest
	if err := bindJSON(c, &req); err != nil {
		return
	}

	err := h.authService.ChangePassword(c.Request.Context(), userID, req.OldPassword, req.NewPassword)
	if err != nil {
		respondAPIErrorWithRequestID(c, err)
		return
	}

	respondJSON(c, http.StatusOK, gin.H{"message": "password changed successfully"})
}

// HandleGetAuthInfo returns information about available authentication methods
// GET /api/v1/auth/info
func (h *AuthHandlers) HandleGetAuthInfo(c *gin.Context) {
	providers := h.providerManager.GetAvailableProviders()
	providerNames := make([]string, len(providers))
	for i, p := range providers {
		providerNames[i] = string(p)
	}

	respondJSON(c, http.StatusOK, gin.H{
		"mode":              h.providerManager.GetMode(),
		"providers":         providerNames,
		"gateway_available": h.providerManager.IsGatewayAvailable(),
	})
}

// ============================================================================
// OAuth/OIDC Handlers
// ============================================================================

// HandleOAuthAuthorize redirects to OAuth provider for authorization
// GET /api/v1/auth/oauth/authorize
func (h *AuthHandlers) HandleOAuthAuthorize(c *gin.Context) {
	if !h.providerManager.IsGatewayAvailable() {
		respondAPIError(c, NewAPIError("OAUTH_NOT_AVAILABLE", "OAuth is not configured", http.StatusNotFound))
		return
	}

	state, err := auth.GenerateState()
	if err != nil {
		respondAPIErrorWithRequestID(c, NewAPIError("STATE_GENERATION_FAILED", "Failed to generate state", http.StatusInternalServerError))
		return
	}

	nonce, err := auth.GenerateNonce()
	if err != nil {
		respondAPIErrorWithRequestID(c, NewAPIError("NONCE_GENERATION_FAILED", "Failed to generate nonce", http.StatusInternalServerError))
		return
	}

	// Store state and nonce in session/cookie for verification
	c.SetCookie("oauth_state", state, 600, "/", "", true, true)
	c.SetCookie("oauth_nonce", nonce, 600, "/", "", true, true)

	authURL := h.providerManager.GetAuthorizationURL(state, nonce)
	if authURL == "" {
		respondAPIErrorWithRequestID(c, NewAPIError("AUTH_URL_GENERATION_FAILED", "Failed to generate authorization URL", http.StatusInternalServerError))
		return
	}

	c.Redirect(http.StatusTemporaryRedirect, authURL)
}

func (h *AuthHandlers) HandleOAuthCallback(c *gin.Context) {
	code := c.Query("code")
	state := c.Query("state")
	errorParam := c.Query("error")

	if errorParam != "" {
		errorDesc := c.Query("error_description")
		respondAPIError(c, NewAPIErrorWithDetails("OAUTH_ERROR", "OAuth authentication failed", http.StatusBadRequest, map[string]interface{}{
			"error":             errorParam,
			"error_description": errorDesc,
		}))
		return
	}

	if code == "" || state == "" {
		respondAPIError(c, NewAPIError("MISSING_OAUTH_PARAMS", "Missing code or state parameter", http.StatusBadRequest))
		return
	}

	savedState, err := c.Cookie("oauth_state")
	if err != nil || savedState != state {
		respondAPIError(c, NewAPIError("INVALID_STATE", "Invalid or expired state parameter", http.StatusBadRequest))
		return
	}

	c.SetCookie("oauth_state", "", -1, "/", "", true, true)
	c.SetCookie("oauth_nonce", "", -1, "/", "", true, true)

	result, err := h.providerManager.HandleOAuthCallback(c.Request.Context(), code, state)
	if err != nil {
		respondAPIErrorWithRequestID(c, err)
		return
	}

	respondJSON(c, http.StatusOK, AuthResponse{
		User:         result.User,
		AccessToken:  result.AccessToken,
		RefreshToken: result.RefreshToken,
		ExpiresIn:    result.ExpiresIn,
		TokenType:    result.TokenType,
	})
}

// ============================================================================
// Admin User Management Handlers
// ============================================================================

// AdminCreateUserRequest represents admin user creation request
type AdminCreateUserRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=8"`
	FullName string `json:"full_name"`
	IsAdmin  bool   `json:"is_admin"`
}

// AdminUpdateUserRequest represents admin user update request
type AdminUpdateUserRequest struct {
	Email    string `json:"email" binding:"required,email"`
	FullName string `json:"full_name"`
	IsActive bool   `json:"is_active"`
	IsAdmin  bool   `json:"is_admin"`
}

func (h *AuthHandlers) HandleAdminListUsers(c *gin.Context) {
	limit := getQueryInt(c, "limit", 50)
	offset := getQueryInt(c, "offset", 0)

	users, total, err := h.authService.ListUsers(c.Request.Context(), limit, offset)
	if err != nil {
		respondAPIErrorWithRequestID(c, err)
		return
	}

	respondSuccess(c, http.StatusOK, users, &MetaInfo{
		Total:  total,
		Limit:  limit,
		Offset: offset,
	})
}

// HandleAdminGetUser gets a user by ID (admin only)
// GET /api/v1/admin/users/:id
func (h *AuthHandlers) HandleAdminGetUser(c *gin.Context) {
	idStr, ok := getParam(c, "id")
	if !ok {
		return
	}

	userID, err := uuid.Parse(idStr)
	if err != nil {
		respondAPIError(c, ErrInvalidID)
		return
	}

	user, err := h.authService.GetUser(c.Request.Context(), userID)
	if err != nil {
		respondAPIErrorWithRequestID(c, err)
		return
	}

	respondJSON(c, http.StatusOK, gin.H{"user": user})
}

// HandleAdminCreateUser creates a new user (admin only)
// POST /api/v1/admin/users
func (h *AuthHandlers) HandleAdminCreateUser(c *gin.Context) {
	var req AdminCreateUserRequest
	if err := bindJSON(c, &req); err != nil {
		return
	}

	user, err := h.authService.CreateUser(c.Request.Context(), &auth.RegisterRequest{
		Email:    req.Email,
		Password: req.Password,
		FullName: req.FullName,
	}, req.IsAdmin)

	if err != nil {
		respondAPIErrorWithRequestID(c, err)
		return
	}

	respondJSON(c, http.StatusCreated, gin.H{"user": user})
}

// HandleAdminUpdateUser updates a user (admin only)
// PUT /api/v1/admin/users/:id
func (h *AuthHandlers) HandleAdminUpdateUser(c *gin.Context) {
	idStr, ok := getParam(c, "id")
	if !ok {
		return
	}

	userID, err := uuid.Parse(idStr)
	if err != nil {
		respondAPIError(c, ErrInvalidID)
		return
	}

	var req AdminUpdateUserRequest
	if err := bindJSON(c, &req); err != nil {
		return
	}

	user, err := h.authService.UpdateUser(c.Request.Context(), userID, req.Email, req.Email, req.FullName, req.IsActive, req.IsAdmin)
	if err != nil {
		respondAPIErrorWithRequestID(c, err)
		return
	}

	respondJSON(c, http.StatusOK, gin.H{"user": user})
}

// HandleAdminDeleteUser deletes a user (admin only)
// DELETE /api/v1/admin/users/:id
func (h *AuthHandlers) HandleAdminDeleteUser(c *gin.Context) {
	idStr, ok := getParam(c, "id")
	if !ok {
		return
	}

	userID, err := uuid.Parse(idStr)
	if err != nil {
		respondAPIError(c, ErrInvalidID)
		return
	}

	currentUserID, _ := GetUserIDAsUUID(c)
	if userID == currentUserID {
		respondAPIError(c, NewAPIError("SELF_DELETION_FORBIDDEN", "Cannot delete your own account", http.StatusBadRequest))
		return
	}

	if err := h.authService.DeleteUser(c.Request.Context(), userID); err != nil {
		respondAPIErrorWithRequestID(c, err)
		return
	}

	respondJSON(c, http.StatusOK, gin.H{"message": "user deleted successfully"})
}

// HandleAdminResetPassword resets a user's password (admin only)
// POST /api/v1/admin/users/:id/reset-password
func (h *AuthHandlers) HandleAdminResetPassword(c *gin.Context) {
	idStr, ok := getParam(c, "id")
	if !ok {
		return
	}

	userID, err := uuid.Parse(idStr)
	if err != nil {
		respondAPIError(c, ErrInvalidID)
		return
	}

	var req struct {
		NewPassword string `json:"new_password" binding:"required,min=8"`
	}
	if err := bindJSON(c, &req); err != nil {
		return
	}

	if err := h.authService.ResetUserPassword(c.Request.Context(), userID, req.NewPassword); err != nil {
		respondAPIErrorWithRequestID(c, err)
		return
	}

	respondJSON(c, http.StatusOK, gin.H{"message": "password reset successfully"})
}

// ============================================================================
// Role Management Handlers
// ============================================================================

func (h *AuthHandlers) HandleListRoles(c *gin.Context) {
	roles, err := h.authService.ListRoles(c.Request.Context())
	if err != nil {
		respondAPIErrorWithRequestID(c, err)
		return
	}

	respondJSON(c, http.StatusOK, gin.H{"roles": roles})
}

// HandleAssignRole assigns a role to a user
// POST /api/v1/admin/users/:id/roles
func (h *AuthHandlers) HandleAssignRole(c *gin.Context) {
	idStr, ok := getParam(c, "id")
	if !ok {
		return
	}

	userID, err := uuid.Parse(idStr)
	if err != nil {
		respondAPIError(c, ErrInvalidID)
		return
	}

	var req struct {
		RoleID string `json:"role_id" binding:"required"`
	}
	if err := bindJSON(c, &req); err != nil {
		return
	}

	roleID, err := uuid.Parse(req.RoleID)
	if err != nil {
		respondAPIError(c, NewAPIError("INVALID_ROLE_ID", "Invalid role ID format", http.StatusBadRequest))
		return
	}

	currentUserID, _ := GetUserIDAsUUID(c)
	if err := h.authService.AssignRoleToUser(c.Request.Context(), userID, roleID, &currentUserID); err != nil {
		respondAPIErrorWithRequestID(c, err)
		return
	}

	respondJSON(c, http.StatusOK, gin.H{"message": "role assigned successfully"})
}

// HandleRemoveRole removes a role from a user
// DELETE /api/v1/admin/users/:id/roles/:role_id
func (h *AuthHandlers) HandleRemoveRole(c *gin.Context) {
	idStr, ok := getParam(c, "id")
	if !ok {
		return
	}

	userID, err := uuid.Parse(idStr)
	if err != nil {
		respondAPIError(c, ErrInvalidID)
		return
	}

	roleIDStr, ok := getParam(c, "role_id")
	if !ok {
		return
	}

	roleID, err := uuid.Parse(roleIDStr)
	if err != nil {
		respondAPIError(c, NewAPIError("INVALID_ROLE_ID", "Invalid role ID format", http.StatusBadRequest))
		return
	}

	if err := h.authService.RemoveRoleFromUser(c.Request.Context(), userID, roleID); err != nil {
		respondAPIErrorWithRequestID(c, err)
		return
	}

	respondJSON(c, http.StatusOK, gin.H{"message": "role removed successfully"})
}

func (h *AuthHandlers) HandleGetUserRoles(c *gin.Context) {
	idStr, ok := getParam(c, "id")
	if !ok {
		return
	}

	userID, err := uuid.Parse(idStr)
	if err != nil {
		respondAPIError(c, ErrInvalidID)
		return
	}

	roles, err := h.authService.GetUserRoles(c.Request.Context(), userID)
	if err != nil {
		respondAPIErrorWithRequestID(c, err)
		return
	}

	respondJSON(c, http.StatusOK, gin.H{"roles": roles})
}
