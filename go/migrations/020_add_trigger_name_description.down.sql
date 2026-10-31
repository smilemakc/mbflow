-- Migration: 020_add_trigger_name_description (rollback)
-- Description: Remove trigger name and description columns

ALTER TABLE mbflow_triggers
    DROP COLUMN IF EXISTS description,
    DROP COLUMN IF EXISTS name;
