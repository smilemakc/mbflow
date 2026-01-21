package servicekey

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"golang.org/x/crypto/bcrypt"

	"github.com/smilemakc/mbflow/internal/domain/repository"
	"github.com/smilemakc/mbflow/pkg/models"
)

type MockServiceKeyRepository struct {
	mock.Mock
}

func (m *MockServiceKeyRepository) Create(ctx context.Context, key *models.ServiceKey) error {
	args := m.Called(ctx, key)
	return args.Error(0)
}

func (m *MockServiceKeyRepository) FindByID(ctx context.Context, id uuid.UUID) (*models.ServiceKey, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.ServiceKey), args.Error(1)
}

func (m *MockServiceKeyRepository) FindByPrefix(ctx context.Context, prefix string) ([]*models.ServiceKey, error) {
	args := m.Called(ctx, prefix)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.ServiceKey), args.Error(1)
}

func (m *MockServiceKeyRepository) FindByUserID(ctx context.Context, userID uuid.UUID) ([]*models.ServiceKey, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.ServiceKey), args.Error(1)
}

func (m *MockServiceKeyRepository) FindAll(ctx context.Context, filter repository.ServiceKeyFilter) ([]*models.ServiceKey, int64, error) {
	args := m.Called(ctx, filter)
	if args.Get(0) == nil {
		return nil, args.Get(1).(int64), args.Error(2)
	}
	return args.Get(0).([]*models.ServiceKey), args.Get(1).(int64), args.Error(2)
}

func (m *MockServiceKeyRepository) Update(ctx context.Context, key *models.ServiceKey) error {
	args := m.Called(ctx, key)
	return args.Error(0)
}

func (m *MockServiceKeyRepository) Delete(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockServiceKeyRepository) Revoke(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockServiceKeyRepository) UpdateLastUsed(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockServiceKeyRepository) CountByUserID(ctx context.Context, userID uuid.UUID) (int64, error) {
	args := m.Called(ctx, userID)
	return args.Get(0).(int64), args.Error(1)
}

func TestNewService(t *testing.T) {
	tests := []struct {
		name            string
		config          Config
		expectedMaxKeys int
		expectedCost    int
	}{
		{
			name: "default config",
			config: Config{
				MaxKeysPerUser: 0,
				BcryptCost:     0,
			},
			expectedMaxKeys: DefaultMaxKeysPerUser,
			expectedCost:    BcryptCost,
		},
		{
			name: "custom config",
			config: Config{
				MaxKeysPerUser: 20,
				BcryptCost:     12,
			},
			expectedMaxKeys: 20,
			expectedCost:    12,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := new(MockServiceKeyRepository)
			service := NewService(repo, tt.config)

			assert.NotNil(t, service)
			assert.Equal(t, tt.expectedMaxKeys, service.config.MaxKeysPerUser)
			assert.Equal(t, tt.expectedCost, service.config.BcryptCost)
		})
	}
}

