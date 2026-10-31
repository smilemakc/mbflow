package auth

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"math/big"
	"net"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/smilemakc/auth-gateway/packages/go-sdk/proto"
	"github.com/smilemakc/mbflow/go/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"
)

// rcFakeGateway is an in-memory auth-gateway gRPC server.
type rcFakeGateway struct {
	proto.UnimplementedAuthServiceServer

	mu sync.Mutex

	loginResp    *proto.LoginResponse
	loginErr     error
	validateResp *proto.ValidateTokenResponse
	validateErr  error
	getUserResp  *proto.GetUserResponse
	getUserErr   error
	createResp   *proto.CreateUserResponse
	createErr    error
	permResp     *proto.CheckPermissionResponse
	permErr      error

	lastLogin  *proto.LoginRequest
	lastCreate *proto.CreateUserRequest
	lastPerm   *proto.CheckPermissionRequest
	lastGetID  string
	lastMD     metadata.MD
}

func (f *rcFakeGateway) record(ctx context.Context) {
	md, _ := metadata.FromIncomingContext(ctx)
	f.lastMD = md
}

func (f *rcFakeGateway) Login(ctx context.Context, req *proto.LoginRequest) (*proto.LoginResponse, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.record(ctx)
	f.lastLogin = req
	if f.loginErr != nil {
		return nil, f.loginErr
	}
	return f.loginResp, nil
}

func (f *rcFakeGateway) ValidateToken(ctx context.Context, req *proto.ValidateTokenRequest) (*proto.ValidateTokenResponse, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.record(ctx)
	if f.validateErr != nil {
		return nil, f.validateErr
	}
	return f.validateResp, nil
}

func (f *rcFakeGateway) GetUser(ctx context.Context, req *proto.GetUserRequest) (*proto.GetUserResponse, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.record(ctx)
	f.lastGetID = req.GetUserId()
	if f.getUserErr != nil {
		return nil, f.getUserErr
	}
	return f.getUserResp, nil
}

func (f *rcFakeGateway) CreateUser(ctx context.Context, req *proto.CreateUserRequest) (*proto.CreateUserResponse, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.record(ctx)
	f.lastCreate = req
	if f.createErr != nil {
		return nil, f.createErr
	}
	return f.createResp, nil
}

func (f *rcFakeGateway) CheckPermission(ctx context.Context, req *proto.CheckPermissionRequest) (*proto.CheckPermissionResponse, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.record(ctx)
	f.lastPerm = req
	if f.permErr != nil {
		return nil, f.permErr
	}
	return f.permResp, nil
}

// rcStartGateway starts the fake gateway on a bufconn listener, optionally with
// TLS, and returns a dial option connecting to it.
func rcStartGateway(t *testing.T, fake *rcFakeGateway, serverCreds credentials.TransportCredentials) grpc.DialOption {
	t.Helper()
	lis := bufconn.Listen(1 << 20)
	var opts []grpc.ServerOption
	if serverCreds != nil {
		opts = append(opts, grpc.Creds(serverCreds))
	}
	srv := grpc.NewServer(opts...)
	proto.RegisterAuthServiceServer(srv, fake)
	go func() { _ = srv.Serve(lis) }()
	t.Cleanup(func() {
		srv.Stop()
		_ = lis.Close()
	})
	return grpc.WithContextDialer(func(ctx context.Context, _ string) (net.Conn, error) {
		return lis.DialContext(ctx)
	})
}

func rcGRPCConfig() *config.AuthConfig {
	return &config.AuthConfig{
		GRPCAddress:       "passthrough:///bufnet",
		GRPCTimeout:       5 * time.Second,
		GRPCInsecure:      true,
		GRPCApplicationID: "app-1",
		GRPCClientName:    "mbflow-test",
		GRPCClientVersion: "1.2.3",
		GRPCPlatform:      "linux",
		GRPCEnvironment:   "test",
	}
}

func rcNewGRPCProvider(t *testing.T, fake *rcFakeGateway) *GRPCProvider {
	t.Helper()
	dialer := rcStartGateway(t, fake, nil)
	p, err := newGRPCProvider(rcGRPCConfig(), dialer)
	require.NoError(t, err)
	require.True(t, p.IsAvailable())
	t.Cleanup(func() { _ = p.Close() })
	return p
}

