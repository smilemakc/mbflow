package auth

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/smilemakc/mbflow/internal/config"
	pkgmodels "github.com/smilemakc/mbflow/pkg/models"
	"golang.org/x/oauth2"
)

var (
	ErrProviderNotConfigured = errors.New("auth gateway provider is not configured")
	ErrOIDCDiscoveryFailed   = errors.New("OIDC discovery failed")
	ErrTokenExchangeFailed   = errors.New("token exchange failed")
	ErrInvalidIDToken        = errors.New("invalid ID token")
	ErrMissingUserInfo       = errors.New("missing user information")
)

// GatewayProvider implements AuthProvider using external OAuth2/OIDC gateway
type GatewayProvider struct {
	config       *config.AuthConfig
	oidcProvider *oidc.Provider
	oauth2Config *oauth2.Config
	verifier     *oidc.IDTokenVerifier
	available    bool
}

// NewGatewayProvider creates a new gateway authentication provider
func NewGatewayProvider(cfg *config.AuthConfig) (*GatewayProvider, error) {
	provider := &GatewayProvider{
		config:    cfg,
		available: false,
	}

	// Only initialize if gateway URL is configured
	if cfg.IssuerURL == "" {
		return provider, nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Perform OIDC discovery
	oidcProvider, err := oidc.NewProvider(ctx, cfg.IssuerURL)
	if err != nil {
		return provider, fmt.Errorf("%w: %v", ErrOIDCDiscoveryFailed, err)
	}

	provider.oidcProvider = oidcProvider

	// Configure OAuth2
	scopes := []string{oidc.ScopeOpenID, "profile", "email"}
	provider.oauth2Config = &oauth2.Config{
		ClientID:     cfg.ClientID,
		ClientSecret: cfg.ClientSecret,
		RedirectURL:  cfg.RedirectURL,
		Endpoint:     oidcProvider.Endpoint(),
		Scopes:       scopes,
	}

	// Configure ID token verifier
	provider.verifier = oidcProvider.Verifier(&oidc.Config{
		ClientID: cfg.ClientID,
	})

	provider.available = true
	return provider, nil
}

// GetType returns the provider type
func (p *GatewayProvider) GetType() ProviderType {
	return ProviderTypeGateway
}

// Authenticate authenticates using OAuth2 password grant (if supported)
// For most OIDC providers, this won't be supported and OAuth flow should be used instead
func (p *GatewayProvider) Authenticate(ctx context.Context, creds *Credentials) (*ProviderAuthResult, error) {
	if !p.available {
		return nil, ErrProviderNotConfigured
	}

	// Try password grant (resource owner password credentials)
	// Note: This is deprecated in OAuth 2.1 and many providers don't support it
	token, err := p.oauth2Config.PasswordCredentialsToken(ctx, creds.Email, creds.Password)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrTokenExchangeFailed, err)
	}

	return p.processOAuthToken(ctx, token)
}

// ValidateToken validates an access token using OIDC provider
func (p *GatewayProvider) ValidateToken(ctx context.Context, token string) (*JWTClaims, error) {
	if !p.available {
		return nil, ErrProviderNotConfigured
	}

	// Verify the token as an ID token
	idToken, err := p.verifier.Verify(ctx, token)
	if err != nil {
		// Token might be an access token, try to get user info
		userInfo, uiErr := p.oidcProvider.UserInfo(ctx, oauth2.StaticTokenSource(&oauth2.Token{
			AccessToken: token,
		}))
		if uiErr != nil {
			return nil, fmt.Errorf("%w: %v", ErrInvalidToken, err)
		}

		return &JWTClaims{
			UserID:   userInfo.Subject,
			Email:    userInfo.Email,
			Username: userInfo.Email, // Use email as username fallback
		}, nil
	}

	var claims struct {
		Email         string   `json:"email"`
		EmailVerified bool     `json:"email_verified"`
		Name          string   `json:"name"`
		PreferredUser string   `json:"preferred_username"`
		Groups        []string `json:"groups"`
		Roles         []string `json:"roles"`
	}

	if err := idToken.Claims(&claims); err != nil {
		return nil, fmt.Errorf("failed to parse claims: %w", err)
	}

	username := claims.PreferredUser
	if username == "" {
		username = claims.Email
	}

	// Combine groups and roles
	roles := append(claims.Roles, claims.Groups...)

	return &JWTClaims{
		UserID:   idToken.Subject,
		Email:    claims.Email,
		Username: username,
		Roles:    roles,
	}, nil
}

// RefreshToken refreshes an access token using a refresh token
func (p *GatewayProvider) RefreshToken(ctx context.Context, refreshToken string) (*ProviderAuthResult, error) {
	if !p.available {
		return nil, ErrProviderNotConfigured
	}

	// Create a token source with just the refresh token
	token := &oauth2.Token{
		RefreshToken: refreshToken,
	}

	// Use the token source to get a new token
	tokenSource := p.oauth2Config.TokenSource(ctx, token)
	newToken, err := tokenSource.Token()
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrTokenExchangeFailed, err)
	}

	return p.processOAuthToken(ctx, newToken)
}

