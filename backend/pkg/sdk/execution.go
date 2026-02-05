package sdk

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/google/uuid"
	"github.com/smilemakc/mbflow/internal/application/engine"
	storagemodels "github.com/smilemakc/mbflow/internal/infrastructure/storage/models"
	"github.com/smilemakc/mbflow/pkg/models"
)

// ExecutionAPI provides methods for running and monitoring workflow executions.
// It supports synchronous and asynchronous execution modes, real-time monitoring,
// and execution history management.
type ExecutionAPI struct {
	client *Client
}

// newExecutionAPI creates a new ExecutionAPI instance.
func newExecutionAPI(client *Client) *ExecutionAPI {
	return &ExecutionAPI{
		client: client,
	}
}

// Run starts a new workflow execution with the given input.
// It returns immediately with an execution ID for asynchronous tracking.
func (e *ExecutionAPI) Run(ctx context.Context, workflowID string, input map[string]interface{}) (*models.Execution, error) {
	if err := e.client.checkClosed(); err != nil {
		return nil, err
	}

	if workflowID == "" {
		return nil, models.ErrInvalidWorkflowID
	}

	req := &ExecutionRequest{
		WorkflowID: workflowID,
		Input:      input,
		Async:      true,
	}

	if e.client.config.Mode == ModeRemote {
		return e.runRemote(ctx, req)
	}

	return e.runEmbedded(ctx, req)
}

// RunSync starts a workflow execution and waits for it to complete.
// This is a blocking call that returns the final execution result.
func (e *ExecutionAPI) RunSync(ctx context.Context, workflowID string, input map[string]interface{}) (*models.Execution, error) {
	if err := e.client.checkClosed(); err != nil {
		return nil, err
	}

	if workflowID == "" {
		return nil, models.ErrInvalidWorkflowID
	}

	req := &ExecutionRequest{
		WorkflowID: workflowID,
		Input:      input,
		Async:      false,
	}

	if e.client.config.Mode == ModeRemote {
		return e.runRemote(ctx, req)
	}

	return e.runEmbedded(ctx, req)
}

// Get retrieves an execution by ID.
func (e *ExecutionAPI) Get(ctx context.Context, executionID string) (*models.Execution, error) {
	if err := e.client.checkClosed(); err != nil {
		return nil, err
	}

	if executionID == "" {
		return nil, models.ErrInvalidExecutionID
	}

	if e.client.config.Mode == ModeRemote {
		return e.getRemote(ctx, executionID)
	}

	return e.getEmbedded(ctx, executionID)
}

// List retrieves executions with optional filtering.
func (e *ExecutionAPI) List(ctx context.Context, opts *ExecutionListOptions) ([]*models.Execution, error) {
	if err := e.client.checkClosed(); err != nil {
		return nil, err
	}

	if e.client.config.Mode == ModeRemote {
		return e.listRemote(ctx, opts)
	}

	return e.listEmbedded(ctx, opts)
}

// Cancel cancels a running execution.
func (e *ExecutionAPI) Cancel(ctx context.Context, executionID string) error {
	if err := e.client.checkClosed(); err != nil {
		return err
	}

	if executionID == "" {
		return models.ErrInvalidExecutionID
	}

	if e.client.config.Mode == ModeRemote {
		return e.cancelRemote(ctx, executionID)
	}

	return e.cancelEmbedded(ctx, executionID)
}

// Retry retries a failed execution from the last failed node.
func (e *ExecutionAPI) Retry(ctx context.Context, executionID string) (*models.Execution, error) {
	if err := e.client.checkClosed(); err != nil {
		return nil, err
	}

	if executionID == "" {
		return nil, models.ErrInvalidExecutionID
	}

	if e.client.config.Mode == ModeRemote {
		return e.retryRemote(ctx, executionID)
	}

	return e.retryEmbedded(ctx, executionID)
}

// Watch watches an execution and streams updates in real-time.
// The returned channel receives execution updates until the execution completes
// or the context is cancelled.
func (e *ExecutionAPI) Watch(ctx context.Context, executionID string) (<-chan *ExecutionUpdate, error) {
	if err := e.client.checkClosed(); err != nil {
		return nil, err
	}

	if executionID == "" {
		return nil, models.ErrInvalidExecutionID
	}

	if e.client.config.Mode == ModeRemote {
		return e.watchRemote(ctx, executionID)
	}

	return e.watchEmbedded(ctx, executionID)
}

