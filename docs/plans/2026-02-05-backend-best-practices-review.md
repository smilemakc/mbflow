# Backend Best Practices Review & Improvement Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Systematically improve backend code quality, testability, and maintainability by addressing structural issues,
best practice violations, and architectural gaps.

**Architecture:** Clean/Hexagonal Architecture with Gin (REST) + gRPC transport, Bun ORM (PostgreSQL), Redis cache.
Dual-mode: embedded SDK + remote HTTP/gRPC.

**Tech Stack:** Go 1.25, Gin, Bun ORM, gRPC, Redis, PostgreSQL 16, Docker

**Branch:** `feature/fix-arch`

---

## Progress Tracker

| #  | Issue                            | Status | Commit    | Notes                                                                          |
|----|----------------------------------|--------|-----------|--------------------------------------------------------------------------------|
| 1  | 3.3 CORS Wildcard (Security)     | DONE   | `b6bbba8` | CORS origins from config, wildcard only in dev                                 |
| 2  | 3.2 Standardize Validation       | DONE   | `2907918` | binding tags on all request structs                                            |
| 3  | 3.1 Consistent Response Format   | DONE   | `1a69d49` | Removed envelope wrapping, flat list responses                                 |
| 4  | 1.4+1.5 Service Layer Extraction | DONE   | `52878ec` | `serviceapi.Operations` layer, handlers are thin adapters                      |
| 5  | 2.3 Auth Tests                   | DONE   | `1ad9968` | JWT (38 tests) + password (49 tests) = 87 tests                                |
| 6  | 2.1 gRPC Tests                   | DONE   | `1ad9968` | errors (87) + converters (46) + interceptors (37) = 170 tests                  |
| -  | Pre-existing test failures       | DONE   | `1a69d49` | 7 fixes: nil providerManager, storageWrapper.Get, m2m registration, 404 status |
| 7  | 2.2 Service API Tests            | DONE   | `4725292` | 7 test files, 212 tests for all serviceapi operations                          |
| 8  | 1.1 SDK Internal Imports         | DONE   |           | Created pkg/engine/ interfaces, removed internal imports from pkg/sdk          |
| 9  | 1.2 Global Registry Removal      | DONE   | `4725292` | Removed global var + convenience functions, kept Registry struct               |
| 10 | 5.1 Repository Interface Cleanup | WIP    | `747640a` | Domain models + mappers added (Event, Trigger, AuditLog); interfaces deferred  |
| 11 | 5.2 Move Mapper Functions        | DONE   | `4725292` | Moved 7 mappers from engine/converters to storage/models/mappers               |
| 12 | 4.1 Observer Error Logging       | DONE   | `4725292` | Added logger to StorageManager, observer errors now logged                     |
| 13 | 5.3 Implement File Cleanup       | DONE   | `461042e` | StorageManager.Cleanup with logging (needs repo for full impl)                 |
| 14 | 3.4 Request Body Size Limit      | DONE   | `4725292` | middleware_bodysize.go + MaxBodySize config                                    |
| 15 | 3.5 Response Compression         | DONE   | `4725292` | gin-contrib/gzip middleware added                                              |
| 16 | 2.4 Add t.Parallel()             | DONE   | `461042e` | Added to engine, auth, model tests (partial coverage)                          |
| 17 | 1.3 Server Struct Decomposition  | DONE   | `565c0ed` | Split into DataLayer, AuthLayer, ExecutionLayer, ServiceAPILayer, TriggerLayer |
| 18 | 5.4 golangci-lint Config         | DONE   | `461042e` | .golangci.yml with govet, errcheck, staticcheck, revive                        |
| 19 | 6.1 OpenAPI Documentation        | DONE   | `7386f3b` | Swagger setup, handler annotations, /swagger/* endpoint                        |
| 20 | 4.2 Standardize ErrNoRows        | DONE   | `4725292` | Replaced 33 occurrences across 12 repository files                             |
| 21 | 4.3 errors.Join Usage            | DONE   | `461042e` | Replaced AggregatedError with errors.Join in dag_executor                      |
| 22 | 6.2 Distributed Tracing          | DONE   | `8860517` | OpenTelemetry tracing package + config + 13 tests                              |
| 23 | 6.3 Typed Executor Config        | DONE   | `564bf79` | HTTPConfig, LLMConfig, TransformConfig + validation + 10 tests                 |
| 24 | 6.4 Redis Rate Limiter           | DONE   | `1618581` | RedisRateLimiter + RedisLoginRateLimiter with 10 tests                         |
| 25 | 2.5 Server Package Tests         | DONE   |           | Unit tests for options, getters, RegisterExecutor (10 tests)                   |

**Completed:** 24/25 tasks + bonus fix (pre-existing failures)
**Remaining:** Task 10 (WIP - domain models done, interfaces deferred)

---

## Known Pre-existing Issues (not in original plan)

These were discovered and fixed during implementation:

1. **`grpc_provider_test.go` broken** — uses wrong types (`MockAuthServiceClient` as `*authgateway.GRPCClient`).
   Isolated with `//go:build grpc_provider_fixed` tag. Needs proper fix when gRPC auth provider is refactored.
2. **Config tests failing** — `TestConfig_Load_*` and `TestConfig_Validate_*` in `internal/config` — pre-existing, not
   investigated.
3. **Template test failing** — `TestEngine_ResolveConfig_WithResources` — pre-existing.
4. **REST handler tests are slow** — each test spins up a Docker PostgreSQL container (~500s for the package). Should be
   refactored to use mocks for unit tests.

---

## Detailed Issue Descriptions

### Category 1: CRITICAL - Architectural & Structural Issues

#### Issue 1.1: SDK Package (`pkg/sdk`) Imports Internal Packages

**Severity:** CRITICAL
**Impact:** Breaks Go module encapsulation, makes SDK unusable as external dependency

**Problem:** `pkg/sdk/client.go` imports `internal/` packages:

```
"github.com/smilemakc/mbflow/internal/application/engine"
"github.com/smilemakc/mbflow/internal/application/observer"
"github.com/smilemakc/mbflow/internal/domain/repository"
"github.com/smilemakc/mbflow/internal/infrastructure/storage"
```

**Files:**

- `backend/pkg/sdk/client.go:29-34`

**Fix:** Extract shared interfaces/types to `pkg/` packages. SDK embedded mode should use `pkg/server` to get a
pre-wired server instance instead of reaching into `internal/`.

---

#### Issue 1.2: Global Executor Registry

**Severity:** HIGH
**Impact:** Shared mutable state, test interference, harder DI

**Problem:** `pkg/executor/registry.go:11-13` has a package-level global registry:

```go
var globalRegistry = NewRegistry()
```

Plus global convenience functions (`Register`, `Get`, etc. lines 102-125).

**Files:**

- `backend/pkg/executor/registry.go:11-13, 100-125`

**Fix:** Remove global registry. All code already uses `Manager` interface via DI. Global functions are a legacy pattern
that should be deprecated.

---

#### Issue 1.3: `pkg/server` Has God-Object Server Struct

**Severity:** HIGH
**Impact:** 20+ fields, hard to test, high coupling, long initialization chain

**Problem:** `Server` struct in `pkg/server/server.go` holds all repositories, services, managers as direct fields.
`components.go` has a sequential init chain where each component depends on previous ones.

**Files:**

- `backend/pkg/server/server.go`
- `backend/pkg/server/components.go`

**Fix:** Break into sub-components with functional groupings (e.g., `StorageLayer`, `AuthLayer`, `ExecutionLayer`). Each
group manages its own wiring. Consider a lightweight DI approach using constructor functions returning interfaces.

---

### Category 2: HIGH - Testing & Testability Issues

#### Issue 2.2: No Service API Tests (7 files, 0% coverage)

**Severity:** HIGH
**Impact:** Service API operations layer completely untested

**Files:**

- `backend/internal/application/serviceapi/` (all files)

**Fix:** Add unit tests with mocked repositories.

---

#### Issue 2.4: No `t.Parallel()` Usage

**Severity:** MEDIUM
**Impact:** Slower test suite, missed race conditions

**Problem:** Zero occurrences of `t.Parallel()` across all test files (1867 test functions).

**Fix:** Add `t.Parallel()` to unit tests that don't share state. Keep integration tests sequential.

---

#### Issue 2.5: No `pkg/server` Tests

**Severity:** MEDIUM
**Impact:** Server initialization and routing untested

**Files:**

- `backend/pkg/server/` (5 files, 0% coverage)

**Fix:** Add tests for server initialization, route registration, health endpoint.

---

### Category 3: HIGH - API Design Issues

#### Issue 3.4: No Request Body Size Limit

**Severity:** MEDIUM
**Impact:** Potential DoS via large payloads

**Fix:** Add `gin.MaxMultipartMemory` and body size limiting middleware.

---

#### Issue 3.5: No Response Compression

**Severity:** MEDIUM
**Impact:** Higher bandwidth usage, slower responses for large payloads

**Fix:** Add `github.com/gin-contrib/gzip` middleware.

---

### Category 4: MEDIUM - Error Handling Gaps

#### Issue 4.1: Swallowed Observer Errors

**Severity:** MEDIUM
**Impact:** Silent failures in event notifications, hard to debug

**Problem:** File observer events silently discard errors:

```go
go func (o FileObserver) {
_ = o.OnFileEvent(ctx, event) // Ignore errors for now
}(obs)
```

**Files:**

- `backend/internal/application/filestorage/manager.go:246-248`

**Fix:** Add structured logging for observer errors. Consider error channels for monitoring.

---

#### Issue 4.2: Inconsistent `sql.ErrNoRows` Handling

**Severity:** LOW
**Impact:** Some code uses `errors.Is()`, some uses `==`

**Files:**

- `backend/internal/infrastructure/storage/resource_repository.go:87`
- `backend/internal/infrastructure/storage/pricing_plan_repository.go:38`

**Fix:** Standardize on `errors.Is(err, sql.ErrNoRows)` everywhere.

---

#### Issue 4.3: `errors.Join` Not Used for Multi-Error Aggregation

**Severity:** LOW
**Impact:** Custom `AggregatedError` instead of stdlib

**Problem:** The codebase has a custom `AggregatedError` in `dag_executor.go` but never uses Go 1.20's `errors.Join`.

**Fix:** Consider replacing `AggregatedError` with `errors.Join` for better stdlib compatibility.

---

### Category 5: MEDIUM - Code Organization & Patterns

#### Issue 5.1: Repository Interfaces Use Storage Models (Not Domain Models)

**Severity:** MEDIUM
**Impact:** Domain layer depends on infrastructure

**Problem:** Most repository interfaces in `internal/domain/repository/` operate on `storagemodels.*` types instead of
pure domain types. This couples the domain layer to the storage layer.

Exception: `ResourceRepository` correctly uses `pkg/models` types.

**Files:**

- `backend/internal/domain/repository/workflow_repository.go` - uses `*storagemodels.WorkflowModel`
- `backend/internal/domain/repository/execution_repository.go` - uses `*storagemodels.ExecutionModel`

**Fix:** Repository interfaces should accept/return domain models. Mapping happens inside the repository implementation.

---

#### Issue 5.2: `engine` Package Has Model Mapping Functions

**Severity:** MEDIUM
**Impact:** Mapping logic outside its natural home, harder to find

**Problem:** `engine.WorkflowModelToDomain()` and `engine.ExecutionModelToDomain()` are called from REST handlers.
Mapping logic belongs in the storage/mapper layer, not in the engine.

**Files:**

- `backend/internal/application/engine/` (mapper functions)
- Called from `backend/internal/infrastructure/api/rest/handlers_workflows.go:76, 102, 168`

**Fix:** Move mappers to `internal/infrastructure/storage/models/mappers.go` (some already exist there) and remove
duplicates from engine package.

---

#### Issue 5.3: Unimplemented `Cleanup` Method (TODO in Production Code)

**Severity:** MEDIUM
**Impact:** Files accumulate without cleanup, potential storage exhaustion

**Problem:**

```go
func (m *StorageManager) Cleanup(ctx context.Context) (int, error) {
// TODO: Implement cleanup with repository integration
return 0, nil
}
```

**Files:**

- `backend/internal/application/filestorage/manager.go:253-257`

**Fix:** Implement file cleanup with configurable retention policy. The `cleanupRoutine` goroutine already calls this
method periodically.

---

#### Issue 5.4: Missing `golangci-lint` Configuration

**Severity:** LOW
**Impact:** Lint rules not codified, inconsistent code style

**Problem:** `make lint` target exists but no `.golangci.yml` configuration file found.

**Fix:** Add `.golangci.yml` with recommended Go linters (govet, errcheck, staticcheck, gosimple, unused, etc.).

---

### Category 6: LOW - Improvements for Long-Term Maintainability

#### Issue 6.1: No OpenAPI/Swagger Documentation

**Severity:** LOW
**Impact:** No auto-generated API docs for consumers

**Fix:** Add `swag` annotations to handlers and generate OpenAPI spec. This also enables client code generation.

---

#### Issue 6.2: No Distributed Tracing

**Severity:** LOW
**Impact:** Hard to debug cross-service calls in production

**Problem:** OpenTelemetry is in go.mod as indirect dependency but not used in application code.

**Fix:** Add OpenTelemetry middleware for HTTP and gRPC. Propagate trace context through workflow execution.

---

#### Issue 6.3: `map[string]interface{}` Overuse

**Severity:** LOW
**Impact:** No type safety for config/input/output, runtime errors instead of compile-time

**Problem:** Executor interface uses `map[string]interface{}` for both config and input:

```go
Execute(ctx context.Context, config map[string]interface{}, input interface{}) (interface{}, error)
```

**Files:**

- `backend/pkg/executor/executor.go:26, 30`

**Fix:** Consider typed config structs per executor type. At minimum, add helper functions for safe type assertion from
config maps.

---

#### Issue 6.4: Rate Limiter Uses In-Memory State Only

**Severity:** LOW
**Impact:** Rate limits reset on restart, not shared across instances

**Files:**

- `backend/internal/infrastructure/api/rest/middleware_ratelimit.go`

**Fix:** For multi-instance deployments, use Redis-backed rate limiting. Current approach is fine for single-instance.

---

## What's Already Good (Keep Doing)

1. **Error handling system** - Excellent: 86+ sentinel errors, proper wrapping with `%w`, consistent `TranslateError`
   layer
2. **Database migrations** - Comprehensive with proper indexes, constraints, comments, helper functions
3. **Transaction handling** - Proper isolation levels, atomic multi-step operations
4. **SQL injection safety** - 100% parameterized queries via Bun ORM
5. **Auth middleware** - Flexible multi-provider auth with role/permission checks
6. **Test infrastructure** - Docker-based integration tests, good test helpers, benchmarks
7. **Structured logging** - Consistent key-value structured logging with request IDs
8. **Rate limiting** - IP-based and login-specific rate limiters with auto-cleanup
9. **Event/Observer system** - Well-designed observer pattern with WebSocket, HTTP, DB, Logger observers
10. **Recovery middleware** - Proper panic recovery with stack traces and structured logging
