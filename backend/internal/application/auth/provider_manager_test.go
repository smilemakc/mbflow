package auth

import (
	"context"
	"errors"
	"testing"

	"github.com/smilemakc/mbflow/internal/config"
	pkgmodels "github.com/smilemakc/mbflow/pkg/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// MockAuthProvider is a mock implementation of AuthProvider interface
type MockAuthProvider struct {
	mock.Mock
	providerType ProviderType
}

func (m *MockAuthProvider) GetType() ProviderType {
	return m.providerType
}

func (m *MockAuthProvider) Authenticate(ctx context.Context, creds *Credentials) (*ProviderAuthResult, error) {
	args := m.Called(ctx, creds)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*ProviderAuthResult), args.Error(1)
}

func (m *MockAuthProvider) ValidateToken(ctx context.Context, token string) (*JWTClaims, error) {
	args := m.Called(ctx, token)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*JWTClaims), args.Error(1)
}

func (m *MockAuthProvider) RefreshToken(ctx context.Context, refreshToken string) (*ProviderAuthResult, error) {
	args := m.Called(ctx, refreshToken)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*ProviderAuthResult), args.Error(1)
}

func (m *MockAuthProvider) GetAuthorizationURL(state, nonce string) string {
	args := m.Called(state, nonce)
	return args.String(0)
}

func (m *MockAuthProvider) HandleCallback(ctx context.Context, code, state string) (*ProviderAuthResult, error) {
	args := m.Called(ctx, code, state)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*ProviderAuthResult), args.Error(1)
}

func (m *MockAuthProvider) IsAvailable() bool {
	args := m.Called()
	return args.Bool(0)
}

func (m *MockAuthProvider) GetUserInfo(ctx context.Context, accessToken string) (*pkgmodels.User, error) {
	args := m.Called(ctx, accessToken)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*pkgmodels.User), args.Error(1)
}

// Helper function to create a mock provider manager with custom providers
func createMockProviderManager(primaryType, fallbackType ProviderType, enableFallback bool) (*ProviderManager, *MockAuthProvider, *MockAuthProvider) {
	primary := &MockAuthProvider{providerType: primaryType}
	fallback := &MockAuthProvider{providerType: fallbackType}

	pm := &ProviderManager{
		providers: map[ProviderType]AuthProvider{
			primaryType:  primary,
			fallbackType: fallback,
		},
		primaryType:    primaryType,
		fallbackType:   fallbackType,
		enableFallback: enableFallback,
	}

	return pm, primary, fallback
}

func TestNewProviderManager_ShouldSetGRPCPrimary_WhenModeIsGRPC(t *testing.T) {
	// Arrange
	cfg := &config.AuthConfig{
		Mode:           "grpc",
		EnableFallback: false,
	}
	authService := &Service{}

	// Act
	pm, err := NewProviderManager(cfg, authService)

	// Assert
	require.NoError(t, err)
	require.NotNil(t, pm)
	assert.Equal(t, ProviderTypeGRPC, pm.primaryType)
	assert.Equal(t, ProviderTypeBuiltin, pm.fallbackType)
	assert.False(t, pm.enableFallback)
}

func TestNewProviderManager_ShouldSetGatewayPrimary_WhenModeIsGateway(t *testing.T) {
	// Arrange
	cfg := &config.AuthConfig{
		Mode:           "gateway",
		EnableFallback: false,
	}
	authService := &Service{}

	// Act
	pm, err := NewProviderManager(cfg, authService)

	// Assert
	require.NoError(t, err)
	require.NotNil(t, pm)
	assert.Equal(t, ProviderTypeGateway, pm.primaryType)
	assert.Equal(t, ProviderTypeBuiltin, pm.fallbackType)
	assert.False(t, pm.enableFallback)
}

func TestNewProviderManager_ShouldSetGatewayPrimary_WhenModeIsOIDC(t *testing.T) {
	// Arrange
	cfg := &config.AuthConfig{
		Mode:           "oidc",
		EnableFallback: false,
	}
	authService := &Service{}

	// Act
	pm, err := NewProviderManager(cfg, authService)

	// Assert
	require.NoError(t, err)
	require.NotNil(t, pm)
	assert.Equal(t, ProviderTypeGateway, pm.primaryType)
	assert.Equal(t, ProviderTypeBuiltin, pm.fallbackType)
}

