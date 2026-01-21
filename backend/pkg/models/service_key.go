package models

import (
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
)

const (
	ServiceKeyStatusActive  = "active"
	ServiceKeyStatusRevoked = "revoked"
)

const (
	ServiceKeyPrefixLength = 8
	ServiceKeyPrefix       = "sk_"
)

var (
	ErrServiceKeyNotFound     = errors.New("service key not found")
	ErrServiceKeyRevoked      = errors.New("service key has been revoked")
	ErrServiceKeyExpired      = errors.New("service key has expired")
	ErrInvalidServiceKey      = errors.New("invalid service key")
	ErrServiceKeyLimitReached = errors.New("service key limit reached for user")
)

type ServiceKey struct {
	ID          string     `json:"id"`
	UserID      string     `json:"user_id"`
	Name        string     `json:"name"`
	Description string     `json:"description,omitempty"`
	KeyPrefix   string     `json:"key_prefix"`
	KeyHash     string     `json:"-"`
	Status      string     `json:"status"`
	LastUsedAt  *time.Time `json:"last_used_at,omitempty"`
	UsageCount  int64      `json:"usage_count"`
	ExpiresAt   *time.Time `json:"expires_at,omitempty"`
	CreatedBy   string     `json:"created_by"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	RevokedAt   *time.Time `json:"revoked_at,omitempty"`
}

func NewServiceKey(userID, name, description, createdBy string) *ServiceKey {
	now := time.Now()
	id := uuid.New().String()

	keyPrefix := generateKeyPrefix()

	return &ServiceKey{
		ID:          id,
		UserID:      userID,
		Name:        name,
		Description: description,
		KeyPrefix:   keyPrefix,
		Status:      ServiceKeyStatusActive,
		UsageCount:  0,
		CreatedBy:   createdBy,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
}

func (k *ServiceKey) Validate() error {
	if k.UserID == "" {
		return &ValidationError{Field: "user_id", Message: "user ID is required"}
	}

	if k.Name == "" {
		return &ValidationError{Field: "name", Message: "name is required"}
	}

	if len(k.Name) > 255 {
		return &ValidationError{Field: "name", Message: "name must be 255 characters or less"}
	}

	if k.KeyPrefix == "" {
		return &ValidationError{Field: "key_prefix", Message: "key prefix is required"}
	}

	if !strings.HasPrefix(k.KeyPrefix, ServiceKeyPrefix) {
		return &ValidationError{Field: "key_prefix", Message: "key prefix must start with 'sk_'"}
	}

	if k.Status != ServiceKeyStatusActive && k.Status != ServiceKeyStatusRevoked {
		return &ValidationError{Field: "status", Message: "status must be 'active' or 'revoked'"}
	}

	if k.CreatedBy == "" {
		return &ValidationError{Field: "created_by", Message: "created by is required"}
	}

	if k.ExpiresAt != nil && k.ExpiresAt.Before(k.CreatedAt) {
		return &ValidationError{Field: "expires_at", Message: "expiration date cannot be before creation date"}
	}

	return nil
}

func (k *ServiceKey) IsActive() bool {
	if k.Status != ServiceKeyStatusActive {
		return false
	}

	if k.IsExpired() {
		return false
	}

	return true
}

func (k *ServiceKey) IsExpired() bool {
	if k.ExpiresAt == nil {
		return false
	}
	return time.Now().After(*k.ExpiresAt)
}

func (k *ServiceKey) Revoke() {
	now := time.Now()
	k.Status = ServiceKeyStatusRevoked
	k.RevokedAt = &now
	k.UpdatedAt = now
}

func (k *ServiceKey) IncrementUsage() {
	now := time.Now()
	k.UsageCount++
	k.LastUsedAt = &now
	k.UpdatedAt = now
}

func (k *ServiceKey) SetExpiration(expiresAt time.Time) error {
	// Check if expiration is in the past first (more user-friendly error)
	if expiresAt.Before(time.Now()) {
		return &ValidationError{Field: "expires_at", Message: "expiration date cannot be in the past"}
	}

	// Then check if it's before creation date (shouldn't happen normally)
	if expiresAt.Before(k.CreatedAt) {
		return &ValidationError{Field: "expires_at", Message: "expiration date cannot be before creation date"}
	}

	k.ExpiresAt = &expiresAt
	k.UpdatedAt = time.Now()
	return nil
}

func (k *ServiceKey) ClearExpiration() {
	k.ExpiresAt = nil
	k.UpdatedAt = time.Now()
}

func (k *ServiceKey) CanUse() error {
	if k.Status == ServiceKeyStatusRevoked {
		return ErrServiceKeyRevoked
	}

	if k.IsExpired() {
		return ErrServiceKeyExpired
	}

	return nil
}

func generateKeyPrefix() string {
	randomPart := uuid.New().String()
	randomPart = strings.ReplaceAll(randomPart, "-", "")

	if len(randomPart) > ServiceKeyPrefixLength-len(ServiceKeyPrefix) {
		randomPart = randomPart[:ServiceKeyPrefixLength-len(ServiceKeyPrefix)]
	}

	return ServiceKeyPrefix + randomPart
}
