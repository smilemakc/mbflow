package grpc

import (
	"context"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	pb "github.com/smilemakc/mbflow/api/proto/serviceapipb"
	"github.com/smilemakc/mbflow/internal/application/serviceapi"
)

func (s *ServiceAPIServer) ListTriggers(ctx context.Context, req *pb.ListTriggersRequest) (*pb.ListTriggersResponse, error) {
	params := serviceapi.ListTriggersParams{
		Limit:  int(req.Limit),
		Offset: int(req.Offset),
	}
	if req.WorkflowId != "" {
		wfUUID, err := uuid.Parse(req.WorkflowId)
		if err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "invalid workflow_id")
		}
		params.WorkflowID = &wfUUID
	}
	if req.Type != "" {
		params.Type = &req.Type
	}
	if params.Limit == 0 {
		params.Limit = 50
	}

	result, err := s.ops.ListTriggers(ctx, params)
	if err != nil {
		return nil, mapError(err)
	}

	return &pb.ListTriggersResponse{
		Triggers: toProtoTriggers(result.Triggers),
		Total:    int32(result.Total),
		Limit:    int32(params.Limit),
		Offset:   int32(params.Offset),
	}, nil
}

func (s *ServiceAPIServer) CreateTrigger(ctx context.Context, req *pb.CreateTriggerRequest) (*pb.TriggerResponse, error) {
	trigger, err := s.ops.CreateTrigger(ctx, serviceapi.CreateTriggerParams{
		WorkflowID:  req.WorkflowId,
		Name:        req.Name,
		Description: req.Description,
		Type:        req.Type,
		Config:      structToMap(req.Config),
		Enabled:     req.Enabled,
	})
	if err != nil {
		return nil, mapError(err)
	}

	return &pb.TriggerResponse{
		Trigger: toProtoTrigger(trigger),
	}, nil
}

func (s *ServiceAPIServer) UpdateTrigger(ctx context.Context, req *pb.UpdateTriggerRequest) (*pb.TriggerResponse, error) {
	triggerUUID, err := uuid.Parse(req.Id)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid trigger ID")
	}

	params := serviceapi.UpdateTriggerParams{
		TriggerID:   triggerUUID,
		Name:        req.Name,
		Description: req.Description,
		Type:        req.Type,
		Config:      structToMap(req.Config),
	}
	if req.Enabled != nil {
		enabled := *req.Enabled
		params.Enabled = &enabled
	}

	trigger, err := s.ops.UpdateTrigger(ctx, params)
	if err != nil {
		return nil, mapError(err)
	}

	return &pb.TriggerResponse{
		Trigger: toProtoTrigger(trigger),
	}, nil
}

func (s *ServiceAPIServer) DeleteTrigger(ctx context.Context, req *pb.DeleteTriggerRequest) (*pb.DeleteResponse, error) {
	triggerUUID, err := uuid.Parse(req.Id)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid trigger ID")
	}

	if err := s.ops.DeleteTrigger(ctx, serviceapi.DeleteTriggerParams{
		TriggerID: triggerUUID,
	}); err != nil {
		return nil, mapError(err)
	}

	return &pb.DeleteResponse{Message: "trigger deleted successfully"}, nil
}
