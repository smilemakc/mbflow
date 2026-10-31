package builtin

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/smilemakc/mbflow/go/pkg/models"
)

type rcWorkflowExecutor struct {
	result     any
	err        error
	gotID      string
	gotInput   map[string]any
	callsCount int
}

func (w *rcWorkflowExecutor) ExecuteWorkflow(_ context.Context, workflowID string, input map[string]any) (any, error) {
	w.callsCount++
	w.gotID = workflowID
	w.gotInput = input
	return w.result, w.err
}

type rcCodeExecutor struct {
	gotLanguage, gotCode string
	gotArgs              map[string]any
}

func (c *rcCodeExecutor) ExecuteCode(_ context.Context, language, code string, args map[string]any) (any, error) {
	c.gotLanguage, c.gotCode, c.gotArgs = language, code, args
	return "code-result", nil
}

func TestRC_ToolCallingRegistry_SubWorkflowInputMapping(t *testing.T) {
	r := NewToolCallingRegistry(nil)
	def := &models.FunctionDefinition{
		Type:         models.FunctionTypeSubWorkflow,
		Name:         "summarize",
		WorkflowID:   "wf-1",
		InputMapping: map[string]string{"text": "input_text", "missing": "unused"},
	}

	_, err := r.ExecuteFunction(context.Background(), def, `{"text":"hi"}`)
	require.ErrorContains(t, err, "workflow executor not configured")

	wf := &rcWorkflowExecutor{result: map[string]any{"summary": "short"}}
	r.SetWorkflowExecutor(wf)
	out, err := r.ExecuteFunction(context.Background(), def, `{"text":"hi","extra":1}`)
	require.NoError(t, err)
	assert.Equal(t, map[string]any{"summary": "short"}, out, "without an extractor the result is returned unchanged")
	assert.Equal(t, "wf-1", wf.gotID)
	assert.Equal(t, map[string]any{"input_text": "hi"}, wf.gotInput)

	wf.err = errors.New("crashed")
	_, err = r.ExecuteFunction(context.Background(), def, `{}`)
	require.ErrorContains(t, err, "sub-workflow execution failed: crashed")
}

func TestRC_ToolCallingRegistry_SubWorkflowOutputExtractor(t *testing.T) {
	type summary struct {
		Title string   `json:"title"`
		Tags  []string `json:"tags"`
		Score int      `json:"score"`
	}
	cases := []struct {
		name      string
		result    any
		extractor string
		want      any
		wantErr   string
	}{
		{name: "field of a map", result: map[string]any{"output": map[string]any{"text": "done"}},
			extractor: ".output.text", want: "done"},
		{name: "struct is normalized through JSON", result: summary{Title: "T", Tags: []string{"a", "b"}, Score: 7},
			extractor: "{title, n: (.tags | length), score}", want: map[string]any{"title": "T", "n": 2, "score": float64(7)}},
		{name: "multiple outputs become an array", result: map[string]any{"items": []any{1, 2, 3}},
			extractor: ".items[] | select(. > 1)", want: []any{float64(2), float64(3)}},
		{name: "no output yields nil", result: map[string]any{"items": []any{}},
			extractor: ".items[]", want: nil},
		{name: "nil result", result: nil, extractor: ".", want: nil},
		{name: "invalid expression", result: map[string]any{}, extractor: ".[", wantErr: "invalid jq expression"},
		{name: "compile error", result: map[string]any{}, extractor: "undefined_fn(1)", wantErr: "failed to compile jq expression"},
		{name: "runtime error", result: map[string]any{"a": "x"}, extractor: ".a | tonumber", wantErr: "jq evaluation error"},
		{name: "unserializable result", result: map[string]any{"c": make(chan int)}, extractor: ".", wantErr: "failed to serialize sub-workflow result"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r := NewToolCallingRegistry(nil)
			r.SetWorkflowExecutor(&rcWorkflowExecutor{result: tc.result})
			out, err := r.ExecuteFunction(context.Background(), &models.FunctionDefinition{
				Type: models.FunctionTypeSubWorkflow, Name: "f", WorkflowID: "wf", OutputExtractor: tc.extractor,
			}, "")
			if tc.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "output extractor failed")
				assert.Contains(t, err.Error(), tc.wantErr)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tc.want, out)
		})
	}
}

