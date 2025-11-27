package websocket

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testSecret = "test-secret-key-for-jwt"

// Helper function to generate a valid test token
func generateTestToken(t *testing.T, userID string, expiresAt time.Time) string {
	auth := NewJWTAuth(testSecret)
	token, err := auth.GenerateToken(userID, jwt.NewNumericDate(expiresAt))
	require.NoError(t, err)
	return token
}

func TestNewJWTAuth(t *testing.T) {
	auth := NewJWTAuth("my-secret-key")

	assert.NotNil(t, auth)
	assert.Equal(t, "my-secret-key", auth.secretKey)
}

func TestJWTAuth_GenerateToken(t *testing.T) {
	auth := NewJWTAuth(testSecret)

	token, err := auth.GenerateToken("user-123", jwt.NewNumericDate(time.Now().Add(time.Hour)))

	assert.NoError(t, err)
	assert.NotEmpty(t, token)
}

func TestJWTAuth_ValidateToken_ValidToken(t *testing.T) {
	auth := NewJWTAuth(testSecret)

	// Generate a valid token
	token, err := auth.GenerateToken("user-123", jwt.NewNumericDate(time.Now().Add(time.Hour)))
	require.NoError(t, err)

	// Validate it
	userID, err := auth.validateToken(token)

	assert.NoError(t, err)
	assert.Equal(t, "user-123", userID)
}

func TestJWTAuth_ValidateToken_ExpiredToken(t *testing.T) {
	auth := NewJWTAuth(testSecret)

	// Generate an expired token
	token, err := auth.GenerateToken("user-123", jwt.NewNumericDate(time.Now().Add(-time.Hour)))
	require.NoError(t, err)

	// Validate it
	userID, err := auth.validateToken(token)

	assert.Error(t, err)
	assert.Equal(t, ErrExpiredToken, err)
	assert.Empty(t, userID)
}

func TestJWTAuth_ValidateToken_InvalidSignature(t *testing.T) {
	auth1 := NewJWTAuth("secret-1")
	auth2 := NewJWTAuth("secret-2")

	// Generate token with one secret
	token, err := auth1.GenerateToken("user-123", jwt.NewNumericDate(time.Now().Add(time.Hour)))
	require.NoError(t, err)

	// Validate with different secret
	userID, err := auth2.validateToken(token)

	assert.Error(t, err)
	assert.Equal(t, ErrInvalidToken, err)
	assert.Empty(t, userID)
}

func TestJWTAuth_ValidateToken_EmptyString(t *testing.T) {
	auth := NewJWTAuth(testSecret)

	userID, err := auth.validateToken("")

	assert.Error(t, err)
	assert.Equal(t, ErrInvalidToken, err)
	assert.Empty(t, userID)
}

func TestJWTAuth_ValidateToken_MalformedToken(t *testing.T) {
	auth := NewJWTAuth(testSecret)

	tests := []struct {
		name  string
		token string
	}{
		{"random string", "not-a-jwt-token"},
		{"partial jwt", "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9"},
		{"invalid base64", "invalid.base64.token"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			userID, err := auth.validateToken(tt.token)

			assert.Error(t, err)
			assert.Equal(t, ErrInvalidToken, err)
			assert.Empty(t, userID)
		})
	}
}

func TestJWTAuth_ValidateToken_WrongSigningMethod(t *testing.T) {
	auth := NewJWTAuth(testSecret)

	// Create a token with a different signing method (none)
	claims := JWTClaims{
		UserID: "user-123",
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   "user-123",
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodNone, claims)
	tokenString, err := token.SignedString(jwt.UnsafeAllowNoneSignatureType)
	require.NoError(t, err)

	userID, err := auth.validateToken(tokenString)

	assert.Error(t, err)
	assert.Equal(t, ErrInvalidToken, err)
	assert.Empty(t, userID)
}

