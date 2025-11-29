package rest

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/smilemakc/mbflow"
	"github.com/smilemakc/mbflow/internal/domain"
)

// ExecuteWorkflowRequest represents the request to execute a workflow
type ExecuteWorkflowRequest struct {
	WorkflowID string                 `json:"workflow_id"`
	TriggerID  string                 `json:"trigger_id,omitempty"`
	Variables  map[string]interface{} `json:"variables,omitempty"`
}

// ExecutionResponse represents the response for an execution
type ExecutionResponse struct {
	ID             string                 `json:"id"`
	WorkflowID     string                 `json:"workflow_id"`
	Status         string                 `json:"status"`
	Phase          string                 `json:"phase"`
	Variables      map[string]interface{} `json:"variables,omitempty"`
	NodeStates     []NodeStateResponse    `json:"node_states,omitempty"`
	StartedAt      *time.Time             `json:"started_at,omitempty"`
	CompletedAt    *time.Time             `json:"completed_at,omitempty"`
	Duration       int64                  `json:"duration_ms,omitempty"`
	Error          string                 `json:"error,omitempty"`
	CurrentNodeID  string                 `json:"current_node_id,omitempty"`
	SequenceNumber int                    `json:"sequence_number"`
}

// NodeStateResponse represents the state of a node in an execution
type NodeStateResponse struct {
	NodeID      string                 `json:"node_id"`
	NodeName    string                 `json:"node_name"`
	Status      string                 `json:"status"`
	StartedAt   *time.Time             `json:"started_at,omitempty"`
	CompletedAt *time.Time             `json:"completed_at,omitempty"`
	Duration    int64                  `json:"duration_ms,omitempty"`
	Output      map[string]interface{} `json:"output,omitempty"`
	Error       string                 `json:"error,omitempty"`
	RetryCount  int                    `json:"retry_count"`
}

// ExecutionEventsResponse represents the events for an execution
type ExecutionEventsResponse struct {
	ExecutionID string          `json:"execution_id"`
	Events      []EventResponse `json:"events"`
}

// EventResponse represents a domain event
type EventResponse struct {
	ID          string                 `json:"id"`
	EventType   string                 `json:"event_type"`
	ExecutionID string                 `json:"execution_id"`
	WorkflowID  string                 `json:"workflow_id,omitempty"`
	Sequence    int                    `json:"sequence"`
	Timestamp   time.Time              `json:"timestamp"`
	Data        map[string]interface{} `json:"data,omitempty"`
}

// handleListExecutions handles GET /api/v1/executions
func (s *Server) handleListExecutions(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Parse query parameters for filtering
	workflowIDStr := r.URL.Query().Get("workflow_id")
	statusFilter := r.URL.Query().Get("status")

	var executions []domain.Execution
	var err error

	if workflowIDStr != "" {
		workflowID, parseErr := uuid.Parse(workflowIDStr)
		if parseErr != nil {
			s.respondError(w, "Invalid workflow ID", http.StatusBadRequest)
			return
		}
		executions, err = s.store.ListExecutionsByWorkflow(ctx, workflowID)
	} else {
		executions, err = s.store.ListAllExecutions(ctx, 100, 0)
	}

	if err != nil {
		s.logger.Error("failed to list executions", "error", err)
		s.respondError(w, "Failed to list executions", http.StatusInternalServerError)
		return
	}

	// Filter by status if needed
	filtered := make([]domain.Execution, 0)
	for _, exec := range executions {
		if statusFilter != "" && string(exec.Phase()) != statusFilter {
			continue
		}
		filtered = append(filtered, exec)
	}

	response := make([]ExecutionResponse, 0, len(filtered))
	for _, exec := range filtered {
		response = append(response, s.executionToResponse(exec))
	}

	s.respondJSON(w, response, http.StatusOK)
}

// handleGetExecution handles GET /api/v1/executions/{id}
func (s *Server) handleGetExecution(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	executionID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		s.respondError(w, "Invalid execution ID", http.StatusBadRequest)
		return
	}

	execution, err := s.store.GetExecution(ctx, executionID)
	if err != nil {
		s.logger.Error("failed to get execution", "error", err, "id", executionID)
		s.respondError(w, "Execution not found", http.StatusNotFound)
		return
	}

	s.respondJSON(w, s.executionToResponse(execution), http.StatusOK)
}

