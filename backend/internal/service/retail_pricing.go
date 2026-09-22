package service

import (
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/config"
)

// applySubscriptionPricing replaces the global retail multiplier for a
// purchased subscription. A subscription's usage_multiplier is already its
// complete customer multiplier, so applying it after the global retail rule
// would compound two independent multipliers (for example 6x * 12x).
//
// Legacy/admin-assigned subscriptions with a zero multiplier keep the global
// retail pricing behavior for backwards compatibility.
func applySubscriptionPricing(subscription *UserSubscription, cost *CostBreakdown) {
	if subscription == nil || cost == nil || subscription.UsageMultiplier <= 0 ||
		cost.TotalCost <= 0 || cost.ActualCost <= 0 {
		return
	}
	cost.ActualCost = cost.TotalCost * subscription.UsageMultiplier
}

// subscriptionBillingCost returns the single customer charge represented by a
// subscription request. It intentionally derives from TotalCost so the
// subscription multiplier is applied once even when the global retail policy
// has already populated ActualCost.
func subscriptionBillingCost(subscription *UserSubscription, cost *CostBreakdown) float64 {
	if cost == nil || cost.ActualCost <= 0 {
		return 0
	}
	if subscription != nil && subscription.UsageMultiplier > 0 && cost.TotalCost > 0 {
		return cost.TotalCost * subscription.UsageMultiplier
	}
	return cost.ActualCost
}

// retailModelRate replaces group/user/peak multipliers. Applying a percentage
// to TotalCost, rather than to ActualCost, makes repeated application idempotent
// and keeps upstream accounting at its original cost.
func retailModelRate(cfg *config.Config, model string) (float64, bool) {
	if cfg == nil || !cfg.Billing.RetailPricing.Enabled {
		return 0, false
	}
	policy := cfg.Billing.RetailPricing
	model = strings.ToLower(strings.TrimSpace(model))
	if i := strings.LastIndex(model, "/"); i >= 0 {
		model = model[i+1:]
	}
	for _, prefix := range policy.LatestModelPrefixes {
		prefix = strings.ToLower(strings.TrimSpace(prefix))
		if prefix != "" && (model == prefix || strings.HasPrefix(model, prefix+"-")) {
			return policy.LatestMultiplier, true
		}
	}
	return policy.StandardMultiplier, true
}

func applyRetailCost(cfg *config.Config, model string, cost *CostBreakdown) {
	if rate, enabled := retailModelRate(cfg, model); enabled && cost != nil {
		cost.ActualCost = cost.TotalCost * rate
	}
}

func retailUsageRate(cfg *config.Config, cost *CostBreakdown, fallback float64) float64 {
	if cfg != nil && cfg.Billing.RetailPricing.Enabled && cost != nil && cost.TotalCost > 0 {
		return cost.ActualCost / cost.TotalCost
	}
	return fallback
}
