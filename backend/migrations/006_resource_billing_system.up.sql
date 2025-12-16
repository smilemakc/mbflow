-- MBFlow Resource and Billing System Migration
-- Creates tables for resource management, pricing plans, billing accounts, and transactions
-- Implements pay-as-you-go billing model with flexible pricing tiers

-- ============================================================================
-- PRICING_PLANS TABLE
-- Stores pricing plans for different resource types
-- Created first as it's referenced by other tables
-- ============================================================================
CREATE TABLE pricing_plans (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    resource_type VARCHAR(50) NOT NULL,
    name VARCHAR(100) NOT NULL,
    description TEXT,
    price_per_unit NUMERIC(20, 8) NOT NULL DEFAULT 0,
    unit VARCHAR(20) NOT NULL,
    storage_limit_bytes BIGINT,
    billing_period VARCHAR(20) NOT NULL DEFAULT 'monthly',
    pricing_model VARCHAR(20) NOT NULL DEFAULT 'fixed',
    is_free BOOLEAN NOT NULL DEFAULT false,
    is_active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),

    CONSTRAINT pricing_plans_pricing_model_check
        CHECK (pricing_model IN ('fixed', 'payg', 'tiered')),
    CONSTRAINT pricing_plans_billing_period_check
        CHECK (billing_period IN ('monthly', 'annual', 'one_time')),
    CONSTRAINT pricing_plans_price_check
        CHECK (price_per_unit >= 0),
    CONSTRAINT pricing_plans_storage_limit_check
        CHECK (storage_limit_bytes IS NULL OR storage_limit_bytes > 0)
);

CREATE INDEX idx_pricing_plans_resource_type ON pricing_plans(resource_type) WHERE is_active = true;
CREATE INDEX idx_pricing_plans_is_active ON pricing_plans(is_active);
CREATE INDEX idx_pricing_plans_pricing_model ON pricing_plans(pricing_model);
CREATE INDEX idx_pricing_plans_is_free ON pricing_plans(is_free) WHERE is_free = true;

COMMENT ON TABLE pricing_plans IS 'Pricing plans for different resource types (storage, compute, etc.)';
COMMENT ON COLUMN pricing_plans.resource_type IS 'Type of resource: file_storage, workflow_execution, api_calls, etc.';
COMMENT ON COLUMN pricing_plans.price_per_unit IS 'Price per unit (USD). 0 for free plans';
COMMENT ON COLUMN pricing_plans.unit IS 'Unit of measurement: storage (bytes), requests, GB, executions';
COMMENT ON COLUMN pricing_plans.storage_limit_bytes IS 'Storage limit in bytes (NULL for unlimited in payg)';
COMMENT ON COLUMN pricing_plans.billing_period IS 'Billing cycle: monthly, annual, one_time';
COMMENT ON COLUMN pricing_plans.pricing_model IS 'Pricing model: fixed (flat rate), payg (usage-based), tiered (volume discounts)';
COMMENT ON COLUMN pricing_plans.is_free IS 'True for free tier plans';
COMMENT ON COLUMN pricing_plans.is_active IS 'Whether plan is available for new subscriptions';

-- ============================================================================
-- RESOURCES TABLE
-- Polymorphic base table for all user resources
-- Supports soft deletes and extensible metadata
-- ============================================================================
CREATE TABLE resources (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    type VARCHAR(50) NOT NULL,
    owner_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    status VARCHAR(20) NOT NULL DEFAULT 'active',
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMP WITH TIME ZONE,

    CONSTRAINT resources_status_check
        CHECK (status IN ('active', 'suspended', 'deleted')),
    CONSTRAINT resources_name_check
        CHECK (length(name) >= 1)
);

