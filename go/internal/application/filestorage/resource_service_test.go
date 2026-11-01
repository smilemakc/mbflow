package filestorage

import (
	"context"
	"io"
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

type resourceFixture struct {
	db       *bun.DB
	svc      *ResourceFileService
	manager  *StorageManager
	resRepo  *storage.ResourceRepositoryImpl
	fileRepo *storage.FileRepository
	resource *models.FileStorageResource
}

func newResourceFixture(t *testing.T, limit int64) *resourceFixture {
	t.Helper()
	idb, _ := testutil.SetupTestTx(t)
	db := idb.(*bun.DB)
	ctx := context.Background()

	owner := uuid.New()
	_, err := db.NewInsert().Model(&storagemodels.UserModel{
		ID: owner, Username: "u_" + owner.String()[:8], Email: owner.String()[:8] + "@test.com",
		PasswordHash: "x", IsActive: true, CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}).Exec(ctx)
	require.NoError(t, err)

	resRepo := storage.NewResourceRepository(db)
	plan, err := storage.NewPricingPlanRepository(db).GetFreePlan(ctx, models.ResourceTypeFileStorage)
	require.NoError(t, err)

	res := models.NewFileStorageResource(owner.String(), "files")
	res.StorageLimitBytes = limit
	res.PricingPlanID = plan.ID
	require.NoError(t, resRepo.Create(ctx, res))

	manager := NewStorageManager(&ManagerConfig{BasePath: t.TempDir()}, nil)
	t.Cleanup(func() { _ = manager.Close() })
	fileRepo := storage.NewFileRepository(db)

	return &resourceFixture{
		db:       db,
		svc:      NewResourceFileService(db, resRepo, fileRepo, manager, 1024),
		manager:  manager,
		resRepo:  resRepo,
		fileRepo: fileRepo,
		resource: res,
	}
}

func (f *resourceFixture) usage(t *testing.T) (int64, int) {
	t.Helper()
	fs, err := f.resRepo.GetFileStorage(context.Background(), f.resource.ID)
	require.NoError(t, err)
	return fs.UsedStorageBytes, fs.FileCount
}

func TestResourceFileService_Lifecycle(t *testing.T) {
	f := newResourceFixture(t, 10_000)
	ctx := context.Background()
	resID := f.resource.ID

	file, err := f.svc.UploadFile(ctx, resID, "notes.txt", 5, "text/plain", strings.NewReader("hello"))
	require.NoError(t, err)
	assert.Equal(t, resID, file.StorageID)
	assert.Equal(t, int64(5), file.Size)
	assert.NotEmpty(t, file.Checksum)
	require.NotNil(t, file.ResourceID)
	assert.Equal(t, resID, file.ResourceID.String())

	used, count := f.usage(t)
	assert.Equal(t, int64(5), used)
	assert.Equal(t, 1, count)

	// Read back the content (regression: the service used to pass the file
	// ID where the storage expects the stored path).
	model, rc, err := f.svc.GetFile(ctx, resID, file.ID.String())
	require.NoError(t, err)
	data, err := io.ReadAll(rc)
	require.NoError(t, err)
	rc.Close()
	assert.Equal(t, "hello", string(data))
	assert.Equal(t, "notes.txt", model.Name)

	meta, err := f.svc.GetFileMetadata(ctx, resID, file.ID.String())
	require.NoError(t, err)
	assert.Equal(t, file.ID, meta.ID)

	files, total, err := f.svc.ListFiles(ctx, resID, 10, 0)
	require.NoError(t, err)
	assert.Equal(t, 1, total)
	require.Len(t, files, 1)
	assert.Equal(t, file.ID, files[0].ID)

	// Delete removes metadata, usage and content (regression: content used
	// to be left on disk).
	store, err := f.manager.GetStorage(resID)
	require.NoError(t, err)
	exists, err := store.Exists(ctx, file.Path)
	require.NoError(t, err)
	require.True(t, exists)

	require.NoError(t, f.svc.DeleteFile(ctx, resID, file.ID.String()))
	exists, err = store.Exists(ctx, file.Path)
	require.NoError(t, err)
	assert.False(t, exists)
	used, count = f.usage(t)
	assert.Zero(t, used)
	assert.Zero(t, count)
	_, err = f.fileRepo.FindByID(ctx, file.ID)
	assert.Error(t, err)
}

func TestResourceFileService_UploadRejections(t *testing.T) {
	f := newResourceFixture(t, 8)
	ctx := context.Background()
	resID := f.resource.ID
	body := func() io.Reader { return strings.NewReader("x") }

	_, err := f.svc.UploadFile(ctx, "bad", "a", 1, "text/plain", body())
	assert.ErrorContains(t, err, "invalid resource ID")

	_, err = f.svc.UploadFile(ctx, uuid.NewString(), "a", 1, "text/plain", body())
	assert.ErrorContains(t, err, "failed to get file storage resource")

	_, err = f.svc.UploadFile(ctx, resID, "a", 2048, "text/plain", body())
	assert.ErrorContains(t, err, "exceeds maximum allowed size")

	_, err = f.svc.UploadFile(ctx, resID, "a", 9, "text/plain", body())
	assert.ErrorContains(t, err, "storage quota exceeded")

	// Storage-level validation failures roll back: no metadata, no usage.
	_, err = f.svc.UploadFile(ctx, resID, "a.exe", 1, "application/x-msdownload", body())
	assert.ErrorContains(t, err, "failed to store file")
	used, count := f.usage(t)
	assert.Zero(t, used)
	assert.Zero(t, count)

	f.resource.Status = models.ResourceStatusSuspended
	require.NoError(t, f.resRepo.Update(ctx, f.resource))
	_, err = f.svc.UploadFile(ctx, resID, "a", 1, "text/plain", body())
	assert.ErrorContains(t, err, "resource is not active")
}

func TestResourceFileService_AccessChecks(t *testing.T) {
	f := newResourceFixture(t, 10_000)
	ctx := context.Background()
	resID := f.resource.ID

	file, err := f.svc.UploadFile(ctx, resID, "a.txt", 1, "text/plain", strings.NewReader("a"))
	require.NoError(t, err)
	id := file.ID.String()
	otherRes := uuid.NewString()

	_, _, err = f.svc.GetFile(ctx, resID, "bad")
	assert.ErrorContains(t, err, "invalid file ID")
	_, _, err = f.svc.GetFile(ctx, resID, uuid.NewString())
	assert.ErrorContains(t, err, "file not found")
	_, _, err = f.svc.GetFile(ctx, otherRes, id)
	assert.ErrorContains(t, err, "does not belong")

	_, err = f.svc.GetFileMetadata(ctx, resID, "bad")
	assert.ErrorContains(t, err, "invalid file ID")
	_, err = f.svc.GetFileMetadata(ctx, resID, uuid.NewString())
	assert.ErrorContains(t, err, "file not found")
	_, err = f.svc.GetFileMetadata(ctx, otherRes, id)
	assert.ErrorContains(t, err, "does not belong")

	assert.ErrorContains(t, f.svc.DeleteFile(ctx, resID, "bad"), "invalid file ID")
	assert.ErrorContains(t, f.svc.DeleteFile(ctx, resID, uuid.NewString()), "file not found")
	assert.ErrorContains(t, f.svc.DeleteFile(ctx, otherRes, id), "does not belong")

	_, _, err = f.svc.ListFiles(ctx, "bad", 10, 0)
	assert.ErrorContains(t, err, "invalid resource ID")

	// Expired files are not served.
	past := time.Now().Add(-time.Minute)
	file.ExpiresAt = &past
	_, err = f.db.NewUpdate().Model(file).Column("expires_at").WherePK().Exec(ctx)
	require.NoError(t, err)
	_, _, err = f.svc.GetFile(ctx, resID, id)
	assert.ErrorContains(t, err, "expired")

	// Missing content is reported.
	file.ExpiresAt = nil
	_, err = f.db.NewUpdate().Model(file).Column("expires_at").WherePK().Exec(ctx)
	require.NoError(t, err)
	store, err := f.manager.GetStorage(resID)
	require.NoError(t, err)
	require.NoError(t, store.Delete(ctx, file.Path))
	_, _, err = f.svc.GetFile(ctx, resID, id)
	assert.ErrorContains(t, err, "failed to retrieve file")
}

func TestResourceFileService_DBErrors(t *testing.T) {
	f := newResourceFixture(t, 10_000)
	ctx := context.Background()
	resID := f.resource.ID

	// Without its file storage row the resource cannot take uploads.
	_, err := f.db.NewDelete().Model((*storagemodels.FileStorageModel)(nil)).Where("resource_id = ?", resID).Exec(ctx)
	require.NoError(t, err)
	_, err = f.svc.UploadFile(ctx, resID, "b.txt", 1, "text/plain", strings.NewReader("b"))
	assert.Error(t, err)

	require.NoError(t, f.db.Close())
	_, _, err = f.svc.ListFiles(ctx, resID, 10, 0)
	assert.ErrorContains(t, err, "failed to list files")
}

// TestStorageManager_WithFileRepository exercises the metadata-backed
// storage against the real repository.
func TestStorageManager_WithFileRepository(t *testing.T) {
	idb, _ := testutil.SetupTestTx(t)
	ctx := context.Background()
	repo := storage.NewFileRepository(idb)

	m := NewStorageManager(&ManagerConfig{BasePath: t.TempDir()}, nil)
	t.Cleanup(func() { _ = m.Close() })
	m.SetMetadataRepository(repo)

	store, err := m.GetStorage("wf-files")
	require.NoError(t, err)
	entry, err := store.Store(ctx, &models.FileEntry{Name: "r.json", MimeType: "application/json", AccessScope: models.ScopeWorkflow}, strings.NewReader(`{}`))
	require.NoError(t, err)

	// The caller records the metadata (as the REST handlers do).
	require.NoError(t, repo.Create(ctx, &storagemodels.FileModel{
		ID: uuid.MustParse(entry.ID), StorageID: entry.StorageID, Name: entry.Name, Path: entry.Path,
		MimeType: entry.MimeType, Size: entry.Size, Checksum: entry.Checksum, AccessScope: string(entry.AccessScope),
	}))

	got, err := store.GetMetadata(ctx, entry.ID)
	require.NoError(t, err)
	assert.Equal(t, "r.json", got.Name)

	require.NoError(t, store.UpdateTags(ctx, entry.ID, []string{"report"}))
	require.NoError(t, store.UpdateMetadata(ctx, entry.ID, map[string]any{"pages": 3.0}))
	list, err := store.List(ctx, &FileQuery{Tags: []string{"report"}})
	require.NoError(t, err)
	require.Len(t, list, 1)
	assert.Equal(t, map[string]any{"pages": 3.0}, list[0].Metadata)

	require.NoError(t, store.Delete(ctx, entry.ID))
	_, err = store.GetMetadata(ctx, entry.ID)
	assert.ErrorIs(t, err, ErrFileNotFound)
	ok, err := store.Exists(ctx, entry.Path)
	require.NoError(t, err)
	assert.False(t, ok)
}
