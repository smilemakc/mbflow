-- MBFlow Auth System Migration - Rollback
-- Removes all auth-related tables and objects

-- Drop functions
DROP FUNCTION IF EXISTS user_has_permission(UUID, TEXT);
DROP FUNCTION IF EXISTS get_user_permissions(UUID);
DROP FUNCTION IF EXISTS cleanup_expired_sessions();

-- Drop triggers
DROP TRIGGER IF EXISTS update_mbflow_users_updated_at ON mbflow_users;
DROP TRIGGER IF EXISTS update_mbflow_roles_updated_at ON mbflow_roles;

-- Note: Keep update_updated_at_column function as it may be used by other tables

-- Remove foreign key from workflows
ALTER TABLE mbflow_workflows DROP CONSTRAINT IF EXISTS fk_mbflow_workflows_created_by;

-- Drop tables in reverse dependency order
DROP TABLE IF EXISTS mbflow_audit_logs;
DROP TABLE IF EXISTS mbflow_user_roles;
DROP TABLE IF EXISTS mbflow_sessions;
DROP TABLE IF EXISTS mbflow_roles;
DROP TABLE IF EXISTS mbflow_users;
