package grpc

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"

	pb "github.com/smilemakc/mbflow/go/api/proto/serviceapipb"
	"github.com/smilemakc/mbflow/go/internal/application/engine"
	storagemodels "github.com/smilemakc/mbflow/go/internal/infrastructure/storage/models"
	"github.com/smilemakc/mbflow/go/pkg/models"
)

func rcCode(t *testing.T, err error) codes.Code {
	t.Helper()
	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok, "expected a gRPC status error, got %v", err)
	return st.Code()
}

func rcStruct(t *testing.T, m map[string]any) *structpb.Struct {
	t.Helper()
	s, err := structpb.NewStruct(m)
	require.NoError(t, err)
	return s
}

func rcWorkflowModel(name, status string) *storagemodels.WorkflowModel {
	id := uuid.New()
	return &storagemodels.WorkflowModel{
		ID:        id,
		Name:      name,
		Status:    status,
		Version:   1,
		CreatedAt: time.Unix(1700000000, 0),
		UpdatedAt: time.Unix(1700000000, 0),
		Nodes: []*storagemodels.NodeModel{
			{ID: uuid.New(), NodeID: "a", WorkflowID: id, Name: "A", Type: rcNodeType, Position: storagemodels.JSONBMap{"x": 1.0, "y": 2.0}},
			{ID: uuid.New(), NodeID: "b", WorkflowID: id, Name: "B", Type: rcNodeType},
		},
		Edges: []*storagemodels.EdgeModel{
			{ID: uuid.New(), EdgeID: "e1", WorkflowID: id, FromNodeID: "a", ToNodeID: "b",
				Condition: storagemodels.JSONBMap{"expression": "x > 1"},
				Loop:      storagemodels.JSONBMap{"max_iterations": 3}},
		},
	}
}

func TestRCNewServiceAPIServer(t *testing.T) {
	env := rcNewEnv()
	assert.Same(t, env.ops, env.srv.ops)
}

// ---------------------------------------------------------------------------
// Workflows
// ---------------------------------------------------------------------------

func TestRCWorkflows_ListAndGet(t *testing.T) {
	ctx := context.Background()
	env := rcNewEnv()
	wf := rcWorkflowModel("one", "active")
	env.workflows.put(wf)
	env.workflows.put(rcWorkflowModel("two", "draft"))

	resp, err := env.srv.ListWorkflows(ctx, &pb.ListWorkflowsRequest{Status: "active"})
	require.NoError(t, err)
	require.Len(t, resp.Workflows, 1)
	assert.Equal(t, "one", resp.Workflows[0].Name)
	assert.Equal(t, int32(50), resp.Limit, "default limit")
	assert.Equal(t, int32(101), resp.Total)
	assert.True(t, env.workflows.lastFilters.IncludeUnowned)

	user := uuid.New()
	resp, err = env.srv.ListWorkflows(ctx, &pb.ListWorkflowsRequest{UserId: user.String(), Limit: 5, Offset: 1})
	require.NoError(t, err)
	assert.Equal(t, int32(5), resp.Limit)
	assert.Equal(t, int32(1), resp.Offset)
	assert.Equal(t, &user, env.workflows.lastFilters.CreatedBy)
	assert.False(t, env.workflows.lastFilters.IncludeUnowned)

	_, err = env.srv.ListWorkflows(ctx, &pb.ListWorkflowsRequest{UserId: "bad"})
	assert.Equal(t, codes.InvalidArgument, rcCode(t, err))

	env.workflows.setErr("FindAllWithFilters", errRCBoom)
	_, err = env.srv.ListWorkflows(ctx, &pb.ListWorkflowsRequest{})
	assert.Equal(t, codes.Internal, rcCode(t, err))

	got, err := env.srv.GetWorkflow(ctx, &pb.GetWorkflowRequest{Id: wf.ID.String()})
	require.NoError(t, err)
	assert.Equal(t, "one", got.Workflow.Name)
	require.Len(t, got.Workflow.Nodes, 2)
	require.Len(t, got.Workflow.Edges, 1)
	assert.Equal(t, "x > 1", got.Workflow.Edges[0].Condition.AsMap()["expression"])
	assert.Equal(t, int32(3), got.Workflow.Edges[0].Loop.GetMaxIterations())

	_, err = env.srv.GetWorkflow(ctx, &pb.GetWorkflowRequest{Id: "nope"})
	assert.Equal(t, codes.InvalidArgument, rcCode(t, err))
	_, err = env.srv.GetWorkflow(ctx, &pb.GetWorkflowRequest{Id: uuid.NewString()})
	assert.Equal(t, codes.NotFound, rcCode(t, err))
}

