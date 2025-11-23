package main

import (
	"context"
	"fmt"
	"log"

	"mbflow"

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
	storage := mbflow.NewMemoryStorage()
	ctx := context.Background()

	workflowID := uuid.NewString()
	spec := map[string]any{
		"description": "Intelligent customer support automation with classification, sentiment analysis, and smart routing",
		"features":    []string{"classification", "sentiment_analysis", "conditional_routing", "quality_control", "escalation"},
	}
	workflow := mbflow.NewWorkflow(
		workflowID,
		"AI-Powered Customer Support Workflow",
		"1.0.0",
		spec,
	)

	if err := storage.SaveWorkflow(ctx, workflow); err != nil {
		log.Fatalf("Failed to save workflow: %v", err)
	}

	fmt.Printf("Created workflow: %s (ID: %s)\n\n", workflow.Name(), workflow.ID())

	// Node 1: Extract customer information
	nodeExtractInfo, err := mbflow.NewNodeFromConfig(mbflow.NodeConfig{
		ID:         uuid.NewString(),
		WorkflowID: workflowID,
		Type:       "openai-completion",
		Name:       "Extract Customer Information",
		Config: map[string]any{
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
	})
	if err != nil {
		log.Fatalf("Failed to create nodeExtractInfo: %v", err)
	}

	// Node 2: Classify inquiry type
	nodeClassifyInquiry, err := mbflow.NewNodeFromConfig(mbflow.NodeConfig{
		ID:         uuid.NewString(),
		WorkflowID: workflowID,
		Type:       "openai-completion",
		Name:       "Classify Inquiry Type",
		Config: map[string]any{
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
	})
	if err != nil {
		log.Fatalf("Failed to create nodeClassifyInquiry: %v", err)
	}

	// Node 3: Analyze sentiment
	nodeAnalyzeSentiment, err := mbflow.NewNodeFromConfig(mbflow.NodeConfig{
		ID:         uuid.NewString(),
		WorkflowID: workflowID,
		Type:       "openai-completion",
		Name:       "Analyze Customer Sentiment",
		Config: map[string]any{
			"model": "gpt-4",
			"prompt": `Analyze the sentiment of this customer message:

{{customer_message}}

Respond with ONLY one word: positive, neutral, or negative`,
			"max_tokens":  10,
			"temperature": 0.1,
			"output_key":  "sentiment",
		},
	})
	if err != nil {
		log.Fatalf("Failed to create nodeAnalyzeSentiment: %v", err)
	}

	// Node 4: Check if billing inquiry
	nodeCheckBilling, err := mbflow.NewNodeFromConfig(mbflow.NodeConfig{
		ID:         uuid.NewString(),
		WorkflowID: workflowID,
		Type:       "conditional-router",
		Name:       "Check if Billing Inquiry",
		Config: map[string]any{
			"input_key": "inquiry_type",
			"routes": map[string]string{
				"billing": "fetch_account",
				"default": "check_escalation",
			},
		},
	})
	if err != nil {
		log.Fatalf("Failed to create nodeCheckBilling: %v", err)
	}

	// Node 5: Fetch account status (for billing inquiries)
	nodeFetchAccount, err := mbflow.NewNodeFromConfig(mbflow.NodeConfig{
		ID:         uuid.NewString(),
		WorkflowID: workflowID,
		Type:       "http-request",
		Name:       "Fetch Account Status",
		Config: map[string]any{
			"url":    "https://api.example.com/accounts/{{customer_info.order_id}}",
			"method": "GET",
			"headers": map[string]string{
				"Authorization": "Bearer {{api_token}}",
			},
			"output_key": "account_status",
		},
	})
	if err != nil {
		log.Fatalf("Failed to create nodeFetchAccount: %v", err)
	}

	// Node 6: Analyze account status
	nodeAnalyzeAccount, err := mbflow.NewNodeFromConfig(mbflow.NodeConfig{
		ID:         uuid.NewString(),
		WorkflowID: workflowID,
		Type:       "openai-completion",
		Name:       "Analyze Account Status",
		Config: map[string]any{
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
	})
	if err != nil {
		log.Fatalf("Failed to create nodeAnalyzeAccount: %v", err)
	}

	// Node 7: Check escalation criteria
	nodeCheckEscalation, err := mbflow.NewNodeFromConfig(mbflow.NodeConfig{
		ID:         uuid.NewString(),
		WorkflowID: workflowID,
		Type:       "script-executor",
		Name:       "Check Escalation Criteria",
		Config: map[string]any{
			"script": `
// Escalate if:
// 1. Technical issue with negative sentiment
// 2. High urgency
// 3. Account needs manual review
const shouldEscalate = (
  (inquiry_type === 'technical' && sentiment === 'negative') ||
  customer_info.urgency === 'high' ||
  account_action === 'needs_manual_review'
);

return shouldEscalate ? 'escalate' : 'generate_response';
`,
			"output_key": "escalation_decision",
		},
	})
	if err != nil {
		log.Fatalf("Failed to create nodeCheckEscalation: %v", err)
	}

	// Node 8: Escalate to human agent
	nodeEscalate, err := mbflow.NewNodeFromConfig(mbflow.NodeConfig{
		ID:         uuid.NewString(),
		WorkflowID: workflowID,
		Type:       "http-request",
		Name:       "Escalate to Human Agent",
		Config: map[string]any{
			"url":    "https://api.example.com/support/escalate",
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
	})
	if err != nil {
		log.Fatalf("Failed to create nodeEscalate: %v", err)
	}

	// Node 9: Generate context for response
	nodeGenerateContext, err := mbflow.NewNodeFromConfig(mbflow.NodeConfig{
		ID:         uuid.NewString(),
		WorkflowID: workflowID,
		Type:       "openai-completion",
		Name:       "Generate Response Context",
		Config: map[string]any{
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
	})
	if err != nil {
		log.Fatalf("Failed to create nodeGenerateContext: %v", err)
	}

	// Node 10: Generate initial response
	nodeGenerateResponse, err := mbflow.NewNodeFromConfig(mbflow.NodeConfig{
		ID:         uuid.NewString(),
		WorkflowID: workflowID,
		Type:       "openai-completion",
		Name:       "Generate Customer Response",
		Config: map[string]any{
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
	})
	if err != nil {
		log.Fatalf("Failed to create nodeGenerateResponse: %v", err)
	}

	// Node 11: Quality check response
	nodeQualityCheck, err := mbflow.NewNodeFromConfig(mbflow.NodeConfig{
		ID:         uuid.NewString(),
		WorkflowID: workflowID,
		Type:       "openai-completion",
		Name:       "Quality Check Response",
		Config: map[string]any{
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
	})
	if err != nil {
		log.Fatalf("Failed to create nodeQualityCheck: %v", err)
	}

	// Node 12: Check quality score
	nodeCheckQuality, err := mbflow.NewNodeFromConfig(mbflow.NodeConfig{
		ID:         uuid.NewString(),
		WorkflowID: workflowID,
		Type:       "conditional-router",
		Name:       "Check Quality Score",
		Config: map[string]any{
			"input_key": "quality_score.pass",
			"routes": map[string]string{
				"true":  "personalize_response",
				"false": "regenerate_response",
			},
		},
	})
	if err != nil {
		log.Fatalf("Failed to create nodeCheckQuality: %v", err)
	}

	// Node 13: Regenerate response with feedback
	nodeRegenerateResponse, err := mbflow.NewNodeFromConfig(mbflow.NodeConfig{
		ID:         uuid.NewString(),
		WorkflowID: workflowID,
		Type:       "openai-completion",
		Name:       "Regenerate Response with Feedback",
		Config: map[string]any{
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
	})
	if err != nil {
		log.Fatalf("Failed to create nodeRegenerateResponse: %v", err)
	}

	// Node 14: Merge responses
	nodeMergeResponses, err := mbflow.NewNodeFromConfig(mbflow.NodeConfig{
		ID:         uuid.NewString(),
		WorkflowID: workflowID,
		Type:       "data-merger",
		Name:       "Merge Responses",
		Config: map[string]any{
			"strategy":   "select_first_available",
			"sources":    []string{"regenerated_response", "generated_response"},
			"output_key": "final_response_draft",
		},
	})
	if err != nil {
		log.Fatalf("Failed to create nodeMergeResponses: %v", err)
	}

	// Node 15: Personalize response
	nodePersonalizeResponse, err := mbflow.NewNodeFromConfig(mbflow.NodeConfig{
		ID:         uuid.NewString(),
		WorkflowID: workflowID,
		Type:       "openai-completion",
		Name:       "Personalize Response",
		Config: map[string]any{
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
	})
	if err != nil {
		log.Fatalf("Failed to create nodePersonalizeResponse: %v", err)
	}

	// Node 16: Generate follow-up suggestions
	nodeGenerateFollowUp, err := mbflow.NewNodeFromConfig(mbflow.NodeConfig{
		ID:         uuid.NewString(),
		WorkflowID: workflowID,
		Type:       "openai-completion",
		Name:       "Generate Follow-up Suggestions",
		Config: map[string]any{
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
	})
	if err != nil {
		log.Fatalf("Failed to create nodeGenerateFollowUp: %v", err)
	}

	// Node 17: Send response
	nodeSendResponse, err := mbflow.NewNodeFromConfig(mbflow.NodeConfig{
		ID:         uuid.NewString(),
		WorkflowID: workflowID,
		Type:       "http-request",
		Name:       "Send Response to Customer",
		Config: map[string]any{
			"url":    "https://api.example.com/support/send",
			"method": "POST",
			"body": map[string]any{
				"customer_email": "{{customer_info.email}}",
				"subject":        "Re: {{customer_info.subject}}",
				"message":        "{{final_response}}",
				"ticket_id":      "{{ticket_id}}",
			},
		},
	})
	if err != nil {
		log.Fatalf("Failed to create nodeSendResponse: %v", err)
	}

	// Node 18: Log interaction
	nodeLogInteraction, err := mbflow.NewNodeFromConfig(mbflow.NodeConfig{
		ID:         uuid.NewString(),
		WorkflowID: workflowID,
		Type:       "http-request",
		Name:       "Log Interaction",
		Config: map[string]any{
			"url":    "https://api.example.com/analytics/log",
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
	})
	if err != nil {
		log.Fatalf("Failed to create nodeLogInteraction: %v", err)
	}

	// Save all nodes
	nodes := []mbflow.Node{
		nodeExtractInfo, nodeClassifyInquiry, nodeAnalyzeSentiment,
		nodeCheckBilling, nodeFetchAccount, nodeAnalyzeAccount,
		nodeCheckEscalation, nodeEscalate, nodeGenerateContext,
		nodeGenerateResponse, nodeQualityCheck, nodeCheckQuality,
		nodeRegenerateResponse, nodeMergeResponses, nodePersonalizeResponse,
		nodeGenerateFollowUp, nodeSendResponse, nodeLogInteraction,
	}

	for _, node := range nodes {
		if err := storage.SaveNode(ctx, node); err != nil {
			log.Fatalf("Failed to save node %s: %v", node.Name(), err)
		}
	}

	// Create edges
	// Create edges using RelationshipBuilder for cleaner and more readable code
	edges := mbflow.NewRelationshipBuilder(workflowID).
		// Initial processing (parallel)
		Parallel(nodeExtractInfo, nodeClassifyInquiry).
		Parallel(nodeExtractInfo, nodeAnalyzeSentiment).
		// Check billing
		Direct(nodeClassifyInquiry, nodeCheckBilling).
		Join(nodeAnalyzeSentiment, nodeCheckBilling).
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
		Direct(nodeQualityCheck, nodeCheckQuality).
		// Quality branching
		Conditional(nodeCheckQuality, nodeMergeResponses, "quality_score.pass == true").
		Conditional(nodeCheckQuality, nodeRegenerateResponse, "quality_score.pass == false").
		Direct(nodeRegenerateResponse, nodeMergeResponses).
		// Finalization
		Direct(nodeMergeResponses, nodePersonalizeResponse).
		Parallel(nodePersonalizeResponse, nodeGenerateFollowUp).
		Parallel(nodePersonalizeResponse, nodeSendResponse).
		// Logging (wait for both)
		Join(nodeGenerateFollowUp, nodeLogInteraction).
		Join(nodeSendResponse, nodeLogInteraction).
		Build()

	for i, edge := range edges {
		if err := storage.SaveEdge(ctx, edge); err != nil {
			log.Fatalf("Failed to save edge %d: %v", i, err)
		}
	}

	// Create trigger
	trigger := mbflow.NewTrigger(
		uuid.NewString(),
		workflowID,
		"webhook",
		map[string]any{
			"path":   "/api/support/incoming",
			"method": "POST",
			"schema": map[string]any{
				"customer_message": "string",
				"customer_email":   "string",
				"ticket_id":        "string",
			},
		},
	)

	if err := storage.SaveTrigger(ctx, trigger); err != nil {
		log.Fatalf("Failed to save trigger: %v", err)
	}

	// Print workflow summary
	fmt.Println("=== Workflow Summary ===")
	fmt.Printf("Workflow: %s\n", workflow.Name())
	fmt.Printf("Nodes: %d\n", len(nodes))
	fmt.Printf("Edges: %d\n\n", len(edges))

	fmt.Println("=== Workflow Structure ===")
	fmt.Println("1. Extract customer information (parallel)")
	fmt.Println("2. Classify inquiry type (parallel)")
	fmt.Println("3. Analyze sentiment (parallel)")
	fmt.Println("4. Routing logic:")
	fmt.Println("   - Billing inquiry → Fetch account → Analyze")
	fmt.Println("   - Other inquiries → Check escalation")
	fmt.Println("5. Escalation check:")
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

	// List all nodes
	savedNodes, err := storage.ListNodes(ctx, workflowID)
	if err != nil {
		log.Fatalf("Failed to list nodes: %v", err)
	}

	fmt.Printf("\n=== All Nodes (%d) ===\n", len(savedNodes))
	for i, n := range savedNodes {
		fmt.Printf("%d. %s (%s)\n", i+1, n.Name(), n.Type())
	}

	// List all edges
	savedEdges, err := storage.ListEdges(ctx, workflowID)
	if err != nil {
		log.Fatalf("Failed to list edges: %v", err)
	}

	fmt.Printf("\n=== All Edges (%d) ===\n", len(savedEdges))
	for i, e := range savedEdges {
		fromNode, _ := storage.GetNode(ctx, e.FromNodeID())
		toNode, _ := storage.GetNode(ctx, e.ToNodeID())
		fmt.Printf("%d. %s → %s (%s)\n", i+1, fromNode.Name(), toNode.Name(), e.Type())
	}
}
