package mbflow

import (
	"context"
	"fmt"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/smilemakc/mbflow/sdk/go/internal/grpcclient"
	pb "github.com/smilemakc/mbflow/sdk/go/internal/pb"
	"github.com/smilemakc/mbflow/sdk/go/models"
)

func resolveOnBehalfOf(opts []RequestOption) string {
	ro := applyRequestOptions(opts)
	return ro.onBehalfOf
}

func convertGRPCError(err error) error {
	st, ok := status.FromError(err)
	if !ok {
		return err
	}
	return &APIError{
		StatusCode: grpcToHTTPStatus(st.Code()),
		Code:       st.Code().String(),
		Message:    st.Message(),
	}
}

func grpcToHTTPStatus(code codes.Code) int {
	switch code {
	case codes.NotFound:
		return 404
	case codes.AlreadyExists:
		return 409
	case codes.InvalidArgument:
		return 422
	case codes.Unauthenticated:
		return 401
	case codes.PermissionDenied:
		return 403
	case codes.ResourceExhausted:
		return 429
	case codes.DeadlineExceeded:
		return 408
	default:
		return 500
	}
}

// --- WorkflowService ---

type grpcWorkflowService struct{ tr *grpcclient.Transport }

func newGRPCWorkflowClient(tr *grpcclient.Transport) WorkflowService {
	return &grpcWorkflowService{tr: tr}
}

func (w *grpcWorkflowService) Create(ctx context.Context, wf *models.Workflow, opts ...RequestOption) (*models.Workflow, error) {
	ctx = w.tr.AuthContext(ctx, resolveOnBehalfOf(opts))
	resp, err := w.tr.Client().CreateWorkflow(ctx, &pb.CreateWorkflowRequest{
		Name:        wf.Name,
		Description: wf.Description,
		Variables:   grpcclient.MapToStruct(wf.Variables),
		Metadata:    grpcclient.MapToStruct(wf.Metadata),
	})
	if err != nil {
		return nil, convertGRPCError(err)
	}
	return grpcclient.WorkflowFromProto(resp.Workflow), nil
}

func (w *grpcWorkflowService) Get(ctx context.Context, id string, opts ...RequestOption) (*models.Workflow, error) {
	ctx = w.tr.AuthContext(ctx, resolveOnBehalfOf(opts))
	resp, err := w.tr.Client().GetWorkflow(ctx, &pb.GetWorkflowRequest{Id: id})
	if err != nil {
		return nil, convertGRPCError(err)
	}
	return grpcclient.WorkflowFromProto(resp.Workflow), nil
}

func (w *grpcWorkflowService) Update(ctx context.Context, id string, wf *models.Workflow, opts ...RequestOption) (*models.Workflow, error) {
	ctx = w.tr.AuthContext(ctx, resolveOnBehalfOf(opts))
	req := &pb.UpdateWorkflowRequest{
		Id:          id,
		Name:        wf.Name,
		Description: wf.Description,
		Variables:   grpcclient.MapToStruct(wf.Variables),
		Metadata:    grpcclient.MapToStruct(wf.Metadata),
	}
	for _, n := range wf.Nodes {
		req.Nodes = append(req.Nodes, grpcclient.NodeToProto(n))
	}
	for _, e := range wf.Edges {
		req.Edges = append(req.Edges, grpcclient.EdgeToProto(e))
	}
	resp, err := w.tr.Client().UpdateWorkflow(ctx, req)
	if err != nil {
		return nil, convertGRPCError(err)
	}
	return grpcclient.WorkflowFromProto(resp.Workflow), nil
}

func (w *grpcWorkflowService) Delete(ctx context.Context, id string, opts ...RequestOption) error {
	ctx = w.tr.AuthContext(ctx, resolveOnBehalfOf(opts))
	_, err := w.tr.Client().DeleteWorkflow(ctx, &pb.DeleteWorkflowRequest{Id: id})
	if err != nil {
		return convertGRPCError(err)
	}
	return nil
}

func (w *grpcWorkflowService) List(ctx context.Context, lo *models.ListOptions, opts ...RequestOption) (*models.Page[models.Workflow], error) {
	ctx = w.tr.AuthContext(ctx, resolveOnBehalfOf(opts))
	req := &pb.ListWorkflowsRequest{}
	if lo != nil {
		req.Limit = int32(lo.Limit)
		req.Offset = int32(lo.Offset)
	}
	resp, err := w.tr.Client().ListWorkflows(ctx, req)
	if err != nil {
		return nil, convertGRPCError(err)
	}
	items := make([]*models.Workflow, 0, len(resp.Workflows))
	for _, pw := range resp.Workflows {
		items = append(items, grpcclient.WorkflowFromProto(pw))
	}
	return &models.Page[models.Workflow]{Items: items, Total: int(resp.Total)}, nil
}

