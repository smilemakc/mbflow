package systemkey

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"github.com/smilemakc/mbflow/go/internal/domain/repository"
	"github.com/smilemakc/mbflow/go/pkg/models"
)

// MockSystemKeyRepository is a mock implementation of SystemKeyRepository
type MockSystemKeyRepository struct {
	keys         map[string]*models.SystemKey
	prefixIndex  map[string][]string // prefix -> []keyID
	countResult  int64
	countErr     error
	createErr    error
	findByIDErr  error
	revokeErr    error
	deleteErr    error
	updateLastUsedErr error
}

func NewMockSystemKeyRepository() *MockSystemKeyRepository {
	return &MockSystemKeyRepository{
		keys:        make(map[string]*models.SystemKey),
		prefixIndex: make(map[string][]string),
	}
}

func (m *MockSystemKeyRepository) Create(ctx context.Context, key *models.SystemKey) error {
	if m.createErr != nil {
		return m.createErr
	}
	m.keys[key.ID] = key
	if key.KeyPrefix != "" {
		m.prefixIndex[key.KeyPrefix] = append(m.prefixIndex[key.KeyPrefix], key.ID)
	}
	return nil
}

func (m *MockSystemKeyRepository) FindByID(ctx context.Context, id uuid.UUID) (*models.SystemKey, error) {
	if m.findByIDErr != nil {
		return nil, m.findByIDErr
	}
	key, ok := m.keys[id.String()]
	if !ok {
		return nil, nil
	}
	return key, nil
}

func (m *MockSystemKeyRepository) FindByPrefix(ctx context.Context, prefix string) ([]*models.SystemKey, error) {
	var result []*models.SystemKey
	keyIDs, ok := m.prefixIndex[prefix]
	if !ok {
		return result, nil
	}
	for _, keyID := range keyIDs {
		if key, exists := m.keys[keyID]; exists {
			result = append(result, key)
		}
	}
	return result, nil
}

func (m *MockSystemKeyRepository) FindAll(ctx context.Context, filter repository.SystemKeyFilter) ([]*models.SystemKey, int64, error) {
	var result []*models.SystemKey
	for _, key := range m.keys {
		result = append(result, key)
	}
	return result, int64(len(result)), nil
}

func (m *MockSystemKeyRepository) Update(ctx context.Context, key *models.SystemKey) error {
	m.keys[key.ID] = key
	return nil
}

func (m *MockSystemKeyRepository) Delete(ctx context.Context, id uuid.UUID) error {
	if m.deleteErr != nil {
		return m.deleteErr
	}
	delete(m.keys, id.String())
	return nil
}

func (m *MockSystemKeyRepository) Revoke(ctx context.Context, id uuid.UUID) error {
	if m.revokeErr != nil {
		return m.revokeErr
	}
	if key, ok := m.keys[id.String()]; ok {
		key.Revoke()
	}
	return nil
}

func (m *MockSystemKeyRepository) UpdateLastUsed(ctx context.Context, id uuid.UUID) error {
	if m.updateLastUsedErr != nil {
		return m.updateLastUsedErr
	}
	if key, ok := m.keys[id.String()]; ok {
		now := time.Now()
		key.LastUsedAt = &now
		key.UpdatedAt = now
	}
	return nil
}

func (m *MockSystemKeyRepository) Count(ctx context.Context) (int64, error) {
	if m.countErr != nil {
		return 0, m.countErr
	}
	if m.countResult > 0 {
		return m.countResult, nil
	}
	return int64(len(m.keys)), nil
}

// Test helper to create a valid key hash
func hashTestKey(plainKey string) string {
	hash, _ := bcrypt.GenerateFromPassword([]byte(plainKey), BcryptCost)
	return string(hash)
}

