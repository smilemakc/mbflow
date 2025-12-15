-- Remove unique constraint on workflow name + version
-- Workflow names should not be unique - only ID is unique
-- This allows users to create multiple workflows with the same name

ALTER TABLE workflows DROP CONSTRAINT IF EXISTS workflows_name_version_unique;

-- Add comment explaining the change
COMMENT ON TABLE workflows IS 'Workflow definitions with versioning. Names are not required to be unique.';
