-- MBFlow Resource and Billing System Migration - Rollback
-- Removes all resource and billing tables in reverse dependency order

-- Drop triggers
DROP TRIGGER IF EXISTS trigger_resources_updated_at ON resources;
DROP TRIGGER IF EXISTS trigger_billing_accounts_updated_at ON billing_accounts;

-- Drop tables in reverse dependency order
DROP TABLE IF EXISTS transactions;
DROP TABLE IF EXISTS billing_accounts;
DROP TABLE IF EXISTS resource_file_storage;
DROP TABLE IF EXISTS resources;
DROP TABLE IF EXISTS pricing_plans;