func TestCreateKey_ShouldReturnPlainKeyWithPrefix_WhenValidDataProvided(t *testing.T) {
	// Arrange
	repo := NewMockSystemKeyRepository()
	service := NewService(repo, Config{
		MaxKeys:    10,
		BcryptCost: BcryptCost,
	})

	createdBy := uuid.New()
	name := "test-key"
	description := "test description"
	serviceName := "test-service"

	// Act
	result, err := service.CreateKey(context.Background(), name, description, serviceName, createdBy, nil)

	// Assert
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if result == nil {
		t.Fatal("expected result, got nil")
	}

	if !strings.HasPrefix(result.PlainKey, models.SystemKeyPrefix) {
		t.Errorf("expected plain key to start with %s, got %s", models.SystemKeyPrefix, result.PlainKey)
	}

	if result.Key.KeyHash == "" {
		t.Error("expected key hash to be set")
	}

	if result.Key.KeyPrefix == "" {
		t.Error("expected key prefix to be set")
	}

	if result.Key.Name != name {
		t.Errorf("expected name %s, got %s", name, result.Key.Name)
	}

	if result.Key.Status != models.SystemKeyStatusActive {
		t.Errorf("expected status %s, got %s", models.SystemKeyStatusActive, result.Key.Status)
	}

	// Verify the plain key can be verified against the hash
	if err := bcrypt.CompareHashAndPassword([]byte(result.Key.KeyHash), []byte(result.PlainKey)); err != nil {
		t.Error("plain key does not match stored hash")
	}
}

func TestCreateKey_ShouldReturnError_WhenLimitReached(t *testing.T) {
	// Arrange
	repo := NewMockSystemKeyRepository()
	repo.countResult = 10 // Set count to max
	service := NewService(repo, Config{
		MaxKeys:    10,
		BcryptCost: BcryptCost,
	})

	createdBy := uuid.New()

	// Act
	result, err := service.CreateKey(context.Background(), "test", "desc", "service", createdBy, nil)

	// Assert
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if !errors.Is(err, models.ErrSystemKeyLimitReached) {
		t.Errorf("expected ErrSystemKeyLimitReached, got %v", err)
	}

	if result != nil {
		t.Error("expected nil result when limit reached")
	}
}

func TestCreateKey_ShouldSetExpiration_WhenExpiresInDaysProvided(t *testing.T) {
	// Arrange
	repo := NewMockSystemKeyRepository()
	service := NewService(repo, Config{
		MaxKeys:    10,
		BcryptCost: BcryptCost,
	})

	createdBy := uuid.New()
	expiresInDays := 30

	// Act
	result, err := service.CreateKey(context.Background(), "test", "desc", "service", createdBy, &expiresInDays)

	// Assert
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if result.Key.ExpiresAt == nil {
		t.Error("expected expiration to be set")
	}

	expectedExpiration := time.Now().AddDate(0, 0, expiresInDays)
	diff := result.Key.ExpiresAt.Sub(expectedExpiration)
	if diff > time.Minute || diff < -time.Minute {
		t.Errorf("expiration not set correctly, expected around %v, got %v", expectedExpiration, result.Key.ExpiresAt)
	}
}

func TestValidateKey_ShouldReturnKey_WhenValidKeyProvided(t *testing.T) {
	// Arrange
	repo := NewMockSystemKeyRepository()
	service := NewService(repo, Config{
		MaxKeys:    10,
		BcryptCost: BcryptCost,
	})

	// Create a key first
	createdBy := uuid.New()
	createResult, err := service.CreateKey(context.Background(), "test", "desc", "service", createdBy, nil)
	if err != nil {
		t.Fatalf("failed to create key: %v", err)
	}

	plainKey := createResult.PlainKey

	// Act
	validatedKey, err := service.ValidateKey(context.Background(), plainKey)

	// Assert
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if validatedKey == nil {
		t.Fatal("expected key, got nil")
	}

	if validatedKey.ID != createResult.Key.ID {
		t.Errorf("expected key ID %s, got %s", createResult.Key.ID, validatedKey.ID)
	}

	if validatedKey.UsageCount != 1 {
		t.Errorf("expected usage count 1, got %d", validatedKey.UsageCount)
	}
}

