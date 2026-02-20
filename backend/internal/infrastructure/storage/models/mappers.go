package models

import (
	"time"

	"github.com/google/uuid"
	pkgmodels "github.com/smilemakc/mbflow/pkg/models"
)

// ============================================================================
// Event Mappers
// ============================================================================

// EventModelToDomain converts a storage EventModel to a domain Event
func EventModelToDomain(em *EventModel) *pkgmodels.Event {
	if em == nil {
		return nil
	}

	var payload map[string]any
	if em.Payload != nil {
		payload = map[string]any(em.Payload)
	}

	return &pkgmodels.Event{
		ID:          em.ID.String(),
		ExecutionID: em.ExecutionID.String(),
		EventType:   em.EventType,
		Sequence:    em.Sequence,
		Payload:     payload,
		CreatedAt:   em.CreatedAt,
	}
}

// EventDomainToModel converts a domain Event to a storage EventModel
func EventDomainToModel(e *pkgmodels.Event) *EventModel {
	if e == nil {
		return nil
	}

	em := &EventModel{
		EventType: e.EventType,
		Sequence:  e.Sequence,
		Payload:   JSONBMap(e.Payload),
		CreatedAt: e.CreatedAt,
	}

	if e.ID != "" {
		if id, err := uuid.Parse(e.ID); err == nil {
			em.ID = id
		}
	}

	if e.ExecutionID != "" {
		if execID, err := uuid.Parse(e.ExecutionID); err == nil {
			em.ExecutionID = execID
		}
	}

	return em
}

// EventModelsToDomain converts a slice of storage EventModels to domain Events
func EventModelsToDomain(models []*EventModel) []*pkgmodels.Event {
	if models == nil {
		return nil
	}

	result := make([]*pkgmodels.Event, len(models))
	for i, m := range models {
		result[i] = EventModelToDomain(m)
	}
	return result
}

// ============================================================================
// Trigger Mappers
// ============================================================================

// TriggerModelToDomain converts a storage TriggerModel to a domain Trigger
func TriggerModelToDomain(tm *TriggerModel) *pkgmodels.Trigger {
	if tm == nil {
		return nil
	}

	var config map[string]any
	if tm.Config != nil {
		config = map[string]any(tm.Config)
	}

	trigger := &pkgmodels.Trigger{
		ID:         tm.ID.String(),
		WorkflowID: tm.WorkflowID.String(),
		Name:       "", // TriggerModel doesn't have Name field
		Type:       pkgmodels.TriggerType(tm.Type),
		Config:     config,
		Enabled:    tm.Enabled,
		CreatedAt:  tm.CreatedAt,
		UpdatedAt:  tm.UpdatedAt,
		LastRun:    tm.LastTriggeredAt,
	}

	return trigger
}

// TriggerDomainToModel converts a domain Trigger to a storage TriggerModel
func TriggerDomainToModel(t *pkgmodels.Trigger) *TriggerModel {
	if t == nil {
		return nil
	}

	tm := &TriggerModel{
		Type:            string(t.Type),
		Config:          JSONBMap(t.Config),
		Enabled:         t.Enabled,
		LastTriggeredAt: t.LastRun,
		CreatedAt:       t.CreatedAt,
		UpdatedAt:       t.UpdatedAt,
	}

	if t.ID != "" {
		if id, err := uuid.Parse(t.ID); err == nil {
			tm.ID = id
		}
	}

	if t.WorkflowID != "" {
		if wfID, err := uuid.Parse(t.WorkflowID); err == nil {
			tm.WorkflowID = wfID
		}
	}

	return tm
}

// TriggerModelsToDomain converts a slice of storage TriggerModels to domain Triggers
func TriggerModelsToDomain(models []*TriggerModel) []*pkgmodels.Trigger {
	if models == nil {
		return nil
	}

	result := make([]*pkgmodels.Trigger, len(models))
	for i, m := range models {
		result[i] = TriggerModelToDomain(m)
	}
	return result
}

