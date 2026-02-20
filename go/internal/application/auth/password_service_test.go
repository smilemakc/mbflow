package auth

import (
	"strings"
	"testing"
	"unicode"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- NewPasswordService ---

func TestNewPasswordService_ShouldUseProvidedMinLength_WhenLengthAboveMinimum(t *testing.T) {
	// Arrange & Act
	svc := NewPasswordService(10)

	// Assert
	assert.Equal(t, 10, svc.minLength)
	assert.True(t, svc.requireUppercase)
	assert.True(t, svc.requireLowercase)
	assert.True(t, svc.requireDigit)
	assert.False(t, svc.requireSpecial)
}

func TestNewPasswordService_ShouldEnforceMinimumSix_WhenLengthBelowMinimum(t *testing.T) {
	// Arrange & Act
	svc := NewPasswordService(3)

	// Assert
	assert.Equal(t, 6, svc.minLength)
}

func TestNewPasswordService_ShouldEnforceMinimumSix_WhenLengthIsZero(t *testing.T) {
	// Arrange & Act
	svc := NewPasswordService(0)

	// Assert
	assert.Equal(t, 6, svc.minLength)
}

func TestNewPasswordService_ShouldEnforceMinimumSix_WhenLengthIsNegative(t *testing.T) {
	// Arrange & Act
	svc := NewPasswordService(-5)

	// Assert
	assert.Equal(t, 6, svc.minLength)
}

func TestNewPasswordService_ShouldAcceptExactlySix_WhenLengthIsSix(t *testing.T) {
	// Arrange & Act
	svc := NewPasswordService(6)

	// Assert
	assert.Equal(t, 6, svc.minLength)
}

func TestNewPasswordService_ShouldAcceptFive_AsClampedToSix(t *testing.T) {
	// Arrange & Act
	svc := NewPasswordService(5)

	// Assert
	assert.Equal(t, 6, svc.minLength)
}

// --- HashPassword ---

func TestHashPassword_ShouldReturnBcryptHash_WhenPasswordValid(t *testing.T) {
	// Arrange
	svc := NewPasswordService(8)

	// Act
	hash, err := svc.HashPassword("TestPass1")

	// Assert
	require.NoError(t, err)
	assert.NotEmpty(t, hash)
	assert.True(t, strings.HasPrefix(hash, "$2a$") || strings.HasPrefix(hash, "$2b$"),
		"hash should be a bcrypt hash, got: %s", hash)
}

func TestHashPassword_ShouldProduceDifferentHashes_ForSamePassword(t *testing.T) {
	// Arrange
	svc := NewPasswordService(8)
	password := "TestPass1"

	// Act
	hash1, err1 := svc.HashPassword(password)
	hash2, err2 := svc.HashPassword(password)

	// Assert
	require.NoError(t, err1)
	require.NoError(t, err2)
	assert.NotEqual(t, hash1, hash2, "bcrypt should produce different hashes due to random salt")
}

func TestHashPassword_ShouldSucceed_WhenPasswordEmpty(t *testing.T) {
	// Arrange
	svc := NewPasswordService(8)

	// Act
	hash, err := svc.HashPassword("")

	// Assert
	require.NoError(t, err)
	assert.NotEmpty(t, hash)
}

func TestHashPassword_ShouldReturnError_WhenPasswordExceedsBcryptLimit(t *testing.T) {
	// Arrange - bcrypt has a 72-byte limit; this implementation returns an error
	svc := NewPasswordService(8)
	longPassword := strings.Repeat("A", 100)

	// Act
	hash, err := svc.HashPassword(longPassword)

	// Assert
	require.Error(t, err)
	assert.Empty(t, hash)
	assert.Contains(t, err.Error(), "failed to hash password")
}

func TestHashPassword_ShouldSucceed_WhenPasswordAtBcryptLimit(t *testing.T) {
	// Arrange - exactly 72 bytes is the bcrypt maximum
	svc := NewPasswordService(8)
	password := strings.Repeat("A", 72)

	// Act
	hash, err := svc.HashPassword(password)

	// Assert
	require.NoError(t, err)
	assert.NotEmpty(t, hash)

	// Verify round-trip
	assert.NoError(t, svc.VerifyPassword(password, hash))
}

// --- VerifyPassword ---

func TestVerifyPassword_ShouldReturnNil_WhenPasswordMatchesHash(t *testing.T) {
	// Arrange
	svc := NewPasswordService(8)
	password := "Correct1Pass"
	hash, err := svc.HashPassword(password)
	require.NoError(t, err)

	// Act
	verifyErr := svc.VerifyPassword(password, hash)

	// Assert
	assert.NoError(t, verifyErr)
}

func TestVerifyPassword_ShouldReturnMismatch_WhenPasswordWrong(t *testing.T) {
	// Arrange
	svc := NewPasswordService(8)
	hash, err := svc.HashPassword("CorrectPass1")
	require.NoError(t, err)

	// Act
	verifyErr := svc.VerifyPassword("WrongPass1", hash)

	// Assert
	require.Error(t, verifyErr)
	assert.ErrorIs(t, verifyErr, ErrPasswordMismatch)
}

func TestVerifyPassword_ShouldReturnInvalidHash_WhenHashCorrupted(t *testing.T) {
	// Arrange
	svc := NewPasswordService(8)

	// Act
	verifyErr := svc.VerifyPassword("SomePass1", "not-a-valid-bcrypt-hash")

	// Assert
	require.Error(t, verifyErr)
	assert.ErrorIs(t, verifyErr, ErrInvalidPasswordHash)
}

func TestVerifyPassword_ShouldReturnInvalidHash_WhenHashEmpty(t *testing.T) {
	// Arrange
	svc := NewPasswordService(8)

	// Act
	verifyErr := svc.VerifyPassword("SomePass1", "")

	// Assert
	require.Error(t, verifyErr)
	assert.ErrorIs(t, verifyErr, ErrInvalidPasswordHash)
}

func TestVerifyPassword_ShouldReturnMismatch_WhenPasswordEmpty(t *testing.T) {
	// Arrange
	svc := NewPasswordService(8)
	hash, err := svc.HashPassword("ValidPass1")
	require.NoError(t, err)

	// Act
	verifyErr := svc.VerifyPassword("", hash)

	// Assert
	require.Error(t, verifyErr)
	assert.ErrorIs(t, verifyErr, ErrPasswordMismatch)
}

func TestVerifyPassword_ShouldSucceed_WhenBothPasswordAndHashFromEmptyString(t *testing.T) {
	// Arrange
	svc := NewPasswordService(8)
	hash, err := svc.HashPassword("")
	require.NoError(t, err)

	// Act
	verifyErr := svc.VerifyPassword("", hash)

	// Assert
	assert.NoError(t, verifyErr)
}

// --- ValidatePassword ---

func TestValidatePassword_ShouldReturnNil_WhenPasswordMeetsAllRequirements(t *testing.T) {
	// Arrange
	svc := NewPasswordService(8)

	// Act
	err := svc.ValidatePassword("StrongP1")

	// Assert
	assert.NoError(t, err)
}

func TestValidatePassword_ShouldReturnTooShort_WhenPasswordBelowMinLength(t *testing.T) {
	// Arrange
	svc := NewPasswordService(10)

	// Act
	err := svc.ValidatePassword("Short1Aa")

	// Assert
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrPasswordTooShort)
	assert.Contains(t, err.Error(), "minimum 10 characters required")
}

