package grpc

import (
	"errors"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/smilemakc/mbflow/go/internal/application/serviceapi"
	"github.com/smilemakc/mbflow/go/pkg/models"
)

// mapError converts domain/operation errors to gRPC status errors.
func mapError(err error) error {
	if err == nil {
		return nil
	}

	var opErr *serviceapi.OperationError
	if errors.As(err, &opErr) {
		return status.Errorf(httpStatusToGRPCCode(opErr.HTTPStatus), "%s", opErr.Message)
	}

	switch {
	case errors.Is(err, models.ErrWorkflowNotFound):
		return status.Errorf(codes.NotFound, "workflow not found")
	case errors.Is(err, models.ErrExecutionNotFound):
		return status.Errorf(codes.NotFound, "execution not found")
	case errors.Is(err, models.ErrTriggerNotFound):
		return status.Errorf(codes.NotFound, "trigger not found")
	case errors.Is(err, models.ErrResourceNotFound):
		return status.Errorf(codes.NotFound, "resource not found")
	case errors.Is(err, models.ErrInvalidID):
		return status.Errorf(codes.InvalidArgument, "invalid ID format")
	case errors.Is(err, models.ErrUnauthorized):
		return status.Errorf(codes.Unauthenticated, "authentication required")
	case errors.Is(err, models.ErrForbidden):
		return status.Errorf(codes.PermissionDenied, "access denied")
	case errors.Is(err, models.ErrWorkflowExists):
		return status.Errorf(codes.AlreadyExists, "workflow already exists")
	case errors.Is(err, models.ErrValidationFailed):
		return status.Errorf(codes.InvalidArgument, "validation failed")
	default:
		return status.Errorf(codes.Internal, "internal error")
	}
}

func httpStatusToGRPCCode(httpStatus int) codes.Code {
	switch {
	case httpStatus >= 200 && httpStatus < 300:
		return codes.OK
	case httpStatus == 400:
		return codes.InvalidArgument
	case httpStatus == 401:
		return codes.Unauthenticated
	case httpStatus == 403:
		return codes.PermissionDenied
	case httpStatus == 404:
		return codes.NotFound
	case httpStatus == 409:
		return codes.AlreadyExists
	case httpStatus == 429:
		return codes.ResourceExhausted
	case httpStatus == 501:
		return codes.Unimplemented
	default:
		return codes.Internal
	}
}
