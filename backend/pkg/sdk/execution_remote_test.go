package sdk

import (
	"bufio"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/smilemakc/mbflow/pkg/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestExecutionAPI_RunRemote_Success tests successful remote execution start
func TestExecutionAPI_RunRemote_Success(t *testing.T) {
	// Create mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "/api/v1/executions", r.URL.Path)

		execution := &models.Execution{
			ID:         "exec-123",
			WorkflowID: "wf-456",
			Status:     models.ExecutionStatusPending,
			StartedAt:  time.Now(),
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(execution)
	}))
	defer server.Close()

	client, err := NewClient(
		WithHTTPEndpoint(server.URL),
		WithAPIKey("test-key"),
	)
	require.NoError(t, err)
	defer client.Close()

	ctx := context.Background()
	exec, err := client.Executions().Run(ctx, "wf-456", map[string]interface{}{"key": "value"})

	require.NoError(t, err)
	assert.Equal(t, "exec-123", exec.ID)
	assert.Equal(t, "wf-456", exec.WorkflowID)
	assert.Equal(t, models.ExecutionStatusPending, exec.Status)
}

// TestExecutionAPI_RunRemote_ServerError tests remote execution with server error
func TestExecutionAPI_RunRemote_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error": "internal server error"}`))
	}))
	defer server.Close()

	client, err := NewClient(
		WithHTTPEndpoint(server.URL),
		WithAPIKey("test-key"),
	)
	require.NoError(t, err)
	defer client.Close()

	ctx := context.Background()
	_, err = client.Executions().Run(ctx, "wf-456", nil)

	assert.Error(t, err)
}

// TestExecutionAPI_GetRemote_Success tests successful remote execution retrieval
func TestExecutionAPI_GetRemote_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "GET", r.Method)
		assert.Contains(t, r.URL.Path, "/api/v1/executions/exec-123")

		execution := &models.Execution{
			ID:          "exec-123",
			WorkflowID:  "wf-456",
			Status:      models.ExecutionStatusCompleted,
			StartedAt:   time.Now(),
			CompletedAt: func() *time.Time { t := time.Now(); return &t }(),
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(execution)
	}))
	defer server.Close()

	client, err := NewClient(
		WithHTTPEndpoint(server.URL),
		WithAPIKey("test-key"),
	)
	require.NoError(t, err)
	defer client.Close()

	ctx := context.Background()
	exec, err := client.Executions().Get(ctx, "exec-123")

	require.NoError(t, err)
	assert.Equal(t, "exec-123", exec.ID)
	assert.Equal(t, models.ExecutionStatusCompleted, exec.Status)
}

