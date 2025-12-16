-- MBFlow Auth System Migration
-- Adds users, sessions, roles, and permissions tables
-- Implements RBAC (Role-Based Access Control) with support for external Auth Gateway

-- Enable pgcrypto for additional crypto functions (optional)
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- ============================================================================
-- USERS TABLE
-- Core user accounts with authentication credentials
-- ============================================================================
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    email VARCHAR(255) NOT NULL,
    username VARCHAR(100) NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    full_name VARCHAR(255),
    is_active BOOLEAN NOT NULL DEFAULT true,
    is_admin BOOLEAN NOT NULL DEFAULT false,
    email_verified BOOLEAN NOT NULL DEFAULT false,
    email_verification_token VARCHAR(255),
    password_reset_token VARCHAR(255),
    password_reset_expires_at TIMESTAMP WITH TIME ZONE,
    failed_login_attempts INTEGER NOT NULL DEFAULT 0,
    locked_until TIMESTAMP WITH TIME ZONE,
    external_provider VARCHAR(100),
    external_id VARCHAR(255),
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    last_login_at TIMESTAMP WITH TIME ZONE,
    deleted_at TIMESTAMP WITH TIME ZONE,

    CONSTRAINT users_email_unique UNIQUE (email),
    CONSTRAINT users_username_unique UNIQUE (username),
    CONSTRAINT users_email_check CHECK (email ~* '^[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Z|a-z]{2,}$'),
    CONSTRAINT users_username_check CHECK (username ~ '^[a-zA-Z0-9_-]{3,50}$'),
    CONSTRAINT users_external_unique UNIQUE (external_provider, external_id)
);

CREATE INDEX idx_users_email ON users(email) WHERE deleted_at IS NULL;
CREATE INDEX idx_users_username ON users(username) WHERE deleted_at IS NULL;
CREATE INDEX idx_users_is_active ON users(is_active) WHERE deleted_at IS NULL;
CREATE INDEX idx_users_is_admin ON users(is_admin) WHERE deleted_at IS NULL AND is_admin = true;
CREATE INDEX idx_users_created_at ON users(created_at DESC);
CREATE INDEX idx_users_external ON users(external_provider, external_id) WHERE external_provider IS NOT NULL;
CREATE INDEX idx_users_metadata ON users USING GIN (metadata);

COMMENT ON TABLE users IS 'System users with authentication credentials';
COMMENT ON COLUMN users.password_hash IS 'Bcrypt hashed password';
COMMENT ON COLUMN users.failed_login_attempts IS 'Count of failed login attempts for rate limiting';
COMMENT ON COLUMN users.locked_until IS 'Account locked until this timestamp due to failed attempts';
COMMENT ON COLUMN users.external_provider IS 'External auth provider name (e.g., keycloak, auth0)';
COMMENT ON COLUMN users.external_id IS 'User ID from external auth provider';

-- ============================================================================
-- SESSIONS TABLE
-- User authentication sessions with JWT tokens
-- ============================================================================
CREATE TABLE sessions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token VARCHAR(500) NOT NULL,
    refresh_token VARCHAR(500),
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
    refresh_expires_at TIMESTAMP WITH TIME ZONE,
    ip_address INET,
    user_agent TEXT,
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    last_activity_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),

    CONSTRAINT sessions_token_unique UNIQUE (token)
);

CREATE INDEX idx_sessions_user_id ON sessions(user_id);
CREATE INDEX idx_sessions_token ON sessions(token);
CREATE INDEX idx_sessions_refresh_token ON sessions(refresh_token) WHERE refresh_token IS NOT NULL;
CREATE INDEX idx_sessions_expires_at ON sessions(expires_at);
CREATE INDEX idx_sessions_last_activity ON sessions(last_activity_at DESC);

COMMENT ON TABLE sessions IS 'User authentication sessions with JWT tokens';
COMMENT ON COLUMN sessions.token IS 'JWT access token (hashed for security)';
COMMENT ON COLUMN sessions.refresh_token IS 'JWT refresh token for token renewal';
COMMENT ON COLUMN sessions.last_activity_at IS 'Last activity timestamp for session timeout';

-- ============================================================================
-- ROLES TABLE
-- User roles with permission sets
-- ============================================================================
CREATE TABLE roles (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(100) NOT NULL,
    description TEXT,
    is_system BOOLEAN NOT NULL DEFAULT false,
    permissions TEXT[] NOT NULL DEFAULT '{}',
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),

    CONSTRAINT roles_name_unique UNIQUE (name),
    CONSTRAINT roles_name_check CHECK (name ~ '^[a-zA-Z0-9_-]+$')
);

CREATE INDEX idx_roles_name ON roles(name);
CREATE INDEX idx_roles_is_system ON roles(is_system);
CREATE INDEX idx_roles_permissions ON roles USING GIN (permissions);

COMMENT ON TABLE roles IS 'User roles with permission sets';
COMMENT ON COLUMN roles.is_system IS 'System roles cannot be deleted or renamed';
COMMENT ON COLUMN roles.permissions IS 'Array of permission strings (e.g., workflow:create, user:manage)';

-- ============================================================================
-- USER_ROLES TABLE (Many-to-Many)
-- Associates users with their roles
-- ============================================================================
CREATE TABLE user_roles (
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role_id UUID NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
    assigned_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    assigned_by UUID REFERENCES users(id) ON DELETE SET NULL,

    PRIMARY KEY (user_id, role_id)
);

CREATE INDEX idx_user_roles_user_id ON user_roles(user_id);
CREATE INDEX idx_user_roles_role_id ON user_roles(role_id);
CREATE INDEX idx_user_roles_assigned_by ON user_roles(assigned_by) WHERE assigned_by IS NOT NULL;