// ============================================================================
// AuditLog Mappers
// ============================================================================

// AuditLogModelToDomain converts a storage AuditLogModel to a domain AuditLog
func AuditLogModelToDomain(am *AuditLogModel) *pkgmodels.AuditLog {
	if am == nil {
		return nil
	}

	var userID *string
	if am.UserID != nil {
		uid := am.UserID.String()
		userID = &uid
	}

	var resourceID *string
	if am.ResourceID != nil {
		rid := am.ResourceID.String()
		resourceID = &rid
	}

	var metadata map[string]any
	if am.Metadata != nil {
		metadata = map[string]any(am.Metadata)
	}

	return &pkgmodels.AuditLog{
		ID:           am.ID.String(),
		UserID:       userID,
		Action:       am.Action,
		ResourceType: am.ResourceType,
		ResourceID:   resourceID,
		IPAddress:    am.IPAddress,
		UserAgent:    am.UserAgent,
		Metadata:     metadata,
		CreatedAt:    am.CreatedAt,
	}
}

// AuditLogDomainToModel converts a domain AuditLog to a storage AuditLogModel
func AuditLogDomainToModel(a *pkgmodels.AuditLog) *AuditLogModel {
	if a == nil {
		return nil
	}

	am := &AuditLogModel{
		Action:       a.Action,
		ResourceType: a.ResourceType,
		IPAddress:    a.IPAddress,
		UserAgent:    a.UserAgent,
		Metadata:     JSONBMap(a.Metadata),
		CreatedAt:    a.CreatedAt,
	}

	if a.ID != "" {
		if id, err := uuid.Parse(a.ID); err == nil {
			am.ID = id
		}
	}

	if a.UserID != nil && *a.UserID != "" {
		if userID, err := uuid.Parse(*a.UserID); err == nil {
			am.UserID = &userID
		}
	}

	if a.ResourceID != nil && *a.ResourceID != "" {
		if resourceID, err := uuid.Parse(*a.ResourceID); err == nil {
			am.ResourceID = &resourceID
		}
	}

	return am
}

// AuditLogModelsToDomain converts a slice of storage AuditLogModels to domain AuditLogs
func AuditLogModelsToDomain(models []*AuditLogModel) []*pkgmodels.AuditLog {
	if models == nil {
		return nil
	}

	result := make([]*pkgmodels.AuditLog, len(models))
	for i, m := range models {
		result[i] = AuditLogModelToDomain(m)
	}
	return result
}

// parseJSONInt converts a JSON numeric value to int.
// Handles float64 (the standard JSON unmarshal type) and int (in-memory map values).
// Returns 0 if the value is nil or not a recognized numeric type.
func parseJSONInt(v any) int {
	switch n := v.(type) {
	case float64:
		return int(n)
	case int:
		return n
	case int64:
		return int(n)
	}
	return 0
}

// WorkflowToStorage converts a domain workflow to a storage workflow model
// This is used for both Create and Update operations
func WorkflowToStorage(w *pkgmodels.Workflow, workflowID uuid.UUID) *WorkflowModel {
	storageNodes := make([]*NodeModel, len(w.Nodes))
	for i, node := range w.Nodes {
		storageNodes[i] = NodeToStorage(node, workflowID)
	}

	storageEdges := make([]*EdgeModel, len(w.Edges))
	for i, edge := range w.Edges {
		storageEdges[i] = EdgeToStorage(edge, workflowID)
	}

	// Store tags in metadata if present
	metadata := JSONBMap(w.Metadata)
	if metadata == nil {
		metadata = make(JSONBMap)
	}
	if len(w.Tags) > 0 {
		metadata["tags"] = w.Tags
	}

	return &WorkflowModel{
		ID:          workflowID,
		Name:        w.Name,
		Description: w.Description,
		Version:     w.Version,
		Status:      string(w.Status),
		Variables:   JSONBMap(w.Variables),
		Metadata:    metadata,
		Nodes:       storageNodes,
		Edges:       storageEdges,
	}
}

