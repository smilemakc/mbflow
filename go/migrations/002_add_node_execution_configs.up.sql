-- Migration: Add config and resolved_config fields to node_executions
-- This migration adds two new JSONB columns to track:
-- 1. config: Original node configuration before template resolution
-- 2. resolved_config: Final configuration after template resolution (used by executor)
--
-- These fields enable:
-- - Debugging: Compare original vs resolved config to understand template resolution
-- - Audit trail: Full visibility into what configuration was actually executed
-- - Reproducibility: Ability to re-run nodes with exact same configuration
-- - Troubleshooting: Identify issues in template resolution vs execution

-- Add config column (original node configuration)
ALTER TABLE mbflow_node_executions
ADD COLUMN config JSONB DEFAULT '{}';

-- Add resolved_config column (configuration after template resolution)
ALTER TABLE mbflow_node_executions
ADD COLUMN resolved_config JSONB DEFAULT '{}';

-- Create GIN index for config queries
CREATE INDEX idx_mbflow_node_executions_config ON mbflow_node_executions USING GIN (config);

-- Create GIN index for resolved_config queries
CREATE INDEX idx_mbflow_node_executions_resolved_config ON mbflow_node_executions USING GIN (resolved_config);

-- Add comments
COMMENT ON COLUMN mbflow_node_executions.config IS 'Original node configuration before template resolution';
COMMENT ON COLUMN mbflow_node_executions.resolved_config IS 'Final configuration after template resolution (actually used by executor)';