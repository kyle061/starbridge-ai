-- Apply Starbridge's launch price once. Subsequent admin price edits remain valid.
-- Fresh installations use payment_amounts.go defaults; avoid seeding settings
-- before InitializeDefaultSettings has created the complete default set.
INSERT INTO settings (key, value, updated_at)
SELECT key, value, NOW() FROM (VALUES
    ('payment_enabled', 'true'),
    ('BALANCE_RECHARGE_MULTIPLIER', '2'),
    ('MIN_RECHARGE_AMOUNT', '0.5'),
    ('BALANCE_PAYMENT_DISABLED', 'false'),
    ('RECHARGE_FEE_RATE', '0'),
    ('subscription_enabled', 'false')
) AS prepaid_settings(key, value)
WHERE EXISTS (SELECT 1 FROM users)
ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value, updated_at = EXCLUDED.updated_at;

-- A paid user's credential survives renewals. Preserve manual disabling and
-- per-key spending limits; only remove the old time-based expiration.
UPDATE api_keys k SET expires_at = NULL,
    status = CASE WHEN k.status = 'expired' THEN 'active' ELSE k.status END,
    updated_at = NOW()
FROM users u
WHERE k.user_id = u.id AND u.role = 'user' AND k.deleted_at IS NULL
  AND (k.expires_at IS NOT NULL OR k.status = 'expired');

CREATE INDEX IF NOT EXISTS idx_payment_orders_prepaid_access
ON payment_orders (user_id)
WHERE order_type = 'balance' AND paid_at IS NOT NULL
  AND status IN ('COMPLETED', 'PARTIALLY_REFUNDED') AND amount > 0 AND pay_amount > 0;
