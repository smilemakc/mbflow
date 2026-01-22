package auth

import (
	"context"
	"errors"
	"testing"
	"time"

	pb "github.com/smilemakc/mbflow/api/proto/authpb"
	"github.com/smilemakc/mbflow/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
)

// MockAuthServiceClient is a mock implementation of pb.AuthServiceClient
type MockAuthServiceClient struct {
	mock.Mock
}

func (m *MockAuthServiceClient) Login(ctx context.Context, in *pb.LoginRequest, opts ...grpc.CallOption) (*pb.LoginResponse, error) {
	args := m.Called(ctx, in)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*pb.LoginResponse), args.Error(1)
}

func (m *MockAuthServiceClient) ValidateToken(ctx context.Context, in *pb.ValidateTokenRequest, opts ...grpc.CallOption) (*pb.ValidateTokenResponse, error) {
	args := m.Called(ctx, in)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*pb.ValidateTokenResponse), args.Error(1)
}

func (m *MockAuthServiceClient) GetUser(ctx context.Context, in *pb.GetUserRequest, opts ...grpc.CallOption) (*pb.GetUserResponse, error) {
	args := m.Called(ctx, in)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*pb.GetUserResponse), args.Error(1)
}

func (m *MockAuthServiceClient) CheckPermission(ctx context.Context, in *pb.CheckPermissionRequest, opts ...grpc.CallOption) (*pb.CheckPermissionResponse, error) {
	args := m.Called(ctx, in)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*pb.CheckPermissionResponse), args.Error(1)
}

func (m *MockAuthServiceClient) IntrospectToken(ctx context.Context, in *pb.IntrospectTokenRequest, opts ...grpc.CallOption) (*pb.IntrospectTokenResponse, error) {
	args := m.Called(ctx, in)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*pb.IntrospectTokenResponse), args.Error(1)
}

// Helper function to create a mock GRPCProvider for testing
func createMockGRPCProvider() (*GRPCProvider, *MockAuthServiceClient) {
	mockClient := new(MockAuthServiceClient)
	provider := &GRPCProvider{
		config: &config.AuthConfig{
			GRPCAddress: "localhost:9090",
			GRPCTimeout: 10 * time.Second,
		},
		client:    mockClient,
		available: true,
		timeout:   10 * time.Second,
	}
	return provider, mockClient
}

func TestNewGRPCProvider_ShouldReturnUnavailableProvider_WhenGRPCAddressEmpty(t *testing.T) {
	// Arrange
	cfg := &config.AuthConfig{
		GRPCAddress: "",
		GRPCTimeout: 0,
	}

	// Act
	provider, err := NewGRPCProvider(cfg)

	// Assert
	require.NoError(t, err)
	require.NotNil(t, provider)
	assert.False(t, provider.IsAvailable())
	assert.Equal(t, 10*time.Second, provider.timeout, "Should use default timeout")
	assert.Nil(t, provider.conn)
	assert.Nil(t, provider.client)
}

func TestNewGRPCProvider_ShouldUseDefaultTimeout_WhenTimeoutNotProvided(t *testing.T) {
	// Arrange
	cfg := &config.AuthConfig{
		GRPCAddress: "",
		GRPCTimeout: 0,
	}

	// Act
	provider, err := NewGRPCProvider(cfg)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, 10*time.Second, provider.timeout)
}

func TestNewGRPCProvider_ShouldUseCustomTimeout_WhenTimeoutProvided(t *testing.T) {
	// Arrange
	customTimeout := 5 * time.Second
	cfg := &config.AuthConfig{
		GRPCAddress: "",
		GRPCTimeout: customTimeout,
	}

	// Act
	provider, err := NewGRPCProvider(cfg)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, customTimeout, provider.timeout)
}

func TestGRPCProvider_GetType_ShouldReturnGRPCType(t *testing.T) {
	// Arrange
	provider, _ := createMockGRPCProvider()

	// Act
	providerType := provider.GetType()

	// Assert
	assert.Equal(t, ProviderTypeGRPC, providerType)
}

