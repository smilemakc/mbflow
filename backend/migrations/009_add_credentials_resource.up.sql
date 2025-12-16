-- MBFlow Credentials Resource Migration
-- Adds support for storing encrypted credentials (API keys, OAuth tokens, etc.)
-- All sensitive data is encrypted using AES-256-GCM before storage

-- ============================================================================
-- RESOURCE_CREDENTIALS TABLE
-- Stores encrypted credentials with support for multiple credential types:
-- - api_key: Simple API key/token
-- - basic_auth: Username/password pair
-- - oauth2: OAuth2 credentials (client_id, client_secret, tokens)
-- - service_account: Service account JSON (e.g., Google Cloud)
-- - custom: Custom key-value pairs
-- ============================================================================
CREATE TABLE resource_credentials (
    resource_id UUID PRIMARY KEY REFERENCES resources(id) ON DELETE CASCADE,

    -- Credential type determines the structure of encrypted_data
    credential_type VARCHAR(50) NOT NULL,

    -- All sensitive data stored as encrypted JSON
    -- Structure depends on credential_type:
    -- api_key: {"api_key": "encrypted_value"}
    -- basic_auth: {"username": "encrypted", "password": "encrypted"}
    -- oauth2: {"client_id": "encrypted", "client_secret": "encrypted", "access_token": "encrypted", "refresh_token": "encrypted"}
    -- service_account: {"json_key": "encrypted_full_json"}
    -- custom: {"key1": "encrypted", "key2": "encrypted", ...}
    encrypted_data JSONB NOT NULL DEFAULT '{}',

    -- Service/provider this credential is for (e.g., "openai", "google", "github")
    provider VARCHAR(100),

    -- Optional expiration for tokens
    expires_at TIMESTAMP WITH TIME ZONE,

    -- Last time credential was used (for auditing)
    last_used_at TIMESTAMP WITH TIME ZONE,

    -- Number of times credential was used
    usage_count BIGINT NOT NULL DEFAULT 0,

    -- Pricing plan reference (for future billing)
    pricing_plan_id UUID REFERENCES pricing_plans(id) ON DELETE SET NULL,

    CONSTRAINT resource_credentials_type_check
        CHECK (credential_type IN ('api_key', 'basic_auth', 'oauth2', 'service_account', 'custom'))
);

-- Indexes for common queries
CREATE INDEX idx_resource_credentials_type ON resource_credentials(credential_type);
CREATE INDEX idx_resource_credentials_provider ON resource_credentials(provider) WHERE provider IS NOT NULL;
CREATE INDEX idx_resource_credentials_expires ON resource_credentials(expires_at) WHERE expires_at IS NOT NULL;
CREATE INDEX idx_resource_credentials_plan ON resource_credentials(pricing_plan_id);

-- Comments
COMMENT ON TABLE resource_credentials IS 'Encrypted credentials storage for API keys, OAuth tokens, service accounts, etc.';
COMMENT ON COLUMN resource_credentials.credential_type IS 'Type of credential: api_key, basic_auth, oauth2, service_account, custom';
COMMENT ON COLUMN resource_credentials.encrypted_data IS 'AES-256-GCM encrypted JSON containing credential values';
COMMENT ON COLUMN resource_credentials.provider IS 'Service/provider name (e.g., openai, google, github)';
COMMENT ON COLUMN resource_credentials.expires_at IS 'Token expiration time (if applicable)';
COMMENT ON COLUMN resource_credentials.last_used_at IS 'Last time credential was accessed for audit purposes';
COMMENT ON COLUMN resource_credentials.usage_count IS 'Number of times credential has been used';

-- ============================================================================
-- PRICING PLANS FOR CREDENTIALS
-- Free tier allows limited number of credentials
-- ============================================================================
INSERT INTO pricing_plans (resource_type, name, description, price_per_unit, unit, billing_period, pricing_model, is_free) VALUES
    (
        'credentials',
        'Free',
        'Free plan with up to 5 credentials',
        0,
        'credential',
        'monthly',
        'fixed',
        true
    ),
    (
        'credentials',
        'Basic',
        'Basic plan: up to 25 credentials for $2/month',
        2.00,
        'credential',
        'monthly',
        'fixed',
        false
    ),
    (
        'credentials',
        'Pro',
        'Professional plan: unlimited credentials for $10/month',
        10.00,
        'credential',
        'monthly',
        'fixed',
        false
    );

-- ============================================================================
-- CREDENTIAL AUDIT LOG
-- Tracks access to credentials for security auditing
-- ============================================================================
CREATE TABLE credential_audit_log (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    credential_id UUID NOT NULL REFERENCES resources(id) ON DELETE CASCADE,
    action VARCHAR(50) NOT NULL,
    actor_id UUID REFERENCES users(id) ON DELETE SET NULL,
    actor_type VARCHAR(50) NOT NULL DEFAULT 'user',
    ip_address INET,
    user_agent TEXT,
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),

    CONSTRAINT credential_audit_action_check
        CHECK (action IN ('created', 'read', 'updated', 'deleted', 'used_in_workflow'))
);

CREATE INDEX idx_credential_audit_credential ON credential_audit_log(credential_id, created_at DESC);
CREATE INDEX idx_credential_audit_actor ON credential_audit_log(actor_id, created_at DESC) WHERE actor_id IS NOT NULL;
CREATE INDEX idx_credential_audit_action ON credential_audit_log(action, created_at DESC);
CREATE INDEX idx_credential_audit_created ON credential_audit_log(created_at DESC);

COMMENT ON TABLE credential_audit_log IS 'Audit trail for credential access and modifications';
COMMENT ON COLUMN credential_audit_log.action IS 'Action performed: created, read, updated, deleted, used_in_workflow';
COMMENT ON COLUMN credential_audit_log.actor_type IS 'Type of actor: user, system, workflow';
