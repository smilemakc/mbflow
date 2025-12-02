-- MBFlow Initial Schema Migration
-- Creates core tables for workflow orchestration with Event Sourcing support
-- All primary keys use UUID type for distributed system compatibility

-- Enable UUID extension
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- ============================================================================
-- WORKFLOWS TABLE
-- Stores workflow definitions with versioning support
-- ============================================================================
CREATE TABLE workflows (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(255) NOT NULL,
    description TEXT,
    status VARCHAR(50) NOT NULL DEFAULT 'draft',
    version INTEGER NOT NULL DEFAULT 1,
    variables JSONB DEFAULT '{}',
    metadata JSONB DEFAULT '{}',
    created_by UUID,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMP WITH TIME ZONE,

    CONSTRAINT workflows_status_check CHECK (status IN ('draft', 'active', 'archived')),
    CONSTRAINT workflows_name_version_unique UNIQUE (name, version)
);

CREATE INDEX idx_workflows_status ON workflows(status) WHERE deleted_at IS NULL;
CREATE INDEX idx_workflows_created_at ON workflows(created_at DESC);
CREATE INDEX idx_workflows_name ON workflows(name) WHERE deleted_at IS NULL;
CREATE INDEX idx_workflows_metadata ON workflows USING GIN (metadata);
CREATE INDEX idx_workflows_variables ON workflows USING GIN (variables);
CREATE INDEX idx_workflows_created_by ON workflows(created_by) WHERE created_by IS NOT NULL;

COMMENT ON TABLE workflows IS 'Workflow definitions with versioning and draft support';
COMMENT ON COLUMN workflows.status IS 'Workflow lifecycle state: draft, active, archived';
COMMENT ON COLUMN workflows.variables IS 'Workflow-level variables for template substitution';
COMMENT ON COLUMN workflows.metadata IS 'Additional workflow metadata stored as JSON';
COMMENT ON COLUMN workflows.created_by IS 'Optional reference to user who created the workflow';

-- ============================================================================
-- NODES TABLE
-- Stores workflow nodes (tasks/steps) with their configurations
-- Uses dual ID pattern: UUID for internal FK, logical node_id for API/templates
-- ============================================================================
CREATE TABLE nodes (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    node_id VARCHAR(100) NOT NULL,
    workflow_id UUID NOT NULL REFERENCES workflows(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    type VARCHAR(50) NOT NULL,
    config JSONB NOT NULL DEFAULT '{}',
    position JSONB DEFAULT '{"x": 0, "y": 0}',
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),

    CONSTRAINT nodes_type_check CHECK (type IN ('http', 'transform', 'llm', 'conditional', 'merge', 'split', 'delay', 'webhook')),
    CONSTRAINT uq_nodes_workflow_node_id UNIQUE (workflow_id, node_id)
);

CREATE INDEX idx_nodes_workflow_id ON nodes(workflow_id);
CREATE INDEX idx_nodes_node_id ON nodes(node_id);
CREATE INDEX idx_nodes_type ON nodes(type);
CREATE INDEX idx_nodes_config ON nodes USING GIN (config);

COMMENT ON TABLE nodes IS 'Workflow nodes representing individual tasks or operations';
COMMENT ON COLUMN nodes.id IS 'Internal UUID for foreign key references (hidden from API)';
COMMENT ON COLUMN nodes.node_id IS 'Logical node identifier unique within workflow (e.g., "step1", "process_data")';
COMMENT ON COLUMN nodes.type IS 'Node executor type: http, transform, llm, conditional, merge, split, delay, webhook';
COMMENT ON COLUMN nodes.config IS 'Node-specific configuration stored as JSON';
COMMENT ON COLUMN nodes.position IS 'UI positioning data as JSON {x, y}';

