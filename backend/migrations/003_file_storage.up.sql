-- MBFlow File Storage Migration
-- Creates files table for persistent file storage with access scopes

-- ============================================================================
-- FILES TABLE
-- Stores file metadata with access scope control
-- ============================================================================
CREATE TABLE files (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    storage_id VARCHAR(100) NOT NULL,
    name VARCHAR(500) NOT NULL,
    path VARCHAR(1000) NOT NULL,
    mime_type VARCHAR(100) NOT NULL,
    size BIGINT NOT NULL DEFAULT 0,
    checksum VARCHAR(64) NOT NULL,
    access_scope VARCHAR(20) NOT NULL DEFAULT 'workflow',
    tags TEXT[] DEFAULT '{}',
    metadata JSONB DEFAULT '{}',
    ttl_seconds INTEGER,
    expires_at TIMESTAMP WITH TIME ZONE,
    workflow_id UUID REFERENCES workflows(id) ON DELETE SET NULL,
    execution_id UUID REFERENCES executions(id) ON DELETE SET NULL,
    source_node_id VARCHAR(100),
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    
    CONSTRAINT files_access_scope_check 
        CHECK (access_scope IN ('workflow', 'edge', 'result')),
    CONSTRAINT files_size_check
        CHECK (size >= 0)
);

-- Indexes for common queries
CREATE INDEX idx_files_storage_id ON files(storage_id);
CREATE INDEX idx_files_workflow_id ON files(workflow_id) WHERE workflow_id IS NOT NULL;
CREATE INDEX idx_files_execution_id ON files(execution_id) WHERE execution_id IS NOT NULL;
CREATE INDEX idx_files_mime_type ON files(mime_type);
CREATE INDEX idx_files_access_scope ON files(access_scope);
CREATE INDEX idx_files_expires_at ON files(expires_at) WHERE expires_at IS NOT NULL;
CREATE INDEX idx_files_tags ON files USING GIN (tags);
CREATE INDEX idx_files_metadata ON files USING GIN (metadata);
CREATE INDEX idx_files_created_at ON files(created_at DESC);
CREATE INDEX idx_files_storage_workflow ON files(storage_id, workflow_id);
CREATE INDEX idx_files_checksum ON files(checksum);

-- Unique constraint on storage + path
CREATE UNIQUE INDEX idx_files_storage_path ON files(storage_id, path);

COMMENT ON TABLE files IS 'File storage metadata with access scope control';
COMMENT ON COLUMN files.storage_id IS 'Storage instance identifier';
COMMENT ON COLUMN files.path IS 'File path within the storage provider';
COMMENT ON COLUMN files.access_scope IS 'Access scope: workflow (all nodes), edge (connected nodes), result (output storage)';
COMMENT ON COLUMN files.tags IS 'Array of tags for filtering and organization';
COMMENT ON COLUMN files.ttl_seconds IS 'Time to live in seconds (NULL = no expiration)';
COMMENT ON COLUMN files.expires_at IS 'Calculated expiration timestamp';
COMMENT ON COLUMN files.source_node_id IS 'Node ID that created this file';


-- ============================================================================
-- STORAGE_CONFIG TABLE (Optional)
-- Stores storage instance configurations
-- ============================================================================
CREATE TABLE storage_configs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    storage_id VARCHAR(100) NOT NULL UNIQUE,
    storage_type VARCHAR(50) NOT NULL DEFAULT 'local',
    config JSONB NOT NULL DEFAULT '{}',
    max_size BIGINT DEFAULT 0,
    max_file_size BIGINT DEFAULT 0,
    default_ttl_seconds INTEGER,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    
    CONSTRAINT storage_configs_type_check 
        CHECK (storage_type IN ('local', 's3', 'gcs', 'azure'))
);

CREATE INDEX idx_storage_configs_storage_id ON storage_configs(storage_id);
CREATE INDEX idx_storage_configs_type ON storage_configs(storage_type);

COMMENT ON TABLE storage_configs IS 'Storage instance configurations';
COMMENT ON COLUMN storage_configs.storage_type IS 'Storage provider type: local, s3, gcs, azure';
COMMENT ON COLUMN storage_configs.config IS 'Provider-specific configuration (credentials, endpoints, etc.)';
