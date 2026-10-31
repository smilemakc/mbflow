package serviceapi

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/smilemakc/mbflow/go/internal/application/engine"
	"github.com/smilemakc/mbflow/go/internal/domain/repository"
	storagemodels "github.com/smilemakc/mbflow/go/internal/infrastructure/storage/models"
	"github.com/smilemakc/mbflow/go/pkg/crypto"
	"github.com/smilemakc/mbflow/go/pkg/executor"
	"github.com/smilemakc/mbflow/go/pkg/models"
)

var rcErr = errors.New("rc failure")

const rcType = "rc-echo"

func rcExecutors(fail bool) executor.Manager {
	m := executor.NewManager()
	_ = m.Register(rcType, &executor.ExecutorFunc{
		ExecuteFn: func(_ context.Context, config map[string]any, _ any) (any, error) {
			if fail {
				return nil, errors.New("node exploded")
			}
			return map[string]any{"echo": config["value"]}, nil
		},
	})
	return m
}

// ---------------------------------------------------------------------------
// Executions
// ---------------------------------------------------------------------------

func TestRCStartExecution_ValidatesWebhooks(t *testing.T) {
	ops := newTestOperations(nil, nil, nil, nil, nil, nil, nil)
	ctx := context.Background()

	_, err := ops.StartExecution(ctx, StartExecutionParams{WorkflowID: "x", Webhooks: []WebhookSubscription{{URL: ""}}})
	var opErr *OperationError
	require.ErrorAs(t, err, &opErr)
	assert.Equal(t, "INVALID_WEBHOOK", opErr.Code)
	assert.Contains(t, opErr.Message, "url is required")

	_, err = ops.StartExecution(ctx, StartExecutionParams{WorkflowID: "x", Webhooks: []WebhookSubscription{{URL: "http://[::1"}}})
	require.ErrorAs(t, err, &opErr)
	assert.Contains(t, opErr.Message, "invalid url")

	_, err = ops.StartExecution(ctx, StartExecutionParams{WorkflowID: "x", Webhooks: []WebhookSubscription{{URL: "https://h/x", Events: []string{"node.completed", "bogus"}}}})
	require.ErrorAs(t, err, &opErr)
	assert.Contains(t, opErr.Message, `unknown event type "bogus"`)
}

func TestRCStartExecution_Async(t *testing.T) {
	ctx := context.Background()
	wfRepo := new(mockWorkflowRepo)
	execRepo := new(mockExecutionRepo)

	wfID := uuid.New()
	wfModel := &storagemodels.WorkflowModel{
		ID: wfID, Name: "async", Status: "active", Version: 1,
		Variables: storagemodels.JSONBMap{"base": "1"},
		Nodes: []*storagemodels.NodeModel{
			{ID: uuid.New(), NodeID: "n1", WorkflowID: wfID, Name: "N1", Type: rcType, Config: storagemodels.JSONBMap{"value": "v"}},
		},
	}
	wfRepo.On("FindByIDWithRelations", mock.Anything, wfID).Return(wfModel, nil)
	wfRepo.On("FindByIDWithRelations", mock.Anything, mock.Anything).Return(nil, models.ErrWorkflowNotFound)

	var mu sync.Mutex
	var created *storagemodels.ExecutionModel
	var finalStatus string
	execRepo.On("Create", mock.Anything, mock.Anything).Run(func(args mock.Arguments) {
		mu.Lock()
		defer mu.Unlock()
		created = args.Get(1).(*storagemodels.ExecutionModel)
	}).Return(nil)
	execRepo.On("Update", mock.Anything, mock.Anything).Run(func(args mock.Arguments) {
		mu.Lock()
		defer mu.Unlock()
		finalStatus = args.Get(1).(*storagemodels.ExecutionModel).Status
	}).Return(nil).Maybe()
	execRepo.On("CreateNodeExecution", mock.Anything, mock.Anything).Return(nil).Maybe()
	execRepo.On("UpdateNodeExecution", mock.Anything, mock.Anything).Return(nil).Maybe()
	execRepo.On("FindNodeExecutionsByExecutionID", mock.Anything, mock.Anything).Return(nil, nil).Maybe()

	executors := rcExecutors(false)
	ops := newTestOperations(wfRepo, execRepo, nil, nil, nil, nil, executors)
	em := engine.NewExecutionManager(executors, wfRepo, execRepo, nil, nil, nil)
	ops.ExecutionMgr = em

	exec, err := ops.StartExecution(ctx, StartExecutionParams{
		WorkflowID: wfID.String(),
		Input:      map[string]any{"in": 1},
		Variables:  map[string]any{"extra": "2"},
		Webhooks:   []WebhookSubscription{{URL: "https://hooks.invalid/x", Events: []string{"execution.completed"}, Headers: map[string]string{"A": "b"}, NodeIDs: []string{"n1"}}},
	})
	require.NoError(t, err)
	assert.Equal(t, wfID.String(), exec.WorkflowID)
	assert.Equal(t, models.ExecutionStatusPending, exec.Status)
	assert.Equal(t, "1", exec.Variables["base"])
	assert.Equal(t, "2", exec.Variables["extra"])

	shutdownCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	require.NoError(t, em.Shutdown(shutdownCtx))

	mu.Lock()
	defer mu.Unlock()
	require.NotNil(t, created)
	assert.Equal(t, exec.ID, created.ID.String())
	assert.Equal(t, string(models.ExecutionStatusCompleted), finalStatus)

	_, err = ops.StartExecution(ctx, StartExecutionParams{WorkflowID: uuid.NewString()})
	assert.ErrorIs(t, err, models.ErrWorkflowNotFound)
}

