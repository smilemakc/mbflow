# CLAUDE.md - AI Assistant Guide for MBFlow

This document provides guidance for AI assistants working with the MBFlow codebase.

## Project Overview

**MBFlow** is a workflow engine library written in Go that enables the creation and execution of complex workflows. The project follows Domain-Driven Design (DDD) principles with a clean architecture separating domain logic from infrastructure concerns.

## Core Purpose

MBFlow allows users to:
- Define workflows as directed graphs of nodes and edges
- Execute workflows with parallel processing capabilities
- Track execution state and events
- Support multiple storage backends (in-memory, PostgreSQL)
- Integrate with external systems via triggers and node executors

## Architecture

The project follows DDD with clear layer separation:

```
mbflow/
├── mbflow.go              # Public API interfaces
├── factory.go             # Factory functions for creating entities
├── adapter.go             # Adapters for internal implementations
├── executor.go            # Execution engine public API
├── wrappers.go            # Wrapper types for internal/external boundary
├── internal/              # Internal implementation (not exported)
│   ├── domain/            # Domain entities and logic
│   │   ├── workflow.go    # Workflow aggregate
│   │   ├── execution.go   # Execution aggregate
│   │   ├── node.go        # Node entity
│   │   ├── edge.go        # Edge entity
│   │   ├── trigger.go     # Trigger entity
│   │   ├── event.go       # Event entity
│   │   └── execution_state.go  # Execution state management
│   ├── application/       # Application services
│   │   └── executor/      # Workflow execution engine
│   │       ├── engine.go  # Main execution engine
│   │       ├── graph.go   # DAG construction and traversal
│   │       ├── state.go   # State management
│   │       └── node_executors.go  # Node type implementations
│   └── infrastructure/    # Infrastructure layer
│       ├── storage/       # Storage implementations
│       │   ├── memory.go  # In-memory storage
│       │   └── bun_store.go  # PostgreSQL storage (using Bun ORM)
│       ├── monitoring/    # Monitoring and observability
│       └── api/           # REST API server
├── cmd/                   # Executable commands
│   └── server/            # HTTP server for workflow execution
└── examples/              # Usage examples
```

## Key Domain Concepts

### 1. Workflow
- **Purpose**: Root aggregate representing a workflow definition
- **Components**: Contains nodes, edges, and triggers
- **File**: `internal/domain/workflow.go`
- **Factory**: `NewWorkflow(id, name, version, spec)`

### 2. Node
- **Purpose**: Represents a single operation/task in a workflow
- **Types**: HTTP requests, transformations, AI operations, code execution
- **File**: `internal/domain/node.go`
- **Configuration**: Each node type has specific config requirements

### 3. Edge
- **Purpose**: Defines transitions between nodes
- **Types**: Direct, conditional, parallel
- **File**: `internal/domain/edge.go`

### 4. Execution
- **Purpose**: Represents a single run of a workflow
- **State**: Pending, Running, Completed, Failed
- **File**: `internal/domain/execution.go`
- **Events**: All state changes are recorded as events

### 5. Event
- **Purpose**: Records all changes during execution
- **Types**: ExecutionStarted, NodeStarted, NodeCompleted, ExecutionCompleted
- **File**: `internal/domain/event.go`

### 6. Trigger
- **Purpose**: Initiates workflow execution
- **Types**: Manual, HTTP, scheduled (future)
- **File**: `internal/domain/trigger.go`

## Execution Engine

The execution engine (`internal/application/executor/`) is the heart of MBFlow:

- **engine.go**: Main execution coordinator
- **graph.go**: DAG construction and topological ordering
- **state.go**: Maintains execution state and variable context
- **node_executors.go**: Implements execution logic for each node type
- **retry.go**: Retry logic with exponential backoff

### Supported Node Types

1. **start**: Entry point (no operation)
2. **transform**: Data transformation using expressions
3. **http**: HTTP requests to external APIs
4. **llm**: AI/LLM integration (OpenAI)
5. **code**: Execute Go code snippets
6. **parallel**: Execute multiple branches concurrently
7. **end**: Exit point (no operation)

## Storage Layer

### Interface
- **File**: `internal/domain/repository.go`
- **Methods**: CRUD for Workflow, Node, Edge, Execution, Trigger, Event

### Implementations
1. **MemoryStorage** (`internal/infrastructure/storage/memory.go`)
   - For development and testing
   - All data in maps with mutex protection

2. **BunStorage** (`internal/infrastructure/storage/bun_store.go`)
   - Production PostgreSQL storage
   - Uses Bun ORM for database operations

