package serviceapi

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/smilemakc/mbflow/go/pkg/models"
)

// --- toCredentialInfo ---

func TestToCredentialInfo_ShouldMapAllFields(t *testing.T) {
	// Arrange
	now := time.Now()
	expiresAt := now.Add(24 * time.Hour)
	lastUsed := now.Add(-1 * time.Hour)
	cred := &models.CredentialsResource{
		BaseResource: models.BaseResource{
			ID:          "cred-123",
			Name:        "My API Key",
			Description: "Test credential",
			Status:      models.ResourceStatusActive,
			CreatedAt:   now,
			UpdatedAt:   now,
		},
		CredentialType: models.CredentialTypeAPIKey,
		Provider:       "openai",
		ExpiresAt:      &expiresAt,
		LastUsedAt:     &lastUsed,
		UsageCount:     42,
		EncryptedData:  map[string]string{"api_key": "enc_value", "extra": "enc_extra"},
	}

	// Act
	info := toCredentialInfo(cred)

	// Assert
	assert.Equal(t, "cred-123", info.ID)
	assert.Equal(t, "My API Key", info.Name)
	assert.Equal(t, "Test credential", info.Description)
	assert.Equal(t, "active", info.Status)
	assert.Equal(t, "api_key", info.CredentialType)
	assert.Equal(t, "openai", info.Provider)
	assert.Equal(t, &expiresAt, info.ExpiresAt)
	assert.Equal(t, &lastUsed, info.LastUsedAt)
	assert.Equal(t, int64(42), info.UsageCount)
	assert.Equal(t, now, info.CreatedAt)
	assert.Equal(t, now, info.UpdatedAt)
	assert.Len(t, info.Fields, 2)
	assert.Contains(t, info.Fields, "api_key")
	assert.Contains(t, info.Fields, "extra")
}

func TestToCredentialInfo_ShouldHandleEmptyEncryptedData(t *testing.T) {
	cred := &models.CredentialsResource{
		BaseResource:  models.BaseResource{ID: "cred-1"},
		EncryptedData: map[string]string{},
	}

	info := toCredentialInfo(cred)

	assert.Empty(t, info.Fields)
}

func TestToCredentialInfo_ShouldHandleNilOptionalFields(t *testing.T) {
	cred := &models.CredentialsResource{
		BaseResource:  models.BaseResource{ID: "cred-2"},
		EncryptedData: map[string]string{"k": "v"},
	}

	info := toCredentialInfo(cred)

	assert.Nil(t, info.ExpiresAt)
	assert.Nil(t, info.LastUsedAt)
	assert.Equal(t, int64(0), info.UsageCount)
}

// --- ListCredentials ---

func TestListCredentials_ShouldReturnError_WhenUserIDEmpty(t *testing.T) {
	// Arrange
	ops := newTestOperations(nil, nil, nil, nil, nil, nil, nil)

	// Act
	result, err := ops.ListCredentials(context.Background(), ListCredentialsParams{
		UserID: "",
	})

	// Assert
	assert.Nil(t, result)
	require.Error(t, err)
	var opErr *OperationError
	require.ErrorAs(t, err, &opErr)
	assert.Equal(t, "USER_ID_REQUIRED", opErr.Code)
}

func TestListCredentials_ShouldReturnByOwner_WhenNoProviderFilter(t *testing.T) {
	// Arrange
	credRepo := new(mockCredentialsRepo)
	ops := newTestOperations(nil, nil, nil, credRepo, nil, nil, nil)

	creds := []*models.CredentialsResource{
		{BaseResource: models.BaseResource{ID: "cred-1", Name: "Key 1"}, EncryptedData: map[string]string{"k": "v"}},
		{BaseResource: models.BaseResource{ID: "cred-2", Name: "Key 2"}, EncryptedData: map[string]string{"a": "b"}},
	}
	credRepo.On("GetCredentialsByOwner", mock.Anything, "user-1").Return(creds, nil)

	// Act
	result, err := ops.ListCredentials(context.Background(), ListCredentialsParams{
		UserID: "user-1",
	})

	// Assert
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Len(t, result.Credentials, 2)
	assert.Equal(t, "cred-1", result.Credentials[0].ID)
	assert.Equal(t, "cred-2", result.Credentials[1].ID)
	credRepo.AssertExpectations(t)
}

