package rest

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/google/uuid"
	"github.com/smilemakc/mbflow/internal/domain"
)

// EdgeDetailResponse represents a detailed edge response
type EdgeDetailResponse struct {
	ID         string                 `json:"id"`
	WorkflowID string                 `json:"workflow_id"`
	From       string                 `json:"from"`
	FromID     string                 `json:"from_id"`
	To         string                 `json:"to"`
	ToID       string                 `json:"to_id"`
	Type       string                 `json:"type"`
	Config     map[string]interface{} `json:"config,omitempty"`
}

// CreateEdgeRequest represents the request to create an edge
type CreateEdgeRequest struct {
	From   string                 `json:"from"`
	To     string                 `json:"to"`
	Type   string                 `json:"type"`
	Config map[string]interface{} `json:"config,omitempty"`
}

// UpdateEdgeRequest represents the request to update an edge
type UpdateEdgeRequest struct {
	From   string                 `json:"from,omitempty"`
	To     string                 `json:"to,omitempty"`
	Type   string                 `json:"type,omitempty"`
	Config map[string]interface{} `json:"config,omitempty"`
}

// handleListEdges handles GET /api/v1/workflows/{workflow_id}/edges
func (s *Server) handleListEdges(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	workflowID, err := uuid.Parse(r.PathValue("workflow_id"))
	if err != nil {
		s.respondError(w, "Invalid workflow ID", http.StatusBadRequest)
		return
	}

	// Get workflow
	workflow, err := s.store.GetWorkflow(ctx, workflowID)
	if err != nil {
		s.logger.Error("failed to get workflow", "error", err, "id", workflowID)
		s.respondError(w, "Workflow not found", http.StatusNotFound)
		return
	}

	// Get all edges
	edges := workflow.GetAllEdges()
	nodes := workflow.GetAllNodes()

	// Build node ID to name map
	nodeIDToName := make(map[uuid.UUID]string)
	for _, node := range nodes {
		nodeIDToName[node.ID()] = node.Name()
	}

	response := make([]EdgeDetailResponse, 0, len(edges))
	for _, edge := range edges {
		fromName := nodeIDToName[edge.FromNodeID()]
		toName := nodeIDToName[edge.ToNodeID()]

		response = append(response, EdgeDetailResponse{
			ID:         edge.ID().String(),
			WorkflowID: workflowID.String(),
			From:       fromName,
			FromID:     edge.FromNodeID().String(),
			To:         toName,
			ToID:       edge.ToNodeID().String(),
			Type:       edge.Type().String(),
			Config:     edge.Config(),
		})
	}

	s.respondJSON(w, response, http.StatusOK)
}

// handleGetEdge handles GET /api/v1/workflows/{workflow_id}/edges/{edge_id}
func (s *Server) handleGetEdge(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	workflowID, err := uuid.Parse(r.PathValue("workflow_id"))
	if err != nil {
		s.respondError(w, "Invalid workflow ID", http.StatusBadRequest)
		return
	}

	edgeID, err := uuid.Parse(r.PathValue("edge_id"))
	if err != nil {
		s.respondError(w, "Invalid edge ID", http.StatusBadRequest)
		return
	}

	// Get workflow
	workflow, err := s.store.GetWorkflow(ctx, workflowID)
	if err != nil {
		s.logger.Error("failed to get workflow", "error", err, "id", workflowID)
		s.respondError(w, "Workflow not found", http.StatusNotFound)
		return
	}

	// Get edge
	edge, err := workflow.GetEdge(edgeID)
	if err != nil {
		s.respondError(w, "Edge not found", http.StatusNotFound)
		return
	}

	// Build node ID to name map
	nodes := workflow.GetAllNodes()
	nodeIDToName := make(map[uuid.UUID]string)
	for _, node := range nodes {
		nodeIDToName[node.ID()] = node.Name()
	}

	fromName := nodeIDToName[edge.FromNodeID()]
	toName := nodeIDToName[edge.ToNodeID()]

	response := EdgeDetailResponse{
		ID:         edge.ID().String(),
		WorkflowID: workflowID.String(),
		From:       fromName,
		FromID:     edge.FromNodeID().String(),
		To:         toName,
		ToID:       edge.ToNodeID().String(),
		Type:       edge.Type().String(),
		Config:     edge.Config(),
	}

	s.respondJSON(w, response, http.StatusOK)
}