// TestExecutionAPI_GetRemote_NotFound tests remote execution not found
func TestExecutionAPI_GetRemote_NotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"error": "execution not found"}`))
	}))
	defer server.Close()

	client, err := NewClient(
		WithHTTPEndpoint(server.URL),
		WithAPIKey("test-key"),
	)
	require.NoError(t, err)
	defer client.Close()

	ctx := context.Background()
	_, err = client.Executions().Get(ctx, "exec-nonexistent")

	assert.Error(t, err)
}

// TestExecutionAPI_ListRemote_Success tests successful remote execution list
func TestExecutionAPI_ListRemote_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "GET", r.Method)
		assert.Equal(t, "/api/v1/executions", r.URL.Path)

		executions := []*models.Execution{
			{
				ID:         "exec-1",
				WorkflowID: "wf-1",
				Status:     models.ExecutionStatusCompleted,
				StartedAt:  time.Now(),
			},
			{
				ID:         "exec-2",
				WorkflowID: "wf-2",
				Status:     models.ExecutionStatusRunning,
				StartedAt:  time.Now(),
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"executions": executions,
			"total":      2,
		})
	}))
	defer server.Close()

	client, err := NewClient(
		WithHTTPEndpoint(server.URL),
		WithAPIKey("test-key"),
	)
	require.NoError(t, err)
	defer client.Close()

	ctx := context.Background()
	executions, err := client.Executions().List(ctx, nil)

	require.NoError(t, err)
	assert.Len(t, executions, 2)
	assert.Equal(t, "exec-1", executions[0].ID)
	assert.Equal(t, "exec-2", executions[1].ID)
}

// TestExecutionAPI_ListRemote_WithFilters tests list with query filters
func TestExecutionAPI_ListRemote_WithFilters(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "GET", r.Method)

		// Check query parameters
		query := r.URL.Query()
		assert.Equal(t, "wf-123", query.Get("workflow_id"))
		assert.Equal(t, "completed", query.Get("status"))
		assert.Equal(t, "10", query.Get("limit"))

		executions := []*models.Execution{
			{
				ID:         "exec-1",
				WorkflowID: "wf-123",
				Status:     models.ExecutionStatusCompleted,
				StartedAt:  time.Now(),
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"executions": executions,
			"total":      1,
		})
	}))
	defer server.Close()

	client, err := NewClient(
		WithHTTPEndpoint(server.URL),
		WithAPIKey("test-key"),
	)
	require.NoError(t, err)
	defer client.Close()

	ctx := context.Background()
	opts := &ExecutionListOptions{
		WorkflowID: "wf-123",
		Status:     string(models.ExecutionStatusCompleted),
		Limit:      10,
	}
	executions, err := client.Executions().List(ctx, opts)

	require.NoError(t, err)
	assert.Len(t, executions, 1)
	assert.Equal(t, "wf-123", executions[0].WorkflowID)
}

// TestExecutionAPI_CancelRemote_Success tests successful remote execution cancellation
func TestExecutionAPI_CancelRemote_Success(t *testing.T) {
	t.Skip("Cancel remote implementation not yet complete")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Contains(t, r.URL.Path, "/api/v1/executions/exec-123/cancel")

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"message": "execution cancelled"}`))
	}))
	defer server.Close()

	client, err := NewClient(
		WithHTTPEndpoint(server.URL),
		WithAPIKey("test-key"),
	)
	require.NoError(t, err)
	defer client.Close()

	ctx := context.Background()
	err = client.Executions().Cancel(ctx, "exec-123")

	assert.NoError(t, err)
}

// TestExecutionAPI_RetryRemote_Success tests successful remote execution retry
func TestExecutionAPI_RetryRemote_Success(t *testing.T) {
	t.Skip("Retry remote implementation not yet complete")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Contains(t, r.URL.Path, "/api/v1/executions/exec-123/retry")

		execution := &models.Execution{
			ID:         "exec-new",
			WorkflowID: "wf-456",
			Status:     models.ExecutionStatusPending,
			StartedAt:  time.Now(),
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(execution)
	}))
	defer server.Close()

	client, err := NewClient(
		WithHTTPEndpoint(server.URL),
		WithAPIKey("test-key"),
	)
	require.NoError(t, err)
	defer client.Close()

	ctx := context.Background()
	exec, err := client.Executions().Retry(ctx, "exec-123")

	require.NoError(t, err)
	assert.Equal(t, "exec-new", exec.ID)
	assert.Equal(t, models.ExecutionStatusPending, exec.Status)
}

// TestExecutionAPI_WatchRemote_Success tests successful remote execution watch
func TestExecutionAPI_WatchRemote_Success(t *testing.T) {
	t.Skip("Watch remote implementation not yet complete")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "GET", r.Method)
		assert.Contains(t, r.URL.Path, "/api/v1/executions/exec-123/watch")

		// Simulate SSE stream
		flusher, ok := w.(http.Flusher)
		require.True(t, ok)

		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")

		// Send one event then close
		update := &ExecutionUpdate{
			ExecutionID: "exec-123",
			Status:      string(models.ExecutionStatusCompleted),
			Event:       "execution.completed",
			Timestamp:   time.Now().Unix(),
		}
		data, _ := json.Marshal(update)
		w.Write([]byte("data: " + string(data) + "\n\n"))
		flusher.Flush()
	}))
	defer server.Close()

	client, err := NewClient(
		WithHTTPEndpoint(server.URL),
		WithAPIKey("test-key"),
	)
	require.NoError(t, err)
	defer client.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	ch, err := client.Executions().Watch(ctx, "exec-123")
	require.NoError(t, err)

	// Should receive at least one update
	select {
	case update := <-ch:
		assert.Equal(t, "exec-123", update.ExecutionID)
	case <-time.After(3 * time.Second):
		t.Fatal("timeout waiting for watch event")
	}
}