-- ============================================================================
-- EDGES TABLE
-- Stores workflow edges (connections between nodes) forming the DAG
-- Uses dual ID pattern: UUID for internal PK, logical IDs for node references
-- ============================================================================
CREATE TABLE edges (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    edge_id VARCHAR(100) NOT NULL,
    workflow_id UUID NOT NULL REFERENCES workflows(id) ON DELETE CASCADE,
    from_node_id VARCHAR(100) NOT NULL,
    to_node_id VARCHAR(100) NOT NULL,
    condition JSONB,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),

    CONSTRAINT edges_no_self_reference CHECK (from_node_id != to_node_id),
    CONSTRAINT uq_edges_workflow_edge_id UNIQUE (workflow_id, edge_id),
    CONSTRAINT fk_edges_source_node FOREIGN KEY (workflow_id, from_node_id) REFERENCES nodes(workflow_id, node_id) ON DELETE CASCADE,
    CONSTRAINT fk_edges_target_node FOREIGN KEY (workflow_id, to_node_id) REFERENCES nodes(workflow_id, node_id) ON DELETE CASCADE
);

CREATE INDEX idx_edges_workflow_id ON edges(workflow_id);
CREATE INDEX idx_edges_edge_id ON edges(edge_id);
CREATE INDEX idx_edges_from_node_id ON edges(workflow_id, from_node_id);
CREATE INDEX idx_edges_to_node_id ON edges(workflow_id, to_node_id);

COMMENT ON TABLE edges IS 'Directed edges connecting workflow nodes to form a DAG';
COMMENT ON COLUMN edges.id IS 'Internal UUID primary key (hidden from API)';
COMMENT ON COLUMN edges.edge_id IS 'Logical edge identifier unique within workflow (e.g., "edge1", "connect_a_to_b")';
COMMENT ON COLUMN edges.from_node_id IS 'Logical ID of source node (references nodes.node_id)';
COMMENT ON COLUMN edges.to_node_id IS 'Logical ID of target node (references nodes.node_id)';
COMMENT ON COLUMN edges.condition IS 'Optional conditional logic for edge traversal';

-- ============================================================================
-- TRIGGERS TABLE
-- Stores workflow trigger configurations
-- ============================================================================
CREATE TABLE triggers (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    workflow_id UUID NOT NULL REFERENCES workflows(id) ON DELETE CASCADE,
    type VARCHAR(50) NOT NULL,
    config JSONB NOT NULL DEFAULT '{}',
    enabled BOOLEAN NOT NULL DEFAULT true,
    last_triggered_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),

    CONSTRAINT triggers_type_check CHECK (type IN ('manual', 'cron', 'webhook', 'event', 'interval'))
);

CREATE INDEX idx_triggers_workflow_id ON triggers(workflow_id);
CREATE INDEX idx_triggers_type ON triggers(type);
CREATE INDEX idx_triggers_enabled ON triggers(workflow_id, enabled) WHERE enabled = true;
CREATE INDEX idx_triggers_config ON triggers USING GIN (config);

COMMENT ON TABLE triggers IS 'Workflow trigger configurations for automated execution';
COMMENT ON COLUMN triggers.type IS 'Trigger type: manual, cron, webhook, event, interval';
COMMENT ON COLUMN triggers.config IS 'Trigger-specific configuration (cron expression, webhook URL, etc.)';

-- ============================================================================
-- EXECUTIONS TABLE
-- Stores workflow execution instances with their state
-- ============================================================================
CREATE TABLE executions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    workflow_id UUID NOT NULL REFERENCES workflows(id) ON DELETE CASCADE,
    trigger_id UUID REFERENCES triggers(id) ON DELETE SET NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'pending',
    started_at TIMESTAMP WITH TIME ZONE,
    completed_at TIMESTAMP WITH TIME ZONE,
    input_data JSONB DEFAULT '{}',
    output_data JSONB,
    variables JSONB DEFAULT '{}',
    error TEXT,
    metadata JSONB DEFAULT '{}',
    strict_mode BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),

    CONSTRAINT executions_status_check CHECK (status IN ('pending', 'running', 'completed', 'failed', 'cancelled', 'paused'))
);

CREATE INDEX idx_executions_workflow_id ON executions(workflow_id, created_at DESC);
CREATE INDEX idx_executions_trigger_id ON executions(trigger_id) WHERE trigger_id IS NOT NULL;
CREATE INDEX idx_executions_status ON executions(status, created_at DESC);
CREATE INDEX idx_executions_created_at ON executions(created_at DESC);
CREATE INDEX idx_executions_completed_at ON executions(completed_at DESC) WHERE completed_at IS NOT NULL;
CREATE INDEX idx_executions_variables ON executions USING GIN (variables);
CREATE INDEX idx_executions_strict_mode ON executions (strict_mode);