func TestListCredentials_ShouldFilterByProvider_WhenProviderSpecified(t *testing.T) {
	// Arrange
	credRepo := new(mockCredentialsRepo)
	ops := newTestOperations(nil, nil, nil, credRepo, nil, nil, nil)

	creds := []*models.CredentialsResource{
		{BaseResource: models.BaseResource{ID: "cred-1"}, EncryptedData: map[string]string{"k": "v"}},
	}
	credRepo.On("GetCredentialsByProvider", mock.Anything, "user-1", "openai").Return(creds, nil)

	// Act
	result, err := ops.ListCredentials(context.Background(), ListCredentialsParams{
		UserID:   "user-1",
		Provider: "openai",
	})

	// Assert
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Len(t, result.Credentials, 1)
	credRepo.AssertExpectations(t)
}

func TestListCredentials_ShouldReturnEmptyList_WhenNoCredentialsFound(t *testing.T) {
	// Arrange
	credRepo := new(mockCredentialsRepo)
	ops := newTestOperations(nil, nil, nil, credRepo, nil, nil, nil)

	credRepo.On("GetCredentialsByOwner", mock.Anything, "user-empty").Return([]*models.CredentialsResource{}, nil)

	// Act
	result, err := ops.ListCredentials(context.Background(), ListCredentialsParams{
		UserID: "user-empty",
	})

	// Assert
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Empty(t, result.Credentials)
}

func TestListCredentials_ShouldReturnError_WhenRepoFails(t *testing.T) {
	// Arrange
	credRepo := new(mockCredentialsRepo)
	ops := newTestOperations(nil, nil, nil, credRepo, nil, nil, nil)

	credRepo.On("GetCredentialsByOwner", mock.Anything, "user-err").Return(([]*models.CredentialsResource)(nil), errors.New("db error"))

	// Act
	result, err := ops.ListCredentials(context.Background(), ListCredentialsParams{
		UserID: "user-err",
	})

	// Assert
	assert.Nil(t, result)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "db error")
}

// --- CreateCredential ---

func TestCreateCredential_ShouldReturnError_WhenUserIDEmpty(t *testing.T) {
	// Arrange
	ops := newTestOperations(nil, nil, nil, nil, nil, nil, nil)

	// Act
	result, err := ops.CreateCredential(context.Background(), CreateCredentialParams{
		UserID:         "",
		Name:           "Test",
		CredentialType: "api_key",
	})

	// Assert
	assert.Nil(t, result)
	require.Error(t, err)
	var opErr *OperationError
	require.ErrorAs(t, err, &opErr)
	assert.Equal(t, "UNAUTHORIZED", opErr.Code)
}

func TestCreateCredential_ShouldReturnError_WhenInvalidCredentialType(t *testing.T) {
	// Arrange
	ops := newTestOperations(nil, nil, nil, nil, nil, nil, nil)

	// Act
	result, err := ops.CreateCredential(context.Background(), CreateCredentialParams{
		UserID:         "user-1",
		Name:           "Test",
		CredentialType: "invalid_type",
	})

	// Assert
	assert.Nil(t, result)
	require.Error(t, err)
	var opErr *OperationError
	require.ErrorAs(t, err, &opErr)
	assert.Equal(t, "INVALID_CREDENTIAL_TYPE", opErr.Code)
}

func TestCreateCredential_ShouldReturnCredentialInfo_WhenValidParams(t *testing.T) {
	// Arrange
	credRepo := new(mockCredentialsRepo)
	ops := newTestOperations(nil, nil, nil, credRepo, nil, nil, nil)

	credRepo.On("CreateCredentials", mock.Anything, mock.AnythingOfType("*models.CredentialsResource")).Return(nil)

	// Act
	result, err := ops.CreateCredential(context.Background(), CreateCredentialParams{
		UserID:         "user-1",
		Name:           "My OpenAI Key",
		Description:    "For production",
		CredentialType: "api_key",
		Provider:       "openai",
		Data:           map[string]string{"api_key": "sk-test123"},
	})

	// Assert
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, "My OpenAI Key", result.Name)
	assert.Equal(t, "For production", result.Description)
	assert.Equal(t, "api_key", result.CredentialType)
	assert.Equal(t, "openai", result.Provider)
	assert.Contains(t, result.Fields, "api_key")
	credRepo.AssertExpectations(t)
}

