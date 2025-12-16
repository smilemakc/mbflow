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
	Username string `json:"username" binding:"required,min=3,max=50"`
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
		Username: req.Username,
		Password: req.Password,
		FullName: req.FullName,
	})
	if err != nil {
		switch err {
		case auth.ErrEmailAlreadyTaken:
			respondError(c, http.StatusConflict, "email is already taken")
		case auth.ErrUsernameAlreadyTaken:
			respondError(c, http.StatusConflict, "username is already taken")
		case auth.ErrRegistrationDisabled:
			respondError(c, http.StatusForbidden, "registration is disabled")
		default:
			if _, ok := err.(*auth.PasswordError); ok {
				respondError(c, http.StatusBadRequest, err.Error())
			} else {
				respondError(c, http.StatusInternalServerError, "registration failed")
			}
		}
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
		// Record failed attempt
		if h.rateLimiter != nil {
			h.rateLimiter.RecordFailedAttempt(clientIP)
		}

		switch err {
		case auth.ErrInvalidCredentials:
			remaining := 0
			if h.rateLimiter != nil {
				remaining = h.rateLimiter.GetRemainingAttempts(clientIP)
			}
			respondErrorWithDetails(c, http.StatusUnauthorized, "invalid email or password", "INVALID_CREDENTIALS", map[string]interface{}{
				"remaining_attempts": remaining,
			})
		case auth.ErrAccountLocked:
			respondError(c, http.StatusForbidden, "account is locked")
		case auth.ErrAccountInactive:
			respondError(c, http.StatusForbidden, "account is inactive")
		default:
			respondError(c, http.StatusInternalServerError, "login failed")
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

// HandleLogout handles user logout
// POST /api/v1/auth/logout
func (h *AuthHandlers) HandleLogout(c *gin.Context) {
	token, ok := GetToken(c)
	if !ok {
		respondError(c, http.StatusUnauthorized, "not authenticated")
		return
	}

	if err := h.authService.Logout(c.Request.Context(), token); err != nil {
		respondError(c, http.StatusInternalServerError, "logout failed")
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
		switch err {
		case auth.ErrInvalidRefreshToken:
			respondError(c, http.StatusUnauthorized, "invalid refresh token")
		case auth.ErrRefreshTokenExpired:
			respondError(c, http.StatusUnauthorized, "refresh token expired")
		default:
			respondError(c, http.StatusInternalServerError, "token refresh failed")
		}
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

// HandleGetMe handles getting current user info
// GET /api/v1/auth/me
func (h *AuthHandlers) HandleGetMe(c *gin.Context) {
	userID, ok := GetUserID(c)
	if !ok {
		respondError(c, http.StatusUnauthorized, "not authenticated")
		return
	}

	user, err := h.authService.GetCurrentUser(c.Request.Context(), userID)
	if err != nil {
		respondError(c, http.StatusInternalServerError, "failed to get user")
		return
	}

	respondJSON(c, http.StatusOK, gin.H{"user": user})
}

// HandleChangePassword handles password change
// POST /api/v1/auth/password
func (h *AuthHandlers) HandleChangePassword(c *gin.Context) {
	userID, ok := GetUserIDAsUUID(c)
	if !ok {
		respondError(c, http.StatusUnauthorized, "not authenticated")
		return
	}

	var req ChangePasswordRequest
	if err := bindJSON(c, &req); err != nil {
		return
	}

	err := h.authService.ChangePassword(c.Request.Context(), userID, req.OldPassword, req.NewPassword)
	if err != nil {
		switch err {
		case auth.ErrInvalidCredentials:
			respondError(c, http.StatusBadRequest, "current password is incorrect")
		default:
			if _, ok := err.(*auth.PasswordError); ok {
				respondError(c, http.StatusBadRequest, err.Error())
			} else {
				respondError(c, http.StatusInternalServerError, "password change failed")
			}
		}
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
		respondError(c, http.StatusNotFound, "OAuth not available")
		return
	}

	state, err := auth.GenerateState()
	if err != nil {
		respondError(c, http.StatusInternalServerError, "failed to generate state")
		return
	}

	nonce, err := auth.GenerateNonce()
	if err != nil {
		respondError(c, http.StatusInternalServerError, "failed to generate nonce")
		return
	}

	// Store state and nonce in session/cookie for verification
	c.SetCookie("oauth_state", state, 600, "/", "", true, true)
	c.SetCookie("oauth_nonce", nonce, 600, "/", "", true, true)

	authURL := h.providerManager.GetAuthorizationURL(state, nonce)
	if authURL == "" {
		respondError(c, http.StatusInternalServerError, "failed to generate authorization URL")
		return
	}

	c.Redirect(http.StatusTemporaryRedirect, authURL)
}

// HandleOAuthCallback handles OAuth provider callback
// GET /api/v1/auth/oauth/callback
func (h *AuthHandlers) HandleOAuthCallback(c *gin.Context) {
	code := c.Query("code")
	state := c.Query("state")
	errorParam := c.Query("error")

	if errorParam != "" {
		errorDesc := c.Query("error_description")
		respondError(c, http.StatusBadRequest, "OAuth error: "+errorParam+": "+errorDesc)
		return
	}

	if code == "" || state == "" {
		respondError(c, http.StatusBadRequest, "missing code or state")
		return
	}

	// Verify state
	savedState, err := c.Cookie("oauth_state")
	if err != nil || savedState != state {
		respondError(c, http.StatusBadRequest, "invalid state parameter")
		return
	}

	// Clear OAuth cookies
	c.SetCookie("oauth_state", "", -1, "/", "", true, true)
	c.SetCookie("oauth_nonce", "", -1, "/", "", true, true)

	result, err := h.providerManager.HandleOAuthCallback(c.Request.Context(), code, state)
	if err != nil {
		respondError(c, http.StatusInternalServerError, "OAuth callback failed: "+err.Error())
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
	Username string `json:"username" binding:"required,min=3,max=50"`
	Password string `json:"password" binding:"required,min=8"`
	FullName string `json:"full_name"`
	IsAdmin  bool   `json:"is_admin"`
}

// AdminUpdateUserRequest represents admin user update request
type AdminUpdateUserRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Username string `json:"username" binding:"required,min=3,max=50"`
	FullName string `json:"full_name"`
	IsActive bool   `json:"is_active"`
	IsAdmin  bool   `json:"is_admin"`
}

// HandleAdminListUsers lists all users (admin only)
// GET /api/v1/admin/users
func (h *AuthHandlers) HandleAdminListUsers(c *gin.Context) {
	limit := getQueryInt(c, "limit", 50)
	offset := getQueryInt(c, "offset", 0)

	users, total, err := h.authService.ListUsers(c.Request.Context(), limit, offset)
	if err != nil {
		respondError(c, http.StatusInternalServerError, "failed to list users")
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
		respondError(c, http.StatusBadRequest, "invalid user ID")
		return
	}

	user, err := h.authService.GetUser(c.Request.Context(), userID)
	if err != nil {
		if err == auth.ErrUserNotFound {
			respondError(c, http.StatusNotFound, "user not found")
		} else {
			respondError(c, http.StatusInternalServerError, "failed to get user")
		}
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
		Username: req.Username,
		Password: req.Password,
		FullName: req.FullName,
	}, req.IsAdmin)

	if err != nil {
		switch err {
		case auth.ErrEmailAlreadyTaken:
			respondError(c, http.StatusConflict, "email is already taken")
		case auth.ErrUsernameAlreadyTaken:
			respondError(c, http.StatusConflict, "username is already taken")
		default:
			respondError(c, http.StatusInternalServerError, "failed to create user")
		}
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
		respondError(c, http.StatusBadRequest, "invalid user ID")
		return
	}

	var req AdminUpdateUserRequest
	if err := bindJSON(c, &req); err != nil {
		return
	}

	user, err := h.authService.UpdateUser(c.Request.Context(), userID, req.Email, req.Username, req.FullName, req.IsActive, req.IsAdmin)
	if err != nil {
		switch err {
		case auth.ErrUserNotFound:
			respondError(c, http.StatusNotFound, "user not found")
		case auth.ErrEmailAlreadyTaken:
			respondError(c, http.StatusConflict, "email is already taken")
		case auth.ErrUsernameAlreadyTaken:
			respondError(c, http.StatusConflict, "username is already taken")
		default:
			respondError(c, http.StatusInternalServerError, "failed to update user")
		}
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
		respondError(c, http.StatusBadRequest, "invalid user ID")
		return
	}

	// Prevent self-deletion
	currentUserID, _ := GetUserIDAsUUID(c)
	if userID == currentUserID {
		respondError(c, http.StatusBadRequest, "cannot delete your own account")
		return
	}

	if err := h.authService.DeleteUser(c.Request.Context(), userID); err != nil {
		if err == auth.ErrUserNotFound {
			respondError(c, http.StatusNotFound, "user not found")
		} else {
			respondError(c, http.StatusInternalServerError, "failed to delete user")
		}
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
		respondError(c, http.StatusBadRequest, "invalid user ID")
		return
	}

	var req struct {
		NewPassword string `json:"new_password" binding:"required,min=8"`
	}
	if err := bindJSON(c, &req); err != nil {
		return
	}

	if err := h.authService.ResetUserPassword(c.Request.Context(), userID, req.NewPassword); err != nil {
		if err == auth.ErrUserNotFound {
			respondError(c, http.StatusNotFound, "user not found")
		} else {
			respondError(c, http.StatusInternalServerError, "failed to reset password")
		}
		return
	}

	respondJSON(c, http.StatusOK, gin.H{"message": "password reset successfully"})
}

// ============================================================================
// Role Management Handlers
// ============================================================================

// HandleListRoles lists all roles
// GET /api/v1/admin/roles
func (h *AuthHandlers) HandleListRoles(c *gin.Context) {
	roles, err := h.authService.ListRoles(c.Request.Context())
	if err != nil {
		respondError(c, http.StatusInternalServerError, "failed to list roles")
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
		respondError(c, http.StatusBadRequest, "invalid user ID")
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
		respondError(c, http.StatusBadRequest, "invalid role ID")
		return
	}

	currentUserID, _ := GetUserIDAsUUID(c)
	if err := h.authService.AssignRoleToUser(c.Request.Context(), userID, roleID, &currentUserID); err != nil {
		if err == auth.ErrUserNotFound {
			respondError(c, http.StatusNotFound, "user not found")
		} else if err == auth.ErrRoleNotFound {
			respondError(c, http.StatusNotFound, "role not found")
		} else {
			respondError(c, http.StatusInternalServerError, "failed to assign role")
		}
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
		respondError(c, http.StatusBadRequest, "invalid user ID")
		return
	}

	roleIDStr, ok := getParam(c, "role_id")
	if !ok {
		return
	}

	roleID, err := uuid.Parse(roleIDStr)
	if err != nil {
		respondError(c, http.StatusBadRequest, "invalid role ID")
		return
	}

	if err := h.authService.RemoveRoleFromUser(c.Request.Context(), userID, roleID); err != nil {
		respondError(c, http.StatusInternalServerError, "failed to remove role")
		return
	}

	respondJSON(c, http.StatusOK, gin.H{"message": "role removed successfully"})
}

// HandleGetUserRoles gets all roles for a user
// GET /api/v1/admin/users/:id/roles
func (h *AuthHandlers) HandleGetUserRoles(c *gin.Context) {
	idStr, ok := getParam(c, "id")
	if !ok {
		return
	}

	userID, err := uuid.Parse(idStr)
	if err != nil {
		respondError(c, http.StatusBadRequest, "invalid user ID")
		return
	}

	roles, err := h.authService.GetUserRoles(c.Request.Context(), userID)
	if err != nil {
		respondError(c, http.StatusInternalServerError, "failed to get user roles")
		return
	}

	respondJSON(c, http.StatusOK, gin.H{"roles": roles})
}
