package filestorage

import (
	"bytes"
	"context"
	"errors"
	"io"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/smilemakc/mbflow/go/internal/infrastructure/storage"
	storagemodels "github.com/smilemakc/mbflow/go/internal/infrastructure/storage/models"
	"github.com/smilemakc/mbflow/go/pkg/models"
)

var errRepo = errors.New("repo failure")

// memRepo is an in-memory MetadataRepository.
type memRepo struct {
	mu        sync.Mutex
	files     map[uuid.UUID]*storagemodels.FileModel
	lastQuery *storage.FileQuery

	updateErr, deleteErr, queryErr, expiredErr error
}

func newMemRepo() *memRepo {
	return &memRepo{files: make(map[uuid.UUID]*storagemodels.FileModel)}
}

func (r *memRepo) add(f *storagemodels.FileModel) *storagemodels.FileModel {
	r.mu.Lock()
	defer r.mu.Unlock()
	if f.ID == uuid.Nil {
		f.ID = uuid.New()
	}
	r.files[f.ID] = f
	return f
}

func (r *memRepo) has(id uuid.UUID) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	_, ok := r.files[id]
	return ok
}

func (r *memRepo) Update(_ context.Context, f *storagemodels.FileModel) error {
	if r.updateErr != nil {
		return r.updateErr
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	cp := *f
	r.files[f.ID] = &cp
	return nil
}

func (r *memRepo) Delete(_ context.Context, id uuid.UUID) error {
	if r.deleteErr != nil {
		return r.deleteErr
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.files, id)
	return nil
}

func (r *memRepo) FindByID(_ context.Context, id uuid.UUID) (*storagemodels.FileModel, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	f, ok := r.files[id]
	if !ok {
		return nil, errors.New("file not found")
	}
	cp := *f
	return &cp, nil
}

func (r *memRepo) FindByQuery(_ context.Context, q *storage.FileQuery) ([]*storagemodels.FileModel, error) {
	if r.queryErr != nil {
		return nil, r.queryErr
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.lastQuery = q
	var out []*storagemodels.FileModel
	for _, f := range r.files {
		if f.StorageID == q.StorageID {
			out = append(out, f)
		}
	}
	return out, nil
}

func (r *memRepo) FindExpired(_ context.Context, limit int) ([]*storagemodels.FileModel, error) {
	if r.expiredErr != nil {
		return nil, r.expiredErr
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	var out []*storagemodels.FileModel
	for _, f := range r.files {
		if f.IsExpired() && len(out) < limit {
			out = append(out, f)
		}
	}
	return out, nil
}

// eventRecorder collects observer events.
type eventRecorder struct {
	mu     sync.Mutex
	events []*FileEvent
}

func (e *eventRecorder) observer() FileObserver {
	return NewFuncObserver("recorder", nil, func(_ context.Context, ev *FileEvent) error {
		e.mu.Lock()
		defer e.mu.Unlock()
		e.events = append(e.events, ev)
		return nil
	})
}

func (e *eventRecorder) types() []FileEventType {
	e.mu.Lock()
	defer e.mu.Unlock()
	out := make([]FileEventType, 0, len(e.events))
	for _, ev := range e.events {
		out = append(out, ev.Type)
	}
	return out
}

// newRepoBackedStorage returns a local storage of a manager with an
// in-memory metadata repository and a file already stored and recorded.
func newRepoBackedStorage(t *testing.T) (*StorageManager, Storage, *memRepo, *storagemodels.FileModel) {
	t.Helper()
	m := NewStorageManager(&ManagerConfig{BasePath: t.TempDir()}, nil)
	t.Cleanup(func() { _ = m.Close() })
	repo := newMemRepo()
	m.SetMetadataRepository(repo)

	store, err := m.GetStorage("s1")
	require.NoError(t, err)

	entry, err := store.Store(context.Background(), &models.FileEntry{
		Name: "report.txt", MimeType: "text/plain", AccessScope: models.ScopeWorkflow,
	}, strings.NewReader("hello"))
	require.NoError(t, err)

	wf := uuid.New()
	ttl := 60
	model := repo.add(&storagemodels.FileModel{
		ID:          uuid.MustParse(entry.ID),
		StorageID:   "s1",
		Name:        entry.Name,
		Path:        entry.Path,
		MimeType:    entry.MimeType,
		Size:        entry.Size,
		Checksum:    entry.Checksum,
		AccessScope: string(entry.AccessScope),
		Tags:        storagemodels.StringArray{"a"},
		Metadata:    storagemodels.JSONBMap{"k": "v", "drop": 1},
		WorkflowID:  &wf,
		TTLSeconds:  &ttl,
	})
	return m, store, repo, model
}

func TestStorageWrapper_GetByIDAndPath(t *testing.T) {
	_, store, _, model := newRepoBackedStorage(t)
	ctx := context.Background()

	entry, rc, err := store.Get(ctx, model.ID.String())
	require.NoError(t, err)
	data, _ := io.ReadAll(rc)
	rc.Close()
	assert.Equal(t, "hello", string(data))
	assert.Equal(t, "report.txt", entry.Name, "resolved by ID: full metadata")
	assert.Equal(t, []string{"a"}, entry.Tags)
	require.NotNil(t, entry.WorkflowID)
	assert.Equal(t, model.WorkflowID.String(), *entry.WorkflowID)
	require.NotNil(t, entry.TTL)
	assert.Equal(t, time.Minute, *entry.TTL)

	entry, rc, err = store.Get(ctx, model.Path)
	require.NoError(t, err)
	rc.Close()
	assert.Equal(t, model.Path, entry.Path, "resolved by path: minimal entry")
	assert.Empty(t, entry.Name)

	_, _, err = store.Get(ctx, uuid.NewString())
	assert.ErrorIs(t, err, ErrFileNotFound)

	_, _, err = store.Get(ctx, "no/such/path")
	assert.ErrorContains(t, err, "failed to get file")
}

func TestStorageWrapper_IDsAreScopedToTheirStorage(t *testing.T) {
	m, _, _, model := newRepoBackedStorage(t)
	other, err := m.GetStorage("s2")
	require.NoError(t, err)

	_, _, err = other.Get(context.Background(), model.ID.String())
	assert.ErrorIs(t, err, ErrFileNotFound)
	ok, err := other.Exists(context.Background(), model.ID.String())
	require.NoError(t, err)
	assert.False(t, ok)
	assert.ErrorIs(t, other.Delete(context.Background(), model.ID.String()), ErrFileNotFound)
}

func TestStorageWrapper_DeleteByID(t *testing.T) {
	m, store, repo, model := newRepoBackedStorage(t)
	rec := &eventRecorder{}
	require.NoError(t, m.RegisterObserver(rec.observer()))
	ctx := context.Background()

	ok, err := store.Exists(ctx, model.ID.String())
	require.NoError(t, err)
	assert.True(t, ok)

	require.NoError(t, store.Delete(ctx, model.ID.String()))
	assert.False(t, repo.has(model.ID), "metadata removed")
	ok, err = store.Exists(ctx, model.Path)
	require.NoError(t, err)
	assert.False(t, ok, "content removed")

	assert.Eventually(t, func() bool {
		for _, ty := range rec.types() {
			if ty == EventFileRemoved {
				return true
			}
		}
		return false
	}, time.Second, 5*time.Millisecond)
}

func TestStorageWrapper_DeleteErrors(t *testing.T) {
	_, store, repo, model := newRepoBackedStorage(t)
	ctx := context.Background()

	repo.deleteErr = errRepo
	assert.ErrorIs(t, store.Delete(ctx, model.ID.String()), errRepo)

	// Provider failures are reported.
	m := NewStorageManager(&ManagerConfig{}, nil)
	t.Cleanup(func() { _ = m.Close() })
	p := newMockProvider()
	p.deleteFn = func(context.Context, string) error { return errRepo }
	m.RegisterFactory(&mockFactory{createFn: func(*models.StorageConfig) (Provider, error) { return p, nil }})
	s, err := m.CreateStorage("x", &models.StorageConfig{Type: models.StorageTypeLocal})
	require.NoError(t, err)
	assert.ErrorContains(t, s.Delete(ctx, "some/path"), "failed to delete file")
}

func TestStorageWrapper_DeleteByPathWithoutRepository(t *testing.T) {
	m := NewStorageManager(&ManagerConfig{BasePath: t.TempDir()}, nil)
	t.Cleanup(func() { _ = m.Close() })
	store, err := m.GetStorage("s")
	require.NoError(t, err)
	ctx := context.Background()

	entry, err := store.Store(ctx, &models.FileEntry{Name: "a.txt", MimeType: "text/plain"}, strings.NewReader("x"))
	require.NoError(t, err)

	ok, err := store.Exists(ctx, entry.Path)
	require.NoError(t, err)
	assert.True(t, ok)
	require.NoError(t, store.Delete(ctx, entry.Path))
	ok, err = store.Exists(ctx, entry.Path)
	require.NoError(t, err)
	assert.False(t, ok)

	// Metadata operations need the repository.
	_, err = store.List(ctx, nil)
	assert.ErrorIs(t, err, ErrMetadataUnavailable)
	_, err = store.GetMetadata(ctx, entry.ID)
	assert.ErrorIs(t, err, ErrMetadataUnavailable)
	assert.ErrorIs(t, store.UpdateMetadata(ctx, entry.ID, nil), ErrMetadataUnavailable)
	assert.ErrorIs(t, store.UpdateTags(ctx, entry.ID, nil), ErrMetadataUnavailable)

	n, err := m.Cleanup(ctx)
	require.NoError(t, err)
	assert.Zero(t, n)
}

func TestStorageWrapper_ExistsProviderError(t *testing.T) {
	m := NewStorageManager(&ManagerConfig{}, nil)
	t.Cleanup(func() { _ = m.Close() })
	m.SetMetadataRepository(newMemRepo())
	p := newMockProvider()
	p.existsFn = func(context.Context, string) (bool, error) { return false, errRepo }
	m.RegisterFactory(&mockFactory{createFn: func(*models.StorageConfig) (Provider, error) { return p, nil }})
	s, err := m.CreateStorage("x", &models.StorageConfig{Type: models.StorageTypeLocal})
	require.NoError(t, err)

	_, err = s.Exists(context.Background(), "path")
	assert.ErrorIs(t, err, errRepo)
}

func TestStorageWrapper_List(t *testing.T) {
	_, store, repo, model := newRepoBackedStorage(t)
	ctx := context.Background()
	repo.add(&storagemodels.FileModel{StorageID: "other", Name: "x"})

	wf, ex := uuid.New(), uuid.New()
	expired := false
	files, err := store.List(ctx, &FileQuery{
		StorageID:   "other", // ignored: a storage lists only its own files
		WorkflowID:  wf.String(),
		ExecutionID: ex.String(),
		MimeTypes:   []string{"text/plain"},
		AccessScope: models.ScopeWorkflow,
		Tags:        []string{"a"},
		Expired:     &expired,
		Limit:       10,
		Offset:      5,
		OrderBy:     "name",
		OrderDir:    "asc",
	})
	require.NoError(t, err)
	require.Len(t, files, 1)
	assert.Equal(t, model.ID.String(), files[0].ID)

	q := repo.lastQuery
	assert.Equal(t, "s1", q.StorageID)
	assert.Equal(t, wf, *q.WorkflowID)
	assert.Equal(t, ex, *q.ExecutionID)
	assert.Equal(t, []string{"text/plain"}, q.MimeTypes)
	assert.Equal(t, "workflow", q.AccessScope)
	assert.Equal(t, []string{"a"}, q.Tags)
	assert.Equal(t, &expired, q.Expired)
	assert.Equal(t, 10, q.Limit)
	assert.Equal(t, 5, q.Offset)
	assert.Equal(t, "name", q.OrderBy)
	assert.Equal(t, "asc", q.OrderDir)

	// nil query lists everything in the storage.
	files, err = store.List(ctx, nil)
	require.NoError(t, err)
	assert.Len(t, files, 1)

	_, err = store.List(ctx, &FileQuery{WorkflowID: "bad"})
	assert.ErrorContains(t, err, "invalid workflow ID")
	_, err = store.List(ctx, &FileQuery{ExecutionID: "bad"})
	assert.ErrorContains(t, err, "invalid execution ID")

	repo.queryErr = errRepo
	_, err = store.List(ctx, nil)
	assert.ErrorIs(t, err, errRepo)
}

func TestStorageWrapper_GetAndUpdateMetadata(t *testing.T) {
	m, store, repo, model := newRepoBackedStorage(t)
	rec := &eventRecorder{}
	require.NoError(t, m.RegisterObserver(rec.observer()))
	ctx := context.Background()
	id := model.ID.String()

	entry, err := store.GetMetadata(ctx, id)
	require.NoError(t, err)
	assert.Equal(t, "report.txt", entry.Name)

	_, err = store.GetMetadata(ctx, "not/an/id")
	assert.ErrorIs(t, err, ErrFileNotFound)
	_, err = store.GetMetadata(ctx, uuid.NewString())
	assert.ErrorIs(t, err, ErrFileNotFound)

	require.NoError(t, store.UpdateMetadata(ctx, id, map[string]any{"k": "v2", "new": true, "drop": nil}))
	entry, err = store.GetMetadata(ctx, id)
	require.NoError(t, err)
	assert.Equal(t, map[string]any{"k": "v2", "new": true}, entry.Metadata)

	tags := []string{"x", "y"}
	require.NoError(t, store.UpdateTags(ctx, id, tags))
	tags[0] = "mutated"
	entry, err = store.GetMetadata(ctx, id)
	require.NoError(t, err)
	assert.Equal(t, []string{"x", "y"}, entry.Tags, "tags are copied")

	assert.Eventually(t, func() bool {
		n := 0
		for _, ty := range rec.types() {
			if ty == EventFileUpdated {
				n++
			}
		}
		return n == 2
	}, time.Second, 5*time.Millisecond)

	// Records without metadata get a fresh map.
	bare := repo.add(&storagemodels.FileModel{StorageID: "s1", Name: "bare"})
	require.NoError(t, store.UpdateMetadata(ctx, bare.ID.String(), map[string]any{"a": 1}))
	entry, err = store.GetMetadata(ctx, bare.ID.String())
	require.NoError(t, err)
	assert.Equal(t, map[string]any{"a": 1}, entry.Metadata)

	assert.ErrorIs(t, store.UpdateTags(ctx, uuid.NewString(), nil), ErrFileNotFound)
	assert.ErrorIs(t, store.UpdateMetadata(ctx, uuid.NewString(), nil), ErrFileNotFound)

	repo.updateErr = errRepo
	assert.ErrorIs(t, store.UpdateTags(ctx, id, nil), errRepo)
}

func TestStorageManager_CleanupRemovesExpiredFiles(t *testing.T) {
	m, store, repo, model := newRepoBackedStorage(t)
	rec := &eventRecorder{}
	require.NoError(t, m.RegisterObserver(rec.observer()))
	ctx := context.Background()

	past := time.Now().Add(-time.Minute)
	future := time.Now().Add(time.Hour)
	model.ExpiresAt = &past
	require.NoError(t, repo.Update(ctx, model))
	live := repo.add(&storagemodels.FileModel{StorageID: "s1", Path: "s1/live", ExpiresAt: &future})

	// An expired file of a storage this process has not opened yet (e.g.
	// after a restart): the storage is opened on demand.
	otherStore, err := m.GetStorage("lazy")
	require.NoError(t, err)
	lazyEntry, err := otherStore.Store(ctx, &models.FileEntry{Name: "l.txt", MimeType: "text/plain"}, bytes.NewReader([]byte("l")))
	require.NoError(t, err)
	lazy := repo.add(&storagemodels.FileModel{StorageID: "lazy", Path: lazyEntry.Path, ExpiresAt: &past})
	m.mu.Lock()
	delete(m.storages, "lazy")
	m.mu.Unlock()

	n, err := m.Cleanup(ctx)
	require.NoError(t, err)
	assert.Equal(t, 2, n)
	assert.False(t, repo.has(model.ID))
	assert.False(t, repo.has(lazy.ID))
	assert.True(t, repo.has(live.ID))

	ok, err := store.Exists(ctx, model.Path)
	require.NoError(t, err)
	assert.False(t, ok, "expired content removed")
	lazyStore, err := m.GetStorage("lazy")
	require.NoError(t, err)
	ok, err = lazyStore.Exists(ctx, lazyEntry.Path)
	require.NoError(t, err)
	assert.False(t, ok, "content of a lazily opened storage removed")

	assert.Eventually(t, func() bool {
		n := 0
		for _, ty := range rec.types() {
			if ty == EventFileExpired {
				n++
			}
		}
		return n == 2
	}, time.Second, 5*time.Millisecond)
}

func TestStorageManager_CleanupErrors(t *testing.T) {
	ctx := context.Background()

	t.Run("query failure", func(t *testing.T) {
		m, _, repo, _ := newRepoBackedStorage(t)
		repo.expiredErr = errRepo
		_, err := m.Cleanup(ctx)
		assert.ErrorIs(t, err, errRepo)
	})

	t.Run("metadata delete failure is reported per file", func(t *testing.T) {
		m, _, repo, model := newRepoBackedStorage(t)
		past := time.Now().Add(-time.Minute)
		model.ExpiresAt = &past
		require.NoError(t, repo.Update(ctx, model))
		repo.deleteErr = errRepo

		n, err := m.Cleanup(ctx)
		assert.Zero(t, n)
		assert.ErrorIs(t, err, errRepo)
		assert.ErrorContains(t, err, model.ID.String())
	})

	t.Run("content delete failure", func(t *testing.T) {
		m := NewStorageManager(&ManagerConfig{}, nil)
		t.Cleanup(func() { _ = m.Close() })
		repo := newMemRepo()
		m.SetMetadataRepository(repo)
		p := newMockProvider()
		p.deleteFn = func(context.Context, string) error { return errRepo }
		m.RegisterFactory(&mockFactory{createFn: func(*models.StorageConfig) (Provider, error) { return p, nil }})
		past := time.Now().Add(-time.Minute)
		f := repo.add(&storagemodels.FileModel{StorageID: "x", Path: "p", ExpiresAt: &past})

		n, err := m.Cleanup(ctx)
		assert.Zero(t, n)
		assert.ErrorContains(t, err, "failed to delete content")
		assert.True(t, repo.has(f.ID), "metadata kept when content could not be removed")
	})

	t.Run("storage cannot be opened", func(t *testing.T) {
		m := NewStorageManager(&ManagerConfig{}, nil)
		t.Cleanup(func() { _ = m.Close() })
		repo := newMemRepo()
		m.SetMetadataRepository(repo)
		m.RegisterFactory(&mockFactory{createFn: func(*models.StorageConfig) (Provider, error) { return nil, errRepo }})
		past := time.Now().Add(-time.Minute)
		repo.add(&storagemodels.FileModel{StorageID: "x", Path: "p", ExpiresAt: &past})

		_, err := m.Cleanup(ctx)
		assert.ErrorContains(t, err, "failed to open storage")
	})
}

func TestStorageManager_CleanupRoutineRunsCleanup(t *testing.T) {
	m := NewStorageManager(&ManagerConfig{BasePath: t.TempDir(), CleanupInterval: 10 * time.Millisecond}, nil)
	defer m.Close()
	repo := newMemRepo()
	past := time.Now().Add(-time.Minute)
	f := repo.add(&storagemodels.FileModel{StorageID: "s", Path: "s/none", ExpiresAt: &past})
	m.SetMetadataRepository(repo)

	assert.Eventually(t, func() bool { return !repo.has(f.ID) }, 2*time.Second, 5*time.Millisecond)
}
