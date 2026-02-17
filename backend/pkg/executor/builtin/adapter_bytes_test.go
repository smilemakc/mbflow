package builtin

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBytesToJsonExecutor_Execute(t *testing.T) {
	executor := NewBytesToJsonExecutor()
	ctx := context.Background()

	tests := []struct {
		name           string
		config         map[string]any
		input          any
		expectedResult any
		expectError    bool
		errorContains  string
	}{
		{
			name: "simple UTF-8 JSON object",
			config: map[string]any{
				"encoding":      "utf-8",
				"validate_json": true,
			},
			input: []byte(`{"name":"John","age":30}`),
			expectedResult: map[string]any{
				"name": "John",
				"age":  json.Number("30"),
			},
			expectError: false,
		},
		{
			name: "UTF-8 with BOM",
			config: map[string]any{
				"encoding":      "utf-8",
				"validate_json": true,
			},
			input: []byte{0xEF, 0xBB, 0xBF, 0x7B, 0x22, 0x6E, 0x61, 0x6D, 0x65, 0x22, 0x3A, 0x22, 0x54, 0x65, 0x73, 0x74, 0x22, 0x7D}, // BOM + {"name":"Test"}
			expectedResult: map[string]any{
				"name": "Test",
			},
			expectError: false,
		},
		{
			name: "JSON array",
			config: map[string]any{
				"encoding":      "utf-8",
				"validate_json": true,
			},
			input: []byte(`[1,2,3]`),
			expectedResult: []any{
				json.Number("1"),
				json.Number("2"),
				json.Number("3"),
			},
			expectError: false,
		},
		{
			name: "string input with auto base64 decode",
			config: map[string]any{
				"encoding":      "utf-8",
				"validate_json": true,
			},
			input:          base64.StdEncoding.EncodeToString([]byte(`{"key":"value"}`)),
			expectedResult: map[string]any{"key": "value"},
			expectError:    false,
		},
		{
			name: "map input with data field",
			config: map[string]any{
				"encoding":      "utf-8",
				"validate_json": true,
			},
			input: map[string]any{
				"data": []byte(`{"test":true}`),
			},
			expectedResult: map[string]any{"test": true},
			expectError:    false,
		},
		{
			name: "invalid JSON with validation enabled",
			config: map[string]any{
				"encoding":      "utf-8",
				"validate_json": true,
			},
			input:         []byte(`{invalid json`),
			expectError:   true,
			errorContains: "JSON parsing failed",
		},
		{
			name: "invalid JSON with validation disabled",
			config: map[string]any{
				"encoding":      "utf-8",
				"validate_json": false,
			},
			input:          []byte(`{invalid json`),
			expectedResult: nil, // Should return null instead of error
			expectError:    false,
		},
		{
			name: "unsupported encoding",
			config: map[string]any{
				"encoding":      "unknown",
				"validate_json": true,
			},
			input:         []byte(`{"test":true}`),
			expectError:   true,
			errorContains: "unsupported encoding",
		},
		{
			name: "nested JSON object",
			config: map[string]any{
				"encoding":      "utf-8",
				"validate_json": true,
			},
			input: []byte(`{"user":{"name":"Alice","profile":{"email":"alice@example.com"}}}`),
			expectedResult: map[string]any{
				"user": map[string]any{
					"name": "Alice",
					"profile": map[string]any{
						"email": "alice@example.com",
					},
				},
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := executor.Execute(ctx, tt.config, tt.input)

			if tt.expectError {
				require.Error(t, err)
				if tt.errorContains != "" {
					assert.Contains(t, err.Error(), tt.errorContains)
				}
				return
			}

			require.NoError(t, err)
			resultMap, ok := result.(map[string]any)
			require.True(t, ok, "result should be a map")

			assert.True(t, resultMap["success"].(bool))
			assert.Equal(t, tt.expectedResult, resultMap["result"])
			assert.NotNil(t, resultMap["encoding_used"])
			assert.NotNil(t, resultMap["byte_size"])
			assert.NotNil(t, resultMap["duration_ms"])
		})
	}
}