func TestValidateKey_ShouldReturnError_WhenKeyRevoked(t *testing.T) {
	// Arrange
	repo := NewMockSystemKeyRepository()
	service := NewService(repo, Config{
		MaxKeys:    10,
		BcryptCost: BcryptCost,
	})

	// Create and revoke a key
	createdBy := uuid.New()
	createResult, err := service.CreateKey(context.Background(), "test", "desc", "service", createdBy, nil)
	if err != nil {
		t.Fatalf("failed to create key: %v", err)
	}

	plainKey := createResult.PlainKey
	keyID := uuid.MustParse(createResult.Key.ID)

	err = service.RevokeKey(context.Background(), keyID)
	if err != nil {
		t.Fatalf("failed to revoke key: %v", err)
	}

	// Act
	validatedKey, err := service.ValidateKey(context.Background(), plainKey)

	// Assert
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if !errors.Is(err, models.ErrSystemKeyRevoked) {
		t.Errorf("expected ErrSystemKeyRevoked, got %v", err)
	}

	if validatedKey != nil {
		t.Error("expected nil key when revoked")
	}
}

func TestValidateKey_ShouldReturnError_WhenKeyExpired(t *testing.T) {
	// Arrange
	repo := NewMockSystemKeyRepository()
	service := NewService(repo, Config{
		MaxKeys:    10,
		BcryptCost: BcryptCost,
	})

	// Create a key
	createdBy := uuid.New()
	key := models.NewSystemKey("test", "desc", "service", createdBy.String())

	// Generate a plain key and hash
	plainKey := models.SystemKeyPrefix + "testkey1234567890"
	keyHash := hashTestKey(plainKey)
	key.KeyHash = keyHash
	key.KeyPrefix = plainKey[:models.SystemKeyPrefixLength]

	// Set expiration to the past
	pastTime := time.Now().Add(-24 * time.Hour)
	key.ExpiresAt = &pastTime

	repo.Create(context.Background(), key)

	// Act
	validatedKey, err := service.ValidateKey(context.Background(), plainKey)

	// Assert
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if !errors.Is(err, models.ErrSystemKeyExpired) {
		t.Errorf("expected ErrSystemKeyExpired, got %v", err)
	}

	if validatedKey != nil {
		t.Error("expected nil key when expired")
	}
}

func TestValidateKey_ShouldReturnError_WhenKeyNotFound(t *testing.T) {
	// Arrange
	repo := NewMockSystemKeyRepository()
	service := NewService(repo, Config{
		MaxKeys:    10,
		BcryptCost: BcryptCost,
	})

	// Use a key that doesn't exist
	plainKey := models.SystemKeyPrefix + "nonexistent123456"

	// Act
	validatedKey, err := service.ValidateKey(context.Background(), plainKey)

	// Assert
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if !errors.Is(err, models.ErrSystemKeyNotFound) {
		t.Errorf("expected ErrSystemKeyNotFound, got %v", err)
	}

	if validatedKey != nil {
		t.Error("expected nil key when not found")
	}
}

func TestValidateKey_ShouldReturnError_WhenInvalidFormat(t *testing.T) {
	// Arrange
	repo := NewMockSystemKeyRepository()
	service := NewService(repo, Config{
		MaxKeys:    10,
		BcryptCost: BcryptCost,
	})

	testCases := []struct {
		name     string
		plainKey string
	}{
		{
			name:     "too short",
			plainKey: "short",
		},
		{
			name:     "empty string",
			plainKey: "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Act
			validatedKey, err := service.ValidateKey(context.Background(), tc.plainKey)

			// Assert
			if err == nil {
				t.Fatal("expected error, got nil")
			}

			if !errors.Is(err, ErrInvalidKeyFormat) {
				t.Errorf("expected ErrInvalidKeyFormat, got %v", err)
			}

			if validatedKey != nil {
				t.Error("expected nil key with invalid format")
			}
		})
	}
}

