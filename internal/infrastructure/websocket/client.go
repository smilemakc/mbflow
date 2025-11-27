package websocket

import (
	"encoding/json"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

const (
	// Time allowed to write a message to the peer.
	writeWait = 10 * time.Second

	// Time allowed to read the next pong message from the peer.
	pongWait = 60 * time.Second

	// Send pings to peer with this period. Must be less than pongWait.
	pingPeriod = (pongWait * 9) / 10

	// Maximum message size allowed from peer.
	maxMessageSize = 512

	// Size of the send channel buffer
	sendBufferSize = 64
)

// Subscriptions tracks what a client is subscribed to
type Subscriptions struct {
	workflows  map[string]bool // workflow_id -> subscribed
	executions map[string]bool // execution_id -> subscribed
	mu         sync.RWMutex
}

// NewSubscriptions creates a new Subscriptions instance
func NewSubscriptions() *Subscriptions {
	return &Subscriptions{
		workflows:  make(map[string]bool),
		executions: make(map[string]bool),
	}
}

// Client represents a WebSocket client connection
type Client struct {
	hub  *Hub
	conn *websocket.Conn
	send chan *WSEvent

	id     string
	userID string
	subs   *Subscriptions
}

// NewClient creates a new Client instance
func NewClient(id, userID string, hub *Hub, conn *websocket.Conn) *Client {
	return &Client{
		hub:    hub,
		conn:   conn,
		send:   make(chan *WSEvent, sendBufferSize),
		id:     id,
		userID: userID,
		subs:   NewSubscriptions(),
	}
}

// shouldReceive checks if the client should receive an event based on subscriptions
func (c *Client) shouldReceive(workflowID, executionID string) bool {
	c.subs.mu.RLock()
	defer c.subs.mu.RUnlock()

	// Check execution subscription (most specific)
	if executionID != "" {
		if _, ok := c.subs.executions[executionID]; ok {
			return true
		}
	}

	// Check workflow subscription
	if workflowID != "" {
		if _, ok := c.subs.workflows[workflowID]; ok {
			return true
		}
	}

	return false
}

// readPump pumps messages from the WebSocket connection to the hub.
// It reads commands from the client and processes them.
func (c *Client) readPump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()

	c.conn.SetReadLimit(maxMessageSize)
	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				c.hub.logger.Warn("websocket unexpected close",
					"client_id", c.id,
					"error", err)
			}
			break
		}

		var cmd WSCommand
		if err := json.Unmarshal(message, &cmd); err != nil {
			c.sendResponse(NewErrorResponse("error", "invalid command format"))
			continue
		}

		c.handleCommand(&cmd)
	}
}

// writePump pumps messages from the hub to the WebSocket connection.
// It sends events to the client.
func (c *Client) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case event, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				// Channel was closed
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			if err := c.writeJSON(event); err != nil {
				return
			}

		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// handleCommand processes a command from the client
func (c *Client) handleCommand(cmd *WSCommand) {
	switch cmd.Action {
	case CmdSubscribe:
		c.handleSubscribe(cmd)
	case CmdUnsubscribe:
		c.handleUnsubscribe(cmd)
	case CmdCancel:
		c.handleCancel(cmd)
	default:
		c.sendResponse(NewErrorResponse("error", "unknown command: "+cmd.Action))
	}
}

// handleSubscribe processes a subscribe command
func (c *Client) handleSubscribe(cmd *WSCommand) {
	if cmd.WorkflowID == "" && cmd.ExecutionID == "" {
		c.sendResponse(NewErrorResponse(CmdSubscribe, "workflow_id or execution_id required"))
		return
	}

	c.hub.Subscribe(c, cmd.WorkflowID, cmd.ExecutionID)

	msg := "subscribed"
	if cmd.ExecutionID != "" {
		msg = "subscribed to execution: " + cmd.ExecutionID
	} else if cmd.WorkflowID != "" {
		msg = "subscribed to workflow: " + cmd.WorkflowID
	}

	c.sendResponse(NewSuccessResponse(CmdSubscribe, msg))
}

// handleUnsubscribe processes an unsubscribe command
func (c *Client) handleUnsubscribe(cmd *WSCommand) {
	if cmd.WorkflowID == "" && cmd.ExecutionID == "" {
		c.sendResponse(NewErrorResponse(CmdUnsubscribe, "workflow_id or execution_id required"))
		return
	}

	c.hub.Unsubscribe(c, cmd.WorkflowID, cmd.ExecutionID)

	msg := "unsubscribed"
	if cmd.ExecutionID != "" {
		msg = "unsubscribed from execution: " + cmd.ExecutionID
	} else if cmd.WorkflowID != "" {
		msg = "unsubscribed from workflow: " + cmd.WorkflowID
	}

	c.sendResponse(NewSuccessResponse(CmdUnsubscribe, msg))
}

// handleCancel processes a cancel command (placeholder for future implementation)
func (c *Client) handleCancel(cmd *WSCommand) {
	if cmd.ExecutionID == "" {
		c.sendResponse(NewErrorResponse(CmdCancel, "execution_id required"))
		return
	}

	// TODO: Implement execution cancellation
	// This would require access to the executor to cancel the execution

	c.sendResponse(NewErrorResponse(CmdCancel, "cancel not implemented yet"))
}

// sendResponse sends a response to the client
func (c *Client) sendResponse(resp *WSResponse) {
	c.conn.SetWriteDeadline(time.Now().Add(writeWait))
	c.writeJSON(resp)
}

// writeJSON writes a JSON message to the WebSocket connection
func (c *Client) writeJSON(v interface{}) error {
	return c.conn.WriteJSON(v)
}
