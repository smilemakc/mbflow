package builtin

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"golang.org/x/text/encoding/unicode"
	"golang.org/x/text/transform"

	"github.com/smilemakc/mbflow/go/pkg/executor"
)

// BytesToJsonExecutor decodes bytes to JSON object
type BytesToJsonExecutor struct {
	*executor.BaseExecutor
}

// NewBytesToJsonExecutor creates a new bytes to JSON executor
func NewBytesToJsonExecutor() *BytesToJsonExecutor {
	return &BytesToJsonExecutor{
		BaseExecutor: executor.NewBaseExecutor("bytes_to_json"),
	}
}

// Execute decodes bytes to JSON
//
// Config:
//   - encoding: "utf-8" | "utf-16" | "latin1" (default: "utf-8")
//   - validate_json: validate JSON structure (default: true)
//
// Input: bytes ([]byte, string, or map with "data" field)
//
// Output:
//   - success: true
//   - result: parsed JSON object/array
//   - encoding_used: actual encoding detected/used
//   - byte_size: original byte size
//   - duration_ms: execution time
func (e *BytesToJsonExecutor) Execute(ctx context.Context, config map[string]any, input any) (any, error) {
	startTime := time.Now()

	// Get configuration
	encoding := e.GetStringDefault(config, "encoding", "utf-8")
	validateJSON := e.GetBoolDefault(config, "validate_json", true)

	// Extract bytes from input
	data, err := e.extractBytes(input)
	if err != nil {
		return nil, fmt.Errorf("bytes_to_json: %w", err)
	}

	originalSize := len(data)

	// Detect encoding if needed
	actualEncoding := encoding
	if encoding == "utf-8" {
		// Auto-detect UTF-8 BOM or UTF-16
		detected := e.detectEncoding(data)
		if detected != "" {
			actualEncoding = detected
		}
	}

	// Decode to string
	var jsonStr string
	switch actualEncoding {
	case "utf-8":
		// Remove UTF-8 BOM if present
		if len(data) >= 3 && data[0] == 0xEF && data[1] == 0xBB && data[2] == 0xBF {
			data = data[3:]
		}
		jsonStr = string(data)

	case "utf-16":
		// Decode UTF-16
		decoder := unicode.UTF16(unicode.LittleEndian, unicode.UseBOM).NewDecoder()
		decoded, _, err := transform.Bytes(decoder, data)
		if err != nil {
			return nil, fmt.Errorf("bytes_to_json: UTF-16 decoding failed: %w", err)
		}
		jsonStr = string(decoded)

	case "latin1":
		// Latin1 (ISO-8859-1) - direct byte to rune conversion
		runes := make([]rune, len(data))
		for i, b := range data {
			runes[i] = rune(b)
		}
		jsonStr = string(runes)

	default:
		return nil, fmt.Errorf("bytes_to_json: unsupported encoding: %s", actualEncoding)
	}

	// Parse JSON
	var result any
	decoder := json.NewDecoder(strings.NewReader(jsonStr))
	decoder.UseNumber() // Preserve number precision

	err = decoder.Decode(&result)
	if err != nil {
		if validateJSON {
			return nil, fmt.Errorf("bytes_to_json: JSON parsing failed: %w", err)
		}
		// In non-validate mode, return null
		result = nil
	}

	return map[string]any{
		"success":       true,
		"result":        result,
		"encoding_used": actualEncoding,
		"byte_size":     originalSize,
		"duration_ms":   time.Since(startTime).Milliseconds(),
	}, nil
}

// Validate validates the configuration
func (e *BytesToJsonExecutor) Validate(config map[string]any) error {
	// Encoding validation
	encoding := e.GetStringDefault(config, "encoding", "utf-8")
	validEncodings := map[string]bool{
		"utf-8":  true,
		"utf-16": true,
		"latin1": true,
	}
	if !validEncodings[encoding] {
		return fmt.Errorf("invalid encoding: %s (must be: utf-8, utf-16, latin1)", encoding)
	}

	return nil
}

// extractBytes extracts bytes from various input types
func (e *BytesToJsonExecutor) extractBytes(input any) ([]byte, error) {
	switch v := input.(type) {
	case []byte:
		return v, nil
	case string:
		// Try to decode as base64 first, if fails use as raw bytes
		if decoded, err := base64.StdEncoding.DecodeString(v); err == nil {
			// Check if it looks like base64 (not just any string)
			if len(v) > 0 && len(v)%4 == 0 {
				return decoded, nil
			}
		}
		// Use string as UTF-8 bytes
		return []byte(v), nil
	case map[string]any:
		// Try to extract from "data" field
		if data, ok := v["data"]; ok {
			return e.extractBytes(data)
		}
		return nil, fmt.Errorf("expected 'data' field in input map")
	default:
		return nil, fmt.Errorf("unsupported input type: %T (expected []byte, string, or map)", input)
	}
}

// detectEncoding detects encoding from BOM or content analysis
func (e *BytesToJsonExecutor) detectEncoding(data []byte) string {
	if len(data) == 0 {
		return ""
	}

	// Check for UTF-8 BOM
	if len(data) >= 3 && data[0] == 0xEF && data[1] == 0xBB && data[2] == 0xBF {
		return "utf-8"
	}

	// Check for UTF-16 BOM
	if len(data) >= 2 {
		if data[0] == 0xFF && data[1] == 0xFE {
			return "utf-16" // UTF-16 LE
		}
		if data[0] == 0xFE && data[1] == 0xFF {
			return "utf-16" // UTF-16 BE
		}
	}

	// Validate UTF-8
	if utf8.Valid(data) {
		return "utf-8"
	}

	// If not valid UTF-8 and no BOM, let caller handle it
	return ""
}
