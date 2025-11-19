package main

import (
	"context"
	"fmt"
	"log"

	"mbflow"

	"github.com/google/uuid"
)

// DataAnalysisReportingWorkflow demonstrates a complex data analysis workflow
// that uses OpenAI to analyze data, generate insights, and create reports.
//
// Workflow structure:
// 1. Fetch data from multiple sources (parallel)
// 2. Validate and clean data
// 3. Perform statistical analysis
// 4. Generate insights using AI
// 5. Detect anomalies
// 6. If anomalies found → Deep dive analysis → Generate alerts
// 7. Create visualizations
// 8. Generate executive summary
// 9. Create detailed report
// 10. Distribute report based on severity
func main() {
	// storage := mbflow.NewMemoryStorage()
	storage := mbflow.NewPostgresStorage("postgres://postgres:postgres@localhost:5566/postgres?sslmode=disable")

	ctx := context.Background()

	workflowID := uuid.NewString()
	spec := map[string]any{
		"description": "Automated data analysis with AI-generated insights and intelligent reporting",
		"features":    []string{"multi_source_data", "anomaly_detection", "ai_insights", "conditional_alerts", "automated_reporting"},
	}
	workflow := mbflow.NewWorkflow(
		workflowID,
		"AI-Powered Data Analysis and Reporting",
		"1.0.0",
		spec,
	)

	if err := storage.SaveWorkflow(ctx, workflow); err != nil {
		log.Fatalf("Failed to save workflow: %v", err)
	}

	fmt.Printf("Created workflow: %s (ID: %s)\n\n", workflow.Name(), workflow.ID())

	// Node 1: Fetch sales data
	nodeFetchSales := mbflow.NewNode(
		uuid.NewString(),
		workflowID,
		"http-request",
		"Fetch Sales Data",
		map[string]any{
			"url":    "https://api.example.com/data/sales?period={{period}}",
			"method": "GET",
			"headers": map[string]string{
				"Authorization": "Bearer {{api_token}}",
			},
			"output_key": "sales_data",
		},
	)

	// Node 2: Fetch customer data
	nodeFetchCustomers := mbflow.NewNode(
		uuid.NewString(),
		workflowID,
		"http-request",
		"Fetch Customer Data",
		map[string]any{
			"url":    "https://api.example.com/data/customers?period={{period}}",
			"method": "GET",
			"headers": map[string]string{
				"Authorization": "Bearer {{api_token}}",
			},
			"output_key": "customer_data",
		},
	)

	// Node 3: Fetch marketing data
	nodeFetchMarketing := mbflow.NewNode(
		uuid.NewString(),
		workflowID,
		"http-request",
		"Fetch Marketing Data",
		map[string]any{
			"url":    "https://api.example.com/data/marketing?period={{period}}",
			"method": "GET",
			"headers": map[string]string{
				"Authorization": "Bearer {{api_token}}",
			},
			"output_key": "marketing_data",
		},
	)

	// Node 4: Fetch operational data
	nodeFetchOperations := mbflow.NewNode(
		uuid.NewString(),
		workflowID,
		"http-request",
		"Fetch Operational Data",
		map[string]any{
			"url":    "https://api.example.com/data/operations?period={{period}}",
			"method": "GET",
			"headers": map[string]string{
				"Authorization": "Bearer {{api_token}}",
			},
			"output_key": "operations_data",
		},
	)

	// Node 5: Validate data quality
	nodeValidateData := mbflow.NewNode(
		uuid.NewString(),
		workflowID,
		"openai-completion",
		"Validate Data Quality",
		map[string]any{
			"model": "gpt-4",
			"prompt": `Analyze the following datasets for quality issues:

Sales Data: {{sales_data}}
Customer Data: {{customer_data}}
Marketing Data: {{marketing_data}}
Operations Data: {{operations_data}}

Check for:
- Missing values
- Inconsistencies
- Outliers that might be errors
- Data format issues

Return JSON:
{
  "valid": true/false,
  "issues": ["issue1", "issue2"] or [],
  "severity": "low/medium/high",
  "recommendations": ["rec1", "rec2"]
}`,
			"max_tokens":  500,
			"temperature": 0.2,
			"output_key":  "data_quality",
		},
	)

	// Node 6: Clean and normalize data
	nodeCleanData := mbflow.NewNode(
		uuid.NewString(),
		workflowID,
		"script-executor",
		"Clean and Normalize Data",
		map[string]any{
			"script": `
// Clean and normalize data based on quality report
const cleanedData = {
  sales: cleanDataset(sales_data, data_quality.issues),
  customers: cleanDataset(customer_data, data_quality.issues),
  marketing: cleanDataset(marketing_data, data_quality.issues),
  operations: cleanDataset(operations_data, data_quality.issues)
};

return cleanedData;
`,
			"output_key": "cleaned_data",
		},
	)

	// Node 7: Calculate statistical metrics
	nodeCalculateMetrics := mbflow.NewNode(
		uuid.NewString(),
		workflowID,
		"script-executor",
		"Calculate Statistical Metrics",
		map[string]any{
			"script": `
const metrics = {
  sales: {
    total: sum(cleaned_data.sales),
    average: avg(cleaned_data.sales),
    growth: calculateGrowth(cleaned_data.sales),
    trend: calculateTrend(cleaned_data.sales)
  },
  customers: {
    total: count(cleaned_data.customers),
    new: countNew(cleaned_data.customers),
    churn: calculateChurn(cleaned_data.customers),
    ltv: calculateLTV(cleaned_data.customers)
  },
  marketing: {
    roi: calculateROI(cleaned_data.marketing, cleaned_data.sales),
    cac: calculateCAC(cleaned_data.marketing, cleaned_data.customers),
    conversion: calculateConversion(cleaned_data.marketing)
  }
};

return metrics;
`,
			"output_key": "metrics",
		},
	)

	// Node 8: Detect anomalies
	nodeDetectAnomalies := mbflow.NewNode(
		uuid.NewString(),
		workflowID,
		"openai-completion",
		"Detect Anomalies",
		map[string]any{
			"model": "gpt-4",
			"prompt": `Analyze these metrics for anomalies and unusual patterns:

Metrics: {{metrics}}
Historical Data: {{historical_metrics}}

Identify:
- Significant deviations from historical trends
- Unexpected correlations
- Potential issues or opportunities

Return JSON:
{
  "anomalies_found": true/false,
  "anomalies": [
    {
      "type": "spike/drop/unusual_pattern",
      "metric": "metric_name",
      "severity": "low/medium/high/critical",
      "description": "detailed description",
      "potential_causes": ["cause1", "cause2"]
    }
  ],
  "requires_deep_dive": true/false
}`,
			"max_tokens":  800,
			"temperature": 0.3,
			"output_key":  "anomaly_report",
		},
	)

	// Node 9: Check if deep dive needed
	nodeCheckDeepDive := mbflow.NewNode(
		uuid.NewString(),
		workflowID,
		"conditional-router",
		"Check Deep Dive Required",
		map[string]any{
			"input_key": "anomaly_report.requires_deep_dive",
			"routes": map[string]string{
				"true":  "deep_dive_analysis",
				"false": "generate_insights",
			},
		},
	)

	// Node 10: Deep dive analysis
	nodeDeepDiveAnalysis := mbflow.NewNode(
		uuid.NewString(),
		workflowID,
		"openai-completion",
		"Deep Dive Analysis",
		map[string]any{
			"model": "gpt-4",
			"prompt": `Perform a deep dive analysis on the detected anomalies:

Anomalies: {{anomaly_report.anomalies}}
Full Dataset: {{cleaned_data}}
Metrics: {{metrics}}

For each critical anomaly:
1. Analyze root causes
2. Assess business impact
3. Recommend immediate actions
4. Suggest preventive measures

Provide detailed analysis in JSON format.`,
			"max_tokens":  2000,
			"temperature": 0.4,
			"output_key":  "deep_dive_results",
		},
	)

	// Node 11: Generate alerts
	nodeGenerateAlerts := mbflow.NewNode(
		uuid.NewString(),
		workflowID,
		"openai-completion",
		"Generate Critical Alerts",
		map[string]any{
			"model": "gpt-4",
			"prompt": `Create urgent alerts for stakeholders based on the deep dive analysis:

Analysis: {{deep_dive_results}}
Anomalies: {{anomaly_report.anomalies}}

For each critical issue, generate:
- Alert title
- Severity level
- Impact summary
- Recommended actions
- Stakeholders to notify

Format as JSON array of alerts.`,
			"max_tokens":  1000,
			"temperature": 0.3,
			"output_key":  "alerts",
		},
	)

	// Node 12: Send critical alerts
	nodeSendAlerts := mbflow.NewNode(
		uuid.NewString(),
		workflowID,
		"http-request",
		"Send Critical Alerts",
		map[string]any{
			"url":    "https://api.example.com/alerts/send",
			"method": "POST",
			"body": map[string]any{
				"alerts":   "{{alerts}}",
				"priority": "high",
				"channel":  "email,slack",
			},
		},
	)

	// Node 13: Generate business insights
	nodeGenerateInsights := mbflow.NewNode(
		uuid.NewString(),
		workflowID,
		"openai-completion",
		"Generate Business Insights",
		map[string]any{
			"model": "gpt-4",
			"prompt": `Generate actionable business insights from this data:

Metrics: {{metrics}}
Anomalies: {{anomaly_report}}
Deep Dive: {{deep_dive_results}}
Period: {{period}}

Provide:
1. Key findings (top 5)
2. Trends and patterns
3. Opportunities identified
4. Risks and concerns
5. Strategic recommendations

Format as structured JSON.`,
			"max_tokens":  1500,
			"temperature": 0.5,
			"output_key":  "insights",
		},
	)

	// Node 14: Generate visualization specs
	nodeGenerateVizSpecs := mbflow.NewNode(
		uuid.NewString(),
		workflowID,
		"openai-completion",
		"Generate Visualization Specifications",
		map[string]any{
			"model": "gpt-4",
			"prompt": `Create visualization specifications for the data:

Metrics: {{metrics}}
Insights: {{insights}}

Generate specs for:
1. Sales trend chart
2. Customer acquisition funnel
3. Marketing ROI dashboard
4. Anomaly highlights
5. KPI scorecards

Return JSON with chart configurations (type, data, options).`,
			"max_tokens":  1000,
			"temperature": 0.3,
			"output_key":  "viz_specs",
		},
	)

	// Node 15: Create visualizations
	nodeCreateVisualizations := mbflow.NewNode(
		uuid.NewString(),
		workflowID,
		"http-request",
		"Create Visualizations",
		map[string]any{
			"url":    "https://api.example.com/charts/generate",
			"method": "POST",
			"body": map[string]any{
				"specs": "{{viz_specs}}",
				"data":  "{{cleaned_data}}",
			},
			"output_key": "visualizations",
		},
	)

	// Node 16: Generate executive summary
	nodeGenerateExecutiveSummary := mbflow.NewNode(
		uuid.NewString(),
		workflowID,
		"openai-completion",
		"Generate Executive Summary",
		map[string]any{
			"model": "gpt-4",
			"prompt": `Create a concise executive summary for C-level stakeholders:

Period: {{period}}
Key Metrics: {{metrics}}
Insights: {{insights}}
Anomalies: {{anomaly_report}}

Structure:
1. Overview (2-3 sentences)
2. Key Highlights (bullet points)
3. Critical Issues (if any)
4. Recommendations (top 3)

Keep it under 300 words, focus on business impact.`,
			"max_tokens":  500,
			"temperature": 0.6,
			"output_key":  "executive_summary",
		},
	)

	// Node 17: Generate detailed report
	nodeGenerateDetailedReport := mbflow.NewNode(
		uuid.NewString(),
		workflowID,
		"openai-completion",
		"Generate Detailed Report",
		map[string]any{
			"model": "gpt-4",
			"prompt": `Create a comprehensive analytical report:

Executive Summary: {{executive_summary}}
Metrics: {{metrics}}
Insights: {{insights}}
Anomalies: {{anomaly_report}}
Deep Dive: {{deep_dive_results}}
Visualizations: {{visualizations}}

Structure:
1. Executive Summary
2. Methodology
3. Data Overview
4. Key Findings
5. Detailed Analysis by Category
6. Anomalies and Concerns
7. Recommendations
8. Appendix

Write in professional business language.`,
			"max_tokens":  3000,
			"temperature": 0.5,
			"output_key":  "detailed_report",
		},
	)

	// Node 18: Determine distribution strategy
	nodeDetermineDistribution := mbflow.NewNode(
		uuid.NewString(),
		workflowID,
		"openai-completion",
		"Determine Distribution Strategy",
		map[string]any{
			"model": "gpt-4",
			"prompt": `Determine who should receive this report and how:

Executive Summary: {{executive_summary}}
Anomalies: {{anomaly_report}}
Severity: {{anomaly_report.severity}}

Return JSON:
{
  "executive_team": true/false,
  "department_heads": ["dept1", "dept2"],
  "analysts": true/false,
  "external_stakeholders": true/false,
  "distribution_method": "email/dashboard/presentation",
  "urgency": "immediate/daily/weekly"
}`,
			"max_tokens":  300,
			"temperature": 0.2,
			"output_key":  "distribution_plan",
		},
	)

	// Node 19: Distribute to executives
	nodeDistributeExecutives := mbflow.NewNode(
		uuid.NewString(),
		workflowID,
		"http-request",
		"Distribute to Executives",
		map[string]any{
			"url":    "https://api.example.com/reports/distribute",
			"method": "POST",
			"body": map[string]any{
				"recipients":  "executives",
				"content":     "{{executive_summary}}",
				"attachments": "{{visualizations}}",
				"format":      "pdf",
			},
		},
	)

	// Node 20: Distribute detailed report
	nodeDistributeDetailed := mbflow.NewNode(
		uuid.NewString(),
		workflowID,
		"http-request",
		"Distribute Detailed Report",
		map[string]any{
			"url":    "https://api.example.com/reports/distribute",
			"method": "POST",
			"body": map[string]any{
				"recipients":  "{{distribution_plan.department_heads}}",
				"content":     "{{detailed_report}}",
				"attachments": "{{visualizations}}",
				"format":      "pdf",
			},
		},
	)

	// Node 21: Update dashboard
	nodeUpdateDashboard := mbflow.NewNode(
		uuid.NewString(),
		workflowID,
		"http-request",
		"Update Analytics Dashboard",
		map[string]any{
			"url":    "https://api.example.com/dashboard/update",
			"method": "POST",
			"body": map[string]any{
				"metrics":        "{{metrics}}",
				"visualizations": "{{visualizations}}",
				"insights":       "{{insights}}",
				"timestamp":      "{{execution_timestamp}}",
			},
		},
	)

	// Node 22: Archive report
	nodeArchiveReport := mbflow.NewNode(
		uuid.NewString(),
		workflowID,
		"http-request",
		"Archive Report",
		map[string]any{
			"url":    "https://api.example.com/reports/archive",
			"method": "POST",
			"body": map[string]any{
				"period":            "{{period}}",
				"executive_summary": "{{executive_summary}}",
				"detailed_report":   "{{detailed_report}}",
				"metrics":           "{{metrics}}",
				"insights":          "{{insights}}",
			},
		},
	)

	// Save all nodes
	nodes := []mbflow.Node{
		nodeFetchSales, nodeFetchCustomers, nodeFetchMarketing, nodeFetchOperations,
		nodeValidateData, nodeCleanData, nodeCalculateMetrics,
		nodeDetectAnomalies, nodeCheckDeepDive, nodeDeepDiveAnalysis,
		nodeGenerateAlerts, nodeSendAlerts, nodeGenerateInsights,
		nodeGenerateVizSpecs, nodeCreateVisualizations,
		nodeGenerateExecutiveSummary, nodeGenerateDetailedReport,
		nodeDetermineDistribution, nodeDistributeExecutives,
		nodeDistributeDetailed, nodeUpdateDashboard, nodeArchiveReport,
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
		// Parallel data fetching
		{nodeFetchSales, nodeValidateData, "join", nil},
		{nodeFetchCustomers, nodeValidateData, "join", nil},
		{nodeFetchMarketing, nodeValidateData, "join", nil},
		{nodeFetchOperations, nodeValidateData, "join", nil},

		// Data processing pipeline
		{nodeValidateData, nodeCleanData, "direct", nil},
		{nodeCleanData, nodeCalculateMetrics, "direct", nil},
		{nodeCalculateMetrics, nodeDetectAnomalies, "direct", nil},
		{nodeDetectAnomalies, nodeCheckDeepDive, "direct", nil},

		// Deep dive branch
		{nodeCheckDeepDive, nodeDeepDiveAnalysis, "conditional", map[string]any{"condition": "requires_deep_dive == true"}},
		{nodeDeepDiveAnalysis, nodeGenerateAlerts, "direct", nil},
		{nodeGenerateAlerts, nodeSendAlerts, "direct", nil},
		{nodeSendAlerts, nodeGenerateInsights, "direct", nil},

		// Normal flow
		{nodeCheckDeepDive, nodeGenerateInsights, "conditional", map[string]any{"condition": "requires_deep_dive == false"}},

		// Parallel report generation
		{nodeGenerateInsights, nodeGenerateVizSpecs, "parallel", nil},
		{nodeGenerateInsights, nodeGenerateExecutiveSummary, "parallel", nil},

		// Visualization creation
		{nodeGenerateVizSpecs, nodeCreateVisualizations, "direct", nil},

		// Report generation (wait for visualizations)
		{nodeCreateVisualizations, nodeGenerateDetailedReport, "direct", nil},
		{nodeGenerateExecutiveSummary, nodeGenerateDetailedReport, "join", nil},

		// Distribution
		{nodeGenerateDetailedReport, nodeDetermineDistribution, "direct", nil},
		{nodeDetermineDistribution, nodeDistributeExecutives, "parallel", nil},
		{nodeDetermineDistribution, nodeDistributeDetailed, "parallel", nil},
		{nodeDetermineDistribution, nodeUpdateDashboard, "parallel", nil},

		// Archive (wait for all distributions)
		{nodeDistributeExecutives, nodeArchiveReport, "join", nil},
		{nodeDistributeDetailed, nodeArchiveReport, "join", nil},
		{nodeUpdateDashboard, nodeArchiveReport, "join", nil},
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
		"schedule",
		map[string]any{
			"cron":     "0 0 * * *", // Daily at midnight
			"timezone": "UTC",
			"params": map[string]any{
				"period": "yesterday",
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
	fmt.Println("1. Data Collection (parallel):")
	fmt.Println("   - Fetch sales, customer, marketing, operations data")
	fmt.Println("2. Data Quality:")
	fmt.Println("   - Validate data quality")
	fmt.Println("   - Clean and normalize data")
	fmt.Println("3. Analysis:")
	fmt.Println("   - Calculate statistical metrics")
	fmt.Println("   - Detect anomalies using AI")
	fmt.Println("4. Conditional Deep Dive:")
	fmt.Println("   - If anomalies require deep dive:")
	fmt.Println("     → Perform detailed analysis")
	fmt.Println("     → Generate critical alerts")
	fmt.Println("     → Send alerts to stakeholders")
	fmt.Println("5. Insights Generation:")
	fmt.Println("   - Generate business insights using AI")
	fmt.Println("6. Reporting (parallel):")
	fmt.Println("   - Generate visualization specs")
	fmt.Println("   - Create executive summary")
	fmt.Println("   - Create visualizations")
	fmt.Println("7. Report Assembly:")
	fmt.Println("   - Generate detailed report")
	fmt.Println("   - Determine distribution strategy")
	fmt.Println("8. Distribution (parallel):")
	fmt.Println("   - Distribute to executives")
	fmt.Println("   - Distribute detailed report")
	fmt.Println("   - Update dashboard")
	fmt.Println("9. Archive report")

	fmt.Println("\n=== Trigger Configuration ===")
	fmt.Println("Type: Scheduled (Cron)")
	fmt.Println("Schedule: Daily at midnight UTC")
	fmt.Println("Auto-runs with previous day's data")

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
