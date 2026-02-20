package builtin

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/smilemakc/mbflow/go/pkg/executor"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/option"
	"google.golang.org/api/sheets/v4"
)

// GoogleSheetsExecutor executes operations with Google Sheets API.
// Supports reading, writing, and appending data to spreadsheets.
type GoogleSheetsExecutor struct {
	*executor.BaseExecutor
}

// NewGoogleSheetsExecutor creates a new Google Sheets executor.
func NewGoogleSheetsExecutor() *GoogleSheetsExecutor {
	return &GoogleSheetsExecutor{
		BaseExecutor: executor.NewBaseExecutor("google_sheets"),
	}
}

// GoogleSheetsOutput represents the output structure.
type GoogleSheetsOutput struct {
	Success       bool           `json:"success"`
	Operation     string         `json:"operation"`
	SpreadsheetID string         `json:"spreadsheet_id"`
	SheetName     string         `json:"sheet_name,omitempty"`
	Range         string         `json:"range,omitempty"`
	Data          [][]any        `json:"data,omitempty"`
	UpdatedCells  int            `json:"updated_cells,omitempty"`
	UpdatedRows   int            `json:"updated_rows,omitempty"`
	UpdatedRange  string         `json:"updated_range,omitempty"`
	RowCount      int            `json:"row_count,omitempty"`
	ColumnCount   int            `json:"column_count,omitempty"`
	Metadata      map[string]any `json:"metadata,omitempty"`
	DurationMs    int64          `json:"duration_ms"`
}

// Execute implements the Executor interface.
func (e *GoogleSheetsExecutor) Execute(ctx context.Context, config map[string]any, input any) (any, error) {
	startTime := time.Now()

	// Extract required config
	operation, err := e.GetString(config, "operation")
	if err != nil {
		return nil, err
	}

	spreadsheetID, err := e.GetString(config, "spreadsheet_id")
	if err != nil {
		return nil, err
	}

	credentialsJSON, err := e.GetString(config, "credentials")
	if err != nil {
		return nil, fmt.Errorf("credentials are required: %w", err)
	}

	// Optional fields
	sheetName := e.GetStringDefault(config, "sheet_name", "")
	rangeNotation := e.GetStringDefault(config, "range", "")
	valueInputOption := e.GetStringDefault(config, "value_input_option", "USER_ENTERED")
	majorDimension := e.GetStringDefault(config, "major_dimension", "ROWS")

	// Extract columns configuration for object-to-row mapping
	columns := e.extractColumns(config)

	// Create Google Sheets service
	srv, err := e.createSheetsService(ctx, credentialsJSON)
	if err != nil {
		return nil, fmt.Errorf("failed to create sheets service: %w", err)
	}

	var output *GoogleSheetsOutput

	switch operation {
	case "read":
		// For read, build full range notation
		fullRange := e.buildRangeNotation(sheetName, rangeNotation)
		output, err = e.executeRead(ctx, srv, spreadsheetID, fullRange, majorDimension)
	case "write":
		// For write, pass sheetName and rangeNotation separately so we can calculate range based on data
		output, err = e.executeWrite(ctx, srv, spreadsheetID, sheetName, rangeNotation, input, valueInputOption, majorDimension, columns)
	case "append":
		// For append, pass sheetName and rangeNotation separately
		output, err = e.executeAppend(ctx, srv, spreadsheetID, sheetName, rangeNotation, input, valueInputOption, majorDimension, columns)
	default:
		return nil, fmt.Errorf("unsupported operation: %s (supported: read, write, append)", operation)
	}

	if err != nil {
		return nil, err
	}

	output.Operation = operation
	output.SpreadsheetID = spreadsheetID
	output.SheetName = sheetName
	output.Range = rangeNotation
	output.DurationMs = time.Since(startTime).Milliseconds()

	return output, nil
}

// createSheetsService creates a Google Sheets API service using service account credentials.
func (e *GoogleSheetsExecutor) createSheetsService(ctx context.Context, credentialsJSON string) (*sheets.Service, error) {
	creds, err := google.CredentialsFromJSON(ctx, []byte(credentialsJSON), sheets.SpreadsheetsScope)
	if err != nil {
		return nil, fmt.Errorf("failed to parse credentials: %w", err)
	}

	srv, err := sheets.NewService(ctx, option.WithCredentials(creds))
	if err != nil {
		return nil, fmt.Errorf("failed to create sheets service: %w", err)
	}

	return srv, nil
}

