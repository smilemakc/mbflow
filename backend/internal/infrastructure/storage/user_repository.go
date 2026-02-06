package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/smilemakc/mbflow/internal/domain/repository"
	"github.com/smilemakc/mbflow/internal/infrastructure/storage/models"
	"github.com/uptrace/bun"
)

// Ensure UserRepository implements the interface
var _ repository.UserRepository = (*UserRepository)(nil)

// UserRepository implements repository.UserRepository using Bun ORM
type UserRepository struct {
	db bun.IDB
}

// NewUserRepository creates a new UserRepository
func NewUserRepository(db bun.IDB) *UserRepository {
	return &UserRepository{db: db}
}

// ============================================================================
// User CRUD Operations
// ============================================================================

// Create creates a new user
func (r *UserRepository) Create(ctx context.Context, user *models.UserModel) error {
	if user.ID == uuid.Nil {
		user.ID = uuid.New()
	}
	now := time.Now()
	user.CreatedAt = now
	user.UpdatedAt = now

	_, err := r.db.NewInsert().Model(user).Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to create user: %w", err)
	}
	return nil
}

// Update updates an existing user
func (r *UserRepository) Update(ctx context.Context, user *models.UserModel) error {
	user.UpdatedAt = time.Now()

	_, err := r.db.NewUpdate().
		Model(user).
		Column("email", "username", "password_hash", "full_name", "is_active", "is_admin",
			"email_verified", "metadata", "updated_at").
		Where("id = ?", user.ID).
		Where("deleted_at IS NULL").
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to update user: %w", err)
	}
	return nil
}

// Delete soft-deletes a user
func (r *UserRepository) Delete(ctx context.Context, id uuid.UUID) error {
	now := time.Now()
	_, err := r.db.NewUpdate().
		Model((*models.UserModel)(nil)).
		Set("deleted_at = ?", now).
		Set("updated_at = ?", now).
		Where("id = ?", id).
		Where("deleted_at IS NULL").
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to soft-delete user: %w", err)
	}
	return nil
}

// HardDelete permanently deletes a user
func (r *UserRepository) HardDelete(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.NewDelete().
		Model((*models.UserModel)(nil)).
		Where("id = ?", id).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to hard-delete user: %w", err)
	}
	return nil
}

// ============================================================================
// User Lookup Operations
// ============================================================================

// FindByID retrieves a user by ID
func (r *UserRepository) FindByID(ctx context.Context, id uuid.UUID) (*models.UserModel, error) {
	user := &models.UserModel{}
	err := r.db.NewSelect().
		Model(user).
		Where("id = ?", id).
		Where("deleted_at IS NULL").
		Scan(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to find user by ID: %w", err)
	}
	return user, nil
}

// FindByEmail retrieves a user by email
func (r *UserRepository) FindByEmail(ctx context.Context, email string) (*models.UserModel, error) {
	user := &models.UserModel{}
	err := r.db.NewSelect().
		Model(user).
		Where("LOWER(email) = LOWER(?)", email).
		Where("deleted_at IS NULL").
		Scan(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to find user by email: %w", err)
	}
	return user, nil
}

// FindByUsername retrieves a user by username
func (r *UserRepository) FindByUsername(ctx context.Context, username string) (*models.UserModel, error) {
	user := &models.UserModel{}
	err := r.db.NewSelect().
		Model(user).
		Where("LOWER(username) = LOWER(?)", username).
		Where("deleted_at IS NULL").
		Scan(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to find user by username: %w", err)
	}
	return user, nil
}

// FindByIDWithRoles retrieves a user with their roles
func (r *UserRepository) FindByIDWithRoles(ctx context.Context, id uuid.UUID) (*models.UserModel, error) {
	user := &models.UserModel{}
	err := r.db.NewSelect().
		Model(user).
		Relation("Roles").
		Where("u.id = ?", id).
		Where("u.deleted_at IS NULL").
		Scan(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to find user with roles: %w", err)
	}
	return user, nil
}

// FindAll retrieves all users with pagination
func (r *UserRepository) FindAll(ctx context.Context, limit, offset int) ([]*models.UserModel, error) {
	var users []*models.UserModel
	err := r.db.NewSelect().
		Model(&users).
		Where("deleted_at IS NULL").
		Limit(limit).
		Offset(offset).
		Order("created_at DESC").
		Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to find all users: %w", err)
	}
	return users, nil
}

// FindAllActive retrieves all active users with pagination
func (r *UserRepository) FindAllActive(ctx context.Context, limit, offset int) ([]*models.UserModel, error) {
	var users []*models.UserModel
	err := r.db.NewSelect().
		Model(&users).
		Where("is_active = ?", true).
		Where("deleted_at IS NULL").
		Limit(limit).
		Offset(offset).
		Order("created_at DESC").
		Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to find active users: %w", err)
	}
	return users, nil
}

