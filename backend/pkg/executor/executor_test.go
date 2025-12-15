package executor

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ==================== ExecutorFunc Tests ====================

func TestExecutorFunc_Execute(t *testing.T) {
	ctx := context.Background()
	config := map[string]interface{}{"key": "value"}
	input := map[string]interface{}{"data": "test"}

	tests := []struct {
		name      string
		executeFn func(ctx context.Context, config map[string]interface{}, input interface{}) (interface{}, error)
		wantOut   interface{}
		wantErr   bool
	}{
		{
			name: "successful execution",
			executeFn: func(ctx context.Context, config map[string]interface{}, input interface{}) (interface{}, error) {
				return map[string]interface{}{"result": "success"}, nil
			},
			wantOut: map[string]interface{}{"result": "success"},
			wantErr: false,
		},
		{
			name: "execution with error",
			executeFn: func(ctx context.Context, config map[string]interface{}, input interface{}) (interface{}, error) {
				return nil, errors.New("execution failed")
			},
			wantOut: nil,
			wantErr: true,
		},
		{
			name: "execution returns input",
			executeFn: func(ctx context.Context, config map[string]interface{}, input interface{}) (interface{}, error) {
				return input, nil
			},
			wantOut: input,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			exec := &ExecutorFunc{
				ExecuteFn: tt.executeFn,
			}

			out, err := exec.Execute(ctx, config, input)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantOut, out)
			}
		})
	}
}

