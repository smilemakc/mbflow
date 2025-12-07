package builtin

import (
	"context"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/smilemakc/mbflow/pkg/executor"
)

// Base64ToBytesExecutor decodes base64 string to bytes
type Base64ToBytesExecutor struct {
	*executor.BaseExecutor
}

// NewBase64ToBytesExecutor creates a new base64 to bytes executor
func NewBase64ToBytesExecutor() *Base64ToBytesExecutor {
	return &Base64ToBytesExecutor{
		BaseExecutor: executor.NewBaseExecutor("base64_to_bytes"),
	}
}

// Execute decodes base64 string to bytes
//
// Config:
//   - encoding: "standard" | "url" | "raw_standard" | "raw_url" (default: "standard")
//   - output_format: "raw" | "hex" (default: "raw")
//
// Input: base64 string
//
// Output:
//   - success: true
//   - result: decoded bytes (as array) or hex string
//   - encoding: encoding used
//   - decoded_size: size in bytes
//   - duration_ms: execution time
func (e *Base64ToBytesExecutor) Execute(ctx context.Context, config map[string]interface{}, input interface{}) (interface{}, error) {
	startTime := time.Now()

	// Get configuration
	encoding := e.GetStringDefault(config, "encoding", "standard")
	outputFormat := e.GetStringDefault(config, "output_format", "raw")

	// Extract base64 string from input
	base64Str, err := e.extractBase64String(input)
	if err != nil {
		return nil, fmt.Errorf("base64_to_bytes: %w", err)
	}

	// Select decoder based on encoding
	var decoder *base64.Encoding
	switch encoding {
	case "standard":
		decoder = base64.StdEncoding
	case "url":
		decoder = base64.URLEncoding
	case "raw_standard":
		decoder = base64.RawStdEncoding
	case "raw_url":
		decoder = base64.RawURLEncoding
	default:
		// Try auto-detection
		decoder, err = e.detectEncoding(base64Str)
		if err != nil {
			return nil, fmt.Errorf("base64_to_bytes: failed to auto-detect encoding: %w", err)
		}
	}

	// Decode
	decoded, err := decoder.DecodeString(base64Str)
	if err != nil {
		return nil, fmt.Errorf("base64_to_bytes: decoding failed: %w", err)
	}

	// Format output
	var result interface{}
	switch outputFormat {
	case "hex":
		result = hex.EncodeToString(decoded)
	case "raw":
		// Convert to byte array for JSON serialization
		result = decoded
	default:
		return nil, fmt.Errorf("base64_to_bytes: invalid output_format: %s", outputFormat)
	}

	return map[string]interface{}{
		"success":      true,
		"result":       result,
		"encoding":     encoding,
		"decoded_size": len(decoded),
		"format":       outputFormat,
		"duration_ms":  time.Since(startTime).Milliseconds(),
	}, nil
}

// Validate validates the configuration
func (e *Base64ToBytesExecutor) Validate(config map[string]interface{}) error {
	// Encoding validation
	encoding := e.GetStringDefault(config, "encoding", "standard")
	validEncodings := map[string]bool{
		"standard":     true,
		"url":          true,
		"raw_standard": true,
		"raw_url":      true,
	}
	if !validEncodings[encoding] {
		return fmt.Errorf("invalid encoding: %s (must be: standard, url, raw_standard, raw_url)", encoding)
	}

	// Output format validation
	outputFormat := e.GetStringDefault(config, "output_format", "raw")
	validFormats := map[string]bool{
		"raw": true,
		"hex": true,
	}
	if !validFormats[outputFormat] {
		return fmt.Errorf("invalid output_format: %s (must be: raw, hex)", outputFormat)
	}

	return nil
}

// extractBase64String extracts base64 string from various input types
func (e *Base64ToBytesExecutor) extractBase64String(input interface{}) (string, error) {
	switch v := input.(type) {
	case string:
		return strings.TrimSpace(v), nil
	case map[string]interface{}:
		// Try to extract from "data" or "base64" field
		if data, ok := v["data"].(string); ok {
			return strings.TrimSpace(data), nil
		}
		if base64Data, ok := v["base64"].(string); ok {
			return strings.TrimSpace(base64Data), nil
		}
		return "", fmt.Errorf("expected 'data' or 'base64' field in input map")
	default:
		return "", fmt.Errorf("unsupported input type: %T (expected string or map)", input)
	}
}