// TestExecutionAPI_GetLogsRemote_Success tests successful remote logs retrieval
func TestExecutionAPI_GetLogsRemote_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "GET", r.Method)
		assert.Contains(t, r.URL.Path, "/api/v1/executions/exec-123/logs")

		logs := []LogEntry{
			{
				Timestamp:   time.Now().Unix(),
				Level:       "info",
				Message:     "Starting execution",
				ExecutionID: "exec-123",
			},
			{
				Timestamp:   time.Now().Unix(),
				Level:       "info",
				Message:     "Node 1 completed",
				ExecutionID: "exec-123",
				NodeID:      "node-1",
			},
			{
				Timestamp:   time.Now().Unix(),
				Level:       "info",
				Message:     "Execution finished",
				ExecutionID: "exec-123",
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"logs": logs,
		})
	}))
	defer server.Close()

	client, err := NewClient(
		WithHTTPEndpoint(server.URL),
		WithAPIKey("test-key"),
	)
	require.NoError(t, err)
	defer client.Close()

	ctx := context.Background()
	logs, err := client.Executions().GetLogs(ctx, "exec-123", nil)

	require.NoError(t, err)
	assert.Len(t, logs, 3)
	assert.Contains(t, logs[0].Message, "Starting execution")
}

// TestExecutionAPI_StreamLogsRemote_Success tests successful remote log streaming
func TestExecutionAPI_StreamLogsRemote_Success(t *testing.T) {
	t.Skip("StreamLogs remote implementation not yet complete")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "GET", r.Method)
		assert.Contains(t, r.URL.Path, "/api/v1/executions/exec-123/logs/stream")

		flusher, ok := w.(http.Flusher)
		require.True(t, ok)

		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")

		// Send a few log lines
		logLines := []string{
			"Log line 1",
			"Log line 2",
			"Log line 3",
		}

		for _, line := range logLines {
			w.Write([]byte(line + "\n"))
			flusher.Flush()
			time.Sleep(10 * time.Millisecond)
		}
	}))
	defer server.Close()

	client, err := NewClient(
		WithHTTPEndpoint(server.URL),
		WithAPIKey("test-key"),
	)
	require.NoError(t, err)
	defer client.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	reader, err := client.Executions().StreamLogs(ctx, "exec-123", nil)
	require.NoError(t, err)
	defer reader.Close()

	// Read log lines
	scanner := bufio.NewScanner(reader)
	var logs []string
	for scanner.Scan() {
		logs = append(logs, scanner.Text())
		if len(logs) >= 3 {
			break
		}
	}

	assert.GreaterOrEqual(t, len(logs), 1)
	if len(logs) >= 3 {
		assert.Equal(t, "Log line 1", logs[0])
	}
}

// TestExecutionAPI_GetNodeResultRemote_Success tests successful remote node result retrieval
func TestExecutionAPI_GetNodeResultRemote_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "GET", r.Method)
		assert.Contains(t, r.URL.Path, "/api/v1/executions/exec-123/nodes/node-456")

		nodeExec := &models.NodeExecution{
			NodeID:      "node-456",
			ExecutionID: "exec-123",
			Status:      models.NodeExecutionStatusCompleted,
			Output: map[string]interface{}{
				"result": "success",
				"data":   []int{1, 2, 3},
			},
			StartedAt:   time.Now(),
			CompletedAt: func() *time.Time { t := time.Now(); return &t }(),
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(nodeExec)
	}))
	defer server.Close()

	client, err := NewClient(
		WithHTTPEndpoint(server.URL),
		WithAPIKey("test-key"),
	)
	require.NoError(t, err)
	defer client.Close()

	ctx := context.Background()
	nodeExec, err := client.Executions().GetNodeResult(ctx, "exec-123", "node-456")

	require.NoError(t, err)
	assert.NotNil(t, nodeExec)
	assert.Equal(t, "node-456", nodeExec.NodeID)
	assert.Equal(t, models.NodeExecutionStatusCompleted, nodeExec.Status)
	assert.Equal(t, "success", nodeExec.Output["result"])
}

