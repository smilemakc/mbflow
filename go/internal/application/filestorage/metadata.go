package filestorage

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"

	storagerepo "github.com/smilemakc/mbflow/go/internal/infrastructure/storage"
	storagemodels "github.com/smilemakc/mbflow/go/internal/infrastructure/storage/models"
	"github.com/smilemakc/mbflow/go/pkg/models"
)

var (
	// ErrMetadataUnavailable is returned by operations that need file
	// metadata when the manager has no metadata repository.
	ErrMetadataUnavailable = errors.New("file metadata repository not configured")

	// ErrFileNotFound is returned when a file ID does not resolve to a file
	// of the storage.
	ErrFileNotFound = errors.New("file not found")
)

// MetadataRepository is the subset of the file metadata repository the
// storage manager needs. *storagerepo.FileRepository implements it.
type MetadataRepository interface {
	Update(ctx context.Context, file *storagemodels.FileModel) error
	Delete(ctx context.Context, id uuid.UUID) error
	FindByID(ctx context.Context, id uuid.UUID) (*storagemodels.FileModel, error)
	FindByQuery(ctx context.Context, query *storagerepo.FileQuery) ([]*storagemodels.FileModel, error)
	FindExpired(ctx context.Context, limit int) ([]*storagemodels.FileModel, error)
}

var _ MetadataRepository = (*storagerepo.FileRepository)(nil)

// modelToEntry converts a metadata record to a FileEntry.
func modelToEntry(m *storagemodels.FileModel) *models.FileEntry {
	entry := &models.FileEntry{
		ID:           m.ID.String(),
		StorageID:    m.StorageID,
		Name:         m.Name,
		Path:         m.Path,
		MimeType:     m.MimeType,
		Size:         m.Size,
		Checksum:     m.Checksum,
		AccessScope:  models.AccessScope(m.AccessScope),
		Tags:         []string(m.Tags),
		Metadata:     map[string]any(m.Metadata),
		ExpiresAt:    m.ExpiresAt,
		SourceNodeID: m.SourceNodeID,
		CreatedAt:    m.CreatedAt,
		UpdatedAt:    m.UpdatedAt,
	}
	if m.TTLSeconds != nil {
		ttl := time.Duration(*m.TTLSeconds) * time.Second
		entry.TTL = &ttl
	}
	if m.WorkflowID != nil {
		id := m.WorkflowID.String()
		entry.WorkflowID = &id
	}
	if m.ExecutionID != nil {
		id := m.ExecutionID.String()
		entry.ExecutionID = &id
	}
	return entry
}
