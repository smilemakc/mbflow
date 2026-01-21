// MBFlow Server - Workflow orchestration engine
package main

import (
	"log"

	"github.com/smilemakc/mbflow/pkg/server"
)

func main() {
	srv, err := server.New()
	if err != nil {
		log.Fatalf("Failed to create server: %v", err)
	}

	if err := srv.Run(); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}
