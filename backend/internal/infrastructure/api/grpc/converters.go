package grpc

import (
	"time"

	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"

	pb "github.com/smilemakc/mbflow/api/proto/serviceapipb"
	"github.com/smilemakc/mbflow/internal/application/serviceapi"
	"github.com/smilemakc/mbflow/pkg/models"
)

func toProtoWorkflow(w *models.Workflow) *pb.Workflow {
	if w == nil {
		return nil
	}

	pw := &pb.Workflow{
		Id:          w.ID,
		Name:        w.Name,
		Description: w.Description,
		Status:      string(w.Status),
		Version:     int32(w.Version),
		CreatedBy:   w.CreatedBy,
		CreatedAt:   timestamppb.New(w.CreatedAt),
		UpdatedAt:   timestamppb.New(w.UpdatedAt),
	}

	pw.Variables = mapToStruct(w.Variables)
	pw.Metadata = mapToStruct(w.Metadata)

	if w.Nodes != nil {
		pw.Nodes = make([]*pb.Node, len(w.Nodes))
		for i, n := range w.Nodes {
			pw.Nodes[i] = toProtoNode(n)
		}
	}

	if w.Edges != nil {
		pw.Edges = make([]*pb.Edge, len(w.Edges))
		for i, e := range w.Edges {
			pw.Edges[i] = toProtoEdge(e)
		}
	}

	return pw
}

func toProtoWorkflows(workflows []*models.Workflow) []*pb.Workflow {
	result := make([]*pb.Workflow, len(workflows))
	for i, w := range workflows {
		result[i] = toProtoWorkflow(w)
	}
	return result
}

func toProtoNode(n *models.Node) *pb.Node {
	if n == nil {
		return nil
	}
	pn := &pb.Node{
		Id:   n.ID,
		Name: n.Name,
		Type: n.Type,
	}
	pn.Config = mapToStruct(n.Config)
	if n.Position != nil {
		pn.Position, _ = structpb.NewStruct(map[string]any{
			"x": n.Position.X,
			"y": n.Position.Y,
		})
	}
	return pn
}

func toProtoEdge(e *models.Edge) *pb.Edge {
	if e == nil {
		return nil
	}
	pe := &pb.Edge{
		Id:           e.ID,
		From:         e.From,
		To:           e.To,
		SourceHandle: e.SourceHandle,
	}
	if e.Condition != "" {
		pe.Condition, _ = structpb.NewStruct(map[string]any{"expression": e.Condition})
	}
	if e.Loop != nil {
		pe.Loop = &pb.EdgeLoopConfig{MaxIterations: int32(e.Loop.MaxIterations)}
	}
	return pe
}

func toProtoExecution(ex *models.Execution) *pb.Execution {
	if ex == nil {
		return nil
	}

	pe := &pb.Execution{
		Id:         ex.ID,
		WorkflowId: ex.WorkflowID,
		Status:     string(ex.Status),
		Error:      ex.Error,
		StartedAt:  timestamppb.New(ex.StartedAt),
		DurationMs: ex.Duration,
	}

	pe.Input = mapToStruct(ex.Input)
	pe.Output = mapToStruct(ex.Output)

	if ex.CompletedAt != nil {
		pe.CompletedAt = timestamppb.New(*ex.CompletedAt)
	}
	pe.CreatedAt = timestamppb.New(ex.StartedAt)

	if ex.NodeExecutions != nil {
		pe.NodeExecutions = make([]*pb.NodeExecution, len(ex.NodeExecutions))
		for i, ne := range ex.NodeExecutions {
			pe.NodeExecutions[i] = toProtoNodeExecution(ne)
		}
	}

	return pe
}

func toProtoExecutions(executions []*models.Execution) []*pb.Execution {
	result := make([]*pb.Execution, len(executions))
	for i, ex := range executions {
		result[i] = toProtoExecution(ex)
	}
	return result
}

func toProtoNodeExecution(ne *models.NodeExecution) *pb.NodeExecution {
	if ne == nil {
		return nil
	}

	pne := &pb.NodeExecution{
		Id:         ne.ID,
		NodeId:     ne.NodeID,
		NodeName:   ne.NodeName,
		NodeType:   ne.NodeType,
		Status:     string(ne.Status),
		Error:      ne.Error,
		StartedAt:  timestamppb.New(ne.StartedAt),
		DurationMs: ne.Duration,
	}

	pne.Input = mapToStruct(ne.Input)
	pne.Output = mapToStruct(ne.Output)

	if ne.CompletedAt != nil {
		pne.CompletedAt = timestamppb.New(*ne.CompletedAt)
	}

	return pne
}

