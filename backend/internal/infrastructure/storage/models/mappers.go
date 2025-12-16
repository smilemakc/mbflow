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
