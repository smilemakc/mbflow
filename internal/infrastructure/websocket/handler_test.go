package websocket

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const handlerTestSecret = "handler-test-secret-key"

// Helper function to generate a valid test token for handler tests
func generateHandlerTestToken(t *testing.T, userID string) string {
	auth := NewJWTAuth(handlerTestSecret)
	token, err := auth.GenerateToken(userID, jwt.NewNumericDate(time.Now().Add(time.Hour)))
	require.NoError(t, err)
	return token
}

// mockAuthenticator is a mock implementation of Authenticator for testing
type mockAuthenticator struct {
	userID string
	err    error
}

func (m *mockAuthenticator) Authenticate(r *http.Request) (string, error) {
	return m.userID, m.err
}

func TestNewHandler(t *testing.T) {
	logger := testLogger()
	hub := NewHub(logger)
	auth := NewNoAuth()

	handler := NewHandler(hub, auth, logger)

	assert.NotNil(t, handler)
	assert.Equal(t, hub, handler.hub)
	assert.Equal(t, auth, handler.auth)
	assert.Equal(t, logger, handler.logger)
}

func TestHandler_ServeHTTP_Success(t *testing.T) {
	logger := testLogger()
	hub := NewHub(logger)
	go hub.Run()
	time.Sleep(10 * time.Millisecond)

	auth := NewNoAuth()
	handler := NewHandler(hub, auth, logger)

	server := httptest.NewServer(handler)
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	ws, resp, err := websocket.DefaultDialer.Dial(wsURL, nil)
	require.NoError(t, err)
	defer ws.Close()

	assert.Equal(t, http.StatusSwitchingProtocols, resp.StatusCode)

	// Wait for client registration
	time.Sleep(50 * time.Millisecond)
	assert.Equal(t, 1, hub.ClientCount())
}

func TestHandler_ServeHTTP_AuthenticationFailed(t *testing.T) {
	logger := testLogger()
	hub := NewHub(logger)
	go hub.Run()
	time.Sleep(10 * time.Millisecond)

	auth := &mockAuthenticator{
		userID: "",
		err:    ErrInvalidToken,
	}
	handler := NewHandler(hub, auth, logger)

	server := httptest.NewServer(handler)
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	ws, resp, err := websocket.DefaultDialer.Dial(wsURL, nil)

	assert.Error(t, err)
	assert.Nil(t, ws)
	if resp != nil {
		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	}

	// No client should be registered
	time.Sleep(50 * time.Millisecond)
	assert.Equal(t, 0, hub.ClientCount())
}

func TestHandler_ServeHTTP_WithJWTAuth(t *testing.T) {
	logger := testLogger()
	hub := NewHub(logger)
	go hub.Run()
	time.Sleep(10 * time.Millisecond)

	auth := NewJWTAuth(handlerTestSecret)
	handler := NewHandler(hub, auth, logger)

	server := httptest.NewServer(handler)
	defer server.Close()

	// Without token - should fail
	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	ws, resp, err := websocket.DefaultDialer.Dial(wsURL, nil)

	assert.Error(t, err)
	assert.Nil(t, ws)
	if resp != nil {
		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	}

	// With valid token
	validToken := generateHandlerTestToken(t, "test-user")
	wsURLWithToken := wsURL + "?token=" + validToken
	ws, resp, err = websocket.DefaultDialer.Dial(wsURLWithToken, nil)
	require.NoError(t, err)
	defer ws.Close()

	assert.Equal(t, http.StatusSwitchingProtocols, resp.StatusCode)
}

func TestHandler_ServeHTTP_MultipleConnections(t *testing.T) {
	logger := testLogger()
	hub := NewHub(logger)
	go hub.Run()
	time.Sleep(10 * time.Millisecond)

	auth := NewNoAuth()
	handler := NewHandler(hub, auth, logger)

	server := httptest.NewServer(handler)
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")

	// Create multiple connections
	var conns []*websocket.Conn
	for i := 0; i < 3; i++ {
		ws, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
		require.NoError(t, err)
		conns = append(conns, ws)
	}

	time.Sleep(100 * time.Millisecond)
	assert.Equal(t, 3, hub.ClientCount())

	// Close all connections
	for _, ws := range conns {
		ws.Close()
	}

	time.Sleep(100 * time.Millisecond)
	assert.Equal(t, 0, hub.ClientCount())
}

