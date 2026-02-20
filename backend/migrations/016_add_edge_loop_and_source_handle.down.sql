ALTER TABLE mbflow_edges DROP CONSTRAINT IF EXISTS mbflow_edges_loop_valid;
ALTER TABLE mbflow_edges
    DROP COLUMN IF EXISTS loop,
    DROP COLUMN IF EXISTS source_handle;
