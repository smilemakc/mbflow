package models

import (
	"strings"
	"testing"
	"time"
)

func TestNewServiceKey(t *testing.T) {
	userID := "user-123"
	name := "Production API Key"
	description := "Key for production environment"
	createdBy := "admin-456"

	key := NewServiceKey(userID, name, description, createdBy)

	if key.UserID != userID {
		t.Errorf("UserID = %q, want %q", key.UserID, userID)
	}
	if key.Name != name {
		t.Errorf("Name = %q, want %q", key.Name, name)
	}
	if key.Description != description {
		t.Errorf("Description = %q, want %q", key.Description, description)
	}
	if key.CreatedBy != createdBy {
		t.Errorf("CreatedBy = %q, want %q", key.CreatedBy, createdBy)
	}
	if key.Status != ServiceKeyStatusActive {
		t.Errorf("Status = %q, want %q", key.Status, ServiceKeyStatusActive)
	}
	if key.UsageCount != 0 {
		t.Errorf("UsageCount = %d, want 0", key.UsageCount)
	}
	if key.ID == "" {
		t.Error("ID should not be empty")
	}
	if key.KeyPrefix == "" {
		t.Error("KeyPrefix should not be empty")
	}
	if !strings.HasPrefix(key.KeyPrefix, ServiceKeyPrefix) {
		t.Errorf("KeyPrefix = %q, should start with %q", key.KeyPrefix, ServiceKeyPrefix)
	}
	if key.LastUsedAt != nil {
		t.Error("LastUsedAt should be nil for new key")
	}
	if key.ExpiresAt != nil {
		t.Error("ExpiresAt should be nil for new key")
	}
	if key.RevokedAt != nil {
		t.Error("RevokedAt should be nil for new key")
	}
}

func TestServiceKey_Validate(t *testing.T) {
	tests := []struct {
		name    string
		setup   func() *ServiceKey
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid service key",
			setup: func() *ServiceKey {
				return NewServiceKey("user-123", "My Key", "Description", "admin-456")
			},
			wantErr: false,
		},
		{
			name: "missing user_id",
			setup: func() *ServiceKey {
				key := NewServiceKey("user-123", "My Key", "Description", "admin-456")
				key.UserID = ""
				return key
			},
			wantErr: true,
			errMsg:  "user ID is required",
		},
		{
			name: "missing name",
			setup: func() *ServiceKey {
				key := NewServiceKey("user-123", "My Key", "Description", "admin-456")
				key.Name = ""
				return key
			},
			wantErr: true,
			errMsg:  "name is required",
		},
		{
			name: "name too long",
			setup: func() *ServiceKey {
				key := NewServiceKey("user-123", "My Key", "Description", "admin-456")
				key.Name = strings.Repeat("a", 256)
				return key
			},
			wantErr: true,
			errMsg:  "name must be 255 characters or less",
		},
		{
			name: "missing key_prefix",
			setup: func() *ServiceKey {
				key := NewServiceKey("user-123", "My Key", "Description", "admin-456")
				key.KeyPrefix = ""
				return key
			},
			wantErr: true,
			errMsg:  "key prefix is required",
		},
		{
			name: "invalid key_prefix",
			setup: func() *ServiceKey {
				key := NewServiceKey("user-123", "My Key", "Description", "admin-456")
				key.KeyPrefix = "invalid_prefix"
				return key
			},
			wantErr: true,
			errMsg:  "key prefix must start with 'sk_'",
		},
		{
			name: "invalid status",
			setup: func() *ServiceKey {
				key := NewServiceKey("user-123", "My Key", "Description", "admin-456")
				key.Status = "invalid"
				return key
			},
			wantErr: true,
			errMsg:  "status must be 'active' or 'revoked'",
		},
		{
			name: "missing created_by",
			setup: func() *ServiceKey {
				key := NewServiceKey("user-123", "My Key", "Description", "admin-456")
				key.CreatedBy = ""
				return key
			},
			wantErr: true,
			errMsg:  "created by is required",
		},
		{
			name: "expires_at before created_at",
			setup: func() *ServiceKey {
				key := NewServiceKey("user-123", "My Key", "Description", "admin-456")
				past := key.CreatedAt.Add(-1 * time.Hour)
				key.ExpiresAt = &past
				return key
			},
			wantErr: true,
			errMsg:  "expiration date cannot be before creation date",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key := tt.setup()
			err := key.Validate()

			if tt.wantErr {
				if err == nil {
					t.Errorf("Validate() error = nil, want error containing %q", tt.errMsg)
					return
				}
				if !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("Validate() error = %q, want error containing %q", err.Error(), tt.errMsg)
				}
			} else {
				if err != nil {
					t.Errorf("Validate() error = %v, want nil", err)
				}
			}
		})
	}
}