func rcGraph(t *testing.T) ([]*pb.Node, []*pb.Edge, []*pb.ResourceAttachment) {
	nodes := []*pb.Node{
		{Id: "a", Name: "A", Type: rcNodeType, Config: rcStruct(t, map[string]any{"value": 1}), Position: rcStruct(t, map[string]any{"x": 1, "y": 2})},
		{Id: "b", Name: "B", Type: rcNodeType},
	}
	edges := []*pb.Edge{
		{Id: "e1", From: "a", To: "b", SourceHandle: "out", Condition: rcStruct(t, map[string]any{"expression": "true"}), Loop: &pb.EdgeLoopConfig{MaxIterations: 2}},
		{Id: "e2", From: "b", To: "a"},
	}
	resources := []*pb.ResourceAttachment{{ResourceId: uuid.NewString(), Alias: "files"}}
	return nodes, edges, resources
}

func TestRCWorkflows_Create(t *testing.T) {
	env := rcNewEnv()
	user := uuid.New()
	ctx := ContextWithUserID(context.Background(), user.String())
	nodes, edges, resources := rcGraph(t)

	resp, err := env.srv.CreateWorkflow(ctx, &pb.CreateWorkflowRequest{
		Name:        "created",
		Description: "desc",
		Variables:   rcStruct(t, map[string]any{"v": "1"}),
		Metadata:    rcStruct(t, map[string]any{"m": "2"}),
		Nodes:       nodes,
		Edges:       edges,
		Resources:   resources,
	})
	require.NoError(t, err)
	assert.Equal(t, "created", resp.Workflow.Name)
	assert.Equal(t, "draft", resp.Workflow.Status)
	assert.Len(t, resp.Workflow.Nodes, 2)

	require.Len(t, env.workflows.created, 1)
	stored := env.workflows.created[0]
	require.NotNil(t, stored.CreatedBy)
	assert.Equal(t, user, *stored.CreatedBy, "creator comes from the authenticated context")
	assert.Equal(t, "out", stored.Edges[0].SourceHandle)
	assert.EqualValues(t, 2, stored.Edges[0].Loop["max_iterations"])
	require.Len(t, stored.Resources, 1)
	assert.Equal(t, "read", stored.Resources[0].AccessType, "default access type")

	// Non-UUID user in context: no creator recorded
	ctx2 := ContextWithUserID(context.Background(), "not-a-uuid")
	_, err = env.srv.CreateWorkflow(ctx2, &pb.CreateWorkflowRequest{Name: "anon"})
	require.NoError(t, err)
	assert.Nil(t, env.workflows.created[1].CreatedBy)

	_, err = env.srv.CreateWorkflow(context.Background(), &pb.CreateWorkflowRequest{})
	assert.Equal(t, codes.InvalidArgument, rcCode(t, err), "name is required")

	_, err = env.srv.CreateWorkflow(context.Background(), &pb.CreateWorkflowRequest{
		Name: "bad", Nodes: []*pb.Node{{Id: "x", Name: "X", Type: "unknown-type"}},
	})
	assert.Equal(t, codes.InvalidArgument, rcCode(t, err))

	env.workflows.setErr("Create", errRCBoom)
	_, err = env.srv.CreateWorkflow(context.Background(), &pb.CreateWorkflowRequest{Name: "x"})
	assert.Equal(t, codes.Internal, rcCode(t, err))
}

