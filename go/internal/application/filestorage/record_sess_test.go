package filestorage

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"

	"github.com/smilemakc/mbflow/go/internal/infrastructure/storage"
	storagemodels "github.com/smilemakc/mbflow/go/internal/infrastructure/storage/models"
	"github.com/smilemakc/mbflow/go/pkg/models"
	"github.com/smilemakc/mbflow/go/testutil"
)

func sessRecordingStorage(t *testing.T, repo MetadataRepository) (*StorageManager, RecordingStorage, string) {
	t.Helper()
	base := t.TempDir()
	m := NewStorageManager(&ManagerConfig{BasePath: base}, nil)
	t.Cleanup(func() { _ = m.Close() })
	if repo != nil {
		m.SetMetadataRepository(repo)
	}
	store, err := m.GetStorage("node-files")
	require.NoError(t, err)
	rs, ok := store.(RecordingStorage)
	require.True(t, ok, "manager storages implement RecordingStorage")
	return m, rs, base
}

func sessStorageFileCount(t *testing.T, base string) int {
	t.Helper()
	n := 0
	_ = filepath.Walk(base, func(_ string, info os.FileInfo, err error) error {
		if err == nil && !info.IsDir() {
			n++
		}
		return nil
	})
	return n
}

func TestSess_StoreRecorded_WritesRecordResolvableByID(t *testing.T) {
	repo := newMemRepo()
	_, rs, _ := sessRecordingStorage(t, repo)
	ctx := context.Background()

	owner := uuid.New()
	wf, ex, node := uuid.New().String(), uuid.New().String(), "n1"
	entry := &models.FileEntry{
		Name: "out.txt", MimeType: "text/plain", AccessScope: models.ScopeEdge,
		Tags: []string{"t"}, WorkflowID: &wf, ExecutionID: &ex, SourceNodeID: &node,
	}
	entry.SetTTL(time.Hour)

	stored, err := rs.StoreRecorded(ctx, entry, strings.NewReader("hello"), RecordOptions{CreatedBy: owner.String()})
	require.NoError(t, err)

	rec, err := repo.FindByID(ctx, uuid.MustParse(stored.ID))
	require.NoError(t, err)
	require.NotNil(t, rec.CreatedBy)
	assert.Equal(t, owner, *rec.CreatedBy)
	assert.Equal(t, "node-files", rec.StorageID)
	assert.Equal(t, wf, rec.WorkflowID.String())
	assert.Equal(t, ex, rec.ExecutionID.String())
	assert.Equal(t, "n1", *rec.SourceNodeID)
	require.NotNil(t, rec.TTLSeconds)
	assert.Equal(t, 3600, *rec.TTLSeconds)
	assert.NotEmpty(t, rec.Checksum)

	// The returned entry mirrors the record and the ID resolves.
	assert.Equal(t, models.ScopeEdge, stored.AccessScope)
	store := rs.(Storage)
	meta, err := store.GetMetadata(ctx, stored.ID)
	require.NoError(t, err)
	assert.Equal(t, "out.txt", meta.Name)
	_, rc, err := store.Get(ctx, stored.ID)
	require.NoError(t, err)
	data, _ := io.ReadAll(rc)
	_ = rc.Close()
	assert.Equal(t, "hello", string(data))
	require.NoError(t, store.Delete(ctx, stored.ID))
	assert.False(t, repo.has(uuid.MustParse(stored.ID)))
}

func TestSess_StoreRecorded_RetriesWithoutDanglingLinks(t *testing.T) {
	repo := newMemRepo()
	repo.createErrs = []error{errRepo, nil}
	_, rs, _ := sessRecordingStorage(t, repo)
	ctx := context.Background()

	wf, ex := uuid.New().String(), uuid.New().String()
	stored, err := rs.StoreRecorded(ctx, &models.FileEntry{
		Name: "a.txt", MimeType: "text/plain", WorkflowID: &wf, ExecutionID: &ex,
	}, strings.NewReader("x"), RecordOptions{CreatedBy: "not-a-uuid"})
	require.NoError(t, err)
	assert.Equal(t, 2, repo.creates)

	rec, err := repo.FindByID(ctx, uuid.MustParse(stored.ID))
	require.NoError(t, err)
	assert.Nil(t, rec.WorkflowID)
	assert.Nil(t, rec.ExecutionID)
	assert.Nil(t, rec.CreatedBy)
	assert.Equal(t, wf, rec.Metadata["workflow_id"])
	assert.Equal(t, ex, rec.Metadata["execution_id"])
	assert.Equal(t, string(models.ScopeWorkflow), rec.AccessScope, "empty scope defaults to workflow")
}

