package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestUsageUnrestrictedIncludesWeeklyWindowStart(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodGet, "/v1/usage", nil)

	weeklyWindowStart := time.Date(2026, time.July, 13, 0, 30, 0, 0, time.FixedZone("UTC+8", 8*60*60))
	c.Set(string(middleware.ContextKeySubscription), &service.UserSubscription{
		WeeklyWindowStart: &weeklyWindowStart,
	})

	handler := &GatewayHandler{}
	handler.usageUnrestricted(
		c,
		context.Background(),
		&service.APIKey{Group: &service.Group{
			Name:             "Weekly plan",
			SubscriptionType: service.SubscriptionTypeSubscription,
		}},
		middleware.AuthSubject{},
		nil,
		nil,
		nil,
	)

	require.Equal(t, http.StatusOK, recorder.Code)
	var response struct {
		Subscription struct {
			WeeklyWindowStart *time.Time `json:"weekly_window_start"`
		} `json:"subscription"`
	}
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &response))
	require.NotNil(t, response.Subscription.WeeklyWindowStart)
	require.True(t, weeklyWindowStart.Equal(*response.Subscription.WeeklyWindowStart))
}

func TestSubscriptionRemainingMatchesSharedQuotaAndPeriodicLimits(t *testing.T) {
	handler := &GatewayHandler{}
	group := &service.Group{SubscriptionType: service.SubscriptionTypeSubscription}
	subscription := &service.UserSubscription{QuotaUSD: 10, QuotaUsedUSD: 3.4}
	require.InDelta(t, 6.6, handler.calculateSubscriptionRemaining(group, subscription), 0.000001)
	subscription.QuotaUsedUSD = 11
	require.Zero(t, handler.calculateSubscriptionRemaining(group, subscription))
	subscription.QuotaUsedUSD = 3.4
	limit := 5.0
	group.DailyLimitUSD = &limit
	subscription.DailyUsageUSD = 4
	require.Equal(t, 1.0, handler.calculateSubscriptionRemaining(group, subscription))
	subscription.QuotaUSD = 0
	subscription.DailyUsageUSD = 0
	require.Equal(t, 5.0, handler.calculateSubscriptionRemaining(group, subscription))
	group.DailyLimitUSD = nil
	require.Equal(t, -1.0, handler.calculateSubscriptionRemaining(group, subscription))
}

func TestUsageResponsesCapKeyRemainingBySubscription(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := &GatewayHandler{}
	group := &service.Group{Name: "Shared plan", SubscriptionType: service.SubscriptionTypeSubscription}
	for _, keyQuota := range []float64{0, 20, 4} {
		recorder := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(recorder)
		c.Request = httptest.NewRequest(http.MethodGet, "/v1/usage", nil)
		c.Set(string(middleware.ContextKeySubscription), &service.UserSubscription{QuotaUSD: 10, QuotaUsedUSD: 3.4})
		key := &service.APIKey{Group: group, Quota: keyQuota, QuotaUsed: 1}
		if keyQuota > 0 {
			handler.usageQuotaLimited(c, context.Background(), key, nil, nil, nil)
		} else {
			handler.usageUnrestricted(c, context.Background(), key, middleware.AuthSubject{}, nil, nil, nil)
		}
		require.Equal(t, http.StatusOK, recorder.Code)
		var response struct {
			Remaining    float64 `json:"remaining"`
			Subscription struct {
				QuotaUSD     float64 `json:"quota_usd"`
				QuotaUsedUSD float64 `json:"quota_used_usd"`
			} `json:"subscription"`
		}
		require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &response))
		if keyQuota == 4 {
			require.Equal(t, 3.0, response.Remaining)
		} else {
			require.InDelta(t, 6.6, response.Remaining, 0.000001)
		}
		require.Equal(t, 10.0, response.Subscription.QuotaUSD)
		require.Equal(t, 3.4, response.Subscription.QuotaUsedUSD)
	}
}

func TestUnlimitedSubscriptionUsageOmitsNegativeRemaining(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodGet, "/v1/usage", nil)
	c.Set(string(middleware.ContextKeySubscription), &service.UserSubscription{})
	(&GatewayHandler{}).usageUnrestricted(c, context.Background(), &service.APIKey{Group: &service.Group{SubscriptionType: service.SubscriptionTypeSubscription}}, middleware.AuthSubject{}, nil, nil, nil)
	require.Equal(t, http.StatusOK, recorder.Code)
	var response map[string]any
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &response))
	require.NotContains(t, response, "remaining")
}
