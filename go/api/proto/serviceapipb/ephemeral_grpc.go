package serviceapipb

import (
	context "context"
	grpc "google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
)

// MBFlowServiceAPI_StreamExecutionEventsServer is the server-side streaming
// interface for the StreamExecutionEvents RPC.
// This file will be replaced when protoc regenerates from the updated .proto.
type MBFlowServiceAPI_StreamExecutionEventsServer interface {
	Send(*ExecutionEvent) error
	grpc.ServerStream
}

const (
	MBFlowServiceAPI_RunEphemeralExecution_FullMethodName = "/serviceapi.MBFlowServiceAPI/RunEphemeralExecution"
	MBFlowServiceAPI_StreamExecutionEvents_FullMethodName = "/serviceapi.MBFlowServiceAPI/StreamExecutionEvents"
)

// Default unimplemented stubs for the new RPCs.
// These are added to UnimplementedMBFlowServiceAPIServer so that
// embedding it still satisfies forward compatibility.
// When protoc regenerates, these will be included in the generated code
// and this file should be deleted.

func (UnimplementedMBFlowServiceAPIServer) RunEphemeralExecution(context.Context, *RunEphemeralExecutionRequest) (*ExecutionResponse, error) {
	return nil, status.Error(codes.Unimplemented, "method RunEphemeralExecution not implemented")
}

func (UnimplementedMBFlowServiceAPIServer) StreamExecutionEvents(*StreamExecutionEventsRequest, MBFlowServiceAPI_StreamExecutionEventsServer) error {
	return status.Error(codes.Unimplemented, "method StreamExecutionEvents not implemented")
}
