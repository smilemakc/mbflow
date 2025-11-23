package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"mbflow"

	"github.com/google/uuid"
)

// TelegramMessageDemo sends a templated message using the telegram-message executor.
// It demonstrates how to:
//  1. Load credentials from environment variables
//  2. Substitute execution variables inside the Telegram message
//  3. Read the Telegram API response stored in the execution state
func main() {
	fmt.Println("=== Telegram Message WorkflowEngine Demo ===\n")

	botToken := os.Getenv("TELEGRAM_BOT_TOKEN")
	chatID := os.Getenv("TELEGRAM_CHAT_ID")
	if botToken == "" || chatID == "" {
		fmt.Println("Please set TELEGRAM_BOT_TOKEN and TELEGRAM_CHAT_ID to send a real message.")
		fmt.Println("Export both variables and rerun: TELEGRAM_BOT_TOKEN=... TELEGRAM_CHAT_ID=... go run main.go")
		return
	}

	executor := mbflow.NewWorkflowEngine(&mbflow.EngineConfig{
		VerboseLogging: true,
	})

	workflowID := uuid.NewString()
	executionID := uuid.NewString()

	nodes := []mbflow.NodeConfig{
		{
			ID:   "telegram-message",
			Name: "Send Telegram Message",
			Type: "telegram-message",
			Config: map[string]any{
				"bot_token":  botToken,
				"chat_id":    chatID,
				"text":       "Hello, {{user_name}}! Workflow {{workflow_id}} started at {{timestamp}}.",
				"parse_mode": "Markdown",
				"output_key": "telegram_message",
			},
		},
	}

	initialVariables := map[string]any{
		"user_name":   "mbflow user",
		"workflow_id": workflowID,
		"timestamp":   time.Now().Format(time.RFC3339),
	}

	ctx := context.Background()
	state, err := executor.ExecuteWorkflow(ctx, workflowID, executionID, nodes, nil, initialVariables)
	if err != nil {
		log.Fatalf("workflow execution failed: %v", err)
	}

	fmt.Println("\n=== Execution Results ===")
	fmt.Printf("Status: %s\n", state.GetStatusString())
	if msg, ok := state.GetVariable("telegram_message"); ok {
		fmt.Printf("Telegram message response: %v\n", msg)
	}
}
