package mbflow_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	mbflow "github.com/smilemakc/mbflow/go/sdk"
	"github.com/smilemakc/mbflow/go/sdk/models"
)

func TestExecutions_Run(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" || r.URL.Path != "/api/v1/service/executions" {
			t.Errorf("unexpected: %s %s", r.Method, r.URL.Path)
		}
		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)
		if body["workflow_id"] != "wf-1" {
			t.Errorf("workflow_id = %v", body["workflow_id"])
		}
		w.WriteHeader(201)
		json.NewEncoder(w).Encode(map[string]any{"id": "exec-1", "workflow_id": "wf-1", "status": "running"})
	}))
	defer server.Close()

	client, _ := mbflow.NewClient(mbflow.WithHTTP(server.URL), mbflow.WithSystemKey("key"))
	defer client.Close()

	exec, err := client.Executions().Run(context.Background(), "wf-1", map[string]any{"key": "val"})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if exec.ID != "exec-1" {
		t.Errorf("ID = %q", exec.ID)
	}
	if exec.Status != models.ExecutionStatusRunning {
		t.Errorf("Status = %q", exec.Status)
	}
}

func TestExecutions_Get(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/service/executions/exec-1" {
			t.Errorf("path = %s", r.URL.Path)
		}
		json.NewEncoder(w).Encode(map[string]any{"id": "exec-1", "status": "completed"})
	}))
	defer server.Close()

	client, _ := mbflow.NewClient(mbflow.WithHTTP(server.URL), mbflow.WithSystemKey("key"))
	exec, err := client.Executions().Get(context.Background(), "exec-1")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if exec.Status != models.ExecutionStatusCompleted {
		t.Errorf("Status = %q", exec.Status)
	}
}

func TestExecutions_Cancel(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" || r.URL.Path != "/api/v1/service/executions/exec-1/cancel" {
			t.Errorf("unexpected: %s %s", r.Method, r.URL.Path)
		}
		json.NewEncoder(w).Encode(map[string]any{"id": "exec-1", "status": "cancelled"})
	}))
	defer server.Close()

	client, _ := mbflow.NewClient(mbflow.WithHTTP(server.URL), mbflow.WithSystemKey("key"))
	exec, err := client.Executions().Cancel(context.Background(), "exec-1")
	if err != nil {
		t.Fatalf("Cancel: %v", err)
	}
	if exec.Status != models.ExecutionStatusCancelled {
		t.Errorf("Status = %q", exec.Status)
	}
}

func TestExecutions_List(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{
			"executions": []any{map[string]any{"id": "exec-1"}},
			"total":      1,
		})
	}))
	defer server.Close()

	client, _ := mbflow.NewClient(mbflow.WithHTTP(server.URL), mbflow.WithSystemKey("key"))
	page, err := client.Executions().List(context.Background(), nil)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(page.Items) != 1 {
		t.Errorf("Items len = %d", len(page.Items))
	}
}

func TestRunEphemeral_HTTP_Sync(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("method = %s, want POST", r.Method)
		}
		if r.URL.Path != "/api/v1/service/executions/ephemeral" {
			t.Errorf("path = %s, want /api/v1/service/executions/ephemeral", r.URL.Path)
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("decode body: %v", err)
		}
		if body["mode"] != "sync" {
			t.Errorf("mode = %v, want sync", body["mode"])
		}
		workflow, ok := body["workflow"].(map[string]any)
		if !ok {
			t.Error("workflow field missing or wrong type")
		} else if workflow["name"] != "inline-wf" {
			t.Errorf("workflow.name = %v, want inline-wf", workflow["name"])
		}
		w.WriteHeader(200)
		json.NewEncoder(w).Encode(map[string]any{
			"id":              "exec-ephemeral-1",
			"status":          "completed",
			"workflow_source": "inline",
		})
	}))
	defer server.Close()

	client, _ := mbflow.NewClient(mbflow.WithHTTP(server.URL), mbflow.WithSystemKey("key"))
	defer client.Close()

	req := &models.EphemeralExecutionRequest{
		Workflow: &models.Workflow{Name: "inline-wf"},
		Input:    map[string]any{"prompt": "hello"},
		Mode:     models.ExecutionModeSync,
	}
	exec, err := client.Executions().RunEphemeral(context.Background(), req)
	if err != nil {
		t.Fatalf("RunEphemeral: %v", err)
	}
	if exec.ID != "exec-ephemeral-1" {
		t.Errorf("ID = %q, want exec-ephemeral-1", exec.ID)
	}
	if exec.Status != models.ExecutionStatusCompleted {
		t.Errorf("Status = %q, want completed", exec.Status)
	}
	if exec.WorkflowSource != "inline" {
		t.Errorf("WorkflowSource = %q, want inline", exec.WorkflowSource)
	}
}

func TestRunEphemeral_HTTP_Async(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("method = %s, want POST", r.Method)
		}
		if r.URL.Path != "/api/v1/service/executions/ephemeral" {
			t.Errorf("path = %s, want /api/v1/service/executions/ephemeral", r.URL.Path)
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("decode body: %v", err)
		}
		if body["mode"] != "async" {
			t.Errorf("mode = %v, want async", body["mode"])
		}
		w.WriteHeader(202)
		json.NewEncoder(w).Encode(map[string]any{
			"id":              "exec-ephemeral-2",
			"status":          "pending",
			"workflow_source": "inline",
		})
	}))
	defer server.Close()

	client, _ := mbflow.NewClient(mbflow.WithHTTP(server.URL), mbflow.WithSystemKey("key"))
	defer client.Close()

	req := &models.EphemeralExecutionRequest{
		Workflow: &models.Workflow{Name: "async-wf"},
		Mode:     models.ExecutionModeAsync,
	}
	exec, err := client.Executions().RunEphemeral(context.Background(), req)
	if err != nil {
		t.Fatalf("RunEphemeral async: %v", err)
	}
	if exec.ID != "exec-ephemeral-2" {
		t.Errorf("ID = %q, want exec-ephemeral-2", exec.ID)
	}
	if exec.Status != models.ExecutionStatusPending {
		t.Errorf("Status = %q, want pending", exec.Status)
	}
	if exec.WorkflowSource != "inline" {
		t.Errorf("WorkflowSource = %q, want inline", exec.WorkflowSource)
	}
}

func TestRunEphemeral_HTTP_Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(413)
		json.NewEncoder(w).Encode(map[string]any{
			"code":    "payload_too_large",
			"message": "workflow definition exceeds maximum allowed size",
		})
	}))
	defer server.Close()

	client, _ := mbflow.NewClient(mbflow.WithHTTP(server.URL), mbflow.WithSystemKey("key"))
	defer client.Close()

	req := &models.EphemeralExecutionRequest{
		Workflow: &models.Workflow{Name: "oversized-wf"},
		Mode:     models.ExecutionModeSync,
	}
	_, err := client.Executions().RunEphemeral(context.Background(), req)
	if err == nil {
		t.Fatal("expected error for 413 response, got nil")
	}

	var apiErr *mbflow.APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected *mbflow.APIError, got %T: %v", err, err)
	}
	if apiErr.StatusCode != 413 {
		t.Errorf("StatusCode = %d, want 413", apiErr.StatusCode)
	}
	if apiErr.Code != "payload_too_large" {
		t.Errorf("Code = %q, want payload_too_large", apiErr.Code)
	}
}
