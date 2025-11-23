package main

import (
	"context"
	"fmt"
	"log"

	"mbflow"

	"github.com/google/uuid"
)

// CodeReviewRefactoringWorkflow demonstrates an AI-powered code review and
// refactoring workflow with quality gates and iterative improvement.
//
// Workflow structure:
// 1. Fetch code changes from repository
// 2. Analyze code complexity and quality
// 3. Perform security scan
// 4. Generate code review using AI
// 5. Check review severity:
//   - Critical issues → Block merge, generate detailed report
//   - Major issues → Request changes, suggest refactoring
//   - Minor issues → Approve with suggestions
//
// 6. If refactoring suggested → Generate refactoring plan → Apply changes → Re-review
// 7. Generate documentation
// 8. Update code quality metrics
func main() {
	storage := mbflow.NewMemoryStorage()
	ctx := context.Background()

	workflowID := uuid.NewString()
	spec := map[string]any{
		"description": "Automated code review with AI-powered analysis, refactoring suggestions, and quality gates",
		"features":    []string{"code_analysis", "security_scan", "ai_review", "auto_refactoring", "quality_gates"},
	}
	workflow := mbflow.NewWorkflow(
		workflowID,
		"AI-Powered Code Review and Refactoring",
		"1.0.0",
		spec,
	)

	if err := storage.SaveWorkflow(ctx, workflow); err != nil {
		log.Fatalf("Failed to save workflow: %v", err)
	}

	fmt.Printf("Created workflow: %s (ID: %s)\n\n", workflow.Name(), workflow.ID())

	// Node 1: Fetch code changes
	nodeFetchChanges, err := mbflow.NewNodeFromConfig(mbflow.NodeConfig{
		ID:         uuid.NewString(),
		WorkflowID: workflowID,
		Type:       mbflow.NodeTypeHTTPRequest,
		Name:       "Fetch Code Changes",
		Config: map[string]any{
			"url":    "https://api.github.com/repos/{{repo}}/pulls/{{pr_number}}/files",
			"method": "GET",
			"headers": map[string]string{
				"Authorization": "Bearer {{github_token}}",
				"Accept":        "application/vnd.github.v3+json",
			},
			"output_key": "code_changes",
		},
	})
	if err != nil {
		log.Fatalf("Failed to create nodeFetchChanges: %v", err)
	}

	// Node 2: Fetch PR context
	nodeFetchPRContext, err := mbflow.NewNodeFromConfig(mbflow.NodeConfig{
		ID:         uuid.NewString(),
		WorkflowID: workflowID,
		Type:       mbflow.NodeTypeHTTPRequest,
		Name:       "Fetch PR Context",
		Config: map[string]any{
			"url":    "https://api.github.com/repos/{{repo}}/pulls/{{pr_number}}",
			"method": "GET",
			"headers": map[string]string{
				"Authorization": "Bearer {{github_token}}",
			},
			"output_key": "pr_context",
		},
	})
	if err != nil {
		log.Fatalf("Failed to create nodeFetchPRContext: %v", err)
	}

	// Node 3: Analyze code complexity
	nodeAnalyzeComplexity, err := mbflow.NewNodeFromConfig(mbflow.NodeConfig{
		ID:         uuid.NewString(),
		WorkflowID: workflowID,
		Type:       mbflow.NodeTypeOpenAICompletion,
		Name:       "Analyze Code Complexity",
		Config: map[string]any{
			"model": "gpt-4",
			"prompt": `Analyze the complexity of these code changes:

{{code_changes}}

Evaluate:
1. Cyclomatic complexity
2. Cognitive complexity
3. Code duplication
4. Function/method length
5. Nesting depth

Return JSON:
{
  "overall_complexity": "low/medium/high/very_high",
  "complexity_score": <1-10>,
  "issues": [
    {
      "file": "filename",
      "line": <line_number>,
      "type": "complexity_type",
      "severity": "low/medium/high",
      "description": "issue description"
    }
  ],
  "metrics": {
    "cyclomatic": <number>,
    "cognitive": <number>,
    "maintainability_index": <number>
  }
}`,
			"max_tokens":  1500,
			"temperature": 0.2,
			"output_key":  "complexity_analysis",
		},
	})
	if err != nil {
		log.Fatalf("Failed to create nodeAnalyzeComplexity: %v", err)
	}

	// Node 4: Security scan
	nodeSecurityScan, err := mbflow.NewNodeFromConfig(mbflow.NodeConfig{
		ID:         uuid.NewString(),
		WorkflowID: workflowID,
		Type:       mbflow.NodeTypeOpenAICompletion,
		Name:       "Security Vulnerability Scan",
		Config: map[string]any{
			"model": "gpt-4",
			"prompt": `Perform a security analysis on these code changes:

{{code_changes}}

Check for:
1. SQL injection vulnerabilities
2. XSS vulnerabilities
3. Authentication/authorization issues
4. Sensitive data exposure
5. Insecure dependencies
6. CSRF vulnerabilities
7. Insecure cryptography
8. Input validation issues

Return JSON:
{
  "vulnerabilities_found": true/false,
  "severity": "none/low/medium/high/critical",
  "vulnerabilities": [
    {
      "type": "vulnerability_type",
      "severity": "low/medium/high/critical",
      "file": "filename",
      "line": <line_number>,
      "description": "detailed description",
      "recommendation": "how to fix",
      "cwe_id": "CWE-XXX"
    }
  ]
}`,
			"max_tokens":  2000,
			"temperature": 0.1,
			"output_key":  "security_scan",
		},
	})
	if err != nil {
		log.Fatalf("Failed to create nodeSecurityScan: %v", err)
	}

	// Node 5: Check test coverage
	nodeCheckTestCoverage, err := mbflow.NewNodeFromConfig(mbflow.NodeConfig{
		ID:         uuid.NewString(),
		WorkflowID: workflowID,
		Type:       mbflow.NodeTypeOpenAICompletion,
		Name:       "Analyze Test Coverage",
		Config: map[string]any{
			"model": "gpt-4",
			"prompt": `Analyze test coverage for these code changes:

Code Changes: {{code_changes}}
PR Description: {{pr_context.body}}

Evaluate:
1. Are there tests for new functionality?
2. Are edge cases covered?
3. Are error paths tested?
4. Test quality and assertions

Return JSON:
{
  "has_tests": true/false,
  "coverage_adequate": true/false,
  "missing_tests": ["scenario1", "scenario2"],
  "test_quality": "poor/fair/good/excellent",
  "recommendations": ["rec1", "rec2"]
}`,
			"max_tokens":  800,
			"temperature": 0.2,
			"output_key":  "test_coverage",
		},
	})
	if err != nil {
		log.Fatalf("Failed to create nodeCheckTestCoverage: %v", err)
	}

	// Node 6: Generate comprehensive code review
	nodeGenerateReview, err := mbflow.NewNodeFromConfig(mbflow.NodeConfig{
		ID:         uuid.NewString(),
		WorkflowID: workflowID,
		Type:       mbflow.NodeTypeOpenAICompletion,
		Name:       "Generate Code Review",
		Config: map[string]any{
			"model": "gpt-4",
			"prompt": `Generate a comprehensive code review:

PR Title: {{pr_context.title}}
PR Description: {{pr_context.body}}
Code Changes: {{code_changes}}
Complexity Analysis: {{complexity_analysis}}
Security Scan: {{security_scan}}
Test Coverage: {{test_coverage}}

Provide:
1. Overall assessment
2. Code quality issues
3. Best practices violations
4. Performance concerns
5. Maintainability issues
6. Positive aspects
7. Specific line-by-line comments

Return structured JSON with:
{
  "overall_rating": "approve/request_changes/reject",
  "severity": "none/minor/major/critical",
  "summary": "brief summary",
  "strengths": ["strength1", "strength2"],
  "issues": [
    {
      "category": "category",
      "severity": "low/medium/high/critical",
      "file": "filename",
      "line": <line_number>,
      "issue": "description",
      "suggestion": "how to fix"
    }
  ],
  "refactoring_needed": true/false
}`,
			"max_tokens":  3000,
			"temperature": 0.4,
			"output_key":  "code_review",
		},
	})
	if err != nil {
		log.Fatalf("Failed to create nodeGenerateReview: %v", err)
	}

	// Node 7: Check review severity
	nodeCheckSeverity, err := mbflow.NewNodeFromConfig(mbflow.NodeConfig{
		ID:         uuid.NewString(),
		WorkflowID: workflowID,
		Type:       mbflow.NodeTypeConditionalRouter,
		Name:       "Route Based on Severity",
		Config: map[string]any{
			"input_key": "code_review.severity",
			"routes": map[string]string{
				"critical": "block_merge",
				"major":    "check_refactoring",
				"minor":    "approve_with_suggestions",
				"none":     "approve_directly",
			},
		},
	})
	if err != nil {
		log.Fatalf("Failed to create nodeCheckSeverity: %v", err)
	}

	// Node 8: Block merge and generate detailed report
	nodeBlockMerge, err := mbflow.NewNodeFromConfig(mbflow.NodeConfig{
		ID:         uuid.NewString(),
		WorkflowID: workflowID,
		Type:       mbflow.NodeTypeOpenAICompletion,
		Name:       "Generate Blocking Issues Report",
		Config: map[string]any{
			"model": "gpt-4",
			"prompt": `Generate a detailed report for critical issues that block this PR:

Code Review: {{code_review}}
Security Issues: {{security_scan.vulnerabilities}}

Create a comprehensive report explaining:
1. Why the PR is blocked
2. Critical issues that must be fixed
3. Step-by-step remediation guide
4. Resources and documentation links
5. Estimated effort to fix

Format as a professional, constructive report.`,
			"max_tokens":  2000,
			"temperature": 0.5,
			"output_key":  "blocking_report",
		},
	})
	if err != nil {
		log.Fatalf("Failed to create nodeBlockMerge: %v", err)
	}

	// Node 9: Post blocking comment
	nodePostBlockingComment, err := mbflow.NewNodeFromConfig(mbflow.NodeConfig{
		ID:         uuid.NewString(),
		WorkflowID: workflowID,
		Type:       mbflow.NodeTypeHTTPRequest,
		Name:       "Post Blocking Comment",
		Config: map[string]any{
			"url":    "https://api.github.com/repos/{{repo}}/pulls/{{pr_number}}/reviews",
			"method": "POST",
			"headers": map[string]string{
				"Authorization": "Bearer {{github_token}}",
			},
			"body": map[string]any{
				"event": "REQUEST_CHANGES",
				"body":  "{{blocking_report}}",
			},
		},
	})
	if err != nil {
		log.Fatalf("Failed to create nodePostBlockingComment: %v", err)
	}

	// Node 10: Check if refactoring needed
	nodeCheckRefactoring, err := mbflow.NewNodeFromConfig(mbflow.NodeConfig{
		ID:         uuid.NewString(),
		WorkflowID: workflowID,
		Type:       mbflow.NodeTypeConditionalRouter,
		Name:       "Check Refactoring Needed",
		Config: map[string]any{
			"input_key": "code_review.refactoring_needed",
			"routes": map[string]string{
				"true":  "generate_refactoring_plan",
				"false": "post_review_comments",
			},
		},
	})
	if err != nil {
		log.Fatalf("Failed to create nodeCheckRefactoring: %v", err)
	}

	// Node 11: Generate refactoring plan
	nodeGenerateRefactoringPlan, err := mbflow.NewNodeFromConfig(mbflow.NodeConfig{
		ID:         uuid.NewString(),
		WorkflowID: workflowID,
		Type:       mbflow.NodeTypeOpenAICompletion,
		Name:       "Generate Refactoring Plan",
		Config: map[string]any{
			"model": "gpt-4",
			"prompt": `Create a detailed refactoring plan:

Current Code: {{code_changes}}
Issues: {{code_review.issues}}
Complexity: {{complexity_analysis}}

Generate a step-by-step refactoring plan:
1. Identify refactoring opportunities
2. Prioritize changes
3. Suggest design patterns
4. Provide before/after examples
5. Estimate impact and effort

Return JSON with detailed plan.`,
			"max_tokens":  2500,
			"temperature": 0.4,
			"output_key":  "refactoring_plan",
		},
	})
	if err != nil {
		log.Fatalf("Failed to create nodeGenerateRefactoringPlan: %v", err)
	}

	// Node 12: Generate refactored code
	nodeGenerateRefactoredCode, err := mbflow.NewNodeFromConfig(mbflow.NodeConfig{
		ID:         uuid.NewString(),
		WorkflowID: workflowID,
		Type:       mbflow.NodeTypeOpenAICompletion,
		Name:       "Generate Refactored Code",
		Config: map[string]any{
			"model": "gpt-4",
			"prompt": `Apply the refactoring plan to generate improved code:

Original Code: {{code_changes}}
Refactoring Plan: {{refactoring_plan}}

Generate refactored code that:
1. Implements the refactoring plan
2. Maintains functionality
3. Improves readability and maintainability
4. Follows best practices
5. Includes comments explaining changes

Provide complete refactored files.`,
			"max_tokens":  3500,
			"temperature": 0.3,
			"output_key":  "refactored_code",
		},
	})
	if err != nil {
		log.Fatalf("Failed to create nodeGenerateRefactoredCode: %v", err)
	}

	// Node 13: Validate refactored code
	nodeValidateRefactoring, err := mbflow.NewNodeFromConfig(mbflow.NodeConfig{
		ID:         uuid.NewString(),
		WorkflowID: workflowID,
		Type:       mbflow.NodeTypeOpenAICompletion,
		Name:       "Validate Refactored Code",
		Config: map[string]any{
			"model": "gpt-4",
			"prompt": `Validate the refactored code:

Original Code: {{code_changes}}
Refactored Code: {{refactored_code}}
Refactoring Plan: {{refactoring_plan}}

Verify:
1. Functionality is preserved
2. Improvements are correctly applied
3. No new issues introduced
4. Code quality improved

Return JSON:
{
  "validation_passed": true/false,
  "improvements": ["improvement1", "improvement2"],
  "issues": ["issue1"] or [],
  "recommendation": "apply/revise/manual_review"
}`,
			"max_tokens":  1000,
			"temperature": 0.2,
			"output_key":  "refactoring_validation",
		},
	})
	if err != nil {
		log.Fatalf("Failed to create nodeValidateRefactoring: %v", err)
	}

	// Node 14: Check refactoring validation
	nodeCheckRefactoringValidation, err := mbflow.NewNodeFromConfig(mbflow.NodeConfig{
		ID:         uuid.NewString(),
		WorkflowID: workflowID,
		Type:       mbflow.NodeTypeConditionalRouter,
		Name:       "Check Refactoring Validation",
		Config: map[string]any{
			"input_key": "refactoring_validation.recommendation",
			"routes": map[string]string{
				"apply":         "create_refactoring_pr",
				"revise":        "generate_refactored_code", // Loop back
				"manual_review": "post_refactoring_suggestions",
			},
		},
	})
	if err != nil {
		log.Fatalf("Failed to create nodeCheckRefactoringValidation: %v", err)
	}

	// Node 15: Create refactoring PR
	nodeCreateRefactoringPR, err := mbflow.NewNodeFromConfig(mbflow.NodeConfig{
		ID:         uuid.NewString(),
		WorkflowID: workflowID,
		Type:       mbflow.NodeTypeHTTPRequest,
		Name:       "Create Refactoring PR",
		Config: map[string]any{
			"url":    "https://api.github.com/repos/{{repo}}/pulls",
			"method": "POST",
			"headers": map[string]string{
				"Authorization": "Bearer {{github_token}}",
			},
			"body": map[string]any{
				"title": "Refactoring: {{pr_context.title}}",
				"body": `Automated refactoring based on code review.

Original PR: #{{pr_number}}

Refactoring Plan:
{{refactoring_plan}}

Improvements:
{{refactoring_validation.improvements}}`,
				"head": "refactor/pr-{{pr_number}}",
				"base": "{{pr_context.base.ref}}",
			},
			"output_key": "refactoring_pr",
		},
	})
	if err != nil {
		log.Fatalf("Failed to create nodeCreateRefactoringPR: %v", err)
	}

	// Node 16: Post refactoring suggestions
	nodePostRefactoringSuggestions, err := mbflow.NewNodeFromConfig(mbflow.NodeConfig{
		ID:         uuid.NewString(),
		WorkflowID: workflowID,
		Type:       mbflow.NodeTypeHTTPRequest,
		Name:       "Post Refactoring Suggestions",
		Config: map[string]any{
			"url":    "https://api.github.com/repos/{{repo}}/pulls/{{pr_number}}/comments",
			"method": "POST",
			"headers": map[string]string{
				"Authorization": "Bearer {{github_token}}",
			},
			"body": map[string]any{
				"body": `## Refactoring Suggestions

{{refactoring_plan}}

### Proposed Changes
{{refactored_code}}

### Validation
{{refactoring_validation}}`,
			},
		},
	})
	if err != nil {
		log.Fatalf("Failed to create nodePostRefactoringSuggestions: %v", err)
	}

	// Node 17: Post review comments
	nodePostReviewComments, err := mbflow.NewNodeFromConfig(mbflow.NodeConfig{
		ID:         uuid.NewString(),
		WorkflowID: workflowID,
		Type:       mbflow.NodeTypeHTTPRequest,
		Name:       "Post Review Comments",
		Config: map[string]any{
			"url":    "https://api.github.com/repos/{{repo}}/pulls/{{pr_number}}/reviews",
			"method": "POST",
			"headers": map[string]string{
				"Authorization": "Bearer {{github_token}}",
			},
			"body": map[string]any{
				"event":    "COMMENT",
				"body":     "{{code_review.summary}}",
				"comments": "{{code_review.issues}}", // Line-by-line comments
			},
		},
	})
	if err != nil {
		log.Fatalf("Failed to create nodePostReviewComments: %v", err)
	}

	// Node 18: Approve with suggestions
	nodeApproveWithSuggestions, err := mbflow.NewNodeFromConfig(mbflow.NodeConfig{
		ID:         uuid.NewString(),
		WorkflowID: workflowID,
		Type:       mbflow.NodeTypeHTTPRequest,
		Name:       "Approve with Suggestions",
		Config: map[string]any{
			"url":    "https://api.github.com/repos/{{repo}}/pulls/{{pr_number}}/reviews",
			"method": "POST",
			"headers": map[string]string{
				"Authorization": "Bearer {{github_token}}",
			},
			"body": map[string]any{
				"event": "APPROVE",
				"body": `✅ Code review passed with minor suggestions.

{{code_review.summary}}

### Suggestions for improvement:
{{code_review.issues}}`,
			},
		},
	})
	if err != nil {
		log.Fatalf("Failed to create nodeApproveWithSuggestions: %v", err)
	}

	// Node 19: Approve directly
	nodeApproveDirect, err := mbflow.NewNodeFromConfig(mbflow.NodeConfig{
		ID:         uuid.NewString(),
		WorkflowID: workflowID,
		Type:       mbflow.NodeTypeHTTPRequest,
		Name:       "Approve Directly",
		Config: map[string]any{
			"url":    "https://api.github.com/repos/{{repo}}/pulls/{{pr_number}}/reviews",
			"method": "POST",
			"headers": map[string]string{
				"Authorization": "Bearer {{github_token}}",
			},
			"body": map[string]any{
				"event": "APPROVE",
				"body": `✅ Excellent code quality! No issues found.

{{code_review.summary}}

### Strengths:
{{code_review.strengths}}`,
			},
		},
	})
	if err != nil {
		log.Fatalf("Failed to create nodeApproveDirect: %v", err)
	}

	// Node 20: Generate documentation
	nodeGenerateDocumentation, err := mbflow.NewNodeFromConfig(mbflow.NodeConfig{
		ID:         uuid.NewString(),
		WorkflowID: workflowID,
		Type:       mbflow.NodeTypeOpenAICompletion,
		Name:       "Generate Documentation",
		Config: map[string]any{
			"model": "gpt-4",
			"prompt": `Generate documentation for these code changes:

Code Changes: {{code_changes}}
PR Description: {{pr_context.body}}

Generate:
1. API documentation (if applicable)
2. Usage examples
3. Migration guide (if breaking changes)
4. Updated README sections

Format as markdown.`,
			"max_tokens":  2000,
			"temperature": 0.5,
			"output_key":  "documentation",
		},
	})
	if err != nil {
		log.Fatalf("Failed to create nodeGenerateDocumentation: %v", err)
	}

	// Node 21: Update code quality metrics
	nodeUpdateMetrics, err := mbflow.NewNodeFromConfig(mbflow.NodeConfig{
		ID:         uuid.NewString(),
		WorkflowID: workflowID,
		Type:       mbflow.NodeTypeHTTPRequest,
		Name:       "Update Code Quality Metrics",
		Config: map[string]any{
			"url":    "https://api.example.com/metrics/code-quality",
			"method": "POST",
			"body": map[string]any{
				"repo":                "{{repo}}",
				"pr_number":           "{{pr_number}}",
				"complexity_score":    "{{complexity_analysis.complexity_score}}",
				"security_issues":     "{{security_scan.vulnerabilities}}",
				"test_coverage":       "{{test_coverage}}",
				"review_rating":       "{{code_review.overall_rating}}",
				"refactoring_applied": "{{refactoring_validation.validation_passed}}",
			},
		},
	})
	if err != nil {
		log.Fatalf("Failed to create nodeUpdateMetrics: %v", err)
	}

	// Node 22: Send summary notification
	nodeSendSummary, err := mbflow.NewNodeFromConfig(mbflow.NodeConfig{
		ID:         uuid.NewString(),
		WorkflowID: workflowID,
		Type:       mbflow.NodeTypeHTTPRequest,
		Name:       "Send Summary Notification",
		Config: map[string]any{
			"url":    "https://api.example.com/notifications/send",
			"method": "POST",
			"body": map[string]any{
				"channel": "slack",
				"message": `Code Review Complete for PR #{{pr_number}}

Rating: {{code_review.overall_rating}}
Severity: {{code_review.severity}}
Security Issues: {{security_scan.severity}}
Test Coverage: {{test_coverage.coverage_adequate}}

{{code_review.summary}}`,
			},
		},
	})
	if err != nil {
		log.Fatalf("Failed to create nodeSendSummary: %v", err)
	}

	// Save all nodes
	nodes := []mbflow.Node{
		nodeFetchChanges, nodeFetchPRContext, nodeAnalyzeComplexity,
		nodeSecurityScan, nodeCheckTestCoverage, nodeGenerateReview,
		nodeCheckSeverity, nodeBlockMerge, nodePostBlockingComment,
		nodeCheckRefactoring, nodeGenerateRefactoringPlan, nodeGenerateRefactoredCode,
		nodeValidateRefactoring, nodeCheckRefactoringValidation, nodeCreateRefactoringPR,
		nodePostRefactoringSuggestions, nodePostReviewComments,
		nodeApproveWithSuggestions, nodeApproveDirect,
		nodeGenerateDocumentation, nodeUpdateMetrics, nodeSendSummary,
	}

	for _, node := range nodes {
		if err := storage.SaveNode(ctx, node); err != nil {
			log.Fatalf("Failed to save node %s: %v", node.Name(), err)
		}
	}

	// Create edges
	// Create edges using RelationshipBuilder for cleaner and more readable code
	edges := mbflow.NewRelationshipBuilder(workflowID).
		// Initial parallel fetching
		Parallel(nodeFetchChanges, nodeAnalyzeComplexity).
		Parallel(nodeFetchChanges, nodeSecurityScan).
		Parallel(nodeFetchChanges, nodeCheckTestCoverage).
		Join(nodeFetchPRContext, nodeCheckTestCoverage).
		// Generate review (wait for all analyses)
		Join(nodeAnalyzeComplexity, nodeGenerateReview).
		Join(nodeSecurityScan, nodeGenerateReview).
		Join(nodeCheckTestCoverage, nodeGenerateReview).
		// Severity routing
		Direct(nodeGenerateReview, nodeCheckSeverity).
		// Critical path
		Conditional(nodeCheckSeverity, nodeBlockMerge, "severity == 'critical'").
		Direct(nodeBlockMerge, nodePostBlockingComment).
		Direct(nodePostBlockingComment, nodeGenerateDocumentation).
		// Major issues path
		Conditional(nodeCheckSeverity, nodeCheckRefactoring, "severity == 'major'").
		Conditional(nodeCheckRefactoring, nodeGenerateRefactoringPlan, "refactoring_needed == true").
		Direct(nodeGenerateRefactoringPlan, nodeGenerateRefactoredCode).
		Direct(nodeGenerateRefactoredCode, nodeValidateRefactoring).
		Direct(nodeValidateRefactoring, nodeCheckRefactoringValidation).
		// Refactoring validation routing
		Conditional(nodeCheckRefactoringValidation, nodeCreateRefactoringPR, "recommendation == 'apply'").
		Conditional(nodeCheckRefactoringValidation, nodeGenerateRefactoredCode, "recommendation == 'revise'").
		Conditional(nodeCheckRefactoringValidation, nodePostRefactoringSuggestions, "recommendation == 'manual_review'").
		// No refactoring needed
		Conditional(nodeCheckRefactoring, nodePostReviewComments, "refactoring_needed == false").
		Direct(nodeCreateRefactoringPR, nodeGenerateDocumentation).
		Direct(nodePostRefactoringSuggestions, nodeGenerateDocumentation).
		Direct(nodePostReviewComments, nodeGenerateDocumentation).
		// Minor issues path
		Conditional(nodeCheckSeverity, nodeApproveWithSuggestions, "severity == 'minor'").
		Direct(nodeApproveWithSuggestions, nodeGenerateDocumentation).
		// No issues path
		Conditional(nodeCheckSeverity, nodeApproveDirect, "severity == 'none'").
		Direct(nodeApproveDirect, nodeGenerateDocumentation).
		// Final steps
		Parallel(nodeGenerateDocumentation, nodeUpdateMetrics).
		Parallel(nodeGenerateDocumentation, nodeSendSummary).
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
			"event":   "pull_request",
			"actions": []string{"opened", "synchronize"},
			"source":  "github",
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
	fmt.Println("1. Fetch code changes and PR context")
	fmt.Println("2. Parallel Analysis:")
	fmt.Println("   - Analyze code complexity")
	fmt.Println("   - Security vulnerability scan")
	fmt.Println("   - Check test coverage")
	fmt.Println("3. Generate comprehensive AI code review")
	fmt.Println("4. Route based on severity:")
	fmt.Println("   - Critical → Block merge, post detailed report")
	fmt.Println("   - Major → Check if refactoring needed")
	fmt.Println("     - If yes → Generate plan → Generate code → Validate → Create PR or post suggestions")
	fmt.Println("     - If no → Post review comments")
	fmt.Println("   - Minor → Approve with suggestions")
	fmt.Println("   - None → Approve directly")
	fmt.Println("5. Generate documentation")
	fmt.Println("6. Update metrics and send notifications")

	fmt.Println("\n=== Trigger Configuration ===")
	fmt.Println("Type: GitHub Webhook")
	fmt.Println("Events: pull_request (opened, synchronize)")
	fmt.Println("Auto-triggers on PR creation or updates")

	// List all nodes
	savedNodes, err := storage.ListNodes(ctx, workflowID)
	if err != nil {
		log.Fatalf("Failed to list nodes: %v", err)
	}

	fmt.Printf("\n=== All Nodes (%d) ===\n", len(savedNodes))
	for i, n := range savedNodes {
		fmt.Printf("%d. %s (%s)\n", i+1, n.Name(), n.Type())
	}
}
