-- Migration: 010_add_rental_key_resource (rollback)
-- Description: Remove rental_key resource type

-- Remove pricing plans for rental_key
DELETE FROM mbflow_pricing_plans WHERE resource_type = 'rental_key';

-- Drop tables
DROP TABLE IF EXISTS mbflow_rental_key_usage;
DROP TABLE IF EXISTS mbflow_resource_rental_key;

-- Restore original type check constraint
ALTER TABLE mbflow_resources DROP CONSTRAINT IF EXISTS mbflow_resources_type_check;
ALTER TABLE mbflow_resources ADD CONSTRAINT mbflow_resources_type_check
    CHECK (type IN ('file_storage', 'credentials'));