COMMENT ON TABLE executions IS 'Workflow execution instances tracking runtime state';
COMMENT ON COLUMN executions.trigger_id IS 'Optional reference to trigger that initiated this execution';
COMMENT ON COLUMN executions.status IS 'Execution state: pending, running, completed, failed, cancelled, paused';
COMMENT ON COLUMN executions.input_data IS 'Input parameters for workflow execution';
COMMENT ON COLUMN executions.output_data IS 'Final output from completed execution';
COMMENT ON COLUMN executions.variables IS 'Runtime variables that override workflow variables for template substitution';
COMMENT ON COLUMN executions.metadata IS 'Additional execution metadata stored as JSON';
COMMENT ON COLUMN executions.strict_mode IS 'Whether to fail execution on first error or continue processing';

-- ============================================================================
-- NODE_EXECUTIONS TABLE
-- Stores individual node execution state within a workflow execution
-- ============================================================================
CREATE TABLE node_executions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    execution_id UUID REFERENCES executions(id) ON DELETE SET NULL,
    node_id UUID NOT NULL REFERENCES nodes(id) ON DELETE CASCADE,
    status VARCHAR(50) NOT NULL DEFAULT 'pending',
    started_at TIMESTAMP WITH TIME ZONE,
    completed_at TIMESTAMP WITH TIME ZONE,
    input_data JSONB DEFAULT '{}',
    output_data JSONB,
    error TEXT,
    retry_count INTEGER NOT NULL DEFAULT 0,
    wave INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),

    CONSTRAINT node_executions_status_check CHECK (status IN ('pending', 'running', 'completed', 'failed', 'skipped', 'retrying')),
    CONSTRAINT node_executions_retry_count_check CHECK (retry_count >= 0),
    CONSTRAINT node_executions_wave_check CHECK (wave >= 0)
);

CREATE INDEX idx_node_executions_execution_id ON node_executions(execution_id);
CREATE INDEX idx_node_executions_node_id ON node_executions(node_id);
CREATE INDEX idx_node_executions_status ON node_executions(status);
CREATE INDEX idx_node_executions_wave ON node_executions(execution_id, wave, created_at);
CREATE UNIQUE INDEX idx_node_executions_execution_node ON node_executions(execution_id, node_id);

COMMENT ON TABLE node_executions IS 'Individual node execution state within workflow runs';
COMMENT ON COLUMN node_executions.wave IS 'Execution wave number for parallel processing';
COMMENT ON COLUMN node_executions.retry_count IS 'Number of retry attempts for this node';

-- ============================================================================
-- EVENTS TABLE
-- Event Sourcing: Immutable log of all execution events
-- ============================================================================
CREATE TABLE events (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    execution_id UUID NOT NULL REFERENCES executions(id) ON DELETE CASCADE,
    event_type VARCHAR(100) NOT NULL,
    sequence BIGSERIAL NOT NULL,
    payload JSONB NOT NULL DEFAULT '{}',
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_events_execution_id ON events(execution_id, sequence);
CREATE INDEX idx_events_event_type ON events(event_type, created_at DESC);
CREATE INDEX idx_events_created_at ON events(created_at DESC);
CREATE UNIQUE INDEX idx_events_execution_sequence ON events(execution_id, sequence);

COMMENT ON TABLE events IS 'Event sourcing log of all execution events (immutable)';
COMMENT ON COLUMN events.event_type IS 'Event type: workflow_started, node_started, node_completed, workflow_failed, etc.';
COMMENT ON COLUMN events.sequence IS 'Monotonically increasing sequence number for ordering';

-- ============================================================================
-- PARTITIONING SETUP FOR EVENTS (Optional, for high-volume scenarios)
-- Uncomment if you expect millions of events
-- ============================================================================
-- ALTER TABLE events PARTITION BY RANGE (created_at);
-- CREATE TABLE events_2024 PARTITION OF events FOR VALUES FROM ('2024-01-01') TO ('2025-01-01');
-- CREATE TABLE events_2025 PARTITION OF events FOR VALUES FROM ('2025-01-01') TO ('2026-01-01');
