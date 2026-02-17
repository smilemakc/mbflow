package grpcclient

import (
	"google.golang.org/protobuf/types/known/structpb"

	pb "github.com/smilemakc/mbflow/sdk/go/internal/pb"
	"github.com/smilemakc/mbflow/sdk/go/models"
)

// --- Proto → Models ---

func WorkflowFromProto(pw *pb.Workflow) *models.Workflow {
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
		Variables:   StructToMap(pw.Variables),
		Metadata:    StructToMap(pw.Metadata),
	}
	if pw.CreatedAt != nil {
		wf.CreatedAt = pw.CreatedAt.AsTime()
	}
	if pw.UpdatedAt != nil {
		wf.UpdatedAt = pw.UpdatedAt.AsTime()
	}
	for _, pn := range pw.Nodes {
		wf.Nodes = append(wf.Nodes, NodeFromProto(pn))
	}
	for _, pe := range pw.Edges {
		wf.Edges = append(wf.Edges, EdgeFromProto(pe))
	}
	return wf
}

func NodeFromProto(pn *pb.Node) *models.Node {
	if pn == nil {
		return nil
	}
	n := &models.Node{
		ID:     pn.Id,
		Name:   pn.Name,
		Type:   pn.Type,
		Config: StructToMap(pn.Config),
	}
	if pn.Position != nil {
		pos := StructToMap(pn.Position)
		if x, ok := pos["x"].(float64); ok {
			if y, ok := pos["y"].(float64); ok {
				n.Position = &models.Position{X: x, Y: y}
			}
		}
	}
	return n
}

func EdgeFromProto(pe *pb.Edge) *models.Edge {
	if pe == nil {
		return nil
	}
	e := &models.Edge{
		ID:   pe.Id,
		From: pe.From,
		To:   pe.To,
	}
	if pe.Condition != nil {
		condMap := StructToMap(pe.Condition)
		if expr, ok := condMap["expression"].(string); ok {
			e.Condition = expr
		}
	}
	return e
}

func ExecutionFromProto(pe *pb.Execution) *models.Execution {
	if pe == nil {
		return nil
	}
	exec := &models.Execution{
		ID:         pe.Id,
		WorkflowID: pe.WorkflowId,
		Status:     models.ExecutionStatus(pe.Status),
		Error:      pe.Error,
		Input:      StructToMap(pe.Input),
		Output:     StructToMap(pe.Output),
		Duration:   pe.DurationMs,
	}
	if pe.StartedAt != nil {
		exec.StartedAt = pe.StartedAt.AsTime()
	}
	if pe.CompletedAt != nil {
		t := pe.CompletedAt.AsTime()
		exec.CompletedAt = &t
	}
	for _, pne := range pe.NodeExecutions {
		exec.NodeExecutions = append(exec.NodeExecutions, NodeExecutionFromProto(pne))
	}
	return exec
}

func NodeExecutionFromProto(pne *pb.NodeExecution) *models.NodeExecution {
	if pne == nil {
		return nil
	}
	ne := &models.NodeExecution{
		ID:       pne.Id,
		NodeID:   pne.NodeId,
		NodeName: pne.NodeName,
		NodeType: pne.NodeType,
		Status:   models.NodeExecutionStatus(pne.Status),
		Error:    pne.Error,
		Input:    StructToMap(pne.Input),
		Output:   StructToMap(pne.Output),
		Duration: pne.DurationMs,
	}
	if pne.StartedAt != nil {
		ne.StartedAt = pne.StartedAt.AsTime()
	}
	if pne.CompletedAt != nil {
		t := pne.CompletedAt.AsTime()
		ne.CompletedAt = &t
	}
	return ne
}

func TriggerFromProto(pt *pb.Trigger) *models.Trigger {
	if pt == nil {
		return nil
	}
	tr := &models.Trigger{
		ID:          pt.Id,
		WorkflowID:  pt.WorkflowId,
		Name:        pt.Name,
		Description: pt.Description,
		Type:        models.TriggerType(pt.Type),
		Config:      StructToMap(pt.Config),
		Enabled:     pt.Enabled,
	}
	if pt.CreatedAt != nil {
		tr.CreatedAt = pt.CreatedAt.AsTime()
	}
	if pt.UpdatedAt != nil {
		tr.UpdatedAt = pt.UpdatedAt.AsTime()
	}
	if pt.LastRun != nil {
		t := pt.LastRun.AsTime()
		tr.LastRun = &t
	}
	return tr
}

func CredentialFromProto(pc *pb.Credential) *models.Credential {
	if pc == nil {
		return nil
	}
	return &models.Credential{
		ID:          pc.Id,
		Name:        pc.Name,
		Type:        pc.CredentialType,
		Description: pc.Description,
	}
}

// --- Models → Proto ---

func WorkflowToProto(wf *models.Workflow) *pb.Workflow {
	pw := &pb.Workflow{
		Name:        wf.Name,
		Description: wf.Description,
		Status:      string(wf.Status),
		Variables:   MapToStruct(wf.Variables),
		Metadata:    MapToStruct(wf.Metadata),
	}
	for _, n := range wf.Nodes {
		pw.Nodes = append(pw.Nodes, NodeToProto(n))
	}
	for _, e := range wf.Edges {
		pw.Edges = append(pw.Edges, EdgeToProto(e))
	}
	return pw
}

func NodeToProto(n *models.Node) *pb.Node {
	pn := &pb.Node{
		Id:     n.ID,
		Name:   n.Name,
		Type:   n.Type,
		Config: MapToStruct(n.Config),
	}
	if n.Position != nil {
		pn.Position = MapToStruct(map[string]any{
			"x": n.Position.X,
			"y": n.Position.Y,
		})
	}
	return pn
}

func EdgeToProto(e *models.Edge) *pb.Edge {
	pe := &pb.Edge{
		Id:   e.ID,
		From: e.From,
		To:   e.To,
	}
	if e.Condition != "" {
		pe.Condition = MapToStruct(map[string]any{
			"expression": e.Condition,
		})
	}
	return pe
}

// --- Helpers ---

func StructToMap(s *structpb.Struct) map[string]any {
	if s == nil {
		return nil
	}
	return s.AsMap()
}

func MapToStruct(m map[string]any) *structpb.Struct {
	if m == nil {
		return nil
	}
	s, _ := structpb.NewStruct(m)
	return s
}
