package crypto

import (
	"encoding/base64"
	"testing"
)

func TestGenerateKey(t *testing.T) {
	key, err := GenerateKey()
	if err != nil {
		t.Fatalf("GenerateKey() error = %v", err)
	}

	if len(key) != AES256KeySize {
		t.Errorf("GenerateKey() returned key of length %d, want %d", len(key), AES256KeySize)
	}

	// Generate another key and ensure they're different
	key2, err := GenerateKey()
	if err != nil {
		t.Fatalf("GenerateKey() second call error = %v", err)
	}

	if string(key) == string(key2) {
		t.Error("GenerateKey() returned same key twice")
	}
}

func TestGenerateKeyBase64(t *testing.T) {
	keyBase64, err := GenerateKeyBase64()
	if err != nil {
		t.Fatalf("GenerateKeyBase64() error = %v", err)
	}

	key, err := base64.StdEncoding.DecodeString(keyBase64)
	if err != nil {
		t.Fatalf("Failed to decode base64 key: %v", err)
	}

	if len(key) != AES256KeySize {
		t.Errorf("GenerateKeyBase64() returned key of length %d, want %d", len(key), AES256KeySize)
	}
}

func TestNewEncryptionService(t *testing.T) {
	tests := []struct {
		name    string
		keyLen  int
		wantErr bool
	}{
		{"valid key", 32, false},
		{"too short", 16, true},
		{"too long", 64, true},
		{"empty", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key := make([]byte, tt.keyLen)
			_, err := NewEncryptionService(key)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewEncryptionService() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestEncryptDecrypt(t *testing.T) {
	key, err := GenerateKey()
	if err != nil {
		t.Fatalf("Failed to generate key: %v", err)
	}

	svc, err := NewEncryptionService(key)
	if err != nil {
		t.Fatalf("Failed to create service: %v", err)
	}

	tests := []struct {
		name      string
		plaintext string
	}{
		{"empty string", ""},
		{"simple text", "Hello, World!"},
		{"unicode text", "Привет, мир! 🌍"},
		{"json data", `{"api_key": "sk-12345", "secret": "my-secret-value"}`},
		{"long text", string(make([]byte, 10000))},
		{"special chars", "!@#$%^&*()_+-=[]{}|;':\",./<>?"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			encrypted, err := svc.EncryptString(tt.plaintext)
			if err != nil {
				t.Fatalf("EncryptString() error = %v", err)
			}

			// Encrypted text should be different from plaintext
			if encrypted == tt.plaintext && tt.plaintext != "" {
				t.Error("EncryptString() returned plaintext unchanged")
			}

			// Encrypted text should be base64
			_, err = base64.StdEncoding.DecodeString(encrypted)
			if err != nil {
				t.Errorf("EncryptString() returned non-base64 string: %v", err)
			}

			decrypted, err := svc.DecryptString(encrypted)
			if err != nil {
				t.Fatalf("DecryptString() error = %v", err)
			}

			if decrypted != tt.plaintext {
				t.Errorf("DecryptString() = %q, want %q", decrypted, tt.plaintext)
			}
		})
	}
}

func TestEncryptDecryptBytes(t *testing.T) {
	key, err := GenerateKey()
	if err != nil {
		t.Fatalf("Failed to generate key: %v", err)
	}

	svc, err := NewEncryptionService(key)
	if err != nil {
		t.Fatalf("Failed to create service: %v", err)
	}

	// Test binary data
	plaintext := []byte{0x00, 0x01, 0x02, 0xFF, 0xFE, 0xFD}

	encrypted, err := svc.Encrypt(plaintext)
	if err != nil {
		t.Fatalf("Encrypt() error = %v", err)
	}

	decrypted, err := svc.Decrypt(encrypted)
	if err != nil {
		t.Fatalf("Decrypt() error = %v", err)
	}

	if string(decrypted) != string(plaintext) {
		t.Errorf("Decrypt() = %v, want %v", decrypted, plaintext)
	}
}

func TestEncryptUniqueness(t *testing.T) {
	key, err := GenerateKey()
	if err != nil {
		t.Fatalf("Failed to generate key: %v", err)
	}

	svc, err := NewEncryptionService(key)
	if err != nil {
		t.Fatalf("Failed to create service: %v", err)
	}

	plaintext := "same-text"

	// Encrypt the same text multiple times
	encrypted1, _ := svc.EncryptString(plaintext)
	encrypted2, _ := svc.EncryptString(plaintext)
	encrypted3, _ := svc.EncryptString(plaintext)

	// Each encryption should produce different ciphertext (due to random nonce)
	if encrypted1 == encrypted2 || encrypted2 == encrypted3 || encrypted1 == encrypted3 {
		t.Error("Multiple encryptions of same plaintext should produce different ciphertexts")
	}

	// But all should decrypt to the same plaintext
	for i, enc := range []string{encrypted1, encrypted2, encrypted3} {
		dec, err := svc.DecryptString(enc)
		if err != nil {
			t.Errorf("Decryption %d failed: %v", i+1, err)
		}
		if dec != plaintext {
			t.Errorf("Decryption %d = %q, want %q", i+1, dec, plaintext)
		}
	}
}

func TestDecryptWithWrongKey(t *testing.T) {
	key1, _ := GenerateKey()
	key2, _ := GenerateKey()

	svc1, _ := NewEncryptionService(key1)
	svc2, _ := NewEncryptionService(key2)

	plaintext := "secret data"
	encrypted, _ := svc1.EncryptString(plaintext)

	// Try to decrypt with wrong key
	_, err := svc2.DecryptString(encrypted)
	if err == nil {
		t.Error("DecryptString() should fail with wrong key")
	}
}

func TestDecryptInvalidCiphertext(t *testing.T) {
	key, _ := GenerateKey()
	svc, _ := NewEncryptionService(key)

	tests := []struct {
		name       string
		ciphertext string
	}{
		{"empty", ""},
		{"not base64", "not-valid-base64!@#"},
		{"too short", base64.StdEncoding.EncodeToString([]byte("short"))},
		{"corrupted", "YWJjZGVmZ2hpamtsbW5vcHFyc3R1dnd4eXo="},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := svc.DecryptString(tt.ciphertext)
			if err == nil {
				t.Error("DecryptString() should fail with invalid ciphertext")
			}
		})
	}
}

