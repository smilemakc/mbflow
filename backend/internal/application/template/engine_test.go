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
