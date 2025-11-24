# Edge CRUD API

Complete API documentation for managing edges in MBFlow workflows.

## Table of Contents

- [Overview](#overview)
- [API Endpoints](#api-endpoints)
- [Edge Types](#edge-types)
- [Request/Response Formats](#requestresponse-formats)
- [Examples](#examples)
- [Best Practices](#best-practices)

## Overview

The Edge API enables dynamic management of connections between nodes in workflows. Edges define the flow of data and control through your workflow, supporting sequential execution, conditional routing, and parallel processing patterns.

**Key Features:**
- Full CRUD operations for edges
- Edge type catalog with descriptions
- Dual representation (node names for users, IDs internally)
- Immutable update pattern (delete + recreate)
- Graph visualization endpoint

**Base URL:** `http://localhost:8080/api/v1`

## API Endpoints

### Get Edge Types Catalog

Get a list of all available edge types with descriptions.

```http
GET /api/v1/edge-types
```

**Response:**
```json
[
  {
    "type": "direct",
    "name": "Direct Edge",
    "description": "Simple sequential transition from one node to another",
    "category": "flow"
  },
  {
    "type": "conditional",
    "name": "Conditional Edge",
    "description": "Transition based on condition evaluation",
    "category": "flow"
  },
  {
    "type": "fork",
    "name": "Fork Edge",
    "description": "Start parallel execution branches",
    "category": "parallel"
  },
  {
    "type": "join",
    "name": "Join Edge",
    "description": "Synchronize parallel execution branches",
    "category": "parallel"
  }
]
```

### List Edges

Get all edges in a workflow.

```http
GET /api/v1/workflows/{workflow_id}/edges
```

**Parameters:**
- `workflow_id` (path, required): UUID of the workflow

**Response:**
```json
[
  {
    "id": "edge-uuid-1",
    "workflow_id": "workflow-uuid",
    "from": "start",
    "from_id": "node-uuid-1",
    "to": "process",
    "to_id": "node-uuid-2",
    "type": "direct",
    "config": {}
  },
  {
    "id": "edge-uuid-2",
    "workflow_id": "workflow-uuid",
    "from": "router",
    "from_id": "node-uuid-3",
    "to": "branch_a",
    "to_id": "node-uuid-4",
    "type": "conditional",
    "config": {
      "expression": "status == 200"
    }
  }
]
```

### Get Specific Edge

Get details of a specific edge.

```http
GET /api/v1/workflows/{workflow_id}/edges/{edge_id}
```

**Parameters:**
- `workflow_id` (path, required): UUID of the workflow
- `edge_id` (path, required): UUID of the edge

**Response:**
```json
{
  "id": "edge-uuid",
  "workflow_id": "workflow-uuid",
  "from": "start",
  "from_id": "node-uuid-1",
  "to": "process",
  "to_id": "node-uuid-2",
  "type": "direct",
  "config": {}
}
```

### Create Edge

Create a new edge in the workflow.

```http
POST /api/v1/workflows/{workflow_id}/edges
```

**Request Body:**
```json
{
  "from": "start",
  "to": "process",
  "type": "direct",
  "config": {}
}
```

**Response:** `201 Created`
```json
{
  "id": "new-edge-uuid",
  "workflow_id": "workflow-uuid",
  "from": "start",
  "from_id": "node-uuid-1",
  "to": "process",
  "to_id": "node-uuid-2",
  "type": "direct",
  "config": {}
}
```

### Update Edge

Update an existing edge. Note: This creates a new edge with a new ID and deletes the old one (immutable pattern).

```http
PUT /api/v1/workflows/{workflow_id}/edges/{edge_id}
```

**Request Body:**
```json
{
  "config": {
    "expression": "status == 201"
  }
}
```

**Response:** `200 OK`
```json
{
  "id": "new-edge-uuid",
  "workflow_id": "workflow-uuid",
  "from": "router",
  "from_id": "node-uuid-1",
  "to": "branch_a",
  "to_id": "node-uuid-2",
  "type": "conditional",
  "config": {
    "expression": "status == 201"
  }
}
```

### Delete Edge

Delete an edge from the workflow.

```http
DELETE /api/v1/workflows/{workflow_id}/edges/{edge_id}
```

**Response:** `204 No Content`

### Get Workflow Graph

Get a visual representation of the workflow graph.

```http
GET /api/v1/workflows/{workflow_id}/graph
```

**Response:**
```json
{
  "workflow_id": "workflow-uuid",
  "nodes": [
    {
      "id": "node-uuid-1",
      "name": "start",
      "type": "start"
    },
    {
      "id": "node-uuid-2",
      "name": "process",
      "type": "transform"
    },
    {
      "id": "node-uuid-3",
      "name": "end",
      "type": "end"
    }
  ],
  "edges": [
    {
      "id": "edge-uuid-1",
      "workflow_id": "workflow-uuid",
      "from": "start",
      "from_id": "node-uuid-1",
      "to": "process",
      "to_id": "node-uuid-2",
      "type": "direct",
      "config": {}
    },
    {
      "id": "edge-uuid-2",
      "workflow_id": "workflow-uuid",
      "from": "process",
      "from_id": "node-uuid-2",
      "to": "end",
      "to_id": "node-uuid-3",
      "type": "direct",
      "config": {}
    }
  ]
}
```

## Edge Types

### 1. Direct Edge

**Type:** `direct`
**Category:** Flow
**Description:** Simple sequential transition from one node to another.

**Use Cases:**
- Linear workflows
- Sequential processing
- Default flow pattern

**Configuration:** None required

**Example:**
```json
{
  "from": "start",
  "to": "process",
  "type": "direct"
}
```

### 2. Conditional Edge

**Type:** `conditional`
**Category:** Flow
**Description:** Transition based on condition evaluation using expr-lang.

**Use Cases:**
- Decision branching
- Status-based routing
- Error handling paths

**Configuration:**
```json
{
  "expression": "status_code == 200",
  "description": "Route to success handler"
}
```

**Available Variables in Expressions:**
- All variables from parent node output
- Execution context variables

**Expression Examples:**
```javascript
// Status check
"status == 'success'"

// Numeric comparison
"count > 100"

// Complex conditions
"status_code >= 200 && status_code < 300"

// String operations
"type == 'premium' && balance > 1000"
```

**Example:**
```json
{
  "from": "check_user",
  "to": "premium_flow",
  "type": "conditional",
  "config": {
    "expression": "user.tier == 'premium'"
  }
}
```

### 3. Fork Edge

**Type:** `fork`
**Category:** Parallel
**Description:** Start parallel execution branches.

**Use Cases:**
- Parallel processing
- Multi-path workflows
- Fan-out patterns

**Configuration:** None required

**Notes:**
- Source node must be a `parallel` node with mode `fork`
- Multiple fork edges can originate from same fork node
- Each fork edge creates independent execution branch

**Example:**
```json
{
  "from": "parallel_start",
  "to": "branch_a",
  "type": "fork"
}
```

### 4. Join Edge

**Type:** `join`
**Category:** Parallel
**Description:** Synchronize parallel execution branches.

**Use Cases:**
- Synchronization points
- Aggregating parallel results
- Fan-in patterns

**Configuration:** None required (join strategy configured on join node)

**Notes:**
- Target node must be a `parallel` node with mode `join`
- Multiple join edges can target same join node
- Join strategy determines synchronization behavior (wait_all, wait_any, wait_n)

**Example:**
```json
{
  "from": "branch_a",
  "to": "parallel_end",
  "type": "join"
}
```

## Request/Response Formats

### CreateEdgeRequest

```typescript
{
  from: string;        // Source node name (required)
  to: string;          // Target node name (required)
  type: EdgeType;      // Edge type: direct, conditional, fork, join (required)
  config?: object;     // Edge configuration (optional)
}
```

### UpdateEdgeRequest

```typescript
{
  from?: string;       // New source node name
  to?: string;         // New target node name
  type?: EdgeType;     // New edge type
  config?: object;     // New configuration
}
```

### EdgeDetailResponse

```typescript
{
  id: string;          // Edge UUID
  workflow_id: string; // Workflow UUID
  from: string;        // Source node name
  from_id: string;     // Source node UUID
  to: string;          // Target node name
  to_id: string;       // Target node UUID
  type: EdgeType;      // Edge type
  config?: object;     // Edge configuration
}
```

## Examples

### Example 1: Simple Sequential Flow

```bash
# Create start -> process edge
curl -X POST http://localhost:8080/api/v1/workflows/$WF_ID/edges \
  -H "Content-Type: application/json" \
  -d '{
    "from": "start",
    "to": "process",
    "type": "direct"
  }'

# Create process -> end edge
curl -X POST http://localhost:8080/api/v1/workflows/$WF_ID/edges \
  -H "Content-Type: application/json" \
  -d '{
    "from": "process",
    "to": "end",
    "type": "direct"
  }'
```

### Example 2: Conditional Routing

```bash
# Create conditional router workflow
WF_ID=$(curl -s -X POST http://localhost:8080/api/v1/workflows \
  -H "Content-Type: application/json" \
  -d '{
    "name": "status-router",
    "version": "1.0.0",
    "nodes": [
      {"type": "start", "name": "start"},
      {"type": "http", "name": "api_call", "config": {"url": "https://api.example.com"}},
      {"type": "conditional-route", "name": "router"},
      {"type": "transform", "name": "success_handler"},
      {"type": "transform", "name": "error_handler"},
      {"type": "end", "name": "end"}
    ],
    "edges": [],
    "triggers": [{"type": "manual"}]
  }' | jq -r '.id')

# Add direct edges
curl -X POST http://localhost:8080/api/v1/workflows/$WF_ID/edges \
  -d '{"from": "start", "to": "api_call", "type": "direct"}'

curl -X POST http://localhost:8080/api/v1/workflows/$WF_ID/edges \
  -d '{"from": "api_call", "to": "router", "type": "direct"}'

# Add conditional edges
curl -X POST http://localhost:8080/api/v1/workflows/$WF_ID/edges \
  -d '{
    "from": "router",
    "to": "success_handler",
    "type": "conditional",
    "config": {"expression": "status_code >= 200 && status_code < 300"}
  }'

curl -X POST http://localhost:8080/api/v1/workflows/$WF_ID/edges \
  -d '{
    "from": "router",
    "to": "error_handler",
    "type": "conditional",
    "config": {"expression": "status_code >= 400"}
  }'

# Connect to end
curl -X POST http://localhost:8080/api/v1/workflows/$WF_ID/edges \
  -d '{"from": "success_handler", "to": "end", "type": "direct"}'

curl -X POST http://localhost:8080/api/v1/workflows/$WF_ID/edges \
  -d '{"from": "error_handler", "to": "end", "type": "direct"}'
```

### Example 3: Parallel Execution

```bash
# Create parallel workflow
WF_ID=$(curl -s -X POST http://localhost:8080/api/v1/workflows \
  -H "Content-Type: application/json" \
  -d '{
    "name": "parallel-processing",
    "version": "1.0.0",
    "nodes": [
      {"type": "start", "name": "start"},
      {"type": "parallel", "name": "fork", "config": {"mode": "fork"}},
      {"type": "transform", "name": "branch_a"},
      {"type": "transform", "name": "branch_b"},
      {"type": "transform", "name": "branch_c"},
      {"type": "parallel", "name": "join", "config": {"mode": "join", "join_strategy": "wait_all"}},
      {"type": "end", "name": "end"}
    ],
    "edges": [],
    "triggers": [{"type": "manual"}]
  }' | jq -r '.id')

# Start to fork
curl -X POST http://localhost:8080/api/v1/workflows/$WF_ID/edges \
  -d '{"from": "start", "to": "fork", "type": "direct"}'

# Fork to branches (parallel)
curl -X POST http://localhost:8080/api/v1/workflows/$WF_ID/edges \
  -d '{"from": "fork", "to": "branch_a", "type": "fork"}'

curl -X POST http://localhost:8080/api/v1/workflows/$WF_ID/edges \
  -d '{"from": "fork", "to": "branch_b", "type": "fork"}'

curl -X POST http://localhost:8080/api/v1/workflows/$WF_ID/edges \
  -d '{"from": "fork", "to": "branch_c", "type": "fork"}'

# Branches to join (synchronization)
curl -X POST http://localhost:8080/api/v1/workflows/$WF_ID/edges \
  -d '{"from": "branch_a", "to": "join", "type": "join"}'

curl -X POST http://localhost:8080/api/v1/workflows/$WF_ID/edges \
  -d '{"from": "branch_b", "to": "join", "type": "join"}'

curl -X POST http://localhost:8080/api/v1/workflows/$WF_ID/edges \
  -d '{"from": "branch_c", "to": "join", "type": "join"}'

# Join to end
curl -X POST http://localhost:8080/api/v1/workflows/$WF_ID/edges \
  -d '{"from": "join", "to": "end", "type": "direct"}'
```

### Example 4: Update Edge Condition

```bash
# Get current edge
EDGE_ID=$(curl -s http://localhost:8080/api/v1/workflows/$WF_ID/edges | \
  jq -r '.[] | select(.from == "router" and .to == "success_handler") | .id')

# Update condition
curl -X PUT http://localhost:8080/api/v1/workflows/$WF_ID/edges/$EDGE_ID \
  -H "Content-Type: application/json" \
  -d '{
    "config": {
      "expression": "status_code == 200 || status_code == 201"
    }
  }'
```

### Example 5: Visualize Workflow Graph

```bash
# Get graph visualization
curl http://localhost:8080/api/v1/workflows/$WF_ID/graph | jq

# Extract just node names and connections
curl -s http://localhost:8080/api/v1/workflows/$WF_ID/graph | \
  jq '{
    nodes: [.nodes[].name],
    connections: [.edges[] | {from: .from, to: .to, type: .type}]
  }'
```

### Example 6: Reconnect Node

```bash
# Change "process" node to connect to different next node

# 1. Find current edge
OLD_EDGE_ID=$(curl -s http://localhost:8080/api/v1/workflows/$WF_ID/edges | \
  jq -r '.[] | select(.from == "process") | .id')

# 2. Delete old edge
curl -X DELETE http://localhost:8080/api/v1/workflows/$WF_ID/edges/$OLD_EDGE_ID

# 3. Create new edge to different target
curl -X POST http://localhost:8080/api/v1/workflows/$WF_ID/edges \
  -d '{"from": "process", "to": "new_target", "type": "direct"}'
```

## Best Practices

### 1. Workflow Structure Validation

**Always validate workflow is a DAG (Directed Acyclic Graph):**
- No cycles allowed
- Single start node
- At least one end node
- All nodes reachable from start

**Example validation:**
```bash
# Get graph and check for issues
curl -s http://localhost:8080/api/v1/workflows/$WF_ID/graph | \
  jq '{
    node_count: (.nodes | length),
    edge_count: (.edges | length),
    nodes_with_no_incoming: [.nodes[].name] - [.edges[].to],
    nodes_with_no_outgoing: [.nodes[].name] - [.edges[].from]
  }'
```

### 2. Conditional Edge Patterns

**Good Practice:**
```json
{
  "from": "router",
  "to": "success",
  "type": "conditional",
  "config": {
    "expression": "status >= 200 && status < 300"
  }
}
```

**Bad Practice:**
```json
{
  "config": {
    "expression": "maybe_undefined_var == true"  // No null safety
  }
}
```

**Best Practices:**
- Always provide default/fallback paths
- Use null-safe expressions
- Test expressions with realistic data
- Document complex conditions

### 3. Parallel Execution Patterns

**Fork/Join Pairs:**
- Always pair fork nodes with join nodes
- Configure appropriate join strategy
- Consider timeout for join operations

**Example:**
```bash
# Fork node config
{
  "type": "parallel",
  "name": "fork",
  "config": {
    "mode": "fork"
  }
}

# Join node config
{
  "type": "parallel",
  "name": "join",
  "config": {
    "mode": "join",
    "join_strategy": "wait_all",
    "timeout": "30s"
  }
}
```

### 4. Edge Naming and Documentation

**Use descriptive edge configurations:**
```json
{
  "from": "router",
  "to": "premium_handler",
  "type": "conditional",
  "config": {
    "expression": "user.tier == 'premium' && balance > 1000",
    "description": "Route premium users with sufficient balance"
  }
}
```

### 5. Error Handling

**Add error handling paths:**
```bash
# Always add error handling routes
curl -X POST http://localhost:8080/api/v1/workflows/$WF_ID/edges \
  -d '{
    "from": "api_call",
    "to": "error_handler",
    "type": "conditional",
    "config": {
      "expression": "error != null || status_code >= 500"
    }
  }'
```

### 6. Testing Edge Changes

**Before updating production edges:**
1. Create test workflow copy
2. Test new edge configuration
3. Verify execution paths
4. Apply to production

```bash
# Clone workflow for testing
# (Implement workflow clone endpoint or manually recreate)

# Test with execution
curl -X POST http://localhost:8080/api/v1/executions \
  -d '{"workflow_id": "$TEST_WF_ID", "variables": {...}}'

# Verify correct paths taken
curl http://localhost:8080/api/v1/executions/$EXEC_ID/events | \
  jq '[.[] | select(.type == "node_started") | .node_name]'
```

### 7. Graph Complexity Management

**Keep workflows maintainable:**
- Limit fanout (fork branches) to 3-5
- Avoid deeply nested conditionals
- Use subworkflows for complex logic
- Document edge purposes

### 8. Performance Considerations

**Conditional expressions:**
- Keep expressions simple
- Avoid expensive operations
- Cache frequently used values
- Use indexed variables

**Parallel edges:**
- Balance branch workloads
- Set appropriate timeouts
- Monitor resource usage
- Consider join strategies carefully

## Error Responses

**400 Bad Request:**
```json
{
  "error": "Invalid edge configuration: missing required field 'from'"
}
```

**404 Not Found:**
```json
{
  "error": "Workflow not found"
}
```

**500 Internal Server Error:**
```json
{
  "error": "Failed to create edge: cycle detected in workflow graph"
}
```

## Additional Resources

- [Node CRUD API](NODES_API.md)
- [Workflow API](README.md)
- [OpenAPI Specification](openapi.yaml)
- [Quick Start Guide](../../API_README.md)

## Support

For questions and bug reports:
https://github.com/smilemakc/mbflow/issues