// buildRangeNotation builds A1 notation range string.
func (e *GoogleSheetsExecutor) buildRangeNotation(sheetName, rangeNotation string) string {
	if sheetName == "" {
		return rangeNotation
	}
	if rangeNotation == "" {
		return sheetName
	}
	return fmt.Sprintf("%s!%s", sheetName, rangeNotation)
}

// extractColumns extracts column names from config.
// Supports: comma-separated string, or array of strings.
func (e *GoogleSheetsExecutor) extractColumns(config map[string]any) []string {
	columnsVal, ok := config["columns"]
	if !ok {
		return nil
	}

	switch v := columnsVal.(type) {
	case string:
		if v == "" {
			return nil
		}
		// Split comma-separated string
		parts := make([]string, 0)
		current := ""
		for _, ch := range v {
			if ch == ',' {
				trimmed := trimString(current)
				if trimmed != "" {
					parts = append(parts, trimmed)
				}
				current = ""
			} else {
				current += string(ch)
			}
		}
		trimmed := trimString(current)
		if trimmed != "" {
			parts = append(parts, trimmed)
		}
		return parts

	case []any:
		result := make([]string, 0, len(v))
		for _, item := range v {
			if str, ok := item.(string); ok && str != "" {
				result = append(result, str)
			}
		}
		return result

	case []string:
		return v

	default:
		return nil
	}
}

// trimString trims whitespace from string (simple implementation to avoid strings import).
func trimString(s string) string {
	start := 0
	end := len(s)
	for start < end && (s[start] == ' ' || s[start] == '\t' || s[start] == '\n' || s[start] == '\r') {
		start++
	}
	for end > start && (s[end-1] == ' ' || s[end-1] == '\t' || s[end-1] == '\n' || s[end-1] == '\r') {
		end--
	}
	return s[start:end]
}

// calculateDefaultRange calculates A1 notation range based on data dimensions and major dimension.
// With ROWS mode: each inner array is a row → values[rows][cols] → range A1:${cols}${rows}
// With COLUMNS mode: each inner array is a column → values[cols][rows] → range A1:${len(values)}${maxInnerLen}
// E.g., for 3 rows x 5 columns with ROWS returns "A1:E3"
// E.g., for 3 arrays x 5 elements with COLUMNS returns "A1:C5" (3 columns, 5 rows)
func (e *GoogleSheetsExecutor) calculateDefaultRange(values [][]any, majorDimension string) string {
	if len(values) == 0 {
		return "A1"
	}

	// Find max inner array length
	maxInnerLen := 0
	for _, inner := range values {
		if len(inner) > maxInnerLen {
			maxInnerLen = len(inner)
		}
	}

	if maxInnerLen == 0 {
		maxInnerLen = 1
	}

	var endCol string
	var endRow int

	if majorDimension == "COLUMNS" {
		// COLUMNS mode: each inner array is a column
		// len(values) = number of columns, maxInnerLen = number of rows
		endCol = columnToLetter(len(values))
		endRow = maxInnerLen
	} else {
		// ROWS mode (default): each inner array is a row
		// len(values) = number of rows, maxInnerLen = number of columns
		endCol = columnToLetter(maxInnerLen)
		endRow = len(values)
	}

	return fmt.Sprintf("A1:%s%d", endCol, endRow)
}

// columnToLetter converts column number (1-based) to Excel-style letter.
// 1 -> A, 26 -> Z, 27 -> AA, 702 -> ZZ, 703 -> AAA
func columnToLetter(col int) string {
	result := ""
	for col > 0 {
		col-- // Make it 0-based
		remainder := col % 26
		result = string(rune('A'+remainder)) + result
		col = col / 26
	}
	return result
}