func toProtoTrigger(t *models.Trigger) *pb.Trigger {
	if t == nil {
		return nil
	}

	pt := &pb.Trigger{
		Id:          t.ID,
		WorkflowId:  t.WorkflowID,
		Name:        t.Name,
		Description: t.Description,
		Type:        string(t.Type),
		Enabled:     t.Enabled,
		CreatedAt:   timestamppb.New(t.CreatedAt),
		UpdatedAt:   timestamppb.New(t.UpdatedAt),
	}

	pt.Config = mapToStruct(t.Config)

	if t.LastRun != nil {
		pt.LastRun = timestamppb.New(*t.LastRun)
	}

	return pt
}

func toProtoTriggers(triggers []*models.Trigger) []*pb.Trigger {
	result := make([]*pb.Trigger, len(triggers))
	for i, t := range triggers {
		result[i] = toProtoTrigger(t)
	}
	return result
}

func toProtoCredential(c *serviceapi.CredentialInfo) *pb.Credential {
	if c == nil {
		return nil
	}

	pc := &pb.Credential{
		Id:             c.ID,
		Name:           c.Name,
		Description:    c.Description,
		Status:         c.Status,
		CredentialType: c.CredentialType,
		Provider:       c.Provider,
		UsageCount:     c.UsageCount,
		CreatedAt:      timestamppb.New(c.CreatedAt),
		UpdatedAt:      timestamppb.New(c.UpdatedAt),
		Fields:         c.Fields,
	}

	if c.ExpiresAt != nil {
		pc.ExpiresAt = timestamppb.New(*c.ExpiresAt)
	}
	if c.LastUsedAt != nil {
		pc.LastUsedAt = timestamppb.New(*c.LastUsedAt)
	}

	return pc
}

func toProtoCredentials(creds []*serviceapi.CredentialInfo) []*pb.Credential {
	result := make([]*pb.Credential, len(creds))
	for i, c := range creds {
		result[i] = toProtoCredential(c)
	}
	return result
}

func toProtoAuditLogEntry(log *models.ServiceAuditLog) *pb.AuditLogEntry {
	if log == nil {
		return nil
	}

	entry := &pb.AuditLogEntry{
		Id:             log.ID,
		SystemKeyId:    log.SystemKeyID,
		ServiceName:    log.ServiceName,
		Action:         log.Action,
		ResourceType:   log.ResourceType,
		Method:         log.RequestMethod,
		Path:           log.RequestPath,
		ClientIp:       log.IPAddress,
		ResponseStatus: int32(log.ResponseStatus),
		CreatedAt:      timestamppb.New(log.CreatedAt),
	}

	if log.ResourceID != nil {
		entry.ResourceId = *log.ResourceID
	}
	if log.ImpersonatedUserID != nil {
		entry.ImpersonatedUserId = *log.ImpersonatedUserID
	}
	if log.RequestBody != nil {
		entry.RequestBody = *log.RequestBody
	}

	return entry
}

func toProtoAuditLogEntries(logs []*models.ServiceAuditLog) []*pb.AuditLogEntry {
	result := make([]*pb.AuditLogEntry, len(logs))
	for i, l := range logs {
		result[i] = toProtoAuditLogEntry(l)
	}
	return result
}

// mapToStruct safely converts map[string]any to protobuf Struct.
func mapToStruct(m map[string]any) *structpb.Struct {
	if m == nil {
		return nil
	}
	s, err := structpb.NewStruct(m)
	if err != nil {
		return nil
	}
	return s
}

// structToMap converts a protobuf Struct to map[string]any.
func structToMap(s *structpb.Struct) map[string]any {
	if s == nil {
		return nil
	}
	return s.AsMap()
}

// optionalTimestamp converts a protobuf Timestamp to *time.Time (nil-safe).
func optionalTimestamp(ts *timestamppb.Timestamp) *time.Time {
	if ts == nil {
		return nil
	}
	t := ts.AsTime()
	return &t
}
