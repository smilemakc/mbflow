package mbflow_test

import (
	"context"
	"encoding/json"
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
