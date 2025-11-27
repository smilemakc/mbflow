package rest

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/smilemakc/mbflow"
	"github.com/smilemakc/mbflow/internal/domain"
	"github.com/smilemakc/mbflow/internal/infrastructure/websocket"
)

// Server represents the REST API server
type Server struct {
	store    domain.Storage
	executor *mbflow.Executor
	mux      *http.ServeMux
	logger   *slog.Logger
	config   ServerConfig

	// WebSocket components
	wsHub     *websocket.Hub
	wsHandler *websocket.Handler
}

// ServerConfig holds server configuration
type ServerConfig struct {
	EnableCORS      bool
	EnableRateLimit bool
	RateLimitMax    int
	RateLimitWindow time.Duration
	APIKeys         []string

	// WebSocket configuration
	EnableWebSocket bool
	JWTSecret       string // Secret for JWT validation (empty = no auth)
}

// DefaultServerConfig returns default server configuration
func DefaultServerConfig() ServerConfig {
	return ServerConfig{
		EnableCORS:      true,
		EnableRateLimit: false,
		RateLimitMax:    100,
		RateLimitWindow: time.Minute,
		APIKeys:         []string{},
		EnableWebSocket: true,
		JWTSecret:       "",
	}
}

// NewServer creates a new REST API server
func NewServer(store domain.Storage, executor *mbflow.Executor, logger *slog.Logger, config ServerConfig) *Server {
	s := &Server{
		store:    store,
		executor: executor,
		mux:      http.NewServeMux(),
		logger:   logger,
		config:   config,
	}

	// Initialize WebSocket if enabled
	if config.EnableWebSocket {
		s.initWebSocket()
	}

	s.routes()
	return s
}

// initWebSocket initializes WebSocket hub and handler
func (s *Server) initWebSocket() {
	// Create WebSocket hub
	s.wsHub = websocket.NewHub(s.logger)

	// Start hub in background
	go s.wsHub.Run()

	// Create authenticator
	var auth websocket.Authenticator
	if s.config.JWTSecret != "" {
		auth = websocket.NewJWTAuth(s.config.JWTSecret)
	} else {
		auth = websocket.NewNoAuth()
	}

	// Create handler
	s.wsHandler = websocket.NewHandler(s.wsHub, auth, s.logger)

	// Register socket observer with executor
	socketObserver := websocket.NewSocketObserver(s.wsHub)
	s.executor.AddObserver(socketObserver)

	s.logger.Info("WebSocket support enabled")
}

// WebSocketHub returns the WebSocket hub (useful for external access)
func (s *Server) WebSocketHub() *websocket.Hub {
	return s.wsHub
}

