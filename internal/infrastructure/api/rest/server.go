package rest

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/smilemakc/mbflow/internal/domain"
)

type Server struct {
	store  domain.Storage
	mux    *http.ServeMux
	logger *slog.Logger
}

func NewServer(store domain.Storage, logger *slog.Logger) *Server {
	s := &Server{
		store:  store,
		mux:    http.NewServeMux(),
		logger: logger,
	}
	s.routes()
	return s
}

func (s *Server) routes() {
	s.mux.HandleFunc("GET /api/v1/workflows", s.handleWorkflows)
	s.mux.HandleFunc("POST /api/v1/workflows/execute", s.handleExecuteWorkflow)
	s.mux.HandleFunc("GET /api/v1/executions", s.handleExecutions)
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.logger.Info("request received", "method", r.Method, "path", r.URL.Path)
	s.mux.ServeHTTP(w, r)
}

func (s *Server) handleWorkflows(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()
	list, err := s.store.ListWorkflows(ctx)
	if err != nil {
		s.logger.Error("failed to list workflows", "error", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	if err := json.NewEncoder(w).Encode(list); err != nil {
		s.logger.Error("failed to encode workflows", "error", err)
	}
}

func (s *Server) handleExecuteWorkflow(w http.ResponseWriter, r *http.Request) {
	// MVP stub: respond OK
	w.WriteHeader(http.StatusAccepted)
	_, _ = w.Write([]byte(`{"status":"queued"}`))
}

func (s *Server) handleExecutions(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()
	list, err := s.store.ListExecutions(ctx)
	if err != nil {
		s.logger.Error("failed to list executions", "error", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	if err := json.NewEncoder(w).Encode(list); err != nil {
		s.logger.Error("failed to encode executions", "error", err)
	}
}
