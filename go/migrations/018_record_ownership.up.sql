-- ============================================================================
-- Per-user ownership for executions and files
--
-- Ownership model:
--   * workflows      -> mbflow_workflows.created_by (existing column)
--   * triggers       -> owned through their workflow (workflow_id NOT NULL)
--   * executions     -> stored executions are owned through their workflow;
--                       inline (ephemeral) executions have no workflow, so
--                       they record the caller in created_by
--   * files          -> created_by (new); backfilled below
--
-- Rows whose owner cannot be determined stay NULL. NULL-owned records are
-- visible only to administrators; an admin can reassign them if needed.
-- ============================================================================

ALTER TABLE mbflow_executions
    ADD COLUMN created_by UUID REFERENCES mbflow_users(id) ON DELETE SET NULL;

CREATE INDEX idx_mbflow_executions_created_by
    ON mbflow_executions(created_by) WHERE created_by IS NOT NULL;

COMMENT ON COLUMN mbflow_executions.created_by IS
    'User who started an inline (ephemeral) execution; stored executions are owned via their workflow';

ALTER TABLE mbflow_files
    ADD COLUMN created_by UUID REFERENCES mbflow_users(id) ON DELETE SET NULL;

CREATE INDEX idx_mbflow_files_created_by
    ON mbflow_files(created_by) WHERE created_by IS NOT NULL;

COMMENT ON COLUMN mbflow_files.created_by IS
    'User who uploaded the file (NULL = legacy/system file, visible to admins only)';

-- Backfill file owners: resource owner first, then the owning workflow's creator.
UPDATE mbflow_files f
SET created_by = r.owner_id
FROM mbflow_resources r
WHERE f.resource_id = r.id
  AND f.created_by IS NULL;

UPDATE mbflow_files f
SET created_by = w.created_by
FROM mbflow_workflows w
WHERE f.workflow_id = w.id
  AND f.created_by IS NULL
  AND w.created_by IS NOT NULL;
