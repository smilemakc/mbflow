// Custom executor example for MBFlow SDK
package main

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/smilemakc/mbflow/pkg/executor"
	"github.com/smilemakc/mbflow/pkg/models"
	"github.com/smilemakc/mbflow/pkg/sdk"
)

// UppercaseExecutor is a custom executor that converts text to uppercase.
type UppercaseExecutor struct {
	*executor.BaseExecutor
}

// NewUppercaseExecutor creates a new uppercase executor.
func NewUppercaseExecutor() *UppercaseExecutor {
	return &UppercaseExecutor{
		BaseExecutor: executor.NewBaseExecutor("uppercase"),
	}
}

// Execute converts the input text to uppercase.
func (e *UppercaseExecutor) Execute(ctx context.Context, config map[string]any, input any) (any, error) {
	// Get the text field from config or input
	var text string

	if config["text"] != nil {
		text = config["text"].(string)
	} else if inputMap, ok := input.(map[string]any); ok {
		if inputMap["text"] != nil {
			text = inputMap["text"].(string)
		}
	}

	if text == "" {
		return nil, fmt.Errorf("text is required")
	}

	// Convert to uppercase
	result := strings.ToUpper(text)

	return map[string]any{
		"original":  text,
		"uppercase": result,
	}, nil
}

// Validate validates the uppercase executor configuration.
func (e *UppercaseExecutor) Validate(config map[string]any) error {
	// Text can come from either config or input, so validation is lenient
	return nil
}

func main() {
	// Create executor manager and register custom executor
	executorManager := executor.NewManager()
	if err := executorManager.Register("uppercase", NewUppercaseExecutor()); err != nil {
		log.Fatalf("Failed to register custom executor: %v", err)
	}

	// Create client with custom executor manager
	client, err := sdk.NewClient(
		sdk.WithEmbeddedMode(
			"postgres://mbflow:mbflow@localhost:5432/mbflow?sslmode=disable",
			"redis://localhost:6379",
		),
		sdk.WithExecutorManager(executorManager),
	)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	ctx := context.Background()

	// Create a workflow using the custom executor
	workflow := &models.Workflow{
		Name:        "Custom Executor Workflow",
		Description: "Demonstrates using a custom executor",
		Nodes: []*models.Node{
			{
				ID:   "uppercase-node",
				Name: "Convert to Uppercase",
				Type: "uppercase",
				Config: map[string]any{
					"text": "hello world from mbflow!",
				},
			},
		},
	}

	// Create the workflow
	fmt.Println("Creating workflow with custom executor...")
	createdWorkflow, err := client.Workflows().Create(ctx, workflow)
	if err != nil {
		log.Fatalf("Failed to create workflow: %v", err)
	}

	fmt.Printf("Workflow created: %s (ID: %s)\n", createdWorkflow.Name, createdWorkflow.ID)

	// Execute the workflow
	fmt.Println("\nExecuting workflow...")
	execution, err := client.Executions().RunSync(ctx, createdWorkflow.ID, nil)
	if err != nil {
		log.Fatalf("Failed to execute workflow: %v", err)
	}

	// Print results
	if execution.Status == models.ExecutionStatusCompleted {
		fmt.Println("\nExecution completed successfully!")
		fmt.Printf("Output: %+v\n", execution.Output)
	} else {
		fmt.Printf("\nExecution failed: %s\n", execution.Error)
	}

	fmt.Println("\nDone!")
}