COMMENT ON TABLE user_roles IS 'User to role assignments (many-to-many)';
COMMENT ON COLUMN user_roles.assigned_by IS 'User who assigned this role';

-- ============================================================================
-- AUDIT_LOGS TABLE
-- Security audit trail of user actions
-- ============================================================================
CREATE TABLE audit_logs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    action VARCHAR(100) NOT NULL,
    resource_type VARCHAR(100),
    resource_id UUID,
    ip_address INET,
    user_agent TEXT,
    status VARCHAR(50) DEFAULT 'success',
    error_message TEXT,
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_audit_logs_user_id ON audit_logs(user_id, created_at DESC);
CREATE INDEX idx_audit_logs_action ON audit_logs(action, created_at DESC);
CREATE INDEX idx_audit_logs_resource ON audit_logs(resource_type, resource_id);
CREATE INDEX idx_audit_logs_created_at ON audit_logs(created_at DESC);
CREATE INDEX idx_audit_logs_status ON audit_logs(status, created_at DESC);

COMMENT ON TABLE audit_logs IS 'Security audit trail of user actions';
COMMENT ON COLUMN audit_logs.action IS 'Action performed (e.g., login, logout, create_workflow)';
COMMENT ON COLUMN audit_logs.resource_type IS 'Type of resource affected (e.g., workflow, user, trigger)';
COMMENT ON COLUMN audit_logs.status IS 'Action status: success, failure';

-- ============================================================================
-- UPDATE WORKFLOWS TABLE
-- Add foreign key to users table for ownership
-- ============================================================================
ALTER TABLE workflows
    ADD CONSTRAINT fk_workflows_created_by
    FOREIGN KEY (created_by) REFERENCES users(id) ON DELETE SET NULL;

-- ============================================================================
-- INSERT DEFAULT SYSTEM ROLES
-- ============================================================================
INSERT INTO roles (id, name, description, is_system, permissions) VALUES
    (
        uuid_generate_v4(),
        'admin',
        'Administrator with full system access',
        true,
        ARRAY[
            'workflow:create', 'workflow:read', 'workflow:update', 'workflow:delete', 'workflow:execute', 'workflow:publish',
            'execution:read', 'execution:cancel', 'execution:retry',
            'trigger:create', 'trigger:read', 'trigger:update', 'trigger:delete',
            'user:create', 'user:read', 'user:update', 'user:delete', 'user:manage',
            'role:create', 'role:read', 'role:update', 'role:delete', 'role:manage',
            'system:admin', 'audit:read'
        ]
    ),
    (
        uuid_generate_v4(),
        'user',
        'Regular user with standard workflow access',
        true,
        ARRAY[
            'workflow:create', 'workflow:read', 'workflow:update', 'workflow:delete', 'workflow:execute',
            'execution:read', 'execution:cancel',
            'trigger:create', 'trigger:read', 'trigger:update', 'trigger:delete'
        ]
    ),
    (
        uuid_generate_v4(),
        'viewer',
        'Read-only access to workflows and executions',
        true,
        ARRAY[
            'workflow:read',
            'execution:read',
            'trigger:read'
        ]
    );

-- ============================================================================
-- TRIGGERS: Auto-update updated_at timestamps
-- ============================================================================
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER update_users_updated_at
    BEFORE UPDATE ON users
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_roles_updated_at
    BEFORE UPDATE ON roles
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- ============================================================================
-- FUNCTION: Cleanup expired sessions
-- Can be called periodically via cron job or application scheduler
-- ============================================================================
CREATE OR REPLACE FUNCTION cleanup_expired_sessions()
RETURNS INTEGER AS $$
DECLARE
    deleted_count INTEGER;
BEGIN
    DELETE FROM sessions WHERE expires_at < NOW();
    GET DIAGNOSTICS deleted_count = ROW_COUNT;
    RETURN deleted_count;
END;
$$ LANGUAGE plpgsql;

COMMENT ON FUNCTION cleanup_expired_sessions() IS 'Removes expired sessions from the database. Call periodically.';

-- ============================================================================
-- FUNCTION: Get user permissions (including from roles)
-- ============================================================================
CREATE OR REPLACE FUNCTION get_user_permissions(p_user_id UUID)
RETURNS TEXT[] AS $$
DECLARE
    result TEXT[];
BEGIN
    SELECT ARRAY_AGG(DISTINCT permission)
    INTO result
    FROM (
        SELECT unnest(r.permissions) AS permission
        FROM user_roles ur
        JOIN roles r ON r.id = ur.role_id
        WHERE ur.user_id = p_user_id
    ) AS perms;

    RETURN COALESCE(result, ARRAY[]::TEXT[]);
END;
$$ LANGUAGE plpgsql;

COMMENT ON FUNCTION get_user_permissions(UUID) IS 'Returns array of all permissions for a user based on their roles';

-- ============================================================================
-- FUNCTION: Check if user has permission
-- ============================================================================
CREATE OR REPLACE FUNCTION user_has_permission(p_user_id UUID, p_permission TEXT)
RETURNS BOOLEAN AS $$
DECLARE
    has_perm BOOLEAN;
BEGIN
    -- Check if user is admin (admins have all permissions)
    SELECT is_admin INTO has_perm FROM users WHERE id = p_user_id;
    IF has_perm THEN
        RETURN true;
    END IF;

    -- Check specific permission through roles
    SELECT EXISTS (
        SELECT 1
        FROM user_roles ur
        JOIN roles r ON r.id = ur.role_id
        WHERE ur.user_id = p_user_id
        AND p_permission = ANY(r.permissions)
    ) INTO has_perm;

    RETURN has_perm;
END;
$$ LANGUAGE plpgsql;

COMMENT ON FUNCTION user_has_permission(UUID, TEXT) IS 'Check if user has a specific permission';
