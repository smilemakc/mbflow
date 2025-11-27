package websocket

import (
	"log/slog"
	"net/http"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	// CheckOrigin allows connections from any origin.
	// In production, configure this based on your CORS policy.
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

// Handler handles WebSocket upgrade requests and manages connections
type Handler struct {
	hub    *Hub
	auth   Authenticator
	logger *slog.Logger
}

// NewHandler creates a new WebSocket handler
func NewHandler(hub *Hub, auth Authenticator, logger *slog.Logger) *Handler {
	return &Handler{
		hub:    hub,
		auth:   auth,
		logger: logger,
	}
}

// ServeHTTP handles the WebSocket upgrade request
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Authenticate the user
	userID, err := h.auth.Authenticate(r)
	if err != nil {
		h.logger.Warn("websocket authentication failed",
			"error", err,
			"remote_addr", r.RemoteAddr)
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Upgrade HTTP connection to WebSocket
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		h.logger.Error("websocket upgrade failed",
			"error", err,
			"remote_addr", r.RemoteAddr)
		return
	}

	// Create a new client
	clientID := uuid.New().String()
	client := NewClient(clientID, userID, h.hub, conn)

	h.logger.Info("websocket client connected",
		"client_id", clientID,
		"user_id", userID,
		"remote_addr", r.RemoteAddr)

	// Register client with hub
	h.hub.register <- client

	// Start client pumps in separate goroutines
	go client.writePump()
	go client.readPump()
}

// SetCheckOrigin allows customizing the origin check function
func SetCheckOrigin(f func(r *http.Request) bool) {
	upgrader.CheckOrigin = f
}

// SetBufferSizes sets the read and write buffer sizes for WebSocket connections
func SetBufferSizes(readSize, writeSize int) {
	upgrader.ReadBufferSize = readSize
	upgrader.WriteBufferSize = writeSize
}
