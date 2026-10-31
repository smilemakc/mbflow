package storage

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/smilemakc/mbflow/go/internal/domain/repository"
	"github.com/smilemakc/mbflow/go/internal/infrastructure/storage/models"
	"github.com/smilemakc/mbflow/go/testutil"
)

// ---------------------------------------------------------------------------
// Executions
// ---------------------------------------------------------------------------

func TestRC_ExecutionRepository_RelationsEventsAndFilters(t *testing.T) {
	t.Parallel()
	db, _ := testutil.SetupTestTx(t)
	ctx := context.Background()
	repo := NewExecutionRepository(db)
	wfRepo := NewWorkflowRepository(db)
	eventRepo := NewEventRepository(db)

	owner := uuid.MustParse(createCredentialsTestUser(t, db))
	other := uuid.MustParse(createCredentialsTestUser(t, db))

	ownedWF := rcWorkflow("owned", &owner)
	otherWF := rcWorkflow("other", &other)
	require.NoError(t, wfRepo.Create(ctx, ownedWF))
	require.NoError(t, wfRepo.Create(ctx, otherWF))

	now := time.Now()
	mkExec := func(wf *uuid.UUID, status string, createdBy *uuid.UUID, started time.Time) *models.ExecutionModel {
		e := &models.ExecutionModel{ID: uuid.New(), WorkflowID: wf, Status: status, CreatedBy: createdBy, StartedAt: &started}
		if wf == nil {
			e.WorkflowSource = "inline"
			e.WorkflowSnapshot = models.JSONBMap{"name": "inline"}
		}
		require.NoError(t, repo.Create(ctx, e))
		return e
	}
	e1 := mkExec(&ownedWF.ID, "completed", nil, now.Add(-3*time.Minute))
	e2 := mkExec(&ownedWF.ID, "failed", nil, now.Add(-2*time.Minute))
	e3 := mkExec(&otherWF.ID, "completed", nil, now.Add(-time.Minute))
	e4 := mkExec(nil, "running", &owner, now)

	// relations
	key := "node-a"
	require.NoError(t, repo.CreateNodeExecution(ctx, &models.NodeExecutionModel{ExecutionID: e1.ID, NodeKey: &key, Status: "completed"}))
	withRel, err := repo.FindByIDWithRelations(ctx, e1.ID)
	require.NoError(t, err)
	require.Len(t, withRel.NodeExecutions, 1)
	assert.Equal(t, "node-a", *withRel.NodeExecutions[0].NodeKey)
	_, err = repo.FindByIDWithRelations(ctx, uuid.New())
	assert.ErrorContains(t, err, "execution not found")

	// events
	for i := 0; i < 3; i++ {
		require.NoError(t, eventRepo.Append(ctx, &models.EventModel{ExecutionID: e1.ID, EventType: "node.started", Payload: models.JSONBMap{"i": i}}))
	}
	events, err := repo.GetEvents(ctx, e1.ID)
	require.NoError(t, err)
	require.Len(t, events, 3)
	assert.Less(t, events[0].Sequence, events[2].Sequence)
	since, err := repo.GetEventsSince(ctx, e1.ID, events[0].Sequence)
	require.NoError(t, err)
	assert.Len(t, since, 2)
	none, err := repo.GetEventsSince(ctx, e1.ID, events[2].Sequence)
	require.NoError(t, err)
	assert.Empty(t, none)

	// filters
	completed := "completed"
	cases := []struct {
		name    string
		filters repository.ExecutionFilters
		want    []uuid.UUID
	}{
		{"all", repository.ExecutionFilters{}, []uuid.UUID{e4.ID, e3.ID, e2.ID, e1.ID}},
		{"workflow", repository.ExecutionFilters{WorkflowID: &ownedWF.ID}, []uuid.UUID{e2.ID, e1.ID}},
		{"status", repository.ExecutionFilters{Status: &completed}, []uuid.UUID{e3.ID, e1.ID}},
		{"owner includes inline", repository.ExecutionFilters{OwnerID: &owner}, []uuid.UUID{e4.ID, e2.ID, e1.ID}},
		{"owner+status", repository.ExecutionFilters{OwnerID: &owner, Status: &completed}, []uuid.UUID{e1.ID}},
		{"other owner", repository.ExecutionFilters{OwnerID: &other}, []uuid.UUID{e3.ID}},
	}
	for _, c := range cases {
		list, err := repo.FindWithFilters(ctx, c.filters, 10, 0)
		require.NoError(t, err, c.name)
		got := make([]uuid.UUID, len(list))
		for i, e := range list {
			got[i] = e.ID
		}
		assert.Equal(t, c.want, got, c.name+" (ordered by started_at DESC)")
	}
	page, err := repo.FindWithFilters(ctx, repository.ExecutionFilters{}, 2, 1)
	require.NoError(t, err)
	require.Len(t, page, 2)
	assert.Equal(t, e3.ID, page[0].ID)
}

