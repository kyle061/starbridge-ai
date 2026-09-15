//go:build unit

package service

import (
	"context"
	"math"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type adminLimitsRepo struct {
	APIKeyRepository
	key    APIKey
	fields APIKeyUpdateFields
}

func (r *adminLimitsRepo) GetByID(context.Context, int64) (*APIKey, error) {
	key := r.key
	return &key, nil
}

func (r *adminLimitsRepo) Update(_ context.Context, key *APIKey, fields APIKeyUpdateFields) error {
	r.key, r.fields = *key, fields
	return nil
}

func TestAdminUpdateAPIKeyLimitsPreservesUsageAndOtherSettings(t *testing.T) {
	expiry := time.Now().Add(time.Hour)
	repo := &adminLimitsRepo{key: APIKey{ID: 1, Key: "same-key", Quota: 10, QuotaUsed: 10,
		Status: StatusAPIKeyQuotaExhausted, RateLimit5h: 2, RateLimit1d: 3, RateLimit7d: 4,
		Usage5h: 1, Usage1d: 2, Usage7d: 3, ExpiresAt: &expiry}}
	cache := &authCacheInvalidatorStub{}
	svc := &adminServiceImpl{apiKeyRepo: repo, authCacheInvalidator: cache}
	zero, quota := 0.0, 20.0
	key, err := svc.AdminUpdateAPIKeyLimits(context.Background(), 1, AdminAPIKeyLimits{Quota: &quota, RateLimit1d: &zero})
	require.NoError(t, err)
	require.Equal(t, APIKeyUpdateFields{Quota: true, RateLimits: true, Status: true}, repo.fields)
	require.Equal(t, 20.0, key.Quota)
	require.Equal(t, StatusActive, key.Status)
	require.Equal(t, 10.0, key.QuotaUsed)
	require.Equal(t, 0.0, key.RateLimit1d)
	require.Equal(t, 2.0, key.RateLimit5h)
	require.Equal(t, 4.0, key.RateLimit7d)
	require.Equal(t, 1.0, key.Usage5h)
	require.Equal(t, 2.0, key.Usage1d)
	require.Equal(t, 3.0, key.Usage7d)
	require.Equal(t, &expiry, key.ExpiresAt)
	require.Equal(t, []string{"same-key"}, cache.keys)

	repo.key.Status = "inactive"
	key, err = svc.AdminUpdateAPIKeyLimits(context.Background(), 1, AdminAPIKeyLimits{Quota: &zero})
	require.NoError(t, err)
	require.Equal(t, "inactive", key.Status)
	for _, invalid := range []float64{-1, math.NaN(), math.Inf(1)} {
		_, err := svc.AdminUpdateAPIKeyLimits(context.Background(), 1, AdminAPIKeyLimits{RateLimit5h: &invalid})
		require.Error(t, err)
	}
}