// NodeToStorage converts a domain node to a storage node model
func NodeToStorage(n *pkgmodels.Node, workflowID uuid.UUID) *NodeModel {
	position := JSONBMap{}
	if n.Position != nil {
		position["x"] = n.Position.X
		position["y"] = n.Position.Y
	}

	return &NodeModel{
		// ID will be set by Repository (preserved on update, new on create)
		NodeID:     n.ID,
		WorkflowID: workflowID,
		Name:       n.Name,
		Type:       n.Type,
		Config:     JSONBMap(n.Config),
		Position:   position,
	}
}

// EdgeToStorage converts a domain edge to a storage edge model
func EdgeToStorage(e *pkgmodels.Edge, workflowID uuid.UUID) *EdgeModel {
	var condition JSONBMap
	if e.Condition != "" {
		condition = JSONBMap{"expression": e.Condition}
	}

	var loop JSONBMap
	if e.Loop != nil {
		loop = JSONBMap{"max_iterations": e.Loop.MaxIterations}
	}

	return &EdgeModel{
		// ID will be set by Repository (preserved on update, new on create)
		EdgeID:       e.ID,
		WorkflowID:   workflowID,
		FromNodeID:   e.From,
		ToNodeID:     e.To,
		SourceHandle: e.SourceHandle,
		Condition:    condition,
		Loop:         loop,
	}
}

// WorkflowFromStorage converts a storage workflow model to a domain workflow
func WorkflowFromStorage(sw *WorkflowModel) *pkgmodels.Workflow {
	nodes := make([]*pkgmodels.Node, len(sw.Nodes))
	for i, node := range sw.Nodes {
		nodes[i] = NodeFromStorage(node)
	}

	edges := make([]*pkgmodels.Edge, len(sw.Edges))
	for i, edge := range sw.Edges {
		edges[i] = EdgeFromStorage(edge)
	}

	var variables map[string]any
	if sw.Variables != nil {
		variables = map[string]any(sw.Variables)
	}

	var metadata map[string]any
	if sw.Metadata != nil {
		metadata = map[string]any(sw.Metadata)
	}

	// Extract tags from metadata if present
	var tags []string
	if metadata != nil {
		// Try both []string and []any for compatibility
		if tagsVal, ok := metadata["tags"].([]string); ok {
			tags = tagsVal
		} else if tagsVal, ok := metadata["tags"].([]any); ok {
			tags = make([]string, len(tagsVal))
			for i, t := range tagsVal {
				if tagStr, ok := t.(string); ok {
					tags[i] = tagStr
				}
			}
		}
	}

	return &pkgmodels.Workflow{
		ID:          sw.ID.String(),
		Name:        sw.Name,
		Description: sw.Description,
		Version:     sw.Version,
		Status:      pkgmodels.WorkflowStatus(sw.Status),
		Tags:        tags,
		Nodes:       nodes,
		Edges:       edges,
		Resources:   WorkflowResourcesFromStorage(sw.Resources),
		Variables:   variables,
		Metadata:    metadata,
		CreatedAt:   sw.CreatedAt,
		UpdatedAt:   sw.UpdatedAt,
	}
}

// NodeFromStorage converts a storage node model to a domain node
func NodeFromStorage(sn *NodeModel) *pkgmodels.Node {
	var position *pkgmodels.Position
	if sn.Position != nil {
		x, _ := sn.Position["x"].(float64)
		y, _ := sn.Position["y"].(float64)
		position = &pkgmodels.Position{X: x, Y: y}
	}

	var config map[string]any
	if sn.Config != nil {
		config = map[string]any(sn.Config)
	}

	var metadata map[string]any
	// NodeModel doesn't have metadata yet, but we're ready for it

	return &pkgmodels.Node{
		ID:          sn.NodeID, // Use logical ID
		Name:        sn.Name,
		Type:        sn.Type,
		Description: "", // NodeModel doesn't have description yet
		Config:      config,
		Position:    position,
		Metadata:    metadata,
	}
}

