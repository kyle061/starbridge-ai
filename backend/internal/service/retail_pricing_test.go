//go:build unit

package service

import (
	"context"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/stretchr/testify/require"
)

func enableRetailPricing(cfg *config.Config) {
	cfg.Billing.RetailPricing = config.RetailPricingConfig{
		Enabled: true, StandardMultiplier: 2, LatestMultiplier: 2.5,
		LatestModelPrefixes: []string{"gpt-6", "deepseek-v4"},
	}
}

func TestRetailPricing_RecordUsageDebitsSameCostAcrossPlatforms(t *testing.T) {
	for _, tc := range []struct {
		model string
		rate  float64
	}{
		{"gpt-6-astra", 2.5}, {"gpt-5.6-luna", 2}, {"gpt-5.6-terra", 2}, {"gpt-5.6-sol", 2},
		{"gpt-5.4", 2}, {"deepseek-chat", 2}, {"deepseek-v4-flash", 2.5},
	} {
		t.Run(tc.model, func(t *testing.T) {
			for _, openAIPath := range []bool{false, true} {
				usageRepo := &openAIRecordUsageLogRepoStub{inserted: true}
				billingRepo := &openAIRecordUsageBillingRepoStub{}
				userRepo := &openAIRecordUsageUserRepoStub{}
				subRepo := &openAIRecordUsageSubRepoStub{}
				key := &APIKey{ID: 501, Quota: 100, Group: &Group{Platform: PlatformComposite, RateMultiplier: 1.5}}
				user := &User{ID: 601}
				account := &Account{ID: 701, Type: AccountTypeAPIKey}
				if openAIPath {
					svc := newOpenAIRecordUsageServiceWithBillingRepoForTest(usageRepo, billingRepo, userRepo, subRepo, nil)
					enableRetailPricing(svc.cfg)
					require.NoError(t, svc.RecordUsage(context.Background(), &OpenAIRecordUsageInput{
						Result: &OpenAIForwardResult{RequestID: "retail", Model: tc.model, Duration: time.Second,
							Usage: OpenAIUsage{InputTokens: 1000, CacheReadInputTokens: 200, OutputTokens: 100}},
						APIKey: key, User: user, Account: account, APIKeyService: &openAIRecordUsageAPIKeyQuotaStub{},
					}))
				} else {
					svc := newGatewayRecordUsageServiceWithBillingRepoForTest(usageRepo, billingRepo, userRepo, subRepo)
					enableRetailPricing(svc.cfg)
					require.NoError(t, svc.RecordUsage(context.Background(), &RecordUsageInput{
						Result: &ForwardResult{RequestID: "retail", Model: tc.model, Duration: time.Second,
							Usage: ClaudeUsage{InputTokens: 800, CacheReadInputTokens: 200, OutputTokens: 100}},
						APIKey: key, User: user, Account: account, APIKeyService: &openAIRecordUsageAPIKeyQuotaStub{},
					}))
				}
				log := usageRepo.lastLog
				require.NotNil(t, log)
				require.Greater(t, log.TotalCost, 0.0)
				require.InDelta(t, log.TotalCost*tc.rate, log.ActualCost, 1e-10)
				require.InDelta(t, tc.rate, log.RateMultiplier, 1e-10)
				require.InDelta(t, log.ActualCost, billingRepo.lastCmd.BalanceCost, 1e-8)
				require.InDelta(t, log.ActualCost, billingRepo.lastCmd.APIKeyQuotaCost, 1e-8)
			}
		})
	}
}

func TestRetailPricing_ModelMatchingAndIdempotency(t *testing.T) {
	cfg := &config.Config{}
	enableRetailPricing(cfg)
	for _, tc := range []struct {
		model string
		rate  float64
	}{
		{" OpenAI/GPT-6-Astra ", 2.5}, {"gpt-6-astra-2026-09-01", 2.5},
		{"gpt-5.6", 2}, {" OpenAI/GPT-5.6-Luna ", 2}, {"gpt-5.6-luna-2026-09-01", 2},
		{"gpt-60", 2}, {"gpt-5.60", 2}, {"deepseek-reasoner", 2},
	} {
		cost := &CostBreakdown{TotalCost: 1, ActualCost: 9, InputCost: .6, OutputCost: .4}
		applyRetailCost(cfg, tc.model, cost)
		applyRetailCost(cfg, tc.model, cost)
		require.Equal(t, tc.rate, cost.ActualCost)
		require.Equal(t, 1.0, cost.TotalCost)
		require.Equal(t, .6, cost.InputCost)
	}
}
