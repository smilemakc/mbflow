-- ============================================================================
-- Migration: 020_add_trigger_name_description
-- Description: Store trigger name and description in dedicated columns.
--
-- Until now the API accepted name/description but did not persist them; the
-- UI worked around this by copying them into config.name / config.description.
-- Existing rows are backfilled from those config keys (which are left in
-- place so the change is non-destructive and the rollback loses nothing).
-- ============================================================================

ALTER TABLE mbflow_triggers
    ADD COLUMN name VARCHAR(255) NOT NULL DEFAULT '',
    ADD COLUMN description TEXT NOT NULL DEFAULT '';

UPDATE mbflow_triggers
SET name = LEFT(config->>'name', 255)
WHERE jsonb_typeof(config->'name') = 'string';

UPDATE mbflow_triggers
SET description = config->>'description'
WHERE jsonb_typeof(config->'description') = 'string';

COMMENT ON COLUMN mbflow_triggers.name IS 'Human-readable trigger name';
COMMENT ON COLUMN mbflow_triggers.description IS 'Optional trigger description';