func TestNewProviderManager_ShouldSetBuiltinPrimary_WhenModeIsBuiltin(t *testing.T) {
	// Arrange
	cfg := &config.AuthConfig{
		Mode:           "builtin",
		EnableFallback: false,
	}
	authService := &Service{}

	// Act
	pm, err := NewProviderManager(cfg, authService)

	// Assert
	require.NoError(t, err)
	require.NotNil(t, pm)
	assert.Equal(t, ProviderTypeBuiltin, pm.primaryType)
	assert.Equal(t, ProviderTypeGateway, pm.fallbackType)
}

func TestNewProviderManager_ShouldSetBuiltinPrimary_WhenModeIsLocal(t *testing.T) {
	// Arrange
	cfg := &config.AuthConfig{
		Mode:           "local",
		EnableFallback: false,
	}
	authService := &Service{}

	// Act
	pm, err := NewProviderManager(cfg, authService)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, ProviderTypeBuiltin, pm.primaryType)
}

func TestNewProviderManager_ShouldSetBuiltinPrimary_WhenModeIsEmpty(t *testing.T) {
	// Arrange
	cfg := &config.AuthConfig{
		Mode:           "",
		EnableFallback: false,
	}
	authService := &Service{}

	// Act
	pm, err := NewProviderManager(cfg, authService)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, ProviderTypeBuiltin, pm.primaryType)
}

func TestNewProviderManager_ShouldEnableFallback_WhenModeIsHybrid(t *testing.T) {
	// Arrange
	cfg := &config.AuthConfig{
		Mode:           "hybrid",
		EnableFallback: false,
	}
	authService := &Service{}

	// Act
	pm, err := NewProviderManager(cfg, authService)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, ProviderTypeGateway, pm.primaryType)
	assert.Equal(t, ProviderTypeBuiltin, pm.fallbackType)
	assert.True(t, pm.enableFallback)
}

func TestNewProviderManager_ShouldAlwaysInitializeBuiltinProvider(t *testing.T) {
	// Arrange
	cfg := &config.AuthConfig{
		Mode:           "builtin",
		EnableFallback: false,
	}
	authService := &Service{}

	// Act
	pm, err := NewProviderManager(cfg, authService)

	// Assert
	require.NoError(t, err)
	provider, err := pm.GetProvider(ProviderTypeBuiltin)
	require.NoError(t, err)
	assert.NotNil(t, provider)
	assert.True(t, provider.IsAvailable())
}

func TestProviderManager_GetProvider_ShouldReturnProvider_WhenProviderAvailable(t *testing.T) {
	// Arrange
	pm, primary, _ := createMockProviderManager(ProviderTypeGRPC, ProviderTypeBuiltin, false)
	primary.On("IsAvailable").Return(true)

	// Act
	provider, err := pm.GetProvider(ProviderTypeGRPC)

	// Assert
	require.NoError(t, err)
	assert.NotNil(t, provider)
	assert.Equal(t, ProviderTypeGRPC, provider.GetType())
}

func TestProviderManager_GetProvider_ShouldReturnError_WhenProviderNotFound(t *testing.T) {
	// Arrange
	pm, _, _ := createMockProviderManager(ProviderTypeGRPC, ProviderTypeBuiltin, false)

	// Act
	provider, err := pm.GetProvider(ProviderTypeOIDC)

	// Assert
	assert.Nil(t, provider)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrProviderNotFound)
}

func TestProviderManager_GetProvider_ShouldReturnError_WhenProviderNotAvailable(t *testing.T) {
	// Arrange
	pm, primary, _ := createMockProviderManager(ProviderTypeGRPC, ProviderTypeBuiltin, false)
	primary.On("IsAvailable").Return(false)

	// Act
	provider, err := pm.GetProvider(ProviderTypeGRPC)

	// Assert
	assert.Nil(t, provider)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrProviderNotFound)
}

func TestProviderManager_GetPrimaryProvider_ShouldReturnPrimaryProvider(t *testing.T) {
	// Arrange
	pm, primary, _ := createMockProviderManager(ProviderTypeGRPC, ProviderTypeBuiltin, false)
	primary.On("IsAvailable").Return(true)

	// Act
	provider, err := pm.GetPrimaryProvider()

	// Assert
	require.NoError(t, err)
	assert.NotNil(t, provider)
	assert.Equal(t, ProviderTypeGRPC, provider.GetType())
}

