package template

import (
	"errors"
	"testing"
)

func TestResolver_ResolveVariable(t *testing.T) {
	ctx := NewVariableContext()
	ctx.WorkflowVars["simpleVar"] = "value"
	ctx.InputVars["data"] = map[string]interface{}{
		"name": "test",
	}

	resolver := NewResolver(ctx, DefaultOptions())

	tests := []struct {
		name     string
		varType  string
		path     string
		want     interface{}
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
			wantErr: true,
			errCheck: func(err error) bool {
				return errors.Is(err, ErrInvalidTemplate)
			},
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
			if !tt.wantErr && got != tt.want {
				t.Errorf("ResolveVariable() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestResolver_ResolveEnvPath(t *testing.T) {
	ctx := NewVariableContext()
	ctx.WorkflowVars["simple"] = "value"
	ctx.WorkflowVars["nested"] = map[string]interface{}{
		"field": "nested value",
	}
	ctx.WorkflowVars["array"] = []interface{}{
		"item0",
		map[string]interface{}{"name": "item1"},
	}

	resolver := NewResolver(ctx, DefaultOptions())

	tests := []struct {
		name      string
		path      string
		wantValue interface{}
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
	ctx.InputVars["nested"] = map[string]interface{}{
		"field": "nested value",
	}
	ctx.InputVars["array"] = []interface{}{
		"item0",
		map[string]interface{}{"name": "item1"},
	}

	resolver := NewResolver(ctx, DefaultOptions())

	tests := []struct {
		name      string
		path      string
		wantValue interface{}
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
		value    interface{}
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
			value: map[string]interface{}{
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
			value: map[string]interface{}{
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
		value     interface{}
		indexExpr string
		wantErr   bool
		errCheck  func(error) bool
	}{
		{
			name:      "simple array index",
			value:     []interface{}{"a", "b", "c"},
			indexExpr: "[1]",
			wantErr:   false,
		},
		{
			name: "field with array index",
			value: map[string]interface{}{
				"items": []interface{}{"a", "b"},
			},
			indexExpr: "items[0]",
			wantErr:   false,
		},
		{
			name:      "chained array index",
			value:     []interface{}{[]interface{}{"a", "b"}, []interface{}{"c", "d"}},
			indexExpr: "[0][1]",
			wantErr:   false,
		},
		{
			name: "missing field",
			value: map[string]interface{}{
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
			value:     []interface{}{"a"},
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
		value    interface{}
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
			value:   []interface{}{"a", "b", "c"},
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
			value:   []interface{}{"a"},
			index:   5,
			wantErr: true,
			errCheck: func(err error) bool {
				return errors.Is(err, ErrArrayOutOfBounds)
			},
		},
		{
			name:    "negative index",
			value:   []interface{}{"a", "b"},
			index:   -1,
			wantErr: true,
			errCheck: func(err error) bool {
				return errors.Is(err, ErrArrayOutOfBounds)
			},
		},
		{
			name: "json array",
			value: []map[string]interface{}{
				{"name": "item1"},
				{"name": "item2"},
			},
			index:   0,
			wantErr: false,
		},
		{
			name: "json array out of bounds",
			value: []map[string]interface{}{
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
			value: []map[string]interface{}{
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
			value:   map[string]interface{}{"key": "value"},
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
		value     interface{}
		parts     []string
		wantFound bool
	}{
		{
			name: "traverse nested map",
			value: map[string]interface{}{
				"level1": map[string]interface{}{
					"level2": "value",
				},
			},
			parts:     []string{"level1", "level2"},
			wantFound: true,
		},
		{
			name:      "traverse array",
			value:     []interface{}{map[string]interface{}{"name": "test"}},
			parts:     []string{"[0]", "name"},
			wantFound: true,
		},
		{
			name: "traverse missing field",
			value: map[string]interface{}{
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
			value: map[string]interface{}{
				"items": []interface{}{
					map[string]interface{}{"id": 1},
					map[string]interface{}{"id": 2},
				},
			},
			parts:     []string{"items[1]", "id"},
			wantFound: true,
		},
		{
			name:      "traverse with invalid array index",
			value:     []interface{}{"a", "b"},
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
	ctx.WorkflowVars["array"] = []interface{}{"a", "b"}

	resolver := NewResolver(ctx, DefaultOptions())

	// Test array index out of bounds
	_, found := resolver.resolveEnvPath("array[10]")
	if found {
		t.Error("resolveEnvPath() should not find array[10], got found=true")
	}
}

func TestResolver_ResolveInputPath_ArrayIndexErrors(t *testing.T) {
	ctx := NewVariableContext()
	ctx.InputVars["array"] = []interface{}{"a", "b"}

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