// GetLogs retrieves execution logs with optional filtering.
func (e *ExecutionAPI) GetLogs(ctx context.Context, executionID string, opts *LogOptions) ([]LogEntry, error) {
	if err := e.client.checkClosed(); err != nil {
		return nil, err
	}

	if executionID == "" {
		return nil, models.ErrInvalidExecutionID
	}

	if e.client.config.Mode == ModeRemote {
		return e.getLogsRemote(ctx, executionID, opts)
	}

	return e.getLogsEmbedded(ctx, executionID, opts)
}

// StreamLogs streams execution logs in real-time.
func (e *ExecutionAPI) StreamLogs(ctx context.Context, executionID string, opts *LogOptions) (io.ReadCloser, error) {
	if err := e.client.checkClosed(); err != nil {
		return nil, err
	}

	if executionID == "" {
		return nil, models.ErrInvalidExecutionID
	}

	if e.client.config.Mode == ModeRemote {
		return e.streamLogsRemote(ctx, executionID, opts)
	}

	return e.streamLogsEmbedded(ctx, executionID, opts)
}

// GetNodeResult retrieves the result of a specific node execution.
func (e *ExecutionAPI) GetNodeResult(ctx context.Context, executionID, nodeID string) (*models.NodeExecution, error) {
	if err := e.client.checkClosed(); err != nil {
		return nil, err
	}

	if executionID == "" {
		return nil, models.ErrInvalidExecutionID
	}

	if nodeID == "" {
		return nil, fmt.Errorf("node ID is required")
	}

	if e.client.config.Mode == ModeRemote {
		return e.getNodeResultRemote(ctx, executionID, nodeID)
	}

	return e.getNodeResultEmbedded(ctx, executionID, nodeID)
}

// ExecutionRequest represents a request to execute a workflow.
type ExecutionRequest struct {
	WorkflowID string                 `json:"workflow_id"`
	Input      map[string]interface{} `json:"input"`
	Async      bool                   `json:"async"`
}

// ExecutionListOptions provides filtering options for listing executions.
type ExecutionListOptions struct {
	WorkflowID string
	Status     string
	Limit      int
	Offset     int
	StartTime  *int64
	EndTime    *int64
}

// ExecutionUpdate represents a real-time update from a running execution.
type ExecutionUpdate struct {
	ExecutionID string                 `json:"execution_id"`
	Status      string                 `json:"status"`
	NodeID      string                 `json:"node_id,omitempty"`
	Event       string                 `json:"event"`
	Data        map[string]interface{} `json:"data,omitempty"`
	Timestamp   int64                  `json:"timestamp"`
}

// LogOptions provides filtering options for execution logs.
type LogOptions struct {
	NodeID    string
	Level     string
	Limit     int
	Offset    int
	StartTime *int64
	EndTime   *int64
	Follow    bool
}

// LogEntry represents a single log entry from an execution.
type LogEntry struct {
	Timestamp   int64                  `json:"timestamp"`
	Level       string                 `json:"level"`
	Message     string                 `json:"message"`
	NodeID      string                 `json:"node_id,omitempty"`
	ExecutionID string                 `json:"execution_id"`
	Fields      map[string]interface{} `json:"fields,omitempty"`
}

// Embedded mode implementations
func (e *ExecutionAPI) runEmbedded(ctx context.Context, req *ExecutionRequest) (*models.Execution, error) {
	// Check if ExecutionManager is available (full mode with persistence)
	execManager := e.client.getExecutionManager()
	if execManager != nil {
		// Full mode with persistence
		opts := engine.DefaultExecutionOptions()
		return execManager.Execute(ctx, req.WorkflowID, req.Input, opts)
	}

	// Fallback to standalone mode (no persistence)
	return nil, fmt.Errorf("embedded execution requires ExecutionManager or use ExecuteWorkflowStandalone instead")
}

func (e *ExecutionAPI) getEmbedded(ctx context.Context, executionID string) (*models.Execution, error) {
	if e.client.executionRepo == nil {
		return nil, fmt.Errorf("execution repository not initialized")
	}

	// Parse UUID
	execUUID, err := uuid.Parse(executionID)
	if err != nil {
		return nil, fmt.Errorf("invalid execution ID: %w", err)
	}

	// Get execution with node executions
	execModel, err := e.client.executionRepo.FindByIDWithRelations(ctx, execUUID)
	if err != nil {
		return nil, fmt.Errorf("failed to find execution: %w", err)
	}

	// Convert to domain model
	execution := storagemodels.ExecutionModelToDomain(execModel)

	return execution, nil
}

