package mbflow_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	mbflow "github.com/smilemakc/mbflow/sdk/go"
	"github.com/smilemakc/mbflow/sdk/go/models"
)

func TestTriggers_Create(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" || r.URL.Path != "/api/v1/service/triggers" {
			t.Errorf("unexpected: %s %s", r.Method, r.URL.Path)
		}
		w.WriteHeader(201)
		json.NewEncoder(w).Encode(map[string]any{"id": "tr-1", "name": "My Trigger", "type": "cron"})
	}))
	defer server.Close()

	client, _ := mbflow.NewClient(mbflow.WithHTTP(server.URL), mbflow.WithSystemKey("key"))
	defer client.Close()

	tr, err := client.Triggers().Create(context.Background(), &models.Trigger{Name: "My Trigger", Type: models.TriggerTypeCron})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if tr.ID != "tr-1" {
		t.Errorf("ID = %q", tr.ID)
	}
}

func TestTriggers_Delete(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "DELETE" || r.URL.Path != "/api/v1/service/triggers/tr-1" {
			t.Errorf("unexpected: %s %s", r.Method, r.URL.Path)
		}
		w.WriteHeader(204)
	}))
	defer server.Close()

	client, _ := mbflow.NewClient(mbflow.WithHTTP(server.URL), mbflow.WithSystemKey("key"))
	err := client.Triggers().Delete(context.Background(), "tr-1")
	if err != nil {
		t.Fatalf("Delete: %v", err)
	}
}