// executeRead reads data from spreadsheet.
func (e *GoogleSheetsExecutor) executeRead(ctx context.Context, srv *sheets.Service, spreadsheetID, rangeNotation, majorDimension string) (*GoogleSheetsOutput, error) {
	resp, err := srv.Spreadsheets.Values.Get(spreadsheetID, rangeNotation).
		MajorDimension(majorDimension).
		Context(ctx).
		Do()
	if err != nil {
		return nil, fmt.Errorf("failed to read from spreadsheet: %w", err)
	}

	rowCount := len(resp.Values)
	columnCount := 0
	if rowCount > 0 {
		columnCount = len(resp.Values[0])
	}

	return &GoogleSheetsOutput{
		Success:     true,
		Data:        resp.Values,
		RowCount:    rowCount,
		ColumnCount: columnCount,
		Metadata: map[string]any{
			"major_dimension": resp.MajorDimension,
			"range":           resp.Range,
		},
	}, nil
}

// executeWrite writes data to spreadsheet (overwrites existing data).
func (e *GoogleSheetsExecutor) executeWrite(ctx context.Context, srv *sheets.Service, spreadsheetID, sheetName, rangeNotation string, input any, valueInputOption, majorDimension string, columns []string) (*GoogleSheetsOutput, error) {
	values, err := e.extractValuesFromInput(input, columns)
	if err != nil {
		return nil, err
	}

	if len(values) == 0 {
		return nil, fmt.Errorf("no data to write: input is empty or could not be converted to rows")
	}

	// Always calculate range based on actual data size and major dimension to avoid dimension mismatch errors
	// User-provided range is ignored for write operations to prevent "tried writing to column X" errors
	rangeNotation = e.calculateDefaultRange(values, majorDimension)

	// Build full range with sheet name
	fullRange := e.buildRangeNotation(sheetName, rangeNotation)

	valueRange := &sheets.ValueRange{
		MajorDimension: majorDimension,
		Values:         values,
	}

	resp, err := srv.Spreadsheets.Values.Update(spreadsheetID, fullRange, valueRange).
		ValueInputOption(valueInputOption).
		Context(ctx).
		Do()
	if err != nil {
		return nil, fmt.Errorf("failed to write to spreadsheet (range: %s, rows: %d, cols: %d): %w", fullRange, len(values), e.maxColumns(values), err)
	}

	return &GoogleSheetsOutput{
		Success:      true,
		UpdatedCells: int(resp.UpdatedCells),
		UpdatedRows:  int(resp.UpdatedRows),
		UpdatedRange: resp.UpdatedRange,
		Metadata: map[string]any{
			"updated_columns": resp.UpdatedColumns,
		},
	}, nil
}

// executeAppend appends data to the end of spreadsheet.
func (e *GoogleSheetsExecutor) executeAppend(ctx context.Context, srv *sheets.Service, spreadsheetID, sheetName, rangeNotation string, input any, valueInputOption, majorDimension string, columns []string) (*GoogleSheetsOutput, error) {
	values, err := e.extractValuesFromInput(input, columns)
	if err != nil {
		return nil, err
	}

	if len(values) == 0 {
		return nil, fmt.Errorf("no data to append: input is empty or could not be converted to rows")
	}

	// Always calculate range based on actual data dimensions and major dimension
	// Google Sheets will find the next empty row automatically
	var endCol string
	if majorDimension == "COLUMNS" {
		// COLUMNS mode: len(values) is number of columns
		endCol = columnToLetter(len(values))
	} else {
		// ROWS mode: maxColumns is number of columns
		maxCols := e.maxColumns(values)
		endCol = columnToLetter(maxCols)
	}
	rangeNotation = fmt.Sprintf("A:%s", endCol)

	// Build full range with sheet name
	fullRange := e.buildRangeNotation(sheetName, rangeNotation)

	valueRange := &sheets.ValueRange{
		MajorDimension: majorDimension,
		Values:         values,
	}

	resp, err := srv.Spreadsheets.Values.Append(spreadsheetID, fullRange, valueRange).
		ValueInputOption(valueInputOption).
		InsertDataOption("INSERT_ROWS").
		Context(ctx).
		Do()
	if err != nil {
		return nil, fmt.Errorf("failed to append to spreadsheet (range: %s, rows: %d, cols: %d): %w", fullRange, len(values), e.maxColumns(values), err)
	}

	updateResp := resp.Updates

	return &GoogleSheetsOutput{
		Success:      true,
		UpdatedCells: int(updateResp.UpdatedCells),
		UpdatedRows:  int(updateResp.UpdatedRows),
		UpdatedRange: updateResp.UpdatedRange,
		Metadata: map[string]any{
			"updated_columns": updateResp.UpdatedColumns,
		},
	}, nil
}