func TestCreateCredential_ShouldAcceptAllValidCredentialTypes(t *testing.T) {
	validTypes := []string{"api_key", "basic_auth", "oauth2", "service_account", "custom"}

	for _, credType := range validTypes {
		t.Run(credType, func(t *testing.T) {
			credRepo := new(mockCredentialsRepo)
			ops := newTestOperations(nil, nil, nil, credRepo, nil, nil, nil)
			credRepo.On("CreateCredentials", mock.Anything, mock.Anything).Return(nil)

			result, err := ops.CreateCredential(context.Background(), CreateCredentialParams{
				UserID:         "user-1",
				Name:           "Test",
				CredentialType: credType,
				Data:           map[string]string{"key": "value"},
			})

			require.NoError(t, err)
			require.NotNil(t, result)
			assert.Equal(t, credType, result.CredentialType)
		})
	}
}

func TestCreateCredential_ShouldReturnError_WhenRepoFails(t *testing.T) {
	// Arrange
	credRepo := new(mockCredentialsRepo)
	ops := newTestOperations(nil, nil, nil, credRepo, nil, nil, nil)

	credRepo.On("CreateCredentials", mock.Anything, mock.Anything).Return(errors.New("db write error"))

	// Act
	result, err := ops.CreateCredential(context.Background(), CreateCredentialParams{
		UserID:         "user-1",
		Name:           "Test",
		CredentialType: "api_key",
		Data:           map[string]string{"api_key": "sk-test"},
	})

	// Assert
	assert.Nil(t, result)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "db write error")
}

func TestCreateCredential_ShouldEncryptData(t *testing.T) {
	// Arrange
	credRepo := new(mockCredentialsRepo)
	ops := newTestOperations(nil, nil, nil, credRepo, nil, nil, nil)

	var savedCred *models.CredentialsResource
	credRepo.On("CreateCredentials", mock.Anything, mock.Anything).
		Run(func(args mock.Arguments) {
			savedCred = args.Get(1).(*models.CredentialsResource)
		}).
		Return(nil)

	// Act
	_, err := ops.CreateCredential(context.Background(), CreateCredentialParams{
		UserID:         "user-1",
		Name:           "Test",
		CredentialType: "api_key",
		Data:           map[string]string{"api_key": "sk-plaintext"},
	})

	// Assert
	require.NoError(t, err)
	require.NotNil(t, savedCred)
	// Encrypted data should not contain the plaintext value
	assert.NotEqual(t, "sk-plaintext", savedCred.EncryptedData["api_key"])
	assert.NotEmpty(t, savedCred.EncryptedData["api_key"])
}

func TestCreateCredential_ShouldHandleEmptyData(t *testing.T) {
	// Arrange
	credRepo := new(mockCredentialsRepo)
	ops := newTestOperations(nil, nil, nil, credRepo, nil, nil, nil)
	credRepo.On("CreateCredentials", mock.Anything, mock.Anything).Return(nil)

	// Act
	result, err := ops.CreateCredential(context.Background(), CreateCredentialParams{
		UserID:         "user-1",
		Name:           "Empty",
		CredentialType: "custom",
		Data:           map[string]string{},
	})

	// Assert
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Empty(t, result.Fields)
}

// --- UpdateCredential ---

func TestUpdateCredential_ShouldReturnError_WhenCredentialNotFound(t *testing.T) {
	// Arrange
	credRepo := new(mockCredentialsRepo)
	ops := newTestOperations(nil, nil, nil, credRepo, nil, nil, nil)

	credRepo.On("GetCredentials", mock.Anything, "nonexistent").Return((*models.CredentialsResource)(nil), models.ErrResourceNotFound)

	// Act
	result, err := ops.UpdateCredential(context.Background(), UpdateCredentialParams{
		CredentialID: "nonexistent",
		Name:         "New Name",
	})

	// Assert
	assert.Nil(t, result)
	require.Error(t, err)
	assert.ErrorIs(t, err, models.ErrResourceNotFound)
}

func TestUpdateCredential_ShouldUpdateName_WhenNameProvided(t *testing.T) {
	// Arrange
	credRepo := new(mockCredentialsRepo)
	ops := newTestOperations(nil, nil, nil, credRepo, nil, nil, nil)

	existingCred := &models.CredentialsResource{
		BaseResource:  models.BaseResource{ID: "cred-1", Name: "Old Name", Description: "Old Desc"},
		EncryptedData: map[string]string{"k": "v"},
	}
	credRepo.On("GetCredentials", mock.Anything, "cred-1").Return(existingCred, nil)
	credRepo.On("UpdateCredentials", mock.Anything, mock.Anything).Return(nil)

	// Act
	result, err := ops.UpdateCredential(context.Background(), UpdateCredentialParams{
		CredentialID: "cred-1",
		Name:         "New Name",
		Description:  "New Desc",
	})

	// Assert
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, "New Name", result.Name)
	assert.Equal(t, "New Desc", result.Description)
	credRepo.AssertExpectations(t)
}

