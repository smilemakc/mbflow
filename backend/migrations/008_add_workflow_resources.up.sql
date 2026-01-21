-- MBFlow Workflow Resources Migration
-- Creates workflow_resources table for attaching resources to workflows with aliases
-- Enables users to reference resources in node configs via {{resource.alias}} syntax

-- ============================================================================
-- WORKFLOW_RESOURCES TABLE
-- Many-to-many relationship between workflows and resources with aliases
-- ============================================================================
CREATE TABLE mbflow_workflow_resources (
    workflow_id UUID NOT NULL REFERENCES mbflow_workflows(id) ON DELETE CASCADE,
    resource_id UUID NOT NULL REFERENCES mbflow_resources(id) ON DELETE CASCADE,
    alias VARCHAR(100) NOT NULL,
    access_type VARCHAR(20) NOT NULL DEFAULT 'read',
    assigned_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    assigned_by UUID REFERENCES mbflow_users(id) ON DELETE SET NULL,

    PRIMARY KEY (workflow_id, resource_id),
    CONSTRAINT mbflow_workflow_resources_alias_unique UNIQUE (workflow_id, alias),
    CONSTRAINT mbflow_workflow_resources_access_type_check CHECK (access_type IN ('read', 'write', 'admin'))
);

-- Indexes for efficient queries
CREATE INDEX idx_mbflow_workflow_resources_workflow ON mbflow_workflow_resources(workflow_id);
CREATE INDEX idx_mbflow_workflow_resources_resource ON mbflow_workflow_resources(resource_id);
CREATE INDEX idx_mbflow_workflow_resources_alias ON mbflow_workflow_resources(workflow_id, alias);

COMMENT ON TABLE mbflow_workflow_resources IS 'Resources attached to workflows with aliases for template references';
COMMENT ON COLUMN mbflow_workflow_resources.alias IS 'Alias used in {{resource.alias}} template syntax';
COMMENT ON COLUMN mbflow_workflow_resources.access_type IS 'Access level: read (query only), write (modify), admin (full control)';
COMMENT ON COLUMN mbflow_workflow_resources.assigned_at IS 'Timestamp when resource was attached to workflow';
COMMENT ON COLUMN mbflow_workflow_resources.assigned_by IS 'User who attached the resource (NULL if system)';
