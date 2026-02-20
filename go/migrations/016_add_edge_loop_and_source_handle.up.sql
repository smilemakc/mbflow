-- Add source_handle and loop columns to mbflow_edges for conditional routing and loop edges

ALTER TABLE mbflow_edges
    ADD COLUMN source_handle VARCHAR(100),
    ADD COLUMN loop JSONB;

COMMENT ON COLUMN mbflow_edges.source_handle IS 'Output handle name for conditional routing (e.g., "true", "false")';
COMMENT ON COLUMN mbflow_edges.loop IS 'Loop configuration as JSON: {"max_iterations": N}. Non-null marks this as a back-edge.';

-- Validate loop config has max_iterations > 0 when present
ALTER TABLE mbflow_edges
    ADD CONSTRAINT mbflow_edges_loop_valid CHECK (
        loop IS NULL OR (loop->>'max_iterations')::int > 0
    );
