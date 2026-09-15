package service

import (
	"context"
	"fmt"

	"entgo.io/ent/dialect"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/ent/apikey"
	"github.com/Wei-Shaw/sub2api/ent/user"
)

// ensurePrepaidKey runs only after verified payment has credited the balance.
// The user row lock makes concurrent payment callbacks issue at most one first
// key. Any existing key is retained across renewals, including disabled keys.
func (s *PaymentService) ensurePrepaidKey(ctx context.Context, order *dbent.PaymentOrder) error {
	if s.apiKeyService == nil || order == nil {
		return nil
	}
	owner, err := s.userRepo.GetByID(ctx, order.UserID)
	if err != nil {
		return err
	}
	if !s.apiKeyService.RequiresBalancePurchase(owner) {
		return nil
	}
	if order.PaidAt == nil || order.PayAmount <= 0 || order.Amount <= 0 {
		return ErrBalancePurchaseRequired
	}
	tx, err := s.entClient.Tx(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	lockQuery := tx.User.Query().Where(user.IDEQ(owner.ID))
	if s.entClient.Driver().Dialect() != dialect.SQLite {
		lockQuery.ForUpdate()
	}
	locked, err := lockQuery.Only(ctx)
	if err != nil {
		return err
	}
	// A renewal can first repay a previous in-flight request's debt.
	if locked.Balance <= 0 {
		return tx.Commit()
	}
	exists, err := tx.APIKey.Query().Where(apikey.UserIDEQ(owner.ID), apikey.DeletedAtIsNil()).Exist(ctx)
	if err != nil {
		return err
	}
	if exists {
		return tx.Commit()
	}
	groups, err := s.groupRepo.ListActive(ctx)
	if err != nil {
		return err
	}
	var groupID *int64
	for _, group := range groups {
		if !group.IsSubscriptionType() && owner.CanBindGroup(group.ID, group.IsExclusive) {
			id := group.ID
			groupID = &id
			break
		}
	}
	key, err := s.apiKeyService.GenerateKey()
	if err != nil {
		return err
	}
	// If the operator has not configured a group yet, the key remains unbound
	// and cannot route requests until the user selects an available group.
	_, err = tx.APIKey.Create().SetUserID(owner.ID).SetKey(key).
		SetName("Starbridge API Key").SetStatus(StatusActive).
		SetNillableGroupID(groupID).Save(ctx)
	if err != nil {
		return fmt.Errorf("issue prepaid api key: %w", err)
	}
	return tx.Commit()
}
