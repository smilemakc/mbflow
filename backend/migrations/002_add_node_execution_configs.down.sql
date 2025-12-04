-- Rollback migration: Remove config and resolved_config fields from node_executions

-- Drop indexes first
DROP INDEX IF EXISTS idx_node_executions_resolved_config;
DROP INDEX IF EXISTS idx_node_executions_config;

-- Drop columns
ALTER TABLE node_executions
DROP COLUMN IF EXISTS resolved_config;

ALTER TABLE node_executions
DROP COLUMN IF EXISTS config;