func TestUpdateCredential_ShouldKeepOldName_WhenNameEmpty(t *testing.T) {
	// Arrange
	credRepo := new(mockCredentialsRepo)
	ops := newTestOperations(nil, nil, nil, credRepo, nil, nil, nil)

	existingCred := &models.CredentialsResource{
		BaseResource:  models.BaseResource{ID: "cred-1", Name: "Original Name"},
		EncryptedData: map[string]string{"k": "v"},
	}
	credRepo.On("GetCredentials", mock.Anything, "cred-1").Return(existingCred, nil)
	credRepo.On("UpdateCredentials", mock.Anything, mock.Anything).Return(nil)

	// Act
	result, err := ops.UpdateCredential(context.Background(), UpdateCredentialParams{
		CredentialID: "cred-1",
		Name:         "",
		Description:  "Updated desc",
	})

	// Assert
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, "Original Name", result.Name)
}

func TestUpdateCredential_ShouldReturnError_WhenUpdateFails(t *testing.T) {
	// Arrange
	credRepo := new(mockCredentialsRepo)
	ops := newTestOperations(nil, nil, nil, credRepo, nil, nil, nil)

	existingCred := &models.CredentialsResource{
		BaseResource:  models.BaseResource{ID: "cred-1", Name: "Name"},
		EncryptedData: map[string]string{"k": "v"},
	}
	credRepo.On("GetCredentials", mock.Anything, "cred-1").Return(existingCred, nil)
	credRepo.On("UpdateCredentials", mock.Anything, mock.Anything).Return(errors.New("update failed"))

	// Act
	result, err := ops.UpdateCredential(context.Background(), UpdateCredentialParams{
		CredentialID: "cred-1",
		Name:         "New",
	})

	// Assert
	assert.Nil(t, result)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "update failed")
}

func TestUpdateCredential_ShouldReturnError_WhenGetReturnsUnexpectedError(t *testing.T) {
	// Arrange
	credRepo := new(mockCredentialsRepo)
	ops := newTestOperations(nil, nil, nil, credRepo, nil, nil, nil)

	credRepo.On("GetCredentials", mock.Anything, "cred-1").Return((*models.CredentialsResource)(nil), errors.New("db connection lost"))

	// Act
	result, err := ops.UpdateCredential(context.Background(), UpdateCredentialParams{
		CredentialID: "cred-1",
		Name:         "New",
	})

	// Assert
	assert.Nil(t, result)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "db connection lost")
}

// --- DeleteCredential ---

func TestDeleteCredential_ShouldReturnError_WhenCredentialNotFound(t *testing.T) {
	// Arrange
	credRepo := new(mockCredentialsRepo)
	ops := newTestOperations(nil, nil, nil, credRepo, nil, nil, nil)

	credRepo.On("GetCredentials", mock.Anything, "nonexistent").Return((*models.CredentialsResource)(nil), models.ErrResourceNotFound)

	// Act
	err := ops.DeleteCredential(context.Background(), DeleteCredentialParams{
		CredentialID: "nonexistent",
	})

	// Assert
	require.Error(t, err)
	assert.ErrorIs(t, err, models.ErrResourceNotFound)
}

func TestDeleteCredential_ShouldReturnValidationError_WhenIDNotUUID(t *testing.T) {
	// Arrange
	credRepo := new(mockCredentialsRepo)
	ops := newTestOperations(nil, nil, nil, credRepo, nil, nil, nil)

	existingCred := &models.CredentialsResource{
		BaseResource:  models.BaseResource{ID: "not-a-uuid"},
		EncryptedData: map[string]string{"k": "v"},
	}
	credRepo.On("GetCredentials", mock.Anything, "not-a-uuid").Return(existingCred, nil)

	// Act
	err := ops.DeleteCredential(context.Background(), DeleteCredentialParams{
		CredentialID: "not-a-uuid",
	})

	// Assert
	require.Error(t, err)
	var opErr *OperationError
	require.ErrorAs(t, err, &opErr)
	assert.Equal(t, "INVALID_ID", opErr.Code)
}

