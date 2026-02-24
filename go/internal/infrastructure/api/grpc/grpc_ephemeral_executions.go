package grpc

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	pb "github.com/smilemakc/mbflow/go/api/proto/serviceapipb"
	"github.com/smilemakc/mbflow/go/internal/application/serviceapi"
	"github.com/smilemakc/mbflow/go/pkg/models"
)

func (s *ServiceAPIServer) RunEphemeralExecution(ctx context.Context, req *pb.RunEphemeralExecutionRequest) (*pb.ExecutionResponse, error) {
	if req.Workflow == nil {
		return nil, status.Errorf(codes.InvalidArgument, "workflow is required")
	}

	workflow := workflowFromProto(req.Workflow)

	params := serviceapi.EphemeralExecutionParams{
		Workflow:         workflow,
		Input:            structToMap(req.Input),
		Mode:             req.Mode,
		CredentialIDs:    req.CredentialIds,
		Variables:        structToMap(req.Variables),
		PersistExecution: req.PersistExecution,
	}

	if len(req.Webhooks) > 0 {
		params.Webhooks = make([]serviceapi.WebhookSubscription, len(req.Webhooks))
		for i, wh := range req.Webhooks {
			params.Webhooks[i] = serviceapi.WebhookSubscription{
				URL:     wh.Url,
				Events:  wh.Events,
				Headers: wh.Headers,
				NodeIDs: wh.NodeIds,
			}
		}
	}

	result, err := s.ops.StartEphemeralExecution(ctx, params)
	if err != nil {
		return nil, mapError(err)
	}

	return &pb.ExecutionResponse{
		Execution: toProtoExecution(result),
	}, nil
}

func (s *ServiceAPIServer) StreamExecutionEvents(_ *pb.StreamExecutionEventsRequest, _ pb.MBFlowServiceAPI_StreamExecutionEventsServer) error {
	return status.Error(codes.Unimplemented, "StreamExecutionEvents not yet implemented")
}

func workflowFromProto(pw *pb.Workflow) *models.Workflow {
	if pw == nil {
		return nil
	}

	wf := &models.Workflow{
		ID:          pw.Id,
		Name:        pw.Name,
		Description: pw.Description,
		Version:     int(pw.Version),
		Status:      models.WorkflowStatus(pw.Status),
		CreatedBy:   pw.CreatedBy,
		Variables:   structToMap(pw.Variables),
		Metadata:    structToMap(pw.Metadata),
	}

	if pw.CreatedAt != nil {
		wf.CreatedAt = pw.CreatedAt.AsTime()
	}
	if pw.UpdatedAt != nil {
		wf.UpdatedAt = pw.UpdatedAt.AsTime()
	}

	for _, pn := range pw.Nodes {
		wf.Nodes = append(wf.Nodes, nodeFromProto(pn))
	}
	for _, pe := range pw.Edges {
		wf.Edges = append(wf.Edges, edgeFromProto(pe))
	}

	return wf
}

func nodeFromProto(pn *pb.Node) *models.Node {
	if pn == nil {
		return nil
	}

	n := &models.Node{
		ID:     pn.Id,
		Name:   pn.Name,
		Type:   pn.Type,
		Config: structToMap(pn.Config),
	}

	if pn.Position != nil {
		pos := structToMap(pn.Position)
		if x, ok := pos["x"].(float64); ok {
			if y, ok := pos["y"].(float64); ok {
				n.Position = &models.Position{X: x, Y: y}
			}
		}
	}

	return n
}

func edgeFromProto(pe *pb.Edge) *models.Edge {
	if pe == nil {
		return nil
	}

	e := &models.Edge{
		ID:           pe.Id,
		From:         pe.From,
		To:           pe.To,
		SourceHandle: pe.SourceHandle,
	}

	if pe.Condition != nil {
		condMap := structToMap(pe.Condition)
		if expr, ok := condMap["expression"].(string); ok {
			e.Condition = expr
		}
	}

	if pe.Loop != nil {
		e.Loop = &models.LoopConfig{MaxIterations: int(pe.Loop.MaxIterations)}
	}

	return e
}