func rcProtoUser(roles ...string) *proto.User {
	return &proto.User{
		Id:        "user-1",
		Email:     "u@example.com",
		Username:  "user1",
		FullName:  "User One",
		IsActive:  true,
		Roles:     roles,
		CreatedAt: 1700000000,
		UpdatedAt: 1700000100,
	}
}

func TestRCGRPCProvider_UnconfiguredAddress(t *testing.T) {
	p, err := NewGRPCProvider(&config.AuthConfig{})
	require.NoError(t, err)
	assert.False(t, p.IsAvailable())
	assert.Equal(t, 10*time.Second, p.timeout)
	assert.Equal(t, ProviderTypeGRPC, p.GetType())
	assert.Nil(t, p.GetSDKClient())
	assert.NoError(t, p.Close())

	ctx := context.Background()
	_, err = p.Authenticate(ctx, &Credentials{Email: "a@b.c"})
	assert.ErrorIs(t, err, ErrGRPCProviderNotConfigured)
	_, err = p.ValidateToken(ctx, "tok")
	assert.ErrorIs(t, err, ErrGRPCProviderNotConfigured)
	_, err = p.GetUserInfo(ctx, "tok")
	assert.ErrorIs(t, err, ErrGRPCProviderNotConfigured)
	_, err = p.CreateUser(ctx, &CreateUserRequest{Email: "a@b.c"})
	assert.ErrorIs(t, err, ErrGRPCProviderNotConfigured)
	ok, err := p.CheckPermission(ctx, "u", "r", "a")
	assert.False(t, ok)
	assert.ErrorIs(t, err, ErrGRPCProviderNotConfigured)

	_, err = p.RefreshToken(ctx, "r")
	assert.ErrorIs(t, err, ErrRefreshNotSupported)
	_, err = p.HandleCallback(ctx, "c", "s")
	assert.ErrorIs(t, err, ErrCallbackNotSupported)
	assert.Empty(t, p.GetAuthorizationURL("s", "n"))
}

func TestRCGRPCProvider_Authenticate(t *testing.T) {
	fake := &rcFakeGateway{loginResp: &proto.LoginResponse{
		User:         rcProtoUser("admin"),
		AccessToken:  "access",
		RefreshToken: "refresh",
		ExpiresIn:    3600,
	}}
	p := rcNewGRPCProvider(t, fake)

	res, err := p.Authenticate(context.Background(), &Credentials{Email: "u@example.com", Phone: "+1", Password: "pw"})
	require.NoError(t, err)
	assert.Equal(t, "access", res.AccessToken)
	assert.Equal(t, "refresh", res.RefreshToken)
	assert.Equal(t, 3600, res.ExpiresIn)
	assert.Equal(t, "user-1", res.User.ID)
	assert.True(t, res.User.IsAdmin)
	assert.Equal(t, time.Unix(1700000000, 0), res.User.CreatedAt)

	fake.mu.Lock()
	defer fake.mu.Unlock()
	assert.Equal(t, "pw", fake.lastLogin.GetPassword())
	assert.Equal(t, "+1", fake.lastLogin.GetPhone())
	assert.Equal(t, []string{"app-1"}, fake.lastMD.Get("x-application-id"))
	assert.Equal(t, []string{"mbflow-test"}, fake.lastMD.Get("x-client-name"))
	assert.Equal(t, []string{"1.2.3"}, fake.lastMD.Get("x-client-version"))
	assert.Equal(t, []string{"linux"}, fake.lastMD.Get("x-platform"))
	assert.Equal(t, []string{"test"}, fake.lastMD.Get("x-environment"))
	assert.NotNil(t, p.GetSDKClient())
}

