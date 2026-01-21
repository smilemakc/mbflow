-- Rollback credentials resource migration

-- Drop audit log table
DROP TABLE IF EXISTS mbflow_credential_audit_log;

-- Drop credentials table
DROP TABLE IF EXISTS mbflow_resource_credentials;

-- Remove pricing plans for credentials
DELETE FROM mbflow_pricing_plans WHERE resource_type = 'credentials';
