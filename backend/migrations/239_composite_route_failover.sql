DROP INDEX IF EXISTS idx_composite_model_routes_unique_active;

CREATE UNIQUE INDEX IF NOT EXISTS idx_composite_model_routes_unique_active
    ON composite_model_routes (
        group_id,
        endpoint,
        match_type,
        public_model,
        target_platform,
        upstream_model
    )
    WHERE deleted_at IS NULL;
