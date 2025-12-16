package auth

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/smilemakc/mbflow/internal/config"
	"github.com/smilemakc/mbflow/internal/domain/repository"
	"github.com/smilemakc/mbflow/internal/infrastructure/storage/models"
	pkgmodels "github.com/smilemakc/mbflow/pkg/models"
)

var (
	ErrUserNotFound           = errors.New("user not found")
	ErrUserAlreadyExists      = errors.New("user already exists")
	ErrInvalidCredentials     = errors.New("invalid credentials")
	ErrAccountLocked          = errors.New("account is locked")
	ErrAccountInactive        = errors.New("account is inactive")
	ErrEmailAlreadyTaken      = errors.New("email is already taken")
	ErrUsernameAlreadyTaken   = errors.New("username is already taken")
	ErrInvalidRefreshToken    = errors.New("invalid refresh token")
	ErrRefreshTokenExpired    = errors.New("refresh token has expired")
	ErrRegistrationDisabled   = errors.New("registration is disabled")
	ErrRoleNotFound           = errors.New("role not found")
	ErrCannotDeleteSystemRole = errors.New("cannot delete system role")
)

// Service handles authentication and authorization operations
type Service struct {
	userRepo        repository.UserRepository
	jwtService      *JWTService
	passwordService *PasswordService
	config          *config.AuthConfig
}

// NewService creates a new auth service
func NewService(
	userRepo repository.UserRepository,
	cfg *config.AuthConfig,
) *Service {
	return &Service{
		userRepo:        userRepo,
		jwtService:      NewJWTService(cfg),
		passwordService: NewPasswordService(cfg.MinPasswordLength),
		config:          cfg,
	}
}

// RegisterRequest contains registration data
type RegisterRequest struct {
	Email    string
	Username string
	Password string
	FullName string
}

// LoginRequest contains login credentials
type LoginRequest struct {
	Email    string
	Password string
}

// AuthResult contains authentication result with tokens
type AuthResult struct {
	User         *pkgmodels.User `json:"user"`
	AccessToken  string          `json:"access_token"`
	RefreshToken string          `json:"refresh_token"`
	ExpiresIn    int             `json:"expires_in"`
	TokenType    string          `json:"token_type"`
}

// Register creates a new user account
func (s *Service) Register(ctx context.Context, req *RegisterRequest) (*AuthResult, error) {
	// Check if registration is allowed
	if !s.config.AllowRegistration {
		return nil, ErrRegistrationDisabled
	}

	// Validate password
	if err := s.passwordService.ValidatePassword(req.Password); err != nil {
		return nil, err
	}

	// Check if email is taken
	exists, err := s.userRepo.ExistsByEmail(ctx, req.Email)
	if err != nil {
		return nil, fmt.Errorf("failed to check email: %w", err)
	}
	if exists {
		return nil, ErrEmailAlreadyTaken
	}

	// Check if username is taken
	exists, err = s.userRepo.ExistsByUsername(ctx, req.Username)
	if err != nil {
		return nil, fmt.Errorf("failed to check username: %w", err)
	}
	if exists {
		return nil, ErrUsernameAlreadyTaken
	}

	// Hash password
	passwordHash, err := s.passwordService.HashPassword(req.Password)
	if err != nil {
		return nil, err
	}

	// Create user
	user := &models.UserModel{
		ID:           uuid.New(),
		Email:        req.Email,
		Username:     req.Username,
		PasswordHash: passwordHash,
		FullName:     req.FullName,
		IsActive:     true,
		IsAdmin:      false,
	}

	if err := s.userRepo.Create(ctx, user); err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	// Assign default "user" role
	defaultRole, err := s.userRepo.FindRoleByName(ctx, "user")
	if err == nil && defaultRole != nil {
		_ = s.userRepo.AssignRole(ctx, user.ID, defaultRole.ID, nil)
	}

	// Generate tokens
	return s.generateTokens(ctx, user, "", "")
}

