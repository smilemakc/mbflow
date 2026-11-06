package rest

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/smilemakc/mbflow/go/internal/application/importer"
	"github.com/smilemakc/mbflow/go/internal/domain/repository"
	"github.com/smilemakc/mbflow/go/internal/infrastructure/logger"
	storagemodels "github.com/smilemakc/mbflow/go/internal/infrastructure/storage/models"
	"github.com/smilemakc/mbflow/go/pkg/executor"
	pkgmodels "github.com/smilemakc/mbflow/go/pkg/models"
)

// ImportHandlers provides HTTP handlers for workflow import/export endpoints.
type ImportHandlers struct {
	workflowRepo    repository.WorkflowRepository
	triggerRepo     repository.TriggerRepository
	logger          *logger.Logger
	executorManager executor.Manager
	importer        *importer.YAMLImporter
}

// NewImportHandlers creates a new ImportHandlers instance.
func NewImportHandlers(
	workflowRepo repository.WorkflowRepository,
	triggerRepo repository.TriggerRepository,
	log *logger.Logger,
	executorManager executor.Manager,
) *ImportHandlers {
	return &ImportHandlers{
		workflowRepo:    workflowRepo,
		triggerRepo:     triggerRepo,
		logger:          log,
		executorManager: executorManager,
		importer:        importer.NewYAMLImporter(executorManager),
	}
}

// ImportResponse represents the response from importing a workflow.
type ImportResponse struct {
	WorkflowID string  `json:"workflow_id"`
	Name       string  `json:"name"`
	Status     string  `json:"status"`
	NodesCount int     `json:"nodes_count"`
	EdgesCount int     `json:"edges_count"`
	TriggerID  *string `json:"trigger_id,omitempty"`
}

// HandleImportWorkflow handles POST /api/v1/workflows/import
// Accepts YAML via multipart form file upload or raw YAML body.
func (h *ImportHandlers) HandleImportWorkflow(c *gin.Context) {
	var yamlData []byte
	var err error

	contentType := c.GetHeader("Content-Type")

	switch {
	case strings.HasPrefix(contentType, "multipart/form-data"):
		// Handle multipart form file upload
		yamlData, err = h.handleMultipartUpload(c)
		if err != nil {
			return // Error already responded
		}
	case strings.Contains(contentType, "yaml") || strings.Contains(contentType, "text/plain"):
		// Handle raw YAML body
		yamlData, err = io.ReadAll(c.Request.Body)
		if err != nil {
			h.logger.Error("Failed to read request body", "error", err, "request_id", GetRequestID(c))
			respondAPIError(c, NewAPIError("READ_ERROR", "Failed to read request body", http.StatusBadRequest))
			return
		}
	default:
		respondAPIError(c, NewAPIError("INVALID_CONTENT_TYPE",
			"Content-Type must be multipart/form-data, application/x-yaml, or text/yaml", http.StatusBadRequest))
		return
	}

	if len(yamlData) == 0 {
		respondAPIError(c, NewAPIError("EMPTY_CONTENT", "No YAML content provided", http.StatusBadRequest))
		return
	}

	// Parse YAML content
	cleanData, err := importer.ParseYAMLContent(yamlData)
	if err != nil {
		h.logger.Error("Failed to parse YAML content", "error", err, "request_id", GetRequestID(c))
		respondAPIError(c, NewAPIError("PARSE_ERROR", err.Error(), http.StatusBadRequest))
		return
	}

	// Import workflow
	result, err := h.importer.ImportFromYAML(cleanData)
	if err != nil {
		h.logger.Error("Failed to import workflow", "error", err, "request_id", GetRequestID(c))
		respondAPIError(c, NewAPIError("IMPORT_ERROR", err.Error(), http.StatusBadRequest))
		return
	}

	// Convert to storage model and save
	workflowModel, err := h.saveWorkflow(c, result)
	if err != nil {
		return // Error already responded
	}

	// Save trigger if present
	var triggerID *string
	if result.Trigger != nil {
		tid, err := h.saveTrigger(c, result, workflowModel.ID)
		if err != nil {
			// Rollback workflow creation
			_ = h.workflowRepo.HardDelete(c.Request.Context(), workflowModel.ID)
			return // Error already responded
		}
		tidStr := tid.String()
		triggerID = &tidStr
	}

	// Return response
	response := ImportResponse{
		WorkflowID: workflowModel.ID.String(),
		Name:       workflowModel.Name,
		Status:     workflowModel.Status,
		NodesCount: result.NodesCount,
		EdgesCount: result.EdgesCount,
		TriggerID:  triggerID,
	}

	h.logger.Info("Workflow imported successfully",
		"workflow_id", workflowModel.ID,
		"workflow_name", workflowModel.Name,
		"nodes_count", result.NodesCount,
		"edges_count", result.EdgesCount,
		"request_id", GetRequestID(c))

	respondJSON(c, http.StatusCreated, response)
}

