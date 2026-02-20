package httpclient_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/smilemakc/mbflow/go/sdk/internal"
	"github.com/smilemakc/mbflow/go/sdk/internal/httpclient"
)

func TestHTTPTransportDo_GET(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("method = %s, want GET", r.Method)
		}
		if r.URL.Path != "/api/v1/service/workflows" {
			t.Errorf("path = %s, want /api/v1/service/workflows", r.URL.Path)
		}
		if r.Header.Get("X-System-Key") != "test-key" {
			t.Errorf("X-System-Key = %q", r.Header.Get("X-System-Key"))
		}
		w.WriteHeader(200)
		json.NewEncoder(w).Encode(map[string]any{"id": "wf-1"})
	}))
	defer server.Close()

	tr := httpclient.New(server.URL, &httpclient.Config{
		SystemKey: "test-key",
	})
	defer tr.Close()

	resp, err := tr.Do(context.Background(), &internal.Request{
		Method: internal.MethodGet,
		Path:   "/workflows",
	})
	if err != nil {
		t.Fatalf("Do: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		t.Errorf("StatusCode = %d, want 200", resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)
	if len(body) == 0 {
		t.Error("empty body")
	}
}

func TestHTTPTransportDo_POST(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("method = %s, want POST", r.Method)
		}
		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)
		if body["name"] != "Test" {
			t.Errorf("name = %v", body["name"])
		}
		if r.Header.Get("X-On-Behalf-Of") != "user-1" {
			t.Errorf("X-On-Behalf-Of = %q", r.Header.Get("X-On-Behalf-Of"))
		}
		w.WriteHeader(201)
		json.NewEncoder(w).Encode(map[string]any{"id": "wf-new"})
	}))
	defer server.Close()

	tr := httpclient.New(server.URL, &httpclient.Config{
		SystemKey:  "test-key",
		OnBehalfOf: "user-1",
	})

	resp, err := tr.Do(context.Background(), &internal.Request{
		Method: internal.MethodPost,
		Path:   "/workflows",
		Body:   map[string]any{"name": "Test"},
	})
	if err != nil {
		t.Fatalf("Do: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 201 {
		t.Errorf("StatusCode = %d, want 201", resp.StatusCode)
	}
}

func TestHTTPTransportDo_APIKey(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer sk-test" {
			t.Errorf("Authorization = %q", r.Header.Get("Authorization"))
		}
		if r.URL.Path != "/api/v1/workflows" {
			t.Errorf("path = %s, want /api/v1/workflows", r.URL.Path)
		}
		w.WriteHeader(200)
		w.Write([]byte("{}"))
	}))
	defer server.Close()

	tr := httpclient.New(server.URL, &httpclient.Config{
		APIKey: "sk-test",
	})

	resp, err := tr.Do(context.Background(), &internal.Request{
		Method: internal.MethodGet,
		Path:   "/workflows",
	})
	if err != nil {
		t.Fatalf("Do: %v", err)
	}
	resp.Body.Close()
}
