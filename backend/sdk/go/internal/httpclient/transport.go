package httpclient

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/smilemakc/mbflow/sdk/go/internal"
)

// Config configures the HTTP transport.
type Config struct {
	APIKey     string
	SystemKey  string
	OnBehalfOf string
	Timeout    time.Duration
	HTTPClient *http.Client
}

type transport struct {
	baseURL    string
	config     *Config
	httpClient *http.Client
}

// New creates a new HTTP transport.
// baseURL is the MBFlow server URL (e.g. "http://localhost:8585").
func New(baseURL string, config *Config) internal.Transport {
	client := config.HTTPClient
	if client == nil {
		client = &http.Client{
			Timeout: config.Timeout,
		}
	}
	return &transport{
		baseURL:    strings.TrimRight(baseURL, "/"),
		config:     config,
		httpClient: client,
	}
}

func (t *transport) Do(ctx context.Context, req *internal.Request) (*internal.Response, error) {
	basePath := "/api/v1/service"
	if t.config.APIKey != "" && t.config.SystemKey == "" {
		basePath = "/api/v1"
	}
	url := t.baseURL + basePath + req.Path

	if len(req.Query) > 0 {
		params := make([]string, 0, len(req.Query))
		for k, v := range req.Query {
			params = append(params, fmt.Sprintf("%s=%s", k, v))
		}
		url += "?" + strings.Join(params, "&")
	}

	var bodyReader *bytes.Reader
	if req.Body != nil {
		data, err := json.Marshal(req.Body)
		if err != nil {
			return nil, fmt.Errorf("marshal request body: %w", err)
		}
		bodyReader = bytes.NewReader(data)
	}

	var httpReq *http.Request
	var err error
	if bodyReader != nil {
		httpReq, err = http.NewRequestWithContext(ctx, string(req.Method), url, bodyReader)
	} else {
		httpReq, err = http.NewRequestWithContext(ctx, string(req.Method), url, nil)
	}
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")

	if t.config.SystemKey != "" {
		httpReq.Header.Set("X-System-Key", t.config.SystemKey)
	}
	if t.config.APIKey != "" {
		httpReq.Header.Set("Authorization", "Bearer "+t.config.APIKey)
	}
	if t.config.OnBehalfOf != "" {
		httpReq.Header.Set("X-On-Behalf-Of", t.config.OnBehalfOf)
	}

	for k, v := range req.Headers {
		httpReq.Header.Set(k, v)
	}

	resp, err := t.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("http request: %w", err)
	}

	return &internal.Response{
		StatusCode: resp.StatusCode,
		Body:       resp.Body,
	}, nil
}

func (t *transport) Close() error {
	return nil
}
