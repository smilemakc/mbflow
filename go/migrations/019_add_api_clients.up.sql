-- Migration: 019_add_api_clients
-- Description: Add API clients imported from OpenAPI 3.x and Swagger 2.0 specifications

-- ============================================================================
-- API_CLIENTS TABLE
-- One row per imported specification. Operations are stored normalized, so
-- workflow nodes of type "openapi" reference them by (client id, operation id)
-- without re-parsing the original document.
-- ============================================================================
CREATE TABLE mbflow_api_clients (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    owner_id UUID NOT NULL REFERENCES mbflow_users(id) ON DELETE CASCADE,

    name VARCHAR(255) NOT NULL,
    description TEXT,

    -- Specification metadata
    title TEXT NOT NULL,
    api_version TEXT,
    spec_version VARCHAR(20) NOT NULL,
    base_url TEXT,

    -- Client settings; secrets live in the credentials resource referenced by auth.credential_id
    servers JSONB NOT NULL DEFAULT '[]',
    security_schemes JSONB NOT NULL DEFAULT '[]',
    auth JSONB NOT NULL DEFAULT '{}',
    default_headers JSONB NOT NULL DEFAULT '{}',

    -- Normalized specification
    operations JSONB NOT NULL DEFAULT '[]',
    schemas JSONB NOT NULL DEFAULT '{}',
    warnings JSONB NOT NULL DEFAULT '[]',
    operation_count INTEGER NOT NULL DEFAULT 0,

    -- Original document, kept for download and re-import
    spec_raw TEXT NOT NULL,
    spec_size INTEGER NOT NULL DEFAULT 0,

    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMP WITH TIME ZONE,

    CONSTRAINT mbflow_api_clients_name_length
        CHECK (char_length(name) >= 1 AND char_length(name) <= 255)
);

CREATE INDEX idx_mbflow_api_clients_owner ON mbflow_api_clients(owner_id, created_at DESC) WHERE deleted_at IS NULL;

CREATE TRIGGER update_mbflow_api_clients_updated_at
    BEFORE UPDATE ON mbflow_api_clients
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

COMMENT ON TABLE mbflow_api_clients IS 'HTTP APIs imported from OpenAPI/Swagger specifications';
COMMENT ON COLUMN mbflow_api_clients.spec_version IS 'Specification format version: 2.0, 3.0.x or 3.1.x';
COMMENT ON COLUMN mbflow_api_clients.base_url IS 'Absolute base URL for requests; empty when the specification has no absolute server';
COMMENT ON COLUMN mbflow_api_clients.auth IS 'Auth settings (type, credential_id, key location); never contains secret values';
COMMENT ON COLUMN mbflow_api_clients.operations IS 'Normalized operations; schemas reference the schemas column via $ref';
COMMENT ON COLUMN mbflow_api_clients.schemas IS 'Component schemas reachable from operations, keyed by name';
COMMENT ON COLUMN mbflow_api_clients.spec_raw IS 'Uploaded specification document as is';