func TestProviderManager_GetFallbackProvider_ShouldReturnFallbackProvider_WhenFallbackEnabled(t *testing.T) {
	// Arrange
	pm, _, fallback := createMockProviderManager(ProviderTypeGRPC, ProviderTypeBuiltin, true)
	fallback.On("IsAvailable").Return(true)

	// Act
	provider, err := pm.GetFallbackProvider()

	// Assert
	require.NoError(t, err)
	assert.NotNil(t, provider)
	assert.Equal(t, ProviderTypeBuiltin, provider.GetType())
}

func TestProviderManager_GetFallbackProvider_ShouldReturnError_WhenFallbackDisabled(t *testing.T) {
	// Arrange
	pm, _, _ := createMockProviderManager(ProviderTypeGRPC, ProviderTypeBuiltin, false)

	// Act
	provider, err := pm.GetFallbackProvider()

	// Assert
	assert.Nil(t, provider)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrProviderNotFound)
}

func TestProviderManager_Authenticate_ShouldUsePrimaryProvider_WhenPrimarySucceeds(t *testing.T) {
	// Arrange
	pm, primary, fallback := createMockProviderManager(ProviderTypeGRPC, ProviderTypeBuiltin, true)
	ctx := context.Background()
	creds := &Credentials{
		Email:    "test@example.com",
		Password: "password123",
	}

	expectedResult := &ProviderAuthResult{
		User: &pkgmodels.User{
			ID:    "user123",
			Email: "test@example.com",
		},
		AccessToken:  "access-token",
		RefreshToken: "refresh-token",
		ExpiresIn:    3600,
	}

	primary.On("IsAvailable").Return(true)
	primary.On("Authenticate", ctx, creds).Return(expectedResult, nil)

	// Act
	result, err := pm.Authenticate(ctx, creds)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, expectedResult, result)
	primary.AssertExpectations(t)
	fallback.AssertNotCalled(t, "Authenticate", mock.Anything, mock.Anything)
}

func TestProviderManager_Authenticate_ShouldUseFallback_WhenPrimaryFailsAndFallbackEnabled(t *testing.T) {
	// Arrange
	pm, primary, fallback := createMockProviderManager(ProviderTypeGRPC, ProviderTypeBuiltin, true)
	ctx := context.Background()
	creds := &Credentials{
		Email:    "test@example.com",
		Password: "password123",
	}

	primaryError := errors.New("connection failed")
	fallbackResult := &ProviderAuthResult{
		User: &pkgmodels.User{
			ID:    "user123",
			Email: "test@example.com",
		},
		AccessToken:  "fallback-token",
		RefreshToken: "fallback-refresh",
		ExpiresIn:    3600,
	}

	primary.On("IsAvailable").Return(true)
	primary.On("Authenticate", ctx, creds).Return(nil, primaryError)
	fallback.On("IsAvailable").Return(true)
	fallback.On("Authenticate", ctx, creds).Return(fallbackResult, nil)

	// Act
	result, err := pm.Authenticate(ctx, creds)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, fallbackResult, result)
	primary.AssertExpectations(t)
	fallback.AssertExpectations(t)
}

func TestProviderManager_Authenticate_ShouldReturnError_WhenPrimaryFailsAndFallbackDisabled(t *testing.T) {
	// Arrange
	pm, primary, fallback := createMockProviderManager(ProviderTypeGRPC, ProviderTypeBuiltin, false)
	ctx := context.Background()
	creds := &Credentials{
		Email:    "test@example.com",
		Password: "password123",
	}

	primaryError := errors.New("authentication failed")

	primary.On("IsAvailable").Return(true)
	primary.On("Authenticate", ctx, creds).Return(nil, primaryError)

	// Act
	result, err := pm.Authenticate(ctx, creds)

	// Assert
	assert.Nil(t, result)
	require.Error(t, err)
	assert.Equal(t, primaryError, err)
	fallback.AssertNotCalled(t, "Authenticate", mock.Anything, mock.Anything)
}

