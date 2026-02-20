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

func TestWorkflows_Create(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" || r.URL.Path != "/api/v1/service/workflows" {
			t.Errorf("unexpected: %s %s", r.Method, r.URL.Path)
		}
		w.WriteHeader(201)
		json.NewEncoder(w).Encode(map[string]any{"id": "wf-1", "name": "Test", "status": "draft", "version": 1})
	}))
	defer server.Close()

	client, _ := mbflow.NewClient(mbflow.WithHTTP(server.URL), mbflow.WithSystemKey("key"))
	defer client.Close()

	wf, err := client.Workflows().Create(context.Background(), &models.Workflow{Name: "Test"})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if wf.ID != "wf-1" {
		t.Errorf("ID = %q, want %q", wf.ID, "wf-1")
	}
}

func TestWorkflows_Get(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" || r.URL.Path != "/api/v1/service/workflows/wf-1" {
			t.Errorf("unexpected: %s %s", r.Method, r.URL.Path)
		}
		json.NewEncoder(w).Encode(map[string]any{"id": "wf-1", "name": "My Workflow", "status": "active"})
	}))
	defer server.Close()

	client, _ := mbflow.NewClient(mbflow.WithHTTP(server.URL), mbflow.WithSystemKey("key"))
	defer client.Close()

	wf, err := client.Workflows().Get(context.Background(), "wf-1")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if wf.Name != "My Workflow" {
		t.Errorf("Name = %q", wf.Name)
	}
}

func TestWorkflows_List(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" || r.URL.Path != "/api/v1/service/workflows" {
			t.Errorf("unexpected: %s %s", r.Method, r.URL.Path)
		}
		json.NewEncoder(w).Encode(map[string]any{
			"workflows": []any{
				map[string]any{"id": "wf-1", "name": "First"},
				map[string]any{"id": "wf-2", "name": "Second"},
			},
			"total": 2,
		})
	}))
	defer server.Close()

	client, _ := mbflow.NewClient(mbflow.WithHTTP(server.URL), mbflow.WithSystemKey("key"))
	defer client.Close()

	page, err := client.Workflows().List(context.Background(), &models.ListOptions{Limit: 10})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(page.Items) != 2 {
		t.Errorf("Items len = %d, want 2", len(page.Items))
	}
	if page.Total != 2 {
		t.Errorf("Total = %d, want 2", page.Total)
	}
}

func TestWorkflows_Delete(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "DELETE" || r.URL.Path != "/api/v1/service/workflows/wf-1" {
			t.Errorf("unexpected: %s %s", r.Method, r.URL.Path)
		}
		w.WriteHeader(204)
	}))
	defer server.Close()

	client, _ := mbflow.NewClient(mbflow.WithHTTP(server.URL), mbflow.WithSystemKey("key"))
	defer client.Close()

	err := client.Workflows().Delete(context.Background(), "wf-1")
	if err != nil {
		t.Fatalf("Delete: %v", err)
	}
}

func TestWorkflows_ErrorHandling(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(404)
		json.NewEncoder(w).Encode(map[string]any{"code": "not_found", "message": "workflow not found"})
	}))
	defer server.Close()

	client, _ := mbflow.NewClient(mbflow.WithHTTP(server.URL), mbflow.WithSystemKey("key"))
	defer client.Close()

	_, err := client.Workflows().Get(context.Background(), "nonexistent")
	if err == nil {
		t.Fatal("expected error")
	}

	var apiErr *mbflow.APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected APIError, got %T: %v", err, err)
	}
	if apiErr.StatusCode != 404 {
		t.Errorf("StatusCode = %d, want 404", apiErr.StatusCode)
	}
}
