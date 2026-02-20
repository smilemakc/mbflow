# MBFlow Database Migrations

This directory contains SQL migration files for the MBFlow workflow orchestration database.

## Schema Overview

The database schema implements a complete workflow orchestration system with Event Sourcing support:

### Core Tables

1. **workflows** - Workflow definitions
   - UUID primary key
   - Versioning support (name + version unique constraint)
   - Status: draft, active, archived
   - Soft delete support
   - JSONB metadata for extensibility

2. **nodes** - Workflow nodes (tasks/steps)
   - UUID primary key
   - Type: http, transform, llm, conditional, merge, split, delay, webhook
   - JSONB config for node-specific configuration
   - JSONB position for UI layout {x, y}

3. **edges** - DAG connections between nodes
   - UUID primary key
   - Links source and target nodes
   - Optional JSONB condition for conditional routing
   - Prevents self-reference (CHECK constraint)
   - Unique index on source+target combination

4. **executions** - Workflow execution instances
   - UUID primary key
   - Status: pending, running, completed, failed, cancelled, paused
   - JSONB input/output data
   - Timestamps for started_at, completed_at
   - Error text for failure debugging

5. **node_executions** - Individual node execution state
   - UUID primary key
   - Status: pending, running, completed, failed, skipped, retrying
   - Wave number for parallel execution tracking
   - Retry count for resilience tracking
   - JSONB input/output data

6. **events** - Event Sourcing log (immutable)
   - UUID primary key
   - Monotonically increasing sequence number
   - Event types: workflow_started, node_started, node_completed, etc.
   - JSONB payload with event data
   - Indexed by execution_id + sequence

7. **triggers** - Workflow trigger configurations
   - UUID primary key
   - Type: manual, cron, webhook, event, interval
   - JSONB config (cron expression, webhook URL, etc.)
   - Enabled flag for activation control
   - Last triggered timestamp

### Key Features

- **UUID Primary Keys**: All tables use UUID for distributed system compatibility
- **JSONB Columns**: Flexible schema with GIN indexes for querying
- **Event Sourcing**: Immutable event log for complete audit trail
- **Soft Deletes**: Workflows support soft delete via deleted_at
- **Foreign Keys**: Proper referential integrity with CASCADE/RESTRICT
- **Indexes**: Optimized for common query patterns
- **Wave-based Execution**: Support for parallel node execution
- **Versioning**: Workflows can have multiple versions

## Migration Files

- `001_init_schema.up.sql` - Initial schema creation
- `001_init_schema.down.sql` - Schema teardown

## Running Migrations

### Using the CLI Tool

```bash
# Run all pending migrations
./bin/migrate -database-url "postgres://user:pass@localhost:5432/mbflow" -command up

# Check migration status
./bin/migrate -database-url "postgres://user:pass@localhost:5432/mbflow" -command status

# Rollback last migration
./bin/migrate -database-url "postgres://user:pass@localhost:5432/mbflow" -command down

# Reset all migrations (CAUTION: drops all tables)
./bin/migrate -database-url "postgres://user:pass@localhost:5432/mbflow" -command reset
```

### Using Environment Variables

```bash
# Set DATABASE_URL
export DATABASE_URL="postgres://user:pass@localhost:5432/mbflow"

# Run migrations
./bin/migrate -command up
```

### Using Docker Compose

```bash
# Start PostgreSQL
docker compose up -d postgres

# Run migrations
export DATABASE_URL="postgres://mbflow:mbflow123@localhost:5432/mbflow"
./bin/migrate -command up
```

## Schema Diagram

```
workflows (1) ----< (N) nodes
   |                     |
   |                     |
   |              edges (connects nodes)
   |
   +----------< (N) triggers
   |
   +----------< (N) executions
                    |
                    +----------< (N) node_executions
                    |
                    +----------< (N) events
```

## Index Strategy

### Performance Indexes
- `workflows`: status, created_at, name, metadata (GIN)
- `nodes`: workflow_id, type, config (GIN)
- `edges`: workflow_id, source_node_id, target_node_id
- `executions`: workflow_id+created_at, status+created_at
- `node_executions`: execution_id, wave, execution_id+node_id (unique)
- `events`: execution_id+sequence (unique), event_type+created_at
- `triggers`: workflow_id+enabled, type, config (GIN)

### Unique Constraints
- `workflows`: (name, version)
- `edges`: (source_node_id, target_node_id)
- `node_executions`: (execution_id, node_id)
- `events`: (execution_id, sequence)

## Event Types

Common event types in the events table:
- `workflow_started` - Workflow execution began
- `workflow_completed` - Workflow execution completed successfully
- `workflow_failed` - Workflow execution failed
- `workflow_cancelled` - Workflow execution was cancelled
- `node_started` - Node execution started
- `node_completed` - Node execution completed
- `node_failed` - Node execution failed
- `node_skipped` - Node execution was skipped
- `node_retrying` - Node execution is being retried
- `wave_started` - Parallel wave execution started
- `wave_completed` - Parallel wave execution completed

## Best Practices

1. **Always use transactions** for multiple related operations
2. **Use soft deletes** for workflows to maintain referential integrity
3. **Append events** - never modify or delete events (immutable)
4. **Index JSONB** columns that are frequently queried
5. **Monitor sequence gaps** in events table for missing events
6. **Partition events table** for high-volume scenarios (optional)
7. **Use wave numbers** for parallel execution tracking
8. **Set retry_count** to track resilience patterns

## Backup Strategy

```bash
# Backup
pg_dump -h localhost -U mbflow -d mbflow > mbflow_backup.sql

# Restore
psql -h localhost -U mbflow -d mbflow < mbflow_backup.sql
```

## Monitoring Queries

```sql
-- Active executions
SELECT COUNT(*) FROM executions WHERE status = 'running';

-- Failed executions in last 24 hours
SELECT COUNT(*) FROM executions
WHERE status = 'failed' AND created_at > NOW() - INTERVAL '24 hours';

-- Average execution duration
SELECT AVG(completed_at - started_at) AS avg_duration
FROM executions WHERE status = 'completed';

-- Event count per execution
SELECT execution_id, COUNT(*) as event_count
FROM events GROUP BY execution_id ORDER BY event_count DESC LIMIT 10;

-- Top workflows by execution count
SELECT w.name, COUNT(e.id) as exec_count
FROM workflows w
LEFT JOIN executions e ON w.id = e.workflow_id
GROUP BY w.id, w.name
ORDER BY exec_count DESC LIMIT 10;
```