// ============================================================================
// Existence Checks
// ============================================================================

// ExistsByEmail checks if a user exists by email
func (r *UserRepository) ExistsByEmail(ctx context.Context, email string) (bool, error) {
	exists, err := r.db.NewSelect().
		Model((*models.UserModel)(nil)).
		Where("LOWER(email) = LOWER(?)", email).
		Where("deleted_at IS NULL").
		Exists(ctx)
	if err != nil {
		return false, fmt.Errorf("failed to check email existence: %w", err)
	}
	return exists, nil
}

// ExistsByUsername checks if a user exists by username
func (r *UserRepository) ExistsByUsername(ctx context.Context, username string) (bool, error) {
	exists, err := r.db.NewSelect().
		Model((*models.UserModel)(nil)).
		Where("LOWER(username) = LOWER(?)", username).
		Where("deleted_at IS NULL").
		Exists(ctx)
	if err != nil {
		return false, fmt.Errorf("failed to check username existence: %w", err)
	}
	return exists, nil
}

// ============================================================================
// Counting
// ============================================================================

// Count returns the total count of users
func (r *UserRepository) Count(ctx context.Context) (int, error) {
	count, err := r.db.NewSelect().
		Model((*models.UserModel)(nil)).
		Where("deleted_at IS NULL").
		Count(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to count users: %w", err)
	}
	return count, nil
}

// CountActive returns the count of active users
func (r *UserRepository) CountActive(ctx context.Context) (int, error) {
	count, err := r.db.NewSelect().
		Model((*models.UserModel)(nil)).
		Where("is_active = ?", true).
		Where("deleted_at IS NULL").
		Count(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to count active users: %w", err)
	}
	return count, nil
}

// ============================================================================
// Login Tracking
// ============================================================================

// UpdateLastLogin updates the last login timestamp
func (r *UserRepository) UpdateLastLogin(ctx context.Context, id uuid.UUID) error {
	now := time.Now()
	_, err := r.db.NewUpdate().
		Model((*models.UserModel)(nil)).
		Set("last_login_at = ?", now).
		Set("updated_at = ?", now).
		Where("id = ?", id).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to update last login: %w", err)
	}
	return nil
}

// IncrementFailedAttempts increments failed login attempts
func (r *UserRepository) IncrementFailedAttempts(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.NewUpdate().
		Model((*models.UserModel)(nil)).
		Set("failed_login_attempts = failed_login_attempts + 1").
		Set("updated_at = ?", time.Now()).
		Where("id = ?", id).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to increment failed attempts: %w", err)
	}
	return nil
}

// ResetFailedAttempts resets failed login attempts to zero
func (r *UserRepository) ResetFailedAttempts(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.NewUpdate().
		Model((*models.UserModel)(nil)).
		Set("failed_login_attempts = 0").
		Set("updated_at = ?", time.Now()).
		Where("id = ?", id).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to reset failed attempts: %w", err)
	}
	return nil
}

// LockAccount locks a user account until specified time
func (r *UserRepository) LockAccount(ctx context.Context, id uuid.UUID, until *string) error {
	var lockedUntil *time.Time
	if until != nil {
		t, err := time.Parse(time.RFC3339, *until)
		if err != nil {
			return fmt.Errorf("invalid lock time format: %w", err)
		}
		lockedUntil = &t
	}

	_, err := r.db.NewUpdate().
		Model((*models.UserModel)(nil)).
		Set("locked_until = ?", lockedUntil).
		Set("updated_at = ?", time.Now()).
		Where("id = ?", id).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to lock account: %w", err)
	}
	return nil
}

// UnlockAccount unlocks a user account
func (r *UserRepository) UnlockAccount(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.NewUpdate().
		Model((*models.UserModel)(nil)).
		Set("locked_until = NULL").
		Set("failed_login_attempts = 0").
		Set("updated_at = ?", time.Now()).
		Where("id = ?", id).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to unlock account: %w", err)
	}
	return nil
}

// ============================================================================
// Session Operations
// ============================================================================

// CreateSession creates a new session
func (r *UserRepository) CreateSession(ctx context.Context, session *models.SessionModel) error {
	if session.ID == uuid.Nil {
		session.ID = uuid.New()
	}
	now := time.Now()
	session.CreatedAt = now
	session.LastActivityAt = now

	// Build insert query with conditional IP address handling
	// PostgreSQL INET type doesn't accept empty strings, so we use NULL for empty IPs
	insertQuery := r.db.NewInsert().Model(session)
	if session.IPAddress == "" {
		insertQuery = insertQuery.ExcludeColumn("ip_address")
	}

	_, err := insertQuery.Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to create session: %w", err)
	}
	return nil
}

