# Auto Output Keys Demo

This example demonstrates MBFlow's automatic output key generation system that prevents data conflicts in parallel execution.

## Problem Solved

When multiple nodes execute in parallel without explicit `output_key` configuration, there was a risk of:
- Race conditions writing to the same variable
- Data loss from conflicting writes
- Unpredictable behavior depending on execution timing

## Solution

MBFlow now automatically generates unique output keys based on **Node Name**:

```
Default key: {node_name}_output
Custom key: {custom_output_key} (if specified in config)
```

Node names are guaranteed to be unique within a workflow, making keys both predictable and collision-free.

## How It Works

### 1. Automatic Key Generation (engine.go:375)

```go
// Generate automatic output key based on node name (guaranteed unique within workflow)
outputKey := fmt.Sprintf("%s_output", node.Name())

// Allow override via config if explicitly specified
if customKey, ok := node.Config()["output_key"].(string); ok && customKey != "" {
    outputKey = customKey
}

// Store full node output under the generated/custom key
execution.SetVariable(outputKey, output, domain.ScopeExecution, uuid.Nil)
```

### 2. No Manual Storage in Executors

Executors no longer call `variables.Set()` - they only return results:

```go
// Before (WRONG):
variables.Set(cfg.OutputKey, content)
return result, nil

// After (CORRECT):
return result, nil  // Engine handles storage
```

### 3. Accessing Node Outputs

To use output from a previous node:

```go
// Option 1: Use NodeOutputs to collect from specific nodes (RECOMMENDED)
AddNodeWithConfig("data-aggregator", "combine", &DataAggregatorConfig{
    NodeOutputs: map[string]string{
        "data1": "transform1",  // Automatically becomes transform1_output
        "data2": "transform2",  // Automatically becomes transform2_output
    },
})

// Option 2: Use the full output keys with Fields mode
AddNodeWithConfig("data-aggregator", "combine", &DataAggregatorConfig{
    Fields: map[string]string{
        "result1": "transform1_output.result",  // Full path
        "result2": "transform2_output.result",
    },
})

// Option 3: Use auto-merged fields (for Transform, HTTP, JSONParser, DataAggregator)
AddNodeWithConfig("data-aggregator", "combine", &DataAggregatorConfig{
    Fields: map[string]string{
        "result1": "result",  // Auto-merged from transform1
        "result2": "result",  // Auto-merged from transform2
    },
})
```

## Benefits

✅ **Guaranteed Uniqueness**: Node names are always unique within a workflow
✅ **Predictable Keys**: {node_name}_output is easy to remember and reference
✅ **No Configuration Overhead**: No need to manually specify output keys
✅ **Clear Data Flow**: Key names match node names for explicit dependencies
✅ **Parallel-Safe**: Multiple parallel nodes never conflict
✅ **Backward Compatible**: Custom output_key still supported

## Example Workflow

```go
workflow.
    AddNodeWithConfig("transform", "node1", &TransformConfig{
        // No output_key needed!
        Transformations: map[string]string{
            "result": "42",
        },
    }).
    AddNodeWithConfig("transform", "node2", &TransformConfig{
        // No output_key needed!
        Transformations: map[string]string{
            "result": "100",
        },
    })
    // Both nodes can run in parallel without conflicts
    // Results stored in: node1_output and node2_output
```

## Migration Guide

### For Existing Workflows

No changes required! The system still supports custom `output_key`:

```go
// Old code still works
AddNodeWithConfig("transform", "node1", &TransformConfig{
    OutputKey: "custom_key",  // Explicit key still supported
    Transformations: map[string]string{
        "result": "value",
    },
})
```

### Best Practices

1. **Omit `output_key`** in most cases - let the system generate it
2. **Use NodeOutputs** in DataAggregator for explicit node dependency (see node-output-aggregation example)
3. **Use explicit keys** only when you need custom names for external access
4. **Reference outputs** using {node_name}_output pattern
5. **Leverage auto-merge** for Transform/HTTP/JSONParser/DataAggregator nodes

## Running the Example

```bash
go run examples/auto-output-keys/main.go
```

## Output

The example shows:
- Parallel execution of transform1 and transform2
- Auto-generated unique keys: `transform1_output`, `transform2_output`
- Data aggregation using auto-merged fields
- No conflicts despite parallel execution

## See Also

- **examples/node-output-aggregation** - Demonstrates using NodeOutputs to collect results from multiple nodes
