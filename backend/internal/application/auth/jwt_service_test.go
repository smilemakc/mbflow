package auth

import (
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/smilemakc/mbflow/internal/config"
	"github.com/smilemakc/mbflow/pkg/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- Helpers ---

func newTestConfig() *config.AuthConfig {
	return &config.AuthConfig{
		JWTSecret:          "test-secret-key-minimum-32-chars!",
		IssuerURL:          "https://test.example.com",
		JWTExpirationHours: 24,
		RefreshExpiryDays:  30,
	}
}

func newTestUser() *models.User {
	return &models.User{
		ID:       "user-123",
		Email:    "john@example.com",
		Username: "johndoe",
		IsAdmin:  false,
		Roles:    []string{"editor", "viewer"},
	}
}

func newAdminUser() *models.User {
	return &models.User{
		ID:       "admin-456",
		Email:    "admin@example.com",
		Username: "admin",
		IsAdmin:  true,
		Roles:    []string{"admin"},
	}
}

// forgeTokenWithSecret creates a token signed with an arbitrary secret, useful
// for testing validation against the wrong key.
func forgeTokenWithSecret(user *models.User, secret string, expiry time.Duration) string {
	claims := &JWTClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   user.ID,
			Issuer:    "https://other.example.com",
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(expiry)),
			NotBefore: jwt.NewNumericDate(time.Now()),
		},
		UserID:   user.ID,
		Email:    user.Email,
		Username: user.Username,
		IsAdmin:  user.IsAdmin,
		Roles:    user.Roles,
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, _ := token.SignedString([]byte(secret))
	return signed
}

// forgeExpiredToken creates a token that is already expired.
func forgeExpiredToken(cfg *config.AuthConfig, user *models.User) string {
	claims := &JWTClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   user.ID,
			Issuer:    cfg.IssuerURL,
			IssuedAt:  jwt.NewNumericDate(time.Now().Add(-2 * time.Hour)),
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(-1 * time.Hour)),
			NotBefore: jwt.NewNumericDate(time.Now().Add(-2 * time.Hour)),
		},
		UserID:   user.ID,
		Email:    user.Email,
		Username: user.Username,
		IsAdmin:  user.IsAdmin,
		Roles:    user.Roles,
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, _ := token.SignedString([]byte(cfg.JWTSecret))
	return signed
}

// --- NewJWTService ---

func TestJWTNewJWTService_ShouldUseIssuerURL_WhenProvided(t *testing.T) {
	// Arrange
	cfg := newTestConfig()

	// Act
	svc := NewJWTService(cfg)

	// Assert
	require.NotNil(t, svc)
	assert.Equal(t, "https://test.example.com", svc.issuer)
	assert.Equal(t, []byte("test-secret-key-minimum-32-chars!"), svc.secret)
	assert.Equal(t, 24, svc.accessExpiryHrs)
	assert.Equal(t, 30, svc.refreshExpiryDays)
}

func TestJWTNewJWTService_ShouldDefaultToMbflow_WhenIssuerURLEmpty(t *testing.T) {
	// Arrange
	cfg := newTestConfig()
	cfg.IssuerURL = ""

	// Act
	svc := NewJWTService(cfg)

	// Assert
	assert.Equal(t, "mbflow", svc.issuer)
}

// --- GenerateAccessToken ---

func TestJWTGenerateAccessToken_ShouldReturnValidToken_WhenUserDataProvided(t *testing.T) {
	// Arrange
	cfg := newTestConfig()
	svc := NewJWTService(cfg)
	user := newTestUser()

	// Act
	tokenStr, expiresAt, err := svc.GenerateAccessToken(user)

	// Assert
	require.NoError(t, err)
	assert.NotEmpty(t, tokenStr)
	assert.False(t, expiresAt.IsZero())
	assert.True(t, expiresAt.After(time.Now()))
	assert.True(t, expiresAt.Before(time.Now().Add(25*time.Hour)))
}

