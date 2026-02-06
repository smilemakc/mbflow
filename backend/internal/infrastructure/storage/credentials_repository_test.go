package storage

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"

	"github.com/smilemakc/mbflow/internal/infrastructure/storage/models"
	pkgmodels "github.com/smilemakc/mbflow/pkg/models"
	"github.com/smilemakc/mbflow/testutil"
)

// TestCredentialsRepository_Create tests credential creation
func TestCredentialsRepository_Create(t *testing.T) {
	t.Parallel()
	db, cleanup := setupCredentialsTestDB(t)
	defer cleanup()

	repo := NewCredentialsRepository(db)
	ctx := context.Background()

	// Create test user first
	userID := createCredentialsTestUser(t, db)

	tests := []struct {
		name       string
		credential *pkgmodels.CredentialsResource
		wantErr    bool
	}{
		{
			name: "valid api_key credential",
			credential: &pkgmodels.CredentialsResource{
				BaseResource: pkgmodels.BaseResource{
					Type:        pkgmodels.ResourceTypeCredentials,
					OwnerID:     userID,
					Name:        "Test API Key",
					Description: "Test description",
					Status:      pkgmodels.ResourceStatusActive,
				},
				CredentialType: pkgmodels.CredentialTypeAPIKey,
				EncryptedData:  map[string]string{"api_key": "encrypted-value"},
				Provider:       "openai",
			},
			wantErr: false,
		},
		{
			name: "valid basic_auth credential",
			credential: &pkgmodels.CredentialsResource{
				BaseResource: pkgmodels.BaseResource{
					Type:    pkgmodels.ResourceTypeCredentials,
					OwnerID: userID,
					Name:    "Test Basic Auth",
					Status:  pkgmodels.ResourceStatusActive,
				},
				CredentialType: pkgmodels.CredentialTypeBasicAuth,
				EncryptedData: map[string]string{
					"username": "encrypted-user",
					"password": "encrypted-pass",
				},
			},
			wantErr: false,
		},
		{
			name: "valid oauth2 credential",
			credential: &pkgmodels.CredentialsResource{
				BaseResource: pkgmodels.BaseResource{
					Type:    pkgmodels.ResourceTypeCredentials,
					OwnerID: userID,
					Name:    "Test OAuth2",
					Status:  pkgmodels.ResourceStatusActive,
				},
				CredentialType: pkgmodels.CredentialTypeOAuth2,
				EncryptedData: map[string]string{
					"client_id":     "encrypted-id",
					"client_secret": "encrypted-secret",
					"access_token":  "encrypted-token",
				},
				Provider: "google",
			},
			wantErr: false,
		},
		{
			name: "valid service_account credential",
			credential: &pkgmodels.CredentialsResource{
				BaseResource: pkgmodels.BaseResource{
					Type:    pkgmodels.ResourceTypeCredentials,
					OwnerID: userID,
					Name:    "Test Service Account",
					Status:  pkgmodels.ResourceStatusActive,
				},
				CredentialType: pkgmodels.CredentialTypeServiceAccount,
				EncryptedData:  map[string]string{"json_key": "encrypted-json"},
				Provider:       "gcp",
			},
			wantErr: false,
		},
		{
			name: "valid custom credential",
			credential: &pkgmodels.CredentialsResource{
				BaseResource: pkgmodels.BaseResource{
					Type:    pkgmodels.ResourceTypeCredentials,
					OwnerID: userID,
					Name:    "Test Custom",
					Status:  pkgmodels.ResourceStatusActive,
				},
				CredentialType: pkgmodels.CredentialTypeCustom,
				EncryptedData: map[string]string{
					"field1": "value1",
					"field2": "value2",
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := repo.CreateCredentials(ctx, tt.credential)

			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.NotEmpty(t, tt.credential.ID)
			assert.NotZero(t, tt.credential.CreatedAt)
			assert.NotZero(t, tt.credential.UpdatedAt)

			// Verify it can be retrieved
			retrieved, err := repo.GetCredentials(ctx, tt.credential.ID)
			require.NoError(t, err)
			assert.Equal(t, tt.credential.Name, retrieved.Name)
			assert.Equal(t, tt.credential.CredentialType, retrieved.CredentialType)
			assert.Equal(t, tt.credential.Provider, retrieved.Provider)
		})
	}
}

// TestCredentialsRepository_GetByOwner tests retrieving credentials by owner
func TestCredentialsRepository_GetByOwner(t *testing.T) {
	t.Parallel()
	db, cleanup := setupCredentialsTestDB(t)
	defer cleanup()

	repo := NewCredentialsRepository(db)
	ctx := context.Background()

	userID := createCredentialsTestUser(t, db)

	// Create multiple credentials
	cred1 := &pkgmodels.CredentialsResource{
		BaseResource: pkgmodels.BaseResource{
			Type:    pkgmodels.ResourceTypeCredentials,
			OwnerID: userID,
			Name:    "Credential 1",
			Status:  pkgmodels.ResourceStatusActive,
		},
		CredentialType: pkgmodels.CredentialTypeAPIKey,
		EncryptedData:  map[string]string{"api_key": "key1"},
		Provider:       "openai",
	}

	cred2 := &pkgmodels.CredentialsResource{
		BaseResource: pkgmodels.BaseResource{
			Type:    pkgmodels.ResourceTypeCredentials,
			OwnerID: userID,
			Name:    "Credential 2",
			Status:  pkgmodels.ResourceStatusActive,
		},
		CredentialType: pkgmodels.CredentialTypeAPIKey,
		EncryptedData:  map[string]string{"api_key": "key2"},
		Provider:       "anthropic",
	}

	require.NoError(t, repo.CreateCredentials(ctx, cred1))
	require.NoError(t, repo.CreateCredentials(ctx, cred2))

	// Get all credentials for owner
	credentials, err := repo.GetCredentialsByOwner(ctx, userID)
	require.NoError(t, err)
	assert.Len(t, credentials, 2)

	// Get credentials by provider
	openaiCreds, err := repo.GetCredentialsByProvider(ctx, userID, "openai")
	require.NoError(t, err)
	assert.Len(t, openaiCreds, 1)
	assert.Equal(t, "openai", openaiCreds[0].Provider)
}

// TestCredentialsRepository_Update tests updating credentials
func TestCredentialsRepository_Update(t *testing.T) {
	t.Parallel()
	db, cleanup := setupCredentialsTestDB(t)
	defer cleanup()

	repo := NewCredentialsRepository(db)
	ctx := context.Background()

	userID := createCredentialsTestUser(t, db)

	cred := &pkgmodels.CredentialsResource{
		BaseResource: pkgmodels.BaseResource{
			Type:        pkgmodels.ResourceTypeCredentials,
			OwnerID:     userID,
			Name:        "Original Name",
			Description: "Original Description",
			Status:      pkgmodels.ResourceStatusActive,
		},
		CredentialType: pkgmodels.CredentialTypeAPIKey,
		EncryptedData:  map[string]string{"api_key": "original-key"},
		Provider:       "openai",
	}

	require.NoError(t, repo.CreateCredentials(ctx, cred))

	// Update
	cred.Name = "Updated Name"
	cred.Description = "Updated Description"
	cred.EncryptedData["api_key"] = "updated-key"

	err := repo.UpdateCredentials(ctx, cred)
	require.NoError(t, err)

	// Verify update
	retrieved, err := repo.GetCredentials(ctx, cred.ID)
	require.NoError(t, err)
	assert.Equal(t, "Updated Name", retrieved.Name)
	assert.Equal(t, "Updated Description", retrieved.Description)
	assert.Equal(t, "updated-key", retrieved.EncryptedData["api_key"])
}

// TestCredentialsRepository_Delete tests soft-deleting credentials
func TestCredentialsRepository_Delete(t *testing.T) {
	t.Parallel()
	db, cleanup := setupCredentialsTestDB(t)
	defer cleanup()

	repo := NewCredentialsRepository(db)
	ctx := context.Background()

	userID := createCredentialsTestUser(t, db)

	cred := &pkgmodels.CredentialsResource{
		BaseResource: pkgmodels.BaseResource{
			Type:    pkgmodels.ResourceTypeCredentials,
			OwnerID: userID,
			Name:    "To Be Deleted",
			Status:  pkgmodels.ResourceStatusActive,
		},
		CredentialType: pkgmodels.CredentialTypeAPIKey,
		EncryptedData:  map[string]string{"api_key": "key"},
	}

	require.NoError(t, repo.CreateCredentials(ctx, cred))

	// Delete
	err := repo.DeleteCredentials(ctx, cred.ID)
	require.NoError(t, err)

	// Verify it's not found
	_, err = repo.GetCredentials(ctx, cred.ID)
	assert.ErrorIs(t, err, pkgmodels.ErrResourceNotFound)
}

// TestCredentialsRepository_IncrementUsage tests usage tracking
func TestCredentialsRepository_IncrementUsage(t *testing.T) {
	t.Parallel()
	db, cleanup := setupCredentialsTestDB(t)
	defer cleanup()

	repo := NewCredentialsRepository(db)
	ctx := context.Background()

	userID := createCredentialsTestUser(t, db)

	cred := &pkgmodels.CredentialsResource{
		BaseResource: pkgmodels.BaseResource{
			Type:    pkgmodels.ResourceTypeCredentials,
			OwnerID: userID,
			Name:    "Usage Test",
			Status:  pkgmodels.ResourceStatusActive,
		},
		CredentialType: pkgmodels.CredentialTypeAPIKey,
		EncryptedData:  map[string]string{"api_key": "key"},
	}

	require.NoError(t, repo.CreateCredentials(ctx, cred))

	// Increment usage multiple times
	require.NoError(t, repo.IncrementUsageCount(ctx, cred.ID))
	require.NoError(t, repo.IncrementUsageCount(ctx, cred.ID))
	require.NoError(t, repo.IncrementUsageCount(ctx, cred.ID))

	// Verify usage count
	retrieved, err := repo.GetCredentials(ctx, cred.ID)
	require.NoError(t, err)
	assert.Equal(t, int64(3), retrieved.UsageCount)
	assert.NotNil(t, retrieved.LastUsedAt)
}

// TestCredentialsRepository_InvalidID tests error handling for invalid IDs
func TestCredentialsRepository_InvalidID(t *testing.T) {
	t.Parallel()
	db, cleanup := setupCredentialsTestDB(t)
	defer cleanup()

	repo := NewCredentialsRepository(db)
	ctx := context.Background()

	// Test GetCredentials with invalid ID
	_, err := repo.GetCredentials(ctx, "invalid-uuid")
	assert.ErrorIs(t, err, pkgmodels.ErrInvalidID)

	// Test with non-existent UUID
	_, err = repo.GetCredentials(ctx, uuid.New().String())
	assert.ErrorIs(t, err, pkgmodels.ErrResourceNotFound)
}

// Helper functions

func setupCredentialsTestDB(t *testing.T) (bun.IDB, func()) {
	t.Helper()
	return testutil.SetupTestTx(t)
}

func createCredentialsTestUser(t *testing.T, db bun.IDB) string {
	ctx := context.Background()
	userID := uuid.New()

	user := &models.UserModel{
		ID:           userID,
		Username:     "test_user_" + userID.String()[:8],
		Email:        "test_" + userID.String()[:8] + "@test.com",
		PasswordHash: "test_hash",
		IsActive:     true,
		IsAdmin:      false,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	_, err := db.NewInsert().Model(user).Exec(ctx)
	require.NoError(t, err)

	return userID.String()
}