func TestExecutorFunc_Validate(t *testing.T) {
	config := map[string]interface{}{"required_field": "value"}

	tests := []struct {
		name       string
		validateFn func(config map[string]interface{}) error
		wantErr    bool
	}{
		{
			name: "validation success",
			validateFn: func(config map[string]interface{}) error {
				return nil
			},
			wantErr: false,
		},
		{
			name: "validation error",
			validateFn: func(config map[string]interface{}) error {
				return errors.New("validation failed")
			},
			wantErr: true,
		},
		{
			name:       "nil validate function",
			validateFn: nil,
			wantErr:    false, // Should not error when ValidateFn is nil
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			exec := &ExecutorFunc{
				ValidateFn: tt.validateFn,
			}

			err := exec.Validate(config)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestNewExecutorFunc(t *testing.T) {
	executeFn := func(ctx context.Context, config map[string]interface{}, input interface{}) (interface{}, error) {
		return "result", nil
	}
	validateFn := func(config map[string]interface{}) error {
		return nil
	}

	exec := NewExecutorFunc(executeFn, validateFn)

	assert.NotNil(t, exec)

	// Verify it implements Executor interface
	_, ok := exec.(Executor)
	assert.True(t, ok, "NewExecutorFunc should return an Executor")

	// Test execution
	out, err := exec.Execute(context.Background(), nil, nil)
	assert.NoError(t, err)
	assert.Equal(t, "result", out)

	// Test validation
	err = exec.Validate(nil)
	assert.NoError(t, err)
}

func TestNewExecutorFunc_NilValidate(t *testing.T) {
	executeFn := func(ctx context.Context, config map[string]interface{}, input interface{}) (interface{}, error) {
		return "result", nil
	}

	exec := NewExecutorFunc(executeFn, nil)

	assert.NotNil(t, exec)

	// Validate should not panic with nil ValidateFn
	err := exec.Validate(nil)
	assert.NoError(t, err)
}

// ==================== BaseExecutor Tests ====================

func TestNewBaseExecutor(t *testing.T) {
	base := NewBaseExecutor("test-type")

	assert.NotNil(t, base)
	assert.Equal(t, "test-type", base.NodeType)
}

func TestBaseExecutor_ValidateRequired(t *testing.T) {
	base := NewBaseExecutor("test")

	tests := []struct {
		name     string
		config   map[string]interface{}
		fields   []string
		wantErr  bool
		errField string
	}{
		{
			name:    "all fields present",
			config:  map[string]interface{}{"field1": "value1", "field2": "value2"},
			fields:  []string{"field1", "field2"},
			wantErr: false,
		},
		{
			name:     "missing required field",
			config:   map[string]interface{}{"field1": "value1"},
			fields:   []string{"field1", "field2"},
			wantErr:  true,
			errField: "field2",
		},
		{
			name:    "empty fields list",
			config:  map[string]interface{}{"field1": "value1"},
			fields:  []string{},
			wantErr: false,
		},
		{
			name:    "nil config with no required fields",
			config:  nil,
			fields:  []string{},
			wantErr: false,
		},
		{
			name:     "nil config with required fields",
			config:   nil,
			fields:   []string{"field1"},
			wantErr:  true,
			errField: "field1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := base.ValidateRequired(tt.config, tt.fields...)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errField != "" {
					assert.Contains(t, err.Error(), tt.errField)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestBaseExecutor_GetString(t *testing.T) {
	base := NewBaseExecutor("test")

	tests := []struct {
		name    string
		config  map[string]interface{}
		key     string
		wantVal string
		wantErr bool
		errMsg  string
	}{
		{
			name:    "get string success",
			config:  map[string]interface{}{"key": "value"},
			key:     "key",
			wantVal: "value",
			wantErr: false,
		},
		{
			name:    "field not found",
			config:  map[string]interface{}{"other": "value"},
			key:     "key",
			wantVal: "",
			wantErr: true,
			errMsg:  "field not found",
		},
		{
			name:    "field is not a string",
			config:  map[string]interface{}{"key": 123},
			key:     "key",
			wantVal: "",
			wantErr: true,
			errMsg:  "not a string",
		},
		{
			name:    "empty string is valid",
			config:  map[string]interface{}{"key": ""},
			key:     "key",
			wantVal: "",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			val, err := base.GetString(tt.config, tt.key)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantVal, val)
			}
		})
	}
}

func TestBaseExecutor_GetStringDefault(t *testing.T) {
	base := NewBaseExecutor("test")

	tests := []struct {
		name         string
		config       map[string]interface{}
		key          string
		defaultValue string
		wantVal      string
	}{
		{
			name:         "get existing string",
			config:       map[string]interface{}{"key": "value"},
			key:          "key",
			defaultValue: "default",
			wantVal:      "value",
		},
		{
			name:         "use default when field not found",
			config:       map[string]interface{}{"other": "value"},
			key:          "key",
			defaultValue: "default",
			wantVal:      "default",
		},
		{
			name:         "use default when field is not a string",
			config:       map[string]interface{}{"key": 123},
			key:          "key",
			defaultValue: "default",
			wantVal:      "default",
		},
		{
			name:         "empty string overrides default",
			config:       map[string]interface{}{"key": ""},
			key:          "key",
			defaultValue: "default",
			wantVal:      "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			val := base.GetStringDefault(tt.config, tt.key, tt.defaultValue)
			assert.Equal(t, tt.wantVal, val)
		})
	}
}

func TestBaseExecutor_GetInt(t *testing.T) {
	base := NewBaseExecutor("test")

	tests := []struct {
		name    string
		config  map[string]interface{}
		key     string
		wantVal int
		wantErr bool
		errMsg  string
	}{
		{
			name:    "get int success",
			config:  map[string]interface{}{"key": 42},
			key:     "key",
			wantVal: 42,
			wantErr: false,
		},
		{
			name:    "get float64 as int (JSON unmarshaling)",
			config:  map[string]interface{}{"key": float64(42.0)},
			key:     "key",
			wantVal: 42,
			wantErr: false,
		},
		{
			name:    "get float64 with decimals truncated",
			config:  map[string]interface{}{"key": float64(42.7)},
			key:     "key",
			wantVal: 42,
			wantErr: false,
		},
		{
			name:    "field not found",
			config:  map[string]interface{}{"other": 42},
			key:     "key",
			wantVal: 0,
			wantErr: true,
			errMsg:  "field not found",
		},
		{
			name:    "field is not a number",
			config:  map[string]interface{}{"key": "not a number"},
			key:     "key",
			wantVal: 0,
			wantErr: true,
			errMsg:  "not a number",
		},
		{
			name:    "zero is valid",
			config:  map[string]interface{}{"key": 0},
			key:     "key",
			wantVal: 0,
			wantErr: false,
		},
		{
			name:    "negative int",
			config:  map[string]interface{}{"key": -42},
			key:     "key",
			wantVal: -42,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			val, err := base.GetInt(tt.config, tt.key)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantVal, val)
			}
		})
	}
}