func rcEphemeralWorkflow() *models.Workflow {
	return &models.Workflow{
		Name:  "inline",
		Nodes: []*models.Node{{ID: "n1", Name: "N1", Type: rcType, Config: map[string]any{"value": "hi"}}},
		Edges: []*models.Edge{},
	}
}

func TestRCStartEphemeralExecution(t *testing.T) {
	ctx := context.Background()

	t.Run("sync success", func(t *testing.T) {
		ops := newTestOperations(nil, nil, nil, nil, nil, nil, nil)
		ops.ExecutionMgr = engine.NewExecutionManager(rcExecutors(false), nil, nil, nil, nil, nil)
		exec, err := ops.StartEphemeralExecution(ctx, EphemeralExecutionParams{
			Workflow: rcEphemeralWorkflow(), Mode: "sync", Input: map[string]any{"a": 1},
			Webhooks: []WebhookSubscription{{URL: "https://hooks.invalid/x"}},
		})
		require.NoError(t, err)
		assert.Equal(t, models.ExecutionStatusCompleted, exec.Status)
		assert.Equal(t, "hi", exec.Output["echo"])
	})
	t.Run("sync workflow failure is a normal result", func(t *testing.T) {
		ops := newTestOperations(nil, nil, nil, nil, nil, nil, nil)
		ops.ExecutionMgr = engine.NewExecutionManager(rcExecutors(true), nil, nil, nil, nil, nil)
		exec, err := ops.StartEphemeralExecution(ctx, EphemeralExecutionParams{Workflow: rcEphemeralWorkflow(), Mode: "sync"})
		require.NoError(t, err)
		require.NotNil(t, exec)
		assert.Equal(t, models.ExecutionStatusFailed, exec.Status)
	})
	t.Run("engine rejects workflow", func(t *testing.T) {
		ops := newTestOperations(nil, nil, nil, nil, nil, nil, nil)
		ops.ExecutionMgr = engine.NewExecutionManager(rcExecutors(false), nil, nil, nil, nil, nil)
		_, err := ops.StartEphemeralExecution(ctx, EphemeralExecutionParams{Workflow: &models.Workflow{}, Mode: "sync"})
		require.Error(t, err)
		var opErr *OperationError
		assert.False(t, errors.As(err, &opErr), "engine errors are passed through")
	})
	t.Run("invalid webhook", func(t *testing.T) {
		ops := newTestOperations(nil, nil, nil, nil, nil, nil, nil)
		_, err := ops.StartEphemeralExecution(ctx, EphemeralExecutionParams{
			Workflow: rcEphemeralWorkflow(), Mode: "async", Webhooks: []WebhookSubscription{{URL: ""}},
		})
		var opErr *OperationError
		require.ErrorAs(t, err, &opErr)
		assert.Equal(t, "INVALID_WEBHOOK", opErr.Code)
	})
	t.Run("parameter validation", func(t *testing.T) {
		ops := newTestOperations(nil, nil, nil, nil, nil, nil, nil)
		cases := map[string]EphemeralExecutionParams{
			"WORKFLOW_REQUIRED":          {Mode: "sync"},
			"MODE_REQUIRED":              {Workflow: rcEphemeralWorkflow()},
			"INVALID_MODE":               {Workflow: rcEphemeralWorkflow(), Mode: "later"},
			"UNSUPPORTED_CREDENTIAL_IDS": {Workflow: rcEphemeralWorkflow(), Mode: "sync", CredentialIDs: []string{"c"}},
		}
		for code, params := range cases {
			_, err := ops.StartEphemeralExecution(ctx, params)
			var opErr *OperationError
			require.ErrorAs(t, err, &opErr, code)
			assert.Equal(t, code, opErr.Code)
		}
	})
}

