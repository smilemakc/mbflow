package grpc

import (
	"context"
	"strings"
	"time"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	pb "github.com/smilemakc/mbflow/go/api/proto/serviceapipb"
	appobserver "github.com/smilemakc/mbflow/go/internal/application/observer"
	"github.com/smilemakc/mbflow/go/internal/application/serviceapi"
	storagemodels "github.com/smilemakc/mbflow/go/internal/infrastructure/storage/models"
	"github.com/smilemakc/mbflow/go/pkg/models"
)

func (s *ServiceAPIServer) RunEphemeralExecution(ctx context.Context, req *pb.RunEphemeralExecutionRequest) (*pb.ExecutionResponse, error) {
	if req.Workflow == nil {
		return nil, status.Errorf(codes.InvalidArgument, "workflow is required")
	}

	workflow := workflowFromProto(req.Workflow)

	params := serviceapi.EphemeralExecutionParams{
		Workflow:         workflow,
		Input:            structToMap(req.Input),
		Mode:             req.Mode,
		CredentialIDs:    req.CredentialIds,
		Variables:        structToMap(req.Variables),
		PersistExecution: req.PersistExecution,
	}

	if len(req.Webhooks) > 0 {
		params.Webhooks = make([]serviceapi.WebhookSubscription, len(req.Webhooks))
		for i, wh := range req.Webhooks {
			params.Webhooks[i] = serviceapi.WebhookSubscription{
				URL:     wh.Url,
				Events:  wh.Events,
				Headers: wh.Headers,
				NodeIDs: wh.NodeIds,
			}
		}
	}

	result, err := s.ops.StartEphemeralExecution(ctx, params)
	if err != nil {
		return nil, mapError(err)
	}

	return &pb.ExecutionResponse{
		Execution: toProtoExecution(result),
	}, nil
}

func (s *ServiceAPIServer) StreamExecutionEvents(req *pb.StreamExecutionEventsRequest, stream pb.MBFlowServiceAPI_StreamExecutionEventsServer) error {
	if req.GetExecutionId() == "" {
		return status.Error(codes.InvalidArgument, "execution_id is required")
	}

	executionUUID, err := uuid.Parse(req.GetExecutionId())
	if err != nil {
		return status.Error(codes.InvalidArgument, "invalid execution_id")
	}

	ctx := stream.Context()
	execModel, err := s.ops.ExecutionRepo.FindByID(ctx, executionUUID)
	if err != nil {
		if s.ops.ExecutionMgr != nil && s.ops.ExecutionMgr.HasEphemeralExecution(req.GetExecutionId()) {
			if s.ops.ExecutionMgr.IsEphemeralExecutionTerminal(req.GetExecutionId()) {
				// Execution already finished and no persisted row exists: close stream immediately.
				return nil
			}
			return s.streamEphemeralExecutionEvents(ctx, req, stream)
		}
		if strings.Contains(strings.ToLower(err.Error()), "not found") {
			return status.Error(codes.NotFound, "execution not found")
		}
		return status.Error(codes.Internal, "failed to load execution")
	}

	return s.streamPersistedExecutionEvents(ctx, req, stream, executionUUID, execModel.WorkflowSource)
}

func (s *ServiceAPIServer) streamPersistedExecutionEvents(
	ctx context.Context,
	req *pb.StreamExecutionEventsRequest,
	stream pb.MBFlowServiceAPI_StreamExecutionEventsServer,
	executionUUID uuid.UUID,
	workflowSource string,
) error {
	lastSequence := req.GetAfterSequence()
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		events, err := s.getExecutionEventsSince(ctx, executionUUID, lastSequence)
		if err != nil {
			return status.Error(codes.Internal, "failed to load execution events")
		}

		for _, ev := range events {
			payload := map[string]any(ev.Payload)
			protoEvent := &pb.ExecutionEvent{
				EventId:        ev.ID.String(),
				ExecutionId:    req.GetExecutionId(),
				Sequence:       ev.Sequence,
				EventType:      ev.EventType,
				WorkflowSource: workflowSource,
				Payload:        mapToStruct(payload),
				SentAt:         timestamppb.New(ev.CreatedAt),
			}

			if err := stream.Send(protoEvent); err != nil {
				return err
			}

			lastSequence = ev.Sequence

			if isTerminalEventType(ev.EventType) {
				return nil
			}
		}

		currentExecution, err := s.ops.ExecutionRepo.FindByID(ctx, executionUUID)
		if err != nil {
			return status.Error(codes.Internal, "failed to load execution status")
		}
		workflowSource = currentExecution.WorkflowSource
		if isTerminalExecutionStatus(currentExecution.Status) {
			return nil
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
	}
}

