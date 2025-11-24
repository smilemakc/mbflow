# Node Output Aggregation Demo

This example demonstrates MBFlow's ability to collect and combine outputs from multiple parallel nodes using the `NodeOutputs` configuration.

## Problem Solved

When you have **multiple parallel nodes** producing different data, you need a way to:
- Collect their outputs in one place
- Combine them into a structured result
- Not worry about auto-generated `{node_id}_output` keys

## Solution: NodeOutputs Configuration

The `DataAggregator` node now supports a third mode: **Node Output Aggregation**

```go
AddNodeWithConfig("data-aggregator", "combine", &DataAggregatorConfig{
    NodeOutputs: map[string]string{
        "user_data":  "process_user",     // Node ID (not key!)
        "order_data": "process_order",
        "api_data":   "api_call",
    },
    MergeStrategy: "separate", // or "flatten"
})
```

## How It Works

### 1. Specify Node IDs, Not Keys

You provide **node IDs** in `NodeOutputs`, not the full variable keys:

```go
NodeOutputs: map[string]string{
    "user": "process_user",  // ← Just the node ID
}
```

The executor automatically appends `_output`:
- `process_user` → `process_user_output`
- `process_order` → `process_order_output`

### 2. Choose Merge Strategy

**Strategy: "separate" (default)**
Each node's output stored under its alias:

```json
{
  "user_data": {
    "user_id": 123,
    "user_name": "John Doe",
    "role": "admin"
  },
  "order_data": {
    "order_id": 456,
    "order_total": 99.99,
    "status": "completed"
  }
}
```

**Strategy: "flatten"**
All fields merged into one level:

```json
{
  "user_id": 123,
  "user_name": "John Doe",
  "role": "admin",
  "order_id": 456,
  "order_total": 99.99,
  "status": "completed"
}
```

⚠️ **Note**: With `flatten`, later values overwrite earlier ones if keys conflict.

### 3. Automatic Dependency Resolution

The workflow engine automatically knows the aggregator depends on all nodes listed in `NodeOutputs`, so it waits for them to complete before executing.

## Example Workflow

```go
workflow.
    // Three parallel nodes
    AddNodeWithConfig("transform", "process_user", &TransformConfig{...}).
    AddNodeWithConfig("transform", "process_order", &TransformConfig{...}).
    AddNodeWithConfig("transform", "process_shipping", &TransformConfig{...}).

    // Aggregator collects all three
    AddNodeWithConfig("data-aggregator", "combine", &DataAggregatorConfig{
        NodeOutputs: map[string]string{
            "user":     "process_user",
            "order":    "process_order",
            "shipping": "process_shipping",
        },
        MergeStrategy: "separate",
    })
```

**Execution flow:**
1. `process_user`, `process_order`, `process_shipping` run **in parallel**
2. Aggregator **waits** for all three to complete
3. Collects outputs: `process_user_output`, `process_order_output`, `process_shipping_output`
4. Combines them according to `MergeStrategy`

## DataAggregator Modes

The `DataAggregator` node now has **three modes** (prioritized in this order):

### Mode 1: Node Output Aggregation (NEW!)
```go
DataAggregatorConfig{
    NodeOutputs: map[string]string{
        "alias1": "node_id_1",
        "alias2": "node_id_2",
    },
    MergeStrategy: "separate", // or "flatten"
}
```
**Use case**: Collect outputs from specific upstream nodes

### Mode 2: Field Extraction
```go
DataAggregatorConfig{
    Fields: map[string]string{
        "output_field": "variable_key",
    },
}
```
**Use case**: Extract specific fields from variables

### Mode 3: Array Aggregation
```go
DataAggregatorConfig{
    InputKey: "my_array",
    Function: "sum", // count, avg, min, max, collect
}
```
**Use case**: Aggregate array values (sum, count, etc.)

## Benefits

✅ **Explicit Dependencies**: NodeOutputs makes it clear which nodes the aggregator depends on
✅ **No Key Management**: Just specify node IDs, not full `{node_id}_output` keys
✅ **Flexible Merging**: Choose between separate and flatten strategies
✅ **Automatic Waiting**: Engine ensures all nodes complete before aggregation
✅ **Type-Safe**: Using `DataAggregatorConfig` struct prevents configuration errors

## Running the Example

```bash
go run examples/node-output-aggregation/main.go
```

## Output

The example demonstrates both merge strategies:

**Separate Strategy:**
```
combine_separate_output: {
  user_data: {user_id: 123, user_name: "John Doe", role: "admin"},
  order_data: {order_id: 456, order_total: 99.99, status: "completed"},
  shipping_data: {tracking_number: "TRACK123456", carrier: "UPS", eta: "2024-12-01"}
}
```

**Flatten Strategy:**
```
combine_flatten_output: {
  user_id: 123,
  user_name: "John Doe",
  role: "admin",
  order_id: 456,
  order_total: 99.99,
  status: "completed",
  tracking_number: "TRACK123456",
  carrier: "UPS",
  eta: "2024-12-01"
}
```

## Best Practices

1. **Use "separate"** when you need to preserve data structure and avoid key conflicts
2. **Use "flatten"** when you want a simple flat map and keys don't conflict
3. **Specify node IDs** without `_output` suffix - let the executor handle it
4. **Use explicit keys** if you've overridden `output_key` in upstream nodes
5. **Check for nil values** in your downstream nodes if some data might be missing

## Advanced: Custom Output Keys

If an upstream node uses a custom `output_key`, you can specify the full key:

```go
// Upstream node
AddNodeWithConfig("transform", "process_user", &TransformConfig{
    OutputKey: "custom_user_key", // Custom key
    Transformations: {...},
})

// Aggregator
AddNodeWithConfig("data-aggregator", "combine", &DataAggregatorConfig{
    NodeOutputs: map[string]string{
        "user": "custom_user_key", // Full key, not node ID
    },
})
```

The executor checks if the key ends with `_output` - if not, it appends it. If your custom key already has that suffix or is completely different, specify it explicitly.
