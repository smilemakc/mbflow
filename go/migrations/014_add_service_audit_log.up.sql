-- Migration: 014_add_service_audit_log
-- Description: Add service_audit_log table for tracking service API actions
-- Date: 2026-02-05

CREATE TABLE mbflow_service_audit_log (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    system_key_id UUID NOT NULL REFERENCES mbflow_system_keys(id) ON DELETE CASCADE,
    service_name VARCHAR(100) NOT NULL,
    impersonated_user_id UUID,
    action VARCHAR(100) NOT NULL,
    resource_type VARCHAR(50) NOT NULL,
    resource_id UUID,
    request_method VARCHAR(10) NOT NULL,
    request_path VARCHAR(500) NOT NULL,
    request_body JSONB,
    response_status INTEGER NOT NULL,
    ip_address VARCHAR(45),
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_mbflow_service_audit_log_system_key ON mbflow_service_audit_log(system_key_id);
CREATE INDEX idx_mbflow_service_audit_log_service ON mbflow_service_audit_log(service_name);
CREATE INDEX idx_mbflow_service_audit_log_action ON mbflow_service_audit_log(action);
CREATE INDEX idx_mbflow_service_audit_log_resource ON mbflow_service_audit_log(resource_type, resource_id);
CREATE INDEX idx_mbflow_service_audit_log_impersonated ON mbflow_service_audit_log(impersonated_user_id) WHERE impersonated_user_id IS NOT NULL;
CREATE INDEX idx_mbflow_service_audit_log_created ON mbflow_service_audit_log(created_at DESC);

COMMENT ON TABLE mbflow_service_audit_log IS 'Audit log for all actions through Service API';