func TestCreateKey(t *testing.T) {
	ctx := context.Background()
	userID := uuid.New()
	createdBy := uuid.New()

	tests := []struct {
		name          string
		userID        uuid.UUID
		keyName       string
		description   string
		createdBy     uuid.UUID
		expiresInDays *int
		setupMock     func(*MockServiceKeyRepository)
		expectError   bool
		errorType     error
	}{
		{
			name:        "successful creation",
			userID:      userID,
			keyName:     "Test Key",
			description: "Test Description",
			createdBy:   createdBy,
			setupMock: func(repo *MockServiceKeyRepository) {
				repo.On("CountByUserID", ctx, userID).Return(int64(5), nil)
				repo.On("Create", ctx, mock.AnythingOfType("*models.ServiceKey")).Return(nil)
			},
			expectError: false,
		},
		{
			name:        "limit reached",
			userID:      userID,
			keyName:     "Test Key",
			description: "Test Description",
			createdBy:   createdBy,
			setupMock: func(repo *MockServiceKeyRepository) {
				repo.On("CountByUserID", ctx, userID).Return(int64(10), nil)
			},
			expectError: true,
			errorType:   models.ErrServiceKeyLimitReached,
		},
		{
			name:          "with expiration",
			userID:        userID,
			keyName:       "Test Key",
			description:   "Test Description",
			createdBy:     createdBy,
			expiresInDays: intPtr(30),
			setupMock: func(repo *MockServiceKeyRepository) {
				repo.On("CountByUserID", ctx, userID).Return(int64(5), nil)
				repo.On("Create", ctx, mock.AnythingOfType("*models.ServiceKey")).Return(nil)
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := new(MockServiceKeyRepository)
			tt.setupMock(repo)

			service := NewService(repo, Config{
				MaxKeysPerUser: 10,
				BcryptCost:     BcryptCost,
			})

			result, err := service.CreateKey(ctx, tt.userID, tt.keyName, tt.description, tt.createdBy, tt.expiresInDays)

			if tt.expectError {
				assert.Error(t, err)
				if tt.errorType != nil {
					assert.ErrorIs(t, err, tt.errorType)
				}
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
				assert.NotNil(t, result.Key)
				assert.NotEmpty(t, result.PlainKey)

				assert.True(t, strings.HasPrefix(result.PlainKey, models.ServiceKeyPrefix))

				assert.NotEmpty(t, result.Key.KeyHash)

				err := bcrypt.CompareHashAndPassword([]byte(result.Key.KeyHash), []byte(result.PlainKey))
				assert.NoError(t, err)

				if tt.expiresInDays != nil {
					assert.NotNil(t, result.Key.ExpiresAt)
				}
			}

			repo.AssertExpectations(t)
		})
	}
}

func TestValidateKey(t *testing.T) {
	ctx := context.Background()

	plainKey, prefix, _ := generatePlainKey()
	hash, _ := bcrypt.GenerateFromPassword([]byte(plainKey), BcryptCost)

	validKey := &models.ServiceKey{
		ID:        uuid.New().String(),
		UserID:    uuid.New().String(),
		Name:      "Test Key",
		KeyPrefix: prefix,
		KeyHash:   string(hash),
		Status:    models.ServiceKeyStatusActive,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	revokedKey := &models.ServiceKey{
		ID:        uuid.New().String(),
		UserID:    uuid.New().String(),
		Name:      "Revoked Key",
		KeyPrefix: prefix,
		KeyHash:   string(hash),
		Status:    models.ServiceKeyStatusRevoked,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	expiredTime := time.Now().Add(-24 * time.Hour)
	expiredKey := &models.ServiceKey{
		ID:        uuid.New().String(),
		UserID:    uuid.New().String(),
		Name:      "Expired Key",
		KeyPrefix: prefix,
		KeyHash:   string(hash),
		Status:    models.ServiceKeyStatusActive,
		ExpiresAt: &expiredTime,
		CreatedAt: time.Now().Add(-48 * time.Hour),
		UpdatedAt: time.Now(),
	}

	tests := []struct {
		name        string
		plainKey    string
		setupMock   func(*MockServiceKeyRepository)
		expectError bool
		errorType   error
	}{
		{
			name:     "valid key",
			plainKey: plainKey,
			setupMock: func(repo *MockServiceKeyRepository) {
				repo.On("FindByPrefix", ctx, prefix).Return([]*models.ServiceKey{validKey}, nil)
				repo.On("UpdateLastUsed", ctx, uuid.MustParse(validKey.ID)).Return(nil)
			},
			expectError: false,
		},
		{
			name:     "revoked key",
			plainKey: plainKey,
			setupMock: func(repo *MockServiceKeyRepository) {
				repo.On("FindByPrefix", ctx, prefix).Return([]*models.ServiceKey{revokedKey}, nil)
			},
			expectError: true,
			errorType:   models.ErrServiceKeyRevoked,
		},
		{
			name:     "expired key",
			plainKey: plainKey,
			setupMock: func(repo *MockServiceKeyRepository) {
				repo.On("FindByPrefix", ctx, prefix).Return([]*models.ServiceKey{expiredKey}, nil)
			},
			expectError: true,
			errorType:   models.ErrServiceKeyExpired,
		},
		{
			name:     "key not found",
			plainKey: plainKey,
			setupMock: func(repo *MockServiceKeyRepository) {
				repo.On("FindByPrefix", ctx, prefix).Return([]*models.ServiceKey{}, nil)
			},
			expectError: true,
			errorType:   models.ErrServiceKeyNotFound,
		},
		{
			name:        "invalid format",
			plainKey:    "short",
			setupMock:   func(repo *MockServiceKeyRepository) {},
			expectError: true,
			errorType:   ErrInvalidKeyFormat,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := new(MockServiceKeyRepository)
			tt.setupMock(repo)

			service := NewService(repo, Config{
				MaxKeysPerUser: 10,
				BcryptCost:     BcryptCost,
			})

			key, err := service.ValidateKey(ctx, tt.plainKey)

			if tt.expectError {
				assert.Error(t, err)
				if tt.errorType != nil {
					assert.ErrorIs(t, err, tt.errorType)
				}
				assert.Nil(t, key)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, key)
			}

			repo.AssertExpectations(t)
		})
	}
}

