# ExecuteWorkflowStandalone with Observer Example

This example demonstrates how to use observers with `ExecuteWorkflowStandalone` to monitor workflow execution in real-time.

## What This Example Shows

1. **Observer Manager**: Create and configure observer manager
2. **Built-in Observer**: Use `LoggerObserver` for structured logging
3. **Custom Observer**: Implement custom `ProgressObserver` to track execution
4. **Real-time Events**: Receive events as workflow executes
5. **Wave-based Execution**: See parallel node execution in waves

## Running the Example

```bash
# Build
go build -o ../../bin/standalone_with_observer .

# Run
../../bin/standalone_with_observer
```

Or directly:
```bash
go run main.go
```

## Expected Output

```
Starting workflow: Workflow with Observers

→ Execution started: 9174eac1-a891-4960-9894-bfe74371125b

→ Wave 0 started (2 nodes)
  ⋯ Node started: Fetch User Data
  ⋯ Node started: Fetch User Posts
  ✓ Node completed: Fetch User Data (took 477ms)
  ✓ Node completed: Fetch User Posts (took 480ms)

→ Wave 1 started (1 nodes)
  ⋯ Node started: Combine Results
  ✓ Node completed: Combine Results (took 0ms)

✓ Execution completed: 9174eac1-a891-4960-9894-bfe74371125b

✓ Execution completed successfully!
  Execution ID: 9174eac1-a891-4960-9894-bfe74371125b
  Status: completed
  Duration: 480ms

Progress Observer Stats:
  Nodes Started: 3
  Nodes Completed: 3
  Nodes Failed: 0
  Waves: 2
```

## Code Walkthrough

### 1. Create Observer Manager

```go
observerManager := observer.NewObserverManager()
```

### 2. Register Logger Observer

```go
loggerInstance := logger.New(config.LoggingConfig{
	Level:  "info",
	Format: "text",
})
loggerObserver := observer.NewLoggerObserver(
	observer.WithLoggerInstance(loggerInstance),
)
observerManager.Register(loggerObserver)
```

### 3. Create Custom Observer

```go
type ProgressObserver struct {
	NodesStarted   int
	NodesCompleted int
	NodesFailed    int
	Waves          int
}

func (p *ProgressObserver) OnEvent(ctx context.Context, event observer.Event) error {
	switch event.Type {
	case observer.EventTypeWaveStarted:
		p.Waves++
		fmt.Printf("→ Wave %d started (%d nodes)\n", *event.WaveIndex, *event.NodeCount)
	// ... handle other events
	}
	return nil
}
```

### 4. Execute with Observers

```go
opts := &engine.ExecutionOptions{
	MaxParallelism:  10,
	ObserverManager: observerManager, // Pass observer manager
}

execution, err := client.ExecuteWorkflowStandalone(ctx, workflow, input, opts)
```

## Workflow Structure

The example workflow has 3 nodes in 2 waves:

**Wave 0** (parallel):
- `fetch-user`: Fetch user data from API
- `fetch-posts`: Fetch user posts from API

**Wave 1**:
- `combine-data`: Combine results from previous nodes

This demonstrates:
- Parallel execution (wave 0 runs both nodes simultaneously)
- Sequential waves (wave 1 waits for wave 0)
- Real-time event notification for each step

## Customizing the Example

### Filter Events

Only receive specific event types:

```go
loggerObserver := observer.NewLoggerObserver(
	observer.WithLoggerInstance(loggerInstance),
	observer.WithLoggerFilter(observer.NewEventTypeFilter(
		observer.EventTypeExecutionStarted,
		observer.EventTypeNodeCompleted,
		observer.EventTypeExecutionCompleted,
	)),
)
```

### Add More Observers

Register multiple observers:

```go
// Logger
observerManager.Register(loggerObserver)

// Progress tracker
observerManager.Register(progressObserver)

// Metrics collector
observerManager.Register(metricsObserver)

// HTTP webhook
httpObserver := observer.NewHTTPCallbackObserver("https://your-webhook.com/events")
observerManager.Register(httpObserver)
```

### Different Workflow

Replace the workflow with your own:

```go
workflow := &models.Workflow{
	Name: "My Custom Workflow",
	Nodes: []*models.Node{
		{
			ID:   "step1",
			Name: "Transform Data",
			Type: "transform",
			Config: map[string]interface{}{
				"type": "template",
				"output": map[string]interface{}{
					"greeting": "Hello, {{input.name}}!",
				},
			},
		},
	},
	Edges: []*models.Edge{},
}

input := map[string]interface{}{
	"name": "World",
}
```

## Next Steps

- See [STANDALONE_WITH_OBSERVERS.md](../../docs/STANDALONE_WITH_OBSERVERS.md) for full documentation
- Explore [other examples](../)
- Learn about [observer architecture](../../internal/application/observer/README.md)