package auth

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/smilemakc/mbflow/internal/config"
	"github.com/smilemakc/mbflow/internal/infrastructure/logger"
	pkgmodels "github.com/smilemakc/mbflow/pkg/models"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	pb "github.com/smilemakc/mbflow/api/proto/authpb"
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

type GRPCProvider struct {
	config    *config.AuthConfig
	client    pb.AuthServiceClient
	conn      *grpc.ClientConn
	available bool
	timeout   time.Duration
}

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

	conn, err := grpc.NewClient(
		cfg.GRPCAddress,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return provider, fmt.Errorf("failed to create gRPC connection: %w", err)
	}

	provider.conn = conn
	provider.client = pb.NewAuthServiceClient(conn)
	provider.available = true

	if cfg.GRPCApplicationID != "" {
		logger.Info("gRPC auth provider: application ID configured", "app_id", cfg.GRPCApplicationID)
	}

	return provider, nil
}

func (p *GRPCProvider) Close() error {
	if p.conn != nil {
		return p.conn.Close()
	}
	return nil
}

// withApplicationID adds x-application-id metadata to the context if configured
func (p *GRPCProvider) withApplicationID(ctx context.Context) context.Context {
	if p.config.GRPCApplicationID != "" {
		md := metadata.Pairs("x-application-id", p.config.GRPCApplicationID)
		ctx = metadata.NewOutgoingContext(ctx, md)
		logger.Debug("gRPC auth: added application ID to context", "app_id", p.config.GRPCApplicationID)
	}
	return ctx
}

func (p *GRPCProvider) GetType() ProviderType {
	return ProviderTypeGRPC
}

