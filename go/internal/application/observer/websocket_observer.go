package observer

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/smilemakc/mbflow/go/internal/infrastructure/logger"
)

// WebSocketObserver broadcasts execution events to WebSocket clients
type WebSocketObserver struct {
	name   string
	filter EventFilter
	logger *logger.Logger
	hub    *WebSocketHub
	mu     sync.RWMutex
}

// WebSocketClient represents a connected WebSocket client
type WebSocketClient struct {
	ID            string
	conn          *websocket.Conn
	send          chan []byte
	hub           *WebSocketHub
	executionID   string // Filter events by execution ID (optional)
	subscriptions map[EventType]bool
	mu            sync.RWMutex
}

// WebSocketHub manages WebSocket connections and broadcasting
type WebSocketHub struct {
	clients    map[*WebSocketClient]bool
	broadcast  chan []byte
	register   chan *WebSocketClient
	unregister chan *WebSocketClient
	logger     *logger.Logger
	mu         sync.RWMutex
}

// WebSocketMessage represents a message sent to WebSocket clients
type WebSocketMessage struct {
	Type      string         `json:"type"` // "event" or "control"
	Event     *EventPayload  `json:"event,omitempty"`
	Control   map[string]any `json:"control,omitempty"`
	Timestamp time.Time      `json:"timestamp"`
}

// EventPayload is the WebSocket-friendly event payload
type EventPayload struct {
	EventType   string         `json:"event_type"`
	ExecutionID string         `json:"execution_id"`
	WorkflowID  string         `json:"workflow_id"`
	Timestamp   time.Time      `json:"timestamp"`
	Status      string         `json:"status"`
	NodeID      *string        `json:"node_id,omitempty"`
	NodeName    *string        `json:"node_name,omitempty"`
	NodeType    *string        `json:"node_type,omitempty"`
	WaveIndex   *int           `json:"wave_index,omitempty"`
	NodeCount   *int           `json:"node_count,omitempty"`
	DurationMs  *int64         `json:"duration_ms,omitempty"`
	Error       *string        `json:"error,omitempty"`
	Output      map[string]any `json:"output,omitempty"`
}

// WebSocketObserverOption configures WebSocketObserver
type WebSocketObserverOption func(*WebSocketObserver)

// WithWebSocketFilter sets event filter
func WithWebSocketFilter(filter EventFilter) WebSocketObserverOption {
	return func(o *WebSocketObserver) {
		o.filter = filter
	}
}

// WithWebSocketLogger sets logger instance
func WithWebSocketLogger(l *logger.Logger) WebSocketObserverOption {
	return func(o *WebSocketObserver) {
		o.logger = l
	}
}

// NewWebSocketHub creates a new WebSocket hub
func NewWebSocketHub(logger *logger.Logger) *WebSocketHub {
	hub := &WebSocketHub{
		clients:    make(map[*WebSocketClient]bool),
		broadcast:  make(chan []byte, 256),
		register:   make(chan *WebSocketClient),
		unregister: make(chan *WebSocketClient),
		logger:     logger,
	}

	// Start hub in background
	go hub.run()

	return hub
}

// NewWebSocketObserver creates a new WebSocket observer
func NewWebSocketObserver(hub *WebSocketHub, opts ...WebSocketObserverOption) *WebSocketObserver {
	obs := &WebSocketObserver{
		name:   "websocket",
		filter: nil,
		hub:    hub,
	}

	for _, opt := range opts {
		opt(obs)
	}

	return obs
}

// Name returns the observer's name
func (o *WebSocketObserver) Name() string {
	return o.name
}

// Filter returns the event filter
func (o *WebSocketObserver) Filter() EventFilter {
	return o.filter
}

// OnEvent handles event by broadcasting to WebSocket clients
func (o *WebSocketObserver) OnEvent(ctx context.Context, event Event) error {
	// Convert to WebSocket message
	message := o.eventToMessage(event)

	// Marshal to JSON
	data, err := json.Marshal(message)
	if err != nil {
		if o.logger != nil {
			o.logger.ErrorContext(ctx, "Failed to marshal WebSocket message",
				"error", err,
				"event_type", string(event.Type),
			)
		}
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	// Broadcast to all connected clients
	o.hub.BroadcastToExecution(event.ExecutionID, data)

	return nil
}

// eventToMessage converts an observer.Event to WebSocketMessage
func (o *WebSocketObserver) eventToMessage(event Event) *WebSocketMessage {
	payload := &EventPayload{
		EventType:   string(event.Type),
		ExecutionID: event.ExecutionID,
		WorkflowID:  event.WorkflowID,
		Timestamp:   event.Timestamp,
		Status:      event.Status,
		NodeID:      event.NodeID,
		NodeName:    event.NodeName,
		NodeType:    event.NodeType,
		WaveIndex:   event.WaveIndex,
		NodeCount:   event.NodeCount,
		DurationMs:  event.DurationMs,
		Output:      event.Output,
	}

	if event.Error != nil {
		errStr := event.Error.Error()
		payload.Error = &errStr
	}

	return &WebSocketMessage{
		Type:      "event",
		Event:     payload,
		Timestamp: time.Now(),
	}
}

// GetHub returns the WebSocket hub (for HTTP handler integration)
func (o *WebSocketObserver) GetHub() *WebSocketHub {
	return o.hub
}

// ============================================================================
// WebSocketHub implementation
// ============================================================================

// run starts the hub's main loop
func (h *WebSocketHub) run() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			h.clients[client] = true
			h.mu.Unlock()

			if h.logger != nil {
				h.logger.Info("WebSocket client connected",
					"client_id", client.ID,
					"execution_id", client.executionID,
				)
			}

		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.send)
			}
			h.mu.Unlock()

			if h.logger != nil {
				h.logger.Info("WebSocket client disconnected",
					"client_id", client.ID,
				)
			}

		case message := <-h.broadcast:
			h.mu.RLock()
			for client := range h.clients {
				select {
				case client.send <- message:
				default:
					// Client's send buffer is full, disconnect
					close(client.send)
					delete(h.clients, client)
				}
			}
			h.mu.RUnlock()
		}
	}
}

