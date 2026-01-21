package auth

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/smilemakc/mbflow/internal/config"
	pkgmodels "github.com/smilemakc/mbflow/pkg/models"
)

var (
	ErrNoProvidersAvailable = errors.New("no authentication providers available")
	ErrProviderNotFound     = errors.New("authentication provider not found")
	ErrAllProvidersFailed   = errors.New("all authentication providers failed")
)

// ProviderManager manages multiple authentication providers with fallback support
type ProviderManager struct {
	mu             sync.RWMutex
	providers      map[ProviderType]AuthProvider
	primaryType    ProviderType
	fallbackType   ProviderType
	enableFallback bool
}

// NewProviderManager creates a new provider manager
func NewProviderManager(cfg *config.AuthConfig, authService *Service) (*ProviderManager, error) {
	pm := &ProviderManager{
		providers:      make(map[ProviderType]AuthProvider),
		enableFallback: cfg.EnableFallback,
	}

	// Determine primary provider based on mode
	switch cfg.Mode {
	case "grpc":
		pm.primaryType = ProviderTypeGRPC
		pm.fallbackType = ProviderTypeBuiltin
	case "gateway", "oidc":
		pm.primaryType = ProviderTypeGateway
		pm.fallbackType = ProviderTypeBuiltin
	case "builtin", "local", "":
		pm.primaryType = ProviderTypeBuiltin
		pm.fallbackType = ProviderTypeGateway
	case "hybrid":
		pm.primaryType = ProviderTypeGateway
		pm.fallbackType = ProviderTypeBuiltin
		pm.enableFallback = true
	default:
		pm.primaryType = ProviderTypeBuiltin
	}

	// Initialize builtin provider (always available)
	builtinProvider := NewBuiltinProvider(authService)
	pm.providers[ProviderTypeBuiltin] = builtinProvider

	// Initialize gateway provider if configured
	if cfg.IssuerURL != "" && cfg.ClientID != "" {
		gatewayProvider, err := NewGatewayProvider(cfg)
		if err != nil {
			// Log warning but don't fail - gateway might not be available
			fmt.Printf("Warning: Failed to initialize gateway provider: %v\n", err)
		} else if gatewayProvider.IsAvailable() {
			pm.providers[ProviderTypeGateway] = gatewayProvider
		}
	}

	// Initialize gRPC provider if configured
	if cfg.GRPCAddress != "" {
		grpcProvider, err := NewGRPCProvider(cfg)
		if err != nil {
			// Log warning but don't fail - gRPC provider might not be available
			fmt.Printf("Warning: Failed to initialize gRPC provider: %v\n", err)
		} else if grpcProvider.IsAvailable() {
			pm.providers[ProviderTypeGRPC] = grpcProvider
		}
	}

	return pm, nil
}

// GetProvider returns a specific provider by type
func (pm *ProviderManager) GetProvider(providerType ProviderType) (AuthProvider, error) {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	provider, ok := pm.providers[providerType]
	if !ok || !provider.IsAvailable() {
		return nil, ErrProviderNotFound
	}

	return provider, nil
}

// GetPrimaryProvider returns the primary provider
func (pm *ProviderManager) GetPrimaryProvider() (AuthProvider, error) {
	return pm.GetProvider(pm.primaryType)
}

// GetFallbackProvider returns the fallback provider
func (pm *ProviderManager) GetFallbackProvider() (AuthProvider, error) {
	if !pm.enableFallback {
		return nil, ErrProviderNotFound
	}
	return pm.GetProvider(pm.fallbackType)
}

// Authenticate attempts authentication with the primary provider, falling back if enabled
func (pm *ProviderManager) Authenticate(ctx context.Context, creds *Credentials) (*ProviderAuthResult, error) {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	// Try primary provider
	if provider, ok := pm.providers[pm.primaryType]; ok && provider.IsAvailable() {
		result, err := provider.Authenticate(ctx, creds)
		if err == nil {
			return result, nil
		}

		// If fallback is disabled, return the error
		if !pm.enableFallback {
			return nil, err
		}

		// Log primary failure and try fallback
		fmt.Printf("Primary provider (%s) failed: %v, trying fallback\n", pm.primaryType, err)
	}

	// Try fallback provider
	if pm.enableFallback {
		if provider, ok := pm.providers[pm.fallbackType]; ok && provider.IsAvailable() {
			result, err := provider.Authenticate(ctx, creds)
			if err == nil {
				return result, nil
			}
			return nil, fmt.Errorf("%w: %v", ErrAllProvidersFailed, err)
		}
	}

	return nil, ErrNoProvidersAvailable
}

