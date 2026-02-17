package internal

import (
	"encoding/json"
	"fmt"
	"io"
)

// DecodeResponse reads and decodes a JSON response body into the target type.
func DecodeResponse[T any](body io.ReadCloser) (*T, error) {
	defer body.Close()
	var result T
	if err := json.NewDecoder(body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	return &result, nil
}

// DecodeListResponse decodes a list response with format {"<key>": [...], "total": N}.
func DecodeListResponse[T any](body io.ReadCloser, key string) ([]*T, int, error) {
	defer body.Close()
	var raw map[string]json.RawMessage
	if err := json.NewDecoder(body).Decode(&raw); err != nil {
		return nil, 0, fmt.Errorf("decode list response: %w", err)
	}
	var items []*T
	if data, ok := raw[key]; ok {
		if err := json.Unmarshal(data, &items); err != nil {
			return nil, 0, fmt.Errorf("decode items: %w", err)
		}
	}
	var total int
	if data, ok := raw["total"]; ok {
		json.Unmarshal(data, &total)
	}
	return items, total, nil
}

// ErrorInfo holds parsed error information from an HTTP response.
type ErrorInfo struct {
	StatusCode int
	Code       string
	Message    string
	Details    map[string]interface{}
}

// ParseErrorResponse reads and parses an error response body.
func ParseErrorResponse(statusCode int, body io.ReadCloser) *ErrorInfo {
	defer body.Close()
	var resp struct {
		Code    string                 `json:"code"`
		Message string                 `json:"message"`
		Details map[string]interface{} `json:"details"`
	}
	json.NewDecoder(body).Decode(&resp)
	return &ErrorInfo{
		StatusCode: statusCode,
		Code:       resp.Code,
		Message:    resp.Message,
		Details:    resp.Details,
	}
}