func TestGRPCProvider_IsAvailable_ShouldReturnTrue_WhenProviderConfigured(t *testing.T) {
	// Arrange
	provider, _ := createMockGRPCProvider()

	// Act & Assert
	assert.True(t, provider.IsAvailable())
}

func TestGRPCProvider_IsAvailable_ShouldReturnFalse_WhenProviderNotConfigured(t *testing.T) {
	// Arrange
	provider := &GRPCProvider{
		available: false,
	}

	// Act & Assert
	assert.False(t, provider.IsAvailable())
}

func TestGRPCProvider_Authenticate_ShouldReturnError_WhenProviderNotConfigured(t *testing.T) {
	// Arrange
	provider := &GRPCProvider{
		available: false,
	}
	creds := &Credentials{
		Email:    "test@example.com",
		Password: "password123",
	}

	// Act
	result, err := provider.Authenticate(context.Background(), creds)

	// Assert
	assert.Nil(t, result)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrGRPCProviderNotConfigured)
}

func TestGRPCProvider_Authenticate_ShouldReturnUser_WhenCredentialsValid(t *testing.T) {
	// Arrange
	provider, mockClient := createMockGRPCProvider()
	ctx := context.Background()
	creds := &Credentials{
		Email:    "test@example.com",
		Password: "password123",
	}

	expectedUser := &pb.User{
		Id:        "user123",
		Email:     "test@example.com",
		Username:  "testuser",
		FullName:  "Test User",
		IsActive:  true,
		Roles:     []string{"user"},
		CreatedAt: time.Now().Unix(),
		UpdatedAt: time.Now().Unix(),
	}

	mockClient.On("Login", mock.Anything, mock.MatchedBy(func(req *pb.LoginRequest) bool {
		return req.Email == creds.Email && req.Password == creds.Password
	})).Return(&pb.LoginResponse{
		User:         expectedUser,
		AccessToken:  "access-token-123",
		RefreshToken: "refresh-token-456",
		ExpiresIn:    3600,
		ErrorMessage: "",
	}, nil)

	// Act
	result, err := provider.Authenticate(ctx, creds)

	// Assert
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, "user123", result.User.ID)
	assert.Equal(t, "test@example.com", result.User.Email)
	assert.Equal(t, "testuser", result.User.Username)
	assert.Equal(t, "Test User", result.User.FullName)
	assert.True(t, result.User.IsActive)
	assert.False(t, result.User.IsAdmin)
	assert.Equal(t, []string{"user"}, result.User.Roles)
	assert.Equal(t, "access-token-123", result.AccessToken)
	assert.Equal(t, "refresh-token-456", result.RefreshToken)
	assert.Equal(t, 3600, result.ExpiresIn)
	mockClient.AssertExpectations(t)
}

func TestGRPCProvider_Authenticate_ShouldSetIsAdmin_WhenUserHasAdminRole(t *testing.T) {
	// Arrange
	provider, mockClient := createMockGRPCProvider()
	ctx := context.Background()
	creds := &Credentials{
		Email:    "admin@example.com",
		Password: "admin123",
	}

	expectedUser := &pb.User{
		Id:        "admin123",
		Email:     "admin@example.com",
		Username:  "admin",
		FullName:  "Admin User",
		IsActive:  true,
		Roles:     []string{"user", "admin"},
		CreatedAt: time.Now().Unix(),
		UpdatedAt: time.Now().Unix(),
	}

	mockClient.On("Login", mock.Anything, mock.Anything).Return(&pb.LoginResponse{
		User:         expectedUser,
		AccessToken:  "admin-token",
		RefreshToken: "admin-refresh",
		ExpiresIn:    3600,
	}, nil)

	// Act
	result, err := provider.Authenticate(ctx, creds)

	// Assert
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.True(t, result.User.IsAdmin)
}

