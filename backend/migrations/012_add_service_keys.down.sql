-- Migration: 012_add_service_keys (rollback)
-- Description: Remove service_keys table

DROP TRIGGER IF EXISTS update_service_keys_updated_at ON service_keys;
DROP TABLE IF EXISTS service_keys CASCADE;