func TestValidatePassword_ShouldReturnTooShort_WhenPasswordEmpty(t *testing.T) {
	// Arrange
	svc := NewPasswordService(8)

	// Act
	err := svc.ValidatePassword("")

	// Assert
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrPasswordTooShort)
}

func TestValidatePassword_ShouldReturnTooWeak_WhenMissingUppercase(t *testing.T) {
	// Arrange
	svc := NewPasswordService(8)

	// Act - all lowercase + digit, no uppercase
	err := svc.ValidatePassword("alllower1")

	// Assert
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrPasswordTooWeak)
	assert.Contains(t, err.Error(), "uppercase letter")
}

func TestValidatePassword_ShouldReturnTooWeak_WhenMissingLowercase(t *testing.T) {
	// Arrange
	svc := NewPasswordService(8)

	// Act - all uppercase + digit, no lowercase
	err := svc.ValidatePassword("ALLUPPER1")

	// Assert
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrPasswordTooWeak)
	assert.Contains(t, err.Error(), "lowercase letter")
}

func TestValidatePassword_ShouldReturnTooWeak_WhenMissingDigit(t *testing.T) {
	// Arrange
	svc := NewPasswordService(8)

	// Act - letters only, no digit
	err := svc.ValidatePassword("NoDigitHere")

	// Assert
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrPasswordTooWeak)
	assert.Contains(t, err.Error(), "digit")
}

func TestValidatePassword_ShouldReturnTooWeak_WhenMissingMultipleRequirements(t *testing.T) {
	// Arrange
	svc := NewPasswordService(8)

	// Act - only lowercase, no uppercase and no digit
	err := svc.ValidatePassword("onlylowercase")

	// Assert
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrPasswordTooWeak)
	assert.Contains(t, err.Error(), "uppercase letter")
	assert.Contains(t, err.Error(), "digit")
}

