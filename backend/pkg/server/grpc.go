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

	s.serviceAPIOps = &serviceapi.Operations{
		WorkflowRepo:    s.workflowRepo,
		ExecutionRepo:   s.executionRepo,
		TriggerRepo:     s.triggerRepo,
		CredentialsRepo: s.credentialsRepo,
		ExecutionMgr:    s.executionManager,
		ExecutorManager: s.executorManager,
		EncryptionSvc:   s.encryptionService,
		AuditService:    s.auditService,
		Logger:          s.logger,
	}

	s.serviceAPIGRPC = serviceapigrpc.NewServiceAPIServer(s.serviceAPIOps)

	s.grpcServer = grpc.NewServer(
		grpc.ChainUnaryInterceptor(
			serviceapigrpc.SystemKeyAuthInterceptor(s.systemKeyService_),
			serviceapigrpc.ImpersonationInterceptor(s.userRepo, s.config.ServiceAPI.SystemUserID),
			serviceapigrpc.AuditInterceptor(s.auditService, s.logger),
		),
	)

	serviceapipb.RegisterMBFlowServiceAPIServer(s.grpcServer, s.serviceAPIGRPC)

	lis, err := net.Listen("tcp", s.config.GRPCServiceAPI.Address)
	if err != nil {
		return fmt.Errorf("failed to listen on %s: %w", s.config.GRPCServiceAPI.Address, err)
	}
	s.grpcListener = lis

	s.logger.Info("gRPC Service API server configured",
		"address", s.config.GRPCServiceAPI.Address,
	)
	return nil
}
