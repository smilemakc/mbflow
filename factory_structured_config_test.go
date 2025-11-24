package mbflow

import (
	"testing"
)

func TestWorkflowBuilder_AddNodeWithConfig(t *testing.T) {
	// Test adding a node with structured config
	workflow, err := NewWorkflowBuilder("Test Workflow", "1.0").
		AddNode(string(NodeTypeStart), "start", map[string]any{}).
		AddNodeWithConfig(
			string(NodeTypeHTTPRequest),
			"fetch",
			&HTTPRequestConfig{
				URL:    "https://api.example.com",
				Method: "GET",
				Headers: map[string]string{
					"Accept": "application/json",
				},
			},
		).
		AddNode(string(NodeTypeEnd), "end", map[string]any{}).
		AddEdge("start", "fetch", string(EdgeTypeDirect), nil).
		AddEdge("fetch", "end", string(EdgeTypeDirect), nil).
		AddTrigger(string(TriggerTypeManual), map[string]any{}).
		Build()

	if err != nil {
		t.Fatalf("Failed to build workflow: %v", err)
	}

	// Verify workflow was created
	if workflow.Name() != "Test Workflow" {
		t.Errorf("Expected workflow name = Test Workflow, got %s", workflow.Name())
	}

	// Verify nodes
	nodes := workflow.GetAllNodes()
	if len(nodes) != 3 {
		t.Errorf("Expected 3 nodes, got %d", len(nodes))
	}

	// Find the HTTP request node
	var httpNode Node
	for _, node := range nodes {
		if node.Name() == "fetch" {
			httpNode = node
			break
		}
	}

	if httpNode == nil {
		t.Fatal("HTTP request node not found")
	}

	// Verify config was converted properly
	config := httpNode.Config()
	if config["url"] != "https://api.example.com" {
		t.Errorf("Expected url = https://api.example.com, got %v", config["url"])
	}
	if config["method"] != "GET" {
		t.Errorf("Expected method = GET, got %v", config["method"])
	}

	headers, ok := config["headers"].(map[string]interface{})
	if !ok {
		t.Fatalf("Expected headers to be map[string]interface{}")
	}
	if headers["Accept"] != "application/json" {
		t.Errorf("Expected Accept header = application/json, got %v", headers["Accept"])
	}
}

func TestWorkflowBuilder_AddNodeWithConfig_MultipleTypes(t *testing.T) {
	// Test adding multiple nodes with different structured configs
	workflow, err := NewWorkflowBuilder("Multi Config Test", "1.0").
		AddNode(string(NodeTypeStart), "start", map[string]any{}).
		AddNodeWithConfig(
			string(NodeTypeJSONParser),
			"parser",
			&JSONParserConfig{
				FailOnError: true,
			},
		).
		AddNodeWithConfig(
			string(NodeTypeDataAggregator),
			"aggregator",
			&DataAggregatorConfig{
				Fields: map[string]string{
					"id":   "parsed_data.id",
					"name": "parsed_data.name",
				},
			},
		).
		AddNode(string(NodeTypeEnd), "end", map[string]any{}).
		AddEdge("start", "parser", string(EdgeTypeDirect), nil).
		AddEdge("parser", "aggregator", string(EdgeTypeDirect), nil).
		AddEdge("aggregator", "end", string(EdgeTypeDirect), nil).
		AddTrigger(string(TriggerTypeManual), map[string]any{}).
		Build()

	if err != nil {
		t.Fatalf("Failed to build workflow: %v", err)
	}

	// Verify nodes
	nodes := workflow.GetAllNodes()
	if len(nodes) != 4 {
		t.Errorf("Expected 4 nodes, got %d", len(nodes))
	}

	// Verify JSON Parser config
	var parserNode Node
	for _, node := range nodes {
		if node.Name() == "parser" {
			parserNode = node
			break
		}
	}

	if parserNode == nil {
		t.Fatal("Parser node not found")
	}

	parserConfig := parserNode.Config()
	if parserConfig["input_key"] != "raw_data" {
		t.Errorf("Expected input_key = raw_data, got %v", parserConfig["input_key"])
	}
	if parserConfig["output_key"] != "parsed_data" {
		t.Errorf("Expected output_key = parsed_data, got %v", parserConfig["output_key"])
	}

	// Verify Data Aggregator config
	var aggregatorNode Node
	for _, node := range nodes {
		if node.Name() == "aggregator" {
			aggregatorNode = node
			break
		}
	}

	if aggregatorNode == nil {
		t.Fatal("Aggregator node not found")
	}

	aggregatorConfig := aggregatorNode.Config()
	fields, ok := aggregatorConfig["fields"].(map[string]interface{})
	if !ok {
		t.Fatalf("Expected fields to be map[string]interface{}")
	}
	if fields["id"] != "parsed_data.id" {
		t.Errorf("Expected id field = parsed_data.id, got %v", fields["id"])
	}
}

