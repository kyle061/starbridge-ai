package service

import (
	"context"
	"math"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestSubscriptionAssignmentCarriesQuotaAndRejectsChangedEntitlements(t *testing.T) {
	repo := newSubscriptionUserSubRepoStub()
	svc := NewSubscriptionService(&subscriptionGroupRepoStub{group: &Group{ID: 9, SubscriptionType: SubscriptionTypeSubscription}}, repo, nil, nil, nil)
	t.Cleanup(svc.Stop)
	input := &AssignSubscriptionInput{UserID: 1, GroupID: 9, ValidityDays: 1, QuotaUSD: 50, UsageMultiplier: 12, PlanName: "Daily"}
	sub, err := svc.AssignSubscription(context.Background(), input)
	require.NoError(t, err)
	require.Equal(t, float64(50), sub.QuotaUSD)
	require.Equal(t, float64(12), sub.UsageMultiplier)
	require.Equal(t, "Daily", sub.PlanName)
	again, err := svc.AssignSubscription(context.Background(), input)
	require.NoError(t, err)
	require.Equal(t, sub.ID, again.ID)
	require.Equal(t, 1, repo.createCalls)
	input.QuotaUSD = 100
	_, err = svc.AssignSubscription(context.Background(), input)
	require.ErrorIs(t, err, ErrSubscriptionAssignConflict)
}

func TestBulkSubscriptionAssignmentCarriesQuota(t *testing.T) {
	repo := newSubscriptionUserSubRepoStub()
	svc := NewSubscriptionService(&subscriptionGroupRepoStub{group: &Group{ID: 9, SubscriptionType: SubscriptionTypeSubscription}}, repo, nil, nil, nil)
	t.Cleanup(svc.Stop)
	result, err := svc.BulkAssignSubscription(context.Background(), &BulkAssignSubscriptionInput{UserIDs: []int64{1, 2}, GroupID: 9, ValidityDays: 7, QuotaUSD: 150, UsageMultiplier: 12, PlanName: "Week"})
	require.NoError(t, err)
	require.Equal(t, 2, result.CreatedCount)
	for _, sub := range result.Subscriptions {
		require.Equal(t, float64(150), sub.QuotaUSD)
		require.Equal(t, float64(12), sub.UsageMultiplier)
		require.Equal(t, "Week", sub.PlanName)
	}
}

func TestSetSubscriptionQuotaPreservesUsageAndChecksExhaustion(t *testing.T) {
	start := time.Now().Add(-time.Hour)
	repo := newSubscriptionUserSubRepoStub()
	repo.seed(&UserSubscription{ID: 7, UserID: 1, GroupID: 9, StartsAt: start, ExpiresAt: start.AddDate(0, 0, 30), Status: SubscriptionStatusActive, QuotaUSD: 50, QuotaUsedUSD: 2.345678, DailyUsageUSD: 2, WeeklyUsageUSD: 3, MonthlyUsageUSD: 4})
	svc := NewSubscriptionService(nil, repo, nil, nil, nil)
	t.Cleanup(svc.Stop)
	rate := float64(12)
	sub, err := svc.SetSubscriptionQuota(context.Background(), 7, 100, &rate)
	require.NoError(t, err)
	require.Equal(t, 2.345678, sub.QuotaUsedUSD)
	require.Equal(t, float64(2), sub.DailyUsageUSD)
	require.Equal(t, float64(3), sub.WeeklyUsageUSD)
	require.Equal(t, float64(4), sub.MonthlyUsageUSD)
	require.Equal(t, start.AddDate(0, 0, 30), sub.ExpiresAt)
	require.Equal(t, rate, sub.UsageMultiplier)
	_, err = svc.SetSubscriptionQuota(context.Background(), 7, 1, nil)
	require.Error(t, err)
	sub, err = svc.SetSubscriptionQuota(context.Background(), 7, sub.QuotaUsedUSD, nil)
	require.NoError(t, err)
	require.Zero(t, sub.RemainingQuotaUSD())
	_, err = svc.ValidateAndCheckLimits(sub, &Group{})
	require.ErrorIs(t, err, ErrSubscriptionQuotaExceeded)
}

func TestSetSubscriptionUsageMultiplierPreservesSubscriptionEntitlements(t *testing.T) {
	start := time.Now().Add(-time.Hour)
	repo := newSubscriptionUserSubRepoStub()
	repo.seed(&UserSubscription{
		ID: 7, UserID: 1, GroupID: 9, StartsAt: start,
		ExpiresAt: start.AddDate(0, 0, 30), Status: SubscriptionStatusActive,
		QuotaUSD: 50, QuotaUsedUSD: 2.345678, UsageMultiplier: 12,
		PlanName: "Daily", DailyUsageUSD: 2, WeeklyUsageUSD: 3, MonthlyUsageUSD: 4,
	})
	svc := NewSubscriptionService(nil, repo, nil, nil, nil)
	t.Cleanup(svc.Stop)
	sub, err := svc.SetSubscriptionUsageMultiplier(context.Background(), 7, 18)
	require.NoError(t, err)
	require.Equal(t, float64(18), sub.UsageMultiplier)
	require.Equal(t, float64(50), sub.QuotaUSD)
	require.Equal(t, 2.345678, sub.QuotaUsedUSD)
	require.Equal(t, "Daily", sub.PlanName)
	require.Equal(t, start.AddDate(0, 0, 30), sub.ExpiresAt)
	require.Equal(t, float64(2), sub.DailyUsageUSD)
	require.Equal(t, float64(3), sub.WeeklyUsageUSD)
	require.Equal(t, float64(4), sub.MonthlyUsageUSD)
}

func TestSetLegacySubscriptionQuotaStartsTracking(t *testing.T) {
	repo := newSubscriptionUserSubRepoStub()
	repo.seed(&UserSubscription{ID: 7, UserID: 1, GroupID: 9, QuotaUSD: 0, MonthlyUsageUSD: 10})
	svc := NewSubscriptionService(nil, repo, nil, nil, nil)
	t.Cleanup(svc.Stop)
	sub, err := svc.SetSubscriptionQuota(context.Background(), 7, 50, nil)
	require.NoError(t, err)
	require.Equal(t, float64(50), sub.RemainingQuotaUSD())
	require.Equal(t, float64(10), sub.MonthlyUsageUSD)
	progress := svc.calculateProgress(sub, &Group{Name: "Test"})
	require.NotNil(t, progress.Quota)
	require.Equal(t, float64(50), progress.Quota.RemainingUSD)
}

func TestSubscriptionQuotaRejectsInvalidNumbers(t *testing.T) {
	svc := NewSubscriptionService(nil, nil, nil, nil, nil)
	t.Cleanup(svc.Stop)
	for _, quota := range []float64{0, -1, math.Inf(1), math.NaN()} {
		_, err := svc.SetSubscriptionQuota(context.Background(), 7, quota, nil)
		require.Error(t, err)
	}
}

func TestSubscriptionExpiredAssignmentRenewsQuotaOnce(t *testing.T) {
	repo := newSubscriptionUserSubRepoStub()
	repo.seed(&UserSubscription{ID: 7, UserID: 1, GroupID: 9, StartsAt: time.Now().AddDate(0, 0, -31), ExpiresAt: time.Now().Add(-time.Hour), Status: SubscriptionStatusExpired, QuotaUSD: 100, QuotaUsedUSD: 80})
	svc := NewSubscriptionService(&subscriptionGroupRepoStub{group: &Group{ID: 9, SubscriptionType: SubscriptionTypeSubscription}}, repo, nil, nil, nil)
	t.Cleanup(svc.Stop)
	input := &AssignSubscriptionInput{UserID: 1, GroupID: 9, ValidityDays: 1, QuotaUSD: 50, UsageMultiplier: 12}
	sub, err := svc.AssignSubscription(context.Background(), input)
	require.NoError(t, err)
	require.Equal(t, float64(50), sub.QuotaUSD)
	require.Zero(t, sub.QuotaUsedUSD)
	expires := sub.ExpiresAt
	sub, err = svc.AssignSubscription(context.Background(), input)
	require.NoError(t, err)
	require.Equal(t, float64(50), sub.QuotaUSD)
	require.Equal(t, expires, sub.ExpiresAt)
}