// TestExecutionAPI_GetNodeResultRemote_NotFound tests node result not found
func TestExecutionAPI_GetNodeResultRemote_NotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"error": "node result not found"}`))
	}))
	defer server.Close()

	client, err := NewClient(
		WithHTTPEndpoint(server.URL),
		WithAPIKey("test-key"),
	)
	require.NoError(t, err)
	defer client.Close()

	ctx := context.Background()
	_, err = client.Executions().GetNodeResult(ctx, "exec-123", "node-nonexistent")

	assert.Error(t, err)
}

// TestExecutionAPI_RunSyncRemote_Success tests successful synchronous remote execution
func TestExecutionAPI_RunSyncRemote_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)

		// Parse request to check if async=false
		var req ExecutionRequest
		json.NewDecoder(r.Body).Decode(&req)

		execution := &models.Execution{
			ID:          "exec-123",
			WorkflowID:  "wf-456",
			Status:      models.ExecutionStatusCompleted,
			StartedAt:   time.Now(),
			CompletedAt: func() *time.Time { t := time.Now(); return &t }(),
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(execution)
	}))
	defer server.Close()

	client, err := NewClient(
		WithHTTPEndpoint(server.URL),
		WithAPIKey("test-key"),
	)
	require.NoError(t, err)
	defer client.Close()

	ctx := context.Background()
	exec, err := client.Executions().RunSync(ctx, "wf-456", nil)

	require.NoError(t, err)
	assert.Equal(t, "exec-123", exec.ID)
	assert.Equal(t, models.ExecutionStatusCompleted, exec.Status)
	assert.NotNil(t, exec.CompletedAt)
}

// TestExecutionAPI_RunRemote_EmptyWorkflowID tests validation of empty workflow ID
func TestExecutionAPI_RunRemote_EmptyWorkflowID(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("should not reach server with empty workflow ID")
	}))
	defer server.Close()

	client, err := NewClient(
		WithHTTPEndpoint(server.URL),
		WithAPIKey("test-key"),
	)
	require.NoError(t, err)
	defer client.Close()

	ctx := context.Background()
	_, err = client.Executions().Run(ctx, "", nil)

	assert.ErrorIs(t, err, models.ErrInvalidWorkflowID)
}

// TestExecutionAPI_GetRemote_EmptyExecutionID tests validation of empty execution ID
func TestExecutionAPI_GetRemote_EmptyExecutionID(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("should not reach server with empty execution ID")
	}))
	defer server.Close()

	client, err := NewClient(
		WithHTTPEndpoint(server.URL),
		WithAPIKey("test-key"),
	)
	require.NoError(t, err)
	defer client.Close()

	ctx := context.Background()
	_, err = client.Executions().Get(ctx, "")

	assert.ErrorIs(t, err, models.ErrInvalidExecutionID)
}

// TestExecutionAPI_RunRemote_WithInput tests execution with input data
func TestExecutionAPI_RunRemote_WithInput(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req ExecutionRequest
		json.NewDecoder(r.Body).Decode(&req)

		// Verify input is sent
		assert.NotNil(t, req.Input)
		assert.Equal(t, "test-value", req.Input["testKey"])
		assert.Equal(t, true, req.Async)

		execution := &models.Execution{
			ID:         "exec-123",
			WorkflowID: "wf-456",
			Status:     models.ExecutionStatusPending,
			StartedAt:  time.Now(),
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(execution)
	}))
	defer server.Close()

	client, err := NewClient(
		WithHTTPEndpoint(server.URL),
		WithAPIKey("test-key"),
	)
	require.NoError(t, err)
	defer client.Close()

	ctx := context.Background()

	exec, err := client.Executions().Run(ctx, "wf-456", map[string]interface{}{"testKey": "test-value"})

	require.NoError(t, err)
	assert.Equal(t, "exec-123", exec.ID)
}
