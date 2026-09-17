-- Repair installations that already have an enabled payment provider but
-- never received the visible-method routing settings introduced later.
-- Preserve an administrator's explicit source choice; only fill an empty
-- source and the matching disabled-by-default legacy flag.

INSERT INTO settings (key, value, updated_at)
SELECT key, value, NOW() FROM (VALUES
    ('payment_visible_method_alipay_source', 'easypay_alipay'),
    ('payment_visible_method_wxpay_source', 'easypay_wxpay')
) AS legacy_sources(key, value)
WHERE EXISTS (
    SELECT 1
    FROM payment_provider_instances p
    WHERE p.enabled = true
      AND p.provider_key = 'easypay'
      AND (
        TRIM(p.supported_types) = ''
        OR LOWER(TRIM(p.supported_types)) = 'easypay'
        OR (key = 'payment_visible_method_alipay_source'
            AND LOWER(',' || REPLACE(p.supported_types, ' ', '') || ',') LIKE '%,alipay,%')
        OR (key = 'payment_visible_method_wxpay_source'
            AND LOWER(',' || REPLACE(p.supported_types, ' ', '') || ',') LIKE '%,wxpay,%')
      )
)
ON CONFLICT (key) DO UPDATE
SET value = EXCLUDED.value, updated_at = EXCLUDED.updated_at
WHERE TRIM(settings.value) = '';

UPDATE settings
SET value = CASE
    WHEN key = 'payment_visible_method_alipay_enabled' THEN 'true'
    WHEN key = 'payment_visible_method_wxpay_enabled' THEN 'true'
    ELSE value
END,
updated_at = NOW()
WHERE key IN ('payment_visible_method_alipay_enabled', 'payment_visible_method_wxpay_enabled')
  AND value <> 'true'
  AND EXISTS (
    SELECT 1
    FROM payment_provider_instances p
    WHERE p.enabled = true
      AND p.provider_key = 'easypay'
      AND (
        TRIM(p.supported_types) = ''
        OR LOWER(TRIM(p.supported_types)) = 'easypay'
        OR (key = 'payment_visible_method_alipay_enabled'
            AND LOWER(',' || REPLACE(p.supported_types, ' ', '') || ',') LIKE '%,alipay,%')
        OR (key = 'payment_visible_method_wxpay_enabled'
            AND LOWER(',' || REPLACE(p.supported_types, ' ', '') || ',') LIKE '%,wxpay,%')
      )
  );
