package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/smilemakc/mbflow/internal/infrastructure/storage/models"
	"github.com/uptrace/bun"
)

// FileRepository handles file database operations
type FileRepository struct {
	db *bun.DB
}

// NewFileRepository creates a new FileRepository
func NewFileRepository(db *bun.DB) *FileRepository {
	return &FileRepository{db: db}
}

// Create creates a new file entry
func (r *FileRepository) Create(ctx context.Context, file *models.FileModel) error {
	if file.ID == uuid.Nil {
		file.ID = uuid.New()
	}
	_, err := r.db.NewInsert().Model(file).Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	return nil
}

// Update updates an existing file entry
func (r *FileRepository) Update(ctx context.Context, file *models.FileModel) error {
	file.UpdatedAt = time.Now()
	_, err := r.db.NewUpdate().
		Model(file).
		Column("name", "mime_type", "tags", "metadata", "ttl_seconds", "expires_at", "updated_at").
		Where("id = ?", file.ID).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to update file: %w", err)
	}
	return nil
}

// Delete deletes a file entry
func (r *FileRepository) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.NewDelete().
		Model((*models.FileModel)(nil)).
		Where("id = ?", id).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to delete file: %w", err)
	}
	return nil
}

// FindByID retrieves a file by ID
func (r *FileRepository) FindByID(ctx context.Context, id uuid.UUID) (*models.FileModel, error) {
	file := &models.FileModel{}
	err := r.db.NewSelect().
		Model(file).
		Where("id = ?", id).
		Scan(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("file not found: %s", id)
		}
		return nil, fmt.Errorf("failed to find file: %w", err)
	}
	return file, nil
}

// FindByStorageAndPath finds a file by storage ID and path
func (r *FileRepository) FindByStorageAndPath(ctx context.Context, storageID, path string) (*models.FileModel, error) {
	file := &models.FileModel{}
	err := r.db.NewSelect().
		Model(file).
		Where("storage_id = ? AND path = ?", storageID, path).
		Scan(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to find file: %w", err)
	}
	return file, nil
}

// FileQuery parameters for listing files
type FileQuery struct {
	StorageID   string
	WorkflowID  *uuid.UUID
	ExecutionID *uuid.UUID
	MimeTypes   []string
	AccessScope string
	Tags        []string
	Expired     *bool
	Limit       int
	Offset      int
	OrderBy     string
	OrderDir    string
}

// FindByQuery finds files matching the query
func (r *FileRepository) FindByQuery(ctx context.Context, query *FileQuery) ([]*models.FileModel, error) {
	var files []*models.FileModel

	q := r.db.NewSelect().Model(&files)

	if query.StorageID != "" {
		q = q.Where("storage_id = ?", query.StorageID)
	}
	if query.WorkflowID != nil {
		q = q.Where("workflow_id = ?", *query.WorkflowID)
	}
	if query.ExecutionID != nil {
		q = q.Where("execution_id = ?", *query.ExecutionID)
	}
	if len(query.MimeTypes) > 0 {
		q = q.Where("mime_type IN (?)", bun.In(query.MimeTypes))
	}
	if query.AccessScope != "" {
		q = q.Where("access_scope = ?", query.AccessScope)
	}
	if len(query.Tags) > 0 {
		q = q.Where("tags && ?", query.Tags)
	}
	if query.Expired != nil {
		now := time.Now()
		if *query.Expired {
			q = q.Where("expires_at IS NOT NULL AND expires_at < ?", now)
		} else {
			q = q.Where("expires_at IS NULL OR expires_at >= ?", now)
		}
	}

	// Ordering
	orderBy := "created_at"
	if query.OrderBy != "" {
		orderBy = query.OrderBy
	}
	orderDir := "DESC"
	if query.OrderDir != "" {
		orderDir = query.OrderDir
	}
	q = q.Order(fmt.Sprintf("%s %s", orderBy, orderDir))

	// Pagination
	if query.Limit > 0 {
		q = q.Limit(query.Limit)
	}
	if query.Offset > 0 {
		q = q.Offset(query.Offset)
	}

	err := q.Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to find files: %w", err)
	}
	return files, nil
}

