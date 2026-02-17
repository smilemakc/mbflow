package builtin

import (
	"context"
	"encoding/base64"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBase64ToBytesExecutor_Execute(t *testing.T) {
	executor := NewBase64ToBytesExecutor()
	ctx := context.Background()

	tests := []struct {
		name       string
		config     map[string]any
		input      any
		wantResult []byte
		wantErr    bool
	}{
		{
			name: "standard encoding - string input",
			config: map[string]any{
				"encoding":      "standard",
				"output_format": "raw",
			},
			input:      base64.StdEncoding.EncodeToString([]byte("Hello, World!")),
			wantResult: []byte("Hello, World!"),
			wantErr:    false,
		},
		{
			name: "URL encoding - string input",
			config: map[string]any{
				"encoding":      "url",
				"output_format": "raw",
			},
			input:      base64.URLEncoding.EncodeToString([]byte("Hello+World=")),
			wantResult: []byte("Hello+World="),
			wantErr:    false,
		},
		{
			name: "hex output format",
			config: map[string]any{
				"encoding":      "standard",
				"output_format": "hex",
			},
			input:      base64.StdEncoding.EncodeToString([]byte{0xDE, 0xAD, 0xBE, 0xEF}),
			wantResult: nil, // Will check hex string instead
			wantErr:    false,
		},
		{
			name: "map input with data field",
			config: map[string]any{
				"encoding":      "standard",
				"output_format": "raw",
			},
			input: map[string]any{
				"data": base64.StdEncoding.EncodeToString([]byte("test")),
			},
			wantResult: []byte("test"),
			wantErr:    false,
		},
		{
			name: "raw standard encoding",
			config: map[string]any{
				"encoding":      "raw_standard",
				"output_format": "raw",
			},
			input:      base64.RawStdEncoding.EncodeToString([]byte("no padding")),
			wantResult: []byte("no padding"),
			wantErr:    false,
		},
		{
			name: "invalid base64 string",
			config: map[string]any{
				"encoding":      "standard",
				"output_format": "raw",
			},
			input:   "not-valid-base64!!",
			wantErr: true,
		},
		{
			name: "unsupported input type",
			config: map[string]any{
				"encoding":      "standard",
				"output_format": "raw",
			},
			input:   12345,
			wantErr: true,
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

			resultMap, ok := result.(map[string]any)
			require.True(t, ok, "result should be a map")

			assert.True(t, resultMap["success"].(bool))

			if tt.wantResult != nil {
				actualBytes, ok := resultMap["result"].([]byte)
				require.True(t, ok, "result should be []byte for raw format")
				assert.Equal(t, tt.wantResult, actualBytes)
			}

			// Check hex output if that's what we're testing
			if tt.config["output_format"] == "hex" {
				_, ok := resultMap["result"].(string)
				assert.True(t, ok, "result should be string for hex format")
			}
		})
	}
}

func TestBase64ToBytesExecutor_Validate(t *testing.T) {
	executor := NewBase64ToBytesExecutor()

	tests := []struct {
		name    string
		config  map[string]any
		wantErr bool
	}{
		{
			name: "valid config - standard",
			config: map[string]any{
				"encoding":      "standard",
				"output_format": "raw",
			},
			wantErr: false,
		},
		{
			name: "valid config - url encoding",
			config: map[string]any{
				"encoding":      "url",
				"output_format": "hex",
			},
			wantErr: false,
		},
		{
			name: "invalid encoding",
			config: map[string]any{
				"encoding":      "invalid",
				"output_format": "raw",
			},
			wantErr: true,
		},
		{
			name: "invalid output format",
			config: map[string]any{
				"encoding":      "standard",
				"output_format": "invalid",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := executor.Validate(tt.config)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestBytesToBase64Executor_Execute(t *testing.T) {
	executor := NewBytesToBase64Executor()
	ctx := context.Background()

	tests := []struct {
		name    string
		config  map[string]any
		input   any
		want    string
		wantErr bool
	}{
		{
			name: "standard encoding - bytes input",
			config: map[string]any{
				"encoding":    "standard",
				"line_length": 0,
			},
			input: []byte("Hello, World!"),
			want:  base64.StdEncoding.EncodeToString([]byte("Hello, World!")),
		},
		{
			name: "URL encoding - string input",
			config: map[string]any{
				"encoding":    "url",
				"line_length": 0,
			},
			input: "test data",
			want:  base64.URLEncoding.EncodeToString([]byte("test data")),
		},
		{
			name: "raw standard encoding",
			config: map[string]any{
				"encoding":    "raw_standard",
				"line_length": 0,
			},
			input: []byte("no padding"),
			want:  base64.RawStdEncoding.EncodeToString([]byte("no padding")),
		},
		{
			name: "with line wrapping",
			config: map[string]any{
				"encoding":    "standard",
				"line_length": 10,
			},
			input:   []byte("This is a long string that needs wrapping"),
			wantErr: false, // Just check it doesn't error
		},
		{
			name: "map input with data field",
			config: map[string]any{
				"encoding":    "standard",
				"line_length": 0,
			},
			input: map[string]any{
				"data": []byte("from map"),
			},
			want: base64.StdEncoding.EncodeToString([]byte("from map")),
		},
		{
			name: "string with base64 input (should decode then re-encode)",
			config: map[string]any{
				"encoding":    "standard",
				"line_length": 0,
			},
			input:   base64.StdEncoding.EncodeToString([]byte("test")),
			wantErr: false,
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

			resultMap, ok := result.(map[string]any)
			require.True(t, ok, "result should be a map")

			assert.True(t, resultMap["success"].(bool))

			if tt.want != "" {
				actualResult, ok := resultMap["result"].(string)
				require.True(t, ok, "result should be string")
				assert.Equal(t, tt.want, actualResult)
			}

			// Check that size fields are present
			assert.NotNil(t, resultMap["original_size"])
			assert.NotNil(t, resultMap["encoded_size"])
		})
	}
}

func TestBytesToBase64Executor_Validate(t *testing.T) {
	executor := NewBytesToBase64Executor()

	tests := []struct {
		name    string
		config  map[string]any
		wantErr bool
	}{
		{
			name: "valid config",
			config: map[string]any{
				"encoding":    "standard",
				"line_length": 76,
			},
			wantErr: false,
		},
		{
			name: "invalid encoding",
			config: map[string]any{
				"encoding":    "invalid",
				"line_length": 0,
			},
			wantErr: true,
		},
		{
			name: "negative line length",
			config: map[string]any{
				"encoding":    "standard",
				"line_length": -1,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := executor.Validate(tt.config)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestBytesToBase64Executor_LineWrapping(t *testing.T) {
	executor := NewBytesToBase64Executor()
	ctx := context.Background()

	// Create a long string that will definitely need wrapping
	longData := make([]byte, 100)
	for i := range longData {
		longData[i] = byte('A' + (i % 26))
	}

	config := map[string]any{
		"encoding":    "standard",
		"line_length": 20,
	}

	result, err := executor.Execute(ctx, config, longData)
	require.NoError(t, err)

	resultMap := result.(map[string]any)
	encoded := resultMap["result"].(string)

	// Check that result contains newlines
	assert.Contains(t, encoded, "\n", "wrapped output should contain newlines")

	// Check that no line exceeds the limit (except possibly the last one)
	lines := strings.Split(encoded, "\n")
	for i, line := range lines {
		if i < len(lines)-1 { // Not the last line
			assert.LessOrEqual(t, len(line), 20, "line %d exceeds line_length", i)
		}
	}
}
