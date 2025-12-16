-- Rollback credentials resource migration

-- Drop audit log table
DROP TABLE IF EXISTS credential_audit_log;

-- Drop credentials table
DROP TABLE IF EXISTS resource_credentials;

-- Remove pricing plans for credentials
DELETE FROM pricing_plans WHERE resource_type = 'credentials';
