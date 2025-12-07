package builtin

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStringToJsonExecutor_Execute(t *testing.T) {
	executor := NewStringToJsonExecutor()
	ctx := context.Background()

	tests := []struct {
		name       string
		config     map[string]interface{}
		input      interface{}
		wantResult interface{}
		wantErr    bool
	}{
		{
			name: "simple JSON object",
			config: map[string]interface{}{
				"strict_mode":     true,
				"trim_whitespace": true,
			},
			input: `{"name": "John", "age": 30}`,
			wantResult: map[string]interface{}{
				"name": "John",
				"age":  json.Number("30"),
			},
		},
		{
			name: "JSON array",
			config: map[string]interface{}{
				"strict_mode":     true,
				"trim_whitespace": true,
			},
			input: `[1, 2, 3, "four"]`,
			wantResult: []interface{}{
				json.Number("1"),
				json.Number("2"),
				json.Number("3"),
				"four",
			},
		},
		{
			name: "nested JSON",
			config: map[string]interface{}{
				"strict_mode":     true,
				"trim_whitespace": true,
			},
			input: `{"user": {"profile": {"name": "Jane"}}}`,
			wantResult: map[string]interface{}{
				"user": map[string]interface{}{
					"profile": map[string]interface{}{
						"name": "Jane",
					},
				},
			},
		},
		{
			name: "with leading/trailing whitespace",
			config: map[string]interface{}{
				"strict_mode":     true,
				"trim_whitespace": true,
			},
			input: `   {"test": true}   `,
			wantResult: map[string]interface{}{
				"test": true,
			},
		},
		{
			name: "bytes input",
			config: map[string]interface{}{
				"strict_mode":     true,
				"trim_whitespace": true,
			},
			input: []byte(`{"key": "value"}`),
			wantResult: map[string]interface{}{
				"key": "value",
			},
		},
		{
			name: "map input with data field",
			config: map[string]interface{}{
				"strict_mode":     true,
				"trim_whitespace": true,
			},
			input: map[string]interface{}{
				"data": `{"status": "ok"}`,
			},
			wantResult: map[string]interface{}{
				"status": "ok",
			},
		},
		{
			name: "invalid JSON - strict mode",
			config: map[string]interface{}{
				"strict_mode":     true,
				"trim_whitespace": true,
			},
			input:   `{invalid json}`,
			wantErr: true,
		},
		{
			name: "invalid JSON - non-strict mode",
			config: map[string]interface{}{
				"strict_mode":     false,
				"trim_whitespace": true,
			},
			input:      `{invalid json}`,
			wantResult: nil, // Should return null instead of error
			wantErr:    false,
		},
		{
			name: "empty string",
			config: map[string]interface{}{
				"strict_mode":     false,
				"trim_whitespace": true,
			},
			input:      ``,
			wantResult: nil,
			wantErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := executor.Execute(ctx, tt.config, tt.input)

			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, result)

			resultMap, ok := result.(map[string]interface{})
			require.True(t, ok, "result should be a map")

			assert.True(t, resultMap["success"].(bool))
			assert.Equal(t, tt.wantResult, resultMap["result"])
		})
	}
}