CREATE INDEX idx_resources_owner ON resources(owner_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_resources_type ON resources(type) WHERE deleted_at IS NULL;
CREATE INDEX idx_resources_status ON resources(status);
CREATE INDEX idx_resources_created_at ON resources(created_at DESC);
CREATE INDEX idx_resources_deleted_at ON resources(deleted_at) WHERE deleted_at IS NOT NULL;
CREATE INDEX idx_resources_metadata ON resources USING GIN (metadata);

COMMENT ON TABLE resources IS 'Base table for all user resources (polymorphic pattern)';
COMMENT ON COLUMN resources.type IS 'Resource type: file_storage, database, api_key, webhook, etc.';
COMMENT ON COLUMN resources.owner_id IS 'User who owns this resource';
COMMENT ON COLUMN resources.status IS 'Resource state: active, suspended (unpaid), deleted';
COMMENT ON COLUMN resources.metadata IS 'Additional resource metadata stored as JSON';
COMMENT ON COLUMN resources.deleted_at IS 'Soft delete timestamp (NULL = not deleted)';

-- ============================================================================
-- RESOURCE_FILE_STORAGE TABLE
-- File storage specific attributes and usage tracking
-- Links to pricing plans for billing
-- ============================================================================
CREATE TABLE resource_file_storage (
    resource_id UUID PRIMARY KEY REFERENCES resources(id) ON DELETE CASCADE,
    storage_limit_bytes BIGINT NOT NULL DEFAULT 5242880,
    used_storage_bytes BIGINT NOT NULL DEFAULT 0,
    file_count INTEGER NOT NULL DEFAULT 0,
    pricing_plan_id UUID REFERENCES pricing_plans(id) ON DELETE SET NULL,

    CONSTRAINT resource_file_storage_used_storage_check
        CHECK (used_storage_bytes >= 0),
    CONSTRAINT resource_file_storage_file_count_check
        CHECK (file_count >= 0),
    CONSTRAINT resource_file_storage_limit_check
        CHECK (storage_limit_bytes > 0)
);

CREATE INDEX idx_resource_file_storage_plan ON resource_file_storage(pricing_plan_id);
CREATE INDEX idx_resource_file_storage_usage ON resource_file_storage(used_storage_bytes, storage_limit_bytes);

COMMENT ON TABLE resource_file_storage IS 'File storage specific attributes for resource polymorphism';
COMMENT ON COLUMN resource_file_storage.storage_limit_bytes IS 'Maximum storage allowed (default: 5MB for free tier)';
COMMENT ON COLUMN resource_file_storage.used_storage_bytes IS 'Current storage usage in bytes';
COMMENT ON COLUMN resource_file_storage.file_count IS 'Number of files stored';
COMMENT ON COLUMN resource_file_storage.pricing_plan_id IS 'Associated pricing plan (NULL = default free plan)';

-- ============================================================================
-- BILLING_ACCOUNTS TABLE
-- User billing accounts with balance tracking
-- One account per user, supports multiple currencies
-- ============================================================================
CREATE TABLE billing_accounts (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    balance NUMERIC(20, 8) NOT NULL DEFAULT 0,
    currency VARCHAR(3) NOT NULL DEFAULT 'USD',
    status VARCHAR(20) NOT NULL DEFAULT 'active',
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),

    CONSTRAINT billing_accounts_user_unique UNIQUE (user_id),
    CONSTRAINT billing_accounts_status_check
        CHECK (status IN ('active', 'suspended', 'closed')),
    CONSTRAINT billing_accounts_currency_check
        CHECK (currency ~ '^[A-Z]{3}$')
);

CREATE INDEX idx_billing_accounts_user ON billing_accounts(user_id);
CREATE INDEX idx_billing_accounts_status ON billing_accounts(status);
CREATE INDEX idx_billing_accounts_currency ON billing_accounts(currency);

COMMENT ON TABLE billing_accounts IS 'User billing accounts with prepaid balance';
COMMENT ON COLUMN billing_accounts.balance IS 'Current account balance (prepaid model)';
COMMENT ON COLUMN billing_accounts.currency IS 'Account currency (ISO 4217 code)';
COMMENT ON COLUMN billing_accounts.status IS 'Account state: active, suspended (negative balance), closed';

