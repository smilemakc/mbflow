package observer

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/smilemakc/mbflow/go/internal/infrastructure/logger"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		// Allow all origins in development
		// In production, implement proper origin checking
		return true
	},
}

// WebSocketHandler handles WebSocket connection requests
type WebSocketHandler struct {
	hub    *WebSocketHub
	logger *logger.Logger
}

// NewWebSocketHandler creates a new WebSocket handler
func NewWebSocketHandler(hub *WebSocketHub, logger *logger.Logger) *WebSocketHandler {
	return &WebSocketHandler{
		hub:    hub,
		logger: logger,
	}
}

// ServeHTTP handles WebSocket upgrade requests
// URL format: /ws/executions/{executionID} or /ws/executions (for all executions)
func (h *WebSocketHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Get execution ID from query parameter (optional)
	executionID := r.URL.Query().Get("execution_id")

	// Upgrade HTTP connection to WebSocket
	conn, err := upgrader.Upgrade(w, r, nil)
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
