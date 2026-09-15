package service

import (
	"context"
	"math"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

var (
	ErrBalancePurchaseRequired  = infraerrors.Forbidden("BALANCE_PURCHASE_REQUIRED", "请先充值 API 美元额度，再生成和使用 Key")
	ErrPrepaidBalanceExhausted  = infraerrors.Forbidden("INSUFFICIENT_BALANCE", "账户余额已用完，请充值后继续使用原 Key")
	ErrPrepaidAccessUnavailable = infraerrors.ServiceUnavailable("PREPAID_ACCESS_UNAVAILABLE", "暂时无法核对充值余额，请稍后重试")
	ErrPrepaidGroupRequired     = infraerrors.Forbidden("PREPAID_GROUP_REQUIRED", "预付费 Key 需要使用按余额计费的分组")
)

// PrepaidBalance is read from the primary database; auth caches are deliberately
// not used for this decision, so a settled debit pauses every key for the user.
type PrepaidBalance struct {
	Balance      float64
	HasPurchased bool
}

type PrepaidAccessRepository interface {
	GetPrepaidBalance(context.Context, int64) (*PrepaidBalance, error)
}

type PrepaidAccess struct {
	Enabled         bool    `json:"enabled"`
	HasPurchased    bool    `json:"has_purchased"`
	Balance         float64 `json:"balance"`
	CanCreateKey    bool    `json:"can_create_key"`
	RequestsAllowed bool    `json:"requests_allowed"`
}

func (s *APIKeyService) RequiresBalancePurchase(user *User) bool {
	return s != nil && s.cfg != nil && s.cfg.Billing.RequireBalancePurchase && user != nil && !user.IsAdmin()
}

func (s *APIKeyService) prepaidAccess(ctx context.Context, user *User) (*PrepaidAccess, error) {
	return loadPrepaidAccess(ctx, s.userRepo, user, s.RequiresBalancePurchase(user))
}

func loadPrepaidAccess(ctx context.Context, userRepo UserRepository, user *User, enabled bool) (*PrepaidAccess, error) {
	access := &PrepaidAccess{Enabled: enabled, Balance: user.Balance, CanCreateKey: user.IsActive(), RequestsAllowed: user.IsActive()}
	if !access.Enabled {
		return access, nil
	}
	repo, ok := userRepo.(PrepaidAccessRepository)
	if !ok {
		return nil, ErrPrepaidAccessUnavailable
	}
	balance, err := repo.GetPrepaidBalance(ctx, user.ID)
	if err != nil || balance == nil || math.IsNaN(balance.Balance) || math.IsInf(balance.Balance, 0) {
		return nil, ErrPrepaidAccessUnavailable
	}
	access.Balance = balance.Balance
	// Existing clients use has_purchased for recharge prompts. Offline credit
	// also unlocks access; an online payment order is no longer required.
	access.HasPurchased = balance.HasPurchased || balance.Balance > 0
	access.RequestsAllowed = user.IsActive() && balance.Balance > 0
	access.CanCreateKey = access.RequestsAllowed
	return access, nil
}

func (s *APIKeyService) GetPrepaidAccess(ctx context.Context, userID int64) (*PrepaidAccess, error) {
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, ErrUserNotFound
	}
	return s.prepaidAccess(ctx, user)
}

func (s *APIKeyService) CheckPrepaidAccess(ctx context.Context, user *User, group *Group) error {
	if !s.RequiresBalancePurchase(user) {
		return nil
	}
	if err := checkPrepaidBalance(ctx, s.userRepo, user, group); err != nil {
		return err
	}
	if group == nil {
		return ErrPrepaidGroupRequired
	}
	return nil
}

func checkPrepaidBalance(ctx context.Context, userRepo UserRepository, user *User, group *Group) error {
	access, err := loadPrepaidAccess(ctx, userRepo, user, true)
	if err != nil {
		return err
	}
	if !access.RequestsAllowed {
		return ErrPrepaidBalanceExhausted
	}
	if group != nil && group.IsSubscriptionType() {
		return ErrPrepaidGroupRequired
	}
	// Hydrated keys are request-local copies. Refresh the balance used by the
	// subsequent legacy auth checks as well, including immediately after renewal.
	user.Balance = access.Balance
	return nil
}
