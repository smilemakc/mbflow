package repository

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/smilemakc/mbflow/go/internal/infrastructure/storage/models"
)

// ExecutionRepository defines the interface for execution persistence
type ExecutionRepository interface {
	// Create creates a new execution
	Create(ctx context.Context, execution *models.ExecutionModel) error

	// Update updates an existing execution
	Update(ctx context.Context, execution *models.ExecutionModel) error

	// Delete deletes an execution
	Delete(ctx context.Context, id uuid.UUID) error

	// FindByID retrieves an execution by ID
	FindByID(ctx context.Context, id uuid.UUID) (*models.ExecutionModel, error)

	// FindByIDWithRelations retrieves an execution with all its node executions
	FindByIDWithRelations(ctx context.Context, id uuid.UUID) (*models.ExecutionModel, error)

	// FindByWorkflowID retrieves executions for a workflow with pagination
	FindByWorkflowID(ctx context.Context, workflowID uuid.UUID, limit, offset int) ([]*models.ExecutionModel, error)

	// FindByStatus retrieves executions by status with pagination
	FindByStatus(ctx context.Context, status string, limit, offset int) ([]*models.ExecutionModel, error)

	// FindAll retrieves all executions with pagination
	FindAll(ctx context.Context, limit, offset int) ([]*models.ExecutionModel, error)

	// FindRunning retrieves all running executions
	FindRunning(ctx context.Context) ([]*models.ExecutionModel, error)

	// Count returns the total count of executions
	Count(ctx context.Context) (int, error)

	// CountByWorkflowID returns the count of executions for a workflow
	CountByWorkflowID(ctx context.Context, workflowID uuid.UUID) (int, error)

	// CountByStatus returns the count of executions by status
	CountByStatus(ctx context.Context, status string) (int, error)

	// CreateNodeExecution creates a new node execution
	CreateNodeExecution(ctx context.Context, nodeExecution *models.NodeExecutionModel) error

	// UpdateNodeExecution updates an existing node execution
	UpdateNodeExecution(ctx context.Context, nodeExecution *models.NodeExecutionModel) error

	// DeleteNodeExecution deletes a node execution
	DeleteNodeExecution(ctx context.Context, id uuid.UUID) error

	// FindNodeExecutionByID retrieves a node execution by ID
	FindNodeExecutionByID(ctx context.Context, id uuid.UUID) (*models.NodeExecutionModel, error)

	// FindNodeExecutionsByExecutionID retrieves all node executions for an execution
	FindNodeExecutionsByExecutionID(ctx context.Context, executionID uuid.UUID) ([]*models.NodeExecutionModel, error)

	// FindNodeExecutionsByWave retrieves node executions by wave number
	FindNodeExecutionsByWave(ctx context.Context, executionID uuid.UUID, wave int) ([]*models.NodeExecutionModel, error)

	// FindNodeExecutionsByStatus retrieves node executions by status
	FindNodeExecutionsByStatus(ctx context.Context, executionID uuid.UUID, status string) ([]*models.NodeExecutionModel, error)

	// GetEvents retrieves all events for an execution
	GetEvents(ctx context.Context, executionID uuid.UUID) ([]*models.EventModel, error)

	// GetStatistics retrieves execution statistics
	GetStatistics(ctx context.Context, workflowID *uuid.UUID, from, to time.Time) (*ExecutionStatistics, error)
}

// ExecutionStatistics holds aggregated execution statistics
type ExecutionStatistics struct {
	TotalExecutions int            `json:"total_executions"`
	CompletedCount  int            `json:"completed_count"`
	FailedCount     int            `json:"failed_count"`
	CancelledCount  int            `json:"cancelled_count"`
	RunningCount    int            `json:"running_count"`
	PendingCount    int            `json:"pending_count"`
	AverageDuration *time.Duration `json:"average_duration,omitempty"`
	SuccessRate     float64        `json:"success_rate"`
	FailureRate     float64        `json:"failure_rate"`
}