// CountByQuery counts files matching the query
func (r *FileRepository) CountByQuery(ctx context.Context, query *FileQuery) (int, error) {
	q := r.db.NewSelect().Model((*models.FileModel)(nil))

	if query.StorageID != "" {
		q = q.Where("storage_id = ?", query.StorageID)
	}
	if query.WorkflowID != nil {
		q = q.Where("workflow_id = ?", *query.WorkflowID)
	}
	if query.ExecutionID != nil {
		q = q.Where("execution_id = ?", *query.ExecutionID)
	}
	if query.AccessScope != "" {
		q = q.Where("access_scope = ?", query.AccessScope)
	}

	count, err := q.Count(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to count files: %w", err)
	}
	return count, nil
}

// FindExpired finds all expired files
func (r *FileRepository) FindExpired(ctx context.Context, limit int) ([]*models.FileModel, error) {
	var files []*models.FileModel
	err := r.db.NewSelect().
		Model(&files).
		Where("expires_at IS NOT NULL AND expires_at < ?", time.Now()).
		Limit(limit).
		Order("expires_at ASC").
		Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to find expired files: %w", err)
	}
	return files, nil
}

// GetStorageUsage returns storage usage statistics
func (r *FileRepository) GetStorageUsage(ctx context.Context, storageID string) (totalSize int64, fileCount int64, err error) {
	type result struct {
		TotalSize int64 `bun:"total_size"`
		FileCount int64 `bun:"file_count"`
	}

	var res result
	err = r.db.NewSelect().
		Model((*models.FileModel)(nil)).
		ColumnExpr("COALESCE(SUM(size), 0) as total_size").
		ColumnExpr("COUNT(*) as file_count").
		Where("storage_id = ?", storageID).
		Scan(ctx, &res)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to get storage usage: %w", err)
	}
	return res.TotalSize, res.FileCount, nil
}

// DeleteByStorageID deletes all files in a storage
func (r *FileRepository) DeleteByStorageID(ctx context.Context, storageID string) (int64, error) {
	result, err := r.db.NewDelete().
		Model((*models.FileModel)(nil)).
		Where("storage_id = ?", storageID).
		Exec(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to delete files by storage: %w", err)
	}
	rows, _ := result.RowsAffected()
	return rows, nil
}

// DeleteExpired deletes expired files
func (r *FileRepository) DeleteExpired(ctx context.Context) (int64, error) {
	result, err := r.db.NewDelete().
		Model((*models.FileModel)(nil)).
		Where("expires_at IS NOT NULL AND expires_at < ?", time.Now()).
		Exec(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to delete expired files: %w", err)
	}
	rows, _ := result.RowsAffected()
	return rows, nil
}

// StorageConfigRepository handles storage config database operations
type StorageConfigRepository struct {
	db *bun.DB
}

// NewStorageConfigRepository creates a new StorageConfigRepository
func NewStorageConfigRepository(db *bun.DB) *StorageConfigRepository {
	return &StorageConfigRepository{db: db}
}

// Create creates a new storage config
func (r *StorageConfigRepository) Create(ctx context.Context, config *models.StorageConfigModel) error {
	if config.ID == uuid.Nil {
		config.ID = uuid.New()
	}
	_, err := r.db.NewInsert().Model(config).Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to create storage config: %w", err)
	}
	return nil
}

// FindByStorageID finds a storage config by storage ID
func (r *StorageConfigRepository) FindByStorageID(ctx context.Context, storageID string) (*models.StorageConfigModel, error) {
	config := &models.StorageConfigModel{}
	err := r.db.NewSelect().
		Model(config).
		Where("storage_id = ?", storageID).
		Scan(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to find storage config: %w", err)
	}
	return config, nil
}

// Delete deletes a storage config
func (r *StorageConfigRepository) Delete(ctx context.Context, storageID string) error {
	_, err := r.db.NewDelete().
		Model((*models.StorageConfigModel)(nil)).
		Where("storage_id = ?", storageID).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to delete storage config: %w", err)
	}
	return nil
}

// FindAll finds all storage configs
func (r *StorageConfigRepository) FindAll(ctx context.Context) ([]*models.StorageConfigModel, error) {
	var configs []*models.StorageConfigModel
	err := r.db.NewSelect().
		Model(&configs).
		Order("created_at ASC").
		Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to find storage configs: %w", err)
	}
	return configs, nil
}
