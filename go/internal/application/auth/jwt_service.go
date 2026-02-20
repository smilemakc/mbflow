package auth

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/smilemakc/mbflow/go/internal/config"
	"github.com/smilemakc/mbflow/go/pkg/models"
)

var (
	ErrInvalidToken     = errors.New("invalid token")
	ErrExpiredToken     = errors.New("token has expired")
	ErrInvalidClaims    = errors.New("invalid token claims")
	ErrTokenNotYetValid = errors.New("token is not yet valid")
)

// JWTClaims represents the claims stored in a JWT token
type JWTClaims struct {
	jwt.RegisteredClaims
	UserID   string   `json:"user_id"`
	Email    string   `json:"email"`
	Username string   `json:"username"`
	IsAdmin  bool     `json:"is_admin"`
	Roles    []string `json:"roles"`
}

// JWTService handles JWT token generation and validation
type JWTService struct {
	secret            []byte
	issuer            string
	accessExpiryHrs   int
	refreshExpiryDays int
}

// NewJWTService creates a new JWTService
func NewJWTService(cfg *config.AuthConfig) *JWTService {
	issuer := "mbflow"
	if cfg.IssuerURL != "" {
		issuer = cfg.IssuerURL
	}

	return &JWTService{
		secret:            []byte(cfg.JWTSecret),
		issuer:            issuer,
		accessExpiryHrs:   cfg.JWTExpirationHours,
		refreshExpiryDays: cfg.RefreshExpiryDays,
	}
}

// GenerateAccessToken generates a new JWT access token for a user
func (s *JWTService) GenerateAccessToken(user *models.User) (string, time.Time, error) {
	expiresAt := time.Now().Add(time.Duration(s.accessExpiryHrs) * time.Hour)

	claims := &JWTClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   user.ID,
			Issuer:    s.issuer,
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			NotBefore: jwt.NewNumericDate(time.Now()),
		},
		UserID:   user.ID,
		Email:    user.Email,
		Username: user.Username,
		IsAdmin:  user.IsAdmin,
		Roles:    user.Roles,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signedToken, err := token.SignedString(s.secret)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("failed to sign token: %w", err)
	}

	return signedToken, expiresAt, nil
}

// GenerateRefreshToken generates a random refresh token
func (s *JWTService) GenerateRefreshToken() (string, time.Time, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", time.Time{}, fmt.Errorf("failed to generate refresh token: %w", err)
	}

	expiresAt := time.Now().Add(time.Duration(s.refreshExpiryDays) * 24 * time.Hour)
	return hex.EncodeToString(bytes), expiresAt, nil
}

// ValidateAccessToken validates a JWT access token and returns the claims
func (s *JWTService) ValidateAccessToken(tokenString string) (*JWTClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &JWTClaims{}, func(token *jwt.Token) (any, error) {
		// Validate signing method
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return s.secret, nil
	})

	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, ErrExpiredToken
		}
		if errors.Is(err, jwt.ErrTokenNotValidYet) {
			return nil, ErrTokenNotYetValid
		}
		return nil, fmt.Errorf("%w: %v", ErrInvalidToken, err)
	}

	claims, ok := token.Claims.(*JWTClaims)
	if !ok || !token.Valid {
		return nil, ErrInvalidClaims
	}

	return claims, nil
}

// ExtractClaimsFromExpiredToken extracts claims from an expired token
// This is useful for refresh token flow where we need user info from expired access token
func (s *JWTService) ExtractClaimsFromExpiredToken(tokenString string) (*JWTClaims, error) {
	token, _, err := jwt.NewParser().ParseUnverified(tokenString, &JWTClaims{})
	if err != nil {
		return nil, fmt.Errorf("failed to parse token: %w", err)
	}

	claims, ok := token.Claims.(*JWTClaims)
	if !ok {
		return nil, ErrInvalidClaims
	}

	return claims, nil
}

// GetAccessTokenExpiry returns the access token expiry duration in seconds
func (s *JWTService) GetAccessTokenExpiry() int {
	return s.accessExpiryHrs * 3600
}

// GetRefreshTokenExpiry returns the refresh token expiry duration in seconds
func (s *JWTService) GetRefreshTokenExpiry() int {
	return s.refreshExpiryDays * 24 * 3600
}
