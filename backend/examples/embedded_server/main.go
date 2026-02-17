// Embedded server example for MBFlow
//
// This example demonstrates two ways to run MBFlow:
//
// 1. Full HTTP Server (using pkg/server):
//   - Complete REST API
//   - All features: auth, triggers, webhooks, websockets
//   - Use when you need the full MBFlow experience
//
// 2. SDK-only mode (using pkg/sdk):
//   - Programmatic workflow execution
//   - Custom HTTP endpoints
//   - Use when you need to embed MBFlow in your application
//
// Run: go run main.go [--full-server]
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/smilemakc/mbflow/pkg/models"
	"github.com/smilemakc/mbflow/pkg/sdk"
	"github.com/smilemakc/mbflow/pkg/server"
)

func main() {
	fullServer := flag.Bool("full-server", false, "Run full MBFlow HTTP server with REST API")
	flag.Parse()

	if *fullServer {
		runFullServer()
	} else {
		runSDKMode()
	}
}

// runFullServer demonstrates using pkg/server for a complete MBFlow server
func runFullServer() {
	log.Println("Starting MBFlow in full server mode...")

	// Create server with default configuration (loads from environment)
	srv, err := server.New()
	if err != nil {
		log.Fatalf("Failed to create server: %v", err)
	}

	// You can register custom executors
	// srv.RegisterExecutor("my-custom-type", myCustomExecutor)

	// You can add custom endpoints to the router
	// srv.Router().GET("/api/v1/custom", myCustomHandler)

	// Run blocks until shutdown signal
	if err := srv.Run(); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}

// runSDKMode demonstrates using pkg/sdk for embedded workflow execution
func runSDKMode() {
	log.Println("Starting MBFlow in SDK mode...")

	// Create embedded MBFlow client
	client, err := sdk.NewClient(
		sdk.WithEmbeddedMode(
			"postgres://mbflow:mbflow@localhost:5432/mbflow?sslmode=disable",
			"redis://localhost:6379",
		),
	)
	if err != nil {
		log.Fatalf("Failed to create MBFlow client: %v", err)
	}
	defer client.Close()

	// Create a simple workflow
	ctx := context.Background()
	workflow := &models.Workflow{
		Name:        "Embedded Workflow",
		Description: "A workflow running in embedded mode",
		Nodes: []*models.Node{
			{
				ID:   "fetch-data",
				Name: "Fetch Data",
				Type: "http",
				Config: map[string]any{
					"method": "GET",
					"url":    "https://api.github.com/users/github",
				},
			},
		},
	}

	createdWorkflow, err := client.Workflows().Create(ctx, workflow)
	if err != nil {
		log.Fatalf("Failed to create workflow: %v", err)
	}

	fmt.Printf("Created workflow: %s (ID: %s)\n", createdWorkflow.Name, createdWorkflow.ID)

	// Create HTTP server with workflow execution endpoint
	mux := http.NewServeMux()

	mux.HandleFunc("/execute", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		fmt.Println("Executing workflow...")

		execution, err := client.Executions().Run(r.Context(), createdWorkflow.ID, nil)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintf(w, `{"error":"%s"}`, err.Error())
			return
		}

		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"execution_id":"%s","status":"%s"}`, execution.ID, execution.Status)
	})

	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		status, err := client.Health(r.Context())
		if err != nil {
			w.WriteHeader(http.StatusServiceUnavailable)
			fmt.Fprintf(w, `{"status":"unhealthy","error":"%s"}`, err.Error())
			return
		}

		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"status":"%s"}`, status.Status)
	})

	httpServer := &http.Server{
		Addr:         ":8080",
		Handler:      mux,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
	}

	// Start server
	serverErrors := make(chan error, 1)
	go func() {
		log.Printf("Server starting on %s", httpServer.Addr)
		serverErrors <- httpServer.ListenAndServe()
	}()

	// Wait for interrupt signal
	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, os.Interrupt, syscall.SIGTERM)

	select {
	case err := <-serverErrors:
		log.Fatalf("Server error: %v", err)

	case sig := <-shutdown:
		log.Printf("Shutdown signal received: %v", sig)

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		if err := httpServer.Shutdown(ctx); err != nil {
			log.Printf("Graceful shutdown failed: %v", err)
		}

		log.Println("Server stopped")
	}
}
