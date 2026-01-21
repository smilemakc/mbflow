-- Restore unique constraint on workflow name + version (rollback)
-- WARNING: This will fail if duplicate names exist

ALTER TABLE mbflow_workflows ADD CONSTRAINT mbflow_workflows_name_version_unique UNIQUE (name, version);