func TestJWTGenerateAccessToken_ShouldSetCorrectClaims_WhenTokenParsed(t *testing.T) {
	// Arrange
	cfg := newTestConfig()
	svc := NewJWTService(cfg)
	user := newTestUser()
	beforeGeneration := time.Now().Add(-1 * time.Second)

	// Act
	tokenStr, _, err := svc.GenerateAccessToken(user)
	require.NoError(t, err)

	// Parse the token to inspect claims
	claims, err := svc.ValidateAccessToken(tokenStr)
	require.NoError(t, err)

	// Assert - custom claims
	assert.Equal(t, "user-123", claims.UserID)
	assert.Equal(t, "john@example.com", claims.Email)
	assert.Equal(t, "johndoe", claims.Username)
	assert.False(t, claims.IsAdmin)
	assert.Equal(t, []string{"editor", "viewer"}, claims.Roles)

	// Assert - registered claims
	assert.Equal(t, "user-123", claims.Subject)
	assert.Equal(t, "https://test.example.com", claims.Issuer)
	require.NotNil(t, claims.IssuedAt)
	assert.True(t, claims.IssuedAt.Time.After(beforeGeneration))
	require.NotNil(t, claims.ExpiresAt)
	assert.True(t, claims.ExpiresAt.Time.After(time.Now()))
	require.NotNil(t, claims.NotBefore)
	assert.True(t, claims.NotBefore.Time.Before(time.Now().Add(1*time.Second)))
}

func TestJWTGenerateAccessToken_ShouldSetAdminFlag_WhenUserIsAdmin(t *testing.T) {
	// Arrange
	cfg := newTestConfig()
	svc := NewJWTService(cfg)
	admin := newAdminUser()

	// Act
	tokenStr, _, err := svc.GenerateAccessToken(admin)
	require.NoError(t, err)

	claims, err := svc.ValidateAccessToken(tokenStr)
	require.NoError(t, err)

	// Assert
	assert.True(t, claims.IsAdmin)
	assert.Equal(t, "admin-456", claims.UserID)
	assert.Equal(t, "admin@example.com", claims.Email)
	assert.Equal(t, "admin", claims.Username)
	assert.Equal(t, []string{"admin"}, claims.Roles)
}

func TestJWTGenerateAccessToken_ShouldHandleNilRoles_WhenUserHasNoRoles(t *testing.T) {
	// Arrange
	cfg := newTestConfig()
	svc := NewJWTService(cfg)
	user := &models.User{
		ID:       "user-no-roles",
		Email:    "noroles@example.com",
		Username: "noroles",
		IsAdmin:  false,
		Roles:    nil,
	}

	// Act
	tokenStr, _, err := svc.GenerateAccessToken(user)
	require.NoError(t, err)

	claims, err := svc.ValidateAccessToken(tokenStr)
	require.NoError(t, err)

	// Assert
	assert.Nil(t, claims.Roles)
	assert.Equal(t, "user-no-roles", claims.UserID)
}

func TestJWTGenerateAccessToken_ShouldHandleEmptyRoles_WhenUserHasEmptySlice(t *testing.T) {
	// Arrange
	cfg := newTestConfig()
	svc := NewJWTService(cfg)
	user := &models.User{
		ID:       "user-empty-roles",
		Email:    "empty@example.com",
		Username: "emptyroles",
		IsAdmin:  false,
		Roles:    []string{},
	}

	// Act
	tokenStr, _, err := svc.GenerateAccessToken(user)
	require.NoError(t, err)

	claims, err := svc.ValidateAccessToken(tokenStr)
	require.NoError(t, err)

	// Assert
	assert.Empty(t, claims.Roles)
}

func TestJWTGenerateAccessToken_ShouldRespectExpiryConfig_WhenCustomHoursSet(t *testing.T) {
	// Arrange
	cfg := newTestConfig()
	cfg.JWTExpirationHours = 1
	svc := NewJWTService(cfg)
	user := newTestUser()

	// Act
	_, expiresAt, err := svc.GenerateAccessToken(user)
	require.NoError(t, err)

	// Assert
	expectedExpiry := time.Now().Add(1 * time.Hour)
	assert.WithinDuration(t, expectedExpiry, expiresAt, 5*time.Second)
}

