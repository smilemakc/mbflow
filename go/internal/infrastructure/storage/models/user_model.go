package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/uptrace/bun"

	pkgmodels "github.com/smilemakc/mbflow/go/pkg/models"
)

// UserModel represents a user account in the database
type UserModel struct {
	bun.BaseModel `bun:"table:mbflow_users,alias:u"`

	ID                  uuid.UUID  `bun:"id,pk,type:uuid,default:uuid_generate_v4()" json:"id"`
	Email               string     `bun:"email,notnull,unique" json:"email" validate:"required,email,max=255"`
	Username            string     `bun:"username,notnull,unique" json:"username" validate:"required,min=3,max=50"`
	PasswordHash        string     `bun:"password_hash,notnull" json:"-"`
	FullName            string     `bun:"full_name" json:"full_name,omitempty" validate:"max=255"`
	IsActive            bool       `bun:"is_active,notnull,default:true" json:"is_active"`
	IsAdmin             bool       `bun:"is_admin,notnull,default:false" json:"is_admin"`
	EmailVerified       bool       `bun:"email_verified,notnull,default:false" json:"email_verified"`
	FailedLoginAttempts int        `bun:"failed_login_attempts,notnull,default:0" json:"failed_login_attempts"`
	LockedUntil         *time.Time `bun:"locked_until" json:"locked_until,omitempty"`
	Metadata            JSONBMap   `bun:"metadata,type:jsonb,default:'{}'" json:"metadata,omitempty"`
	CreatedAt           time.Time  `bun:"created_at,notnull,default:current_timestamp" json:"created_at"`
	UpdatedAt           time.Time  `bun:"updated_at,notnull,default:current_timestamp" json:"updated_at"`
	LastLoginAt         *time.Time `bun:"last_login_at" json:"last_login_at,omitempty"`
	DeletedAt           *time.Time `bun:"deleted_at" json:"deleted_at,omitempty"`

	// Relations
	Roles    []*RoleModel    `bun:"m2m:mbflow_user_roles,join:User=Role" json:"roles,omitempty"`
	Sessions []*SessionModel `bun:"rel:has-many,join:id=user_id" json:"sessions,omitempty"`
}

// TableName returns the table name for UserModel
func (UserModel) TableName() string {
	return "mbflow_users"
}

// BeforeInsert hook to set timestamps and defaults
func (u *UserModel) BeforeInsert(ctx any) error {
	now := time.Now()
	u.CreatedAt = now
	u.UpdatedAt = now
	if u.ID == uuid.Nil {
		u.ID = uuid.New()
	}
	if u.Metadata == nil {
		u.Metadata = make(JSONBMap)
	}
	return nil
}

// BeforeUpdate hook to update timestamp
func (u *UserModel) BeforeUpdate(ctx any) error {
	u.UpdatedAt = time.Now()
	return nil
}

// IsLocked returns true if user account is locked
func (u *UserModel) IsLocked() bool {
	if u.LockedUntil == nil {
		return false
	}
	return time.Now().Before(*u.LockedUntil)
}

// IsDeleted returns true if user is soft-deleted
func (u *UserModel) IsDeleted() bool {
	return u.DeletedAt != nil
}

// CanLogin returns true if user can authenticate
func (u *UserModel) CanLogin() bool {
	return u.IsActive && !u.IsLocked() && !u.IsDeleted()
}

// SessionModel represents an authentication session in the database
type SessionModel struct {
	bun.BaseModel `bun:"table:mbflow_sessions,alias:s"`

	ID             uuid.UUID `bun:"id,pk,type:uuid,default:uuid_generate_v4()" json:"id"`
	UserID         uuid.UUID `bun:"user_id,notnull,type:uuid" json:"user_id" validate:"required"`
	Token          string    `bun:"token,notnull,unique" json:"token" validate:"required"`
	RefreshToken   string    `bun:"refresh_token" json:"refresh_token,omitempty"`
	ExpiresAt      time.Time `bun:"expires_at,notnull" json:"expires_at" validate:"required"`
	IPAddress      string    `bun:"ip_address" json:"ip_address,omitempty" validate:"max=45"`
	UserAgent      string    `bun:"user_agent" json:"user_agent,omitempty" validate:"max=500"`
	Metadata       JSONBMap  `bun:"metadata,type:jsonb,default:'{}'" json:"metadata,omitempty"`
	CreatedAt      time.Time `bun:"created_at,notnull,default:current_timestamp" json:"created_at"`
	LastActivityAt time.Time `bun:"last_activity_at,notnull,default:current_timestamp" json:"last_activity_at"`

	// Relations
	User *UserModel `bun:"rel:belongs-to,join:user_id=id" json:"user,omitempty"`
}