func TestJWTAuth_ValidateToken_NoUserID(t *testing.T) {
	auth := NewJWTAuth(testSecret)

	// Create a token without user_id and subject
	claims := jwt.RegisteredClaims{
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(testSecret))
	require.NoError(t, err)

	userID, err := auth.validateToken(tokenString)

	assert.Error(t, err)
	assert.Equal(t, ErrInvalidToken, err)
	assert.Empty(t, userID)
}

func TestJWTAuth_ValidateToken_SubjectFallback(t *testing.T) {
	auth := NewJWTAuth(testSecret)

	// Create a token with only subject (no user_id claim)
	claims := jwt.RegisteredClaims{
		Subject:   "user-from-subject",
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(testSecret))
	require.NoError(t, err)

	userID, err := auth.validateToken(tokenString)

	assert.NoError(t, err)
	assert.Equal(t, "user-from-subject", userID)
}

func TestJWTAuth_AuthenticateFromAuthorizationHeader(t *testing.T) {
	auth := NewJWTAuth(testSecret)
	token := generateTestToken(t, "header-user", time.Now().Add(time.Hour))

	req := httptest.NewRequest(http.MethodGet, "/ws", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	userID, err := auth.Authenticate(req)

	assert.NoError(t, err)
	assert.Equal(t, "header-user", userID)
}

func TestJWTAuth_AuthenticateFromQueryParam(t *testing.T) {
	auth := NewJWTAuth(testSecret)
	token := generateTestToken(t, "query-user", time.Now().Add(time.Hour))

	req := httptest.NewRequest(http.MethodGet, "/ws?token="+token, nil)

	userID, err := auth.Authenticate(req)

	assert.NoError(t, err)
	assert.Equal(t, "query-user", userID)
}

func TestJWTAuth_AuthenticateFromWebSocketProtocol(t *testing.T) {
	auth := NewJWTAuth(testSecret)
	token := generateTestToken(t, "protocol-user", time.Now().Add(time.Hour))

	req := httptest.NewRequest(http.MethodGet, "/ws", nil)
	req.Header.Set("Sec-WebSocket-Protocol", "auth-"+token)

	userID, err := auth.Authenticate(req)

	assert.NoError(t, err)
	assert.Equal(t, "protocol-user", userID)
}

func TestJWTAuth_AuthenticateFromWebSocketProtocol_MultipleProtocols(t *testing.T) {
	auth := NewJWTAuth(testSecret)
	token := generateTestToken(t, "multi-protocol-user", time.Now().Add(time.Hour))

	req := httptest.NewRequest(http.MethodGet, "/ws", nil)
	req.Header.Set("Sec-WebSocket-Protocol", "chat, auth-"+token+", binary")

	userID, err := auth.Authenticate(req)

	assert.NoError(t, err)
	assert.Equal(t, "multi-protocol-user", userID)
}

func TestJWTAuth_AuthenticatePriority(t *testing.T) {
	auth := NewJWTAuth(testSecret)
	headerToken := generateTestToken(t, "header-priority", time.Now().Add(time.Hour))
	queryToken := generateTestToken(t, "query-priority", time.Now().Add(time.Hour))
	protocolToken := generateTestToken(t, "protocol-priority", time.Now().Add(time.Hour))

	// Authorization header should take priority
	req := httptest.NewRequest(http.MethodGet, "/ws?token="+queryToken, nil)
	req.Header.Set("Authorization", "Bearer "+headerToken)
	req.Header.Set("Sec-WebSocket-Protocol", "auth-"+protocolToken)

	userID, err := auth.Authenticate(req)

	assert.NoError(t, err)
	assert.Equal(t, "header-priority", userID) // From Authorization header
}

func TestJWTAuth_AuthenticateMissingToken(t *testing.T) {
	auth := NewJWTAuth(testSecret)

	req := httptest.NewRequest(http.MethodGet, "/ws", nil)

	userID, err := auth.Authenticate(req)

	assert.Error(t, err)
	assert.Equal(t, ErrMissingToken, err)
	assert.Empty(t, userID)
}

func TestJWTAuth_AuthenticateInvalidToken_Empty(t *testing.T) {
	auth := NewJWTAuth(testSecret)

	req := httptest.NewRequest(http.MethodGet, "/ws?token=", nil)

	userID, err := auth.Authenticate(req)

	assert.Error(t, err)
	assert.Equal(t, ErrMissingToken, err) // Empty token is treated as missing
	assert.Empty(t, userID)
}

func TestJWTAuth_AuthenticateInvalidToken_Malformed(t *testing.T) {
	auth := NewJWTAuth(testSecret)

	req := httptest.NewRequest(http.MethodGet, "/ws?token=not-a-valid-jwt", nil)

	userID, err := auth.Authenticate(req)

	assert.Error(t, err)
	assert.Equal(t, ErrInvalidToken, err)
	assert.Empty(t, userID)
}

func TestJWTAuth_AuthenticateBearerPrefix(t *testing.T) {
	auth := NewJWTAuth(testSecret)
	queryToken := generateTestToken(t, "fallback-user", time.Now().Add(time.Hour))

	// Without Bearer prefix, Authorization header should be ignored
	req := httptest.NewRequest(http.MethodGet, "/ws?token="+queryToken, nil)
	req.Header.Set("Authorization", "Basic somebasicauth")

	userID, err := auth.Authenticate(req)

	assert.NoError(t, err)
	assert.Equal(t, "fallback-user", userID) // Falls back to query param
}

func TestJWTAuth_AuthenticateFromWebSocketProtocol_NoAuthPrefix(t *testing.T) {
	auth := NewJWTAuth(testSecret)

	req := httptest.NewRequest(http.MethodGet, "/ws", nil)
	req.Header.Set("Sec-WebSocket-Protocol", "chat, binary")

	userID, err := auth.Authenticate(req)

	assert.Error(t, err)
	assert.Equal(t, ErrMissingToken, err)
	assert.Empty(t, userID)
}

func TestJWTAuth_AuthenticateExpiredToken(t *testing.T) {
	auth := NewJWTAuth(testSecret)
	expiredToken := generateTestToken(t, "expired-user", time.Now().Add(-time.Hour))

	req := httptest.NewRequest(http.MethodGet, "/ws?token="+expiredToken, nil)

	userID, err := auth.Authenticate(req)

	assert.Error(t, err)
	assert.Equal(t, ErrExpiredToken, err)
	assert.Empty(t, userID)
}

func TestNewNoAuth(t *testing.T) {
	auth := NewNoAuth()

	assert.NotNil(t, auth)
}

func TestNoAuth_Authenticate_Anonymous(t *testing.T) {
	auth := NewNoAuth()

	req := httptest.NewRequest(http.MethodGet, "/ws", nil)

	userID, err := auth.Authenticate(req)

	assert.NoError(t, err)
	assert.Equal(t, "anonymous", userID)
}

func TestNoAuth_Authenticate_WithUserIDParam(t *testing.T) {
	auth := NewNoAuth()

	req := httptest.NewRequest(http.MethodGet, "/ws?user_id=test-user-123", nil)

	userID, err := auth.Authenticate(req)

	assert.NoError(t, err)
	assert.Equal(t, "test-user-123", userID)
}

func TestNoAuth_Authenticate_EmptyUserIDParam(t *testing.T) {
	auth := NewNoAuth()

	req := httptest.NewRequest(http.MethodGet, "/ws?user_id=", nil)

	userID, err := auth.Authenticate(req)

	assert.NoError(t, err)
	assert.Equal(t, "anonymous", userID) // Empty string treated as anonymous
}

func TestAuthenticator_Interface(t *testing.T) {
	// Verify both types implement Authenticator interface
	var _ Authenticator = (*JWTAuth)(nil)
	var _ Authenticator = (*NoAuth)(nil)
}

func TestErrMissingToken(t *testing.T) {
	assert.Equal(t, "missing authentication token", ErrMissingToken.Error())
}

func TestErrInvalidToken(t *testing.T) {
	assert.Equal(t, "invalid authentication token", ErrInvalidToken.Error())
}

func TestErrExpiredToken(t *testing.T) {
	assert.Equal(t, "token has expired", ErrExpiredToken.Error())
}

func TestNoAuth_NeverFails(t *testing.T) {
	auth := NewNoAuth()

	// Various request types should all succeed
	requests := []*http.Request{
		httptest.NewRequest(http.MethodGet, "/ws", nil),
		httptest.NewRequest(http.MethodGet, "/ws?foo=bar", nil),
		httptest.NewRequest(http.MethodPost, "/ws", nil),
	}

	for i, req := range requests {
		userID, err := auth.Authenticate(req)
		assert.NoError(t, err, "request %d should not fail", i)
		assert.NotEmpty(t, userID, "request %d should return userID", i)
	}
}

func TestJWTAuth_QueryParamOverWebSocketProtocol(t *testing.T) {
	auth := NewJWTAuth(testSecret)
	queryToken := generateTestToken(t, "query-priority-user", time.Now().Add(time.Hour))
	protocolToken := generateTestToken(t, "protocol-priority-user", time.Now().Add(time.Hour))

	// Query param should take priority over WebSocket protocol
	req := httptest.NewRequest(http.MethodGet, "/ws?token="+queryToken, nil)
	req.Header.Set("Sec-WebSocket-Protocol", "auth-"+protocolToken)

	userID, err := auth.Authenticate(req)

	assert.NoError(t, err)
	assert.Equal(t, "query-priority-user", userID) // From query param, not protocol
}

func TestJWTClaims_Structure(t *testing.T) {
	claims := JWTClaims{
		UserID: "test-user",
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   "test-user",
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
		},
	}

	assert.Equal(t, "test-user", claims.UserID)
	assert.Equal(t, "test-user", claims.Subject)
	assert.NotNil(t, claims.ExpiresAt)
}

