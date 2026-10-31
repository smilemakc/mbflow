package grpc

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/smilemakc/mbflow/go/internal/application/serviceapi"
	"github.com/smilemakc/mbflow/go/internal/application/systemkey"
	"github.com/smilemakc/mbflow/go/internal/config"
	"github.com/smilemakc/mbflow/go/internal/domain/repository"
	"github.com/smilemakc/mbflow/go/internal/infrastructure/logger"
	storagemodels "github.com/smilemakc/mbflow/go/internal/infrastructure/storage/models"
	"github.com/smilemakc/mbflow/go/pkg/crypto"
	"github.com/smilemakc/mbflow/go/pkg/executor"
	"github.com/smilemakc/mbflow/go/pkg/models"
)

var errRCBoom = errors.New("boom")

// The fakes embed the repository interface: methods that a test does not
// expect to be called panic (nil interface), which makes unexpected calls loud.

// ---------------------------------------------------------------------------
// Workflow repository
// ---------------------------------------------------------------------------

type rcWorkflowRepo struct {
	repository.WorkflowRepository

	mu          sync.Mutex
	workflows   map[uuid.UUID]*storagemodels.WorkflowModel
	errs        map[string]error
	lastFilters repository.WorkflowFilters
	created     []*storagemodels.WorkflowModel
	deleted     []uuid.UUID
	unassigned  []uuid.UUID
}

func rcNewWorkflowRepo() *rcWorkflowRepo {
	return &rcWorkflowRepo{workflows: map[uuid.UUID]*storagemodels.WorkflowModel{}, errs: map[string]error{}}
}

func (r *rcWorkflowRepo) setErr(op string, err error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.errs[op] = err
}

func (r *rcWorkflowRepo) put(wf *storagemodels.WorkflowModel) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.workflows[wf.ID] = wf
}

func (r *rcWorkflowRepo) Create(_ context.Context, wf *storagemodels.WorkflowModel) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if err := r.errs["Create"]; err != nil {
		return err
	}
	r.created = append(r.created, wf)
	r.workflows[wf.ID] = wf
	return nil
}

func (r *rcWorkflowRepo) Update(_ context.Context, wf *storagemodels.WorkflowModel) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if err := r.errs["Update"]; err != nil {
		return err
	}
	r.workflows[wf.ID] = wf
	return nil
}

func (r *rcWorkflowRepo) Delete(_ context.Context, id uuid.UUID) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if err := r.errs["Delete"]; err != nil {
		return err
	}
	if _, ok := r.workflows[id]; !ok {
		return models.ErrWorkflowNotFound
	}
	delete(r.workflows, id)
	r.deleted = append(r.deleted, id)
	return nil
}

func (r *rcWorkflowRepo) find(op string, id uuid.UUID) (*storagemodels.WorkflowModel, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if err := r.errs[op]; err != nil {
		return nil, err
	}
	wf, ok := r.workflows[id]
	if !ok {
		return nil, models.ErrWorkflowNotFound
	}
	return wf, nil
}

func (r *rcWorkflowRepo) FindByID(_ context.Context, id uuid.UUID) (*storagemodels.WorkflowModel, error) {
	return r.find("FindByID", id)
}

func (r *rcWorkflowRepo) FindByIDWithRelations(_ context.Context, id uuid.UUID) (*storagemodels.WorkflowModel, error) {
	return r.find("FindByIDWithRelations", id)
}

func (r *rcWorkflowRepo) FindAllWithFilters(_ context.Context, filters repository.WorkflowFilters, _, _ int) ([]*storagemodels.WorkflowModel, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.lastFilters = filters
	if err := r.errs["FindAllWithFilters"]; err != nil {
		return nil, err
	}
	var out []*storagemodels.WorkflowModel
	for _, wf := range r.workflows {
		if filters.Status != nil && wf.Status != *filters.Status {
			continue
		}
		out = append(out, wf)
	}
	return out, nil
}

func (r *rcWorkflowRepo) CountWithFilters(ctx context.Context, filters repository.WorkflowFilters) (int, error) {
	wfs, err := r.FindAllWithFilters(ctx, filters, 0, 0)
	return len(wfs) + 100, err // +100 proves the count comes from the repository
}

