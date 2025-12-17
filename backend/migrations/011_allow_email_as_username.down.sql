-- Revert to original username constraint
ALTER TABLE users DROP CONSTRAINT IF EXISTS users_username_check;

ALTER TABLE users ADD CONSTRAINT users_username_check
    CHECK (username ~ '^[a-zA-Z0-9_-]{3,50}$');
