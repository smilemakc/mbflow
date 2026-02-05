package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/uptrace/bun"

	pkgmodels "github.com/smilemakc/mbflow/pkg/models"
)

// SystemKeyModel represents system_keys table
type SystemKeyModel struct {
	bun.BaseModel `bun:"table:mbflow_system_keys,alias:syk"`

	ID          uuid.UUID  `bun:"id,pk,type:uuid,default:gen_random_uuid()"`
	Name        string     `bun:"name,notnull"`
	Description string     `bun:"description"`
	KeyPrefix   string     `bun:"key_prefix,notnull,unique"`
	KeyHash     string     `bun:"key_hash,notnull"`
	ServiceName string     `bun:"service_name,notnull"`
	Status      string     `bun:"status,notnull,default:'active'"`
	LastUsedAt  *time.Time `bun:"last_used_at"`
	UsageCount  int64      `bun:"usage_count,notnull,default:0"`
	ExpiresAt   *time.Time `bun:"expires_at"`
	CreatedBy   uuid.UUID  `bun:"created_by,notnull,type:uuid"`
	CreatedAt   time.Time  `bun:"created_at,notnull,default:current_timestamp"`
	UpdatedAt   time.Time  `bun:"updated_at,notnull,default:current_timestamp"`
	RevokedAt   *time.Time `bun:"revoked_at"`
}

// TableName returns the table name for SystemKeyModel
func (SystemKeyModel) TableName() string {
	return "mbflow_system_keys"
}

// BeforeInsert hook to set timestamps and defaults
func (s *SystemKeyModel) BeforeInsert(ctx interface{}) error {
	now := time.Now()
	s.CreatedAt = now
	s.UpdatedAt = now
	if s.ID == uuid.Nil {
		s.ID = uuid.New()
	}
	if s.Status == "" {
		s.Status = pkgmodels.SystemKeyStatusActive
	}
	return nil
}

// BeforeUpdate hook to update timestamp
func (s *SystemKeyModel) BeforeUpdate(ctx interface{}) error {
	s.UpdatedAt = time.Now()
	return nil
}

// ToSystemKeyDomain converts DB model to domain model
func (s *SystemKeyModel) ToSystemKeyDomain() *pkgmodels.SystemKey {
	if s == nil {
		return nil
	}

	return &pkgmodels.SystemKey{
		ID:          s.ID.String(),
		Name:        s.Name,
		Description: s.Description,
		KeyPrefix:   s.KeyPrefix,
		KeyHash:     s.KeyHash,
		ServiceName: s.ServiceName,
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

// FromSystemKeyDomain creates DB model from domain model
func FromSystemKeyDomain(key *pkgmodels.SystemKey) *SystemKeyModel {
	if key == nil {
		return nil
	}

	var id uuid.UUID
	if key.ID != "" {
		id = uuid.MustParse(key.ID)
	}

	return &SystemKeyModel{
		ID:          id,
		Name:        key.Name,
		Description: key.Description,
		KeyPrefix:   key.KeyPrefix,
		KeyHash:     key.KeyHash,
		ServiceName: key.ServiceName,
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
