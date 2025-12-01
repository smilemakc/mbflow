package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/smilemakc/mbflow/internal/domain/repository"
	"github.com/smilemakc/mbflow/internal/infrastructure/storage/models"
	"github.com/uptrace/bun"
)

// Ensure ExecutionRepository implements the interface
var _ repository.ExecutionRepository = (*ExecutionRepository)(nil)

// ExecutionRepository implements repository.ExecutionRepository using Bun ORM
type ExecutionRepository struct {
	db *bun.DB
}

// NewExecutionRepository creates a new ExecutionRepository
func NewExecutionRepository(db *bun.DB) *ExecutionRepository {
	return &ExecutionRepository{db: db}
}

// Create creates a new execution
func (r *ExecutionRepository) Create(ctx context.Context, execution *models.ExecutionModel) error {
	if execution.ID == uuid.Nil {
		execution.ID = uuid.New()
	}
	_, err := r.db.NewInsert().Model(execution).Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to create execution: %w", err)
	}
	return nil
}

// Update updates an existing execution
func (r *ExecutionRepository) Update(ctx context.Context, execution *models.ExecutionModel) error {
	return r.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		// Update execution record
		_, err := tx.NewUpdate().
			Model(execution).
			Column("status", "output_data", "error", "completed_at", "variables", "updated_at").
			Where("id = ?", execution.ID).
			Exec(ctx)
		if err != nil {
			return fmt.Errorf("failed to update execution: %w", err)
		}

		// Delete existing node executions
		_, err = tx.NewDelete().
			Model((*models.NodeExecutionModel)(nil)).
			Where("execution_id = ?", execution.ID).
			Exec(ctx)
		if err != nil {
			return fmt.Errorf("failed to delete old node executions: %w", err)
		}

		// Insert new node executions if any
		if len(execution.NodeExecutions) > 0 {
			_, err = tx.NewInsert().
				Model(&execution.NodeExecutions).
				Exec(ctx)
			if err != nil {
				return fmt.Errorf("failed to insert node executions: %w", err)
			}
		}

		return nil
	})
}

// Delete deletes an execution
func (r *ExecutionRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return r.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		// Delete node executions first (cascade)
		_, err := tx.NewDelete().
			Model((*models.NodeExecutionModel)(nil)).
			Where("execution_id = ?", id).
			Exec(ctx)
		if err != nil {
			return fmt.Errorf("failed to delete node executions: %w", err)
		}

		// Delete execution
		_, err = tx.NewDelete().
			Model((*models.ExecutionModel)(nil)).
			Where("id = ?", id).
			Exec(ctx)
		if err != nil {
			return fmt.Errorf("failed to delete execution: %w", err)
		}

		return nil
	})
}

// FindByID retrieves an execution by ID
func (r *ExecutionRepository) FindByID(ctx context.Context, id uuid.UUID) (*models.ExecutionModel, error) {
	execution := &models.ExecutionModel{}
	err := r.db.NewSelect().
		Model(execution).
		Where("id = ?", id).
		Scan(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("execution not found: %s", id)
		}
		return nil, fmt.Errorf("failed to find execution: %w", err)
	}
	return execution, nil
}

// FindByIDWithRelations retrieves an execution with all its node executions
func (r *ExecutionRepository) FindByIDWithRelations(ctx context.Context, id uuid.UUID) (*models.ExecutionModel, error) {
	execution := &models.ExecutionModel{}
	err := r.db.NewSelect().
		Model(execution).
		Relation("NodeExecutions").
		Where("ex.id = ?", id).
		Scan(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("execution not found: %s", id)
		}
		return nil, fmt.Errorf("failed to find execution with relations: %w", err)
	}
	return execution, nil
}

// FindByWorkflowID retrieves executions for a workflow with pagination
func (r *ExecutionRepository) FindByWorkflowID(ctx context.Context, workflowID uuid.UUID, limit, offset int) ([]*models.ExecutionModel, error) {
	var executions []*models.ExecutionModel
	err := r.db.NewSelect().
		Model(&executions).
		Where("workflow_id = ?", workflowID).
		Order("started_at DESC").
		Limit(limit).
		Offset(offset).
		Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to find executions by workflow ID: %w", err)
	}
	return executions, nil
}

