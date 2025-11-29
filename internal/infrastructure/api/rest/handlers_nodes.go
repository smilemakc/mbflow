package rest

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/google/uuid"
	"github.com/smilemakc/mbflow/internal/domain"
)

// NodeDetailResponse represents a detailed node response
type NodeDetailResponse struct {
	ID          string                 `json:"id"`
	WorkflowID  string                 `json:"workflow_id"`
	Type        string                 `json:"type"`
	Name        string                 `json:"name"`
	Config      map[string]interface{} `json:"config,omitempty"`
	Description string                 `json:"description,omitempty"`
}

// CreateNodeRequest represents the request to create a node
type CreateNodeRequest struct {
	Type   string                 `json:"type"`
	Name   string                 `json:"name"`
	Config map[string]interface{} `json:"config,omitempty"`
}

// UpdateNodeRequest represents the request to update a node
type UpdateNodeRequest struct {
	Type   string                 `json:"type,omitempty"`
	Name   string                 `json:"name,omitempty"`
	Config map[string]interface{} `json:"config,omitempty"`
}

// handleListNodes handles GET /api/v1/workflows/{workflow_id}/nodes
func (s *Server) handleListNodes(w http.ResponseWriter, r *http.Request) {
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

	// Get all nodes
	nodes := workflow.GetAllNodes()
	response := make([]NodeDetailResponse, 0, len(nodes))
	for _, node := range nodes {
		response = append(response, NodeDetailResponse{
			ID:         node.ID().String(),
			WorkflowID: workflowID.String(),
			Type:       node.Type().String(),
			Name:       node.Name(),
			Config:     node.Config(),
		})
	}

	s.respondJSON(w, response, http.StatusOK)
}

// handleGetNode handles GET /api/v1/workflows/{workflow_id}/nodes/{node_id}
func (s *Server) handleGetNode(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	workflowID, err := uuid.Parse(r.PathValue("workflow_id"))
	if err != nil {
		s.respondError(w, "Invalid workflow ID", http.StatusBadRequest)
		return
	}

	nodeID, err := uuid.Parse(r.PathValue("node_id"))
	if err != nil {
		s.respondError(w, "Invalid node ID", http.StatusBadRequest)
		return
	}

	// Get workflow
	workflow, err := s.store.GetWorkflow(ctx, workflowID)
	if err != nil {
		s.logger.Error("failed to get workflow", "error", err, "id", workflowID)
		s.respondError(w, "Workflow not found", http.StatusNotFound)
		return
	}

	// Get node
	node, err := workflow.GetNode(nodeID)
	if err != nil {
		s.respondError(w, "Node not found", http.StatusNotFound)
		return
	}

	response := NodeDetailResponse{
		ID:         node.ID().String(),
		WorkflowID: workflowID.String(),
		Type:       node.Type().String(),
		Name:       node.Name(),
		Config:     node.Config(),
	}

	s.respondJSON(w, response, http.StatusOK)
}

// handleCreateNode handles POST /api/v1/workflows/{workflow_id}/nodes
func (s *Server) handleCreateNode(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	workflowID, err := uuid.Parse(r.PathValue("workflow_id"))
	if err != nil {
		s.respondError(w, "Invalid workflow ID", http.StatusBadRequest)
		return
	}

	var req CreateNodeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.respondError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate request
	if req.Type == "" {
		s.respondError(w, "Node type is required", http.StatusBadRequest)
		return
	}
	if req.Name == "" {
		s.respondError(w, "Node name is required", http.StatusBadRequest)
		return
	}

	// Get workflow
	workflow, err := s.store.GetWorkflow(ctx, workflowID)
	if err != nil {
		s.logger.Error("failed to get workflow", "error", err, "id", workflowID)
		s.respondError(w, "Workflow not found", http.StatusNotFound)
		return
	}

	// Add node to workflow
	nodeID, err := workflow.AddNode(domain.NodeType(req.Type), req.Name, req.Config)
	if err != nil {
		s.respondError(w, fmt.Sprintf("Failed to add node: %v", err), http.StatusBadRequest)
		return
	}

	// Save workflow
	if err := s.store.SaveWorkflow(ctx, workflow); err != nil {
		s.logger.Error("failed to save workflow", "error", err)
		s.respondError(w, "Failed to save workflow", http.StatusInternalServerError)
		return
	}

	// Get the created node
	node, err := workflow.GetNode(nodeID)
	if err != nil {
		s.respondError(w, "Failed to retrieve created node", http.StatusInternalServerError)
		return
	}

	response := NodeDetailResponse{
		ID:         node.ID().String(),
		WorkflowID: workflowID.String(),
		Type:       node.Type().String(),
		Name:       node.Name(),
		Config:     node.Config(),
	}

	s.respondJSON(w, response, http.StatusCreated)
}

