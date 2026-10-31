package auth

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"math/big"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/smilemakc/mbflow/go/internal/config"
)

const rcClientID = "mbflow-client"

// rcOIDCServer is a minimal OpenID Connect provider for tests.
type rcOIDCServer struct {
	srv *httptest.Server
	key *rsa.PrivateKey

	mu sync.Mutex
	// tokenFn builds the token endpoint response.
	tokenFn func(form url.Values) (int, map[string]any)
	// userinfo maps access tokens to userinfo claims; missing → 401.
	userinfo  map[string]map[string]any
	lastForm  url.Values
	discovery bool
}

func rcNewOIDCServer(t *testing.T) *rcOIDCServer {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	s := &rcOIDCServer{key: key, userinfo: map[string]map[string]any{}, discovery: true}

	mux := http.NewServeMux()
	mux.HandleFunc("/.well-known/openid-configuration", func(w http.ResponseWriter, r *http.Request) {
		s.mu.Lock()
		ok := s.discovery
		s.mu.Unlock()
		if !ok {
			http.Error(w, "down", http.StatusInternalServerError)
			return
		}
		rcWriteJSON(w, http.StatusOK, map[string]any{
			"issuer":                                s.srv.URL,
			"authorization_endpoint":                s.srv.URL + "/authorize",
			"token_endpoint":                        s.srv.URL + "/token",
			"userinfo_endpoint":                     s.srv.URL + "/userinfo",
			"jwks_uri":                              s.srv.URL + "/jwks",
			"id_token_signing_alg_values_supported": []string{"RS256"},
		})
	})
	mux.HandleFunc("/jwks", func(w http.ResponseWriter, r *http.Request) {
		pub := s.key.PublicKey
		rcWriteJSON(w, http.StatusOK, map[string]any{"keys": []map[string]any{{
			"kty": "RSA", "alg": "RS256", "use": "sig", "kid": "k1",
			"n": base64.RawURLEncoding.EncodeToString(pub.N.Bytes()),
			"e": base64.RawURLEncoding.EncodeToString(big.NewInt(int64(pub.E)).Bytes()),
		}}})
	})
	mux.HandleFunc("/token", func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		s.mu.Lock()
		s.lastForm = r.PostForm
		fn := s.tokenFn
		s.mu.Unlock()
		if fn == nil {
			rcWriteJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid_grant"})
			return
		}
		code, body := fn(r.PostForm)
		rcWriteJSON(w, code, body)
	})
	mux.HandleFunc("/userinfo", func(w http.ResponseWriter, r *http.Request) {
		tok := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
		s.mu.Lock()
		claims, ok := s.userinfo[tok]
		s.mu.Unlock()
		if !ok {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		rcWriteJSON(w, http.StatusOK, claims)
	})
	s.srv = httptest.NewServer(mux)
	t.Cleanup(s.srv.Close)
	return s
}

func rcWriteJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

func (s *rcOIDCServer) setToken(fn func(form url.Values) (int, map[string]any)) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.tokenFn = fn
}

func (s *rcOIDCServer) setUserinfo(token string, claims map[string]any) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.userinfo[token] = claims
}

// idToken signs an ID token; extra overrides the default claims.
func (s *rcOIDCServer) idToken(t *testing.T, extra map[string]any) string {
	t.Helper()
	claims := jwt.MapClaims{
		"iss": s.srv.URL,
		"aud": rcClientID,
		"sub": "oidc-user",
		"iat": time.Now().Unix(),
		"exp": time.Now().Add(time.Hour).Unix(),
	}
	for k, v := range extra {
		claims[k] = v
	}
	tok := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	tok.Header["kid"] = "k1"
	signed, err := tok.SignedString(s.key)
	require.NoError(t, err)
	return signed
}

func (s *rcOIDCServer) cfg() *config.AuthConfig {
	return &config.AuthConfig{
		IssuerURL:    s.srv.URL,
		ClientID:     rcClientID,
		ClientSecret: "secret",
		RedirectURL:  "http://localhost/callback",
	}
}