// maxColumns returns the maximum number of columns across all rows.
func (e *GoogleSheetsExecutor) maxColumns(values [][]any) int {
	maxCols := 0
	for _, row := range values {
		if len(row) > maxCols {
			maxCols = len(row)
		}
	}
	if maxCols == 0 {
		maxCols = 1
	}
	return maxCols
}

// extractValuesFromInput extracts 2D array of values from input.
// columns parameter allows specifying which fields to extract and in what order.
func (e *GoogleSheetsExecutor) extractValuesFromInput(input any, columns []string) ([][]any, error) {
	switch v := input.(type) {
	case [][]any:
		// Direct 2D array - serialize any complex values
		return e.serializeNestedValues(v), nil

	case []map[string]any:
		// Typed slice of maps - convert each map to row
		result := make([][]any, 0, len(v))
		for _, row := range v {
			values := e.objectToRow(row, columns)
			result = append(result, values)
		}
		return result, nil

	case []any:
		// Array of rows - convert each row to []any
		result := make([][]any, 0, len(v))
		for i, row := range v {
			switch r := row.(type) {
			case []any:
				result = append(result, e.serializeRow(r))
			case map[string]any:
				// Convert object to array of values using specified columns or all fields
				values := e.objectToRow(r, columns)
				result = append(result, values)
			default:
				return nil, fmt.Errorf("unsupported row type at index %d: %T", i, row)
			}
		}
		return result, nil

	case map[string]any:
		// Try to extract from common field names
		for _, field := range []string{"data", "values", "rows", "content", "body", "items"} {
			if val, ok := v[field]; ok {
				return e.extractValuesFromInput(val, columns)
			}
		}

		// Try to parse from JSON string field
		for _, field := range []string{"json", "text"} {
			if str, ok := v[field].(string); ok {
				var parsed any
				if err := json.Unmarshal([]byte(str), &parsed); err == nil {
					return e.extractValuesFromInput(parsed, columns)
				}
			}
		}

		// Single object - treat as single row
		values := e.objectToRow(v, columns)
		return [][]any{values}, nil

	case string:
		// Try to parse JSON string
		var parsed any
		if err := json.Unmarshal([]byte(v), &parsed); err != nil {
			return nil, fmt.Errorf("failed to parse JSON from string input: %w", err)
		}
		return e.extractValuesFromInput(parsed, columns)

	case []byte:
		// Try to parse JSON bytes
		var parsed any
		if err := json.Unmarshal(v, &parsed); err != nil {
			return nil, fmt.Errorf("failed to parse JSON from bytes input: %w", err)
		}
		return e.extractValuesFromInput(parsed, columns)

	default:
		return nil, fmt.Errorf("unsupported input type: %T (expected 2D array, array of objects, or JSON string)", input)
	}
}

// objectToRow converts a map to an array of values.
// If columns is specified, extracts values in that order.
// Otherwise extracts all values in alphabetical key order.
func (e *GoogleSheetsExecutor) objectToRow(obj map[string]any, columns []string) []any {
	if len(columns) > 0 {
		// Use specified column order
		values := make([]any, len(columns))
		for i, col := range columns {
			if val, ok := obj[col]; ok {
				values[i] = e.serializeValue(val)
			} else {
				values[i] = "" // Empty string for missing fields
			}
		}
		return values
	}

	// No columns specified - extract all fields in sorted order for consistency
	keys := make([]string, 0, len(obj))
	for k := range obj {
		keys = append(keys, k)
	}
	// Sort keys alphabetically for consistent ordering
	for i := 0; i < len(keys)-1; i++ {
		for j := i + 1; j < len(keys); j++ {
			if keys[i] > keys[j] {
				keys[i], keys[j] = keys[j], keys[i]
			}
		}
	}

	values := make([]any, len(keys))
	for i, k := range keys {
		values[i] = e.serializeValue(obj[k])
	}
	return values
}