-- ============================================================================
-- TRANSACTIONS TABLE
-- Financial transaction history with idempotency support
-- Immutable audit trail of all financial operations
-- ============================================================================
CREATE TABLE transactions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    account_id UUID NOT NULL REFERENCES billing_accounts(id) ON DELETE CASCADE,
    type VARCHAR(20) NOT NULL,
    amount NUMERIC(20, 8) NOT NULL,
    currency VARCHAR(3) NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'completed',
    description TEXT,
    reference_type VARCHAR(50),
    reference_id UUID,
    idempotency_key VARCHAR(255) NOT NULL,
    balance_before NUMERIC(20, 8) NOT NULL,
    balance_after NUMERIC(20, 8) NOT NULL,
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),

    CONSTRAINT transactions_amount_check
        CHECK (amount > 0),
    CONSTRAINT transactions_type_check
        CHECK (type IN ('deposit', 'charge', 'refund', 'adjustment')),
    CONSTRAINT transactions_status_check
        CHECK (status IN ('pending', 'completed', 'failed', 'reversed')),
    CONSTRAINT transactions_currency_check
        CHECK (currency ~ '^[A-Z]{3}$')
);

CREATE UNIQUE INDEX idx_transactions_idempotency ON transactions(idempotency_key);
CREATE INDEX idx_transactions_account ON transactions(account_id, created_at DESC);
CREATE INDEX idx_transactions_created ON transactions(created_at DESC);
CREATE INDEX idx_transactions_type ON transactions(type, created_at DESC);
CREATE INDEX idx_transactions_status ON transactions(status, created_at DESC);
CREATE INDEX idx_transactions_reference ON transactions(reference_type, reference_id) WHERE reference_type IS NOT NULL;
CREATE INDEX idx_transactions_metadata ON transactions USING GIN (metadata);

COMMENT ON TABLE transactions IS 'Financial transaction history with idempotency support';
COMMENT ON COLUMN transactions.type IS 'Transaction type: deposit (top-up), charge (usage), refund, adjustment';
COMMENT ON COLUMN transactions.amount IS 'Transaction amount (always positive)';
COMMENT ON COLUMN transactions.status IS 'Transaction state: pending, completed, failed, reversed';
COMMENT ON COLUMN transactions.reference_type IS 'Referenced entity type (e.g., resource, execution, subscription)';
COMMENT ON COLUMN transactions.reference_id IS 'Referenced entity ID';
COMMENT ON COLUMN transactions.idempotency_key IS 'Unique key to prevent duplicate transactions';
COMMENT ON COLUMN transactions.balance_before IS 'Account balance before transaction';
COMMENT ON COLUMN transactions.balance_after IS 'Account balance after transaction';

-- ============================================================================
-- TRIGGERS: Auto-update updated_at timestamps
-- ============================================================================
CREATE TRIGGER trigger_resources_updated_at
    BEFORE UPDATE ON resources
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER trigger_billing_accounts_updated_at
    BEFORE UPDATE ON billing_accounts
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- ============================================================================
-- INITIAL DATA: Default Pricing Plans
-- Create standard pricing tiers for file storage
-- ============================================================================
INSERT INTO pricing_plans (resource_type, name, description, price_per_unit, unit, storage_limit_bytes, pricing_model, is_free) VALUES
    (
        'file_storage',
        'Free',
        'Free plan with 5 MB storage limit',
        0,
        'storage',
        5242880,
        'fixed',
        true
    ),
    (
        'file_storage',
        'Basic',
        'Basic plan: 1 GB for $1/month',
        1.00,
        'storage',
        1073741824,
        'fixed',
        false
    ),
    (
        'file_storage',
        'Pro',
        'Professional plan: 10 GB for $5/month',
        5.00,
        'storage',
        10737418240,
        'fixed',
        false
    ),
    (
        'file_storage',
        'PayAsYouGo',
        'Pay-as-you-go: $0.05/GB/month',
        0.05,
        'GB',
        NULL,
        'payg',
        false
    );