func (s *ServiceAPIServer) streamEphemeralExecutionEvents(
	ctx context.Context,
	req *pb.StreamExecutionEventsRequest,
	stream pb.MBFlowServiceAPI_StreamExecutionEventsServer,
) error {
	if s.ops.ExecutionMgr == nil || s.ops.ExecutionMgr.ObserverManager() == nil {
		return status.Error(codes.Unavailable, "execution stream is not available")
	}

	observer := newStreamExecutionObserver(req.GetExecutionId())
	manager := s.ops.ExecutionMgr.ObserverManager()
	if err := manager.Register(observer); err != nil {
		return status.Error(codes.Internal, "failed to register execution stream observer")
	}
	defer manager.Unregister(observer.Name())

	lastSequence := req.GetAfterSequence()
	terminalTicker := time.NewTicker(500 * time.Millisecond)
	defer terminalTicker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-terminalTicker.C:
			if s.ops.ExecutionMgr.IsEphemeralExecutionTerminal(req.GetExecutionId()) {
				return nil
			}
		case event := <-observer.events:
			sequence := extractEventSequence(event, lastSequence)
			if sequence <= lastSequence {
				continue
			}

			protoEvent := observerEventToProto(req.GetExecutionId(), "inline", event, sequence)
			if err := stream.Send(protoEvent); err != nil {
				return err
			}

			lastSequence = sequence
			if isTerminalEventType(protoEvent.EventType) {
				return nil
			}
		}
	}
}

type executionEventsSinceReader interface {
	GetEventsSince(ctx context.Context, executionID uuid.UUID, afterSequence int64) ([]*storagemodels.EventModel, error)
}

func (s *ServiceAPIServer) getExecutionEventsSince(ctx context.Context, executionID uuid.UUID, afterSequence int64) ([]*storagemodels.EventModel, error) {
	if repo, ok := s.ops.ExecutionRepo.(executionEventsSinceReader); ok {
		return repo.GetEventsSince(ctx, executionID, afterSequence)
	}

	events, err := s.ops.ExecutionRepo.GetEvents(ctx, executionID)
	if err != nil {
		return nil, err
	}

	if afterSequence <= 0 {
		return events, nil
	}

	filtered := make([]*storagemodels.EventModel, 0, len(events))
	for _, ev := range events {
		if ev.Sequence > afterSequence {
			filtered = append(filtered, ev)
		}
	}
	return filtered, nil
}

type streamExecutionObserver struct {
	name   string
	filter appobserver.EventFilter
	events chan appobserver.Event
}

func newStreamExecutionObserver(executionID string) *streamExecutionObserver {
	return &streamExecutionObserver{
		name:   "grpc-stream-" + executionID + "-" + uuid.NewString(),
		filter: appobserver.NewExecutionIDFilter(executionID),
		events: make(chan appobserver.Event, 1000),
	}
}

func (o *streamExecutionObserver) Name() string {
	return o.name
}

func (o *streamExecutionObserver) Filter() appobserver.EventFilter {
	return o.filter
}

func (o *streamExecutionObserver) OnEvent(_ context.Context, event appobserver.Event) error {
	select {
	case o.events <- event:
	default:
		// Non-blocking delivery for stream sink: drop oldest behavior is handled at observer layer.
	}
	return nil
}

