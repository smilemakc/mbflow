-- Ephemeral execution support: make workflow_id nullable, add inline workflow tracking

-- 1. mbflow_executions: allow nullable workflow_id for inline workflows
ALTER TABLE mbflow_executions
    ALTER COLUMN workflow_id DROP NOT NULL;

ALTER TABLE mbflow_executions
    ADD COLUMN workflow_source VARCHAR(10) NOT NULL DEFAULT 'stored';

ALTER TABLE mbflow_executions
    ADD COLUMN workflow_snapshot JSONB;

ALTER TABLE mbflow_executions
    ADD CONSTRAINT chk_execution_workflow_source CHECK (
        (workflow_source = 'stored' AND workflow_id IS NOT NULL AND workflow_snapshot IS NULL)
            OR
        (workflow_source = 'inline' AND workflow_id IS NULL AND workflow_snapshot IS NOT NULL)
    );

-- 2. mbflow_node_executions: allow nullable node_id for inline workflows
ALTER TABLE mbflow_node_executions
    ALTER COLUMN node_id DROP NOT NULL;

ALTER TABLE mbflow_node_executions
    ADD COLUMN node_key VARCHAR(255);

ALTER TABLE mbflow_node_executions
    ADD COLUMN node_name VARCHAR(255);

ALTER TABLE mbflow_node_executions
    ADD COLUMN node_type VARCHAR(100);

-- Replace old unique index with partial indexes
DROP INDEX IF EXISTS idx_mbflow_node_executions_execution_node;

CREATE UNIQUE INDEX idx_mbflow_node_executions_stored
    ON mbflow_node_executions (execution_id, node_id)
    WHERE node_id IS NOT NULL;

CREATE UNIQUE INDEX idx_mbflow_node_executions_inline
    ON mbflow_node_executions (execution_id, node_key)
    WHERE node_key IS NOT NULL;

ALTER TABLE mbflow_node_executions
    ADD CONSTRAINT chk_node_execution_identity CHECK (
        node_id IS NOT NULL OR node_key IS NOT NULL
    );
