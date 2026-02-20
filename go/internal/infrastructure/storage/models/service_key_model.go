package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/uptrace/bun"

	pkgmodels "github.com/smilemakc/mbflow/go/pkg/models"
)

// ServiceKeyModel represents service_keys table
type ServiceKeyModel struct {
	bun.BaseModel `bun:"table:mbflow_service_keys,alias:sk"`

	ID          uuid.UUID  `bun:"id,pk,type:uuid,default:gen_random_uuid()"`
	UserID      uuid.UUID  `bun:"user_id,notnull,type:uuid"`
	Name        string     `bun:"name,notnull"`
	Description string     `bun:"description"`
	KeyPrefix   string     `bun:"key_prefix,notnull,unique"`
	KeyHash     string     `bun:"key_hash,notnull"`
	Status      string     `bun:"status,notnull,default:'active'"`
	LastUsedAt  *time.Time `bun:"last_used_at"`
	UsageCount  int64      `bun:"usage_count,notnull,default:0"`
	ExpiresAt   *time.Time `bun:"expires_at"`
	CreatedBy   uuid.UUID  `bun:"created_by,notnull,type:uuid"`
	CreatedAt   time.Time  `bun:"created_at,notnull,default:current_timestamp"`
	UpdatedAt   time.Time  `bun:"updated_at,notnull,default:current_timestamp"`
	RevokedAt   *time.Time `bun:"revoked_at"`

	// Relations
	User    *UserModel `bun:"rel:belongs-to,join:user_id=id"`
	Creator *UserModel `bun:"rel:belongs-to,join:created_by=id"`
}

// TableName returns the table name for ServiceKeyModel
func (ServiceKeyModel) TableName() string {
	return "mbflow_service_keys"
}

// BeforeInsert hook to set timestamps and defaults
func (s *ServiceKeyModel) BeforeInsert(ctx any) error {
	now := time.Now()
	s.CreatedAt = now
	s.UpdatedAt = now
	if s.ID == uuid.Nil {
		s.ID = uuid.New()
	}
	if s.Status == "" {
		s.Status = pkgmodels.ServiceKeyStatusActive
	}
	return nil
}

// BeforeUpdate hook to update timestamp
func (s *ServiceKeyModel) BeforeUpdate(ctx any) error {
	s.UpdatedAt = time.Now()
	return nil
}

// ToServiceKeyDomain converts DB model to domain model
func (s *ServiceKeyModel) ToServiceKeyDomain() *pkgmodels.ServiceKey {
	if s == nil {
		return nil
	}

	return &pkgmodels.ServiceKey{
		ID:          s.ID.String(),
		UserID:      s.UserID.String(),
		Name:        s.Name,
		Description: s.Description,
		KeyPrefix:   s.KeyPrefix,
		KeyHash:     s.KeyHash,
		Status:      s.Status,
		LastUsedAt:  s.LastUsedAt,
		UsageCount:  s.UsageCount,
		ExpiresAt:   s.ExpiresAt,
		CreatedBy:   s.CreatedBy.String(),
		CreatedAt:   s.CreatedAt,
		UpdatedAt:   s.UpdatedAt,
		RevokedAt:   s.RevokedAt,
	}
}

// FromServiceKeyDomain creates DB model from domain model
func FromServiceKeyDomain(key *pkgmodels.ServiceKey) *ServiceKeyModel {
	if key == nil {
		return nil
	}

	var id uuid.UUID
	if key.ID != "" {
		id = uuid.MustParse(key.ID)
	}

	return &ServiceKeyModel{
		ID:          id,
		UserID:      uuid.MustParse(key.UserID),
		Name:        key.Name,
		Description: key.Description,
		KeyPrefix:   key.KeyPrefix,
		KeyHash:     key.KeyHash,
		Status:      key.Status,
		LastUsedAt:  key.LastUsedAt,
		UsageCount:  key.UsageCount,
		ExpiresAt:   key.ExpiresAt,
		CreatedBy:   uuid.MustParse(key.CreatedBy),
		CreatedAt:   key.CreatedAt,
		UpdatedAt:   key.UpdatedAt,
		RevokedAt:   key.RevokedAt,
	}
}
