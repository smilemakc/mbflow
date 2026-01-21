-- Remove unique constraint on workflow name + version
-- Workflow names should not be unique - only ID is unique
-- This allows users to create multiple workflows with the same name

ALTER TABLE mbflow_workflows DROP CONSTRAINT IF EXISTS mbflow_workflows_name_version_unique;

-- Add comment explaining the change
COMMENT ON TABLE mbflow_workflows IS 'Workflow definitions with versioning. Names are not required to be unique.';
