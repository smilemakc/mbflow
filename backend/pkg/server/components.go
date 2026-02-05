package server

import (
	"fmt"
	"time"

	"github.com/smilemakc/mbflow/internal/application/auth"
	"github.com/smilemakc/mbflow/internal/application/engine"
	"github.com/smilemakc/mbflow/internal/application/filestorage"
	"github.com/smilemakc/mbflow/internal/application/observer"
	"github.com/smilemakc/mbflow/internal/application/rentalkey"
	"github.com/smilemakc/mbflow/internal/application/servicekey"
	"github.com/smilemakc/mbflow/internal/application/systemkey"
	"github.com/smilemakc/mbflow/internal/application/trigger"
	"github.com/smilemakc/mbflow/internal/infrastructure/api/rest"
	"github.com/smilemakc/mbflow/internal/infrastructure/cache"
	"github.com/smilemakc/mbflow/internal/infrastructure/storage"
	"github.com/smilemakc/mbflow/pkg/crypto"
	"github.com/smilemakc/mbflow/pkg/executor"
	"github.com/smilemakc/mbflow/pkg/executor/builtin"
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

	s.db = db
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

	s.redisCache = redisCache
	s.logger.Info("Redis cache connected")
	return nil
}

func (s *Server) initExecutorManager() error {
	s.executorManager = executor.NewManager()

	if err := builtin.RegisterBuiltins(s.executorManager); err != nil {
		return fmt.Errorf("failed to register built-in executors: %w", err)
	}

	s.logger.Info("Registered executors", "types", s.executorManager.List())
	return nil
}

func (s *Server) initFileStorageManager() error {
	fileStorageConfig := filestorage.DefaultManagerConfig()
	fileStorageConfig.BasePath = s.config.FileStorage.StoragePath
	fileStorageConfig.MaxFileSize = s.config.FileStorage.MaxFileSize

	s.fileStorageManager = filestorage.NewStorageManager(fileStorageConfig)

	s.logger.Info("File storage manager initialized",
		"base_path", s.config.FileStorage.StoragePath,
		"max_file_size", s.config.FileStorage.MaxFileSize,
	)

	if err := builtin.RegisterFileStorage(s.executorManager, s.fileStorageManager); err != nil {
		return fmt.Errorf("failed to register file_storage executor: %w", err)
	}

	if err := builtin.RegisterAdapters(s.executorManager); err != nil {
		return fmt.Errorf("failed to register adapter executors: %w", err)
	}

	if err := builtin.RegisterFileAdapters(s.executorManager, s.fileStorageManager); err != nil {
		return fmt.Errorf("failed to register file adapter executors: %w", err)
	}

	return nil
}