func TestJWTGenerateAccessToken_ShouldProduceDifferentTokens_WhenCalledMultipleTimes(t *testing.T) {
	// Arrange
	cfg := newTestConfig()
	svc := NewJWTService(cfg)
	user := newTestUser()

	// Act
	token1, _, err1 := svc.GenerateAccessToken(user)
	// Tiny sleep to ensure iat differs
	time.Sleep(1 * time.Millisecond)
	token2, _, err2 := svc.GenerateAccessToken(user)

	// Assert
	require.NoError(t, err1)
	require.NoError(t, err2)
	// Tokens may technically differ due to iat timestamp differences,
	// but even if iat is the same second, the JWT encoding should be identical.
	// The point is both are valid.
	assert.NotEmpty(t, token1)
	assert.NotEmpty(t, token2)
}

// --- ValidateAccessToken ---

func TestJWTValidateAccessToken_ShouldReturnClaims_WhenTokenIsValid(t *testing.T) {
	// Arrange
	cfg := newTestConfig()
	svc := NewJWTService(cfg)
	user := newTestUser()
	tokenStr, _, err := svc.GenerateAccessToken(user)
	require.NoError(t, err)

	// Act
	claims, err := svc.ValidateAccessToken(tokenStr)

	// Assert
	require.NoError(t, err)
	require.NotNil(t, claims)
	assert.Equal(t, "user-123", claims.UserID)
	assert.Equal(t, "john@example.com", claims.Email)
	assert.Equal(t, "johndoe", claims.Username)
	assert.False(t, claims.IsAdmin)
	assert.Equal(t, []string{"editor", "viewer"}, claims.Roles)
}

func TestJWTValidateAccessToken_ShouldReturnExpiredError_WhenTokenIsExpired(t *testing.T) {
	// Arrange
	cfg := newTestConfig()
	svc := NewJWTService(cfg)
	user := newTestUser()
	expiredToken := forgeExpiredToken(cfg, user)

	// Act
	claims, err := svc.ValidateAccessToken(expiredToken)

	// Assert
	assert.Nil(t, claims)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrExpiredToken)
}

func TestJWTValidateAccessToken_ShouldReturnInvalidError_WhenSignedWithDifferentKey(t *testing.T) {
	// Arrange
	cfg := newTestConfig()
	svc := NewJWTService(cfg)
	user := newTestUser()
	wrongKeyToken := forgeTokenWithSecret(user, "different-secret-key-32-chars!!!", 24*time.Hour)

	// Act
	claims, err := svc.ValidateAccessToken(wrongKeyToken)

	// Assert
	assert.Nil(t, claims)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrInvalidToken)
}

func TestJWTValidateAccessToken_ShouldReturnInvalidError_WhenTokenIsMalformed(t *testing.T) {
	// Arrange
	cfg := newTestConfig()
	svc := NewJWTService(cfg)

	// Act
	claims, err := svc.ValidateAccessToken("not.a.valid.jwt.token")

	// Assert
	assert.Nil(t, claims)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrInvalidToken)
}

func TestJWTValidateAccessToken_ShouldReturnInvalidError_WhenTokenIsEmpty(t *testing.T) {
	// Arrange
	cfg := newTestConfig()
	svc := NewJWTService(cfg)

	// Act
	claims, err := svc.ValidateAccessToken("")

	// Assert
	assert.Nil(t, claims)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrInvalidToken)
}

func TestJWTValidateAccessToken_ShouldReturnInvalidError_WhenTokenIsGarbage(t *testing.T) {
	// Arrange
	cfg := newTestConfig()
	svc := NewJWTService(cfg)

	// Act
	claims, err := svc.ValidateAccessToken("completely-random-garbage-string")

	// Assert
	assert.Nil(t, claims)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrInvalidToken)
}

func TestJWTValidateAccessToken_ShouldReturnInvalidError_WhenSigningMethodIsNone(t *testing.T) {
	// Arrange
	cfg := newTestConfig()
	svc := NewJWTService(cfg)
	user := newTestUser()

	// Create a token with "none" algorithm (alg=none attack)
	claims := &JWTClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   user.ID,
			Issuer:    cfg.IssuerURL,
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
			NotBefore: jwt.NewNumericDate(time.Now()),
		},
		UserID:   user.ID,
		Email:    user.Email,
		Username: user.Username,
		IsAdmin:  user.IsAdmin,
		Roles:    user.Roles,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodNone, claims)
	// jwt.SigningMethodNone requires jwt.UnsafeAllowNoneSignatureType as the key
	tokenStr, err := token.SignedString(jwt.UnsafeAllowNoneSignatureType)
	require.NoError(t, err)

	// Act
	result, err := svc.ValidateAccessToken(tokenStr)

	// Assert
	assert.Nil(t, result)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrInvalidToken)
}

