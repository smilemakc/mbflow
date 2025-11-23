package main

import (
	"context"
	"fmt"
	"log"

	"mbflow"

	"github.com/google/uuid"
)

// JSONParserDemo demonstrates the json-parser node functionality.
// This example shows how to parse JSON strings into objects for nested field access.
func main() {
	fmt.Println("=== JSON Parser Node Demo ===\n")

	// Create executor
	executor := mbflow.NewWorkflowEngine(&mbflow.EngineConfig{
		VerboseLogging: true,
	})

	ctx := context.Background()
	workflowID := uuid.NewString()
	executionID := uuid.NewString()

	// Example 1: Parse a JSON string and access nested fields
	fmt.Println("Example 1: Parsing JSON string for nested field access")
	fmt.Println("-------------------------------------------------------")

	// Node 1: Parse the JSON string
	nodeParseJSON, err := mbflow.NewNodeFromConfig(mbflow.NodeConfig{
		ID:         uuid.NewString(),
		WorkflowID: workflowID,
		Type:       mbflow.NodeTypeJSONParser,
		Name:       "Parse API Response",
		Config: map[string]any{
			"input_key":  "api_response",
			"output_key": "parsed_response",
		},
	})
	if err != nil {
		log.Fatalf("Failed to create nodeParseJSON: %v", err)
	}

	// Node 2: Use nested field in conditional routing
	nodeCheckStatus, err := mbflow.NewNodeFromConfig(mbflow.NodeConfig{
		ID:         uuid.NewString(),
		WorkflowID: workflowID,
		Type:       mbflow.NodeTypeConditionalRouter,
		Name:       "Check User Status",
		Config: map[string]any{
			"input_key": "parsed_response.user.status",
			"routes": map[string]string{
				"active":   "active_path",
				"inactive": "inactive_path",
				"default":  "default_path",
			},
		},
	})
	if err != nil {
		log.Fatalf("Failed to create nodeCheckStatus: %v", err)
	}

	// Node 3a: Active path
	nodeActivePath, err := mbflow.NewNodeFromConfig(mbflow.NodeConfig{
		ID:         uuid.NewString(),
		WorkflowID: workflowID,
		Type:       mbflow.NodeTypeDataAggregator,
		Name:       "Handle Active User",
		Config: map[string]any{
			"fields": map[string]string{
				"message": "parsed_response.user.name",
			},
			"output_key": "result",
		},
	})
	if err != nil {
		log.Fatalf("Failed to create nodeActivePath: %v", err)
	}

	// Node 3b: Inactive path
	nodeInactivePath, err := mbflow.NewNodeFromConfig(mbflow.NodeConfig{
		ID:         uuid.NewString(),
		WorkflowID: workflowID,
		Type:       mbflow.NodeTypeDataAggregator,
		Name:       "Handle Inactive User",
		Config: map[string]any{
			"fields": map[string]string{
				"message": "parsed_response.user.name",
			},
			"output_key": "result",
		},
	})
	if err != nil {
		log.Fatalf("Failed to create nodeInactivePath: %v", err)
	}

	// Create workflow
	nodes := []mbflow.Node{
		nodeParseJSON,
		nodeCheckStatus,
		nodeActivePath,
		nodeInactivePath,
	}

	edges := mbflow.NewRelationshipBuilder(workflowID).
		Direct(nodeParseJSON, nodeCheckStatus).
		Conditional(nodeCheckStatus, nodeActivePath, "parsed_response.user.status == 'active'").
		Conditional(nodeCheckStatus, nodeInactivePath, "parsed_response.user.status == 'inactive'").
		Build()

	nodeConfigs := mbflow.NodesToConfigs(nodes)
	edgeConfigs := mbflow.EdgesToConfigs(edges)

	// Test with active user
	fmt.Println("\n▶ Test Case 1: Active User")
	initialVariables := map[string]interface{}{
		"api_response": `{
			"user": {
				"id": 123,
				"name": "John Doe",
				"email": "john@example.com",
				"status": "active",
				"roles": ["admin", "user"]
			},
			"timestamp": "2025-11-23T21:00:00Z"
		}`,
	}

	state, err := executor.ExecuteWorkflow(ctx, workflowID, executionID, nodeConfigs, edgeConfigs, initialVariables)
	if err != nil {
		log.Fatalf("Workflow execution failed: %v", err)
	}

	fmt.Printf("\nStatus: %s\n", state.GetStatusString())
	variables := state.GetAllVariables()

	// Display parsed response
	if parsedResponse, ok := variables["parsed_response"]; ok {
		fmt.Printf("\n✅ Parsed Response (type: %T):\n", parsedResponse)
		if respMap, ok := parsedResponse.(map[string]interface{}); ok {
			if user, ok := respMap["user"].(map[string]interface{}); ok {
				fmt.Printf("   User ID: %v\n", user["id"])
				fmt.Printf("   User Name: %v\n", user["name"])
				fmt.Printf("   User Email: %v\n", user["email"])
				fmt.Printf("   User Status: %v\n", user["status"])
				fmt.Printf("   User Roles: %v\n", user["roles"])
			}
		}
	}

	// Test with inactive user
	fmt.Println("\n\n▶ Test Case 2: Inactive User")
	executionID2 := uuid.NewString()
	initialVariables2 := map[string]interface{}{
		"api_response": `{
			"user": {
				"id": 456,
				"name": "Jane Smith",
				"email": "jane@example.com",
				"status": "inactive",
				"roles": ["user"]
			},
			"timestamp": "2025-11-23T21:00:00Z"
		}`,
	}

	state2, err := executor.ExecuteWorkflow(ctx, workflowID, executionID2, nodeConfigs, edgeConfigs, initialVariables2)
	if err != nil {
		log.Fatalf("Workflow execution failed: %v", err)
	}

	fmt.Printf("\nStatus: %s\n", state2.GetStatusString())
	variables2 := state2.GetAllVariables()

	if parsedResponse, ok := variables2["parsed_response"]; ok {
		fmt.Printf("\n✅ Parsed Response (type: %T):\n", parsedResponse)
		if respMap, ok := parsedResponse.(map[string]interface{}); ok {
			if user, ok := respMap["user"].(map[string]interface{}); ok {
				fmt.Printf("   User ID: %v\n", user["id"])
				fmt.Printf("   User Name: %v\n", user["name"])
				fmt.Printf("   User Email: %v\n", user["email"])
				fmt.Printf("   User Status: %v\n", user["status"])
				fmt.Printf("   User Roles: %v\n", user["roles"])
			}
		}
	}

	// Example 2: Handling parse errors gracefully
	fmt.Println("\n\n▶ Test Case 3: Invalid JSON with fail_on_error=false")
	executionID3 := uuid.NewString()

	nodeParseInvalid, err := mbflow.NewNodeFromConfig(mbflow.NodeConfig{
		ID:         uuid.NewString(),
		WorkflowID: workflowID,
		Type:       mbflow.NodeTypeJSONParser,
		Name:       "Try Parse Invalid JSON",
		Config: map[string]any{
			"input_key":     "invalid_json",
			"fail_on_error": false,
		},
	})
	if err != nil {
		log.Fatalf("Failed to create nodeParseInvalid: %v", err)
	}

	nodes3 := []mbflow.Node{nodeParseInvalid}
	nodeConfigs3 := mbflow.NodesToConfigs(nodes3)

	initialVariables3 := map[string]interface{}{
		"invalid_json": "this is not valid JSON",
	}

	state3, err := executor.ExecuteWorkflow(ctx, workflowID, executionID3, nodeConfigs3, nil, initialVariables3)
	if err != nil {
		log.Fatalf("Workflow execution failed: %v", err)
	}

	fmt.Printf("\nStatus: %s\n", state3.GetStatusString())
	fmt.Printf("Original value preserved: %v\n", state3.GetAllVariables()["invalid_json"])

	fmt.Println("\n=== Demo Complete ===")
	fmt.Println("\nKey Takeaways:")
	fmt.Println("1. ✅ JSON strings are parsed into structured objects")
	fmt.Println("2. ✅ Nested fields can be accessed using dot notation (e.g., 'user.status')")
	fmt.Println("3. ✅ Conditional routing works with nested fields")
	fmt.Println("4. ✅ Parse errors can be handled gracefully with fail_on_error=false")
	fmt.Println("5. ✅ Already-parsed objects are passed through unchanged")
}