func TestRC_ToolCallingRegistry_ExtractorHonoursContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := applyJQExtractor(ctx, map[string]any{}, "def f: f; f")
	require.Error(t, err)
}

func TestRC_ToolCallingRegistry_CustomCode(t *testing.T) {
	r := NewToolCallingRegistry(nil)
	def := &models.FunctionDefinition{Type: models.FunctionTypeCustomCode, Name: "c", Language: "python", Code: "return 1"}
	_, err := r.ExecuteFunction(context.Background(), def, `{}`)
	require.ErrorContains(t, err, "code executor not configured")

	ce := &rcCodeExecutor{}
	r.SetCodeExecutor(ce)
	out, err := r.ExecuteFunction(context.Background(), def, `{"x":1}`)
	require.NoError(t, err)
	assert.Equal(t, "code-result", out)
	assert.Equal(t, "python", ce.gotLanguage)
	assert.Equal(t, "return 1", ce.gotCode)
	assert.Equal(t, map[string]any{"x": float64(1)}, ce.gotArgs)
}

func TestRC_ToolCallingRegistry_BuiltinAndErrors(t *testing.T) {
	r := NewToolCallingRegistry(nil)
	_, err := r.ExecuteFunction(context.Background(), &models.FunctionDefinition{Type: models.FunctionTypeBuiltin, BuiltinName: "x"}, "")
	require.ErrorContains(t, err, "builtin registry not configured")

	r = NewToolCallingRegistry(models.NewFunctionRegistry())
	_, err = r.ExecuteFunction(context.Background(), &models.FunctionDefinition{Type: models.FunctionTypeBuiltin, BuiltinName: "x"}, "")
	require.ErrorContains(t, err, "builtin function not found: x")

	_, err = r.ExecuteFunction(context.Background(), &models.FunctionDefinition{Type: "mystery"}, "")
	require.ErrorContains(t, err, "unknown function type: mystery")

	_, err = r.ExecuteFunction(context.Background(), &models.FunctionDefinition{Type: models.FunctionTypeBuiltin}, "{bad")
	require.ErrorContains(t, err, "failed to parse arguments")

	_, err = r.ExecuteFunction(context.Background(), &models.FunctionDefinition{Type: models.FunctionTypeBuiltin}, `{} {}`)
	require.ErrorContains(t, err, "unexpected data after the JSON object")

	_, err = r.ExecuteFunction(context.Background(), &models.FunctionDefinition{Type: models.FunctionTypeOpenAPI}, `{"n": 12345678901234567890}`)
	require.ErrorContains(t, err, "OpenAPI executor not configured")

	_, err = r.APITools(context.Background(), nil)
	require.ErrorContains(t, err, "OpenAPI executor not configured")
}

