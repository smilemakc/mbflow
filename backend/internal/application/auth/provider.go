package auth

import (
	"context"

	pkgmodels "github.com/smilemakc/mbflow/pkg/models"
)

// ProviderType represents the type of authentication provider
type ProviderType string

const (
	ProviderTypeBuiltin ProviderType = "builtin"
	ProviderTypeGateway ProviderType = "gateway"
	ProviderTypeOIDC    ProviderType = "oidc"
)

// Credentials represents authentication credentials
type Credentials struct {
	Email        string `json:"email"`
	Password     string `json:"password"`
	Username     string `json:"username,omitempty"`
	RefreshToken string `json:"refresh_token,omitempty"`

	// OAuth/OIDC fields
	Code        string `json:"code,omitempty"`
	State       string `json:"state,omitempty"`
	RedirectURI string `json:"redirect_uri,omitempty"`

	// Request metadata
	IPAddress string `json:"ip_address,omitempty"`
	UserAgent string `json:"user_agent,omitempty"`
}

// ProviderAuthResult represents the result of authentication from a provider
type ProviderAuthResult struct {
	User         *pkgmodels.User `json:"user"`
	AccessToken  string          `json:"access_token"`
	RefreshToken string          `json:"refresh_token,omitempty"`
	IDToken      string          `json:"id_token,omitempty"`
	ExpiresIn    int             `json:"expires_in"`
	TokenType    string          `json:"token_type"`
	Scope        string          `json:"scope,omitempty"`
}

// AuthProvider defines the interface for authentication providers
type AuthProvider interface {
	// GetType returns the provider type
	GetType() ProviderType

	// Authenticate authenticates a user with the given credentials
	Authenticate(ctx context.Context, creds *Credentials) (*ProviderAuthResult, error)

	// ValidateToken validates an access token and returns user claims
	ValidateToken(ctx context.Context, token string) (*JWTClaims, error)

	// RefreshToken refreshes an access token using a refresh token
	RefreshToken(ctx context.Context, refreshToken string) (*ProviderAuthResult, error)

	// GetAuthorizationURL returns the OAuth2 authorization URL (for OIDC/OAuth providers)
	// Returns empty string for providers that don't support OAuth flow
	GetAuthorizationURL(state, nonce string) string

	// HandleCallback handles OAuth2 callback (for OIDC/OAuth providers)
	// Returns error for providers that don't support OAuth flow
	HandleCallback(ctx context.Context, code, state string) (*ProviderAuthResult, error)

	// IsAvailable checks if the provider is available/configured
	IsAvailable() bool

	// GetUserInfo retrieves user information from the provider
	GetUserInfo(ctx context.Context, accessToken string) (*pkgmodels.User, error)
}

// ProviderConfig contains configuration for initializing providers
type ProviderConfig struct {
	Type         ProviderType
	Enabled      bool
	Priority     int
	ClientID     string
	ClientSecret string
	IssuerURL    string
	JWKSURL      string
	RedirectURL  string
	Scopes       []string
}