// Register registers a new WebSocket client
func (h *WebSocketHub) Register(client *WebSocketClient) {
	h.register <- client
}

// Unregister unregisters a WebSocket client
func (h *WebSocketHub) Unregister(client *WebSocketClient) {
	h.unregister <- client
}

// Broadcast broadcasts a message to all connected clients
func (h *WebSocketHub) Broadcast(message []byte) {
	h.broadcast <- message
}

// BroadcastToExecution broadcasts a message to clients subscribed to specific execution
func (h *WebSocketHub) BroadcastToExecution(executionID string, message []byte) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	for client := range h.clients {
		// Send to clients that:
		// 1. Have no execution filter (want all events), OR
		// 2. Are subscribed to this specific execution
		if client.executionID == "" || client.executionID == executionID {
			select {
			case client.send <- message:
			default:
				// Client's send buffer is full, skip
				if h.logger != nil {
					h.logger.Warn("WebSocket client send buffer full, skipping message",
						"client_id", client.ID,
					)
				}
			}
		}
	}
}

// ClientCount returns the number of connected clients
func (h *WebSocketHub) ClientCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients)
}

// ============================================================================
// WebSocketClient implementation
// ============================================================================

// NewWebSocketClient creates a new WebSocket client
func NewWebSocketClient(id string, conn *websocket.Conn, hub *WebSocketHub, executionID string) *WebSocketClient {
	return &WebSocketClient{
		ID:            id,
		conn:          conn,
		send:          make(chan []byte, 256),
		hub:           hub,
		executionID:   executionID,
		subscriptions: make(map[EventType]bool),
	}
}

// ReadPump reads messages from the WebSocket connection
func (c *WebSocketClient) ReadPump() {
	defer func() {
		c.hub.Unregister(c)
		c.conn.Close()
	}()

	c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				if c.hub.logger != nil {
					c.hub.logger.Error("WebSocket read error",
						"client_id", c.ID,
						"error", err,
					)
				}
			}
			break
		}

		// Handle client messages (e.g., subscription updates)
		c.handleMessage(message)
	}
}

// WritePump writes messages to the WebSocket connection
func (c *WebSocketClient) WritePump() {
	ticker := time.NewTicker(54 * time.Second)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if !ok {
				// Hub closed the channel
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)

			// Add queued messages to current WebSocket message
			n := len(c.send)
			for i := 0; i < n; i++ {
				w.Write([]byte{'\n'})
				w.Write(<-c.send)
			}

			if err := w.Close(); err != nil {
				return
			}

		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// handleMessage handles messages from the client (e.g., subscription updates)
func (c *WebSocketClient) handleMessage(message []byte) {
	var msg map[string]any
	if err := json.Unmarshal(message, &msg); err != nil {
		return
	}

	// Handle subscription commands
	if cmd, ok := msg["command"].(string); ok {
		switch cmd {
		case "subscribe":
			// Subscribe to specific event types
			if eventTypes, ok := msg["event_types"].([]any); ok {
				c.mu.Lock()
				for _, et := range eventTypes {
					if eventType, ok := et.(string); ok {
						c.subscriptions[EventType(eventType)] = true
					}
				}
				c.mu.Unlock()
			}

		case "unsubscribe":
			// Unsubscribe from specific event types
			if eventTypes, ok := msg["event_types"].([]any); ok {
				c.mu.Lock()
				for _, et := range eventTypes {
					if eventType, ok := et.(string); ok {
						delete(c.subscriptions, EventType(eventType))
					}
				}
				c.mu.Unlock()
			}
		}
	}
}

// IsSubscribed checks if client is subscribed to an event type
func (c *WebSocketClient) IsSubscribed(eventType EventType) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()

	// If no subscriptions, client receives all events
	if len(c.subscriptions) == 0 {
		return true
	}

	return c.subscriptions[eventType]
}
