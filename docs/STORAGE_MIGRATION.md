# Migration Guide: MemoryStore to BunStore

This guide explains the changes made to switch from in-memory storage to PostgreSQL-backed BunStore as the default storage backend.

## What Changed

### 1. Default Storage Backend

**Before:**

```go
store := storage.NewMemoryStore()
```

**After:**

```go
store := storage.NewBunStore(cfg.DatabaseDSN)
```

### 2. Configuration

A new environment variable `DATABASE_DSN` is now used to configure the PostgreSQL connection.

**Default value:**

```
postgres://postgres:postgres@localhost:5432/mbflow?sslmode=disable
```

**Custom configuration:**

```bash
export DATABASE_DSN="postgres://username:password@host:port/database?sslmode=disable"
```

### 3. Schema Initialization

BunStore automatically initializes the database schema on startup:

```go
ctx := context.Background()
if err := store.InitSchema(ctx); err != nil {
    log.Error("failed to initialize database schema", "error", err)
    os.Exit(1)
}
```

## Setup Instructions

### Option 1: Using Docker (Recommended)

```bash
# Start PostgreSQL container
docker run --name mbflow-postgres \
  -e POSTGRES_PASSWORD=postgres \
  -e POSTGRES_DB=mbflow \
  -p 5432:5432 \
  -d postgres:15

# Verify it's running
docker ps | grep mbflow-postgres

# Start the server
go run cmd/server/main.go
```

### Option 2: Using Local PostgreSQL

**macOS (Homebrew):**

```bash
brew install postgresql@15
brew services start postgresql@15
createdb mbflow
go run cmd/server/main.go
```

**Ubuntu/Debian:**

```bash
sudo apt-get install postgresql-15
sudo systemctl start postgresql
sudo -u postgres createdb mbflow
DATABASE_DSN="postgres://postgres:postgres@localhost:5432/mbflow?sslmode=disable" go run cmd/server/main.go
```

**Windows:**

1. Download and install PostgreSQL from <https://www.postgresql.org/download/windows/>
2. Create database `mbflow` using pgAdmin or psql
3. Run: `set DATABASE_DSN=postgres://postgres:postgres@localhost:5432/mbflow?sslmode=disable && go run cmd/server/main.go`

### Option 3: Using Cloud PostgreSQL

For production deployments, you can use managed PostgreSQL services:

**AWS RDS:**

```bash
export DATABASE_DSN="postgres://username:password@your-rds-instance.region.rds.amazonaws.com:5432/mbflow?sslmode=require"
```

**Google Cloud SQL:**

```bash
export DATABASE_DSN="postgres://username:password@/mbflow?host=/cloudsql/project:region:instance&sslmode=disable"
```

**Azure Database for PostgreSQL:**

```bash
export DATABASE_DSN="postgres://username@servername:password@servername.postgres.database.azure.com:5432/mbflow?sslmode=require"
```

## Database Schema

BunStore creates the following tables automatically:

- `workflows` - Workflow definitions
- `executions` - Execution records
- `events` - Event sourcing events
- `nodes` - Workflow nodes
- `edges` - Workflow edges (connections)
- `triggers` - Workflow triggers
- `execution_states` - Execution state snapshots

## Benefits of BunStore

1. **Persistence** - Data survives server restarts
2. **ACID Compliance** - Guaranteed data consistency
3. **Event Sourcing** - Complete audit trail
4. **Scalability** - Handle large volumes of workflows and executions
5. **Production Ready** - Battle-tested PostgreSQL backend
6. **Transaction Support** - Atomic operations

## Rollback to MemoryStore (Development Only)

If you need to use MemoryStore for development or testing:

```go
// In cmd/server/main.go, replace:
store := storage.NewBunStore(cfg.DatabaseDSN)

// With:
store := storage.NewMemoryStore()
```

**Note:** This is not recommended for production use as all data will be lost on server restart.

## Troubleshooting

### Connection Refused

**Error:** `dial tcp [::1]:5432: connect: connection refused`

**Solution:** Ensure PostgreSQL is running:

```bash
# macOS
brew services list | grep postgresql

# Linux
sudo systemctl status postgresql

# Docker
docker ps | grep postgres
```

### Authentication Failed

**Error:** `pq: password authentication failed`

**Solution:** Check your credentials in the DSN:

```bash
# Test connection with psql
psql "postgres://postgres:postgres@localhost:5432/mbflow"
```

### Database Does Not Exist

**Error:** `pq: database "mbflow" does not exist`

**Solution:** Create the database:

```bash
# Using psql
createdb mbflow

# Or using SQL
psql -U postgres -c "CREATE DATABASE mbflow;"
```

### Schema Initialization Failed

**Error:** `failed to initialize database schema`

**Solution:** Ensure the user has CREATE TABLE permissions:

```sql
GRANT ALL PRIVILEGES ON DATABASE mbflow TO postgres;
```

## Performance Tuning

For production deployments, consider these PostgreSQL settings:

```sql
-- Increase connection pool
ALTER SYSTEM SET max_connections = 200;

-- Optimize for write-heavy workloads (event sourcing)
ALTER SYSTEM SET shared_buffers = '256MB';
ALTER SYSTEM SET effective_cache_size = '1GB';
ALTER SYSTEM SET maintenance_work_mem = '64MB';
ALTER SYSTEM SET checkpoint_completion_target = 0.9;
ALTER SYSTEM SET wal_buffers = '16MB';
ALTER SYSTEM SET default_statistics_target = 100;

-- Reload configuration
SELECT pg_reload_conf();
```

## Monitoring

Monitor your PostgreSQL instance:

```sql
-- Check active connections
SELECT count(*) FROM pg_stat_activity WHERE datname = 'mbflow';

-- Check table sizes
SELECT 
    schemaname,
    tablename,
    pg_size_pretty(pg_total_relation_size(schemaname||'.'||tablename)) AS size
FROM pg_tables
WHERE schemaname = 'public'
ORDER BY pg_total_relation_size(schemaname||'.'||tablename) DESC;

-- Check event count
SELECT COUNT(*) FROM events;
```

## Support

For issues or questions:

- GitHub Issues: <https://github.com/smilemakc/mbflow/issues>
- Documentation: See README.md
