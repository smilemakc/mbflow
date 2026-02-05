// Package server provides an embeddable HTTP server for MBFlow.
package server

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/uptrace/bun"
	grpclib "google.golang.org/grpc"

	"github.com/smilemakc/mbflow/internal/application/auth"
	"github.com/smilemakc/mbflow/internal/application/engine"
	"github.com/smilemakc/mbflow/internal/application/filestorage"
	"github.com/smilemakc/mbflow/internal/application/observer"
	"github.com/smilemakc/mbflow/internal/application/rentalkey"
	"github.com/smilemakc/mbflow/internal/application/serviceapi"
	"github.com/smilemakc/mbflow/internal/application/servicekey"
	"github.com/smilemakc/mbflow/internal/application/systemkey"
	"github.com/smilemakc/mbflow/internal/application/trigger"
	"github.com/smilemakc/mbflow/internal/config"
	"github.com/smilemakc/mbflow/internal/domain/repository"
	serviceapigrpc "github.com/smilemakc/mbflow/internal/infrastructure/api/grpc"
	"github.com/smilemakc/mbflow/internal/infrastructure/api/rest"
	"github.com/smilemakc/mbflow/internal/infrastructure/cache"
	"github.com/smilemakc/mbflow/internal/infrastructure/logger"
	"github.com/smilemakc/mbflow/internal/infrastructure/storage"
	"github.com/smilemakc/mbflow/pkg/crypto"
	"github.com/smilemakc/mbflow/pkg/executor"
)

// Server represents the MBFlow HTTP server
type Server struct {
	config     *config.Config
	logger     *logger.Logger
	router     *gin.Engine
	httpServer *http.Server

	db         *bun.DB
	redisCache *cache.RedisCache

	executorManager    executor.Manager
	executionManager   *engine.ExecutionManager
	triggerManager     *trigger.Manager
	observerManager    *observer.ObserverManager
	fileStorageManager *filestorage.StorageManager

	workflowRepo    *storage.WorkflowRepository
	executionRepo   *storage.ExecutionRepository
	eventRepo       *storage.EventRepository
	triggerRepo     repository.TriggerRepository
	userRepo        *storage.UserRepository
	fileRepo        *storage.FileRepository
	accountRepo     *storage.AccountRepositoryImpl
	transactionRepo *storage.TransactionRepositoryImpl
	resourceRepo    *storage.ResourceRepositoryImpl
	pricingPlanRepo *storage.PricingPlanRepositoryImpl
	credentialsRepo *storage.CredentialsRepositoryImpl
	serviceKeyRepo  *storage.ServiceKeyRepositoryImpl
	systemKeyRepo   *storage.SystemKeyRepoImpl
	auditLogRepo    *storage.ServiceAuditLogRepoImpl

	authService       *auth.Service
	serviceKeyService *servicekey.Service
	providerManager   *auth.ProviderManager

	systemKeyService_ *systemkey.Service
	auditService      *systemkey.AuditService

	systemAuthMiddleware *rest.SystemAuthMiddleware
	auditMiddleware      *rest.AuditMiddleware

	grpcServer     *grpclib.Server
	grpcListener   net.Listener
	serviceAPIOps  *serviceapi.Operations
	serviceAPIGRPC *serviceapigrpc.ServiceAPIServer

	wsHub             *observer.WebSocketHub
	encryptionService *crypto.EncryptionService
	rentalKeyRepo     *storage.RentalKeyRepositoryImpl
	rentalKeyProvider *rentalkey.Provider
	authMiddleware    *rest.AuthMiddleware
	loginRateLimiter  *rest.LoginRateLimiter
}

// New creates a new server with the given options
func New(opts ...Option) (*Server, error) {
	s := &Server{}

	for _, opt := range opts {
		if err := opt(s); err != nil {
			return nil, fmt.Errorf("failed to apply option: %w", err)
		}
	}

	if s.config == nil {
		cfg, err := config.Load()
		if err != nil {
			return nil, fmt.Errorf("failed to load configuration: %w", err)
		}
		s.config = cfg
	}

	if s.logger == nil {
		s.logger = logger.New(s.config.Logging)
		logger.SetDefault(s.logger)
	}

	if err := s.initComponents(); err != nil {
		return nil, fmt.Errorf("failed to initialize components: %w", err)
	}

	if err := s.setupRoutes(); err != nil {
		return nil, fmt.Errorf("failed to setup routes: %w", err)
	}

	if err := s.setupGRPCServer(); err != nil {
		return nil, fmt.Errorf("failed to setup gRPC server: %w", err)
	}

	s.httpServer = &http.Server{
		Addr:         fmt.Sprintf("%s:%d", s.config.Server.Host, s.config.Server.Port),
		Handler:      s.router,
		ReadTimeout:  s.config.Server.ReadTimeout,
		WriteTimeout: s.config.Server.WriteTimeout,
		IdleTimeout:  120 * time.Second,
	}

	return s, nil
}