func TestRCWorkflows_UpdateAndDelete(t *testing.T) {
	ctx := context.Background()
	env := rcNewEnv()
	wf := rcWorkflowModel("orig", "draft")
	env.workflows.put(wf)
	nodes, edges, resources := rcGraph(t)

	resp, err := env.srv.UpdateWorkflow(ctx, &pb.UpdateWorkflowRequest{
		Id:          wf.ID.String(),
		Name:        "renamed",
		Description: "d",
		Variables:   rcStruct(t, map[string]any{"k": "v"}),
		Nodes:       nodes,
		Edges:       edges,
		Resources:   resources,
	})
	require.NoError(t, err)
	assert.Equal(t, "renamed", resp.Workflow.Name)
	require.Len(t, resp.Workflow.Edges, 2)

	_, err = env.srv.UpdateWorkflow(ctx, &pb.UpdateWorkflowRequest{Id: "bad"})
	assert.Equal(t, codes.InvalidArgument, rcCode(t, err))
	_, err = env.srv.UpdateWorkflow(ctx, &pb.UpdateWorkflowRequest{Id: uuid.NewString()})
	assert.Equal(t, codes.NotFound, rcCode(t, err))

	del, err := env.srv.DeleteWorkflow(ctx, &pb.DeleteWorkflowRequest{Id: wf.ID.String()})
	require.NoError(t, err)
	assert.Equal(t, "workflow deleted successfully", del.Message)
	assert.Equal(t, []uuid.UUID{wf.ID}, env.workflows.deleted)

	_, err = env.srv.DeleteWorkflow(ctx, &pb.DeleteWorkflowRequest{Id: wf.ID.String()})
	assert.Equal(t, codes.NotFound, rcCode(t, err))
	_, err = env.srv.DeleteWorkflow(ctx, &pb.DeleteWorkflowRequest{Id: "bad"})
	assert.Equal(t, codes.InvalidArgument, rcCode(t, err))
}

// ---------------------------------------------------------------------------
// Executions
// ---------------------------------------------------------------------------

func TestRCExecutions_ListGetCancelRetry(t *testing.T) {
	ctx := context.Background()
	env := rcNewEnv()
	wf := rcWorkflowModel("wf", "active")
	env.workflows.put(wf)

	started := time.Unix(1700000000, 0)
	stored := &storagemodels.ExecutionModel{
		ID: uuid.New(), WorkflowID: &wf.ID, WorkflowSource: "stored", Status: "completed", StartedAt: &started,
		NodeExecutions: []*storagemodels.NodeExecutionModel{
			{ID: uuid.New(), NodeID: &wf.Nodes[0].ID, Status: "completed", StartedAt: &started},
		},
	}
	inline := &storagemodels.ExecutionModel{ID: uuid.New(), WorkflowSource: "inline", Status: "running", StartedAt: &started}
	env.execs.put(stored)
	env.execs.put(inline)

	list, err := env.srv.ListExecutions(ctx, &pb.ListExecutionsRequest{})
	require.NoError(t, err)
	assert.Len(t, list.Executions, 2)
	assert.Equal(t, int32(2), list.Total)
	assert.Equal(t, int32(50), list.Limit)
	assert.Equal(t, "all", env.execs.listedBy)

	_, err = env.srv.ListExecutions(ctx, &pb.ListExecutionsRequest{WorkflowId: wf.ID.String(), Limit: 3})
	require.NoError(t, err)
	assert.Equal(t, "workflow", env.execs.listedBy)
	_, err = env.srv.ListExecutions(ctx, &pb.ListExecutionsRequest{Status: "running"})
	require.NoError(t, err)
	assert.Equal(t, "status", env.execs.listedBy)
	_, err = env.srv.ListExecutions(ctx, &pb.ListExecutionsRequest{WorkflowId: "bad"})
	assert.Equal(t, codes.InvalidArgument, rcCode(t, err))

	got, err := env.srv.GetExecution(ctx, &pb.GetExecutionRequest{Id: stored.ID.String()})
	require.NoError(t, err)
	assert.Equal(t, "completed", got.Execution.Status)
	require.Len(t, got.Execution.NodeExecutions, 1)
	assert.Equal(t, "a", got.Execution.NodeExecutions[0].NodeId, "node UUIDs are mapped to logical IDs")

	got, err = env.srv.GetExecution(ctx, &pb.GetExecutionRequest{Id: inline.ID.String()})
	require.NoError(t, err)
	assert.Equal(t, "inline", got.Execution.WorkflowSource)

	_, err = env.srv.GetExecution(ctx, &pb.GetExecutionRequest{Id: "bad"})
	assert.Equal(t, codes.InvalidArgument, rcCode(t, err))
	_, err = env.srv.GetExecution(ctx, &pb.GetExecutionRequest{Id: uuid.NewString()})
	assert.Equal(t, codes.NotFound, rcCode(t, err))

	_, err = env.srv.CancelExecution(ctx, &pb.CancelExecutionRequest{Id: stored.ID.String()})
	assert.Equal(t, codes.FailedPrecondition, rcCode(t, err), "a completed execution is not cancellable")
	_, err = env.srv.CancelExecution(ctx, &pb.CancelExecutionRequest{Id: "bad"})
	assert.Equal(t, codes.InvalidArgument, rcCode(t, err))
	_, err = env.srv.RetryExecution(ctx, &pb.RetryExecutionRequest{Id: stored.ID.String()})
	assert.Equal(t, codes.FailedPrecondition, rcCode(t, err), "a completed execution is not retryable")
	_, err = env.srv.RetryExecution(ctx, &pb.RetryExecutionRequest{Id: "bad"})
	assert.Equal(t, codes.InvalidArgument, rcCode(t, err))

	env.execs.setErr("List", errRCBoom)
	_, err = env.srv.ListExecutions(ctx, &pb.ListExecutionsRequest{})
	assert.Equal(t, codes.Internal, rcCode(t, err))
}

