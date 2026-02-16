-- Sub-Workflow support: parent-child execution hierarchy

ALTER TABLE mbflow_executions
    ADD COLUMN parent_execution_id UUID REFERENCES mbflow_executions(id) ON DELETE CASCADE,
    ADD COLUMN parent_node_id VARCHAR(100),
    ADD COLUMN item_index INTEGER,
    ADD COLUMN item_key VARCHAR(500);

CREATE INDEX idx_mbflow_executions_parent
    ON mbflow_executions(parent_execution_id, item_index)
    WHERE parent_execution_id IS NOT NULL;

CREATE INDEX idx_mbflow_executions_parent_status
    ON mbflow_executions(parent_execution_id, status)
    WHERE parent_execution_id IS NOT NULL;

ALTER TABLE mbflow_executions
    ADD CONSTRAINT mbflow_executions_parent_item_check
    CHECK (
        (parent_execution_id IS NULL AND item_index IS NULL)
        OR
        (parent_execution_id IS NOT NULL AND item_index IS NOT NULL)
    );

CREATE UNIQUE INDEX idx_mbflow_executions_parent_node_item
    ON mbflow_executions(parent_execution_id, parent_node_id, item_index)
    WHERE parent_execution_id IS NOT NULL;

COMMENT ON COLUMN mbflow_executions.parent_execution_id IS 'Parent execution ID for sub-workflow children. NULL for root executions';
COMMENT ON COLUMN mbflow_executions.parent_node_id IS 'Node ID of sub_workflow node in parent that spawned this execution';
COMMENT ON COLUMN mbflow_executions.item_index IS 'Zero-based index of this item in the fan-out array';
COMMENT ON COLUMN mbflow_executions.item_key IS 'Optional human-readable key for this item';