func TestRevokeKey_ShouldRevokeKey_WhenKeyExists(t *testing.T) {
	// Arrange
	repo := NewMockSystemKeyRepository()
	service := NewService(repo, Config{
		MaxKeys:    10,
		BcryptCost: BcryptCost,
	})

	// Create a key
	createdBy := uuid.New()
	createResult, err := service.CreateKey(context.Background(), "test", "desc", "service", createdBy, nil)
	if err != nil {
		t.Fatalf("failed to create key: %v", err)
	}

	keyID := uuid.MustParse(createResult.Key.ID)

	// Act
	err = service.RevokeKey(context.Background(), keyID)

	// Assert
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Verify key is revoked
	key, _ := repo.FindByID(context.Background(), keyID)
	if key.Status != models.SystemKeyStatusRevoked {
		t.Errorf("expected status %s, got %s", models.SystemKeyStatusRevoked, key.Status)
	}

	if key.RevokedAt == nil {
		t.Error("expected revoked_at to be set")
	}
}

func TestRevokeKey_ShouldReturnError_WhenKeyNotFound(t *testing.T) {
	// Arrange
	repo := NewMockSystemKeyRepository()
	service := NewService(repo, Config{
		MaxKeys:    10,
		BcryptCost: BcryptCost,
	})

	nonExistentID := uuid.New()

	// Act
	err := service.RevokeKey(context.Background(), nonExistentID)

	// Assert
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if !errors.Is(err, models.ErrSystemKeyNotFound) {
		t.Errorf("expected ErrSystemKeyNotFound, got %v", err)
	}
}

func TestDeleteKey_ShouldDeleteKey_WhenKeyExists(t *testing.T) {
	// Arrange
	repo := NewMockSystemKeyRepository()
	service := NewService(repo, Config{
		MaxKeys:    10,
		BcryptCost: BcryptCost,
	})

	// Create a key
	createdBy := uuid.New()
	createResult, err := service.CreateKey(context.Background(), "test", "desc", "service", createdBy, nil)
	if err != nil {
		t.Fatalf("failed to create key: %v", err)
	}

	keyID := uuid.MustParse(createResult.Key.ID)

	// Act
	err = service.DeleteKey(context.Background(), keyID)

	// Assert
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Verify key is deleted
	key, _ := repo.FindByID(context.Background(), keyID)
	if key != nil {
		t.Error("expected key to be deleted")
	}
}

func TestDeleteKey_ShouldReturnError_WhenKeyNotFound(t *testing.T) {
	// Arrange
	repo := NewMockSystemKeyRepository()
	service := NewService(repo, Config{
		MaxKeys:    10,
		BcryptCost: BcryptCost,
	})

	nonExistentID := uuid.New()

	// Act
	err := service.DeleteKey(context.Background(), nonExistentID)

	// Assert
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if !errors.Is(err, models.ErrSystemKeyNotFound) {
		t.Errorf("expected ErrSystemKeyNotFound, got %v", err)
	}
}

func TestGetByID_ShouldReturnKey_WhenKeyExists(t *testing.T) {
	// Arrange
	repo := NewMockSystemKeyRepository()
	service := NewService(repo, Config{
		MaxKeys:    10,
		BcryptCost: BcryptCost,
	})

	// Create a key
	createdBy := uuid.New()
	createResult, err := service.CreateKey(context.Background(), "test", "desc", "service", createdBy, nil)
	if err != nil {
		t.Fatalf("failed to create key: %v", err)
	}

	keyID := uuid.MustParse(createResult.Key.ID)

	// Act
	key, err := service.GetByID(context.Background(), keyID)

	// Assert
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if key == nil {
		t.Fatal("expected key, got nil")
	}

	if key.ID != createResult.Key.ID {
		t.Errorf("expected key ID %s, got %s", createResult.Key.ID, key.ID)
	}
}

func TestGetByID_ShouldReturnError_WhenKeyNotFound(t *testing.T) {
	// Arrange
	repo := NewMockSystemKeyRepository()
	service := NewService(repo, Config{
		MaxKeys:    10,
		BcryptCost: BcryptCost,
	})

	nonExistentID := uuid.New()

	// Act
	key, err := service.GetByID(context.Background(), nonExistentID)

	// Assert
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if !errors.Is(err, models.ErrSystemKeyNotFound) {
		t.Errorf("expected ErrSystemKeyNotFound, got %v", err)
	}

	if key != nil {
		t.Error("expected nil key when not found")
	}
}