func TestRCExecutions_Start(t *testing.T) {
	ctx := context.Background()
	env := rcNewEnv()
	env.ops.ExecutionMgr = engine.NewExecutionManager(env.executors, env.workflows, env.execs, nil, nil, nil)

	_, err := env.srv.StartExecution(ctx, &pb.StartExecutionRequest{
		WorkflowId: uuid.NewString(),
		Webhooks:   []*pb.WebhookSubscription{{Url: ""}},
	})
	assert.Equal(t, codes.InvalidArgument, rcCode(t, err), "webhook URL is validated")

	_, err = env.srv.StartExecution(ctx, &pb.StartExecutionRequest{
		WorkflowId: uuid.NewString(),
		Webhooks:   []*pb.WebhookSubscription{{Url: "https://hooks.example/x", Events: []string{"no.such.event"}}},
	})
	assert.Equal(t, codes.InvalidArgument, rcCode(t, err), "webhook events are validated")

	_, err = env.srv.StartExecution(ctx, &pb.StartExecutionRequest{
		WorkflowId: "not-a-uuid",
		Input:      rcStruct(t, map[string]any{"a": 1}),
		Webhooks:   []*pb.WebhookSubscription{{Url: "https://hooks.example/x", Events: []string{"execution.completed"}, Headers: map[string]string{"A": "b"}, NodeIds: []string{"a"}}},
	})
	assert.Equal(t, codes.Internal, rcCode(t, err))

	_, err = env.srv.StartExecution(ctx, &pb.StartExecutionRequest{WorkflowId: uuid.NewString()})
	assert.Equal(t, codes.NotFound, rcCode(t, err), "unknown workflow maps to NotFound")
}

