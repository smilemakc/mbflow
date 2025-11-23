# Workflow Examples

This directory contains advanced workflow examples demonstrating complex patterns with branching logic, parallel processing, and data transformation using OpenAI API.

## Examples Overview

### 1. AI Content Pipeline (`ai-content-pipeline/`)

**Complexity**: High  
**Features**: Branching logic, parallel processing, iterative refinement, quality control

A sophisticated content generation pipeline that:

- Generates content using OpenAI GPT-4
- Analyzes quality and routes based on results
- Implements iterative improvement loops
- Translates content to multiple languages in parallel
- Generates SEO metadata for each language
- Aggregates and publishes results

**Workflow Pattern**:

```
Generate Content → Analyze Quality → Route by Quality
                                    ├─ High → Continue
                                    ├─ Medium → Enhance → Continue
                                    └─ Low → Regenerate → Re-analyze (loop)
                                    
Continue → Parallel Translation (ES, FR, DE) → Generate SEO → Aggregate → Publish
```

**Key Concepts**:

- Conditional branching based on AI analysis
- Feedback loops for quality improvement
- Parallel processing for translations
- Join nodes to synchronize parallel branches

**Run**:

```bash
cd ai-content-pipeline
go run main.go
```

---

### 2. Customer Support AI (`customer-support-ai/`)

**Complexity**: Very High  
**Features**: Multi-stage classification, sentiment analysis, conditional routing, escalation logic, quality gates

An intelligent customer support automation system that:

- Extracts customer information from inquiries
- Classifies inquiry types (technical, billing, shipping, etc.)
- Analyzes customer sentiment
- Routes based on classification and sentiment
- Fetches account data for billing inquiries
- Escalates critical issues to human agents
- Generates AI responses with quality checks
- Regenerates responses if quality is insufficient
- Personalizes responses and creates follow-up plans

**Workflow Pattern**:

```
Extract Info ─┬─ Classify Inquiry ─┐
              └─ Analyze Sentiment ─┘
                      ↓
              Check Billing? ─┬─ Yes → Fetch Account → Analyze
                              └─ No → Continue
                      ↓
              Check Escalation? ─┬─ Yes → Escalate to Human
                                 └─ No → Generate Response
                      ↓
              Quality Check ─┬─ Pass → Continue
                            └─ Fail → Regenerate (loop)
                      ↓
              Personalize → Send + Generate Follow-up → Log
```

**Key Concepts**:

- Multi-criteria decision making
- Nested conditional routing
- Quality control loops
- Parallel final steps (send + follow-up)
- Data enrichment (fetching account status)

**Run**:

```bash
cd customer-support-ai
go run main.go
```

---

### 3. Data Analysis & Reporting (`data-analysis-reporting/`)

**Complexity**: Very High  
**Features**: Multi-source data collection, anomaly detection, conditional deep dive, intelligent distribution

An automated data analysis and reporting system that:

- Fetches data from multiple sources in parallel
- Validates and cleans data using AI
- Calculates statistical metrics
- Detects anomalies with AI analysis
- Performs deep dive analysis for critical anomalies
- Generates and sends alerts for critical issues
- Creates AI-powered business insights
- Generates visualizations
- Creates executive summaries and detailed reports
- Distributes reports based on severity and audience

**Workflow Pattern**:

```
Parallel Data Fetch (Sales, Customers, Marketing, Operations)
              ↓
Validate → Clean → Calculate Metrics → Detect Anomalies
                                              ↓
                              Deep Dive Needed? ─┬─ Yes → Deep Dive → Alerts
                                                 └─ No → Continue
                                              ↓
                              Generate Insights ─┬─ Viz Specs → Create Viz
                                                 └─ Executive Summary
                                              ↓
                              Generate Report → Determine Distribution
                                              ↓
                              Parallel Distribution (Execs, Teams, Dashboard)
                                              ↓
                                            Archive
```

**Key Concepts**:

- Multi-source parallel data collection
- Conditional deep dive analysis
- Dynamic distribution based on content
- Join nodes for synchronization
- Alert generation for critical issues

**Run**:

```bash
cd data-analysis-reporting
go run main.go
```

---

### 4. Code Review & Refactoring (`code-review-refactoring/`)

**Complexity**: Very High  
**Features**: Multi-dimensional analysis, automated refactoring, validation loops, quality gates

An AI-powered code review system that:

- Fetches code changes from GitHub
- Performs parallel analysis (complexity, security, test coverage)
- Generates comprehensive AI code review
- Routes based on issue severity
- Blocks merge for critical issues
- Generates refactoring plans for major issues
- Applies automated refactoring
- Validates refactored code
- Creates refactoring PRs or posts suggestions
- Generates documentation
- Updates code quality metrics

