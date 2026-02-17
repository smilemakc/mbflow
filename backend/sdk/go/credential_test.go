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

func TestCredentials_Create(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" || r.URL.Path != "/api/v1/service/credentials" {
			t.Errorf("unexpected: %s %s", r.Method, r.URL.Path)
		}
		w.WriteHeader(201)
		json.NewEncoder(w).Encode(map[string]any{"id": "cred-1", "name": "API Key", "credential_type": "api_key"})
	}))
	defer server.Close()

	client, _ := mbflow.NewClient(mbflow.WithHTTP(server.URL), mbflow.WithSystemKey("key"))
	defer client.Close()

	cred, err := client.Credentials().Create(context.Background(), &models.Credential{Name: "API Key", CredentialType: "api_key"})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if cred.ID != "cred-1" {
		t.Errorf("ID = %q", cred.ID)
	}
}

func TestCredentials_List(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{
			"credentials": []any{map[string]any{"id": "cred-1", "name": "Test"}},
			"total":       1,
		})
	}))
	defer server.Close()

	client, _ := mbflow.NewClient(mbflow.WithHTTP(server.URL), mbflow.WithSystemKey("key"))
	page, err := client.Credentials().List(context.Background(), nil)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if page.Total != 1 {
		t.Errorf("Total = %d", page.Total)
	}
}
