package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/smilemakc/mbflow/internal/infrastructure/storage/models"
)

// WorkflowRepository defines the interface for workflow persistence
type WorkflowRepository interface {
	// Create creates a new workflow with its nodes and edges
	Create(ctx context.Context, workflow *models.WorkflowModel) error

	// Update updates an existing workflow
	Update(ctx context.Context, workflow *models.WorkflowModel) error

	// Delete soft-deletes a workflow
	Delete(ctx context.Context, id uuid.UUID) error

	// HardDelete permanently deletes a workflow
	HardDelete(ctx context.Context, id uuid.UUID) error

	// FindByID retrieves a workflow by ID
	FindByID(ctx context.Context, id uuid.UUID) (*models.WorkflowModel, error)

	// FindByIDWithRelations retrieves a workflow with all its relations (nodes, edges, triggers)
	FindByIDWithRelations(ctx context.Context, id uuid.UUID) (*models.WorkflowModel, error)

	// FindByName retrieves a workflow by name and version
	FindByName(ctx context.Context, name string, version int) (*models.WorkflowModel, error)

	// FindAll retrieves all workflows with pagination
	FindAll(ctx context.Context, limit, offset int) ([]*models.WorkflowModel, error)

	// FindByStatus retrieves workflows by status with pagination
	FindByStatus(ctx context.Context, status string, limit, offset int) ([]*models.WorkflowModel, error)

	// Count returns the total count of workflows
	Count(ctx context.Context) (int, error)

	// CountByStatus returns the count of workflows by status
	CountByStatus(ctx context.Context, status string) (int, error)

	// CreateNode creates a new node for a workflow
	CreateNode(ctx context.Context, node *models.NodeModel) error

	// UpdateNode updates an existing node
	UpdateNode(ctx context.Context, node *models.NodeModel) error

	// DeleteNode deletes a node
	DeleteNode(ctx context.Context, id uuid.UUID) error

	// FindNodeByID retrieves a node by ID
	FindNodeByID(ctx context.Context, id uuid.UUID) (*models.NodeModel, error)

	// FindNodesByWorkflowID retrieves all nodes for a workflow
	FindNodesByWorkflowID(ctx context.Context, workflowID uuid.UUID) ([]*models.NodeModel, error)

	// CreateEdge creates a new edge for a workflow
	CreateEdge(ctx context.Context, edge *models.EdgeModel) error

	// UpdateEdge updates an existing edge
	UpdateEdge(ctx context.Context, edge *models.EdgeModel) error

	// DeleteEdge deletes an edge
	DeleteEdge(ctx context.Context, id uuid.UUID) error

	// FindEdgeByID retrieves an edge by ID
	FindEdgeByID(ctx context.Context, id uuid.UUID) (*models.EdgeModel, error)

	// FindEdgesByWorkflowID retrieves all edges for a workflow
	FindEdgesByWorkflowID(ctx context.Context, workflowID uuid.UUID) ([]*models.EdgeModel, error)

	// ValidateDAG validates that the workflow forms a valid DAG (no cycles)
	ValidateDAG(ctx context.Context, workflowID uuid.UUID) error
}
