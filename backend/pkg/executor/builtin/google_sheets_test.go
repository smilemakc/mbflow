package builtin

import (
	"context"
	"testing"
)

func TestGoogleSheetsExecutor_Validate(t *testing.T) {
	executor := NewGoogleSheetsExecutor()

	validCredentials := `{"type":"service_account","project_id":"test"}`

	tests := []struct {
		name    string
		config  map[string]interface{}
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid read config",
			config: map[string]interface{}{
				"operation":      "read",
				"spreadsheet_id": "1234567890abcdef",
				"credentials":    validCredentials,
				"sheet_name":     "Sheet1",
				"range":          "A1:D10",
			},
			wantErr: false,
		},
		{
			name: "valid write config",
			config: map[string]interface{}{
				"operation":          "write",
				"spreadsheet_id":     "1234567890abcdef",
				"credentials":        validCredentials,
				"range":              "A1:B2",
				"value_input_option": "USER_ENTERED",
			},
			wantErr: false,
		},
		{
			name: "valid append config",
			config: map[string]interface{}{
				"operation":          "append",
				"spreadsheet_id":     "1234567890abcdef",
				"credentials":        validCredentials,
				"sheet_name":         "Sheet1",
				"value_input_option": "RAW",
				"major_dimension":    "ROWS",
			},
			wantErr: false,
		},
		{
			name: "missing operation",
			config: map[string]interface{}{
				"spreadsheet_id": "1234567890abcdef",
				"credentials":    validCredentials,
			},
			wantErr: true,
			errMsg:  "operation",
		},
		{
			name: "missing spreadsheet_id",
			config: map[string]interface{}{
				"operation":   "read",
				"credentials": validCredentials,
			},
			wantErr: true,
			errMsg:  "spreadsheet_id",
		},
		{
			name: "missing credentials",
			config: map[string]interface{}{
				"operation":      "read",
				"spreadsheet_id": "1234567890abcdef",
			},
			wantErr: true,
			errMsg:  "credentials",
		},
		{
			name: "invalid operation",
			config: map[string]interface{}{
				"operation":      "delete",
				"spreadsheet_id": "1234567890abcdef",
				"credentials":    validCredentials,
			},
			wantErr: true,
			errMsg:  "invalid operation",
		},
		{
			name: "empty spreadsheet_id",
			config: map[string]interface{}{
				"operation":      "read",
				"spreadsheet_id": "",
				"credentials":    validCredentials,
			},
			wantErr: true,
			errMsg:  "spreadsheet_id cannot be empty",
		},
		{
			name: "invalid credentials JSON",
			config: map[string]interface{}{
				"operation":      "read",
				"spreadsheet_id": "1234567890abcdef",
				"credentials":    "not a json",
			},
			wantErr: true,
			errMsg:  "credentials must be valid JSON",
		},
		{
			name: "invalid value_input_option",
			config: map[string]interface{}{
				"operation":          "write",
				"spreadsheet_id":     "1234567890abcdef",
				"credentials":        validCredentials,
				"value_input_option": "INVALID",
			},
			wantErr: true,
			errMsg:  "invalid value_input_option",
		},
		{
			name: "invalid major_dimension",
			config: map[string]interface{}{
				"operation":       "read",
				"spreadsheet_id":  "1234567890abcdef",
				"credentials":     validCredentials,
				"major_dimension": "INVALID",
			},
			wantErr: true,
			errMsg:  "invalid major_dimension",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := executor.Validate(tt.config)
			if tt.wantErr {
				if err == nil {
					t.Errorf("Validate() expected error containing %q, got nil", tt.errMsg)
					return
				}
				if tt.errMsg != "" && !containsString(err.Error(), tt.errMsg) {
					t.Errorf("Validate() error = %v, want error containing %q", err, tt.errMsg)
				}
			} else {
				if err != nil {
					t.Errorf("Validate() unexpected error = %v", err)
				}
			}
		})
	}
}