func TestRCExecutions_RunEphemeral(t *testing.T) {
	ctx := context.Background()
	env := rcNewEnv()
	env.ops.ExecutionMgr = engine.NewExecutionManager(env.executors, env.workflows, env.execs, nil, nil, nil)

	_, err := env.srv.RunEphemeralExecution(ctx, &pb.RunEphemeralExecutionRequest{})
	assert.Equal(t, codes.InvalidArgument, rcCode(t, err), "workflow is required")

	wf := &pb.Workflow{
		Name: "inline",
		Nodes: []*pb.Node{
			{Id: "n1", Name: "N1", Type: rcNodeType, Config: rcStruct(t, map[string]any{"value": "hello"})},
		},
	}
	resp, err := env.srv.RunEphemeralExecution(ctx, &pb.RunEphemeralExecutionRequest{
		Workflow: wf,
		Mode:     "sync",
		Input:    rcStruct(t, map[string]any{"in": 1}),
	})
	require.NoError(t, err)
	assert.Equal(t, string(models.ExecutionStatusCompleted), resp.Execution.Status)
	assert.Equal(t, "inline", resp.Execution.WorkflowSource)
	assert.Equal(t, "hello", resp.Execution.Output.AsMap()["echo"])

	_, err = env.srv.RunEphemeralExecution(ctx, &pb.RunEphemeralExecutionRequest{Workflow: wf, Mode: "bogus"})
	assert.Equal(t, codes.InvalidArgument, rcCode(t, err))

	_, err = env.srv.RunEphemeralExecution(ctx, &pb.RunEphemeralExecutionRequest{
		Workflow: wf, Mode: "sync",
		Webhooks: []*pb.WebhookSubscription{{Url: "", Events: []string{"execution.completed"}, Headers: map[string]string{"a": "b"}, NodeIds: []string{"n1"}}},
	})
	assert.Equal(t, codes.InvalidArgument, rcCode(t, err))
}

// ---------------------------------------------------------------------------
// Triggers
// ---------------------------------------------------------------------------

func TestRCTriggers(t *testing.T) {
	ctx := context.Background()
	env := rcNewEnv()
	wf := rcWorkflowModel("wf", "active")
	env.workflows.put(wf)

	created, err := env.srv.CreateTrigger(ctx, &pb.CreateTriggerRequest{
		WorkflowId: wf.ID.String(), Name: "every-minute", Description: "d", Type: "cron",
		Config: rcStruct(t, map[string]any{"schedule": "* * * * *"}), Enabled: true,
	})
	require.NoError(t, err)
	assert.Equal(t, "every-minute", created.Trigger.Name)
	assert.Equal(t, "cron", created.Trigger.Type)
	assert.True(t, created.Trigger.Enabled)
	triggerID := created.Trigger.Id

	_, err = env.srv.CreateTrigger(ctx, &pb.CreateTriggerRequest{WorkflowId: wf.ID.String(), Name: "x", Type: "cron", Config: rcStruct(t, map[string]any{"schedule": "not cron"})})
	assert.Equal(t, codes.InvalidArgument, rcCode(t, err))
	_, err = env.srv.CreateTrigger(ctx, &pb.CreateTriggerRequest{WorkflowId: uuid.NewString(), Name: "x", Type: "manual"})
	assert.Equal(t, codes.NotFound, rcCode(t, err))

	// List variants
	list, err := env.srv.ListTriggers(ctx, &pb.ListTriggersRequest{})
	require.NoError(t, err)
	assert.Len(t, list.Triggers, 1)
	assert.Equal(t, int32(1), list.Total)
	assert.Equal(t, int32(50), list.Limit)
	assert.Equal(t, "all", env.triggers.listedBy)
	_, err = env.srv.ListTriggers(ctx, &pb.ListTriggersRequest{WorkflowId: wf.ID.String()})
	require.NoError(t, err)
	assert.Equal(t, "workflow", env.triggers.listedBy)
	_, err = env.srv.ListTriggers(ctx, &pb.ListTriggersRequest{Type: "cron", Limit: 2})
	require.NoError(t, err)
	assert.Equal(t, "type", env.triggers.listedBy)
	_, err = env.srv.ListTriggers(ctx, &pb.ListTriggersRequest{WorkflowId: "bad"})
	assert.Equal(t, codes.InvalidArgument, rcCode(t, err))

	// Get
	got, err := env.srv.GetTrigger(ctx, &pb.GetTriggerRequest{Id: triggerID})
	require.NoError(t, err)
	assert.Equal(t, wf.ID.String(), got.Trigger.WorkflowId)
	_, err = env.srv.GetTrigger(ctx, &pb.GetTriggerRequest{Id: uuid.NewString()})
	assert.Equal(t, codes.NotFound, rcCode(t, err))
	_, err = env.srv.GetTrigger(ctx, &pb.GetTriggerRequest{Id: "bad"})
	assert.Equal(t, codes.InvalidArgument, rcCode(t, err))

	// Update
	disabled := false
	upd, err := env.srv.UpdateTrigger(ctx, &pb.UpdateTriggerRequest{Id: triggerID, Name: "renamed", Type: "manual", Enabled: &disabled})
	require.NoError(t, err)
	assert.Equal(t, "manual", upd.Trigger.Type)
	assert.False(t, upd.Trigger.Enabled)
	assert.Equal(t, "renamed", upd.Trigger.Name)
	_, err = env.srv.UpdateTrigger(ctx, &pb.UpdateTriggerRequest{Id: triggerID, Type: "bogus"})
	assert.Equal(t, codes.InvalidArgument, rcCode(t, err))
	_, err = env.srv.UpdateTrigger(ctx, &pb.UpdateTriggerRequest{Id: uuid.NewString()})
	assert.Equal(t, codes.NotFound, rcCode(t, err))
	_, err = env.srv.UpdateTrigger(ctx, &pb.UpdateTriggerRequest{Id: "bad"})
	assert.Equal(t, codes.InvalidArgument, rcCode(t, err))

	// Delete
	del, err := env.srv.DeleteTrigger(ctx, &pb.DeleteTriggerRequest{Id: triggerID})
	require.NoError(t, err)
	assert.Equal(t, "trigger deleted successfully", del.Message)
	_, err = env.srv.DeleteTrigger(ctx, &pb.DeleteTriggerRequest{Id: triggerID})
	assert.Equal(t, codes.NotFound, rcCode(t, err))
	_, err = env.srv.DeleteTrigger(ctx, &pb.DeleteTriggerRequest{Id: "bad"})
	assert.Equal(t, codes.InvalidArgument, rcCode(t, err))

	env.triggers.setErr("List", errRCBoom)
	_, err = env.srv.ListTriggers(ctx, &pb.ListTriggersRequest{})
	assert.Equal(t, codes.Internal, rcCode(t, err))
}

