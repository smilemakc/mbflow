package server

import (
	"fmt"
	"net"

	"google.golang.org/grpc"

	"github.com/smilemakc/mbflow/api/proto/serviceapipb"
	"github.com/smilemakc/mbflow/internal/application/serviceapi"
	serviceapigrpc "github.com/smilemakc/mbflow/internal/infrastructure/api/grpc"
)

func (s *Server) setupGRPCServer() error {
	if !s.config.GRPCServiceAPI.Enabled {
		s.logger.Info("gRPC Service API server disabled")
		return nil
	}

	s.serviceAPI.Operations = &serviceapi.Operations{
		WorkflowRepo:    s.data.WorkflowRepo,
		ExecutionRepo:   s.data.ExecutionRepo,
		TriggerRepo:     s.data.TriggerRepo,
		CredentialsRepo: s.data.CredentialsRepo,
		ExecutionMgr:    s.execution.ExecutionManager,
		ExecutorManager: s.execution.ExecutorManager,
		EncryptionSvc:   s.auth.EncryptionService,
		AuditService:    s.serviceAPI.AuditService,
		Logger:          s.logger,
	}

	s.serviceAPI.GRPCServer = serviceapigrpc.NewServiceAPIServer(s.serviceAPI.Operations)

	s.serviceAPI.GRPCServerInstance = grpc.NewServer(
		grpc.ChainUnaryInterceptor(
			serviceapigrpc.SystemKeyAuthInterceptor(s.serviceAPI.SystemKeyService),
			serviceapigrpc.ImpersonationInterceptor(s.data.UserRepo, s.config.ServiceAPI.SystemUserID),
			serviceapigrpc.AuditInterceptor(s.serviceAPI.AuditService, s.logger),
		),
	)

	serviceapipb.RegisterMBFlowServiceAPIServer(s.serviceAPI.GRPCServerInstance, s.serviceAPI.GRPCServer)

	lis, err := net.Listen("tcp", s.config.GRPCServiceAPI.Address)
	if err != nil {
		return fmt.Errorf("failed to listen on %s: %w", s.config.GRPCServiceAPI.Address, err)
	}
	s.serviceAPI.GRPCListener = lis

	s.logger.Info("gRPC Service API server configured",
		"address", s.config.GRPCServiceAPI.Address,
	)
	return nil
}
