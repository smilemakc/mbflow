// MBFlow Server - Workflow orchestration engine
package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/smilemakc/mbflow/internal/application/auth"
	"github.com/smilemakc/mbflow/internal/application/engine"
	"github.com/smilemakc/mbflow/internal/application/filestorage"
	"github.com/smilemakc/mbflow/internal/application/observer"
	"github.com/smilemakc/mbflow/internal/application/rentalkey"
	"github.com/smilemakc/mbflow/internal/application/trigger"
	"github.com/smilemakc/mbflow/internal/config"
	"github.com/smilemakc/mbflow/internal/infrastructure/api/rest"
	"github.com/smilemakc/mbflow/internal/infrastructure/cache"
	"github.com/smilemakc/mbflow/internal/infrastructure/logger"
	"github.com/smilemakc/mbflow/internal/infrastructure/storage"
	"github.com/smilemakc/mbflow/pkg/crypto"
	"github.com/smilemakc/mbflow/pkg/executor"
	"github.com/smilemakc/mbflow/pkg/executor/builtin"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Initialize logger
	appLogger := logger.New(cfg.Logging)
	logger.SetDefault(appLogger)

	appLogger.Info("Starting MBFlow Server",
		"version", "1.0.0",
		"port", cfg.Server.Port,
	)

	// Initialize database
	dbConfig := &storage.Config{
		DSN:             cfg.Database.URL,
		MaxOpenConns:    cfg.Database.MaxConnections,
		MaxIdleConns:    cfg.Database.MinConnections,
		ConnMaxLifetime: cfg.Database.MaxConnLifetime,
		ConnMaxIdleTime: cfg.Database.MaxIdleTime,
		Debug:           cfg.Logging.Level == "debug",
	}

	db, err := storage.NewDB(dbConfig)
	if err != nil {
		appLogger.Error("Failed to initialize database", "error", err)
		os.Exit(1)
	}
	defer storage.Close(db)

	appLogger.Info("Database connected",
		"max_conns", cfg.Database.MaxConnections,
	)

	// Initialize Redis cache
	redisCache, err := cache.NewRedisCache(cfg.Redis)
	if err != nil {
		appLogger.Warn("Failed to initialize Redis cache", "error", err)
		// Continue without Redis - it's optional
		redisCache = nil
	} else {
		defer redisCache.Close()
		appLogger.Info("Redis cache connected")
	}

	// Initialize executor registry
	executorManager := executor.NewManager()

	// Register all built-in executors (http, transform, llm, function_call, telegram, conditional, merge)
	if err := builtin.RegisterBuiltins(executorManager); err != nil {
		appLogger.Error("Failed to register built-in executors", "error", err)
		os.Exit(1)
	}

	// Initialize file storage manager
	fileStorageConfig := filestorage.DefaultManagerConfig()
	fileStorageConfig.BasePath = cfg.FileStorage.StoragePath
	fileStorageConfig.MaxFileSize = cfg.FileStorage.MaxFileSize
	fileStorageManager := filestorage.NewStorageManager(fileStorageConfig)
	appLogger.Info("File storage manager initialized",
		"base_path", cfg.FileStorage.StoragePath,
		"max_file_size", cfg.FileStorage.MaxFileSize,
	)

	// Register file_storage executor
	if err := builtin.RegisterFileStorage(executorManager, fileStorageManager); err != nil {
		appLogger.Error("Failed to register file_storage executor", "error", err)
		os.Exit(1)
	}

	if err := builtin.RegisterAdapters(executorManager); err != nil {
		appLogger.Error("Failed to register adapter executors", "error", err)
		os.Exit(1)
	}

	if err := builtin.RegisterFileAdapters(executorManager, fileStorageManager); err != nil {
		appLogger.Error("Failed to register file adapter executors", "error", err)
		os.Exit(1)
	}

	appLogger.Info("Registered executors", "types", executorManager.List())

	// Initialize WebSocket hub (if enabled)
	var wsHub *observer.WebSocketHub
	if cfg.Observer.EnableWebSocket {
		wsHub = observer.NewWebSocketHub(appLogger)
		appLogger.Info("WebSocket hub initialized")
	}

	// Initialize observer manager
	observerManager := observer.NewObserverManager(
		observer.WithLogger(appLogger),
		observer.WithBufferSize(cfg.Observer.BufferSize),
	)

	// Register observers based on configuration
	if cfg.Observer.EnableDatabase {
		eventRepo := storage.NewEventRepository(db)
		dbObserver := observer.NewDatabaseObserver(eventRepo)
		if err := observerManager.Register(dbObserver); err != nil {
			appLogger.Error("Failed to register database observer", "error", err)
		} else {
			appLogger.Info("Database observer registered")
		}
	}

	if cfg.Observer.EnableHTTP && cfg.Observer.HTTPCallbackURL != "" {
		httpObserver := observer.NewHTTPCallbackObserver(
			cfg.Observer.HTTPCallbackURL,
			observer.WithHTTPMethod(cfg.Observer.HTTPMethod),
			observer.WithHTTPHeaders(cfg.Observer.HTTPHeaders),
			observer.WithHTTPTimeout(cfg.Observer.HTTPTimeout),
			observer.WithHTTPRetry(
				cfg.Observer.HTTPMaxRetries,
				cfg.Observer.HTTPRetryDelay,
				2.0, // backoff multiplier
			),
		)
		if err := observerManager.Register(httpObserver); err != nil {
			appLogger.Error("Failed to register HTTP observer", "error", err)
		} else {
			appLogger.Info("HTTP callback observer registered",
				"url", cfg.Observer.HTTPCallbackURL,
				"method", cfg.Observer.HTTPMethod,
			)
		}
	}

	if cfg.Observer.EnableLogger {
		loggerObserver := observer.NewLoggerObserver(
			observer.WithLoggerInstance(appLogger),
		)
		if err := observerManager.Register(loggerObserver); err != nil {
			appLogger.Error("Failed to register logger observer", "error", err)
		} else {
			appLogger.Info("Logger observer registered")
		}
	}

	if cfg.Observer.EnableWebSocket && wsHub != nil {
		wsObserver := observer.NewWebSocketObserver(
			wsHub,
			observer.WithWebSocketLogger(appLogger),
		)
		if err := observerManager.Register(wsObserver); err != nil {
			appLogger.Error("Failed to register WebSocket observer", "error", err)
		} else {
			appLogger.Info("WebSocket observer registered")
		}
	}

	appLogger.Info("Observer system initialized",
		"observer_count", observerManager.Count(),
	)

	// Initialize repositories
	workflowRepo := storage.NewWorkflowRepository(db)
	executionRepo := storage.NewExecutionRepository(db)
	eventRepo := storage.NewEventRepository(db)
	triggerRepo := storage.NewTriggerRepository(db)
	userRepo := storage.NewUserRepository(db)
	fileRepo := storage.NewFileRepository(db)
	accountRepo := storage.NewAccountRepository(db)
	transactionRepo := storage.NewTransactionRepository(db)
	resourceRepo := storage.NewResourceRepository(db)
	pricingPlanRepo := storage.NewPricingPlanRepository(db)
	credentialsRepo := storage.NewCredentialsRepository(db)

	appLogger.Info("Repositories initialized")

	// Initialize encryption service for credentials and rental keys
	encryptionService, err := crypto.GetDefaultService()
	if err != nil {
		appLogger.Warn("Encryption service not available - credentials and rental keys features disabled", "error", err)
		encryptionService = nil
	} else {
		appLogger.Info("Encryption service initialized")
	}

	// Initialize rental key repository and provider (requires encryption service)
	var rentalKeyRepo *storage.RentalKeyRepositoryImpl
	var rentalKeyProvider *rentalkey.Provider
	if encryptionService != nil {
		rentalKeyRepo = storage.NewRentalKeyRepository(db, encryptionService)
		rentalKeyProvider = rentalkey.NewProvider(rentalKeyRepo, encryptionService)
		appLogger.Info("Rental key provider initialized")
	}

	// Initialize auth system
	authService := auth.NewService(userRepo, accountRepo, &cfg.Auth)
	providerManager, err := auth.NewProviderManager(&cfg.Auth, authService)
	if err != nil {
		appLogger.Warn("Failed to initialize auth provider manager", "error", err)
		// Continue with builtin provider only
	}

	authMiddleware := rest.NewAuthMiddleware(providerManager, authService)
	loginRateLimiter := rest.NewLoginRateLimiter(
		cfg.Auth.MaxLoginAttempts,
		time.Duration(cfg.Auth.MaxLoginAttempts)*time.Minute,
		cfg.Auth.LockoutDuration,
	)

	appLogger.Info("Auth system initialized",
		"mode", cfg.Auth.Mode,
		"registration_enabled", cfg.Auth.AllowRegistration,
	)

	// Initialize execution engine
	executionManager := engine.NewExecutionManager(
		executorManager,
		workflowRepo,
		executionRepo,
		eventRepo,
		resourceRepo,
		observerManager,
	)

	appLogger.Info("Execution engine initialized")

	// Initialize trigger manager (only if Redis is available)
	var triggerManager *trigger.Manager
	if redisCache != nil {
		triggerManager, err = trigger.NewManager(trigger.ManagerConfig{
			TriggerRepo:  triggerRepo,
			WorkflowRepo: workflowRepo,
			ExecutionMgr: executionManager,
			Cache:        redisCache,
		})
		if err != nil {
			appLogger.Error("Failed to initialize trigger manager", "error", err)
		} else {
			appLogger.Info("Trigger manager initialized")
			// Start trigger manager
			if err := triggerManager.Start(); err != nil {
				appLogger.Error("Failed to start trigger manager", "error", err)
			} else {
				appLogger.Info("Trigger manager started")
			}
		}
	} else {
		appLogger.Warn("Trigger manager disabled - Redis cache not available")
	}

	// Set Gin mode based on log level
	if cfg.Logging.Level == "debug" {
		gin.SetMode(gin.DebugMode)
	} else {
		gin.SetMode(gin.ReleaseMode)
	}

	// Create Gin router
	router := gin.New()

	// Initialize middleware
	loggingMiddleware := rest.NewLoggingMiddleware(appLogger)
	recoveryMiddleware := rest.NewRecoveryMiddleware(appLogger)

	// Add middleware in correct order:
	// 1. Recovery (catches panics)
	// 2. Logging (logs all requests with request_id)
	router.Use(recoveryMiddleware.Recovery())
	router.Use(loggingMiddleware.RequestLogger())

	// CORS middleware (if enabled)
	if cfg.Server.CORS {
		router.Use(func(c *gin.Context) {
			c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
			c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, PATCH, OPTIONS")
			c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-API-Key")
			c.Writer.Header().Set("Access-Control-Max-Age", "86400")

			if c.Request.Method == "OPTIONS" {
				c.AbortWithStatus(http.StatusNoContent)
				return
			}

			c.Next()
		})
		appLogger.Info("CORS enabled")
	}

	// Health check endpoints
	router.GET("/health", func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
		defer cancel()

		// Check database health
		if err := storage.Ping(ctx, db); err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"status": "unhealthy",
				"error":  fmt.Sprintf("database: %s", err.Error()),
			})
			return
		}

		// Check Redis health (if configured)
		if redisCache != nil {
			if err := redisCache.Health(ctx); err != nil {
				c.JSON(http.StatusServiceUnavailable, gin.H{
					"status": "unhealthy",
					"error":  fmt.Sprintf("redis: %s", err.Error()),
				})
				return
			}
		}

		c.JSON(http.StatusOK, gin.H{"status": "healthy"})
	})

	router.GET("/ready", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ready"})
	})

	// Metrics endpoint
	router.GET("/metrics", func(c *gin.Context) {
		dbStats := storage.Stats(db)

		metrics := gin.H{
			"database": gin.H{
				"open_connections": dbStats.OpenConnections,
				"in_use":           dbStats.InUse,
				"idle":             dbStats.Idle,
				"max_open_conns":   dbStats.MaxOpenConnections,
			},
		}

		if redisCache != nil {
			cacheStats := redisCache.Stats()
			metrics["redis"] = gin.H{
				"hits":        cacheStats.Hits,
				"misses":      cacheStats.Misses,
				"total_conns": cacheStats.TotalConns,
				"idle_conns":  cacheStats.IdleConns,
			}
		}

		c.JSON(http.StatusOK, gin.H{"metrics": metrics})
	})

	// WebSocket endpoints
	if cfg.Observer.EnableWebSocket && wsHub != nil {
		wsHandler := observer.NewWebSocketHandler(wsHub, appLogger)
		router.GET("/ws/executions", func(c *gin.Context) {
			wsHandler.ServeHTTP(c.Writer, c.Request)
		})
		router.GET("/ws/health", func(c *gin.Context) {
			wsHandler.HandleHealthCheck(c.Writer, c.Request)
		})
		appLogger.Info("WebSocket endpoints registered",
			"endpoints", []string{"/ws/executions", "/ws/health"},
		)
	}

	// API v1 routes
	apiV1 := router.Group("/api/v1")
	{
		// Initialize handlers
		workflowHandlers := rest.NewWorkflowHandlers(workflowRepo, appLogger, executorManager)
		nodeHandlers := rest.NewNodeHandlers(workflowRepo, appLogger)
		edgeHandlers := rest.NewEdgeHandlers(workflowRepo, appLogger)
		executionHandlers := rest.NewExecutionHandlers(executionRepo, workflowRepo, executionManager, appLogger)
		triggerHandlers := rest.NewTriggerHandlers(triggerRepo, workflowRepo, appLogger)
		authHandlers := rest.NewAuthHandlers(authService, providerManager, loginRateLimiter)
		fileHandlers := rest.NewFileHandlers(fileRepo, fileStorageManager, appLogger)
		resourceHandlers := rest.NewResourceHandlers(resourceRepo, pricingPlanRepo, workflowRepo, appLogger)
		accountHandlers := rest.NewAccountHandlers(accountRepo, transactionRepo, appLogger)

		// Initialize resource file service and handlers
		resourceFileService := filestorage.NewResourceFileService(
			db,
			resourceRepo,
			fileRepo,
			fileStorageManager,
			cfg.FileStorage.MaxFileSize,
		)
		fileStorageHandlers := rest.NewFileStorageHandlers(resourceRepo, resourceFileService, appLogger)

		// Auth endpoints (public)
		authGroup := apiV1.Group("/auth")
		{
			authGroup.POST("/register", authHandlers.HandleRegister)
			authGroup.POST("/login", loginRateLimiter.Middleware(), authHandlers.HandleLogin)
			authGroup.POST("/refresh", authHandlers.HandleRefresh)
			authGroup.GET("/info", authHandlers.HandleGetAuthInfo)

			// OAuth endpoints
			authGroup.GET("/oauth/authorize", authHandlers.HandleOAuthAuthorize)
			authGroup.GET("/oauth/callback", authHandlers.HandleOAuthCallback)

			// Protected auth endpoints
			authGroup.POST("/logout", authMiddleware.RequireAuth(), authHandlers.HandleLogout)
			authGroup.GET("/me", authMiddleware.RequireAuth(), authHandlers.HandleGetMe)
			authGroup.POST("/password", authMiddleware.RequireAuth(), authHandlers.HandleChangePassword)
		}

		// Admin endpoints (requires admin role)
		adminGroup := apiV1.Group("/admin")
		adminGroup.Use(authMiddleware.RequireAdmin())
		{
			// User management
			adminGroup.GET("/users", authHandlers.HandleAdminListUsers)
			adminGroup.POST("/users", authHandlers.HandleAdminCreateUser)
			adminGroup.GET("/users/:id", authHandlers.HandleAdminGetUser)
			adminGroup.PUT("/users/:id", authHandlers.HandleAdminUpdateUser)
			adminGroup.DELETE("/users/:id", authHandlers.HandleAdminDeleteUser)
			adminGroup.POST("/users/:id/reset-password", authHandlers.HandleAdminResetPassword)

			// Role management
			adminGroup.GET("/roles", authHandlers.HandleListRoles)
			adminGroup.GET("/users/:id/roles", authHandlers.HandleGetUserRoles)
			adminGroup.POST("/users/:id/roles", authHandlers.HandleAssignRole)
			adminGroup.DELETE("/users/:id/roles/:role_id", authHandlers.HandleRemoveRole)
		}

		appLogger.Info("Auth endpoints registered")

		// Workflow endpoints (with optional auth for ownership tracking)
		workflows := apiV1.Group("/workflows")
		workflows.Use(authMiddleware.OptionalAuth())
		{
			workflows.POST("", workflowHandlers.HandleCreateWorkflow)
			workflows.GET("", workflowHandlers.HandleListWorkflows)
			workflows.GET("/:workflow_id", workflowHandlers.HandleGetWorkflow)
			workflows.PUT("/:workflow_id", workflowHandlers.HandleUpdateWorkflow)
			workflows.POST("/:workflow_id/execute", executionHandlers.HandleRunExecution)
			workflows.DELETE("/:workflow_id", workflowHandlers.HandleDeleteWorkflow)
			workflows.POST("/:workflow_id/publish", workflowHandlers.HandlePublishWorkflow)
			workflows.POST("/:workflow_id/unpublish", workflowHandlers.HandleUnpublishWorkflow)
			workflows.GET("/:workflow_id/diagram", workflowHandlers.HandleGetWorkflowDiagram)

			// Workflow resources endpoints
			workflows.POST("/:workflow_id/resources", workflowHandlers.AttachWorkflowResource)
			workflows.GET("/:workflow_id/resources", workflowHandlers.GetWorkflowResources)
			workflows.PUT("/:workflow_id/resources/:resource_id", workflowHandlers.UpdateWorkflowResourceAlias)
			workflows.DELETE("/:workflow_id/resources/:resource_id", workflowHandlers.DetachWorkflowResource)

			// Node endpoints
			workflows.POST("/:workflow_id/nodes", nodeHandlers.HandleAddNode)
			workflows.GET("/:workflow_id/nodes", nodeHandlers.HandleListNodes)
			workflows.GET("/:workflow_id/nodes/:node_id", nodeHandlers.HandleGetNode)
			workflows.PUT("/:workflow_id/nodes/:node_id", nodeHandlers.HandleUpdateNode)
			workflows.DELETE("/:workflow_id/nodes/:node_id", nodeHandlers.HandleDeleteNode)

			// Edge endpoints
			workflows.POST("/:workflow_id/edges", edgeHandlers.HandleAddEdge)
			workflows.GET("/:workflow_id/edges", edgeHandlers.HandleListEdges)
			workflows.GET("/:workflow_id/edges/:edge_id", edgeHandlers.HandleGetEdge)
			workflows.PUT("/:workflow_id/edges/:edge_id", edgeHandlers.HandleUpdateEdge)
			workflows.DELETE("/:workflow_id/edges/:edge_id", edgeHandlers.HandleDeleteEdge)
		}

		// Execution endpoints
		executions := apiV1.Group("/executions")
		{
			executions.POST("/run/:workflow_id", executionHandlers.HandleRunExecution)
			executions.GET("", executionHandlers.HandleListExecutions)
			executions.GET("/:id", executionHandlers.HandleGetExecution)
			executions.GET("/:id/logs", executionHandlers.HandleGetLogs)
			executions.GET("/:id/nodes/:node_id/result", executionHandlers.HandleGetNodeResult)
			executions.POST("/:id/cancel", executionHandlers.HandleCancelExecution)
			executions.POST("/:id/retry", executionHandlers.HandleRetryExecution)
			executions.GET("/:id/watch", executionHandlers.HandleWatchExecution)
			executions.GET("/:id/stream", executionHandlers.HandleStreamLogs)
		}

		// Trigger endpoints
		triggers := apiV1.Group("/triggers")
		{
			triggers.POST("", triggerHandlers.HandleCreateTrigger)
			triggers.GET("", triggerHandlers.HandleListTriggers)
			triggers.GET("/:id", triggerHandlers.HandleGetTrigger)
			triggers.PUT("/:id", triggerHandlers.HandleUpdateTrigger)
			triggers.DELETE("/:id", triggerHandlers.HandleDeleteTrigger)
			triggers.POST("/:id/enable", triggerHandlers.HandleEnableTrigger)
			triggers.POST("/:id/disable", triggerHandlers.HandleDisableTrigger)
			triggers.POST("/:id/execute", triggerHandlers.HandleTriggerManual)
		}

		// File endpoints
		files := apiV1.Group("/files")
		files.Use(authMiddleware.OptionalAuth())
		{
			files.POST("", fileHandlers.HandleUploadFile)
			files.GET("", fileHandlers.HandleListFiles)
			files.GET("/:id", fileHandlers.HandleGetFile)
			files.GET("/:id/metadata", fileHandlers.HandleGetFileMetadata)
			files.DELETE("/:id", fileHandlers.HandleDeleteFile)
			files.GET("/storage/:storage_id/usage", fileHandlers.HandleGetStorageUsage)
		}

		// Resource endpoints (require authentication)
		resources := apiV1.Group("/resources")
		resources.Use(authMiddleware.RequireAuth())
		{
			resources.POST("/file-storage", resourceHandlers.CreateFileStorage)
			resources.GET("", resourceHandlers.ListResources)
			resources.GET("/:id", resourceHandlers.GetResource)
			resources.PUT("/:id", resourceHandlers.UpdateResource)
			resources.DELETE("/:id", resourceHandlers.DeleteResource)
			resources.GET("/pricing-plans", resourceHandlers.ListPricingPlans)

			// File storage resource endpoints
			resources.POST("/:id/files", fileStorageHandlers.UploadFile)
			resources.GET("/:id/files", fileStorageHandlers.ListFiles)
			resources.GET("/:id/files/:file_id", fileStorageHandlers.GetFileMetadata)
			resources.GET("/:id/files/:file_id/download", fileStorageHandlers.DownloadFile)
			resources.DELETE("/:id/files/:file_id", fileStorageHandlers.DeleteFile)
		}

		// Account endpoints (require authentication)
		account := apiV1.Group("/account")
		account.Use(authMiddleware.RequireAuth())
		{
			account.GET("", accountHandlers.GetAccount)
			account.POST("/deposit", accountHandlers.Deposit)
			account.GET("/transactions", accountHandlers.ListTransactions)
			account.GET("/transactions/:id", accountHandlers.GetTransaction)
		}

		// Credentials endpoints (require authentication and encryption service)
		if encryptionService != nil {
			credentialsHandlers := rest.NewCredentialsHandlers(credentialsRepo, workflowRepo, encryptionService, appLogger)

			credentials := apiV1.Group("/credentials")
			credentials.Use(authMiddleware.RequireAuth())
			{
				// Create endpoints for different credential types
				credentials.POST("/api-key", credentialsHandlers.CreateAPIKey)
				credentials.POST("/basic-auth", credentialsHandlers.CreateBasicAuth)
				credentials.POST("/oauth2", credentialsHandlers.CreateOAuth2)
				credentials.POST("/service-account", credentialsHandlers.CreateServiceAccount)
				credentials.POST("/custom", credentialsHandlers.CreateCustom)

				// CRUD endpoints
				credentials.GET("", credentialsHandlers.ListCredentials)
				credentials.GET("/:id", credentialsHandlers.GetCredential)
				credentials.GET("/:id/secrets", credentialsHandlers.GetCredentialSecrets)
				credentials.PUT("/:id", credentialsHandlers.UpdateCredential)
				credentials.DELETE("/:id", credentialsHandlers.DeleteCredential)
			}

			appLogger.Info("Credentials endpoints registered")
		} else {
			appLogger.Warn("Credentials endpoints disabled - encryption key not configured")
		}

		// Rental Keys endpoints (require authentication and encryption service)
		if rentalKeyProvider != nil {
			rentalKeyHandlers := rest.NewRentalKeyHandlers(rentalKeyProvider, appLogger)
			rentalKeyAdminHandlers := rest.NewRentalKeyAdminHandlers(rentalkey.NewAdminService(rentalKeyRepo, encryptionService), appLogger)

			// User endpoints - can only view their own rental keys (never see the actual key value)
			rentalKeys := apiV1.Group("/rental-keys")
			rentalKeys.Use(authMiddleware.RequireAuth())
			{
				rentalKeys.GET("", rentalKeyHandlers.ListRentalKeys)
				rentalKeys.GET("/:id", rentalKeyHandlers.GetRentalKey)
				rentalKeys.GET("/:id/usage", rentalKeyHandlers.GetRentalKeyUsage)
				rentalKeys.GET("/:id/summary", rentalKeyHandlers.GetRentalKeyUsageSummary)
			}

			// Admin endpoints - full CRUD access to all rental keys
			adminRentalKeys := apiV1.Group("/admin/rental-keys")
			adminRentalKeys.Use(authMiddleware.RequireAuth())
			adminRentalKeys.Use(authMiddleware.RequireAdmin())
			{
				adminRentalKeys.POST("", rentalKeyAdminHandlers.CreateRentalKey)
				adminRentalKeys.GET("", rentalKeyAdminHandlers.ListAllRentalKeys)
				adminRentalKeys.GET("/:id", rentalKeyAdminHandlers.GetRentalKey)
				adminRentalKeys.PUT("/:id", rentalKeyAdminHandlers.UpdateRentalKey)
				adminRentalKeys.DELETE("/:id", rentalKeyAdminHandlers.DeleteRentalKey)
				adminRentalKeys.POST("/:id/rotate-key", rentalKeyAdminHandlers.RotateRentalKeyAPIKey)
				adminRentalKeys.POST("/reset-daily", rentalKeyAdminHandlers.ResetDailyUsage)
				adminRentalKeys.POST("/reset-monthly", rentalKeyAdminHandlers.ResetMonthlyUsage)
			}

			appLogger.Info("Rental Keys endpoints registered")
		} else {
			appLogger.Warn("Rental Keys endpoints disabled - encryption key not configured")
		}

		// Webhook endpoints
		if triggerManager != nil {
			webhookHandlers := rest.NewWebhookHandlers(triggerManager.WebhookRegistry(), appLogger)
			apiV1.POST("/webhooks/:path", webhookHandlers.HandleWebhook)
			apiV1.GET("/webhooks/:path", webhookHandlers.HandleWebhookGet)

			// Telegram webhook endpoints
			telegramWebhookHandlers := rest.NewTelegramWebhookHandlers(triggerManager.WebhookRegistry(), appLogger)
			apiV1.POST("/webhooks/telegram/:trigger_id", telegramWebhookHandlers.HandleTelegramWebhook)

			appLogger.Info("Webhook endpoints registered",
				"endpoints", []string{"/api/v1/webhooks/:path", "/api/v1/webhooks/telegram/:trigger_id"},
			)
		}
	}

	appLogger.Info("REST API routes registered")

	// Create HTTP server with timeouts
	server := &http.Server{
		Addr:         fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port),
		Handler:      router,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
		IdleTimeout:  120 * time.Second,
	}

	// Start server in a goroutine
	serverErrors := make(chan error, 1)
	go func() {
		appLogger.Info("HTTP server starting",
			"host", cfg.Server.Host,
			"port", cfg.Server.Port,
		)
		serverErrors <- server.ListenAndServe()
	}()

	// Wait for interrupt signal
	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, os.Interrupt, syscall.SIGTERM)

	// Block until we receive a signal or server error
	select {
	case err := <-serverErrors:
		appLogger.Error("Server error", "error", err)
		os.Exit(1)

	case sig := <-shutdown:
		appLogger.Info("Server shutdown initiated", "signal", sig)

		// Create context with timeout for shutdown
		ctx, cancel := context.WithTimeout(context.Background(), cfg.Server.ShutdownTimeout)
		defer cancel()

		// Stop trigger manager first
		if triggerManager != nil {
			appLogger.Info("Stopping trigger manager...")
			if err := triggerManager.Stop(); err != nil {
				appLogger.Error("Trigger manager shutdown failed", "error", err)
			} else {
				appLogger.Info("Trigger manager stopped")
			}
		}

		// Close file storage manager
		appLogger.Info("Closing file storage manager...")
		if err := fileStorageManager.Close(); err != nil {
			appLogger.Error("File storage manager shutdown failed", "error", err)
		} else {
			appLogger.Info("File storage manager closed")
		}

		// Note: WebSocket hub cleanup happens automatically when server stops
		// as clients will be disconnected when the server shuts down

		// Gracefully shutdown the server
		if err := server.Shutdown(ctx); err != nil {
			appLogger.Error("Graceful shutdown failed", "error", err)
			if err := server.Close(); err != nil {
				appLogger.Error("Server close failed", "error", err)
			}
		}

		appLogger.Info("Server stopped")
	}
}