// handleMultipartUpload handles file upload from multipart form.
func (h *ImportHandlers) handleMultipartUpload(c *gin.Context) ([]byte, error) {
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		h.logger.Error("Failed to get file from form", "error", err, "request_id", GetRequestID(c))
		respondAPIError(c, NewAPIError("FILE_REQUIRED", "File is required in 'file' field", http.StatusBadRequest))
		return nil, err
	}
	defer func() { _ = file.Close() }()

	// Validate file extension
	filename := strings.ToLower(header.Filename)
	if !strings.HasSuffix(filename, ".yaml") && !strings.HasSuffix(filename, ".yml") {
		respondAPIError(c, NewAPIError("INVALID_FILE_TYPE", "File must have .yaml or .yml extension", http.StatusBadRequest))
		return nil, fmt.Errorf("invalid file type")
	}

	// Read file content
	data, err := io.ReadAll(file)
	if err != nil {
		h.logger.Error("Failed to read file", "error", err, "filename", header.Filename, "request_id", GetRequestID(c))
		respondAPIError(c, NewAPIError("READ_ERROR", "Failed to read file", http.StatusBadRequest))
		return nil, err
	}

	return data, nil
}

// saveWorkflow converts the import result to storage model and saves it.
func (h *ImportHandlers) saveWorkflow(c *gin.Context, result *importer.ImportResult) (*storagemodels.WorkflowModel, error) {
	workflow := result.Workflow
	now := time.Now()

	workflowModel := &storagemodels.WorkflowModel{
		ID:          uuid.New(),
		Name:        workflow.Name,
		Description: workflow.Description,
		Status:      "draft",
		Version:     workflow.Version,
		Variables:   storagemodels.JSONBMap(workflow.Variables),
		Metadata:    storagemodels.JSONBMap(workflow.Metadata),
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	// Set created_by if user is authenticated
	if userID, ok := GetUserIDAsUUID(c); ok {
		workflowModel.CreatedBy = &userID
	}

	// Convert nodes
	workflowModel.Nodes = make([]*storagemodels.NodeModel, 0, len(workflow.Nodes))
	for _, node := range workflow.Nodes {
		nodeModel := &storagemodels.NodeModel{
			ID:         uuid.New(),
			NodeID:     node.ID,
			WorkflowID: workflowModel.ID,
			Name:       node.Name,
			Type:       node.Type,
			Config:     storagemodels.JSONBMap(node.Config),
			CreatedAt:  now,
			UpdatedAt:  now,
		}
		if node.Position != nil {
			nodeModel.Position = storagemodels.JSONBMap{
				"x": node.Position.X,
				"y": node.Position.Y,
			}
		}
		workflowModel.Nodes = append(workflowModel.Nodes, nodeModel)
	}

	// Convert edges
	workflowModel.Edges = make([]*storagemodels.EdgeModel, 0, len(workflow.Edges))
	for _, edge := range workflow.Edges {
		edgeModel := &storagemodels.EdgeModel{
			ID:           uuid.New(),
			EdgeID:       edge.ID,
			WorkflowID:   workflowModel.ID,
			FromNodeID:   edge.From,
			ToNodeID:     edge.To,
			SourceHandle: edge.SourceHandle,
			CreatedAt:    now,
			UpdatedAt:    now,
		}
		if edge.Condition != "" {
			edgeModel.Condition = storagemodels.JSONBMap{
				"expression": edge.Condition,
			}
		}
		workflowModel.Edges = append(workflowModel.Edges, edgeModel)
	}

	// Save workflow
	if err := h.workflowRepo.Create(c.Request.Context(), workflowModel); err != nil {
		h.logger.Error("Failed to create workflow", "error", err, "workflow_name", workflow.Name, "request_id", GetRequestID(c))
		respondAPIErrorWithRequestID(c, TranslateError(err))
		return nil, err
	}

	return workflowModel, nil
}

// saveTrigger saves the trigger for the imported workflow.
func (h *ImportHandlers) saveTrigger(c *gin.Context, result *importer.ImportResult, workflowID uuid.UUID) (uuid.UUID, error) {
	trigger := result.Trigger
	now := time.Now()

	// Store name and description in config
	config := storagemodels.JSONBMap(trigger.Config)
	if config == nil {
		config = storagemodels.JSONBMap{}
	}
	config["name"] = trigger.Name
	if trigger.Description != "" {
		config["description"] = trigger.Description
	}

	triggerModel := &storagemodels.TriggerModel{
		ID:         uuid.New(),
		WorkflowID: workflowID,
		Type:       string(trigger.Type),
		Config:     config,
		Enabled:    trigger.Enabled,
		CreatedAt:  now,
		UpdatedAt:  now,
	}

	if err := h.triggerRepo.Create(c.Request.Context(), triggerModel); err != nil {
		h.logger.Error("Failed to create trigger", "error", err, "workflow_id", workflowID, "request_id", GetRequestID(c))
		respondAPIErrorWithRequestID(c, TranslateError(err))
		return uuid.Nil, err
	}

	return triggerModel.ID, nil
}

// HandleExportWorkflow handles GET /api/v1/workflows/:workflow_id/export
// Exports a workflow to YAML or JSON format.
func (h *ImportHandlers) HandleExportWorkflow(c *gin.Context) {
	workflowID := c.Param("workflow_id")
	if workflowID == "" {
		respondAPIError(c, ErrMissingParameter)
		return
	}

	workflowUUID, err := uuid.Parse(workflowID)
	if err != nil {
		h.logger.Error("Invalid workflow ID format", "error", err, "workflow_id", workflowID, "request_id", GetRequestID(c))
		respondAPIError(c, ErrInvalidID)
		return
	}

	// Get workflow with relations
	workflowModel, err := h.workflowRepo.FindByIDWithRelations(c.Request.Context(), workflowUUID)
	if err != nil {
		h.logger.Error("Failed to find workflow", "error", err, "workflow_id", workflowUUID, "request_id", GetRequestID(c))
		respondAPIErrorWithRequestID(c, TranslateError(err))
		return
	}

	// Convert to domain model
	workflow := storagemodels.WorkflowModelToDomain(workflowModel)

	// Get trigger if exists (the importer stores name/description in config)
	var trigger *exportTrigger
	triggers, err := h.triggerRepo.FindByWorkflowID(c.Request.Context(), workflowUUID)
	if err == nil && len(triggers) > 0 {
		trigger = newExportTrigger(triggers[0])
	}

	// Check format parameter
	format := c.DefaultQuery("format", "yaml")

	switch strings.ToLower(format) {
	case "yaml", "yml":
		h.exportYAML(c, workflow, trigger)
	case "json":
		h.exportJSON(c, workflow, trigger)
	default:
		respondAPIError(c, NewAPIError("INVALID_FORMAT", "Format must be 'yaml' or 'json'", http.StatusBadRequest))
	}
}

// exportTrigger is the trigger representation used by workflow exports.
type exportTrigger struct {
	Name        string
	Description string
	Type        string
	Enabled     bool
	Config      map[string]any
	Metadata    map[string]any
}

func newExportTrigger(tm *storagemodels.TriggerModel) *exportTrigger {
	t := &exportTrigger{
		Type:    tm.Type,
		Enabled: tm.Enabled,
		Config:  map[string]any(tm.Config),
	}
	if n, ok := tm.Config["name"].(string); ok {
		t.Name = n
	}
	if d, ok := tm.Config["description"].(string); ok {
		t.Description = d
	}
	return t
}

// toDomain converts the export trigger to a domain trigger for the YAML
// exporter. The name/description keys the importer adds to the config are
// emitted as dedicated fields instead.
func (t *exportTrigger) toDomain() *pkgmodels.Trigger {
	if t == nil {
		return nil
	}
	config := make(map[string]any, len(t.Config))
	for k, v := range t.Config {
		if k == "name" || k == "description" {
			continue
		}
		config[k] = v
	}
	return &pkgmodels.Trigger{
		Name:        t.Name,
		Description: t.Description,
		Type:        pkgmodels.TriggerType(t.Type),
		Enabled:     t.Enabled,
		Config:      config,
		Metadata:    t.Metadata,
	}
}

// exportYAML exports the workflow in the YAML format accepted by the importer.
func (h *ImportHandlers) exportYAML(c *gin.Context, workflow *pkgmodels.Workflow, trigger *exportTrigger) {
	yamlData, err := h.importer.ExportToYAML(workflow, trigger.toDomain())
	if err != nil {
		h.logger.Error("Failed to build YAML export", "error", err, "request_id", GetRequestID(c))
		respondAPIError(c, NewAPIError("EXPORT_ERROR", "Failed to export workflow", http.StatusInternalServerError))
		return
	}

	c.Header("Content-Type", "application/x-yaml")
	c.Header("Content-Disposition", "attachment; filename=workflow.yaml")
	c.Data(http.StatusOK, "application/x-yaml", yamlData)
}

// exportJSON exports the workflow in JSON format.
func (h *ImportHandlers) exportJSON(c *gin.Context, workflow *pkgmodels.Workflow, trigger *exportTrigger) {
	export := gin.H{
		"workflow": workflow,
	}
	if trigger != nil {
		export["trigger"] = trigger
	}

	c.Header("Content-Disposition", "attachment; filename=workflow.json")
	c.JSON(http.StatusOK, export)
}

// HandleGetSupportedTypes handles GET /api/v1/workflows/import/types
// Returns a list of supported node types.
func (h *ImportHandlers) HandleGetSupportedTypes(c *gin.Context) {
	types := h.importer.GetSupportedNodeTypes()
	respondJSON(c, http.StatusOK, gin.H{
		"node_types": types,
	})
}