func TestGRPCProvider_Authenticate_ShouldSetIsAdmin_WhenUserHasAdministratorRole(t *testing.T) {
	// Arrange
	provider, mockClient := createMockGRPCProvider()
	ctx := context.Background()
	creds := &Credentials{
		Email:    "admin@example.com",
		Password: "admin123",
	}

	expectedUser := &pb.User{
		Id:        "admin123",
		Email:     "admin@example.com",
		Username:  "admin",
		FullName:  "Admin User",
		IsActive:  true,
		Roles:     []string{"user", "administrator"},
		CreatedAt: time.Now().Unix(),
		UpdatedAt: time.Now().Unix(),
	}

	mockClient.On("Login", mock.Anything, mock.Anything).Return(&pb.LoginResponse{
		User:         expectedUser,
		AccessToken:  "admin-token",
		RefreshToken: "admin-refresh",
		ExpiresIn:    3600,
	}, nil)

	// Act
	result, err := provider.Authenticate(ctx, creds)

	// Assert
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.True(t, result.User.IsAdmin)
}

func TestGRPCProvider_Authenticate_ShouldReturnError_WhenGRPCCallFails(t *testing.T) {
	// Arrange
	provider, mockClient := createMockGRPCProvider()
	ctx := context.Background()
	creds := &Credentials{
		Email:    "test@example.com",
		Password: "password123",
	}

	grpcError := errors.New("connection failed")
	mockClient.On("Login", mock.Anything, mock.Anything).Return(nil, grpcError)

	// Act
	result, err := provider.Authenticate(ctx, creds)

	// Assert
	assert.Nil(t, result)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrGRPCLoginFailed)
	assert.Contains(t, err.Error(), "connection failed")
}

func TestGRPCProvider_Authenticate_ShouldReturnError_WhenResponseContainsError(t *testing.T) {
	// Arrange
	provider, mockClient := createMockGRPCProvider()
	ctx := context.Background()
	creds := &Credentials{
		Email:    "test@example.com",
		Password: "wrongpassword",
	}

	mockClient.On("Login", mock.Anything, mock.Anything).Return(&pb.LoginResponse{
		User:         nil,
		AccessToken:  "",
		RefreshToken: "",
		ExpiresIn:    0,
		ErrorMessage: "invalid email or password",
	}, nil)

	// Act
	result, err := provider.Authenticate(ctx, creds)

	// Assert
	assert.Nil(t, result)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrInvalidCredentials)
	assert.Contains(t, err.Error(), "invalid email or password")
}

func TestGRPCProvider_Authenticate_ShouldReturnError_WhenUserNotReturned(t *testing.T) {
	// Arrange
	provider, mockClient := createMockGRPCProvider()
	ctx := context.Background()
	creds := &Credentials{
		Email:    "test@example.com",
		Password: "password123",
	}

	mockClient.On("Login", mock.Anything, mock.Anything).Return(&pb.LoginResponse{
		User:         nil,
		AccessToken:  "some-token",
		RefreshToken: "some-refresh",
		ExpiresIn:    3600,
		ErrorMessage: "",
	}, nil)

	// Act
	result, err := provider.Authenticate(ctx, creds)

	// Assert
	assert.Nil(t, result)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrGRPCLoginFailed)
	assert.Contains(t, err.Error(), "user not returned")
}

func TestGRPCProvider_Authenticate_ShouldSupportPhoneLogin(t *testing.T) {
	// Arrange
	provider, mockClient := createMockGRPCProvider()
	ctx := context.Background()
	creds := &Credentials{
		Phone:    "+1234567890",
		Password: "password123",
	}

	expectedUser := &pb.User{
		Id:        "user123",
		Email:     "test@example.com",
		Username:  "testuser",
		FullName:  "Test User",
		IsActive:  true,
		Roles:     []string{"user"},
		CreatedAt: time.Now().Unix(),
		UpdatedAt: time.Now().Unix(),
	}

	mockClient.On("Login", mock.Anything, mock.MatchedBy(func(req *pb.LoginRequest) bool {
		return req.Phone == creds.Phone && req.Password == creds.Password
	})).Return(&pb.LoginResponse{
		User:         expectedUser,
		AccessToken:  "access-token-123",
		RefreshToken: "refresh-token-456",
		ExpiresIn:    3600,
	}, nil)

	// Act
	result, err := provider.Authenticate(ctx, creds)

	// Assert
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, "user123", result.User.ID)
	mockClient.AssertExpectations(t)
}

