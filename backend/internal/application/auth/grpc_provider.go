package auth

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/smilemakc/mbflow/internal/config"
	pkgmodels "github.com/smilemakc/mbflow/pkg/models"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	pb "github.com/smilemakc/mbflow/api/proto/authpb"
)

var (
	ErrGRPCProviderNotConfigured = errors.New("gRPC auth provider is not configured")
	ErrAuthenticateNotSupported  = errors.New("authenticate not supported via gRPC proxy")
	ErrRefreshNotSupported       = errors.New("refresh token not supported via gRPC proxy")
	ErrCallbackNotSupported      = errors.New("OAuth callback not supported via gRPC proxy")
	ErrGRPCTokenValidationFailed = errors.New("token validation failed via gRPC")
	ErrGRPCUserFetchFailed       = errors.New("user fetch failed via gRPC")
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

	return provider, nil
}

func (p *GRPCProvider) Close() error {
	if p.conn != nil {
		return p.conn.Close()
	}
	return nil
}

func (p *GRPCProvider) GetType() ProviderType {
	return ProviderTypeGRPC
}

func (p *GRPCProvider) Authenticate(ctx context.Context, creds *Credentials) (*ProviderAuthResult, error) {
	return nil, ErrAuthenticateNotSupported
}

func (p *GRPCProvider) ValidateToken(ctx context.Context, token string) (*JWTClaims, error) {
	if !p.available {
		return nil, ErrGRPCProviderNotConfigured
	}

	ctx, cancel := context.WithTimeout(ctx, p.timeout)
	defer cancel()

	req := &pb.ValidateTokenRequest{
		AccessToken: token,
	}

	resp, err := p.client.ValidateToken(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrGRPCTokenValidationFailed, err)
	}

	if !resp.GetValid() {
		errMsg := resp.GetErrorMessage()
		if errMsg == "" {
			errMsg = "token validation failed"
		}
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
		return nil, ErrGRPCProviderNotConfigured
	}

	claims, err := p.ValidateToken(ctx, accessToken)
	if err != nil {
		return nil, fmt.Errorf("failed to validate token: %w", err)
	}

	ctx, cancel := context.WithTimeout(ctx, p.timeout)
	defer cancel()

	req := &pb.GetUserRequest{
		UserId: claims.UserID,
	}

	resp, err := p.client.GetUser(ctx, req)
	if err != nil {
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