func TestHandler_ServeHTTP_UserIDFromAuth(t *testing.T) {
	logger := testLogger()
	hub := NewHub(logger)
	go hub.Run()
	time.Sleep(10 * time.Millisecond)

	auth := NewNoAuth()
	handler := NewHandler(hub, auth, logger)

	server := httptest.NewServer(handler)
	defer server.Close()

	// Connect with custom user_id
	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "?user_id=custom-user-123"
	ws, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	require.NoError(t, err)
	defer ws.Close()

	time.Sleep(50 * time.Millisecond)

	// Verify user is indexed
	hub.mu.RLock()
	clients, exists := hub.byUserID["custom-user-123"]
	hub.mu.RUnlock()

	assert.True(t, exists)
	assert.Len(t, clients, 1)
}

func TestHandler_ServeHTTP_WithAuthorizationHeader(t *testing.T) {
	logger := testLogger()
	hub := NewHub(logger)
	go hub.Run()
	time.Sleep(10 * time.Millisecond)

	auth := NewJWTAuth(handlerTestSecret)
	handler := NewHandler(hub, auth, logger)

	server := httptest.NewServer(handler)
	defer server.Close()

	validToken := generateHandlerTestToken(t, "header-auth-user")
	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	header := http.Header{}
	header.Set("Authorization", "Bearer "+validToken)

	ws, resp, err := websocket.DefaultDialer.Dial(wsURL, header)
	require.NoError(t, err)
	defer ws.Close()

	assert.Equal(t, http.StatusSwitchingProtocols, resp.StatusCode)
}

func TestSetCheckOrigin(t *testing.T) {
	// Save original
	originalCheckOrigin := upgrader.CheckOrigin
	defer func() {
		upgrader.CheckOrigin = originalCheckOrigin
	}()

	// Set custom check origin
	customCalled := false
	SetCheckOrigin(func(r *http.Request) bool {
		customCalled = true
		return r.Header.Get("Origin") == "https://allowed.com"
	})

	// Test with allowed origin
	req := httptest.NewRequest(http.MethodGet, "/ws", nil)
	req.Header.Set("Origin", "https://allowed.com")
	assert.True(t, upgrader.CheckOrigin(req))
	assert.True(t, customCalled)

	// Test with disallowed origin
	customCalled = false
	req = httptest.NewRequest(http.MethodGet, "/ws", nil)
	req.Header.Set("Origin", "https://disallowed.com")
	assert.False(t, upgrader.CheckOrigin(req))
	assert.True(t, customCalled)
}

func TestSetBufferSizes(t *testing.T) {
	// Save original values
	originalRead := upgrader.ReadBufferSize
	originalWrite := upgrader.WriteBufferSize
	defer func() {
		upgrader.ReadBufferSize = originalRead
		upgrader.WriteBufferSize = originalWrite
	}()

	SetBufferSizes(4096, 8192)

	assert.Equal(t, 4096, upgrader.ReadBufferSize)
	assert.Equal(t, 8192, upgrader.WriteBufferSize)
}

