-- Store subscription plan entitlements separately from the payment amount.
ALTER TABLE subscription_plans
    ADD COLUMN IF NOT EXISTS quota_multiplier DECIMAL(20,4) NOT NULL DEFAULT 10,
    ADD COLUMN IF NOT EXISTS usage_multiplier DECIMAL(20,4) NOT NULL DEFAULT 12;

-- Snapshot purchased entitlements on the order and the user subscription.
ALTER TABLE payment_orders
    ADD COLUMN IF NOT EXISTS subscription_quota_usd DECIMAL(20,10),
    ADD COLUMN IF NOT EXISTS subscription_usage_multiplier DECIMAL(20,4),
    ADD COLUMN IF NOT EXISTS subscription_plan_name VARCHAR(100);

ALTER TABLE user_subscriptions
    ADD COLUMN IF NOT EXISTS quota_usd DECIMAL(20,10) NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS quota_used_usd DECIMAL(20,10) NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS usage_multiplier DECIMAL(20,4) NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS plan_name VARCHAR(100) NOT NULL DEFAULT '';