func rcNewGateway(t *testing.T, s *rcOIDCServer) *GatewayProvider {
	t.Helper()
	p, err := NewGatewayProvider(s.cfg())
	require.NoError(t, err)
	require.True(t, p.IsAvailable())
	return p
}

func TestRCGatewayProvider_NotConfigured(t *testing.T) {
	p, err := NewGatewayProvider(&config.AuthConfig{})
	require.NoError(t, err)
	assert.False(t, p.IsAvailable())
	assert.Equal(t, ProviderTypeGateway, p.GetType())

	ctx := context.Background()
	_, err = p.Authenticate(ctx, &Credentials{})
	assert.ErrorIs(t, err, ErrProviderNotConfigured)
	_, err = p.ValidateToken(ctx, "t")
	assert.ErrorIs(t, err, ErrProviderNotConfigured)
	_, err = p.RefreshToken(ctx, "t")
	assert.ErrorIs(t, err, ErrProviderNotConfigured)
	_, err = p.HandleCallback(ctx, "c", "s")
	assert.ErrorIs(t, err, ErrProviderNotConfigured)
	_, err = p.GetUserInfo(ctx, "t")
	assert.ErrorIs(t, err, ErrProviderNotConfigured)
	assert.Empty(t, p.GetAuthorizationURL("s", "n"))
}

func TestRCGatewayProvider_DiscoveryFailure(t *testing.T) {
	s := rcNewOIDCServer(t)
	s.discovery = false
	p, err := NewGatewayProvider(s.cfg())
	assert.ErrorIs(t, err, ErrOIDCDiscoveryFailed)
	require.NotNil(t, p)
	assert.False(t, p.IsAvailable())
}

func TestRCGatewayProvider_AuthorizationURL(t *testing.T) {
	s := rcNewOIDCServer(t)
	p := rcNewGateway(t, s)
	raw := p.GetAuthorizationURL("st4te", "n0nce")
	u, err := url.Parse(raw)
	require.NoError(t, err)
	assert.Equal(t, "/authorize", u.Path)
	q := u.Query()
	assert.Equal(t, "st4te", q.Get("state"))
	assert.Equal(t, "n0nce", q.Get("nonce"))
	assert.Equal(t, rcClientID, q.Get("client_id"))
	assert.Equal(t, "openid profile email", q.Get("scope"))
	assert.Equal(t, "http://localhost/callback", q.Get("redirect_uri"))
}

func TestRCGatewayProvider_AuthenticateWithIDToken(t *testing.T) {
	s := rcNewOIDCServer(t)
	p := rcNewGateway(t, s)
	idt := s.idToken(t, map[string]any{
		"email": "jane@example.com", "name": "Jane", "roles": []string{"viewer"}, "groups": []string{"Admin"},
	})
	s.setToken(func(form url.Values) (int, map[string]any) {
		return http.StatusOK, map[string]any{
			"access_token": "at", "refresh_token": "rt", "token_type": "Bearer", "expires_in": 3600, "id_token": idt,
		}
	})

	res, err := p.Authenticate(context.Background(), &Credentials{Email: "jane@example.com", Password: "pw"})
	require.NoError(t, err)
	assert.Equal(t, "password", s.lastForm.Get("grant_type"))
	assert.Equal(t, "jane@example.com", s.lastForm.Get("username"))
	assert.Equal(t, "oidc-user", res.User.ID)
	assert.Equal(t, "jane@example.com", res.User.Username, "falls back to email without preferred_username")
	assert.Equal(t, "Jane", res.User.FullName)
	assert.Equal(t, []string{"viewer", "Admin"}, res.User.Roles)
	assert.True(t, res.User.IsAdmin, "admin group is case-insensitive")
	assert.Equal(t, "at", res.AccessToken)
	assert.Equal(t, "rt", res.RefreshToken)
	assert.Equal(t, idt, res.IDToken)
	assert.Equal(t, "Bearer", res.TokenType)
	assert.InDelta(t, 3600, res.ExpiresIn, 5)
}

