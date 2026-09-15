-- Keep the purchase page visible, but pause new balance orders until launch.
-- Administrators can explicitly enable balance recharge later in payment settings.
-- Fresh installs receive these values from InitializeDefaultSettings.
INSERT INTO settings (key, value, updated_at)
SELECT key, value, NOW() FROM (VALUES
    ('payment_enabled', 'true'),
    ('BALANCE_PAYMENT_DISABLED', 'true')
) AS preview_settings(key, value)
WHERE EXISTS (SELECT 1 FROM users)
ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value, updated_at = EXCLUDED.updated_at;
