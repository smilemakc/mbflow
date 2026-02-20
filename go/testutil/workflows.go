package testutil

import (
	"time"

	"github.com/google/uuid"
	storagemodels "github.com/smilemakc/mbflow/go/internal/infrastructure/storage/models"
	"github.com/smilemakc/mbflow/go/pkg/builder"
	pkgmodels "github.com/smilemakc/mbflow/go/pkg/models"
)

// CreateSimpleWorkflow creates a simple 3-node linear workflow for testing
// Chain: Transform (passthrough) → Transform (passthrough) → Transform (passthrough)
func CreateSimpleWorkflow() *pkgmodels.Workflow {
	return builder.NewWorkflow("Simple Chain Test").
		AddNode(builder.NewPassthroughNode("n1", "Node 1")).
		AddNode(builder.NewPassthroughNode("n2", "Node 2")).
		AddNode(builder.NewPassthroughNode("n3", "Node 3")).
		Connect("n1", "n2").
		Connect("n2", "n3").
		MustBuild()
}

// CreateParallelWorkflow creates a workflow with parallel branches
// Structure:
//
//	┌─> n2 ─┐
//
// n1 ─┼─> n3 ─┼─> n5
//
//	└─> n4 ─┘
func CreateParallelWorkflow() *pkgmodels.Workflow {
	return builder.NewWorkflow("Parallel Test").
		AddNode(builder.NewPassthroughNode("n1", "Node 1")).
		AddNode(builder.NewPassthroughNode("n2", "Node 2")).
		AddNode(builder.NewPassthroughNode("n3", "Node 3")).
		AddNode(builder.NewPassthroughNode("n4", "Node 4")).
		AddNode(builder.NewNode("n5", "merge", "Merge Node",
			builder.WithConfigValue("strategy", "all"))).
		Connect("n1", "n2").
		Connect("n1", "n3").
		Connect("n1", "n4").
		Connect("n2", "n5").
		Connect("n3", "n5").
		Connect("n4", "n5").
		MustBuild()
}

// CreateHTTPChainWorkflow creates a workflow chain with HTTP calls
func CreateHTTPChainWorkflow(fetchURL, postURL string) *pkgmodels.Workflow {
	wb := builder.NewWorkflow("HTTP Chain Test")

	if fetchURL != "" {
		wb.AddNode(builder.NewHTTPGetNode("fetch", "Fetch Data", fetchURL))
	}

	wb.AddNode(builder.NewTemplateNode("transform", "Transform", "Result: {{input}}"))

	if postURL != "" {
		wb.AddNode(builder.NewHTTPPostNode("post", "Post Data", postURL, nil))
	}

	return wb.Connect("fetch", "transform").
		Connect("transform", "post").
		MustBuild()
}

// CreateLLMWorkflow creates a workflow with LLM node
func CreateLLMWorkflow(apiURL string) *pkgmodels.Workflow {
	return builder.NewWorkflow("LLM Test").
		AddNode(builder.NewPassthroughNode("input", "Input")).
		AddNode(builder.NewOpenAINode("llm", "LLM Process", "gpt-4",
			"Process this: {{input}}",
			builder.LLMSystemPrompt("You are a helpful assistant"),
			builder.WithConfigValue("api_endpoint", apiURL))).
		AddNode(builder.NewPassthroughNode("output", "Output")).
		Connect("input", "llm").
		Connect("llm", "output").
		MustBuild()
}

// CreateVariableSubstitutionWorkflow creates a workflow to test template variables
func CreateVariableSubstitutionWorkflow() *pkgmodels.Workflow {
	return builder.NewWorkflow("Variable Substitution Test",
		builder.WithVariable("api_key", "test-key-123"),
		builder.WithVariable("base_url", "https://api.example.com")).
		AddNode(builder.NewTemplateNode("prepare", "Prepare",
			"API Key: {{env.api_key}}, URL: {{env.base_url}}")).
		AddNode(builder.NewPassthroughNode("result", "Result")).
		Connect("prepare", "result").
		MustBuild()
}

// CreateErrorHandlingWorkflow creates a workflow to test error handling
func CreateErrorHandlingWorkflow() *pkgmodels.Workflow {
	return builder.NewWorkflow("Error Handling Test").
		AddNode(builder.NewHTTPGetNode("failing_http", "Failing HTTP",
			"http://invalid-url-that-will-fail.local")).
		AddNode(builder.NewPassthroughNode("result", "Result")).
		Connect("failing_http", "result").
		MustBuild()
}

// CreateTimeoutWorkflow creates a workflow to test timeout handling
func CreateTimeoutWorkflow() *pkgmodels.Workflow {
	return builder.NewWorkflow("Timeout Test").
		AddNode(builder.NewNode("slow_http", "http", "Slow HTTP",
			builder.WithConfigValue("method", "GET"),
			builder.WithConfigValue("url", "https://httpbin.org/delay/10"),
			builder.WithConfigValue("timeout", 1))).
		AddNode(builder.NewPassthroughNode("result", "Result")).
		Connect("slow_http", "result").
		MustBuild()
}

// WorkflowDomainToModel converts domain Workflow to storage WorkflowModel
func WorkflowDomainToModel(w *pkgmodels.Workflow) *storagemodels.WorkflowModel {
	wm := &storagemodels.WorkflowModel{
		ID:          uuid.New(),
		Name:        w.Name,
		Description: w.Description,
		Status:      "draft",
		Version:     1,
		Variables:   make(storagemodels.JSONBMap),
		Metadata:    make(storagemodels.JSONBMap),
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	// Convert variables
	for k, v := range w.Variables {
		wm.Variables[k] = v
	}

	// Convert metadata
	for k, v := range w.Metadata {
		wm.Metadata[k] = v
	}

	// Convert nodes
	wm.Nodes = make([]*storagemodels.NodeModel, 0, len(w.Nodes))
	for _, node := range w.Nodes {
		nm := &storagemodels.NodeModel{
			ID:       uuid.New(),
			NodeID:   node.ID,
			Name:     node.Name,
			Type:     node.Type,
			Config:   make(storagemodels.JSONBMap),
			Position: storagemodels.JSONBMap{"x": 0, "y": 0},
		}
		// Convert config
		for k, v := range node.Config {
			nm.Config[k] = v
		}
		wm.Nodes = append(wm.Nodes, nm)
	}

	// Convert edges
	wm.Edges = make([]*storagemodels.EdgeModel, 0, len(w.Edges))
	for _, edge := range w.Edges {
		em := &storagemodels.EdgeModel{
			ID:         uuid.New(),
			EdgeID:     edge.ID,
			FromNodeID: edge.From,
			ToNodeID:   edge.To,
			Condition:  make(storagemodels.JSONBMap),
		}
		// If edge has a condition string, store it in the JSONBMap
		if edge.Condition != "" {
			em.Condition["expression"] = edge.Condition
		}
		wm.Edges = append(wm.Edges, em)
	}

	return wm
}