func TestRCGatewayProvider_AuthenticateWithoutIDTokenUsesUserinfo(t *testing.T) {
	s := rcNewOIDCServer(t)
	p := rcNewGateway(t, s)
	s.setToken(func(form url.Values) (int, map[string]any) {
		return http.StatusOK, map[string]any{"access_token": "at2", "token_type": "Bearer", "expires_in": 60}
	})
	s.setUserinfo("at2", map[string]any{
		"sub": "ui-user", "email": "ui@example.com", "name": "UI", "preferred_username": "ui", "groups": []string{"g"},
	})

	res, err := p.Authenticate(context.Background(), &Credentials{Email: "ui@example.com", Password: "pw"})
	require.NoError(t, err)
	assert.Equal(t, "ui-user", res.User.ID)
	assert.Equal(t, "ui", res.User.Username)
	assert.Equal(t, []string{"g"}, res.User.Roles)
	assert.Empty(t, res.IDToken)

	// Username falls back to email
	s.setUserinfo("at2", map[string]any{"sub": "ui-user", "email": "ui@example.com"})
	res, err = p.Authenticate(context.Background(), &Credentials{Email: "ui@example.com", Password: "pw"})
	require.NoError(t, err)
	assert.Equal(t, "ui@example.com", res.User.Username)
}

func TestRCGatewayProvider_AuthenticateErrors(t *testing.T) {
	ctx := context.Background()

	t.Run("token endpoint rejects", func(t *testing.T) {
		s := rcNewOIDCServer(t)
		p := rcNewGateway(t, s)
		_, err := p.Authenticate(ctx, &Credentials{Email: "x", Password: "y"})
		assert.ErrorIs(t, err, ErrTokenExchangeFailed)
	})
	t.Run("userinfo fails", func(t *testing.T) {
		s := rcNewOIDCServer(t)
		p := rcNewGateway(t, s)
		s.setToken(func(url.Values) (int, map[string]any) {
			return http.StatusOK, map[string]any{"access_token": "unknown", "token_type": "Bearer"}
		})
		_, err := p.Authenticate(ctx, &Credentials{Email: "x", Password: "y"})
		assert.ErrorIs(t, err, ErrMissingUserInfo)
	})
	t.Run("userinfo claims malformed", func(t *testing.T) {
		s := rcNewOIDCServer(t)
		p := rcNewGateway(t, s)
		s.setToken(func(url.Values) (int, map[string]any) {
			return http.StatusOK, map[string]any{"access_token": "at", "token_type": "Bearer"}
		})
		s.setUserinfo("at", map[string]any{"sub": "u", "roles": "not-a-list"})
		_, err := p.Authenticate(ctx, &Credentials{Email: "x", Password: "y"})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to parse claims")
	})
	t.Run("id token with wrong audience", func(t *testing.T) {
		s := rcNewOIDCServer(t)
		p := rcNewGateway(t, s)
		bad := s.idToken(t, map[string]any{"aud": "someone-else"})
		s.setToken(func(url.Values) (int, map[string]any) {
			return http.StatusOK, map[string]any{"access_token": "at", "token_type": "Bearer", "id_token": bad}
		})
		_, err := p.Authenticate(ctx, &Credentials{Email: "x", Password: "y"})
		assert.ErrorIs(t, err, ErrInvalidIDToken)
	})
	t.Run("id token claims malformed", func(t *testing.T) {
		s := rcNewOIDCServer(t)
		p := rcNewGateway(t, s)
		bad := s.idToken(t, map[string]any{"groups": "not-a-list"})
		s.setToken(func(url.Values) (int, map[string]any) {
			return http.StatusOK, map[string]any{"access_token": "at", "token_type": "Bearer", "id_token": bad}
		})
		_, err := p.Authenticate(ctx, &Credentials{Email: "x", Password: "y"})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to parse ID token claims")
	})
}

