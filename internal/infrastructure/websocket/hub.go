package websocket

import (
	"log/slog"
	"sync"
)

// Broadcaster interface for broadcasting events to WebSocket clients.
// This interface enables future Redis adapter implementation for horizontal scaling.
type Broadcaster interface {
	Broadcast(userID, workflowID, executionID string, event *WSEvent)
}

// broadcastMsg represents a message to be broadcast to clients
type broadcastMsg struct {
	userID      string
	workflowID  string
	executionID string
	event       *WSEvent
}

// Hub manages WebSocket connections and broadcasting events to clients.
// It implements the Broadcaster interface.
type Hub struct {
	// Registered clients
	clients map[*Client]bool

	// Channel for registering clients
	register chan *Client

	// Channel for unregistering clients
	unregister chan *Client

	// Channel for broadcasting events
	broadcast chan *broadcastMsg

	// Subscriptions indexes for fast lookup
	byUserID      map[string]map[*Client]bool
	byWorkflowID  map[string]map[*Client]bool
	byExecutionID map[string]map[*Client]bool

	logger *slog.Logger
	mu     sync.RWMutex
}

// NewHub creates a new Hub instance
func NewHub(logger *slog.Logger) *Hub {
	return &Hub{
		clients:       make(map[*Client]bool),
		register:      make(chan *Client),
		unregister:    make(chan *Client),
		broadcast:     make(chan *broadcastMsg, 256),
		byUserID:      make(map[string]map[*Client]bool),
		byWorkflowID:  make(map[string]map[*Client]bool),
		byExecutionID: make(map[string]map[*Client]bool),
		logger:        logger,
	}
}

// Run starts the hub's main event loop.
// This should be called in a goroutine.
func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.registerClient(client)

		case client := <-h.unregister:
			h.unregisterClient(client)

		case msg := <-h.broadcast:
			h.broadcastEvent(msg)
		}
	}
}

// registerClient adds a client to the hub
func (h *Hub) registerClient(client *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.clients[client] = true

	// Index by user ID
	if client.userID != "" {
		if h.byUserID[client.userID] == nil {
			h.byUserID[client.userID] = make(map[*Client]bool)
		}
		h.byUserID[client.userID][client] = true
	}

	h.logger.Debug("client registered",
		"client_id", client.id,
		"user_id", client.userID,
		"total_clients", len(h.clients))
}

// unregisterClient removes a client from the hub
func (h *Hub) unregisterClient(client *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if _, ok := h.clients[client]; !ok {
		return
	}

	delete(h.clients, client)
	close(client.send)

	// Remove from user index
	if client.userID != "" {
		if clients, ok := h.byUserID[client.userID]; ok {
			delete(clients, client)
			if len(clients) == 0 {
				delete(h.byUserID, client.userID)
			}
		}
	}

	// Remove from subscription indexes
	client.subs.mu.RLock()
	for wfID := range client.subs.workflows {
		if clients, ok := h.byWorkflowID[wfID]; ok {
			delete(clients, client)
			if len(clients) == 0 {
				delete(h.byWorkflowID, wfID)
			}
		}
	}
	for execID := range client.subs.executions {
		if clients, ok := h.byExecutionID[execID]; ok {
			delete(clients, client)
			if len(clients) == 0 {
				delete(h.byExecutionID, execID)
			}
		}
	}
	client.subs.mu.RUnlock()

	h.logger.Debug("client unregistered",
		"client_id", client.id,
		"user_id", client.userID,
		"total_clients", len(h.clients))
}

// Broadcast sends an event to relevant clients.
// Implements the Broadcaster interface.
func (h *Hub) Broadcast(userID, workflowID, executionID string, event *WSEvent) {
	h.broadcast <- &broadcastMsg{
		userID:      userID,
		workflowID:  workflowID,
		executionID: executionID,
		event:       event,
	}
}

// broadcastEvent sends an event to all matching clients
func (h *Hub) broadcastEvent(msg *broadcastMsg) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	// Collect target clients
	targets := make(map[*Client]bool)

	// If userID is specified, only send to that user's clients
	if msg.userID != "" {
		if clients, ok := h.byUserID[msg.userID]; ok {
			for client := range clients {
				if client.shouldReceive(msg.workflowID, msg.executionID) {
					targets[client] = true
				}
			}
		}
	} else {
		// Send to all clients that match the subscription
		// First check execution subscriptions (most specific)
		if msg.executionID != "" {
			if clients, ok := h.byExecutionID[msg.executionID]; ok {
				for client := range clients {
					targets[client] = true
				}
			}
		}

		// Then check workflow subscriptions
		if msg.workflowID != "" {
			if clients, ok := h.byWorkflowID[msg.workflowID]; ok {
				for client := range clients {
					targets[client] = true
				}
			}
		}
	}

	// Send to all target clients
	for client := range targets {
		select {
		case client.send <- msg.event:
		default:
			// Client send buffer full, skip this message
			h.logger.Warn("client buffer full, dropping message",
				"client_id", client.id,
				"event_type", msg.event.Type)
		}
	}
}

// Subscribe adds a subscription for a client
func (h *Hub) Subscribe(client *Client, workflowID, executionID string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	client.subs.mu.Lock()
	defer client.subs.mu.Unlock()

	if workflowID != "" {
		client.subs.workflows[workflowID] = true
		if h.byWorkflowID[workflowID] == nil {
			h.byWorkflowID[workflowID] = make(map[*Client]bool)
		}
		h.byWorkflowID[workflowID][client] = true

		h.logger.Debug("client subscribed to workflow",
			"client_id", client.id,
			"workflow_id", workflowID)
	}

	if executionID != "" {
		client.subs.executions[executionID] = true
		if h.byExecutionID[executionID] == nil {
			h.byExecutionID[executionID] = make(map[*Client]bool)
		}
		h.byExecutionID[executionID][client] = true

		h.logger.Debug("client subscribed to execution",
			"client_id", client.id,
			"execution_id", executionID)
	}
}

// Unsubscribe removes a subscription for a client
func (h *Hub) Unsubscribe(client *Client, workflowID, executionID string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	client.subs.mu.Lock()
	defer client.subs.mu.Unlock()

	if workflowID != "" {
		delete(client.subs.workflows, workflowID)
		if clients, ok := h.byWorkflowID[workflowID]; ok {
			delete(clients, client)
			if len(clients) == 0 {
				delete(h.byWorkflowID, workflowID)
			}
		}

		h.logger.Debug("client unsubscribed from workflow",
			"client_id", client.id,
			"workflow_id", workflowID)
	}

	if executionID != "" {
		delete(client.subs.executions, executionID)
		if clients, ok := h.byExecutionID[executionID]; ok {
			delete(clients, client)
			if len(clients) == 0 {
				delete(h.byExecutionID, executionID)
			}
		}

		h.logger.Debug("client unsubscribed from execution",
			"client_id", client.id,
			"execution_id", executionID)
	}
}

// ClientCount returns the number of connected clients
func (h *Hub) ClientCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients)
}
