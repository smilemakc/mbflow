package websocket

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var (
	// ErrMissingToken is returned when no authentication token is provided
	ErrMissingToken = errors.New("missing authentication token")
	// ErrInvalidToken is returned when the token is invalid
	ErrInvalidToken = errors.New("invalid authentication token")
	// ErrExpiredToken is returned when the token has expired
	ErrExpiredToken = errors.New("token has expired")
)

// Authenticator defines the interface for authenticating WebSocket connections
type Authenticator interface {
	// Authenticate extracts and validates user identity from the request.
	// Returns userID on success, or error if authentication fails.
	Authenticate(r *http.Request) (userID string, err error)
}

// JWTAuth implements Authenticator using JWT tokens
type JWTAuth struct {
	secretKey string
	// In production, add: validator *jwt.Validator
}

// NewJWTAuth creates a new JWTAuth instance
func NewJWTAuth(secretKey string) *JWTAuth {
	return &JWTAuth{
		secretKey: secretKey,
	}
}

// Authenticate extracts and validates JWT from the request.
// It tries multiple sources in order:
// 1. Authorization header (Bearer token)
// 2. Query parameter "token"
// 3. Sec-WebSocket-Protocol header (for browsers that can't set custom headers)
func (a *JWTAuth) Authenticate(r *http.Request) (string, error) {
	// Try Authorization header first
	authHeader := r.Header.Get("Authorization")
	if authHeader != "" && strings.HasPrefix(authHeader, "Bearer ") {
		token := strings.TrimPrefix(authHeader, "Bearer ")
		return a.validateToken(token)
	}

	// Try query parameter (for WebSocket connections where headers are limited)
	token := r.URL.Query().Get("token")
	if token != "" {
		return a.validateToken(token)
	}

	// Try Sec-WebSocket-Protocol for token
	// Format: "auth-<token>" as one of the protocols
	protocols := r.Header.Get("Sec-WebSocket-Protocol")
	if protocols != "" {
		parts := strings.Split(protocols, ",")
		for _, p := range parts {
			p = strings.TrimSpace(p)
			if strings.HasPrefix(p, "auth-") {
				token := strings.TrimPrefix(p, "auth-")
				return a.validateToken(token)
			}
		}
	}

	return "", ErrMissingToken
}

// JWTClaims represents the claims in the JWT token
type JWTClaims struct {
	UserID string `json:"user_id"`
	jwt.RegisteredClaims
}

// validateToken validates a JWT token and extracts the user ID.
func (a *JWTAuth) validateToken(tokenString string) (string, error) {
	if tokenString == "" {
		return "", ErrInvalidToken
	}

	// Parse and validate the token
	token, err := jwt.ParseWithClaims(tokenString, &JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
		// Validate signing method
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, ErrInvalidToken
		}
		return []byte(a.secretKey), nil
	})

	if err != nil {
		// Check for specific JWT errors
		if errors.Is(err, jwt.ErrTokenExpired) {
			return "", ErrExpiredToken
		}
		return "", ErrInvalidToken
	}

	// Extract claims
	claims, ok := token.Claims.(*JWTClaims)
	if !ok || !token.Valid {
		return "", ErrInvalidToken
	}

	// Get user ID from claims
	userID := claims.UserID
	if userID == "" {
		// Fallback to subject if user_id claim is not set
		userID = claims.Subject
	}

	if userID == "" {
		return "", ErrInvalidToken
	}

	return userID, nil
}

// GenerateToken creates a new JWT token for the given user ID.
// This is a helper function for testing and token generation.
func (a *JWTAuth) GenerateToken(userID string, expiresAt *jwt.NumericDate) (string, error) {
	claims := JWTClaims{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID,
			ExpiresAt: expiresAt,
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(a.secretKey))
}

// NoAuth is an Authenticator that allows all connections without authentication.
// Use this for development or when authentication is handled elsewhere.
type NoAuth struct{}

// NewNoAuth creates a new NoAuth instance
func NewNoAuth() *NoAuth {
	return &NoAuth{}
}

// Authenticate always succeeds with an anonymous user
func (a *NoAuth) Authenticate(r *http.Request) (string, error) {
	// Try to get user_id from query param for debugging
	if userID := r.URL.Query().Get("user_id"); userID != "" {
		return userID, nil
	}
	return "anonymous", nil
}