func TestGRPCProvider_ValidateToken_ShouldReturnError_WhenProviderNotConfigured(t *testing.T) {
	// Arrange
	provider := &GRPCProvider{
		available: false,
	}

	// Act
	claims, err := provider.ValidateToken(context.Background(), "some-token")

	// Assert
	assert.Nil(t, claims)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrGRPCProviderNotConfigured)
}

func TestGRPCProvider_ValidateToken_ShouldReturnClaims_WhenTokenValid(t *testing.T) {
	// Arrange
	provider, mockClient := createMockGRPCProvider()
	ctx := context.Background()
	token := "valid-token-123"

	mockClient.On("ValidateToken", mock.Anything, mock.MatchedBy(func(req *pb.ValidateTokenRequest) bool {
		return req.AccessToken == token
	})).Return(&pb.ValidateTokenResponse{
		Valid:        true,
		UserId:       "user123",
		Email:        "test@example.com",
		Username:     "testuser",
		Roles:        []string{"user", "editor"},
		ErrorMessage: "",
		ExpiresAt:    time.Now().Add(1 * time.Hour).Unix(),
		IsActive:     true,
	}, nil)

	// Act
	claims, err := provider.ValidateToken(ctx, token)

	// Assert
	require.NoError(t, err)
	require.NotNil(t, claims)
	assert.Equal(t, "user123", claims.UserID)
	assert.Equal(t, "test@example.com", claims.Email)
	assert.Equal(t, "testuser", claims.Username)
	assert.Equal(t, []string{"user", "editor"}, claims.Roles)
	mockClient.AssertExpectations(t)
}

func TestGRPCProvider_ValidateToken_ShouldReturnError_WhenGRPCCallFails(t *testing.T) {
	// Arrange
	provider, mockClient := createMockGRPCProvider()
	ctx := context.Background()
	token := "some-token"

	grpcError := errors.New("network error")
	mockClient.On("ValidateToken", mock.Anything, mock.Anything).Return(nil, grpcError)

	// Act
	claims, err := provider.ValidateToken(ctx, token)

	// Assert
	assert.Nil(t, claims)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrGRPCTokenValidationFailed)
	assert.Contains(t, err.Error(), "network error")
}

func TestGRPCProvider_ValidateToken_ShouldReturnError_WhenTokenInvalid(t *testing.T) {
	// Arrange
	provider, mockClient := createMockGRPCProvider()
	ctx := context.Background()
	token := "invalid-token"

	mockClient.On("ValidateToken", mock.Anything, mock.Anything).Return(&pb.ValidateTokenResponse{
		Valid:        false,
		UserId:       "",
		Email:        "",
		Username:     "",
		Roles:        nil,
		ErrorMessage: "token expired",
		ExpiresAt:    0,
		IsActive:     false,
	}, nil)

	// Act
	claims, err := provider.ValidateToken(ctx, token)

	// Assert
	assert.Nil(t, claims)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrInvalidToken)
	assert.Contains(t, err.Error(), "token expired")
}

func TestGRPCProvider_ValidateToken_ShouldReturnError_WhenTokenInvalidWithoutMessage(t *testing.T) {
	// Arrange
	provider, mockClient := createMockGRPCProvider()
	ctx := context.Background()
	token := "invalid-token"

	mockClient.On("ValidateToken", mock.Anything, mock.Anything).Return(&pb.ValidateTokenResponse{
		Valid:        false,
		ErrorMessage: "",
	}, nil)

	// Act
	claims, err := provider.ValidateToken(ctx, token)

	// Assert
	assert.Nil(t, claims)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrInvalidToken)
	assert.Contains(t, err.Error(), "token validation failed")
}