func TestRCGRPCProvider_AuthenticateErrors(t *testing.T) {
	ctx := context.Background()

	t.Run("rpc error", func(t *testing.T) {
		p := rcNewGRPCProvider(t, &rcFakeGateway{loginErr: status.Error(codes.Unavailable, "down")})
		_, err := p.Authenticate(ctx, &Credentials{Email: "x"})
		assert.ErrorIs(t, err, ErrGRPCLoginFailed)
	})
	t.Run("error message", func(t *testing.T) {
		p := rcNewGRPCProvider(t, &rcFakeGateway{loginResp: &proto.LoginResponse{ErrorMessage: "bad password"}})
		_, err := p.Authenticate(ctx, &Credentials{Email: "x"})
		assert.ErrorIs(t, err, ErrGRPCLoginFailed)
		assert.Contains(t, err.Error(), "bad password")
	})
	t.Run("no user", func(t *testing.T) {
		p := rcNewGRPCProvider(t, &rcFakeGateway{loginResp: &proto.LoginResponse{AccessToken: "a"}})
		_, err := p.Authenticate(ctx, &Credentials{Email: "x"})
		assert.ErrorIs(t, err, ErrGRPCLoginFailed)
		assert.Contains(t, err.Error(), "user not returned")
	})
}

func TestRCGRPCProvider_ValidateToken(t *testing.T) {
	ctx := context.Background()

	t.Run("valid", func(t *testing.T) {
		p := rcNewGRPCProvider(t, &rcFakeGateway{validateResp: &proto.ValidateTokenResponse{
			Valid: true, UserId: "u1", Email: "e@x", Username: "un", Roles: []string{"user"},
		}})
		claims, err := p.ValidateToken(ctx, "tok")
		require.NoError(t, err)
		assert.Equal(t, "u1", claims.UserID)
		assert.Equal(t, "e@x", claims.Email)
		assert.Equal(t, "un", claims.Username)
		assert.Equal(t, []string{"user"}, claims.Roles)
	})
	t.Run("rpc error", func(t *testing.T) {
		p := rcNewGRPCProvider(t, &rcFakeGateway{validateErr: errors.New("boom")})
		_, err := p.ValidateToken(ctx, "tok")
		assert.ErrorIs(t, err, ErrGRPCTokenValidationFailed)
	})
	t.Run("invalid without message", func(t *testing.T) {
		p := rcNewGRPCProvider(t, &rcFakeGateway{validateResp: &proto.ValidateTokenResponse{Valid: false}})
		_, err := p.ValidateToken(ctx, "tok")
		assert.ErrorIs(t, err, ErrInvalidToken)
		assert.Contains(t, err.Error(), "token validation failed")
	})
	t.Run("error message from gateway", func(t *testing.T) {
		p := rcNewGRPCProvider(t, &rcFakeGateway{validateResp: &proto.ValidateTokenResponse{ErrorMessage: "expired"}})
		_, err := p.ValidateToken(ctx, "tok")
		assert.ErrorIs(t, err, ErrGRPCTokenValidationFailed)
	})
}

func TestRCGRPCProvider_GetUserInfo(t *testing.T) {
	ctx := context.Background()
	valid := &proto.ValidateTokenResponse{Valid: true, UserId: "user-1"}

	t.Run("success", func(t *testing.T) {
		fake := &rcFakeGateway{validateResp: valid, getUserResp: &proto.GetUserResponse{User: rcProtoUser("administrator")}}
		p := rcNewGRPCProvider(t, fake)
		u, err := p.GetUserInfo(ctx, "tok")
		require.NoError(t, err)
		assert.Equal(t, "u@example.com", u.Email)
		assert.True(t, u.IsAdmin)
		fake.mu.Lock()
		assert.Equal(t, "user-1", fake.lastGetID)
		fake.mu.Unlock()
	})
	t.Run("invalid token", func(t *testing.T) {
		p := rcNewGRPCProvider(t, &rcFakeGateway{validateResp: &proto.ValidateTokenResponse{}})
		_, err := p.GetUserInfo(ctx, "tok")
		assert.ErrorIs(t, err, ErrInvalidToken)
	})
	t.Run("get user fails", func(t *testing.T) {
		p := rcNewGRPCProvider(t, &rcFakeGateway{validateResp: valid, getUserErr: status.Error(codes.NotFound, "nope")})
		_, err := p.GetUserInfo(ctx, "tok")
		assert.ErrorIs(t, err, ErrGRPCUserFetchFailed)
	})
	t.Run("user missing", func(t *testing.T) {
		p := rcNewGRPCProvider(t, &rcFakeGateway{validateResp: valid, getUserResp: &proto.GetUserResponse{}})
		u, err := p.GetUserInfo(ctx, "tok")
		require.NoError(t, err)
		assert.Nil(t, u, "a nil proto user converts to nil")
	})
}