// Login authenticates a user and returns tokens
func (s *Service) Login(ctx context.Context, req *LoginRequest, ipAddress, userAgent string) (*AuthResult, error) {
	// Find user by email
	user, err := s.userRepo.FindByEmail(ctx, req.Email)
	if err != nil {
		return nil, fmt.Errorf("failed to find user: %w", err)
	}
	if user == nil {
		return nil, ErrInvalidCredentials
	}

	// Check if account is active
	if !user.IsActive {
		s.logAuditEvent(ctx, user.ID, "login_failed", "account_inactive", ipAddress, userAgent)
		return nil, ErrAccountInactive
	}

	// Check if account is locked
	if user.IsLocked() {
		s.logAuditEvent(ctx, user.ID, "login_failed", "account_locked", ipAddress, userAgent)
		return nil, ErrAccountLocked
	}

	// Verify password
	if err := s.passwordService.VerifyPassword(req.Password, user.PasswordHash); err != nil {
		// Increment failed attempts
		_ = s.userRepo.IncrementFailedAttempts(ctx, user.ID)

		// Check if should lock account
		if user.FailedLoginAttempts+1 >= s.config.MaxLoginAttempts {
			lockUntil := time.Now().Add(s.config.LockoutDuration).Format(time.RFC3339)
			_ = s.userRepo.LockAccount(ctx, user.ID, &lockUntil)
		}

		s.logAuditEvent(ctx, user.ID, "login_failed", "invalid_password", ipAddress, userAgent)
		return nil, ErrInvalidCredentials
	}

	// Reset failed attempts on successful login
	_ = s.userRepo.ResetFailedAttempts(ctx, user.ID)
	_ = s.userRepo.UpdateLastLogin(ctx, user.ID)

	// Generate tokens
	result, err := s.generateTokens(ctx, user, ipAddress, userAgent)
	if err != nil {
		return nil, err
	}

	s.logAuditEvent(ctx, user.ID, "login_success", "", ipAddress, userAgent)
	return result, nil
}

// Logout invalidates a user session
func (s *Service) Logout(ctx context.Context, token string) error {
	if err := s.userRepo.DeleteSession(ctx, token); err != nil {
		return fmt.Errorf("failed to delete session: %w", err)
	}
	return nil
}

// LogoutAll invalidates all sessions for a user
func (s *Service) LogoutAll(ctx context.Context, userID uuid.UUID) error {
	if err := s.userRepo.DeleteSessionsByUserID(ctx, userID); err != nil {
		return fmt.Errorf("failed to delete all sessions: %w", err)
	}
	return nil
}

// RefreshToken refreshes an access token using a refresh token
func (s *Service) RefreshToken(ctx context.Context, refreshToken string) (*AuthResult, error) {
	// Find session by refresh token
	session, err := s.userRepo.FindSessionByRefreshToken(ctx, refreshToken)
	if err != nil {
		return nil, fmt.Errorf("failed to find session: %w", err)
	}
	if session == nil {
		return nil, ErrInvalidRefreshToken
	}

	// Check if user is still valid
	user, err := s.userRepo.FindByID(ctx, session.UserID)
	if err != nil {
		return nil, fmt.Errorf("failed to find user: %w", err)
	}
	if user == nil || !user.IsActive {
		// Delete the invalid session
		_ = s.userRepo.DeleteSessionByID(ctx, session.ID)
		return nil, ErrInvalidRefreshToken
	}

	// Delete old session
	if err := s.userRepo.DeleteSessionByID(ctx, session.ID); err != nil {
		return nil, fmt.Errorf("failed to delete old session: %w", err)
	}

	// Generate new tokens
	return s.generateTokens(ctx, user, session.IPAddress, session.UserAgent)
}

// ValidateToken validates an access token and returns the claims
func (s *Service) ValidateToken(ctx context.Context, token string) (*JWTClaims, error) {
	claims, err := s.jwtService.ValidateAccessToken(token)
	if err != nil {
		return nil, err
	}

	// Update session activity
	_ = s.userRepo.UpdateSessionActivity(ctx, token)

	return claims, nil
}

