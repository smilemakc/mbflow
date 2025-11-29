package rest

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/google/uuid"
	"github.com/smilemakc/mbflow/internal/domain"
)

// CreateWorkflowRequest represents the request body for creating a workflow
type CreateWorkflowRequest struct {
	Name        string                 `json:"name"`
	Version     string                 `json:"version"`
	Description string                 `json:"description,omitempty"`
	Nodes       []NodeRequest          `json:"nodes"`
	Edges       []EdgeRequest          `json:"edges"`
	Triggers    []TriggerRequest       `json:"triggers"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// NodeRequest represents a node in the workflow creation request
type NodeRequest struct {
	Type   string                 `json:"type"`
	Name   string                 `json:"name"`
	Config map[string]interface{} `json:"config,omitempty"`
}

// EdgeRequest represents an edge in the workflow creation request
type EdgeRequest struct {
	From      string                 `json:"from"`
	To        string                 `json:"to"`
	Type      string                 `json:"type"`
	Condition map[string]interface{} `json:"condition,omitempty"`
}

// TriggerRequest represents a trigger in the workflow creation request
type TriggerRequest struct {
	Type   string                 `json:"type"`
	Config map[string]interface{} `json:"config,omitempty"`
}

// UpdateWorkflowRequest represents the request body for updating a workflow
// All fields are optional for partial updates
type UpdateWorkflowRequest struct {
	Name        *string                `json:"name,omitempty"`
	Version     *string                `json:"version,omitempty"`
	Description *string                `json:"description,omitempty"`
	Nodes       []NodeRequest          `json:"nodes,omitempty"`
	Edges       []EdgeRequest          `json:"edges,omitempty"`
	Triggers    []TriggerRequest       `json:"triggers,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// WorkflowResponse represents the response for a workflow
type WorkflowResponse struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Version     string                 `json:"version"`
	Description string                 `json:"description,omitempty"`
	Nodes       []NodeResponse         `json:"nodes"`
	Edges       []EdgeResponse         `json:"edges"`
	Triggers    []TriggerResponse      `json:"triggers"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt   string                 `json:"created_at,omitempty"`
}

// NodeResponse represents a node in the workflow response
type NodeResponse struct {
	ID     string                 `json:"id"`
	Type   string                 `json:"type"`
	Name   string                 `json:"name"`
	Config map[string]interface{} `json:"config,omitempty"`
}

// EdgeResponse represents an edge in the workflow response
type EdgeResponse struct {
	ID        string                 `json:"id"`
	From      string                 `json:"from"`
	To        string                 `json:"to"`
	Type      string                 `json:"type"`
	Condition map[string]interface{} `json:"condition,omitempty"`
}

// TriggerResponse represents a trigger in the workflow response
type TriggerResponse struct {
	ID     string                 `json:"id"`
	Type   string                 `json:"type"`
	Config map[string]interface{} `json:"config,omitempty"`
}

// handleListWorkflows handles GET /api/v1/workflows
func (s *Server) handleListWorkflows(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	workflows, err := s.store.ListWorkflows(ctx)
	if err != nil {
		s.logger.Error("failed to list workflows", "error", err)
		s.respondError(w, "Failed to list workflows", http.StatusInternalServerError)
		return
	}

	response := make([]WorkflowResponse, 0, len(workflows))
	for _, wf := range workflows {
		response = append(response, s.workflowToResponse(wf))
	}

	s.respondJSON(w, response, http.StatusOK)
}

// handleGetWorkflow handles GET /api/v1/workflows/{id}
func (s *Server) handleGetWorkflow(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	workflowID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		s.respondError(w, "Invalid workflow ID", http.StatusBadRequest)
		return
	}

	workflow, err := s.store.GetWorkflow(ctx, workflowID)
	if err != nil {
		s.logger.Error("failed to get workflow", "error", err, "id", workflowID)
		s.respondError(w, "Workflow not found", http.StatusNotFound)
		return
	}

	s.respondJSON(w, s.workflowToResponse(workflow), http.StatusOK)
}

// handleCreateWorkflow handles POST /api/v1/workflows
func (s *Server) handleCreateWorkflow(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req CreateWorkflowRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.respondError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate request
	if req.Name == "" {
		s.respondError(w, "Workflow name is required", http.StatusBadRequest)
		return
	}
	if req.Version == "" {
		s.respondError(w, "Workflow version is required", http.StatusBadRequest)
		return
	}

	// Create workflow using domain factory
	workflow, err := domain.NewWorkflow(req.Name, req.Version, req.Description, req.Metadata)
	if err != nil {
		s.respondError(w, fmt.Sprintf("Failed to create workflow: %v", err), http.StatusBadRequest)
		return
	}

	// Add nodes
	nodeNameToID := make(map[string]uuid.UUID)
	for _, nodeReq := range req.Nodes {
		nodeID, err := workflow.AddNode(domain.NodeType(nodeReq.Type), nodeReq.Name, nodeReq.Config)
		if err != nil {
			s.respondError(w, fmt.Sprintf("Failed to add node %s: %v", nodeReq.Name, err), http.StatusBadRequest)
			return
		}
		nodeNameToID[nodeReq.Name] = nodeID
	}

	// Add edges
	for _, edgeReq := range req.Edges {
		fromID, ok := nodeNameToID[edgeReq.From]
		if !ok {
			s.respondError(w, fmt.Sprintf("Node not found: %s", edgeReq.From), http.StatusBadRequest)
			return
		}
		toID, ok := nodeNameToID[edgeReq.To]
		if !ok {
			s.respondError(w, fmt.Sprintf("Node not found: %s", edgeReq.To), http.StatusBadRequest)
			return
		}

		_, err := workflow.AddEdge(fromID, toID, domain.EdgeType(edgeReq.Type), edgeReq.Condition)
		if err != nil {
			s.respondError(w, fmt.Sprintf("Failed to add edge: %v", err), http.StatusBadRequest)
			return
		}
	}

	// Add triggers
	for _, triggerReq := range req.Triggers {
		_, err := workflow.AddTrigger(domain.TriggerType(triggerReq.Type), triggerReq.Config)
		if err != nil {
			s.respondError(w, fmt.Sprintf("Failed to add trigger: %v", err), http.StatusBadRequest)
			return
		}
	}

	// Validate workflow
	if err := workflow.Validate(); err != nil {
		s.respondError(w, fmt.Sprintf("Invalid workflow: %v", err), http.StatusBadRequest)
		return
	}

	// Save workflow
	if err := s.store.SaveWorkflow(ctx, workflow); err != nil {
		s.logger.Error("failed to save workflow", "error", err)
		s.respondError(w, "Failed to save workflow", http.StatusInternalServerError)
		return
	}

	s.respondJSON(w, s.workflowToResponse(workflow), http.StatusCreated)
}

// handleUpdateWorkflow handles PUT /api/v1/workflows/{id}
// Supports partial updates - only provided fields will be updated
func (s *Server) handleUpdateWorkflow(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	workflowID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		s.respondError(w, "Invalid workflow ID", http.StatusBadRequest)
		return
	}

	var req UpdateWorkflowRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.respondError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Get existing workflow
	existingWorkflow, err := s.store.GetWorkflow(ctx, workflowID)
	if err != nil {
		s.logger.Error("failed to get workflow for update", "error", err, "id", workflowID)
		s.respondError(w, "Workflow not found", http.StatusNotFound)
		return
	}

	// Prepare updated values - use existing values as defaults
	name := existingWorkflow.Name()
	version := existingWorkflow.Version()
	description := existingWorkflow.Description()
	spec := existingWorkflow.Spec()

	// Apply partial updates
	if req.Name != nil {
		name = *req.Name
	}
	if req.Version != nil {
		version = *req.Version
	}
	if req.Description != nil {
		description = *req.Description
	}
	if req.Metadata != nil {
		spec = req.Metadata
	}

	// If nodes/edges/triggers are provided, rebuild workflow structure
	// Otherwise, preserve existing structure
	hasStructuralChanges := len(req.Nodes) > 0 || len(req.Edges) > 0 || len(req.Triggers) > 0

	var workflow domain.Workflow

	if hasStructuralChanges {
		// Full structural update - create new workflow with provided structure
		workflow, err = domain.RestoreWorkflow(workflowID, name, version, description, spec)
		if err != nil {
			s.respondError(w, fmt.Sprintf("Failed to create workflow: %v", err), http.StatusBadRequest)
			return
		}

		// Add nodes
		nodeNameToID := make(map[string]uuid.UUID)
		for _, nodeReq := range req.Nodes {
			nodeID, err := workflow.AddNode(domain.NodeType(nodeReq.Type), nodeReq.Name, nodeReq.Config)
			if err != nil {
				s.respondError(w, fmt.Sprintf("Failed to add node %s: %v", nodeReq.Name, err), http.StatusBadRequest)
				return
			}
			nodeNameToID[nodeReq.Name] = nodeID
		}

		// Add edges
		for _, edgeReq := range req.Edges {
			fromID, ok := nodeNameToID[edgeReq.From]
			if !ok {
				s.respondError(w, fmt.Sprintf("Node not found: %s", edgeReq.From), http.StatusBadRequest)
				return
			}
			toID, ok := nodeNameToID[edgeReq.To]
			if !ok {
				s.respondError(w, fmt.Sprintf("Node not found: %s", edgeReq.To), http.StatusBadRequest)
				return
			}

			_, err := workflow.AddEdge(fromID, toID, domain.EdgeType(edgeReq.Type), edgeReq.Condition)
			if err != nil {
				s.respondError(w, fmt.Sprintf("Failed to add edge: %v", err), http.StatusBadRequest)
				return
			}
		}

		// Add triggers
		for _, triggerReq := range req.Triggers {
			_, err := workflow.AddTrigger(domain.TriggerType(triggerReq.Type), triggerReq.Config)
			if err != nil {
				s.respondError(w, fmt.Sprintf("Failed to add trigger: %v", err), http.StatusBadRequest)
				return
			}
		}
	} else {
		// Metadata-only update - preserve existing structure
		existingNodes := existingWorkflow.GetAllNodes()
		existingEdges := existingWorkflow.GetAllEdges()
		existingTriggers := existingWorkflow.GetAllTriggers()

		workflow, err = domain.ReconstructWorkflow(
			workflowID,
			name, version, description, spec,
			existingWorkflow.CreatedAt(), existingWorkflow.UpdatedAt(),
			existingNodes, existingEdges, existingTriggers,
		)
		if err != nil {
			s.respondError(w, fmt.Sprintf("Failed to update workflow: %v", err), http.StatusBadRequest)
			return
		}
	}

	// Validate workflow
	if err := workflow.Validate(); err != nil {
		s.respondError(w, fmt.Sprintf("Invalid workflow: %v", err), http.StatusBadRequest)
		return
	}

	// Save updated workflow
	if err := s.store.SaveWorkflow(ctx, workflow); err != nil {
		s.logger.Error("failed to update workflow", "error", err)
		s.respondError(w, "Failed to update workflow", http.StatusInternalServerError)
		return
	}

	s.respondJSON(w, s.workflowToResponse(workflow), http.StatusOK)
}

// handleDeleteWorkflow handles DELETE /api/v1/workflows/{id}
func (s *Server) handleDeleteWorkflow(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	workflowID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		s.respondError(w, "Invalid workflow ID", http.StatusBadRequest)
		return
	}

	if err := s.store.DeleteWorkflow(ctx, workflowID); err != nil {
		s.logger.Error("failed to delete workflow", "error", err)
		s.respondError(w, "Failed to delete workflow", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// workflowToResponse converts a domain workflow to a response DTO
func (s *Server) workflowToResponse(wf domain.Workflow) WorkflowResponse {
	nodes := make([]NodeResponse, 0)
	for _, node := range wf.GetAllNodes() {
		nodes = append(nodes, NodeResponse{
			ID:     node.ID().String(),
			Type:   node.Type().String(),
			Name:   node.Name(),
			Config: node.Config(),
		})
	}

	edges := make([]EdgeResponse, 0)
	nodeIDToName := make(map[uuid.UUID]string)
	for _, node := range wf.GetAllNodes() {
		nodeIDToName[node.ID()] = node.Name()
	}

	for _, edge := range wf.GetAllEdges() {
		edges = append(edges, EdgeResponse{
			ID:        edge.ID().String(),
			From:      nodeIDToName[edge.FromNodeID()],
			To:        nodeIDToName[edge.ToNodeID()],
			Type:      edge.Type().String(),
			Condition: edge.Config(),
		})
	}

	triggers := make([]TriggerResponse, 0)
	for _, trigger := range wf.GetAllTriggers() {
		triggers = append(triggers, TriggerResponse{
			ID:     trigger.ID().String(),
			Type:   trigger.Type().String(),
			Config: trigger.Config(),
		})
	}

	return WorkflowResponse{
		ID:          wf.ID().String(),
		Name:        wf.Name(),
		Version:     wf.Version(),
		Description: wf.Description(),
		Nodes:       nodes,
		Edges:       edges,
		Triggers:    triggers,
		Metadata:    wf.Spec(),
		CreatedAt:   wf.CreatedAt().Format("2006-01-02T15:04:05Z07:00"),
	}
}
