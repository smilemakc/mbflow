-- Rollback: Remove resource_id from files table

DROP INDEX IF EXISTS idx_files_resource_id;
ALTER TABLE files DROP COLUMN IF EXISTS resource_id;

-- Restore original access_scope constraint (without 'resource')
ALTER TABLE files DROP CONSTRAINT IF EXISTS files_access_scope_check;
ALTER TABLE files ADD CONSTRAINT files_access_scope_check
    CHECK (access_scope IN ('workflow', 'edge', 'result'));
