package builtin

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	core "mbflow/internal/node"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type HTTPRequestConfig struct {
	Method  string
	URL     string
	Headers map[string]string
	Timeout time.Duration
}

// HTTPClient is a minimal http client abstraction for testing/mocking.
type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

// HTTPRequestNode performs an HTTP request and adapts the response to T
// using the provided DataAdapter.
type HTTPRequestNode[T any] struct {
	id      string
	name    string
	version string
	cfg     HTTPRequestConfig
	client  HTTPClient
	adapter core.DataAdapter[*http.Response, T]
	// Optional: decide whether response status is an error
	failOnStatus func(code int) bool
}

func NewHTTPRequestNode[T any](id, name, version string, cfg HTTPRequestConfig, client HTTPClient, adapter core.DataAdapter[*http.Response, T]) *HTTPRequestNode[T] {
	if client == nil {
		client = &http.Client{Timeout: cfg.Timeout}
	}
	return &HTTPRequestNode[T]{id: id, name: name, version: version, cfg: cfg, client: client, adapter: adapter}
}

func (n *HTTPRequestNode[T]) ID() string      { return n.id }
func (n *HTTPRequestNode[T]) Name() string    { return n.name }
func (n *HTTPRequestNode[T]) Version() string { return n.version }

func (n *HTTPRequestNode[T]) Validate(input core.NodeInput) error {
	// For POST/PUT we allow any JSON-serializable body in input.Data
	if n.cfg.Method == "" || n.cfg.URL == "" {
		return errors.New("http node: method and url must be set")
	}
	return nil
}

func (n *HTTPRequestNode[T]) InputSchema() core.Schema  { return core.Schema{"body": "any"} }
func (n *HTTPRequestNode[T]) OutputSchema() core.Schema { return core.Schema{"data": "generic"} }

func (n *HTTPRequestNode[T]) Execute(ctx context.Context, input core.NodeInput) (core.NodeOutput, error) {
	var body io.Reader
	if input.Data != nil {
		// Default: JSON encode input.Data
		buf := new(bytes.Buffer)
		if err := json.NewEncoder(buf).Encode(input.Data); err != nil {
			return core.NodeOutput{}, err
		}
		body = buf
	}
	// Expand placeholders in URL and headers using metadata: {key}
	url := expandPlaceholders(n.cfg.URL, input.Metadata)
	req, err := http.NewRequestWithContext(ctx, n.cfg.Method, url, body)
	if err != nil {
		return core.NodeOutput{}, err
	}
	for k, v := range n.cfg.Headers {
		req.Header.Set(k, expandPlaceholders(v, input.Metadata))
	}
	if req.Header.Get("Content-Type") == "" && body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := n.client.Do(req)
	if err != nil {
		return core.NodeOutput{}, err
	}
	defer resp.Body.Close()

	// Optional status policy
	if n.failOnStatus != nil && n.failOnStatus(resp.StatusCode) {
		// read small snippet of body for error context
		snippet := make([]byte, 512)
		_, _ = io.ReadFull(resp.Body, snippet)
		return core.NodeOutput{}, errors.New("http node: unexpected status " + resp.Status)
	}

	// Adapt response to T
	var out T
	if n.adapter != nil {
		out, err = n.adapter.Adapt(ctx, resp)
		if err != nil {
			return core.NodeOutput{}, err
		}
	}
	// Map status and headers into metadata
	md := map[string]string{
		"http.status_code": itoa(resp.StatusCode),
		"http.status_text": http.StatusText(resp.StatusCode),
	}
	for hk, hv := range resp.Header {
		if len(hv) > 0 {
			md["http.header."+normalizeHeaderKey(hk)] = hv[0]
		}
	}
	return core.NodeOutput{Data: out, Metadata: md}, nil
}

// expandPlaceholders replaces occurrences of {key} in s with values from meta
func expandPlaceholders(s string, meta map[string]string) string {
	if meta == nil || s == "" {
		return s
	}
	out := s
	for k, v := range meta {
		ph := "{" + k + "}"
		out = strings.ReplaceAll(out, ph, v)
	}
	return out
}

func normalizeHeaderKey(k string) string { return strings.ToLower(strings.ReplaceAll(k, " ", "-")) }

func itoa(i int) string {
	return strconv.FormatInt(int64(i), 10)
}