func TestRC_ToolCallingRegistry_ValidateFunctionDefinition(t *testing.T) {
	fr := models.NewFunctionRegistry()
	fr.Register("known", func(map[string]any) (any, error) { return nil, nil })
	r := NewToolCallingRegistry(fr)

	valid := []models.FunctionDefinition{
		{Name: "a", Type: models.FunctionTypeBuiltin, BuiltinName: "known"},
		{Name: "b", Type: models.FunctionTypeSubWorkflow, WorkflowID: "wf"},
		{Name: "b2", Type: models.FunctionTypeSubWorkflow, WorkflowID: "wf", OutputExtractor: ".result"},
		{Name: "c", Type: models.FunctionTypeCustomCode, Language: "javascript", Code: "1"},
		{Name: "d", Type: models.FunctionTypeOpenAPI, ClientID: "c", OperationID: "o",
			Parameters: map[string]any{"type": "object"}},
	}
	for _, def := range valid {
		def := def
		assert.NoError(t, r.ValidateFunctionDefinition(&def), def.Name)
	}

	invalid := map[string]struct {
		def  models.FunctionDefinition
		want string
	}{
		"no name":            {models.FunctionDefinition{}, "function name is required"},
		"builtin no name":    {models.FunctionDefinition{Name: "x", Type: models.FunctionTypeBuiltin}, "builtin_name is required"},
		"builtin unknown":    {models.FunctionDefinition{Name: "x", Type: models.FunctionTypeBuiltin, BuiltinName: "nope"}, "builtin function not found"},
		"workflow no id":     {models.FunctionDefinition{Name: "x", Type: models.FunctionTypeSubWorkflow}, "workflow_id is required"},
		"workflow bad jq":    {models.FunctionDefinition{Name: "x", Type: models.FunctionTypeSubWorkflow, WorkflowID: "w", OutputExtractor: ".["}, "invalid output_extractor"},
		"code missing":       {models.FunctionDefinition{Name: "x", Type: models.FunctionTypeCustomCode}, "language and code are required"},
		"code bad language":  {models.FunctionDefinition{Name: "x", Type: models.FunctionTypeCustomCode, Language: "ruby", Code: "1"}, "unsupported language: ruby"},
		"openapi incomplete": {models.FunctionDefinition{Name: "x", Type: models.FunctionTypeOpenAPI, ClientID: "c"}, "client_id and operation_id are required"},
		"unknown type":       {models.FunctionDefinition{Name: "x", Type: "other"}, "unknown function type"},
		"bad parameters": {models.FunctionDefinition{Name: "x", Type: models.FunctionTypeSubWorkflow, WorkflowID: "w",
			Parameters: map[string]any{"c": make(chan int)}}, "invalid parameters schema"},
	}
	for name, tc := range invalid {
		t.Run(name, func(t *testing.T) {
			err := r.ValidateFunctionDefinition(&tc.def)
			require.Error(t, err)
			assert.Contains(t, err.Error(), tc.want)
		})
	}

	// Without a builtin registry the builtin name is not checked.
	assert.NoError(t, NewToolCallingRegistry(nil).ValidateFunctionDefinition(
		&models.FunctionDefinition{Name: "x", Type: models.FunctionTypeBuiltin, BuiltinName: "anything"}))
}

func TestRC_LLMExecutor_AutoToolCallingWithSubWorkflowExtractor(t *testing.T) {
	registry := NewToolCallingRegistry(models.NewFunctionRegistry())
	wf := &rcWorkflowExecutor{result: map[string]any{"report": map[string]any{"total": 42}, "debug": "noise"}}
	registry.SetWorkflowExecutor(wf)

	exec := NewLLMExecutor()
	exec.SetToolCallingRegistry(registry)
	exec.RegisterProvider(models.LLMProviderOpenAI, &rcScriptedProvider{responses: []*models.LLMResponse{
		toolCallResponse("c1", "report", `{"period":"q1"}`),
		{Content: "Total is 42", FinishReason: "stop"},
	}})

	out, err := exec.Execute(context.Background(), map[string]any{
		"provider": "openai", "model": "m", "prompt": "report",
		"tool_call_config": map[string]any{"mode": "auto"},
		"functions": []any{map[string]any{
			"type": "sub_workflow", "name": "report", "workflow_id": "wf-report",
			"input_mapping":    map[string]any{"period": "period"},
			"output_extractor": ".report.total",
		}},
	}, nil)
	require.NoError(t, err)
	result := out.(map[string]any)
	assert.Equal(t, "Total is 42", result["content"])
	assert.Equal(t, map[string]any{"period": "q1"}, wf.gotInput)
	te := result["tool_executions"].([]any)[0].(map[string]any)
	assert.Equal(t, float64(42), te["result"])
	msgs := result["messages"].([]any)
	assert.Equal(t, "42", msgs[2].(map[string]any)["content"], "the model sees only the extracted value")
}
