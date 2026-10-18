-- Migration: 019_add_api_clients (rollback)
-- Description: Remove API clients table

DROP TRIGGER IF EXISTS update_mbflow_api_clients_updated_at ON mbflow_api_clients;
DROP TABLE IF EXISTS mbflow_api_clients CASCADE;