// ValidateToken validates a token using the appropriate provider
func (pm *ProviderManager) ValidateToken(ctx context.Context, token string) (*JWTClaims, error) {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	// Try primary provider first
	if provider, ok := pm.providers[pm.primaryType]; ok && provider.IsAvailable() {
		claims, err := provider.ValidateToken(ctx, token)
		if err == nil {
			return claims, nil
		}

		// If not expired error and fallback enabled, try fallback
		if pm.enableFallback && !errors.Is(err, ErrExpiredToken) {
			if fbProvider, fbOk := pm.providers[pm.fallbackType]; fbOk && fbProvider.IsAvailable() {
				return fbProvider.ValidateToken(ctx, token)
			}
		}
		return nil, err
	}

	// Try fallback if primary not available
	if pm.enableFallback {
		if provider, ok := pm.providers[pm.fallbackType]; ok && provider.IsAvailable() {
			return provider.ValidateToken(ctx, token)
		}
	}

	return nil, ErrNoProvidersAvailable
}

// RefreshToken refreshes a token using the appropriate provider
func (pm *ProviderManager) RefreshToken(ctx context.Context, refreshToken string) (*ProviderAuthResult, error) {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	// Try primary provider first
	if provider, ok := pm.providers[pm.primaryType]; ok && provider.IsAvailable() {
		result, err := provider.RefreshToken(ctx, refreshToken)
		if err == nil {
			return result, nil
		}

		// If fallback enabled, try fallback
		if pm.enableFallback {
			if fbProvider, fbOk := pm.providers[pm.fallbackType]; fbOk && fbProvider.IsAvailable() {
				return fbProvider.RefreshToken(ctx, refreshToken)
			}
		}
		return nil, err
	}

	// Try fallback if primary not available
	if pm.enableFallback {
		if provider, ok := pm.providers[pm.fallbackType]; ok && provider.IsAvailable() {
			return provider.RefreshToken(ctx, refreshToken)
		}
	}

	return nil, ErrNoProvidersAvailable
}

// GetAuthorizationURL returns the OAuth2 authorization URL from the gateway provider
func (pm *ProviderManager) GetAuthorizationURL(state, nonce string) string {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	if provider, ok := pm.providers[ProviderTypeGateway]; ok && provider.IsAvailable() {
		return provider.GetAuthorizationURL(state, nonce)
	}

	return ""
}

// HandleOAuthCallback handles OAuth2 callback
func (pm *ProviderManager) HandleOAuthCallback(ctx context.Context, code, state string) (*ProviderAuthResult, error) {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	if provider, ok := pm.providers[ProviderTypeGateway]; ok && provider.IsAvailable() {
		return provider.HandleCallback(ctx, code, state)
	}

	return nil, ErrProviderNotFound
}

// GetUserInfo retrieves user information from the appropriate provider
func (pm *ProviderManager) GetUserInfo(ctx context.Context, accessToken string) (*pkgmodels.User, error) {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	// Try primary provider first
	if provider, ok := pm.providers[pm.primaryType]; ok && provider.IsAvailable() {
		user, err := provider.GetUserInfo(ctx, accessToken)
		if err == nil {
			return user, nil
		}

		// Try fallback if enabled
		if pm.enableFallback {
			if fbProvider, fbOk := pm.providers[pm.fallbackType]; fbOk && fbProvider.IsAvailable() {
				return fbProvider.GetUserInfo(ctx, accessToken)
			}
		}
		return nil, err
	}

	// Try fallback if primary not available
	if pm.enableFallback {
		if provider, ok := pm.providers[pm.fallbackType]; ok && provider.IsAvailable() {
			return provider.GetUserInfo(ctx, accessToken)
		}
	}

	return nil, ErrNoProvidersAvailable
}

// GetAvailableProviders returns a list of available provider types
func (pm *ProviderManager) GetAvailableProviders() []ProviderType {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	var types []ProviderType
	for providerType, provider := range pm.providers {
		if provider.IsAvailable() {
			types = append(types, providerType)
		}
	}
	return types
}

// IsGatewayAvailable returns whether the gateway provider is available
func (pm *ProviderManager) IsGatewayAvailable() bool {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	if provider, ok := pm.providers[ProviderTypeGateway]; ok {
		return provider.IsAvailable()
	}
	return false
}

// GetMode returns the current authentication mode
func (pm *ProviderManager) GetMode() string {
	if pm.enableFallback {
		return "hybrid"
	}
	return string(pm.primaryType)
}
