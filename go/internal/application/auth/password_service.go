package auth

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"unicode"

	"golang.org/x/crypto/bcrypt"
)

var (
	ErrPasswordTooShort    = errors.New("password is too short")
	ErrPasswordTooWeak     = errors.New("password is too weak")
	ErrPasswordMismatch    = errors.New("password does not match")
	ErrInvalidPasswordHash = errors.New("invalid password hash")
)

// PasswordError represents a password validation error
type PasswordError struct {
	Message string
}

func (e *PasswordError) Error() string {
	return e.Message
}

// PasswordService handles password hashing and validation
type PasswordService struct {
	minLength        int
	requireUppercase bool
	requireLowercase bool
	requireDigit     bool
	requireSpecial   bool
	bcryptCost       int
}

// NewPasswordService creates a new PasswordService with default settings
func NewPasswordService(minLength int) *PasswordService {
	if minLength < 6 {
		minLength = 6
	}

	return &PasswordService{
		minLength:        minLength,
		requireUppercase: true,
		requireLowercase: true,
		requireDigit:     true,
		requireSpecial:   false, // Optional by default
		bcryptCost:       bcrypt.DefaultCost,
	}
}

// HashPassword creates a bcrypt hash of the password
func (s *PasswordService) HashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), s.bcryptCost)
	if err != nil {
		return "", fmt.Errorf("failed to hash password: %w", err)
	}
	return string(hash), nil
}

// VerifyPassword compares a password with its hash
func (s *PasswordService) VerifyPassword(password, hash string) error {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	if err != nil {
		if errors.Is(err, bcrypt.ErrMismatchedHashAndPassword) {
			return ErrPasswordMismatch
		}
		return ErrInvalidPasswordHash
	}
	return nil
}

// ValidatePassword checks if a password meets the requirements
func (s *PasswordService) ValidatePassword(password string) error {
	if len(password) < s.minLength {
		return fmt.Errorf("%w: minimum %d characters required", ErrPasswordTooShort, s.minLength)
	}

	var (
		hasUpper   bool
		hasLower   bool
		hasDigit   bool
		hasSpecial bool
	)

	for _, char := range password {
		switch {
		case unicode.IsUpper(char):
			hasUpper = true
		case unicode.IsLower(char):
			hasLower = true
		case unicode.IsDigit(char):
			hasDigit = true
		case unicode.IsPunct(char) || unicode.IsSymbol(char):
			hasSpecial = true
		}
	}

	var missing []string

	if s.requireUppercase && !hasUpper {
		missing = append(missing, "uppercase letter")
	}
	if s.requireLowercase && !hasLower {
		missing = append(missing, "lowercase letter")
	}
	if s.requireDigit && !hasDigit {
		missing = append(missing, "digit")
	}
	if s.requireSpecial && !hasSpecial {
		missing = append(missing, "special character")
	}

	if len(missing) > 0 {
		return fmt.Errorf("%w: must contain %s", ErrPasswordTooWeak, strings.Join(missing, ", "))
	}

	return nil
}

// GenerateResetToken generates a random password reset token
func (s *PasswordService) GenerateResetToken() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("failed to generate reset token: %w", err)
	}
	return hex.EncodeToString(bytes), nil
}

// GenerateRandomPassword generates a random password that meets requirements
func (s *PasswordService) GenerateRandomPassword(length int) (string, error) {
	if length < s.minLength {
		length = s.minLength
	}

	// Character sets
	const (
		uppercase = "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
		lowercase = "abcdefghijklmnopqrstuvwxyz"
		digits    = "0123456789"
		special   = "!@#$%^&*()_+-=[]{}|;:,.<>?"
	)

	// Build character set based on requirements
	var charset string
	var required []byte

	if s.requireUppercase {
		charset += uppercase
		required = append(required, randomChar(uppercase))
	}
	if s.requireLowercase {
		charset += lowercase
		required = append(required, randomChar(lowercase))
	}
	if s.requireDigit {
		charset += digits
		required = append(required, randomChar(digits))
	}
	if s.requireSpecial {
		charset += special
		required = append(required, randomChar(special))
	}

	// Generate remaining characters
	remaining := length - len(required)
	password := make([]byte, remaining)
	for i := range password {
		password[i] = randomChar(charset)
	}

	// Combine required and random characters
	password = append(password, required...)

	// Shuffle the password
	shuffle(password)

	return string(password), nil
}

// randomChar returns a random character from the given string
func randomChar(charset string) byte {
	bytes := make([]byte, 1)
	rand.Read(bytes)
	return charset[int(bytes[0])%len(charset)]
}

// shuffle randomly shuffles a byte slice
func shuffle(b []byte) {
	for i := len(b) - 1; i > 0; i-- {
		bytes := make([]byte, 1)
		rand.Read(bytes)
		j := int(bytes[0]) % (i + 1)
		b[i], b[j] = b[j], b[i]
	}
}