func (p *GRPCProvider) Authenticate(ctx context.Context, creds *Credentials) (*ProviderAuthResult, error) {
	if !p.available {
		logger.Error("gRPC auth provider not available", "address", p.config.GRPCAddress)
		return nil, ErrGRPCProviderNotConfigured
	}

	logger.Info("gRPC auth: attempting login", "address", p.config.GRPCAddress, "email", creds.Email)

	ctx, cancel := context.WithTimeout(ctx, p.timeout)
	defer cancel()
	ctx = p.withApplicationID(ctx)

	req := &pb.LoginRequest{
		Email:    creds.Email,
		Phone:    creds.Phone,
		Password: creds.Password,
	}

	resp, err := p.client.Login(ctx, req)
	if err != nil {
		grpcStatus, ok := status.FromError(err)
		if ok {
			logger.Error("gRPC auth: login failed",
				"code", grpcStatus.Code().String(),
				"message", grpcStatus.Message(),
				"details", fmt.Sprintf("%v", grpcStatus.Details()))
		} else {
			logger.Error("gRPC auth: login failed", "error", err.Error())
		}
		return nil, fmt.Errorf("%w: %v", ErrGRPCLoginFailed, err)
	}

	if resp.GetErrorMessage() != "" {
		logger.Error("gRPC auth: login returned error", "error", resp.GetErrorMessage())
		return nil, fmt.Errorf("%w: %s", ErrInvalidCredentials, resp.GetErrorMessage())
	}

	protoUser := resp.GetUser()
	if protoUser == nil {
		logger.Error("gRPC auth: login succeeded but no user returned")
		return nil, fmt.Errorf("%w: user not returned", ErrGRPCLoginFailed)
	}

	logger.Info("gRPC auth: login succeeded", "user_id", protoUser.GetId(), "email", protoUser.GetEmail())

	isAdmin := false
	for _, role := range protoUser.GetRoles() {
		if role == "admin" || role == "administrator" {
			isAdmin = true
			break
		}
	}

	user := &pkgmodels.User{
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

	return &ProviderAuthResult{
		User:         user,
		AccessToken:  resp.GetAccessToken(),
		RefreshToken: resp.GetRefreshToken(),
		ExpiresIn:    int(resp.GetExpiresIn()),
	}, nil
}

func (p *GRPCProvider) ValidateToken(ctx context.Context, token string) (*JWTClaims, error) {
	if !p.available {
		logger.Error("gRPC auth: ValidateToken - provider not available")
		return nil, ErrGRPCProviderNotConfigured
	}

	ctx, cancel := context.WithTimeout(ctx, p.timeout)
	defer cancel()
	ctx = p.withApplicationID(ctx)

	req := &pb.ValidateTokenRequest{
		AccessToken: token,
	}

	resp, err := p.client.ValidateToken(ctx, req)
	if err != nil {
		grpcStatus, ok := status.FromError(err)
		if ok {
			logger.Error("gRPC auth: ValidateToken failed",
				"code", grpcStatus.Code().String(),
				"message", grpcStatus.Message())
		} else {
			logger.Error("gRPC auth: ValidateToken failed", "error", err.Error())
		}
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

func (p *GRPCProvider) RefreshToken(ctx context.Context, refreshToken string) (*ProviderAuthResult, error) {
	return nil, ErrRefreshNotSupported
}

func (p *GRPCProvider) GetAuthorizationURL(state, nonce string) string {
	return ""
}

func (p *GRPCProvider) HandleCallback(ctx context.Context, code, state string) (*ProviderAuthResult, error) {
	return nil, ErrCallbackNotSupported
}

func (p *GRPCProvider) IsAvailable() bool {
	return p.available
}

func (p *GRPCProvider) GetUserInfo(ctx context.Context, accessToken string) (*pkgmodels.User, error) {
	if !p.available {
		logger.Error("gRPC auth: GetUserInfo - provider not available")
		return nil, ErrGRPCProviderNotConfigured
	}

	claims, err := p.ValidateToken(ctx, accessToken)
	if err != nil {
		logger.Error("gRPC auth: GetUserInfo - token validation failed", "error", err.Error())
		return nil, fmt.Errorf("failed to validate token: %w", err)
	}

	logger.Debug("gRPC auth: fetching user info", "user_id", claims.UserID)

	ctx, cancel := context.WithTimeout(ctx, p.timeout)
	defer cancel()
	ctx = p.withApplicationID(ctx)

	req := &pb.GetUserRequest{
		UserId: claims.UserID,
	}

	resp, err := p.client.GetUser(ctx, req)
	if err != nil {
		grpcStatus, ok := status.FromError(err)
		if ok {
			logger.Error("gRPC auth: GetUser failed",
				"code", grpcStatus.Code().String(),
				"message", grpcStatus.Message())
		} else {
			logger.Error("gRPC auth: GetUser failed", "error", err.Error())
		}
		return nil, fmt.Errorf("%w: %v", ErrGRPCUserFetchFailed, err)
	}

	if resp.GetErrorMessage() != "" {
		return nil, fmt.Errorf("%w: %s", ErrGRPCUserFetchFailed, resp.GetErrorMessage())
	}

	protoUser := resp.GetUser()
	if protoUser == nil {
		return nil, fmt.Errorf("%w: user not found", ErrGRPCUserFetchFailed)
	}

	isAdmin := false
	for _, role := range protoUser.GetRoles() {
		if role == "admin" || role == "administrator" {
			isAdmin = true
			break
		}
	}

	user := &pkgmodels.User{
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

	return user, nil
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
	ctx = p.withApplicationID(ctx)

	pbReq := &pb.CreateUserRequest{
		Email:       req.Email,
		Phone:       req.Phone,
		Username:    req.Username,
		Password:    req.Password,
		FullName:    req.FullName,
		AccountType: req.AccountType,
	}

	resp, err := p.client.CreateUser(ctx, pbReq)
	if err != nil {
		grpcStatus, ok := status.FromError(err)
		if ok {
			logger.Error("gRPC auth: CreateUser failed",
				"code", grpcStatus.Code().String(),
				"message", grpcStatus.Message(),
				"details", fmt.Sprintf("%v", grpcStatus.Details()))
		} else {
			logger.Error("gRPC auth: CreateUser failed", "error", err.Error())
		}
		return nil, fmt.Errorf("%w: %v", ErrGRPCUserCreateFailed, err)
	}

	if resp.GetErrorMessage() != "" {
		logger.Error("gRPC auth: CreateUser returned error", "error", resp.GetErrorMessage())
		return nil, fmt.Errorf("%w: %s", ErrGRPCUserCreateFailed, resp.GetErrorMessage())
	}

	protoUser := resp.GetUser()
	if protoUser == nil {
		logger.Error("gRPC auth: CreateUser succeeded but no user returned")
		return nil, fmt.Errorf("%w: user not returned", ErrGRPCUserCreateFailed)
	}

	logger.Info("gRPC auth: user created", "user_id", protoUser.GetId(), "email", protoUser.GetEmail())

	isAdmin := false
	for _, role := range protoUser.GetRoles() {
		if role == "admin" || role == "administrator" {
			isAdmin = true
			break
		}
	}

	user := &pkgmodels.User{
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

	return &ProviderAuthResult{
		User:         user,
		AccessToken:  resp.GetAccessToken(),
		RefreshToken: resp.GetRefreshToken(),
		ExpiresIn:    int(resp.GetExpiresIn()),
	}, nil
}
