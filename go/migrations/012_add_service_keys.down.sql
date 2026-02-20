-- Migration: 012_add_service_keys (rollback)
-- Description: Remove service_keys table

DROP TRIGGER IF EXISTS update_mbflow_service_keys_updated_at ON mbflow_service_keys;
DROP TABLE IF EXISTS mbflow_service_keys CASCADE;
