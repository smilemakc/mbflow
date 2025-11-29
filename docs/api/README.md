# MBFlow REST API Documentation

Complete REST API for managing workflows and executions in MBFlow.

## Quick Start

### Running the Server

```bash
# Using Go directly
go run cmd/server/main.go

# With custom port
go run cmd/server/main.go -port 8181

# With CORS enabled
go run cmd/server/main.go -cors=true

# With API key authentication
go run cmd/server/main.go -api-keys="key1,key2,key3"

# With metrics collection
go run cmd/server/main.go -metrics=true
```

### Using Docker Compose

```bash
# Start all services (API, PostgreSQL, Redis, Swagger UI)
docker-compose up -d

# View logs
docker-compose logs -f mbflow-api

# Stop services
docker-compose down

# Stop and remove volumes
docker-compose down -v
```

## API Endpoints

### Health Checks

- `GET /health` - Health check endpoint
- `GET /ready` - Readiness check endpoint

### Workflow Management

- `GET /api/v1/workflows` - List all workflows
- `GET /api/v1/workflows/{id}` - Get workflow by ID
- `POST /api/v1/workflows` - Create new workflow
- `PUT /api/v1/workflows/{id}` - Update workflow
- `DELETE /api/v1/workflows/{id}` - Delete workflow

### Execution Management

- `GET /api/v1/executions` - List all executions
- `GET /api/v1/executions/{id}` - Get execution by ID
- `POST /api/v1/executions` - Execute a workflow
- `GET /api/v1/executions/{id}/events` - Get execution events
- `POST /api/v1/executions/{id}/cancel` - Cancel execution
- `POST /api/v1/executions/{id}/pause` - Pause execution
- `POST /api/v1/executions/{id}/resume` - Resume execution

## Examples

### Creating a Workflow

```bash
curl -X POST http://localhost:8181/api/v1/workflows \
  -H "Content-Type: application/json" \
  -d '{
    "name": "simple-workflow",
    "version": "1.0.0",
    "description": "A simple transformation workflow",
    "nodes": [
      {
        "type": "start",
        "name": "start"
      },
      {
        "type": "transform",
        "name": "double",
        "config": {
          "transformations": {
            "result": "input * 2"
          }
        }
      },
      {
        "type": "end",
        "name": "end"
      }
    ],
    "edges": [
      {
        "from": "start",
        "to": "double",
        "type": "direct"
      },
      {
        "from": "double",
        "to": "end",
        "type": "direct"
      }
    ],
    "triggers": [
      {
        "type": "manual"
      }
    ]
  }'
```

### Listing Workflows

```bash
curl http://localhost:8181/api/v1/workflows
```

### Getting a Workflow

```bash
curl http://localhost:8181/api/v1/workflows/{workflow-id}
```

### Executing a Workflow

```bash
curl -X POST http://localhost:8181/api/v1/executions \
  -H "Content-Type: application/json" \
  -d '{
    "workflow_id": "{workflow-id}",
    "variables": {
      "input": 42
    }
  }'
```

### Listing Executions

```bash
# All executions
curl http://localhost:8181/api/v1/executions

# Filter by workflow ID
curl "http://localhost:8181/api/v1/executions?workflow_id={workflow-id}"

# Filter by status
curl "http://localhost:8181/api/v1/executions?status=completed"
```

### Getting Execution Details

```bash
curl http://localhost:8181/api/v1/executions/{execution-id}
```

### Getting Execution Events

```bash
curl http://localhost:8181/api/v1/executions/{execution-id}/events
```

### Canceling an Execution

```bash
curl -X POST http://localhost:8181/api/v1/executions/{execution-id}/cancel
```

### Pausing an Execution

```bash
curl -X POST http://localhost:8181/api/v1/executions/{execution-id}/pause
```

### Resuming an Execution

```bash
curl -X POST http://localhost:8181/api/v1/executions/{execution-id}/resume
```

## Authentication

If API keys are configured, include them in requests:

```bash
# Using X-API-Key header
curl -H "X-API-Key: your-api-key" http://localhost:8181/api/v1/workflows

# Using Authorization Bearer token
curl -H "Authorization: Bearer your-api-key" http://localhost:8181/api/v1/workflows
```

## Interactive API Documentation

The Swagger UI is available at:

```
http://localhost:8081/docs
```

This provides:
- Interactive API explorer
- Request/response examples
- Schema documentation
- Try-it-out functionality

## Advanced Workflow Examples

### Parallel Processing Workflow

