package grpc

import (
	"context"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	pb "github.com/smilemakc/mbflow/api/proto/serviceapipb"
	"github.com/smilemakc/mbflow/internal/application/serviceapi"
)

func (s *ServiceAPIServer) ListExecutions(ctx context.Context, req *pb.ListExecutionsRequest) (*pb.ListExecutionsResponse, error) {
	params := serviceapi.ListExecutionsParams{
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
	if req.Status != "" {
		params.Status = &req.Status
	}
	if params.Limit == 0 {
		params.Limit = 50
	}

	result, err := s.ops.ListExecutions(ctx, params)
	if err != nil {
		return nil, mapError(err)
	}

	return &pb.ListExecutionsResponse{
		Executions: toProtoExecutions(result.Executions),
		Total:      int32(result.Total),
		Limit:      int32(params.Limit),
		Offset:     int32(params.Offset),
	}, nil
}

func (s *ServiceAPIServer) GetExecution(ctx context.Context, req *pb.GetExecutionRequest) (*pb.ExecutionResponse, error) {
	execUUID, err := uuid.Parse(req.Id)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid execution ID")
	}

	execution, err := s.ops.GetExecution(ctx, serviceapi.GetExecutionParams{
		ExecutionID: execUUID,
	})
	if err != nil {
		return nil, mapError(err)
	}

	return &pb.ExecutionResponse{
		Execution: toProtoExecution(execution),
	}, nil
}

func (s *ServiceAPIServer) StartExecution(ctx context.Context, req *pb.StartExecutionRequest) (*pb.ExecutionResponse, error) {
	params := serviceapi.StartExecutionParams{
		WorkflowID: req.WorkflowId,
		Input:      structToMap(req.Input),
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

	execution, err := s.ops.StartExecution(ctx, params)
	if err != nil {
		return nil, mapError(err)
	}

	return &pb.ExecutionResponse{
		Execution: toProtoExecution(execution),
	}, nil
}

func (s *ServiceAPIServer) CancelExecution(ctx context.Context, req *pb.CancelExecutionRequest) (*pb.ExecutionResponse, error) {
	execUUID, err := uuid.Parse(req.Id)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid execution ID")
	}

	if err := s.ops.CancelExecution(ctx, serviceapi.CancelExecutionParams{
		ExecutionID: execUUID,
	}); err != nil {
		return nil, mapError(err)
	}

	return &pb.ExecutionResponse{}, nil
}

func (s *ServiceAPIServer) RetryExecution(ctx context.Context, req *pb.RetryExecutionRequest) (*pb.ExecutionResponse, error) {
	execUUID, err := uuid.Parse(req.Id)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid execution ID")
	}

	if err := s.ops.RetryExecution(ctx, serviceapi.RetryExecutionParams{
		ExecutionID: execUUID,
	}); err != nil {
		return nil, mapError(err)
	}

	return &pb.ExecutionResponse{}, nil
}