// Run starts the server and blocks until a shutdown signal is received
func (s *Server) Run() error {
	s.logger.Info("Starting MBFlow Server",
		"version", "1.0.0",
		"host", s.config.Server.Host,
		"port", s.config.Server.Port,
	)

	serverErrors := make(chan error, 1)
	go func() {
		s.logger.Info("HTTP server starting",
			"host", s.config.Server.Host,
			"port", s.config.Server.Port,
		)
		serverErrors <- s.httpServer.ListenAndServe()
	}()

	if s.grpcServer != nil {
		go func() {
			s.logger.Info("gRPC Service API server starting", "address", s.config.GRPCServiceAPI.Address)
			if err := s.grpcServer.Serve(s.grpcListener); err != nil {
				s.logger.Error("gRPC server error", "error", err)
				serverErrors <- err
			}
		}()
	}

	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, os.Interrupt, syscall.SIGTERM)

	select {
	case err := <-serverErrors:
		return fmt.Errorf("server error: %w", err)

	case sig := <-shutdown:
		s.logger.Info("Server shutdown initiated", "signal", sig)

		ctx, cancel := context.WithTimeout(context.Background(), s.config.Server.ShutdownTimeout)
		defer cancel()

		return s.Shutdown(ctx)
	}
}

// Shutdown gracefully stops the server
func (s *Server) Shutdown(ctx context.Context) error {
	if s.triggerManager != nil {
		s.logger.Info("Stopping trigger manager...")
		if err := s.triggerManager.Stop(); err != nil {
			s.logger.Error("Trigger manager shutdown failed", "error", err)
		} else {
			s.logger.Info("Trigger manager stopped")
		}
	}

	if s.fileStorageManager != nil {
		s.logger.Info("Closing file storage manager...")
		if err := s.fileStorageManager.Close(); err != nil {
			s.logger.Error("File storage manager shutdown failed", "error", err)
		} else {
			s.logger.Info("File storage manager closed")
		}
	}

	if s.grpcServer != nil {
		s.logger.Info("Stopping gRPC Service API server...")
		s.grpcServer.GracefulStop()
		s.logger.Info("gRPC Service API server stopped")
	}

	if err := s.httpServer.Shutdown(ctx); err != nil {
		s.logger.Error("Graceful shutdown failed", "error", err)
		if err := s.httpServer.Close(); err != nil {
			s.logger.Error("Server close failed", "error", err)
		}
	}

	// Close Redis cache
	if s.redisCache != nil {
		s.logger.Info("Closing Redis cache...")
		if err := s.redisCache.Close(); err != nil {
			s.logger.Error("Redis cache close failed", "error", err)
		} else {
			s.logger.Info("Redis cache closed")
		}
	}

	// Close database connection
	if s.db != nil {
		s.logger.Info("Closing database connection...")
		if err := storage.Close(s.db); err != nil {
			s.logger.Error("Database close failed", "error", err)
		} else {
			s.logger.Info("Database connection closed")
		}
	}

	s.logger.Info("Server stopped")
	return nil
}

// Router returns the Gin router for adding custom endpoints
func (s *Server) Router() *gin.Engine {
	return s.router
}

// RegisterExecutor registers a custom executor
func (s *Server) RegisterExecutor(nodeType string, exec executor.Executor) error {
	if s.executorManager == nil {
		return fmt.Errorf("executor manager not initialized")
	}
	return s.executorManager.Register(nodeType, exec)
}

// Config returns the server configuration
func (s *Server) Config() *config.Config {
	return s.config
}

// Logger returns the server logger
func (s *Server) Logger() *logger.Logger {
	return s.logger
}

// DB returns the database connection
func (s *Server) DB() *bun.DB {
	return s.db
}

// ExecutorManager returns the executor manager
func (s *Server) ExecutorManager() executor.Manager {
	return s.executorManager
}

// ExecutionManager returns the execution manager
func (s *Server) ExecutionManager() *engine.ExecutionManager {
	return s.executionManager
}

// TriggerManager returns the trigger manager
func (s *Server) TriggerManager() *trigger.Manager {
	return s.triggerManager
}

// ObserverManager returns the observer manager
func (s *Server) ObserverManager() *observer.ObserverManager {
	return s.observerManager
}

// FileStorageManager returns the file storage manager
func (s *Server) FileStorageManager() *filestorage.StorageManager {
	return s.fileStorageManager
}

// WorkflowRepository returns the workflow repository
func (s *Server) WorkflowRepository() *storage.WorkflowRepository {
	return s.workflowRepo
}

// ExecutionRepository returns the execution repository
func (s *Server) ExecutionRepository() *storage.ExecutionRepository {
	return s.executionRepo
}

// AuthService returns the auth service
func (s *Server) AuthService() *auth.Service {
	return s.authService
}

// ServiceKeyService returns the service key service
func (s *Server) ServiceKeyService() *servicekey.Service {
	return s.serviceKeyService
}
