package serviceapi

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/smilemakc/mbflow/internal/application/engine"
	"github.com/smilemakc/mbflow/internal/domain/repository"
	storagemodels "github.com/smilemakc/mbflow/internal/infrastructure/storage/models"
	"github.com/smilemakc/mbflow/pkg/models"
)

// ListWorkflowsParams contains parameters for listing workflows.
type ListWorkflowsParams struct {
	Limit  int
	Offset int
	Status *string
	UserID *uuid.UUID
}

// ListWorkflowsResult contains the result of listing workflows.
type ListWorkflowsResult struct {
	Workflows []*models.Workflow
	Total     int
}

func (o *Operations) ListWorkflows(ctx context.Context, params ListWorkflowsParams) (*ListWorkflowsResult, error) {
	filters := repository.WorkflowFilters{
		IncludeUnowned: true,
	}

	if params.Status != nil {
		filters.Status = params.Status
	}

	if params.UserID != nil {
		filters.CreatedBy = params.UserID
		filters.IncludeUnowned = false
	}

	workflowModels, err := o.WorkflowRepo.FindAllWithFilters(ctx, filters, params.Limit, params.Offset)
	if err != nil {
		o.Logger.Error("Failed to list workflows", "error", err, "limit", params.Limit, "offset", params.Offset)
		return nil, err
	}

	workflows := make([]*models.Workflow, len(workflowModels))
	for i, wm := range workflowModels {
		workflows[i] = engine.WorkflowModelToDomain(wm)
	}

	total, err := o.WorkflowRepo.CountWithFilters(ctx, filters)
	if err != nil {
		total = len(workflows)
	}

	return &ListWorkflowsResult{
		Workflows: workflows,
		Total:     total,
	}, nil
}

// GetWorkflowParams contains parameters for getting a workflow.
type GetWorkflowParams struct {
	WorkflowID uuid.UUID
}

func (o *Operations) GetWorkflow(ctx context.Context, params GetWorkflowParams) (*models.Workflow, error) {
	workflowModel, err := o.WorkflowRepo.FindByIDWithRelations(ctx, params.WorkflowID)
	if err != nil {
		o.Logger.Error("Failed to find workflow", "error", err, "workflow_id", params.WorkflowID)
		return nil, err
	}

	return engine.WorkflowModelToDomain(workflowModel), nil
}

// CreateWorkflowParams contains parameters for creating a workflow.
type CreateWorkflowParams struct {
	Name        string
	Description string
	Variables   map[string]any
	Metadata    map[string]any
	CreatedBy   *uuid.UUID
}

