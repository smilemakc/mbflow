# MBFlow

MBFlow is a workflow orchestration engine. You define workflows as DAGs of nodes (HTTP, transform, LLM and others) and MBFlow executes them. It has two parts:

- **Backend** (`go/`): a Go service that exposes a REST API with WebSocket updates. It stores data in PostgreSQL and uses Redis for caching and triggers. It also ships as an embeddable SDK.
- **UI** (`ui/`): a React + TypeScript single-page app built with Vite, for building and monitoring workflows.

## Repository layout

```
go/                  Go module github.com/smilemakc/mbflow/go
  cmd/server/        API server
  cmd/migrate/       Database migration tool (mbflow-migrate)
  cmd/cli/           CLI (workflows, users, admin, system keys)
  migrations/        SQL migrations
  pkg/, internal/    Engine, executors, API, storage
  sdk/               Go SDK (separate module)
  docker-compose.yml Full stack: postgres, redis, migrate, api, ui
ui/                  React/Vite frontend (served by nginx in Docker)
.github/             CI workflow and Dependabot config
```

## Quick start (Docker Compose)

```bash
cd go
cat > .env <<EOF
MBFLOW_JWT_SECRET=$(openssl rand -hex 32)
POSTGRES_PASSWORD=$(openssl rand -hex 16)
REDIS_PASSWORD=$(openssl rand -hex 16)
# Optional: key for encrypting stored credentials
# MBFLOW_ENCRYPTION_KEY=$(openssl rand -base64 32)
EOF
docker compose up -d --build
```

- UI: http://localhost:8080
- API: http://localhost:8585

The one-shot `mbflow-migrate` service applies database migrations. The API starts only after it finishes successfully.

## Local development

```bash
# Backend (needs PostgreSQL and Redis; see go/.env.example)
cd go
cp .env.example .env    # adjust MBFLOW_DATABASE_URL, MBFLOW_JWT_SECRET, ...
make migrate-up         # uses MBFLOW_DATABASE_URL (or DATABASE_URL)
make run-dev

# Frontend (dev server on http://localhost:3435, proxies to VITE_BACKEND_URL)
cd ui
cp .env.example .env    # VITE_BACKEND_URL=http://localhost:8585
npm install
npm run dev
```

Useful targets in `go/Makefile`: `make test`, `make lint` (golangci-lint v2), `make fmt` and `make docker-build`.

## Health endpoints

| Endpoint      | Purpose                                                                                                 |
|---------------|---------------------------------------------------------------------------------------------------------|
| `GET /health` | Liveness. Reports that the process is up. It does not check dependencies.                                 |
| `GET /ready`  | Readiness. Checks PostgreSQL and Redis and returns `503` if either is unavailable or the server is shutting down. |

## Production checklist

- [ ] Set `MBFLOW_ENV=production`. This turns on strict config validation: the server requires explicit `MBFLOW_CORS_ALLOWED_ORIGINS` (no wildcard) and a non-default `MBFLOW_DATABASE_URL`, and it rejects `MBFLOW_DB_DEBUG=true`.
- [ ] Use strong secrets for `MBFLOW_JWT_SECRET` (at least 32 characters), `POSTGRES_PASSWORD` and `REDIS_PASSWORD`. Set `MBFLOW_ENCRYPTION_KEY` to `openssl rand -base64 32` (without it, credentials and rental keys are disabled), back it up, and never rotate it without re-encrypting stored data.
- [ ] Terminate TLS at a reverse proxy, and list that proxy in `MBFLOW_TRUSTED_PROXIES` so client IPs are correct for rate limiting and audit logs.
- [ ] Set `MBFLOW_CORS_ALLOWED_ORIGINS` to the exact UI origin(s).
- [ ] Keep self-registration off with `MBFLOW_ALLOW_REGISTRATION=false`, which is the default in production.
- [ ] Users only see their own workflows, executions, triggers and files, while admins see everything (see "Data isolation between users" in [go/README.md](go/README.md)). After upgrading an existing installation, workflows without an owner are visible to admins only, so assign their `created_by` if regular users need them.
- [ ] Leave `MBFLOW_HTTP_ALLOW_PRIVATE_NETWORKS=false` unless workflows must call internal services. If they must, allow only specific hosts with `MBFLOW_HTTP_ALLOWED_HOSTS`.
- [ ] Back up the `postgres_data` volume (for example with `pg_dump`) and the `file_storage` volume (`/app/data/storage`).
- [ ] Create the first admin user. You can do this in either of two ways:
  - **CLI** (writes directly to the database; with Compose, PostgreSQL is published on `127.0.0.1:5566`):
    ```bash
    cd go
    DATABASE_URL="postgres://mbflow:$POSTGRES_PASSWORD@127.0.0.1:5566/mbflow?sslmode=disable" \
      go run ./cmd/cli admin create -email admin@example.com -username admin
    ```
    It prompts for the password if you omit `-password`.
  - **Bootstrap on startup**: set `MBFLOW_ADMIN_EMAIL` and `MBFLOW_ADMIN_PASSWORD` in the API container's environment. The server creates this admin if no user with that email exists. Remove the password from the environment afterwards.

## Known limitations

These were found while deploying and testing the full stack and are not fixed yet:

- **UI placeholders.** The Monitoring page, the "System health" block on the Dashboard and the profile/notification sections of Settings show static demo data, not real metrics. Treat them as mock-ups; use `/ready`, `/metrics` and logs for real monitoring.
- **Triggers in the UI.** The "Create trigger" dialog is a stub, and the triggers list expects a different data shape (`type: schedule`, `status`) than the API returns (`type: cron`, `enabled`). Manage triggers through the API for now.
- **Single API replica for triggers.** Cron/interval triggers run in every API replica, so running more than one replica fires them more than once. The login lockout counters are also per process. Run one API replica, or add leader election and the Redis-based limiter, before scaling out.
- **No execution concurrency limit or crash recovery.** Every execution runs in its own goroutine without a global cap, and executions that were running when the process was killed (not gracefully stopped) stay in `running`.
- **UI dependency advisories.** Two moderate advisories in `react-router` are fixed only in v7, which needs a migration.

## More documentation

- [go/README.md](go/README.md): backend architecture, configuration, API and SDK
- [ui/README.md](ui/README.md): frontend setup and development
- [go/.env.example](go/.env.example): every backend environment variable