func TestProviderManager_Authenticate_ShouldReturnError_WhenBothProvidersFail(t *testing.T) {
	// Arrange
	pm, primary, fallback := createMockProviderManager(ProviderTypeGRPC, ProviderTypeBuiltin, true)
	ctx := context.Background()
	creds := &Credentials{
		Email:    "test@example.com",
		Password: "password123",
	}

	primaryError := errors.New("primary failed")
	fallbackError := errors.New("fallback failed")

	primary.On("IsAvailable").Return(true)
	primary.On("Authenticate", ctx, creds).Return(nil, primaryError)
	fallback.On("IsAvailable").Return(true)
	fallback.On("Authenticate", ctx, creds).Return(nil, fallbackError)

	// Act
	result, err := pm.Authenticate(ctx, creds)

	// Assert
	assert.Nil(t, result)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrAllProvidersFailed)
	assert.Contains(t, err.Error(), "fallback failed")
}

func TestProviderManager_Authenticate_ShouldReturnError_WhenNoProvidersAvailable(t *testing.T) {
	// Arrange
	pm, primary, fallback := createMockProviderManager(ProviderTypeGRPC, ProviderTypeBuiltin, false)
	ctx := context.Background()
	creds := &Credentials{
		Email:    "test@example.com",
		Password: "password123",
	}

	primary.On("IsAvailable").Return(false)
	fallback.On("IsAvailable").Return(false)

	// Act
	result, err := pm.Authenticate(ctx, creds)

	// Assert
	assert.Nil(t, result)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrNoProvidersAvailable)
}

func TestProviderManager_ValidateToken_ShouldUsePrimaryProvider_WhenPrimarySucceeds(t *testing.T) {
	// Arrange
	pm, primary, fallback := createMockProviderManager(ProviderTypeGRPC, ProviderTypeBuiltin, true)
	ctx := context.Background()
	token := "valid-token"

	expectedClaims := &JWTClaims{
		UserID:   "user123",
		Email:    "test@example.com",
		Username: "testuser",
		Roles:    []string{"user"},
	}

	primary.On("IsAvailable").Return(true)
	primary.On("ValidateToken", ctx, token).Return(expectedClaims, nil)

	// Act
	claims, err := pm.ValidateToken(ctx, token)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, expectedClaims, claims)
	primary.AssertExpectations(t)
	fallback.AssertNotCalled(t, "ValidateToken", mock.Anything, mock.Anything)
}

func TestProviderManager_ValidateToken_ShouldUseFallback_WhenPrimaryFailsWithNonExpiredError(t *testing.T) {
	// Arrange
	pm, primary, fallback := createMockProviderManager(ProviderTypeGRPC, ProviderTypeBuiltin, true)
	ctx := context.Background()
	token := "valid-token"

	primaryError := ErrInvalidToken
	fallbackClaims := &JWTClaims{
		UserID:   "user123",
		Email:    "test@example.com",
		Username: "testuser",
		Roles:    []string{"user"},
	}

	primary.On("IsAvailable").Return(true)
	primary.On("ValidateToken", ctx, token).Return(nil, primaryError)
	fallback.On("IsAvailable").Return(true)
	fallback.On("ValidateToken", ctx, token).Return(fallbackClaims, nil)

	// Act
	claims, err := pm.ValidateToken(ctx, token)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, fallbackClaims, claims)
	primary.AssertExpectations(t)
	fallback.AssertExpectations(t)
}

func TestProviderManager_ValidateToken_ShouldNotUseFallback_WhenTokenExpired(t *testing.T) {
	// Arrange
	pm, primary, fallback := createMockProviderManager(ProviderTypeGRPC, ProviderTypeBuiltin, true)
	ctx := context.Background()
	token := "expired-token"

	primary.On("IsAvailable").Return(true)
	primary.On("ValidateToken", ctx, token).Return(nil, ErrExpiredToken)

	// Act
	claims, err := pm.ValidateToken(ctx, token)

	// Assert
	assert.Nil(t, claims)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrExpiredToken)
	fallback.AssertNotCalled(t, "ValidateToken", mock.Anything, mock.Anything)
}