// EdgeFromStorage converts a storage edge model to a domain edge
func EdgeFromStorage(se *EdgeModel) *pkgmodels.Edge {
	var condition string
	if se.Condition != nil {
		if expr, ok := se.Condition["expression"].(string); ok {
			condition = expr
		}
	}

	var loop *pkgmodels.LoopConfig
	if se.Loop != nil {
		if maxIter := parseJSONInt(se.Loop["max_iterations"]); maxIter > 0 {
			loop = &pkgmodels.LoopConfig{MaxIterations: maxIter}
		}
	}

	return &pkgmodels.Edge{
		ID:           se.EdgeID,
		From:         se.FromNodeID,
		To:           se.ToNodeID,
		SourceHandle: se.SourceHandle,
		Condition:    condition,
		Loop:         loop,
	}
}

// WorkflowResourceToStorage converts domain WorkflowResource to storage model
func WorkflowResourceToStorage(domain *pkgmodels.WorkflowResource, workflowID uuid.UUID) *WorkflowResourceModel {
	resourceID, _ := uuid.Parse(domain.ResourceID)
	return &WorkflowResourceModel{
		WorkflowID: workflowID,
		ResourceID: resourceID,
		Alias:      domain.Alias,
		AccessType: domain.AccessType,
		AssignedAt: time.Now(),
	}
}

// WorkflowResourceFromStorage converts storage model to domain WorkflowResource
func WorkflowResourceFromStorage(storage *WorkflowResourceModel) *pkgmodels.WorkflowResource {
	return &pkgmodels.WorkflowResource{
		ResourceID: storage.ResourceID.String(),
		Alias:      storage.Alias,
		AccessType: storage.AccessType,
	}
}

// WorkflowResourcesToStorage converts slice of domain WorkflowResource to storage models
func WorkflowResourcesToStorage(domains []pkgmodels.WorkflowResource, workflowID uuid.UUID) []*WorkflowResourceModel {
	result := make([]*WorkflowResourceModel, len(domains))
	for i, d := range domains {
		result[i] = WorkflowResourceToStorage(&d, workflowID)
	}
	return result
}

// WorkflowResourcesFromStorage converts slice of storage models to domain WorkflowResources
func WorkflowResourcesFromStorage(storage []*WorkflowResourceModel) []pkgmodels.WorkflowResource {
	result := make([]pkgmodels.WorkflowResource, len(storage))
	for i, s := range storage {
		result[i] = *WorkflowResourceFromStorage(s)
	}
	return result
}

// WorkflowModelToDomain converts storage WorkflowModel to domain Workflow
func WorkflowModelToDomain(wm *WorkflowModel) *pkgmodels.Workflow {
	if wm == nil {
		return nil
	}

	workflow := &pkgmodels.Workflow{
		ID:          wm.ID.String(),
		Name:        wm.Name,
		Description: wm.Description,
		Status:      pkgmodels.WorkflowStatus(wm.Status),
		Variables:   make(map[string]any),
		Metadata:    make(map[string]any),
		CreatedAt:   wm.CreatedAt,
		UpdatedAt:   wm.UpdatedAt,
	}

	if wm.CreatedBy != nil {
		workflow.CreatedBy = wm.CreatedBy.String()
	}

	if wm.Variables != nil {
		workflow.Variables = map[string]any(wm.Variables)
	}

	if wm.Metadata != nil {
		workflow.Metadata = map[string]any(wm.Metadata)
	}

	workflow.Nodes = make([]*pkgmodels.Node, 0, len(wm.Nodes))
	for _, nm := range wm.Nodes {
		workflow.Nodes = append(workflow.Nodes, NodeModelToDomain(nm))
	}

	workflow.Edges = make([]*pkgmodels.Edge, 0, len(wm.Edges))
	for _, em := range wm.Edges {
		workflow.Edges = append(workflow.Edges, EdgeModelToDomain(em))
	}

	workflow.Resources = make([]pkgmodels.WorkflowResource, 0, len(wm.Resources))
	for _, rm := range wm.Resources {
		wr := pkgmodels.WorkflowResource{
			ResourceID: rm.ResourceID.String(),
			Alias:      rm.Alias,
			AccessType: rm.AccessType,
		}
		if rm.Resource != nil {
			wr.ResourceName = rm.Resource.Name
			wr.ResourceType = rm.Resource.Type
		}
		workflow.Resources = append(workflow.Resources, wr)
	}

	return workflow
}