func TestRCGatewayProvider_ValidateToken(t *testing.T) {
	ctx := context.Background()
	s := rcNewOIDCServer(t)
	p := rcNewGateway(t, s)

	idt := s.idToken(t, map[string]any{
		"email": "v@example.com", "preferred_username": "vee", "roles": []string{"r1"}, "groups": []string{"g1"},
	})
	claims, err := p.ValidateToken(ctx, idt)
	require.NoError(t, err)
	assert.Equal(t, "oidc-user", claims.UserID)
	assert.Equal(t, "vee", claims.Username)
	assert.Equal(t, []string{"r1", "g1"}, claims.Roles)

	noUsername := s.idToken(t, map[string]any{"email": "v@example.com"})
	claims, err = p.ValidateToken(ctx, noUsername)
	require.NoError(t, err)
	assert.Equal(t, "v@example.com", claims.Username)

	// Opaque access token resolved via userinfo
	s.setUserinfo("opaque", map[string]any{"sub": "opq", "email": "o@example.com"})
	claims, err = p.ValidateToken(ctx, "opaque")
	require.NoError(t, err)
	assert.Equal(t, "opq", claims.UserID)
	assert.Equal(t, "o@example.com", claims.Username)

	_, err = p.ValidateToken(ctx, "nobody-knows-this")
	assert.ErrorIs(t, err, ErrInvalidToken)

	malformed := s.idToken(t, map[string]any{"roles": 42})
	_, err = p.ValidateToken(ctx, malformed)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse claims")
}

func TestRCGatewayProvider_RefreshAndCallback(t *testing.T) {
	ctx := context.Background()
	s := rcNewOIDCServer(t)
	p := rcNewGateway(t, s)

	// Failures first (no token handler)
	_, err := p.RefreshToken(ctx, "rt")
	assert.ErrorIs(t, err, ErrTokenExchangeFailed)
	_, err = p.HandleCallback(ctx, "code", "state")
	assert.ErrorIs(t, err, ErrTokenExchangeFailed)

	idt := s.idToken(t, map[string]any{"email": "r@example.com", "preferred_username": "r"})
	s.setToken(func(form url.Values) (int, map[string]any) {
		return http.StatusOK, map[string]any{"access_token": "new-" + form.Get("grant_type"), "token_type": "Bearer", "expires_in": 60, "id_token": idt}
	})

	res, err := p.RefreshToken(ctx, "rt")
	require.NoError(t, err)
	assert.Equal(t, "new-refresh_token", res.AccessToken)
	assert.Equal(t, "rt", s.lastForm.Get("refresh_token"))
	assert.False(t, res.User.IsAdmin)

	res, err = p.HandleCallback(ctx, "the-code", "state")
	require.NoError(t, err)
	assert.Equal(t, "new-authorization_code", res.AccessToken)
	assert.Equal(t, "the-code", s.lastForm.Get("code"))
	assert.Equal(t, "r", res.User.Username)
}

func TestRCGatewayProvider_GetUserInfo(t *testing.T) {
	ctx := context.Background()
	s := rcNewOIDCServer(t)
	p := rcNewGateway(t, s)

	s.setUserinfo("at", map[string]any{"sub": "s1", "email": "e@x", "name": "N", "preferred_username": "pu", "roles": []string{"a"}, "groups": []string{"b"}})
	u, err := p.GetUserInfo(ctx, "at")
	require.NoError(t, err)
	assert.Equal(t, "s1", u.ID)
	assert.Equal(t, "pu", u.Username)
	assert.Equal(t, "N", u.FullName)
	assert.True(t, u.IsActive)
	assert.Equal(t, []string{"a", "b"}, u.Roles)

	s.setUserinfo("at", map[string]any{"sub": "s1", "email": "e@x"})
	u, err = p.GetUserInfo(ctx, "at")
	require.NoError(t, err)
	assert.Equal(t, "e@x", u.Username)

	_, err = p.GetUserInfo(ctx, "missing")
	assert.Error(t, err)

	s.setUserinfo("bad", map[string]any{"sub": "s1", "name": 5})
	_, err = p.GetUserInfo(ctx, "bad")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse user claims")
}

