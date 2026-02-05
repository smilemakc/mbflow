package rest

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/smilemakc/mbflow/internal/application/engine"
	"github.com/smilemakc/mbflow/internal/domain/repository"
	"github.com/smilemakc/mbflow/internal/infrastructure/logger"
	storagemodels "github.com/smilemakc/mbflow/internal/infrastructure/storage/models"
	"github.com/smilemakc/mbflow/pkg/models"
)

type ServiceAPIExecutionHandlers struct {
	executionRepo repository.ExecutionRepository
	workflowRepo  repository.WorkflowRepository
	executionMgr  *engine.ExecutionManager
	logger        *logger.Logger
}

func NewServiceAPIExecutionHandlers(
	executionRepo repository.ExecutionRepository,
	workflowRepo repository.WorkflowRepository,
	executionMgr *engine.ExecutionManager,
	log *logger.Logger,
) *ServiceAPIExecutionHandlers {
	return &ServiceAPIExecutionHandlers{
		executionRepo: executionRepo,
		workflowRepo:  workflowRepo,
		executionMgr:  executionMgr,
		logger:        log,
	}
}

func (h *ServiceAPIExecutionHandlers) ListExecutions(c *gin.Context) {
	limit := getQueryInt(c, "limit", 50)
	offset := getQueryInt(c, "offset", 0)
	workflowID := c.Query("workflow_id")
	status := c.Query("status")

	var execModels []*storagemodels.ExecutionModel
	var err error

	if workflowID != "" {
		wfUUID, parseErr := uuid.Parse(workflowID)
		if parseErr != nil {
			respondAPIError(c, ErrInvalidID)
			return
		}
		execModels, err = h.executionRepo.FindByWorkflowID(c.Request.Context(), wfUUID, limit, offset)
	} else if status != "" {
		execModels, err = h.executionRepo.FindByStatus(c.Request.Context(), status, limit, offset)
	} else {
		execModels, err = h.executionRepo.FindAll(c.Request.Context(), limit, offset)
	}

	if err != nil {
		h.logger.Error("Failed to list executions", "error", err, "workflow_id", workflowID, "status", status, "limit", limit, "offset", offset)
		respondAPIErrorWithRequestID(c, TranslateError(err))
		return
	}

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

func (h *ServiceAPIExecutionHandlers) GetExecution(c *gin.Context) {
	executionID, ok := getParam(c, "id")
	if !ok {
		return
	}

	execUUID, err := uuid.Parse(executionID)
	if err != nil {
		respondAPIError(c, ErrInvalidID)
		return
	}

	execModel, err := h.executionRepo.FindByIDWithRelations(c.Request.Context(), execUUID)
	if err != nil {
		h.logger.Error("Failed to find execution", "error", err, "execution_id", execUUID)
		respondAPIErrorWithRequestID(c, TranslateError(err))
		return
	}

	execution := engine.ExecutionModelToDomain(execModel)

	workflowModel, err := h.workflowRepo.FindByIDWithRelations(c.Request.Context(), execModel.WorkflowID)
	if err == nil && workflowModel != nil {
		nodeIDMap := make(map[string]string)
		nodeNameMap := make(map[string]string)
		nodeTypeMap := make(map[string]string)
		for _, node := range workflowModel.Nodes {
			nodeIDMap[node.ID.String()] = node.NodeID
			nodeNameMap[node.ID.String()] = node.Name
			nodeTypeMap[node.ID.String()] = node.Type
		}

		for _, ne := range execution.NodeExecutions {
			if logicalID, found := nodeIDMap[ne.NodeID]; found {
				ne.NodeID = logicalID
			}
			if nodeName, found := nodeNameMap[ne.NodeID]; found {
				ne.NodeName = nodeName
			} else if ne.NodeID != "" {
				for _, node := range workflowModel.Nodes {
					if node.NodeID == ne.NodeID {
						ne.NodeName = node.Name
						ne.NodeType = node.Type
						break
					}
				}
			}
			if nodeType, found := nodeTypeMap[ne.NodeID]; found {
				ne.NodeType = nodeType
			}
		}
	}

	respondJSON(c, http.StatusOK, execution)
}

func (h *ServiceAPIExecutionHandlers) StartExecution(c *gin.Context) {
	workflowID, ok := getParam(c, "id")
	if !ok {
		return
	}

	var req struct {
		Input map[string]any `json:"input"`
	}

	if err := bindJSON(c, &req); err != nil {
		return
	}

	opts := engine.DefaultExecutionOptions()
	execution, err := h.executionMgr.ExecuteAsync(c.Request.Context(), workflowID, req.Input, opts)
	if err != nil {
		h.logger.Error("Failed to start workflow execution", "error", err, "workflow_id", workflowID)
		respondAPIErrorWithRequestID(c, TranslateError(err))
		return
	}

	h.logger.Info("Workflow execution started via service API", "execution_id", execution.ID, "workflow_id", workflowID)
	respondJSON(c, http.StatusAccepted, execution)
}

func (h *ServiceAPIExecutionHandlers) CancelExecution(c *gin.Context) {
	respondAPIError(c, NewAPIError("NOT_IMPLEMENTED", "execution cancellation not yet implemented", http.StatusNotImplemented))
}

func (h *ServiceAPIExecutionHandlers) RetryExecution(c *gin.Context) {
	respondAPIError(c, NewAPIError("NOT_IMPLEMENTED", "execution retry not yet implemented", http.StatusNotImplemented))
}