// NodeModelToDomain converts storage NodeModel to domain Node
func NodeModelToDomain(nm *NodeModel) *pkgmodels.Node {
	if nm == nil {
		return nil
	}

	node := &pkgmodels.Node{
		ID:     nm.NodeID,
		Name:   nm.Name,
		Type:   nm.Type,
		Config: make(map[string]any),
	}

	if nm.Config != nil {
		node.Config = map[string]any(nm.Config)
	}

	if nm.Position != nil {
		posMap := map[string]any(nm.Position)
		if x, ok := posMap["x"].(float64); ok {
			if y, ok := posMap["y"].(float64); ok {
				node.Position = &pkgmodels.Position{X: x, Y: y}
			}
		}
	}

	return node
}

// EdgeModelToDomain converts storage EdgeModel to domain Edge
func EdgeModelToDomain(em *EdgeModel) *pkgmodels.Edge {
	if em == nil {
		return nil
	}

	edge := &pkgmodels.Edge{
		ID:           em.EdgeID,
		From:         em.FromNodeID,
		To:           em.ToNodeID,
		SourceHandle: em.SourceHandle,
	}

	if em.Condition != nil {
		if expr, ok := em.Condition["expression"].(string); ok {
			edge.Condition = expr
		}
	}

	if em.Loop != nil {
		if maxIter := parseJSONInt(em.Loop["max_iterations"]); maxIter > 0 {
			edge.Loop = &pkgmodels.LoopConfig{MaxIterations: maxIter}
		}
	}

	return edge
}

// ExecutionModelToDomain converts storage ExecutionModel to domain Execution
func ExecutionModelToDomain(exm *ExecutionModel) *pkgmodels.Execution {
	if exm == nil {
		return nil
	}

	exec := &pkgmodels.Execution{
		ID:         exm.ID.String(),
		WorkflowID: exm.WorkflowID.String(),
		Status:     pkgmodels.ExecutionStatus(exm.Status),
		Input:      make(map[string]any),
		Output:     make(map[string]any),
		Variables:  make(map[string]any),
	}

	if exm.StartedAt != nil {
		exec.StartedAt = *exm.StartedAt
	}

	if exm.InputData != nil {
		exec.Input = exm.InputData
	}

	if exm.OutputData != nil {
		exec.Output = exm.OutputData
	}

	if exm.Variables != nil {
		exec.Variables = exm.Variables
	}

	if exm.CompletedAt != nil {
		exec.CompletedAt = exm.CompletedAt
	}

	if exm.Error != "" {
		exec.Error = exm.Error
	}

	if len(exm.NodeExecutions) > 0 {
		exec.NodeExecutions = make([]*pkgmodels.NodeExecution, len(exm.NodeExecutions))
		for i, ne := range exm.NodeExecutions {
			exec.NodeExecutions[i] = NodeExecutionModelToDomain(ne)
		}
	}

	return exec
}

