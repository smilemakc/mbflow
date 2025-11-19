package main

import (
	"context"
	"errors"
	"mbflow/internal/infrastructure/api/rest"
	"mbflow/internal/infrastructure/config"
	"mbflow/internal/infrastructure/logger"
	"mbflow/internal/infrastructure/storage"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	// 1. Load Config
	cfg := config.Load()

	// 2. Setup Logger
	log := logger.Setup(cfg.LogLevel)
	log.Info("starting server", "port", cfg.Port)

	// 3. Wire Dependencies
	store := storage.NewMemoryStore()
	srv := rest.NewServer(store, log)

	// 4. Setup HTTP Server
	httpServer := &http.Server{
		Addr:    ":" + cfg.Port,
		Handler: srv,
	}

	// 5. Start Server in Goroutine
	go func() {
		if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Error("server failed", "error", err)
			os.Exit(1)
		}
	}()

	// 6. Graceful Shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Info("shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := httpServer.Shutdown(ctx); err != nil {
		log.Error("server forced to shutdown", "error", err)
	}

	log.Info("server exited")
}