func (o *Operations) CreateWorkflow(ctx context.Context, params CreateWorkflowParams) (*models.Workflow, error) {
	if params.Name == "" {
		return nil, NewValidationError("NAME_REQUIRED", "Workflow name is required")
	}

	workflowModel := &storagemodels.WorkflowModel{
		ID:          uuid.New(),
		Name:        params.Name,
		Description: params.Description,
		Status:      "draft",
		Version:     1,
		Variables:   storagemodels.JSONBMap(params.Variables),
		Metadata:    storagemodels.JSONBMap(params.Metadata),
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	if params.CreatedBy != nil {
		workflowModel.CreatedBy = params.CreatedBy
	}

	if err := o.WorkflowRepo.Create(ctx, workflowModel); err != nil {
		o.Logger.Error("Failed to create workflow", "error", err, "workflow_name", params.Name)
		return nil, err
	}

	return engine.WorkflowModelToDomain(workflowModel), nil
}

// NodeInput represents a node in an update request.
type NodeInput struct {
	ID       string
	Name     string
	Type     string
	Config   map[string]any
	Position map[string]any
}

// EdgeInput represents an edge in an update request.
type EdgeInput struct {
	ID        string
	From      string
	To        string
	Condition map[string]any
}

// ResourceInput represents a resource attachment in an update request.
type ResourceInput struct {
	ResourceID string
	Alias      string
	AccessType string
}

// UpdateWorkflowParams contains parameters for updating a workflow.
type UpdateWorkflowParams struct {
	WorkflowID  uuid.UUID
	Name        string
	Description string
	Variables   map[string]any
	Metadata    map[string]any
	Nodes       []NodeInput
	Edges       []EdgeInput
	Resources   []ResourceInput
}

func (o *Operations) UpdateWorkflow(ctx context.Context, params UpdateWorkflowParams) (*models.Workflow, error) {
	if err := o.validateNodes(params.Nodes); err != nil {
		return nil, NewValidationError("NODE_VALIDATION_FAILED", err.Error())
	}

	if err := o.validateEdges(params.Edges, params.Nodes); err != nil {
		return nil, NewValidationError("EDGE_VALIDATION_FAILED", err.Error())
	}

	workflowModel, err := o.WorkflowRepo.FindByID(ctx, params.WorkflowID)
	if err != nil {
		o.Logger.Error("Failed to find workflow for update", "error", err, "workflow_id", params.WorkflowID)
		return nil, err
	}

	if params.Name != "" {
		workflowModel.Name = params.Name
	}
	if params.Description != "" {
		workflowModel.Description = params.Description
	}
	if params.Variables != nil {
		workflowModel.Variables = storagemodels.JSONBMap(params.Variables)
	}
	if params.Metadata != nil {
		workflowModel.Metadata = storagemodels.JSONBMap(params.Metadata)
	}

	if params.Nodes != nil {
		workflowModel.Nodes = make([]*storagemodels.NodeModel, len(params.Nodes))
		for i, nodeReq := range params.Nodes {
			workflowModel.Nodes[i] = &storagemodels.NodeModel{
				NodeID:     nodeReq.ID,
				WorkflowID: params.WorkflowID,
				Name:       nodeReq.Name,
				Type:       nodeReq.Type,
				Config:     storagemodels.JSONBMap(nodeReq.Config),
				Position:   storagemodels.JSONBMap(nodeReq.Position),
			}
		}
	}

	if params.Edges != nil {
		workflowModel.Edges = make([]*storagemodels.EdgeModel, len(params.Edges))
		for i, edgeReq := range params.Edges {
			workflowModel.Edges[i] = &storagemodels.EdgeModel{
				EdgeID:     edgeReq.ID,
				WorkflowID: params.WorkflowID,
				FromNodeID: edgeReq.From,
				ToNodeID:   edgeReq.To,
				Condition:  storagemodels.JSONBMap(edgeReq.Condition),
			}
		}
	}

	if params.Resources != nil {
		workflowModel.Resources = make([]*storagemodels.WorkflowResourceModel, len(params.Resources))
		for i, resReq := range params.Resources {
			resourceUUID, parseErr := uuid.Parse(resReq.ResourceID)
			if parseErr != nil {
				return nil, NewValidationError("INVALID_RESOURCE_ID", fmt.Sprintf("invalid resource_id: %s", resReq.ResourceID))
			}

			accessType := resReq.AccessType
			if accessType == "" {
				accessType = "read"
			}

			workflowModel.Resources[i] = &storagemodels.WorkflowResourceModel{
				WorkflowID: params.WorkflowID,
				ResourceID: resourceUUID,
				Alias:      resReq.Alias,
				AccessType: accessType,
			}
		}
	}

	if err := o.WorkflowRepo.Update(ctx, workflowModel); err != nil {
		o.Logger.Error("Failed to update workflow", "error", err, "workflow_id", params.WorkflowID)
		return nil, err
	}

	updatedWorkflow, err := o.WorkflowRepo.FindByIDWithRelations(ctx, params.WorkflowID)
	if err != nil {
		o.Logger.Error("Failed to fetch updated workflow", "error", err, "workflow_id", params.WorkflowID)
		return nil, err
	}

	return engine.WorkflowModelToDomain(updatedWorkflow), nil
}

// DeleteWorkflowParams contains parameters for deleting a workflow.
type DeleteWorkflowParams struct {
	WorkflowID uuid.UUID
}

func (o *Operations) DeleteWorkflow(ctx context.Context, params DeleteWorkflowParams) error {
	if err := o.WorkflowRepo.Delete(ctx, params.WorkflowID); err != nil {
		o.Logger.Error("Failed to delete workflow", "error", err, "workflow_id", params.WorkflowID)
		return err
	}
	return nil
}

func (o *Operations) validateNodes(nodes []NodeInput) error {
	if nodes == nil {
		return nil
	}

	uiOnlyTypes := map[string]bool{
		"comment": true,
	}

	nodeIDs := make(map[string]bool)

	for i, node := range nodes {
		if node.ID == "" {
			return fmt.Errorf("node at index %d: id is required", i)
		}
		if node.Name == "" {
			return fmt.Errorf("node at index %d: name is required", i)
		}
		if node.Type == "" {
			return fmt.Errorf("node at index %d: type is required", i)
		}

		if nodeIDs[node.ID] {
			return fmt.Errorf("duplicate node id: %s", node.ID)
		}
		nodeIDs[node.ID] = true

		if !uiOnlyTypes[node.Type] && !o.ExecutorManager.Has(node.Type) {
			return fmt.Errorf("node %s: invalid type '%s'", node.ID, node.Type)
		}

		if len(node.ID) > 100 {
			return fmt.Errorf("node id too long (max 100 chars): %s", node.ID)
		}
		if len(node.Name) > 255 {
			return fmt.Errorf("node %s: name too long (max 255 chars)", node.ID)
		}
	}

	return nil
}

func (o *Operations) validateEdges(edges []EdgeInput, nodes []NodeInput) error {
	if edges == nil {
		return nil
	}

	nodeIDSet := make(map[string]bool)
	for _, node := range nodes {
		nodeIDSet[node.ID] = true
	}

	edgeIDs := make(map[string]bool)

	for i, edge := range edges {
		if edge.ID == "" {
			return fmt.Errorf("edge at index %d: id is required", i)
		}
		if edge.From == "" {
			return fmt.Errorf("edge at index %d: from is required", i)
		}
		if edge.To == "" {
			return fmt.Errorf("edge at index %d: to is required", i)
		}

		if edgeIDs[edge.ID] {
			return fmt.Errorf("duplicate edge id: %s", edge.ID)
		}
		edgeIDs[edge.ID] = true

		if edge.From == edge.To {
			return fmt.Errorf("edge %s: self-reference not allowed (from=%s, to=%s)", edge.ID, edge.From, edge.To)
		}

		if len(nodes) > 0 {
			if !nodeIDSet[edge.From] {
				return fmt.Errorf("edge %s: from node '%s' not found in nodes", edge.ID, edge.From)
			}
			if !nodeIDSet[edge.To] {
				return fmt.Errorf("edge %s: to node '%s' not found in nodes", edge.ID, edge.To)
			}
		}

		if len(edge.ID) > 100 {
			return fmt.Errorf("edge id too long (max 100 chars): %s", edge.ID)
		}
		if len(edge.From) > 100 {
			return fmt.Errorf("edge %s: from node id too long (max 100 chars)", edge.ID)
		}
		if len(edge.To) > 100 {
			return fmt.Errorf("edge %s: to node id too long (max 100 chars)", edge.ID)
		}
	}

	return nil
}
