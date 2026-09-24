package observer

import (
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/smilemakc/mbflow/go/internal/infrastructure/logger"
)

// WebSocketHandler handles WebSocket connection requests
type WebSocketHandler struct {
	hub      *WebSocketHub
	logger   *logger.Logger
	upgrader websocket.Upgrader

	allowedOrigins map[string]struct{}
	allowAll       bool
}

// NewWebSocketHandler creates a new WebSocket handler
func NewWebSocketHandler(hub *WebSocketHub, logger *logger.Logger) *WebSocketHandler {
	h := &WebSocketHandler{
		hub:    hub,
		logger: logger,
	}
	h.upgrader = websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin:     h.checkOrigin,
	}
	return h
}

// SetAllowedOrigins configures which browser origins may open connections.
// Same-origin requests and non-browser clients (no Origin header) are always
// allowed; allowAll disables the check (development only).
func (h *WebSocketHandler) SetAllowedOrigins(origins []string, allowAll bool) {
	h.allowedOrigins = make(map[string]struct{}, len(origins))
	for _, o := range origins {
		h.allowedOrigins[strings.TrimRight(o, "/")] = struct{}{}
	}
	h.allowAll = allowAll
}

func (h *WebSocketHandler) checkOrigin(r *http.Request) bool {
	origin := r.Header.Get("Origin")
	if origin == "" || h.allowAll {
		return true
	}
	if _, ok := h.allowedOrigins[origin]; ok {
		return true
	}
	u, err := url.Parse(origin)
	if err != nil {
		return false
	}
	return strings.EqualFold(u.Host, r.Host)
}

// ServeHTTP handles WebSocket upgrade requests
// URL format: /ws/executions/{executionID} or /ws/executions (for all executions)
func (h *WebSocketHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Get execution ID from query parameter (optional)
	executionID := r.URL.Query().Get("execution_id")

	// Upgrade HTTP connection to WebSocket
	conn, err := h.upgrader.Upgrade(w, r, nil)
	if err != nil {
		if h.logger != nil {
			h.logger.Error("Failed to upgrade WebSocket connection", "error", err)
		}
		return
	}

	// Create new client
	clientID := uuid.New().String()
	client := NewWebSocketClient(clientID, conn, h.hub, executionID)

	// Register client with hub
	h.hub.Register(client)

	// Send welcome message
	welcomeMsg := map[string]any{
		"type":         "control",
		"message":      "Connected to MBFlow WebSocket",
		"client_id":    clientID,
		"execution_id": executionID,
		"timestamp":    time.Now().Format(time.RFC3339),
	}

	if data, err := json.Marshal(welcomeMsg); err == nil {
		select {
		case client.send <- data:
		default:
		}
	}

	// Start client read/write pumps
	go client.WritePump()
	go client.ReadPump()

	if h.logger != nil {
		h.logger.Info("WebSocket connection established",
			"client_id", clientID,
			"execution_id", executionID,
			"remote_addr", r.RemoteAddr,
		)
	}
}

// HandleHealthCheck returns WebSocket hub status
func (h *WebSocketHandler) HandleHealthCheck(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	status := map[string]any{
		"status":            "healthy",
		"connected_clients": h.hub.ClientCount(),
		"timestamp":         time.Now().Format(time.RFC3339),
	}

	if data, err := json.Marshal(status); err == nil {
		w.Write(data)
	}
}