// --- ExecutionService ---

type grpcExecutionService struct{ tr *grpcclient.Transport }

func newGRPCExecutionClient(tr *grpcclient.Transport) ExecutionService {
	return &grpcExecutionService{tr: tr}
}

func (e *grpcExecutionService) Run(ctx context.Context, wfID string, input map[string]any, opts ...RequestOption) (*models.Execution, error) {
	ctx = e.tr.AuthContext(ctx, resolveOnBehalfOf(opts))
	resp, err := e.tr.Client().StartExecution(ctx, &pb.StartExecutionRequest{
		WorkflowId: wfID,
		Input:      grpcclient.MapToStruct(input),
	})
	if err != nil {
		return nil, convertGRPCError(err)
	}
	return grpcclient.ExecutionFromProto(resp.Execution), nil
}

func (e *grpcExecutionService) Get(ctx context.Context, id string, opts ...RequestOption) (*models.Execution, error) {
	ctx = e.tr.AuthContext(ctx, resolveOnBehalfOf(opts))
	resp, err := e.tr.Client().GetExecution(ctx, &pb.GetExecutionRequest{Id: id})
	if err != nil {
		return nil, convertGRPCError(err)
	}
	return grpcclient.ExecutionFromProto(resp.Execution), nil
}

func (e *grpcExecutionService) List(ctx context.Context, lo *models.ListOptions, opts ...RequestOption) (*models.Page[models.Execution], error) {
	ctx = e.tr.AuthContext(ctx, resolveOnBehalfOf(opts))
	req := &pb.ListExecutionsRequest{}
	if lo != nil {
		req.Limit = int32(lo.Limit)
		req.Offset = int32(lo.Offset)
		req.WorkflowId = lo.WorkflowID
	}
	resp, err := e.tr.Client().ListExecutions(ctx, req)
	if err != nil {
		return nil, convertGRPCError(err)
	}
	items := make([]*models.Execution, 0, len(resp.Executions))
	for _, pe := range resp.Executions {
		items = append(items, grpcclient.ExecutionFromProto(pe))
	}
	return &models.Page[models.Execution]{Items: items, Total: int(resp.Total)}, nil
}

func (e *grpcExecutionService) Cancel(ctx context.Context, id string, opts ...RequestOption) (*models.Execution, error) {
	ctx = e.tr.AuthContext(ctx, resolveOnBehalfOf(opts))
	resp, err := e.tr.Client().CancelExecution(ctx, &pb.CancelExecutionRequest{Id: id})
	if err != nil {
		return nil, convertGRPCError(err)
	}
	return grpcclient.ExecutionFromProto(resp.Execution), nil
}

func (e *grpcExecutionService) Retry(ctx context.Context, id string, opts ...RequestOption) (*models.Execution, error) {
	ctx = e.tr.AuthContext(ctx, resolveOnBehalfOf(opts))
	resp, err := e.tr.Client().RetryExecution(ctx, &pb.RetryExecutionRequest{Id: id})
	if err != nil {
		return nil, convertGRPCError(err)
	}
	return grpcclient.ExecutionFromProto(resp.Execution), nil
}

// --- TriggerService ---

type grpcTriggerService struct{ tr *grpcclient.Transport }

func newGRPCTriggerClient(tr *grpcclient.Transport) TriggerService {
	return &grpcTriggerService{tr: tr}
}

func (t *grpcTriggerService) Create(ctx context.Context, trigger *models.Trigger, opts ...RequestOption) (*models.Trigger, error) {
	ctx = t.tr.AuthContext(ctx, resolveOnBehalfOf(opts))
	resp, err := t.tr.Client().CreateTrigger(ctx, &pb.CreateTriggerRequest{
		WorkflowId:  trigger.WorkflowID,
		Name:        trigger.Name,
		Description: trigger.Description,
		Type:        string(trigger.Type),
		Config:      grpcclient.MapToStruct(trigger.Config),
		Enabled:     trigger.Enabled,
	})
	if err != nil {
		return nil, convertGRPCError(err)
	}
	return grpcclient.TriggerFromProto(resp.Trigger), nil
}

func (t *grpcTriggerService) Get(ctx context.Context, id string, opts ...RequestOption) (*models.Trigger, error) {
	return nil, fmt.Errorf("mbflow: GetTrigger is not supported by the gRPC API; use List with a workflow ID filter")
}