// ---------------------------------------------------------------------------
// Triggers
// ---------------------------------------------------------------------------

func TestRC_TriggerRepository_FiltersAndDisabledCreate(t *testing.T) {
	t.Parallel()
	db, _ := testutil.SetupTestTx(t)
	ctx := context.Background()
	repo := NewTriggerRepository(db).(*TriggerRepository)
	wfRepo := NewWorkflowRepository(db)

	owner := uuid.MustParse(createCredentialsTestUser(t, db))
	ownedWF := rcWorkflow("owned", &owner)
	otherWF := rcWorkflow("other", nil)
	require.NoError(t, wfRepo.Create(ctx, ownedWF))
	require.NoError(t, wfRepo.Create(ctx, otherWF))

	mk := func(wf uuid.UUID, typ string, enabled bool) *models.TriggerModel {
		tr := &models.TriggerModel{ID: uuid.New(), WorkflowID: wf, Type: typ, Config: models.JSONBMap{}, Enabled: enabled}
		require.NoError(t, repo.Create(ctx, tr))
		return tr
	}
	mk(ownedWF.ID, "cron", true)
	disabled := mk(ownedWF.ID, "manual", false)
	mk(otherWF.ID, "cron", true)

	got, err := repo.FindByID(ctx, disabled.ID)
	require.NoError(t, err)
	assert.False(t, got.Enabled, "an explicitly disabled trigger must stay disabled")

	cron := "cron"
	cases := []struct {
		name    string
		filters repository.TriggerFilters
		want    int
	}{
		{"none", repository.TriggerFilters{}, 3},
		{"workflow", repository.TriggerFilters{WorkflowID: &ownedWF.ID}, 2},
		{"type", repository.TriggerFilters{Type: &cron}, 2},
		{"owner", repository.TriggerFilters{OwnerID: &owner}, 2},
		{"owner+type", repository.TriggerFilters{OwnerID: &owner, Type: &cron}, 1},
	}
	for _, c := range cases {
		list, err := repo.FindWithFilters(ctx, c.filters, 10, 0)
		require.NoError(t, err, c.name)
		assert.Len(t, list, c.want, c.name)
		n, err := repo.CountWithFilters(ctx, c.filters)
		require.NoError(t, err, c.name)
		assert.Equal(t, c.want, n, c.name)
	}
	page, err := repo.FindWithFilters(ctx, repository.TriggerFilters{}, 2, 2)
	require.NoError(t, err)
	assert.Len(t, page, 1)
}

// ---------------------------------------------------------------------------
// Storage configs and file queries
// ---------------------------------------------------------------------------

func TestRC_StorageConfigRepository(t *testing.T) {
	t.Parallel()
	db, _ := testutil.SetupTestTx(t)
	ctx := context.Background()
	repo := NewStorageConfigRepository(db)

	ttl := 60
	c1 := &models.StorageConfigModel{StorageID: "s1", StorageType: "local", Config: models.JSONBMap{"root": "/tmp"}, MaxSize: 100, MaxFileSize: 10, DefaultTTLSeconds: &ttl}
	require.NoError(t, repo.Create(ctx, c1))
	assert.NotEqual(t, uuid.Nil, c1.ID)
	require.NoError(t, repo.Create(ctx, &models.StorageConfigModel{StorageID: "s2"}))
	assert.Error(t, repo.Create(ctx, &models.StorageConfigModel{StorageID: "s1"}), "storage_id is unique")

	got, err := repo.FindByStorageID(ctx, "s1")
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, "/tmp", got.Config["root"])
	assert.Equal(t, int64(100), got.MaxSize)
	require.NotNil(t, got.DefaultTTLSeconds)
	assert.Equal(t, 60, *got.DefaultTTLSeconds)

	s2, err := repo.FindByStorageID(ctx, "s2")
	require.NoError(t, err)
	require.NotNil(t, s2)
	assert.Equal(t, "local", s2.StorageType, "DB default applies")

	missing, err := repo.FindByStorageID(ctx, "nope")
	require.NoError(t, err)
	assert.Nil(t, missing)

	all, err := repo.FindAll(ctx)
	require.NoError(t, err)
	require.Len(t, all, 2)
	assert.Equal(t, "s1", all[0].StorageID)

	require.NoError(t, repo.Delete(ctx, "s1"))
	all, err = repo.FindAll(ctx)
	require.NoError(t, err)
	assert.Len(t, all, 1)
}

