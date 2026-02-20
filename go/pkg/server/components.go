package server

import (
	"fmt"
	"time"

	"github.com/smilemakc/mbflow/go/internal/application/auth"
	"github.com/smilemakc/mbflow/go/internal/application/engine"
	"github.com/smilemakc/mbflow/go/internal/application/filestorage"
	"github.com/smilemakc/mbflow/go/internal/application/observer"
	"github.com/smilemakc/mbflow/go/internal/application/rentalkey"
	"github.com/smilemakc/mbflow/go/internal/application/servicekey"
	"github.com/smilemakc/mbflow/go/internal/application/systemkey"
	"github.com/smilemakc/mbflow/go/internal/application/trigger"
	"github.com/smilemakc/mbflow/go/internal/infrastructure/api/rest"
	"github.com/smilemakc/mbflow/go/internal/infrastructure/cache"
	"github.com/smilemakc/mbflow/go/internal/infrastructure/storage"
	"github.com/smilemakc/mbflow/go/pkg/crypto"
	"github.com/smilemakc/mbflow/go/pkg/executor"
	"github.com/smilemakc/mbflow/go/pkg/executor/builtin"
)

func (s *Server) initComponents() error {
	if err := s.initDatabase(); err != nil {
		return fmt.Errorf("failed to initialize database: %w", err)
	}

	if err := s.initRedisCache(); err != nil {
		s.logger.Warn("Failed to initialize Redis cache", "error", err)
	}

	if err := s.initExecutorManager(); err != nil {
		return fmt.Errorf("failed to initialize executor manager: %w", err)
	}

	if err := s.initFileStorageManager(); err != nil {
		return fmt.Errorf("failed to initialize file storage manager: %w", err)
	}

	// Initialize repositories before observer manager (observer uses eventRepo)
	if err := s.initRepositories(); err != nil {
		return fmt.Errorf("failed to initialize repositories: %w", err)
	}

	if err := s.initObserverManager(); err != nil {
		return fmt.Errorf("failed to initialize observer manager: %w", err)
	}

	if err := s.initEncryptionServices(); err != nil {
		s.logger.Warn("Encryption service not available - credentials and rental keys features disabled", "error", err)
	}

	if err := s.initAuthSystem(); err != nil {
		return fmt.Errorf("failed to initialize auth system: %w", err)
	}

	if err := s.initSystemKeySystem(); err != nil {
		return fmt.Errorf("failed to initialize system key system: %w", err)
	}

	if err := s.initExecutionEngine(); err != nil {
		return fmt.Errorf("failed to initialize execution engine: %w", err)
	}

	if err := s.initTriggerManager(); err != nil {
		s.logger.Warn("Failed to initialize trigger manager", "error", err)
	}

	return nil
}

func (s *Server) initDatabase() error {
	dbConfig := &storage.Config{
		DSN:             s.config.Database.URL,
		MaxOpenConns:    s.config.Database.MaxConnections,
		MaxIdleConns:    s.config.Database.MinConnections,
		ConnMaxLifetime: s.config.Database.MaxConnLifetime,
		ConnMaxIdleTime: s.config.Database.MaxIdleTime,
		Debug:           s.config.Logging.Level == "debug",
	}

	db, err := storage.NewDB(dbConfig)
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}

	s.data.DB = db
	s.logger.Info("Database connected",
		"max_conns", s.config.Database.MaxConnections,
	)

	return nil
}

func (s *Server) initRedisCache() error {
	redisCache, err := cache.NewRedisCache(s.config.Redis)
	if err != nil {
		return fmt.Errorf("failed to create redis cache: %w", err)
	}

	s.data.RedisCache = redisCache
	s.logger.Info("Redis cache connected")
	return nil
}

func (s *Server) initExecutorManager() error {
	s.execution.ExecutorManager = executor.NewManager()

	if err := builtin.RegisterBuiltins(s.execution.ExecutorManager); err != nil {
		return fmt.Errorf("failed to register built-in executors: %w", err)
	}

	s.logger.Info("Registered executors", "types", s.execution.ExecutorManager.List())
	return nil
}

