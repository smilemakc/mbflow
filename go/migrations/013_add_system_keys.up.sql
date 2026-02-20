-- Migration: 013_add_system_keys
-- Description: Add system_keys table for service-to-service authentication
-- Date: 2026-02-05

CREATE TABLE mbflow_system_keys (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(255) NOT NULL,
    description TEXT,
    service_name VARCHAR(100) NOT NULL,
    key_prefix VARCHAR(20) NOT NULL,
    key_hash TEXT NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'active',
    last_used_at TIMESTAMP WITH TIME ZONE,
    usage_count BIGINT NOT NULL DEFAULT 0,
    expires_at TIMESTAMP WITH TIME ZONE,
    revoked_at TIMESTAMP WITH TIME ZONE,
    created_by UUID NOT NULL REFERENCES mbflow_users(id) ON DELETE RESTRICT,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),

    CONSTRAINT mbflow_system_keys_status_check
        CHECK (status IN ('active', 'revoked')),
    CONSTRAINT mbflow_system_keys_name_length
        CHECK (char_length(name) >= 1 AND char_length(name) <= 255),
    CONSTRAINT mbflow_system_keys_service_name_length
        CHECK (char_length(service_name) >= 1 AND char_length(service_name) <= 100),
    CONSTRAINT mbflow_system_keys_prefix_format
        CHECK (key_prefix ~ '^sysk_[a-zA-Z0-9]{5,}$')
);

CREATE UNIQUE INDEX idx_mbflow_system_keys_prefix ON mbflow_system_keys(key_prefix);
CREATE INDEX idx_mbflow_system_keys_status ON mbflow_system_keys(status) WHERE status = 'active';
CREATE INDEX idx_mbflow_system_keys_service_name ON mbflow_system_keys(service_name);
CREATE INDEX idx_mbflow_system_keys_expires ON mbflow_system_keys(expires_at) WHERE expires_at IS NOT NULL AND status = 'active';

CREATE TRIGGER update_mbflow_system_keys_updated_at
    BEFORE UPDATE ON mbflow_system_keys
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

COMMENT ON TABLE mbflow_system_keys IS 'System keys for service-to-service authentication with full superadmin access';
COMMENT ON COLUMN mbflow_system_keys.service_name IS 'Identifier of the calling service (e.g. crm, billing)';
COMMENT ON COLUMN mbflow_system_keys.key_prefix IS 'Plaintext prefix for key identification (e.g., sysk_a1b2c)';
COMMENT ON COLUMN mbflow_system_keys.key_hash IS 'Bcrypt hash of the full key';
