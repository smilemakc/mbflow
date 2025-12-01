// Embedded server example for MBFlow SDK
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

	"github.com/smilemakc/mbflow/pkg/models"
	"github.com/smilemakc/mbflow/pkg/sdk"
)

func main() {
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
				Config: map[string]interface{}{
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

	server := &http.Server{
		Addr:         ":8080",
		Handler:      mux,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
	}

	// Start server
	serverErrors := make(chan error, 1)
	go func() {
		log.Printf("Server starting on %s", server.Addr)
		serverErrors <- server.ListenAndServe()
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

		if err := server.Shutdown(ctx); err != nil {
			log.Printf("Graceful shutdown failed: %v", err)
		}

		log.Println("Server stopped")
	}
}