// ExecutionDomainToModel converts domain Execution to storage ExecutionModel
func ExecutionDomainToModel(exec *pkgmodels.Execution) *ExecutionModel {
	if exec == nil {
		return nil
	}

	exm := &ExecutionModel{
		Status:     string(exec.Status),
		InputData:  JSONBMap(exec.Input),
		OutputData: JSONBMap(exec.Output),
		Variables:  JSONBMap(exec.Variables),
		StartedAt:  &exec.StartedAt,
		Error:      exec.Error,
	}

	if exec.ID != "" {
		if id, err := uuid.Parse(exec.ID); err == nil {
			exm.ID = id
		}
	}

	if exec.WorkflowID != "" {
		if wfID, err := uuid.Parse(exec.WorkflowID); err == nil {
			exm.WorkflowID = wfID
		}
	}

	if exec.CompletedAt != nil {
		exm.CompletedAt = exec.CompletedAt
	}

	if len(exec.NodeExecutions) > 0 {
		exm.NodeExecutions = make([]*NodeExecutionModel, 0, len(exec.NodeExecutions))
		for _, ne := range exec.NodeExecutions {
			nem := NodeExecutionDomainToModel(ne)
			if nem != nil {
				exm.NodeExecutions = append(exm.NodeExecutions, nem)
			}
		}
	}

	return exm
}

// NodeExecutionModelToDomain converts storage NodeExecutionModel to domain NodeExecution
func NodeExecutionModelToDomain(nem *NodeExecutionModel) *pkgmodels.NodeExecution {
	if nem == nil {
		return nil
	}

	ne := &pkgmodels.NodeExecution{
		ID:             nem.ID.String(),
		ExecutionID:    nem.ExecutionID.String(),
		NodeID:         nem.NodeID.String(),
		Status:         pkgmodels.NodeExecutionStatus(nem.Status),
		Input:          make(map[string]any),
		Output:         make(map[string]any),
		Config:         make(map[string]any),
		ResolvedConfig: make(map[string]any),
		RetryCount:     nem.RetryCount,
	}

	if nem.InputData != nil {
		ne.Input = nem.InputData
	}

	if nem.OutputData != nil {
		ne.Output = nem.OutputData
	}

	if nem.Config != nil {
		ne.Config = nem.Config
	}

	if nem.ResolvedConfig != nil {
		ne.ResolvedConfig = nem.ResolvedConfig
	}

	if nem.StartedAt != nil {
		ne.StartedAt = *nem.StartedAt
	}

	if nem.CompletedAt != nil {
		ne.CompletedAt = nem.CompletedAt
	}

	if nem.Error != "" {
		ne.Error = nem.Error
	}

	return ne
}

// NodeExecutionDomainToModel converts domain NodeExecution to storage NodeExecutionModel
func NodeExecutionDomainToModel(ne *pkgmodels.NodeExecution) *NodeExecutionModel {
	if ne == nil {
		return nil
	}

	nem := &NodeExecutionModel{
		Status:         string(ne.Status),
		InputData:      JSONBMap(ne.Input),
		OutputData:     JSONBMap(ne.Output),
		Config:         JSONBMap(ne.Config),
		ResolvedConfig: JSONBMap(ne.ResolvedConfig),
		RetryCount:     ne.RetryCount,
		Error:          ne.Error,
	}

	if ne.ID != "" {
		if id, err := uuid.Parse(ne.ID); err == nil {
			nem.ID = id
		} else {
			nem.ID = uuid.New()
		}
	} else {
		nem.ID = uuid.New()
	}

	if ne.ExecutionID != "" {
		if execID, err := uuid.Parse(ne.ExecutionID); err == nil {
			nem.ExecutionID = execID
		}
	}

	if ne.NodeID != "" {
		if nodeID, err := uuid.Parse(ne.NodeID); err == nil {
			nem.NodeID = nodeID
		}
	}

	if !ne.StartedAt.IsZero() {
		nem.StartedAt = &ne.StartedAt
	}
	if ne.CompletedAt != nil && !ne.CompletedAt.IsZero() {
		nem.CompletedAt = ne.CompletedAt
	}

	return nem
}
