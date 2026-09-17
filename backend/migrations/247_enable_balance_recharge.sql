-- Balance recharge is now part of the live payment flow. Migration 243 paused
-- new balance orders while the payment UI was being prepared; reopen the switch
-- for existing installations so users can pay with the configured providers.
INSERT INTO settings (key, value, updated_at)
SELECT key, value, NOW() FROM (VALUES
    ('BALANCE_PAYMENT_DISABLED', 'false')
) AS balance_recharge_settings(key, value)
WHERE EXISTS (SELECT 1 FROM users)
ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value, updated_at = EXCLUDED.updated_at;
