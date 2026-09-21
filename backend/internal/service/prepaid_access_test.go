//go:build unit

package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/stretchr/testify/require"
)

type prepaidUserRepoStub struct {
	UserRepository
	owner   *User
	balance PrepaidBalance
	err     error
}

func (r *prepaidUserRepoStub) GetByID(context.Context, int64) (*User, error) { return r.owner, nil }
func (r *prepaidUserRepoStub) GetPrepaidBalance(context.Context, int64) (*PrepaidBalance, error) {
	return &r.balance, r.err
}

type prepaidKeyRepoStub struct {
	APIKeyRepository
	key *APIKey
}

func (r *prepaidKeyRepoStub) Create(_ context.Context, key *APIKey) error     { r.key = key; return nil }
func (r *prepaidKeyRepoStub) GetByID(context.Context, int64) (*APIKey, error) { return r.key, nil }
func (r *prepaidKeyRepoStub) Update(_ context.Context, key *APIKey, _ APIKeyUpdateFields) error {
	r.key = key
	return nil
}

type prepaidAccessGroupRepoStub struct {
	GroupRepository
	group *Group
}

func (r *prepaidAccessGroupRepoStub) GetByID(context.Context, int64) (*Group, error) {
	return r.group, nil
}

type prepaidSubscriptionRepoStub struct {
	UserSubscriptionRepository
	sub *UserSubscription
	err error
}

func (r *prepaidSubscriptionRepoStub) GetActiveByUserIDAndGroupID(context.Context, int64, int64) (*UserSubscription, error) {
	return r.sub, r.err
}

func prepaidTestService() (*APIKeyService, *prepaidUserRepoStub, *prepaidKeyRepoStub) {
	users := &prepaidUserRepoStub{owner: &User{ID: 7, Role: RoleUser, Status: StatusActive}, balance: PrepaidBalance{Balance: 20, HasPurchased: true}}
	keys := &prepaidKeyRepoStub{}
	cfg := &config.Config{RunMode: config.RunModeStandard, Billing: config.BillingConfig{RequireBalancePurchase: true}}
	return NewAPIKeyService(keys, users, nil, nil, nil, nil, cfg), users, keys
}

func TestPrepaidCreateAcceptsOfflineCreditAndRequiresPositiveBalance(t *testing.T) {
	for _, tc := range []struct {
		name    string
		balance float64
		paid    bool
		want    error
	}{
		{"admin or redeem credit without an online order", 20, false, nil},
		{"first recharge required", 0, false, ErrPrepaidBalanceExhausted},
		{"offline credit exhausted", 0, false, ErrPrepaidBalanceExhausted},
		{"exhausted", 0, true, ErrPrepaidBalanceExhausted},
		{"debt", -1, true, ErrPrepaidBalanceExhausted},
		{"paid", 20, true, nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			svc, users, keys := prepaidTestService()
			users.balance = PrepaidBalance{Balance: tc.balance, HasPurchased: tc.paid}
			days := 7
			key, err := svc.Create(context.Background(), 7, CreateAPIKeyRequest{Name: "paid key", ExpiresInDays: &days})
			if tc.want != nil {
				require.ErrorIs(t, err, tc.want)
				require.Nil(t, keys.key)
			} else {
				require.NoError(t, err)
				require.Nil(t, key.ExpiresAt)
				require.Equal(t, StatusActive, key.Status)
			}
		})
	}
}

func TestPrepaidBalancePauseAndRenewalUsesDatabase(t *testing.T) {
	svc, users, _ := prepaidTestService()
	ctx := context.Background()
	users.owner.Balance = 999 // stale auth snapshot
	users.balance.Balance = 0
	require.ErrorIs(t, svc.CheckPrepaidAccess(ctx, users.owner, nil), ErrPrepaidBalanceExhausted)
	users.owner.Balance = 0 // a stale zero must not prevent renewal
	users.balance.Balance = 20
	require.NoError(t, svc.CheckPrepaidAccess(ctx, users.owner, &Group{}))
	require.Equal(t, 20.0, users.owner.Balance)
	users.err = errors.New("database unavailable")
	require.ErrorIs(t, svc.CheckPrepaidAccess(ctx, users.owner, nil), ErrPrepaidAccessUnavailable)
}