func (r *rcWorkflowRepo) UnassignResourceFromAllWorkflows(_ context.Context, id uuid.UUID) (int64, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if err := r.errs["Unassign"]; err != nil {
		return 0, err
	}
	r.unassigned = append(r.unassigned, id)
	return 2, nil
}

// ---------------------------------------------------------------------------
// Execution repository
// ---------------------------------------------------------------------------

type rcExecutionRepo struct {
	repository.ExecutionRepository

	mu         sync.Mutex
	executions map[uuid.UUID]*storagemodels.ExecutionModel
	events     map[uuid.UUID][]*storagemodels.EventModel
	errs       map[string]error
	// findByIDCalls counts FindByID calls; failFindByIDAfter > 0 makes the
	// call with that index (1-based) and later ones fail.
	findByIDCalls     int
	failFindByIDAfter int
	listedBy          string
}

func rcNewExecutionRepo() *rcExecutionRepo {
	return &rcExecutionRepo{
		executions: map[uuid.UUID]*storagemodels.ExecutionModel{},
		events:     map[uuid.UUID][]*storagemodels.EventModel{},
		errs:       map[string]error{},
	}
}

func (r *rcExecutionRepo) setErr(op string, err error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.errs[op] = err
}

func (r *rcExecutionRepo) put(ex *storagemodels.ExecutionModel) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.executions[ex.ID] = ex
}

func (r *rcExecutionRepo) setStatus(id uuid.UUID, status string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.executions[id].Status = status
}

func (r *rcExecutionRepo) addEvent(execID uuid.UUID, seq int64, eventType string, payload map[string]any) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.events[execID] = append(r.events[execID], &storagemodels.EventModel{
		ID:          uuid.New(),
		ExecutionID: execID,
		EventType:   eventType,
		Sequence:    seq,
		Payload:     storagemodels.JSONBMap(payload),
		CreatedAt:   time.Unix(1700000000+seq, 0),
	})
}

func (r *rcExecutionRepo) FindByID(_ context.Context, id uuid.UUID) (*storagemodels.ExecutionModel, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.findByIDCalls++
	if r.failFindByIDAfter > 0 && r.findByIDCalls >= r.failFindByIDAfter {
		return nil, errRCBoom
	}
	if err := r.errs["FindByID"]; err != nil {
		return nil, err
	}
	ex, ok := r.executions[id]
	if !ok {
		return nil, errors.New("execution not found")
	}
	cp := *ex
	return &cp, nil
}

func (r *rcExecutionRepo) FindByIDWithRelations(_ context.Context, id uuid.UUID) (*storagemodels.ExecutionModel, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	ex, ok := r.executions[id]
	if !ok {
		return nil, models.ErrExecutionNotFound
	}
	return ex, nil
}

func (r *rcExecutionRepo) list(by string) ([]*storagemodels.ExecutionModel, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.listedBy = by
	if err := r.errs["List"]; err != nil {
		return nil, err
	}
	var out []*storagemodels.ExecutionModel
	for _, ex := range r.executions {
		out = append(out, ex)
	}
	return out, nil
}

func (r *rcExecutionRepo) FindByWorkflowID(_ context.Context, _ uuid.UUID, _, _ int) ([]*storagemodels.ExecutionModel, error) {
	return r.list("workflow")
}

func (r *rcExecutionRepo) FindByStatus(_ context.Context, _ string, _, _ int) ([]*storagemodels.ExecutionModel, error) {
	return r.list("status")
}

func (r *rcExecutionRepo) FindAll(_ context.Context, _, _ int) ([]*storagemodels.ExecutionModel, error) {
	return r.list("all")
}

func (r *rcExecutionRepo) GetEvents(_ context.Context, id uuid.UUID) ([]*storagemodels.EventModel, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if err := r.errs["GetEvents"]; err != nil {
		return nil, err
	}
	return append([]*storagemodels.EventModel(nil), r.events[id]...), nil
}

