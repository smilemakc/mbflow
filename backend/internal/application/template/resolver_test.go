package template

import (
	"errors"
	"reflect"
	"testing"
)

func TestResolver_ResolveVariable(t *testing.T) {
	ctx := NewVariableContext()
	ctx.WorkflowVars["simpleVar"] = "value"
	ctx.InputVars["data"] = map[string]any{
		"name": "test",
	}

	resolver := NewResolver(ctx, DefaultOptions())

	tests := []struct {
		name     string
		varType  string
		path     string
		want     any
		wantErr  bool
		errCheck func(error) bool
	}{
		{
			name:    "valid env variable",
			varType: "env",
			path:    "simpleVar",
			want:    "value",
			wantErr: false,
		},
		{
			name:    "valid input variable",
			varType: "input",
			path:    "data.name",
			want:    "test",
			wantErr: false,
		},
		{
			name:    "env without path",
			varType: "env",
			path:    "",
			wantErr: true,
			errCheck: func(err error) bool {
				return errors.Is(err, ErrInvalidTemplate)
			},
		},
		{
			name:    "input without path",
			varType: "input",
			path:    "",
			want: map[string]any{
				"data": map[string]any{
					"name": "test",
				},
			},
			wantErr: false,
		},
		{
			name:    "unknown variable type",
			varType: "unknown",
			path:    "test",
			wantErr: true,
			errCheck: func(err error) bool {
				return errors.Is(err, ErrInvalidTemplate)
			},
		},
		{
			name:    "missing env variable",
			varType: "env",
			path:    "missing",
			wantErr: true,
			errCheck: func(err error) bool {
				return errors.Is(err, ErrVariableNotFound)
			},
		},
		{
			name:    "missing input variable",
			varType: "input",
			path:    "missing",
			wantErr: true,
			errCheck: func(err error) bool {
				return errors.Is(err, ErrVariableNotFound)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := resolver.ResolveVariable(tt.varType, tt.path)
			if (err != nil) != tt.wantErr {
				t.Errorf("ResolveVariable() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && tt.errCheck != nil {
				if !tt.errCheck(err) {
					t.Errorf("ResolveVariable() error check failed for error: %v", err)
				}
			}
			if !tt.wantErr && !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ResolveVariable() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestResolver_ResolveEnvPath(t *testing.T) {
	ctx := NewVariableContext()
	ctx.WorkflowVars["simple"] = "value"
	ctx.WorkflowVars["nested"] = map[string]any{
		"field": "nested value",
	}
	ctx.WorkflowVars["array"] = []any{
		"item0",
		map[string]any{"name": "item1"},
	}

	resolver := NewResolver(ctx, DefaultOptions())

	tests := []struct {
		name      string
		path      string
		wantValue any
		wantFound bool
	}{
		{
			name:      "empty path",
			path:      "",
			wantFound: false,
		},
		{
			name:      "simple variable",
			path:      "simple",
			wantValue: "value",
			wantFound: true,
		},
		{
			name:      "nested field",
			path:      "nested.field",
			wantValue: "nested value",
			wantFound: true,
		},
		{
			name:      "array index",
			path:      "array[0]",
			wantValue: "item0",
			wantFound: true,
		},
		{
			name:      "array element field",
			path:      "array[1].name",
			wantValue: "item1",
			wantFound: true,
		},
		{
			name:      "missing variable",
			path:      "missing",
			wantFound: false,
		},
		{
			name:      "missing nested field",
			path:      "nested.missing",
			wantFound: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, found := resolver.resolveEnvPath(tt.path)
			if found != tt.wantFound {
				t.Errorf("resolveEnvPath() found = %v, want %v", found, tt.wantFound)
				return
			}
			if tt.wantFound && got != tt.wantValue {
				t.Errorf("resolveEnvPath() = %v, want %v", got, tt.wantValue)
			}
		})
	}
}

func TestResolver_ResolveInputPath(t *testing.T) {
	ctx := NewVariableContext()
	ctx.InputVars["simple"] = "value"
	ctx.InputVars["nested"] = map[string]any{
		"field": "nested value",
	}
	ctx.InputVars["array"] = []any{
		"item0",
		map[string]any{"name": "item1"},
	}

	resolver := NewResolver(ctx, DefaultOptions())

	tests := []struct {
		name      string
		path      string
		wantValue any
		wantFound bool
	}{
		{
			name:      "empty path",
			path:      "",
			wantFound: false,
		},
		{
			name:      "simple variable",
			path:      "simple",
			wantValue: "value",
			wantFound: true,
		},
		{
			name:      "nested field",
			path:      "nested.field",
			wantValue: "nested value",
			wantFound: true,
		},
		{
			name:      "array index",
			path:      "array[0]",
			wantValue: "item0",
			wantFound: true,
		},
		{
			name:      "array element field",
			path:      "array[1].name",
			wantValue: "item1",
			wantFound: true,
		},
		{
			name:      "missing variable",
			path:      "missing",
			wantFound: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, found := resolver.resolveInputPath(tt.path)
			if found != tt.wantFound {
				t.Errorf("resolveInputPath() found = %v, want %v", found, tt.wantFound)
				return
			}
			if tt.wantFound && got != tt.wantValue {
				t.Errorf("resolveInputPath() = %v, want %v", got, tt.wantValue)
			}
		})
	}
}

func TestResolver_ResolveField(t *testing.T) {
	ctx := NewVariableContext()
	resolver := NewResolver(ctx, DefaultOptions())

	tests := []struct {
		name     string
		value    any
		field    string
		wantNil  bool
		wantType string
	}{
		{
			name:    "nil value",
			value:   nil,
			field:   "field",
			wantNil: true,
		},
		{
			name: "map access",
			value: map[string]any{
				"field": "value",
			},
			field:    "field",
			wantType: "string",
		},
		{
			name: "struct access",
			value: struct {
				Field string
			}{Field: "value"},
			field:    "Field",
			wantType: "string",
		},
		{
			name: "pointer to struct",
			value: &struct {
				Field string
			}{Field: "value"},
			field:    "Field",
			wantType: "string",
		},
		{
			name: "missing map field",
			value: map[string]any{
				"other": "value",
			},
			field:   "missing",
			wantNil: true,
		},
		{
			name: "missing struct field",
			value: struct {
				Field string
			}{Field: "value"},
			field:   "Missing",
			wantNil: true,
		},
		{
			name: "json marshalable struct",
			value: struct {
				Field string `json:"field"`
			}{Field: "value"},
			field:    "field",
			wantType: "string",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := resolver.resolveField(tt.value, tt.field)
			if tt.wantNil {
				if got != nil {
					t.Errorf("resolveField() = %v, want nil", got)
				}
			} else {
				if got == nil {
					t.Error("resolveField() = nil, want non-nil")
				}
			}
		})
	}
}

func TestResolver_ResolveArrayIndex(t *testing.T) {
	ctx := NewVariableContext()
	resolver := NewResolver(ctx, DefaultOptions())

	tests := []struct {
		name      string
		value     any
		indexExpr string
		wantErr   bool
		errCheck  func(error) bool
	}{
		{
			name:      "simple array index",
			value:     []any{"a", "b", "c"},
			indexExpr: "[1]",
			wantErr:   false,
		},
		{
			name: "field with array index",
			value: map[string]any{
				"items": []any{"a", "b"},
			},
			indexExpr: "items[0]",
			wantErr:   false,
		},
		{
			name:      "chained array index",
			value:     []any{[]any{"a", "b"}, []any{"c", "d"}},
			indexExpr: "[0][1]",
			wantErr:   false,
		},
		{
			name: "missing field",
			value: map[string]any{
				"other": "value",
			},
			indexExpr: "missing[0]",
			wantErr:   true,
			errCheck: func(err error) bool {
				return errors.Is(err, ErrInvalidPath)
			},
		},
		{
			name:      "invalid index expression",
			value:     []any{"a"},
			indexExpr: "[invalid]",
			wantErr:   true,
			errCheck: func(err error) bool {
				return errors.Is(err, ErrArrayIndexInvalid)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := resolver.resolveArrayIndex(tt.value, tt.indexExpr)
			if (err != nil) != tt.wantErr {
				t.Errorf("resolveArrayIndex() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && tt.errCheck != nil {
				if !tt.errCheck(err) {
					t.Errorf("resolveArrayIndex() error check failed for error: %v", err)
				}
			}
		})
	}
}

func TestResolver_IndexArray(t *testing.T) {
	ctx := NewVariableContext()
	resolver := NewResolver(ctx, DefaultOptions())

	tests := []struct {
		name     string
		value    any
		index    int
		wantErr  bool
		errCheck func(error) bool
	}{
		{
			name:    "nil value",
			value:   nil,
			index:   0,
			wantErr: true,
			errCheck: func(err error) bool {
				return errors.Is(err, ErrTypeNotSupported)
			},
		},
		{
			name:    "slice access",
			value:   []any{"a", "b", "c"},
			index:   1,
			wantErr: false,
		},
		{
			name:    "array access",
			value:   [3]string{"a", "b", "c"},
			index:   1,
			wantErr: false,
		},
		{
			name:    "slice out of bounds",
			value:   []any{"a"},
			index:   5,
			wantErr: true,
			errCheck: func(err error) bool {
				return errors.Is(err, ErrArrayOutOfBounds)
			},
		},
		{
			name:    "negative index",
			value:   []any{"a", "b"},
			index:   -1,
			wantErr: true,
			errCheck: func(err error) bool {
				return errors.Is(err, ErrArrayOutOfBounds)
			},
		},
		{
			name: "json array",
			value: []map[string]any{
				{"name": "item1"},
				{"name": "item2"},
			},
			index:   0,
			wantErr: false,
		},
		{
			name: "json array out of bounds",
			value: []map[string]any{
				{"name": "item1"},
			},
			index:   5,
			wantErr: true,
			errCheck: func(err error) bool {
				return errors.Is(err, ErrArrayOutOfBounds)
			},
		},
		{
			name: "json array negative index",
			value: []map[string]any{
				{"name": "item1"},
			},
			index:   -1,
			wantErr: true,
			errCheck: func(err error) bool {
				return errors.Is(err, ErrArrayOutOfBounds)
			},
		},
		{
			name:    "non-array type",
			value:   "not an array",
			index:   0,
			wantErr: true,
			errCheck: func(err error) bool {
				return errors.Is(err, ErrTypeNotSupported)
			},
		},
		{
			name:    "map type (not array)",
			value:   map[string]any{"key": "value"},
			index:   0,
			wantErr: true,
			errCheck: func(err error) bool {
				return errors.Is(err, ErrTypeNotSupported)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := resolver.indexArray(tt.value, tt.index)
			if (err != nil) != tt.wantErr {
				t.Errorf("indexArray() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && tt.errCheck != nil {
				if !tt.errCheck(err) {
					t.Errorf("indexArray() error check failed for error: %v", err)
				}
			}
		})
	}
}

func TestSplitPath(t *testing.T) {
	tests := []struct {
		name string
		path string
		want []string
	}{
		{
			name: "empty path",
			path: "",
			want: nil,
		},
		{
			name: "simple path",
			path: "field",
			want: []string{"field"},
		},
		{
			name: "nested path",
			path: "user.profile.name",
			want: []string{"user", "profile", "name"},
		},
		{
			name: "path with array index",
			path: "items[0].name",
			want: []string{"items[0]", "name"},
		},
		{
			name: "path with multiple array indices",
			path: "matrix[0][1].value",
			want: []string{"matrix[0][1]", "value"},
		},
		{
			name: "complex path",
			path: "data.users[0].profile.emails[1]",
			want: []string{"data", "users[0]", "profile", "emails[1]"},
		},
		{
			name: "path with dots in brackets",
			path: "field[key.with.dots]",
			want: []string{"field[key.with.dots]"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := splitPath(tt.path)
			if len(got) != len(tt.want) {
				t.Errorf("splitPath() length = %v, want %v", len(got), len(tt.want))
				return
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("splitPath()[%d] = %v, want %v", i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestParseArrayIndices(t *testing.T) {
	tests := []struct {
		name string
		expr string
		want []int
	}{
		{
			name: "single index",
			expr: "[0]",
			want: []int{0},
		},
		{
			name: "multiple indices",
			expr: "[0][1][2]",
			want: []int{0, 1, 2},
		},
		{
			name: "index with spaces",
			expr: "[ 5 ]",
			want: []int{5},
		},
		{
			name: "no brackets",
			expr: "abc",
			want: nil,
		},
		{
			name: "invalid index",
			expr: "[abc]",
			want: nil,
		},
		{
			name: "unclosed bracket",
			expr: "[0",
			want: nil,
		},
		{
			name: "empty brackets",
			expr: "[]",
			want: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseArrayIndices(tt.expr)
			if len(got) != len(tt.want) {
				t.Errorf("parseArrayIndices() length = %v, want %v", len(got), len(tt.want))
				return
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("parseArrayIndices()[%d] = %v, want %v", i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestResolver_TraversePath(t *testing.T) {
	ctx := NewVariableContext()
	resolver := NewResolver(ctx, DefaultOptions())

	tests := []struct {
		name      string
		value     any
		parts     []string
		wantFound bool
	}{
		{
			name: "traverse nested map",
			value: map[string]any{
				"level1": map[string]any{
					"level2": "value",
				},
			},
			parts:     []string{"level1", "level2"},
			wantFound: true,
		},
		{
			name:      "traverse array",
			value:     []any{map[string]any{"name": "test"}},
			parts:     []string{"[0]", "name"},
			wantFound: true,
		},
		{
			name: "traverse missing field",
			value: map[string]any{
				"field": "value",
			},
			parts:     []string{"missing"},
			wantFound: false,
		},
		{
			name:      "empty parts",
			value:     "value",
			parts:     []string{},
			wantFound: true,
		},
		{
			name: "traverse with field and array access",
			value: map[string]any{
				"items": []any{
					map[string]any{"id": 1},
					map[string]any{"id": 2},
				},
			},
			parts:     []string{"items[1]", "id"},
			wantFound: true,
		},
		{
			name:      "traverse with invalid array index",
			value:     []any{"a", "b"},
			parts:     []string{"[10]"},
			wantFound: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, found := resolver.traversePath(tt.value, tt.parts)
			if found != tt.wantFound {
				t.Errorf("traversePath() found = %v, want %v", found, tt.wantFound)
			}
		})
	}
}

func TestResolver_ResolveEnvPath_ArrayIndexErrors(t *testing.T) {
	ctx := NewVariableContext()
	ctx.WorkflowVars["array"] = []any{"a", "b"}

	resolver := NewResolver(ctx, DefaultOptions())

	// Test array index out of bounds
	_, found := resolver.resolveEnvPath("array[10]")
	if found {
		t.Error("resolveEnvPath() should not find array[10], got found=true")
	}
}

func TestResolver_ResolveInputPath_ArrayIndexErrors(t *testing.T) {
	ctx := NewVariableContext()
	ctx.InputVars["array"] = []any{"a", "b"}

	resolver := NewResolver(ctx, DefaultOptions())

	// Test array index out of bounds
	_, found := resolver.resolveInputPath("array[10]")
	if found {
		t.Error("resolveInputPath() should not find array[10], got found=true")
	}
}

func TestResolver_ResolveField_UnmarshalError(t *testing.T) {
	ctx := NewVariableContext()
	resolver := NewResolver(ctx, DefaultOptions())

	// Test with a type that cannot be marshaled to JSON then unmarshaled to map
	type unmarshalableToMap struct {
		Ch chan int
	}

	value := unmarshalableToMap{Ch: make(chan int)}
	result := resolver.resolveField(value, "field")
	if result != nil {
		t.Errorf("resolveField() should return nil for unmarshalable type, got %v", result)
	}
}

// Test resource variable resolution
func TestResolver_ResolveResourceVariable(t *testing.T) {
	tests := []struct {
		name         string
		resourceVars map[string]any
		path         string
		expected     any
		expectError  bool
	}{
		{
			name: "resolve full resource object",
			resourceVars: map[string]any{
				"myStorage": map[string]any{
					"id":   "res-123",
					"name": "My Storage",
					"type": "file_storage",
				},
			},
			path:     "myStorage",
			expected: map[string]any{"id": "res-123", "name": "My Storage", "type": "file_storage"},
		},
		{
			name: "resolve resource id field",
			resourceVars: map[string]any{
				"myStorage": map[string]any{
					"id":   "res-123",
					"name": "My Storage",
					"type": "file_storage",
				},
			},
			path:     "myStorage.id",
			expected: "res-123",
		},
		{
			name: "resolve resource name field",
			resourceVars: map[string]any{
				"myStorage": map[string]any{
					"id":   "res-123",
					"name": "My Storage",
					"type": "file_storage",
				},
			},
			path:     "myStorage.name",
			expected: "My Storage",
		},
		{
			name: "resolve resource type field",
			resourceVars: map[string]any{
				"myStorage": map[string]any{
					"id":   "res-123",
					"name": "My Storage",
					"type": "file_storage",
				},
			},
			path:     "myStorage.type",
			expected: "file_storage",
		},
		{
			name: "resolve nested resource field",
			resourceVars: map[string]any{
				"apiResource": map[string]any{
					"id": "api-456",
					"config": map[string]any{
						"endpoint": "https://api.example.com",
						"apiKey":   "secret-key",
					},
				},
			},
			path:     "apiResource.config.endpoint",
			expected: "https://api.example.com",
		},
		{
			name: "resolve nested resource field deep",
			resourceVars: map[string]any{
				"apiResource": map[string]any{
					"id": "api-456",
					"config": map[string]any{
						"endpoint": "https://api.example.com",
						"apiKey":   "secret-key",
					},
				},
			},
			path:     "apiResource.config.apiKey",
			expected: "secret-key",
		},
		{
			name: "multiple resources - access first",
			resourceVars: map[string]any{
				"storage1": map[string]any{
					"id":   "res-001",
					"type": "s3",
				},
				"storage2": map[string]any{
					"id":   "res-002",
					"type": "local",
				},
			},
			path:     "storage1.type",
			expected: "s3",
		},
		{
			name: "multiple resources - access second",
			resourceVars: map[string]any{
				"storage1": map[string]any{
					"id":   "res-001",
					"type": "s3",
				},
				"storage2": map[string]any{
					"id":   "res-002",
					"type": "local",
				},
			},
			path:     "storage2.type",
			expected: "local",
		},
		{
			name: "resource with array field",
			resourceVars: map[string]any{
				"database": map[string]any{
					"id": "db-789",
					"tables": []any{
						"users",
						"products",
						"orders",
					},
				},
			},
			path:     "database.tables[0]",
			expected: "users",
		},
		{
			name: "resource with array of objects",
			resourceVars: map[string]any{
				"cluster": map[string]any{
					"id": "cluster-001",
					"nodes": []any{
						map[string]any{
							"id":   "node-1",
							"host": "192.168.1.1",
						},
						map[string]any{
							"id":   "node-2",
							"host": "192.168.1.2",
						},
					},
				},
			},
			path:     "cluster.nodes[1].host",
			expected: "192.168.1.2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := &VariableContext{
				ResourceVars: tt.resourceVars,
			}
			resolver := NewResolver(ctx, TemplateOptions{})

			result, err := resolver.ResolveVariable("resource", tt.path)

			if tt.expectError {
				if err == nil {
					t.Errorf("ResolveVariable() expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("ResolveVariable() unexpected error = %v", err)
					return
				}
				if !reflect.DeepEqual(result, tt.expected) {
					t.Errorf("ResolveVariable() = %v, want %v", result, tt.expected)
				}
			}
		})
	}
}

func TestResolver_ResolveResourceVariable_Errors(t *testing.T) {
	tests := []struct {
		name         string
		resourceVars map[string]any
		path         string
		errCheck     func(error) bool
	}{
		{
			name:         "empty path",
			resourceVars: map[string]any{},
			path:         "",
			errCheck: func(err error) bool {
				return errors.Is(err, ErrInvalidTemplate)
			},
		},
		{
			name:         "resource not found",
			resourceVars: map[string]any{},
			path:         "unknown",
			errCheck: func(err error) bool {
				return errors.Is(err, ErrVariableNotFound)
			},
		},
		{
			name:         "nil resource vars",
			resourceVars: nil,
			path:         "myStorage",
			errCheck: func(err error) bool {
				return errors.Is(err, ErrVariableNotFound)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := &VariableContext{
				ResourceVars: tt.resourceVars,
			}
			resolver := NewResolver(ctx, TemplateOptions{})

			result, err := resolver.ResolveVariable("resource", tt.path)
			if err == nil {
				t.Errorf("ResolveVariable() expected error, got nil with result %v", result)
				return
			}
			if tt.errCheck != nil && !tt.errCheck(err) {
				t.Errorf("ResolveVariable() error check failed for error: %v", err)
			}
		})
	}
}

func TestGetResourceVariable(t *testing.T) {
	tests := []struct {
		name         string
		resourceVars map[string]any
		alias        string
		expected     any
		expectFound  bool
	}{
		{
			name: "get existing resource",
			resourceVars: map[string]any{
				"myStorage": map[string]any{
					"id": "res-123",
				},
			},
			alias:       "myStorage",
			expected:    map[string]any{"id": "res-123"},
			expectFound: true,
		},
		{
			name:         "get non-existing resource",
			resourceVars: map[string]any{},
			alias:        "unknown",
			expected:     nil,
			expectFound:  false,
		},
		{
			name:         "nil resource vars",
			resourceVars: nil,
			alias:        "myStorage",
			expected:     nil,
			expectFound:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := &VariableContext{
				ResourceVars: tt.resourceVars,
			}

			result, found := ctx.GetResourceVariable(tt.alias)
			if found != tt.expectFound {
				t.Errorf("GetResourceVariable() found = %v, want %v", found, tt.expectFound)
			}
			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("GetResourceVariable() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestResolveResourcePath_ComplexScenarios(t *testing.T) {
	tests := []struct {
		name         string
		resourceVars map[string]any
		path         string
		expected     any
		expectFound  bool
	}{
		{
			name: "deeply nested object",
			resourceVars: map[string]any{
				"config": map[string]any{
					"database": map[string]any{
						"connections": map[string]any{
							"primary": map[string]any{
								"host": "db.example.com",
								"port": 5432,
							},
						},
					},
				},
			},
			path:        "config.database.connections.primary.host",
			expected:    "db.example.com",
			expectFound: true,
		},
		{
			name: "array within nested object",
			resourceVars: map[string]any{
				"service": map[string]any{
					"endpoints": map[string]any{
						"api": []any{
							"https://api1.example.com",
							"https://api2.example.com",
						},
					},
				},
			},
			path:        "service.endpoints.api[1]",
			expected:    "https://api2.example.com",
			expectFound: true,
		},
		{
			name: "mixed types with numbers",
			resourceVars: map[string]any{
				"metrics": map[string]any{
					"stats": map[string]any{
						"count":   100,
						"average": 75.5,
						"enabled": true,
					},
				},
			},
			path:        "metrics.stats.count",
			expected:    100,
			expectFound: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := &VariableContext{
				ResourceVars: tt.resourceVars,
			}
			resolver := NewResolver(ctx, TemplateOptions{})

			result, found := resolver.resolveResourcePath(tt.path)
			if found != tt.expectFound {
				t.Errorf("resolveResourcePath() found = %v, want %v", found, tt.expectFound)
			}
			if tt.expectFound && !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("resolveResourcePath() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestResolver_ResolveVariable_AllTypesIncludingResource(t *testing.T) {
	// Test that all variable types (env, input, resource) work together
	ctx := &VariableContext{
		WorkflowVars: map[string]any{
			"apiKey": "workflow-key",
		},
		ExecutionVars: map[string]any{
			"executionID": "exec-123",
		},
		InputVars: map[string]any{
			"userId": "user-456",
		},
		ResourceVars: map[string]any{
			"storage": map[string]any{
				"id":   "res-789",
				"type": "s3",
			},
		},
	}
	resolver := NewResolver(ctx, TemplateOptions{})

	tests := []struct {
		name     string
		varType  string
		path     string
		expected any
	}{
		{
			name:     "env variable",
			varType:  "env",
			path:     "apiKey",
			expected: "workflow-key",
		},
		{
			name:     "execution variable overrides workflow",
			varType:  "env",
			path:     "executionID",
			expected: "exec-123",
		},
		{
			name:     "input variable",
			varType:  "input",
			path:     "userId",
			expected: "user-456",
		},
		{
			name:     "resource full object",
			varType:  "resource",
			path:     "storage",
			expected: map[string]any{"id": "res-789", "type": "s3"},
		},
		{
			name:     "resource field",
			varType:  "resource",
			path:     "storage.id",
			expected: "res-789",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := resolver.ResolveVariable(tt.varType, tt.path)
			if err != nil {
				t.Errorf("ResolveVariable() error = %v", err)
				return
			}
			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("ResolveVariable() = %v, want %v", result, tt.expected)
			}
		})
	}
}
