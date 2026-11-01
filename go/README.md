# MBFlow Backend

A sophisticated workflow orchestration engine written in Go implementing Domain-Driven Design (DDD) principles with
Event Sourcing.

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

- Go 1.25 or later
- PostgreSQL 16
- Redis 7 (optional)
- Docker and Docker Compose (for containerized setup)

### Using Docker Compose (Recommended)

1. Clone the repository:

```bash
git clone https://github.com/smilemakc/mbflow.git
cd mbflow/go
```

2. Create a `.env` file with secrets (see the top of `docker-compose.yml`):

```bash
cat > .env <<ENV
MBFLOW_JWT_SECRET=$(openssl rand -hex 32)
POSTGRES_PASSWORD=$(openssl rand -hex 16)
REDIS_PASSWORD=$(openssl rand -hex 16)
MBFLOW_ADMIN_EMAIL=admin@example.com
MBFLOW_ADMIN_PASSWORD=$(openssl rand -base64 18)Aa1!  # print it: grep ADMIN_PASSWORD .env
ENV
```

3. Start all services (migrations run automatically via the one-shot `mbflow-migrate` service):

```bash
docker compose up -d --build
```

4. Check service health and open the UI at http://localhost:8080:

```bash
curl http://localhost:8585/health   # liveness
curl http://localhost:8585/ready    # readiness (DB + Redis)
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

4. Apply database migrations, then build and run the server:

```bash
go run ./cmd/migrate -command up   # uses MBFLOW_DATABASE_URL
go build -o mbflow-server ./cmd/server
./mbflow-server
```

The server will start on `http://localhost:8585`

## Configuration

Configuration is managed through environment variables. See `.env.example` for all available options.

### Key Configuration Options

| Variable                    | Description                       | Default                |
|-----------------------------|-----------------------------------|------------------------|
| `MBFLOW_PORT`               | Server port                       | 8585                   |
| `MBFLOW_DATABASE_URL`       | PostgreSQL connection string      | See .env.example       |
| `MBFLOW_REDIS_URL`          | Redis connection string           | redis://localhost:6379 |
| `MBFLOW_LOG_LEVEL`          | Log level (debug/info/warn/error) | info                   |
| `MBFLOW_LOG_FORMAT`         | Log format (json/text)            | json                   |
| `MBFLOW_DB_MAX_CONNECTIONS` | Maximum database connections      | 20                     |
| `MBFLOW_API_KEYS`           | Comma-separated API keys          | -                      |
| `MBFLOW_MAX_CONCURRENT_EXECUTIONS` | Executions running at once per process; more wait as `pending` (0 = unlimited) | 100 |
| `MBFLOW_EXECUTION_HEARTBEAT_INTERVAL` | How often running executions renew their lease, abandoned ones are swept and cancellations made on other replicas are applied (0 disables crash recovery) | 30s |
| `MBFLOW_EXECUTION_STALE_AFTER` | Lease age after which a `pending`/`running` execution is marked failed as interrupted | 3x heartbeat |

Executions left `running` by a process that crashed are marked `failed` with
"execution interrupted: server stopped while it was running" once their lease
expires; any replica sharing the database does this, including at startup.
With Redis configured, login lockout counters are stored in Redis and shared
by all replicas; without it they are kept per process.

### Authentication Configuration

MBFlow supports multiple authentication modes:

| Mode          | Description                             |
|---------------|-----------------------------------------|
| `builtin`     | Local authentication with JWT (default) |
| `grpc`        | External auth via gRPC auth-gateway     |
| `grpc_hybrid` | gRPC primary with local fallback        |
| `gateway`     | OAuth2/OIDC gateway                     |
| `hybrid`      | Gateway primary with local fallback     |

#### Built-in Authentication

```bash
MBFLOW_AUTH_MODE=builtin
MBFLOW_JWT_SECRET=your-secret-key-at-least-32-characters
MBFLOW_JWT_EXPIRATION_HOURS=24
MBFLOW_ALLOW_REGISTRATION=true
```

#### gRPC Auth-Gateway Integration

```bash
MBFLOW_AUTH_MODE=grpc              # or grpc_hybrid for fallback support
MBFLOW_AUTH_GRPC_ADDRESS=localhost:50051
MBFLOW_AUTH_GRPC_TIMEOUT=10s
MBFLOW_AUTH_ENABLE_FALLBACK=true   # Enable local fallback (for grpc_hybrid)

# gRPC Metadata (sent with every request to auth-gateway)
MBFLOW_AUTH_APPLICATION_ID=your-app-uuid
MBFLOW_AUTH_CLIENT_NAME=mbflow
MBFLOW_AUTH_CLIENT_VERSION=1.0.0
MBFLOW_AUTH_PLATFORM=kubernetes
MBFLOW_AUTH_ENVIRONMENT=production

# Transport security for the auth-gateway connection
MBFLOW_AUTH_GRPC_INSECURE=false                       # plaintext when true
MBFLOW_AUTH_GRPC_TLS_CA_FILE=/etc/mbflow/gateway-ca.pem   # optional, system roots if empty
MBFLOW_AUTH_GRPC_TLS_SERVER_NAME=auth.internal        # optional SNI / cert name override
```

