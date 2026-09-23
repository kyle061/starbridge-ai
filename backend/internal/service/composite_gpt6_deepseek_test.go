//go:build unit

package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCompositeGPT6SchedulerKeepsFinalExecutionOnOpenAI(t *testing.T) {
	groupID := int64(77)
	for _, tc := range []struct {
		name              string
		deepSeekAvailable bool
		openAIAvailable   bool
		expectedPlatform  string
		expectedModel     string
	}{
		{"OpenAI primary", true, true, PlatformOpenAI, "gpt-6-astra"},
		{"DeepSeek unavailable", false, true, PlatformOpenAI, "gpt-6-astra"},
		{"DeepSeek is not final GPT6 fallback", true, false, "", ""},
		{"both unavailable", false, false, "", ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			repo := stubOpenAIAccountRepo{accounts: []Account{
				{ID: 1, Platform: PlatformOpenAI, Type: AccountTypeAPIKey, Status: StatusActive, Schedulable: tc.openAIAvailable,
					Concurrency: 1, Priority: 1, AccountGroups: []AccountGroup{{GroupID: groupID}}},
				{ID: 2, Platform: PlatformDeepseek, Type: AccountTypeAPIKey, Status: StatusActive, Schedulable: tc.deepSeekAvailable,
					Concurrency: 1, Priority: 1, AccountGroups: []AccountGroup{{GroupID: groupID}}},
			}}
			svc := &OpenAIGatewayService{accountRepo: repo, cache: &stubGatewayCache{}}
			candidates := []CompositeRouteDecision{
				{Matched: true, Source: CompositeRouteSourceExplicit, GroupID: groupID, PublicModel: "gpt-6-astra",
					TargetPlatform: PlatformOpenAI, UpstreamModel: "gpt-6-astra", Endpoint: CompositeRouteEndpointResponses},
				{Matched: true, Source: CompositeRouteSourceExplicit, GroupID: groupID, PublicModel: "gpt-6-astra",
					TargetPlatform: PlatformDeepseek, UpstreamModel: "deepseek-v4-pro", Endpoint: CompositeRouteEndpointResponses},
			}
			ctx := WithCompositeRouteCandidates(context.Background(), candidates)
			ctx = WithCompositeRouteDecision(ctx, candidates[0])
			selection, _, err := svc.SelectAccountWithSchedulerForCapability(ctx, &groupID, "", "", "gpt-6-astra", nil,
				OpenAIUpstreamTransportAny, OpenAIEndpointCapabilityResponses, false, false, true)
			if tc.expectedPlatform == "" {
				require.ErrorIs(t, err, ErrNoAvailableAccounts)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, selection)
			require.Equal(t, tc.expectedPlatform, selection.Account.Platform)
			mapped, matched := selection.Account.ResolveMappedModel("gpt-6-astra")
			require.True(t, matched)
			require.Equal(t, tc.expectedModel, mapped)
			platform, ok := ResolvedTargetPlatformFromContext(ctx)
			require.True(t, ok)
			require.Equal(t, tc.expectedPlatform, platform)
			publicModel, ok := RequestedPublicModelFromContext(ctx)
			require.True(t, ok)
			require.Equal(t, "gpt-6-astra", publicModel)
		})
	}
}

func TestCompositeGPT6UsageKeepsRequestedAndActualModelsSeparate(t *testing.T) {
	usageRepo := &openAIRecordUsageLogRepoStub{inserted: true}
	billingRepo := &openAIRecordUsageBillingRepoStub{result: &UsageBillingApplyResult{Applied: true}}
	svc := newOpenAIRecordUsageServiceWithBillingRepoForTest(usageRepo, billingRepo,
		&openAIRecordUsageUserRepoStub{}, &openAIRecordUsageSubRepoStub{}, nil)
	err := svc.RecordUsage(context.Background(), &OpenAIRecordUsageInput{
		Result: &OpenAIForwardResult{RequestID: "resp_gpt6_deepseek", Model: "deepseek-v4-pro", UpstreamModel: "deepseek-v4-pro"},
		APIKey: &APIKey{ID: 1, Group: &Group{Platform: PlatformComposite, RateMultiplier: 1}},
		User:   &User{ID: 2}, Account: &Account{ID: 3, Type: AccountTypeAPIKey, Platform: PlatformDeepseek},
		APIKeyService:      &openAIRecordUsageAPIKeyQuotaStub{},
		ChannelUsageFields: ChannelUsageFields{OriginalModel: "gpt-6-astra"},
	})
	require.NoError(t, err)
	require.NotNil(t, usageRepo.lastLog)
	require.Equal(t, "gpt-6-astra", usageRepo.lastLog.RequestedModel)
	require.Equal(t, "deepseek-v4-pro", usageRepo.lastLog.Model)
	require.NotNil(t, usageRepo.lastLog.UpstreamModel)
	require.Equal(t, "deepseek-v4-pro", *usageRepo.lastLog.UpstreamModel)
}

func TestGPT6PreparationMultiplierChargesConfiguredRate(t *testing.T) {
	usageRepo := &openAIRecordUsageLogRepoStub{inserted: true}
	billingRepo := &openAIRecordUsageBillingRepoStub{result: &UsageBillingApplyResult{Applied: true}}
	svc := newOpenAIRecordUsageServiceWithBillingRepoForTest(usageRepo, billingRepo,
		&openAIRecordUsageUserRepoStub{}, &openAIRecordUsageSubRepoStub{}, nil)
	multiplier := 6.0
	err := svc.RecordUsage(context.Background(), &OpenAIRecordUsageInput{
		Result: &OpenAIForwardResult{RequestID: "prep", Model: "deepseek-v4-pro", BillingModel: "deepseek-v4-pro", UpstreamModel: "deepseek-v4-pro", Usage: OpenAIUsage{InputTokens: 10, OutputTokens: 5}},
		APIKey: &APIKey{ID: 1, Group: &Group{Platform: PlatformComposite, RateMultiplier: 1}},
		User:   &User{ID: 2}, Account: &Account{ID: 3, Type: AccountTypeAPIKey, Platform: PlatformDeepseek},
		APIKeyService: &openAIRecordUsageAPIKeyQuotaStub{}, BillingMultiplierOverride: &multiplier,
		ChannelUsageFields: ChannelUsageFields{OriginalModel: "gpt-6-astra", BillingModelSource: BillingModelSourceUpstream},
	})
	require.NoError(t, err)
	require.NotNil(t, usageRepo.lastLog)
	require.Equal(t, "gpt-6-astra", usageRepo.lastLog.RequestedModel)
	require.Equal(t, 6.0, usageRepo.lastLog.RateMultiplier)
	require.Greater(t, usageRepo.lastLog.ActualCost, usageRepo.lastLog.TotalCost)
}
