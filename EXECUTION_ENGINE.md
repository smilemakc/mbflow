# Workflow Execution Engine

The workflow execution engine provides complete runtime execution capabilities for mbflow workflows with monitoring, error handling, and retry logic.

## Features

✅ **Execution Engine Core**

- Workflow orchestration and state management
- Node-by-node execution with dependency handling
- Thread-safe execution state tracking
- Variable storage and substitution

✅ **Node Executors**

- `openai-completion` - OpenAI GPT API integration
- `http-request` - HTTP/REST API calls
- `conditional-router` - Conditional branching logic
- `data-merger` - Data merging from multiple sources
- `data-aggregator` - Data aggregation and transformation
- `script-executor` - JavaScript execution (placeholder)

✅ **Monitoring & Observability**

- Structured logging with execution traces
- Metrics collection (execution counts, durations, success rates)
- AI API usage tracking (tokens, costs, latency)
- Observer pattern for custom monitoring

✅ **Error Handling & Retry**

- Configurable retry policies with exponential backoff
- Automatic retry for transient failures
- Error categorization (retryable vs. permanent)
- Graceful error propagation

## Quick Start

### Basic Usage

```go
package main

import (
    "context"
    "fmt"
    "mbflow"
    "github.com/google/uuid"
)

func main() {
    // Create executor
    executor := mbflow.NewExecutor(&mbflow.ExecutorConfig{
        OpenAIAPIKey:     "your-api-key",
        MaxRetryAttempts: 3,
        EnableMonitoring: true,
        VerboseLogging:   false,
    })
    
    // Define workflow nodes
    nodes := []mbflow.ExecutorNodeConfig{
        {
            NodeID:   "node-1",
            NodeType: "openai-completion",
            Config: map[string]any{
                "model":      "gpt-4",
                "prompt":     "Summarize: {{input_text}}",
                "max_tokens": 100,
                "output_key": "summary",
            },
        },
    }
    
    // Execute workflow
    ctx := context.Background()
    state, err := executor.ExecuteWorkflow(
        ctx,
        uuid.NewString(), // workflowID
        uuid.NewString(), // executionID
        nodes,
        map[string]interface{}{
            "input_text": "Your text here",
        },
    )
    
    if err != nil {
        panic(err)
    }
    
    // Get results
    summary, _ := state.GetVariable("summary")
    fmt.Println("Summary:", summary)
}
```

### Running the Demo

```bash
# Without OpenAI (uses mock nodes)
cd examples/execution-demo
go run main.go

# With OpenAI
export OPENAI_API_KEY=your-key-here
cd examples/execution-demo
go run main.go
```

## Node Types

### OpenAI Completion

Executes OpenAI GPT API requests with automatic token tracking.

```go
{
    NodeID:   "ai-node",
    NodeType: "openai-completion",
    Config: map[string]any{
        "model":       "gpt-4",           // Model to use
        "prompt":      "{{variable}}",     // Prompt with variable substitution
        "max_tokens":  1000,               // Max tokens to generate
        "temperature": 0.7,                // Temperature (0-1)
        "output_key":  "result",           // Variable name for output
    },
}
```

**Output**: Stores generated text in the specified output_key variable.

### HTTP Request

Makes HTTP/REST API calls with automatic retry for network errors.

```go
{
    NodeID:   "api-node",
    NodeType: "http-request",
    Config: map[string]any{
        "url":    "https://api.example.com/data",
        "method": "POST",
        "headers": map[string]string{
            "Authorization": "Bearer {{token}}",
            "Content-Type":  "application/json",
        },
        "body": map[string]any{
            "data": "{{input_data}}",
        },
        "output_key": "api_response",
    },
}
```

**Output**: Stores response body (parsed as JSON if possible) in output_key.

### Conditional Router

Routes execution based on variable values.

```go
{
    NodeID:   "router-node",
    NodeType: "conditional-router",
    Config: map[string]any{
        "input_key": "status",
        "routes": map[string]string{
            "success": "success_handler",
            "failure": "failure_handler",
            "default": "default_handler",
        },
    },
}
```

**Output**: Returns the selected route name.

### Data Merger

Merges data from multiple sources.

```go
{
    NodeID:   "merge-node",
    NodeType: "data-merger",
    Config: map[string]any{
        "strategy": "select_first_available", // or "merge_all"
        "sources":  []string{"source1", "source2", "source3"},
        "output_key": "merged_data",
    },
}
```

**Strategies**:

- `select_first_available`: Returns first non-nil source
- `merge_all`: Merges all sources into a map

### Data Aggregator

Aggregates multiple variables into a structured output.

```go
{
    NodeID:   "aggregate-node",
    NodeType: "data-aggregator",
    Config: map[string]any{
        "fields": map[string]string{
            "user_id":   "extracted_user_id",
            "timestamp": "current_time",
            "status":    "processing_status",
        },
        "output_key": "aggregated_result",
    },
}
```

**Output**: Creates a map with specified fields.

## Variable Substitution

All string values in node configurations support variable substitution using `{{variable_name}}` syntax.

