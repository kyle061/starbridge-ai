-- Some legacy installations have no registration_enabled row at all. The
-- public settings endpoint treats that missing value as false, so migration
-- 240's UPDATE alone cannot open registration there.
--
-- Only backfill databases that already contain users. During a fresh setup
-- migrations run before the first admin is created; leaving the table empty
-- lets InitializeDefaultSettings create the complete default settings set.
INSERT INTO settings (key, value, updated_at)
SELECT 'registration_enabled', 'true', NOW()
WHERE EXISTS (SELECT 1 FROM users)
ON CONFLICT (key) DO UPDATE
SET value = EXCLUDED.value,
    updated_at = EXCLUDED.updated_at;
