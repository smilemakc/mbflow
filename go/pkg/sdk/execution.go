package sdk

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/smilemakc/mbflow/go/pkg/models"
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
func (e *ExecutionAPI) Run(ctx context.Context, workflowID string, input map[string]any, variables map[string]any) (*models.Execution, error) {
	if err := e.client.checkClosed(); err != nil {
		return nil, err
	}

	if workflowID == "" {
		return nil, models.ErrInvalidWorkflowID
	}

	req := &ExecutionRequest{
		WorkflowID: workflowID,
		Input:      input,
		Variables:  variables,
		Async:      true,
	}

	if e.client.config.Mode == ModeRemote {
		return e.runRemote(ctx, req)
	}

	return e.runEmbedded(ctx, req)
}

// RunSync starts a workflow execution and waits for it to complete.
// This is a blocking call that returns the final execution result.
func (e *ExecutionAPI) RunSync(ctx context.Context, workflowID string, input map[string]any, variables map[string]any) (*models.Execution, error) {
	if err := e.client.checkClosed(); err != nil {
		return nil, err
	}

	if workflowID == "" {
		return nil, models.ErrInvalidWorkflowID
	}

	req := &ExecutionRequest{
		WorkflowID: workflowID,
		Input:      input,
		Variables:  variables,
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
	WorkflowID string         `json:"workflow_id"`
	Input      map[string]any `json:"input"`
	Variables  map[string]any `json:"variables,omitempty"`
	Async      bool           `json:"async"`
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
	ExecutionID string         `json:"execution_id"`
	Status      string         `json:"status"`
	NodeID      string         `json:"node_id,omitempty"`
	Event       string         `json:"event"`
	Data        map[string]any `json:"data,omitempty"`
	Timestamp   int64          `json:"timestamp"`
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
	// Timestamp is a Unix timestamp in seconds. When decoding, RFC 3339
	// strings (as sent by the REST API) are accepted as well.
	Timestamp   int64          `json:"timestamp"`
	Level       string         `json:"level"`
	Message     string         `json:"message"`
	EventType   string         `json:"event_type,omitempty"`
	NodeID      string         `json:"node_id,omitempty"`
	ExecutionID string         `json:"execution_id"`
	Fields      map[string]any `json:"fields,omitempty"`
}

// UnmarshalJSON accepts both numeric (Unix seconds) and RFC 3339 timestamps,
// and maps the REST API's "data" field onto Fields.
func (l *LogEntry) UnmarshalJSON(b []byte) error {
	type plain LogEntry
	var aux struct {
		plain
		Timestamp json.RawMessage `json:"timestamp"`
		Data      map[string]any  `json:"data,omitempty"`
	}
	if err := json.Unmarshal(b, &aux); err != nil {
		return err
	}
	*l = LogEntry(aux.plain)
	if l.Fields == nil && aux.Data != nil {
		l.Fields = aux.Data
	}
	ts := aux.Timestamp
	if len(ts) == 0 || string(ts) == "null" {
		l.Timestamp = 0
		return nil
	}
	if ts[0] == '"' {
		var str string
		if err := json.Unmarshal(ts, &str); err != nil {
			return err
		}
		if str == "" {
			return nil
		}
		parsed, err := time.Parse(time.RFC3339Nano, str)
		if err != nil {
			return fmt.Errorf("invalid log timestamp %q: %w", str, err)
		}
		l.Timestamp = parsed.Unix()
		return nil
	}
	var n json.Number
	if err := json.Unmarshal(ts, &n); err != nil {
		return fmt.Errorf("invalid log timestamp: %w", err)
	}
	f, err := n.Float64()
	if err != nil {
		return fmt.Errorf("invalid log timestamp: %w", err)
	}
	l.Timestamp = int64(f)
	return nil
}

// Embedded mode implementations (standalone mode - no database persistence)
// For full persistence support, use pkg/server.Server directly.

var errStandaloneModeNotSupported = fmt.Errorf("operation not available in standalone mode; use remote mode or pkg/server.Server for persistence")

func (e *ExecutionAPI) runEmbedded(ctx context.Context, req *ExecutionRequest) (*models.Execution, error) {
	// Standalone mode doesn't support Run() - use ExecuteWorkflowStandalone() instead
	return nil, fmt.Errorf("Run() not available in standalone mode; use ExecuteWorkflowStandalone() for in-memory execution or remote mode for persistence")
}

func (e *ExecutionAPI) getEmbedded(ctx context.Context, executionID string) (*models.Execution, error) {
	return nil, errStandaloneModeNotSupported
}

func (e *ExecutionAPI) listEmbedded(ctx context.Context, opts *ExecutionListOptions) ([]*models.Execution, error) {
	return nil, errStandaloneModeNotSupported
}

func (e *ExecutionAPI) cancelEmbedded(ctx context.Context, executionID string) error {
	return errStandaloneModeNotSupported
}

func (e *ExecutionAPI) retryEmbedded(ctx context.Context, executionID string) (*models.Execution, error) {
	return nil, errStandaloneModeNotSupported
}

func (e *ExecutionAPI) watchEmbedded(ctx context.Context, executionID string) (<-chan *ExecutionUpdate, error) {
	return nil, errStandaloneModeNotSupported
}

func (e *ExecutionAPI) getLogsEmbedded(ctx context.Context, executionID string, opts *LogOptions) ([]LogEntry, error) {
	return nil, errStandaloneModeNotSupported
}

func (e *ExecutionAPI) streamLogsEmbedded(ctx context.Context, executionID string, opts *LogOptions) (io.ReadCloser, error) {
	return nil, errStandaloneModeNotSupported
}

func (e *ExecutionAPI) getNodeResultEmbedded(ctx context.Context, executionID, nodeID string) (*models.NodeExecution, error) {
	return nil, errStandaloneModeNotSupported
}

// Remote mode implementations.
//
// These call the MBFlow REST API (see pkg/server/routes.go):
//
//	POST /api/v1/executions/run/{workflow_id}
//	GET  /api/v1/executions/{id}
//	GET  /api/v1/executions?workflow_id=&status=&limit=&offset=
//	GET  /api/v1/executions/{id}/logs
//	GET  /api/v1/executions/{id}/nodes/{node_id}/result
//	POST /api/v1/executions/{id}/cancel
//	POST /api/v1/executions/{id}/retry
//
// Watch and StreamLogs need WebSocket/SSE transports that the SDK does not
// implement yet; they return *UnsupportedOperationError. Endpoints that the
// server answers with 501 surface as *APIError matching ErrNotSupportedByAPI.

// syncPollInterval is how often RunSync polls a remote execution until it
// reaches a terminal state.
var syncPollInterval = 500 * time.Millisecond

func executionPath(executionID string) string {
	return "/api/v1/executions/" + pathEscape(executionID)
}

func isTerminalExecutionStatus(status models.ExecutionStatus) bool {
	switch status {
	case models.ExecutionStatusCompleted, models.ExecutionStatusFailed,
		models.ExecutionStatusCancelled, models.ExecutionStatusTimeout:
		return true
	}
	return false
}

func (e *ExecutionAPI) runRemote(ctx context.Context, req *ExecutionRequest) (*models.Execution, error) {
	var execution models.Execution
	path := "/api/v1/executions/run/" + pathEscape(req.WorkflowID)
	if err := e.client.doRemote(ctx, http.MethodPost, path, nil, req, &execution); err != nil {
		return nil, err
	}

	if req.Async || isTerminalExecutionStatus(execution.Status) || execution.ID == "" {
		return &execution, nil
	}

	// The server always starts executions asynchronously; RunSync waits for
	// the execution to finish by polling it.
	ticker := time.NewTicker(syncPollInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return &execution, ctx.Err()
		case <-ticker.C:
		}
		current, err := e.getRemote(ctx, execution.ID)
		if err != nil {
			if ctxErr := ctx.Err(); ctxErr != nil {
				return &execution, ctxErr
			}
			return nil, err
		}
		if isTerminalExecutionStatus(current.Status) {
			return current, nil
		}
		execution = *current
	}
}

