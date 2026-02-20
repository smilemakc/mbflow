package main

import (
	"fmt"
	"log"

	"github.com/smilemakc/mbflow/go/pkg/models"
	"github.com/smilemakc/mbflow/go/pkg/visualization"
)

func main() {
	fmt.Println("=== MBFlow Workflow Visualization Example ===")

	// Create a sample workflow: User Onboarding
	workflow := createUserOnboardingWorkflow()

	// Example 1: Print Mermaid diagram
	fmt.Println("1. Mermaid Flowchart Diagram:")
	fmt.Println("----------------------------")
	if err := visualization.PrintWorkflow(workflow, "mermaid", nil); err != nil {
		log.Fatal(err)
	}

	// Example 2: Print ASCII tree (compact mode)
	fmt.Println("\n2. ASCII Tree (Compact Mode):")
	fmt.Println("------------------------------")
	compactOpts := &visualization.RenderOptions{
		ShowConfig:     false,
		ShowConditions: true,
		CompactMode:    true,
		UseColor:       true,
	}
	asciiCompact, err := visualization.RenderWorkflow(workflow, "ascii", compactOpts)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(asciiCompact)

	// Example 3: Print ASCII tree (detailed mode)
	fmt.Println("\n3. ASCII Tree (Detailed Mode):")
	fmt.Println("-------------------------------")
	detailedOpts := &visualization.RenderOptions{
		ShowConfig:     true,
		ShowConditions: true,
		CompactMode:    false,
		UseColor:       true,
	}
	asciiDetailed, err := visualization.RenderWorkflow(workflow, "ascii", detailedOpts)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(asciiDetailed)

	// Example 4: Generate Mermaid diagram with ELK layout (adaptive for complex graphs)
	fmt.Println("\n4. Mermaid Diagram (ELK Layout with Colors):")
	fmt.Println("---------------------------------------------")
	elkOpts := &visualization.RenderOptions{
		ShowConfig:     true,
		ShowConditions: true,
		Direction:      "elk", // Adaptive layout
	}
	mermaidELK, err := visualization.RenderWorkflow(workflow, "mermaid", elkOpts)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(mermaidELK)

	// Example 5: Save Mermaid diagram to file
	fmt.Println("\n5. Saving Mermaid diagram to file...")
	filename := "user_onboarding_workflow.mmd"
	if err := visualization.SaveWorkflowToFile(workflow, "mermaid", filename, nil); err != nil {
		log.Fatal(err)
	}
	fmt.Printf("✓ Diagram saved to %s\n", filename)

	fmt.Println("\n=== Example Complete ===")
}

func createUserOnboardingWorkflow() *models.Workflow {
	workflow := &models.Workflow{
		ID:          "user-onboarding-v1",
		Name:        "User Onboarding Workflow",
		Description: "Automated user onboarding process with email and task creation",
		Version:     1,
		Status:      models.WorkflowStatusActive,
		Tags:        []string{"onboarding", "automation"},
	}

	// Define nodes
	createProfile := &models.Node{
		ID:          "create_profile",
		Name:        "Create User Profile",
		Type:        "http",
		Description: "Creates a new user profile in the system",
		Config: map[string]any{
			"method":  "POST",
			"url":     "https://api.example.com/v1/profiles",
			"headers": map[string]string{"Content-Type": "application/json"},
		},
	}

	sendWelcomeEmail := &models.Node{
		ID:          "send_welcome_email",
		Name:        "Send Welcome Email",
		Type:        "http",
		Description: "Sends a welcome email via SendGrid",
		Config: map[string]any{
			"method": "POST",
			"url":    "https://api.sendgrid.com/v3/mail/send",
		},
	}

	createOnboardingTasks := &models.Node{
		ID:          "create_onboarding_tasks",
		Name:        "Create Onboarding Tasks",
		Type:        "http",
		Description: "Creates initial onboarding tasks",
		Config: map[string]any{
			"method": "POST",
			"url":    "https://api.example.com/v1/tasks/bulk",
		},
	}

	trackEvent := &models.Node{
		ID:          "track_event",
		Name:        "Track Onboarding Event",
		Type:        "http",
		Description: "Tracks the onboarding event in analytics",
		Config: map[string]any{
			"method": "POST",
			"url":    "https://api.example.com/v1/analytics/track",
		},
	}

	// Add nodes to workflow
	workflow.Nodes = []*models.Node{
		createProfile,
		sendWelcomeEmail,
		createOnboardingTasks,
		trackEvent,
	}

	// Define edges
	workflow.Edges = []*models.Edge{
		{
			ID:   "e1",
			From: "create_profile",
			To:   "send_welcome_email",
		},
		{
			ID:   "e2",
			From: "create_profile",
			To:   "create_onboarding_tasks",
		},
		{
			ID:   "e3",
			From: "send_welcome_email",
			To:   "track_event",
		},
	}

	return workflow
}