func TestRCNormalizeEphemeralExecutionParams(t *testing.T) {
	var opErr *OperationError
	require.ErrorAs(t, normalizeEphemeralExecutionParams(nil), &opErr)
	assert.Equal(t, "INVALID_REQUEST", opErr.Code)

	unmarshalable := rcEphemeralWorkflow()
	unmarshalable.Nodes[0].Config["bad"] = make(chan int)
	err := normalizeEphemeralExecutionParams(&EphemeralExecutionParams{Workflow: unmarshalable, Mode: "sync"})
	require.ErrorAs(t, err, &opErr)
	assert.Equal(t, "INVALID_WORKFLOW", opErr.Code)

	huge := rcEphemeralWorkflow()
	huge.Description = strings.Repeat("x", maxEphemeralWorkflowSnapshotSize+1)
	err = normalizeEphemeralExecutionParams(&EphemeralExecutionParams{Workflow: huge, Mode: "sync"})
	require.ErrorAs(t, err, &opErr)
	assert.Equal(t, "WORKFLOW_TOO_LARGE", opErr.Code)
	assert.Equal(t, 413, opErr.HTTPStatus)

	withEdges := rcEphemeralWorkflow()
	withEdges.Edges = []*models.Edge{{From: "a", To: "b"}, {ID: "keep", From: "b", To: "c"}}
	require.NoError(t, normalizeEphemeralExecutionParams(&EphemeralExecutionParams{Workflow: withEdges, Mode: "async"}))
	assert.NotEmpty(t, withEdges.Edges[0].ID, "missing edge IDs are generated")
	assert.Equal(t, "keep", withEdges.Edges[1].ID)
}

func TestRCListExecutions_ByOwner(t *testing.T) {
	execRepo := new(mockExecutionRepo)
	ops := newTestOperations(nil, execRepo, nil, nil, nil, nil, nil)
	owner, wf := uuid.New(), uuid.New()
	status := "completed"
	execRepo.On("FindWithFilters", mock.Anything, repository.ExecutionFilters{WorkflowID: &wf, Status: &status, OwnerID: &owner}, 10, 0).
		Return([]*storagemodels.ExecutionModel{{ID: uuid.New(), Status: "completed"}}, nil)

	res, err := ops.ListExecutions(context.Background(), ListExecutionsParams{Limit: 10, WorkflowID: &wf, Status: &status, OwnerID: &owner})
	require.NoError(t, err)
	assert.Equal(t, 1, res.Total)
	execRepo.AssertExpectations(t)
}

func TestRCGetExecution_RepoError(t *testing.T) {
	execRepo := new(mockExecutionRepo)
	ops := newTestOperations(nil, execRepo, nil, nil, nil, nil, nil)
	execRepo.On("FindByIDWithRelations", mock.Anything, mock.Anything).Return(nil, rcErr)
	_, err := ops.GetExecution(context.Background(), GetExecutionParams{ExecutionID: uuid.New()})
	assert.ErrorIs(t, err, rcErr)
}