func TestRCGRPCProvider_CreateUser(t *testing.T) {
	ctx := context.Background()
	req := &CreateUserRequest{Email: "n@x", Phone: "+2", Username: "n", Password: "pw", FullName: "N", AccountType: "service"}

	t.Run("success", func(t *testing.T) {
		fake := &rcFakeGateway{createResp: &proto.CreateUserResponse{User: rcProtoUser("user"), AccessToken: "a", RefreshToken: "r", ExpiresIn: 60}}
		p := rcNewGRPCProvider(t, fake)
		res, err := p.CreateUser(ctx, req)
		require.NoError(t, err)
		assert.Equal(t, "a", res.AccessToken)
		assert.Equal(t, 60, res.ExpiresIn)
		assert.False(t, res.User.IsAdmin)
		fake.mu.Lock()
		assert.Equal(t, "service", fake.lastCreate.GetAccountType())
		assert.Equal(t, "N", fake.lastCreate.GetFullName())
		fake.mu.Unlock()
	})
	t.Run("rpc error", func(t *testing.T) {
		p := rcNewGRPCProvider(t, &rcFakeGateway{createErr: errors.New("x")})
		_, err := p.CreateUser(ctx, req)
		assert.ErrorIs(t, err, ErrGRPCUserCreateFailed)
	})
	t.Run("no user", func(t *testing.T) {
		p := rcNewGRPCProvider(t, &rcFakeGateway{createResp: &proto.CreateUserResponse{}})
		_, err := p.CreateUser(ctx, req)
		assert.ErrorIs(t, err, ErrGRPCUserCreateFailed)
	})
}

func TestRCGRPCProvider_CheckPermission(t *testing.T) {
	fake := &rcFakeGateway{permResp: &proto.CheckPermissionResponse{Allowed: true}}
	p := rcNewGRPCProvider(t, fake)
	ok, err := p.CheckPermission(context.Background(), "u1", "workflows", "read")
	require.NoError(t, err)
	assert.True(t, ok)
	fake.mu.Lock()
	assert.Equal(t, "workflows", fake.lastPerm.GetResource())
	assert.Equal(t, "read", fake.lastPerm.GetAction())
	fake.mu.Unlock()

	denied := rcNewGRPCProvider(t, &rcFakeGateway{permResp: &proto.CheckPermissionResponse{ErrorMessage: "forbidden"}})
	ok, err = denied.CheckPermission(context.Background(), "u1", "workflows", "write")
	assert.Error(t, err)
	assert.False(t, ok)
}

func TestRCProtoUserToUser_Nil(t *testing.T) {
	assert.Nil(t, protoUserToUser(nil))
}

// ---------------------------------------------------------------------------
// Transport security
// ---------------------------------------------------------------------------

// rcSelfSignedCert creates a self-signed certificate for "localhost" and
// returns the TLS certificate and its PEM encoding.
func rcSelfSignedCert(t *testing.T) (tls.Certificate, []byte) {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	tmpl := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: "localhost"},
		DNSNames:              []string{"localhost"},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(time.Hour),
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		IsCA:                  true,
		BasicConstraintsValid: true,
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	require.NoError(t, err)
	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
	keyDER, err := x509.MarshalECPrivateKey(key)
	require.NoError(t, err)
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDER})
	cert, err := tls.X509KeyPair(certPEM, keyPEM)
	require.NoError(t, err)
	return cert, certPEM
}

func TestRCGRPCTransportCredentials(t *testing.T) {
	t.Run("insecure", func(t *testing.T) {
		creds, err := grpcTransportCredentials(&config.AuthConfig{GRPCInsecure: true})
		require.NoError(t, err)
		assert.Equal(t, "insecure", creds.Info().SecurityProtocol)
	})
	t.Run("tls with system roots", func(t *testing.T) {
		creds, err := grpcTransportCredentials(&config.AuthConfig{})
		require.NoError(t, err)
		assert.Equal(t, "tls", creds.Info().SecurityProtocol)
	})
	t.Run("missing CA file", func(t *testing.T) {
		_, err := grpcTransportCredentials(&config.AuthConfig{GRPCTLSCAFile: filepath.Join(t.TempDir(), "missing.pem")})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to read CA file")
	})
	t.Run("CA file without certificates", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "bad.pem")
		require.NoError(t, os.WriteFile(path, []byte("not a cert"), 0o600))
		_, err := grpcTransportCredentials(&config.AuthConfig{GRPCTLSCAFile: path})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "no valid certificates")
	})
}