`MBFLOW_AUTH_GRPC_INSECURE` defaults to `true` (plaintext gRPC) when
`MBFLOW_ENV` is not `production`, which keeps local development working
against an auth-gateway without certificates. In production it defaults to
`false`, so the connection uses TLS (minimum TLS 1.2) and verifies the
gateway certificate against `MBFLOW_AUTH_GRPC_TLS_CA_FILE` or, if unset, the
system root CAs. Set `MBFLOW_AUTH_GRPC_INSECURE=true` explicitly only when the
gateway is reachable over a trusted private network; tokens and passwords are
sent over this connection.

#### OAuth2/OIDC Gateway

```bash
MBFLOW_AUTH_MODE=gateway           # or hybrid for fallback support
MBFLOW_AUTH_GATEWAY_URL=https://auth.example.com
MBFLOW_AUTH_CLIENT_ID=your-client-id
MBFLOW_AUTH_CLIENT_SECRET=your-client-secret
MBFLOW_AUTH_ISSUER_URL=https://auth.example.com
MBFLOW_AUTH_JWKS_URL=https://auth.example.com/.well-known/jwks.json
MBFLOW_AUTH_REDIRECT_URL=http://localhost:8585/auth/callback
```

## API Endpoints

### Health & Monitoring

- `GET /health` (alias `/livez`) - Liveness check; does not touch dependencies
- `GET /ready` (alias `/readyz`) - Readiness check; verifies PostgreSQL and Redis, returns 503 while shutting down
- `GET /metrics` - System metrics

### API v1

- `POST /api/v1/workflows` - Create workflow (optionally with `nodes` and `edges`)
- `GET /api/v1/workflows` - List workflows
- `GET /api/v1/workflows/:id` - Get workflow
- `PUT /api/v1/workflows/:id` - Update workflow (`nodes`, `edges`, `resources` are replaced only when sent)
- `DELETE /api/v1/workflows/:id` - Delete workflow
- `POST /api/v1/workflows/:id/execute` - Execute workflow (async; poll the execution or use the WebSocket)
- `POST /api/v1/executions/ephemeral` - Run an inline workflow (`mode: sync|async`)
- `GET /api/v1/executions/:id` - Get execution
- `POST /api/v1/executions/:id/cancel` - Cancel a `pending`/`running` execution (200 once stopped; 202 if it is still stopping on another replica, which happens within one heartbeat interval; 409 if already finished)
- `POST /api/v1/executions/:id/retry` - Re-run a `failed`/`cancelled`/`timeout` stored-workflow execution with its original input and variables (202 with the new execution, whose `metadata.retry_of` is the original ID; 409 otherwise)
- `POST /api/v1/triggers` - Create trigger
- `POST /api/v1/webhooks/:trigger_id` - Call a webhook trigger

(Full API documentation coming soon)

### Triggers

Trigger changes made through the API (create, update, enable, disable,
delete) take effect immediately; executions started by a trigger record its
`trigger_id`. Triggers are created disabled unless `"enabled": true` is sent.

| Type | Config |
|------|--------|
| `cron` | `{"schedule": "0 9 * * 1-5", "timezone": "Europe/Moscow", "input": {...}}`. Standard 5-field cron, 6-field with leading seconds, or descriptors such as `@hourly`. `timezone` is an IANA name (default UTC). |
| `interval` | `{"interval": "30s"}` (Go duration or number of seconds) |
| `webhook` | `{"secret": "...", "ip_whitelist": [...], "input": {...}}`, all optional. Call `POST /api/v1/webhooks/<trigger_id>`. |

Invalid schedules are rejected with `400 INVALID_TRIGGER_CONFIG`.

`POST /api/v1/triggers/:id/execute` runs an enabled trigger now: the optional
body `{"input": {...}}` is merged over the trigger's `config.input`, and the
response is `202 Accepted` with the queued execution (`id`, `status`, ...).
Reads return `next_run` for enabled cron and interval triggers, computed from
the schedule the same way the scheduler computes it.

### Using credentials in nodes

Store a secret once as an encrypted credential (requires `MBFLOW_ENCRYPTION_KEY`,
a base64-encoded 32-byte key), attach it to a workflow under an alias, and
reference its fields in node configs as `{{resource.<alias>.<field>}}`:

```bash
# 1. Create the credential (the response never contains the key)
curl -X POST /api/v1/credentials/api-key \
  -d '{"name": "anthropic", "provider": "anthropic", "api_key": "sk-ant-..."}'
# -> {"id": "<credential-id>", ...}

# 2. Attach it to the workflow as "anthropic"
curl -X POST /api/v1/workflows/<workflow-id>/resources \
  -d '{"resource_id": "<credential-id>", "alias": "anthropic"}'

# 3. Use it in a node, e.g. llm: "api_key": "{{resource.anthropic.api_key}}"
#    or http: "headers": {"Authorization": "Bearer {{resource.anthropic.api_key}}"}
```

