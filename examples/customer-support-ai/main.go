package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/smilemakc/mbflow"

	"github.com/google/uuid"
)

// CustomerSupportAIWorkflow demonstrates a complex customer support automation workflow
// with intelligent routing, sentiment analysis, and escalation logic.
//
// Workflow structure:
// 1. Receive customer inquiry
// 2. Classify inquiry type (technical, billing, general)
// 3. Analyze sentiment (positive, neutral, negative)
// 4. Route based on classification and sentiment:
//   - Technical + Negative → Escalate to senior support
//   - Technical + Neutral/Positive → Generate technical response
//   - Billing → Check account status → Handle accordingly
//   - General → Generate standard response
//
// 5. Generate personalized response
// 6. Quality check response
// 7. If quality is low, regenerate with more context
// 8. Send response and log interaction
func main() {
	// Parse command line arguments
	customerMessageFlag := flag.String("message", "I've been charged twice for my order #12345. This is unacceptable! I want a refund immediately.", "Customer message")
	flag.Parse()

	customerMessage := *customerMessageFlag

	fmt.Printf("=== AI-Powered Customer Support Workflow Demo ===\n\n")
	fmt.Printf("Customer Message: %s\n\n", customerMessage)

	// Get OpenAI API key from environment
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		fmt.Println("ERROR: OPENAI_API_KEY environment variable is required for this demo.")
		fmt.Printf("Please set OPENAI_API_KEY to run this example.\n\n")
		os.Exit(1)
	}

	// Start mock server in background
	mockServer := NewMockServer("8081")
	go func() {
		if err := mockServer.Start(); err != nil {
			log.Printf("Mock server error: %v", err)
		}
	}()

	// Give mock server time to start
	fmt.Println("Starting mock server on port 8081...")
	time.Sleep(500 * time.Millisecond)
	fmt.Println("Mock server ready!")
	fmt.Println()

	// Create executor with monitoring enabled
	executor := mbflow.NewWorkflowEngine(&mbflow.EngineConfig{
		OpenAIAPIKey:     apiKey,
		EnableMonitoring: true,
		VerboseLogging:   true,
	})

	ctx := context.Background()
	workflowID := uuid.New()
	executionID := uuid.New()

	fmt.Printf("Workflow ID: %s\n", workflowID)
	fmt.Printf("Execution ID: %s\n\n", executionID)

	// Node 1: Extract customer information
	nodeExtractInfo, err := mbflow.NewNode(
		uuid.New(),
		workflowID,
		mbflow.NodeTypeOpenAICompletion,
		"Extract Customer Information",
		map[string]any{
			"model": "gpt-4",
			"prompt": `Extract structured information from this customer inquiry:

{{customer_message}}

Return JSON with:
{
  "customer_name": "extracted name or 'Unknown'",
  "order_id": "order ID if mentioned or null",
  "product": "product name if mentioned or null",
  "urgency": "high/medium/low"
}`,
			"max_tokens":  200,
			"temperature": 0.1,
			"output_key":  "customer_info",
		},
	)
	if err != nil {
		log.Fatalf("Failed to create nodeExtractInfo: %v", err)
	}

	// Node 2: Classify inquiry type
	nodeClassifyInquiry, err := mbflow.NewNode(
		uuid.New(),
		workflowID,
		mbflow.NodeTypeOpenAICompletion,
		"Classify Inquiry Type",
		map[string]any{
			"model": "gpt-4",
			"prompt": `Classify this customer inquiry into ONE category:

Categories:
- technical: Technical issues, bugs, how-to questions
- billing: Payment, refunds, subscription issues
- shipping: Delivery, tracking, shipping problems
- product: Product features, compatibility, recommendations
- account: Login, password, account settings
- general: General questions, feedback

Inquiry: {{customer_message}}

Respond with ONLY the category name.`,
			"max_tokens":  10,
			"temperature": 0.1,
			"output_key":  "inquiry_type",
		},
	)
	if err != nil {
		log.Fatalf("Failed to create nodeClassifyInquiry: %v", err)
	}

	// Node 3: Analyze sentiment
	nodeAnalyzeSentiment, err := mbflow.NewNode(
		uuid.New(),
		workflowID,
		mbflow.NodeTypeOpenAICompletion,
		"Analyze Customer Sentiment",
		map[string]any{
			"model": "gpt-4",
			"prompt": `Analyze the sentiment of this customer message:

{{customer_message}}

Respond with ONLY one word: positive, neutral, or negative`,
			"max_tokens":  10,
			"temperature": 0.1,
			"output_key":  "sentiment",
		},
	)
	if err != nil {
		log.Fatalf("Failed to create nodeAnalyzeSentiment: %v", err)
	}

	// Node 4: Check if billing inquiry
	nodeCheckBilling, err := mbflow.NewNode(
		uuid.New(),
		workflowID,
		mbflow.NodeTypeConditionalRouter,
		"Check if Billing Inquiry",
		map[string]any{
			"input_key": "inquiry_type",
			"routes": map[string]string{
				"billing": "fetch_account",
				"default": "check_escalation",
			},
		},
	)
	if err != nil {
		log.Fatalf("Failed to create nodeCheckBilling: %v", err)
	}

	// Node 5: Fetch account status (for billing inquiries)
	nodeFetchAccount, err := mbflow.NewNode(
		uuid.New(),
		workflowID,
		mbflow.NodeTypeHTTPRequest,
		"Fetch Account Status",
		map[string]any{
			"url":    "http://localhost:8081/accounts/{{customer_info.order_id}}",
			"method": "GET",
			"headers": map[string]string{
				"Authorization": "Bearer {{api_token}}",
			},
			"output_key": "account_status",
		},
	)
	if err != nil {
		log.Fatalf("Failed to create nodeFetchAccount: %v", err)
	}

	// Node 6: Analyze account status
	nodeAnalyzeAccount, err := mbflow.NewNode(
		uuid.New(),
		workflowID,
		mbflow.NodeTypeOpenAICompletion,
		"Analyze Account Status",
		map[string]any{
			"model": "gpt-4",
			"prompt": `Based on this account information, determine the appropriate action:

Account Status: {{account_status}}
Customer Inquiry: {{customer_message}}

Respond with ONE of:
- refund_eligible
- payment_issue
- subscription_active
- account_suspended
- needs_manual_review`,
			"max_tokens":  20,
			"temperature": 0.1,
			"output_key":  "account_action",
		},
	)
	if err != nil {
		log.Fatalf("Failed to create nodeAnalyzeAccount: %v", err)
	}

	// Node 7: Check escalation criteria
	nodeCheckEscalation, err := mbflow.NewNode(
		uuid.New(),
		workflowID,
		mbflow.NodeTypeOpenAICompletion,
		"Check Escalation Criteria",
		map[string]any{
			"model": "gpt-4",
			"prompt": `Determine if this customer support case should be escalated to a human agent.

Inquiry Type: {{inquiry_type}}
Sentiment: {{sentiment}}
Customer Urgency: {{customer_info.urgency}}
Account Action: {{account_action}}

Escalation criteria:
1. Technical issue with negative sentiment
2. High urgency customer
3. Account needs manual review
4. Complex billing disputes
5. Sensitive or legal matters

Respond with ONLY one word: escalate or generate_response`,
			"max_tokens":  10,
			"temperature": 0.1,
			"output_key":  "escalation_decision",
		},
	)
	if err != nil {
		log.Fatalf("Failed to create nodeCheckEscalation: %v", err)
	}

	// Node 8: Escalate to human agent
	nodeEscalate, err := mbflow.NewNode(
		uuid.New(),
		workflowID,
		mbflow.NodeTypeHTTPRequest,
		"Escalate to Human Agent",
		map[string]any{
			"url":    "http://localhost:8081/support/escalate",
			"method": "POST",
			"body": map[string]any{
				"customer_message": "{{customer_message}}",
				"customer_info":    "{{customer_info}}",
				"inquiry_type":     "{{inquiry_type}}",
				"sentiment":        "{{sentiment}}",
				"priority":         "high",
			},
			"output_key": "escalation_ticket",
		},
	)
	if err != nil {
		log.Fatalf("Failed to create nodeEscalate: %v", err)
	}

	// Node 9: Generate context for response
	nodeGenerateContext, err := mbflow.NewNode(
		uuid.New(),
		workflowID,
		mbflow.NodeTypeOpenAICompletion,
		"Generate Response Context",
		map[string]any{
			"model": "gpt-4",
			"prompt": `Based on the customer inquiry and available information, create a structured context for generating a response:

Inquiry Type: {{inquiry_type}}
Customer Message: {{customer_message}}
Customer Info: {{customer_info}}
Sentiment: {{sentiment}}
Account Status: {{account_status}}

Generate a JSON with:
{
  "key_points": ["point1", "point2"],
  "tone": "empathetic/professional/friendly",
  "suggested_solutions": ["solution1", "solution2"],
  "additional_info_needed": ["info1"] or []
}`,
			"max_tokens":  500,
			"temperature": 0.3,
			"output_key":  "response_context",
		},
	)
	if err != nil {
		log.Fatalf("Failed to create nodeGenerateContext: %v", err)
	}

	// Node 10: Generate initial response
	nodeGenerateResponse, err := mbflow.NewNode(
		uuid.New(),
		workflowID,
		mbflow.NodeTypeOpenAICompletion,
		"Generate Customer Response",
		map[string]any{
			"model": "gpt-4",
			"prompt": `Generate a helpful, empathetic customer support response:

Customer Message: {{customer_message}}
Inquiry Type: {{inquiry_type}}
Sentiment: {{sentiment}}
Context: {{response_context}}

Requirements:
- Address the customer by name if available
- Acknowledge their concern
- Provide clear, actionable solutions
- Match the tone to the sentiment
- Be concise but thorough
- End with an offer for further assistance`,
			"max_tokens":  800,
			"temperature": 0.7,
			"output_key":  "generated_response",
		},
	)
	if err != nil {
		log.Fatalf("Failed to create nodeGenerateResponse: %v", err)
	}

	// Node 11: Quality check response
	nodeQualityCheck, err := mbflow.NewNode(
		uuid.New(),
		workflowID,
		mbflow.NodeTypeOpenAICompletion,
		"Quality Check Response",
		map[string]any{
			"model": "gpt-4",
			"prompt": `Evaluate this customer support response:

Original Inquiry: {{customer_message}}
Generated Response: {{generated_response}}

Rate the response quality (1-10) based on:
- Relevance to the inquiry
- Clarity and helpfulness
- Tone appropriateness
- Completeness

Respond with JSON:
{
  "score": <1-10>,
  "issues": ["issue1", "issue2"] or [],
  "pass": true/false (pass if score >= 7)
}`,
			"max_tokens":  200,
			"temperature": 0.1,
			"output_key":  "quality_score",
		},
	)
	if err != nil {
		log.Fatalf("Failed to create nodeQualityCheck: %v", err)
	}

	// Node 11.5: Parse quality score JSON
	nodeParseQuality, err := mbflow.NewNode(
		uuid.New(),
		workflowID,
		mbflow.NodeTypeJSONParser,
		"Parse Quality Score JSON",
		map[string]any{
			"input_key": "quality_score",
			// output_key defaults to input_key, so quality_score will be overwritten with parsed object
		},
	)
	if err != nil {
		log.Fatalf("Failed to create nodeParseQuality: %v", err)
	}

	// Node 12: Check quality score
	nodeCheckQuality, err := mbflow.NewNode(
		uuid.New(),
		workflowID,
		mbflow.NodeTypeConditionalRouter,
		"Check Quality Score",
		map[string]any{
			"input_key": "quality_score.pass",
			"routes": map[string]string{
				"true":  "personalize_response",
				"false": "regenerate_response",
			},
		},
	)
	if err != nil {
		log.Fatalf("Failed to create nodeCheckQuality: %v", err)
	}

	// Node 13: Regenerate response with feedback
	nodeRegenerateResponse, err := mbflow.NewNode(
		uuid.New(),
		workflowID,
		mbflow.NodeTypeOpenAICompletion,
		"Regenerate Response with Feedback",
		map[string]any{
			"model": "gpt-4",
			"prompt": `The previous response had quality issues. Generate an improved version:

Customer Message: {{customer_message}}
Previous Response: {{generated_response}}
Issues Found: {{quality_score.issues}}
Context: {{response_context}}

Generate a better response addressing the identified issues.`,
			"max_tokens":  800,
			"temperature": 0.6,
			"output_key":  "regenerated_response",
		},
	)
	if err != nil {
		log.Fatalf("Failed to create nodeRegenerateResponse: %v", err)
	}

	// Node 14: Merge responses
	nodeMergeResponses, err := mbflow.NewNode(
		uuid.New(),
		workflowID,
		mbflow.NodeTypeDataMerger,
		"Merge Responses",
		map[string]any{
			"strategy":   "select_first_available",
			"sources":    []string{"regenerated_response", "generated_response"},
			"output_key": "final_response_draft",
		},
	)
	if err != nil {
		log.Fatalf("Failed to create nodeMergeResponses: %v", err)
	}

	// Node 15: Personalize response
	nodePersonalizeResponse, err := mbflow.NewNode(
		uuid.New(),
		workflowID,
		mbflow.NodeTypeOpenAICompletion,
		"Personalize Response",
		map[string]any{
			"model": "gpt-4",
			"prompt": `Add personalization to this response:

Response: {{final_response_draft}}
Customer Info: {{customer_info}}
Previous Interactions: {{customer_history}}

Add:
- Personal greeting with name
- Reference to order/product if applicable
- Personalized sign-off
- Agent name and contact info`,
			"max_tokens":  1000,
			"temperature": 0.5,
			"output_key":  "final_response",
		},
	)
	if err != nil {
		log.Fatalf("Failed to create nodePersonalizeResponse: %v", err)
	}

	// Node 16: Generate follow-up suggestions
	nodeGenerateFollowUp, err := mbflow.NewNode(
		uuid.New(),
		workflowID,
		mbflow.NodeTypeOpenAICompletion,
		"Generate Follow-up Suggestions",
		map[string]any{
			"model": "gpt-4",
			"prompt": `Based on this customer interaction, suggest follow-up actions:

Inquiry: {{customer_message}}
Response: {{final_response}}
Inquiry Type: {{inquiry_type}}

Generate JSON:
{
  "schedule_follow_up": true/false,
  "follow_up_days": <number>,
  "suggested_actions": ["action1", "action2"],
  "knowledge_base_update": "suggestion for KB update or null"
}`,
			"max_tokens":  300,
			"temperature": 0.3,
			"output_key":  "follow_up_plan",
		},
	)
	if err != nil {
		log.Fatalf("Failed to create nodeGenerateFollowUp: %v", err)
	}

	// Node 17: Send response
	nodeSendResponse, err := mbflow.NewNode(
		uuid.New(),
		workflowID,
		mbflow.NodeTypeHTTPRequest,
		"Send Response to Customer",
		map[string]any{
			"url":    "http://localhost:8081/support/send",
			"method": "POST",
			"body": map[string]any{
				"customer_email": "{{customer_info.email}}",
				"subject":        "Re: {{customer_info.subject}}",
				"message":        "{{final_response}}",
				"ticket_id":      "{{ticket_id}}",
			},
		},
	)
	if err != nil {
		log.Fatalf("Failed to create nodeSendResponse: %v", err)
	}

	// Node 18: Log interaction
	nodeLogInteraction, err := mbflow.NewNode(
		uuid.New(),
		workflowID,
		mbflow.NodeTypeHTTPRequest,
		"Log Interaction",
		map[string]any{
			"url":    "http://localhost:8081/analytics/log",
			"method": "POST",
			"body": map[string]any{
				"inquiry_type":   "{{inquiry_type}}",
				"sentiment":      "{{sentiment}}",
				"escalated":      "{{escalation_decision}}",
				"quality_score":  "{{quality_score.score}}",
				"response_time":  "{{execution_time}}",
				"follow_up_plan": "{{follow_up_plan}}",
			},
		},
	)
	if err != nil {
		log.Fatalf("Failed to create nodeLogInteraction: %v", err)
	}

	// Collect all nodes
	nodes := []mbflow.Node{
		nodeExtractInfo, nodeClassifyInquiry, nodeAnalyzeSentiment,
		nodeCheckBilling, nodeFetchAccount, nodeAnalyzeAccount,
		nodeCheckEscalation, nodeEscalate, nodeGenerateContext,
		nodeGenerateResponse, nodeQualityCheck, nodeParseQuality, nodeCheckQuality,
		nodeRegenerateResponse, nodeMergeResponses, nodePersonalizeResponse,
		nodeGenerateFollowUp, nodeSendResponse, nodeLogInteraction,
	}

	// Create edges
	// Create edges using RelationshipBuilder for cleaner and more readable code
	edges := mbflow.NewRelationshipBuilder(workflowID).
		// Fork: initial processing (multiple outgoing edges from nodeExtractInfo)
		Direct(nodeExtractInfo, nodeClassifyInquiry).
		Direct(nodeExtractInfo, nodeAnalyzeSentiment).
		// Join: check billing (multiple incoming edges)
		Direct(nodeClassifyInquiry, nodeCheckBilling).
		Direct(nodeAnalyzeSentiment, nodeCheckBilling).
		// Billing path
		Conditional(nodeCheckBilling, nodeFetchAccount, "inquiry_type == 'billing'").
		Direct(nodeFetchAccount, nodeAnalyzeAccount).
		Direct(nodeAnalyzeAccount, nodeCheckEscalation).
		// Non-billing path
		Conditional(nodeCheckBilling, nodeCheckEscalation, "inquiry_type != 'billing'").
		// Escalation check
		Conditional(nodeCheckEscalation, nodeEscalate, "escalation_decision == 'escalate'").
		Conditional(nodeCheckEscalation, nodeGenerateContext, "escalation_decision != 'escalate'").
		// Response generation
		Direct(nodeGenerateContext, nodeGenerateResponse).
		Direct(nodeGenerateResponse, nodeQualityCheck).
		Direct(nodeQualityCheck, nodeParseQuality).
		Direct(nodeParseQuality, nodeCheckQuality).
		// Quality branching
		Conditional(nodeCheckQuality, nodeMergeResponses, "quality_score.pass == true").
		Conditional(nodeCheckQuality, nodeRegenerateResponse, "quality_score.pass == false").
		Direct(nodeRegenerateResponse, nodeMergeResponses).
		// Finalization
		Direct(nodeMergeResponses, nodePersonalizeResponse).
		// Fork: parallel finalization (multiple outgoing edges from nodePersonalizeResponse)
		Direct(nodePersonalizeResponse, nodeGenerateFollowUp).
		Direct(nodePersonalizeResponse, nodeSendResponse).
		// Join: logging (multiple incoming edges to nodeLogInteraction)
		Direct(nodeGenerateFollowUp, nodeLogInteraction).
		Direct(nodeSendResponse, nodeLogInteraction).
		Build()

	// Set initial variables
	initialVariables := map[string]interface{}{
		"customer_message": customerMessage,
		"customer_email":   "customer@example.com",
		"ticket_id":        "TICKET-" + uuid.NewString()[:8],
		"api_token":        "mock-api-token-12345",
	}

	fmt.Println("=== Workflow Structure ===")
	fmt.Println("1. Extract customer information (parallel)")
	fmt.Println("2. Classify inquiry type (parallel)")
	fmt.Println("3. Analyze sentiment (parallel)")
	fmt.Println("4. Routing logic:")
	fmt.Println("   - Billing inquiry → Fetch account → Analyze")
	fmt.Println("   - Other inquiries → Check escalation")
	fmt.Println("5. Escalation check (AI-powered):")
	fmt.Println("   - Technical + Negative → Escalate to human")
	fmt.Println("   - High urgency → Escalate to human")
	fmt.Println("   - Otherwise → Generate AI response")
	fmt.Println("6. Response generation:")
	fmt.Println("   - Generate context")
	fmt.Println("   - Generate response")
	fmt.Println("   - Quality check")
	fmt.Println("   - Regenerate if quality is low")
	fmt.Println("7. Personalize response")
	fmt.Println("8. Send response and generate follow-up plan (parallel)")
	fmt.Println("9. Log interaction")
	fmt.Println()

	fmt.Printf("Nodes: %d\n", len(nodes))
	fmt.Printf("Edges: %d\n\n", len(edges))

	fmt.Printf("=== Executing Workflow ===\n\n")
	startTime := time.Now()

	// Execute workflow
	state, err := executor.ExecuteWorkflow(ctx, workflowID, executionID, nodes, edges, initialVariables)

	if err != nil {
		log.Fatalf("Workflow execution failed: %v", err)
	}

	executionDuration := time.Since(startTime)

	fmt.Printf("\n=== Execution Results ===\n\n")
	fmt.Printf("Status: %s\n", state.GetStatusString())
	fmt.Printf("Execution Duration: %s\n", executionDuration)
	fmt.Printf("State Duration: %s\n\n", state.GetExecutionDuration())

	// Get all variables
	variables := state.GetAllVariables()

	// Display detailed results
	fmt.Printf("\n=== CUSTOMER SUPPORT WORKFLOW RESULTS ===\n\n")

	// Helper function to safely get string value
	getStringValue := func(key string) string {
		if val, ok := variables[key]; ok {
			return fmt.Sprintf("%v", val)
		}
		return ""
	}

	// Customer Information
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("👤 CUSTOMER INFORMATION")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	customerInfo := getStringValue("customer_info")
	if customerInfo != "" {
		fmt.Printf("\n%s\n", customerInfo)
	} else {
		fmt.Println("\n[Customer information not available]")
	}
	fmt.Println()

	// Classification & Analysis
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("📊 CLASSIFICATION & ANALYSIS")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Printf("Inquiry Type: %s\n", getStringValue("inquiry_type"))
	fmt.Printf("Sentiment: %s\n", getStringValue("sentiment"))
	fmt.Printf("Escalation Decision: %s\n", getStringValue("escalation_decision"))
	fmt.Println()

	// Account Status (if billing inquiry)
	accountStatus := getStringValue("account_status")
	if accountStatus != "" {
		fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
		fmt.Println("💳 ACCOUNT STATUS")
		fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
		fmt.Printf("\n%s\n", accountStatus)
		fmt.Println()
	}

	// Escalation (if escalated)
	escalationTicket := getStringValue("escalation_ticket")
	if escalationTicket != "" {
		fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
		fmt.Println("🚨 ESCALATION TICKET")
		fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
		fmt.Printf("\n%s\n", escalationTicket)
		fmt.Println()
	}

	// Final Response
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("💬 FINAL CUSTOMER RESPONSE")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	finalResponse := getStringValue("final_response")
	if finalResponse != "" {
		fmt.Printf("\n%s\n", finalResponse)
		fmt.Printf("\n[Length: %d characters]\n", len(finalResponse))
	} else {
		fmt.Println("\n[Final response not available]")
	}
	fmt.Println()

	// Quality Score
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("✅ QUALITY CHECK")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	qualityScore := getStringValue("quality_score")
	if qualityScore != "" {
		fmt.Printf("\n%s\n", qualityScore)
	} else {
		fmt.Println("\n[Quality score not available]")
	}
	fmt.Println()

	// Follow-up Plan
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("📅 FOLLOW-UP PLAN")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	followUpPlan := getStringValue("follow_up_plan")
	if followUpPlan != "" {
		fmt.Printf("\n%s\n", followUpPlan)
	} else {
		fmt.Println("\n[Follow-up plan not available]")
	}
	fmt.Println()

	// Display summary of all available variables
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("📊 ALL EXECUTION VARIABLES SUMMARY")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	for key, value := range variables {
		valueStr := fmt.Sprintf("%v", value)
		if len(valueStr) > 200 {
			fmt.Printf("  %s: [%d characters]\n", key, len(valueStr))
		} else {
			fmt.Printf("  %s: %s\n", key, valueStr)
		}
	}
	fmt.Println()

	// Display metrics
	nodeIDs := []uuid.UUID{
		nodeExtractInfo.ID(),
		nodeClassifyInquiry.ID(),
		nodeAnalyzeSentiment.ID(),
		nodeCheckBilling.ID(),
		nodeFetchAccount.ID(),
		nodeAnalyzeAccount.ID(),
		nodeCheckEscalation.ID(),
		nodeEscalate.ID(),
		nodeGenerateContext.ID(),
		nodeGenerateResponse.ID(),
		nodeQualityCheck.ID(),
		nodeParseQuality.ID(),
		nodeCheckQuality.ID(),
		nodeRegenerateResponse.ID(),
		nodeMergeResponses.ID(),
		nodePersonalizeResponse.ID(),
		nodeGenerateFollowUp.ID(),
		nodeSendResponse.ID(),
		nodeLogInteraction.ID(),
	}
	mbflow.DisplayMetrics(executor.GetMetrics(), workflowID, nodeIDs, true)

	fmt.Println("\n=== Demo Complete ===")
	fmt.Println("\nNote: This workflow demonstrates:")
	fmt.Println("- AI-powered escalation decision making (OpenAI)")
	fmt.Println("- Mock server for API endpoints (localhost:8081)")
	fmt.Println("- Parallel processing (classification, sentiment, extraction)")
	fmt.Println("- Conditional routing based on inquiry type")
	fmt.Println("- Quality control with automatic regeneration")
	fmt.Println("- Complete customer support automation pipeline")
}
