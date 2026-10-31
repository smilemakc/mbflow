package engine

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/smilemakc/mbflow/go/internal/application/observer"
	"github.com/smilemakc/mbflow/go/internal/domain/repository"
	storagemodels "github.com/smilemakc/mbflow/go/internal/infrastructure/storage/models"
	"github.com/smilemakc/mbflow/go/pkg/executor"
	"github.com/smilemakc/mbflow/go/pkg/models"
	"github.com/stretchr/testify/require"
)

// Test doubles for the stored-execution path. They embed the repository
// interfaces so only the methods the engine uses need implementing.

type fakeWorkflowRepo struct {
	repository.WorkflowRepository
	wf  *storagemodels.WorkflowModel
	err error
}

func (f *fakeWorkflowRepo) FindByIDWithRelations(_ context.Context, id uuid.UUID) (*storagemodels.WorkflowModel, error) {
	if f.err != nil {
		return nil, f.err
	}
	if f.wf == nil || f.wf.ID != id {
		return nil, errors.New("workflow not found")
	}
	return f.wf, nil
}

type fakeExecutionRepo struct {
	repository.ExecutionRepository

	mu       sync.Mutex
	rows     map[uuid.UUID]storagemodels.ExecutionModel
	history  map[uuid.UUID][]string
	updates  int
	failNext []error // errors returned by subsequent Update calls, in order

	createErr   error
	panicUpdate bool // first Update panics
}

func newFakeExecutionRepo() *fakeExecutionRepo {
	return &fakeExecutionRepo{
		rows:    map[uuid.UUID]storagemodels.ExecutionModel{},
		history: map[uuid.UUID][]string{},
	}
}

func (f *fakeExecutionRepo) Create(_ context.Context, m *storagemodels.ExecutionModel) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.createErr != nil {
		return f.createErr
	}
	f.rows[m.ID] = *m
	f.history[m.ID] = append(f.history[m.ID], m.Status)
	return nil
}

func (f *fakeExecutionRepo) Update(_ context.Context, m *storagemodels.ExecutionModel) error {
	f.mu.Lock()
	f.updates++
	if f.panicUpdate {
		f.panicUpdate = false
		f.mu.Unlock()
		panic("boom in update")
	}
	defer f.mu.Unlock()
	if len(f.failNext) > 0 {
		err := f.failNext[0]
		f.failNext = f.failNext[1:]
		if err != nil {
			return err
		}
	}
	f.rows[m.ID] = *m
	f.history[m.ID] = append(f.history[m.ID], m.Status)
	return nil
}

func (f *fakeExecutionRepo) get(id string) (storagemodels.ExecutionModel, bool) {
	f.mu.Lock()
	defer f.mu.Unlock()
	row, ok := f.rows[uuid.MustParse(id)]
	return row, ok
}

func (f *fakeExecutionRepo) status(id string) string {
	row, _ := f.get(id)
	return row.Status
}

func (f *fakeExecutionRepo) statusHistory(id string) []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]string(nil), f.history[uuid.MustParse(id)]...)
}

func (f *fakeExecutionRepo) count() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.rows)
}

// fakeLeaseRepo adds lease support to fakeExecutionRepo.
type fakeLeaseRepo struct {
	*fakeExecutionRepo

	leaseMu    sync.Mutex
	touched    map[uuid.UUID]int
	sweeps     int
	staleAfter time.Duration
	errMsg     string
	staleIDs   []uuid.UUID
	touchErr   error
	sweepErr   error
}

func newFakeLeaseRepo() *fakeLeaseRepo {
	return &fakeLeaseRepo{fakeExecutionRepo: newFakeExecutionRepo(), touched: map[uuid.UUID]int{}}
}

func (f *fakeLeaseRepo) TouchExecutions(_ context.Context, ids []uuid.UUID) (int, error) {
	f.leaseMu.Lock()
	defer f.leaseMu.Unlock()
	if f.touchErr != nil {
		return 0, f.touchErr
	}
	for _, id := range ids {
		f.touched[id]++
	}
	return len(ids), nil
}

func (f *fakeLeaseRepo) FailStaleExecutions(_ context.Context, staleAfter time.Duration, errMsg string) ([]uuid.UUID, error) {
	f.leaseMu.Lock()
	defer f.leaseMu.Unlock()
	f.sweeps++
	f.staleAfter = staleAfter
	f.errMsg = errMsg
	if f.sweepErr != nil {
		return nil, f.sweepErr
	}
	ids := f.staleIDs
	f.staleIDs = nil
	return ids, nil
}

