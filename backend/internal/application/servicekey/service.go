package servicekey

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"github.com/smilemakc/mbflow/internal/domain/repository"
	"github.com/smilemakc/mbflow/pkg/models"
)

const (
	KeyLength             = 32
	BcryptCost            = 10
	DefaultMaxKeysPerUser = 10
)

var (
	ErrInvalidKeyFormat    = errors.New("invalid service key format")
	ErrKeyGenerationFailed = errors.New("failed to generate service key")
)

type Config struct {
	MaxKeysPerUser    int
	DefaultExpiryDays int
	BcryptCost        int
}

type Service struct {
	repo   repository.ServiceKeyRepository
	config Config
}

func NewService(repo repository.ServiceKeyRepository, config Config) *Service {
	if config.MaxKeysPerUser <= 0 {
		config.MaxKeysPerUser = DefaultMaxKeysPerUser
	}
	if config.BcryptCost <= 0 {
		config.BcryptCost = BcryptCost
	}
	return &Service{
		repo:   repo,
		config: config,
	}
}

type CreateResult struct {
	Key      *models.ServiceKey
	PlainKey string
}

func (s *Service) CreateKey(ctx context.Context, userID uuid.UUID, name, description string, createdBy uuid.UUID, expiresInDays *int) (*CreateResult, error) {
	count, err := s.repo.CountByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to count user keys: %w", err)
	}

	if int(count) >= s.config.MaxKeysPerUser {
		return nil, models.ErrServiceKeyLimitReached
	}

	plainKey, keyPrefix, err := generatePlainKey()
	if err != nil {
		return nil, err
	}

	keyHash, err := s.hashKey(plainKey)
	if err != nil {
		return nil, fmt.Errorf("failed to hash key: %w", err)
	}

	key := models.NewServiceKey(userID.String(), name, description, createdBy.String())
	key.KeyPrefix = keyPrefix
	key.KeyHash = keyHash

	if expiresInDays != nil && *expiresInDays > 0 {
		expiresAt := time.Now().AddDate(0, 0, *expiresInDays)
		if err := key.SetExpiration(expiresAt); err != nil {
			return nil, fmt.Errorf("failed to set expiration: %w", err)
		}
	} else if s.config.DefaultExpiryDays > 0 {
		expiresAt := time.Now().AddDate(0, 0, s.config.DefaultExpiryDays)
		if err := key.SetExpiration(expiresAt); err != nil {
			return nil, fmt.Errorf("failed to set default expiration: %w", err)
		}
	}

	if err := key.Validate(); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	if err := s.repo.Create(ctx, key); err != nil {
		return nil, fmt.Errorf("failed to create service key: %w", err)
	}

	return &CreateResult{
		Key:      key,
		PlainKey: plainKey,
	}, nil
}

func (s *Service) ValidateKey(ctx context.Context, plainKey string) (*models.ServiceKey, error) {
	if len(plainKey) < models.ServiceKeyPrefixLength {
		return nil, ErrInvalidKeyFormat
	}

	prefix := plainKey[:models.ServiceKeyPrefixLength]

	keys, err := s.repo.FindByPrefix(ctx, prefix)
	if err != nil {
		return nil, fmt.Errorf("failed to find keys by prefix: %w", err)
	}

	if len(keys) == 0 {
		return nil, models.ErrServiceKeyNotFound
	}

	for _, key := range keys {
		if verifyKey(plainKey, key.KeyHash) {
			if err := key.CanUse(); err != nil {
				return nil, err
			}

			if err := s.repo.UpdateLastUsed(ctx, uuid.MustParse(key.ID)); err != nil {
				return nil, fmt.Errorf("failed to update last used: %w", err)
			}

			key.IncrementUsage()

			return key, nil
		}
	}

	return nil, models.ErrInvalidServiceKey
}

func (s *Service) GetByID(ctx context.Context, id uuid.UUID) (*models.ServiceKey, error) {
	key, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get service key: %w", err)
	}
	if key == nil {
		return nil, models.ErrServiceKeyNotFound
	}
	return key, nil
}

func (s *Service) GetByUserID(ctx context.Context, userID uuid.UUID) ([]*models.ServiceKey, error) {
	keys, err := s.repo.FindByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user keys: %w", err)
	}
	return keys, nil
}

func (s *Service) ListAll(ctx context.Context, filter repository.ServiceKeyFilter) ([]*models.ServiceKey, int64, error) {
	keys, total, err := s.repo.FindAll(ctx, filter)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list service keys: %w", err)
	}
	return keys, total, nil
}

func (s *Service) RevokeKey(ctx context.Context, id uuid.UUID) error {
	key, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to find service key: %w", err)
	}
	if key == nil {
		return models.ErrServiceKeyNotFound
	}

	if err := s.repo.Revoke(ctx, id); err != nil {
		return fmt.Errorf("failed to revoke service key: %w", err)
	}

	return nil
}

func (s *Service) DeleteKey(ctx context.Context, id uuid.UUID) error {
	key, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to find service key: %w", err)
	}
	if key == nil {
		return models.ErrServiceKeyNotFound
	}

	if err := s.repo.Delete(ctx, id); err != nil {
		return fmt.Errorf("failed to delete service key: %w", err)
	}

	return nil
}

func generatePlainKey() (string, string, error) {
	randomBytes := make([]byte, KeyLength)
	if _, err := rand.Read(randomBytes); err != nil {
		return "", "", fmt.Errorf("%w: %v", ErrKeyGenerationFailed, err)
	}

	encoded := base64.RawURLEncoding.EncodeToString(randomBytes)

	plainKey := models.ServiceKeyPrefix + encoded

	prefix := plainKey
	if len(prefix) > models.ServiceKeyPrefixLength {
		prefix = prefix[:models.ServiceKeyPrefixLength]
	}

	return plainKey, prefix, nil
}

func (s *Service) hashKey(plainKey string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(plainKey), s.config.BcryptCost)
	if err != nil {
		return "", fmt.Errorf("bcrypt hash failed: %w", err)
	}
	return string(hash), nil
}

func verifyKey(plainKey, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(plainKey))
	return err == nil
}
