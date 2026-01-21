-- Rollback: Remove resource_id from files table

DROP INDEX IF EXISTS idx_mbflow_files_resource_id;
ALTER TABLE mbflow_files DROP COLUMN IF EXISTS resource_id;

-- Restore original access_scope constraint (without 'resource')
ALTER TABLE mbflow_files DROP CONSTRAINT IF EXISTS mbflow_files_access_scope_check;
ALTER TABLE mbflow_files ADD CONSTRAINT mbflow_files_access_scope_check
    CHECK (access_scope IN ('workflow', 'edge', 'result'));