// GetCurrentUser retrieves the current authenticated user
func (s *Service) GetCurrentUser(ctx context.Context, userID string) (*pkgmodels.User, error) {
	id, err := uuid.Parse(userID)
	if err != nil {
		return nil, fmt.Errorf("invalid user ID: %w", err)
	}

	user, err := s.userRepo.FindByIDWithRoles(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to find user: %w", err)
	}
	if user == nil {
		return nil, ErrUserNotFound
	}

	return s.toDomainUser(user), nil
}

// ChangePassword changes a user's password
func (s *Service) ChangePassword(ctx context.Context, userID uuid.UUID, oldPassword, newPassword string) error {
	user, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {
		return fmt.Errorf("failed to find user: %w", err)
	}
	if user == nil {
		return ErrUserNotFound
	}

	// Verify old password
	if err := s.passwordService.VerifyPassword(oldPassword, user.PasswordHash); err != nil {
		return ErrInvalidCredentials
	}

	// Validate new password
	if err := s.passwordService.ValidatePassword(newPassword); err != nil {
		return err
	}

	// Hash new password
	passwordHash, err := s.passwordService.HashPassword(newPassword)
	if err != nil {
		return err
	}

	// Update password
	user.PasswordHash = passwordHash
	if err := s.userRepo.Update(ctx, user); err != nil {
		return fmt.Errorf("failed to update password: %w", err)
	}

	// Invalidate all sessions (security measure)
	_ = s.userRepo.DeleteSessionsByUserID(ctx, userID)

	return nil
}

// generateTokens creates access and refresh tokens for a user
func (s *Service) generateTokens(ctx context.Context, user *models.UserModel, ipAddress, userAgent string) (*AuthResult, error) {
	// Get user roles
	roles, err := s.userRepo.GetUserRoles(ctx, user.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user roles: %w", err)
	}

	roleNames := make([]string, len(roles))
	for i, role := range roles {
		roleNames[i] = role.Name
	}

	// Convert to domain user for token generation
	domainUser := &pkgmodels.User{
		ID:       user.ID.String(),
		Email:    user.Email,
		Username: user.Username,
		FullName: user.FullName,
		IsActive: user.IsActive,
		IsAdmin:  user.IsAdmin,
		Roles:    roleNames,
	}

	// Generate access token
	accessToken, accessExpiresAt, err := s.jwtService.GenerateAccessToken(domainUser)
	if err != nil {
		return nil, fmt.Errorf("failed to generate access token: %w", err)
	}

	// Generate refresh token
	refreshToken, _, err := s.jwtService.GenerateRefreshToken()
	if err != nil {
		return nil, fmt.Errorf("failed to generate refresh token: %w", err)
	}

	// Create session
	session := &models.SessionModel{
		UserID:       user.ID,
		Token:        accessToken,
		RefreshToken: refreshToken,
		ExpiresAt:    accessExpiresAt,
		IPAddress:    ipAddress,
		UserAgent:    userAgent,
	}

	if err := s.userRepo.CreateSession(ctx, session); err != nil {
		return nil, fmt.Errorf("failed to create session: %w", err)
	}

	return &AuthResult{
		User:         domainUser,
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    s.jwtService.GetAccessTokenExpiry(),
		TokenType:    "Bearer",
	}, nil
}

// toDomainUser converts a storage model to domain model
func (s *Service) toDomainUser(user *models.UserModel) *pkgmodels.User {
	roleNames := make([]string, len(user.Roles))
	for i, role := range user.Roles {
		roleNames[i] = role.Name
	}

	var metadata map[string]interface{}
	if user.Metadata != nil {
		metadata = user.Metadata
	}

	return &pkgmodels.User{
		ID:          user.ID.String(),
		Email:       user.Email,
		Username:    user.Username,
		FullName:    user.FullName,
		IsActive:    user.IsActive,
		IsAdmin:     user.IsAdmin,
		Roles:       roleNames,
		Metadata:    metadata,
		CreatedAt:   user.CreatedAt,
		UpdatedAt:   user.UpdatedAt,
		LastLoginAt: user.LastLoginAt,
	}
}

