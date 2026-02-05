package grpc

import (
	"context"

	"github.com/google/uuid"

	pb "github.com/smilemakc/mbflow/api/proto/serviceapipb"
	"github.com/smilemakc/mbflow/internal/application/serviceapi"
)

func (s *ServiceAPIServer) ListAuditLog(ctx context.Context, req *pb.ListAuditLogRequest) (*pb.ListAuditLogResponse, error) {
	params := serviceapi.ListAuditLogParams{
		Limit:  int(req.Limit),
		Offset: int(req.Offset),
	}

	if req.ServiceName != "" {
		params.ServiceName = &req.ServiceName
	}
	if req.Action != "" {
		params.Action = &req.Action
	}
	if req.ResourceType != "" {
		params.ResourceType = &req.ResourceType
	}
	if req.ImpersonatedUserId != "" {
		if parsed, err := uuid.Parse(req.ImpersonatedUserId); err == nil {
			params.ImpersonatedUserID = &parsed
		}
	}
	params.DateFrom = optionalTimestamp(req.DateFrom)
	params.DateTo = optionalTimestamp(req.DateTo)

	if params.Limit == 0 {
		params.Limit = 50
	}

	result, err := s.ops.ListAuditLog(ctx, params)
	if err != nil {
		return nil, mapError(err)
	}

	return &pb.ListAuditLogResponse{
		AuditLogs: toProtoAuditLogEntries(result.AuditLogs),
		Total:     result.Total,
		Limit:     int32(params.Limit),
		Offset:    int32(params.Offset),
	}, nil
}