func TestValidatePassword_ShouldNotRequireSpecial_ByDefault(t *testing.T) {
	// Arrange
	svc := NewPasswordService(8)

	// Act - valid password without special characters
	err := svc.ValidatePassword("ValidPas1")

	// Assert
	assert.NoError(t, err)
}

func TestValidatePassword_ShouldRequireSpecial_WhenExplicitlyEnabled(t *testing.T) {
	// Arrange
	svc := NewPasswordService(8)
	svc.requireSpecial = true

	// Act - no special character
	err := svc.ValidatePassword("NoSpecial1A")

	// Assert
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrPasswordTooWeak)
	assert.Contains(t, err.Error(), "special character")
}

func TestValidatePassword_ShouldPass_WhenSpecialRequiredAndPresent(t *testing.T) {
	// Arrange
	svc := NewPasswordService(8)
	svc.requireSpecial = true

	// Act
	err := svc.ValidatePassword("Valid1Pa!")

	// Assert
	assert.NoError(t, err)
}

func TestValidatePassword_ShouldReturnTooShort_WhenExactlyOneBelowMinLength(t *testing.T) {
	// Arrange
	svc := NewPasswordService(8)

	// Act - 7 characters, exactly one below minimum
	err := svc.ValidatePassword("Abcde1Z")

	// Assert
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrPasswordTooShort)
}

func TestValidatePassword_ShouldPass_WhenExactlyAtMinLength(t *testing.T) {
	// Arrange
	svc := NewPasswordService(8)

	// Act - exactly 8 characters meeting all requirements
	err := svc.ValidatePassword("Abcdef1Z")

	// Assert
	assert.NoError(t, err)
}

func TestValidatePassword_ShouldCheckLengthInBytes_NotRunes(t *testing.T) {
	// Arrange - The implementation uses len(password) which counts bytes, not runes.
	// Multi-byte unicode characters will count as more than 1.
	svc := NewPasswordService(8)

	// A string with 5 runes but more than 8 bytes due to multi-byte characters
	// Each of these runes is 3 bytes in UTF-8, so "AaAa1" + 2 multi-byte chars
	password := "Aa1\u00e9\u00fc\u00f1\u00e8\u00e0" // 3 ASCII + 5 x 2-byte = 13 bytes, but 8 runes

	// Act
	err := svc.ValidatePassword(password)

	// Assert - should pass because byte length > 8
	assert.NoError(t, err)
}

// --- GenerateResetToken ---

func TestGenerateResetToken_ShouldReturnHexString(t *testing.T) {
	// Arrange
	svc := NewPasswordService(8)

	// Act
	token, err := svc.GenerateResetToken()

	// Assert
	require.NoError(t, err)
	assert.Len(t, token, 64, "32 bytes encoded as hex should be 64 characters")

	// Verify it's valid hex (only 0-9, a-f)
	for _, ch := range token {
		assert.True(t, (ch >= '0' && ch <= '9') || (ch >= 'a' && ch <= 'f'),
			"token should only contain hex characters, got: %c", ch)
	}
}

func TestGenerateResetToken_ShouldProduceUniqueTokens(t *testing.T) {
	// Arrange
	svc := NewPasswordService(8)
	tokenCount := 50
	tokens := make(map[string]bool, tokenCount)

	// Act
	for i := 0; i < tokenCount; i++ {
		token, err := svc.GenerateResetToken()
		require.NoError(t, err)
		tokens[token] = true
	}

	// Assert - all tokens should be unique
	assert.Len(t, tokens, tokenCount, "all generated tokens should be unique")
}

// --- GenerateRandomPassword ---

func TestGenerateRandomPassword_ShouldMeetMinLength_WhenRequestedLengthBelowMin(t *testing.T) {
	// Arrange
	svc := NewPasswordService(10)

	// Act - request length 5, but minLength is 10
	password, err := svc.GenerateRandomPassword(5)

	// Assert
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(password), 10)
}

func TestGenerateRandomPassword_ShouldUseRequestedLength_WhenAboveMinLength(t *testing.T) {
	// Arrange
	svc := NewPasswordService(8)

	// Act
	password, err := svc.GenerateRandomPassword(20)

	// Assert
	require.NoError(t, err)
	assert.Len(t, password, 20)
}

