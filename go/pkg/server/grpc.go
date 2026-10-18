package server

import (
	"fmt"
	"net"

	"google.golang.org/grpc"

	"github.com/smilemakc/mbflow/go/api/proto/serviceapipb"
	serviceapigrpc "github.com/smilemakc/mbflow/go/internal/infrastructure/api/grpc"
)

func (s *Server) setupGRPCServer() error {
	if !s.config.GRPCServiceAPI.Enabled {
		s.logger.Info("gRPC Service API server disabled")
		return nil
	}

	ops := s.operations()

	s.serviceAPI.GRPCServer = serviceapigrpc.NewServiceAPIServer(ops)

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