```json
{
  "name": "parallel-workflow",
  "version": "1.0.0",
  "nodes": [
    {
      "type": "start",
      "name": "start"
    },
    {
      "type": "parallel",
      "name": "fork"
    },
    {
      "type": "transform",
      "name": "branch1",
      "config": {
        "transformations": {
          "result": "input * 2"
        }
      }
    },
    {
      "type": "transform",
      "name": "branch2",
      "config": {
        "transformations": {
          "result": "input * 3"
        }
      }
    },
    {
      "type": "parallel",
      "name": "join",
      "config": {
        "join_strategy": "wait_all"
      }
    },
    {
      "type": "end",
      "name": "end"
    }
  ],
  "edges": [
    {
      "from": "start",
      "to": "fork",
      "type": "direct"
    },
    {
      "from": "fork",
      "to": "branch1",
      "type": "fork"
    },
    {
      "from": "fork",
      "to": "branch2",
      "type": "fork"
    },
    {
      "from": "branch1",
      "to": "join",
      "type": "join"
    },
    {
      "from": "branch2",
      "to": "join",
      "type": "join"
    },
    {
      "from": "join",
      "to": "end",
      "type": "direct"
    }
  ],
  "triggers": [
    {
      "type": "manual"
    }
  ]
}
```

### Conditional Routing Workflow

```json
{
  "name": "conditional-workflow",
  "version": "1.0.0",
  "nodes": [
    {
      "type": "start",
      "name": "start"
    },
    {
      "type": "conditional-route",
      "name": "router"
    },
    {
      "type": "transform",
      "name": "positive-path",
      "config": {
        "transformations": {
          "message": "\"Number is positive\""
        }
      }
    },
    {
      "type": "transform",
      "name": "negative-path",
      "config": {
        "transformations": {
          "message": "\"Number is negative\""
        }
      }
    },
    {
      "type": "end",
      "name": "end"
    }
  ],
  "edges": [
    {
      "from": "start",
      "to": "router",
      "type": "direct"
    },
    {
      "from": "router",
      "to": "positive-path",
      "type": "conditional",
      "condition": {
        "expression": "input > 0"
      }
    },
    {
      "from": "router",
      "to": "negative-path",
      "type": "conditional",
      "condition": {
        "expression": "input <= 0"
      }
    },
    {
      "from": "positive-path",
      "to": "end",
      "type": "direct"
    },
    {
      "from": "negative-path",
      "to": "end",
      "type": "direct"
    }
  ],
  "triggers": [
    {
      "type": "manual"
    }
  ]
}
```

## Error Handling

All errors return a standard error response:

```json
{
  "error": "Error message description"
}
```

Common HTTP status codes:
- `200 OK` - Successful request
- `201 Created` - Resource created successfully
- `204 No Content` - Resource deleted successfully
- `400 Bad Request` - Invalid request body or parameters
- `404 Not Found` - Resource not found
- `500 Internal Server Error` - Server error

## Rate Limiting

Rate limiting can be enabled with:

```bash
go run cmd/server/main.go -rate-limit=true
```

Default limits:
- 100 requests per minute per IP address

When rate limited, the API returns:

```json
{
  "error": "Rate limit exceeded"
}
```

HTTP status code: `429 Too Many Requests`

## Configuration

### Environment Variables

- `PORT` - Server port (default: 8181)
- `LOG_LEVEL` - Logging level: debug, info, warn, error (default: info)
- `CORS_ENABLED` - Enable CORS (default: true)
- `METRICS_ENABLED` - Enable metrics collection (default: true)

### Command-Line Flags

- `-port` - Override server port
- `-cors` - Enable/disable CORS
- `-metrics` - Enable/disable metrics collection
- `-api-keys` - Comma-separated API keys for authentication

## Monitoring and Observability

The API includes built-in observability features:

### Metrics

When metrics are enabled, the server collects:
- Total executions
- Successful/failed executions
- Node execution counts
- Execution durations
- Error rates

### Logging

All requests are logged with:
- Method and path
- Status code
- Duration
- Response size
- Client information

### Health Checks

- `/health` - Always returns 200 if server is running
- `/ready` - Returns 200 when server is ready to accept traffic

## Security Best Practices

1. **Use API Keys**: Always enable API key authentication in production
2. **HTTPS**: Use HTTPS in production environments
3. **Rate Limiting**: Enable rate limiting to prevent abuse
4. **Environment Variables**: Store sensitive configuration in environment variables
5. **Docker Security**: Run containers as non-root user (already configured)

## Troubleshooting

### Server won't start

Check that the port is not already in use:

```bash
lsof -i :8080
```

### API returns 401 Unauthorized

Ensure you're including the API key in your requests when authentication is enabled.

### Execution fails immediately

Check the workflow definition for errors using the validation endpoint.

### Events not appearing

Events are stored in the event store. Check that the event store is properly initialized.

## Development

### Running Tests

```bash
# Run all tests
go test ./...

# Run with coverage
go test -cover ./...

# Run specific package tests
go test ./internal/infrastructure/api/rest/...
```

### Building

```bash
# Build server binary
go build -o mbflow-server ./cmd/server

# Build Docker image
docker build -t mbflow-api:latest .
```

## Support

For issues and feature requests, please visit:
https://github.com/smilemakc/mbflow/issues