func TestRCGenerateStateAndNonce(t *testing.T) {
	s1, err := GenerateState()
	require.NoError(t, err)
	s2, err := GenerateState()
	require.NoError(t, err)
	assert.NotEqual(t, s1, s2)
	raw, err := base64.URLEncoding.DecodeString(s1)
	require.NoError(t, err)
	assert.Len(t, raw, 32)

	n, err := GenerateNonce()
	require.NoError(t, err)
	raw, err = base64.URLEncoding.DecodeString(n)
	require.NoError(t, err)
	assert.Len(t, raw, 32)
}

// ---------------------------------------------------------------------------
// ProviderManager construction and remaining branches
// ---------------------------------------------------------------------------

func TestRCNewProviderManager_RegistersConfiguredProviders(t *testing.T) {
	s := rcNewOIDCServer(t)
	cfg := s.cfg()
	cfg.Mode = "hybrid"
	cfg.GRPCAddress = "passthrough:///auth-gateway:50051"
	cfg.GRPCInsecure = true

	pm, err := NewProviderManager(cfg, &Service{})
	require.NoError(t, err)
	assert.ElementsMatch(t, []ProviderType{ProviderTypeBuiltin, ProviderTypeGateway, ProviderTypeGRPC}, pm.GetAvailableProviders())
	assert.True(t, pm.IsGatewayAvailable())
	assert.Equal(t, "hybrid", pm.GetMode())
	assert.True(t, pm.ShouldHandleAuth())
	assert.Contains(t, pm.GetAuthorizationURL("st", "no"), "state=st")

	grpcProvider, err := pm.GetProvider(ProviderTypeGRPC)
	require.NoError(t, err)
	t.Cleanup(func() { _ = grpcProvider.(*GRPCProvider).Close() })
}

func TestRCNewProviderManager_SkipsFailingProviders(t *testing.T) {
	s := rcNewOIDCServer(t)
	s.discovery = false
	cfg := s.cfg()
	cfg.Mode = "grpc_hybrid"
	cfg.GRPCAddress = "passthrough:///auth-gateway:50051"
	cfg.GRPCInsecure = false
	cfg.GRPCTLSCAFile = "/nonexistent/ca.pem"

	pm, err := NewProviderManager(cfg, &Service{})
	require.NoError(t, err)
	assert.Equal(t, []ProviderType{ProviderTypeBuiltin}, pm.GetAvailableProviders())
	assert.False(t, pm.IsGatewayAvailable())
	assert.Equal(t, "grpc_hybrid", pm.GetMode())
	assert.True(t, pm.ShouldHandleAuth())
	assert.Empty(t, pm.GetAuthorizationURL("s", "n"))
	_, err = pm.HandleOAuthCallback(context.Background(), "c", "s")
	assert.ErrorIs(t, err, ErrProviderNotFound)
}

func TestRCNewProviderManager_UnknownModeDefaultsToBuiltin(t *testing.T) {
	pm, err := NewProviderManager(&config.AuthConfig{Mode: "something-else"}, &Service{})
	require.NoError(t, err)
	assert.Equal(t, "builtin", pm.GetMode())
	assert.False(t, pm.ShouldHandleAuth())
	_, err = pm.GetFallbackProvider()
	assert.ErrorIs(t, err, ErrProviderNotFound)
}

func rcUnavailable(pt ProviderType) *MockAuthProvider {
	m := &MockAuthProvider{providerType: pt}
	m.On("IsAvailable").Return(false)
	return m
}

