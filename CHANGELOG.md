# Changelog

## Unreleased — release candidate

### Upgrade notes

- Run the database migrations: `020` stores trigger name and description in their own columns (filled from the trigger config), and `021` widens session token columns to `TEXT`.
- New settings, all optional:

  | Variable | Default | Purpose |
  |---|---|---|
  | `MBFLOW_MAX_CONCURRENT_EXECUTIONS` | `100` | Executions running at once per replica (`0` = unlimited) |
  | `MBFLOW_EXECUTION_HEARTBEAT_INTERVAL` | `30s` | Lease renewal of running executions (`0` turns crash recovery off) |
  | `MBFLOW_EXECUTION_STALE_AFTER` | 3 × heartbeat | Lease age after which an execution counts as interrupted |
  | `MBFLOW_AUTH_GRPC_INSECURE` | `true`, but `false` in production | Plaintext connection to the gRPC auth gateway |
  | `MBFLOW_AUTH_GRPC_TLS_CA_FILE` | system roots | CA bundle for the gRPC auth gateway |
  | `MBFLOW_AUTH_GRPC_TLS_SERVER_NAME` | host name | TLS server name override for the gRPC auth gateway |

- Production now connects to the gRPC auth gateway over TLS unless `MBFLOW_AUTH_GRPC_INSECURE=true` is set.
- Logging out, changing a password or deactivating a user now invalidates access tokens immediately (built-in auth mode).
- Interval triggers fire on multiples of their interval rather than counting from process start.
- Executions cancelled by the shutdown deadline are recorded as `cancelled` instead of `failed`.

### Architecture

- **Several API replicas.** Each cron/interval tick and each event fires its trigger once across replicas (Redis claim). Login lockout counters are shared through Redis.
- **Execution concurrency limit.** A global cap per replica; queued executions stay `pending` until a slot frees up.
- **Crash recovery.** Running executions hold a lease. Any replica marks executions whose lease expired as failed, including at startup.
- **Cancel and retry.** Executions can be cancelled from any replica and retried (REST, service API, gRPC).
- **Token revocation.** Built-in access tokens are tied to their session.
- **Model write hooks run.** IDs, defaults, `updated_at` and the edge self-reference check are now applied on every write.
- **Remote SDK.** The `pkg/sdk` remote mode calls the real REST routes instead of stubs.
- **File storage metadata.** Listing, metadata, tags, cleanup of expired files and files stored by workflow nodes now go through the file metadata repository.

### Security fixes

- `GET /api/v1/files` passed `order_by`/`order_dir` straight into SQL.
- `POST /api/v1/files` accepted a `storage_id` that escaped the storage root.
- Webhook IP allowlists trusted `X-Forwarded-For` from any client.
- Replaying another account's deposit idempotency key returned that account's transaction.
- Audit log redaction missed secrets inside arrays.

### Other fixes

- Single node/edge API routes always returned 400 (parameter name mismatch).
- Trigger "run now", execution cancel and retry returned 501.
- About 15% of service/system key creations failed with 500 (non-alphanumeric prefix).
- Audit entries were written with a cancelled request context.
- Tracing could not be enabled (OpenTelemetry schema URL conflict).
- Workflow version was always reported as 0; YAML export returned only a header.
- The OpenAI Responses provider sent structured-output schemas in the Chat Completions shape.
- Telegram message entities were sliced by bytes instead of UTF-16 units; malformed entities crashed the parser.
- A deadlock in the cron scheduler shutdown, event triggers created after a start with none never firing, and several 500 responses for client errors.

### Tests and CI

- Go statement coverage went from 50% to 96.9%. CI fails below 90% (`go/scripts/coverage.sh`, `make test-cover`).
- The UI has Vitest + Testing Library tests with a 90% gate on the tested modules (`npm run test:coverage`).
