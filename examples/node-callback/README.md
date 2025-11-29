# Node Callback Example

This example demonstrates how to configure callbacks for nodes in a workflow. Callbacks are executed asynchronously after successful node execution and do not affect the workflow execution flow.

## Features Demonstrated

- **Node Success Callbacks**: Configure HTTP callbacks to be invoked after a node successfully completes
- **Callback Configuration**: Customize callback URL, method, headers, and timeout
- **Variable Inclusion**: Optionally include execution variables in the callback payload
- **Asynchronous Execution**: Callbacks run asynchronously and don't affect the workflow
- **Error Isolation**: Callback errors don't fail the workflow

## Running the Example

1. Set your OpenAI API key:
   ```bash
   export OPENAI_API_KEY=your-api-key-here
   ```

2. Run the example:
   ```bash
   cd examples/node-callback
   go run main.go
   ```

## How It Works

The example creates a workflow with two nodes:

1. **generate-text**: Generates a poem about a topic using OpenAI
2. **analyze-poem**: Analyzes and critiques the generated poem

Both nodes are configured with `on_success_callback` to send HTTP POST requests to a local callback server when they complete successfully.

## Callback Configuration

The callback is configured in the node config using the `on_success_callback` key:

```go
{
    ID:   "my-node",
    Type: "openai-completion",
    Config: map[string]any{
        // ... other config ...
        "on_success_callback": map[string]any{
            "url":               "http://localhost:8181/callback",
            "method":            "POST",              // Optional, default: POST
            "timeout_seconds":   10,                  // Optional, default: 30
            "include_variables": true,                // Optional, default: false
            "headers": map[string]string{             // Optional
                "X-Custom-Header": "my-value",
            },
        },
    },
}
```

## Callback Payload

The callback receives a JSON payload with the following structure:

```json
{
  "execution_id": "exec-1234567890",
  "workflow_id": "callback-demo-workflow",
  "node_id": "generate-text",
  "node_type": "openai-completion",
  "output": "... node output ...",
  "duration_ms": 1500,
  "started_at": "2024-01-15T10:30:00Z",
  "completed_at": "2024-01-15T10:30:01.5Z",
  "variables": {
    "topic": "artificial intelligence",
    "poem": "... generated poem ..."
  }
}
```

**Note**: The `variables` field is only included if `include_variables` is set to `true` in the callback configuration.

## Important Characteristics

1. **Asynchronous**: Callbacks are executed in a separate goroutine and don't block the workflow
2. **Non-blocking**: Callback execution happens after the node is marked as completed
3. **Error Isolation**: If a callback fails, the workflow continues normally
4. **Observability**: Callback execution is tracked through observers (OnNodeCallbackStarted, OnNodeCallbackCompleted)

## Use Cases

Node callbacks are useful for:

- **Logging and Auditing**: Send execution data to external logging systems
- **Notifications**: Trigger alerts or notifications when specific nodes complete
- **Data Integration**: Push results to external systems or databases
- **Monitoring**: Send metrics to monitoring platforms
- **Webhooks**: Integrate with third-party services via webhooks

## Expected Output

When you run the example, you should see:

1. The workflow executing both nodes
2. Callback requests being received by the local server
3. The generated poem and critique
4. Execution metrics

Example output:
```
Callback server listening on :8181
Starting workflow execution...

=== Callback Received ===
Node ID: generate-text
Node Type: openai-completion
Execution ID: exec-1705318200
Duration: 1234 ms
Variables included: 2
  - topic
  - poem
...

=== Callback Received ===
Node ID: analyze-poem
Node Type: openai-completion
Execution ID: exec-1705318200
Duration: 987 ms
Variables: not included
...

Workflow completed with status: completed
Execution duration: 2.5s

=== Generated Poem ===
In circuits deep and code so bright,
Where silicon dreams take flight...

=== Critique ===
This poem effectively captures the essence...
```
