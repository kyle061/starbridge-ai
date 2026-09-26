-- Rebuild derived monitoring data with provider attribution and inference-only
-- error counting. Keep raw usage/error logs and existing aggregates; the reset
-- coverage watermark hides unrecomputed history while bounded backfill runs.
UPDATE channel_monitor_v2_watermarks
SET usage_coverage_start = NULL,
    error_coverage_start = NULL,
    data_through = NULL,
    last_successful_at = NULL,
    backfill_cursor = NULL,
    updated_at = NOW()
WHERE id = 1;
