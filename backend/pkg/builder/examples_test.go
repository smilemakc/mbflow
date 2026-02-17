package builder_test

import (
	"fmt"

	"github.com/smilemakc/mbflow/pkg/builder"
	"github.com/smilemakc/mbflow/pkg/models"
)

// Example of creating a simple HTTP workflow
func ExampleNewWorkflow_simpleHTTP() {
	workflow := builder.NewWorkflow("Fetch User Data",
		builder.WithDescription("Fetch user data from API"),
		builder.WithVariable("api_base", "https://api.example.com"),
	).AddNode(
		builder.NewHTTPGetNode(
			"fetch",
			"Fetch User",
			"{{env.api_base}}/users/{{input.user_id}}",
		),
	).MustBuild()

	fmt.Println(workflow.Name)
	fmt.Println(len(workflow.Nodes))
	fmt.Println(workflow.Nodes[0].Type)
	// Output:
	// Fetch User Data
	// 1
	// http
}

// Example of creating an LLM workflow with OpenAI
func ExampleNewOpenAINode() {
	workflow := builder.NewWorkflow("Code Analysis",
		builder.WithVariable("openai_api_key", "sk-..."),
	).AddNode(
		builder.NewOpenAINode(
			"analyze",
			"Analyze Code",
			"gpt-4",
			"Analyze this code: {{input.code}}",
			builder.LLMAPIKey("{{env.openai_api_key}}"),
			builder.LLMTemperature(0.2),
			builder.LLMMaxTokens(1000),
		),
	).MustBuild()

	fmt.Println(workflow.Name)
	fmt.Println(workflow.Nodes[0].Config["provider"])
	fmt.Println(workflow.Nodes[0].Config["model"])
	// Output:
	// Code Analysis
	// openai
	// gpt-4
}

// Example of creating a multi-node workflow with edges
func ExampleWorkflowBuilder_Connect() {
	workflow := builder.NewWorkflow("ETL Pipeline",
		builder.WithTags("etl", "data"),
	).AddNode(
		builder.NewHTTPGetNode("extract", "Extract Data", "https://api.example.com/data"),
	).AddNode(
		builder.NewJQNode("transform", "Transform", `.[] | {id, name}`),
	).AddNode(
		builder.NewHTTPPostNode("load", "Load Data", "https://warehouse.example.com/data", nil),
	).Connect("extract", "transform").
		Connect("transform", "load").
		MustBuild()

	fmt.Println(len(workflow.Nodes))
	fmt.Println(len(workflow.Edges))
	fmt.Println(workflow.Edges[0].From, "->", workflow.Edges[0].To)
	// Output:
	// 3
	// 2
	// extract -> transform
}

// Example of using conditional edges
func ExampleWhenTrue() {
	workflow := builder.NewWorkflow("Conditional Flow").
		AddNode(
			builder.NewHTTPGetNode("check", "Check Status", "https://api.example.com/status"),
		).AddNode(
		builder.NewHTTPPostNode("success", "Handle Success", "https://api.example.com/success", nil),
	).AddNode(
		builder.NewHTTPPostNode("failure", "Handle Failure", "https://api.example.com/failure", nil),
	).Connect("check", "success", builder.WhenTrue("output.success")).
		Connect("check", "failure", builder.WhenFalse("output.success")).
		MustBuild()

	fmt.Println(len(workflow.Edges))
	fmt.Println(workflow.Edges[0].Condition)
	// Output:
	// 2
	// output.success
}

// Example of using grid positioning
func ExampleGridPosition() {
	workflow := builder.NewWorkflow("Grid Layout").
		AddNode(
			builder.NewHTTPGetNode(
				"node1",
				"Node 1",
				"https://api.example.com",
				builder.GridPosition(0, 0),
			),
		).AddNode(
		builder.NewHTTPGetNode(
			"node2",
			"Node 2",
			"https://api.example.com",
			builder.GridPosition(0, 1),
		),
	).AddNode(
		builder.NewHTTPGetNode(
			"node3",
			"Node 3",
			"https://api.example.com",
			builder.GridPosition(1, 0),
		),
	).MustBuild()

	fmt.Printf("Node1: (%.0f, %.0f)\n", workflow.Nodes[0].Position.X, workflow.Nodes[0].Position.Y)
	fmt.Printf("Node2: (%.0f, %.0f)\n", workflow.Nodes[1].Position.X, workflow.Nodes[1].Position.Y)
	// Output:
	// Node1: (0, 0)
	// Node2: (200, 0)
}

