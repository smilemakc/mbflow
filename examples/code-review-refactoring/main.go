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
	nodeFetchChanges := mbflow.NewNode(
		uuid.NewString(),
		workflowID,
		"http-request",
		"Fetch Code Changes",
		map[string]any{
			"url":    "https://api.github.com/repos/{{repo}}/pulls/{{pr_number}}/files",
			"method": "GET",
			"headers": map[string]string{
				"Authorization": "Bearer {{github_token}}",
				"Accept":        "application/vnd.github.v3+json",
			},
			"output_key": "code_changes",
		},
	)

	// Node 2: Fetch PR context
	nodeFetchPRContext := mbflow.NewNode(
		uuid.NewString(),
		workflowID,
		"http-request",
		"Fetch PR Context",
		map[string]any{
			"url":    "https://api.github.com/repos/{{repo}}/pulls/{{pr_number}}",
			"method": "GET",
			"headers": map[string]string{
				"Authorization": "Bearer {{github_token}}",
			},
			"output_key": "pr_context",
		},
	)

	// Node 3: Analyze code complexity
	nodeAnalyzeComplexity := mbflow.NewNode(
		uuid.NewString(),
		workflowID,
		"openai-completion",
		"Analyze Code Complexity",
		map[string]any{
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
	)

	// Node 4: Security scan
	nodeSecurityScan := mbflow.NewNode(
		uuid.NewString(),
		workflowID,
		"openai-completion",
		"Security Vulnerability Scan",
		map[string]any{
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
	)

	// Node 5: Check test coverage
	nodeCheckTestCoverage := mbflow.NewNode(
		uuid.NewString(),
		workflowID,
		"openai-completion",
		"Analyze Test Coverage",
		map[string]any{
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
	)

	// Node 6: Generate comprehensive code review
	nodeGenerateReview := mbflow.NewNode(
		uuid.NewString(),
		workflowID,
		"openai-completion",
		"Generate Code Review",
		map[string]any{
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
	)

	// Node 7: Check review severity
	nodeCheckSeverity := mbflow.NewNode(
		uuid.NewString(),
		workflowID,
		"conditional-router",
		"Route Based on Severity",
		map[string]any{
			"input_key": "code_review.severity",
			"routes": map[string]string{
				"critical": "block_merge",
				"major":    "check_refactoring",
				"minor":    "approve_with_suggestions",
				"none":     "approve_directly",
			},
		},
	)

	// Node 8: Block merge and generate detailed report
	nodeBlockMerge := mbflow.NewNode(
		uuid.NewString(),
		workflowID,
		"openai-completion",
		"Generate Blocking Issues Report",
		map[string]any{
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
	)

	// Node 9: Post blocking comment
	nodePostBlockingComment := mbflow.NewNode(
		uuid.NewString(),
		workflowID,
		"http-request",
		"Post Blocking Comment",
		map[string]any{
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
	)

	// Node 10: Check if refactoring needed
	nodeCheckRefactoring := mbflow.NewNode(
		uuid.NewString(),
		workflowID,
		"conditional-router",
		"Check Refactoring Needed",
		map[string]any{
			"input_key": "code_review.refactoring_needed",
			"routes": map[string]string{
				"true":  "generate_refactoring_plan",
				"false": "post_review_comments",
			},
		},
	)

	// Node 11: Generate refactoring plan
	nodeGenerateRefactoringPlan := mbflow.NewNode(
		uuid.NewString(),
		workflowID,
		"openai-completion",
		"Generate Refactoring Plan",
		map[string]any{
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
	)

	// Node 12: Generate refactored code
	nodeGenerateRefactoredCode := mbflow.NewNode(
		uuid.NewString(),
		workflowID,
		"openai-completion",
		"Generate Refactored Code",
		map[string]any{
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
	)

	// Node 13: Validate refactored code
	nodeValidateRefactoring := mbflow.NewNode(
		uuid.NewString(),
		workflowID,
		"openai-completion",
		"Validate Refactored Code",
		map[string]any{
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
	)

	// Node 14: Check refactoring validation
	nodeCheckRefactoringValidation := mbflow.NewNode(
		uuid.NewString(),
		workflowID,
		"conditional-router",
		"Check Refactoring Validation",
		map[string]any{
			"input_key": "refactoring_validation.recommendation",
			"routes": map[string]string{
				"apply":         "create_refactoring_pr",
				"revise":        "generate_refactored_code", // Loop back
				"manual_review": "post_refactoring_suggestions",
			},
		},
	)

	// Node 15: Create refactoring PR
	nodeCreateRefactoringPR := mbflow.NewNode(
		uuid.NewString(),
		workflowID,
		"http-request",
		"Create Refactoring PR",
		map[string]any{
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
	)

	// Node 16: Post refactoring suggestions
	nodePostRefactoringSuggestions := mbflow.NewNode(
		uuid.NewString(),
		workflowID,
		"http-request",
		"Post Refactoring Suggestions",
		map[string]any{
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
	)

	// Node 17: Post review comments
	nodePostReviewComments := mbflow.NewNode(
		uuid.NewString(),
		workflowID,
		"http-request",
		"Post Review Comments",
		map[string]any{
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
	)

	// Node 18: Approve with suggestions
	nodeApproveWithSuggestions := mbflow.NewNode(
		uuid.NewString(),
		workflowID,
		"http-request",
		"Approve with Suggestions",
		map[string]any{
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
	)

	// Node 19: Approve directly
	nodeApproveDirect := mbflow.NewNode(
		uuid.NewString(),
		workflowID,
		"http-request",
		"Approve Directly",
		map[string]any{
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
	)

	// Node 20: Generate documentation
	nodeGenerateDocumentation := mbflow.NewNode(
		uuid.NewString(),
		workflowID,
		"openai-completion",
		"Generate Documentation",
		map[string]any{
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
	)

	// Node 21: Update code quality metrics
	nodeUpdateMetrics := mbflow.NewNode(
		uuid.NewString(),
		workflowID,
		"http-request",
		"Update Code Quality Metrics",
		map[string]any{
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
	)

	// Node 22: Send summary notification
	nodeSendSummary := mbflow.NewNode(
		uuid.NewString(),
		workflowID,
		"http-request",
		"Send Summary Notification",
		map[string]any{
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
	)

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
	edges := []struct {
		from     mbflow.Node
		to       mbflow.Node
		edgeType string
		config   map[string]any
	}{
		// Initial parallel fetching
		{nodeFetchChanges, nodeAnalyzeComplexity, "parallel", nil},
		{nodeFetchChanges, nodeSecurityScan, "parallel", nil},
		{nodeFetchChanges, nodeCheckTestCoverage, "parallel", nil},
		{nodeFetchPRContext, nodeCheckTestCoverage, "join", nil},

		// Generate review (wait for all analyses)
		{nodeAnalyzeComplexity, nodeGenerateReview, "join", nil},
		{nodeSecurityScan, nodeGenerateReview, "join", nil},
		{nodeCheckTestCoverage, nodeGenerateReview, "join", nil},

		// Severity routing
		{nodeGenerateReview, nodeCheckSeverity, "direct", nil},

		// Critical path
		{nodeCheckSeverity, nodeBlockMerge, "conditional", map[string]any{"condition": "severity == 'critical'"}},
		{nodeBlockMerge, nodePostBlockingComment, "direct", nil},
		{nodePostBlockingComment, nodeGenerateDocumentation, "direct", nil},

		// Major issues path
		{nodeCheckSeverity, nodeCheckRefactoring, "conditional", map[string]any{"condition": "severity == 'major'"}},
		{nodeCheckRefactoring, nodeGenerateRefactoringPlan, "conditional", map[string]any{"condition": "refactoring_needed == true"}},
		{nodeGenerateRefactoringPlan, nodeGenerateRefactoredCode, "direct", nil},
		{nodeGenerateRefactoredCode, nodeValidateRefactoring, "direct", nil},
		{nodeValidateRefactoring, nodeCheckRefactoringValidation, "direct", nil},

		// Refactoring validation routing
		{nodeCheckRefactoringValidation, nodeCreateRefactoringPR, "conditional", map[string]any{"condition": "recommendation == 'apply'"}},
		{nodeCheckRefactoringValidation, nodeGenerateRefactoredCode, "conditional", map[string]any{"condition": "recommendation == 'revise'"}},
		{nodeCheckRefactoringValidation, nodePostRefactoringSuggestions, "conditional", map[string]any{"condition": "recommendation == 'manual_review'"}},

		// No refactoring needed
		{nodeCheckRefactoring, nodePostReviewComments, "conditional", map[string]any{"condition": "refactoring_needed == false"}},
		{nodeCreateRefactoringPR, nodeGenerateDocumentation, "direct", nil},
		{nodePostRefactoringSuggestions, nodeGenerateDocumentation, "direct", nil},
		{nodePostReviewComments, nodeGenerateDocumentation, "direct", nil},

		// Minor issues path
		{nodeCheckSeverity, nodeApproveWithSuggestions, "conditional", map[string]any{"condition": "severity == 'minor'"}},
		{nodeApproveWithSuggestions, nodeGenerateDocumentation, "direct", nil},

		// No issues path
		{nodeCheckSeverity, nodeApproveDirect, "conditional", map[string]any{"condition": "severity == 'none'"}},
		{nodeApproveDirect, nodeGenerateDocumentation, "direct", nil},

		// Final steps
		{nodeGenerateDocumentation, nodeUpdateMetrics, "parallel", nil},
		{nodeGenerateDocumentation, nodeSendSummary, "parallel", nil},
	}

	for i, e := range edges {
		config := e.config
		if config == nil {
			config = map[string]any{}
		}

		edge := mbflow.NewEdge(
			uuid.NewString(),
			workflowID,
			e.from.ID(),
			e.to.ID(),
			e.edgeType,
			config,
		)

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
