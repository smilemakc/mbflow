-- Migration: 012_add_service_keys
-- Description: Add service_keys table for API access tokens
-- Date: 2026-01-21

-- ============================================================================
-- SERVICE_KEYS TABLE
-- Stores API access tokens for authentication and authorization
-- Keys are hashed using bcrypt, never stored in plaintext
-- ============================================================================
CREATE TABLE service_keys (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),

    -- Owner and creator
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_by UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,

    -- Key metadata
    name VARCHAR(255) NOT NULL,
    description TEXT,

    -- Key prefix for identification (e.g., "mbf_live_" or "mbf_test_")
    -- Stored in plaintext for lookup purposes
    key_prefix VARCHAR(20) NOT NULL,

    -- Hashed key value (bcrypt)
    -- Original key shown only once at creation time
    key_hash TEXT NOT NULL,

    -- Key status and lifecycle
    status VARCHAR(20) NOT NULL DEFAULT 'active',
    last_used_at TIMESTAMP WITH TIME ZONE,
    usage_count BIGINT NOT NULL DEFAULT 0,
    expires_at TIMESTAMP WITH TIME ZONE,
    revoked_at TIMESTAMP WITH TIME ZONE,

    -- Timestamps
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),

    -- Constraints
    CONSTRAINT service_keys_status_check
        CHECK (status IN ('active', 'revoked')),
    CONSTRAINT service_keys_name_length
        CHECK (char_length(name) >= 1 AND char_length(name) <= 255),
    CONSTRAINT service_keys_prefix_format
        CHECK (key_prefix ~ '^sk_[a-zA-Z0-9]{5,}$')
);

-- Indexes for common queries
CREATE INDEX idx_service_keys_user_id ON service_keys(user_id) WHERE status = 'active';
CREATE INDEX idx_service_keys_prefix ON service_keys(key_prefix);
CREATE INDEX idx_service_keys_status ON service_keys(status) WHERE status = 'active';
CREATE INDEX idx_service_keys_created_by ON service_keys(created_by);
CREATE INDEX idx_service_keys_expires ON service_keys(expires_at) WHERE expires_at IS NOT NULL AND status = 'active';
CREATE INDEX idx_service_keys_last_used ON service_keys(last_used_at DESC NULLS LAST) WHERE status = 'active';

-- Trigger for updated_at
CREATE TRIGGER update_service_keys_updated_at
    BEFORE UPDATE ON service_keys
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- Comments
COMMENT ON TABLE service_keys IS 'API access tokens for authentication and authorization';
COMMENT ON COLUMN service_keys.user_id IS 'Owner of the service key';
COMMENT ON COLUMN service_keys.created_by IS 'User who created this service key';
COMMENT ON COLUMN service_keys.name IS 'Human-readable name for the service key';
COMMENT ON COLUMN service_keys.description IS 'Optional description of the key purpose';
COMMENT ON COLUMN service_keys.key_prefix IS 'Plaintext prefix for key identification (e.g., sk_a1b2c)';
COMMENT ON COLUMN service_keys.key_hash IS 'Bcrypt hash of the full key - original key never stored';
COMMENT ON COLUMN service_keys.status IS 'Key status: active or revoked';
COMMENT ON COLUMN service_keys.last_used_at IS 'Last time this key was used for authentication';
COMMENT ON COLUMN service_keys.usage_count IS 'Total number of times this key has been used';
COMMENT ON COLUMN service_keys.expires_at IS 'Optional expiration time for the key';
COMMENT ON COLUMN service_keys.revoked_at IS 'Timestamp when the key was revoked (if applicable)';
