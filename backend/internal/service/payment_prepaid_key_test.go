//go:build unit

package service

import (
	"context"
	"errors"
	"testing"
	"time"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/stretchr/testify/require"
)

type prepaidGroupRepoStub struct {
	GroupRepository
	groups []Group
	err    error
}

func (r *prepaidGroupRepoStub) ListActive(context.Context) ([]Group, error) { return r.groups, r.err }

type prepaidFulfillmentUserRepo struct {
	prepaidUserRepoStub
	client *dbent.Client
}

func (r *prepaidFulfillmentUserRepo) UpdateBalance(ctx context.Context, id int64, amount float64) error {
	client := r.client
	if tx := dbent.TxFromContext(ctx); tx != nil {
		client = tx.Client()
	}
	_, err := client.User.UpdateOneID(id).AddBalance(amount).Save(ctx)
	return err
}

func TestPrepaidFulfillmentRetryCreditsOnceAndIssuesKey(t *testing.T) {
	ctx := context.Background()
	client := newPaymentConfigServiceTestClient(t)
	owner, err := client.User.Create().SetEmail("retry@example.com").SetPasswordHash("test").Save(ctx)
	require.NoError(t, err)
	group, err := client.Group.Create().SetName("prepaid-retry").Save(ctx)
	require.NoError(t, err)
	order, err := client.PaymentOrder.Create().SetUserID(owner.ID).
		SetUserEmail(owner.Email).SetUserName("prepaid-user").SetFeeRate(0).
		SetOutTradeNo("PREPAID-RETRY").SetPaymentType("alipay").SetPaymentTradeNo("paid-retry").
		SetClientIP("127.0.0.1").SetSrcHost("localhost").
		SetAmount(20).SetPayAmount(10).SetOrderType("balance").SetStatus(OrderStatusPaid).
		SetPaidAt(time.Now()).SetExpiresAt(time.Now().Add(time.Hour)).SetRechargeCode("PREPAID-RETRY").Save(ctx)
	require.NoError(t, err)
	keySvc, _, _ := prepaidTestService()
	users := &prepaidFulfillmentUserRepo{client: client, prepaidUserRepoStub: prepaidUserRepoStub{owner: &User{ID: owner.ID, Role: RoleUser, Status: StatusActive}}}
	groups := &prepaidGroupRepoStub{groups: []Group{{ID: group.ID}}, err: errors.New("temporary group lookup failure")}
	redeem := NewRedeemService(&paymentFulfillmentRedeemRepo{}, users, nil, nil, nil, client, nil, nil)
	svc := &PaymentService{entClient: client, apiKeyService: keySvc, userRepo: users, groupRepo: groups, redeemService: redeem}
	// Credit commits first; a subsequent key-issuance failure remains retryable.
	require.Error(t, svc.ExecuteBalanceFulfillment(ctx, order.ID))
	credited, err := client.User.Get(ctx, owner.ID)
	require.NoError(t, err)
	require.Equal(t, 20.0, credited.Balance)
	count, err := client.APIKey.Query().Count(ctx)
	require.NoError(t, err)
	require.Zero(t, count)
	groups.err = nil
	require.NoError(t, svc.RetryFulfillment(ctx, order.ID))
	require.NoError(t, svc.ExecuteBalanceFulfillment(ctx, order.ID))
	credited, err = client.User.Get(ctx, owner.ID)
	require.NoError(t, err)
	require.Equal(t, 20.0, credited.Balance, "retry must not credit the balance twice")
	key, err := client.APIKey.Query().Only(ctx)
	require.NoError(t, err)
	require.Nil(t, key.ExpiresAt)
	completed, err := client.PaymentOrder.Get(ctx, order.ID)
	require.NoError(t, err)
	require.Equal(t, OrderStatusCompleted, completed.Status)
}

func TestPaymentPrepaidKeyIssuedOnceAndRetainedAfterRenewal(t *testing.T) {
	ctx := context.Background()
	client := newPaymentConfigServiceTestClient(t)
	owner, err := client.User.Create().SetEmail("prepaid@example.com").SetPasswordHash("test").SetBalance(20).Save(ctx)
	require.NoError(t, err)
	group, err := client.Group.Create().SetName("prepaid").Save(ctx)
	require.NoError(t, err)
	keySvc, users, _ := prepaidTestService()
	users.owner.ID = owner.ID
	svc := &PaymentService{entClient: client, apiKeyService: keySvc, userRepo: users, groupRepo: &prepaidGroupRepoStub{groups: []Group{{ID: group.ID}}}}
	paidAt := time.Now()
	order := &dbent.PaymentOrder{UserID: owner.ID, PaidAt: &paidAt, Amount: 20, PayAmount: 10}
	require.NoError(t, svc.ensurePrepaidKey(ctx, order))
	first, err := client.APIKey.Query().Only(ctx)
	require.NoError(t, err)
	require.Nil(t, first.ExpiresAt)
	require.Equal(t, group.ID, *first.GroupID)
	require.Equal(t, StatusActive, first.Status)
	// Duplicate callbacks and later renewals retain the exact same credential.
	require.NoError(t, svc.ensurePrepaidKey(ctx, order))
	order.ID++
	require.NoError(t, svc.ensurePrepaidKey(ctx, order))
	keys, err := client.APIKey.Query().All(ctx)
	require.NoError(t, err)
	require.Len(t, keys, 1)
	require.Equal(t, first.Key, keys[0].Key)
	_, err = client.APIKey.UpdateOneID(first.ID).SetStatus("inactive").Save(ctx)
	require.NoError(t, err)
	require.NoError(t, svc.ensurePrepaidKey(ctx, order))
	retained, err := client.APIKey.Get(ctx, first.ID)
	require.NoError(t, err)
	require.Equal(t, "inactive", retained.Status)
}

func TestPaymentPrepaidKeyNotIssuedBeforeConfirmedPayment(t *testing.T) {
	keySvc, users, _ := prepaidTestService()
	svc := &PaymentService{apiKeyService: keySvc, userRepo: users}
	require.ErrorIs(t, svc.ensurePrepaidKey(context.Background(), &dbent.PaymentOrder{UserID: 7, Amount: 20, PayAmount: 10}), ErrBalancePurchaseRequired)
}