func TestRCProviderManager_RefreshTokenBranches(t *testing.T) {
	ctx := context.Background()

	t.Run("primary fails, fallback disabled", func(t *testing.T) {
		pm, primary, _ := createMockProviderManager(ProviderTypeGateway, ProviderTypeBuiltin, false)
		primary.On("IsAvailable").Return(true)
		primary.On("RefreshToken", mock.Anything, "rt").Return(nil, ErrTokenExchangeFailed)
		_, err := pm.RefreshToken(ctx, "rt")
		assert.ErrorIs(t, err, ErrTokenExchangeFailed)
	})
	t.Run("primary unavailable, fallback used", func(t *testing.T) {
		pm, _, fallback := createMockProviderManager(ProviderTypeGateway, ProviderTypeBuiltin, true)
		pm.providers[ProviderTypeGateway] = rcUnavailable(ProviderTypeGateway)
		fallback.On("IsAvailable").Return(true)
		fallback.On("RefreshToken", mock.Anything, "rt").Return(&ProviderAuthResult{AccessToken: "fb"}, nil)
		res, err := pm.RefreshToken(ctx, "rt")
		require.NoError(t, err)
		assert.Equal(t, "fb", res.AccessToken)
	})
	t.Run("nothing available", func(t *testing.T) {
		pm, _, _ := createMockProviderManager(ProviderTypeGateway, ProviderTypeBuiltin, true)
		pm.providers[ProviderTypeGateway] = rcUnavailable(ProviderTypeGateway)
		pm.providers[ProviderTypeBuiltin] = rcUnavailable(ProviderTypeBuiltin)
		_, err := pm.RefreshToken(ctx, "rt")
		assert.ErrorIs(t, err, ErrNoProvidersAvailable)
	})
}

func TestRCProviderManager_ValidateAndUserInfoBranches(t *testing.T) {
	ctx := context.Background()

	t.Run("validate: primary unavailable uses fallback", func(t *testing.T) {
		pm, _, fallback := createMockProviderManager(ProviderTypeGRPC, ProviderTypeBuiltin, true)
		pm.providers[ProviderTypeGRPC] = rcUnavailable(ProviderTypeGRPC)
		fallback.On("IsAvailable").Return(true)
		fallback.On("ValidateToken", mock.Anything, "t").Return(&JWTClaims{UserID: "fb"}, nil)
		c, err := pm.ValidateToken(ctx, "t")
		require.NoError(t, err)
		assert.Equal(t, "fb", c.UserID)
	})
	t.Run("validate: fallback unavailable returns primary error", func(t *testing.T) {
		pm, primary, _ := createMockProviderManager(ProviderTypeGRPC, ProviderTypeBuiltin, true)
		primary.On("IsAvailable").Return(true)
		primary.On("ValidateToken", mock.Anything, "t").Return(nil, ErrInvalidToken)
		pm.providers[ProviderTypeBuiltin] = rcUnavailable(ProviderTypeBuiltin)
		_, err := pm.ValidateToken(ctx, "t")
		assert.ErrorIs(t, err, ErrInvalidToken)
	})
	t.Run("userinfo: primary fails without fallback", func(t *testing.T) {
		pm, primary, _ := createMockProviderManager(ProviderTypeGRPC, ProviderTypeBuiltin, false)
		primary.On("IsAvailable").Return(true)
		primary.On("GetUserInfo", mock.Anything, "t").Return(nil, ErrUserNotFound)
		_, err := pm.GetUserInfo(ctx, "t")
		assert.ErrorIs(t, err, ErrUserNotFound)
	})
	t.Run("userinfo: primary unavailable uses fallback", func(t *testing.T) {
		pm, _, fallback := createMockProviderManager(ProviderTypeGRPC, ProviderTypeBuiltin, true)
		pm.providers[ProviderTypeGRPC] = rcUnavailable(ProviderTypeGRPC)
		fallback.On("IsAvailable").Return(true)
		fallback.On("GetUserInfo", mock.Anything, "t").Return(nil, ErrUserNotFound)
		_, err := pm.GetUserInfo(ctx, "t")
		assert.ErrorIs(t, err, ErrUserNotFound)
	})
}