func TestGRPCProvider_GetUserInfo_ShouldReturnError_WhenProviderNotConfigured(t *testing.T) {
	// Arrange
	provider := &GRPCProvider{
		available: false,
	}

	// Act
	user, err := provider.GetUserInfo(context.Background(), "some-token")

	// Assert
	assert.Nil(t, user)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrGRPCProviderNotConfigured)
}

func TestGRPCProvider_GetUserInfo_ShouldReturnUser_WhenTokenValidAndUserExists(t *testing.T) {
	// Arrange
	provider, mockClient := createMockGRPCProvider()
	ctx := context.Background()
	token := "valid-token-123"

	mockClient.On("ValidateToken", mock.Anything, mock.MatchedBy(func(req *pb.ValidateTokenRequest) bool {
		return req.AccessToken == token
	})).Return(&pb.ValidateTokenResponse{
		Valid:        true,
		UserId:       "user123",
		Email:        "test@example.com",
		Username:     "testuser",
		Roles:        []string{"user"},
		ErrorMessage: "",
		ExpiresAt:    time.Now().Add(1 * time.Hour).Unix(),
		IsActive:     true,
	}, nil)

	expectedUser := &pb.User{
		Id:        "user123",
		Email:     "test@example.com",
		Username:  "testuser",
		FullName:  "Test User",
		IsActive:  true,
		Roles:     []string{"user"},
		CreatedAt: time.Now().Unix(),
		UpdatedAt: time.Now().Unix(),
	}

	mockClient.On("GetUser", mock.Anything, mock.MatchedBy(func(req *pb.GetUserRequest) bool {
		return req.UserId == "user123"
	})).Return(&pb.GetUserResponse{
		User:         expectedUser,
		ErrorMessage: "",
	}, nil)

	// Act
	user, err := provider.GetUserInfo(ctx, token)

	// Assert
	require.NoError(t, err)
	require.NotNil(t, user)
	assert.Equal(t, "user123", user.ID)
	assert.Equal(t, "test@example.com", user.Email)
	assert.Equal(t, "testuser", user.Username)
	assert.Equal(t, "Test User", user.FullName)
	assert.True(t, user.IsActive)
	assert.False(t, user.IsAdmin)
	assert.Equal(t, []string{"user"}, user.Roles)
	mockClient.AssertExpectations(t)
}

func TestGRPCProvider_GetUserInfo_ShouldSetIsAdmin_WhenUserHasAdminRole(t *testing.T) {
	// Arrange
	provider, mockClient := createMockGRPCProvider()
	ctx := context.Background()
	token := "admin-token"

	mockClient.On("ValidateToken", mock.Anything, mock.Anything).Return(&pb.ValidateTokenResponse{
		Valid:    true,
		UserId:   "admin123",
		Email:    "admin@example.com",
		Username: "admin",
		Roles:    []string{"user", "admin"},
		IsActive: true,
	}, nil)

	expectedUser := &pb.User{
		Id:        "admin123",
		Email:     "admin@example.com",
		Username:  "admin",
		FullName:  "Admin User",
		IsActive:  true,
		Roles:     []string{"user", "admin"},
		CreatedAt: time.Now().Unix(),
		UpdatedAt: time.Now().Unix(),
	}

	mockClient.On("GetUser", mock.Anything, mock.Anything).Return(&pb.GetUserResponse{
		User:         expectedUser,
		ErrorMessage: "",
	}, nil)

	// Act
	user, err := provider.GetUserInfo(ctx, token)

	// Assert
	require.NoError(t, err)
	require.NotNil(t, user)
	assert.True(t, user.IsAdmin)
}

func TestGRPCProvider_GetUserInfo_ShouldReturnError_WhenTokenValidationFails(t *testing.T) {
	// Arrange
	provider, mockClient := createMockGRPCProvider()
	ctx := context.Background()
	token := "invalid-token"

	mockClient.On("ValidateToken", mock.Anything, mock.Anything).Return(&pb.ValidateTokenResponse{
		Valid:        false,
		ErrorMessage: "token expired",
	}, nil)

	// Act
	user, err := provider.GetUserInfo(ctx, token)

	// Assert
	assert.Nil(t, user)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to validate token")
}