func TestProviderManager_ValidateToken_ShouldReturnError_WhenNoProvidersAvailable(t *testing.T) {
	// Arrange
	pm, primary, _ := createMockProviderManager(ProviderTypeGRPC, ProviderTypeBuiltin, false)
	ctx := context.Background()
	token := "some-token"

	primary.On("IsAvailable").Return(false)

	// Act
	claims, err := pm.ValidateToken(ctx, token)

	// Assert
	assert.Nil(t, claims)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrNoProvidersAvailable)
}

func TestProviderManager_RefreshToken_ShouldUsePrimaryProvider_WhenPrimarySucceeds(t *testing.T) {
	// Arrange
	pm, primary, fallback := createMockProviderManager(ProviderTypeGRPC, ProviderTypeBuiltin, true)
	ctx := context.Background()
	refreshToken := "refresh-token"

	expectedResult := &ProviderAuthResult{
		User: &pkgmodels.User{
			ID:    "user123",
			Email: "test@example.com",
		},
		AccessToken:  "new-access-token",
		RefreshToken: "new-refresh-token",
		ExpiresIn:    3600,
	}

	primary.On("IsAvailable").Return(true)
	primary.On("RefreshToken", ctx, refreshToken).Return(expectedResult, nil)

	// Act
	result, err := pm.RefreshToken(ctx, refreshToken)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, expectedResult, result)
	primary.AssertExpectations(t)
	fallback.AssertNotCalled(t, "RefreshToken", mock.Anything, mock.Anything)
}

func TestProviderManager_RefreshToken_ShouldUseFallback_WhenPrimaryFailsAndFallbackEnabled(t *testing.T) {
	// Arrange
	pm, primary, fallback := createMockProviderManager(ProviderTypeGRPC, ProviderTypeBuiltin, true)
	ctx := context.Background()
	refreshToken := "refresh-token"

	primaryError := errors.New("refresh failed")
	fallbackResult := &ProviderAuthResult{
		User: &pkgmodels.User{
			ID:    "user123",
			Email: "test@example.com",
		},
		AccessToken:  "fallback-access-token",
		RefreshToken: "fallback-refresh-token",
		ExpiresIn:    3600,
	}

	primary.On("IsAvailable").Return(true)
	primary.On("RefreshToken", ctx, refreshToken).Return(nil, primaryError)
	fallback.On("IsAvailable").Return(true)
	fallback.On("RefreshToken", ctx, refreshToken).Return(fallbackResult, nil)

	// Act
	result, err := pm.RefreshToken(ctx, refreshToken)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, fallbackResult, result)
	primary.AssertExpectations(t)
	fallback.AssertExpectations(t)
}

func TestProviderManager_GetUserInfo_ShouldUsePrimaryProvider_WhenPrimarySucceeds(t *testing.T) {
	// Arrange
	pm, primary, fallback := createMockProviderManager(ProviderTypeGRPC, ProviderTypeBuiltin, true)
	ctx := context.Background()
	accessToken := "access-token"

	expectedUser := &pkgmodels.User{
		ID:       "user123",
		Email:    "test@example.com",
		Username: "testuser",
		FullName: "Test User",
		IsActive: true,
		IsAdmin:  false,
	}

	primary.On("IsAvailable").Return(true)
	primary.On("GetUserInfo", ctx, accessToken).Return(expectedUser, nil)

	// Act
	user, err := pm.GetUserInfo(ctx, accessToken)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, expectedUser, user)
	primary.AssertExpectations(t)
	fallback.AssertNotCalled(t, "GetUserInfo", mock.Anything, mock.Anything)
}

func TestProviderManager_GetUserInfo_ShouldUseFallback_WhenPrimaryFailsAndFallbackEnabled(t *testing.T) {
	// Arrange
	pm, primary, fallback := createMockProviderManager(ProviderTypeGRPC, ProviderTypeBuiltin, true)
	ctx := context.Background()
	accessToken := "access-token"

	primaryError := errors.New("user fetch failed")
	fallbackUser := &pkgmodels.User{
		ID:       "user123",
		Email:    "test@example.com",
		Username: "testuser",
		FullName: "Test User",
		IsActive: true,
		IsAdmin:  false,
	}

	primary.On("IsAvailable").Return(true)
	primary.On("GetUserInfo", ctx, accessToken).Return(nil, primaryError)
	fallback.On("IsAvailable").Return(true)
	fallback.On("GetUserInfo", ctx, accessToken).Return(fallbackUser, nil)

	// Act
	user, err := pm.GetUserInfo(ctx, accessToken)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, fallbackUser, user)
	primary.AssertExpectations(t)
	fallback.AssertExpectations(t)
}