// handleExecuteWorkflow handles POST /api/v1/executions
func (s *Server) handleExecuteWorkflow(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req ExecuteWorkflowRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.respondError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate request
	if req.WorkflowID == "" {
		s.respondError(w, "Workflow ID is required", http.StatusBadRequest)
		return
	}

	workflowID, err := uuid.Parse(req.WorkflowID)
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

	// Determine trigger
	var trigger domain.Trigger
	if req.TriggerID != "" {
		triggerID, err := uuid.Parse(req.TriggerID)
		if err != nil {
			s.respondError(w, "Invalid trigger ID", http.StatusBadRequest)
			return
		}

		// Find trigger in workflow
		trigger, err = workflow.GetTrigger(triggerID)
		if err != nil {
			s.respondError(w, "Trigger not found in workflow", http.StatusBadRequest)
			return
		}
	} else {
		// Use first manual trigger or create one
		triggers := workflow.GetAllTriggers()
		for _, t := range triggers {
			if t.Type() == domain.TriggerTypeManual {
				trigger = t
				break
			}
		}
		if trigger == nil {
			s.respondError(w, "No manual trigger found in workflow", http.StatusBadRequest)
			return
		}
	}

	// Execute workflow
	execution, err := s.executor.ExecuteWorkflow(ctx, workflow, trigger, req.Variables)
	if err != nil {
		s.logger.Error("failed to execute workflow", "error", err)
		s.respondError(w, fmt.Sprintf("Failed to execute workflow: %v", err), http.StatusInternalServerError)
		return
	}

	// Save execution snapshot to store
	if err := s.store.SaveSnapshot(ctx, execution); err != nil {
		s.logger.Error("failed to save execution snapshot", "error", err)
		// Don't fail the request - execution completed successfully
	}

	s.respondJSON(w, s.executionToResponse(execution), http.StatusCreated)
}

// handleGetExecutionEvents handles GET /api/v1/executions/{id}/events
func (s *Server) handleGetExecutionEvents(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	executionID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		s.respondError(w, "Invalid execution ID", http.StatusBadRequest)
		return
	}

	events, err := s.executor.EventStore().GetEvents(ctx, executionID)
	if err != nil {
		s.logger.Error("failed to get execution events", "error", err, "id", executionID)
		s.respondError(w, "Failed to get execution events", http.StatusInternalServerError)
		return
	}

	response := ExecutionEventsResponse{
		ExecutionID: executionID.String(),
		Events:      make([]EventResponse, 0, len(events)),
	}

	for _, event := range events {
		response.Events = append(response.Events, s.eventToResponse(event))
	}

	s.respondJSON(w, response, http.StatusOK)
}

// handleCancelExecution handles POST /api/v1/executions/{id}/cancel
func (s *Server) handleCancelExecution(w http.ResponseWriter, r *http.Request) {
	_, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		s.respondError(w, "Invalid execution ID", http.StatusBadRequest)
		return
	}

	s.respondError(w, "Cancel execution not yet implemented", http.StatusNotImplemented)
}

// handlePauseExecution handles POST /api/v1/executions/{id}/pause
func (s *Server) handlePauseExecution(w http.ResponseWriter, r *http.Request) {
	_, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		s.respondError(w, "Invalid execution ID", http.StatusBadRequest)
		return
	}

	s.respondError(w, "Pause execution not yet implemented", http.StatusNotImplemented)
}

// handleResumeExecution handles POST /api/v1/executions/{id}/resume
func (s *Server) handleResumeExecution(w http.ResponseWriter, r *http.Request) {
	_, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		s.respondError(w, "Invalid execution ID", http.StatusBadRequest)
		return
	}

	s.respondError(w, "Resume execution not yet implemented", http.StatusNotImplemented)
}

// executionToResponse converts a domain execution to a response DTO
func (s *Server) executionToResponse(exec domain.Execution) ExecutionResponse {
	nodeStates := make([]NodeStateResponse, 0)
	for nodeID, state := range exec.GetAllNodeStates() {
		nodeState := NodeStateResponse{
			NodeID:     nodeID.String(),
			NodeName:   state.NodeName(),
			Status:     state.Status().String(),
			RetryCount: state.RetryCount(),
		}

		if startedAt := state.StartedAt(); startedAt != nil && !startedAt.IsZero() {
			nodeState.StartedAt = startedAt
		}
		if finishedAt := state.FinishedAt(); finishedAt != nil && !finishedAt.IsZero() {
			nodeState.CompletedAt = finishedAt
			duration := finishedAt.Sub(*state.StartedAt()).Milliseconds()
			nodeState.Duration = duration
		}
		if output := state.Output(); output != nil {
			nodeState.Output = output
		}
		if err := state.Error(); err != "" {
			nodeState.Error = err
		}

		nodeStates = append(nodeStates, nodeState)
	}

	response := ExecutionResponse{
		ID:         exec.ID().String(),
		WorkflowID: exec.WorkflowID().String(),
		Phase:      string(exec.Phase()),
		NodeStates: nodeStates,
	}

	startedAt := exec.StartedAt()
	if !startedAt.IsZero() {
		response.StartedAt = &startedAt
	}

	finishedAt := exec.FinishedAt()
	if finishedAt != nil && !finishedAt.IsZero() {
		response.CompletedAt = finishedAt
		duration := finishedAt.Sub(startedAt).Milliseconds()
		response.Duration = duration
	}

	if exec.HasError() {
		errMsg := exec.Error()
		response.Error = errMsg
	}

	variables := exec.Variables()
	if variables != nil && len(variables.All()) > 0 {
		response.Variables = variables.All()
	}

	return response
}

// eventToResponse converts a domain event to a response DTO
func (s *Server) eventToResponse(event mbflow.Event) EventResponse {
	return EventResponse{
		ID:          event.EventID().String(),
		EventType:   string(event.EventType()),
		ExecutionID: event.ExecutionID().String(),
		WorkflowID:  event.WorkflowID().String(),
		Sequence:    int(event.SequenceNumber()),
		Timestamp:   event.Timestamp(),
		Data:        event.Data(),
	}
}
