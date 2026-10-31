-- ============================================================================
-- Migration: 021_widen_session_tokens
-- Description: Store session tokens as TEXT.
--
-- Access tokens now carry a unique token ID (jti) so that two logins of the
-- same user within one second get different tokens. With the extra claim a
-- token can exceed the previous VARCHAR(500) limit.
-- ============================================================================

ALTER TABLE mbflow_sessions
    ALTER COLUMN token TYPE TEXT,
    ALTER COLUMN refresh_token TYPE TEXT;
