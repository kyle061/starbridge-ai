-- Recalculate only buckets touched by client disconnects. Assigning the exact
-- counts from raw logs makes this safe to rerun and preserves other errors.
WITH affected_hours AS (
  SELECT DISTINCT date_trunc('hour', created_at AT TIME ZONE 'UTC') AT TIME ZONE 'UTC' AS bucket_start
  FROM ops_error_logs
  WHERE status_code = 499 AND is_count_tokens = FALSE
),
affected_dims AS (
  SELECT bucket_start,
    CASE WHEN GROUPING(platform) = 1 THEN NULL ELSE platform END AS platform,
    CASE WHEN GROUPING(group_id) = 1 THEN NULL ELSE group_id END AS group_id
  FROM (
    SELECT date_trunc('hour', created_at AT TIME ZONE 'UTC') AT TIME ZONE 'UTC' AS bucket_start,
      COALESCE(platform, 'unknown') AS platform, group_id
    FROM ops_error_logs
    WHERE status_code = 499 AND is_count_tokens = FALSE
  ) cancelled
  GROUP BY GROUPING SETS (
    (bucket_start), (bucket_start, platform), (bucket_start, platform, group_id)
  )
  HAVING GROUPING(group_id) = 1 OR group_id IS NOT NULL
),
error_base AS (
  SELECT a.bucket_start, COALESCE(e.platform, 'unknown') AS platform,
    e.group_id, e.is_business_limited, e.error_owner,
    e.status_code, COALESCE(e.upstream_status_code, e.status_code, 0) AS effective_status_code
  FROM ops_error_logs e
  JOIN affected_hours a ON e.created_at >= a.bucket_start
    AND e.created_at < a.bucket_start + INTERVAL '1 hour'
  WHERE e.is_count_tokens = FALSE AND e.status_code IS DISTINCT FROM 499
),
error_agg AS (
  SELECT bucket_start,
    CASE WHEN GROUPING(platform) = 1 THEN NULL ELSE platform END AS platform,
    CASE WHEN GROUPING(group_id) = 1 THEN NULL ELSE group_id END AS group_id,
    COUNT(*) FILTER (WHERE COALESCE(status_code, 0) >= 400) AS error_count_total,
    COUNT(*) FILTER (WHERE COALESCE(status_code, 0) >= 400 AND is_business_limited) AS business_limited_count,
    COUNT(*) FILTER (WHERE COALESCE(status_code, 0) >= 400 AND NOT is_business_limited) AS error_count_sla,
    COUNT(*) FILTER (WHERE error_owner = 'provider' AND NOT is_business_limited AND effective_status_code NOT IN (429, 529)) AS upstream_error_count_excl_429_529,
    COUNT(*) FILTER (WHERE error_owner = 'provider' AND NOT is_business_limited AND effective_status_code = 429) AS upstream_429_count,
    COUNT(*) FILTER (WHERE error_owner = 'provider' AND NOT is_business_limited AND effective_status_code = 529) AS upstream_529_count
  FROM error_base
  GROUP BY GROUPING SETS (
    (bucket_start), (bucket_start, platform), (bucket_start, platform, group_id)
  )
  HAVING GROUPING(group_id) = 1 OR group_id IS NOT NULL
)
UPDATE ops_metrics_hourly m SET
  error_count_total = COALESCE(e.error_count_total, 0),
  business_limited_count = COALESCE(e.business_limited_count, 0),
  error_count_sla = COALESCE(e.error_count_sla, 0),
  upstream_error_count_excl_429_529 = COALESCE(e.upstream_error_count_excl_429_529, 0),
  upstream_429_count = COALESCE(e.upstream_429_count, 0),
  upstream_529_count = COALESCE(e.upstream_529_count, 0),
  computed_at = NOW()
FROM affected_dims d
LEFT JOIN error_agg e ON e.bucket_start = d.bucket_start
  AND COALESCE(e.platform, '') = COALESCE(d.platform, '')
  AND COALESCE(e.group_id, 0) = COALESCE(d.group_id, 0)
WHERE m.bucket_start = d.bucket_start
  AND COALESCE(m.platform, '') = COALESCE(d.platform, '')
  AND COALESCE(m.group_id, 0) = COALESCE(d.group_id, 0);

-- Daily rows are rollups of hourly rows, so refresh only affected dates.
WITH affected_dates AS (
  SELECT DISTINCT (created_at AT TIME ZONE 'UTC')::date AS bucket_date
  FROM ops_error_logs
  WHERE status_code = 499 AND is_count_tokens = FALSE
),
rollup AS (
  SELECT (h.bucket_start AT TIME ZONE 'UTC')::date AS bucket_date,
    h.platform, h.group_id,
    SUM(h.error_count_total) AS error_count_total,
    SUM(h.business_limited_count) AS business_limited_count,
    SUM(h.error_count_sla) AS error_count_sla,
    SUM(h.upstream_error_count_excl_429_529) AS upstream_error_count_excl_429_529,
    SUM(h.upstream_429_count) AS upstream_429_count,
    SUM(h.upstream_529_count) AS upstream_529_count
  FROM ops_metrics_hourly h
  JOIN affected_dates a ON (h.bucket_start AT TIME ZONE 'UTC')::date = a.bucket_date
  GROUP BY 1, 2, 3
)
UPDATE ops_metrics_daily d SET
  error_count_total = r.error_count_total,
  business_limited_count = r.business_limited_count,
  error_count_sla = r.error_count_sla,
  upstream_error_count_excl_429_529 = r.upstream_error_count_excl_429_529,
  upstream_429_count = r.upstream_429_count,
  upstream_529_count = r.upstream_529_count,
  computed_at = NOW()
FROM rollup r
WHERE d.bucket_date = r.bucket_date
  AND COALESCE(d.platform, '') = COALESCE(r.platform, '')
  AND COALESCE(d.group_id, 0) = COALESCE(r.group_id, 0);
