package trigger

import (
	"bytes"
	"context"
	"encoding/json"
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHTTPTrigger_Handler_Basic(t *testing.T) {
	tr := NewHTTP(HTTPConfig{Path: "/x", Method: http.MethodPost})
	h := tr.Handler(func(ctx context.Context, payload map[string]any) (int, any) {
		return http.StatusOK, map[string]any{"ok": true, "p": payload["a"]}
	})
	rr := httptest.NewRecorder()
	body, _ := json.Marshal(map[string]any{"a": 5})
	req := httptest.NewRequest(http.MethodPost, "/x", bytes.NewReader(body))
	h(rr, req)
	assert.Equal(t, http.StatusOK, rr.Code)
}