// FindSessionByToken retrieves a session by token
func (r *UserRepository) FindSessionByToken(ctx context.Context, token string) (*models.SessionModel, error) {
	session := &models.SessionModel{}
	err := r.db.NewSelect().
		Model(session).
		Relation("User").
		Where("s.token = ?", token).
		Where("s.expires_at > ?", time.Now()).
		Scan(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to find session by token: %w", err)
	}
	return session, nil
}

// FindSessionByRefreshToken retrieves a session by refresh token
func (r *UserRepository) FindSessionByRefreshToken(ctx context.Context, refreshToken string) (*models.SessionModel, error) {
	session := &models.SessionModel{}
	err := r.db.NewSelect().
		Model(session).
		Relation("User").
		Where("s.refresh_token = ?", refreshToken).
		Scan(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to find session by refresh token: %w", err)
	}
	return session, nil
}

// FindSessionsByUserID retrieves all sessions for a user
func (r *UserRepository) FindSessionsByUserID(ctx context.Context, userID uuid.UUID) ([]*models.SessionModel, error) {
	var sessions []*models.SessionModel
	err := r.db.NewSelect().
		Model(&sessions).
		Where("user_id = ?", userID).
		Where("expires_at > ?", time.Now()).
		Order("created_at DESC").
		Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to find sessions by user ID: %w", err)
	}
	return sessions, nil
}

// DeleteSession deletes a session by token
func (r *UserRepository) DeleteSession(ctx context.Context, token string) error {
	_, err := r.db.NewDelete().
		Model((*models.SessionModel)(nil)).
		Where("token = ?", token).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to delete session: %w", err)
	}
	return nil
}

// DeleteSessionByID deletes a session by ID
func (r *UserRepository) DeleteSessionByID(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.NewDelete().
		Model((*models.SessionModel)(nil)).
		Where("id = ?", id).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to delete session by ID: %w", err)
	}
	return nil
}

// DeleteSessionsByUserID deletes all sessions for a user
func (r *UserRepository) DeleteSessionsByUserID(ctx context.Context, userID uuid.UUID) error {
	_, err := r.db.NewDelete().
		Model((*models.SessionModel)(nil)).
		Where("user_id = ?", userID).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to delete sessions by user ID: %w", err)
	}
	return nil
}

// DeleteExpiredSessions deletes all expired sessions
func (r *UserRepository) DeleteExpiredSessions(ctx context.Context) (int64, error) {
	res, err := r.db.NewDelete().
		Model((*models.SessionModel)(nil)).
		Where("expires_at < ?", time.Now()).
		Exec(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to delete expired sessions: %w", err)
	}
	count, _ := res.RowsAffected()
	return count, nil
}

// UpdateSessionActivity updates the last activity timestamp
func (r *UserRepository) UpdateSessionActivity(ctx context.Context, token string) error {
	_, err := r.db.NewUpdate().
		Model((*models.SessionModel)(nil)).
		Set("last_activity_at = ?", time.Now()).
		Where("token = ?", token).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to update session activity: %w", err)
	}
	return nil
}

// ============================================================================
// Role Operations
// ============================================================================

// FindRoleByID retrieves a role by ID
func (r *UserRepository) FindRoleByID(ctx context.Context, id uuid.UUID) (*models.RoleModel, error) {
	role := &models.RoleModel{}
	err := r.db.NewSelect().
		Model(role).
		Where("id = ?", id).
		Scan(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to find role by ID: %w", err)
	}
	return role, nil
}

// FindRoleByName retrieves a role by name
func (r *UserRepository) FindRoleByName(ctx context.Context, name string) (*models.RoleModel, error) {
	role := &models.RoleModel{}
	err := r.db.NewSelect().
		Model(role).
		Where("LOWER(name) = LOWER(?)", name).
		Scan(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to find role by name: %w", err)
	}
	return role, nil
}

// FindAllRoles retrieves all roles
func (r *UserRepository) FindAllRoles(ctx context.Context) ([]*models.RoleModel, error) {
	var roles []*models.RoleModel
	err := r.db.NewSelect().
		Model(&roles).
		Order("name ASC").
		Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to find all roles: %w", err)
	}
	return roles, nil
}

// CreateRole creates a new role
func (r *UserRepository) CreateRole(ctx context.Context, role *models.RoleModel) error {
	if role.ID == uuid.Nil {
		role.ID = uuid.New()
	}
	now := time.Now()
	role.CreatedAt = now
	role.UpdatedAt = now

	_, err := r.db.NewInsert().Model(role).Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to create role: %w", err)
	}
	return nil
}

// UpdateRole updates an existing role
func (r *UserRepository) UpdateRole(ctx context.Context, role *models.RoleModel) error {
	role.UpdatedAt = time.Now()

	_, err := r.db.NewUpdate().
		Model(role).
		Column("name", "description", "permissions", "metadata", "updated_at").
		Where("id = ?", role.ID).
		Where("is_system = ?", false). // Prevent updating system roles
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to update role: %w", err)
	}
	return nil
}

