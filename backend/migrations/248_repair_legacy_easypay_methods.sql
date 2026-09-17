-- Repair EasyPay instances created by older versions.
--
-- Before the visible-method routing was introduced, EasyPay used the provider
-- key ("easypay") as its supported_types value. Migration 101 removed that
-- provider key and could leave an existing instance with an empty list. An
-- empty list is accepted by the order selector but produces no user-facing
-- methods in checkout-info, making both recharge and subscription checkout
-- appear unavailable. EasyPay's built-in integration supports both methods.
UPDATE payment_provider_instances
SET supported_types = 'alipay,wxpay',
    updated_at = NOW()
WHERE provider_key = 'easypay'
  AND (
    TRIM(supported_types) = ''
    OR LOWER(TRIM(supported_types)) = 'easypay'
  );
