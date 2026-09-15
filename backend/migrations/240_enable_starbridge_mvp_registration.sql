-- Starbridge MVP requires a public user flow. Existing installations may have
-- registration disabled from the original admin-only rollout, so enable it
-- once during this release. Administrators can still disable it afterwards.
--
-- Do not insert a missing row here: on a fresh database the settings service
-- must remain responsible for atomically creating the complete default set.
UPDATE settings
SET value = 'true',
    updated_at = NOW()
WHERE key = 'registration_enabled'
  AND value IS DISTINCT FROM 'true';
