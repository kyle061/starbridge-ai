-- Align the default group labels with the central customer pricing policy.
-- Actual model rates are selected by billing.retail_pricing, including 2.5x
-- for the configured latest model families. No historical charges are changed.
UPDATE groups SET rate_multiplier = 2, peak_rate_enabled = false, updated_at = NOW()
WHERE deleted_at IS NULL;