| Credential type | Fields |
|-----------------|--------|
| `api_key` | `api_key` |
| `basic_auth` | `username`, `password` |
| `oauth2` | `client_id`, `client_secret`, `access_token`, `refresh_token`, `token_url`, `scopes` |
| `service_account` | `json_key` |
| `custom` | its own keys (also under `fields.<key>`; keys named `id`, `name`, `type`, `access_type`, `credential_type`, `provider` are only reachable there) |

Values are decrypted when the execution starts, only if the workflow owner
owns the credential; each execution increments the credential's `usage_count`.
An inline run (`POST /api/v1/executions/ephemeral`) lists the credentials in
`credential_ids`; the alias is taken from the inline workflow's
`resources: [{"resource_id": "<credential-id>", "alias": "anthropic"}]`, otherwise
it is the credential ID. Inline runs may only use the caller's own credentials
(service API callers set `workflow.created_by` to the owner).

Decrypted values (except OAuth2 `token_url` and `scopes`) are replaced by
`***` in everything the execution stores or emits: node `config`,
`resolved_config`, `input`, `output` and `error`, the execution's output and
error, and all events (event log, WebSocket, webhooks, logs). Nodes themselves
still receive the real values. Values shorter than 4 characters are not
redacted, and transformed forms of a secret (e.g. base64) are not recognised.
Without an encryption key, an execution whose nodes reference a credential
fails with `credential decryption is not available`.

## SDK Usage

### Remote Mode (Connect to API Server)

```go
package main

import (
	"context"
	"github.com/smilemakc/mbflow/go/pkg/sdk"
	"github.com/smilemakc/mbflow/go/pkg/models"
)

func main() {
	// Create client
	client, err := sdk.NewClient(
		sdk.WithHTTPEndpoint("http://localhost:8585"),
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
				Config: map[string]any{
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
	"github.com/smilemakc/mbflow/go/pkg/executor"
)

type MyExecutor struct {
	*executor.BaseExecutor
}

func (e *MyExecutor) Execute(ctx context.Context, config map[string]any, input any) (any, error) {
	// Your custom logic here
	return map[string]any{"result": "success"}, nil
}

func (e *MyExecutor) Validate(config map[string]any) error {
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
# Apply migrations (safe to run from several replicas: guarded by a lock)
docker run --rm -e MBFLOW_DATABASE_URL=postgres://... \
  --entrypoint ./mbflow-migrate mbflow:latest -command up

docker run -d \
  -p 8585:8585 \
  -e MBFLOW_ENV=production \
  -e MBFLOW_DATABASE_URL=postgres://... \
  -e MBFLOW_REDIS_URL=redis://... \
  -e MBFLOW_JWT_SECRET=... \
  -e MBFLOW_CORS_ALLOWED_ORIGINS=https://mbflow.example.com \
  mbflow:latest
```

For Kubernetes, point the liveness probe at `/health` and the readiness probe
at `/ready`, and set `terminationGracePeriodSeconds` above
`MBFLOW_SHUTDOWN_TIMEOUT` (default 30s) so running executions can drain.

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
curl http://localhost:8585/metrics
```

Metrics include:

- Database connection pool statistics
- Redis cache statistics

## Security

- API key authentication support
- Environment variable-based secrets
- No secrets in source code
- CORS support for browser clients

### Data isolation between users

Every `/api/v1` workflow, execution, trigger and file endpoint requires
authentication and is scoped to the caller:

| Record | Owner |
|--------|-------|
| Workflow (and its nodes, edges, attached resources) | `mbflow_workflows.created_by` |
| Trigger | its workflow's owner |
| Stored execution | its workflow's owner |
| Inline (ephemeral) execution | `mbflow_executions.created_by` |
| File | `mbflow_files.created_by`, or the owner of the workflow it is attached to |

- Lists only return the caller's records; administrators see everything.
- Reading, changing or deleting someone else's record returns **404**, so the
  API does not reveal whether it exists.
- Users can only attach their own resources (credentials, storage, keys) to a
  workflow, and a `sub_workflow` node can only call workflows of the same owner.
- The execution WebSocket (`/ws/executions?execution_id=...`) only streams
  executions the caller owns. Non-persisted inline executions are not stored,
  so only admins can watch them there; use webhooks or the gRPC stream instead.
- Records created before ownership was tracked (`created_by IS NULL`) are
  visible to administrators only. Migration `018_record_ownership` backfills
  file owners from their resource or workflow; assign any remaining legacy
  workflows by setting `created_by` directly in the database.
- The Service API (`/api/v1/service/*`, system keys) is a trusted
  administrative interface and is not restricted by these rules.

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
- Issues: [GitHub Issues](https://github.com/smilemakc/mbflow/go/issues)
- Discussions: [GitHub Discussions](https://github.com/smilemakc/mbflow/go/discussions)
