package mbflow

import (
	"context"

	"github.com/smilemakc/mbflow/internal/domain"
	"github.com/smilemakc/mbflow/internal/infrastructure/storage"

	"github.com/rs/zerolog/log"
)

// NewMemoryStorage creates a new in-memory storage.
// This storage is suitable for testing and development.
func NewMemoryStorage() Storage {
	return &storageAdapter{store: storage.NewMemoryStore()}
}

// NewPostgresStorage creates a new PostgreSQL-based storage.
// dsn - database connection string, for example:
// "postgres://user:password@localhost:5432/dbname?sslmode=disable"
func NewPostgresStorage(dsn string) Storage {
	bunStore := storage.NewBunStore(dsn)
	if err := bunStore.InitSchema(context.Background()); err != nil {
		log.Fatal().Err(err).Msg("Failed to initialize schema")
	}
	return &storageAdapter{store: bunStore}
}

// NewWorkflow creates a new workflow.
func NewWorkflow(id, name, version string, spec map[string]any) Workflow {
	return domain.NewWorkflow(id, name, version, spec)
}

// NewExecution creates a new workflow execution.
func NewExecution(id, workflowID string) Execution {
	return wrapExecution(domain.NewExecution(id, workflowID))
}

// NewNodeFromConfig creates a new node with UUID validation.
// Returns an error if ID or WorkflowID are not valid UUIDs.
func NewNodeFromConfig(cfg NodeConfig) (Node, error) {
	return domain.NewNode(domain.NodeConfig{
		ID:         cfg.ID,
		WorkflowID: cfg.WorkflowID,
		Type:       cfg.Type,
		Name:       cfg.Name,
		Config:     cfg.Config,
	})
}

// NewEdge creates a new edge between nodes.
func NewEdge(id, workflowID, fromNodeID, toNodeID, edgeType string, config map[string]any) Edge {
	return domain.NewEdge(id, workflowID, fromNodeID, toNodeID, edgeType, config)
}

// NewTrigger creates a new trigger.
func NewTrigger(id, workflowID, triggerType string, config map[string]any) Trigger {
	return domain.NewTrigger(id, workflowID, triggerType, config)
}

// NewEvent creates a new event.
func NewEvent(eventID, eventType, workflowID, executionID, workflowName, nodeID string, payload []byte, metadata map[string]string) Event {
	return domain.NewEvent(eventID, eventType, workflowID, executionID, workflowName, nodeID, payload, metadata)
}
