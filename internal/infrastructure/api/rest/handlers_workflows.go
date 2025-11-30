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
	Draft       bool                   `json:"draft,omitempty"` // If true, allows saving without full validation
}

// NodeRequest represents a node in the workflow creation request
type NodeRequest struct {
	ID     string                 `json:"id,omitempty"` // Optional - for preserving node ID during updates
	Type   string                 `json:"type"`
	Name   string                 `json:"name"`
	Config map[string]interface{} `json:"config,omitempty"`
}

// EdgeRequest represents an edge in the workflow creation request
type EdgeRequest struct {
	ID        string                 `json:"id,omitempty"` // Optional - for preserving edge ID during updates
	From      string                 `json:"from"`
	To        string                 `json:"to"`
	Type      string                 `json:"type"`
	Condition map[string]interface{} `json:"condition,omitempty"`
}

// TriggerRequest represents a trigger in the workflow creation request
type TriggerRequest struct {
	ID     string                 `json:"id,omitempty"` // Optional - for preserving trigger ID during updates
	Type   string                 `json:"type"`
	Config map[string]interface{} `json:"config,omitempty"`
}

// UpdateWorkflowRequest represents the request body for updating a workflow
// All fields are optional for partial updates
type UpdateWorkflowRequest struct {
	Name        *string                `json:"name,omitempty"`
	Version     *string                `json:"version,omitempty"`
	Description *string                `json:"description,omitempty"`
	Nodes       *[]NodeRequest         `json:"nodes,omitempty"`
	Edges       *[]EdgeRequest         `json:"edges,omitempty"`
	Triggers    *[]TriggerRequest      `json:"triggers,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
	Draft       *bool                  `json:"draft,omitempty"` // If true, allows saving without full validation
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

// findNodeID finds a node ID by name or UUID string
// Returns the UUID and true if found, zero UUID and false otherwise
func findNodeID(identifier string, nodeNameToID map[string]uuid.UUID, workflow domain.Workflow) (uuid.UUID, bool) {
	// First, try to find by name
	if nodeID, ok := nodeNameToID[identifier]; ok {
		return nodeID, true
	}

	// Second, try to parse as UUID and check if it exists in workflow
	if nodeUUID, err := uuid.Parse(identifier); err == nil {
		// Check if this UUID exists in the workflow
		for _, node := range workflow.GetAllNodes() {
			if node.ID() == nodeUUID {
				return nodeUUID, true
			}
		}
	}

	return uuid.UUID{}, false
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

	// Add nodes - support both new nodes and nodes with existing IDs
	nodeNameToID := make(map[string]uuid.UUID)
	for _, nodeReq := range req.Nodes {
		var nodeID uuid.UUID
		var err error

		if nodeReq.ID != "" {
			// Use existing ID - parse and validate
			nodeID, err = uuid.Parse(nodeReq.ID)
			if err != nil {
				s.respondError(w, fmt.Sprintf("Invalid node ID '%s': %v", nodeReq.ID, err), http.StatusBadRequest)
				return
			}

			// Create node with existing ID using RestoreNode
			node := domain.RestoreNode(nodeID, domain.NodeType(nodeReq.Type), nodeReq.Name, nodeReq.Config)
			if err := workflow.UseNode(node); err != nil {
				s.respondError(w, fmt.Sprintf("Failed to add node %s: %v", nodeReq.Name, err), http.StatusBadRequest)
				return
			}
		} else {
			// Create new node with generated ID
			nodeID, err = workflow.AddNode(domain.NodeType(nodeReq.Type), nodeReq.Name, nodeReq.Config)
			if err != nil {
				s.respondError(w, fmt.Sprintf("Failed to add node %s: %v", nodeReq.Name, err), http.StatusBadRequest)
				return
			}
		}

		nodeNameToID[nodeReq.Name] = nodeID
	}

	// Add edges
	for _, edgeReq := range req.Edges {
		// Find 'from' node by name or ID
		fromID, ok := findNodeID(edgeReq.From, nodeNameToID, workflow)
		if !ok {
			s.respondError(w, fmt.Sprintf("Node not found: %s", edgeReq.From), http.StatusBadRequest)
			return
		}

		// Find 'to' node by name or ID
		toID, ok := findNodeID(edgeReq.To, nodeNameToID, workflow)
		if !ok {
			s.respondError(w, fmt.Sprintf("Node not found: %s", edgeReq.To), http.StatusBadRequest)
			return
		}

		if edgeReq.ID != "" {
			// Use existing edge ID
			edgeID, err := uuid.Parse(edgeReq.ID)
			if err != nil {
				s.respondError(w, fmt.Sprintf("Invalid edge ID '%s': %v", edgeReq.ID, err), http.StatusBadRequest)
				return
			}

			// Create edge with existing ID using RestoreEdge
			edge := domain.RestoreEdge(edgeID, fromID, toID, domain.EdgeType(edgeReq.Type), edgeReq.Condition)
			if err := workflow.UseEdge(edge); err != nil {
				s.respondError(w, fmt.Sprintf("Failed to add edge: %v", err), http.StatusBadRequest)
				return
			}
		} else {
			// Create new edge with generated ID
			_, err := workflow.AddEdge(fromID, toID, domain.EdgeType(edgeReq.Type), edgeReq.Condition)
			if err != nil {
				s.respondError(w, fmt.Sprintf("Failed to add edge: %v", err), http.StatusBadRequest)
				return
			}
		}
	}

	// Add triggers
	// Note: Currently triggers always get new IDs. ID preservation for triggers
	// requires adding UseTrigger method to Workflow interface (future enhancement)
	for _, triggerReq := range req.Triggers {
		_, err := workflow.AddTrigger(domain.TriggerType(triggerReq.Type), triggerReq.Config)
		if err != nil {
			s.respondError(w, fmt.Sprintf("Failed to add trigger: %v", err), http.StatusBadRequest)
			return
		}
	}

	// Validate workflow based on draft mode
	if req.Draft {
		// For drafts, only validate structure (allows saving without triggers)
		if err := workflow.ValidateStructure(); err != nil {
			s.respondError(w, fmt.Sprintf("Invalid workflow structure: %v", err), http.StatusBadRequest)
			return
		}
	} else {
		// For production workflows, validate for execution (requires triggers)
		if err := workflow.ValidateForExecution(); err != nil {
			s.respondError(w, fmt.Sprintf("Workflow not ready for execution: %v", err), http.StatusBadRequest)
			return
		}
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
	hasStructuralChanges := req.Nodes != nil || req.Edges != nil || req.Triggers != nil

	var workflow domain.Workflow

	if hasStructuralChanges {
		// Full structural update - create new workflow with provided structure
		workflow, err = domain.RestoreWorkflow(workflowID, name, version, description, spec)
		if err != nil {
			s.respondError(w, fmt.Sprintf("Failed to create workflow: %v", err), http.StatusBadRequest)
			return
		}

		// Add nodes - support both new nodes and nodes with existing IDs
		nodeNameToID := make(map[string]uuid.UUID)

		if req.Nodes != nil {
			// Use provided nodes
			for _, nodeReq := range *req.Nodes {
				var nodeID uuid.UUID
				var err error

				if nodeReq.ID != "" {
					// Use existing ID - parse and validate
					nodeID, err = uuid.Parse(nodeReq.ID)
					if err != nil {
						s.respondError(w, fmt.Sprintf("Invalid node ID '%s': %v", nodeReq.ID, err), http.StatusBadRequest)
						return
					}

					// Create node with existing ID using RestoreNode
					node := domain.RestoreNode(nodeID, domain.NodeType(nodeReq.Type), nodeReq.Name, nodeReq.Config)
					if err := workflow.UseNode(node); err != nil {
						s.respondError(w, fmt.Sprintf("Failed to add node %s: %v", nodeReq.Name, err), http.StatusBadRequest)
						return
					}
				} else {
					// Create new node with generated ID
					nodeID, err = workflow.AddNode(domain.NodeType(nodeReq.Type), nodeReq.Name, nodeReq.Config)
					if err != nil {
						s.respondError(w, fmt.Sprintf("Failed to add node %s: %v", nodeReq.Name, err), http.StatusBadRequest)
						return
					}
				}

				nodeNameToID[nodeReq.Name] = nodeID
			}
		} else {
			// Preserve existing nodes
			for _, n := range existingWorkflow.GetAllNodes() {
				if err := workflow.UseNode(n); err != nil {
					s.respondError(w, fmt.Sprintf("Failed to preserve node %s: %v", n.Name(), err), http.StatusInternalServerError)
					return
				}
				nodeNameToID[n.Name()] = n.ID()
			}
		}

		// Add edges
		if req.Edges != nil {
			for _, edgeReq := range *req.Edges {
				// Find 'from' node by name or ID
				fromID, ok := findNodeID(edgeReq.From, nodeNameToID, workflow)
				if !ok {
					s.respondError(w, fmt.Sprintf("Node not found: %s", edgeReq.From), http.StatusBadRequest)
					return
				}

				// Find 'to' node by name or ID
				toID, ok := findNodeID(edgeReq.To, nodeNameToID, workflow)
				if !ok {
					s.respondError(w, fmt.Sprintf("Node not found: %s", edgeReq.To), http.StatusBadRequest)
					return
				}

				if edgeReq.ID != "" {
					// Use existing edge ID
					edgeID, err := uuid.Parse(edgeReq.ID)
					if err != nil {
						s.respondError(w, fmt.Sprintf("Invalid edge ID '%s': %v", edgeReq.ID, err), http.StatusBadRequest)
						return
					}

					// Create edge with existing ID using RestoreEdge
					edge := domain.RestoreEdge(edgeID, fromID, toID, domain.EdgeType(edgeReq.Type), edgeReq.Condition)
					if err := workflow.UseEdge(edge); err != nil {
						s.respondError(w, fmt.Sprintf("Failed to add edge: %v", err), http.StatusBadRequest)
						return
					}
				} else {
					// Create new edge with generated ID
					_, err := workflow.AddEdge(fromID, toID, domain.EdgeType(edgeReq.Type), edgeReq.Condition)
					if err != nil {
						s.respondError(w, fmt.Sprintf("Failed to add edge: %v", err), http.StatusBadRequest)
						return
					}
				}
			}
		} else {
			// Preserve existing edges
			for _, e := range existingWorkflow.GetAllEdges() {
				if err := workflow.UseEdge(e); err != nil {
					s.respondError(w, fmt.Sprintf("Failed to preserve edge: %v", err), http.StatusInternalServerError)
					return
				}
			}
		}

		// Add triggers
		if req.Triggers != nil {
			for _, triggerReq := range *req.Triggers {
				_, err := workflow.AddTrigger(domain.TriggerType(triggerReq.Type), triggerReq.Config)
				if err != nil {
					s.respondError(w, fmt.Sprintf("Failed to add trigger: %v", err), http.StatusBadRequest)
					return
				}
			}
		} else {
			// Preserve existing triggers
			// Note: IDs are not preserved for triggers yet
			for _, t := range existingWorkflow.GetAllTriggers() {
				_, err := workflow.AddTrigger(t.Type(), t.Config())
				if err != nil {
					s.respondError(w, fmt.Sprintf("Failed to preserve trigger: %v", err), http.StatusInternalServerError)
					return
				}
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

	// Validate workflow based on draft mode
	isDraft := req.Draft != nil && *req.Draft
	if isDraft {
		// For drafts, only validate structure (allows saving without triggers)
		if err := workflow.ValidateStructure(); err != nil {
			s.respondError(w, fmt.Sprintf("Invalid workflow structure: %v", err), http.StatusBadRequest)
			return
		}
	} else {
		// For production workflows, validate for execution (requires triggers)
		if err := workflow.ValidateForExecution(); err != nil {
			s.respondError(w, fmt.Sprintf("Workflow not ready for execution: %v", err), http.StatusBadRequest)
			return
		}
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
