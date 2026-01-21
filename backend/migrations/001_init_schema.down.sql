-- MBFlow Schema Teardown Migration
-- Drops all tables in reverse order of dependencies

-- Drop tables in reverse dependency order
-- Events must go first (depends on executions)
DROP TABLE IF EXISTS mbflow_events CASCADE;

-- Node executions (depends on executions and nodes)
DROP TABLE IF EXISTS mbflow_node_executions CASCADE;

-- Executions (depends on workflows and triggers)
DROP TABLE IF EXISTS mbflow_executions CASCADE;

-- Triggers (depends on workflows)
DROP TABLE IF EXISTS mbflow_triggers CASCADE;

-- Edges (depends on nodes and workflows)
DROP TABLE IF EXISTS mbflow_edges CASCADE;

-- Nodes (depends on workflows)
DROP TABLE IF EXISTS mbflow_nodes CASCADE;

-- Workflows (root table)
DROP TABLE IF EXISTS mbflow_workflows CASCADE;

-- Drop extension (optional, only if not used by other schemas)
-- DROP EXTENSION IF EXISTS "uuid-ossp";