func TestGenerateRandomPassword_ShouldContainUppercase_WhenRequired(t *testing.T) {
	// Arrange
	svc := NewPasswordService(8)

	// Act
	password, err := svc.GenerateRandomPassword(16)

	// Assert
	require.NoError(t, err)
	hasUpper := false
	for _, ch := range password {
		if unicode.IsUpper(ch) {
			hasUpper = true
			break
		}
	}
	assert.True(t, hasUpper, "generated password should contain at least one uppercase letter, got: %s", password)
}

func TestGenerateRandomPassword_ShouldContainLowercase_WhenRequired(t *testing.T) {
	// Arrange
	svc := NewPasswordService(8)

	// Act
	password, err := svc.GenerateRandomPassword(16)

	// Assert
	require.NoError(t, err)
	hasLower := false
	for _, ch := range password {
		if unicode.IsLower(ch) {
			hasLower = true
			break
		}
	}
	assert.True(t, hasLower, "generated password should contain at least one lowercase letter, got: %s", password)
}

func TestGenerateRandomPassword_ShouldContainDigit_WhenRequired(t *testing.T) {
	// Arrange
	svc := NewPasswordService(8)

	// Act
	password, err := svc.GenerateRandomPassword(16)

	// Assert
	require.NoError(t, err)
	hasDigit := false
	for _, ch := range password {
		if unicode.IsDigit(ch) {
			hasDigit = true
			break
		}
	}
	assert.True(t, hasDigit, "generated password should contain at least one digit, got: %s", password)
}

func TestGenerateRandomPassword_ShouldContainSpecial_WhenRequired(t *testing.T) {
	// Arrange
	svc := NewPasswordService(8)
	svc.requireSpecial = true

	// Act
	password, err := svc.GenerateRandomPassword(16)

	// Assert
	require.NoError(t, err)
	hasSpecial := false
	for _, ch := range password {
		if unicode.IsPunct(ch) || unicode.IsSymbol(ch) {
			hasSpecial = true
			break
		}
	}
	assert.True(t, hasSpecial, "generated password should contain at least one special character, got: %s", password)
}

func TestGenerateRandomPassword_ShouldPassValidation(t *testing.T) {
	// Arrange
	svc := NewPasswordService(8)

	// Act - generate multiple passwords and validate each
	for i := 0; i < 20; i++ {
		password, err := svc.GenerateRandomPassword(12)
		require.NoError(t, err)

		// Assert - every generated password should pass its own validation
		validationErr := svc.ValidatePassword(password)
		assert.NoError(t, validationErr, "generated password %q should pass validation", password)
	}
}

func TestGenerateRandomPassword_ShouldPassValidation_WhenSpecialRequired(t *testing.T) {
	// Arrange
	svc := NewPasswordService(8)
	svc.requireSpecial = true

	// Act & Assert
	for i := 0; i < 20; i++ {
		password, err := svc.GenerateRandomPassword(12)
		require.NoError(t, err)

		validationErr := svc.ValidatePassword(password)
		assert.NoError(t, validationErr, "generated password %q should pass validation with special char requirement", password)
	}
}

func TestGenerateRandomPassword_ShouldProduceUniquePasswords(t *testing.T) {
	// Arrange
	svc := NewPasswordService(8)
	passwordCount := 30
	passwords := make(map[string]bool, passwordCount)

	// Act
	for i := 0; i < passwordCount; i++ {
		password, err := svc.GenerateRandomPassword(16)
		require.NoError(t, err)
		passwords[password] = true
	}

	// Assert
	assert.Len(t, passwords, passwordCount, "all generated passwords should be unique")
}

func TestGenerateRandomPassword_ShouldUseMinLength_WhenRequestedLengthIsZero(t *testing.T) {
	// Arrange
	svc := NewPasswordService(8)

	// Act
	password, err := svc.GenerateRandomPassword(0)

	// Assert
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(password), 8)
}

func TestGenerateRandomPassword_ShouldHandleLargeLength(t *testing.T) {
	// Arrange
	svc := NewPasswordService(8)

	// Act
	password, err := svc.GenerateRandomPassword(100)

	// Assert
	require.NoError(t, err)
	assert.Len(t, password, 100)
}

func TestGenerateRandomPassword_ShouldUseExactMinLength_WhenRequestedEqualsMin(t *testing.T) {
	// Arrange
	svc := NewPasswordService(12)

	// Act
	password, err := svc.GenerateRandomPassword(12)

	// Assert
	require.NoError(t, err)
	assert.Len(t, password, 12)
}

// --- PasswordError ---

func TestPasswordError_ShouldReturnMessage(t *testing.T) {
	// Arrange
	pe := &PasswordError{Message: "some validation error"}

	// Act
	msg := pe.Error()

	// Assert
	assert.Equal(t, "some validation error", msg)
}

// --- HashPassword + VerifyPassword round-trip ---