func TestRC_FileRepository_QueryFiltersOwnershipAndOrdering(t *testing.T) {
	t.Parallel()
	db, _ := testutil.SetupTestTx(t)
	ctx := context.Background()
	repo := NewFileRepository(db)

	owner := uuid.MustParse(createCredentialsTestUser(t, db))
	other := uuid.MustParse(createCredentialsTestUser(t, db))
	ownedWF := rcWorkflow("owned", &owner)
	require.NoError(t, NewWorkflowRepository(db).Create(ctx, ownedWF))
	exec := &models.ExecutionModel{ID: uuid.New(), WorkflowID: &ownedWF.ID, Status: "running"}
	require.NoError(t, NewExecutionRepository(db).Create(ctx, exec))

	past := time.Now().Add(-time.Hour)
	future := time.Now().Add(time.Hour)
	mk := func(name, mime, scope string, size int64, mut func(f *models.FileModel)) *models.FileModel {
		f := &models.FileModel{StorageID: "st", Name: name, Path: "/" + name, MimeType: mime, Size: size, Checksum: "c", AccessScope: scope}
		if mut != nil {
			mut(f)
		}
		require.NoError(t, repo.Create(ctx, f))
		return f
	}
	fOwned := mk("a.txt", "text/plain", "workflow", 3, func(f *models.FileModel) { f.CreatedBy = &owner })
	fWF := mk("b.png", "image/png", "result", 1, func(f *models.FileModel) { f.WorkflowID = &ownedWF.ID; f.ExecutionID = &exec.ID })
	fOther := mk("c.txt", "text/plain", "edge", 2, func(f *models.FileModel) { f.CreatedBy = &other; f.ExpiresAt = &past })
	mk("d.json", "application/json", "workflow", 4, func(f *models.FileModel) { f.ExpiresAt = &future })

	ids := func(files []*models.FileModel) []uuid.UUID {
		out := make([]uuid.UUID, len(files))
		for i, f := range files {
			out[i] = f.ID
		}
		return out
	}

	byOwner, err := repo.FindByQuery(ctx, &FileQuery{OwnerID: &owner})
	require.NoError(t, err)
	assert.ElementsMatch(t, []uuid.UUID{fOwned.ID, fWF.ID}, ids(byOwner), "own uploads plus files of own workflows")
	n, err := repo.CountByQuery(ctx, &FileQuery{OwnerID: &owner})
	require.NoError(t, err)
	assert.Equal(t, 2, n)

	byOther, err := repo.FindByQuery(ctx, &FileQuery{OwnerID: &other})
	require.NoError(t, err)
	assert.Equal(t, []uuid.UUID{fOther.ID}, ids(byOther))

	byExec, err := repo.FindByQuery(ctx, &FileQuery{ExecutionID: &exec.ID})
	require.NoError(t, err)
	assert.Equal(t, []uuid.UUID{fWF.ID}, ids(byExec))
	n, err = repo.CountByQuery(ctx, &FileQuery{ExecutionID: &exec.ID, WorkflowID: &ownedWF.ID, StorageID: "st"})
	require.NoError(t, err)
	assert.Equal(t, 1, n)

	byMime, err := repo.FindByQuery(ctx, &FileQuery{MimeTypes: []string{"text/plain", "image/png"}})
	require.NoError(t, err)
	assert.Len(t, byMime, 3)

	byScope, err := repo.FindByQuery(ctx, &FileQuery{AccessScope: "workflow"})
	require.NoError(t, err)
	assert.Len(t, byScope, 2)
	n, err = repo.CountByQuery(ctx, &FileQuery{AccessScope: "edge"})
	require.NoError(t, err)
	assert.Equal(t, 1, n)

	yes, no := true, false
	expired, err := repo.FindByQuery(ctx, &FileQuery{Expired: &yes})
	require.NoError(t, err)
	assert.Equal(t, []uuid.UUID{fOther.ID}, ids(expired))
	notExpired, err := repo.FindByQuery(ctx, &FileQuery{Expired: &no})
	require.NoError(t, err)
	assert.Len(t, notExpired, 3)

	bySizeAsc, err := repo.FindByQuery(ctx, &FileQuery{OrderBy: "size", OrderDir: "asc"})
	require.NoError(t, err)
	require.Len(t, bySizeAsc, 4)
	for i, want := range []int64{1, 2, 3, 4} {
		assert.Equal(t, want, bySizeAsc[i].Size)
	}
	byNameDesc, err := repo.FindByQuery(ctx, &FileQuery{OrderBy: "NAME", OrderDir: "desc"})
	require.NoError(t, err)
	assert.Equal(t, "d.json", byNameDesc[0].Name)

	// Untrusted ordering input must never reach the SQL text.
	injected, err := repo.FindByQuery(ctx, &FileQuery{
		OrderBy:  "(SELECT CASE WHEN (SELECT count(*) FROM mbflow_users) > 0 THEN size ELSE 1/0 END)",
		OrderDir: "; DROP TABLE mbflow_files; --",
	})
	require.NoError(t, err)
	assert.Len(t, injected, 4)
	assert.Equal(t, 4, rcCountRows(t, db, "mbflow_files"))
}

