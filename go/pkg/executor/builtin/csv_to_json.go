package builtin

import (
	"context"
	"encoding/csv"
	"fmt"
	"strings"
	"time"

	"github.com/smilemakc/mbflow/go/pkg/executor"
)

// CSVToJSONExecutor converts CSV data to JSON array of objects.
// It supports various delimiters, custom headers, and processing options.
type CSVToJSONExecutor struct {
	*executor.BaseExecutor
}

// NewCSVToJSONExecutor creates a new CSV to JSON executor.
func NewCSVToJSONExecutor() *CSVToJSONExecutor {
	return &CSVToJSONExecutor{
		BaseExecutor: executor.NewBaseExecutor("csv_to_json"),
	}
}

// Execute converts CSV input to JSON array of objects.
func (e *CSVToJSONExecutor) Execute(_ context.Context, config map[string]any, input any) (any, error) {
	startTime := time.Now()

	// Get config options with defaults
	delimiter := e.GetStringDefault(config, "delimiter", ",")
	hasHeader := e.GetBoolDefault(config, "has_header", true)
	customHeaders := e.getStringSlice(config, "custom_headers")
	trimSpaces := e.GetBoolDefault(config, "trim_spaces", true)
	skipEmptyRows := e.GetBoolDefault(config, "skip_empty_rows", true)
	inputKey := e.GetStringDefault(config, "input_key", "")

	// Extract CSV content from input
	csvContent, err := e.extractContentFromInput(input, inputKey)
	if err != nil {
		return nil, err
	}

	if csvContent == "" {
		return nil, fmt.Errorf("input CSV content is empty")
	}

	// Parse delimiter (handle escape sequences)
	delimRune := e.parseDelimiter(delimiter)

	// Parse CSV
	reader := csv.NewReader(strings.NewReader(csvContent))
	reader.Comma = delimRune
	reader.LazyQuotes = true
	reader.TrimLeadingSpace = trimSpaces
	reader.FieldsPerRecord = -1 // Allow variable number of fields per record

	// Read all records
	records, err := reader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("failed to parse CSV: %w", err)
	}

	if len(records) == 0 {
		return e.buildOutput([]map[string]any{}, []string{}, 0, 0, startTime), nil
	}

	// Determine headers
	var headers []string
	var dataStartIndex int

	if hasHeader {
		// Use first row as headers
		headers = records[0]
		dataStartIndex = 1
	} else if len(customHeaders) > 0 {
		// Use custom headers
		headers = customHeaders
		dataStartIndex = 0
	} else {
		// Auto-generate headers: col_0, col_1, etc.
		if len(records) > 0 {
			headers = make([]string, len(records[0]))
			for i := range headers {
				headers[i] = fmt.Sprintf("col_%d", i)
			}
		}
		dataStartIndex = 0
	}

	// Trim spaces from headers if enabled
	if trimSpaces {
		for i := range headers {
			headers[i] = strings.TrimSpace(headers[i])
		}
	}

	// Convert records to JSON objects
	var result []map[string]any
	for i := dataStartIndex; i < len(records); i++ {
		row := records[i]

		// Skip empty rows if enabled
		if skipEmptyRows && e.isEmptyRow(row) {
			continue
		}

		obj := make(map[string]any)
		for j, value := range row {
			if j < len(headers) {
				if trimSpaces {
					value = strings.TrimSpace(value)
				}
				obj[headers[j]] = value
			}
		}
		result = append(result, obj)
	}

	// Ensure result is never nil
	if result == nil {
		result = []map[string]any{}
	}

	return e.buildOutput(result, headers, len(result), len(headers), startTime), nil
}

// Validate validates the CSV to JSON executor configuration.
func (e *CSVToJSONExecutor) Validate(config map[string]any) error {
	// Validate delimiter
	delimiter := e.GetStringDefault(config, "delimiter", ",")
	if len(delimiter) == 0 {
		return fmt.Errorf("delimiter cannot be empty")
	}
	// Allow single character or escape sequences like \t
	if len(delimiter) > 2 || (len(delimiter) == 2 && delimiter[0] != '\\') {
		return fmt.Errorf("delimiter must be a single character or escape sequence (\\t)")
	}

	return nil
}

// extractContentFromInput extracts CSV string from input.
func (e *CSVToJSONExecutor) extractContentFromInput(input any, inputKey string) (string, error) {
	switch v := input.(type) {
	case string:
		return v, nil
	case []byte:
		return string(v), nil
	case map[string]any:
		// If specific key is provided, use it
		if inputKey != "" {
			if val, ok := v[inputKey]; ok {
				switch contentVal := val.(type) {
				case string:
					return contentVal, nil
				case []byte:
					return string(contentVal), nil
				}
			}
			return "", fmt.Errorf("key '%s' not found in input or has unsupported type", inputKey)
		}
		// Try common field names
		for _, field := range []string{"csv", "data", "content", "body", "text"} {
			if val, ok := v[field]; ok {
				switch contentVal := val.(type) {
				case string:
					return contentVal, nil
				case []byte:
					return string(contentVal), nil
				}
			}
		}
		return "", fmt.Errorf("no CSV content found in input map (tried: csv, data, content, body, text). Specify input_key in config")
	default:
		return "", fmt.Errorf("unsupported input type: %T (expected string, []byte, or map)", input)
	}
}

// parseDelimiter converts delimiter string to rune, handling escape sequences.
func (e *CSVToJSONExecutor) parseDelimiter(delimiter string) rune {
	switch delimiter {
	case "\\t", "\t":
		return '\t'
	case "\\n", "\n":
		return '\n'
	default:
		if len(delimiter) > 0 {
			return rune(delimiter[0])
		}
		return ','
	}
}

// getStringSlice extracts a string slice from config.
func (e *CSVToJSONExecutor) getStringSlice(config map[string]any, key string) []string {
	val, ok := config[key]
	if !ok {
		return nil
	}

	switch v := val.(type) {
	case []string:
		return v
	case []any:
		result := make([]string, 0, len(v))
		for _, item := range v {
			if str, ok := item.(string); ok {
				result = append(result, str)
			}
		}
		return result
	default:
		return nil
	}
}

// isEmptyRow checks if all fields in a row are empty.
func (e *CSVToJSONExecutor) isEmptyRow(row []string) bool {
	for _, field := range row {
		if strings.TrimSpace(field) != "" {
			return false
		}
	}
	return true
}

// buildOutput creates the output map.
func (e *CSVToJSONExecutor) buildOutput(result []map[string]any, headers []string, rowCount, columnCount int, startTime time.Time) map[string]any {
	return map[string]any{
		"success":      true,
		"result":       result,
		"row_count":    rowCount,
		"column_count": columnCount,
		"headers":      headers,
		"duration_ms":  time.Since(startTime).Milliseconds(),
	}
}
