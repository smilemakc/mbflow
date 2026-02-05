package grpc

import (
	"github.com/smilemakc/mbflow/api/proto/serviceapipb"
	"github.com/smilemakc/mbflow/internal/application/serviceapi"
)

// ServiceAPIServer implements the MBFlowServiceAPI gRPC service.
type ServiceAPIServer struct {
	serviceapipb.UnimplementedMBFlowServiceAPIServer
	ops *serviceapi.Operations
}

// NewServiceAPIServer creates a new gRPC server backed by the operations layer.
func NewServiceAPIServer(ops *serviceapi.Operations) *ServiceAPIServer {
	return &ServiceAPIServer{ops: ops}
}