// handleCreateEdge handles POST /api/v1/workflows/{workflow_id}/edges
func (s *Server) handleCreateEdge(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	workflowID, err := uuid.Parse(r.PathValue("workflow_id"))
	if err != nil {
		s.respondError(w, "Invalid workflow ID", http.StatusBadRequest)
		return
	}

	var req CreateEdgeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.respondError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate request
	if req.From == "" {
		s.respondError(w, "Source node (from) is required", http.StatusBadRequest)
		return
	}
	if req.To == "" {
		s.respondError(w, "Target node (to) is required", http.StatusBadRequest)
		return
	}
	if req.Type == "" {
		s.respondError(w, "Edge type is required", http.StatusBadRequest)
		return
	}

	// Get workflow
	workflow, err := s.store.GetWorkflow(ctx, workflowID)
	if err != nil {
		s.logger.Error("failed to get workflow", "error", err, "id", workflowID)
		s.respondError(w, "Workflow not found", http.StatusNotFound)
		return
	}

	// Find nodes by name
	nodes := workflow.GetAllNodes()
	nodeNameToID := make(map[string]uuid.UUID)
	for _, node := range nodes {
		nodeNameToID[node.Name()] = node.ID()
	}

	fromID, ok := nodeNameToID[req.From]
	if !ok {
		s.respondError(w, fmt.Sprintf("Source node not found: %s", req.From), http.StatusBadRequest)
		return
	}

	toID, ok := nodeNameToID[req.To]
	if !ok {
		s.respondError(w, fmt.Sprintf("Target node not found: %s", req.To), http.StatusBadRequest)
		return
	}

	// Add edge to workflow
	edgeID, err := workflow.AddEdge(fromID, toID, domain.EdgeType(req.Type), req.Config)
	if err != nil {
		s.respondError(w, fmt.Sprintf("Failed to add edge: %v", err), http.StatusBadRequest)
		return
	}

	// Save workflow
	if err := s.store.SaveWorkflow(ctx, workflow); err != nil {
		s.logger.Error("failed to save workflow", "error", err)
		s.respondError(w, "Failed to save workflow", http.StatusInternalServerError)
		return
	}

	// Get the created edge
	edge, err := workflow.GetEdge(edgeID)
	if err != nil {
		s.respondError(w, "Failed to retrieve created edge", http.StatusInternalServerError)
		return
	}

	response := EdgeDetailResponse{
		ID:         edge.ID().String(),
		WorkflowID: workflowID.String(),
		From:       req.From,
		FromID:     edge.FromNodeID().String(),
		To:         req.To,
		ToID:       edge.ToNodeID().String(),
		Type:       edge.Type().String(),
		Config:     edge.Config(),
	}

	s.respondJSON(w, response, http.StatusCreated)
}