// DeleteRole deletes a role (only non-system roles)
func (r *UserRepository) DeleteRole(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.NewDelete().
		Model((*models.RoleModel)(nil)).
		Where("id = ?", id).
		Where("is_system = ?", false). // Prevent deleting system roles
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to delete role: %w", err)
	}
	return nil
}

// ============================================================================
// User-Role Associations
// ============================================================================

// AssignRole assigns a role to a user
func (r *UserRepository) AssignRole(ctx context.Context, userID, roleID uuid.UUID, assignedBy *uuid.UUID) error {
	userRole := &models.UserRoleModel{
		UserID:     userID,
		RoleID:     roleID,
		AssignedAt: time.Now(),
		AssignedBy: assignedBy,
	}

	_, err := r.db.NewInsert().
		Model(userRole).
		On("CONFLICT (user_id, role_id) DO NOTHING").
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to assign role: %w", err)
	}
	return nil
}

// RemoveRole removes a role from a user
func (r *UserRepository) RemoveRole(ctx context.Context, userID, roleID uuid.UUID) error {
	_, err := r.db.NewDelete().
		Model((*models.UserRoleModel)(nil)).
		Where("user_id = ?", userID).
		Where("role_id = ?", roleID).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to remove role: %w", err)
	}
	return nil
}

// GetUserRoles retrieves all roles for a user
func (r *UserRepository) GetUserRoles(ctx context.Context, userID uuid.UUID) ([]*models.RoleModel, error) {
	var roles []*models.RoleModel
	err := r.db.NewSelect().
		Model(&roles).
		Join("JOIN mbflow_user_roles ur ON ur.role_id = r.id").
		Where("ur.user_id = ?", userID).
		Order("r.name ASC").
		Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get user roles: %w", err)
	}
	return roles, nil
}

// GetUserPermissions retrieves all unique permissions for a user
func (r *UserRepository) GetUserPermissions(ctx context.Context, userID uuid.UUID) ([]string, error) {
	// First check if user is admin
	user, err := r.FindByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, nil
	}

	// Admins have all permissions implicitly
	if user.IsAdmin {
		return []string{"*"}, nil
	}

	// Get permissions from roles
	var permissions []string
	err = r.db.NewSelect().
		ColumnExpr("DISTINCT unnest(r.permissions) AS permission").
		Table("mbflow_roles").
		TableExpr("r").
		Join("JOIN mbflow_user_roles ur ON ur.role_id = r.id").
		Where("ur.user_id = ?", userID).
		Scan(ctx, &permissions)
	if err != nil {
		return nil, fmt.Errorf("failed to get user permissions: %w", err)
	}
	return permissions, nil
}

// HasPermission checks if a user has a specific permission
func (r *UserRepository) HasPermission(ctx context.Context, userID uuid.UUID, permission string) (bool, error) {
	// First check if user is admin
	user, err := r.FindByID(ctx, userID)
	if err != nil {
		return false, err
	}
	if user == nil {
		return false, nil
	}
	if user.IsAdmin {
		return true, nil
	}

	// Check permission through roles
	exists, err := r.db.NewSelect().
		Table("mbflow_roles").
		TableExpr("r").
		Join("JOIN mbflow_user_roles ur ON ur.role_id = r.id").
		Where("ur.user_id = ?", userID).
		Where("? = ANY(r.permissions)", permission).
		Exists(ctx)
	if err != nil {
		return false, fmt.Errorf("failed to check permission: %w", err)
	}
	return exists, nil
}

// ============================================================================
// Audit Logging
// ============================================================================

// CreateAuditLog creates a new audit log entry
func (r *UserRepository) CreateAuditLog(ctx context.Context, log *models.AuditLogModel) error {
	if log.ID == uuid.Nil {
		log.ID = uuid.New()
	}
	log.CreatedAt = time.Now()

	_, err := r.db.NewInsert().Model(log).Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to create audit log: %w", err)
	}
	return nil
}

// FindAuditLogs retrieves audit logs with optional filtering
func (r *UserRepository) FindAuditLogs(ctx context.Context, userID *uuid.UUID, action string, limit, offset int) ([]*models.AuditLogModel, error) {
	var logs []*models.AuditLogModel
	query := r.db.NewSelect().
		Model(&logs).
		Order("created_at DESC").
		Limit(limit).
		Offset(offset)

	if userID != nil {
		query = query.Where("user_id = ?", *userID)
	}
	if action != "" {
		query = query.Where("action = ?", action)
	}

	err := query.Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to find audit logs: %w", err)
	}
	return logs, nil
}
