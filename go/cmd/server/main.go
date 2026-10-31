// MBFlow Server - Workflow orchestration engine
package main

import (
	"fmt"
	"log"

	"github.com/smilemakc/mbflow/go/pkg/server"
)

// runner is the part of *server.Server that main needs.
type runner interface {
	Run() error
}

// Process hooks, replaceable in tests.
var (
	newServer = server.New
	fatal     = log.Fatal
)

func main() {
	if err := run(newServer); err != nil {
		fatal(err)
	}
}

// run creates the server with newServer and runs it until it stops.
func run[R runner](newServer func(...server.Option) (R, error)) error {
	srv, err := newServer()
	if err != nil {
		return fmt.Errorf("failed to create server: %w", err)
	}
	if err := srv.Run(); err != nil {
		return fmt.Errorf("server error: %w", err)
	}
	return nil
}