func TestRCNewGRPCProvider_InvalidCAFileFailsConstruction(t *testing.T) {
	cfg := rcGRPCConfig()
	cfg.GRPCInsecure = false
	cfg.GRPCTLSCAFile = filepath.Join(t.TempDir(), "missing.pem")
	p, err := NewGRPCProvider(cfg)
	require.Error(t, err)
	require.NotNil(t, p)
	assert.False(t, p.IsAvailable())
}

func TestRCNewGRPCProvider_InvalidAddressFails(t *testing.T) {
	cfg := rcGRPCConfig()
	cfg.GRPCAddress = "unknownscheme-%%:///x"
	p, err := NewGRPCProvider(cfg)
	require.Error(t, err)
	assert.False(t, p.IsAvailable())
}

func TestRCGRPCProvider_TLSConnectionWithCustomCA(t *testing.T) {
	cert, certPEM := rcSelfSignedCert(t)
	fake := &rcFakeGateway{validateResp: &proto.ValidateTokenResponse{Valid: true, UserId: "tls-user"}}
	dialer := rcStartGateway(t, fake, credentials.NewTLS(&tls.Config{Certificates: []tls.Certificate{cert}, MinVersion: tls.VersionTLS12}))

	caPath := filepath.Join(t.TempDir(), "ca.pem")
	require.NoError(t, os.WriteFile(caPath, certPEM, 0o600))

	cfg := rcGRPCConfig()
	cfg.GRPCInsecure = false
	cfg.GRPCTLSCAFile = caPath
	cfg.GRPCTLSServerName = "localhost"
	cfg.GRPCTimeout = 3 * time.Second

	p, err := newGRPCProvider(cfg, dialer)
	require.NoError(t, err)
	t.Cleanup(func() { _ = p.Close() })

	claims, err := p.ValidateToken(context.Background(), "tok")
	require.NoError(t, err)
	assert.Equal(t, "tls-user", claims.UserID)
}

func TestRCGRPCProvider_TLSRejectsUntrustedServer(t *testing.T) {
	cert, _ := rcSelfSignedCert(t)
	fake := &rcFakeGateway{validateResp: &proto.ValidateTokenResponse{Valid: true, UserId: "x"}}
	dialer := rcStartGateway(t, fake, credentials.NewTLS(&tls.Config{Certificates: []tls.Certificate{cert}, MinVersion: tls.VersionTLS12}))

	// A different, unrelated CA: the server's self-signed cert must not verify.
	_, otherPEM := rcSelfSignedCert(t)
	caPath := filepath.Join(t.TempDir(), "other.pem")
	require.NoError(t, os.WriteFile(caPath, otherPEM, 0o600))

	cfg := rcGRPCConfig()
	cfg.GRPCInsecure = false
	cfg.GRPCTLSCAFile = caPath
	cfg.GRPCTLSServerName = "localhost"
	cfg.GRPCTimeout = 2 * time.Second

	p, err := newGRPCProvider(cfg, dialer)
	require.NoError(t, err)
	t.Cleanup(func() { _ = p.Close() })

	_, err = p.ValidateToken(context.Background(), "tok")
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrGRPCTokenValidationFailed)
}

func TestRCGRPCProvider_InsecureClientCannotTalkToTLSServer(t *testing.T) {
	cert, _ := rcSelfSignedCert(t)
	fake := &rcFakeGateway{validateResp: &proto.ValidateTokenResponse{Valid: true, UserId: "x"}}
	dialer := rcStartGateway(t, fake, credentials.NewTLS(&tls.Config{Certificates: []tls.Certificate{cert}, MinVersion: tls.VersionTLS12}))

	cfg := rcGRPCConfig()
	cfg.GRPCTimeout = 2 * time.Second
	p, err := newGRPCProvider(cfg, dialer)
	require.NoError(t, err)
	t.Cleanup(func() { _ = p.Close() })

	_, err = p.ValidateToken(context.Background(), "tok")
	assert.Error(t, err)
}
