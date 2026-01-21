-- Revert to original username constraint
ALTER TABLE mbflow_users DROP CONSTRAINT IF EXISTS mbflow_users_username_check;

ALTER TABLE mbflow_users ADD CONSTRAINT mbflow_users_username_check
    CHECK (username ~ '^[a-zA-Z0-9_-]{3,50}$');
