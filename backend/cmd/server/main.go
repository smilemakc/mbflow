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

	"github.com/smilemakc/mbflow/internal/application/observer"
	"github.com/smilemakc/mbflow/internal/config"
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

	// Register built-in executors
	if err := executorManager.Register("http", builtin.NewHTTPExecutor()); err != nil {
		appLogger.Error("Failed to register HTTP executor", "error", err)
		os.Exit(1)
	}

	if err := executorManager.Register("transform", builtin.NewTransformExecutor()); err != nil {
		appLogger.Error("Failed to register transform executor", "error", err)
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

	// Create HTTP server
	mux := http.NewServeMux()

	// WebSocket endpoints
	if cfg.Observer.EnableWebSocket && wsHub != nil {
		wsHandler := observer.NewWebSocketHandler(wsHub, appLogger)
		mux.HandleFunc("/ws/executions", wsHandler.ServeHTTP)
		mux.HandleFunc("/ws/health", wsHandler.HandleHealthCheck)
		appLogger.Info("WebSocket endpoints registered",
			"endpoints", []string{"/ws/executions", "/ws/health"},
		)
	}

	// Health check endpoint
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		// Check database health
		if err := storage.Ping(ctx, db); err != nil {
			w.WriteHeader(http.StatusServiceUnavailable)
			fmt.Fprintf(w, `{"status":"unhealthy","error":"database: %s"}`, err.Error())
			return
		}

		// Check Redis health (if configured)
		if redisCache != nil {
			if err := redisCache.Health(ctx); err != nil {
				w.WriteHeader(http.StatusServiceUnavailable)
				fmt.Fprintf(w, `{"status":"unhealthy","error":"redis: %s"}`, err.Error())
				return
			}
		}

		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `{"status":"healthy"}`)
	})

	// Readiness check endpoint
	mux.HandleFunc("/ready", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `{"status":"ready"}`)
	})

	// Metrics endpoint
	mux.HandleFunc("/metrics", func(w http.ResponseWriter, r *http.Request) {
		dbStats := storage.Stats(db)

		metrics := map[string]interface{}{
			"database": map[string]interface{}{
				"open_connections": dbStats.OpenConnections,
				"in_use":           dbStats.InUse,
				"idle":             dbStats.Idle,
				"max_open_conns":   dbStats.MaxOpenConnections,
			},
		}

		if redisCache != nil {
			cacheStats := redisCache.Stats()
			metrics["redis"] = map[string]interface{}{
				"hits":        cacheStats.Hits,
				"misses":      cacheStats.Misses,
				"total_conns": cacheStats.TotalConns,
				"idle_conns":  cacheStats.IdleConns,
			}
		}

		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"metrics":%v}`, metrics)
	})

	// API routes placeholder
	mux.HandleFunc("/api/v1/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"message":"MBFlow API v1","status":"ok"}`)
	})

	// Create HTTP server with timeouts
	server := &http.Server{
		Addr:         fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port),
		Handler:      mux,
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