func TestBaseExecutor_GetIntDefault(t *testing.T) {
	base := NewBaseExecutor("test")

	tests := []struct {
		name         string
		config       map[string]interface{}
		key          string
		defaultValue int
		wantVal      int
	}{
		{
			name:         "get existing int",
			config:       map[string]interface{}{"key": 42},
			key:          "key",
			defaultValue: 100,
			wantVal:      42,
		},
		{
			name:         "get float64 as int",
			config:       map[string]interface{}{"key": float64(42.0)},
			key:          "key",
			defaultValue: 100,
			wantVal:      42,
		},
		{
			name:         "use default when field not found",
			config:       map[string]interface{}{"other": 42},
			key:          "key",
			defaultValue: 100,
			wantVal:      100,
		},
		{
			name:         "use default when field is not a number",
			config:       map[string]interface{}{"key": "not a number"},
			key:          "key",
			defaultValue: 100,
			wantVal:      100,
		},
		{
			name:         "zero overrides default",
			config:       map[string]interface{}{"key": 0},
			key:          "key",
			defaultValue: 100,
			wantVal:      0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			val := base.GetIntDefault(tt.config, tt.key, tt.defaultValue)
			assert.Equal(t, tt.wantVal, val)
		})
	}
}

func TestBaseExecutor_GetBool(t *testing.T) {
	base := NewBaseExecutor("test")

	tests := []struct {
		name    string
		config  map[string]interface{}
		key     string
		wantVal bool
		wantErr bool
		errMsg  string
	}{
		{
			name:    "get true",
			config:  map[string]interface{}{"key": true},
			key:     "key",
			wantVal: true,
			wantErr: false,
		},
		{
			name:    "get false",
			config:  map[string]interface{}{"key": false},
			key:     "key",
			wantVal: false,
			wantErr: false,
		},
		{
			name:    "field not found",
			config:  map[string]interface{}{"other": true},
			key:     "key",
			wantVal: false,
			wantErr: true,
			errMsg:  "field not found",
		},
		{
			name:    "field is not a boolean",
			config:  map[string]interface{}{"key": "not a bool"},
			key:     "key",
			wantVal: false,
			wantErr: true,
			errMsg:  "not a boolean",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			val, err := base.GetBool(tt.config, tt.key)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantVal, val)
			}
		})
	}
}

func TestBaseExecutor_GetBoolDefault(t *testing.T) {
	base := NewBaseExecutor("test")

	tests := []struct {
		name         string
		config       map[string]interface{}
		key          string
		defaultValue bool
		wantVal      bool
	}{
		{
			name:         "get existing true",
			config:       map[string]interface{}{"key": true},
			key:          "key",
			defaultValue: false,
			wantVal:      true,
		},
		{
			name:         "get existing false",
			config:       map[string]interface{}{"key": false},
			key:          "key",
			defaultValue: true,
			wantVal:      false,
		},
		{
			name:         "use default when field not found",
			config:       map[string]interface{}{"other": true},
			key:          "key",
			defaultValue: true,
			wantVal:      true,
		},
		{
			name:         "use default when field is not a boolean",
			config:       map[string]interface{}{"key": "not a bool"},
			key:          "key",
			defaultValue: true,
			wantVal:      true,
		},
		{
			name:         "false overrides default",
			config:       map[string]interface{}{"key": false},
			key:          "key",
			defaultValue: true,
			wantVal:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			val := base.GetBoolDefault(tt.config, tt.key, tt.defaultValue)
			assert.Equal(t, tt.wantVal, val)
		})
	}
}

