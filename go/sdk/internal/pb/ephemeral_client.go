package serviceapipb

import (
	"context"

	"google.golang.org/grpc"
)

const (
	MBFlowServiceAPI_RunEphemeralExecution_FullMethodName = "/serviceapi.MBFlowServiceAPI/RunEphemeralExecution"
	MBFlowServiceAPI_StreamExecutionEvents_FullMethodName = "/serviceapi.MBFlowServiceAPI/StreamExecutionEvents"
)

// EphemeralClient provides client methods for the ephemeral execution RPCs
// that are not yet in the generated MBFlowServiceAPIClient interface.
type EphemeralClient interface {
	RunEphemeralExecution(ctx context.Context, in *RunEphemeralExecutionRequest, opts ...grpc.CallOption) (*ExecutionResponse, error)
	StreamExecutionEvents(ctx context.Context, in *StreamExecutionEventsRequest, opts ...grpc.CallOption) (MBFlowServiceAPI_StreamExecutionEventsClient, error)
}

// MBFlowServiceAPI_StreamExecutionEventsClient is the client-side streaming
// interface for the StreamExecutionEvents RPC.
type MBFlowServiceAPI_StreamExecutionEventsClient interface {
	Recv() (*ExecutionEvent, error)
	grpc.ClientStream
}

type ephemeralClient struct {
	cc grpc.ClientConnInterface
}

// NewEphemeralClient creates a new EphemeralClient from the same connection
// used by the main MBFlowServiceAPIClient.
func NewEphemeralClient(cc grpc.ClientConnInterface) EphemeralClient {
	return &ephemeralClient{cc: cc}
}

func (c *ephemeralClient) RunEphemeralExecution(ctx context.Context, in *RunEphemeralExecutionRequest, opts ...grpc.CallOption) (*ExecutionResponse, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(ExecutionResponse)
	err := c.cc.Invoke(ctx, MBFlowServiceAPI_RunEphemeralExecution_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *ephemeralClient) StreamExecutionEvents(ctx context.Context, in *StreamExecutionEventsRequest, opts ...grpc.CallOption) (MBFlowServiceAPI_StreamExecutionEventsClient, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	stream, err := c.cc.NewStream(ctx, &grpc.StreamDesc{
		StreamName:    "StreamExecutionEvents",
		ServerStreams: true,
	}, MBFlowServiceAPI_StreamExecutionEvents_FullMethodName, cOpts...)
	if err != nil {
		return nil, err
	}
	x := &streamExecutionEventsClient{ClientStream: stream}
	if err := x.ClientStream.SendMsg(in); err != nil {
		return nil, err
	}
	if err := x.ClientStream.CloseSend(); err != nil {
		return nil, err
	}
	return x, nil
}

type streamExecutionEventsClient struct {
	grpc.ClientStream
}

func (x *streamExecutionEventsClient) Recv() (*ExecutionEvent, error) {
	m := new(ExecutionEvent)
	if err := x.ClientStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}
