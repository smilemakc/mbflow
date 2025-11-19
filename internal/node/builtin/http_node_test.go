package builtin

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	core "mbflow/internal/node"

	"github.com/stretchr/testify/assert"
)

type priceResp struct {
	Total float64 `json:"total"`
}

func TestHTTPRequestNode_JSONAdapter_Placeholders_Metadata(t *testing.T) {
	// test server
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/price/SKU-1" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"total": 12.5})
	}))
	defer srv.Close()

	n := NewHTTPRequestNodeBuilder[priceResp]().
		ID("http-1").Name("HTTP").Version("1.0").
		Method(http.MethodGet).
		URL(srv.URL+"/price/{sku}").
		Header("X-Trace", "{trace}").
		Timeout(2 * time.Second).
		Adapter(&JSONResponseAdapter[priceResp]{}).
		Build()

	out, err := n.Execute(context.Background(), core.NodeInput{Metadata: map[string]string{"sku": "SKU-1", "trace": "t-1"}})
	assert.NoError(t, err)
	pr := out.Data.(priceResp)
	assert.Equal(t, 12.5, pr.Total)
	assert.Equal(t, "200", out.Metadata["http.status_code"])
	assert.NotEmpty(t, out.Metadata["http.header.content-type"])
}

func TestHTTPRequestNode_FailOnStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "bad", http.StatusBadRequest)
	}))
	defer srv.Close()
	n := NewHTTPRequestNodeBuilder[string]().
		ID("http-2").Name("HTTP").Version("1.0").
		Method(http.MethodGet).
		URL(srv.URL).
		Adapter(&StringAdapter{}).
		FailOnStatus(func(code int) bool { return code >= 400 }).
		Build()
	_, err := n.Execute(context.Background(), core.NodeInput{})
	assert.Error(t, err)
}
