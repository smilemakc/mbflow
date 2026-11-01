-- Migration: 021_widen_session_tokens (rollback)
-- Description: Restore VARCHAR(500) session token columns.
--
-- Sessions whose tokens do not fit are deleted; their users sign in again.

DELETE FROM mbflow_sessions
WHERE length(token) > 500 OR length(refresh_token) > 500;

ALTER TABLE mbflow_sessions
    ALTER COLUMN token TYPE VARCHAR(500),
    ALTER COLUMN refresh_token TYPE VARCHAR(500);
