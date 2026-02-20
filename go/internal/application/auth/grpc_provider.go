package auth

import (
	"context"
	"errors"
	"fmt"
	"time"

	authgateway "github.com/smilemakc/auth-gateway/packages/go-sdk"
	"github.com/smilemakc/auth-gateway/packages/go-sdk/proto"
	"github.com/smilemakc/mbflow/go/internal/config"
	"github.com/smilemakc/mbflow/go/internal/infrastructure/logger"
	pkgmodels "github.com/smilemakc/mbflow/go/pkg/models"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

var (
	ErrGRPCProviderNotConfigured = errors.New("gRPC auth provider is not configured")
	ErrRefreshNotSupported       = errors.New("refresh token not supported via gRPC proxy")
	ErrCallbackNotSupported      = errors.New("OAuth callback not supported via gRPC proxy")
	ErrGRPCTokenValidationFailed = errors.New("token validation failed via gRPC")
	ErrGRPCUserFetchFailed       = errors.New("user fetch failed via gRPC")
	ErrGRPCLoginFailed           = errors.New("login failed via gRPC")
	ErrGRPCUserCreateFailed      = errors.New("user creation failed via gRPC")
)

// GRPCProvider implements AuthProvider using auth-gateway SDK
type GRPCProvider struct {
	config    *config.AuthConfig
	client    *authgateway.GRPCClient
	available bool
	timeout   time.Duration
}

// NewGRPCProvider creates a new gRPC auth provider using auth-gateway SDK
func NewGRPCProvider(cfg *config.AuthConfig) (*GRPCProvider, error) {
	timeout := cfg.GRPCTimeout
	if timeout == 0 {
		timeout = 10 * time.Second
	}

	provider := &GRPCProvider{
		config:    cfg,
		available: false,
		timeout:   timeout,
	}

	if cfg.GRPCAddress == "" {
		return provider, nil
	}

	// Build metadata map from config
	metadata := make(map[string]string)
	if cfg.GRPCApplicationID != "" {
		metadata["x-application-id"] = cfg.GRPCApplicationID
	}
	if cfg.GRPCClientName != "" {
		metadata["x-client-name"] = cfg.GRPCClientName
	}
	if cfg.GRPCClientVersion != "" {
		metadata["x-client-version"] = cfg.GRPCClientVersion
	}
	if cfg.GRPCPlatform != "" {
		metadata["x-platform"] = cfg.GRPCPlatform
	}
	if cfg.GRPCEnvironment != "" {
		metadata["x-environment"] = cfg.GRPCEnvironment
	}

	// Create SDK client
	client, err := authgateway.NewGRPCClient(authgateway.GRPCConfig{
		Address:     cfg.GRPCAddress,
		Insecure:    true, // TODO: make configurable
		DialTimeout: timeout,
		DialOptions: []grpc.DialOption{
			grpc.WithTransportCredentials(insecure.NewCredentials()),
		},
		Metadata: metadata,
	})
	if err != nil {
		return provider, fmt.Errorf("failed to create gRPC client: %w", err)
	}

	provider.client = client
	provider.available = true

	logger.Info("gRPC auth provider initialized (using SDK)",
		"address", cfg.GRPCAddress,
		"app_id", cfg.GRPCApplicationID,
		"client_name", cfg.GRPCClientName,
		"environment", cfg.GRPCEnvironment)

	return provider, nil
}

// Close closes the gRPC connection
func (p *GRPCProvider) Close() error {
	if p.client != nil {
		return p.client.Close()
	}
	return nil
}

// GetType returns the provider type
func (p *GRPCProvider) GetType() ProviderType {
	return ProviderTypeGRPC
}

// IsAvailable returns whether the provider is available
func (p *GRPCProvider) IsAvailable() bool {
	return p.available
}

// Authenticate authenticates a user with credentials
func (p *GRPCProvider) Authenticate(ctx context.Context, creds *Credentials) (*ProviderAuthResult, error) {
	if !p.available {
		logger.Error("gRPC auth provider not available", "address", p.config.GRPCAddress)
		return nil, ErrGRPCProviderNotConfigured
	}

	logger.Info("gRPC auth: attempting login", "address", p.config.GRPCAddress, "email", creds.Email)

	ctx, cancel := context.WithTimeout(ctx, p.timeout)
	defer cancel()

	resp, err := p.client.Login(ctx, &proto.LoginRequest{
		Email:    creds.Email,
		Phone:    creds.Phone,
		Password: creds.Password,
	})
	if err != nil {
		logger.Error("gRPC auth: login failed", "error", err.Error())
		return nil, fmt.Errorf("%w: %v", ErrGRPCLoginFailed, err)
	}

	protoUser := resp.GetUser()
	if protoUser == nil {
		logger.Error("gRPC auth: login succeeded but no user returned")
		return nil, fmt.Errorf("%w: user not returned", ErrGRPCLoginFailed)
	}

	logger.Info("gRPC auth: login succeeded", "user_id", protoUser.GetId(), "email", protoUser.GetEmail())

	user := protoUserToUser(protoUser)

	return &ProviderAuthResult{
		User:         user,
		AccessToken:  resp.GetAccessToken(),
		RefreshToken: resp.GetRefreshToken(),
		ExpiresIn:    int(resp.GetExpiresIn()),
	}, nil
}

// ValidateToken validates a JWT token
func (p *GRPCProvider) ValidateToken(ctx context.Context, token string) (*JWTClaims, error) {
	if !p.available {
		logger.Error("gRPC auth: ValidateToken - provider not available")
		return nil, ErrGRPCProviderNotConfigured
	}

	ctx, cancel := context.WithTimeout(ctx, p.timeout)
	defer cancel()

	resp, err := p.client.ValidateToken(ctx, token)
	if err != nil {
		logger.Error("gRPC auth: ValidateToken failed", "error", err.Error())
		return nil, fmt.Errorf("%w: %v", ErrGRPCTokenValidationFailed, err)
	}

	if !resp.GetValid() {
		errMsg := resp.GetErrorMessage()
		if errMsg == "" {
			errMsg = "token validation failed"
		}
		logger.Warn("gRPC auth: token invalid", "error", errMsg)
		return nil, fmt.Errorf("%w: %s", ErrInvalidToken, errMsg)
	}

	return &JWTClaims{
		UserID:   resp.GetUserId(),
		Email:    resp.GetEmail(),
		Username: resp.GetUsername(),
		Roles:    resp.GetRoles(),
	}, nil
}

// RefreshToken refreshes an access token (not supported via gRPC)
func (p *GRPCProvider) RefreshToken(ctx context.Context, refreshToken string) (*ProviderAuthResult, error) {
	return nil, ErrRefreshNotSupported
}

// GetAuthorizationURL returns OAuth authorization URL (not supported via gRPC)
func (p *GRPCProvider) GetAuthorizationURL(state, nonce string) string {
	return ""
}

// HandleCallback handles OAuth callback (not supported via gRPC)
func (p *GRPCProvider) HandleCallback(ctx context.Context, code, state string) (*ProviderAuthResult, error) {
	return nil, ErrCallbackNotSupported
}

// GetUserInfo retrieves user information by access token
func (p *GRPCProvider) GetUserInfo(ctx context.Context, accessToken string) (*pkgmodels.User, error) {
	if !p.available {
		logger.Error("gRPC auth: GetUserInfo - provider not available")
		return nil, ErrGRPCProviderNotConfigured
	}

	// First validate token to get user ID
	claims, err := p.ValidateToken(ctx, accessToken)
	if err != nil {
		logger.Error("gRPC auth: GetUserInfo - token validation failed", "error", err.Error())
		return nil, fmt.Errorf("failed to validate token: %w", err)
	}

	logger.Debug("gRPC auth: fetching user info", "user_id", claims.UserID)

	ctx, cancel := context.WithTimeout(ctx, p.timeout)
	defer cancel()

	protoUser, err := p.client.GetUser(ctx, claims.UserID)
	if err != nil {
		logger.Error("gRPC auth: GetUser failed", "error", err.Error())
		return nil, fmt.Errorf("%w: %v", ErrGRPCUserFetchFailed, err)
	}

	return protoUserToUser(protoUser), nil
}

// CreateUserRequest contains data for creating a new user via gRPC
type CreateUserRequest struct {
	Email       string
	Phone       string
	Username    string
	Password    string
	FullName    string
	AccountType string // "human" or "service"
}

// CreateUser creates a new user via gRPC auth-gateway
func (p *GRPCProvider) CreateUser(ctx context.Context, req *CreateUserRequest) (*ProviderAuthResult, error) {
	if !p.available {
		logger.Error("gRPC auth: CreateUser - provider not available", "address", p.config.GRPCAddress)
		return nil, ErrGRPCProviderNotConfigured
	}

	logger.Info("gRPC auth: creating user",
		"email", req.Email,
		"username", req.Username,
		"account_type", req.AccountType)

	ctx, cancel := context.WithTimeout(ctx, p.timeout)
	defer cancel()

	resp, err := p.client.CreateUser(ctx, &proto.CreateUserRequest{
		Email:       req.Email,
		Phone:       req.Phone,
		Username:    req.Username,
		Password:    req.Password,
		FullName:    req.FullName,
		AccountType: req.AccountType,
	})
	if err != nil {
		logger.Error("gRPC auth: CreateUser failed", "error", err.Error())
		return nil, fmt.Errorf("%w: %v", ErrGRPCUserCreateFailed, err)
	}

	protoUser := resp.GetUser()
	if protoUser == nil {
		logger.Error("gRPC auth: CreateUser succeeded but no user returned")
		return nil, fmt.Errorf("%w: user not returned", ErrGRPCUserCreateFailed)
	}

	logger.Info("gRPC auth: user created", "user_id", protoUser.GetId(), "email", protoUser.GetEmail())

	user := protoUserToUser(protoUser)

	return &ProviderAuthResult{
		User:         user,
		AccessToken:  resp.GetAccessToken(),
		RefreshToken: resp.GetRefreshToken(),
		ExpiresIn:    int(resp.GetExpiresIn()),
	}, nil
}

// CheckPermission checks if a user has a specific permission
func (p *GRPCProvider) CheckPermission(ctx context.Context, userID, resource, action string) (bool, error) {
	if !p.available {
		return false, ErrGRPCProviderNotConfigured
	}

	ctx, cancel := context.WithTimeout(ctx, p.timeout)
	defer cancel()

	return p.client.HasPermission(ctx, userID, resource, action)
}

// GetSDKClient returns the underlying SDK client for advanced usage
func (p *GRPCProvider) GetSDKClient() *authgateway.GRPCClient {
	return p.client
}

// protoUserToUser converts proto.User to pkgmodels.User
func protoUserToUser(protoUser *proto.User) *pkgmodels.User {
	if protoUser == nil {
		return nil
	}

	isAdmin := false
	for _, role := range protoUser.GetRoles() {
		if role == "admin" || role == "administrator" {
			isAdmin = true
			break
		}
	}

	return &pkgmodels.User{
		ID:        protoUser.GetId(),
		Email:     protoUser.GetEmail(),
		Username:  protoUser.GetUsername(),
		FullName:  protoUser.GetFullName(),
		IsActive:  protoUser.GetIsActive(),
		IsAdmin:   isAdmin,
		Roles:     protoUser.GetRoles(),
		CreatedAt: time.Unix(protoUser.GetCreatedAt(), 0),
		UpdatedAt: time.Unix(protoUser.GetUpdatedAt(), 0),
	}
}