// handleUpdateEdge handles PUT /api/v1/workflows/{workflow_id}/edges/{edge_id}
func (s *Server) handleUpdateEdge(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	workflowID, err := uuid.Parse(r.PathValue("workflow_id"))
	if err != nil {
		s.respondError(w, "Invalid workflow ID", http.StatusBadRequest)
		return
	}

	edgeID, err := uuid.Parse(r.PathValue("edge_id"))
	if err != nil {
		s.respondError(w, "Invalid edge ID", http.StatusBadRequest)
		return
	}

	var req UpdateEdgeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.respondError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Get workflow
	workflow, err := s.store.GetWorkflow(ctx, workflowID)
	if err != nil {
		s.logger.Error("failed to get workflow", "error", err, "id", workflowID)
		s.respondError(w, "Workflow not found", http.StatusNotFound)
		return
	}

	// Get existing edge
	existingEdge, err := workflow.GetEdge(edgeID)
	if err != nil {
		s.respondError(w, "Edge not found", http.StatusNotFound)
		return
	}

	// Build node maps
	nodes := workflow.GetAllNodes()
	nodeNameToID := make(map[string]uuid.UUID)
	nodeIDToName := make(map[uuid.UUID]string)
	for _, node := range nodes {
		nodeNameToID[node.Name()] = node.ID()
		nodeIDToName[node.ID()] = node.Name()
	}

	// Determine new values (use existing if not provided)
	fromID := existingEdge.FromNodeID()
	fromName := nodeIDToName[fromID]
	if req.From != "" {
		var ok bool
		fromID, ok = nodeNameToID[req.From]
		if !ok {
			s.respondError(w, fmt.Sprintf("Source node not found: %s", req.From), http.StatusBadRequest)
			return
		}
		fromName = req.From
	}

	toID := existingEdge.ToNodeID()
	toName := nodeIDToName[toID]
	if req.To != "" {
		var ok bool
		toID, ok = nodeNameToID[req.To]
		if !ok {
			s.respondError(w, fmt.Sprintf("Target node not found: %s", req.To), http.StatusBadRequest)
			return
		}
		toName = req.To
	}

	edgeType := existingEdge.Type()
	if req.Type != "" {
		edgeType = domain.EdgeType(req.Type)
	}

	edgeConfig := existingEdge.Config()
	if req.Config != nil {
		edgeConfig = req.Config
	}

	// Remove old edge
	if err := workflow.RemoveEdge(edgeID); err != nil {
		s.respondError(w, fmt.Sprintf("Failed to remove old edge: %v", err), http.StatusInternalServerError)
		return
	}

	// Add updated edge
	newEdgeID, err := workflow.AddEdge(fromID, toID, edgeType, edgeConfig)
	if err != nil {
		s.respondError(w, fmt.Sprintf("Failed to add updated edge: %v", err), http.StatusBadRequest)
		return
	}

	// Save workflow
	if err := s.store.SaveWorkflow(ctx, workflow); err != nil {
		s.logger.Error("failed to save workflow", "error", err)
		s.respondError(w, "Failed to save workflow", http.StatusInternalServerError)
		return
	}

	// Get the updated edge
	edge, err := workflow.GetEdge(newEdgeID)
	if err != nil {
		s.respondError(w, "Failed to retrieve updated edge", http.StatusInternalServerError)
		return
	}

	response := EdgeDetailResponse{
		ID:         edge.ID().String(),
		WorkflowID: workflowID.String(),
		From:       fromName,
		FromID:     edge.FromNodeID().String(),
		To:         toName,
		ToID:       edge.ToNodeID().String(),
		Type:       edge.Type().String(),
		Config:     edge.Config(),
	}

	s.respondJSON(w, response, http.StatusOK)
}