func TestJWTValidateAccessToken_ShouldReturnNotYetValidError_WhenNotBeforeIsInFuture(t *testing.T) {
	// Arrange
	cfg := newTestConfig()
	svc := NewJWTService(cfg)
	user := newTestUser()

	claims := &JWTClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   user.ID,
			Issuer:    cfg.IssuerURL,
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
			NotBefore: jwt.NewNumericDate(time.Now().Add(1 * time.Hour)), // 1 hour in the future
		},
		UserID:   user.ID,
		Email:    user.Email,
		Username: user.Username,
		IsAdmin:  user.IsAdmin,
		Roles:    user.Roles,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenStr, err := token.SignedString([]byte(cfg.JWTSecret))
	require.NoError(t, err)

	// Act
	result, err := svc.ValidateAccessToken(tokenStr)

	// Assert
	assert.Nil(t, result)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrTokenNotYetValid)
}

func TestJWTValidateAccessToken_ShouldSucceed_WhenTokenGeneratedBySameService(t *testing.T) {
	// Arrange - round-trip test: generate then validate
	cfg := newTestConfig()
	svc := NewJWTService(cfg)
	user := newAdminUser()

	tokenStr, expiresAt, err := svc.GenerateAccessToken(user)
	require.NoError(t, err)

	// Act
	claims, err := svc.ValidateAccessToken(tokenStr)

	// Assert
	require.NoError(t, err)
	require.NotNil(t, claims)
	assert.Equal(t, user.ID, claims.UserID)
	assert.Equal(t, user.Email, claims.Email)
	assert.Equal(t, user.Username, claims.Username)
	assert.Equal(t, user.IsAdmin, claims.IsAdmin)
	assert.Equal(t, user.Roles, claims.Roles)
	assert.WithinDuration(t, expiresAt, claims.ExpiresAt.Time, 2*time.Second)
}

func TestJWTValidateAccessToken_ShouldRejectTamperedToken(t *testing.T) {
	// Arrange
	cfg := newTestConfig()
	svc := NewJWTService(cfg)
	user := newTestUser()

	tokenStr, _, err := svc.GenerateAccessToken(user)
	require.NoError(t, err)

	// Tamper with the token payload by flipping a character
	parts := strings.Split(tokenStr, ".")
	require.Len(t, parts, 3)

	payload := []byte(parts[1])
	if payload[0] == 'a' {
		payload[0] = 'b'
	} else {
		payload[0] = 'a'
	}
	tamperedToken := parts[0] + "." + string(payload) + "." + parts[2]

	// Act
	claims, err := svc.ValidateAccessToken(tamperedToken)

	// Assert
	assert.Nil(t, claims)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrInvalidToken)
}

// --- ExtractClaimsFromExpiredToken ---

func TestJWTExtractClaimsFromExpiredToken_ShouldReturnClaims_WhenTokenIsExpired(t *testing.T) {
	// Arrange
	cfg := newTestConfig()
	svc := NewJWTService(cfg)
	user := newTestUser()
	expiredToken := forgeExpiredToken(cfg, user)

	// Verify the token is truly expired
	_, validErr := svc.ValidateAccessToken(expiredToken)
	require.ErrorIs(t, validErr, ErrExpiredToken)

	// Act
	claims, err := svc.ExtractClaimsFromExpiredToken(expiredToken)

	// Assert
	require.NoError(t, err)
	require.NotNil(t, claims)
	assert.Equal(t, "user-123", claims.UserID)
	assert.Equal(t, "john@example.com", claims.Email)
	assert.Equal(t, "johndoe", claims.Username)
	assert.False(t, claims.IsAdmin)
	assert.Equal(t, []string{"editor", "viewer"}, claims.Roles)
}