func TestJsonToStringExecutor_Execute(t *testing.T) {
	executor := NewJsonToStringExecutor()
	ctx := context.Background()

	tests := []struct {
		name       string
		config     map[string]interface{}
		input      interface{}
		wantResult string
		wantErr    bool
	}{
		{
			name: "simple object - compact",
			config: map[string]interface{}{
				"pretty":      false,
				"escape_html": true,
				"sort_keys":   false,
			},
			input: map[string]interface{}{
				"name": "John",
				"age":  30,
			},
			wantResult: `{"age":30,"name":"John"}`,
		},
		{
			name: "simple object - pretty",
			config: map[string]interface{}{
				"pretty":      true,
				"indent":      "  ",
				"escape_html": true,
				"sort_keys":   false,
			},
			input: map[string]interface{}{
				"name": "John",
			},
			wantErr: false, // Will check formatting separately
		},
		{
			name: "array",
			config: map[string]interface{}{
				"pretty":      false,
				"escape_html": true,
				"sort_keys":   false,
			},
			input:      []interface{}{1, 2, "three"},
			wantResult: `[1,2,"three"]`,
		},
		{
			name: "nested object",
			config: map[string]interface{}{
				"pretty":      false,
				"escape_html": true,
				"sort_keys":   false,
			},
			input: map[string]interface{}{
				"user": map[string]interface{}{
					"name": "Jane",
				},
			},
			wantErr: false, // Will check it doesn't error
		},
		{
			name: "HTML escaping enabled",
			config: map[string]interface{}{
				"pretty":      false,
				"escape_html": true,
				"sort_keys":   false,
			},
			input: map[string]interface{}{
				"html": "<script>alert('xss')</script>",
			},
			wantErr: false, // Will check escaping separately
		},
		{
			name: "sort keys enabled",
			config: map[string]interface{}{
				"pretty":      false,
				"escape_html": true,
				"sort_keys":   true,
			},
			input: map[string]interface{}{
				"zebra": 1,
				"apple": 2,
				"mango": 3,
			},
			wantResult: `{"apple":2,"mango":3,"zebra":1}`,
		},
		{
			name: "nested sorting",
			config: map[string]interface{}{
				"pretty":      false,
				"escape_html": true,
				"sort_keys":   true,
			},
			input: map[string]interface{}{
				"z_field": map[string]interface{}{
					"nested_z": 1,
					"nested_a": 2,
				},
				"a_field": "value",
			},
			wantResult: `{"a_field":"value","z_field":{"nested_a":2,"nested_z":1}}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := executor.Execute(ctx, tt.config, tt.input)

			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, result)

			resultMap, ok := result.(map[string]interface{})
			require.True(t, ok, "result should be a map")

			assert.True(t, resultMap["success"].(bool))

			actualResult, ok := resultMap["result"].(string)
			require.True(t, ok, "result should be string")

			if tt.wantResult != "" {
				assert.Equal(t, tt.wantResult, actualResult)
			}

			// Check that string_length is present and correct
			assert.Equal(t, len(actualResult), int(resultMap["string_length"].(int)))
		})
	}
}

func TestJsonToStringExecutor_PrettyPrint(t *testing.T) {
	executor := NewJsonToStringExecutor()
	ctx := context.Background()

	config := map[string]interface{}{
		"pretty":      true,
		"indent":      "  ",
		"escape_html": true,
		"sort_keys":   false,
	}

	input := map[string]interface{}{
		"name": "John",
		"age":  30,
	}

	result, err := executor.Execute(ctx, config, input)
	require.NoError(t, err)

	resultMap := result.(map[string]interface{})
	jsonStr := resultMap["result"].(string)

	// Check that output contains indentation
	assert.Contains(t, jsonStr, "\n", "pretty output should contain newlines")
	assert.Contains(t, jsonStr, "  ", "pretty output should contain indentation")

	// Verify it's still valid JSON
	var parsed interface{}
	err = json.Unmarshal([]byte(jsonStr), &parsed)
	assert.NoError(t, err, "pretty output should be valid JSON")
}

func TestJsonToStringExecutor_HTMLEscaping(t *testing.T) {
	executor := NewJsonToStringExecutor()
	ctx := context.Background()

	input := map[string]interface{}{
		"script": "<script>alert('xss')</script>",
		"link":   "<a href='http://example.com'>click</a>",
	}

	// Test with escaping enabled
	t.Run("escaping enabled", func(t *testing.T) {
		config := map[string]interface{}{
			"pretty":      false,
			"escape_html": true,
			"sort_keys":   false,
		}

		result, err := executor.Execute(ctx, config, input)
		require.NoError(t, err)

		resultMap := result.(map[string]interface{})
		jsonStr := resultMap["result"].(string)

		// HTML characters should be escaped
		assert.Contains(t, jsonStr, "\\u003c", "< should be escaped")
		assert.Contains(t, jsonStr, "\\u003e", "> should be escaped")
	})

	// Test with escaping disabled
	t.Run("escaping disabled", func(t *testing.T) {
		config := map[string]interface{}{
			"pretty":      false,
			"escape_html": false,
			"sort_keys":   false,
		}

		result, err := executor.Execute(ctx, config, input)
		require.NoError(t, err)

		resultMap := result.(map[string]interface{})
		jsonStr := resultMap["result"].(string)

		// HTML characters should NOT be escaped
		assert.Contains(t, jsonStr, "<", "< should not be escaped")
		assert.Contains(t, jsonStr, ">", "> should not be escaped")
	})
}

func TestJsonToStringExecutor_KeySorting(t *testing.T) {
	executor := NewJsonToStringExecutor()
	ctx := context.Background()

	tests := []struct {
		name       string
		sortKeys   bool
		input      map[string]interface{}
		wantResult string
	}{
		{
			name:     "sorted keys",
			sortKeys: true,
			input: map[string]interface{}{
				"zebra": 1,
				"apple": 2,
				"mango": 3,
			},
			wantResult: `{"apple":2,"mango":3,"zebra":1}`,
		},
		{
			name:     "unsorted keys (order may vary)",
			sortKeys: false,
			input: map[string]interface{}{
				"a": 1,
				"b": 2,
			},
			// Just check it doesn't error
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := map[string]interface{}{
				"pretty":      false,
				"escape_html": true,
				"sort_keys":   tt.sortKeys,
			}

			result, err := executor.Execute(ctx, config, tt.input)
			require.NoError(t, err)

			resultMap := result.(map[string]interface{})
			jsonStr := resultMap["result"].(string)

			if tt.wantResult != "" {
				assert.Equal(t, tt.wantResult, jsonStr)
			}
		})
	}
}

func TestStringToJsonExecutor_Validate(t *testing.T) {
	executor := NewStringToJsonExecutor()

	// All configs should be valid since we have defaults
	config := map[string]interface{}{}
	err := executor.Validate(config)
	assert.NoError(t, err)
}

func TestJsonToStringExecutor_Validate(t *testing.T) {
	executor := NewJsonToStringExecutor()

	// All configs should be valid since we have defaults
	config := map[string]interface{}{}
	err := executor.Validate(config)
	assert.NoError(t, err)
}