func TestRCGetNodeResult(t *testing.T) {
	ctx := context.Background()
	key := "n1"
	other := "n2"
	inlineID, storedID, orphanID, brokenID := uuid.New(), uuid.New(), uuid.New(), uuid.New()
	wfID := uuid.New()
	nodeUUID := uuid.New()
	started := time.Unix(1700000000, 0)

	execRepo := new(mockExecutionRepo)
	wfRepo := new(mockWorkflowRepo)
	execRepo.On("FindByIDWithRelations", mock.Anything, inlineID).Return(&storagemodels.ExecutionModel{
		ID: inlineID, WorkflowSource: "inline",
		NodeExecutions: []*storagemodels.NodeExecutionModel{
			{ID: uuid.New(), NodeKey: &other, Status: "completed", StartedAt: &started},
			{ID: uuid.New(), NodeKey: &key, Status: "completed", StartedAt: &started, OutputData: storagemodels.JSONBMap{"x": 1}},
		},
	}, nil)
	execRepo.On("FindByIDWithRelations", mock.Anything, storedID).Return(&storagemodels.ExecutionModel{
		ID: storedID, WorkflowSource: "stored", WorkflowID: &wfID,
		NodeExecutions: []*storagemodels.NodeExecutionModel{
			{ID: uuid.New(), Status: "pending"},
			{ID: uuid.New(), NodeID: &nodeUUID, Status: "failed", StartedAt: &started},
		},
	}, nil)
	execRepo.On("FindByIDWithRelations", mock.Anything, orphanID).Return(&storagemodels.ExecutionModel{ID: orphanID, WorkflowSource: "stored"}, nil)
	execRepo.On("FindByIDWithRelations", mock.Anything, brokenID).Return(&storagemodels.ExecutionModel{ID: brokenID, WorkflowSource: "stored", WorkflowID: &wfID}, nil).Once()
	wfRepo.On("FindByIDWithRelations", mock.Anything, wfID).Return(&storagemodels.WorkflowModel{
		ID: wfID, Nodes: []*storagemodels.NodeModel{{ID: nodeUUID, NodeID: "logical"}},
	}, nil)
	ops := newTestOperations(wfRepo, execRepo, nil, nil, nil, nil, nil)

	ne, err := ops.GetNodeResult(ctx, GetNodeResultParams{ExecutionID: inlineID, NodeID: "n1"})
	require.NoError(t, err)
	assert.EqualValues(t, 1, ne.Output["x"])

	_, err = ops.GetNodeResult(ctx, GetNodeResultParams{ExecutionID: inlineID, NodeID: "missing"})
	var opErr *OperationError
	require.ErrorAs(t, err, &opErr)
	assert.Equal(t, "NODE_EXECUTION_NOT_FOUND", opErr.Code)

	ne, err = ops.GetNodeResult(ctx, GetNodeResultParams{ExecutionID: storedID, NodeID: "logical"})
	require.NoError(t, err)
	assert.Equal(t, "logical", ne.NodeID, "the logical node ID is reported")

	_, err = ops.GetNodeResult(ctx, GetNodeResultParams{ExecutionID: storedID, NodeID: "other"})
	require.ErrorAs(t, err, &opErr)

	_, err = ops.GetNodeResult(ctx, GetNodeResultParams{ExecutionID: orphanID, NodeID: "x"})
	require.ErrorAs(t, err, &opErr, "stored execution without workflow")

	wfRepo2 := new(mockWorkflowRepo)
	wfRepo2.On("FindByIDWithRelations", mock.Anything, wfID).Return(nil, rcErr)
	ops.WorkflowRepo = wfRepo2
	_, err = ops.GetNodeResult(ctx, GetNodeResultParams{ExecutionID: brokenID, NodeID: "x"})
	assert.ErrorIs(t, err, rcErr)
}

// ---------------------------------------------------------------------------
// Triggers
// ---------------------------------------------------------------------------

type rcSyncer struct {
	mu  sync.Mutex
	ids []string
	err error
}

func (s *rcSyncer) SyncTrigger(_ context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.ids = append(s.ids, id)
	return s.err
}

func TestRCTriggers_OwnerListAndSyncErrors(t *testing.T) {
	ctx := context.Background()
	trigRepo := new(mockTriggerRepo)
	ops := newTestOperations(nil, nil, trigRepo, nil, nil, nil, nil)
	owner := uuid.New()
	filters := repository.TriggerFilters{OwnerID: &owner}
	trigRepo.On("FindWithFilters", mock.Anything, filters, 5, 0).Return([]*storagemodels.TriggerModel{{ID: uuid.New(), Type: "manual"}}, nil)
	trigRepo.On("CountWithFilters", mock.Anything, filters).Return(42, nil)

	res, err := ops.ListTriggers(ctx, ListTriggersParams{Limit: 5, OwnerID: &owner})
	require.NoError(t, err)
	assert.Len(t, res.Triggers, 1)
	assert.Equal(t, 42, res.Total)

	// Update with an invalid cron config is rejected before persisting.
	id := uuid.New()
	trigRepo.On("FindByID", mock.Anything, id).Return(&storagemodels.TriggerModel{ID: id, Type: "cron", Config: storagemodels.JSONBMap{"schedule": "* * * * *"}}, nil)
	_, err = ops.UpdateTrigger(ctx, UpdateTriggerParams{TriggerID: id, Config: map[string]any{"schedule": "nope"}})
	var opErr *OperationError
	require.ErrorAs(t, err, &opErr)
	assert.Equal(t, "INVALID_TRIGGER_CONFIG", opErr.Code)

	// Scheduler sync failures are logged, not returned.
	syncer := &rcSyncer{err: rcErr}
	ops.TriggerSync = syncer
	trigRepo.On("Delete", mock.Anything, id).Return(nil)
	require.NoError(t, ops.DeleteTrigger(ctx, DeleteTriggerParams{TriggerID: id}))
	assert.Equal(t, []string{id.String()}, syncer.ids)

	// A nil logger is tolerated when the sync fails.
	ops.Logger = nil
	ops.syncTrigger(ctx, "t")
	assert.Len(t, syncer.ids, 2)
}

