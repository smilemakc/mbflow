-- Allow email format in username field
-- This enables using email as username for simplified registration

-- Drop the old username constraint that only allows alphanumeric chars
ALTER TABLE mbflow_users DROP CONSTRAINT IF EXISTS mbflow_users_username_check;

-- Add new constraint that allows email format (or alphanumeric)
-- Accepts both: traditional usernames AND email addresses
ALTER TABLE mbflow_users ADD CONSTRAINT mbflow_users_username_check
    CHECK (username ~ '^[a-zA-Z0-9_.@+-]{3,255}$');
