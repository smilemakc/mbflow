package filestorage

import (
	"context"
	"database/sql"
	"fmt"
	"io"

	"github.com/google/uuid"
	"github.com/smilemakc/mbflow/go/internal/domain/repository"
	"github.com/smilemakc/mbflow/go/internal/infrastructure/storage"
	storagemodels "github.com/smilemakc/mbflow/go/internal/infrastructure/storage/models"
	"github.com/smilemakc/mbflow/go/pkg/models"
	"github.com/uptrace/bun"
)

type ResourceFileService struct {
	db             *bun.DB
	resourceRepo   repository.FileStorageRepository
	fileRepo       *storage.FileRepository
	storageManager Manager
	maxFileSize    int64
}

func NewResourceFileService(
	db *bun.DB,
	resourceRepo repository.FileStorageRepository,
	fileRepo *storage.FileRepository,
	storageManager Manager,
	maxFileSize int64,
) *ResourceFileService {
	return &ResourceFileService{
		db:             db,
		resourceRepo:   resourceRepo,
		fileRepo:       fileRepo,
		storageManager: storageManager,
		maxFileSize:    maxFileSize,
	}
}

func (s *ResourceFileService) UploadFile(
	ctx context.Context,
	resourceID string,
	fileName string,
	fileSize int64,
	mimeType string,
	reader io.Reader,
) (*storagemodels.FileModel, error) {
	resUUID, err := uuid.Parse(resourceID)
	if err != nil {
		return nil, fmt.Errorf("invalid resource ID: %w", err)
	}

	fsResource, err := s.resourceRepo.GetFileStorage(ctx, resourceID)
	if err != nil {
		return nil, fmt.Errorf("failed to get file storage resource: %w", err)
	}

	if !fsResource.IsActive() {
		return nil, fmt.Errorf("resource is not active")
	}

	if fileSize > s.maxFileSize {
		return nil, fmt.Errorf("file size %d exceeds maximum allowed size %d", fileSize, s.maxFileSize)
	}

	if !fsResource.CanAddFile(fileSize) {
		return nil, fmt.Errorf("storage quota exceeded: available %d bytes, required %d bytes",
			fsResource.GetAvailableSpace(), fileSize)
	}

	store, err := s.storageManager.GetStorage(resourceID)
	if err != nil {
		store, err = s.storageManager.CreateStorage(resourceID, &models.StorageConfig{
			Type:     models.StorageTypeLocal,
			BasePath: fmt.Sprintf("./data/storage/%s", resourceID),
			MaxSize:  fsResource.StorageLimitBytes,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to create storage: %w", err)
		}
	}

	entry := &models.FileEntry{
		StorageID:   resourceID,
		Name:        fileName,
		MimeType:    mimeType,
		Size:        fileSize,
		AccessScope: models.ScopeResource,
		Metadata:    make(map[string]any),
	}

	var fileModel *storagemodels.FileModel

	err = s.db.RunInTx(ctx, &sql.TxOptions{}, func(ctx context.Context, tx bun.Tx) error {
		stored, err := store.Store(ctx, entry, reader)
		if err != nil {
			return fmt.Errorf("failed to store file: %w", err)
		}

		fileModel = &storagemodels.FileModel{
			StorageID:   stored.StorageID,
			Name:        stored.Name,
			Path:        stored.Path,
			MimeType:    stored.MimeType,
			Size:        stored.Size,
			Checksum:    stored.Checksum,
			AccessScope: string(stored.AccessScope),
			Tags:        storagemodels.StringArray(stored.Tags),
			Metadata:    storagemodels.JSONBMap(stored.Metadata),
			ResourceID:  &resUUID,
		}

		if id, err := uuid.Parse(stored.ID); err == nil {
			fileModel.ID = id
		}

		if _, err := tx.NewInsert().Model(fileModel).Exec(ctx); err != nil {
			return fmt.Errorf("failed to save file metadata: %w", err)
		}

		if err := s.resourceRepo.IncrementUsage(ctx, resourceID, fileSize); err != nil {
			return fmt.Errorf("failed to update resource usage: %w", err)
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return fileModel, nil
}

func (s *ResourceFileService) GetFile(
	ctx context.Context,
	resourceID string,
	fileID string,
) (*storagemodels.FileModel, io.ReadCloser, error) {
	fileUUID, err := uuid.Parse(fileID)
	if err != nil {
		return nil, nil, fmt.Errorf("invalid file ID: %w", err)
	}

	fileModel, err := s.fileRepo.FindByID(ctx, fileUUID)
	if err != nil {
		return nil, nil, fmt.Errorf("file not found: %w", err)
	}

	if fileModel.ResourceID == nil || fileModel.ResourceID.String() != resourceID {
		return nil, nil, fmt.Errorf("file does not belong to this resource")
	}

	if fileModel.IsExpired() {
		return nil, nil, fmt.Errorf("file has expired")
	}

	store, err := s.storageManager.GetStorage(fileModel.StorageID)
	if err != nil {
		return nil, nil, fmt.Errorf("storage not available: %w", err)
	}

	_, reader, err := store.Get(ctx, fileID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to retrieve file: %w", err)
	}

	return fileModel, reader, nil
}

func (s *ResourceFileService) DeleteFile(
	ctx context.Context,
	resourceID string,
	fileID string,
) error {
	fileUUID, err := uuid.Parse(fileID)
	if err != nil {
		return fmt.Errorf("invalid file ID: %w", err)
	}

	fileModel, err := s.fileRepo.FindByID(ctx, fileUUID)
	if err != nil {
		return fmt.Errorf("file not found: %w", err)
	}

	if fileModel.ResourceID == nil || fileModel.ResourceID.String() != resourceID {
		return fmt.Errorf("file does not belong to this resource")
	}

	return s.db.RunInTx(ctx, &sql.TxOptions{}, func(ctx context.Context, tx bun.Tx) error {
		store, err := s.storageManager.GetStorage(fileModel.StorageID)
		if err == nil {
			_ = store.Delete(ctx, fileID)
		}

		if _, err := tx.NewDelete().Model(fileModel).WherePK().Exec(ctx); err != nil {
			return fmt.Errorf("failed to delete file metadata: %w", err)
		}

		if err := s.resourceRepo.DecrementUsage(ctx, resourceID, fileModel.Size); err != nil {
			return fmt.Errorf("failed to update resource usage: %w", err)
		}

		return nil
	})
}

func (s *ResourceFileService) ListFiles(
	ctx context.Context,
	resourceID string,
	limit int,
	offset int,
) ([]*storagemodels.FileModel, int, error) {
	resUUID, err := uuid.Parse(resourceID)
	if err != nil {
		return nil, 0, fmt.Errorf("invalid resource ID: %w", err)
	}

	var files []*storagemodels.FileModel
	err = s.db.NewSelect().
		Model(&files).
		Where("resource_id = ?", resUUID).
		Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Scan(ctx)

	if err != nil {
		return nil, 0, fmt.Errorf("failed to list files: %w", err)
	}

	count, err := s.db.NewSelect().
		Model((*storagemodels.FileModel)(nil)).
		Where("resource_id = ?", resUUID).
		Count(ctx)

	if err != nil {
		return nil, 0, fmt.Errorf("failed to count files: %w", err)
	}

	return files, count, nil
}

func (s *ResourceFileService) GetFileMetadata(
	ctx context.Context,
	resourceID string,
	fileID string,
) (*storagemodels.FileModel, error) {
	fileUUID, err := uuid.Parse(fileID)
	if err != nil {
		return nil, fmt.Errorf("invalid file ID: %w", err)
	}

	fileModel, err := s.fileRepo.FindByID(ctx, fileUUID)
	if err != nil {
		return nil, fmt.Errorf("file not found: %w", err)
	}

	if fileModel.ResourceID == nil || fileModel.ResourceID.String() != resourceID {
		return nil, fmt.Errorf("file does not belong to this resource")
	}

	return fileModel, nil
}