func TestBaseExecutor_GetMap(t *testing.T) {
	base := NewBaseExecutor("test")

	tests := []struct {
		name    string
		config  map[string]interface{}
		key     string
		wantVal map[string]interface{}
		wantErr bool
		errMsg  string
	}{
		{
			name:    "get map success",
			config:  map[string]interface{}{"key": map[string]interface{}{"nested": "value"}},
			key:     "key",
			wantVal: map[string]interface{}{"nested": "value"},
			wantErr: false,
		},
		{
			name:    "field not found",
			config:  map[string]interface{}{"other": map[string]interface{}{}},
			key:     "key",
			wantVal: nil,
			wantErr: true,
			errMsg:  "field not found",
		},
		{
			name:    "field is not a map",
			config:  map[string]interface{}{"key": "not a map"},
			key:     "key",
			wantVal: nil,
			wantErr: true,
			errMsg:  "not a map",
		},
		{
			name:    "empty map is valid",
			config:  map[string]interface{}{"key": map[string]interface{}{}},
			key:     "key",
			wantVal: map[string]interface{}{},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			val, err := base.GetMap(tt.config, tt.key)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantVal, val)
			}
		})
	}
}

// ==================== ExecutionContext Tests ====================

func TestExecutionContext(t *testing.T) {
	// Test struct creation
	ctx := &ExecutionContext{
		ExecutionID: "exec-123",
		NodeID:      "node-456",
		WorkflowID:  "workflow-789",
		Metadata: map[string]interface{}{
			"key":   "value",
			"count": 42,
		},
	}

	assert.Equal(t, "exec-123", ctx.ExecutionID)
	assert.Equal(t, "node-456", ctx.NodeID)
	assert.Equal(t, "workflow-789", ctx.WorkflowID)
	assert.NotNil(t, ctx.Metadata)
	assert.Equal(t, "value", ctx.Metadata["key"])
	assert.Equal(t, 42, ctx.Metadata["count"])
}

func TestExecutionContext_EmptyMetadata(t *testing.T) {
	ctx := &ExecutionContext{
		ExecutionID: "exec-123",
		NodeID:      "node-456",
		WorkflowID:  "workflow-789",
		Metadata:    nil,
	}

	assert.Nil(t, ctx.Metadata)
}

// ==================== Integration Tests ====================

func TestBaseExecutor_IntegrationScenario(t *testing.T) {
	base := NewBaseExecutor("http")

	// Simulate HTTP executor config
	config := map[string]interface{}{
		"url":     "https://api.example.com/users",
		"method":  "POST",
		"timeout": float64(30),
		"retry":   true,
		"headers": map[string]interface{}{
			"Content-Type": "application/json",
		},
	}

	// Validate required fields
	err := base.ValidateRequired(config, "url", "method")
	require.NoError(t, err)

	// Get string values
	url, err := base.GetString(config, "url")
	require.NoError(t, err)
	assert.Equal(t, "https://api.example.com/users", url)

	method, err := base.GetString(config, "method")
	require.NoError(t, err)
	assert.Equal(t, "POST", method)

	// Get int with default
	timeout := base.GetIntDefault(config, "timeout", 10)
	assert.Equal(t, 30, timeout)

	// Get bool
	retry, err := base.GetBool(config, "retry")
	require.NoError(t, err)
	assert.True(t, retry)

	// Get map
	headers, err := base.GetMap(config, "headers")
	require.NoError(t, err)
	assert.Equal(t, "application/json", headers["Content-Type"])

	// Get optional field with default
	maxRetries := base.GetIntDefault(config, "max_retries", 3)
	assert.Equal(t, 3, maxRetries)
}
