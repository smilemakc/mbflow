# MBFlow Backend

A sophisticated workflow orchestration engine written in Go implementing Domain-Driven Design (DDD) principles with Event Sourcing.

## Features

- DAG-based workflow automation with cycle detection
- Event-sourced execution with complete audit trail
- Wave-based parallel execution
- Retry mechanisms with exponential backoff
- Built-in executors (HTTP, Transform, LLM)
- Custom executor support
- REST API with real-time WebSocket updates
- PostgreSQL for persistence
- Redis for caching and pub/sub

## Architecture

```
pkg/
├── sdk/              # Public SDK for embedding MBFlow
├── models/           # Domain models
└── executor/         # Executor interface and registry

internal/
├── config/           # Configuration management
└── infrastructure/   # Database, cache, logging

cmd/
└── server/           # HTTP server

examples/
├── basic_usage/      # Basic SDK usage
├── custom_executor/  # Custom executor example
└── embedded_server/  # Embedded mode example
```

## Quick Start

### Prerequisites

- Go 1.23 or later
- PostgreSQL 16
- Redis 7 (optional)
- Docker and Docker Compose (for containerized setup)

### Using Docker Compose (Recommended)

1. Clone the repository:
```bash
git clone https://github.com/smilemakc/mbflow.git
cd mbflow/backend
```

2. Start all services:
```bash
docker compose up -d
```

3. Check service health:
```bash
curl http://localhost:8181/health
```

### Local Development Setup

1. Install dependencies:
```bash
go mod download
```

2. Start PostgreSQL and Redis:
```bash
# Using Docker
docker run -d --name mbflow-postgres \
  -e POSTGRES_DB=mbflow \
  -e POSTGRES_USER=mbflow \
  -e POSTGRES_PASSWORD=mbflow \
  -p 5432:5432 \
  postgres:16-alpine

docker run -d --name mbflow-redis \
  -p 6379:6379 \
  redis:7-alpine
```

3. Copy environment file:
```bash
cp .env.example .env
# Edit .env with your configuration
```

4. Build and run the server:
```bash
go build -o mbflow-server ./cmd/server
./mbflow-server
```

The server will start on `http://localhost:8181`

## Configuration

Configuration is managed through environment variables. See `.env.example` for all available options.

### Key Configuration Options

| Variable | Description | Default |
|----------|-------------|---------|
| `PORT` | Server port | 8181 |
| `DATABASE_URL` | PostgreSQL connection string | See .env.example |
| `REDIS_URL` | Redis connection string | redis://localhost:6379 |
| `LOG_LEVEL` | Log level (debug/info/warn/error) | info |
| `LOG_FORMAT` | Log format (json/text) | json |
| `DB_MAX_CONNECTIONS` | Maximum database connections | 20 |
| `API_KEYS` | Comma-separated API keys | - |

## API Endpoints

### Health & Monitoring

- `GET /health` - Health check
- `GET /ready` - Readiness check
- `GET /metrics` - System metrics

### API v1

- `POST /api/v1/workflows` - Create workflow
- `GET /api/v1/workflows` - List workflows
- `GET /api/v1/workflows/:id` - Get workflow
- `PUT /api/v1/workflows/:id` - Update workflow
- `DELETE /api/v1/workflows/:id` - Delete workflow
- `POST /api/v1/executions` - Execute workflow
- `GET /api/v1/executions/:id` - Get execution
- `POST /api/v1/triggers` - Create trigger

(Full API documentation coming soon)

## SDK Usage

### Remote Mode (Connect to API Server)