func (e *ExecutionAPI) listEmbedded(ctx context.Context, opts *ExecutionListOptions) ([]*models.Execution, error) {
	if e.client.executionRepo == nil {
		return nil, fmt.Errorf("execution repository not initialized")
	}

	// Set defaults
	if opts == nil {
		opts = &ExecutionListOptions{
			Limit:  50,
			Offset: 0,
		}
	}
	if opts.Limit <= 0 {
		opts.Limit = 50
	}

	var execModels []*storagemodels.ExecutionModel
	var err error

	// Route to appropriate repository method based on filters
	if opts.WorkflowID != "" {
		wfUUID, err := uuid.Parse(opts.WorkflowID)
		if err != nil {
			return nil, fmt.Errorf("invalid workflow ID: %w", err)
		}
		execModels, err = e.client.executionRepo.FindByWorkflowID(ctx, wfUUID, opts.Limit, opts.Offset)
	} else if opts.Status != "" {
		execModels, err = e.client.executionRepo.FindByStatus(ctx, opts.Status, opts.Limit, opts.Offset)
	} else {
		execModels, err = e.client.executionRepo.FindAll(ctx, opts.Limit, opts.Offset)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to list executions: %w", err)
	}

	// Convert to domain models
	executions := make([]*models.Execution, len(execModels))
	for i, em := range execModels {
		executions[i] = storagemodels.ExecutionModelToDomain(em)
	}

	return executions, nil
}

func (e *ExecutionAPI) cancelEmbedded(ctx context.Context, executionID string) error {
	// Deferred for MVP - requires context propagation through execution engine
	return fmt.Errorf("execution cancellation not yet implemented")
}

func (e *ExecutionAPI) retryEmbedded(ctx context.Context, executionID string) (*models.Execution, error) {
	// Deferred for MVP - can be implemented in future iteration
	return nil, fmt.Errorf("execution retry not yet implemented")
}

func (e *ExecutionAPI) watchEmbedded(ctx context.Context, executionID string) (<-chan *ExecutionUpdate, error) {
	// Deferred for MVP - requires EventRepository.Stream()
	return nil, fmt.Errorf("real-time execution watching not yet implemented")
}

func (e *ExecutionAPI) getLogsEmbedded(ctx context.Context, executionID string, opts *LogOptions) ([]LogEntry, error) {
	// For MVP, return empty logs since EventRepository is deferred
	// In full implementation, this would query eventRepo.FindByExecutionID()
	return []LogEntry{}, nil
}

func (e *ExecutionAPI) streamLogsEmbedded(ctx context.Context, executionID string, opts *LogOptions) (io.ReadCloser, error) {
	// Deferred for MVP - requires EventRepository.Stream()
	return nil, fmt.Errorf("log streaming not yet implemented")
}

func (e *ExecutionAPI) getNodeResultEmbedded(ctx context.Context, executionID, nodeID string) (*models.NodeExecution, error) {
	if e.client.executionRepo == nil || e.client.workflowRepo == nil {
		return nil, fmt.Errorf("repositories not initialized")
	}

	// Parse UUID
	execUUID, err := uuid.Parse(executionID)
	if err != nil {
		return nil, fmt.Errorf("invalid execution ID: %w", err)
	}

	// Get execution with relations
	execModel, err := e.client.executionRepo.FindByIDWithRelations(ctx, execUUID)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve execution: %w", err)
	}

	// Load workflow to map node UUIDs to logical IDs
	workflowModel, err := e.client.workflowRepo.FindByIDWithRelations(ctx, execModel.WorkflowID)
	if err != nil {
		return nil, fmt.Errorf("failed to load workflow: %w", err)
	}

	// Build node ID mapping (UUID -> logical ID)
	nodeIDMap := make(map[uuid.UUID]string)
	for _, node := range workflowModel.Nodes {
		nodeIDMap[node.ID] = node.NodeID
	}

	// Find matching node execution
	for _, ne := range execModel.NodeExecutions {
		if logicalID, ok := nodeIDMap[ne.NodeID]; ok && logicalID == nodeID {
			// Convert to domain model
			return storagemodels.NodeExecutionModelToDomain(ne), nil
		}
	}

	return nil, fmt.Errorf("node execution not found for node: %s", nodeID)
}

// Remote mode implementations
func (e *ExecutionAPI) runRemote(ctx context.Context, req *ExecutionRequest) (*models.Execution, error) {
	u := fmt.Sprintf("%s/api/v1/executions", e.client.config.BaseURL)

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", u, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	if e.client.config.APIKey != "" {
		httpReq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", e.client.config.APIKey))
	}

	resp, err := e.client.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error (%d): %s", resp.StatusCode, string(respBody))
	}

	var execution models.Execution
	if err := json.NewDecoder(resp.Body).Decode(&execution); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &execution, nil
}

