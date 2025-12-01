# E2E Tests for MBFlow Workflow Trigger System

## Overview

This directory contains comprehensive End-to-End tests for the MBFlow user onboarding workflow trigger system.

## Test Files

1. **`workflow_trigger_e2e_test.go`** - Main E2E test suite with 8 test scenarios
2. **`workflow_trigger_bench_test.go`** - Performance benchmarks (8 benchmarks)
3. **`../../../test/fixtures/user_onboarding_test.json`** - Test workflow definition

## Test Scenarios

### 1. Happy Path
Tests complete successful workflow execution from event trigger to completion with all API calls.

### 2. Event Filtering - Should NOT Trigger
Table-driven test with 3 scenarios for events that should be filtered out:
- Wrong source (webhook instead of api)
- Wrong status (pending instead of active)
- Both wrong

### 3. Event Filtering - Should Trigger
Tests that correctly filtered events DO trigger the workflow.

### 4. Template Resolution
Validates that all template variables (`{{.user_id}}`, `{{.email}}`, etc.) are correctly resolved in HTTP request bodies.

### 5. Parallel Execution
Verifies wave-based parallel execution of nodes 2 and 3.

### 6. Execution Order
Validates DAG topological sort execution order.

### 7. Error Handling
Tests error propagation when nodes fail (invalid URL).

### 8. Concurrent Events
Processes 10 user creation events simultaneously.

## Prerequisites

### Docker
E2E tests use Dockertest to spin up PostgreSQL 16 containers.
**Docker must be running** before executing tests.

### macOS Docker Desktop Setup

If you're on macOS with Docker Desktop and getting `EOF` errors:

```bash
# Option 1: Set DOCKER_HOST environment variable
export DOCKER_HOST=unix://$HOME/.docker/run/docker.sock

# Option 2: Create symlink (one-time setup)
sudo ln -sf $HOME/.docker/run/docker.sock /var/run/docker.sock

# Option 3: Use Docker Desktop default context
docker context use desktop-linux
```

### Linux Docker Setup

On Linux, Docker should work out of the box. Ensure your user has Docker permissions:

```bash
sudo usermod -aG docker $USER
newgrp docker
```

## Running Tests

### All E2E Tests
```bash
cd /Users/balashov/PycharmProjects/mbflow/backend
go test -v ./internal/infrastructure/api/rest -run TestUserOnboardingWorkflow
```

### Specific Test
```bash
go test -v ./internal/infrastructure/api/rest -run TestUserOnboardingWorkflow_HappyPath
```

### With Race Detector
```bash
go test -race ./internal/infrastructure/api/rest -run TestUserOnboardingWorkflow
```

### Skip E2E Tests (Short Mode)
```bash
go test -short ./internal/infrastructure/api/rest
```

### Benchmarks
```bash
# Event processing latency
go test -bench=BenchmarkEventProcessingLatency ./internal/infrastructure/api/rest

# All benchmarks with memory stats
go test -bench=. -benchmem ./internal/infrastructure/api/rest
```

## Mock HTTP Servers

The tests include thread-safe mock servers for external APIs:

- **MockExampleAPI** - Simulates `api.example.com` (profiles, tasks)
- **MockSendGridAPI** - Simulates SendGrid email API
- **MockSegmentAPI** - Simulates Segment analytics API

All mocks record requests for assertion and are cleaned up automatically.

## Test Infrastructure

### Dockertest
- Automatically starts PostgreSQL 16 containers
- Runs migrations before each test
- Provides isolated test databases
- Auto-cleanup on test completion (5-minute expiry as safety net)

### Database
- Fresh PostgreSQL instance per test
- Full schema migration
- Isolated execution environment

### Test Fixtures
- JSON workflow definition with placeholder URLs
- Placeholders replaced at runtime with mock server URLs

## Troubleshooting

### Error: "Failed to connect to Docker"
- Ensure Docker Desktop is running
- Check `docker ps` works from command line
- Set `DOCKER_HOST` environment variable (macOS)

### Error: "EOF" when creating containers
- Docker socket permission issues
- Try setting `DOCKER_HOST=unix://$HOME/.docker/run/docker.sock`
- Or create symlink (see Prerequisites)

### Tests timeout
- Increase timeout: `-timeout 5m`
- Check Docker has enough resources (CPU/Memory)
- Verify PostgreSQL image downloads successfully

### Port conflicts
- Dockertest assigns random ports automatically
- No manual port configuration needed

## Known Limitations

1. **Event Trigger System**: Current tests simulate event triggering by directly calling ExecutionManager. Full pub/sub event triggering requires Redis integration (future enhancement).

2. **macOS Docker Socket**: Dockertest may require DOCKER_HOST configuration on macOS Docker Desktop.

3. **Test Duration**: Each test starts a PostgreSQL container, so full suite takes ~30-60 seconds.

## Test Coverage

The E2E tests cover:
- ✅ Workflow creation and persistence
- ✅ Event-triggered workflow execution
- ✅ Template variable resolution
- ✅ DAG execution order
- ✅ Parallel node execution
- ✅ HTTP node executor with external APIs
- ✅ Error handling and propagation
- ✅ Concurrent workflow executions
- ✅ Database operations (PostgreSQL)
- ✅ Event filtering logic

## Future Enhancements

- [ ] Redis pub/sub integration for true event-driven triggering
- [ ] Kubernetes test environment support
- [ ] Additional trigger types (cron, manual, API)
- [ ] Workflow versioning tests
- [ ] Rollback and retry scenarios
- [ ] Performance regression tests

## Bug Fixes

During test development, the following production bugs were discovered and fixed:

1. **JSONBMap.Value()** - Was returning `[]byte` instead of `string`, causing PostgreSQL JSONB errors
   - File: `internal/infrastructure/storage/models/types.go:13-23`

2. **WorkflowRepository.FindByIDWithRelations()** - Incorrect SQL table alias
   - File: `internal/infrastructure/storage/workflow_repository.go:288`
   - Fixed: `workflow.id` → `w.id`

## Contributing

When adding new tests:
1. Follow the existing test naming convention
2. Use table-driven tests where applicable
3. Ensure proper cleanup in defer statements
4. Add assertions for both success and error cases
5. Update this README with new test scenarios