// TableName returns the table name for SessionModel
func (SessionModel) TableName() string {
	return "mbflow_sessions"
}

// BeforeInsert hook to set timestamps
func (s *SessionModel) BeforeInsert(ctx any) error {
	now := time.Now()
	s.CreatedAt = now
	s.LastActivityAt = now
	if s.ID == uuid.Nil {
		s.ID = uuid.New()
	}
	if s.Metadata == nil {
		s.Metadata = make(JSONBMap)
	}
	return nil
}

// IsExpired returns true if session has expired
func (s *SessionModel) IsExpired() bool {
	return time.Now().After(s.ExpiresAt)
}

// IsValid returns true if session is valid and not expired
func (s *SessionModel) IsValid() bool {
	return !s.IsExpired()
}

// RoleModel represents a user role in the database
type RoleModel struct {
	bun.BaseModel `bun:"table:mbflow_roles,alias:r"`

	ID          uuid.UUID   `bun:"id,pk,type:uuid,default:uuid_generate_v4()" json:"id"`
	Name        string      `bun:"name,notnull,unique" json:"name" validate:"required,min=2,max=100"`
	Description string      `bun:"description" json:"description,omitempty" validate:"max=500"`
	IsSystem    bool        `bun:"is_system,notnull,default:false" json:"is_system"`
	Permissions StringArray `bun:"permissions,type:text[],notnull,default:'{}'" json:"permissions"`
	Metadata    JSONBMap    `bun:"metadata,type:jsonb,default:'{}'" json:"metadata,omitempty"`
	CreatedAt   time.Time   `bun:"created_at,notnull,default:current_timestamp" json:"created_at"`
	UpdatedAt   time.Time   `bun:"updated_at,notnull,default:current_timestamp" json:"updated_at"`

	// Relations
	Users []*UserModel `bun:"m2m:mbflow_user_roles,join:Role=User" json:"users,omitempty"`
}

// TableName returns the table name for RoleModel
func (RoleModel) TableName() string {
	return "mbflow_roles"
}

// BeforeInsert hook to set timestamps and defaults
func (r *RoleModel) BeforeInsert(ctx any) error {
	now := time.Now()
	r.CreatedAt = now
	r.UpdatedAt = now
	if r.ID == uuid.Nil {
		r.ID = uuid.New()
	}
	if r.Metadata == nil {
		r.Metadata = make(JSONBMap)
	}
	if r.Permissions == nil {
		r.Permissions = make(StringArray, 0)
	}
	return nil
}

// BeforeUpdate hook to update timestamp
func (r *RoleModel) BeforeUpdate(ctx any) error {
	r.UpdatedAt = time.Now()
	return nil
}

// HasPermission checks if role has specific permission
func (r *RoleModel) HasPermission(permission string) bool {
	for _, p := range r.Permissions {
		if p == permission {
			return true
		}
	}
	return false
}

// UserRoleModel represents the many-to-many relationship between users and roles
type UserRoleModel struct {
	bun.BaseModel `bun:"table:mbflow_user_roles,alias:ur"`

	UserID     uuid.UUID  `bun:"user_id,pk,type:uuid" json:"user_id" validate:"required"`
	RoleID     uuid.UUID  `bun:"role_id,pk,type:uuid" json:"role_id" validate:"required"`
	AssignedAt time.Time  `bun:"assigned_at,notnull,default:current_timestamp" json:"assigned_at"`
	AssignedBy *uuid.UUID `bun:"assigned_by,type:uuid" json:"assigned_by,omitempty"`

	// Relations for m2m
	User *UserModel `bun:"rel:belongs-to,join:user_id=id"`
	Role *RoleModel `bun:"rel:belongs-to,join:role_id=id"`
}