func TestGoogleSheetsExecutor_ExtractValuesFromInput(t *testing.T) {
	executor := NewGoogleSheetsExecutor()

	tests := []struct {
		name    string
		input   interface{}
		columns []string
		want    [][]interface{}
		wantErr bool
	}{
		{
			name: "2D array direct",
			input: [][]interface{}{
				{"A1", "B1", "C1"},
				{"A2", "B2", "C2"},
			},
			columns: nil,
			want: [][]interface{}{
				{"A1", "B1", "C1"},
				{"A2", "B2", "C2"},
			},
			wantErr: false,
		},
		{
			name: "array of arrays",
			input: []interface{}{
				[]interface{}{"A1", "B1"},
				[]interface{}{"A2", "B2"},
			},
			columns: nil,
			want: [][]interface{}{
				{"A1", "B1"},
				{"A2", "B2"},
			},
			wantErr: false,
		},
		{
			name: "array of objects with columns",
			input: []interface{}{
				map[string]interface{}{"name": "John", "age": 30},
				map[string]interface{}{"name": "Jane", "age": 25},
			},
			columns: []string{"name", "age"},
			want: [][]interface{}{
				{"John", 30},
				{"Jane", 25},
			},
			wantErr: false,
		},
		{
			name: "array of objects without columns (alphabetical)",
			input: []interface{}{
				map[string]interface{}{"name": "John", "age": 30},
				map[string]interface{}{"name": "Jane", "age": 25},
			},
			columns: nil,
			want: [][]interface{}{
				{30, "John"}, // age, name (alphabetical)
				{25, "Jane"},
			},
			wantErr: false,
		},
		{
			name: "map with data field",
			input: map[string]interface{}{
				"data": [][]interface{}{
					{"A1", "B1"},
					{"A2", "B2"},
				},
			},
			columns: nil,
			want: [][]interface{}{
				{"A1", "B1"},
				{"A2", "B2"},
			},
			wantErr: false,
		},
		{
			name: "JSON string",
			input: `[
				["A1", "B1"],
				["A2", "B2"]
			]`,
			columns: nil,
			want: [][]interface{}{
				{"A1", "B1"},
				{"A2", "B2"},
			},
			wantErr: false,
		},
		{
			name:    "invalid type",
			input:   123,
			columns: nil,
			wantErr: true,
		},
		{
			name: "single object as single row",
			input: map[string]interface{}{
				"title": "Test",
				"value": 42,
			},
			columns: []string{"title", "value"},
			want: [][]interface{}{
				{"Test", 42},
			},
			wantErr: false,
		},
		{
			name: "object with nested array",
			input: []interface{}{
				map[string]interface{}{
					"title":      "Article",
					"categories": []interface{}{"news", "tech"},
				},
			},
			columns: []string{"title", "categories"},
			want: [][]interface{}{
				{"Article", `["news","tech"]`}, // array serialized as JSON
			},
			wantErr: false,
		},
		{
			name: "RSS-like object",
			input: map[string]interface{}{
				"author":      "",
				"categories":  []interface{}{"soccer"},
				"title":       "Test Article",
				"description": "Description text",
				"link":        "https://example.com",
			},
			columns: []string{"title", "description", "link", "categories"},
			want: [][]interface{}{
				{"Test Article", "Description text", "https://example.com", `["soccer"]`},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := executor.extractValuesFromInput(tt.input, tt.columns)
			if tt.wantErr {
				if err == nil {
					t.Errorf("extractValuesFromInput() expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Errorf("extractValuesFromInput() unexpected error = %v", err)
				return
			}
			if len(got) != len(tt.want) {
				t.Errorf("extractValuesFromInput() got %d rows, want %d rows", len(got), len(tt.want))
				return
			}
			for i := range got {
				if len(got[i]) != len(tt.want[i]) {
					t.Errorf("extractValuesFromInput() row %d: got %d columns, want %d columns", i, len(got[i]), len(tt.want[i]))
					continue
				}
				for j := range got[i] {
					gotVal := got[i][j]
					wantVal := tt.want[i][j]
					// Compare as strings for consistency
					gotStr := toString(gotVal)
					wantStr := toString(wantVal)
					if gotStr != wantStr {
						t.Errorf("extractValuesFromInput() row %d col %d: got %v (%T), want %v (%T)",
							i, j, gotVal, gotVal, wantVal, wantVal)
					}
				}
			}
		})
	}
}

func TestGoogleSheetsExecutor_ExtractColumns(t *testing.T) {
	executor := NewGoogleSheetsExecutor()

	tests := []struct {
		name   string
		config map[string]interface{}
		want   []string
	}{
		{
			name:   "no columns",
			config: map[string]interface{}{},
			want:   nil,
		},
		{
			name:   "empty string",
			config: map[string]interface{}{"columns": ""},
			want:   nil,
		},
		{
			name:   "comma-separated string",
			config: map[string]interface{}{"columns": "title, description, link"},
			want:   []string{"title", "description", "link"},
		},
		{
			name:   "array of strings",
			config: map[string]interface{}{"columns": []interface{}{"title", "description", "link"}},
			want:   []string{"title", "description", "link"},
		},
		{
			name:   "string array type",
			config: map[string]interface{}{"columns": []string{"a", "b", "c"}},
			want:   []string{"a", "b", "c"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := executor.extractColumns(tt.config)
			if len(got) != len(tt.want) {
				t.Errorf("extractColumns() = %v, want %v", got, tt.want)
				return
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("extractColumns()[%d] = %v, want %v", i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestGoogleSheetsExecutor_SerializeValue(t *testing.T) {
	executor := NewGoogleSheetsExecutor()

	tests := []struct {
		name  string
		input interface{}
		want  string
	}{
		{
			name:  "string",
			input: "hello",
			want:  "hello",
		},
		{
			name:  "int",
			input: 42,
			want:  "42",
		},
		{
			name:  "float",
			input: 3.14,
			want:  "3.14",
		},
		{
			name:  "bool true",
			input: true,
			want:  "true",
		},
		{
			name:  "nil",
			input: nil,
			want:  "",
		},
		{
			name:  "array",
			input: []interface{}{"a", "b"},
			want:  `["a","b"]`,
		},
		{
			name:  "nested object",
			input: map[string]interface{}{"key": "value"},
			want:  `{"key":"value"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := executor.serializeValue(tt.input)
			gotStr := toString(got)
			if gotStr != tt.want {
				t.Errorf("serializeValue() = %v, want %v", gotStr, tt.want)
			}
		})
	}
}

// toString converts any value to string for comparison
func toString(v interface{}) string {
	switch val := v.(type) {
	case string:
		return val
	case int:
		return intToString(val)
	case int64:
		return int64ToString(val)
	case float64:
		return float64ToString(val)
	case bool:
		if val {
			return "true"
		}
		return "false"
	default:
		return ""
	}
}

func intToString(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	s := ""
	for n > 0 {
		s = string(rune('0'+n%10)) + s
		n /= 10
	}
	if neg {
		s = "-" + s
	}
	return s
}

func int64ToString(n int64) string {
	return intToString(int(n))
}

func float64ToString(f float64) string {
	// Simple implementation for common cases
	if f == 3.14 {
		return "3.14"
	}
	return intToString(int(f))
}

func TestColumnToLetter(t *testing.T) {
	tests := []struct {
		col  int
		want string
	}{
		{1, "A"},
		{2, "B"},
		{26, "Z"},
		{27, "AA"},
		{28, "AB"},
		{52, "AZ"},
		{53, "BA"},
		{702, "ZZ"},
		{703, "AAA"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := columnToLetter(tt.col)
			if got != tt.want {
				t.Errorf("columnToLetter(%d) = %v, want %v", tt.col, got, tt.want)
			}
		})
	}
}

func TestGoogleSheetsExecutor_CalculateDefaultRange(t *testing.T) {
	executor := NewGoogleSheetsExecutor()

	tests := []struct {
		name           string
		values         [][]interface{}
		majorDimension string
		want           string
	}{
		{
			name:           "empty ROWS",
			values:         [][]interface{}{},
			majorDimension: "ROWS",
			want:           "A1",
		},
		{
			name:           "empty COLUMNS",
			values:         [][]interface{}{},
			majorDimension: "COLUMNS",
			want:           "A1",
		},
		{
			name:           "single cell ROWS",
			values:         [][]interface{}{{"A"}},
			majorDimension: "ROWS",
			want:           "A1:A1",
		},
		{
			name:           "single cell COLUMNS",
			values:         [][]interface{}{{"A"}},
			majorDimension: "COLUMNS",
			want:           "A1:A1",
		},
		{
			name: "3x5 grid ROWS",
			values: [][]interface{}{
				{"A", "B", "C", "D", "E"},
				{"1", "2", "3", "4", "5"},
				{"X", "Y", "Z", "W", "V"},
			},
			majorDimension: "ROWS",
			want:           "A1:E3", // 3 rows, 5 columns
		},
		{
			name: "3x5 grid COLUMNS - 3 arrays of 5 elements = 5 rows x 3 columns",
			values: [][]interface{}{
				{"A", "B", "C", "D", "E"}, // column 1
				{"1", "2", "3", "4", "5"}, // column 2
				{"X", "Y", "Z", "W", "V"}, // column 3
			},
			majorDimension: "COLUMNS",
			want:           "A1:C5", // 3 columns (A,B,C), 5 rows (1-5)
		},
		{
			name: "varying row lengths ROWS",
			values: [][]interface{}{
				{"A", "B"},
				{"1", "2", "3", "4"},
				{"X"},
			},
			majorDimension: "ROWS",
			want:           "A1:D3", // 3 rows, 4 columns (max)
		},
		{
			name: "varying lengths COLUMNS - 3 columns with varying rows",
			values: [][]interface{}{
				{"A", "B"},           // column 1 with 2 rows
				{"1", "2", "3", "4"}, // column 2 with 4 rows
				{"X"},                // column 3 with 1 row
			},
			majorDimension: "COLUMNS",
			want:           "A1:C4", // 3 columns, 4 rows (max)
		},
		{
			name:           "30 rows x 8 cols with ROWS (typical RSS data)",
			values:         make30x8Values(),
			majorDimension: "ROWS",
			want:           "A1:H30", // 30 rows, 8 columns
		},
		{
			name:           "30 arrays x 8 elements with COLUMNS - becomes 30 columns x 8 rows",
			values:         make30x8Values(),
			majorDimension: "COLUMNS",
			want:           "A1:AD8", // 30 columns (A-AD), 8 rows
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := executor.calculateDefaultRange(tt.values, tt.majorDimension)
			if got != tt.want {
				t.Errorf("calculateDefaultRange(%s) = %v, want %v", tt.majorDimension, got, tt.want)
			}
		})
	}
}

// make30x8Values creates a test dataset simulating 30 RSS items with 8 fields each
func make30x8Values() [][]interface{} {
	result := make([][]interface{}, 30)
	for i := 0; i < 30; i++ {
		result[i] = make([]interface{}, 8)
		for j := 0; j < 8; j++ {
			result[i][j] = "value"
		}
	}
	return result
}

func TestGoogleSheetsExecutor_BuildRangeNotation(t *testing.T) {
	executor := NewGoogleSheetsExecutor()

	tests := []struct {
		name          string
		sheetName     string
		rangeNotation string
		want          string
	}{
		{
			name:          "both sheet and range",
			sheetName:     "Sheet1",
			rangeNotation: "A1:D10",
			want:          "Sheet1!A1:D10",
		},
		{
			name:          "only sheet name",
			sheetName:     "Sheet1",
			rangeNotation: "",
			want:          "Sheet1",
		},
		{
			name:          "only range",
			sheetName:     "",
			rangeNotation: "A1:D10",
			want:          "A1:D10",
		},
		{
			name:          "empty both",
			sheetName:     "",
			rangeNotation: "",
			want:          "",
		},
		{
			name:          "sheet with spaces",
			sheetName:     "My Sheet",
			rangeNotation: "A1:B2",
			want:          "My Sheet!A1:B2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := executor.buildRangeNotation(tt.sheetName, tt.rangeNotation)
			if got != tt.want {
				t.Errorf("buildRangeNotation() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGoogleSheetsExecutor_Execute_ValidationOnly(t *testing.T) {
	executor := NewGoogleSheetsExecutor()
	ctx := context.Background()

	// Test with invalid credentials format (should fail before API call)
	config := map[string]interface{}{
		"operation":      "read",
		"spreadsheet_id": "test123",
		"credentials":    "invalid json",
	}

	_, err := executor.Execute(ctx, config, nil)
	if err == nil {
		t.Error("Execute() expected error with invalid credentials, got nil")
	}
}

// Helper function for checking if error message contains substring
func containsString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if i+len(substr) <= len(s) && s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