// handleUpdateNode handles PUT /api/v1/workflows/{workflow_id}/nodes/{node_id}
func (s *Server) handleUpdateNode(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	workflowID, err := uuid.Parse(r.PathValue("workflow_id"))
	if err != nil {
		s.respondError(w, "Invalid workflow ID", http.StatusBadRequest)
		return
	}

	nodeID, err := uuid.Parse(r.PathValue("node_id"))
	if err != nil {
		s.respondError(w, "Invalid node ID", http.StatusBadRequest)
		return
	}

	var req UpdateNodeRequest
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

	// Get existing node
	existingNode, err := workflow.GetNode(nodeID)
	if err != nil {
		s.respondError(w, "Node not found", http.StatusNotFound)
		return
	}

	// Determine new values (use existing if not provided)
	nodeType := existingNode.Type()
	if req.Type != "" {
		nodeType = domain.NodeType(req.Type)
	}

	nodeName := existingNode.Name()
	if req.Name != "" {
		nodeName = req.Name
	}

	nodeConfig := existingNode.Config()
	if req.Config != nil {
		nodeConfig = req.Config
	}

	// Remove old node
	if err := workflow.RemoveNode(nodeID); err != nil {
		s.respondError(w, fmt.Sprintf("Failed to remove old node: %v", err), http.StatusInternalServerError)
		return
	}

	// Add updated node with the same ID
	// Note: This is a workaround since the domain doesn't support direct update
	// In a real implementation, you might want to add an UpdateNode method to the Workflow interface
	newNodeID, err := workflow.AddNode(nodeType, nodeName, nodeConfig)
	if err != nil {
		s.respondError(w, fmt.Sprintf("Failed to add updated node: %v", err), http.StatusBadRequest)
		return
	}

	// Save workflow
	if err := s.store.SaveWorkflow(ctx, workflow); err != nil {
		s.logger.Error("failed to save workflow", "error", err)
		s.respondError(w, "Failed to save workflow", http.StatusInternalServerError)
		return
	}

	// Get the updated node
	node, err := workflow.GetNode(newNodeID)
	if err != nil {
		s.respondError(w, "Failed to retrieve updated node", http.StatusInternalServerError)
		return
	}

	response := NodeDetailResponse{
		ID:         node.ID().String(),
		WorkflowID: workflowID.String(),
		Type:       node.Type().String(),
		Name:       node.Name(),
		Config:     node.Config(),
	}

	s.respondJSON(w, response, http.StatusOK)
}

// handleDeleteNode handles DELETE /api/v1/workflows/{workflow_id}/nodes/{node_id}
func (s *Server) handleDeleteNode(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	workflowID, err := uuid.Parse(r.PathValue("workflow_id"))
	if err != nil {
		s.respondError(w, "Invalid workflow ID", http.StatusBadRequest)
		return
	}

	nodeID, err := uuid.Parse(r.PathValue("node_id"))
	if err != nil {
		s.respondError(w, "Invalid node ID", http.StatusBadRequest)
		return
	}

	// Get workflow
	workflow, err := s.store.GetWorkflow(ctx, workflowID)
	if err != nil {
		s.logger.Error("failed to get workflow", "error", err, "id", workflowID)
		s.respondError(w, "Workflow not found", http.StatusNotFound)
		return
	}

	// Check if node exists
	_, err = workflow.GetNode(nodeID)
	if err != nil {
		s.respondError(w, "Node not found", http.StatusNotFound)
		return
	}

	// Remove node
	if err := workflow.RemoveNode(nodeID); err != nil {
		s.respondError(w, fmt.Sprintf("Failed to remove node: %v", err), http.StatusBadRequest)
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

// handleGetNodeTypes handles GET /api/v1/node-types
func (s *Server) handleGetNodeTypes(w http.ResponseWriter, r *http.Request) {
	nodeTypes := []map[string]interface{}{
		{
			"type":        "transform",
			"name":        "Transform",
			"description": "Transform data using expressions",
			"category":    "data",
			"config_schema": map[string]interface{}{
				"transformations": "map[string]string - expressions to evaluate",
			},
		},
		{
			"type":        "http",
			"name":        "HTTP Request",
			"description": "Make HTTP requests",
			"category":    "integration",
			"config_schema": map[string]interface{}{
				"url":    "string - target URL",
				"method": "string - HTTP method (GET, POST, etc.)",
			},
		},
		{
			"type":        "conditional-route",
			"name":        "Conditional Router",
			"description": "Route execution based on conditions",
			"category":    "control",
		},
		{
			"type":        "parallel",
			"name":        "Parallel",
			"description": "Execute multiple branches in parallel",
			"category":    "control",
			"config_schema": map[string]interface{}{
				"join_strategy": "string - wait_all, wait_any, wait_first, wait_n",
			},
		},
		{
			"type":        "openai-completion",
			"name":        "OpenAI Completion",
			"description": "Call OpenAI API for text completion",
			"category":    "ai",
			"config_schema": map[string]interface{}{
				"api_key": "string - OpenAI API key",
				"model":   "string - model name (gpt-4, gpt-3.5-turbo, etc.)",
				"prompt":  "string - prompt template",
			},
		},
		{
			"type":        "json-parser",
			"name":        "JSON Parser",
			"description": "Parse JSON data",
			"category":    "data",
		},
		{
			"type":        "telegram-message",
			"name":        "Telegram Message",
			"description": "Send message via Telegram",
			"category":    "integration",
			"config_schema": map[string]interface{}{
				"bot_token": "string - Telegram bot token",
				"chat_id":   "string - target chat ID",
				"message":   "string - message template",
			},
		},
	}

	s.respondJSON(w, nodeTypes, http.StatusOK)
}