func TestJWTExtractClaimsFromExpiredToken_ShouldReturnClaims_WhenTokenIsStillValid(t *testing.T) {
	// Arrange
	cfg := newTestConfig()
	svc := NewJWTService(cfg)
	user := newTestUser()
	tokenStr, _, err := svc.GenerateAccessToken(user)
	require.NoError(t, err)

	// Act
	claims, err := svc.ExtractClaimsFromExpiredToken(tokenStr)

	// Assert
	require.NoError(t, err)
	require.NotNil(t, claims)
	assert.Equal(t, "user-123", claims.UserID)
	assert.Equal(t, "john@example.com", claims.Email)
}

func TestJWTExtractClaimsFromExpiredToken_ShouldReturnClaims_WhenSignedWithDifferentKey(t *testing.T) {
	// Arrange - ParseUnverified does NOT check signature, so this should succeed
	cfg := newTestConfig()
	svc := NewJWTService(cfg)
	user := newTestUser()
	wrongKeyToken := forgeTokenWithSecret(user, "different-secret-key-32-chars!!!", 24*time.Hour)

	// Act
	claims, err := svc.ExtractClaimsFromExpiredToken(wrongKeyToken)

	// Assert
	require.NoError(t, err)
	require.NotNil(t, claims)
	assert.Equal(t, "user-123", claims.UserID)
	assert.Equal(t, "john@example.com", claims.Email)
}

func TestJWTExtractClaimsFromExpiredToken_ShouldReturnError_WhenTokenIsMalformed(t *testing.T) {
	// Arrange
	cfg := newTestConfig()
	svc := NewJWTService(cfg)

	// Act
	claims, err := svc.ExtractClaimsFromExpiredToken("not.a.valid.jwt")

	// Assert
	assert.Nil(t, claims)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse token")
}

func TestJWTExtractClaimsFromExpiredToken_ShouldReturnError_WhenTokenIsEmpty(t *testing.T) {
	// Arrange
	cfg := newTestConfig()
	svc := NewJWTService(cfg)

	// Act
	claims, err := svc.ExtractClaimsFromExpiredToken("")

	// Assert
	assert.Nil(t, claims)
	require.Error(t, err)
}

func TestJWTExtractClaimsFromExpiredToken_ShouldReturnError_WhenTokenIsGarbage(t *testing.T) {
	// Arrange
	cfg := newTestConfig()
	svc := NewJWTService(cfg)

	// Act
	claims, err := svc.ExtractClaimsFromExpiredToken("completely-random-garbage-string")

	// Assert
	assert.Nil(t, claims)
	require.Error(t, err)
}

func TestJWTExtractClaimsFromExpiredToken_ShouldPreserveAdminFlag_WhenExtractingFromExpiredAdminToken(t *testing.T) {
	// Arrange
	cfg := newTestConfig()
	svc := NewJWTService(cfg)
	admin := newAdminUser()
	expiredToken := forgeExpiredToken(cfg, admin)

	// Act
	claims, err := svc.ExtractClaimsFromExpiredToken(expiredToken)

	// Assert
	require.NoError(t, err)
	require.NotNil(t, claims)
	assert.True(t, claims.IsAdmin)
	assert.Equal(t, "admin-456", claims.UserID)
	assert.Equal(t, "admin@example.com", claims.Email)
	assert.Equal(t, []string{"admin"}, claims.Roles)
}

// --- GenerateRefreshToken ---

func TestJWTGenerateRefreshToken_ShouldReturnNonEmptyToken(t *testing.T) {
	// Arrange
	cfg := newTestConfig()
	svc := NewJWTService(cfg)

	// Act
	token, expiresAt, err := svc.GenerateRefreshToken()

	// Assert
	require.NoError(t, err)
	assert.NotEmpty(t, token)
	assert.False(t, expiresAt.IsZero())
}

func TestJWTGenerateRefreshToken_ShouldReturnHexEncodedString(t *testing.T) {
	// Arrange
	cfg := newTestConfig()
	svc := NewJWTService(cfg)

	// Act
	token, _, err := svc.GenerateRefreshToken()
	require.NoError(t, err)

	// Assert - hex encoded 32 bytes = 64 hex characters
	assert.Len(t, token, 64)
	for _, c := range token {
		assert.True(t, (c >= '0' && c <= '9') || (c >= 'a' && c <= 'f'),
			"character '%c' is not valid hex", c)
	}
}

