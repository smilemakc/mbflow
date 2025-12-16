package auth

import (
	"context"
	"errors"

	pkgmodels "github.com/smilemakc/mbflow/pkg/models"
)

// BuiltinProvider implements AuthProvider using local database authentication
type BuiltinProvider struct {
	authService *Service
}

// NewBuiltinProvider creates a new built-in authentication provider
func NewBuiltinProvider(authService *Service) *BuiltinProvider {
	return &BuiltinProvider{
		authService: authService,
	}
}

// GetType returns the provider type
func (p *BuiltinProvider) GetType() ProviderType {
	return ProviderTypeBuiltin
}

// Authenticate authenticates a user with email and password
func (p *BuiltinProvider) Authenticate(ctx context.Context, creds *Credentials) (*ProviderAuthResult, error) {
	result, err := p.authService.Login(ctx, &LoginRequest{
		Email:    creds.Email,
		Password: creds.Password,
	}, creds.IPAddress, creds.UserAgent)
	if err != nil {
		return nil, err
	}

	return &ProviderAuthResult{
		User:         result.User,
		AccessToken:  result.AccessToken,
		RefreshToken: result.RefreshToken,
		ExpiresIn:    result.ExpiresIn,
		TokenType:    result.TokenType,
	}, nil
}

// ValidateToken validates an access token and returns claims
func (p *BuiltinProvider) ValidateToken(ctx context.Context, token string) (*JWTClaims, error) {
	return p.authService.ValidateToken(ctx, token)
}

// RefreshToken refreshes an access token using a refresh token
func (p *BuiltinProvider) RefreshToken(ctx context.Context, refreshToken string) (*ProviderAuthResult, error) {
	result, err := p.authService.RefreshToken(ctx, refreshToken)
	if err != nil {
		return nil, err
	}

	return &ProviderAuthResult{
		User:         result.User,
		AccessToken:  result.AccessToken,
		RefreshToken: result.RefreshToken,
		ExpiresIn:    result.ExpiresIn,
		TokenType:    result.TokenType,
	}, nil
}

// GetAuthorizationURL returns empty string as builtin provider doesn't support OAuth
func (p *BuiltinProvider) GetAuthorizationURL(state, nonce string) string {
	return ""
}

// HandleCallback returns error as builtin provider doesn't support OAuth
func (p *BuiltinProvider) HandleCallback(ctx context.Context, code, state string) (*ProviderAuthResult, error) {
	return nil, errors.New("builtin provider does not support OAuth callback")
}

// IsAvailable returns true as builtin provider is always available
func (p *BuiltinProvider) IsAvailable() bool {
	return true
}

// GetUserInfo retrieves user information from the local database
func (p *BuiltinProvider) GetUserInfo(ctx context.Context, accessToken string) (*pkgmodels.User, error) {
	claims, err := p.ValidateToken(ctx, accessToken)
	if err != nil {
		return nil, err
	}

	return p.authService.GetCurrentUser(ctx, claims.UserID)
}