// ---------------------------------------------------------------------------
// Credentials
// ---------------------------------------------------------------------------

func TestRCCredentials(t *testing.T) {
	env := rcNewEnv()
	user := uuid.NewString()
	ctx := ContextWithUserID(context.Background(), user)

	_, err := env.srv.CreateCredential(context.Background(), &pb.CreateCredentialRequest{Name: "x", CredentialType: "api_key"})
	assert.Equal(t, codes.Unauthenticated, rcCode(t, err))

	created, err := env.srv.CreateCredential(ctx, &pb.CreateCredentialRequest{
		Name: "openai", Description: "key", CredentialType: "api_key", Provider: "openai",
		Data: map[string]string{"api_key": "sk-secret"},
	})
	require.NoError(t, err)
	assert.Equal(t, "openai", created.Credential.Name)
	assert.Equal(t, []string{"api_key"}, created.Credential.Fields)
	id := created.Credential.Id

	stored := env.creds.creds[id]
	assert.NotEqual(t, "sk-secret", stored.EncryptedData["api_key"], "data is encrypted at rest")

	_, err = env.srv.CreateCredential(ctx, &pb.CreateCredentialRequest{Name: "x", CredentialType: "nope"})
	assert.Equal(t, codes.InvalidArgument, rcCode(t, err))

	list, err := env.srv.ListCredentials(ctx, &pb.ListCredentialsRequest{UserId: user})
	require.NoError(t, err)
	assert.Len(t, list.Credentials, 1)
	assert.Equal(t, "owner", env.creds.listed)
	list, err = env.srv.ListCredentials(ctx, &pb.ListCredentialsRequest{UserId: user, Provider: "other"})
	require.NoError(t, err)
	assert.Empty(t, list.Credentials)
	assert.Equal(t, "provider", env.creds.listed)

	got, err := env.srv.GetCredential(ctx, &pb.GetCredentialRequest{Id: id})
	require.NoError(t, err)
	assert.Equal(t, "openai", got.Credential.Provider)
	_, err = env.srv.GetCredential(ctx, &pb.GetCredentialRequest{Id: "missing"})
	assert.Equal(t, codes.NotFound, rcCode(t, err))

	upd, err := env.srv.UpdateCredential(ctx, &pb.UpdateCredentialRequest{Id: id, Name: "renamed", Description: "new"})
	require.NoError(t, err)
	assert.Equal(t, "renamed", upd.Credential.Name)
	assert.Equal(t, "new", upd.Credential.Description)
	_, err = env.srv.UpdateCredential(ctx, &pb.UpdateCredentialRequest{Id: "missing"})
	assert.Equal(t, codes.NotFound, rcCode(t, err))

	del, err := env.srv.DeleteCredential(ctx, &pb.DeleteCredentialRequest{Id: id})
	require.NoError(t, err)
	assert.Equal(t, "credential deleted successfully", del.Message)
	assert.Equal(t, []string{id}, env.creds.deleted)
	assert.Equal(t, []uuid.UUID{uuid.MustParse(id)}, env.workflows.unassigned, "credential is detached from workflows first")
	_, err = env.srv.DeleteCredential(ctx, &pb.DeleteCredentialRequest{Id: id})
	assert.Equal(t, codes.NotFound, rcCode(t, err))

	env.creds.setErr("List", errRCBoom)
	_, err = env.srv.ListCredentials(ctx, &pb.ListCredentialsRequest{UserId: user})
	assert.Equal(t, codes.Internal, rcCode(t, err))
}

