package builtin

import (
	"context"
	"encoding/base64"
	"errors"
	"sync"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/smilemakc/mbflow/go/internal/application/filestorage"
	"github.com/smilemakc/mbflow/go/internal/infrastructure/storage"
	storagemodels "github.com/smilemakc/mbflow/go/internal/infrastructure/storage/models"
	"github.com/smilemakc/mbflow/go/pkg/executor"
)

// sessFileRepo is an in-memory filestorage.MetadataRepository.
type sessFileRepo struct {
	mu    sync.Mutex
	files map[uuid.UUID]*storagemodels.FileModel
}

func (r *sessFileRepo) Create(_ context.Context, f *storagemodels.FileModel) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	cp := *f
	r.files[f.ID] = &cp
	return nil
}

func (r *sessFileRepo) Update(_ context.Context, f *storagemodels.FileModel) error {
	return r.Create(context.Background(), f)
}

func (r *sessFileRepo) Delete(_ context.Context, id uuid.UUID) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.files, id)
	return nil
}

func (r *sessFileRepo) FindByID(_ context.Context, id uuid.UUID) (*storagemodels.FileModel, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	f, ok := r.files[id]
	if !ok {
		return nil, errors.New("not found")
	}
	cp := *f
	return &cp, nil
}

func (r *sessFileRepo) FindByQuery(context.Context, *storage.FileQuery) ([]*storagemodels.FileModel, error) {
	return nil, nil
}

func (r *sessFileRepo) FindExpired(context.Context, int) ([]*storagemodels.FileModel, error) {
	return nil, nil
}

// TestSess_FileStorageExecutor_StoreRecordsFileForLaterActions checks that a
// file stored by the node can be read, inspected and deleted by its file_id,
// and that the record carries the workflow owner and run ids from ctx.
func TestSess_FileStorageExecutor_StoreRecordsFileForLaterActions(t *testing.T) {
	m := filestorage.NewStorageManager(&filestorage.ManagerConfig{BasePath: t.TempDir()}, nil)
	t.Cleanup(func() { _ = m.Close() })
	repo := &sessFileRepo{files: map[uuid.UUID]*storagemodels.FileModel{}}
	m.SetMetadataRepository(repo)
	exec := NewFileStorageExecutor(m)

	owner, wf, run := uuid.New(), uuid.New(), uuid.New()
	ctx := executor.WithExecutionContext(context.Background(), &executor.ExecutionContextData{
		ExecutionID: run.String(), WorkflowID: wf.String(), NodeID: "store", OwnerID: owner.String(),
	})

	out, err := exec.Execute(ctx, map[string]any{
		"action":    "store",
		"file_data": base64.StdEncoding.EncodeToString([]byte("hello node")),
		"file_name": "greeting.txt",
		"mime_type": "text/plain",
	}, nil)
	require.NoError(t, err)
	fileID := out.(map[string]any)["file_id"].(string)

	rec, err := repo.FindByID(ctx, uuid.MustParse(fileID))
	require.NoError(t, err)
	require.NotNil(t, rec.CreatedBy)
	assert.Equal(t, owner, *rec.CreatedBy)
	assert.Equal(t, wf, *rec.WorkflowID)
	assert.Equal(t, run, *rec.ExecutionID)
	assert.Equal(t, "store", *rec.SourceNodeID)

	meta, err := exec.Execute(ctx, map[string]any{"action": "metadata", "file_id": fileID}, nil)
	require.NoError(t, err)
	assert.Equal(t, "greeting.txt", meta.(map[string]any)["file_name"])

	got, err := exec.Execute(ctx, map[string]any{"action": "get", "file_id": fileID}, nil)
	require.NoError(t, err)
	assert.Equal(t, base64.StdEncoding.EncodeToString([]byte("hello node")), got.(map[string]any)["file_data"])

	_, err = exec.Execute(ctx, map[string]any{"action": "delete", "file_id": fileID}, nil)
	require.NoError(t, err)
	_, err = repo.FindByID(ctx, uuid.MustParse(fileID))
	assert.Error(t, err, "record removed with the file")

	// Without an execution context the file is still recorded, unowned.
	out, err = exec.Execute(context.Background(), map[string]any{
		"action":    "store",
		"file_data": base64.StdEncoding.EncodeToString([]byte("x")),
		"mime_type": "text/plain",
	}, nil)
	require.NoError(t, err)
	rec, err = repo.FindByID(ctx, uuid.MustParse(out.(map[string]any)["file_id"].(string)))
	require.NoError(t, err)
	assert.Nil(t, rec.CreatedBy)
	assert.Nil(t, rec.WorkflowID)
}
