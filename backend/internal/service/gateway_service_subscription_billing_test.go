//go:build unit

package service

import (
	"testing"
)

// TestBuildUsageBillingCommand_SubscriptionAppliesRateMultiplier locks in the
// subscription billing contract: a purchased subscription multiplier is the
// complete customer multiplier and is applied once to raw TotalCost.
func TestBuildUsageBillingCommand_SubscriptionAppliesRateMultiplier(t *testing.T) {
	t.Parallel()

	groupID := int64(7)
	subID := int64(42)

	tests := []struct {
		name                   string
		totalCost              float64
		actualCost             float64
		subscriptionMultiplier float64
		isSubscription         bool
		wantSub                float64
		wantBalance            float64
	}{
		{
			name:                   "subscription with group multiplier consumes actual cost",
			totalCost:              1.0,
			actualCost:             2.0,
			subscriptionMultiplier: 0,
			isSubscription:         true,
			wantSub:                2.0,
			wantBalance:            0,
		},
		{
			name:                   "purchased subscription multiplier applies once to raw cost",
			totalCost:              1.0,
			actualCost:             2.0,
			subscriptionMultiplier: 12.0,
			isSubscription:         true,
			wantSub:                12.0,
			wantBalance:            0,
		},
		{
			name:                   "subscription with zero actual cost consumes no quota",
			totalCost:              1.0,
			actualCost:             0,
			subscriptionMultiplier: 12.0,
			isSubscription:         true,
			wantSub:                0,
			wantBalance:            0,
		},
		{
			name:                   "balance billing keeps using ActualCost",
			totalCost:              1.0,
			actualCost:             2.0,
			subscriptionMultiplier: 12.0,
			isSubscription:         false,
			wantSub:                0,
			wantBalance:            2.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			p := &postUsageBillingParams{
				Cost:               &CostBreakdown{TotalCost: tt.totalCost, ActualCost: tt.actualCost},
				User:               &User{ID: 1},
				APIKey:             &APIKey{ID: 2, GroupID: &groupID},
				Account:            &Account{ID: 3},
				Subscription:       &UserSubscription{ID: subID, UsageMultiplier: tt.subscriptionMultiplier},
				IsSubscriptionBill: tt.isSubscription,
			}

			cmd := buildUsageBillingCommand("req-1", nil, p)
			if cmd == nil {
				t.Fatal("buildUsageBillingCommand returned nil")
			}
			if cmd.SubscriptionCost != tt.wantSub {
				t.Errorf("SubscriptionCost = %v, want %v", cmd.SubscriptionCost, tt.wantSub)
			}
			if cmd.BalanceCost != tt.wantBalance {
				t.Errorf("BalanceCost = %v, want %v", cmd.BalanceCost, tt.wantBalance)
			}
		})
	}
}
