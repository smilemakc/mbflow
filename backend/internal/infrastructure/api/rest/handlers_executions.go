package rest

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/smilemakc/mbflow/internal/application/engine"
	"github.com/smilemakc/mbflow/internal/domain/repository"
	storagemodels "github.com/smilemakc/mbflow/internal/infrastructure/storage/models"
	"github.com/smilemakc/mbflow/pkg/models"
)

// ExecutionHandlers provides HTTP handlers for execution-related endpoints
type ExecutionHandlers struct {
	executionRepo    repository.ExecutionRepository
	workflowRepo     repository.WorkflowRepository
	executionManager *engine.ExecutionManager
}

// NewExecutionHandlers creates a new ExecutionHandlers instance
func NewExecutionHandlers(
	executionRepo repository.ExecutionRepository,
	workflowRepo repository.WorkflowRepository,
	executionManager *engine.ExecutionManager,
) *ExecutionHandlers {
	return &ExecutionHandlers{
		executionRepo:    executionRepo,
		workflowRepo:     workflowRepo,
		executionManager: executionManager,
	}
}

// HandleRunExecution handles POST /api/v1/executions
func (h *ExecutionHandlers) HandleRunExecution(c *gin.Context) {
	var req struct {
		WorkflowID string                 `json:"workflow_id"`
		Input      map[string]interface{} `json:"input"`
		Async      bool                   `json:"async"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.WorkflowID == "" {
		respondError(c, http.StatusBadRequest, "workflow_id is required")
		return
	}

	// Execute workflow
	opts := engine.DefaultExecutionOptions()
	execution, err := h.executionManager.Execute(c.Request.Context(), req.WorkflowID, req.Input, opts)
	if err != nil {
		respondError(c, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(c, http.StatusCreated, execution)
}

// HandleGetExecution handles GET /api/v1/executions/{id}
func (h *ExecutionHandlers) HandleGetExecution(c *gin.Context) {
	executionID := c.Param("id")
	if executionID == "" {
		respondError(c, http.StatusBadRequest, "execution ID is required")
		return
	}

	execUUID, err := uuid.Parse(executionID)
	if err != nil {
		respondError(c, http.StatusBadRequest, "invalid execution ID")
		return
	}

	execModel, err := h.executionRepo.FindByIDWithRelations(c.Request.Context(), execUUID)
	if err != nil {
		respondError(c, http.StatusNotFound, "execution not found")
		return
	}

	execution := engine.ExecutionModelToDomain(execModel)
	respondJSON(c, http.StatusOK, execution)
}

// HandleListExecutions handles GET /api/v1/executions
func (h *ExecutionHandlers) HandleListExecutions(c *gin.Context) {
	limit := getQueryInt(c, "limit", 50)
	offset := getQueryInt(c, "offset", 0)
	workflowID := c.Query("workflow_id")
	status := c.Query("status")

	var execModels []*storagemodels.ExecutionModel
	var err error

	if workflowID != "" {
		wfUUID, parseErr := uuid.Parse(workflowID)
		if parseErr != nil {
			respondError(c, http.StatusBadRequest, "invalid workflow_id")
			return
		}
		execModels, err = h.executionRepo.FindByWorkflowID(c.Request.Context(), wfUUID, limit, offset)
	} else if status != "" {
		execModels, err = h.executionRepo.FindByStatus(c.Request.Context(), status, limit, offset)
	} else {
		execModels, err = h.executionRepo.FindAll(c.Request.Context(), limit, offset)
	}

	if err != nil {
		respondError(c, http.StatusInternalServerError, err.Error())
		return
	}

	// Convert to domain models
	executions := make([]*models.Execution, len(execModels))
	for i, em := range execModels {
		executions[i] = engine.ExecutionModelToDomain(em)
	}

	c.JSON(http.StatusOK, gin.H{
		"executions": executions,
		"total":      len(executions),
		"limit":      limit,
		"offset":     offset,
	})
}

// HandleGetLogs handles GET /api/v1/executions/{id}/logs
func (h *ExecutionHandlers) HandleGetLogs(c *gin.Context) {
	executionID := c.Param("id")
	if executionID == "" {
		respondError(c, http.StatusBadRequest, "execution ID is required")
		return
	}

	// For MVP, return empty logs since EventRepository is deferred
	// In full implementation, this would query eventRepo.FindByExecutionID()
	c.JSON(http.StatusOK, gin.H{
		"logs":  []interface{}{},
		"total": 0,
	})
}

// HandleGetNodeResult handles GET /api/v1/executions/{id}/nodes/{nodeId}
func (h *ExecutionHandlers) HandleGetNodeResult(c *gin.Context) {
	executionID := c.Param("id")
	nodeID := c.Param("nodeId")

	if executionID == "" || nodeID == "" {
		respondError(c, http.StatusBadRequest, "execution ID and node ID are required")
		return
	}

	execUUID, err := uuid.Parse(executionID)
	if err != nil {
		respondError(c, http.StatusBadRequest, "invalid execution ID")
		return
	}

	execModel, err := h.executionRepo.FindByIDWithRelations(c.Request.Context(), execUUID)
	if err != nil {
		respondError(c, http.StatusNotFound, "execution not found")
		return
	}

	workflowModel, err := h.workflowRepo.FindByIDWithRelations(c.Request.Context(), execModel.WorkflowID)
	if err != nil {
		respondError(c, http.StatusNotFound, "workflow not found")
		return
	}

	// Build node ID mapping (UUID -> logical ID)
	nodeIDMap := make(map[uuid.UUID]string)
	for _, node := range workflowModel.Nodes {
		nodeIDMap[node.ID] = node.NodeID
	}

	// Find matching node execution
	for _, ne := range execModel.NodeExecutions {
		if logicalID, ok := nodeIDMap[ne.NodeID]; ok && logicalID == nodeID {
			nodeExec := engine.NodeExecutionModelToDomain(ne)
			nodeExec.NodeID = nodeID // Replace UUID with logical ID
			respondJSON(c, http.StatusOK, nodeExec)
			return
		}
	}

	respondError(c, http.StatusNotFound, "node execution not found")
}

// HandleCancelExecution handles POST /api/v1/executions/{id}/cancel (deferred)
func (h *ExecutionHandlers) HandleCancelExecution(c *gin.Context) {
	respondError(c, http.StatusNotImplemented, "execution cancellation not yet implemented")
}

// HandleRetryExecution handles POST /api/v1/executions/{id}/retry (deferred)
func (h *ExecutionHandlers) HandleRetryExecution(c *gin.Context) {
	respondError(c, http.StatusNotImplemented, "execution retry not yet implemented")
}

// HandleWatchExecution handles GET /api/v1/executions/{id}/watch (deferred)
func (h *ExecutionHandlers) HandleWatchExecution(c *gin.Context) {
	respondError(c, http.StatusNotImplemented, "real-time execution watching not yet implemented")
}

// HandleStreamLogs handles GET /api/v1/executions/{id}/logs/stream (deferred)
func (h *ExecutionHandlers) HandleStreamLogs(c *gin.Context) {
	respondError(c, http.StatusNotImplemented, "log streaming not yet implemented")
}
