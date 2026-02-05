package models

import (
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
)

const (
	SystemKeyStatusActive  = "active"
	SystemKeyStatusRevoked = "revoked"

	SystemKeyPrefixLength = 10
	SystemKeyPrefix       = "sysk_"
)

var (
	ErrSystemKeyNotFound     = errors.New("system key not found")
	ErrSystemKeyRevoked      = errors.New("system key has been revoked")
	ErrSystemKeyExpired      = errors.New("system key has expired")
	ErrInvalidSystemKey      = errors.New("invalid system key")
	ErrSystemKeyLimitReached = errors.New("system key limit reached")
)

type SystemKey struct {
	ID          string     `json:"id"`
	Name        string     `json:"name"`
	Description string     `json:"description,omitempty"`
	KeyPrefix   string     `json:"key_prefix"`
	KeyHash     string     `json:"-"`
	ServiceName string     `json:"service_name"`
	Status      string     `json:"status"`
	LastUsedAt  *time.Time `json:"last_used_at,omitempty"`
	UsageCount  int64      `json:"usage_count"`
	ExpiresAt   *time.Time `json:"expires_at,omitempty"`
	CreatedBy   string     `json:"created_by"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	RevokedAt   *time.Time `json:"revoked_at,omitempty"`
}

func NewSystemKey(name, description, serviceName, createdBy string) *SystemKey {
	now := time.Now()
	return &SystemKey{
		ID:          uuid.New().String(),
		Name:        name,
		Description: description,
		ServiceName: serviceName,
		Status:      SystemKeyStatusActive,
		UsageCount:  0,
		CreatedBy:   createdBy,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
}

func (k *SystemKey) Validate() error {
	if strings.TrimSpace(k.Name) == "" {
		return &ValidationError{Field: "name", Message: "name is required"}
	}
	if len(k.Name) > 255 {
		return &ValidationError{Field: "name", Message: "name must be 255 characters or less"}
	}
	if strings.TrimSpace(k.ServiceName) == "" {
		return &ValidationError{Field: "service_name", Message: "service name is required"}
	}
	if len(k.ServiceName) > 100 {
		return &ValidationError{Field: "service_name", Message: "service name must be 100 characters or less"}
	}
	if k.CreatedBy == "" {
		return &ValidationError{Field: "created_by", Message: "created_by is required"}
	}
	return nil
}

func (k *SystemKey) IsActive() bool {
	if k.Status != SystemKeyStatusActive {
		return false
	}
	if k.ExpiresAt != nil && time.Now().After(*k.ExpiresAt) {
		return false
	}
	return true
}

func (k *SystemKey) IsExpired() bool {
	return k.ExpiresAt != nil && time.Now().After(*k.ExpiresAt)
}

func (k *SystemKey) CanUse() error {
	if k.Status == SystemKeyStatusRevoked {
		return ErrSystemKeyRevoked
	}
	if k.IsExpired() {
		return ErrSystemKeyExpired
	}
	return nil
}

func (k *SystemKey) Revoke() {
	now := time.Now()
	k.Status = SystemKeyStatusRevoked
	k.RevokedAt = &now
	k.UpdatedAt = now
}

func (k *SystemKey) IncrementUsage() {
	now := time.Now()
	k.LastUsedAt = &now
	k.UsageCount++
}

func (k *SystemKey) SetExpiration(expiresAt time.Time) error {
	if expiresAt.Before(time.Now()) {
		return &ValidationError{Field: "expires_at", Message: "expiration must be in the future"}
	}
	k.ExpiresAt = &expiresAt
	k.UpdatedAt = time.Now()
	return nil
}