func observerEventToProto(executionID, workflowSource string, event appobserver.Event, sequence int64) *pb.ExecutionEvent {
	payload := map[string]any{
		"workflow_id": event.WorkflowID,
		"status":      event.Status,
	}
	if event.NodeID != nil {
		payload["node_id"] = *event.NodeID
	}
	if event.NodeName != nil {
		payload["node_name"] = *event.NodeName
	}
	if event.NodeType != nil {
		payload["node_type"] = *event.NodeType
	}
	if event.WaveIndex != nil {
		payload["wave_index"] = *event.WaveIndex
	}
	if event.NodeCount != nil {
		payload["node_count"] = *event.NodeCount
	}
	if event.DurationMs != nil {
		payload["duration_ms"] = *event.DurationMs
	}
	if event.Error != nil {
		payload["error"] = event.Error.Error()
	}
	if event.Input != nil {
		payload["input"] = event.Input
	}
	if event.Output != nil {
		payload["output"] = event.Output
	}
	if event.Variables != nil {
		payload["variables"] = event.Variables
	}
	if event.Metadata != nil {
		payload["metadata"] = event.Metadata
	}

	return &pb.ExecutionEvent{
		EventId:        uuid.NewString(),
		ExecutionId:    executionID,
		Sequence:       sequence,
		EventType:      string(event.Type),
		WorkflowSource: workflowSource,
		Payload:        mapToStruct(payload),
		SentAt:         timestamppb.New(event.Timestamp),
	}
}

func extractEventSequence(event appobserver.Event, fallback int64) int64 {
	if event.Metadata != nil {
		if raw, ok := event.Metadata["sequence"]; ok {
			switch v := raw.(type) {
			case int64:
				return v
			case int:
				return int64(v)
			case float64:
				return int64(v)
			case float32:
				return int64(v)
			}
		}
	}
	return fallback + 1
}

func isTerminalEventType(eventType string) bool {
	switch eventType {
	case "execution.completed", "execution.failed", "execution.cancelled", "execution.timeout":
		return true
	default:
		return false
	}
}

func isTerminalExecutionStatus(status string) bool {
	switch strings.ToLower(status) {
	case string(models.ExecutionStatusCompleted), string(models.ExecutionStatusFailed), string(models.ExecutionStatusCancelled), string(models.ExecutionStatusTimeout):
		return true
	default:
		return false
	}
}

func workflowFromProto(pw *pb.Workflow) *models.Workflow {
	if pw == nil {
		return nil
	}

	wf := &models.Workflow{
		ID:          pw.Id,
		Name:        pw.Name,
		Description: pw.Description,
		Version:     int(pw.Version),
		Status:      models.WorkflowStatus(pw.Status),
		CreatedBy:   pw.CreatedBy,
		Variables:   structToMap(pw.Variables),
		Metadata:    structToMap(pw.Metadata),
	}

	if pw.CreatedAt != nil {
		wf.CreatedAt = pw.CreatedAt.AsTime()
	}
	if pw.UpdatedAt != nil {
		wf.UpdatedAt = pw.UpdatedAt.AsTime()
	}

	for _, pn := range pw.Nodes {
		wf.Nodes = append(wf.Nodes, nodeFromProto(pn))
	}
	for _, pe := range pw.Edges {
		wf.Edges = append(wf.Edges, edgeFromProto(pe))
	}

	return wf
}

func nodeFromProto(pn *pb.Node) *models.Node {
	if pn == nil {
		return nil
	}

	n := &models.Node{
		ID:     pn.Id,
		Name:   pn.Name,
		Type:   pn.Type,
		Config: structToMap(pn.Config),
	}

	if pn.Position != nil {
		pos := structToMap(pn.Position)
		if x, ok := pos["x"].(float64); ok {
			if y, ok := pos["y"].(float64); ok {
				n.Position = &models.Position{X: x, Y: y}
			}
		}
	}

	return n
}

func edgeFromProto(pe *pb.Edge) *models.Edge {
	if pe == nil {
		return nil
	}

	e := &models.Edge{
		ID:           pe.Id,
		From:         pe.From,
		To:           pe.To,
		SourceHandle: pe.SourceHandle,
	}

	if pe.Condition != nil {
		condMap := structToMap(pe.Condition)
		if expr, ok := condMap["expression"].(string); ok {
			e.Condition = expr
		}
	}

	if pe.Loop != nil {
		e.Loop = &models.LoopConfig{MaxIterations: int(pe.Loop.MaxIterations)}
	}

	return e
}