// handleDeleteEdge handles DELETE /api/v1/workflows/{workflow_id}/edges/{edge_id}
func (s *Server) handleDeleteEdge(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	workflowID, err := uuid.Parse(r.PathValue("workflow_id"))
	if err != nil {
		s.respondError(w, "Invalid workflow ID", http.StatusBadRequest)
		return
	}

	edgeID, err := uuid.Parse(r.PathValue("edge_id"))
	if err != nil {
		s.respondError(w, "Invalid edge ID", http.StatusBadRequest)
		return
	}

	// Get workflow
	workflow, err := s.store.GetWorkflow(ctx, workflowID)
	if err != nil {
		s.logger.Error("failed to get workflow", "error", err, "id", workflowID)
		s.respondError(w, "Workflow not found", http.StatusNotFound)
		return
	}

	// Check if edge exists
	_, err = workflow.GetEdge(edgeID)
	if err != nil {
		s.respondError(w, "Edge not found", http.StatusNotFound)
		return
	}

	// Remove edge
	if err := workflow.RemoveEdge(edgeID); err != nil {
		s.respondError(w, fmt.Sprintf("Failed to remove edge: %v", err), http.StatusBadRequest)
		return
	}

	// Save workflow
	if err := s.store.SaveWorkflow(ctx, workflow); err != nil {
		s.logger.Error("failed to save workflow", "error", err)
		s.respondError(w, "Failed to save workflow", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// handleGetEdgeTypes handles GET /api/v1/edge-types
func (s *Server) handleGetEdgeTypes(w http.ResponseWriter, r *http.Request) {
	edgeTypes := []map[string]interface{}{
		{
			"type":        "direct",
			"name":        "Direct Edge",
			"description": "Simple sequential transition from one node to another",
			"category":    "basic",
			"example": map[string]interface{}{
				"from": "node-a",
				"to":   "node-b",
				"type": "direct",
			},
		},
		{
			"type":        "conditional",
			"name":        "Conditional Edge",
			"description": "Conditional transition based on expression evaluation",
			"category":    "control",
			"config_schema": map[string]interface{}{
				"expression": "string - condition expression (e.g., 'status == \"success\"')",
			},
			"example": map[string]interface{}{
				"from": "check-status",
				"to":   "success-path",
				"type": "conditional",
				"config": map[string]interface{}{
					"expression": "status == \"success\"",
				},
			},
		},
		{
			"type":        "fork",
			"name":        "Fork Edge",
			"description": "Start parallel execution branches",
			"category":    "parallel",
			"example": map[string]interface{}{
				"from": "fork-node",
				"to":   "branch-1",
				"type": "fork",
			},
		},
		{
			"type":        "join",
			"name":        "Join Edge",
			"description": "Synchronize parallel branches",
			"category":    "parallel",
			"example": map[string]interface{}{
				"from": "branch-1",
				"to":   "join-node",
				"type": "join",
			},
		},
	}

	s.respondJSON(w, edgeTypes, http.StatusOK)
}

// handleGetWorkflowGraph handles GET /api/v1/workflows/{workflow_id}/graph
func (s *Server) handleGetWorkflowGraph(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	workflowID, err := uuid.Parse(r.PathValue("workflow_id"))
	if err != nil {
		s.respondError(w, "Invalid workflow ID", http.StatusBadRequest)
		return
	}

	// Get workflow
	workflow, err := s.store.GetWorkflow(ctx, workflowID)
	if err != nil {
		s.logger.Error("failed to get workflow", "error", err, "id", workflowID)
		s.respondError(w, "Workflow not found", http.StatusNotFound)
		return
	}

	// Build graph representation
	nodes := workflow.GetAllNodes()
	edges := workflow.GetAllEdges()

	// Build node ID to name map
	nodeIDToName := make(map[uuid.UUID]string)
	nodesResponse := make([]map[string]interface{}, 0, len(nodes))
	for _, node := range nodes {
		nodeIDToName[node.ID()] = node.Name()
		nodesResponse = append(nodesResponse, map[string]interface{}{
			"id":   node.ID().String(),
			"name": node.Name(),
			"type": node.Type().String(),
		})
	}

	edgesResponse := make([]map[string]interface{}, 0, len(edges))
	for _, edge := range edges {
		edgesResponse = append(edgesResponse, map[string]interface{}{
			"id":      edge.ID().String(),
			"from":    nodeIDToName[edge.FromNodeID()],
			"from_id": edge.FromNodeID().String(),
			"to":      nodeIDToName[edge.ToNodeID()],
			"to_id":   edge.ToNodeID().String(),
			"type":    edge.Type().String(),
		})
	}

	response := map[string]interface{}{
		"workflow_id": workflowID.String(),
		"name":        workflow.Name(),
		"version":     workflow.Version(),
		"nodes":       nodesResponse,
		"edges":       edgesResponse,
	}

	s.respondJSON(w, response, http.StatusOK)
}
