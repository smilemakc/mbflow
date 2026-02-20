package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/smilemakc/mbflow/go/pkg/builder"
	"github.com/smilemakc/mbflow/go/pkg/models"
	"github.com/smilemakc/mbflow/go/pkg/sdk"
)

func main() {
	// Get API key from environment
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		log.Fatal("OPENAI_API_KEY environment variable is required")
	}

	// Create MBFlow client in standalone mode (no database required)
	client, err := sdk.NewStandaloneClient()
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	ctx := context.Background()

	// Example 1: Simple HTTP workflow
	fmt.Println("=== Example 1: Simple HTTP Workflow ===")
	simpleWorkflow := builder.NewWorkflow("Fetch User Data",
		builder.WithDescription("Fetch user data from JSONPlaceholder API"),
		builder.WithVariable("api_base", "https://jsonplaceholder.typicode.com"),
		builder.WithTags("http", "example"),
	).AddNode(
		builder.NewHTTPGetNode(
			"fetch-user",
			"Fetch User",
			"{{env.api_base}}/users/1",
		),
	).MustBuild()

	created, err := client.Workflows().Create(ctx, simpleWorkflow)
	if err != nil {
		log.Fatalf("Failed to create workflow: %v", err)
	}
	fmt.Printf("Created workflow: %s (ID: %s)\n", created.Name, created.ID)

	// Example 2: Multi-step ETL pipeline
	fmt.Println("\n=== Example 2: ETL Pipeline ===")
	etlWorkflow := builder.NewWorkflow("ETL Pipeline",
		builder.WithDescription("Extract, transform, and load data"),
		builder.WithStatus(models.WorkflowStatusActive),
		builder.WithTags("etl", "data-pipeline"),
		builder.WithVariable("api_base", "https://jsonplaceholder.typicode.com"),
		builder.WithAutoLayout(),
	).AddNode(
		builder.NewHTTPGetNode(
			"extract",
			"Extract Users",
			"{{env.api_base}}/users",
		),
	).AddNode(
		builder.NewJQNode(
			"transform",
			"Transform to Summary",
			`.[] | {id, name, email}`,
		),
	).AddNode(
		builder.NewPassthroughNode(
			"load",
			"Load Results",
			builder.WithNodeDescription("In a real scenario, this would save to a database"),
		),
	).Connect("extract", "transform").
		Connect("transform", "load").
		MustBuild()

	created, err = client.Workflows().Create(ctx, etlWorkflow)
	if err != nil {
		log.Fatalf("Failed to create ETL workflow: %v", err)
	}
	fmt.Printf("Created workflow: %s (ID: %s)\n", created.Name, created.ID)
	fmt.Printf("Nodes: %d, Edges: %d\n", len(created.Nodes), len(created.Edges))

	// Example 3: LLM workflow
	fmt.Println("\n=== Example 3: LLM Analysis Workflow ===")
	llmWorkflow := builder.NewWorkflow("Code Analysis",
		builder.WithDescription("Analyze code using OpenAI GPT-4"),
		builder.WithVariables(map[string]any{
			"openai_api_key": apiKey,
			"model":          "gpt-4",
		}),
		builder.WithAutoLayout(),
	).AddNode(
		builder.NewOpenAINode(
			"detect-language",
			"Detect Language",
			"{{env.model}}",
			"Identify the programming language in this code snippet and respond with just the language name: {{input.code}}",
			builder.LLMAPIKey("{{env.openai_api_key}}"),
			builder.LLMTemperature(0.0),
			builder.LLMMaxTokens(50),
		),
	).AddNode(
		builder.NewOpenAINode(
			"analyze-code",
			"Analyze Code",
			"{{env.model}}",
			"Analyze this code and provide a brief summary of what it does: {{input.code}}",
			builder.LLMAPIKey("{{env.openai_api_key}}"),
			builder.LLMTemperature(0.3),
			builder.LLMMaxTokens(500),
		),
	).Connect("detect-language", "analyze-code").
		MustBuild()

	created, err = client.Workflows().Create(ctx, llmWorkflow)
	if err != nil {
		log.Fatalf("Failed to create LLM workflow: %v", err)
	}
	fmt.Printf("Created workflow: %s (ID: %s)\n", created.Name, created.ID)

	// Example 4: Conditional workflow
	fmt.Println("\n=== Example 4: Conditional Workflow ===")
	conditionalWorkflow := builder.NewWorkflow("User Validation",
		builder.WithDescription("Validate user and route to success/failure handlers"),
		builder.WithVariable("api_base", "https://jsonplaceholder.typicode.com"),
		builder.WithAutoLayout(),
	).AddNode(
		builder.NewHTTPGetNode(
			"fetch-user",
			"Fetch User",
			"{{env.api_base}}/users/{{input.user_id}}",
		),
	).AddNode(
		builder.NewExpressionNode(
			"validate",
			"Validate User",
			`{valid: input.email != nil && input.name != nil}`,
		),
	).AddNode(
		builder.NewPassthroughNode(
			"handle-success",
			"Handle Valid User",
			builder.WithNodeDescription("Process valid user"),
		),
	).AddNode(
		builder.NewPassthroughNode(
			"handle-failure",
			"Handle Invalid User",
			builder.WithNodeDescription("Handle invalid user"),
		),
	).Connect("fetch-user", "validate").
		Connect("validate", "handle-success", builder.WhenTrue("output.valid")).
		Connect("validate", "handle-failure", builder.WhenFalse("output.valid")).
		MustBuild()

	created, err = client.Workflows().Create(ctx, conditionalWorkflow)
	if err != nil {
		log.Fatalf("Failed to create conditional workflow: %v", err)
	}
	fmt.Printf("Created workflow: %s (ID: %s)\n", created.Name, created.ID)
	fmt.Printf("Conditional edges: %d\n", countConditionalEdges(created.Edges))

	// Example 5: Complex multi-node workflow with grid positioning
	fmt.Println("\n=== Example 5: Complex Workflow with Grid Layout ===")
	complexWorkflow := builder.NewWorkflow("Data Processing Pipeline",
		builder.WithDescription("Complex data processing with multiple stages"),
		builder.WithStatus(models.WorkflowStatusActive),
		builder.WithTags("production", "data-pipeline", "complex"),
		builder.WithMetadata("author", "Builder Example"),
		builder.WithMetadata("version", "1.0.0"),
		builder.WithVariable("api_base", "https://jsonplaceholder.typicode.com"),
	).AddNode(
		builder.NewHTTPGetNode(
			"fetch-users",
			"Fetch Users",
			"{{env.api_base}}/users",
			builder.GridPosition(0, 0),
		),
	).AddNode(
		builder.NewHTTPGetNode(
			"fetch-posts",
			"Fetch Posts",
			"{{env.api_base}}/posts",
			builder.GridPosition(0, 1),
		),
	).AddNode(
		builder.NewJQNode(
			"transform-users",
			"Transform Users",
			`.[] | {userId: .id, userName: .name}`,
			builder.GridPosition(1, 0),
		),
	).AddNode(
		builder.NewJQNode(
			"transform-posts",
			"Transform Posts",
			`.[] | {postId: .id, userId: .userId, title: .title}`,
			builder.GridPosition(1, 1),
		),
	).AddNode(
		builder.NewExpressionNode(
			"merge-data",
			"Merge User and Post Data",
			`{users: input.users, posts: input.posts}`,
			builder.GridPosition(2, 0),
		),
	).Connect("fetch-users", "transform-users").
		Connect("fetch-posts", "transform-posts").
		Connect("transform-users", "merge-data").
		Connect("transform-posts", "merge-data").
		MustBuild()

	created, err = client.Workflows().Create(ctx, complexWorkflow)
	if err != nil {
		log.Fatalf("Failed to create complex workflow: %v", err)
	}
	fmt.Printf("Created workflow: %s (ID: %s)\n", created.Name, created.ID)
	fmt.Printf("Total nodes: %d, Total edges: %d\n", len(created.Nodes), len(created.Edges))
	fmt.Printf("Tags: %v\n", created.Tags)
	fmt.Printf("Metadata: %v\n", created.Metadata)

	fmt.Println("\n=== All Examples Completed Successfully ===")
}

func countConditionalEdges(edges []*models.Edge) int {
	count := 0
	for _, edge := range edges {
		if edge.Condition != "" {
			count++
		}
	}
	return count
}