**Workflow Pattern**:

```
Fetch Changes ─┬─ Analyze Complexity ─┐
               ├─ Security Scan ──────┤
               └─ Check Test Coverage ─┘
                        ↓
                Generate Review → Route by Severity
                                    ├─ Critical → Block + Report
                                    ├─ Major → Refactoring? ─┬─ Yes → Plan → Generate → Validate
                                    │                         │              ├─ Apply → Create PR
                                    │                         │              ├─ Revise → Loop back
                                    │                         │              └─ Manual → Post Suggestions
                                    │                         └─ No → Post Comments
                                    ├─ Minor → Approve with Suggestions
                                    └─ None → Approve Directly
                                    ↓
                Generate Docs → Update Metrics + Send Notification
```

**Key Concepts**:

- Multi-dimensional parallel analysis
- Severity-based routing with multiple paths
- Automated code generation and validation
- Iterative refinement loops
- Quality gates and blocking conditions

**Run**:

```bash
cd code-review-refactoring
go run main.go
```

---

### 5. Telegram Message Executor (`telegram-message-demo/`)

**Complexity**: Low
**Features**: Simple single-node workflow, variable substitution, Telegram Bot API call

A minimal demo that:

- Loads `TELEGRAM_BOT_TOKEN` and `TELEGRAM_CHAT_ID` from the environment
- Sends a templated message using the `telegram-message` node
- Stores the Telegram API response in the execution state

**Run**:

```bash
cd telegram-message-demo
go run main.go
```
---

## Common Patterns Demonstrated

### 1. Conditional Branching

All examples use AI-powered decision making to route workflow execution:

- Quality-based routing (content pipeline)
- Severity-based routing (customer support, code review)
- Anomaly-based routing (data analysis)

### 2. Parallel Processing

Efficient execution through parallel branches:

- Multi-language translation (content pipeline)
- Multi-source data fetching (data analysis)
- Multi-dimensional code analysis (code review)

### 3. Iterative Refinement

Feedback loops for quality improvement:

- Content regeneration based on quality (content pipeline)
- Response regeneration based on quality (customer support)
- Code refactoring validation loops (code review)

### 4. Join Nodes

Synchronization of parallel branches:

- Waiting for all translations before aggregation
- Waiting for all analyses before review generation
- Waiting for all distributions before archiving

### 5. Data Transformation

Progressive data enrichment through the workflow:

- Extract → Classify → Analyze → Enrich → Process
- Fetch → Validate → Clean → Calculate → Analyze
- Fetch → Analyze → Review → Refactor → Validate

### 6. Quality Gates

Validation checkpoints that control flow:

- Quality checks with pass/fail routing
- Security scans with blocking conditions
- Validation steps with retry logic

## Node Types Used

### OpenAI Completion Nodes

- Content generation
- Classification and analysis
- Sentiment analysis
- Code review
- Refactoring suggestions
- Documentation generation

### Conditional Router Nodes

- Quality-based routing
- Severity-based routing
- Type-based routing
- Boolean decision routing

### HTTP Request Nodes

- API data fetching
- Publishing results
- Sending notifications
- Creating GitHub PRs

### Script Executor Nodes

- Data cleaning
- Metric calculation
- Complex business logic

### Data Processing Nodes

- Data merger (combining multiple sources)
- Data aggregator (collecting results)

## Edge Types Used

### Direct Edges

Simple sequential flow from one node to the next

### Conditional Edges

Routing based on data values or conditions

### Parallel Edges

Starting multiple branches simultaneously

### Join Edges

Waiting for multiple branches to complete before proceeding

## Running the Examples

All examples use the mbflow library. To run any example:

1. Ensure you have Go installed
2. Navigate to the example directory
3. Run `go run main.go`

The examples will:

- Create the workflow structure
- Save all nodes and edges
- Print a detailed summary
- Show the workflow structure
- List all nodes and edges

## Extending the Examples

These examples serve as templates for building your own complex workflows. Key considerations:

1. **Start Simple**: Begin with a linear workflow, then add branching
2. **Add Validation**: Include quality checks and validation nodes
3. **Use Parallel Processing**: Identify independent operations
4. **Implement Feedback Loops**: Add retry logic for quality control
5. **Monitor and Log**: Include logging and metrics collection
6. **Handle Errors**: Add error handling and fallback paths

## Architecture Notes

All examples follow DDD principles:

- Domain entities (Workflow, Node, Edge, Trigger)
- Repository pattern for data access
- Factory functions for object creation
- Clear separation of concerns

## Next Steps

- Implement execution engine to actually run these workflows
- Add monitoring and observability
- Implement error handling and retry logic
- Add workflow versioning
- Create visual workflow designer
- Add workflow testing framework
