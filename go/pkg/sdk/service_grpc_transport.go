package sdk

import (
	"context"
	"fmt"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"

	pb "github.com/smilemakc/mbflow/go/api/proto/serviceapipb"
)

// grpcServiceTransport wraps the gRPC client for the Service API.
type grpcServiceTransport struct {
	conn       *grpc.ClientConn
	client     pb.MBFlowServiceAPIClient
	systemKey  string
	onBehalfOf string
}

// newGRPCServiceTransport creates a gRPC transport from the client config.
func newGRPCServiceTransport(cfg ServiceClientConfig) (*grpcServiceTransport, error) {
	if cfg.GRPCAddress == "" {
		return nil, fmt.Errorf("gRPC address is required for gRPC transport")
	}

	var opts []grpc.DialOption
	if cfg.GRPCInsecure {
		opts = append(opts, grpc.WithTransportCredentials(insecure.NewCredentials()))
	}

	timeout := cfg.Timeout
	if timeout == 0 {
		timeout = 30 * time.Second
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	conn, err := grpc.DialContext(ctx, cfg.GRPCAddress, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to gRPC server: %w", err)
	}

	return &grpcServiceTransport{
		conn:       conn,
		client:     pb.NewMBFlowServiceAPIClient(conn),
		systemKey:  cfg.SystemKey,
		onBehalfOf: cfg.OnBehalfOf,
	}, nil
}

// withOnBehalfOf returns a copy of the transport with a different impersonation user.
func (t *grpcServiceTransport) withOnBehalfOf(userID string) *grpcServiceTransport {
	return &grpcServiceTransport{
		conn:       t.conn,
		client:     t.client,
		systemKey:  t.systemKey,
		onBehalfOf: userID,
	}
}

// close closes the underlying gRPC connection.
func (t *grpcServiceTransport) close() error {
	if t.conn != nil {
		return t.conn.Close()
	}
	return nil
}

// contextWithAuth returns a context with gRPC metadata for authentication.
func (t *grpcServiceTransport) contextWithAuth(ctx context.Context) context.Context {
	md := metadata.Pairs("x-system-key", t.systemKey)
	if t.onBehalfOf != "" {
		md.Append("x-on-behalf-of", t.onBehalfOf)
	}
	return metadata.NewOutgoingContext(ctx, md)
}
