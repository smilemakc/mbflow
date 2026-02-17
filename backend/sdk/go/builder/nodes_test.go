package builder_test

import (
	"testing"

	"github.com/smilemakc/mbflow/sdk/go/builder"
)

func TestAddHTTPNode(t *testing.T) {
	wf, err := builder.NewWorkflow("HTTP Test").
		AddHTTPNode("fetch", "Fetch Data",
			builder.URL("https://api.example.com/users"),
			builder.Method("GET"),
			builder.Header("Authorization", "Bearer token"),
		).
		Build()

	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	node := wf.Nodes[0]
	if node.Type != "http" {
		t.Errorf("Type = %q, want http", node.Type)
	}
	if node.Config["url"] != "https://api.example.com/users" {
		t.Errorf("url = %v", node.Config["url"])
	}
	if node.Config["method"] != "GET" {
		t.Errorf("method = %v", node.Config["method"])
	}
	headers := node.Config["headers"].(map[string]string)
	if headers["Authorization"] != "Bearer token" {
		t.Errorf("Authorization header = %v", headers["Authorization"])
	}
}

func TestAddLLMNode(t *testing.T) {
	wf, err := builder.NewWorkflow("LLM Test").
		AddLLMNode("gen", "Generate",
			builder.Provider("openai"),
			builder.Model("gpt-4"),
			builder.Prompt("Write about {{input.topic}}"),
			builder.APIKey("sk-test"),
			builder.Temperature(0.7),
			builder.MaxTokens(1000),
		).
		Build()

	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	node := wf.Nodes[0]
	if node.Type != "llm" {
		t.Errorf("Type = %q", node.Type)
	}
	if node.Config["provider"] != "openai" {
		t.Errorf("provider = %v", node.Config["provider"])
	}
	if node.Config["temperature"] != 0.7 {
		t.Errorf("temperature = %v", node.Config["temperature"])
	}
}

func TestAddTransformNode(t *testing.T) {
	wf, err := builder.NewWorkflow("Transform Test").
		AddTransformNode("tx", "Transform",
			builder.TransformType("jq"),
			builder.TransformExpression(".data | map(.name)"),
		).
		Build()

	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	if wf.Nodes[0].Config["type"] != "jq" {
		t.Errorf("type = %v", wf.Nodes[0].Config["type"])
	}
}

func TestAddConditionalNode(t *testing.T) {
	wf, err := builder.NewWorkflow("Cond Test").
		AddConditionalNode("check", "Check",
			builder.Expression("len(input.items) > 0"),
		).
		Build()

	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	if wf.Nodes[0].Type != "conditional" {
		t.Errorf("Type = %q", wf.Nodes[0].Type)
	}
}

func TestAddSubWorkflowNode(t *testing.T) {
	wf, err := builder.NewWorkflow("SubWF Test").
		AddSubWorkflowNode("sub", "Process Items",
			builder.WorkflowID("child-wf-1"),
			builder.ForEach("input.items"),
			builder.ItemVar("item"),
			builder.MaxParallelism(5),
		).
		Build()

	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	node := wf.Nodes[0]
	if node.Type != "sub_workflow" {
		t.Errorf("Type = %q", node.Type)
	}
	if node.Config["workflow_id"] != "child-wf-1" {
		t.Errorf("workflow_id = %v", node.Config["workflow_id"])
	}
	if node.Config["for_each"] != "input.items" {
		t.Errorf("for_each = %v", node.Config["for_each"])
	}
}