func TestJWTAuth_GenerateToken_NoExpiration(t *testing.T) {
	auth := NewJWTAuth(testSecret)

	// Generate token without expiration
	token, err := auth.GenerateToken("user-123", nil)
	require.NoError(t, err)

	// Should be valid
	userID, err := auth.validateToken(token)
	assert.NoError(t, err)
	assert.Equal(t, "user-123", userID)
}

func TestJWTAuth_TokenRoundTrip(t *testing.T) {
	auth := NewJWTAuth(testSecret)
	expectedUserID := "round-trip-user-12345"

	// Generate token
	token, err := auth.GenerateToken(expectedUserID, jwt.NewNumericDate(time.Now().Add(time.Hour)))
	require.NoError(t, err)

	// Validate it
	actualUserID, err := auth.validateToken(token)

	assert.NoError(t, err)
	assert.Equal(t, expectedUserID, actualUserID)
}

func TestJWTAuth_DifferentSecrets(t *testing.T) {
	auth1 := NewJWTAuth("secret-key-1")
	auth2 := NewJWTAuth("secret-key-2")

	// Generate with auth1
	token1, err := auth1.GenerateToken("user-1", jwt.NewNumericDate(time.Now().Add(time.Hour)))
	require.NoError(t, err)

	token2, err := auth2.GenerateToken("user-2", jwt.NewNumericDate(time.Now().Add(time.Hour)))
	require.NoError(t, err)

	// Validate with same auth - should work
	userID1, err := auth1.validateToken(token1)
	assert.NoError(t, err)
	assert.Equal(t, "user-1", userID1)

	userID2, err := auth2.validateToken(token2)
	assert.NoError(t, err)
	assert.Equal(t, "user-2", userID2)

	// Cross-validate - should fail
	_, err = auth1.validateToken(token2)
	assert.Error(t, err)

	_, err = auth2.validateToken(token1)
	assert.Error(t, err)
}
