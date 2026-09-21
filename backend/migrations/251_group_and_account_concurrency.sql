-- Keep upstream account concurrency independent from group capacity.
-- Existing legacy defaults (3/5/10) are normalized to the requested per-account
-- default of 5. The group cap is an independent ceiling; physical capacity still
-- cannot exceed the sum of schedulable account capacities.
ALTER TABLE accounts
    ALTER COLUMN concurrency SET DEFAULT 5;

UPDATE accounts
SET concurrency = 5,
    updated_at = NOW()
WHERE deleted_at IS NULL
  AND concurrency IN (3, 10);

ALTER TABLE groups
    ADD COLUMN IF NOT EXISTS concurrency_limit INTEGER NOT NULL DEFAULT 1000;

UPDATE groups
SET concurrency_limit = 1000
WHERE concurrency_limit IS NULL OR concurrency_limit = 0;