func TestSess_StoreRecorded_FailureRemovesContent(t *testing.T) {
	t.Run("without links no retry", func(t *testing.T) {
		repo := newMemRepo()
		repo.createErrs = []error{errRepo}
		_, rs, base := sessRecordingStorage(t, repo)
		_, err := rs.StoreRecorded(context.Background(), &models.FileEntry{Name: "a.txt", MimeType: "text/plain"},
			strings.NewReader("x"), RecordOptions{})
		require.ErrorIs(t, err, errRepo)
		assert.Equal(t, 1, repo.creates)
		assert.Equal(t, 0, sessStorageFileCount(t, base), "content is removed when the record fails")
	})

	t.Run("retry fails too", func(t *testing.T) {
		repo := newMemRepo()
		repo.createErrs = []error{errRepo, errRepo}
		_, rs, base := sessRecordingStorage(t, repo)
		wf := uuid.New().String()
		_, err := rs.StoreRecorded(context.Background(), &models.FileEntry{Name: "a.txt", MimeType: "text/plain", WorkflowID: &wf},
			strings.NewReader("x"), RecordOptions{})
		require.ErrorIs(t, err, errRepo)
		assert.Equal(t, 2, repo.creates)
		assert.Equal(t, 0, sessStorageFileCount(t, base))
	})
}

func TestSess_StoreRecorded_NoRepositoryAndStoreErrors(t *testing.T) {
	_, rs, _ := sessRecordingStorage(t, nil)
	stored, err := rs.StoreRecorded(context.Background(), &models.FileEntry{Name: "a.txt", MimeType: "text/plain"},
		strings.NewReader("x"), RecordOptions{})
	require.NoError(t, err)
	assert.NotEmpty(t, stored.Path)

	_, err = rs.StoreRecorded(context.Background(), &models.FileEntry{Name: "a.exe", MimeType: "application/x-msdownload"},
		strings.NewReader("x"), RecordOptions{})
	require.Error(t, err)
}

func TestSess_EntryToModel_InvalidIDs(t *testing.T) {
	bad := "nope"
	m := entryToModel(&models.FileEntry{ID: "x", WorkflowID: &bad, ExecutionID: nil, AccessScope: models.ScopeResult})
	assert.Equal(t, uuid.Nil, m.ID)
	assert.Nil(t, m.WorkflowID)
	assert.Nil(t, m.ExecutionID)
	assert.Equal(t, "result", m.AccessScope)
}

// TestSess_StoreRecorded_WithFileRepository runs against PostgreSQL: the
// owner is recorded and an unknown workflow (foreign key violation) falls back
// to a record without links.
func TestSess_StoreRecorded_WithFileRepository(t *testing.T) {
	idb, _ := testutil.SetupTestTx(t)
	db := idb.(*bun.DB)
	ctx := context.Background()

	owner := uuid.New()
	_, err := db.NewInsert().Model(&storagemodels.UserModel{
		ID: owner, Username: "u_" + owner.String()[:8], Email: owner.String()[:8] + "@test.com",
		PasswordHash: "x", IsActive: true, CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}).Exec(ctx)
	require.NoError(t, err)

	repo := storage.NewFileRepository(db)
	_, rs, _ := sessRecordingStorage(t, repo)

	unknownWorkflow := uuid.New().String()
	node := "store-node"
	stored, err := rs.StoreRecorded(ctx, &models.FileEntry{
		Name: "r.json", MimeType: "application/json", WorkflowID: &unknownWorkflow, SourceNodeID: &node,
	}, strings.NewReader(`{}`), RecordOptions{CreatedBy: owner.String()})
	require.NoError(t, err)

	rec, err := repo.FindByID(ctx, uuid.MustParse(stored.ID))
	require.NoError(t, err)
	require.NotNil(t, rec.CreatedBy)
	assert.Equal(t, owner, *rec.CreatedBy)
	assert.Nil(t, rec.WorkflowID)
	assert.Equal(t, unknownWorkflow, rec.Metadata["workflow_id"])
	assert.Equal(t, "store-node", *rec.SourceNodeID)

	meta, err := rs.(Storage).GetMetadata(ctx, stored.ID)
	require.NoError(t, err)
	assert.Equal(t, "r.json", meta.Name)
}
