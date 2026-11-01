package crypto

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// rcResetDefault clears the memoized default service so GetDefaultService
// re-reads the environment.
func rcResetDefault(t *testing.T) {
	t.Helper()
	serviceOnce = sync.Once{}
	defaultService = nil
	defaultErr = nil
	t.Cleanup(func() {
		serviceOnce = sync.Once{}
		defaultService = nil
		defaultErr = nil
	})
}

func TestRCGetDefaultService(t *testing.T) {
	t.Run("not configured error is remembered", func(t *testing.T) {
		rcResetDefault(t)
		t.Setenv(KeyEnvVar, "")
		svc, err := GetDefaultService()
		assert.Nil(t, svc)
		assert.ErrorIs(t, err, ErrKeyNotConfigured)

		// Setting the key later does not change the memoized result.
		key, genErr := GenerateKeyBase64()
		require.NoError(t, genErr)
		t.Setenv(KeyEnvVar, key)
		svc, err = GetDefaultService()
		assert.Nil(t, svc)
		assert.ErrorIs(t, err, ErrKeyNotConfigured)
	})
	t.Run("invalid base64", func(t *testing.T) {
		rcResetDefault(t)
		t.Setenv(KeyEnvVar, "!!!not-base64!!!")
		_, err := GetDefaultService()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to decode encryption key")
	})
	t.Run("wrong key size", func(t *testing.T) {
		rcResetDefault(t)
		t.Setenv(KeyEnvVar, base64.StdEncoding.EncodeToString([]byte("too-short")))
		_, err := GetDefaultService()
		assert.ErrorIs(t, err, ErrInvalidKey)
	})
	t.Run("valid key returns a shared service", func(t *testing.T) {
		rcResetDefault(t)
		key, err := GenerateKeyBase64()
		require.NoError(t, err)
		t.Setenv(KeyEnvVar, key)

		svc, err := GetDefaultService()
		require.NoError(t, err)
		require.NotNil(t, svc)
		again, err := GetDefaultService()
		require.NoError(t, err)
		assert.Same(t, svc, again)

		ct, err := svc.EncryptString("hello")
		require.NoError(t, err)
		pt, err := again.DecryptString(ct)
		require.NoError(t, err)
		assert.Equal(t, "hello", pt)
	})
}

type rcFailingReader struct{}

func (rcFailingReader) Read(p []byte) (int, error) { return 0, errors.New("entropy exhausted") }

func TestRCRandomSourceFailures(t *testing.T) {
	orig := rand.Reader
	rand.Reader = rcFailingReader{}
	t.Cleanup(func() { rand.Reader = orig })

	_, err := GenerateKey()
	assert.ErrorContains(t, err, "failed to generate key")
	_, err = GenerateKeyBase64()
	assert.ErrorContains(t, err, "failed to generate key")

	svc, err := NewEncryptionService(make([]byte, AES256KeySize))
	require.NoError(t, err)
	_, err = svc.EncryptString("x")
	assert.ErrorContains(t, err, "failed to generate nonce")
}

func TestRCInvalidInternalKey(t *testing.T) {
	// A service constructed with a malformed key must fail cleanly rather than panic.
	svc := &EncryptionService{key: []byte("short")}
	_, err := svc.Encrypt([]byte("x"))
	assert.ErrorContains(t, err, "failed to create cipher")
	_, err = svc.Decrypt(base64.StdEncoding.EncodeToString(make([]byte, 32)))
	assert.ErrorContains(t, err, "failed to create cipher")
	_, err = svc.EncryptMap(map[string]string{"k": "v"})
	assert.ErrorContains(t, err, `failed to encrypt key "k"`)
}

func TestRCDecryptMapAndStringErrors(t *testing.T) {
	key, err := GenerateKey()
	require.NoError(t, err)
	svc, err := NewEncryptionService(key)
	require.NoError(t, err)

	_, err = svc.DecryptString("%%%")
	assert.ErrorContains(t, err, "failed to decode ciphertext")

	_, err = svc.Decrypt(base64.StdEncoding.EncodeToString([]byte("short")))
	assert.ErrorIs(t, err, ErrInvalidCiphertext)

	good, err := svc.EncryptString("v")
	require.NoError(t, err)
	_, err = svc.DecryptMap(map[string]string{"ok": good, "bad": "%%%"})
	assert.ErrorContains(t, err, `failed to decrypt key "bad"`)

	// Tampering with the ciphertext is detected by GCM authentication.
	raw, err := base64.StdEncoding.DecodeString(good)
	require.NoError(t, err)
	raw[len(raw)-1] ^= 0xFF
	_, err = svc.Decrypt(base64.StdEncoding.EncodeToString(raw))
	assert.ErrorContains(t, err, "failed to decrypt")

	empty, err := svc.EncryptMap(map[string]string{})
	require.NoError(t, err)
	assert.Empty(t, empty)
}