func (e *ExecutionAPI) getRemote(ctx context.Context, executionID string) (*models.Execution, error) {
	u := fmt.Sprintf("%s/api/v1/executions/%s", e.client.config.BaseURL, executionID)

	httpReq, err := http.NewRequestWithContext(ctx, "GET", u, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	if e.client.config.APIKey != "" {
		httpReq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", e.client.config.APIKey))
	}

	resp, err := e.client.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error (%d): %s", resp.StatusCode, string(respBody))
	}

	var execution models.Execution
	if err := json.NewDecoder(resp.Body).Decode(&execution); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &execution, nil
}

func (e *ExecutionAPI) listRemote(ctx context.Context, opts *ExecutionListOptions) ([]*models.Execution, error) {
	baseURL := fmt.Sprintf("%s/api/v1/executions", e.client.config.BaseURL)

	// Add query parameters
	if opts != nil {
		query := make(url.Values)
		if opts.WorkflowID != "" {
			query.Set("workflow_id", opts.WorkflowID)
		}
		if opts.Status != "" {
			query.Set("status", opts.Status)
		}
		if opts.Limit > 0 {
			query.Set("limit", fmt.Sprintf("%d", opts.Limit))
		}
		if opts.Offset > 0 {
			query.Set("offset", fmt.Sprintf("%d", opts.Offset))
		}
		if len(query) > 0 {
			baseURL += "?" + query.Encode()
		}
	}

	httpReq, err := http.NewRequestWithContext(ctx, "GET", baseURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	if e.client.config.APIKey != "" {
		httpReq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", e.client.config.APIKey))
	}

	resp, err := e.client.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error (%d): %s", resp.StatusCode, string(respBody))
	}

	var result struct {
		Executions []*models.Execution `json:"executions"`
		Total      int                 `json:"total"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return result.Executions, nil
}

func (e *ExecutionAPI) cancelRemote(ctx context.Context, executionID string) error {
	// Deferred for MVP
	return fmt.Errorf("execution cancellation not yet implemented")
}

func (e *ExecutionAPI) retryRemote(ctx context.Context, executionID string) (*models.Execution, error) {
	// Deferred for MVP
	return nil, fmt.Errorf("execution retry not yet implemented")
}

func (e *ExecutionAPI) watchRemote(ctx context.Context, executionID string) (<-chan *ExecutionUpdate, error) {
	// Deferred for MVP - requires WebSocket
	return nil, fmt.Errorf("real-time execution watching not yet implemented")
}

func (e *ExecutionAPI) getLogsRemote(ctx context.Context, executionID string, opts *LogOptions) ([]LogEntry, error) {
	baseURL := fmt.Sprintf("%s/api/v1/executions/%s/logs", e.client.config.BaseURL, executionID)

	// Add query parameters
	if opts != nil {
		query := make(url.Values)
		if opts.NodeID != "" {
			query.Set("node_id", opts.NodeID)
		}
		if opts.Level != "" {
			query.Set("level", opts.Level)
		}
		if opts.Limit > 0 {
			query.Set("limit", fmt.Sprintf("%d", opts.Limit))
		}
		if opts.Offset > 0 {
			query.Set("offset", fmt.Sprintf("%d", opts.Offset))
		}
		if len(query) > 0 {
			baseURL += "?" + query.Encode()
		}
	}

	httpReq, err := http.NewRequestWithContext(ctx, "GET", baseURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	if e.client.config.APIKey != "" {
		httpReq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", e.client.config.APIKey))
	}

	resp, err := e.client.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error (%d): %s", resp.StatusCode, string(respBody))
	}

	var result struct {
		Logs  []LogEntry `json:"logs"`
		Total int        `json:"total"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return result.Logs, nil
}

func (e *ExecutionAPI) streamLogsRemote(ctx context.Context, executionID string, opts *LogOptions) (io.ReadCloser, error) {
	// Deferred for MVP - requires SSE streaming
	return nil, fmt.Errorf("log streaming not yet implemented")
}

func (e *ExecutionAPI) getNodeResultRemote(ctx context.Context, executionID, nodeID string) (*models.NodeExecution, error) {
	u := fmt.Sprintf("%s/api/v1/executions/%s/nodes/%s", e.client.config.BaseURL, executionID, nodeID)

	httpReq, err := http.NewRequestWithContext(ctx, "GET", u, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	if e.client.config.APIKey != "" {
		httpReq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", e.client.config.APIKey))
	}

	resp, err := e.client.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error (%d): %s", resp.StatusCode, string(respBody))
	}

	var nodeExec models.NodeExecution
	if err := json.NewDecoder(resp.Body).Decode(&nodeExec); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &nodeExec, nil
}