// rcEventsSinceRepo additionally implements GetEventsSince, which the stream
// handler prefers over GetEvents.
type rcEventsSinceRepo struct {
	*rcExecutionRepo
	sinceCalls []int64
}

func (r *rcEventsSinceRepo) GetEventsSince(_ context.Context, id uuid.UUID, after int64) ([]*storagemodels.EventModel, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.sinceCalls = append(r.sinceCalls, after)
	var out []*storagemodels.EventModel
	for _, ev := range r.events[id] {
		if ev.Sequence > after {
			out = append(out, ev)
		}
	}
	return out, nil
}

// ---------------------------------------------------------------------------
// Trigger repository
// ---------------------------------------------------------------------------

type rcTriggerRepo struct {
	repository.TriggerRepository

	mu       sync.Mutex
	triggers map[uuid.UUID]*storagemodels.TriggerModel
	errs     map[string]error
	listedBy string
}

func rcNewTriggerRepo() *rcTriggerRepo {
	return &rcTriggerRepo{triggers: map[uuid.UUID]*storagemodels.TriggerModel{}, errs: map[string]error{}}
}

func (r *rcTriggerRepo) setErr(op string, err error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.errs[op] = err
}

func (r *rcTriggerRepo) Create(_ context.Context, t *storagemodels.TriggerModel) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if err := r.errs["Create"]; err != nil {
		return err
	}
	r.triggers[t.ID] = t
	return nil
}

func (r *rcTriggerRepo) Update(_ context.Context, t *storagemodels.TriggerModel) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if err := r.errs["Update"]; err != nil {
		return err
	}
	r.triggers[t.ID] = t
	return nil
}

func (r *rcTriggerRepo) Delete(_ context.Context, id uuid.UUID) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.triggers[id]; !ok {
		return models.ErrTriggerNotFound
	}
	delete(r.triggers, id)
	return nil
}

func (r *rcTriggerRepo) FindByID(_ context.Context, id uuid.UUID) (*storagemodels.TriggerModel, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	t, ok := r.triggers[id]
	if !ok {
		return nil, models.ErrTriggerNotFound
	}
	return t, nil
}

func (r *rcTriggerRepo) list(by string) ([]*storagemodels.TriggerModel, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.listedBy = by
	if err := r.errs["List"]; err != nil {
		return nil, err
	}
	var out []*storagemodels.TriggerModel
	for _, t := range r.triggers {
		out = append(out, t)
	}
	return out, nil
}

func (r *rcTriggerRepo) FindByWorkflowID(_ context.Context, _ uuid.UUID) ([]*storagemodels.TriggerModel, error) {
	return r.list("workflow")
}

func (r *rcTriggerRepo) FindByType(_ context.Context, _ string, _, _ int) ([]*storagemodels.TriggerModel, error) {
	return r.list("type")
}

func (r *rcTriggerRepo) FindAll(_ context.Context, _, _ int) ([]*storagemodels.TriggerModel, error) {
	return r.list("all")
}

func (r *rcTriggerRepo) count() (int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.triggers), nil
}

func (r *rcTriggerRepo) Count(context.Context) (int, error) { return r.count() }
func (r *rcTriggerRepo) CountByWorkflowID(context.Context, uuid.UUID) (int, error) {
	return r.count()
}
func (r *rcTriggerRepo) CountByType(context.Context, string) (int, error) { return r.count() }

// ---------------------------------------------------------------------------
// Credentials repository
// ---------------------------------------------------------------------------

type rcCredentialsRepo struct {
	repository.CredentialsRepository

	mu      sync.Mutex
	creds   map[string]*models.CredentialsResource
	deleted []string
	errs    map[string]error
	listed  string
}

func rcNewCredentialsRepo() *rcCredentialsRepo {
	return &rcCredentialsRepo{creds: map[string]*models.CredentialsResource{}, errs: map[string]error{}}
}

func (r *rcCredentialsRepo) setErr(op string, err error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.errs[op] = err
}

func (r *rcCredentialsRepo) CreateCredentials(_ context.Context, c *models.CredentialsResource) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if c.ID == "" {
		c.ID = uuid.NewString()
	}
	r.creds[c.ID] = c
	return nil
}

