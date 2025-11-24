package main

import (
	"context"
	"errors"
	"flag"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/smilemakc/mbflow"
	"github.com/smilemakc/mbflow/internal/infrastructure/api/rest"
	"github.com/smilemakc/mbflow/internal/infrastructure/config"
	"github.com/smilemakc/mbflow/internal/infrastructure/logger"
	"github.com/smilemakc/mbflow/internal/infrastructure/storage"
)

func main() {
	// Parse command line flags
	var (
		port          = flag.String("port", "", "Server port (overrides config)")
		enableCORS    = flag.Bool("cors", true, "Enable CORS")
		enableMetrics = flag.Bool("metrics", true, "Enable metrics collection")
		apiKeys       = flag.String("api-keys", "", "Comma-separated API keys for authentication")
	)
	flag.Parse()

	// Load configuration
	cfg := config.Load()

	// Override port if provided via flag
	if *port != "" {
		cfg.Port = *port
	}

	// Setup logger
	log := logger.Setup(cfg.LogLevel)
	log.Info("starting mbflow rest api server",
		"version", "1.0.0",
		"port", cfg.Port,
		"cors", *enableCORS,
		"metrics", *enableMetrics,
	)

	// Create storage (in-memory for MVP, can be replaced with PostgreSQL)
	store := storage.NewMemoryStore()
	log.Info("using in-memory storage")

	// Create executor
	executorOpts := []mbflow.ExecutorOption{
		mbflow.WithEventStore(store),
	}

	executor := mbflow.NewExecutor(executorOpts...)
	log.Info("executor initialized")

	// Parse API keys
	var apiKeysList []string
	if *apiKeys != "" {
		// Simple split by comma
		for _, key := range parseAPIKeys(*apiKeys) {
			if key != "" {
				apiKeysList = append(apiKeysList, key)
			}
		}
		log.Info("api key authentication enabled", "count", len(apiKeysList))
	}

	// Create REST API server
	serverConfig := rest.ServerConfig{
		EnableCORS:      *enableCORS,
		EnableRateLimit: false,
		RateLimitMax:    100,
		RateLimitWindow: time.Minute,
		APIKeys:         apiKeysList,
	}
	srv := rest.NewServer(store, executor, log, serverConfig)

	// Setup HTTP server
	httpServer := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      srv,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server in goroutine
	go func() {
		log.Info("server listening", "address", httpServer.Addr)
		if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Error("server failed", "error", err)
			os.Exit(1)
		}
	}()

	// Print API endpoints
	log.Info("available endpoints",
		"health", "GET /health",
		"ready", "GET /ready",
		"workflows", "GET /api/v1/workflows",
		"create_workflow", "POST /api/v1/workflows",
		"executions", "GET /api/v1/executions",
		"execute_workflow", "POST /api/v1/executions",
	)

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info("shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := httpServer.Shutdown(ctx); err != nil {
		log.Error("server forced to shutdown", "error", err)
		os.Exit(1)
	}

	log.Info("server exited gracefully")
}

// parseAPIKeys parses comma-separated API keys
func parseAPIKeys(keys string) []string {
	result := []string{}
	current := ""
	for _, ch := range keys {
		if ch == ',' {
			if current != "" {
				result = append(result, current)
				current = ""
			}
		} else {
			current += string(ch)
		}
	}
	if current != "" {
		result = append(result, current)
	}
	return result
}