func TestListAll_ShouldReturnAllKeys(t *testing.T) {
	// Arrange
	repo := NewMockSystemKeyRepository()
	service := NewService(repo, Config{
		MaxKeys:    10,
		BcryptCost: BcryptCost,
	})

	// Create multiple keys
	createdBy := uuid.New()
	for i := 0; i < 3; i++ {
		_, err := service.CreateKey(context.Background(), "test", "desc", "service", createdBy, nil)
		if err != nil {
			t.Fatalf("failed to create key: %v", err)
		}
	}

	// Act
	keys, total, err := service.ListAll(context.Background(), repository.SystemKeyFilter{})

	// Assert
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(keys) != 3 {
		t.Errorf("expected 3 keys, got %d", len(keys))
	}

	if total != 3 {
		t.Errorf("expected total 3, got %d", total)
	}
}

func TestValidateKey_ShouldReturnError_WhenWrongKeyProvided(t *testing.T) {
	// Arrange
	repo := NewMockSystemKeyRepository()
	service := NewService(repo, Config{
		MaxKeys:    10,
		BcryptCost: BcryptCost,
	})

	// Create a key
	createdBy := uuid.New()
	key := models.NewSystemKey("test", "desc", "service", createdBy.String())

	// Generate a plain key and hash - both keys must share the same 10-char prefix
	// SystemKeyPrefix = "sysk_" (5 chars), so we need 5 more chars for the prefix
	plainKey := models.SystemKeyPrefix + "12345correctsuffix"
	keyHash := hashTestKey(plainKey)
	key.KeyHash = keyHash
	key.KeyPrefix = plainKey[:models.SystemKeyPrefixLength] // "sysk_12345"

	repo.Create(context.Background(), key)

	// Try to validate with wrong key (same 10-char prefix but different suffix)
	wrongKey := models.SystemKeyPrefix + "12345wrongsuffix"

	// Act
	validatedKey, err := service.ValidateKey(context.Background(), wrongKey)

	// Assert
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if !errors.Is(err, models.ErrInvalidSystemKey) {
		t.Errorf("expected ErrInvalidSystemKey, got %v", err)
	}

	if validatedKey != nil {
		t.Error("expected nil key with wrong key")
	}
}

func TestNewService_ShouldUseDefaultConfig_WhenConfigInvalid(t *testing.T) {
	// Arrange
	repo := NewMockSystemKeyRepository()

	// Act
	service := NewService(repo, Config{
		MaxKeys:    0,  // Invalid
		BcryptCost: 0,  // Invalid
	})

	// Assert
	if service.config.MaxKeys != DefaultMaxKeys {
		t.Errorf("expected MaxKeys %d, got %d", DefaultMaxKeys, service.config.MaxKeys)
	}

	if service.config.BcryptCost != BcryptCost {
		t.Errorf("expected BcryptCost %d, got %d", BcryptCost, service.config.BcryptCost)
	}
}

func TestCreateKey_ShouldApplyDefaultExpiry_WhenNoExpiryProvidedAndDefaultSet(t *testing.T) {
	// Arrange
	repo := NewMockSystemKeyRepository()
	defaultExpiryDays := 90
	service := NewService(repo, Config{
		MaxKeys:           10,
		BcryptCost:        BcryptCost,
		DefaultExpiryDays: defaultExpiryDays,
	})

	createdBy := uuid.New()

	// Act
	result, err := service.CreateKey(context.Background(), "test", "desc", "service", createdBy, nil)

	// Assert
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if result.Key.ExpiresAt == nil {
		t.Error("expected default expiration to be set")
	}

	expectedExpiration := time.Now().AddDate(0, 0, defaultExpiryDays)
	diff := result.Key.ExpiresAt.Sub(expectedExpiration)
	if diff > time.Minute || diff < -time.Minute {
		t.Errorf("expiration not set correctly, expected around %v, got %v", expectedExpiration, result.Key.ExpiresAt)
	}
}
