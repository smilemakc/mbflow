// Package models defines the public domain models for MBFlow authorization system.
package models

import (
	"time"
)

// User represents a user account in the system.
type User struct {
	ID           string                 `json:"id"`
	Email        string                 `json:"email"`
	Username     string                 `json:"username"`
	PasswordHash string                 `json:"-"`
	FullName     string                 `json:"full_name,omitempty"`
	IsActive     bool                   `json:"is_active"`
	IsAdmin      bool                   `json:"is_admin"`
	Roles        []string               `json:"roles,omitempty"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt    time.Time              `json:"created_at"`
	UpdatedAt    time.Time              `json:"updated_at"`
	LastLoginAt  *time.Time             `json:"last_login_at,omitempty"`
}

// Session represents an authenticated user session.
type Session struct {
	ID           string    `json:"id"`
	UserID       string    `json:"user_id"`
	Token        string    `json:"token"`
	RefreshToken string    `json:"refresh_token,omitempty"`
	ExpiresAt    time.Time `json:"expires_at"`
	IPAddress    string    `json:"ip_address,omitempty"`
	UserAgent    string    `json:"user_agent,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
}

// Role represents a user role with associated permissions.
type Role struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description,omitempty"`
	Permissions []string  `json:"permissions"`
	IsSystem    bool      `json:"is_system"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// Permission constants define granular access control.
const (
	PermissionWorkflowCreate  = "workflow:create"
	PermissionWorkflowRead    = "workflow:read"
	PermissionWorkflowUpdate  = "workflow:update"
	PermissionWorkflowDelete  = "workflow:delete"
	PermissionWorkflowExecute = "workflow:execute"

	PermissionExecutionRead   = "execution:read"
	PermissionExecutionCancel = "execution:cancel"
	PermissionExecutionRetry  = "execution:retry"

	PermissionTriggerCreate = "trigger:create"
	PermissionTriggerRead   = "trigger:read"
	PermissionTriggerUpdate = "trigger:update"
	PermissionTriggerDelete = "trigger:delete"

	PermissionUserManage  = "user:manage"
	PermissionRoleManage  = "role:manage"
	PermissionSystemAdmin = "system:admin"
)

// AuthResult represents the result of successful authentication.
type AuthResult struct {
	User         *User  `json:"user"`
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
	TokenType    string `json:"token_type"`
}

// JWTClaims represents JWT token claims.
type JWTClaims struct {
	UserID   string   `json:"user_id"`
	Email    string   `json:"email"`
	Username string   `json:"username"`
	IsAdmin  bool     `json:"is_admin"`
	Roles    []string `json:"roles"`
}

// HasRole checks if the user has a specific role.
func (u *User) HasRole(role string) bool {
	if u.Roles == nil {
		return false
	}

	for _, r := range u.Roles {
		if r == role {
			return true
		}
	}

	return false
}

// HasPermission checks if the user has a specific permission through their roles.
func (u *User) HasPermission(permission string) bool {
	if u.IsAdmin {
		return true
	}

	rolePermissions := getRolePermissions()

	for _, roleName := range u.Roles {
		permissions, exists := rolePermissions[roleName]
		if !exists {
			continue
		}

		for _, p := range permissions {
			if p == permission || p == PermissionSystemAdmin {
				return true
			}
		}
	}

	return false
}

// IsOwner checks if the user owns a resource by comparing user ID with resource owner ID.
func (u *User) IsOwner(resourceOwnerID string) bool {
	return u.ID == resourceOwnerID
}

// CanAccessResource checks if the user can access a resource based on ownership or permissions.
func (u *User) CanAccessResource(resourceOwnerID string, requiredPermission string) bool {
	if u.IsAdmin {
		return true
	}

	if u.IsOwner(resourceOwnerID) {
		return true
	}

	return u.HasPermission(requiredPermission)
}

// Validate validates the user structure.
func (u *User) Validate() error {
	if u.Email == "" {
		return &ValidationError{Field: "email", Message: "email is required"}
	}

	if u.Username == "" {
		return &ValidationError{Field: "username", Message: "username is required"}
	}

	if u.PasswordHash == "" {
		return &ValidationError{Field: "password_hash", Message: "password hash is required"}
	}

	return nil
}

// Validate validates the session structure.
func (s *Session) Validate() error {
	if s.UserID == "" {
		return &ValidationError{Field: "user_id", Message: "user ID is required"}
	}

	if s.Token == "" {
		return &ValidationError{Field: "token", Message: "token is required"}
	}

	if s.ExpiresAt.IsZero() {
		return &ValidationError{Field: "expires_at", Message: "expiration time is required"}
	}

	return nil
}

// IsExpired checks if the session has expired.
func (s *Session) IsExpired() bool {
	return time.Now().After(s.ExpiresAt)
}

// Validate validates the role structure.
func (r *Role) Validate() error {
	if r.Name == "" {
		return &ValidationError{Field: "name", Message: "role name is required"}
	}

	if len(r.Permissions) == 0 {
		return &ValidationError{Field: "permissions", Message: "at least one permission is required"}
	}

	return nil
}

// HasPermission checks if the role has a specific permission.
func (r *Role) HasPermission(permission string) bool {
	for _, p := range r.Permissions {
		if p == permission {
			return true
		}
	}
	return false
}

// AuditLog represents an audit log entry for user actions.
type AuditLog struct {
	ID           string                 `json:"id"`
	UserID       *string                `json:"user_id,omitempty"`
	Action       string                 `json:"action"`
	ResourceType string                 `json:"resource_type,omitempty"`
	ResourceID   *string                `json:"resource_id,omitempty"`
	IPAddress    string                 `json:"ip_address,omitempty"`
	UserAgent    string                 `json:"user_agent,omitempty"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt    time.Time              `json:"created_at"`
}

// Validate validates the audit log structure.
func (a *AuditLog) Validate() error {
	if a.Action == "" {
		return &ValidationError{Field: "action", Message: "action is required"}
	}
	return nil
}

// getRolePermissions returns a mapping of role names to their permissions.
func getRolePermissions() map[string][]string {
	return map[string][]string{
		"admin": {
			PermissionSystemAdmin,
			PermissionWorkflowCreate,
			PermissionWorkflowRead,
			PermissionWorkflowUpdate,
			PermissionWorkflowDelete,
			PermissionWorkflowExecute,
			PermissionExecutionRead,
			PermissionExecutionCancel,
			PermissionExecutionRetry,
			PermissionTriggerCreate,
			PermissionTriggerRead,
			PermissionTriggerUpdate,
			PermissionTriggerDelete,
			PermissionUserManage,
			PermissionRoleManage,
		},
		"editor": {
			PermissionWorkflowCreate,
			PermissionWorkflowRead,
			PermissionWorkflowUpdate,
			PermissionWorkflowDelete,
			PermissionWorkflowExecute,
			PermissionExecutionRead,
			PermissionExecutionCancel,
			PermissionExecutionRetry,
			PermissionTriggerCreate,
			PermissionTriggerRead,
			PermissionTriggerUpdate,
			PermissionTriggerDelete,
		},
		"viewer": {
			PermissionWorkflowRead,
			PermissionExecutionRead,
			PermissionTriggerRead,
		},
		"executor": {
			PermissionWorkflowRead,
			PermissionWorkflowExecute,
			PermissionExecutionRead,
			PermissionTriggerRead,
		},
	}
}