func TestProviderManager_GetUserInfo_ShouldReturnError_WhenNoProvidersAvailable(t *testing.T) {
	// Arrange
	pm, primary, _ := createMockProviderManager(ProviderTypeGRPC, ProviderTypeBuiltin, false)
	ctx := context.Background()
	accessToken := "access-token"

	primary.On("IsAvailable").Return(false)

	// Act
	user, err := pm.GetUserInfo(ctx, accessToken)

	// Assert
	assert.Nil(t, user)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrNoProvidersAvailable)
}

func TestProviderManager_GetAvailableProviders_ShouldReturnOnlyAvailableProviders(t *testing.T) {
	// Arrange
	pm, primary, fallback := createMockProviderManager(ProviderTypeGRPC, ProviderTypeBuiltin, true)

	primary.On("IsAvailable").Return(true)
	fallback.On("IsAvailable").Return(false)

	// Act
	types := pm.GetAvailableProviders()

	// Assert
	assert.Len(t, types, 1)
	assert.Contains(t, types, ProviderTypeGRPC)
	assert.NotContains(t, types, ProviderTypeBuiltin)
}

func TestProviderManager_GetAvailableProviders_ShouldReturnEmpty_WhenNoProvidersAvailable(t *testing.T) {
	// Arrange
	pm, primary, fallback := createMockProviderManager(ProviderTypeGRPC, ProviderTypeBuiltin, true)

	primary.On("IsAvailable").Return(false)
	fallback.On("IsAvailable").Return(false)

	// Act
	types := pm.GetAvailableProviders()

	// Assert
	assert.Len(t, types, 0)
}

func TestProviderManager_IsGatewayAvailable_ShouldReturnTrue_WhenGatewayAvailable(t *testing.T) {
	// Arrange
	gateway := &MockAuthProvider{providerType: ProviderTypeGateway}
	pm := &ProviderManager{
		providers: map[ProviderType]AuthProvider{
			ProviderTypeGateway: gateway,
		},
		primaryType:  ProviderTypeGateway,
		fallbackType: ProviderTypeBuiltin,
	}

	gateway.On("IsAvailable").Return(true)

	// Act
	available := pm.IsGatewayAvailable()

	// Assert
	assert.True(t, available)
}

func TestProviderManager_IsGatewayAvailable_ShouldReturnFalse_WhenGatewayNotAvailable(t *testing.T) {
	// Arrange
	gateway := &MockAuthProvider{providerType: ProviderTypeGateway}
	pm := &ProviderManager{
		providers: map[ProviderType]AuthProvider{
			ProviderTypeGateway: gateway,
		},
		primaryType:  ProviderTypeGateway,
		fallbackType: ProviderTypeBuiltin,
	}

	gateway.On("IsAvailable").Return(false)

	// Act
	available := pm.IsGatewayAvailable()

	// Assert
	assert.False(t, available)
}

func TestProviderManager_IsGatewayAvailable_ShouldReturnFalse_WhenGatewayNotConfigured(t *testing.T) {
	// Arrange
	pm := &ProviderManager{
		providers:    map[ProviderType]AuthProvider{},
		primaryType:  ProviderTypeBuiltin,
		fallbackType: ProviderTypeGateway,
	}

	// Act
	available := pm.IsGatewayAvailable()

	// Assert
	assert.False(t, available)
}

func TestProviderManager_GetMode_ShouldReturnHybrid_WhenFallbackEnabled(t *testing.T) {
	// Arrange
	pm := &ProviderManager{
		primaryType:    ProviderTypeGateway,
		fallbackType:   ProviderTypeBuiltin,
		enableFallback: true,
	}

	// Act
	mode := pm.GetMode()

	// Assert
	assert.Equal(t, "hybrid", mode)
}

