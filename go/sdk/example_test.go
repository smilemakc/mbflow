package mbflow_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"

	mbflow "github.com/smilemakc/mbflow/go/sdk"
	"github.com/smilemakc/mbflow/go/sdk/builder"
	"github.com/smilemakc/mbflow/go/sdk/mock"
	"github.com/smilemakc/mbflow/go/sdk/models"
)

func ExampleNewClient() {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{
			"id": "wf-1", "name": "Hello", "status": "active",
		})
	}))
	defer server.Close()

	client, err := mbflow.NewClient(
		mbflow.WithHTTP(server.URL),
		mbflow.WithSystemKey("test-key"),
	)
	if err != nil {
		panic(err)
	}
	defer client.Close()

	wf, err := client.Workflows().Get(context.Background(), "wf-1")
	if err != nil {
		panic(err)
	}
	fmt.Println(wf.Name)
	// Output: Hello
}

func Example_builder() {
	wf, err := builder.NewWorkflow("Content Pipeline",
		builder.WithDescription("Fetch and process content"),
		builder.WithVariable("api_url", "https://api.example.com"),
	).
		AddHTTPNode("fetch", "Fetch Data",
			builder.URL("{{env.api_url}}/data"),
			builder.Method("GET"),
		).
		AddLLMNode("process", "Process with AI",
			builder.Provider("openai"),
			builder.Model("gpt-4"),
			builder.Prompt("Summarize: {{nodes.fetch.output.body}}"),
		).
		Connect("fetch", "process").
		Build()

	if err != nil {
		panic(err)
	}
	fmt.Println(wf.Name, len(wf.Nodes), len(wf.Edges))
	// Output: Content Pipeline 2 1
}

func Example_mock() {
	m := mock.NewClient()
	m.Workflows().(*mock.WorkflowServiceMock).OnGet("wf-1", &models.Workflow{
		ID: "wf-1", Name: "Mocked",
	}, nil)

	wf, _ := m.Workflows().Get(context.Background(), "wf-1")
	fmt.Println(wf.Name)
	// Output: Mocked
}