```go
// Simple substitution
"prompt": "Analyze this: {{input_text}}"

// Nested access
"url": "https://api.example.com/users/{{user.id}}/profile"

// In headers
"Authorization": "Bearer {{auth.token}}"
```

## Monitoring

### Execution Logging

The engine provides structured logging for all execution events:

```
[WorkflowEngine] Execution started: workflow=xxx execution=yyy
[WorkflowEngine] Node started: execution=yyy node=node-1 type=openai-completion
[WorkflowEngine] Node completed: execution=yyy node=node-1 type=openai-completion duration=1.2s
[WorkflowEngine] Execution completed: workflow=xxx execution=yyy duration=2.5s
```

### Metrics Collection

Access execution metrics programmatically:

```go
metrics := executor.GetMetrics()

// Workflow metrics
workflowMetrics := metrics.GetWorkflowMetrics(workflowID)
fmt.Printf("Success rate: %.2f%%\n", workflowMetrics["success_count"].(int) * 100.0 / workflowMetrics["execution_count"].(int))

// Node metrics
nodeMetrics := metrics.GetNodeMetrics("openai-completion")
fmt.Printf("Average duration: %s\n", nodeMetrics["average_duration"])

// AI usage metrics
aiMetrics := metrics.GetAIMetrics()
fmt.Printf("Total cost: $%.4f\n", aiMetrics["estimated_cost_usd"])
fmt.Printf("Total tokens: %d\n", aiMetrics["total_tokens"])
```

### Custom Observers

Implement custom monitoring by creating an observer:

```go
type MyObserver struct{}

func (o *MyObserver) OnExecutionStarted(workflowID, executionID string) {
    // Custom logic
}

func (o *MyObserver) OnNodeCompleted(executionID, nodeID, nodeType string, output interface{}, duration string) {
    // Custom logic
}

// ... implement other methods ...

// Add to executor
executor.AddObserver(&MyObserver{})
```

## Error Handling

### Retry Configuration

Configure retry behavior:

```go
executor := mbflow.NewExecutor(&mbflow.ExecutorConfig{
    MaxRetryAttempts: 5,  // Retry up to 5 times
    // ... other config
})
```

### Retry Policy

The default retry policy uses exponential backoff:

- Initial delay: 1 second
- Backoff multiplier: 2x
- Max delay: 30 seconds
- Only retries transient errors (network failures, rate limits, etc.)

### Error Types

Errors are categorized as:

- **Retryable**: Network errors, API rate limits, temporary failures
- **Non-retryable**: Configuration errors, validation errors, permanent failures

## Architecture

```
mbflow/
├── executor.go                          # Public API
├── internal/
│   ├── application/
│   │   └── executor/
│   │       ├── engine.go                # Workflow engine
│   │       ├── node_executors.go        # Node executor implementations
│   │       ├── state.go                 # Execution state management
│   │       └── retry.go                 # Retry logic
│   ├── domain/
│   │   └── errors/
│   │       └── errors.go                # Domain-specific errors
│   └── infrastructure/
│       └── monitoring/
│           ├── logger.go                # Structured logging
│           ├── metrics.go               # Metrics collection
│           └── observer.go              # Observer pattern
└── examples/
    └── execution-demo/
        └── main.go                      # Execution demo
```

## Performance

Typical execution times (without external API calls):

- Node initialization: < 1µs
- Variable substitution: < 1µs
- State updates: < 1µs
- Data merger: < 2µs
- Conditional router: < 1µs

External API calls (OpenAI, HTTP) dominate execution time.

## Limitations

1. **Sequential Execution**: Current implementation executes nodes sequentially. Parallel execution of independent nodes is not yet implemented.

2. **Script Executor**: The `script-executor` node type is a placeholder. Full JavaScript execution requires integrating a JS engine (e.g., goja, otto).

3. **Graph Traversal**: The engine doesn't yet follow the full workflow graph with edges. It executes nodes in the order provided.

4. **State Persistence**: Execution state is in-memory only. Database persistence is not yet implemented.

## Roadmap

- [ ] Parallel node execution
- [ ] Full workflow graph traversal
- [ ] State persistence to database
- [ ] JavaScript execution support
- [ ] Workflow pause/resume
- [ ] Distributed execution
- [ ] Web UI for monitoring

## Examples

See the `examples/` directory for complete examples:

- `execution-demo/` - Basic execution demo
- `ai-content-pipeline/` - Complex AI workflow (coming soon)
- `customer-support-ai/` - Customer support automation (coming soon)

## Testing

Run the execution demo:

```bash
cd examples/execution-demo
go run main.go
```

Expected output shows:

- Workflow execution with monitoring logs
- Execution results and variables
- Metrics summary
- Node-level metrics

## Contributing

The execution engine follows DDD principles:

- Domain layer: Error types, business logic
- Application layer: Execution engine, node executors
- Infrastructure layer: Monitoring, logging, external integrations

When adding new node types:

1. Implement the `NodeExecutor` interface
2. Register in `NewWorkflowEngine`
3. Add tests
4. Update documentation