func TestServiceKey_IsActive(t *testing.T) {
	tests := []struct {
		name  string
		setup func() *ServiceKey
		want  bool
	}{
		{
			name: "active key without expiration",
			setup: func() *ServiceKey {
				return NewServiceKey("user-123", "My Key", "Description", "admin-456")
			},
			want: true,
		},
		{
			name: "revoked key",
			setup: func() *ServiceKey {
				key := NewServiceKey("user-123", "My Key", "Description", "admin-456")
				key.Status = ServiceKeyStatusRevoked
				return key
			},
			want: false,
		},
		{
			name: "expired key",
			setup: func() *ServiceKey {
				key := NewServiceKey("user-123", "My Key", "Description", "admin-456")
				past := time.Now().Add(-1 * time.Hour)
				key.ExpiresAt = &past
				return key
			},
			want: false,
		},
		{
			name: "active key with future expiration",
			setup: func() *ServiceKey {
				key := NewServiceKey("user-123", "My Key", "Description", "admin-456")
				future := time.Now().Add(24 * time.Hour)
				key.ExpiresAt = &future
				return key
			},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key := tt.setup()
			if got := key.IsActive(); got != tt.want {
				t.Errorf("IsActive() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestServiceKey_IsExpired(t *testing.T) {
	tests := []struct {
		name  string
		setup func() *ServiceKey
		want  bool
	}{
		{
			name: "no expiration",
			setup: func() *ServiceKey {
				return NewServiceKey("user-123", "My Key", "Description", "admin-456")
			},
			want: false,
		},
		{
			name: "future expiration",
			setup: func() *ServiceKey {
				key := NewServiceKey("user-123", "My Key", "Description", "admin-456")
				future := time.Now().Add(24 * time.Hour)
				key.ExpiresAt = &future
				return key
			},
			want: false,
		},
		{
			name: "past expiration",
			setup: func() *ServiceKey {
				key := NewServiceKey("user-123", "My Key", "Description", "admin-456")
				past := time.Now().Add(-1 * time.Hour)
				key.ExpiresAt = &past
				return key
			},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key := tt.setup()
			if got := key.IsExpired(); got != tt.want {
				t.Errorf("IsExpired() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestServiceKey_Revoke(t *testing.T) {
	key := NewServiceKey("user-123", "My Key", "Description", "admin-456")
	originalUpdatedAt := key.UpdatedAt

	time.Sleep(10 * time.Millisecond)
	key.Revoke()

	if key.Status != ServiceKeyStatusRevoked {
		t.Errorf("Status = %q, want %q", key.Status, ServiceKeyStatusRevoked)
	}
	if key.RevokedAt == nil {
		t.Error("RevokedAt should not be nil after revoke")
	}
	if !key.UpdatedAt.After(originalUpdatedAt) {
		t.Error("UpdatedAt should be updated after revoke")
	}
	if key.IsActive() {
		t.Error("IsActive() should return false after revoke")
	}
}

func TestServiceKey_IncrementUsage(t *testing.T) {
	key := NewServiceKey("user-123", "My Key", "Description", "admin-456")
	originalUpdatedAt := key.UpdatedAt

	if key.UsageCount != 0 {
		t.Errorf("Initial UsageCount = %d, want 0", key.UsageCount)
	}
	if key.LastUsedAt != nil {
		t.Error("Initial LastUsedAt should be nil")
	}

	time.Sleep(10 * time.Millisecond)
	key.IncrementUsage()

	if key.UsageCount != 1 {
		t.Errorf("UsageCount = %d, want 1", key.UsageCount)
	}
	if key.LastUsedAt == nil {
		t.Error("LastUsedAt should not be nil after increment")
	}
	if !key.UpdatedAt.After(originalUpdatedAt) {
		t.Error("UpdatedAt should be updated after increment")
	}

	key.IncrementUsage()
	if key.UsageCount != 2 {
		t.Errorf("UsageCount = %d, want 2", key.UsageCount)
	}
}

func TestServiceKey_SetExpiration(t *testing.T) {
	t.Run("valid future expiration", func(t *testing.T) {
		key := NewServiceKey("user-123", "My Key", "Description", "admin-456")
		originalUpdatedAt := key.UpdatedAt
		expiresAt := time.Now().Add(24 * time.Hour)

		time.Sleep(10 * time.Millisecond)
		err := key.SetExpiration(expiresAt)

		if err != nil {
			t.Errorf("SetExpiration() error = %v, want nil", err)
			return
		}
		if key.ExpiresAt == nil {
			t.Error("ExpiresAt should not be nil after SetExpiration")
			return
		}
		if !key.ExpiresAt.Equal(expiresAt) {
			t.Errorf("ExpiresAt = %v, want %v", key.ExpiresAt, expiresAt)
		}
		if !key.UpdatedAt.After(originalUpdatedAt) {
			t.Error("UpdatedAt should be updated after SetExpiration")
		}
	})

	t.Run("expiration before creation", func(t *testing.T) {
		// Create key with CreatedAt in the future to test this case independently
		key := NewServiceKey("user-123", "My Key", "Description", "admin-456")
		key.CreatedAt = time.Now().Add(48 * time.Hour) // CreatedAt in the future
		// Try to set expiration to a time after now but before CreatedAt
		expiresAt := time.Now().Add(24 * time.Hour)

		err := key.SetExpiration(expiresAt)

		if err == nil {
			t.Error("SetExpiration() error = nil, want error containing 'expiration date cannot be before creation date'")
			return
		}
		if !strings.Contains(err.Error(), "expiration date cannot be before creation date") {
			t.Errorf("SetExpiration() error = %q, want error containing %q", err.Error(), "expiration date cannot be before creation date")
		}
	})

	t.Run("expiration in the past", func(t *testing.T) {
		key := NewServiceKey("user-123", "My Key", "Description", "admin-456")
		expiresAt := time.Now().Add(-1 * time.Hour)

		err := key.SetExpiration(expiresAt)

		if err == nil {
			t.Error("SetExpiration() error = nil, want error containing 'expiration date cannot be in the past'")
			return
		}
		if !strings.Contains(err.Error(), "expiration date cannot be in the past") {
			t.Errorf("SetExpiration() error = %q, want error containing %q", err.Error(), "expiration date cannot be in the past")
		}
	})
}

func TestServiceKey_ClearExpiration(t *testing.T) {
	key := NewServiceKey("user-123", "My Key", "Description", "admin-456")
	future := time.Now().Add(24 * time.Hour)
	key.ExpiresAt = &future
	originalUpdatedAt := key.UpdatedAt

	time.Sleep(10 * time.Millisecond)
	key.ClearExpiration()

	if key.ExpiresAt != nil {
		t.Error("ExpiresAt should be nil after ClearExpiration")
	}
	if !key.UpdatedAt.After(originalUpdatedAt) {
		t.Error("UpdatedAt should be updated after ClearExpiration")
	}
}

func TestServiceKey_CanUse(t *testing.T) {
	tests := []struct {
		name    string
		setup   func() *ServiceKey
		wantErr error
	}{
		{
			name: "active key can be used",
			setup: func() *ServiceKey {
				return NewServiceKey("user-123", "My Key", "Description", "admin-456")
			},
			wantErr: nil,
		},
		{
			name: "revoked key cannot be used",
			setup: func() *ServiceKey {
				key := NewServiceKey("user-123", "My Key", "Description", "admin-456")
				key.Revoke()
				return key
			},
			wantErr: ErrServiceKeyRevoked,
		},
		{
			name: "expired key cannot be used",
			setup: func() *ServiceKey {
				key := NewServiceKey("user-123", "My Key", "Description", "admin-456")
				past := time.Now().Add(-1 * time.Hour)
				key.ExpiresAt = &past
				return key
			},
			wantErr: ErrServiceKeyExpired,
		},
		{
			name: "active key with future expiration can be used",
			setup: func() *ServiceKey {
				key := NewServiceKey("user-123", "My Key", "Description", "admin-456")
				future := time.Now().Add(24 * time.Hour)
				key.ExpiresAt = &future
				return key
			},
			wantErr: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key := tt.setup()
			err := key.CanUse()

			if tt.wantErr != nil {
				if err == nil {
					t.Errorf("CanUse() error = nil, want %v", tt.wantErr)
					return
				}
				if err != tt.wantErr {
					t.Errorf("CanUse() error = %v, want %v", err, tt.wantErr)
				}
			} else {
				if err != nil {
					t.Errorf("CanUse() error = %v, want nil", err)
				}
			}
		})
	}
}

func TestGenerateKeyPrefix(t *testing.T) {
	prefix1 := generateKeyPrefix()
	prefix2 := generateKeyPrefix()

	if !strings.HasPrefix(prefix1, ServiceKeyPrefix) {
		t.Errorf("prefix1 = %q, should start with %q", prefix1, ServiceKeyPrefix)
	}
	if !strings.HasPrefix(prefix2, ServiceKeyPrefix) {
		t.Errorf("prefix2 = %q, should start with %q", prefix2, ServiceKeyPrefix)
	}
	if prefix1 == prefix2 {
		t.Error("generateKeyPrefix() should generate unique prefixes")
	}
	if len(prefix1) != ServiceKeyPrefixLength {
		t.Errorf("len(prefix1) = %d, want %d", len(prefix1), ServiceKeyPrefixLength)
	}
}
