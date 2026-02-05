package systemkey

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
	KeyLength      = 32
	BcryptCost     = 10
	DefaultMaxKeys = 50
)

var (
	ErrInvalidKeyFormat    = errors.New("invalid system key format")
	ErrKeyGenerationFailed = errors.New("failed to generate system key")
)

type Config struct {
	MaxKeys           int
	DefaultExpiryDays int
	BcryptCost        int
}

type Service struct {
	repo   repository.SystemKeyRepository
	config Config
}

func NewService(repo repository.SystemKeyRepository, config Config) *Service {
	if config.MaxKeys <= 0 {
		config.MaxKeys = DefaultMaxKeys
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
	Key      *models.SystemKey
	PlainKey string
}

func (s *Service) CreateKey(ctx context.Context, name, description, serviceName string, createdBy uuid.UUID, expiresInDays *int) (*CreateResult, error) {
	count, err := s.repo.Count(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to count system keys: %w", err)
	}

	if int(count) >= s.config.MaxKeys {
		return nil, models.ErrSystemKeyLimitReached
	}

	plainKey, keyPrefix, err := generatePlainKey()
	if err != nil {
		return nil, err
	}

	keyHash, err := s.hashKey(plainKey)
	if err != nil {
		return nil, fmt.Errorf("failed to hash key: %w", err)
	}

	key := models.NewSystemKey(name, description, serviceName, createdBy.String())
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
		return nil, fmt.Errorf("failed to create system key: %w", err)
	}

	return &CreateResult{
		Key:      key,
		PlainKey: plainKey,
	}, nil
}

func (s *Service) ValidateKey(ctx context.Context, plainKey string) (*models.SystemKey, error) {
	if len(plainKey) < models.SystemKeyPrefixLength {
		return nil, ErrInvalidKeyFormat
	}

	prefix := plainKey[:models.SystemKeyPrefixLength]

	keys, err := s.repo.FindByPrefix(ctx, prefix)
	if err != nil {
		return nil, fmt.Errorf("failed to find keys by prefix: %w", err)
	}

	if len(keys) == 0 {
		return nil, models.ErrSystemKeyNotFound
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

	return nil, models.ErrInvalidSystemKey
}

func (s *Service) GetByID(ctx context.Context, id uuid.UUID) (*models.SystemKey, error) {
	key, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get system key: %w", err)
	}
	if key == nil {
		return nil, models.ErrSystemKeyNotFound
	}
	return key, nil
}

func (s *Service) ListAll(ctx context.Context, filter repository.SystemKeyFilter) ([]*models.SystemKey, int64, error) {
	keys, total, err := s.repo.FindAll(ctx, filter)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list system keys: %w", err)
	}
	return keys, total, nil
}

func (s *Service) RevokeKey(ctx context.Context, id uuid.UUID) error {
	key, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to find system key: %w", err)
	}
	if key == nil {
		return models.ErrSystemKeyNotFound
	}

	if err := s.repo.Revoke(ctx, id); err != nil {
		return fmt.Errorf("failed to revoke system key: %w", err)
	}

	return nil
}

func (s *Service) DeleteKey(ctx context.Context, id uuid.UUID) error {
	key, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to find system key: %w", err)
	}
	if key == nil {
		return models.ErrSystemKeyNotFound
	}

	if err := s.repo.Delete(ctx, id); err != nil {
		return fmt.Errorf("failed to delete system key: %w", err)
	}

	return nil
}

func generatePlainKey() (string, string, error) {
	randomBytes := make([]byte, KeyLength)
	if _, err := rand.Read(randomBytes); err != nil {
		return "", "", fmt.Errorf("%w: %v", ErrKeyGenerationFailed, err)
	}

	encoded := base64.RawURLEncoding.EncodeToString(randomBytes)

	plainKey := models.SystemKeyPrefix + encoded

	prefix := plainKey
	if len(prefix) > models.SystemKeyPrefixLength {
		prefix = prefix[:models.SystemKeyPrefixLength]
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