func (s *Server) initObserverManager() error {
	if s.config.Observer.EnableWebSocket {
		s.wsHub = observer.NewWebSocketHub(s.logger)
		s.logger.Info("WebSocket hub initialized")
	}

	s.observerManager = observer.NewObserverManager(
		observer.WithLogger(s.logger),
		observer.WithBufferSize(s.config.Observer.BufferSize),
	)

	if s.config.Observer.EnableDatabase {
		dbObserver := observer.NewDatabaseObserver(s.eventRepo)
		if err := s.observerManager.Register(dbObserver); err != nil {
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
		if err := s.observerManager.Register(httpObserver); err != nil {
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
		if err := s.observerManager.Register(loggerObserver); err != nil {
			s.logger.Error("Failed to register logger observer", "error", err)
		} else {
			s.logger.Info("Logger observer registered")
		}
	}

	if s.config.Observer.EnableWebSocket && s.wsHub != nil {
		wsObserver := observer.NewWebSocketObserver(
			s.wsHub,
			observer.WithWebSocketLogger(s.logger),
		)
		if err := s.observerManager.Register(wsObserver); err != nil {
			s.logger.Error("Failed to register WebSocket observer", "error", err)
		} else {
			s.logger.Info("WebSocket observer registered")
		}
	}

	s.logger.Info("Observer system initialized",
		"observer_count", s.observerManager.Count(),
	)

	return nil
}

func (s *Server) initRepositories() error {
	s.workflowRepo = storage.NewWorkflowRepository(s.db)
	s.executionRepo = storage.NewExecutionRepository(s.db)
	s.eventRepo = storage.NewEventRepository(s.db)
	s.triggerRepo = storage.NewTriggerRepository(s.db)
	s.userRepo = storage.NewUserRepository(s.db)
	s.fileRepo = storage.NewFileRepository(s.db)
	s.accountRepo = storage.NewAccountRepository(s.db)
	s.transactionRepo = storage.NewTransactionRepository(s.db)
	s.resourceRepo = storage.NewResourceRepository(s.db)
	s.pricingPlanRepo = storage.NewPricingPlanRepository(s.db)
	s.credentialsRepo = storage.NewCredentialsRepository(s.db)
	s.serviceKeyRepo = storage.NewServiceKeyRepository(s.db)
	s.systemKeyRepo = storage.NewSystemKeyRepo(s.db)
	s.auditLogRepo = storage.NewServiceAuditLogRepo(s.db)

	s.logger.Info("Repositories initialized")
	return nil
}

func (s *Server) initEncryptionServices() error {
	encryptionService, err := crypto.GetDefaultService()
	if err != nil {
		return fmt.Errorf("encryption service not available: %w", err)
	}

	s.encryptionService = encryptionService
	s.logger.Info("Encryption service initialized")

	s.rentalKeyRepo = storage.NewRentalKeyRepository(s.db, encryptionService)
	s.rentalKeyProvider = rentalkey.NewProvider(s.rentalKeyRepo, encryptionService)

	s.logger.Info("Rental key provider initialized")
	return nil
}

func (s *Server) initAuthSystem() error {
	s.authService = auth.NewService(s.userRepo, s.accountRepo, &s.config.Auth)

	providerManager, err := auth.NewProviderManager(&s.config.Auth, s.authService)
	if err != nil {
		s.logger.Warn("Failed to initialize auth provider manager", "error", err)
	}
	s.providerManager = providerManager

	s.serviceKeyService = servicekey.NewService(s.serviceKeyRepo, servicekey.Config{
		MaxKeysPerUser:    s.config.ServiceKeys.MaxKeysPerUser,
		DefaultExpiryDays: s.config.ServiceKeys.DefaultExpiryDays,
	})

	s.authMiddleware = rest.NewAuthMiddleware(s.providerManager, s.authService, s.serviceKeyService)
	s.loginRateLimiter = rest.NewLoginRateLimiter(
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
	s.executionManager = engine.NewExecutionManager(
		s.executorManager,
		s.workflowRepo,
		s.executionRepo,
		s.eventRepo,
		s.resourceRepo,
		s.observerManager,
	)

	s.logger.Info("Execution engine initialized")
	return nil
}

func (s *Server) initTriggerManager() error {
	if s.redisCache == nil {
		return fmt.Errorf("trigger manager disabled - Redis cache not available")
	}

	triggerManager, err := trigger.NewManager(trigger.ManagerConfig{
		TriggerRepo:  s.triggerRepo,
		WorkflowRepo: s.workflowRepo,
		ExecutionMgr: s.executionManager,
		Cache:        s.redisCache,
	})
	if err != nil {
		return fmt.Errorf("failed to create trigger manager: %w", err)
	}

	s.triggerManager = triggerManager
	s.logger.Info("Trigger manager initialized")

	if err := s.triggerManager.Start(); err != nil {
		return fmt.Errorf("failed to start trigger manager: %w", err)
	}

	s.logger.Info("Trigger manager started")
	return nil
}

func (s *Server) initSystemKeySystem() error {
	s.systemKeyService_ = systemkey.NewService(s.systemKeyRepo, systemkey.Config{
		MaxKeys:           s.config.ServiceAPI.MaxKeys,
		DefaultExpiryDays: s.config.ServiceAPI.DefaultExpiryDays,
		BcryptCost:        s.config.ServiceAPI.BcryptCost,
	})
	s.auditService = systemkey.NewAuditService(s.auditLogRepo, s.config.ServiceAPI.AuditRetentionDays)
	s.systemAuthMiddleware = rest.NewSystemAuthMiddleware(s.systemKeyService_, s.userRepo, s.config.ServiceAPI.SystemUserID, s.logger)
	s.auditMiddleware = rest.NewAuditMiddleware(s.auditService, s.logger)
	s.logger.Info("System key system initialized")
	return nil
}
