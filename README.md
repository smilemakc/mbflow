# MBFlow - Modern Workflow Orchestration Engine

[![Go Version](https://img.shields.io/badge/Go-1.21+-00ADD8?style=flat&logo=go)](https://go.dev/)
[![License](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

MBFlow is a sophisticated workflow orchestration engine written in Go that implements Domain-Driven Design (DDD) principles with Event Sourcing. It provides powerful features for building complex, reliable, and scalable workflow automation systems.

## 🌟 Key Features

### Core Capabilities

- **Event Sourcing Architecture** - Complete audit trail and state reconstruction
- **Domain-Driven Design** - Clean, maintainable, and testable codebase
- **Parallel Execution** - Automatic wave-based parallel node execution
- **Scoped Variable Handling** - Intelligent data flow with automatic collision resolution
- **Retry Mechanisms** - Configurable retry policies with exponential backoff
- **Circuit Breakers** - Fault tolerance for external service calls
- **Complex Routing** - Conditional branching and dynamic workflow paths
- **Schema Validation** - Type-safe input/output contracts for nodes

### Advanced Features

- **Multi-Parent Nodes** - Automatic namespace collision resolution
- **Expression Language** - Dynamic transformations using expr-lang
- **Template Processing** - Variable substitution in configurations
- **Join Strategies** - WaitAll, WaitAny, WaitN for synchronizing parallel branches
- **Error Strategies** - FailFast, ContinueOnError, BestEffort, RequireN
- **Real-time Monitoring** - Observer pattern for execution tracking
- **Metrics & Tracing** - Prometheus-compatible metrics and distributed tracing

## 📦 Installation

```bash
go get github.com/smilemakc/mbflow
```

## 🚀 Quick Start

### Simple Workflow Example

```go
package main

import (
    "context"
    "fmt"
    "log"

    "github.com/smilemakc/mbflow"
)

func main() {
    // Create a simple workflow
    workflow, err := mbflow.NewWorkflowBuilder("Simple Workflow", "1.0").
        WithDescription("Process and transform data").
        // Define nodes
        AddNode(string(mbflow.NodeTypeStart), "start", map[string]any{}).
        AddNodeWithConfig(string(mbflow.NodeTypeTransform), "process", &mbflow.TransformConfig{
            Transformations: map[string]string{
                "doubled": "input * 2",
                "message": `"Processed: " + string(doubled)`,
            },
        }).
        AddNode(string(mbflow.NodeTypeEnd), "end", map[string]any{
            "output_keys": []string{"doubled", "message"},
        }).
        // Connect nodes
        AddEdge("start", "process", string(mbflow.EdgeTypeDirect), nil).
        AddEdge("process", "end", string(mbflow.EdgeTypeDirect), nil).
        // Add trigger
        AddTrigger(string(mbflow.TriggerTypeManual), map[string]any{
            "name": "Manual Start",
        }).
        Build()

    if err != nil {
        log.Fatal(err)
    }

    // Create executor
    executor := mbflow.NewExecutorBuilder().
        EnableParallelExecution(10).
        EnableRetry(3).
        Build()

    // Execute workflow
    ctx := context.Background()
    execution, err := executor.ExecuteWorkflow(ctx, workflow, workflow.GetAllTriggers()[0], map[string]any{
        "input": 21.0,
    })

    if err != nil {
        log.Fatal(err)
    }

    // Get results
    vars := execution.Variables().All()
    fmt.Printf("Result: doubled=%v, message=%v\n", vars["doubled"], vars["message"])
    // Output: Result: doubled=42, message=Processed: 42
}
```

### Parallel Workflow with Collision Resolution

```go
workflow, _ := mbflow.NewWorkflowBuilder("Parallel Processing", "1.0").
    AddNode(string(mbflow.NodeTypeStart), "start", map[string]any{}).
    AddNode(string(mbflow.NodeTypeParallel), "fork", map[string]any{}).

    // Three parallel branches
    AddNodeWithConfig(string(mbflow.NodeTypeTransform), "branch1", &mbflow.TransformConfig{
        Transformations: map[string]string{"result": "value * 2"},
    }).
    AddNodeWithConfig(string(mbflow.NodeTypeTransform), "branch2", &mbflow.TransformConfig{
        Transformations: map[string]string{"result": "value * value"},
    }).
    AddNodeWithConfig(string(mbflow.NodeTypeTransform), "branch3", &mbflow.TransformConfig{
        Transformations: map[string]string{"result": "value + 100"},
    }).

    // Aggregate results with automatic namespace collision resolution
    AddNodeWithConfig(string(mbflow.NodeTypeTransform), "aggregate", &mbflow.TransformConfig{
        Transformations: map[string]string{
            // Variables automatically namespaced: branch1_result, branch2_result, branch3_result
            "sum": "branch1_result + branch2_result + branch3_result",
        },
    }).

    AddNode(string(mbflow.NodeTypeEnd), "end", map[string]any{}).

    // Connect workflow
    AddEdge("start", "fork", string(mbflow.EdgeTypeDirect), nil).
    AddEdge("fork", "branch1", string(mbflow.EdgeTypeFork), nil).
    AddEdge("fork", "branch2", string(mbflow.EdgeTypeFork), nil).
    AddEdge("fork", "branch3", string(mbflow.EdgeTypeFork), nil).
    AddEdge("branch1", "aggregate", string(mbflow.EdgeTypeJoin), map[string]any{
        "join_strategy": string(mbflow.JoinStrategyWaitAll),
    }).
    AddEdge("branch2", "aggregate", string(mbflow.EdgeTypeJoin), nil).
    AddEdge("branch3", "aggregate", string(mbflow.EdgeTypeJoin), nil).
    AddEdge("aggregate", "end", string(mbflow.EdgeTypeDirect), nil).

    AddTrigger(string(mbflow.TriggerTypeManual), map[string]any{}).
    Build()

// Execute with value=10
// branch1: 10 * 2 = 20
// branch2: 10 * 10 = 100
// branch3: 10 + 100 = 110
// sum: 20 + 100 + 110 = 230
```

## 🏗️ Architecture

### Layered Design (DDD)

```
mbflow/
├── internal/
│   ├── domain/              # Domain layer - Business logic & aggregates
│   │   ├── workflow.go      # Workflow aggregate
│   │   ├── execution.go     # Execution aggregate (event sourced)
│   │   ├── node.go          # Node entity
│   │   ├── edge.go          # Edge entity
│   │   ├── variables.go     # Variable handling
│   │   └── events.go        # Domain events
│   │
│   ├── application/         # Application layer - Use cases & orchestration
│   │   └── executor/
│   │       ├── engine.go           # Workflow execution engine
│   │       ├── variable_binder.go  # Scoped variable binding
│   │       ├── node_executors.go   # Built-in node executors
│   │       └── graph.go            # Execution graph & planning
│   │
│   └── infrastructure/      # Infrastructure layer - External concerns
│       ├── storage/         # Event store implementations
│       └── monitoring/      # Observability
│
├── mbflow.go               # Public API
└── executor.go             # Executor builder API
```

### Event Sourcing Flow

```
Command → Aggregate → Event → EventStore → State Reconstruction
   ↓                              ↓
State Change                  Observers
```

All state changes in `Execution` are captured as immutable events:

- `ExecutionStarted`
- `NodeStarted`, `NodeCompleted`, `NodeFailed`
- `VariableSet`
- `ExecutionCompleted`, `ExecutionFailed`

Events are the source of truth - state can be completely reconstructed by replaying events.

### Storage Layer

MBFlow supports multiple storage backends for persistence:

#### BunStore (PostgreSQL) - Default

BunStore is the default production-ready storage implementation using PostgreSQL with the Bun ORM.

**Setup PostgreSQL:**

```bash
# Using Docker
docker run --name mbflow-postgres \
  -e POSTGRES_PASSWORD=postgres \
  -e POSTGRES_DB=mbflow \
  -p 5432:5432 \
  -d postgres:15

# Or using Homebrew (macOS)
brew install postgresql@15
brew services start postgresql@15
createdb mbflow
```

**Configuration:**

Set the `DATABASE_DSN` environment variable:

```bash
export DATABASE_DSN="postgres://postgres:postgres@localhost:5432/mbflow?sslmode=disable"
```

Default DSN (if not set): `postgres://postgres:postgres@localhost:5432/mbflow?sslmode=disable`

**Features:**

- ✅ Automatic schema initialization
- ✅ Event sourcing support
- ✅ Transaction support
- ✅ Production-ready
- ✅ Full ACID compliance

**Starting the server:**

```bash
# With default PostgreSQL connection
go run cmd/server/main.go

# With custom DSN
DATABASE_DSN="postgres://user:pass@host:5432/dbname" go run cmd/server/main.go
```

#### MemoryStore (Development)

For development and testing, you can use the in-memory storage:

```go
import "github.com/smilemakc/mbflow/internal/infrastructure/storage"

store := storage.NewMemoryStore()
executor := mbflow.NewExecutor(mbflow.WithEventStore(store))
```

**Note:** MemoryStore is not suitable for production as data is lost on restart.

## 🔄 Scoped Variable Handling

MBFlow implements a sophisticated variable scoping system that ensures data isolation and prevents unintended side effects.

### Key Concepts

**1. Three Variable Contexts**

- **Global Variables**: Read-only context available to all nodes (from `initialValues`)
- **Scoped Variables**: Only contains outputs from direct parent nodes
- **Node Outputs**: Separately tracked per-node for precise data lineage

**2. NodeExecutionInputs**

Every node executor receives:

```go
type NodeExecutionInputs struct {
    Variables     *VariableSet  // Scoped variables from parents
    GlobalContext *VariableSet  // Read-only global context
    ParentOutputs map[UUID]*VariableSet  // Raw parent outputs
    ExecutionID   UUID
    WorkflowID    UUID
}
```

**3. Automatic Collision Resolution**

When multiple parent nodes produce the same variable name:

```
Strategy: NamespaceByParent (default)
├── branch1 → { result: 10 }
├── branch2 → { result: 20 }
└── Child receives: { branch1_result: 10, branch2_result: 20 }

Strategy: Collect
├── branch1 → { result: 10 }
├── branch2 → { result: 20 }
└── Child receives: { result: [10, 20] }

Strategy: Error
└── Fails execution if collision detected
```

### Configuration

```go
// Custom variable binding
bindingConfig := &InputBindingConfig{
    AutoBind: true,
    CollisionStrategy: CollisionStrategyNamespaceByParent,
    Mappings: map[string]string{
        "user_id": "fetch_user.id",    // Explicit mapping
        "total":   "calculate.sum",
    },
}
```

## 🎯 Node Types

### Built-in Node Types

| Type | Description | Use Case |
|------|-------------|----------|
| **Start** | Entry point | Workflow initialization |
| **End** | Exit point | Workflow completion |
| **Transform** | Data transformation | Expression-based data manipulation |
| **HTTP** | HTTP requests | API calls, webhooks |
| **ConditionalRoute** | Branching logic | Dynamic routing based on conditions |
| **Parallel** | Fork execution | Start parallel branches |
| **Join** | Synchronize | Wait for parallel branches |
| **JSONParser** | Parse JSON | Extract structured data |
| **DataAggregator** | Aggregate data | Sum, count, min, max, collect |
| **DataMerger** | Merge objects | Combine multiple data sources |
| **ScriptExecutor** | Run scripts | Execute expr-lang scripts |
| **OpenAICompletion** | AI integration | OpenAI API calls |

### Custom Node Executors

Create custom nodes by implementing the `NodeExecutor` interface:

```go
type CustomExecutor struct{}

func (e *CustomExecutor) Execute(
    ctx context.Context,
    node domain.Node,
    inputs *NodeExecutionInputs,
) (map[string]any, error) {
    // Access scoped variables
    value, _ := inputs.Variables.Get("input_key")

    // Access global context
    apiKey, _ := inputs.GlobalContext.Get("api_key")

    // Your custom logic here
    result := processData(value, apiKey)

    return map[string]any{
        "output_key": result,
    }, nil
}

// Register the executor
executor.RegisterExecutor("custom_type", &CustomExecutor{})
```

## 🔧 Configuration

### Workflow Builder

```go
workflow := mbflow.NewWorkflowBuilder("MyWorkflow", "1.0").
    WithDescription("Workflow description").
    WithMetadata(map[string]string{
        "author": "team",
        "environment": "production",
    }).
    AddNode(...).
    AddEdge(...).
    AddTrigger(...).
    Build()
```

### Executor Builder

```go
executor := mbflow.NewExecutorBuilder().
    // Parallel execution
    EnableParallelExecution(10).  // Max 10 concurrent nodes

    // Retry configuration
    EnableRetry(3).  // Max 3 retry attempts
    WithRetryPolicy(&RetryPolicy{
        MaxAttempts:  3,
        InitialDelay: time.Second,
        MaxDelay:     30 * time.Second,
        Multiplier:   2.0,
    }).

    // Circuit breaker
    EnableCircuitBreaker().
    WithCircuitBreakerConfig(&CircuitBreakerConfig{
        Threshold:     5,
        Timeout:       60 * time.Second,
        HalfOpenMax:   3,
    }).

    // Monitoring
    WithObserver(observer).
    EnableMetrics().

    // Storage
    WithEventStore(customEventStore).

    Build()
```

## 📊 Monitoring & Observability

### Observer Pattern

```go
type MyObserver struct{}

func (o *MyObserver) OnExecutionStarted(executionID uuid.UUID, workflow Workflow) {
    log.Printf("Execution %s started for workflow %s", executionID, workflow.Name())
}

func (o *MyObserver) OnNodeCompleted(executionID, nodeID uuid.UUID, output map[string]any, duration time.Duration) {
    log.Printf("Node completed in %v: %+v", duration, output)
}

// Attach observer
executor := mbflow.NewExecutorBuilder().
    WithObserver(&MyObserver{}).
    Build()
```

### Metrics Collection

```go
import "github.com/smilemakc/mbflow/internal/infrastructure/monitoring"

metricsCollector := monitoring.NewMetricsCollector()
executor := mbflow.NewExecutorBuilder().
    WithObserver(metricsCollector).
    EnableMetrics().
    Build()

// Access Prometheus metrics
metrics := metricsCollector.GetMetrics()
```

## 🧪 Testing

### Unit Testing Nodes

```go
func TestCustomNode(t *testing.T) {
    executor := &CustomExecutor{}

    // Create test inputs
    variables := domain.NewVariableSet(nil)
    variables.Set("input", 42)

    globalContext := domain.NewVariableSet(nil)
    globalContext.Set("config", "value")
    globalContext.SetReadOnly(true)

    inputs := &NodeExecutionInputs{
        Variables:     variables,
        GlobalContext: globalContext,
    }

    // Execute
    node := createTestNode("custom", nil)
    output, err := executor.Execute(context.Background(), node, inputs)

    // Assert
    require.NoError(t, err)
    assert.Equal(t, expectedValue, output["result"])
}
```

### Integration Testing

```go
func TestWorkflowExecution(t *testing.T) {
    workflow := buildTestWorkflow()
    executor := mbflow.NewExecutorBuilder().Build()

    execution, err := executor.ExecuteWorkflow(
        context.Background(),
        workflow,
        workflow.GetAllTriggers()[0],
        map[string]any{"input": "test"},
    )

    require.NoError(t, err)
    assert.Equal(t, domain.ExecutionPhaseCompleted, execution.Phase())

    // Verify results
    vars := execution.Variables().All()
    assert.Equal(t, expectedOutput, vars["output_key"])
}
```

## 📚 Examples

Explore comprehensive examples in the `/examples` directory:

- **simple-workflow** - Basic transformation workflow
- **parallel-workflow** - Parallel execution with join
- **error-handling** - Error strategies and recovery
- **ai-content-pipeline** - OpenAI integration
- **customer-support-ai** - Complex AI workflow
- **data-analysis-reporting** - Data processing pipeline

Run examples:

```bash
go run examples/simple-workflow/main.go
go run examples/parallel-workflow/main.go
```

## 🤝 Contributing

Contributions are welcome! Please follow these guidelines:

1. Fork the repository
2. Create a feature branch
3. Write tests for new functionality
4. Ensure all tests pass: `go test ./...`
5. Run linter: `go vet ./...`
6. Submit a pull request

### Development Setup

```bash
# Clone repository
git clone https://github.com/smilemakc/mbflow.git
cd mbflow

# Run tests
go test ./...

# Run with coverage
go test -cover ./...

# Build all examples
go build ./examples/...
```

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## 🙏 Acknowledgments

- Built with [expr-lang/expr](https://github.com/expr-lang/expr) for expression evaluation
- Inspired by modern workflow engines and DDD principles
- Event sourcing patterns from Domain-Driven Design community

## 📞 Support

- 📧 Email: <support@mbflow.dev>
- 🐛 Issues: [GitHub Issues](https://github.com/smilemakc/mbflow/issues)
- 💬 Discussions: [GitHub Discussions](https://github.com/smilemakc/mbflow/discussions)

---

**MBFlow** - Build reliable, scalable workflow automation systems with confidence.
