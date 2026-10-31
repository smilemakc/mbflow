package sdk

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

// ErrNotSupportedByAPI is returned (wrapped) by remote-mode operations that the
// MBFlow REST API does not provide, or that the server reports as not
// implemented (HTTP 501). Use errors.Is(err, sdk.ErrNotSupportedByAPI).
var ErrNotSupportedByAPI = errors.New("operation not supported by the MBFlow REST API")

// UnsupportedOperationError describes an SDK operation that has no
// corresponding REST API endpoint.
type UnsupportedOperationError struct {
	Operation string
	Reason    string
}

func (e *UnsupportedOperationError) Error() string {
	if e.Reason == "" {
		return fmt.Sprintf("%s: %s", e.Operation, ErrNotSupportedByAPI.Error())
	}
	return fmt.Sprintf("%s: %s (%s)", e.Operation, ErrNotSupportedByAPI.Error(), e.Reason)
}

// Unwrap allows errors.Is(err, ErrNotSupportedByAPI).
func (e *UnsupportedOperationError) Unwrap() error { return ErrNotSupportedByAPI }

func errUnsupported(operation, reason string) error {
	return &UnsupportedOperationError{Operation: operation, Reason: reason}
}

// APIError is returned by remote-mode operations when the MBFlow server
// responds with a non-2xx status code.
type APIError struct {
	StatusCode int            `json:"-"`
	Code       string         `json:"code"`
	Message    string         `json:"message"`
	Details    map[string]any `json:"details,omitempty"`
	// Body holds the raw response body when it was not a structured API error.
	Body string `json:"-"`
}

func (e *APIError) Error() string {
	switch {
	case e.Code != "" || e.Message != "":
		return fmt.Sprintf("API error (%d) %s: %s", e.StatusCode, e.Code, e.Message)
	case e.Body != "":
		return fmt.Sprintf("API error (%d): %s", e.StatusCode, e.Body)
	default:
		return fmt.Sprintf("API error (%d)", e.StatusCode)
	}
}

// Is reports HTTP 501 responses as ErrNotSupportedByAPI.
func (e *APIError) Is(target error) bool {
	return target == ErrNotSupportedByAPI && e.StatusCode == http.StatusNotImplemented
}

// newAPIError builds an APIError from a failed response.
func newAPIError(resp *http.Response) *APIError {
	apiErr := &APIError{StatusCode: resp.StatusCode}
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if len(body) == 0 {
		return apiErr
	}
	if err := json.Unmarshal(body, apiErr); err != nil || (apiErr.Code == "" && apiErr.Message == "") {
		apiErr.Code, apiErr.Message, apiErr.Details = "", "", nil
		apiErr.Body = strings.TrimSpace(string(body))
	}
	return apiErr
}

// doRemote performs an authenticated JSON request against the remote API.
// path is relative to BaseURL (e.g. "/api/v1/workflows"). When out is non-nil
// the 2xx response body is decoded into it.
func (c *Client) doRemote(ctx context.Context, method, path string, query url.Values, body, out any) error {
	u := strings.TrimRight(c.config.BaseURL, "/") + path
	if len(query) > 0 {
		u += "?" + query.Encode()
	}

	var reader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("failed to marshal request: %w", err)
		}
		reader = bytes.NewReader(data)
	}

	req, err := http.NewRequestWithContext(ctx, method, u, reader)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if c.config.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.config.APIKey)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return newAPIError(resp)
	}

	if out == nil {
		_, _ = io.Copy(io.Discard, resp.Body)
		return nil
	}
	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}
	return nil
}

// listEnvelope matches the REST API's paginated list response
// ({"data": [...], "total": N, "limit": L, "offset": O}). legacyKey allows
// older servers that used a resource-named key (e.g. "executions").
func decodeListEnvelope[T any](raw map[string]json.RawMessage, legacyKey string) ([]T, error) {
	data, ok := raw["data"]
	if !ok && legacyKey != "" {
		data, ok = raw[legacyKey]
	}
	if !ok || len(data) == 0 || string(data) == "null" {
		return []T{}, nil
	}
	var items []T
	if err := json.Unmarshal(data, &items); err != nil {
		return nil, fmt.Errorf("failed to decode list items: %w", err)
	}
	return items, nil
}

// pathEscape escapes an identifier for use as a single URL path segment.
func pathEscape(id string) string {
	return url.PathEscape(id)
}

// setPagination adds limit/offset query parameters when positive.
func setPagination(q url.Values, limit, offset int) {
	if limit > 0 {
		q.Set("limit", fmt.Sprintf("%d", limit))
	}
	if offset > 0 {
		q.Set("offset", fmt.Sprintf("%d", offset))
	}
}
