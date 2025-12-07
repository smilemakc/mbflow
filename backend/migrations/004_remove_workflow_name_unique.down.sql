-- Restore unique constraint on workflow name + version (rollback)
-- WARNING: This will fail if duplicate names exist

ALTER TABLE workflows ADD CONSTRAINT workflows_name_version_unique UNIQUE (name, version);