```go
package main

import (
    "context"
    "github.com/smilemakc/mbflow/pkg/sdk"
    "github.com/smilemakc/mbflow/pkg/models"
)

func main() {
    // Create client
    client, err := sdk.NewClient(
        sdk.WithHTTPEndpoint("http://localhost:8181"),
        sdk.WithAPIKey("your-api-key"),
    )
    if err != nil {
        panic(err)
    }
    defer client.Close()

    ctx := context.Background()

    // Create workflow
    workflow := &models.Workflow{
        Name: "My Workflow",
        Nodes: []*models.Node{
            {
                ID:   "node-1",
                Name: "Fetch Data",
                Type: "http",
                Config: map[string]interface{}{
                    "method": "GET",
                    "url":    "https://api.example.com/data",
                },
            },
        },
    }

    created, err := client.Workflows().Create(ctx, workflow)
    if err != nil {
        panic(err)
    }

    // Execute workflow
    execution, err := client.Executions().Run(ctx, created.ID, nil)
    if err != nil {
        panic(err)
    }
}
```

### Embedded Mode (In-Process Engine)

```go
client, err := sdk.NewClient(
    sdk.WithEmbeddedMode(
        "postgres://user:pass@localhost:5432/mbflow",
        "redis://localhost:6379",
    ),
)
```

### Custom Executors

```go
package main

import (
    "context"
    "github.com/smilemakc/mbflow/pkg/executor"
)

type MyExecutor struct {
    *executor.BaseExecutor
}

func (e *MyExecutor) Execute(ctx context.Context, config map[string]interface{}, input interface{}) (interface{}, error) {
    // Your custom logic here
    return map[string]interface{}{"result": "success"}, nil
}

func (e *MyExecutor) Validate(config map[string]interface{}) error {
    return e.ValidateRequired(config, "required_field")
}

// Register
manager := executor.NewManager()
manager.Register("my-executor", &MyExecutor{
    BaseExecutor: executor.NewBaseExecutor("my-executor"),
})
```

## Examples

Run the examples:

```bash
# Basic usage
go run examples/basic_usage/main.go

# Custom executor
go run examples/custom_executor/main.go

# Embedded server
go run examples/embedded_server/main.go
```

## Development

### Running Tests

```bash
# Run all tests
go test ./...

# Run with coverage
go test -cover ./...

# Run with race detector
go test -race ./...
```

### Building

```bash
# Build server binary
go build -o mbflow-server ./cmd/server

# Build for Linux
GOOS=linux GOARCH=amd64 go build -o mbflow-server-linux ./cmd/server

# Build Docker image
docker build -t mbflow:latest .
```

### Code Structure

- `pkg/` - Public packages that can be imported by other projects
- `internal/` - Private implementation details
- `cmd/` - Application entry points
- `examples/` - Example applications

## Deployment

### Docker

```bash
docker build -t mbflow:latest .
docker run -d \
  -p 8181:8181 \
  -e DATABASE_URL=postgres://... \
  -e REDIS_URL=redis://... \
  mbflow:latest
```

### Docker Compose

```bash
docker compose up -d
```

### Kubernetes

Kubernetes manifests coming soon.

## Performance

- Supports 1000+ concurrent executions
- Database connection pooling (20 connections default)
- Redis caching for improved performance
- DAG validation: O(V+E) complexity
- API response: p95 <100ms

## Monitoring

The server exposes metrics at `/metrics`:

```bash
curl http://localhost:8181/metrics
```

Metrics include:
- Database connection pool statistics
- Redis cache statistics
- Request counts and latencies
- Execution statistics

## Security

- API key authentication support
- Environment variable-based secrets
- No secrets in source code
- CORS support for browser clients

## Troubleshooting

### Database Connection Issues

```bash
# Test database connection
psql "postgres://mbflow:mbflow@localhost:5432/mbflow"
```

### Redis Connection Issues

```bash
# Test Redis connection
redis-cli ping
```

### View Logs

```bash
# Docker Compose
docker compose logs -f mbflow-api

# View specific service logs
docker compose logs -f postgres
docker compose logs -f redis
```

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests
5. Submit a pull request

## License

MIT License - see LICENSE file for details

## Support

- Documentation: [docs/](../docs/)
- Issues: [GitHub Issues](https://github.com/smilemakc/mbflow/issues)
- Discussions: [GitHub Discussions](https://github.com/smilemakc/mbflow/discussions)