func (r *rcCredentialsRepo) GetCredentials(_ context.Context, id string) (*models.CredentialsResource, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	c, ok := r.creds[id]
	if !ok {
		return nil, models.ErrResourceNotFound
	}
	return c, nil
}

func (r *rcCredentialsRepo) GetCredentialsByOwner(_ context.Context, owner string) ([]*models.CredentialsResource, error) {
	return r.byOwner("owner", owner, "")
}

func (r *rcCredentialsRepo) GetCredentialsByProvider(_ context.Context, owner, provider string) ([]*models.CredentialsResource, error) {
	return r.byOwner("provider", owner, provider)
}

func (r *rcCredentialsRepo) byOwner(by, owner, provider string) ([]*models.CredentialsResource, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.listed = by
	if err := r.errs["List"]; err != nil {
		return nil, err
	}
	var out []*models.CredentialsResource
	for _, c := range r.creds {
		if c.OwnerID == owner && (provider == "" || c.Provider == provider) {
			out = append(out, c)
		}
	}
	return out, nil
}

func (r *rcCredentialsRepo) UpdateCredentials(_ context.Context, c *models.CredentialsResource) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if err := r.errs["Update"]; err != nil {
		return err
	}
	r.creds[c.ID] = c
	return nil
}

func (r *rcCredentialsRepo) DeleteCredentials(_ context.Context, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.creds, id)
	r.deleted = append(r.deleted, id)
	return nil
}

// ---------------------------------------------------------------------------
// Audit log repository
// ---------------------------------------------------------------------------

type rcAuditRepo struct {
	mu         sync.Mutex
	logs       []*models.ServiceAuditLog
	lastFilter repository.ServiceAuditLogFilter
	err        error
}

func (r *rcAuditRepo) Create(_ context.Context, l *models.ServiceAuditLog) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.logs = append(r.logs, l)
	return nil
}

func (r *rcAuditRepo) FindAll(_ context.Context, f repository.ServiceAuditLogFilter) ([]*models.ServiceAuditLog, int64, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.lastFilter = f
	if r.err != nil {
		return nil, 0, r.err
	}
	return r.logs, int64(len(r.logs)), nil
}

func (r *rcAuditRepo) DeleteOlderThan(context.Context, time.Time) (int64, error) { return 0, nil }

// ---------------------------------------------------------------------------
// Server wiring
// ---------------------------------------------------------------------------

const rcNodeType = "rc-echo"

type rcEnv struct {
	srv       *ServiceAPIServer
	ops       *serviceapi.Operations
	workflows *rcWorkflowRepo
	execs     *rcExecutionRepo
	triggers  *rcTriggerRepo
	creds     *rcCredentialsRepo
	audit     *rcAuditRepo
	executors executor.Manager
}

func rcLogger() *logger.Logger {
	return logger.New(config.LoggingConfig{Level: "error", Format: "json"})
}

func rcNewEnv() *rcEnv {
	execs := executor.NewManager()
	_ = execs.Register(rcNodeType, &executor.ExecutorFunc{
		ExecuteFn: func(_ context.Context, config map[string]any, input any) (any, error) {
			return map[string]any{"echo": config["value"]}, nil
		},
	})

	key := make([]byte, crypto.AES256KeySize)
	enc, _ := crypto.NewEncryptionService(key)

	env := &rcEnv{
		workflows: rcNewWorkflowRepo(),
		execs:     rcNewExecutionRepo(),
		triggers:  rcNewTriggerRepo(),
		creds:     rcNewCredentialsRepo(),
		audit:     &rcAuditRepo{},
		executors: execs,
	}
	env.ops = &serviceapi.Operations{
		WorkflowRepo:    env.workflows,
		ExecutionRepo:   env.execs,
		TriggerRepo:     env.triggers,
		CredentialsRepo: env.creds,
		ExecutorManager: execs,
		EncryptionSvc:   enc,
		AuditService:    systemkey.NewAuditService(env.audit, 30),
		Logger:          rcLogger(),
	}
	env.srv = NewServiceAPIServer(env.ops)
	return env
}