// FindByStatus retrieves executions by status with pagination
func (r *ExecutionRepository) FindByStatus(ctx context.Context, status string, limit, offset int) ([]*models.ExecutionModel, error) {
	var executions []*models.ExecutionModel
	err := r.db.NewSelect().
		Model(&executions).
		Where("status = ?", status).
		Order("started_at DESC").
		Limit(limit).
		Offset(offset).
		Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to find executions by status: %w", err)
	}
	return executions, nil
}

// FindAll retrieves all executions with pagination
func (r *ExecutionRepository) FindAll(ctx context.Context, limit, offset int) ([]*models.ExecutionModel, error) {
	var executions []*models.ExecutionModel
	err := r.db.NewSelect().
		Model(&executions).
		Order("started_at DESC").
		Limit(limit).
		Offset(offset).
		Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to find all executions: %w", err)
	}
	return executions, nil
}

// FindRunning retrieves all running executions
func (r *ExecutionRepository) FindRunning(ctx context.Context) ([]*models.ExecutionModel, error) {
	var executions []*models.ExecutionModel
	err := r.db.NewSelect().
		Model(&executions).
		Where("status = ?", "running").
		Order("started_at DESC").
		Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to find running executions: %w", err)
	}
	return executions, nil
}

// Count returns the total count of executions
func (r *ExecutionRepository) Count(ctx context.Context) (int, error) {
	count, err := r.db.NewSelect().
		Model((*models.ExecutionModel)(nil)).
		Count(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to count executions: %w", err)
	}
	return count, nil
}

// CountByWorkflowID returns the count of executions for a workflow
func (r *ExecutionRepository) CountByWorkflowID(ctx context.Context, workflowID uuid.UUID) (int, error) {
	count, err := r.db.NewSelect().
		Model((*models.ExecutionModel)(nil)).
		Where("workflow_id = ?", workflowID).
		Count(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to count executions by workflow ID: %w", err)
	}
	return count, nil
}

// CountByStatus returns the count of executions by status
func (r *ExecutionRepository) CountByStatus(ctx context.Context, status string) (int, error) {
	count, err := r.db.NewSelect().
		Model((*models.ExecutionModel)(nil)).
		Where("status = ?", status).
		Count(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to count executions by status: %w", err)
	}
	return count, nil
}

// CreateNodeExecution creates a new node execution
func (r *ExecutionRepository) CreateNodeExecution(ctx context.Context, nodeExecution *models.NodeExecutionModel) error {
	if nodeExecution.ID == uuid.Nil {
		nodeExecution.ID = uuid.New()
	}
	_, err := r.db.NewInsert().Model(nodeExecution).Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to create node execution: %w", err)
	}
	return nil
}

// UpdateNodeExecution updates an existing node execution
func (r *ExecutionRepository) UpdateNodeExecution(ctx context.Context, nodeExecution *models.NodeExecutionModel) error {
	_, err := r.db.NewUpdate().
		Model(nodeExecution).
		Column("status", "output_data", "error", "retry_count", "completed_at", "updated_at").
		Where("id = ?", nodeExecution.ID).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to update node execution: %w", err)
	}
	return nil
}

// DeleteNodeExecution deletes a node execution
func (r *ExecutionRepository) DeleteNodeExecution(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.NewDelete().
		Model((*models.NodeExecutionModel)(nil)).
		Where("id = ?", id).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to delete node execution: %w", err)
	}
	return nil
}

// FindNodeExecutionByID retrieves a node execution by ID
func (r *ExecutionRepository) FindNodeExecutionByID(ctx context.Context, id uuid.UUID) (*models.NodeExecutionModel, error) {
	nodeExec := &models.NodeExecutionModel{}
	err := r.db.NewSelect().
		Model(nodeExec).
		Where("id = ?", id).
		Scan(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("node execution not found: %s", id)
		}
		return nil, fmt.Errorf("failed to find node execution: %w", err)
	}
	return nodeExec, nil
}

// FindNodeExecutionsByExecutionID retrieves all node executions for an execution
func (r *ExecutionRepository) FindNodeExecutionsByExecutionID(ctx context.Context, executionID uuid.UUID) ([]*models.NodeExecutionModel, error) {
	var nodeExecutions []*models.NodeExecutionModel
	err := r.db.NewSelect().
		Model(&nodeExecutions).
		Where("execution_id = ?", executionID).
		Order("wave ASC", "created_at ASC").
		Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to find node executions by execution ID: %w", err)
	}
	return nodeExecutions, nil
}