// ---------------------------------------------------------------------------
// Workflows
// ---------------------------------------------------------------------------

func TestRCCreateWorkflow_Resources(t *testing.T) {
	ctx := context.Background()
	wfRepo := new(mockWorkflowRepo)
	ops := newTestOperations(wfRepo, nil, nil, nil, nil, nil, newMockExecutorManager(rcType))
	var stored *storagemodels.WorkflowModel
	wfRepo.On("Create", mock.Anything, mock.Anything).Run(func(args mock.Arguments) {
		stored = args.Get(1).(*storagemodels.WorkflowModel)
	}).Return(nil)

	res1, res2 := uuid.New(), uuid.New()
	_, err := ops.CreateWorkflow(ctx, CreateWorkflowParams{
		Name: "with resources",
		Resources: []ResourceInput{
			{ResourceID: res1.String(), Alias: "a"},
			{ResourceID: res2.String(), Alias: "b", AccessType: "write"},
		},
	})
	require.NoError(t, err)
	require.Len(t, stored.Resources, 2)
	assert.Equal(t, res1, stored.Resources[0].ResourceID)
	assert.Equal(t, "read", stored.Resources[0].AccessType)
	assert.Equal(t, "write", stored.Resources[1].AccessType)
	assert.Equal(t, stored.ID, stored.Resources[1].WorkflowID)

	_, err = ops.CreateWorkflow(ctx, CreateWorkflowParams{Name: "bad", Resources: []ResourceInput{{ResourceID: "nope"}}})
	var opErr *OperationError
	require.ErrorAs(t, err, &opErr)
	assert.Equal(t, "INVALID_RESOURCE_ID", opErr.Code)

	_, err = ops.CreateWorkflow(ctx, CreateWorkflowParams{Name: "bad", Edges: []EdgeInput{{ID: "e"}}})
	require.ErrorAs(t, err, &opErr)
	assert.Equal(t, "EDGE_VALIDATION_FAILED", opErr.Code)
}

func TestRCUpdateWorkflow_ReplacesGraph(t *testing.T) {
	ctx := context.Background()
	wfRepo := new(mockWorkflowRepo)
	ops := newTestOperations(wfRepo, nil, nil, nil, nil, nil, newMockExecutorManager(rcType))
	id := uuid.New()
	current := &storagemodels.WorkflowModel{ID: id, Name: "old", Status: "draft", Version: 1}
	wfRepo.On("FindByIDWithRelations", mock.Anything, id).Return(current, nil)
	wfRepo.On("Update", mock.Anything, mock.Anything).Return(nil)

	_, err := ops.UpdateWorkflow(ctx, UpdateWorkflowParams{
		WorkflowID: id,
		Metadata:   map[string]any{"m": 1},
		Nodes:      []NodeInput{{ID: "a", Name: "A", Type: rcType}, {ID: "b", Name: "B", Type: rcType}},
		Edges:      []EdgeInput{{ID: "e", From: "a", To: "b", SourceHandle: "h", Condition: map[string]any{"expression": "1"}, Loop: &LoopInput{MaxIterations: 5}}},
	})
	require.NoError(t, err)
	require.Len(t, current.Edges, 1)
	assert.Equal(t, "a", current.Edges[0].FromNodeID)
	assert.Equal(t, "h", current.Edges[0].SourceHandle)
	assert.Equal(t, 5, current.Edges[0].Loop["max_iterations"])
	assert.Equal(t, id, current.Edges[0].WorkflowID)
	assert.EqualValues(t, 1, current.Metadata["m"])

	var opErr *OperationError
	_, err = ops.UpdateWorkflow(ctx, UpdateWorkflowParams{WorkflowID: id, Nodes: []NodeInput{{ID: "x"}}})
	require.ErrorAs(t, err, &opErr)
	assert.Equal(t, "NODE_VALIDATION_FAILED", opErr.Code)
	_, err = ops.UpdateWorkflow(ctx, UpdateWorkflowParams{WorkflowID: id, Edges: []EdgeInput{{ID: "e", From: "a", To: "a"}}})
	require.ErrorAs(t, err, &opErr)
	assert.Equal(t, "EDGE_VALIDATION_FAILED", opErr.Code)
}