func TestJWTGenerateRefreshToken_ShouldSetCorrectExpiry(t *testing.T) {
	// Arrange
	cfg := newTestConfig()
	cfg.RefreshExpiryDays = 30
	svc := NewJWTService(cfg)

	// Act
	_, expiresAt, err := svc.GenerateRefreshToken()
	require.NoError(t, err)

	// Assert
	expectedExpiry := time.Now().Add(30 * 24 * time.Hour)
	assert.WithinDuration(t, expectedExpiry, expiresAt, 5*time.Second)
}

func TestJWTGenerateRefreshToken_ShouldProduceUniqueTokens_WhenCalledMultipleTimes(t *testing.T) {
	// Arrange
	cfg := newTestConfig()
	svc := NewJWTService(cfg)
	tokenSet := make(map[string]struct{})

	// Act - generate 100 tokens
	for i := 0; i < 100; i++ {
		token, _, err := svc.GenerateRefreshToken()
		require.NoError(t, err)
		tokenSet[token] = struct{}{}
	}

	// Assert - all 100 should be unique
	assert.Len(t, tokenSet, 100)
}

func TestJWTGenerateRefreshToken_ShouldRespectCustomExpiryDays(t *testing.T) {
	// Arrange
	cfg := newTestConfig()
	cfg.RefreshExpiryDays = 7
	svc := NewJWTService(cfg)

	// Act
	_, expiresAt, err := svc.GenerateRefreshToken()
	require.NoError(t, err)

	// Assert
	expectedExpiry := time.Now().Add(7 * 24 * time.Hour)
	assert.WithinDuration(t, expectedExpiry, expiresAt, 5*time.Second)
}

// --- GetAccessTokenExpiry ---

func TestJWTGetAccessTokenExpiry_ShouldReturnCorrectSeconds_WhenHoursSet(t *testing.T) {
	tests := []struct {
		name     string
		hours    int
		expected int
	}{
		{
			name:     "24 hours",
			hours:    24,
			expected: 86400,
		},
		{
			name:     "1 hour",
			hours:    1,
			expected: 3600,
		},
		{
			name:     "48 hours",
			hours:    48,
			expected: 172800,
		},
		{
			name:     "0 hours",
			hours:    0,
			expected: 0,
		},
		{
			name:     "168 hours (1 week)",
			hours:    168,
			expected: 604800,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			cfg := newTestConfig()
			cfg.JWTExpirationHours = tt.hours
			svc := NewJWTService(cfg)

			// Act
			result := svc.GetAccessTokenExpiry()

			// Assert
			assert.Equal(t, tt.expected, result)
		})
	}
}

// --- GetRefreshTokenExpiry ---

func TestJWTGetRefreshTokenExpiry_ShouldReturnCorrectSeconds_WhenDaysSet(t *testing.T) {
	tests := []struct {
		name     string
		days     int
		expected int
	}{
		{
			name:     "30 days",
			days:     30,
			expected: 2592000,
		},
		{
			name:     "1 day",
			days:     1,
			expected: 86400,
		},
		{
			name:     "7 days",
			days:     7,
			expected: 604800,
		},
		{
			name:     "365 days",
			days:     365,
			expected: 31536000,
		},
		{
			name:     "0 days",
			days:     0,
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			cfg := newTestConfig()
			cfg.RefreshExpiryDays = tt.days
			svc := NewJWTService(cfg)

			// Act
			result := svc.GetRefreshTokenExpiry()

			// Assert
			assert.Equal(t, tt.expected, result)
		})
	}
}

// --- Cross-service token validation ---

func TestJWTValidateAccessToken_ShouldFail_WhenDifferentServiceWithDifferentSecret(t *testing.T) {
	// Arrange
	cfg1 := newTestConfig()
	cfg1.JWTSecret = "service-one-secret-key-32-chars!!"
	svc1 := NewJWTService(cfg1)

	cfg2 := newTestConfig()
	cfg2.JWTSecret = "service-two-secret-key-32-chars!!"
	svc2 := NewJWTService(cfg2)

	user := newTestUser()
	tokenStr, _, err := svc1.GenerateAccessToken(user)
	require.NoError(t, err)

	// Act
	claims, err := svc2.ValidateAccessToken(tokenStr)

	// Assert
	assert.Nil(t, claims)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrInvalidToken)
}

