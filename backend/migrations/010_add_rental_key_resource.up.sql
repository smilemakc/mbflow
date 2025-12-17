-- Migration: 010_add_rental_key_resource
-- Description: Add rental_key resource type for LLM API key management
-- Date: 2025-12-17

-- Rental Key Resource table (type-specific data)
CREATE TABLE resource_rental_key (
    resource_id UUID PRIMARY KEY REFERENCES resources(id) ON DELETE CASCADE,

    -- Provider information
    provider VARCHAR(50) NOT NULL,
    encrypted_api_key TEXT NOT NULL,
    provider_config JSONB DEFAULT '{}',

    -- Usage limits
    daily_request_limit INTEGER,
    monthly_token_limit BIGINT,

    -- Current usage counters (reset periodically)
    requests_today INTEGER NOT NULL DEFAULT 0,
    tokens_this_month BIGINT NOT NULL DEFAULT 0,
    last_usage_reset_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),

    -- Total statistics
    total_requests BIGINT NOT NULL DEFAULT 0,

    -- Text tokens
    total_prompt_tokens BIGINT NOT NULL DEFAULT 0,
    total_completion_tokens BIGINT NOT NULL DEFAULT 0,

    -- Image tokens
    total_image_input_tokens BIGINT NOT NULL DEFAULT 0,
    total_image_output_tokens BIGINT NOT NULL DEFAULT 0,

    -- Audio tokens
    total_audio_input_tokens BIGINT NOT NULL DEFAULT 0,
    total_audio_output_tokens BIGINT NOT NULL DEFAULT 0,

    -- Video tokens
    total_video_input_tokens BIGINT NOT NULL DEFAULT 0,
    total_video_output_tokens BIGINT NOT NULL DEFAULT 0,

    -- Cost tracking
    total_cost NUMERIC(20, 8) NOT NULL DEFAULT 0,

    -- Timestamps and relations
    last_used_at TIMESTAMP WITH TIME ZONE,
    pricing_plan_id UUID REFERENCES pricing_plans(id) ON DELETE SET NULL,
    created_by UUID REFERENCES users(id) ON DELETE SET NULL,
    provisioner_type VARCHAR(50) NOT NULL DEFAULT 'manual',

    -- Constraints
    CONSTRAINT rental_key_provider_check
        CHECK (provider IN ('openai', 'anthropic', 'google_ai')),
    CONSTRAINT rental_key_provisioner_check
        CHECK (provisioner_type IN ('manual', 'auto_openai', 'auto_anthropic', 'auto_google'))
);

-- Usage log table for detailed tracking (multimodal)
CREATE TABLE rental_key_usage (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    rental_key_id UUID NOT NULL REFERENCES resources(id) ON DELETE CASCADE,
    model VARCHAR(100) NOT NULL,

    -- Text tokens
    prompt_tokens INTEGER NOT NULL DEFAULT 0,
    completion_tokens INTEGER NOT NULL DEFAULT 0,

    -- Image tokens
    image_input_tokens INTEGER NOT NULL DEFAULT 0,
    image_output_tokens INTEGER NOT NULL DEFAULT 0,

    -- Audio tokens
    audio_input_tokens INTEGER NOT NULL DEFAULT 0,
    audio_output_tokens INTEGER NOT NULL DEFAULT 0,

    -- Video tokens
    video_input_tokens INTEGER NOT NULL DEFAULT 0,
    video_output_tokens INTEGER NOT NULL DEFAULT 0,

    -- Computed total tokens
    total_tokens INTEGER GENERATED ALWAYS AS (
        prompt_tokens + completion_tokens +
        image_input_tokens + image_output_tokens +
        audio_input_tokens + audio_output_tokens +
        video_input_tokens + video_output_tokens
    ) STORED,

    -- Cost and context
    estimated_cost NUMERIC(20, 8) NOT NULL DEFAULT 0,
    execution_id UUID REFERENCES executions(id) ON DELETE SET NULL,
    workflow_id UUID REFERENCES workflows(id) ON DELETE SET NULL,
    node_id VARCHAR(255),

    -- Status tracking
    status VARCHAR(20) NOT NULL DEFAULT 'success',
    error_message TEXT,
    response_time_ms INTEGER,

    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),

    CONSTRAINT rental_key_usage_status_check
        CHECK (status IN ('success', 'failed', 'rate_limited'))
);

-- Indexes for resource_rental_key
CREATE INDEX idx_rental_key_provider ON resource_rental_key(provider);
CREATE INDEX idx_rental_key_pricing_plan ON resource_rental_key(pricing_plan_id);
CREATE INDEX idx_rental_key_created_by ON resource_rental_key(created_by);
CREATE INDEX idx_rental_key_last_used ON resource_rental_key(last_used_at DESC NULLS LAST);

-- Indexes for rental_key_usage
CREATE INDEX idx_rental_key_usage_key_time ON rental_key_usage(rental_key_id, created_at DESC);
CREATE INDEX idx_rental_key_usage_execution ON rental_key_usage(execution_id) WHERE execution_id IS NOT NULL;
CREATE INDEX idx_rental_key_usage_workflow ON rental_key_usage(workflow_id) WHERE workflow_id IS NOT NULL;
CREATE INDEX idx_rental_key_usage_created ON rental_key_usage(created_at DESC);
CREATE INDEX idx_rental_key_usage_model ON rental_key_usage(model);
CREATE INDEX idx_rental_key_usage_status ON rental_key_usage(status) WHERE status != 'success';

-- Add rental_key to resources type check
ALTER TABLE resources DROP CONSTRAINT IF EXISTS resources_type_check;
ALTER TABLE resources ADD CONSTRAINT resources_type_check
    CHECK (type IN ('file_storage', 'credentials', 'rental_key'));

-- Comments
COMMENT ON TABLE resource_rental_key IS 'Rental API keys for LLM providers - managed by platform, never exposed to users';
COMMENT ON COLUMN resource_rental_key.encrypted_api_key IS 'AES-256-GCM encrypted API key - NEVER exposed via any endpoint';
COMMENT ON COLUMN resource_rental_key.provider_config IS 'Provider-specific configuration (org_id, base_url, etc.)';
COMMENT ON COLUMN resource_rental_key.provisioner_type IS 'How the key was provisioned: manual by admin or auto by system';

COMMENT ON TABLE rental_key_usage IS 'Detailed multimodal usage logs for billing and analytics';
COMMENT ON COLUMN rental_key_usage.total_tokens IS 'Computed sum of all token types for this usage record';

-- Insert default pricing plans for rental_key
INSERT INTO pricing_plans (id, resource_type, name, description, price_per_unit, unit, billing_period, pricing_model, is_free, is_active)
VALUES
    (uuid_generate_v4(), 'rental_key', 'Free Trial', 'Trial plan: 100 requests/day, 10K tokens/month', 0, 'request', 'monthly', 'fixed', true, true),
    (uuid_generate_v4(), 'rental_key', 'Starter', 'Starter plan: 1000 requests/day, 100K tokens/month', 5.00, 'request', 'monthly', 'fixed', false, true),
    (uuid_generate_v4(), 'rental_key', 'PayAsYouGo', 'Pay-as-you-go: billed per token with 20% markup', 0.0000012, 'token', 'monthly', 'payg', false, true);
