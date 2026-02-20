package grpc

import (
	"context"
	"errors"
	"strings"

	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	"github.com/smilemakc/mbflow/go/internal/application/systemkey"
	"github.com/smilemakc/mbflow/go/internal/infrastructure/logger"
	"github.com/smilemakc/mbflow/go/internal/infrastructure/storage"
	"github.com/smilemakc/mbflow/go/pkg/models"
)

// SystemKeyAuthInterceptor authenticates requests using system keys from gRPC metadata.
func SystemKeyAuthInterceptor(svc *systemkey.Service) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		token, err := extractSystemKeyFromMetadata(ctx)
		if err != nil {
			return nil, status.Errorf(codes.Unauthenticated, "system key required")
		}

		if !strings.HasPrefix(token, models.SystemKeyPrefix) {
			return nil, status.Errorf(codes.Unauthenticated, "system key required")
		}

		key, err := svc.ValidateKey(ctx, token)
		if err != nil {
			if errors.Is(err, models.ErrSystemKeyRevoked) {
				return nil, status.Errorf(codes.Unauthenticated, "system key has been revoked")
			}
			if errors.Is(err, models.ErrSystemKeyExpired) {
				return nil, status.Errorf(codes.Unauthenticated, "system key has expired")
			}
			return nil, status.Errorf(codes.Unauthenticated, "invalid system key")
		}

		ctx = ContextWithAuthMethod(ctx, "system_key")
		ctx = ContextWithSystemKeyID(ctx, key.ID)
		ctx = ContextWithServiceName(ctx, key.ServiceName)
		ctx = ContextWithIsAdmin(ctx, true)

		return handler(ctx, req)
	}
}

// ImpersonationInterceptor reads X-On-Behalf-Of metadata and sets the user context.
func ImpersonationInterceptor(userRepo *storage.UserRepository, systemUserID string) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		onBehalfOf := getMetadataValue(ctx, "x-on-behalf-of")

		if onBehalfOf != "" {
			userUUID, err := uuid.Parse(onBehalfOf)
			if err != nil {
				return nil, status.Errorf(codes.InvalidArgument, "invalid user ID format")
			}

			user, err := userRepo.FindByID(ctx, userUUID)
			if err != nil {
				return nil, status.Errorf(codes.Internal, "failed to validate user")
			}
			if user == nil {
				return nil, status.Errorf(codes.InvalidArgument, "user not found for impersonation")
			}

			ctx = ContextWithUserID(ctx, userUUID.String())
			ctx = ContextWithImpersonated(ctx, true)
		} else {
			ctx = ContextWithUserID(ctx, systemUserID)
			ctx = ContextWithImpersonated(ctx, false)
		}

		return handler(ctx, req)
	}
}

// AuditInterceptor logs actions via AuditService after the handler executes.
func AuditInterceptor(auditService *systemkey.AuditService, log *logger.Logger) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		resp, handlerErr := handler(ctx, req)

		systemKeyID, _ := SystemKeyIDFromContext(ctx)
		serviceName, _ := ServiceNameFromContext(ctx)

		var impersonatedUserID *string
		if ImpersonatedFromContext(ctx) {
			if uid, ok := UserIDFromContext(ctx); ok {
				impersonatedUserID = &uid
			}
		}

		action, resourceType, resourceID := parseGRPCMethod(info.FullMethod)

		responseStatus := 0
		if handlerErr != nil {
			if st, ok := status.FromError(handlerErr); ok {
				responseStatus = grpcCodeToHTTPStatus(st.Code())
			}
		} else {
			responseStatus = 200
		}

		go func() {
			if err := auditService.LogAction(
				ctx,
				systemKeyID,
				serviceName,
				action,
				resourceType,
				resourceID,
				impersonatedUserID,
				"gRPC",
				info.FullMethod,
				nil,
				"",
				responseStatus,
			); err != nil {
				log.Error("Failed to log gRPC audit action", "error", err, "method", info.FullMethod)
			}
		}()

		return resp, handlerErr
	}
}

func extractSystemKeyFromMetadata(ctx context.Context) (string, error) {
	if key := getMetadataValue(ctx, "x-system-key"); key != "" {
		return key, nil
	}

	if auth := getMetadataValue(ctx, "authorization"); auth != "" {
		parts := strings.SplitN(auth, " ", 2)
		if len(parts) == 2 && strings.EqualFold(parts[0], "Bearer") {
			if strings.HasPrefix(parts[1], models.SystemKeyPrefix) {
				return parts[1], nil
			}
		}
	}

	return "", errors.New("no system key provided")
}

func getMetadataValue(ctx context.Context, key string) string {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return ""
	}
	values := md.Get(key)
	if len(values) == 0 {
		return ""
	}
	return values[0]
}

// parseGRPCMethod extracts action and resource type from a gRPC method name.
// e.g., "/serviceapi.MBFlowServiceAPI/ListWorkflows" -> ("workflow.list", "workflow", nil)
func parseGRPCMethod(fullMethod string) (action, resourceType string, resourceID *string) {
	parts := strings.Split(fullMethod, "/")
	if len(parts) < 3 {
		return "unknown", "unknown", nil
	}

	method := parts[2]

	prefixes := []struct {
		prefix string
		rt     string
	}{
		{"ListWorkflows", "workflow"},
		{"GetWorkflow", "workflow"},
		{"CreateWorkflow", "workflow"},
		{"UpdateWorkflow", "workflow"},
		{"DeleteWorkflow", "workflow"},
		{"ListExecutions", "execution"},
		{"GetExecution", "execution"},
		{"StartExecution", "execution"},
		{"CancelExecution", "execution"},
		{"RetryExecution", "execution"},
		{"ListTriggers", "trigger"},
		{"CreateTrigger", "trigger"},
		{"UpdateTrigger", "trigger"},
		{"DeleteTrigger", "trigger"},
		{"ListCredentials", "credential"},
		{"CreateCredential", "credential"},
		{"UpdateCredential", "credential"},
		{"DeleteCredential", "credential"},
		{"ListAuditLog", "audit_log"},
	}

	for _, p := range prefixes {
		if method == p.prefix {
			resourceType = p.rt
			break
		}
	}
	if resourceType == "" {
		resourceType = "unknown"
	}

	switch {
	case strings.HasPrefix(method, "List"):
		action = resourceType + ".list"
	case strings.HasPrefix(method, "Get"):
		action = resourceType + ".get"
	case strings.HasPrefix(method, "Create"):
		action = resourceType + ".create"
	case strings.HasPrefix(method, "Update"):
		action = resourceType + ".update"
	case strings.HasPrefix(method, "Delete"):
		action = resourceType + ".delete"
	case strings.HasPrefix(method, "Start"):
		action = resourceType + ".start"
	case strings.HasPrefix(method, "Cancel"):
		action = resourceType + ".cancel"
	case strings.HasPrefix(method, "Retry"):
		action = resourceType + ".retry"
	default:
		action = resourceType + "." + strings.ToLower(method)
	}

	return action, resourceType, nil
}

func grpcCodeToHTTPStatus(code codes.Code) int {
	switch code {
	case codes.OK:
		return 200
	case codes.InvalidArgument:
		return 400
	case codes.Unauthenticated:
		return 401
	case codes.PermissionDenied:
		return 403
	case codes.NotFound:
		return 404
	case codes.AlreadyExists:
		return 409
	case codes.Unimplemented:
		return 501
	default:
		return 500
	}
}
