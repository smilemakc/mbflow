package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/smilemakc/mbflow/go/internal/infrastructure/storage/models"
)

// UserRepository defines the interface for user persistence
type UserRepository interface {
	// User CRUD operations
	Create(ctx context.Context, user *models.UserModel) error
	Update(ctx context.Context, user *models.UserModel) error
	Delete(ctx context.Context, id uuid.UUID) error
	HardDelete(ctx context.Context, id uuid.UUID) error

	// User lookup operations
	FindByID(ctx context.Context, id uuid.UUID) (*models.UserModel, error)
	FindByEmail(ctx context.Context, email string) (*models.UserModel, error)
	FindByUsername(ctx context.Context, username string) (*models.UserModel, error)
	FindByIDWithRoles(ctx context.Context, id uuid.UUID) (*models.UserModel, error)
	FindAll(ctx context.Context, limit, offset int) ([]*models.UserModel, error)
	FindAllActive(ctx context.Context, limit, offset int) ([]*models.UserModel, error)

	// Existence checks
	ExistsByEmail(ctx context.Context, email string) (bool, error)
	ExistsByUsername(ctx context.Context, username string) (bool, error)

	// Counting
	Count(ctx context.Context) (int, error)
	CountActive(ctx context.Context) (int, error)

	// Login tracking
	UpdateLastLogin(ctx context.Context, id uuid.UUID) error
	IncrementFailedAttempts(ctx context.Context, id uuid.UUID) error
	ResetFailedAttempts(ctx context.Context, id uuid.UUID) error
	LockAccount(ctx context.Context, id uuid.UUID, until *string) error
	UnlockAccount(ctx context.Context, id uuid.UUID) error

	// Session operations
	CreateSession(ctx context.Context, session *models.SessionModel) error
	FindSessionByToken(ctx context.Context, token string) (*models.SessionModel, error)
	FindSessionByRefreshToken(ctx context.Context, refreshToken string) (*models.SessionModel, error)
	FindSessionsByUserID(ctx context.Context, userID uuid.UUID) ([]*models.SessionModel, error)
	DeleteSession(ctx context.Context, token string) error
	DeleteSessionByID(ctx context.Context, id uuid.UUID) error
	DeleteSessionsByUserID(ctx context.Context, userID uuid.UUID) error
	DeleteExpiredSessions(ctx context.Context) (int64, error)
	UpdateSessionActivity(ctx context.Context, token string) error

	// Role operations
	FindRoleByID(ctx context.Context, id uuid.UUID) (*models.RoleModel, error)
	FindRoleByName(ctx context.Context, name string) (*models.RoleModel, error)
	FindAllRoles(ctx context.Context) ([]*models.RoleModel, error)
	CreateRole(ctx context.Context, role *models.RoleModel) error
	UpdateRole(ctx context.Context, role *models.RoleModel) error
	DeleteRole(ctx context.Context, id uuid.UUID) error

	// User-Role associations
	AssignRole(ctx context.Context, userID, roleID uuid.UUID, assignedBy *uuid.UUID) error
	RemoveRole(ctx context.Context, userID, roleID uuid.UUID) error
	GetUserRoles(ctx context.Context, userID uuid.UUID) ([]*models.RoleModel, error)
	GetUserPermissions(ctx context.Context, userID uuid.UUID) ([]string, error)
	HasPermission(ctx context.Context, userID uuid.UUID, permission string) (bool, error)

	// Audit logging
	CreateAuditLog(ctx context.Context, log *models.AuditLogModel) error
	FindAuditLogs(ctx context.Context, userID *uuid.UUID, action string, limit, offset int) ([]*models.AuditLogModel, error)
}
