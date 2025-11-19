package trigger

import (
	"context"
	"encoding/json"
	"net/http"
)

type HTTPConfig struct {
	Path   string
	Method string
}

type HTTPTrigger struct {
	cfg HTTPConfig
}

func NewHTTP(cfg HTTPConfig) *HTTPTrigger { return &HTTPTrigger{cfg: cfg} }

func (t *HTTPTrigger) Handler(fn func(ctx context.Context, payload map[string]any) (int, any)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if t.cfg.Method != "" && r.Method != t.cfg.Method {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		var payload map[string]any
		if r.Body != nil {
			_ = json.NewDecoder(r.Body).Decode(&payload)
		}
		ctx := r.Context()
		status, resp := fn(ctx, payload)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_ = json.NewEncoder(w).Encode(resp)
	}
}
