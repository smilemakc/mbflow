package serviceapipb

import (
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// RunEphemeralExecutionRequest is a manually-defined message for the
// RunEphemeralExecution RPC. This file will be replaced when protoc
// regenerates from the updated .proto.
type RunEphemeralExecutionRequest struct {
	Workflow         *Workflow              `protobuf:"bytes,1,opt,name=workflow,proto3" json:"workflow,omitempty"`
	Input            *structpb.Struct       `protobuf:"bytes,2,opt,name=input,proto3" json:"input,omitempty"`
	Mode             string                 `protobuf:"bytes,3,opt,name=mode,proto3" json:"mode,omitempty"`
	CredentialIds    []string               `protobuf:"bytes,4,rep,name=credential_ids,json=credentialIds,proto3" json:"credential_ids,omitempty"`
	Variables        *structpb.Struct       `protobuf:"bytes,5,opt,name=variables,proto3" json:"variables,omitempty"`
	PersistExecution bool                   `protobuf:"varint,6,opt,name=persist_execution,json=persistExecution,proto3" json:"persist_execution,omitempty"`
	Webhooks         []*WebhookSubscription `protobuf:"bytes,7,rep,name=webhooks,proto3" json:"webhooks,omitempty"`
}

// StreamExecutionEventsRequest is a manually-defined message for the
// StreamExecutionEvents RPC.
type StreamExecutionEventsRequest struct {
	ExecutionId string `protobuf:"bytes,1,opt,name=execution_id,json=executionId,proto3" json:"execution_id,omitempty"`
}

// ExecutionEvent is a manually-defined message representing a single
// streamed execution event.
type ExecutionEvent struct {
	EventId     string                 `protobuf:"bytes,1,opt,name=event_id,json=eventId,proto3" json:"event_id,omitempty"`
	ExecutionId string                 `protobuf:"bytes,2,opt,name=execution_id,json=executionId,proto3" json:"execution_id,omitempty"`
	Sequence    int64                  `protobuf:"varint,3,opt,name=sequence,proto3" json:"sequence,omitempty"`
	EventType   string                 `protobuf:"bytes,4,opt,name=event_type,json=eventType,proto3" json:"event_type,omitempty"`
	Payload     *structpb.Struct       `protobuf:"bytes,6,opt,name=payload,proto3" json:"payload,omitempty"`
	SentAt      *timestamppb.Timestamp `protobuf:"bytes,7,opt,name=sent_at,json=sentAt,proto3" json:"sent_at,omitempty"`
}
