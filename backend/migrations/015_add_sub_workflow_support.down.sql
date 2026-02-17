ALTER TABLE mbflow_executions DROP CONSTRAINT IF EXISTS mbflow_executions_parent_item_check;
DROP INDEX IF EXISTS idx_mbflow_executions_parent_node_item;
DROP INDEX IF EXISTS idx_mbflow_executions_parent_status;
DROP INDEX IF EXISTS idx_mbflow_executions_parent;
ALTER TABLE mbflow_executions
    DROP COLUMN IF EXISTS item_key,
    DROP COLUMN IF EXISTS item_index,
    DROP COLUMN IF EXISTS parent_node_id,
    DROP COLUMN IF EXISTS parent_execution_id;