// detectEncoding tries to auto-detect base64 encoding
func (e *Base64ToBytesExecutor) detectEncoding(s string) (*base64.Encoding, error) {
	// Try all encodings
	encodings := []*base64.Encoding{
		base64.StdEncoding,
		base64.URLEncoding,
		base64.RawStdEncoding,
		base64.RawURLEncoding,
	}

	for _, enc := range encodings {
		if _, err := enc.DecodeString(s); err == nil {
			return enc, nil
		}
	}

	return nil, fmt.Errorf("could not detect valid base64 encoding")
}

// BytesToBase64Executor encodes bytes to base64 string
type BytesToBase64Executor struct {
	*executor.BaseExecutor
}

// NewBytesToBase64Executor creates a new bytes to base64 executor
func NewBytesToBase64Executor() *BytesToBase64Executor {
	return &BytesToBase64Executor{
		BaseExecutor: executor.NewBaseExecutor("bytes_to_base64"),
	}
}

// Execute encodes bytes to base64 string
//
// Config:
//   - encoding: "standard" | "url" | "raw_standard" | "raw_url" (default: "standard")
//   - line_length: line wrapping (0 = no wrapping, 76 = MIME format) (default: 0)
//
// Input: bytes ([]byte, string, or map with "data" field)
//
// Output:
//   - success: true
//   - result: base64 encoded string
//   - encoding: encoding used
//   - original_size: original byte size
//   - encoded_size: encoded string size
//   - duration_ms: execution time
func (e *BytesToBase64Executor) Execute(ctx context.Context, config map[string]interface{}, input interface{}) (interface{}, error) {
	startTime := time.Now()

	// Get configuration
	encoding := e.GetStringDefault(config, "encoding", "standard")
	lineLength := e.GetIntDefault(config, "line_length", 0)

	// Extract bytes from input
	data, err := e.extractBytes(input)
	if err != nil {
		return nil, fmt.Errorf("bytes_to_base64: %w", err)
	}

	// Select encoder
	var encoder *base64.Encoding
	switch encoding {
	case "standard":
		encoder = base64.StdEncoding
	case "url":
		encoder = base64.URLEncoding
	case "raw_standard":
		encoder = base64.RawStdEncoding
	case "raw_url":
		encoder = base64.RawURLEncoding
	default:
		return nil, fmt.Errorf("bytes_to_base64: invalid encoding: %s", encoding)
	}

	// Encode
	encoded := encoder.EncodeToString(data)

	// Apply line wrapping if requested
	if lineLength > 0 {
		encoded = e.wrapLines(encoded, lineLength)
	}

	return map[string]interface{}{
		"success":       true,
		"result":        encoded,
		"encoding":      encoding,
		"original_size": len(data),
		"encoded_size":  len(encoded),
		"duration_ms":   time.Since(startTime).Milliseconds(),
	}, nil
}

// Validate validates the configuration
func (e *BytesToBase64Executor) Validate(config map[string]interface{}) error {
	// Encoding validation
	encoding := e.GetStringDefault(config, "encoding", "standard")
	validEncodings := map[string]bool{
		"standard":     true,
		"url":          true,
		"raw_standard": true,
		"raw_url":      true,
	}
	if !validEncodings[encoding] {
		return fmt.Errorf("invalid encoding: %s (must be: standard, url, raw_standard, raw_url)", encoding)
	}

	// Line length validation
	lineLength := e.GetIntDefault(config, "line_length", 0)
	if lineLength < 0 {
		return fmt.Errorf("line_length must be >= 0")
	}

	return nil
}

// extractBytes extracts bytes from various input types
func (e *BytesToBase64Executor) extractBytes(input interface{}) ([]byte, error) {
	switch v := input.(type) {
	case []byte:
		return v, nil
	case string:
		// Try to decode as base64 first, if fails use as raw bytes
		if decoded, err := base64.StdEncoding.DecodeString(v); err == nil {
			return decoded, nil
		}
		// Use string as UTF-8 bytes
		return []byte(v), nil
	case map[string]interface{}:
		// Try to extract from "data" field
		if data, ok := v["data"]; ok {
			return e.extractBytes(data)
		}
		return nil, fmt.Errorf("expected 'data' field in input map")
	default:
		return nil, fmt.Errorf("unsupported input type: %T (expected []byte, string, or map)", input)
	}
}

// wrapLines wraps encoded string to specified line length
func (e *BytesToBase64Executor) wrapLines(s string, lineLength int) string {
	if lineLength <= 0 || len(s) <= lineLength {
		return s
	}

	var result strings.Builder
	for i := 0; i < len(s); i += lineLength {
		end := i + lineLength
		if end > len(s) {
			end = len(s)
		}
		result.WriteString(s[i:end])
		if end < len(s) {
			result.WriteString("\n")
		}
	}
	return result.String()
}
