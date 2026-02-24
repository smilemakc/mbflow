-- Revert ephemeral execution changes

-- 1. mbflow_node_executions: restore original constraints
ALTER TABLE mbflow_node_executions
    DROP CONSTRAINT IF EXISTS chk_node_execution_identity;

DROP INDEX IF EXISTS idx_mbflow_node_executions_inline;
DROP INDEX IF EXISTS idx_mbflow_node_executions_stored;

-- Restore original unique index
CREATE UNIQUE INDEX idx_mbflow_node_executions_execution_node
    ON mbflow_node_executions (execution_id, node_id);

ALTER TABLE mbflow_node_executions
    DROP COLUMN IF EXISTS node_type;

ALTER TABLE mbflow_node_executions
    DROP COLUMN IF EXISTS node_name;

ALTER TABLE mbflow_node_executions
    DROP COLUMN IF EXISTS node_key;

-- Delete any inline node_executions before restoring NOT NULL
DELETE FROM mbflow_node_executions WHERE node_id IS NULL;

ALTER TABLE mbflow_node_executions
    ALTER COLUMN node_id SET NOT NULL;

-- 2. mbflow_executions: restore original constraints
ALTER TABLE mbflow_executions
    DROP CONSTRAINT IF EXISTS chk_execution_workflow_source;

-- Delete any inline executions before restoring NOT NULL
DELETE FROM mbflow_executions WHERE workflow_id IS NULL;

ALTER TABLE mbflow_executions
    DROP COLUMN IF EXISTS workflow_snapshot;

ALTER TABLE mbflow_executions
    DROP COLUMN IF EXISTS workflow_source;

ALTER TABLE mbflow_executions
    ALTER COLUMN workflow_id SET NOT NULL;
