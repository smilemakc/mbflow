package grpcclient

import (
	"context"
	"fmt"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"

	"github.com/smilemakc/mbflow/go/sdk/internal"
	pb "github.com/smilemakc/mbflow/go/sdk/internal/pb"
)

// Config configures the gRPC transport.
type Config struct {
	SystemKey  string
	OnBehalfOf string
	Insecure   bool
}

// Transport wraps the gRPC connection and provides auth-aware access to the proto client.
type Transport struct {
	conn   *grpc.ClientConn
	client pb.MBFlowServiceAPIClient
	config *Config
}

// New creates a new gRPC transport.
func New(address string, config *Config) (*Transport, error) {
	var opts []grpc.DialOption
	if config.Insecure {
		opts = append(opts, grpc.WithTransportCredentials(insecure.NewCredentials()))
	}

	conn, err := grpc.NewClient(address, opts...)
	if err != nil {
		return nil, fmt.Errorf("grpc dial: %w", err)
	}

	return &Transport{
		conn:   conn,
		client: pb.NewMBFlowServiceAPIClient(conn),
		config: config,
	}, nil
}

// Client returns the underlying gRPC client for direct service calls.
func (t *Transport) Client() pb.MBFlowServiceAPIClient {
	return t.client
}

// AuthContext adds authentication metadata to the context.
func (t *Transport) AuthContext(ctx context.Context, onBehalfOf string) context.Context {
	md := metadata.New(map[string]string{})
	if t.config.SystemKey != "" {
		md.Set("x-system-key", t.config.SystemKey)
	}
	obo := onBehalfOf
	if obo == "" {
		obo = t.config.OnBehalfOf
	}
	if obo != "" {
		md.Set("x-on-behalf-of", obo)
	}
	return metadata.NewOutgoingContext(ctx, md)
}

// Do implements internal.Transport but gRPC services use the client directly.
func (t *Transport) Do(ctx context.Context, req *internal.Request) (*internal.Response, error) {
	return nil, fmt.Errorf("gRPC transport: use typed client methods instead of Do()")
}

// Close closes the underlying gRPC connection.
func (t *Transport) Close() error {
	return t.conn.Close()
}
