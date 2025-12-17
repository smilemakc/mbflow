-- Migration: 010_add_rental_key_resource (rollback)
-- Description: Remove rental_key resource type

-- Remove pricing plans for rental_key
DELETE FROM pricing_plans WHERE resource_type = 'rental_key';

-- Drop tables
DROP TABLE IF EXISTS rental_key_usage;
DROP TABLE IF EXISTS resource_rental_key;

-- Restore original type check constraint
ALTER TABLE resources DROP CONSTRAINT IF EXISTS resources_type_check;
ALTER TABLE resources ADD CONSTRAINT resources_type_check
    CHECK (type IN ('file_storage', 'credentials'));