func TestRCValidateEdges_LongNodeIDsWithoutNodes(t *testing.T) {
	ops := newTestOperations(nil, nil, nil, nil, nil, nil, newMockExecutorManager())
	long := strings.Repeat("f", 101)
	err := ops.validateEdges([]EdgeInput{{ID: "e", From: long, To: "b"}}, nil)
	assert.ErrorContains(t, err, "from node id too long")
	err = ops.validateEdges([]EdgeInput{{ID: "e", From: "a", To: long}}, nil)
	assert.ErrorContains(t, err, "to node id too long")
}

func TestRCWorkflowResourceErrors(t *testing.T) {
	ctx := context.Background()
	wfRepo := new(mockWorkflowRepo)
	ops := newTestOperations(wfRepo, nil, nil, nil, nil, nil, nil)
	id := uuid.New()
	wfRepo.On("FindByID", mock.Anything, id).Return(&storagemodels.WorkflowModel{ID: id}, nil)
	wfRepo.On("FindByIDWithRelations", mock.Anything, id).Return(&storagemodels.WorkflowModel{ID: id, Status: "active"}, nil)
	wfRepo.On("AssignResource", mock.Anything, id, mock.Anything, mock.Anything).Return(rcErr)
	wfRepo.On("GetWorkflowResources", mock.Anything, id).Return(nil, rcErr)
	wfRepo.On("Update", mock.Anything, mock.Anything).Return(rcErr)

	assert.ErrorIs(t, ops.AttachWorkflowResource(ctx, AttachWorkflowResourceParams{WorkflowID: id, ResourceID: uuid.New()}), rcErr)
	_, err := ops.GetWorkflowResources(ctx, GetWorkflowResourcesParams{WorkflowID: id})
	assert.ErrorIs(t, err, rcErr)
	_, err = ops.UnpublishWorkflow(ctx, UnpublishWorkflowParams{WorkflowID: id})
	assert.ErrorIs(t, err, rcErr)
}

// ---------------------------------------------------------------------------
// Credentials
// ---------------------------------------------------------------------------

func TestRCCredentials_GetAndErrors(t *testing.T) {
	ctx := context.Background()
	credRepo := new(mockCredentialsRepo)
	ops := newTestOperations(nil, nil, nil, credRepo, nil, nil, nil)

	cred := models.NewCredentialsResource("u", "n", models.CredentialTypeAPIKey)
	cred.ID = uuid.NewString()
	cred.EncryptedData = map[string]string{"api_key": "enc"}
	credRepo.On("GetCredentials", mock.Anything, cred.ID).Return(cred, nil)
	credRepo.On("GetCredentials", mock.Anything, "missing").Return(nil, models.ErrResourceNotFound)
	credRepo.On("GetCredentials", mock.Anything, "broken").Return(nil, rcErr)

	info, err := ops.GetCredential(ctx, GetCredentialParams{CredentialID: cred.ID})
	require.NoError(t, err)
	assert.Equal(t, cred.ID, info.ID)
	assert.Equal(t, []string{"api_key"}, info.Fields)

	_, err = ops.GetCredential(ctx, GetCredentialParams{CredentialID: "missing"})
	assert.ErrorIs(t, err, models.ErrResourceNotFound)
	_, err = ops.GetCredential(ctx, GetCredentialParams{CredentialID: "broken"})
	assert.ErrorIs(t, err, rcErr)
	assert.ErrorIs(t, ops.DeleteCredential(ctx, DeleteCredentialParams{CredentialID: "broken"}), rcErr)

	// Credentials disabled (no encryption key)
	ops.EncryptionSvc = nil
	_, err = ops.CreateCredential(ctx, CreateCredentialParams{UserID: "u", CredentialType: "api_key"})
	var opErr *OperationError
	require.ErrorAs(t, err, &opErr)
	assert.Equal(t, 501, opErr.HTTPStatus)

	// Encryption failure
	ops.EncryptionSvc = &crypto.EncryptionService{}
	_, err = ops.CreateCredential(ctx, CreateCredentialParams{UserID: "u", CredentialType: "api_key", Data: map[string]string{"api_key": "x"}})
	assert.ErrorContains(t, err, "encryption failed")
}