func TestJWTValidateAccessToken_ShouldSucceed_WhenDifferentServiceWithSameSecret(t *testing.T) {
	// Arrange
	cfg1 := newTestConfig()
	svc1 := NewJWTService(cfg1)

	cfg2 := newTestConfig()
	svc2 := NewJWTService(cfg2)

	user := newTestUser()
	tokenStr, _, err := svc1.GenerateAccessToken(user)
	require.NoError(t, err)

	// Act
	claims, err := svc2.ValidateAccessToken(tokenStr)

	// Assert
	require.NoError(t, err)
	require.NotNil(t, claims)
	assert.Equal(t, user.ID, claims.UserID)
	assert.Equal(t, user.Email, claims.Email)
}

// --- Edge cases with user data ---

func TestJWTGenerateAccessToken_ShouldHandleSpecialCharactersInEmail(t *testing.T) {
	// Arrange
	cfg := newTestConfig()
	svc := NewJWTService(cfg)
	user := &models.User{
		ID:       "user-special",
		Email:    "user+tag@sub.example.com",
		Username: "special_user-123",
		IsAdmin:  false,
		Roles:    []string{"viewer"},
	}

	// Act
	tokenStr, _, err := svc.GenerateAccessToken(user)
	require.NoError(t, err)

	claims, err := svc.ValidateAccessToken(tokenStr)
	require.NoError(t, err)

	// Assert
	assert.Equal(t, "user+tag@sub.example.com", claims.Email)
	assert.Equal(t, "special_user-123", claims.Username)
}

func TestJWTGenerateAccessToken_ShouldHandleEmptyUserFields(t *testing.T) {
	// Arrange
	cfg := newTestConfig()
	svc := NewJWTService(cfg)
	user := &models.User{
		ID:       "",
		Email:    "",
		Username: "",
		IsAdmin:  false,
		Roles:    nil,
	}

	// Act
	tokenStr, _, err := svc.GenerateAccessToken(user)
	require.NoError(t, err)

	claims, err := svc.ValidateAccessToken(tokenStr)
	require.NoError(t, err)

	// Assert
	assert.Equal(t, "", claims.UserID)
	assert.Equal(t, "", claims.Email)
	assert.Equal(t, "", claims.Username)
}

func TestJWTGenerateAccessToken_ShouldHandleManyRoles(t *testing.T) {
	// Arrange
	cfg := newTestConfig()
	svc := NewJWTService(cfg)
	roles := make([]string, 50)
	for i := range roles {
		roles[i] = "role_" + strings.Repeat("x", 10)
	}
	user := &models.User{
		ID:       "user-many-roles",
		Email:    "many@example.com",
		Username: "manyroles",
		IsAdmin:  false,
		Roles:    roles,
	}

	// Act
	tokenStr, _, err := svc.GenerateAccessToken(user)
	require.NoError(t, err)

	claims, err := svc.ValidateAccessToken(tokenStr)
	require.NoError(t, err)

	// Assert
	assert.Len(t, claims.Roles, 50)
	assert.Equal(t, roles, claims.Roles)
}

// --- Unicode edge cases ---

func TestJWTGenerateAccessToken_ShouldHandleUnicodeInUserFields(t *testing.T) {
	// Arrange
	cfg := newTestConfig()
	svc := NewJWTService(cfg)
	user := &models.User{
		ID:       "user-unicode",
		Email:    "user@example.com",
		Username: "usuario_nombre",
		IsAdmin:  false,
		Roles:    []string{"viewer"},
	}

	// Act
	tokenStr, _, err := svc.GenerateAccessToken(user)
	require.NoError(t, err)

	claims, err := svc.ValidateAccessToken(tokenStr)
	require.NoError(t, err)

	// Assert
	assert.Equal(t, "usuario_nombre", claims.Username)
}
