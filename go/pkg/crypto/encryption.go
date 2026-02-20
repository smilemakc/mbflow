package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"os"
	"sync"
)

var (
	// ErrInvalidKey is returned when the encryption key is invalid
	ErrInvalidKey = errors.New("invalid encryption key: must be 32 bytes for AES-256")
	// ErrInvalidCiphertext is returned when the ciphertext is too short or invalid
	ErrInvalidCiphertext = errors.New("invalid ciphertext")
	// ErrKeyNotConfigured is returned when encryption key is not set
	ErrKeyNotConfigured = errors.New("encryption key not configured")
)

const (
	// KeyEnvVar is the environment variable name for the encryption key
	KeyEnvVar = "MBFLOW_ENCRYPTION_KEY"
	// AES256KeySize is the required key size for AES-256
	AES256KeySize = 32
	// NonceSize is the standard nonce size for GCM
	NonceSize = 12
)

// EncryptionService provides AES-256-GCM encryption/decryption operations
type EncryptionService struct {
	key []byte
	mu  sync.RWMutex
}

var (
	defaultService *EncryptionService
	serviceOnce    sync.Once
)

// GetDefaultService returns the default encryption service configured from environment
func GetDefaultService() (*EncryptionService, error) {
	var initErr error
	serviceOnce.Do(func() {
		keyStr := os.Getenv(KeyEnvVar)
		if keyStr == "" {
			initErr = ErrKeyNotConfigured
			return
		}

		key, err := base64.StdEncoding.DecodeString(keyStr)
		if err != nil {
			initErr = fmt.Errorf("failed to decode encryption key: %w", err)
			return
		}

		defaultService, initErr = NewEncryptionService(key)
	})

	if initErr != nil {
		return nil, initErr
	}

	return defaultService, nil
}

// NewEncryptionService creates a new encryption service with the given key
func NewEncryptionService(key []byte) (*EncryptionService, error) {
	if len(key) != AES256KeySize {
		return nil, ErrInvalidKey
	}

	return &EncryptionService{
		key: key,
	}, nil
}

// GenerateKey generates a new random 32-byte key for AES-256
func GenerateKey() ([]byte, error) {
	key := make([]byte, AES256KeySize)
	if _, err := io.ReadFull(rand.Reader, key); err != nil {
		return nil, fmt.Errorf("failed to generate key: %w", err)
	}
	return key, nil
}

// GenerateKeyBase64 generates a new random key and returns it as base64
func GenerateKeyBase64() (string, error) {
	key, err := GenerateKey()
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(key), nil
}

// Encrypt encrypts plaintext using AES-256-GCM
// Returns base64-encoded ciphertext (nonce + encrypted data + auth tag)
func (s *EncryptionService) Encrypt(plaintext []byte) (string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	block, err := aes.NewCipher(s.key)
	if err != nil {
		return "", fmt.Errorf("failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("failed to create GCM: %w", err)
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("failed to generate nonce: %w", err)
	}

	ciphertext := gcm.Seal(nonce, nonce, plaintext, nil)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// EncryptString encrypts a string using AES-256-GCM
func (s *EncryptionService) EncryptString(plaintext string) (string, error) {
	return s.Encrypt([]byte(plaintext))
}

// Decrypt decrypts base64-encoded ciphertext using AES-256-GCM
func (s *EncryptionService) Decrypt(ciphertextBase64 string) ([]byte, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	ciphertext, err := base64.StdEncoding.DecodeString(ciphertextBase64)
	if err != nil {
		return nil, fmt.Errorf("failed to decode ciphertext: %w", err)
	}

	block, err := aes.NewCipher(s.key)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	if len(ciphertext) < gcm.NonceSize() {
		return nil, ErrInvalidCiphertext
	}

	nonce, ciphertext := ciphertext[:gcm.NonceSize()], ciphertext[gcm.NonceSize():]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt: %w", err)
	}

	return plaintext, nil
}

// DecryptString decrypts base64-encoded ciphertext and returns a string
func (s *EncryptionService) DecryptString(ciphertextBase64 string) (string, error) {
	plaintext, err := s.Decrypt(ciphertextBase64)
	if err != nil {
		return "", err
	}
	return string(plaintext), nil
}

// EncryptMap encrypts all string values in a map
func (s *EncryptionService) EncryptMap(data map[string]string) (map[string]string, error) {
	result := make(map[string]string, len(data))
	for k, v := range data {
		encrypted, err := s.EncryptString(v)
		if err != nil {
			return nil, fmt.Errorf("failed to encrypt key %q: %w", k, err)
		}
		result[k] = encrypted
	}
	return result, nil
}

// DecryptMap decrypts all string values in a map
func (s *EncryptionService) DecryptMap(data map[string]string) (map[string]string, error) {
	result := make(map[string]string, len(data))
	for k, v := range data {
		decrypted, err := s.DecryptString(v)
		if err != nil {
			return nil, fmt.Errorf("failed to decrypt key %q: %w", k, err)
		}
		result[k] = decrypted
	}
	return result, nil
}