// Example of using auto layout
func ExampleWithAutoLayout() {
	workflow := builder.NewWorkflow("Auto Layout", builder.WithAutoLayout()).
		AddNode(builder.NewHTTPGetNode("fetch", "Fetch", "https://api.example.com")).
		AddNode(builder.NewPassthroughNode("process", "Process")).
		AddNode(builder.NewHTTPPostNode("send", "Send", "https://api.example.com", nil)).
		MustBuild()

	fmt.Printf("Node 1: X=%.0f\n", workflow.Nodes[0].Position.X)
	fmt.Printf("Node 2: X=%.0f\n", workflow.Nodes[1].Position.X)
	fmt.Printf("Node 3: X=%.0f\n", workflow.Nodes[2].Position.X)
	// Output:
	// Node 1: X=0
	// Node 2: X=200
	// Node 3: X=400
}

// Example of different transform types
func ExampleNewExpressionNode() {
	workflow := builder.NewWorkflow("Transforms").
		AddNode(
			builder.NewExpressionNode(
				"filter",
				"Filter Active",
				`filter(input, {.status == "active"})`,
			),
		).MustBuild()

	fmt.Println(workflow.Nodes[0].Config["type"])
	fmt.Println(workflow.Nodes[0].Config["expression"])
	// Output:
	// expression
	// filter(input, {.status == "active"})
}

// Example of LLM with Anthropic
func ExampleNewAnthropicNode() {
	workflow := builder.NewWorkflow("Anthropic Workflow").
		AddNode(
			builder.NewAnthropicNode(
				"analyze",
				"Analyze Text",
				"claude-3-5-sonnet-20241022",
				"Analyze: {{input.text}}",
				builder.LLMAPIKey("sk-ant-..."),
				builder.LLMTemperature(0.5),
				builder.LLMMaxTokens(2000),
			),
		).MustBuild()

	fmt.Println(workflow.Nodes[0].Config["provider"])
	fmt.Println(workflow.Nodes[0].Config["model"])
	// Output:
	// anthropic
	// claude-3-5-sonnet-20241022
}

// Example of using workflow status
func ExampleWithStatus() {
	workflow := builder.NewWorkflow("Production Workflow",
		builder.WithStatus(models.WorkflowStatusActive),
	).AddNode(
		builder.NewHTTPGetNode("fetch", "Fetch", "https://api.example.com"),
	).MustBuild()

	fmt.Println(workflow.Status)
	// Output:
	// active
}

// Example of using metadata
func ExampleWithMetadata() {
	workflow := builder.NewWorkflow("Documented Workflow",
		builder.WithMetadata("author", "John Doe"),
		builder.WithMetadata("version", "1.0.0"),
	).AddNode(
		builder.NewHTTPGetNode("fetch", "Fetch", "https://api.example.com"),
	).MustBuild()

	fmt.Println(workflow.Metadata["author"])
	fmt.Println(workflow.Metadata["version"])
	// Output:
	// John Doe
	// 1.0.0
}

// Example of HTTP methods
func ExampleNewHTTPPostNode() {
	workflow := builder.NewWorkflow("POST Example").
		AddNode(
			builder.NewHTTPPostNode(
				"create",
				"Create User",
				"https://api.example.com/users",
				map[string]any{
					"name":  "{{input.name}}",
					"email": "{{input.email}}",
				},
				builder.HTTPHeader("Authorization", "Bearer {{env.token}}"),
			),
		).MustBuild()

	fmt.Println(workflow.Nodes[0].Config["method"])
	// Output:
	// POST
}

// Example of complex workflow with multiple node types
func ExampleNewWorkflow_complex() {
	workflow := builder.NewWorkflow("Complex Pipeline",
		builder.WithDescription("A complex multi-step workflow"),
		builder.WithTags("production", "critical"),
		builder.WithVariable("api_key", "secret"),
		builder.WithAutoLayout(),
	).AddNode(
		builder.NewHTTPGetNode("fetch", "Fetch Data", "https://api.example.com/data"),
	).AddNode(
		builder.NewJQNode("parse", "Parse JSON", `.data[] | select(.active == true)`),
	).AddNode(
		builder.NewOpenAINode(
			"analyze",
			"Analyze with AI",
			"gpt-4",
			"Analyze this data: {{input}}",
			builder.LLMAPIKey("{{env.api_key}}"),
			builder.LLMTemperature(0.3),
		),
	).AddNode(
		builder.NewHTTPPostNode(
			"store",
			"Store Results",
			"https://api.example.com/results",
			map[string]any{"result": "{{input}}"},
		),
	).Connect("fetch", "parse").
		Connect("parse", "analyze").
		Connect("analyze", "store").
		MustBuild()

	fmt.Println(workflow.Name)
	fmt.Println(len(workflow.Nodes))
	fmt.Println(len(workflow.Edges))
	// Output:
	// Complex Pipeline
	// 4
	// 3
}
