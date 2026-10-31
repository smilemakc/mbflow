package filestorage

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/google/uuid"

	storagemodels "github.com/smilemakc/mbflow/go/internal/infrastructure/storage/models"
	"github.com/smilemakc/mbflow/go/pkg/models"
)

// RecordOptions carries the metadata-record fields that a storage cannot
// derive from the FileEntry itself.
type RecordOptions struct {
	// CreatedBy is the owning user's ID. Empty or invalid means no owner.
	CreatedBy string
}

// RecordingStorage is implemented by storages that can persist the metadata
// record of a file they store, so that the file can later be resolved by the
// returned ID (Get, Delete, GetMetadata, ...).
//
// Plain Storage.Store deliberately does not write the record: the REST upload
// handler stores and then inserts the record itself (with fields such as the
// uploading user and resource), and both writing it would collide. Callers
// without such a step — workflow nodes — use StoreRecorded instead.
type RecordingStorage interface {
	StoreRecorded(ctx context.Context, entry *models.FileEntry, reader io.Reader, opts RecordOptions) (*models.FileEntry, error)
}

var _ RecordingStorage = (*storageWrapper)(nil)

// StoreRecorded stores a file and creates its metadata record.
//
// Without a metadata repository it behaves like Store (the file is then only
// addressable by path). If the record cannot be written the stored content is
// removed again, so no unreferenced file is left behind.
//
// The record's workflow_id and execution_id columns reference persisted rows;
// an entry pointing at a workflow or execution that is not stored (e.g. an
// inline workflow) would violate those foreign keys. In that case the record is
// written once more without the links, which are kept in the record metadata
// instead.
func (s *storageWrapper) StoreRecorded(ctx context.Context, entry *models.FileEntry, reader io.Reader, opts RecordOptions) (*models.FileEntry, error) {
	stored, err := s.Store(ctx, entry, reader)
	if err != nil {
		return nil, err
	}
	repo := s.manager.metadataRepository()
	if repo == nil {
		return stored, nil
	}

	model := entryToModel(stored)
	if id, err := uuid.Parse(opts.CreatedBy); err == nil && id != uuid.Nil {
		model.CreatedBy = &id
	}

	createErr := repo.Create(ctx, model)
	if createErr != nil && (model.WorkflowID != nil || model.ExecutionID != nil) {
		if model.Metadata == nil {
			model.Metadata = storagemodels.JSONBMap{}
		}
		if model.WorkflowID != nil {
			model.Metadata["workflow_id"] = model.WorkflowID.String()
		}
		if model.ExecutionID != nil {
			model.Metadata["execution_id"] = model.ExecutionID.String()
		}
		model.WorkflowID, model.ExecutionID = nil, nil
		if retryErr := repo.Create(ctx, model); retryErr == nil {
			createErr = nil
		} else {
			createErr = errors.Join(createErr, retryErr)
		}
	}
	if createErr != nil {
		if delErr := s.provider.Delete(ctx, stored.Path); delErr != nil && s.manager.logger != nil {
			s.manager.logger.Error("Failed to remove file content after metadata failure",
				"storage_id", s.storage.storageID, "path", stored.Path, "error", delErr)
		}
		return nil, fmt.Errorf("failed to save file metadata: %w", createErr)
	}

	return modelToEntry(model), nil
}

// entryToModel converts a stored FileEntry to a metadata record. It is the
// inverse of modelToEntry; IDs that are not valid UUIDs are dropped.
func entryToModel(entry *models.FileEntry) *storagemodels.FileModel {
	model := &storagemodels.FileModel{
		StorageID:    entry.StorageID,
		Name:         entry.Name,
		Path:         entry.Path,
		MimeType:     entry.MimeType,
		Size:         entry.Size,
		Checksum:     entry.Checksum,
		AccessScope:  string(entry.AccessScope),
		Tags:         storagemodels.StringArray(entry.Tags),
		Metadata:     storagemodels.JSONBMap(entry.Metadata),
		ExpiresAt:    entry.ExpiresAt,
		SourceNodeID: entry.SourceNodeID,
		CreatedAt:    entry.CreatedAt,
		UpdatedAt:    entry.UpdatedAt,
	}
	if model.AccessScope == "" {
		model.AccessScope = string(models.ScopeWorkflow)
	}
	if id, err := uuid.Parse(entry.ID); err == nil {
		model.ID = id
	}
	model.WorkflowID = parseOptionalUUID(entry.WorkflowID)
	model.ExecutionID = parseOptionalUUID(entry.ExecutionID)
	if entry.TTL != nil {
		ttlSeconds := int(entry.TTL.Seconds())
		model.TTLSeconds = &ttlSeconds
	}
	return model
}

func parseOptionalUUID(s *string) *uuid.UUID {
	if s == nil {
		return nil
	}
	id, err := uuid.Parse(*s)
	if err != nil {
		return nil
	}
	return &id
}
