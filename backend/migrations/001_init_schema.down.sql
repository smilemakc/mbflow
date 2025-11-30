-- MBFlow Schema Teardown Migration
-- Drops all tables in reverse order of dependencies

-- Drop tables in reverse dependency order
-- Events must go first (depends on executions)
DROP TABLE IF EXISTS events CASCADE;

-- Node executions (depends on executions and nodes)
DROP TABLE IF EXISTS node_executions CASCADE;

-- Executions (depends on workflows and triggers)
DROP TABLE IF EXISTS executions CASCADE;

-- Triggers (depends on workflows)
DROP TABLE IF EXISTS triggers CASCADE;

-- Edges (depends on nodes and workflows)
DROP TABLE IF EXISTS edges CASCADE;

-- Nodes (depends on workflows)
DROP TABLE IF EXISTS nodes CASCADE;

-- Workflows (root table)
DROP TABLE IF EXISTS workflows CASCADE;

-- Drop extension (optional, only if not used by other schemas)
-- DROP EXTENSION IF EXISTS "uuid-ossp";
