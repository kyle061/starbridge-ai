package service

import (
	"context"
	"math"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

func validateSubscriptionEntitlements(quota, multiplier float64) error {
	if math.IsNaN(quota) || math.IsInf(quota, 0) || quota < 0 || math.IsNaN(multiplier) || math.IsInf(multiplier, 0) || multiplier < 0 {
		return infraerrors.BadRequest("INVALID_SUBSCRIPTION_QUOTA", "subscription quota and multiplier must be finite non-negative numbers")
	}
	return nil
}

// SetSubscriptionQuota sets an absolute total, preserving all recorded usage and
// the current term. A row lock prevents overwriting a concurrently billed request.
func (s *SubscriptionService) SetSubscriptionQuota(ctx context.Context, id int64, quota float64, multiplier *float64) (*UserSubscription, error) {
	value := float64(0)
	if multiplier != nil {
		value = *multiplier
	}
	if err := validateSubscriptionEntitlements(quota, value); err != nil {
		return nil, err
	}
	if quota <= 0 || (multiplier != nil && value <= 0) {
		return nil, infraerrors.BadRequest("INVALID_SUBSCRIPTION_QUOTA", "quota and an explicit multiplier must be greater than zero")
	}
	var userID, groupID int64
	err := s.withSubscriptionUpdateTx(ctx, func(txCtx context.Context) error {
		sub, err := s.userSubRepo.GetByIDForUpdate(txCtx, id)
		if err != nil {
			return err
		}
		if quota < sub.QuotaUsedUSD {
			return infraerrors.BadRequest("QUOTA_BELOW_USAGE", "total quota cannot be less than consumed subscription quota")
		}
		sub.QuotaUSD = quota
		if multiplier != nil {
			sub.UsageMultiplier = value
		}
		userID, groupID = sub.UserID, sub.GroupID
		return s.userSubRepo.Update(txCtx, sub)
	})
	if err != nil {
		return nil, err
	}
	if err := s.invalidateSubscriptionCaches(userID, groupID); err != nil {
		return nil, err
	}
	return s.userSubRepo.GetByID(ctx, id)
}