func TestRC_FileOrderClause(t *testing.T) {
	cases := []struct{ by, dir, wantCol, wantDir string }{
		{"", "", "f.created_at", "DESC"},
		{"created_at", "ASC", "f.created_at", "ASC"},
		{" Size ", " asc ", "f.size", "ASC"},
		{"updated_at", "DESC", "f.updated_at", "DESC"},
		{"mime_type", "bogus", "f.mime_type", "DESC"},
		{"expires_at", "", "f.expires_at", "DESC"},
		{"name", "", "f.name", "DESC"},
		{"id; DROP TABLE x", "ASC", "f.created_at", "ASC"},
	}
	for _, c := range cases {
		col, dir := fileOrderClause(c.by, c.dir)
		assert.Equal(t, c.wantCol, col, c.by)
		assert.Equal(t, c.wantDir, dir, c.dir)
	}
}

// ---------------------------------------------------------------------------
// Event streaming
// ---------------------------------------------------------------------------

func TestRC_EventRepository_StreamDeliversExistingAndNewEvents(t *testing.T) {
	t.Parallel()
	db, _ := testutil.SetupTestTx(t)
	repo := NewEventRepository(db)
	exec := createTestExecution(t, db)

	bg := context.Background()
	require.NoError(t, repo.Append(bg, &models.EventModel{ExecutionID: exec.ID, EventType: "node.started"}))
	require.NoError(t, repo.Append(bg, &models.EventModel{ExecutionID: exec.ID, EventType: "node.completed"}))

	ctx, cancel := context.WithCancel(bg)
	defer cancel()
	events, errs := repo.Stream(ctx, exec.ID, 0)

	recv := func() *models.EventModel {
		select {
		case ev := <-events:
			return ev
		case err := <-errs:
			t.Fatalf("unexpected stream error: %v", err)
		case <-time.After(5 * time.Second):
			t.Fatal("timed out waiting for event")
		}
		return nil
	}
	first := recv()
	second := recv()
	assert.Equal(t, "node.started", first.EventType)
	assert.Equal(t, "node.completed", second.EventType)

	// a new event is picked up by the polling loop
	require.NoError(t, repo.Append(bg, &models.EventModel{ExecutionID: exec.ID, EventType: "execution.completed"}))
	third := recv()
	assert.Equal(t, "execution.completed", third.EventType)
	assert.Greater(t, third.Sequence, second.Sequence)

	cancel()
	select {
	case err := <-errs:
		assert.ErrorIs(t, err, context.Canceled)
	case <-time.After(5 * time.Second):
		t.Fatal("stream did not stop after cancel")
	}
	_, open := <-events
	assert.False(t, open, "event channel is closed after the stream stops")
}

func TestRC_EventRepository_StreamCancelledBeforeDelivery(t *testing.T) {
	t.Parallel()
	db, _ := testutil.SetupTestTx(t)
	repo := NewEventRepository(db)
	exec := createTestExecution(t, db)

	bg := context.Background()
	// more events than the channel buffer, none consumed
	for i := 0; i < 12; i++ {
		require.NoError(t, repo.Append(bg, &models.EventModel{ExecutionID: exec.ID, EventType: "node.started"}))
	}
	ctx, cancel := context.WithCancel(bg)
	events, errs := repo.Stream(ctx, exec.ID, 0)

	// wait until the buffer is full, then cancel while the producer is blocked
	require.Eventually(t, func() bool { return len(events) == cap(events) }, 5*time.Second, 10*time.Millisecond)
	cancel()
	select {
	case err := <-errs:
		assert.ErrorIs(t, err, context.Canceled)
	case <-time.After(5 * time.Second):
		t.Fatal("stream did not stop after cancel")
	}
}