func (e *ExecutionAPI) getRemote(ctx context.Context, executionID string) (*models.Execution, error) {
	var execution models.Execution
	if err := e.client.doRemote(ctx, http.MethodGet, executionPath(executionID), nil, nil, &execution); err != nil {
		return nil, err
	}
	return &execution, nil
}

func (e *ExecutionAPI) listRemote(ctx context.Context, opts *ExecutionListOptions) ([]*models.Execution, error) {
	query := url.Values{}
	if opts != nil {
		if opts.WorkflowID != "" {
			query.Set("workflow_id", opts.WorkflowID)
		}
		if opts.Status != "" {
			query.Set("status", opts.Status)
		}
		setPagination(query, opts.Limit, opts.Offset)
	}

	var raw map[string]json.RawMessage
	if err := e.client.doRemote(ctx, http.MethodGet, "/api/v1/executions", query, nil, &raw); err != nil {
		return nil, err
	}
	return decodeListEnvelope[*models.Execution](raw, "executions")
}

func (e *ExecutionAPI) cancelRemote(ctx context.Context, executionID string) error {
	return e.client.doRemote(ctx, http.MethodPost, executionPath(executionID)+"/cancel", nil, nil, nil)
}

func (e *ExecutionAPI) retryRemote(ctx context.Context, executionID string) (*models.Execution, error) {
	var execution models.Execution
	if err := e.client.doRemote(ctx, http.MethodPost, executionPath(executionID)+"/retry", nil, nil, &execution); err != nil {
		return nil, err
	}
	return &execution, nil
}

func (e *ExecutionAPI) watchRemote(ctx context.Context, executionID string) (<-chan *ExecutionUpdate, error) {
	return nil, errUnsupported("ExecutionAPI.Watch", "real-time watching requires the WebSocket transport, which the SDK does not implement yet")
}

func (e *ExecutionAPI) getLogsRemote(ctx context.Context, executionID string, opts *LogOptions) ([]LogEntry, error) {
	query := url.Values{}
	if opts != nil {
		if opts.NodeID != "" {
			query.Set("node_id", opts.NodeID)
		}
		if opts.Level != "" {
			query.Set("level", opts.Level)
		}
		setPagination(query, opts.Limit, opts.Offset)
	}

	var result struct {
		Logs  []LogEntry `json:"logs"`
		Total int        `json:"total"`
	}
	if err := e.client.doRemote(ctx, http.MethodGet, executionPath(executionID)+"/logs", query, nil, &result); err != nil {
		return nil, err
	}
	return result.Logs, nil
}

func (e *ExecutionAPI) streamLogsRemote(ctx context.Context, executionID string, opts *LogOptions) (io.ReadCloser, error) {
	return nil, errUnsupported("ExecutionAPI.StreamLogs", "log streaming requires the SSE transport, which the SDK does not implement yet")
}

func (e *ExecutionAPI) getNodeResultRemote(ctx context.Context, executionID, nodeID string) (*models.NodeExecution, error) {
	var nodeExec models.NodeExecution
	path := executionPath(executionID) + "/nodes/" + pathEscape(nodeID) + "/result"
	if err := e.client.doRemote(ctx, http.MethodGet, path, nil, nil, &nodeExec); err != nil {
		return nil, err
	}
	return &nodeExec, nil
}