## Development Guidelines

### 1. Domain Logic
- Keep domain entities pure (no external dependencies)
- Use value objects for immutable data
- Validate invariants in constructors
- Domain events for state changes

### 2. Factory Pattern
- Use factory functions in `factory.go` for entity creation
- Return public interfaces, not internal types
- Validate inputs before creating entities

### 3. Adapters
- Use adapters in `adapter.go` to convert between internal and public types
- Never expose internal types directly

### 4. Error Handling
- Domain-specific errors in `internal/domain/errors/errors.go`
- Wrap errors with context
- Use sentinel errors for known error conditions

### 5. Testing
- Unit tests for domain logic
- Integration tests for storage implementations
- Example-based tests in `examples/`
- Test naming: `TestFunctionName_Scenario_ExpectedResult`

## Common Tasks

### Adding a New Node Type

1. Define node config struct in `internal/application/executor/node_configs.go`
2. Implement executor in `internal/application/executor/node_executors.go`
3. Register in `engine.go` executor map
4. Add example in `examples/`

### Adding a New Storage Backend

1. Implement `Storage` interface from `internal/domain/repository.go`
2. Add in `internal/infrastructure/storage/`
3. Create factory function in `factory.go`
4. Add tests following `memory_test.go` pattern

### Extending the Execution Engine

1. Modify state management in `state.go` if needed
2. Update graph processing in `graph.go` for new edge types
3. Add retry logic in `retry.go` if applicable
4. Update monitoring in `internal/infrastructure/monitoring/`

## Important Patterns

### 1. Context Propagation
Always pass `context.Context` as first parameter for:
- Cancellation support
- Timeout handling
- Request-scoped values

### 2. State Management
- Execution state is immutable once created
- State transitions via events only
- Current state accessible via `State()` method

### 3. Concurrency
- Use mutexes for in-memory storage
- Database transactions for persistent storage
- Parallel execution via goroutines in executor

### 4. Variable Context
Variables flow through the workflow execution:
- Input variables at start
- Node outputs stored in context
- Template expressions using `{{.variable_name}}`

## Configuration

### Environment Variables
- `DB_URL`: PostgreSQL connection string
- `LOG_LEVEL`: Logging level (debug, info, warn, error)
- `SERVER_PORT`: HTTP server port (default: 8080)

### Workflow YAML Format
See `examples/order-processing.yaml` for reference structure:
- `id`, `name`, `version`
- `nodes[]`: Array of node definitions
- `edges[]`: Array of edge definitions
- `triggers[]`: Array of trigger definitions

## Testing Strategy

### Unit Tests
- Test domain logic in isolation
- Mock storage layer
- Test edge cases and error conditions

### Integration Tests
- Test storage implementations with real databases
- Test execution engine with sample workflows
- Verify state transitions and event recording

### Examples as Tests
- Each example in `examples/` serves as an integration test
- Run with `go run main.go`
- Verify output and state

## Common Pitfalls

1. **Don't expose internal types**: Always use adapters
2. **Don't modify domain entities after creation**: Use events for state changes
3. **Don't skip context**: Always pass context.Context
4. **Don't hardcode IDs**: Use UUID generation
5. **Don't ignore errors**: Always handle and wrap errors

## Key Files Reference

When working on specific areas, reference these files:

- **Public API**: `mbflow.go`, `factory.go`, `executor.go`
- **Domain Model**: Files in `internal/domain/`
- **Execution Logic**: `internal/application/executor/engine.go`
- **Storage**: `internal/infrastructure/storage/`
- **Examples**: `examples/basic/main.go`, `examples/ai-content-pipeline/main.go`
- **Server**: `cmd/server/main.go`

## Dependencies

- **github.com/google/uuid**: UUID generation
- **github.com/uptrace/bun**: PostgreSQL ORM
- **github.com/rs/zerolog**: Structured logging
- **github.com/expr-lang/expr**: Expression evaluation
- **github.com/sashabaranov/go-openai**: OpenAI integration

## Language and Localization

- Code and comments: English
- Documentation (README, examples): Russian
- API responses: English
- Error messages: English

## Next Steps for Development

When asked to implement features, consider:
1. Does it fit the DDD model?
2. Which layer does it belong to?
3. Does it need new domain entities or extend existing ones?
4. What tests are needed?
5. Should an example be created?

## Getting Help

- Check `README.md` for usage examples
- Review `examples/` for patterns
- See `EXECUTION_ENGINE.md` for execution details
- Look at tests for edge cases and usage patterns