// serializeValue converts a value to a string suitable for Google Sheets.
// Complex types (arrays, objects) are serialized as JSON.
// String values are trimmed of leading/trailing whitespace.
func (e *GoogleSheetsExecutor) serializeValue(val any) any {
	if val == nil {
		return ""
	}

	switch v := val.(type) {
	case string:
		// Trim whitespace (newlines, tabs, spaces) from strings
		return trimString(v)
	case bool:
		return v
	case int, int8, int16, int32, int64:
		return v
	case uint, uint8, uint16, uint32, uint64:
		return v
	case float32, float64:
		return v
	case []any, map[string]any:
		// Serialize complex types as JSON
		jsonBytes, err := json.Marshal(v)
		if err != nil {
			return fmt.Sprintf("%v", v)
		}
		return string(jsonBytes)
	default:
		// For other types, try JSON serialization first
		jsonBytes, err := json.Marshal(v)
		if err != nil {
			return fmt.Sprintf("%v", v)
		}
		// If it's a simple quoted string, unquote it
		str := string(jsonBytes)
		if len(str) >= 2 && str[0] == '"' && str[len(str)-1] == '"' {
			return str[1 : len(str)-1]
		}
		return str
	}
}

// serializeRow serializes all values in a row.
func (e *GoogleSheetsExecutor) serializeRow(row []any) []any {
	result := make([]any, len(row))
	for i, val := range row {
		result[i] = e.serializeValue(val)
	}
	return result
}

// serializeNestedValues serializes any nested complex values in a 2D array.
func (e *GoogleSheetsExecutor) serializeNestedValues(data [][]any) [][]any {
	result := make([][]any, len(data))
	for i, row := range data {
		result[i] = e.serializeRow(row)
	}
	return result
}

// Validate validates the executor configuration.
func (e *GoogleSheetsExecutor) Validate(config map[string]any) error {
	// Validate required fields
	if err := e.ValidateRequired(config, "operation", "spreadsheet_id", "credentials"); err != nil {
		return err
	}

	// Validate operation
	operation, err := e.GetString(config, "operation")
	if err != nil {
		return err
	}

	validOperations := map[string]bool{
		"read":   true,
		"write":  true,
		"append": true,
	}

	if !validOperations[operation] {
		return fmt.Errorf("invalid operation: %s (supported: read, write, append)", operation)
	}

	// Validate spreadsheet_id
	spreadsheetID, err := e.GetString(config, "spreadsheet_id")
	if err != nil {
		return err
	}
	if spreadsheetID == "" {
		return fmt.Errorf("spreadsheet_id cannot be empty")
	}

	// Validate credentials (must be valid JSON)
	credentials, err := e.GetString(config, "credentials")
	if err != nil {
		return err
	}
	if credentials == "" {
		return fmt.Errorf("credentials cannot be empty")
	}

	// Try to parse credentials as JSON
	var creds map[string]any
	if err := json.Unmarshal([]byte(credentials), &creds); err != nil {
		return fmt.Errorf("credentials must be valid JSON: %w", err)
	}

	// Validate value_input_option if present
	if valueInputOption, exists := config["value_input_option"]; exists {
		if str, ok := valueInputOption.(string); ok {
			validOptions := map[string]bool{
				"RAW":          true,
				"USER_ENTERED": true,
			}
			if !validOptions[str] {
				return fmt.Errorf("invalid value_input_option: %s (supported: RAW, USER_ENTERED)", str)
			}
		}
	}

	// Validate major_dimension if present
	if majorDimension, exists := config["major_dimension"]; exists {
		if str, ok := majorDimension.(string); ok {
			validDimensions := map[string]bool{
				"ROWS":    true,
				"COLUMNS": true,
			}
			if !validDimensions[str] {
				return fmt.Errorf("invalid major_dimension: %s (supported: ROWS, COLUMNS)", str)
			}
		}
	}

	return nil
}