// GetAuthorizationURL returns the OAuth2 authorization URL
func (p *GatewayProvider) GetAuthorizationURL(state, nonce string) string {
	if !p.available {
		return ""
	}

	opts := []oauth2.AuthCodeOption{
		oauth2.SetAuthURLParam("nonce", nonce),
	}

	return p.oauth2Config.AuthCodeURL(state, opts...)
}

// HandleCallback handles OAuth2 callback
func (p *GatewayProvider) HandleCallback(ctx context.Context, code, state string) (*ProviderAuthResult, error) {
	if !p.available {
		return nil, ErrProviderNotConfigured
	}

	// Exchange code for token
	token, err := p.oauth2Config.Exchange(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrTokenExchangeFailed, err)
	}

	return p.processOAuthToken(ctx, token)
}

// IsAvailable returns whether the provider is configured and available
func (p *GatewayProvider) IsAvailable() bool {
	return p.available
}

// GetUserInfo retrieves user information from the OIDC provider
func (p *GatewayProvider) GetUserInfo(ctx context.Context, accessToken string) (*pkgmodels.User, error) {
	if !p.available {
		return nil, ErrProviderNotConfigured
	}

	userInfo, err := p.oidcProvider.UserInfo(ctx, oauth2.StaticTokenSource(&oauth2.Token{
		AccessToken: accessToken,
	}))
	if err != nil {
		return nil, fmt.Errorf("failed to get user info: %w", err)
	}

	var claims struct {
		Name          string   `json:"name"`
		PreferredUser string   `json:"preferred_username"`
		Groups        []string `json:"groups"`
		Roles         []string `json:"roles"`
	}

	if err := userInfo.Claims(&claims); err != nil {
		return nil, fmt.Errorf("failed to parse user claims: %w", err)
	}

	username := claims.PreferredUser
	if username == "" {
		username = userInfo.Email
	}

	roles := append(claims.Roles, claims.Groups...)

	return &pkgmodels.User{
		ID:       userInfo.Subject,
		Email:    userInfo.Email,
		Username: username,
		FullName: claims.Name,
		IsActive: true,
		Roles:    roles,
	}, nil
}

// processOAuthToken processes an OAuth token and extracts user information
func (p *GatewayProvider) processOAuthToken(ctx context.Context, token *oauth2.Token) (*ProviderAuthResult, error) {
	// Extract ID token if present
	rawIDToken, ok := token.Extra("id_token").(string)
	if !ok || rawIDToken == "" {
		// No ID token, get user info from userinfo endpoint
		userInfo, err := p.oidcProvider.UserInfo(ctx, oauth2.StaticTokenSource(token))
		if err != nil {
			return nil, fmt.Errorf("%w: %v", ErrMissingUserInfo, err)
		}

		var claims struct {
			Name          string   `json:"name"`
			PreferredUser string   `json:"preferred_username"`
			Groups        []string `json:"groups"`
			Roles         []string `json:"roles"`
		}

		if err := userInfo.Claims(&claims); err != nil {
			return nil, fmt.Errorf("failed to parse claims: %w", err)
		}

		username := claims.PreferredUser
		if username == "" {
			username = userInfo.Email
		}

		roles := append(claims.Roles, claims.Groups...)

		return &ProviderAuthResult{
			User: &pkgmodels.User{
				ID:       userInfo.Subject,
				Email:    userInfo.Email,
				Username: username,
				FullName: claims.Name,
				IsActive: true,
				Roles:    roles,
			},
			AccessToken:  token.AccessToken,
			RefreshToken: token.RefreshToken,
			ExpiresIn:    int(time.Until(token.Expiry).Seconds()),
			TokenType:    token.TokenType,
		}, nil
	}

	// Verify ID token
	idToken, err := p.verifier.Verify(ctx, rawIDToken)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidIDToken, err)
	}

	var claims struct {
		Email         string   `json:"email"`
		EmailVerified bool     `json:"email_verified"`
		Name          string   `json:"name"`
		PreferredUser string   `json:"preferred_username"`
		Groups        []string `json:"groups"`
		Roles         []string `json:"roles"`
	}

	if err := idToken.Claims(&claims); err != nil {
		return nil, fmt.Errorf("failed to parse ID token claims: %w", err)
	}

	username := claims.PreferredUser
	if username == "" {
		username = claims.Email
	}

	// Combine roles and groups
	roles := append(claims.Roles, claims.Groups...)

	// Check for admin role
	isAdmin := false
	for _, role := range roles {
		if strings.EqualFold(role, "admin") || strings.EqualFold(role, "administrator") {
			isAdmin = true
			break
		}
	}

	return &ProviderAuthResult{
		User: &pkgmodels.User{
			ID:       idToken.Subject,
			Email:    claims.Email,
			Username: username,
			FullName: claims.Name,
			IsActive: true,
			IsAdmin:  isAdmin,
			Roles:    roles,
		},
		AccessToken:  token.AccessToken,
		RefreshToken: token.RefreshToken,
		IDToken:      rawIDToken,
		ExpiresIn:    int(time.Until(token.Expiry).Seconds()),
		TokenType:    token.TokenType,
	}, nil
}

// GenerateState generates a random state parameter for OAuth2
func GenerateState() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil
}

// GenerateNonce generates a random nonce for OIDC
func GenerateNonce() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil
}
