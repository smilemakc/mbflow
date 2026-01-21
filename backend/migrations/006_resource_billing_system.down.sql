-- MBFlow Resource and Billing System Migration - Rollback
-- Removes all resource and billing tables in reverse dependency order

-- Drop triggers
DROP TRIGGER IF EXISTS trigger_mbflow_resources_updated_at ON mbflow_resources;
DROP TRIGGER IF EXISTS trigger_mbflow_billing_accounts_updated_at ON mbflow_billing_accounts;

-- Drop tables in reverse dependency order
DROP TABLE IF EXISTS mbflow_transactions;
DROP TABLE IF EXISTS mbflow_billing_accounts;
DROP TABLE IF EXISTS mbflow_resource_file_storage;
DROP TABLE IF EXISTS mbflow_resources;
DROP TABLE IF EXISTS mbflow_pricing_plans;
