package grpc

import (
	"context"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	pb "github.com/smilemakc/mbflow/go/api/proto/serviceapipb"
	"github.com/smilemakc/mbflow/go/internal/application/serviceapi"
)

func (s *ServiceAPIServer) ListWorkflows(ctx context.Context, req *pb.ListWorkflowsRequest) (*pb.ListWorkflowsResponse, error) {
	params := serviceapi.ListWorkflowsParams{
		Limit:  int(req.Limit),
		Offset: int(req.Offset),
	}
	if req.Status != "" {
		params.Status = &req.Status
	}
	if req.UserId != "" {
		uid, err := uuid.Parse(req.UserId)
		if err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "invalid user_id")
		}
		params.UserID = &uid
	}

	if params.Limit == 0 {
		params.Limit = 50
	}

	result, err := s.ops.ListWorkflows(ctx, params)
	if err != nil {
		return nil, mapError(err)
	}

	return &pb.ListWorkflowsResponse{
		Workflows: toProtoWorkflows(result.Workflows),
		Total:     int32(result.Total),
		Limit:     int32(params.Limit),
		Offset:    int32(params.Offset),
	}, nil
}

func (s *ServiceAPIServer) GetWorkflow(ctx context.Context, req *pb.GetWorkflowRequest) (*pb.WorkflowResponse, error) {
	wfUUID, err := uuid.Parse(req.Id)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid workflow ID")
	}

	workflow, err := s.ops.GetWorkflow(ctx, serviceapi.GetWorkflowParams{
		WorkflowID: wfUUID,
	})
	if err != nil {
		return nil, mapError(err)
	}

	return &pb.WorkflowResponse{
		Workflow: toProtoWorkflow(workflow),
	}, nil
}

func (s *ServiceAPIServer) CreateWorkflow(ctx context.Context, req *pb.CreateWorkflowRequest) (*pb.WorkflowResponse, error) {
	var createdBy *uuid.UUID
	if uid, ok := UserIDFromContext(ctx); ok {
		if parsed, err := uuid.Parse(uid); err == nil {
			createdBy = &parsed
		}
	}

	var nodes []serviceapi.NodeInput
	if req.Nodes != nil {
		nodes = make([]serviceapi.NodeInput, len(req.Nodes))
		for i, n := range req.Nodes {
			nodes[i] = serviceapi.NodeInput{
				ID:       n.Id,
				Name:     n.Name,
				Type:     n.Type,
				Config:   structToMap(n.Config),
				Position: structToMap(n.Position),
			}
		}
	}

	var edges []serviceapi.EdgeInput
	if req.Edges != nil {
		edges = make([]serviceapi.EdgeInput, len(req.Edges))
		for i, e := range req.Edges {
			ei := serviceapi.EdgeInput{
				ID:           e.Id,
				From:         e.From,
				To:           e.To,
				SourceHandle: e.SourceHandle,
				Condition:    structToMap(e.Condition),
			}
			if e.Loop != nil {
				ei.Loop = &serviceapi.LoopInput{MaxIterations: int(e.Loop.MaxIterations)}
			}
			edges[i] = ei
		}
	}

	var resources []serviceapi.ResourceInput
	if req.Resources != nil {
		resources = make([]serviceapi.ResourceInput, len(req.Resources))
		for i, r := range req.Resources {
			resources[i] = serviceapi.ResourceInput{
				ResourceID: r.ResourceId,
				Alias:      r.Alias,
				AccessType: r.AccessType,
			}
		}
	}

	workflow, err := s.ops.CreateWorkflow(ctx, serviceapi.CreateWorkflowParams{
		Name:        req.Name,
		Description: req.Description,
		Variables:   structToMap(req.Variables),
		Metadata:    structToMap(req.Metadata),
		CreatedBy:   createdBy,
		Nodes:       nodes,
		Edges:       edges,
		Resources:   resources,
	})
	if err != nil {
		return nil, mapError(err)
	}

	return &pb.WorkflowResponse{
		Workflow: toProtoWorkflow(workflow),
	}, nil
}

func (s *ServiceAPIServer) UpdateWorkflow(ctx context.Context, req *pb.UpdateWorkflowRequest) (*pb.WorkflowResponse, error) {
	wfUUID, err := uuid.Parse(req.Id)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid workflow ID")
	}

	var nodes []serviceapi.NodeInput
	if req.Nodes != nil {
		nodes = make([]serviceapi.NodeInput, len(req.Nodes))
		for i, n := range req.Nodes {
			nodes[i] = serviceapi.NodeInput{
				ID:       n.Id,
				Name:     n.Name,
				Type:     n.Type,
				Config:   structToMap(n.Config),
				Position: structToMap(n.Position),
			}
		}
	}

	var edges []serviceapi.EdgeInput
	if req.Edges != nil {
		edges = make([]serviceapi.EdgeInput, len(req.Edges))
		for i, e := range req.Edges {
			ei := serviceapi.EdgeInput{
				ID:           e.Id,
				From:         e.From,
				To:           e.To,
				SourceHandle: e.SourceHandle,
				Condition:    structToMap(e.Condition),
			}
			if e.Loop != nil {
				ei.Loop = &serviceapi.LoopInput{MaxIterations: int(e.Loop.MaxIterations)}
			}
			edges[i] = ei
		}
	}

	var resources []serviceapi.ResourceInput
	if req.Resources != nil {
		resources = make([]serviceapi.ResourceInput, len(req.Resources))
		for i, r := range req.Resources {
			resources[i] = serviceapi.ResourceInput{
				ResourceID: r.ResourceId,
				Alias:      r.Alias,
				AccessType: r.AccessType,
			}
		}
	}

	workflow, err := s.ops.UpdateWorkflow(ctx, serviceapi.UpdateWorkflowParams{
		WorkflowID:  wfUUID,
		Name:        req.Name,
		Description: req.Description,
		Variables:   structToMap(req.Variables),
		Metadata:    structToMap(req.Metadata),
		Nodes:       nodes,
		Edges:       edges,
		Resources:   resources,
	})
	if err != nil {
		return nil, mapError(err)
	}

	return &pb.WorkflowResponse{
		Workflow: toProtoWorkflow(workflow),
	}, nil
}

func (s *ServiceAPIServer) DeleteWorkflow(ctx context.Context, req *pb.DeleteWorkflowRequest) (*pb.DeleteResponse, error) {
	wfUUID, err := uuid.Parse(req.Id)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid workflow ID")
	}

	if err := s.ops.DeleteWorkflow(ctx, serviceapi.DeleteWorkflowParams{
		WorkflowID: wfUUID,
	}); err != nil {
		return nil, mapError(err)
	}

	return &pb.DeleteResponse{Message: "workflow deleted successfully"}, nil
}