// logAuditEvent logs an audit event
func (s *Service) logAuditEvent(ctx context.Context, userID uuid.UUID, action, errorMsg, ipAddress, userAgent string) {
	status := "success"
	if errorMsg != "" {
		status = "failure"
	}

	log := &models.AuditLogModel{
		UserID:       &userID,
		Action:       action,
		ResourceType: "session",
		IPAddress:    ipAddress,
		UserAgent:    userAgent,
		Metadata: models.JSONBMap{
			"status": status,
		},
	}

	if errorMsg != "" {
		log.Metadata["error"] = errorMsg
	}

	_ = s.userRepo.CreateAuditLog(ctx, log)
}

// ============================================================================
// User Management (Admin operations)
// ============================================================================

// CreateUser creates a new user (admin operation)
func (s *Service) CreateUser(ctx context.Context, req *RegisterRequest, isAdmin bool) (*pkgmodels.User, error) {
	// Validate password
	if err := s.passwordService.ValidatePassword(req.Password); err != nil {
		return nil, err
	}

	// Check if email is taken
	exists, err := s.userRepo.ExistsByEmail(ctx, req.Email)
	if err != nil {
		return nil, fmt.Errorf("failed to check email: %w", err)
	}
	if exists {
		return nil, ErrEmailAlreadyTaken
	}

	// Check if username is taken
	exists, err = s.userRepo.ExistsByUsername(ctx, req.Username)
	if err != nil {
		return nil, fmt.Errorf("failed to check username: %w", err)
	}
	if exists {
		return nil, ErrUsernameAlreadyTaken
	}

	// Hash password
	passwordHash, err := s.passwordService.HashPassword(req.Password)
	if err != nil {
		return nil, err
	}

	// Create user
	user := &models.UserModel{
		ID:           uuid.New(),
		Email:        req.Email,
		Username:     req.Username,
		PasswordHash: passwordHash,
		FullName:     req.FullName,
		IsActive:     true,
		IsAdmin:      isAdmin,
	}

	if err := s.userRepo.Create(ctx, user); err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	return s.toDomainUser(user), nil
}

// GetUser retrieves a user by ID (admin operation)
func (s *Service) GetUser(ctx context.Context, userID uuid.UUID) (*pkgmodels.User, error) {
	user, err := s.userRepo.FindByIDWithRoles(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to find user: %w", err)
	}
	if user == nil {
		return nil, ErrUserNotFound
	}

	return s.toDomainUser(user), nil
}

// ListUsers retrieves all users with pagination (admin operation)
func (s *Service) ListUsers(ctx context.Context, limit, offset int) ([]*pkgmodels.User, int, error) {
	users, err := s.userRepo.FindAll(ctx, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list users: %w", err)
	}

	count, err := s.userRepo.Count(ctx)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count users: %w", err)
	}

	result := make([]*pkgmodels.User, len(users))
	for i, user := range users {
		result[i] = s.toDomainUser(user)
	}

	return result, count, nil
}

// UpdateUser updates a user (admin operation)
func (s *Service) UpdateUser(ctx context.Context, userID uuid.UUID, email, username, fullName string, isActive, isAdmin bool) (*pkgmodels.User, error) {
	user, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to find user: %w", err)
	}
	if user == nil {
		return nil, ErrUserNotFound
	}

	// Check email uniqueness if changed
	if email != user.Email {
		exists, err := s.userRepo.ExistsByEmail(ctx, email)
		if err != nil {
			return nil, fmt.Errorf("failed to check email: %w", err)
		}
		if exists {
			return nil, ErrEmailAlreadyTaken
		}
		user.Email = email
	}

	// Check username uniqueness if changed
	if username != user.Username {
		exists, err := s.userRepo.ExistsByUsername(ctx, username)
		if err != nil {
			return nil, fmt.Errorf("failed to check username: %w", err)
		}
		if exists {
			return nil, ErrUsernameAlreadyTaken
		}
		user.Username = username
	}

	user.FullName = fullName
	user.IsActive = isActive
	user.IsAdmin = isAdmin

	if err := s.userRepo.Update(ctx, user); err != nil {
		return nil, fmt.Errorf("failed to update user: %w", err)
	}

	return s.toDomainUser(user), nil
}