// TableName returns the table name for UserRoleModel
func (UserRoleModel) TableName() string {
	return "mbflow_user_roles"
}

// BeforeInsert hook to set timestamps
func (ur *UserRoleModel) BeforeInsert(ctx any) error {
	ur.AssignedAt = time.Now()
	return nil
}

// AuditLogModel represents an audit log entry for security tracking
type AuditLogModel struct {
	bun.BaseModel `bun:"table:mbflow_audit_logs,alias:al"`

	ID           uuid.UUID  `bun:"id,pk,type:uuid,default:uuid_generate_v4()" json:"id"`
	UserID       *uuid.UUID `bun:"user_id,type:uuid" json:"user_id,omitempty"`
	Action       string     `bun:"action,notnull" json:"action" validate:"required,max=100"`
	ResourceType string     `bun:"resource_type" json:"resource_type,omitempty" validate:"max=100"`
	ResourceID   *uuid.UUID `bun:"resource_id,type:uuid" json:"resource_id,omitempty"`
	IPAddress    string     `bun:"ip_address" json:"ip_address,omitempty" validate:"max=45"`
	UserAgent    string     `bun:"user_agent" json:"user_agent,omitempty" validate:"max=500"`
	Metadata     JSONBMap   `bun:"metadata,type:jsonb,default:'{}'" json:"metadata,omitempty"`
	CreatedAt    time.Time  `bun:"created_at,notnull,default:current_timestamp" json:"created_at"`
}

// TableName returns the table name for AuditLogModel
func (AuditLogModel) TableName() string {
	return "mbflow_audit_logs"
}

// BeforeInsert hook to set timestamps
func (a *AuditLogModel) BeforeInsert(ctx any) error {
	a.CreatedAt = time.Now()
	if a.ID == uuid.Nil {
		a.ID = uuid.New()
	}
	if a.Metadata == nil {
		a.Metadata = make(JSONBMap)
	}
	return nil
}

// ============================================================================
// Domain Model Conversions
// ============================================================================

// ToUserDomain converts a UserModel to the domain User model
func ToUserDomain(u *UserModel, roles []string) *pkgmodels.User {
	if u == nil {
		return nil
	}

	var metadata map[string]any
	if u.Metadata != nil {
		metadata = u.Metadata
	}

	return &pkgmodels.User{
		ID:           u.ID.String(),
		Email:        u.Email,
		Username:     u.Username,
		PasswordHash: u.PasswordHash,
		FullName:     u.FullName,
		IsActive:     u.IsActive,
		IsAdmin:      u.IsAdmin,
		Roles:        roles,
		Metadata:     metadata,
		CreatedAt:    u.CreatedAt,
		UpdatedAt:    u.UpdatedAt,
		LastLoginAt:  u.LastLoginAt,
	}
}

// ToRoleDomain converts a RoleModel to the domain Role model
func ToRoleDomain(r *RoleModel) *pkgmodels.Role {
	if r == nil {
		return nil
	}

	permissions := make([]string, len(r.Permissions))
	copy(permissions, r.Permissions)

	return &pkgmodels.Role{
		ID:          r.ID.String(),
		Name:        r.Name,
		Description: r.Description,
		Permissions: permissions,
		IsSystem:    r.IsSystem,
		CreatedAt:   r.CreatedAt,
		UpdatedAt:   r.UpdatedAt,
	}
}

// ToSessionDomain converts a SessionModel to the domain Session model
func ToSessionDomain(s *SessionModel) *pkgmodels.Session {
	if s == nil {
		return nil
	}

	return &pkgmodels.Session{
		ID:           s.ID.String(),
		UserID:       s.UserID.String(),
		Token:        s.Token,
		RefreshToken: s.RefreshToken,
		ExpiresAt:    s.ExpiresAt,
		IPAddress:    s.IPAddress,
		UserAgent:    s.UserAgent,
		CreatedAt:    s.CreatedAt,
	}
}