func TestBytesToJsonExecutor_Validate(t *testing.T) {
	executor := NewBytesToJsonExecutor()

	tests := []struct {
		name          string
		config        map[string]any
		expectError   bool
		errorContains string
	}{
		{
			name: "valid config with utf-8",
			config: map[string]any{
				"encoding":      "utf-8",
				"validate_json": true,
			},
			expectError: false,
		},
		{
			name: "valid config with utf-16",
			config: map[string]any{
				"encoding":      "utf-16",
				"validate_json": false,
			},
			expectError: false,
		},
		{
			name: "valid config with latin1",
			config: map[string]any{
				"encoding": "latin1",
			},
			expectError: false,
		},
		{
			name: "invalid encoding",
			config: map[string]any{
				"encoding": "ascii",
			},
			expectError:   true,
			errorContains: "invalid encoding",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := executor.Validate(tt.config)

			if tt.expectError {
				require.Error(t, err)
				if tt.errorContains != "" {
					assert.Contains(t, err.Error(), tt.errorContains)
				}
				return
			}

			require.NoError(t, err)
		})
	}
}

func TestBytesToJsonExecutor_EncodingDetection(t *testing.T) {
	executor := NewBytesToJsonExecutor()
	ctx := context.Background()

	tests := []struct {
		name             string
		input            []byte
		expectedEncoding string
	}{
		{
			name:             "UTF-8 BOM detection",
			input:            []byte{0xEF, 0xBB, 0xBF, 0x7B, 0x7D}, // BOM + {}
			expectedEncoding: "utf-8",
		},
		{
			name:             "UTF-16 LE BOM detection",
			input:            []byte{0xFF, 0xFE, 0x7B, 0x00, 0x7D, 0x00}, // BOM + {} in UTF-16 LE
			expectedEncoding: "utf-16",
		},
		{
			name:             "UTF-16 BE BOM detection",
			input:            []byte{0xFE, 0xFF, 0x00, 0x7B, 0x00, 0x7D}, // BOM + {} in UTF-16 BE
			expectedEncoding: "utf-16",
		},
		{
			name:             "plain UTF-8 (no BOM)",
			input:            []byte(`{"test":true}`),
			expectedEncoding: "utf-8",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := map[string]any{
				"encoding":      "utf-8", // Start with utf-8, should auto-detect
				"validate_json": false,   // Don't validate to avoid parsing errors
			}

			result, err := executor.Execute(ctx, config, tt.input)
			require.NoError(t, err)

			resultMap := result.(map[string]any)
			assert.Equal(t, tt.expectedEncoding, resultMap["encoding_used"])
		})
	}
}

func TestBytesToJsonExecutor_ExtractBytes(t *testing.T) {
	executor := NewBytesToJsonExecutor()

	tests := []struct {
		name          string
		input         any
		expected      []byte
		expectError   bool
		errorContains string
	}{
		{
			name:     "direct byte slice",
			input:    []byte("test"),
			expected: []byte("test"),
		},
		{
			name:     "string input (not base64)",
			input:    "hello",
			expected: []byte("hello"),
		},
		{
			name:     "base64 string input",
			input:    base64.StdEncoding.EncodeToString([]byte("decoded")),
			expected: []byte("decoded"),
		},
		{
			name:     "map with data field (bytes)",
			input:    map[string]any{"data": []byte("test")},
			expected: []byte("test"),
		},
		{
			name:     "map with data field (string)",
			input:    map[string]any{"data": "plaintext"},
			expected: []byte("plaintext"),
		},
		{
			name:          "map without data field",
			input:         map[string]any{"other": "value"},
			expectError:   true,
			errorContains: "expected 'data' field",
		},
		{
			name:          "unsupported type",
			input:         12345,
			expectError:   true,
			errorContains: "unsupported input type",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := executor.extractBytes(tt.input)

			if tt.expectError {
				require.Error(t, err)
				if tt.errorContains != "" {
					assert.Contains(t, err.Error(), tt.errorContains)
				}
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}