func (s *Server) initFileStorageManager() error {
	fileStorageConfig := filestorage.DefaultManagerConfig()
	fileStorageConfig.BasePath = s.config.FileStorage.StoragePath
	fileStorageConfig.MaxFileSize = s.config.FileStorage.MaxFileSize

	s.fileStorage.FileStorageManager = filestorage.NewStorageManager(fileStorageConfig, s.logger)

	s.logger.Info("File storage manager initialized",
		"base_path", s.config.FileStorage.StoragePath,
		"max_file_size", s.config.FileStorage.MaxFileSize,
	)

	if err := builtin.RegisterFileStorage(s.execution.ExecutorManager, s.fileStorage.FileStorageManager); err != nil {
		return fmt.Errorf("failed to register file_storage executor: %w", err)
	}

	if err := builtin.RegisterAdapters(s.execution.ExecutorManager); err != nil {
		return fmt.Errorf("failed to register adapter executors: %w", err)
	}

	if err := builtin.RegisterFileAdapters(s.execution.ExecutorManager, s.fileStorage.FileStorageManager); err != nil {
		return fmt.Errorf("failed to register file adapter executors: %w", err)
	}

	return nil
}

func (s *Server) initObserverManager() error {
	if s.config.Observer.EnableWebSocket {
		s.execution.WSHub = observer.NewWebSocketHub(s.logger)
		s.logger.Info("WebSocket hub initialized")
	}

	s.execution.ObserverManager = observer.NewObserverManager(
		observer.WithLogger(s.logger),
		observer.WithBufferSize(s.config.Observer.BufferSize),
	)

	if s.config.Observer.EnableDatabase {
		dbObserver := observer.NewDatabaseObserver(s.data.EventRepo)
		if err := s.execution.ObserverManager.Register(dbObserver); err != nil {
			s.logger.Error("Failed to register database observer", "error", err)
		} else {
			s.logger.Info("Database observer registered")
		}
	}

	if s.config.Observer.EnableHTTP && s.config.Observer.HTTPCallbackURL != "" {
		httpObserver := observer.NewHTTPCallbackObserver(
			s.config.Observer.HTTPCallbackURL,
			observer.WithHTTPMethod(s.config.Observer.HTTPMethod),
			observer.WithHTTPHeaders(s.config.Observer.HTTPHeaders),
			observer.WithHTTPTimeout(s.config.Observer.HTTPTimeout),
			observer.WithHTTPRetry(
				s.config.Observer.HTTPMaxRetries,
				s.config.Observer.HTTPRetryDelay,
				2.0,
			),
		)
		if err := s.execution.ObserverManager.Register(httpObserver); err != nil {
			s.logger.Error("Failed to register HTTP observer", "error", err)
		} else {
			s.logger.Info("HTTP callback observer registered",
				"url", s.config.Observer.HTTPCallbackURL,
				"method", s.config.Observer.HTTPMethod,
			)
		}
	}

	if s.config.Observer.EnableLogger {
		loggerObserver := observer.NewLoggerObserver(
			observer.WithLoggerInstance(s.logger),
		)
		if err := s.execution.ObserverManager.Register(loggerObserver); err != nil {
			s.logger.Error("Failed to register logger observer", "error", err)
		} else {
			s.logger.Info("Logger observer registered")
		}
	}

	if s.config.Observer.EnableWebSocket && s.execution.WSHub != nil {
		wsObserver := observer.NewWebSocketObserver(
			s.execution.WSHub,
			observer.WithWebSocketLogger(s.logger),
		)
		if err := s.execution.ObserverManager.Register(wsObserver); err != nil {
			s.logger.Error("Failed to register WebSocket observer", "error", err)
		} else {
			s.logger.Info("WebSocket observer registered")
		}
	}

	s.logger.Info("Observer system initialized",
		"observer_count", s.execution.ObserverManager.Count(),
	)

	return nil
}

func (s *Server) initRepositories() error {
	s.data.WorkflowRepo = storage.NewWorkflowRepository(s.data.DB)
	s.data.ExecutionRepo = storage.NewExecutionRepository(s.data.DB)
	s.data.EventRepo = storage.NewEventRepository(s.data.DB)
	s.data.TriggerRepo = storage.NewTriggerRepository(s.data.DB)
	s.data.UserRepo = storage.NewUserRepository(s.data.DB)
	s.data.FileRepo = storage.NewFileRepository(s.data.DB)
	s.data.AccountRepo = storage.NewAccountRepository(s.data.DB)
	s.data.TransactionRepo = storage.NewTransactionRepository(s.data.DB)
	s.data.ResourceRepo = storage.NewResourceRepository(s.data.DB)
	s.data.PricingPlanRepo = storage.NewPricingPlanRepository(s.data.DB)
	s.data.CredentialsRepo = storage.NewCredentialsRepository(s.data.DB)
	s.data.ServiceKeyRepo = storage.NewServiceKeyRepository(s.data.DB)
	s.data.SystemKeyRepo = storage.NewSystemKeyRepo(s.data.DB)
	s.data.AuditLogRepo = storage.NewServiceAuditLogRepo(s.data.DB)

	s.logger.Info("Repositories initialized")
	return nil
}

