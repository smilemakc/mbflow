package builtin

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/smilemakc/mbflow/pkg/executor"
)

// StringToJsonExecutor parses JSON string to object
type StringToJsonExecutor struct {
	*executor.BaseExecutor
}

// NewStringToJsonExecutor creates a new string to JSON executor
func NewStringToJsonExecutor() *StringToJsonExecutor {
	return &StringToJsonExecutor{
		BaseExecutor: executor.NewBaseExecutor("string_to_json"),
	}
}

// Execute parses JSON string
//
// Config:
//   - strict_mode: fail on invalid JSON vs return null (default: true)
//   - trim_whitespace: trim whitespace before parsing (default: true)
//
// Input: JSON string
//
// Output:
//   - success: true
//   - result: parsed JSON object/array
//   - string_length: original string length
//   - duration_ms: execution time
func (e *StringToJsonExecutor) Execute(ctx context.Context, config map[string]interface{}, input interface{}) (interface{}, error) {
	startTime := time.Now()

	// Get configuration
	strictMode := e.GetBoolDefault(config, "strict_mode", true)
	trimWhitespace := e.GetBoolDefault(config, "trim_whitespace", true)

	// Extract string from input
	jsonStr, err := e.extractString(input)
	if err != nil {
		return nil, fmt.Errorf("string_to_json: %w", err)
	}

	originalLength := len(jsonStr)

	// Trim whitespace if configured
	if trimWhitespace {
		jsonStr = strings.TrimSpace(jsonStr)
	}

	// Parse JSON
	var result interface{}
	decoder := json.NewDecoder(strings.NewReader(jsonStr))
	decoder.UseNumber() // Preserve number precision

	err = decoder.Decode(&result)
	if err != nil {
		if strictMode {
			return nil, fmt.Errorf("string_to_json: JSON parsing failed: %w", err)
		}
		// In non-strict mode, return null on error
		result = nil
	}

	return map[string]interface{}{
		"success":       true,
		"result":        result,
		"string_length": originalLength,
		"duration_ms":   time.Since(startTime).Milliseconds(),
	}, nil
}

// Validate validates the configuration
func (e *StringToJsonExecutor) Validate(config map[string]interface{}) error {
	// No specific validation needed - all config fields have defaults
	return nil
}

// extractString extracts string from various input types
func (e *StringToJsonExecutor) extractString(input interface{}) (string, error) {
	switch v := input.(type) {
	case string:
		return v, nil
	case []byte:
		return string(v), nil
	case map[string]interface{}:
		// Try to extract from "data" or "json" field
		if data, ok := v["data"].(string); ok {
			return data, nil
		}
		if jsonData, ok := v["json"].(string); ok {
			return jsonData, nil
		}
		return "", fmt.Errorf("expected 'data' or 'json' field in input map")
	default:
		return "", fmt.Errorf("unsupported input type: %T (expected string, []byte, or map)", input)
	}
}

// JsonToStringExecutor serializes JSON to string
type JsonToStringExecutor struct {
	*executor.BaseExecutor
}

// NewJsonToStringExecutor creates a new JSON to string executor
func NewJsonToStringExecutor() *JsonToStringExecutor {
	return &JsonToStringExecutor{
		BaseExecutor: executor.NewBaseExecutor("json_to_string"),
	}
}

// Execute serializes JSON to string
//
// Config:
//   - pretty: pretty-print with indentation (default: false)
//   - indent: indent string (default: "  ") - only used if pretty=true
//   - escape_html: escape HTML characters (default: true)
//   - sort_keys: sort object keys alphabetically (default: false)
//
// Input: JSON object/array
//
// Output:
//   - success: true
//   - result: JSON string
//   - string_length: resulting string length
//   - pretty: whether output is pretty-printed
//   - duration_ms: execution time
func (e *JsonToStringExecutor) Execute(ctx context.Context, config map[string]interface{}, input interface{}) (interface{}, error) {
	startTime := time.Now()

	// Get configuration
	pretty := e.GetBoolDefault(config, "pretty", false)
	indent := e.GetStringDefault(config, "indent", "  ")
	escapeHTML := e.GetBoolDefault(config, "escape_html", true)
	sortKeys := e.GetBoolDefault(config, "sort_keys", false)

	// Sort keys if requested
	var data interface{} = input
	if sortKeys {
		data = e.sortMapKeys(input)
	}

	// Marshal JSON
	var result string
	var err error

	if pretty {
		result, err = e.marshalPretty(data, indent, escapeHTML)
	} else {
		result, err = e.marshalCompact(data, escapeHTML)
	}

	if err != nil {
		return nil, fmt.Errorf("json_to_string: serialization failed: %w", err)
	}

	return map[string]interface{}{
		"success":       true,
		"result":        result,
		"string_length": len(result),
		"pretty":        pretty,
		"duration_ms":   time.Since(startTime).Milliseconds(),
	}, nil
}

// Validate validates the configuration
func (e *JsonToStringExecutor) Validate(config map[string]interface{}) error {
	// No specific validation needed - all config fields have defaults
	return nil
}

// marshalCompact marshals JSON without indentation
func (e *JsonToStringExecutor) marshalCompact(data interface{}, escapeHTML bool) (string, error) {
	var buf bytes.Buffer
	encoder := json.NewEncoder(&buf)
	encoder.SetEscapeHTML(escapeHTML)

	if err := encoder.Encode(data); err != nil {
		return "", err
	}

	// Remove trailing newline added by Encoder
	result := buf.String()
	return strings.TrimSuffix(result, "\n"), nil
}

// marshalPretty marshals JSON with indentation
func (e *JsonToStringExecutor) marshalPretty(data interface{}, indent string, escapeHTML bool) (string, error) {
	var buf bytes.Buffer
	encoder := json.NewEncoder(&buf)
	encoder.SetEscapeHTML(escapeHTML)
	encoder.SetIndent("", indent)

	if err := encoder.Encode(data); err != nil {
		return "", err
	}

	// Remove trailing newline added by Encoder
	result := buf.String()
	return strings.TrimSuffix(result, "\n"), nil
}

// sortMapKeys recursively sorts map keys
func (e *JsonToStringExecutor) sortMapKeys(data interface{}) interface{} {
	switch v := data.(type) {
	case map[string]interface{}:
		// Create sorted map
		keys := make([]string, 0, len(v))
		for k := range v {
			keys = append(keys, k)
		}
		sort.Strings(keys)

		sorted := make(map[string]interface{}, len(v))
		for _, k := range keys {
			sorted[k] = e.sortMapKeys(v[k]) // Recursively sort nested maps
		}
		return sorted

	case []interface{}:
		// Recursively sort maps in arrays
		sorted := make([]interface{}, len(v))
		for i, item := range v {
			sorted[i] = e.sortMapKeys(item)
		}
		return sorted

	default:
		return data
	}
}
