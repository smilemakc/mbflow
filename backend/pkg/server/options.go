package server

import (
	"github.com/smilemakc/mbflow/internal/config"
	"github.com/smilemakc/mbflow/internal/infrastructure/logger"
	"github.com/smilemakc/mbflow/pkg/executor"
)

// Option is a functional option for configuring the server
type Option func(*Server) error

// WithConfig sets the server configuration
func WithConfig(cfg *config.Config) Option {
	return func(s *Server) error {
		s.config = cfg
		return nil
	}
}

// WithLogger sets a custom logger
func WithLogger(l *logger.Logger) Option {
	return func(s *Server) error {
		s.logger = l
		return nil
	}
}

// WithExecutorManager sets a custom executor manager
func WithExecutorManager(m executor.Manager) Option {
	return func(s *Server) error {
		s.executorManager = m
		return nil
	}
}
