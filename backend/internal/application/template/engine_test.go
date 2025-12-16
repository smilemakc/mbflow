package template

import (
	"errors"
	"testing"
)

func TestEngine_ResolveString_SimpleSubstitution(t *testing.T) {
	ctx := NewVariableContext()
	ctx.WorkflowVars["name"] = "World"
	ctx.InputVars["greeting"] = "Hello"

	engine := NewEngineWithDefaults(ctx)

	tests := []struct {
		name     string
		template string
		want     string
		wantErr  bool
	}{
		{
			name:     "env variable",
			template: "Hello {{env.name}}",
			want:     "Hello World",
			wantErr:  false,
		},
		{
			name:     "input variable",
			template: "{{input.greeting}} there",
			want:     "Hello there",
			wantErr:  false,
		},
		{
			name:     "multiple variables",
			template: "{{input.greeting}} {{env.name}}!",
			want:     "Hello World!",
			wantErr:  false,
		},
		{
			name:     "no templates",
			template: "Plain text",
			want:     "Plain text",
			wantErr:  false,
		},
		{
			name:     "empty string",
			template: "",
			want:     "",
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := engine.ResolveString(tt.template)
			if (err != nil) != tt.wantErr {
				t.Errorf("ResolveString() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("ResolveString() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestEngine_ResolveString_NestedPaths(t *testing.T) {
	ctx := NewVariableContext()
	ctx.InputVars["user"] = map[string]interface{}{
		"name": "John",
		"profile": map[string]interface{}{
			"email": "john@example.com",
			"age":   30,
		},
	}

	engine := NewEngineWithDefaults(ctx)

	tests := []struct {
		name     string
		template string
		want     string
		wantErr  bool
	}{
		{
			name:     "nested field",
			template: "Email: {{input.user.profile.email}}",
			want:     "Email: john@example.com",
			wantErr:  false,
		},
		{
			name:     "root field",
			template: "Name: {{input.user.name}}",
			want:     "Name: John",
			wantErr:  false,
		},
		{
			name:     "number field",
			template: "Age: {{input.user.profile.age}}",
			want:     "Age: 30",
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := engine.ResolveString(tt.template)
			if (err != nil) != tt.wantErr {
				t.Errorf("ResolveString() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("ResolveString() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestEngine_ResolveString_ArrayAccess(t *testing.T) {
	ctx := NewVariableContext()
	ctx.InputVars["items"] = []interface{}{
		map[string]interface{}{"name": "Item1", "id": 1},
		map[string]interface{}{"name": "Item2", "id": 2},
	}
	ctx.InputVars["numbers"] = []interface{}{10, 20, 30}

	engine := NewEngineWithDefaults(ctx)

	tests := []struct {
		name     string
		template string
		want     string
		wantErr  bool
	}{
		{
			name:     "array element field",
			template: "Name: {{input.items[0].name}}",
			want:     "Name: Item1",
			wantErr:  false,
		},
		{
			name:     "array element number",
			template: "ID: {{input.items[1].id}}",
			want:     "ID: 2",
			wantErr:  false,
		},
		{
			name:     "array element primitive",
			template: "Number: {{input.numbers[1]}}",
			want:     "Number: 20",
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := engine.ResolveString(tt.template)
			if (err != nil) != tt.wantErr {
				t.Errorf("ResolveString() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("ResolveString() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestEngine_VariablePrecedence(t *testing.T) {
	ctx := NewVariableContext()
	ctx.WorkflowVars["apiKey"] = "workflow-key"
	ctx.ExecutionVars["apiKey"] = "execution-key"

	engine := NewEngineWithDefaults(ctx)

	got, err := engine.ResolveString("Key: {{env.apiKey}}")
	if err != nil {
		t.Fatalf("ResolveString() error = %v", err)
	}

	want := "Key: execution-key"
	if got != want {
		t.Errorf("ResolveString() = %v, want %v (execution vars should override workflow vars)", got, want)
	}
}

func TestEngine_StrictMode_MissingVariable(t *testing.T) {
	ctx := NewVariableContext()
	ctx.WorkflowVars["existing"] = "value"

	strictEngine := NewEngine(ctx, TemplateOptions{StrictMode: true})
	nonStrictEngine := NewEngine(ctx, TemplateOptions{StrictMode: false})

	template := "Value: {{env.missing}}"

	// Strict mode should return error
	_, err := strictEngine.ResolveString(template)
	if err == nil {
		t.Error("StrictMode: expected error for missing variable, got nil")
	}
	if !errors.Is(err, ErrVariableNotFound) {
		t.Errorf("StrictMode: expected ErrVariableNotFound, got %v", err)
	}

	// Non-strict mode should handle gracefully
	got, err := nonStrictEngine.ResolveString(template)
	if err != nil {
		t.Errorf("NonStrictMode: unexpected error: %v", err)
	}
	if got != "Value: " {
		t.Errorf("NonStrictMode: got %v, want 'Value: ' (empty string replacement)", got)
	}
}

func TestEngine_PlaceholderOnMissing(t *testing.T) {
	ctx := NewVariableContext()
	ctx.WorkflowVars["existing"] = "value"

	engine := NewEngine(ctx, TemplateOptions{
		StrictMode:           false,
		PlaceholderOnMissing: true,
	})

	template := "Value: {{env.missing}}"
	got, err := engine.ResolveString(template)
	if err != nil {
		t.Fatalf("ResolveString() error = %v", err)
	}

	want := "Value: {{env.missing}}"
	if got != want {
		t.Errorf("ResolveString() = %v, want %v (placeholder should be kept)", got, want)
	}
}

func TestEngine_ResolveMap(t *testing.T) {
	ctx := NewVariableContext()
	ctx.WorkflowVars["apiUrl"] = "https://api.example.com"
	ctx.InputVars["userId"] = "123"

	engine := NewEngineWithDefaults(ctx)

	input := map[string]interface{}{
		"url":    "{{env.apiUrl}}/users/{{input.userId}}",
		"method": "GET",
		"nested": map[string]interface{}{
			"header": "Bearer {{env.apiUrl}}",
		},
	}

	result, err := engine.Resolve(input)
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}

	resultMap, ok := result.(map[string]interface{})
	if !ok {
		t.Fatal("Resolve() did not return map[string]interface{}")
	}

	if resultMap["url"] != "https://api.example.com/users/123" {
		t.Errorf("url = %v, want https://api.example.com/users/123", resultMap["url"])
	}

	if resultMap["method"] != "GET" {
		t.Errorf("method = %v, want GET", resultMap["method"])
	}

	nested, ok := resultMap["nested"].(map[string]interface{})
	if !ok {
		t.Fatal("nested is not a map")
	}

	if nested["header"] != "Bearer https://api.example.com" {
		t.Errorf("nested.header = %v, want 'Bearer https://api.example.com'", nested["header"])
	}
}

func TestEngine_ResolveSlice(t *testing.T) {
	ctx := NewVariableContext()
	ctx.WorkflowVars["prefix"] = "Item"

	engine := NewEngineWithDefaults(ctx)

	input := []interface{}{
		"{{env.prefix}} 1",
		"{{env.prefix}} 2",
		map[string]interface{}{
			"name": "{{env.prefix}} 3",
		},
	}

	result, err := engine.Resolve(input)
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}

	resultSlice, ok := result.([]interface{})
	if !ok {
		t.Fatal("Resolve() did not return []interface{}")
	}

	if resultSlice[0] != "Item 1" {
		t.Errorf("resultSlice[0] = %v, want 'Item 1'", resultSlice[0])
	}

	if resultSlice[1] != "Item 2" {
		t.Errorf("resultSlice[1] = %v, want 'Item 2'", resultSlice[1])
	}

	nestedMap, ok := resultSlice[2].(map[string]interface{})
	if !ok {
		t.Fatal("resultSlice[2] is not a map")
	}

	if nestedMap["name"] != "Item 3" {
		t.Errorf("nestedMap['name'] = %v, want 'Item 3'", nestedMap["name"])
	}
}

func TestEngine_ValueToString(t *testing.T) {
	ctx := NewVariableContext()
	ctx.InputVars["string"] = "text"
	ctx.InputVars["number"] = 42
	ctx.InputVars["float"] = 3.14
	ctx.InputVars["bool"] = true
	ctx.InputVars["object"] = map[string]interface{}{"key": "value"}

	engine := NewEngineWithDefaults(ctx)

	tests := []struct {
		name     string
		template string
		want     string
	}{
		{
			name:     "string",
			template: "{{input.string}}",
			want:     "text",
		},
		{
			name:     "number",
			template: "{{input.number}}",
			want:     "42",
		},
		{
			name:     "float",
			template: "{{input.float}}",
			want:     "3.14",
		},
		{
			name:     "bool",
			template: "{{input.bool}}",
			want:     "true",
		},
		{
			name:     "object",
			template: "{{input.object}}",
			want:     `{"key":"value"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := engine.ResolveString(tt.template)
			if err != nil {
				t.Fatalf("ResolveString() error = %v", err)
			}
			if got != tt.want {
				t.Errorf("ResolveString() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestHasTemplates(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  bool
	}{
		{
			name:  "has template",
			input: "Hello {{env.name}}",
			want:  true,
		},
		{
			name:  "no template",
			input: "Hello World",
			want:  false,
		},
		{
			name:  "multiple templates",
			input: "{{input.greeting}} {{env.name}}",
			want:  true,
		},
		{
			name:  "empty string",
			input: "",
			want:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := HasTemplates(tt.input)
			if got != tt.want {
				t.Errorf("HasTemplates() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestExtractVariables(t *testing.T) {
	tests := []struct {
		name     string
		template string
		want     []string
	}{
		{
			name:     "single variable",
			template: "Hello {{env.name}}",
			want:     []string{"env.name"},
		},
		{
			name:     "multiple variables",
			template: "{{input.greeting}} {{env.name}}!",
			want:     []string{"input.greeting", "env.name"},
		},
		{
			name:     "nested path",
			template: "{{input.user.profile.email}}",
			want:     []string{"input.user.profile.email"},
		},
		{
			name:     "no variables",
			template: "Plain text",
			want:     []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ExtractVariables(tt.template)
			if len(got) != len(tt.want) {
				t.Errorf("ExtractVariables() length = %v, want %v", len(got), len(tt.want))
				return
			}
			for i, v := range got {
				if v != tt.want[i] {
					t.Errorf("ExtractVariables()[%d] = %v, want %v", i, v, tt.want[i])
				}
			}
		})
	}
}

func TestValidateTemplate(t *testing.T) {
	tests := []struct {
		name     string
		template string
		wantErr  bool
	}{
		{
			name:     "valid env template",
			template: "{{env.varName}}",
			wantErr:  false,
		},
		{
			name:     "valid input template",
			template: "{{input.field.path}}",
			wantErr:  false,
		},
		{
			name:     "invalid type",
			template: "{{unknown.field}}",
			wantErr:  true,
		},
		{
			name:     "missing path",
			template: "{{env}}",
			wantErr:  true,
		},
		{
			name:     "empty path",
			template: "{{env.}}",
			wantErr:  true,
		},
		{
			name:     "no templates",
			template: "Plain text",
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateTemplate(tt.template)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateTemplate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestEngine_ComplexScenario(t *testing.T) {
	// This test simulates a real workflow scenario with nested data
	ctx := NewVariableContext()

	// Workflow variables
	ctx.WorkflowVars["apiUrl"] = "https://api.example.com"
	ctx.WorkflowVars["apiKey"] = "workflow-key"

	// Execution variables (override workflow)
	ctx.ExecutionVars["apiKey"] = "execution-key-123"

	// Input from previous node
	ctx.InputVars["response"] = map[string]interface{}{
		"status": 200,
		"data": map[string]interface{}{
			"users": []interface{}{
				map[string]interface{}{
					"id":    1,
					"name":  "Alice",
					"email": "alice@example.com",
				},
				map[string]interface{}{
					"id":    2,
					"name":  "Bob",
					"email": "bob@example.com",
				},
			},
		},
	}

	engine := NewEngineWithDefaults(ctx)

	// Complex configuration that would be used in an HTTP node
	config := map[string]interface{}{
		"url":    "{{env.apiUrl}}/users/{{input.response.data.users[0].id}}",
		"method": "GET",
		"headers": map[string]interface{}{
			"Authorization": "Bearer {{env.apiKey}}",
			"Content-Type":  "application/json",
		},
		"body": map[string]interface{}{
			"email":   "{{input.response.data.users[1].email}}",
			"message": "Hello {{input.response.data.users[0].name}}!",
		},
	}

	result, err := engine.Resolve(config)
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}

	resultMap := result.(map[string]interface{})

	// Verify URL resolution
	expectedURL := "https://api.example.com/users/1"
	if resultMap["url"] != expectedURL {
		t.Errorf("url = %v, want %v", resultMap["url"], expectedURL)
	}

	// Verify headers resolution (execution var overrides workflow var)
	headers := resultMap["headers"].(map[string]interface{})
	expectedAuth := "Bearer execution-key-123"
	if headers["Authorization"] != expectedAuth {
		t.Errorf("Authorization = %v, want %v", headers["Authorization"], expectedAuth)
	}

	// Verify body resolution
	body := resultMap["body"].(map[string]interface{})
	if body["email"] != "bob@example.com" {
		t.Errorf("body.email = %v, want bob@example.com", body["email"])
	}
	if body["message"] != "Hello Alice!" {
		t.Errorf("body.message = %v, want 'Hello Alice!'", body["message"])
	}
}

// TestEngine_Resolve_NilValue tests that Resolve handles nil correctly
func TestEngine_Resolve_NilValue(t *testing.T) {
	ctx := NewVariableContext()
	engine := NewEngineWithDefaults(ctx)

	result, err := engine.Resolve(nil)
	if err != nil {
		t.Errorf("Resolve(nil) error = %v, want nil", err)
	}
	if result != nil {
		t.Errorf("Resolve(nil) = %v, want nil", result)
	}
}

// TestEngine_ResolveConfig tests the ResolveConfig convenience method
func TestEngine_ResolveConfig(t *testing.T) {
	ctx := NewVariableContext()
	ctx.WorkflowVars["apiUrl"] = "https://api.example.com"
	ctx.InputVars["userId"] = "123"

	engine := NewEngineWithDefaults(ctx)

	tests := []struct {
		name    string
		config  map[string]interface{}
		want    map[string]interface{}
		wantErr bool
	}{
		{
			name: "simple config",
			config: map[string]interface{}{
				"url":    "{{env.apiUrl}}/users/{{input.userId}}",
				"method": "GET",
			},
			want: map[string]interface{}{
				"url":    "https://api.example.com/users/123",
				"method": "GET",
			},
			wantErr: false,
		},
		{
			name: "nested config",
			config: map[string]interface{}{
				"request": map[string]interface{}{
					"url": "{{env.apiUrl}}",
					"headers": map[string]interface{}{
						"Authorization": "Bearer token",
					},
				},
			},
			want: map[string]interface{}{
				"request": map[string]interface{}{
					"url": "https://api.example.com",
					"headers": map[string]interface{}{
						"Authorization": "Bearer token",
					},
				},
			},
			wantErr: false,
		},
		{
			name:    "empty config",
			config:  map[string]interface{}{},
			want:    map[string]interface{}{},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := engine.ResolveConfig(tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("ResolveConfig() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				// Deep comparison would be better, but simple check for now
				if len(got) != len(tt.want) {
					t.Errorf("ResolveConfig() returned map with %d keys, want %d", len(got), len(tt.want))
				}
			}
		})
	}
}

// TestEngine_ResolveConfig_Error tests error handling in ResolveConfig
func TestEngine_ResolveConfig_Error(t *testing.T) {
	ctx := NewVariableContext()
	engine := NewEngine(ctx, TemplateOptions{StrictMode: true})

	config := map[string]interface{}{
		"url": "{{env.missing}}",
	}

	_, err := engine.ResolveConfig(config)
	if err == nil {
		t.Error("ResolveConfig() expected error for missing variable in strict mode, got nil")
	}
}

// TestEngine_ResolveComplex tests resolveComplex with various types
func TestEngine_ResolveComplex(t *testing.T) {
	ctx := NewVariableContext()
	ctx.WorkflowVars["value"] = "test"
	engine := NewEngineWithDefaults(ctx)

	tests := []struct {
		name    string
		input   interface{}
		wantErr bool
	}{
		{
			name:    "bool",
			input:   true,
			wantErr: false,
		},
		{
			name:    "int",
			input:   42,
			wantErr: false,
		},
		{
			name:    "int8",
			input:   int8(8),
			wantErr: false,
		},
		{
			name:    "int16",
			input:   int16(16),
			wantErr: false,
		},
		{
			name:    "int32",
			input:   int32(32),
			wantErr: false,
		},
		{
			name:    "int64",
			input:   int64(64),
			wantErr: false,
		},
		{
			name:    "uint",
			input:   uint(42),
			wantErr: false,
		},
		{
			name:    "uint8",
			input:   uint8(8),
			wantErr: false,
		},
		{
			name:    "uint16",
			input:   uint16(16),
			wantErr: false,
		},
		{
			name:    "uint32",
			input:   uint32(32),
			wantErr: false,
		},
		{
			name:    "uint64",
			input:   uint64(64),
			wantErr: false,
		},
		{
			name:    "float32",
			input:   float32(3.14),
			wantErr: false,
		},
		{
			name:    "float64",
			input:   float64(3.14159),
			wantErr: false,
		},
		{
			name:    "complex64",
			input:   complex64(1 + 2i),
			wantErr: false,
		},
		{
			name:    "complex128",
			input:   complex128(1 + 2i),
			wantErr: false,
		},
		{
			name: "struct with templates",
			input: struct {
				Name  string
				Value string
			}{
				Name:  "test",
				Value: "{{env.value}}",
			},
			wantErr: false,
		},
		{
			name: "struct without templates",
			input: struct {
				Name  string
				Count int
			}{
				Name:  "test",
				Count: 42,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := engine.Resolve(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("Resolve() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if result == nil && tt.input != nil {
				t.Error("Resolve() returned nil for non-nil input")
			}
		})
	}
}

// TestEngine_ResolveString_StrictMode_InvalidVariableRef tests strict mode error for invalid variable reference
func TestEngine_ResolveString_StrictMode_InvalidVariableRef(t *testing.T) {
	ctx := NewVariableContext()
	engine := NewEngine(ctx, TemplateOptions{StrictMode: true})

	// Test invalid variable reference (no dot separator)
	template := "Value: {{invalid}}"
	_, err := engine.ResolveString(template)
	if err == nil {
		t.Error("ResolveString() expected error for invalid variable reference in strict mode, got nil")
	}
	if !errors.Is(err, ErrInvalidTemplate) {
		t.Errorf("ResolveString() expected ErrInvalidTemplate, got %v", err)
	}
}

// TestEngine_ResolveString_NonStrictMode_InvalidVariableRef tests non-strict mode with invalid variable reference
func TestEngine_ResolveString_NonStrictMode_InvalidVariableRef(t *testing.T) {
	ctx := NewVariableContext()

	tests := []struct {
		name     string
		opts     TemplateOptions
		template string
		want     string
	}{
		{
			name: "invalid ref with empty replacement",
			opts: TemplateOptions{
				StrictMode:           false,
				PlaceholderOnMissing: false,
			},
			template: "Value: {{invalid}}",
			want:     "Value: ",
		},
		{
			name: "invalid ref with placeholder",
			opts: TemplateOptions{
				StrictMode:           false,
				PlaceholderOnMissing: true,
			},
			template: "Value: {{invalid}}",
			want:     "Value: {{invalid}}",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			engine := NewEngine(ctx, tt.opts)
			got, err := engine.ResolveString(tt.template)
			if err != nil {
				t.Errorf("ResolveString() unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf("ResolveString() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestEngine_ResolveMap_Error tests error handling in resolveMap
func TestEngine_ResolveMap_Error(t *testing.T) {
	ctx := NewVariableContext()
	engine := NewEngine(ctx, TemplateOptions{StrictMode: true})

	m := map[string]interface{}{
		"valid":   "plain text",
		"invalid": "{{env.missing}}",
	}

	_, err := engine.Resolve(m)
	if err == nil {
		t.Error("Resolve() expected error for missing variable in strict mode, got nil")
	}
}

// TestEngine_ResolveSlice_Error tests error handling in resolveSlice
func TestEngine_ResolveSlice_Error(t *testing.T) {
	ctx := NewVariableContext()
	engine := NewEngine(ctx, TemplateOptions{StrictMode: true})

	slice := []interface{}{
		"valid text",
		"{{env.missing}}",
	}

	_, err := engine.Resolve(slice)
	if err == nil {
		t.Error("Resolve() expected error for missing variable in strict mode, got nil")
	}
}

// TestEngine_ValueToString_AllTypes tests valueToString with all type cases
func TestEngine_ValueToString_AllTypes(t *testing.T) {
	ctx := NewVariableContext()
	ctx.InputVars["nil"] = nil
	ctx.InputVars["string"] = "text"
	ctx.InputVars["bool"] = false
	ctx.InputVars["int"] = int(42)
	ctx.InputVars["int8"] = int8(8)
	ctx.InputVars["int16"] = int16(16)
	ctx.InputVars["int32"] = int32(32)
	ctx.InputVars["int64"] = int64(64)
	ctx.InputVars["uint"] = uint(42)
	ctx.InputVars["uint8"] = uint8(8)
	ctx.InputVars["uint16"] = uint16(16)
	ctx.InputVars["uint32"] = uint32(32)
	ctx.InputVars["uint64"] = uint64(64)
	ctx.InputVars["float32"] = float32(3.14)
	ctx.InputVars["float64"] = float64(3.14159)
	ctx.InputVars["slice"] = []interface{}{1, 2, 3}
	ctx.InputVars["map"] = map[string]interface{}{"key": "value"}

	// Type that can't be marshaled to JSON
	type unmarshalableType struct {
		Ch chan int
	}
	ctx.InputVars["unmarshalable"] = unmarshalableType{Ch: make(chan int)}

	engine := NewEngineWithDefaults(ctx)

	tests := []struct {
		name     string
		template string
		want     string
	}{
		{
			name:     "nil",
			template: "{{input.nil}}",
			want:     "",
		},
		{
			name:     "string",
			template: "{{input.string}}",
			want:     "text",
		},
		{
			name:     "bool false",
			template: "{{input.bool}}",
			want:     "false",
		},
		{
			name:     "int",
			template: "{{input.int}}",
			want:     "42",
		},
		{
			name:     "int8",
			template: "{{input.int8}}",
			want:     "8",
		},
		{
			name:     "int16",
			template: "{{input.int16}}",
			want:     "16",
		},
		{
			name:     "int32",
			template: "{{input.int32}}",
			want:     "32",
		},
		{
			name:     "int64",
			template: "{{input.int64}}",
			want:     "64",
		},
		{
			name:     "uint",
			template: "{{input.uint}}",
			want:     "42",
		},
		{
			name:     "uint8",
			template: "{{input.uint8}}",
			want:     "8",
		},
		{
			name:     "uint16",
			template: "{{input.uint16}}",
			want:     "16",
		},
		{
			name:     "uint32",
			template: "{{input.uint32}}",
			want:     "32",
		},
		{
			name:     "uint64",
			template: "{{input.uint64}}",
			want:     "64",
		},
		{
			name:     "float32",
			template: "{{input.float32}}",
			want:     "3.14",
		},
		{
			name:     "float64",
			template: "{{input.float64}}",
			want:     "3.14159",
		},
		{
			name:     "slice",
			template: "{{input.slice}}",
			want:     "[1,2,3]",
		},
		{
			name:     "map",
			template: "{{input.map}}",
			want:     `{"key":"value"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := engine.ResolveString(tt.template)
			if err != nil {
				t.Fatalf("ResolveString() error = %v", err)
			}
			if got != tt.want {
				t.Errorf("ResolveString() = %v, want %v", got, tt.want)
			}
		})
	}

	// Test unmarshalable type (should use fmt.Sprintf fallback)
	t.Run("unmarshalable type", func(t *testing.T) {
		got, err := engine.ResolveString("{{input.unmarshalable}}")
		if err != nil {
			t.Fatalf("ResolveString() error = %v", err)
		}
		// Should not be empty (fallback to fmt.Sprintf)
		if got == "" {
			t.Error("ResolveString() = empty string, expected fallback formatting")
		}
	})
}

// TestEngine_ParseVariableRef tests parseVariableRef edge cases
func TestEngine_ParseVariableRef(t *testing.T) {
	ctx := NewVariableContext()
	engine := NewEngineWithDefaults(ctx)

	tests := []struct {
		name        string
		ref         string
		wantVarType string
		wantPath    string
	}{
		{
			name:        "valid reference",
			ref:         "env.varName",
			wantVarType: "env",
			wantPath:    "varName",
		},
		{
			name:        "nested path",
			ref:         "input.user.profile.email",
			wantVarType: "input",
			wantPath:    "user.profile.email",
		},
		{
			name:        "no separator",
			ref:         "invalid",
			wantVarType: "",
			wantPath:    "",
		},
		{
			name:        "with spaces",
			ref:         " env . varName ",
			wantVarType: "env",
			wantPath:    "varName",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotVarType, gotPath := engine.parseVariableRef(tt.ref)
			if gotVarType != tt.wantVarType {
				t.Errorf("parseVariableRef() varType = %v, want %v", gotVarType, tt.wantVarType)
			}
			if gotPath != tt.wantPath {
				t.Errorf("parseVariableRef() path = %v, want %v", gotPath, tt.wantPath)
			}
		})
	}
}

// TestEngine_ResolveComplex_EdgeCases tests edge cases in resolveComplex
func TestEngine_ResolveComplex_EdgeCases(t *testing.T) {
	ctx := NewVariableContext()
	ctx.WorkflowVars["value"] = "test"
	engine := NewEngineWithDefaults(ctx)

	t.Run("unmarshalable type returns as-is", func(t *testing.T) {
		// Channel types cannot be marshaled to JSON
		type unmarshalableType struct {
			Ch chan int
		}
		input := unmarshalableType{Ch: make(chan int)}
		result, err := engine.Resolve(input)
		if err != nil {
			t.Errorf("Resolve() error = %v, want nil", err)
		}
		// Should return input as-is when marshal fails
		if result != input {
			t.Error("Resolve() should return input as-is when marshal fails")
		}
	})

	t.Run("function type cannot be marshaled", func(t *testing.T) {
		// Functions cannot be marshaled to JSON
		input := func() string { return "test" }

		result, err := engine.Resolve(input)
		if err != nil {
			t.Errorf("Resolve() error = %v, want nil", err)
		}

		// Should return input as-is when marshal fails
		// Note: we can't compare functions directly
		if result == nil {
			t.Error("Resolve() returned nil, expected input as-is")
		}
	})

	t.Run("struct that unmarshals to map with templates", func(t *testing.T) {
		// Struct that will be marshaled to JSON and back
		type structWithTemplates struct {
			Name   string                 `json:"name"`
			Value  string                 `json:"value"`
			Nested map[string]interface{} `json:"nested"`
		}
		input := structWithTemplates{
			Name:  "test",
			Value: "{{env.value}}",
			Nested: map[string]interface{}{
				"key": "{{env.value}}",
			},
		}
		result, err := engine.Resolve(input)
		if err != nil {
			t.Errorf("Resolve() error = %v, want nil", err)
		}

		// Should be resolved to a map
		resultMap, ok := result.(map[string]interface{})
		if !ok {
			t.Fatal("Resolve() should return map[string]interface{}")
		}

		// Check template was resolved
		if resultMap["value"] != "test" {
			t.Errorf("value = %v, want 'test'", resultMap["value"])
		}
	})

	t.Run("struct that unmarshals to slice", func(t *testing.T) {
		// Use a slice type that will unmarshal back to []interface{}
		type stringSlice []string
		input := stringSlice{"{{env.value}}", "plain"}

		result, err := engine.Resolve(input)
		if err != nil {
			t.Errorf("Resolve() error = %v, want nil", err)
		}

		// Should be a slice
		resultSlice, ok := result.([]interface{})
		if !ok {
			t.Fatalf("Resolve() should return []interface{}, got %T", result)
		}

		// Check template was resolved
		if resultSlice[0] != "test" {
			t.Errorf("resultSlice[0] = %v, want 'test'", resultSlice[0])
		}
	})

	t.Run("custom type that marshals to JSON string", func(t *testing.T) {
		// Test a custom type that implements json.Marshaler and returns a JSON string
		// This will cause the unmarshaled result to be a string
		type customString string

		input := customString("{{env.value}}")

		result, err := engine.Resolve(input)
		if err != nil {
			t.Errorf("Resolve() error = %v", err)
		}

		// Result should be a string with resolved template
		resultStr, ok := result.(string)
		if !ok {
			t.Fatalf("Resolve() should return string, got %T", result)
		}

		if resultStr != "test" {
			t.Errorf("result = %v, want 'test'", resultStr)
		}
	})

	t.Run("custom type that marshals to JSON number", func(t *testing.T) {
		// Test a custom type that marshals to a JSON number
		// After unmarshaling, this should be a float64 (default case)
		type customInt int

		input := customInt(42)

		result, err := engine.Resolve(input)
		if err != nil {
			t.Errorf("Resolve() error = %v, want nil", err)
		}

		// Should return the unmarshaled number (float64 after JSON round-trip)
		if result == nil {
			t.Error("Resolve() returned nil")
		}
	})

	t.Run("function type cannot be marshaled", func(t *testing.T) {
		// Functions cannot be marshaled to JSON
		input := func() {}

		result, err := engine.Resolve(input)
		if err != nil {
			t.Errorf("Resolve() error = %v, want nil", err)
		}

		// Should return input as-is when marshal fails
		// Note: we can't compare functions directly, so just check it's not nil
		if result == nil {
			t.Error("Resolve() returned nil, expected input as-is")
		}
	})
}

func TestEngine_ResolveString_ResourceVariables(t *testing.T) {
	ctx := NewVariableContext()
	ctx.ResourceVars["storage"] = map[string]interface{}{
		"id":   "res-123",
		"name": "My Storage",
		"type": "file_storage",
		"config": map[string]interface{}{
			"bucket": "my-bucket",
			"region": "us-east-1",
		},
	}
	ctx.ResourceVars["apiKey"] = map[string]interface{}{
		"id":    "key-456",
		"value": "secret-api-key",
	}

	engine := NewEngineWithDefaults(ctx)

	tests := []struct {
		name     string
		template string
		want     string
		wantErr  bool
	}{
		{
			name:     "resource id field",
			template: "Storage ID: {{resource.storage.id}}",
			want:     "Storage ID: res-123",
			wantErr:  false,
		},
		{
			name:     "resource name field",
			template: "Name: {{resource.storage.name}}",
			want:     "Name: My Storage",
			wantErr:  false,
		},
		{
			name:     "nested resource field",
			template: "Bucket: {{resource.storage.config.bucket}}",
			want:     "Bucket: my-bucket",
			wantErr:  false,
		},
		{
			name:     "multiple resource references",
			template: "{{resource.storage.type}} in {{resource.storage.config.region}}",
			want:     "file_storage in us-east-1",
			wantErr:  false,
		},
		{
			name:     "different resources",
			template: "API Key: {{resource.apiKey.value}}",
			want:     "API Key: secret-api-key",
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := engine.ResolveString(tt.template)
			if (err != nil) != tt.wantErr {
				t.Errorf("ResolveString() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("ResolveString() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestEngine_ResolveConfig_WithResources(t *testing.T) {
	ctx := NewVariableContext()
	ctx.WorkflowVars["apiUrl"] = "https://api.example.com"
	ctx.ResourceVars["storage"] = map[string]interface{}{
		"id":   "res-123",
		"type": "s3",
		"config": map[string]interface{}{
			"bucket":    "my-bucket",
			"accessKey": "AKIA...",
		},
	}
	ctx.ResourceVars["database"] = map[string]interface{}{
		"id":   "db-456",
		"host": "db.example.com",
		"port": 5432,
	}

	engine := NewEngineWithDefaults(ctx)

	tests := []struct {
		name    string
		config  map[string]interface{}
		want    map[string]interface{}
		wantErr bool
	}{
		{
			name: "resource in config",
			config: map[string]interface{}{
				"storageId":   "{{resource.storage.id}}",
				"storageType": "{{resource.storage.type}}",
			},
			want: map[string]interface{}{
				"storageId":   "res-123",
				"storageType": "s3",
			},
			wantErr: false,
		},
		{
			name: "nested resource config",
			config: map[string]interface{}{
				"s3": map[string]interface{}{
					"bucket": "{{resource.storage.config.bucket}}",
					"key":    "{{resource.storage.config.accessKey}}",
				},
			},
			want: map[string]interface{}{
				"s3": map[string]interface{}{
					"bucket": "my-bucket",
					"key":    "AKIA...",
				},
			},
			wantErr: false,
		},
		{
			name: "mixed variables and resources",
			config: map[string]interface{}{
				"url":      "{{env.apiUrl}}/data",
				"database": "{{resource.database.host}}:{{resource.database.port}}",
			},
			want: map[string]interface{}{
				"url":      "https://api.example.com/data",
				"database": "db.example.com:5432",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := engine.ResolveConfig(tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("ResolveConfig() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				for key, expectedVal := range tt.want {
					gotVal := result[key]
					if gotVal != expectedVal {
						// Check nested maps
						if expectedMap, ok := expectedVal.(map[string]interface{}); ok {
							gotMap, ok := gotVal.(map[string]interface{})
							if !ok {
								t.Errorf("ResolveConfig()[%s] is not a map", key)
								continue
							}
							for nestedKey, nestedExpected := range expectedMap {
								if gotMap[nestedKey] != nestedExpected {
									t.Errorf("ResolveConfig()[%s][%s] = %v, want %v", key, nestedKey, gotMap[nestedKey], nestedExpected)
								}
							}
						} else {
							t.Errorf("ResolveConfig()[%s] = %v, want %v", key, gotVal, expectedVal)
						}
					}
				}
			}
		})
	}
}

func TestValidateTemplate_WithResources(t *testing.T) {
	tests := []struct {
		name     string
		template string
		wantErr  bool
	}{
		{
			name:     "valid resource template",
			template: "{{resource.storage}}",
			wantErr:  false,
		},
		{
			name:     "valid resource with field",
			template: "{{resource.storage.id}}",
			wantErr:  false,
		},
		{
			name:     "valid resource nested field",
			template: "{{resource.storage.config.bucket}}",
			wantErr:  false,
		},
		{
			name:     "resource without path",
			template: "{{resource}}",
			wantErr:  true,
		},
		{
			name:     "resource with empty path",
			template: "{{resource.}}",
			wantErr:  true,
		},
		{
			name:     "mixed valid templates",
			template: "{{env.var}} {{input.field}} {{resource.storage.id}}",
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateTemplate(tt.template)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateTemplate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestEngine_ResourceWithArrays(t *testing.T) {
	ctx := NewVariableContext()
	ctx.ResourceVars["cluster"] = map[string]interface{}{
		"id": "cluster-001",
		"nodes": []interface{}{
			map[string]interface{}{
				"id":   "node-1",
				"host": "192.168.1.1",
				"port": 8080,
			},
			map[string]interface{}{
				"id":   "node-2",
				"host": "192.168.1.2",
				"port": 8081,
			},
		},
	}

	engine := NewEngineWithDefaults(ctx)

	tests := []struct {
		name     string
		template string
		want     string
		wantErr  bool
	}{
		{
			name:     "array element access",
			template: "{{resource.cluster.nodes[0].host}}",
			want:     "192.168.1.1",
			wantErr:  false,
		},
		{
			name:     "second array element",
			template: "{{resource.cluster.nodes[1].host}}:{{resource.cluster.nodes[1].port}}",
			want:     "192.168.1.2:8081",
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := engine.ResolveString(tt.template)
			if (err != nil) != tt.wantErr {
				t.Errorf("ResolveString() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("ResolveString() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestEngine_ComplexScenario_WithResources(t *testing.T) {
	// This test simulates a real workflow scenario with resources
	ctx := NewVariableContext()

	// Workflow variables
	ctx.WorkflowVars["apiUrl"] = "https://api.example.com"

	// Execution variables
	ctx.ExecutionVars["executionId"] = "exec-123"

	// Resources
	ctx.ResourceVars["storage"] = map[string]interface{}{
		"id":   "res-456",
		"type": "s3",
		"config": map[string]interface{}{
			"bucket":    "my-data-bucket",
			"region":    "us-west-2",
			"accessKey": "AKIA123456",
		},
	}
	ctx.ResourceVars["database"] = map[string]interface{}{
		"id":               "db-789",
		"connectionString": "postgresql://user:pass@db.example.com:5432/mydb",
		"credentials": map[string]interface{}{
			"username": "dbuser",
			"password": "dbpass",
		},
	}

	// Input from previous node
	ctx.InputVars["userId"] = "user-001"
	ctx.InputVars["fileName"] = "data.csv"

	engine := NewEngineWithDefaults(ctx)

	// Complex configuration using all variable types
	config := map[string]interface{}{
		"apiEndpoint": "{{env.apiUrl}}/upload",
		"executionId": "{{env.executionId}}",
		"storage": map[string]interface{}{
			"type":      "{{resource.storage.type}}",
			"bucket":    "{{resource.storage.config.bucket}}",
			"region":    "{{resource.storage.config.region}}",
			"accessKey": "{{resource.storage.config.accessKey}}",
		},
		"database": map[string]interface{}{
			"connection": "{{resource.database.connectionString}}",
			"user":       "{{resource.database.credentials.username}}",
		},
		"metadata": map[string]interface{}{
			"userId":   "{{input.userId}}",
			"fileName": "{{input.fileName}}",
			"bucket":   "{{resource.storage.config.bucket}}",
		},
	}

	result, err := engine.Resolve(config)
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}

	resultMap := result.(map[string]interface{})

	// Verify all substitutions
	if resultMap["apiEndpoint"] != "https://api.example.com/upload" {
		t.Errorf("apiEndpoint = %v, want https://api.example.com/upload", resultMap["apiEndpoint"])
	}
	if resultMap["executionId"] != "exec-123" {
		t.Errorf("executionId = %v, want exec-123", resultMap["executionId"])
	}

	storage := resultMap["storage"].(map[string]interface{})
	if storage["type"] != "s3" {
		t.Errorf("storage.type = %v, want s3", storage["type"])
	}
	if storage["bucket"] != "my-data-bucket" {
		t.Errorf("storage.bucket = %v, want my-data-bucket", storage["bucket"])
	}
	if storage["region"] != "us-west-2" {
		t.Errorf("storage.region = %v, want us-west-2", storage["region"])
	}

	database := resultMap["database"].(map[string]interface{})
	if database["connection"] != "postgresql://user:pass@db.example.com:5432/mydb" {
		t.Errorf("database.connection = %v", database["connection"])
	}
	if database["user"] != "dbuser" {
		t.Errorf("database.user = %v, want dbuser", database["user"])
	}

	metadata := resultMap["metadata"].(map[string]interface{})
	if metadata["userId"] != "user-001" {
		t.Errorf("metadata.userId = %v, want user-001", metadata["userId"])
	}
	if metadata["fileName"] != "data.csv" {
		t.Errorf("metadata.fileName = %v, want data.csv", metadata["fileName"])
	}
	if metadata["bucket"] != "my-data-bucket" {
		t.Errorf("metadata.bucket = %v, want my-data-bucket", metadata["bucket"])
	}
}
