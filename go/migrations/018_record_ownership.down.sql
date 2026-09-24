DROP INDEX IF EXISTS idx_mbflow_files_created_by;
ALTER TABLE mbflow_files DROP COLUMN IF EXISTS created_by;

DROP INDEX IF EXISTS idx_mbflow_executions_created_by;
ALTER TABLE mbflow_executions DROP COLUMN IF EXISTS created_by;
