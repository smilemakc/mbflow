-- Migration: Add resource_id to files table for FileStorage resource integration
-- This links files with FileStorage resources for quota and billing management

-- Add resource_id column to files table
ALTER TABLE files
    ADD COLUMN resource_id UUID REFERENCES resources (id) ON DELETE CASCADE;

-- Create index for resource_id lookups
CREATE INDEX idx_files_resource_id ON files (resource_id) WHERE resource_id IS NOT NULL;

-- Update access_scope constraint to include 'resource' scope
ALTER TABLE files
    DROP CONSTRAINT IF EXISTS files_access_scope_check;
ALTER TABLE files
    ADD CONSTRAINT files_access_scope_check
        CHECK (access_scope IN ('workflow', 'edge', 'result', 'resource'));

-- Add comment
COMMENT ON COLUMN files.resource_id IS 'Reference to FileStorage resource (NULL for legacy workflow files)';