// routes registers all HTTP routes
func (s *Server) routes() {
	// Health check
	s.mux.HandleFunc("GET /health", s.handleHealth)
	s.mux.HandleFunc("GET /ready", s.handleReady)

	// Workflow management
	s.mux.HandleFunc("GET /api/v1/workflows", s.handleListWorkflows)
	s.mux.HandleFunc("GET /api/v1/workflows/{id}", s.handleGetWorkflow)
	s.mux.HandleFunc("POST /api/v1/workflows", s.handleCreateWorkflow)
	s.mux.HandleFunc("PUT /api/v1/workflows/{id}", s.handleUpdateWorkflow)
	s.mux.HandleFunc("DELETE /api/v1/workflows/{id}", s.handleDeleteWorkflow)

	// Node management
	s.mux.HandleFunc("GET /api/v1/workflows/{workflow_id}/nodes", s.handleListNodes)
	s.mux.HandleFunc("GET /api/v1/workflows/{workflow_id}/nodes/{node_id}", s.handleGetNode)
	s.mux.HandleFunc("POST /api/v1/workflows/{workflow_id}/nodes", s.handleCreateNode)
	s.mux.HandleFunc("PUT /api/v1/workflows/{workflow_id}/nodes/{node_id}", s.handleUpdateNode)
	s.mux.HandleFunc("DELETE /api/v1/workflows/{workflow_id}/nodes/{node_id}", s.handleDeleteNode)

	// Node types reference
	s.mux.HandleFunc("GET /api/v1/node-types", s.handleGetNodeTypes)

	// Edge management
	s.mux.HandleFunc("GET /api/v1/edge-types", s.handleGetEdgeTypes)
	s.mux.HandleFunc("GET /api/v1/workflows/{workflow_id}/edges", s.handleListEdges)
	s.mux.HandleFunc("GET /api/v1/workflows/{workflow_id}/edges/{edge_id}", s.handleGetEdge)
	s.mux.HandleFunc("POST /api/v1/workflows/{workflow_id}/edges", s.handleCreateEdge)
	s.mux.HandleFunc("PUT /api/v1/workflows/{workflow_id}/edges/{edge_id}", s.handleUpdateEdge)
	s.mux.HandleFunc("DELETE /api/v1/workflows/{workflow_id}/edges/{edge_id}", s.handleDeleteEdge)
	s.mux.HandleFunc("GET /api/v1/workflows/{workflow_id}/graph", s.handleGetWorkflowGraph)

	// Execution management
	s.mux.HandleFunc("GET /api/v1/executions", s.handleListExecutions)
	s.mux.HandleFunc("GET /api/v1/executions/{id}", s.handleGetExecution)
	s.mux.HandleFunc("POST /api/v1/executions", s.handleExecuteWorkflow)
	s.mux.HandleFunc("GET /api/v1/executions/{id}/events", s.handleGetExecutionEvents)
	s.mux.HandleFunc("POST /api/v1/executions/{id}/cancel", s.handleCancelExecution)
	s.mux.HandleFunc("POST /api/v1/executions/{id}/pause", s.handlePauseExecution)
	s.mux.HandleFunc("POST /api/v1/executions/{id}/resume", s.handleResumeExecution)

	// WebSocket endpoint
	if s.wsHandler != nil {
		s.mux.Handle("GET /ws", s.wsHandler)
	}
}

// ServeHTTP implements http.Handler with middleware chain
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	handler := http.Handler(s.mux)

	// Apply middleware in reverse order (last to first)
	handler = loggingMiddleware(s.logger, handler)
	handler = recoveryMiddleware(s.logger, handler)
	handler = contentTypeMiddleware(handler)

	if s.config.EnableCORS {
		handler = corsMiddleware(handler)
	}

	if s.config.EnableRateLimit {
		rateLimiter := newRateLimiter(s.config.RateLimitMax, s.config.RateLimitWindow)
		handler = rateLimiter.middleware(handler)
	}

	if len(s.config.APIKeys) > 0 {
		auth := newAuthMiddleware(s.config.APIKeys)
		handler = auth.middleware(handler)
	}

	handler.ServeHTTP(w, r)
}

// handleHealth handles GET /health
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	s.respondJSON(w, map[string]string{
		"status": "ok",
		"time":   time.Now().Format(time.RFC3339),
	}, http.StatusOK)
}

// handleReady handles GET /ready
func (s *Server) handleReady(w http.ResponseWriter, r *http.Request) {
	s.respondJSON(w, map[string]string{
		"status": "ready",
		"time":   time.Now().Format(time.RFC3339),
	}, http.StatusOK)
}

// respondJSON writes a JSON response
func (s *Server) respondJSON(w http.ResponseWriter, data interface{}, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		s.logger.Error("failed to encode JSON response", "error", err)
	}
}

// respondError writes an error response
func (s *Server) respondError(w http.ResponseWriter, message string, statusCode int) {
	s.respondJSON(w, map[string]string{
		"error": message,
	}, statusCode)
}

// ErrorResponse represents an error response
type ErrorResponse struct {
	Error   string `json:"error"`
	Code    string `json:"code,omitempty"`
	Details string `json:"details,omitempty"`
}

// HealthResponse represents a health check response
type HealthResponse struct {
	Status  string    `json:"status"`
	Time    time.Time `json:"time"`
	Version string    `json:"version,omitempty"`
}
