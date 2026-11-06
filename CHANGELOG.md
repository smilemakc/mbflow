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
- **Credentials in nodes.** Credentials attached to a workflow (or passed as `credential_ids` to an inline run, which now works and checks that the caller owns them) resolve in templates as `{{resource.<alias>.<field>}}`, e.g. an llm node's `api_key`. Decrypted values are redacted (`***`) from stored node configs, inputs, outputs, errors, execution results and all events.

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

### Found by functional testing

- A jq transform could crash the whole server ("concurrent map iteration and map write"): gojq rewrote the previous node's output while observers serialized it.
- Nodes after a conditional lost the data: the conditional returned a bare boolean. It now passes its input through with `result`.
- The `anthropic` LLM provider was accepted but not implemented. It now uses the Messages API with tools and structured output.
- The UI offered LLM providers the backend does not support, and its JSON response mode was ignored.
- Webhook executions stored callers' `Authorization`, `Cookie` and API-key headers in plain text.
- Against the real Anthropic API, LLM nodes with `temperature` or `top_p` failed on current Claude models (`temperature` is deprecated for this model). The provider now sends sampling parameters only to older models that accept them.
- Anthropic structured output (`response_format` `json_schema`) failed on Claude Opus 5.5, Sonnet 5.5 and Fable 5.1, which reject forced `tool_choice`. It now uses native structured outputs (`output_config.format`); objects get `additionalProperties: false` and unsupported constraints are dropped. Older models without structured outputs keep the forced-tool path.
- In the Anthropic tool-calling loop, the assistant turn is now sent back exactly as returned, so thinking blocks stay in front of the tool calls they led to.
- LLM tools of type `sub_workflow` (an agent calling another workflow) always failed with "workflow executor not configured". They now run the workflow with the execution engine; the workflow must belong to the same owner, inherits the caller's credentials, and tool workflows nest at most 5 deep.
- Creating or updating a workflow rejected edge conditions given as strings, the form `GET` returns, so a workflow with conditional edges could not be saved back after reading it. Both the string and `{"expression": "..."}` are accepted.
- The condition cache had a data race between concurrent condition evaluations.

### Tests and CI

- Go statement coverage went from 50% to 96.9%. CI fails below 90% (`go/scripts/coverage.sh`, `make test-cover`).
- `go/test/functional` runs 13 pipelines against a running stack: third-party APIs, branching, sub-workflows, and LLM pipelines (summary, structured output, tool-calling agents including one whose tool is a workflow, map-reduce over child workflows, a revision loop, routing by edge conditions, parallel fan-in, failure handling). They pass on Claude Haiku 5.5, Sonnet 5.5 and Opus 5.5.
- The UI has Vitest + Testing Library tests with a 90% gate on the tested modules (`npm run test:coverage`).