func (f *fakeLeaseRepo) touchCount(id string) int {
	f.leaseMu.Lock()
	defer f.leaseMu.Unlock()
	return f.touched[uuid.MustParse(id)]
}

func (f *fakeLeaseRepo) sweepCount() int {
	f.leaseMu.Lock()
	defer f.leaseMu.Unlock()
	return f.sweeps
}

type fakeResourceRepo struct {
	repository.ResourceRepository
	resources map[string]models.Resource
}

func (f *fakeResourceRepo) GetByID(_ context.Context, id string) (models.Resource, error) {
	if r, ok := f.resources[id]; ok {
		return r, nil
	}
	return nil, errors.New("resource not found")
}

// blockingExecutor blocks every node until release is closed or the
// execution context ends; started receives one value per node start.
type blockingExecutor struct {
	started chan string
	release chan struct{}
}

func newBlockingExecutor() *blockingExecutor {
	return &blockingExecutor{started: make(chan string, 16), release: make(chan struct{})}
}

func (b *blockingExecutor) Execute(ctx context.Context, config map[string]any, _ any) (any, error) {
	id, _ := config["nodeID"].(string)
	b.started <- id
	select {
	case <-b.release:
		return map[string]any{"done": id}, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func (b *blockingExecutor) Validate(map[string]any) error { return nil }

func waitStarted(t *testing.T, b *blockingExecutor) string {
	t.Helper()
	select {
	case id := <-b.started:
		return id
	case <-time.After(5 * time.Second):
		t.Fatal("node did not start")
		return ""
	}
}

// storedWorkflow returns a two-node workflow model (n1 -> n2) of nodeType.
func storedWorkflow(nodeType string) *storagemodels.WorkflowModel {
	return &storagemodels.WorkflowModel{
		ID:        uuid.New(),
		Name:      "stored",
		Variables: storagemodels.JSONBMap{"wfvar": "w"},
		Nodes: []*storagemodels.NodeModel{
			{ID: uuid.New(), NodeID: "n1", Name: "First", Type: nodeType, Config: storagemodels.JSONBMap{"nodeID": "n1"}},
			{ID: uuid.New(), NodeID: "n2", Name: "Second", Type: nodeType, Config: storagemodels.JSONBMap{"nodeID": "n2"}},
		},
		Edges: []*storagemodels.EdgeModel{{EdgeID: "e1", FromNodeID: "n1", ToNodeID: "n2"}},
	}
}

type storedFixture struct {
	em        *ExecutionManager
	wf        *storagemodels.WorkflowModel
	execRepo  *fakeExecutionRepo
	resources *fakeResourceRepo
	recorder  *recordingObserver
	om        *observer.ObserverManager
}

// newStoredFixture builds an ExecutionManager through NewExecutionManager with
// fake repositories; execRepo may be a *fakeExecutionRepo or *fakeLeaseRepo.
func newStoredFixture(t *testing.T, exec executor.Executor, execRepo repository.ExecutionRepository) *storedFixture {
	t.Helper()
	registry := executor.NewManager()
	require.NoError(t, registry.Register("mock", exec))

	wf := storedWorkflow("mock")
	recorder := &recordingObserver{}
	om := observer.NewObserverManager()
	require.NoError(t, om.Register(recorder))
	resources := &fakeResourceRepo{resources: map[string]models.Resource{}}

	em := NewExecutionManager(registry, &fakeWorkflowRepo{wf: wf}, execRepo, nil, resources, om, NewEphemeralStreamRegistry(time.Minute))
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = em.Shutdown(ctx)
	})

	var base *fakeExecutionRepo
	switch r := execRepo.(type) {
	case *fakeExecutionRepo:
		base = r
	case *fakeLeaseRepo:
		base = r.fakeExecutionRepo
	}
	return &storedFixture{em: em, wf: wf, execRepo: base, resources: resources, recorder: recorder, om: om}
}

func (r *recordingObserver) types(executionID string) []observer.EventType {
	var out []observer.EventType
	for _, e := range r.snapshot() {
		if e.ExecutionID == executionID {
			out = append(out, e.Type)
		}
	}
	return out
}

func eventually(t *testing.T, cond func() bool, msg string) {
	t.Helper()
	require.Eventually(t, cond, 5*time.Second, 5*time.Millisecond, msg)
}