func TestGRPCProvider_GetUserInfo_ShouldReturnError_WhenGetUserFails(t *testing.T) {
	// Arrange
	provider, mockClient := createMockGRPCProvider()
	ctx := context.Background()
	token := "valid-token"

	mockClient.On("ValidateToken", mock.Anything, mock.Anything).Return(&pb.ValidateTokenResponse{
		Valid:    true,
		UserId:   "user123",
		Email:    "test@example.com",
		Username: "testuser",
		Roles:    []string{"user"},
		IsActive: true,
	}, nil)

	grpcError := errors.New("database error")
	mockClient.On("GetUser", mock.Anything, mock.Anything).Return(nil, grpcError)

	// Act
	user, err := provider.GetUserInfo(ctx, token)

	// Assert
	assert.Nil(t, user)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrGRPCUserFetchFailed)
	assert.Contains(t, err.Error(), "database error")
}

func TestGRPCProvider_GetUserInfo_ShouldReturnError_WhenResponseContainsError(t *testing.T) {
	// Arrange
	provider, mockClient := createMockGRPCProvider()
	ctx := context.Background()
	token := "valid-token"

	mockClient.On("ValidateToken", mock.Anything, mock.Anything).Return(&pb.ValidateTokenResponse{
		Valid:    true,
		UserId:   "user123",
		Email:    "test@example.com",
		Username: "testuser",
		Roles:    []string{"user"},
		IsActive: true,
	}, nil)

	mockClient.On("GetUser", mock.Anything, mock.Anything).Return(&pb.GetUserResponse{
		User:         nil,
		ErrorMessage: "user not found in database",
	}, nil)

	// Act
	user, err := provider.GetUserInfo(ctx, token)

	// Assert
	assert.Nil(t, user)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrGRPCUserFetchFailed)
	assert.Contains(t, err.Error(), "user not found in database")
}

func TestGRPCProvider_GetUserInfo_ShouldReturnError_WhenUserNotInResponse(t *testing.T) {
	// Arrange
	provider, mockClient := createMockGRPCProvider()
	ctx := context.Background()
	token := "valid-token"

	mockClient.On("ValidateToken", mock.Anything, mock.Anything).Return(&pb.ValidateTokenResponse{
		Valid:    true,
		UserId:   "user123",
		Email:    "test@example.com",
		Username: "testuser",
		Roles:    []string{"user"},
		IsActive: true,
	}, nil)

	mockClient.On("GetUser", mock.Anything, mock.Anything).Return(&pb.GetUserResponse{
		User:         nil,
		ErrorMessage: "",
	}, nil)

	// Act
	user, err := provider.GetUserInfo(ctx, token)

	// Assert
	assert.Nil(t, user)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrGRPCUserFetchFailed)
	assert.Contains(t, err.Error(), "user not found")
}

func TestGRPCProvider_RefreshToken_ShouldReturnNotSupported(t *testing.T) {
	// Arrange
	provider, _ := createMockGRPCProvider()

	// Act
	result, err := provider.RefreshToken(context.Background(), "refresh-token")

	// Assert
	assert.Nil(t, result)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrRefreshNotSupported)
}

func TestGRPCProvider_GetAuthorizationURL_ShouldReturnEmptyString(t *testing.T) {
	// Arrange
	provider, _ := createMockGRPCProvider()

	// Act
	url := provider.GetAuthorizationURL("state123", "nonce456")

	// Assert
	assert.Empty(t, url)
}

func TestGRPCProvider_HandleCallback_ShouldReturnNotSupported(t *testing.T) {
	// Arrange
	provider, _ := createMockGRPCProvider()

	// Act
	result, err := provider.HandleCallback(context.Background(), "code123", "state456")

	// Assert
	assert.Nil(t, result)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrCallbackNotSupported)
}

func TestGRPCProvider_Close_ShouldReturnNil_WhenNoConnection(t *testing.T) {
	// Arrange
	provider := &GRPCProvider{
		conn: nil,
	}

	// Act
	err := provider.Close()

	// Assert
	assert.NoError(t, err)
}