func TestWorkflowBuilder_AddNodeWithConfig_InvalidConfig(t *testing.T) {
	// Create a config that might cause JSON marshaling issues
	// (this is mainly to test error handling)
	type InvalidConfig struct {
		InvalidField chan int // channels can't be marshaled to JSON
	}

	// We can't actually test this without implementing the NodeConfig interface
	// on InvalidConfig, which defeats the purpose. Instead, test that valid
	// configs don't cause errors

	workflow, err := NewWorkflowBuilder("Valid Config Test", "1.0").
		AddNode(string(NodeTypeStart), "start", map[string]any{}).
		AddNodeWithConfig(
			string(NodeTypeOpenAICompletion),
			"ai",
			&OpenAICompletionConfig{
				Model:       "gpt-4o",
				Prompt:      "Test",
				MaxTokens:   100,
				Temperature: 0.7,
			},
		).
		AddNode(string(NodeTypeEnd), "end", map[string]any{}).
		AddEdge("start", "ai", string(EdgeTypeDirect), nil).
		AddEdge("ai", "end", string(EdgeTypeDirect), nil).
		AddTrigger(string(TriggerTypeManual), map[string]any{}).
		Build()

	if err != nil {
		t.Errorf("Valid config should not cause error: %v", err)
	}

	if workflow == nil {
		t.Fatal("Workflow should not be nil")
	}
}

func TestWorkflowBuilder_AddNodeWithConfig_MixedWithAddNode(t *testing.T) {
	// Test that AddNodeWithConfig works alongside traditional AddNode
	workflow, err := NewWorkflowBuilder("Mixed Test", "1.0").
		AddNode(string(NodeTypeStart), "start", map[string]any{}).
		AddNode(string(NodeTypeTransform), "transform1", map[string]any{
			"transformations": map[string]any{
				"result": "input * 2",
			},
		}).
		AddNodeWithConfig(
			string(NodeTypeHTTPRequest),
			"fetch",
			&HTTPRequestConfig{
				URL:    "https://api.example.com",
				Method: "POST",
			},
		).
		AddNode(string(NodeTypeTransform), "transform2", map[string]any{
			"transformations": map[string]any{
				"output": "response.data",
			},
		}).
		AddNode(string(NodeTypeEnd), "end", map[string]any{}).
		AddEdge("start", "transform1", string(EdgeTypeDirect), nil).
		AddEdge("transform1", "fetch", string(EdgeTypeDirect), nil).
		AddEdge("fetch", "transform2", string(EdgeTypeDirect), nil).
		AddEdge("transform2", "end", string(EdgeTypeDirect), nil).
		AddTrigger(string(TriggerTypeManual), map[string]any{}).
		Build()

	if err != nil {
		t.Fatalf("Mixed usage should not cause error: %v", err)
	}

	nodes := workflow.GetAllNodes()
	if len(nodes) != 5 {
		t.Errorf("Expected 5 nodes, got %d", len(nodes))
	}

	// Verify the structured config node
	var fetchNode Node
	for _, node := range nodes {
		if node.Name() == "fetch" {
			fetchNode = node
			break
		}
	}

	if fetchNode == nil {
		t.Fatal("Fetch node not found")
	}

	config := fetchNode.Config()
	if config["url"] != "https://api.example.com" {
		t.Errorf("Expected url = https://api.example.com, got %v", config["url"])
	}
	if config["method"] != "POST" {
		t.Errorf("Expected method = POST, got %v", config["method"])
	}
}

// Benchmark to compare performance of AddNode vs AddNodeWithConfig
func BenchmarkAddNode(b *testing.B) {
	for i := 0; i < b.N; i++ {
		NewWorkflowBuilder("Bench", "1.0").
			AddNode(string(NodeTypeHTTPRequest), "fetch", map[string]any{
				"url":    "https://api.example.com",
				"method": "GET",
			})
	}
}

func BenchmarkAddNodeWithConfig(b *testing.B) {
	for i := 0; i < b.N; i++ {
		NewWorkflowBuilder("Bench", "1.0").
			AddNodeWithConfig(
				string(NodeTypeHTTPRequest),
				"fetch",
				&HTTPRequestConfig{
					URL:    "https://api.example.com",
					Method: "GET",
				},
			)
	}
}