func TestHandler_ServeHTTP_UpgradeFails(t *testing.T) {
	logger := testLogger()
	hub := NewHub(logger)
	go hub.Run()
	time.Sleep(10 * time.Millisecond)

	auth := NewNoAuth()
	handler := NewHandler(hub, auth, logger)

	// Make a regular HTTP request (not WebSocket upgrade)
	req := httptest.NewRequest(http.MethodGet, "/ws", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	// Should fail because it's not a WebSocket upgrade request
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestHandler_ServeHTTP_ClientCommunication(t *testing.T) {
	logger := testLogger()
	hub := NewHub(logger)
	go hub.Run()
	time.Sleep(10 * time.Millisecond)

	auth := NewNoAuth()
	handler := NewHandler(hub, auth, logger)

	server := httptest.NewServer(handler)
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	ws, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	require.NoError(t, err)
	defer ws.Close()

	time.Sleep(50 * time.Millisecond)

	// Send a subscribe command
	cmd := WSCommand{
		Action:     CmdSubscribe,
		WorkflowID: "wf-test",
	}
	err = ws.WriteJSON(cmd)
	require.NoError(t, err)

	// Read response
	var resp WSResponse
	ws.SetReadDeadline(time.Now().Add(time.Second))
	err = ws.ReadJSON(&resp)
	require.NoError(t, err)

	assert.True(t, resp.Success)
	assert.Equal(t, CmdSubscribe, resp.Type)
}

func TestHandler_HandlerImplementsHTTPHandler(t *testing.T) {
	logger := testLogger()
	hub := NewHub(logger)
	auth := NewNoAuth()
	handler := NewHandler(hub, auth, logger)

	// Verify Handler implements http.Handler
	var _ http.Handler = handler
}

func TestHandler_ServeHTTP_CustomAuthenticator(t *testing.T) {
	logger := testLogger()
	hub := NewHub(logger)
	go hub.Run()
	time.Sleep(10 * time.Millisecond)

	// Custom authenticator that extracts user from header
	auth := &mockAuthenticator{
		userID: "header-user",
		err:    nil,
	}
	handler := NewHandler(hub, auth, logger)

	server := httptest.NewServer(handler)
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	ws, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	require.NoError(t, err)
	defer ws.Close()

	time.Sleep(50 * time.Millisecond)

	// Verify user is indexed with custom user ID
	hub.mu.RLock()
	_, exists := hub.byUserID["header-user"]
	hub.mu.RUnlock()
	assert.True(t, exists)
}

func TestHandler_ServeHTTP_AuthErrorTypes(t *testing.T) {
	tests := []struct {
		name   string
		err    error
		expect int
	}{
		{
			name:   "missing token",
			err:    ErrMissingToken,
			expect: http.StatusUnauthorized,
		},
		{
			name:   "invalid token",
			err:    ErrInvalidToken,
			expect: http.StatusUnauthorized,
		},
		{
			name:   "custom error",
			err:    errors.New("custom auth error"),
			expect: http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := testLogger()
			hub := NewHub(logger)
			go hub.Run()
			time.Sleep(10 * time.Millisecond)

			auth := &mockAuthenticator{
				userID: "",
				err:    tt.err,
			}
			handler := NewHandler(hub, auth, logger)

			server := httptest.NewServer(handler)
			defer server.Close()

			wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
			ws, resp, err := websocket.DefaultDialer.Dial(wsURL, nil)

			assert.Error(t, err)
			assert.Nil(t, ws)
			if resp != nil {
				assert.Equal(t, tt.expect, resp.StatusCode)
			}
		})
	}
}

func TestUpgrader_DefaultConfiguration(t *testing.T) {
	// Verify default upgrader configuration
	assert.Equal(t, 1024, upgrader.ReadBufferSize)
	assert.Equal(t, 1024, upgrader.WriteBufferSize)

	// CheckOrigin should allow all by default
	req := httptest.NewRequest(http.MethodGet, "/ws", nil)
	req.Header.Set("Origin", "http://any-origin.com")
	assert.True(t, upgrader.CheckOrigin(req))
}

func TestHandler_ServeHTTP_ConcurrentConnections(t *testing.T) {
	logger := testLogger()
	hub := NewHub(logger)
	go hub.Run()
	time.Sleep(10 * time.Millisecond)

	auth := NewNoAuth()
	handler := NewHandler(hub, auth, logger)

	server := httptest.NewServer(handler)
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")

	numConnections := 10
	conns := make(chan *websocket.Conn, numConnections)
	errs := make(chan error, numConnections)

	// Connect concurrently
	for i := 0; i < numConnections; i++ {
		go func() {
			ws, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
			if err != nil {
				errs <- err
				return
			}
			conns <- ws
		}()
	}

	// Collect connections
	var connList []*websocket.Conn
	timeout := time.After(2 * time.Second)

	for i := 0; i < numConnections; i++ {
		select {
		case ws := <-conns:
			connList = append(connList, ws)
		case err := <-errs:
			t.Errorf("connection error: %v", err)
		case <-timeout:
			t.Fatal("timeout waiting for connections")
		}
	}

	time.Sleep(100 * time.Millisecond)
	assert.Equal(t, numConnections, hub.ClientCount())

	// Close all
	for _, ws := range connList {
		ws.Close()
	}
}

func TestHandler_ServeHTTP_WebSocketProtocolSubprotocol(t *testing.T) {
	logger := testLogger()
	hub := NewHub(logger)
	go hub.Run()
	time.Sleep(10 * time.Millisecond)

	auth := NewJWTAuth(handlerTestSecret)
	handler := NewHandler(hub, auth, logger)

	server := httptest.NewServer(handler)
	defer server.Close()

	validToken := generateHandlerTestToken(t, "subprotocol-user")
	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")

	// Connect using Sec-WebSocket-Protocol for auth
	dialer := websocket.Dialer{
		Subprotocols: []string{"auth-" + validToken},
	}

	ws, _, err := dialer.Dial(wsURL, nil)
	require.NoError(t, err)
	defer ws.Close()

	time.Sleep(50 * time.Millisecond)
	assert.Equal(t, 1, hub.ClientCount())
}