func TestEncryptDecryptMap(t *testing.T) {
	key, _ := GenerateKey()
	svc, _ := NewEncryptionService(key)

	original := map[string]string{
		"api_key":       "sk-12345-secret-key",
		"client_secret": "my-client-secret",
		"password":      "super-secure-password",
	}

	encrypted, err := svc.EncryptMap(original)
	if err != nil {
		t.Fatalf("EncryptMap() error = %v", err)
	}

	// Verify all values are encrypted (different from original)
	for k, v := range original {
		if encrypted[k] == v {
			t.Errorf("Value for key %q was not encrypted", k)
		}
	}

	decrypted, err := svc.DecryptMap(encrypted)
	if err != nil {
		t.Fatalf("DecryptMap() error = %v", err)
	}

	// Verify decrypted values match original
	for k, v := range original {
		if decrypted[k] != v {
			t.Errorf("Decrypted[%q] = %q, want %q", k, decrypted[k], v)
		}
	}
}

func BenchmarkEncrypt(b *testing.B) {
	key, _ := GenerateKey()
	svc, _ := NewEncryptionService(key)
	plaintext := "benchmark-plaintext-data-for-encryption"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = svc.EncryptString(plaintext)
	}
}

func BenchmarkDecrypt(b *testing.B) {
	key, _ := GenerateKey()
	svc, _ := NewEncryptionService(key)
	encrypted, _ := svc.EncryptString("benchmark-plaintext-data-for-decryption")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = svc.DecryptString(encrypted)
	}
}