func TestProviderManager_GetMode_ShouldReturnPrimaryType_WhenFallbackDisabled(t *testing.T) {
	// Arrange
	pm := &ProviderManager{
		primaryType:    ProviderTypeGRPC,
		fallbackType:   ProviderTypeBuiltin,
		enableFallback: false,
	}

	// Act
	mode := pm.GetMode()

	// Assert
	assert.Equal(t, string(ProviderTypeGRPC), mode)
}

func TestProviderManager_GetAuthorizationURL_ShouldReturnURL_WhenGatewayProviderAvailable(t *testing.T) {
	// Arrange
	gateway := &MockAuthProvider{providerType: ProviderTypeGateway}
	pm := &ProviderManager{
		providers: map[ProviderType]AuthProvider{
			ProviderTypeGateway: gateway,
		},
	}

	expectedURL := "https://auth.example.com/authorize?state=state123&nonce=nonce456"
	gateway.On("IsAvailable").Return(true)
	gateway.On("GetAuthorizationURL", "state123", "nonce456").Return(expectedURL)

	// Act
	url := pm.GetAuthorizationURL("state123", "nonce456")

	// Assert
	assert.Equal(t, expectedURL, url)
	gateway.AssertExpectations(t)
}

func TestProviderManager_GetAuthorizationURL_ShouldReturnEmpty_WhenGatewayNotAvailable(t *testing.T) {
	// Arrange
	gateway := &MockAuthProvider{providerType: ProviderTypeGateway}
	pm := &ProviderManager{
		providers: map[ProviderType]AuthProvider{
			ProviderTypeGateway: gateway,
		},
	}

	gateway.On("IsAvailable").Return(false)

	// Act
	url := pm.GetAuthorizationURL("state123", "nonce456")

	// Assert
	assert.Empty(t, url)
}

func TestProviderManager_HandleOAuthCallback_ShouldReturnResult_WhenGatewayAvailable(t *testing.T) {
	// Arrange
	gateway := &MockAuthProvider{providerType: ProviderTypeGateway}
	pm := &ProviderManager{
		providers: map[ProviderType]AuthProvider{
			ProviderTypeGateway: gateway,
		},
	}

	ctx := context.Background()
	expectedResult := &ProviderAuthResult{
		User: &pkgmodels.User{
			ID:    "user123",
			Email: "test@example.com",
		},
		AccessToken:  "access-token",
		RefreshToken: "refresh-token",
		ExpiresIn:    3600,
	}

	gateway.On("IsAvailable").Return(true)
	gateway.On("HandleCallback", ctx, "code123", "state456").Return(expectedResult, nil)

	// Act
	result, err := pm.HandleOAuthCallback(ctx, "code123", "state456")

	// Assert
	require.NoError(t, err)
	assert.Equal(t, expectedResult, result)
	gateway.AssertExpectations(t)
}

func TestProviderManager_HandleOAuthCallback_ShouldReturnError_WhenGatewayNotAvailable(t *testing.T) {
	// Arrange
	gateway := &MockAuthProvider{providerType: ProviderTypeGateway}
	pm := &ProviderManager{
		providers: map[ProviderType]AuthProvider{
			ProviderTypeGateway: gateway,
		},
	}

	ctx := context.Background()
	gateway.On("IsAvailable").Return(false)

	// Act
	result, err := pm.HandleOAuthCallback(ctx, "code123", "state456")

	// Assert
	assert.Nil(t, result)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrProviderNotFound)
}

func TestProviderManager_ConcurrentAccess_ShouldBeSafe(t *testing.T) {
	// Arrange
	pm, primary, _ := createMockProviderManager(ProviderTypeGRPC, ProviderTypeBuiltin, false)
	ctx := context.Background()
	token := "valid-token"

	expectedClaims := &JWTClaims{
		UserID:   "user123",
		Email:    "test@example.com",
		Username: "testuser",
		Roles:    []string{"user"},
	}

	primary.On("IsAvailable").Return(true)
	primary.On("ValidateToken", ctx, token).Return(expectedClaims, nil)

	// Act - Run multiple concurrent validation requests
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func() {
			claims, err := pm.ValidateToken(ctx, token)
			assert.NoError(t, err)
			assert.Equal(t, expectedClaims, claims)
			done <- true
		}()
	}

	// Wait for all goroutines to complete
	for i := 0; i < 10; i++ {
		<-done
	}

	// Assert
	primary.AssertExpectations(t)
}