func TestHashAndVerify_ShouldSucceed_ForVariousPasswords(t *testing.T) {
	svc := NewPasswordService(6)

	passwords := []string{
		"Simple1A",
		"C0mpl3xP@ssw0rd!",
		"Ab1" + strings.Repeat("x", 50),
		"   Spaces1  ",
		"\tTab1And\nNewline",
		"Unicode\u00e91Aa",
	}

	for _, pw := range passwords {
		t.Run(pw[:min(len(pw), 20)], func(t *testing.T) {
			// Arrange & Act
			hash, err := svc.HashPassword(pw)
			require.NoError(t, err)

			// Assert - correct password verifies
			assert.NoError(t, svc.VerifyPassword(pw, hash))

			// Assert - wrong password does not verify
			assert.ErrorIs(t, svc.VerifyPassword(pw+"x", hash), ErrPasswordMismatch)
		})
	}
}

// --- ValidatePassword table-driven tests ---

func TestValidatePassword_TableDriven(t *testing.T) {
	svc := NewPasswordService(8)

	tests := []struct {
		name        string
		password    string
		wantErr     error
		errContains string
	}{
		{
			name:     "valid password with all requirements",
			password: "GoodPass1",
			wantErr:  nil,
		},
		{
			name:        "too short",
			password:    "Ab1",
			wantErr:     ErrPasswordTooShort,
			errContains: "minimum 8 characters",
		},
		{
			name:        "missing uppercase",
			password:    "nouppercase1",
			wantErr:     ErrPasswordTooWeak,
			errContains: "uppercase letter",
		},
		{
			name:        "missing lowercase",
			password:    "NOLOWERCASE1",
			wantErr:     ErrPasswordTooWeak,
			errContains: "lowercase letter",
		},
		{
			name:        "missing digit",
			password:    "NoDigitHere",
			wantErr:     ErrPasswordTooWeak,
			errContains: "digit",
		},
		{
			name:        "only digits",
			password:    "12345678",
			wantErr:     ErrPasswordTooWeak,
			errContains: "uppercase letter",
		},
		{
			name:        "empty string",
			password:    "",
			wantErr:     ErrPasswordTooShort,
			errContains: "minimum 8 characters",
		},
		{
			name:     "all three requirements met at boundary length",
			password: "Abcdef1X",
			wantErr:  nil,
		},
		{
			name:     "long valid password",
			password: "ThisIsAVeryLongPassword1WithMixedCase",
			wantErr:  nil,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Act
			err := svc.ValidatePassword(tc.password)

			// Assert
			if tc.wantErr == nil {
				assert.NoError(t, err)
			} else {
				require.Error(t, err)
				assert.ErrorIs(t, err, tc.wantErr)
				if tc.errContains != "" {
					assert.Contains(t, err.Error(), tc.errContains)
				}
			}
		})
	}
}

// --- Edge cases ---

func TestVerifyPassword_ShouldReturnInvalidHash_WhenHashIsTruncated(t *testing.T) {
	// Arrange
	svc := NewPasswordService(8)
	hash, err := svc.HashPassword("ValidPass1")
	require.NoError(t, err)

	truncatedHash := hash[:10]

	// Act
	verifyErr := svc.VerifyPassword("ValidPass1", truncatedHash)

	// Assert
	require.Error(t, verifyErr)
	assert.ErrorIs(t, verifyErr, ErrInvalidPasswordHash)
}

func TestValidatePassword_ShouldCheckLengthBeforeComplexity(t *testing.T) {
	// Arrange - a short password that also lacks uppercase and digit
	svc := NewPasswordService(20)

	// Act
	err := svc.ValidatePassword("short")

	// Assert - should get ErrPasswordTooShort, not ErrPasswordTooWeak
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrPasswordTooShort)
}

func TestHashPassword_ShouldHandleSpecialCharacters(t *testing.T) {
	// Arrange
	svc := NewPasswordService(8)
	password := `P@ss!#$%^&*()_+-=[]{}|;':\",./<>?1a`

	// Act
	hash, err := svc.HashPassword(password)
	require.NoError(t, err)

	// Assert
	verifyErr := svc.VerifyPassword(password, hash)
	assert.NoError(t, verifyErr)
}

func TestGenerateResetToken_ShouldHaveConsistentLength(t *testing.T) {
	// Arrange
	svc := NewPasswordService(8)

	// Act & Assert - generate multiple tokens and verify consistent length
	for i := 0; i < 10; i++ {
		token, err := svc.GenerateResetToken()
		require.NoError(t, err)
		assert.Len(t, token, 64)
	}
}
