-- Group concurrency is derived from the capacity of its schedulable accounts.
-- Keep the legacy column for backward-compatible reads, but clear old values
-- so administrative clients do not see a stale per-group ceiling.
ALTER TABLE groups
    ALTER COLUMN concurrency_limit SET DEFAULT 0;

UPDATE groups
SET concurrency_limit = 0,
    updated_at = NOW()
WHERE deleted_at IS NULL
  AND concurrency_limit <> 0;