// DeleteUser soft-deletes a user (admin operation)
func (s *Service) DeleteUser(ctx context.Context, userID uuid.UUID) error {
	user, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {
		return fmt.Errorf("failed to find user: %w", err)
	}
	if user == nil {
		return ErrUserNotFound
	}

	// Delete all sessions
	_ = s.userRepo.DeleteSessionsByUserID(ctx, userID)

	// Soft delete user
	if err := s.userRepo.Delete(ctx, userID); err != nil {
		return fmt.Errorf("failed to delete user: %w", err)
	}

	return nil
}

// ResetUserPassword resets a user's password (admin operation)
func (s *Service) ResetUserPassword(ctx context.Context, userID uuid.UUID, newPassword string) error {
	user, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {
		return fmt.Errorf("failed to find user: %w", err)
	}
	if user == nil {
		return ErrUserNotFound
	}

	// Validate new password
	if err := s.passwordService.ValidatePassword(newPassword); err != nil {
		return err
	}

	// Hash new password
	passwordHash, err := s.passwordService.HashPassword(newPassword)
	if err != nil {
		return err
	}

	// Update password
	user.PasswordHash = passwordHash
	if err := s.userRepo.Update(ctx, user); err != nil {
		return fmt.Errorf("failed to update password: %w", err)
	}

	// Invalidate all sessions
	_ = s.userRepo.DeleteSessionsByUserID(ctx, userID)

	return nil
}

// ============================================================================
// Role Management
// ============================================================================

// ListRoles retrieves all roles
func (s *Service) ListRoles(ctx context.Context) ([]*pkgmodels.Role, error) {
	roles, err := s.userRepo.FindAllRoles(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list roles: %w", err)
	}

	result := make([]*pkgmodels.Role, len(roles))
	for i, role := range roles {
		result[i] = &pkgmodels.Role{
			ID:          role.ID.String(),
			Name:        role.Name,
			Description: role.Description,
			Permissions: role.Permissions,
			IsSystem:    role.IsSystem,
			CreatedAt:   role.CreatedAt,
			UpdatedAt:   role.UpdatedAt,
		}
	}

	return result, nil
}

// AssignRoleToUser assigns a role to a user
func (s *Service) AssignRoleToUser(ctx context.Context, userID, roleID uuid.UUID, assignedBy *uuid.UUID) error {
	// Check user exists
	user, err := s.userRepo.FindByID(ctx, userID)
	if err != nil || user == nil {
		return ErrUserNotFound
	}

	// Check role exists
	role, err := s.userRepo.FindRoleByID(ctx, roleID)
	if err != nil || role == nil {
		return ErrRoleNotFound
	}

	return s.userRepo.AssignRole(ctx, userID, roleID, assignedBy)
}

// RemoveRoleFromUser removes a role from a user
func (s *Service) RemoveRoleFromUser(ctx context.Context, userID, roleID uuid.UUID) error {
	return s.userRepo.RemoveRole(ctx, userID, roleID)
}

// GetUserRoles retrieves all roles for a user
func (s *Service) GetUserRoles(ctx context.Context, userID uuid.UUID) ([]*pkgmodels.Role, error) {
	roles, err := s.userRepo.GetUserRoles(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user roles: %w", err)
	}

	result := make([]*pkgmodels.Role, len(roles))
	for i, role := range roles {
		result[i] = &pkgmodels.Role{
			ID:          role.ID.String(),
			Name:        role.Name,
			Description: role.Description,
			Permissions: role.Permissions,
			IsSystem:    role.IsSystem,
			CreatedAt:   role.CreatedAt,
			UpdatedAt:   role.UpdatedAt,
		}
	}

	return result, nil
}

// HasPermission checks if a user has a specific permission
func (s *Service) HasPermission(ctx context.Context, userID uuid.UUID, permission string) (bool, error) {
	return s.userRepo.HasPermission(ctx, userID, permission)
}