func TestDeleteCredential_ShouldDetachAndDelete_WhenCredentialExists(t *testing.T) {
	// Arrange
	credRepo := new(mockCredentialsRepo)
	wfRepo := new(mockWorkflowRepo)
	ops := newTestOperations(wfRepo, nil, nil, credRepo, nil, nil, nil)

	credID := uuid.New()
	existingCred := &models.CredentialsResource{
		BaseResource:  models.BaseResource{ID: credID.String()},
		EncryptedData: map[string]string{"k": "v"},
	}
	credRepo.On("GetCredentials", mock.Anything, credID.String()).Return(existingCred, nil)
	wfRepo.On("UnassignResourceFromAllWorkflows", mock.Anything, credID).Return(int64(2), nil)
	credRepo.On("DeleteCredentials", mock.Anything, credID.String()).Return(nil)

	// Act
	err := ops.DeleteCredential(context.Background(), DeleteCredentialParams{
		CredentialID: credID.String(),
	})

	// Assert
	require.NoError(t, err)
	wfRepo.AssertExpectations(t)
	credRepo.AssertExpectations(t)
}

func TestDeleteCredential_ShouldDelete_WhenNoWorkflowsAttached(t *testing.T) {
	// Arrange
	credRepo := new(mockCredentialsRepo)
	wfRepo := new(mockWorkflowRepo)
	ops := newTestOperations(wfRepo, nil, nil, credRepo, nil, nil, nil)

	credID := uuid.New()
	existingCred := &models.CredentialsResource{
		BaseResource:  models.BaseResource{ID: credID.String()},
		EncryptedData: map[string]string{"k": "v"},
	}
	credRepo.On("GetCredentials", mock.Anything, credID.String()).Return(existingCred, nil)
	wfRepo.On("UnassignResourceFromAllWorkflows", mock.Anything, credID).Return(int64(0), nil)
	credRepo.On("DeleteCredentials", mock.Anything, credID.String()).Return(nil)

	// Act
	err := ops.DeleteCredential(context.Background(), DeleteCredentialParams{
		CredentialID: credID.String(),
	})

	// Assert
	require.NoError(t, err)
}

func TestDeleteCredential_ShouldReturnError_WhenDetachFails(t *testing.T) {
	// Arrange
	credRepo := new(mockCredentialsRepo)
	wfRepo := new(mockWorkflowRepo)
	ops := newTestOperations(wfRepo, nil, nil, credRepo, nil, nil, nil)

	credID := uuid.New()
	existingCred := &models.CredentialsResource{
		BaseResource:  models.BaseResource{ID: credID.String()},
		EncryptedData: map[string]string{"k": "v"},
	}
	credRepo.On("GetCredentials", mock.Anything, credID.String()).Return(existingCred, nil)
	wfRepo.On("UnassignResourceFromAllWorkflows", mock.Anything, credID).Return(int64(0), errors.New("detach failed"))

	// Act
	err := ops.DeleteCredential(context.Background(), DeleteCredentialParams{
		CredentialID: credID.String(),
	})

	// Assert
	require.Error(t, err)
	assert.Contains(t, err.Error(), "detach failed")
}

func TestDeleteCredential_ShouldReturnError_WhenDeleteFails(t *testing.T) {
	// Arrange
	credRepo := new(mockCredentialsRepo)
	wfRepo := new(mockWorkflowRepo)
	ops := newTestOperations(wfRepo, nil, nil, credRepo, nil, nil, nil)

	credID := uuid.New()
	existingCred := &models.CredentialsResource{
		BaseResource:  models.BaseResource{ID: credID.String()},
		EncryptedData: map[string]string{"k": "v"},
	}
	credRepo.On("GetCredentials", mock.Anything, credID.String()).Return(existingCred, nil)
	wfRepo.On("UnassignResourceFromAllWorkflows", mock.Anything, credID).Return(int64(0), nil)
	credRepo.On("DeleteCredentials", mock.Anything, credID.String()).Return(errors.New("delete error"))

	// Act
	err := ops.DeleteCredential(context.Background(), DeleteCredentialParams{
		CredentialID: credID.String(),
	})

	// Assert
	require.Error(t, err)
	assert.Contains(t, err.Error(), "delete error")
}
