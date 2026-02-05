package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/smilemakc/mbflow/pkg/models"
)

// WorkflowToStorage converts a domain workflow to a storage workflow model
// This is used for both Create and Update operations
func WorkflowToStorage(w *models.Workflow, workflowID uuid.UUID) *WorkflowModel {
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
func NodeToStorage(n *models.Node, workflowID uuid.UUID) *NodeModel {
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
func EdgeToStorage(e *models.Edge, workflowID uuid.UUID) *EdgeModel {
	var condition JSONBMap
	if e.Condition != "" {
		// Store condition as a simple map for now
		condition = JSONBMap{"expression": e.Condition}
	}

	return &EdgeModel{
		// ID will be set by Repository (preserved on update, new on create)
		EdgeID:     e.ID,
		WorkflowID: workflowID,
		FromNodeID: e.From,
		ToNodeID:   e.To,
		Condition:  condition,
	}
}

// WorkflowFromStorage converts a storage workflow model to a domain workflow
func WorkflowFromStorage(sw *WorkflowModel) *models.Workflow {
	nodes := make([]*models.Node, len(sw.Nodes))
	for i, node := range sw.Nodes {
		nodes[i] = NodeFromStorage(node)
	}

	edges := make([]*models.Edge, len(sw.Edges))
	for i, edge := range sw.Edges {
		edges[i] = EdgeFromStorage(edge)
	}

	var variables map[string]interface{}
	if sw.Variables != nil {
		variables = map[string]interface{}(sw.Variables)
	}

	var metadata map[string]interface{}
	if sw.Metadata != nil {
		metadata = map[string]interface{}(sw.Metadata)
	}

	// Extract tags from metadata if present
	var tags []string
	if metadata != nil {
		// Try both []string and []interface{} for compatibility
		if tagsVal, ok := metadata["tags"].([]string); ok {
			tags = tagsVal
		} else if tagsVal, ok := metadata["tags"].([]interface{}); ok {
			tags = make([]string, len(tagsVal))
			for i, t := range tagsVal {
				if tagStr, ok := t.(string); ok {
					tags[i] = tagStr
				}
			}
		}
	}

	return &models.Workflow{
		ID:          sw.ID.String(),
		Name:        sw.Name,
		Description: sw.Description,
		Version:     sw.Version,
		Status:      models.WorkflowStatus(sw.Status),
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
func NodeFromStorage(sn *NodeModel) *models.Node {
	var position *models.Position
	if sn.Position != nil {
		x, _ := sn.Position["x"].(float64)
		y, _ := sn.Position["y"].(float64)
		position = &models.Position{X: x, Y: y}
	}

	var config map[string]interface{}
	if sn.Config != nil {
		config = map[string]interface{}(sn.Config)
	}

	var metadata map[string]interface{}
	// NodeModel doesn't have metadata yet, but we're ready for it

	return &models.Node{
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
func EdgeFromStorage(se *EdgeModel) *models.Edge {
	var condition string
	if se.Condition != nil {
		if expr, ok := se.Condition["expression"].(string); ok {
			condition = expr
		}
	}

	var metadata map[string]interface{}
	// EdgeModel doesn't have metadata yet, but we're ready for it

	return &models.Edge{
		ID:        se.EdgeID,     // Use logical ID
		From:      se.FromNodeID, // Use logical ID
		To:        se.ToNodeID,   // Use logical ID
		Condition: condition,
		Metadata:  metadata,
	}
}

// WorkflowResourceToStorage converts domain WorkflowResource to storage model
func WorkflowResourceToStorage(domain *models.WorkflowResource, workflowID uuid.UUID) *WorkflowResourceModel {
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
func WorkflowResourceFromStorage(storage *WorkflowResourceModel) *models.WorkflowResource {
	return &models.WorkflowResource{
		ResourceID: storage.ResourceID.String(),
		Alias:      storage.Alias,
		AccessType: storage.AccessType,
	}
}

// WorkflowResourcesToStorage converts slice of domain WorkflowResource to storage models
func WorkflowResourcesToStorage(domains []models.WorkflowResource, workflowID uuid.UUID) []*WorkflowResourceModel {
	result := make([]*WorkflowResourceModel, len(domains))
	for i, d := range domains {
		result[i] = WorkflowResourceToStorage(&d, workflowID)
	}
	return result
}

// WorkflowResourcesFromStorage converts slice of storage models to domain WorkflowResources
func WorkflowResourcesFromStorage(storage []*WorkflowResourceModel) []models.WorkflowResource {
	result := make([]models.WorkflowResource, len(storage))
	for i, s := range storage {
		result[i] = *WorkflowResourceFromStorage(s)
	}
	return result
}

// WorkflowModelToDomain converts storage WorkflowModel to domain Workflow
func WorkflowModelToDomain(wm *WorkflowModel) *models.Workflow {
	if wm == nil {
		return nil
	}

	workflow := &models.Workflow{
		ID:          wm.ID.String(),
		Name:        wm.Name,
		Description: wm.Description,
		Status:      models.WorkflowStatus(wm.Status),
		Variables:   make(map[string]interface{}),
		Metadata:    make(map[string]interface{}),
		CreatedAt:   wm.CreatedAt,
		UpdatedAt:   wm.UpdatedAt,
	}

	if wm.CreatedBy != nil {
		workflow.CreatedBy = wm.CreatedBy.String()
	}

	if wm.Variables != nil {
		workflow.Variables = map[string]interface{}(wm.Variables)
	}

	if wm.Metadata != nil {
		workflow.Metadata = map[string]interface{}(wm.Metadata)
	}

	workflow.Nodes = make([]*models.Node, 0, len(wm.Nodes))
	for _, nm := range wm.Nodes {
		workflow.Nodes = append(workflow.Nodes, NodeModelToDomain(nm))
	}

	workflow.Edges = make([]*models.Edge, 0, len(wm.Edges))
	for _, em := range wm.Edges {
		workflow.Edges = append(workflow.Edges, EdgeModelToDomain(em))
	}

	workflow.Resources = make([]models.WorkflowResource, 0, len(wm.Resources))
	for _, rm := range wm.Resources {
		wr := models.WorkflowResource{
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
func NodeModelToDomain(nm *NodeModel) *models.Node {
	if nm == nil {
		return nil
	}

	node := &models.Node{
		ID:     nm.NodeID,
		Name:   nm.Name,
		Type:   nm.Type,
		Config: make(map[string]interface{}),
	}

	if nm.Config != nil {
		node.Config = map[string]interface{}(nm.Config)
	}

	if nm.Position != nil {
		posMap := map[string]interface{}(nm.Position)
		if x, ok := posMap["x"].(float64); ok {
			if y, ok := posMap["y"].(float64); ok {
				node.Position = &models.Position{X: x, Y: y}
			}
		}
	}

	return node
}

// EdgeModelToDomain converts storage EdgeModel to domain Edge
func EdgeModelToDomain(em *EdgeModel) *models.Edge {
	if em == nil {
		return nil
	}

	edge := &models.Edge{
		ID:   em.EdgeID,
		From: em.FromNodeID,
		To:   em.ToNodeID,
	}

	if em.Condition != nil {
		if expr, ok := em.Condition["expression"].(string); ok {
			edge.Condition = expr
		}
	}

	return edge
}

// ExecutionModelToDomain converts storage ExecutionModel to domain Execution
func ExecutionModelToDomain(exm *ExecutionModel) *models.Execution {
	if exm == nil {
		return nil
	}

	exec := &models.Execution{
		ID:         exm.ID.String(),
		WorkflowID: exm.WorkflowID.String(),
		Status:     models.ExecutionStatus(exm.Status),
		Input:      make(map[string]interface{}),
		Output:     make(map[string]interface{}),
		Variables:  make(map[string]interface{}),
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
		exec.NodeExecutions = make([]*models.NodeExecution, len(exm.NodeExecutions))
		for i, ne := range exm.NodeExecutions {
			exec.NodeExecutions[i] = NodeExecutionModelToDomain(ne)
		}
	}

	return exec
}

// ExecutionDomainToModel converts domain Execution to storage ExecutionModel
func ExecutionDomainToModel(exec *models.Execution) *ExecutionModel {
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
func NodeExecutionModelToDomain(nem *NodeExecutionModel) *models.NodeExecution {
	if nem == nil {
		return nil
	}

	ne := &models.NodeExecution{
		ID:             nem.ID.String(),
		ExecutionID:    nem.ExecutionID.String(),
		NodeID:         nem.NodeID.String(),
		Status:         models.NodeExecutionStatus(nem.Status),
		Input:          make(map[string]interface{}),
		Output:         make(map[string]interface{}),
		Config:         make(map[string]interface{}),
		ResolvedConfig: make(map[string]interface{}),
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
func NodeExecutionDomainToModel(ne *models.NodeExecution) *NodeExecutionModel {
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