func TestRevokeKey(t *testing.T) {
	ctx := context.Background()
	keyID := uuid.New()

	tests := []struct {
		name        string
		keyID       uuid.UUID
		setupMock   func(*MockServiceKeyRepository)
		expectError bool
		errorType   error
	}{
		{
			name:  "successful revoke",
			keyID: keyID,
			setupMock: func(repo *MockServiceKeyRepository) {
				key := &models.ServiceKey{
					ID:     keyID.String(),
					Status: models.ServiceKeyStatusActive,
				}
				repo.On("FindByID", ctx, keyID).Return(key, nil)
				repo.On("Revoke", ctx, keyID).Return(nil)
			},
			expectError: false,
		},
		{
			name:  "key not found",
			keyID: keyID,
			setupMock: func(repo *MockServiceKeyRepository) {
				repo.On("FindByID", ctx, keyID).Return(nil, nil)
			},
			expectError: true,
			errorType:   models.ErrServiceKeyNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := new(MockServiceKeyRepository)
			tt.setupMock(repo)

			service := NewService(repo, Config{
				MaxKeysPerUser: 10,
				BcryptCost:     BcryptCost,
			})

			err := service.RevokeKey(ctx, tt.keyID)

			if tt.expectError {
				assert.Error(t, err)
				if tt.errorType != nil {
					assert.ErrorIs(t, err, tt.errorType)
				}
			} else {
				assert.NoError(t, err)
			}

			repo.AssertExpectations(t)
		})
	}
}

func TestGeneratePlainKey(t *testing.T) {
	plainKey, prefix, err := generatePlainKey()

	assert.NoError(t, err)
	assert.NotEmpty(t, plainKey)
	assert.NotEmpty(t, prefix)

	assert.True(t, strings.HasPrefix(plainKey, models.ServiceKeyPrefix))

	assert.Equal(t, models.ServiceKeyPrefixLength, len(prefix))

	secondKey, _, _ := generatePlainKey()
	assert.NotEqual(t, plainKey, secondKey)
}

func TestHashAndVerifyKey(t *testing.T) {
	service := NewService(nil, Config{
		BcryptCost: BcryptCost,
	})

	plainKey := "sk_testkey12345678901234567890123456"

	hash, err := service.hashKey(plainKey)
	assert.NoError(t, err)
	assert.NotEmpty(t, hash)

	assert.True(t, verifyKey(plainKey, hash))

	wrongKey := "sk_wrongkey12345678901234567890123456"
	assert.False(t, verifyKey(wrongKey, hash))
}

func intPtr(i int) *int {
	return &i
}