func (s *Server) initEncryptionServices() error {
	encryptionService, err := crypto.GetDefaultService()
	if err != nil {
		return fmt.Errorf("encryption service not available: %w", err)
	}

	s.auth.EncryptionService = encryptionService
	s.logger.Info("Encryption service initialized")

	s.data.RentalKeyRepo = storage.NewRentalKeyRepository(s.data.DB, encryptionService)
	s.auth.RentalKeyProvider = rentalkey.NewProvider(s.data.RentalKeyRepo, encryptionService)

	s.logger.Info("Rental key provider initialized")
	return nil
}

func (s *Server) initAuthSystem() error {
	s.auth.AuthService = auth.NewService(s.data.UserRepo, s.data.AccountRepo, &s.config.Auth)

	providerManager, err := auth.NewProviderManager(&s.config.Auth, s.auth.AuthService)
	if err != nil {
		s.logger.Warn("Failed to initialize auth provider manager", "error", err)
	}
	s.auth.ProviderManager = providerManager

	s.auth.ServiceKeyService = servicekey.NewService(s.data.ServiceKeyRepo, servicekey.Config{
		MaxKeysPerUser:    s.config.ServiceKeys.MaxKeysPerUser,
		DefaultExpiryDays: s.config.ServiceKeys.DefaultExpiryDays,
	})

	s.auth.AuthMiddleware = rest.NewAuthMiddleware(s.auth.ProviderManager, s.auth.AuthService, s.auth.ServiceKeyService)
	s.auth.LoginRateLimiter = rest.NewLoginRateLimiter(
		s.config.Auth.MaxLoginAttempts,
		time.Duration(s.config.Auth.MaxLoginAttempts)*time.Minute,
		s.config.Auth.LockoutDuration,
	)

	s.logger.Info("Auth system initialized",
		"mode", s.config.Auth.Mode,
		"registration_enabled", s.config.Auth.AllowRegistration,
	)

	s.logger.Info("Service key service initialized",
		"max_keys_per_user", s.config.ServiceKeys.MaxKeysPerUser,
		"default_expiry_days", s.config.ServiceKeys.DefaultExpiryDays,
	)

	return nil
}

func (s *Server) initExecutionEngine() error {
	s.execution.ExecutionManager = engine.NewExecutionManager(
		s.execution.ExecutorManager,
		s.data.WorkflowRepo,
		s.data.ExecutionRepo,
		s.data.EventRepo,
		s.data.ResourceRepo,
		s.execution.ObserverManager,
	)

	s.logger.Info("Execution engine initialized")
	return nil
}

func (s *Server) initTriggerManager() error {
	if s.data.RedisCache == nil {
		return fmt.Errorf("trigger manager disabled - Redis cache not available")
	}

	triggerManager, err := trigger.NewManager(trigger.ManagerConfig{
		TriggerRepo:  s.data.TriggerRepo,
		WorkflowRepo: s.data.WorkflowRepo,
		ExecutionMgr: s.execution.ExecutionManager,
		Cache:        s.data.RedisCache,
	})
	if err != nil {
		return fmt.Errorf("failed to create trigger manager: %w", err)
	}

	s.triggers.TriggerManager = triggerManager
	s.logger.Info("Trigger manager initialized")

	if err := s.triggers.TriggerManager.Start(); err != nil {
		return fmt.Errorf("failed to start trigger manager: %w", err)
	}

	s.logger.Info("Trigger manager started")
	return nil
}

func (s *Server) initSystemKeySystem() error {
	s.serviceAPI.SystemKeyService = systemkey.NewService(s.data.SystemKeyRepo, systemkey.Config{
		MaxKeys:           s.config.ServiceAPI.MaxKeys,
		DefaultExpiryDays: s.config.ServiceAPI.DefaultExpiryDays,
		BcryptCost:        s.config.ServiceAPI.BcryptCost,
	})
	s.serviceAPI.AuditService = systemkey.NewAuditService(s.data.AuditLogRepo, s.config.ServiceAPI.AuditRetentionDays)
	s.serviceAPI.SystemAuthMiddleware = rest.NewSystemAuthMiddleware(s.serviceAPI.SystemKeyService, s.data.UserRepo, s.config.ServiceAPI.SystemUserID, s.logger)
	s.serviceAPI.AuditMiddleware = rest.NewAuditMiddleware(s.serviceAPI.AuditService, s.logger)
	s.logger.Info("System key system initialized")
	return nil
}