// ---------------------------------------------------------------------------
// Audit log
// ---------------------------------------------------------------------------

func TestRCListAuditLog(t *testing.T) {
	ctx := context.Background()
	env := rcNewEnv()
	resID := "wf-1"
	env.audit.logs = []*models.ServiceAuditLog{{
		ID: uuid.NewString(), SystemKeyID: "k", ServiceName: "billing", Action: "workflow.create",
		ResourceType: "workflow", ResourceID: &resID, RequestMethod: "POST", RequestPath: "/x",
		ResponseStatus: 201, CreatedAt: time.Unix(1700000000, 0),
	}}

	imp := uuid.New()
	from := time.Unix(1690000000, 0).UTC()
	to := time.Unix(1710000000, 0).UTC()
	resp, err := env.srv.ListAuditLog(ctx, &pb.ListAuditLogRequest{
		ServiceName: "billing", Action: "workflow.create", ResourceType: "workflow",
		ImpersonatedUserId: imp.String(),
		DateFrom:           timestamppb.New(from), DateTo: timestamppb.New(to),
		Offset: 3,
	})
	require.NoError(t, err)
	require.Len(t, resp.AuditLogs, 1)
	assert.Equal(t, "billing", resp.AuditLogs[0].ServiceName)
	assert.Equal(t, "wf-1", resp.AuditLogs[0].ResourceId)
	assert.Equal(t, int64(1), resp.Total)
	assert.Equal(t, int32(50), resp.Limit)
	assert.Equal(t, int32(3), resp.Offset)

	f := env.audit.lastFilter
	assert.Equal(t, "billing", *f.ServiceName)
	assert.Equal(t, "workflow.create", *f.Action)
	assert.Equal(t, "workflow", *f.ResourceType)
	assert.Equal(t, imp, *f.ImpersonatedUserID)
	assert.True(t, from.Equal(*f.DateFrom))
	assert.True(t, to.Equal(*f.DateTo))

	// Invalid impersonated user IDs are ignored rather than rejected.
	_, err = env.srv.ListAuditLog(ctx, &pb.ListAuditLogRequest{ImpersonatedUserId: "garbage", Limit: 500})
	require.NoError(t, err)
	assert.Nil(t, env.audit.lastFilter.ImpersonatedUserID)
	assert.Equal(t, 100, env.audit.lastFilter.Limit, "limit is capped")

	env.audit.err = errRCBoom
	_, err = env.srv.ListAuditLog(ctx, &pb.ListAuditLogRequest{})
	assert.Equal(t, codes.Internal, rcCode(t, err))
}
