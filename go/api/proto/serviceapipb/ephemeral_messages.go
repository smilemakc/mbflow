package serviceapipb

import (
	structpb "google.golang.org/protobuf/types/known/structpb"
	timestamppb "google.golang.org/protobuf/types/known/timestamppb"
)

// RunEphemeralExecutionRequest is a manually-defined message struct.
// This file will be replaced when protoc regenerates from the updated .proto.
type RunEphemeralExecutionRequest struct {
	Workflow         *Workflow              `protobuf:"bytes,1,opt,name=workflow,proto3" json:"workflow,omitempty"`
	Input            *structpb.Struct       `protobuf:"bytes,2,opt,name=input,proto3" json:"input,omitempty"`
	Mode             string                 `protobuf:"bytes,3,opt,name=mode,proto3" json:"mode,omitempty"`
	CredentialIds    []string               `protobuf:"bytes,4,rep,name=credential_ids,json=credentialIds,proto3" json:"credential_ids,omitempty"`
	Variables        *structpb.Struct       `protobuf:"bytes,5,opt,name=variables,proto3" json:"variables,omitempty"`
	PersistExecution bool                   `protobuf:"varint,6,opt,name=persist_execution,json=persistExecution,proto3" json:"persist_execution,omitempty"`
	Webhooks         []*WebhookSubscription `protobuf:"bytes,7,rep,name=webhooks,proto3" json:"webhooks,omitempty"`
}

func (x *RunEphemeralExecutionRequest) GetWorkflow() *Workflow {
	if x != nil {
		return x.Workflow
	}
	return nil
}

func (x *RunEphemeralExecutionRequest) GetInput() *structpb.Struct {
	if x != nil {
		return x.Input
	}
	return nil
}

func (x *RunEphemeralExecutionRequest) GetMode() string {
	if x != nil {
		return x.Mode
	}
	return ""
}

func (x *RunEphemeralExecutionRequest) GetCredentialIds() []string {
	if x != nil {
		return x.CredentialIds
	}
	return nil
}

func (x *RunEphemeralExecutionRequest) GetVariables() *structpb.Struct {
	if x != nil {
		return x.Variables
	}
	return nil
}

func (x *RunEphemeralExecutionRequest) GetPersistExecution() bool {
	if x != nil {
		return x.PersistExecution
	}
	return false
}

func (x *RunEphemeralExecutionRequest) GetWebhooks() []*WebhookSubscription {
	if x != nil {
		return x.Webhooks
	}
	return nil
}

// StreamExecutionEventsRequest is a manually-defined message struct.
// This file will be replaced when protoc regenerates from the updated .proto.
type StreamExecutionEventsRequest struct {
	ExecutionId   string `protobuf:"bytes,1,opt,name=execution_id,json=executionId,proto3" json:"execution_id,omitempty"`
	AfterSequence int64  `protobuf:"varint,2,opt,name=after_sequence,json=afterSequence,proto3" json:"after_sequence,omitempty"`
}

func (x *StreamExecutionEventsRequest) GetExecutionId() string {
	if x != nil {
		return x.ExecutionId
	}
	return ""
}

func (x *StreamExecutionEventsRequest) GetAfterSequence() int64 {
	if x != nil {
		return x.AfterSequence
	}
	return 0
}

// ExecutionEvent is a manually-defined message struct.
// This file will be replaced when protoc regenerates from the updated .proto.
type ExecutionEvent struct {
	EventId        string                 `protobuf:"bytes,1,opt,name=event_id,json=eventId,proto3" json:"event_id,omitempty"`
	ExecutionId    string                 `protobuf:"bytes,2,opt,name=execution_id,json=executionId,proto3" json:"execution_id,omitempty"`
	Sequence       int64                  `protobuf:"varint,3,opt,name=sequence,proto3" json:"sequence,omitempty"`
	EventType      string                 `protobuf:"bytes,4,opt,name=event_type,json=eventType,proto3" json:"event_type,omitempty"`
	WorkflowSource string                 `protobuf:"bytes,5,opt,name=workflow_source,json=workflowSource,proto3" json:"workflow_source,omitempty"`
	Payload        *structpb.Struct       `protobuf:"bytes,6,opt,name=payload,proto3" json:"payload,omitempty"`
	SentAt         *timestamppb.Timestamp `protobuf:"bytes,7,opt,name=sent_at,json=sentAt,proto3" json:"sent_at,omitempty"`
}

func (x *ExecutionEvent) GetEventId() string {
	if x != nil {
		return x.EventId
	}
	return ""
}

func (x *ExecutionEvent) GetExecutionId() string {
	if x != nil {
		return x.ExecutionId
	}
	return ""
}

func (x *ExecutionEvent) GetSequence() int64 {
	if x != nil {
		return x.Sequence
	}
	return 0
}

func (x *ExecutionEvent) GetEventType() string {
	if x != nil {
		return x.EventType
	}
	return ""
}

func (x *ExecutionEvent) GetWorkflowSource() string {
	if x != nil {
		return x.WorkflowSource
	}
	return ""
}

func (x *ExecutionEvent) GetPayload() *structpb.Struct {
	if x != nil {
		return x.Payload
	}
	return nil
}

func (x *ExecutionEvent) GetSentAt() *timestamppb.Timestamp {
	if x != nil {
		return x.SentAt
	}
	return nil
}