// FindNodeExecutionsByWave retrieves node executions by wave number
func (r *ExecutionRepository) FindNodeExecutionsByWave(ctx context.Context, executionID uuid.UUID, wave int) ([]*models.NodeExecutionModel, error) {
	var nodeExecutions []*models.NodeExecutionModel
	err := r.db.NewSelect().
		Model(&nodeExecutions).
		Where("execution_id = ? AND wave = ?", executionID, wave).
		Order("created_at ASC").
		Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to find node executions by wave: %w", err)
	}
	return nodeExecutions, nil
}

// FindNodeExecutionsByStatus retrieves node executions by status
func (r *ExecutionRepository) FindNodeExecutionsByStatus(ctx context.Context, executionID uuid.UUID, status string) ([]*models.NodeExecutionModel, error) {
	var nodeExecutions []*models.NodeExecutionModel
	err := r.db.NewSelect().
		Model(&nodeExecutions).
		Where("execution_id = ? AND status = ?", executionID, status).
		Order("created_at ASC").
		Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to find node executions by status: %w", err)
	}
	return nodeExecutions, nil
}

// GetStatistics retrieves execution statistics
func (r *ExecutionRepository) GetStatistics(ctx context.Context, workflowID *uuid.UUID, from, to time.Time) (*repository.ExecutionStatistics, error) {
	stats := &repository.ExecutionStatistics{}

	// Build base query
	query := r.db.NewSelect().
		Model((*models.ExecutionModel)(nil)).
		Where("started_at >= ? AND started_at <= ?", from, to)

	if workflowID != nil {
		query = query.Where("workflow_id = ?", *workflowID)
	}

	// Get total count
	total, err := query.Count(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to count total executions: %w", err)
	}
	stats.TotalExecutions = total

	// Count by status
	type StatusCount struct {
		Status string
		Count  int
	}
	var statusCounts []StatusCount
	err = r.db.NewSelect().
		Model((*models.ExecutionModel)(nil)).
		Column("status").
		ColumnExpr("COUNT(*) as count").
		Where("started_at >= ? AND started_at <= ?", from, to).
		Apply(func(q *bun.SelectQuery) *bun.SelectQuery {
			if workflowID != nil {
				return q.Where("workflow_id = ?", *workflowID)
			}
			return q
		}).
		Group("status").
		Scan(ctx, &statusCounts)
	if err != nil && err != sql.ErrNoRows {
		return nil, fmt.Errorf("failed to count by status: %w", err)
	}

	// Map status counts
	for _, sc := range statusCounts {
		switch sc.Status {
		case "completed":
			stats.CompletedCount = sc.Count
		case "failed":
			stats.FailedCount = sc.Count
		case "cancelled":
			stats.CancelledCount = sc.Count
		case "running":
			stats.RunningCount = sc.Count
		case "pending":
			stats.PendingCount = sc.Count
		}
	}

	// Calculate average duration for completed executions
	var avgDuration struct {
		AvgDuration float64
	}
	err = r.db.NewSelect().
		Model((*models.ExecutionModel)(nil)).
		ColumnExpr("AVG(EXTRACT(EPOCH FROM (completed_at - started_at))) as avg_duration").
		Where("started_at >= ? AND started_at <= ? AND status = ? AND completed_at IS NOT NULL", from, to, "completed").
		Apply(func(q *bun.SelectQuery) *bun.SelectQuery {
			if workflowID != nil {
				return q.Where("workflow_id = ?", *workflowID)
			}
			return q
		}).
		Scan(ctx, &avgDuration)
	if err == nil && avgDuration.AvgDuration > 0 {
		duration := time.Duration(avgDuration.AvgDuration * float64(time.Second))
		stats.AverageDuration = &duration
	}

	// Calculate success and failure rates
	if stats.TotalExecutions > 0 {
		stats.SuccessRate = float64(stats.CompletedCount) / float64(stats.TotalExecutions)
		stats.FailureRate = float64(stats.FailedCount) / float64(stats.TotalExecutions)
	}

	return stats, nil
}