func (t *grpcTriggerService) Update(ctx context.Context, id string, trigger *models.Trigger, opts ...RequestOption) (*models.Trigger, error) {
	ctx = t.tr.AuthContext(ctx, resolveOnBehalfOf(opts))
	enabled := trigger.Enabled
	resp, err := t.tr.Client().UpdateTrigger(ctx, &pb.UpdateTriggerRequest{
		Id:          id,
		Name:        trigger.Name,
		Description: trigger.Description,
		Type:        string(trigger.Type),
		Config:      grpcclient.MapToStruct(trigger.Config),
		Enabled:     &enabled,
	})
	if err != nil {
		return nil, convertGRPCError(err)
	}
	return grpcclient.TriggerFromProto(resp.Trigger), nil
}

func (t *grpcTriggerService) Delete(ctx context.Context, id string, opts ...RequestOption) error {
	ctx = t.tr.AuthContext(ctx, resolveOnBehalfOf(opts))
	_, err := t.tr.Client().DeleteTrigger(ctx, &pb.DeleteTriggerRequest{Id: id})
	if err != nil {
		return convertGRPCError(err)
	}
	return nil
}

func (t *grpcTriggerService) List(ctx context.Context, lo *models.ListOptions, opts ...RequestOption) (*models.Page[models.Trigger], error) {
	ctx = t.tr.AuthContext(ctx, resolveOnBehalfOf(opts))
	req := &pb.ListTriggersRequest{}
	if lo != nil {
		req.Limit = int32(lo.Limit)
		req.Offset = int32(lo.Offset)
		req.WorkflowId = lo.WorkflowID
	}
	resp, err := t.tr.Client().ListTriggers(ctx, req)
	if err != nil {
		return nil, convertGRPCError(err)
	}
	items := make([]*models.Trigger, 0, len(resp.Triggers))
	for _, pt := range resp.Triggers {
		items = append(items, grpcclient.TriggerFromProto(pt))
	}
	return &models.Page[models.Trigger]{Items: items, Total: int(resp.Total)}, nil
}

// --- CredentialService ---

type grpcCredentialService struct{ tr *grpcclient.Transport }

func newGRPCCredentialClient(tr *grpcclient.Transport) CredentialService {
	return &grpcCredentialService{tr: tr}
}

func (c *grpcCredentialService) Create(ctx context.Context, cred *models.Credential, opts ...RequestOption) (*models.Credential, error) {
	ctx = c.tr.AuthContext(ctx, resolveOnBehalfOf(opts))
	resp, err := c.tr.Client().CreateCredential(ctx, &pb.CreateCredentialRequest{
		Name:           cred.Name,
		Description:    cred.Description,
		CredentialType: cred.CredentialType,
		Data:           cred.Data,
	})
	if err != nil {
		return nil, convertGRPCError(err)
	}
	return grpcclient.CredentialFromProto(resp.Credential), nil
}

func (c *grpcCredentialService) Get(ctx context.Context, id string, opts ...RequestOption) (*models.Credential, error) {
	return nil, fmt.Errorf("mbflow: GetCredential is not supported by the gRPC API; use List to retrieve credentials")
}

func (c *grpcCredentialService) Update(ctx context.Context, id string, cred *models.Credential, opts ...RequestOption) (*models.Credential, error) {
	ctx = c.tr.AuthContext(ctx, resolveOnBehalfOf(opts))
	resp, err := c.tr.Client().UpdateCredential(ctx, &pb.UpdateCredentialRequest{
		Id:          id,
		Name:        cred.Name,
		Description: cred.Description,
	})
	if err != nil {
		return nil, convertGRPCError(err)
	}
	return grpcclient.CredentialFromProto(resp.Credential), nil
}

func (c *grpcCredentialService) Delete(ctx context.Context, id string, opts ...RequestOption) error {
	ctx = c.tr.AuthContext(ctx, resolveOnBehalfOf(opts))
	_, err := c.tr.Client().DeleteCredential(ctx, &pb.DeleteCredentialRequest{Id: id})
	if err != nil {
		return convertGRPCError(err)
	}
	return nil
}

func (c *grpcCredentialService) List(ctx context.Context, lo *models.ListOptions, opts ...RequestOption) (*models.Page[models.Credential], error) {
	ctx = c.tr.AuthContext(ctx, resolveOnBehalfOf(opts))
	resp, err := c.tr.Client().ListCredentials(ctx, &pb.ListCredentialsRequest{})
	if err != nil {
		return nil, convertGRPCError(err)
	}
	items := make([]*models.Credential, 0, len(resp.Credentials))
	for _, pc := range resp.Credentials {
		items = append(items, grpcclient.CredentialFromProto(pc))
	}
	return &models.Page[models.Credential]{Items: items, Total: len(items)}, nil
}