func TestPrepaidBalanceAppliesToAdministratorAccounts(t *testing.T) {
	svc, users, _ := prepaidTestService()
	users.owner.Role = RoleAdmin

	require.True(t, svc.RequiresBalancePurchase(users.owner))
	users.balance.Balance = 0
	require.ErrorIs(t, svc.CheckPrepaidAccess(context.Background(), users.owner, &Group{}), ErrPrepaidBalanceExhausted)

	users.balance.Balance = 3.4
	access, err := svc.GetPrepaidAccess(context.Background(), users.owner.ID)
	require.NoError(t, err)
	require.True(t, access.Enabled)
	require.True(t, access.RequestsAllowed)
	require.True(t, access.CanCreateKey)
	require.InDelta(t, 3.4, access.Balance, 0.00000001)
}

func TestPrepaidGroupGateAndEditsPreserveAdminExpiration(t *testing.T) {
	svc, users, keys := prepaidTestService()
	require.ErrorIs(t, svc.CheckPrepaidAccess(context.Background(), users.owner, &Group{SubscriptionType: SubscriptionTypeSubscription}), ErrPrepaidGroupRequired)
	require.ErrorIs(t, svc.CheckPrepaidAccess(context.Background(), users.owner, nil), ErrPrepaidGroupRequired)
	expiration := time.Now().Add(time.Hour)
	keys.key = &APIKey{ID: 1, UserID: 7, Key: "same-key", Status: StatusActive, ExpiresAt: &expiration}
	name := "Renamed"
	key, err := svc.Update(context.Background(), 1, 7, UpdateAPIKeyRequest{Name: &name})
	require.NoError(t, err)
	require.Equal(t, &expiration, key.ExpiresAt)
	require.Equal(t, "Renamed", key.Name)
	require.Equal(t, "same-key", key.Key)
}

func TestSubscriptionGroupDoesNotRequirePrepaidBalance(t *testing.T) {
	svc, users, keys := prepaidTestService()
	group := &Group{ID: 42, SubscriptionType: SubscriptionTypeSubscription}
	sub := &UserSubscription{
		UserID:    users.owner.ID,
		GroupID:   group.ID,
		Status:    SubscriptionStatusActive,
		ExpiresAt: time.Now().Add(time.Hour),
		PlanName:  "GPT Pro",
	}
	svc.groupRepo = &prepaidAccessGroupRepoStub{group: group}
	svc.userSubRepo = &prepaidSubscriptionRepoStub{sub: sub}
	users.balance.Balance = 0

	groupID := group.ID
	key, err := svc.Create(context.Background(), users.owner.ID, CreateAPIKeyRequest{
		Name:    "subscription key",
		GroupID: &groupID,
	})
	require.NoError(t, err)
	require.Equal(t, group.ID, *key.GroupID)
	require.Same(t, key, keys.key)
	require.NoError(t, svc.CheckPrepaidAccess(context.Background(), users.owner, group))

	billing := &BillingCacheService{cfg: svc.cfg, userRepo: users, subRepo: svc.userSubRepo}
	require.NoError(t, billing.CheckBillingEligibility(
		context.Background(), users.owner, key, group, sub, "openai",
	))
}

func TestPrepaidRechecksBalanceForQueuedAndWebSocketRequests(t *testing.T) {
	svc, users, _ := prepaidTestService()
	billing := &BillingCacheService{cfg: svc.cfg, userRepo: users}
	users.balance.Balance = 0
	require.ErrorIs(t, billing.CheckBillingEligibility(context.Background(), users.owner, nil, nil, nil, ""), ErrPrepaidBalanceExhausted)
}

func TestPrepaidBillingEligibilityAppliesToAdministratorAccounts(t *testing.T) {
	svc, users, _ := prepaidTestService()
	users.owner.Role = RoleAdmin
	users.balance.Balance = 0
	billing := &BillingCacheService{cfg: svc.cfg, userRepo: users}

	require.ErrorIs(t, billing.CheckBillingEligibility(
		context.Background(), users.owner, nil, &Group{}, nil, "openai",
	), ErrPrepaidBalanceExhausted)
}

func TestStarbridgePrepaidPrice(t *testing.T) {
	require.Equal(t, 1.0, calculateCreditedBalance(0.5, defaultBalanceRechargeMultiplier))
	require.Equal(t, 20.0, calculateCreditedBalance(10, defaultBalanceRechargeMultiplier))
}
