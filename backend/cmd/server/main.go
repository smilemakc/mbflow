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
	"github.com/smilemakc/mbflow/internal/application/engine"
	"github.com/smilemakc/mbflow/internal/application/observer"
	"github.com/smilemakc/mbflow/internal/application/trigger"
	"github.com/smilemakc/mbflow/internal/config"
	"github.com/smilemakc/mbflow/internal/infrastructure/api/rest"
	"github.com/smilemakc/mbflow/internal/infrastructure/cache"
	"github.com/smilemakc/mbflow/internal/infrastructure/logger"
	"github.com/smilemakc/mbflow/internal/infrastructure/storage"
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

	// Register all built-in executors (http, transform, llm, function_call)
	if err := builtin.RegisterBuiltins(executorManager); err != nil {
		appLogger.Error("Failed to register built-in executors", "error", err)
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

	appLogger.Info("Repositories initialized")

	// Initialize execution engine
	executionManager := engine.NewExecutionManager(
		executorManager,
		workflowRepo,
		executionRepo,
		eventRepo,
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

	// Add middleware
	router.Use(gin.Recovery())
	router.Use(func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		raw := c.Request.URL.RawQuery

		c.Next()

		latency := time.Since(start)
		statusCode := c.Writer.Status()
		method := c.Request.Method

		if raw != "" {
			path = path + "?" + raw
		}

		appLogger.Info("HTTP request",
			"method", method,
			"path", path,
			"status", statusCode,
			"latency", latency,
			"ip", c.ClientIP(),
		)
	})

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
		workflowHandlers := rest.NewWorkflowHandlers(workflowRepo, appLogger)
		nodeHandlers := rest.NewNodeHandlers(workflowRepo, appLogger)
		edgeHandlers := rest.NewEdgeHandlers(workflowRepo, appLogger)
		executionHandlers := rest.NewExecutionHandlers(executionRepo, workflowRepo, executionManager)
		triggerHandlers := rest.NewTriggerHandlers(triggerRepo, workflowRepo)

		// Workflow endpoints
		workflows := apiV1.Group("/workflows")
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

		// Webhook endpoints
		if triggerManager != nil {
			webhookHandlers := rest.NewWebhookHandlers(triggerManager.WebhookRegistry())
			apiV1.POST("/webhooks/:path", webhookHandlers.HandleWebhook)
			apiV1.GET("/webhooks/:path", webhookHandlers.HandleWebhookGet)
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
